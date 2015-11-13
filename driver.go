package dind

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/samalba/dockerclient"
)

const (
	defaultDockerHost = "unix:///var/run/docker.sock"
)

type Driver struct {
	*drivers.BaseDriver
	Id            string
	ContainerHost string
	DindImage     string
	DockerHost    string
	CertPath      string
	BeingCreated  bool
}

func NewDriver(hostName, storePath string) Driver {
	return Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
		},
	}
}

func (d *Driver) newDockerClient() (*dockerclient.DockerClient, error) {
	tlsc := &tls.Config{}

	if d.CertPath != "" {
		cert, err := tls.LoadX509KeyPair(filepath.Join(d.CertPath, "cert.pem"), filepath.Join(d.CertPath, "key.pem"))
		if err != nil {
			return nil, fmt.Errorf("Error loading x509 key pair: %s", err)
		}

		tlsc.Certificates = append(tlsc.Certificates, cert)
		tlsc.InsecureSkipVerify = true
	}

	dc, err := dockerclient.NewDockerClient(d.DockerHost, tlsc)
	if err != nil {
		return nil, fmt.Errorf("Error getting Docker Client: %s", err)
	}

	return dc, nil
}

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			Name:  "dind-image",
			Usage: "Image to run for the Docker-in-Docker stack.",
			Value: "nathanleclaire/docker-machine-dind",
		},
		mcnflag.StringFlag{
			Name:   "dind-host",
			Usage:  "URL of Docker host to use for the dind container.",
			Value:  defaultDockerHost,
			EnvVar: "DOCKER_HOST",
		},
		mcnflag.StringFlag{
			Name:   "dind-cert-path",
			Usage:  "Cert path for the docker host in usage for the dind container.",
			EnvVar: "DOCKER_CERT_PATH",
			Value:  "",
		},
	}
}

func (d *Driver) Create() error {
	d.BeingCreated = true

	dc, err := d.newDockerClient()
	if err != nil {
		return err
	}

	u, err := url.Parse(os.Getenv("DOCKER_HOST"))
	if err != nil {
		return fmt.Errorf("Error parsing URL from DOCKER_HOST: %s", err)
	}

	if d.DockerHost == defaultDockerHost {
		d.ContainerHost = "localhost"
	} else {
		d.ContainerHost = strings.Split(u.Host, ":")[0]
	}

	containerConfig := &dockerclient.ContainerConfig{
		Image: d.DindImage,
		HostConfig: dockerclient.HostConfig{
			PublishAllPorts: true,
			Privileged:      true,
		},
	}

	id, err := dc.CreateContainer(containerConfig, d.MachineName)
	if err != nil {
		return fmt.Errorf("Error creating container: %s", err)
	}

	d.Id = id

	if err := d.Start(); err != nil {
		return err
	}

	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return err
	}

	f, err := os.Open(d.GetSSHKeyPath() + ".pub")
	if err != nil {
		return fmt.Errorf("Error opening pub key file: %s", err)
	}

	pubKey, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Error reading from pub key file: %s", err)
	}

	execConfig := &dockerclient.ExecConfig{
		Cmd:       []string{"sh", "-c", fmt.Sprintf("echo %q >/root/.ssh/authorized_keys", strings.TrimSpace(string(pubKey)))},
		Container: d.Id,
	}

	spew.Dump(execConfig)

	execId, err := dc.ExecCreate(execConfig)
	if err != nil {
		return fmt.Errorf("Error creating exec: %s", err)
	}

	if err := dc.ExecStart(execId, execConfig); err != nil {
		return fmt.Errorf("Error starting exec: %s", err)
	}

	return nil
}

func (d *Driver) DriverName() string {
	return "dind"
}

func (d *Driver) GetIP() (string, error) {
	if d.DockerHost == "unix:///var/run/docker.sock" {
		info, err := d.getContainerInfo()
		if err != nil {
			return "", err
		}

		return info.NetworkSettings.IPAddress, nil
	}
	return d.ContainerHost, nil
}

func (d *Driver) GetMachineName() string {
	return d.MachineName
}

func (d *Driver) GetSSHHostname() (string, error) {
	if d.DockerHost == "unix:///var/run/docker.sock" {
		return d.GetIP()
	}
	return d.ContainerHost, nil
}

func (d *Driver) getContainerInfo() (*dockerclient.ContainerInfo, error) {
	dc, err := d.newDockerClient()
	if err != nil {
		return nil, err
	}

	info, err := dc.InspectContainer(d.Id)
	if err != nil {
		return nil, fmt.Errorf("Error inspecting container: %s", err)
	}

	return info, nil
}

func (d *Driver) getExposedPort(containerPort string) (int, error) {
	info, err := d.getContainerInfo()
	if err != nil {
		return 0, err
	}

	exposedPort, err := strconv.Atoi(info.NetworkSettings.Ports[fmt.Sprintf("%s/tcp", containerPort)][0].HostPort)
	if err != nil {
		return 0, fmt.Errorf("Error parsing host port to int: %s")
	}

	return exposedPort, nil
}

func (d *Driver) GetSSHPort() (int, error) {
	if d.DockerHost == "unix:///var/run/docker.sock" {
		return 22, nil
	}
	return d.getExposedPort("22")
}

func (d *Driver) GetSSHUsername() string {
	return "root"
}

func (d *Driver) GetURL() (string, error) {
	if d.DockerHost == "unix:///var/run/docker.sock" {
		ip, err := d.GetIP()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("tcp://%s:2376", ip), nil
	} else {
		if d.BeingCreated {
			// HACK: First time on creation, trick provisioning into using 2376 for the URL.
			d.BeingCreated = false
			return fmt.Sprintf("tcp://%s:2376", d.ContainerHost), nil
		}

		s, err := d.GetState()
		if err != nil {
			return "", fmt.Errorf("Error getting state: %s", err)
		}

		if s != state.Running {
			return "", nil
		}

		dockerPort, err := d.getExposedPort("2376")
		if err != nil {
			return "", fmt.Errorf("Error trying to get exposed port: %s", err)
		}

		return fmt.Sprintf("tcp://%s:%d", d.ContainerHost, dockerPort), nil
	}
}

func (d *Driver) GetState() (state.State, error) {
	dc, err := d.newDockerClient()
	if err != nil {
		return state.Error, err
	}

	info, err := dc.InspectContainer(d.Id)
	if err != nil {
		return state.Error, fmt.Errorf("Error inspecting container: %s", err)
	}

	spew.Dump(info)

	if info.State.Running {
		return state.Running, nil
	}

	return state.Stopped, nil
}

func (d *Driver) Kill() error {
	return nil
}

func (d *Driver) PreCreateCheck() error {
	return nil
}

func (d *Driver) Remove() error {
	dc, err := d.newDockerClient()
	if err != nil {
		return err
	}

	return dc.RemoveContainer(d.Id, true, true)
}

func (d *Driver) Restart() error {
	return nil
}

func (d *Driver) SetConfigFromFlags(opts drivers.DriverOptions) error {
	spew.Dump(opts)
	d.DindImage = opts.String("dind-image")
	d.DockerHost = opts.String("dind-host")
	d.CertPath = opts.String("dind-cert-path")
	return nil
}

func (d *Driver) Start() error {
	dc, err := d.newDockerClient()
	if err != nil {
		return err
	}

	return dc.StartContainer(d.Id, nil)
}

func (d *Driver) Stop() error {
	dc, err := d.newDockerClient()
	if err != nil {
		return err
	}

	return dc.StopContainer(d.Id, 10)
}
