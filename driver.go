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

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/samalba/dockerclient"
)

type Driver struct {
	*drivers.BaseDriver
	Id            string
	ContainerHost string
}

func NewDriver(hostName, artifactPath string) Driver {
	return Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName:  hostName,
			ArtifactPath: artifactPath,
		},
	}
}

func newDockerClient() (*dockerclient.DockerClient, error) {
	tlsc := &tls.Config{}
	dockerCertPath := os.Getenv("DOCKER_CERT_PATH")

	cert, err := tls.LoadX509KeyPair(filepath.Join(dockerCertPath, "cert.pem"), filepath.Join(dockerCertPath, "key.pem"))
	if err != nil {
		return nil, fmt.Errorf("Error loading x509 key pair: %s", err)
	}

	tlsc.Certificates = append(tlsc.Certificates, cert)
	tlsc.InsecureSkipVerify = true

	dc, err := dockerclient.NewDockerClient(os.Getenv("DOCKER_HOST"), tlsc)
	if err != nil {
		return nil, fmt.Errorf("Error getting Docker Client: %s", err)
	}

	return dc, nil
}

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{}
}

func (d *Driver) Create() error {
	dc, err := newDockerClient()
	if err != nil {
		return err
	}

	u, err := url.Parse(os.Getenv("DOCKER_HOST"))
	if err != nil {
		return fmt.Errorf("Error parsing URL from DOCKER_HOST: %s", err)
	}

	d.ContainerHost = strings.Split(u.Host, ":")[0]

	containerConfig := &dockerclient.ContainerConfig{
		Image: "nathanleclaire/docker-machine-dind",
		HostConfig: dockerclient.HostConfig{
			PublishAllPorts: true,
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
		Cmd:       []string{"printf", fmt.Sprintf("'%s'", strings.TrimSpace(string(pubKey))), ">/root/.ssh/authorized_keys"},
		Container: d.Id,
	}

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
	return d.ContainerHost, nil
}

func (d *Driver) GetMachineName() string {
	return d.MachineName
}

func (d *Driver) GetSSHHostname() (string, error) {
	return d.ContainerHost, nil
}

func (d *Driver) getExposedPort(containerPort string) (int, error) {
	dc, err := newDockerClient()
	if err != nil {
		return 0, err
	}

	info, err := dc.InspectContainer(d.Id)
	if err != nil {
		return 0, fmt.Errorf("Error inspecting container: %s", err)
	}

	exposedPort, err := strconv.Atoi(info.NetworkSettings.Ports[fmt.Sprintf("%s/tcp", containerPort)][0].HostPort)
	if err != nil {
		return 0, fmt.Errorf("Error parsing host port to int: %s")
	}

	return exposedPort, nil
}

func (d *Driver) GetSSHPort() (int, error) {
	return d.getExposedPort("22")
}

func (d *Driver) GetSSHUsername() string {
	return "root"
}

func (d *Driver) GetURL() (string, error) {
	dockerPort, err := d.getExposedPort("2376")
	if err != nil {
		return "", fmt.Errorf("Error trying to get exposed port: %s", err)
	}
	return fmt.Sprintf("tcp://%s:%d", d.ContainerHost, dockerPort), nil
}

func (d *Driver) GetState() (state.State, error) {
	dc, err := newDockerClient()
	if err != nil {
		return state.Error, err
	}

	info, err := dc.InspectContainer(d.Id)
	if err != nil {
		return state.Error, fmt.Errorf("Error inspecting container: %s", err)
	}

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
	dc, err := newDockerClient()
	if err != nil {
		return err
	}

	return dc.RemoveContainer(d.Id, true, true)
}

func (d *Driver) Restart() error {
	return nil
}

func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	return nil
}

func (d *Driver) Start() error {
	dc, err := newDockerClient()
	if err != nil {
		return err
	}

	return dc.StartContainer(d.Id, nil)
}

func (d *Driver) Stop() error {
	return nil
}
