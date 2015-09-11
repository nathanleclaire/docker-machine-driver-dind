package dind

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
	"github.com/samalba/dockerclient"
)

type Driver struct {
	*drivers.BaseDriver
	Id string
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

	return nil
}

func (d *Driver) DriverName() string {
	return "dind"
}

func (d *Driver) GetIP() (string, error) {
	return "", nil
}

func (d *Driver) GetMachineName() string {
	return d.MachineName
}

func (d *Driver) GetSSHHostname() (string, error) {
	return "", nil
}

func (d *Driver) GetSSHKeyPath() string {
	return ""
}

func (d *Driver) GetSSHPort() (int, error) {
	return 0, nil
}

func (d *Driver) GetSSHUsername() string {
	return ""
}

func (d *Driver) GetURL() (string, error) {
	return "", nil
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
	return nil
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
