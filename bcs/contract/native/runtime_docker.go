package native

import (
	"os/exec"
	"sync"
	"time"

	"github.com/docker/go-units"
	docker "github.com/fsouza/go-dockerclient"
	log "github.com/xuperchain/log15"

	"github.com/xuperchain/xupercore/kernel/contract"
)

var (
	dockerOnce   sync.Once
	dockerClient *docker.Client
)

// DockerProcess is the process running as a docker container
type DockerProcess struct {
	basedir  string
	startcmd *exec.Cmd
	envs     []string
	mounts   []string
	ports    []string
	cfg      *contract.NativeDockerConfig

	id string
	log.Logger
}

func (d *DockerProcess) resourceConfig() (int64, int64, error) {
	const cpuPeriod = 100000

	var cpuLimit, memLimit int64
	cpuLimit = int64(cpuPeriod * d.cfg.Cpus)
	if d.cfg.Memory != "" {
		var err error
		memLimit, err = units.RAMInBytes(d.cfg.Memory)
		if err != nil {
			return 0, 0, err
		}
	}
	return cpuLimit, memLimit, nil
}

func (d *DockerProcess) Start() error {
	return d.start()
}

// Stop implements process interface
func (d *DockerProcess) Stop(timeout time.Duration) error {
	client, err := getDockerClient()
	if err != nil {
		return err
	}
	err = client.StopContainer(d.id, uint(timeout.Seconds()))
	if err != nil {
		return err
	}
	d.Info("stop container success", "id", d.id)
	client.WaitContainer(d.id)
	d.Info("wait container success", "id", d.id)
	return nil
}

func getDockerClient() (*docker.Client, error) {
	var err error
	dockerOnce.Do(func() {
		dockerClient, err = docker.NewClientFromEnv()
	})
	if err != nil {
		return nil, err
	}
	return dockerClient, nil
}
