package native

import (
	"os"
	"strconv"

	docker "github.com/fsouza/go-dockerclient"
)

// Start implements process interface
func (d *DockerProcess) start() error {
	client, err := getDockerClient()
	if err != nil {
		return err
	}
	volumes := map[string]struct{}{}
	for _, mount := range d.mounts {
		volumes[mount] = struct{}{}
	}
	// cmd.Args contains cmd binpath
	cmd := d.startcmd.Args

	env := []string{
		"XCHAIN_PING_TIMEOUT=" + strconv.Itoa(pingTimeoutSecond),
		// 合约进程其实只需要 expose 第一个端口
		"XCHAIN_CODE_ADDR=" + "tcp://0.0.0.0:" + d.ports[0],
	}
	env = append(env, d.envs...)

	user := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())

	cpulimit, memlimit, err := d.resourceConfig()
	if err != nil {
		return err
	}

	binds := make([]string, len(d.mounts))
	for i := range d.mounts {
		binds[i] = d.mounts[i] + ":" + d.mounts[i]
	}

	portBinds := make(map[docker.Port][]docker.PortBinding)
	exposedPorts := map[docker.Port]struct{}{}

	for _, port := range d.ports {
		key := docker.Port(port + "/tcp")
		value := []docker.PortBinding{
			{
				HostIP:   "127.0.0.1",
				HostPort: port,
			},
		}
		portBinds[key] = value
		exposedPorts[key] = struct{}{}
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			ExposedPorts: exposedPorts,
			Volumes:      volumes,
			Env:          env,
			WorkingDir:   d.basedir,
			Image:        d.cfg.ImageName,
			Cmd:          cmd,
			User:         user,
		},
		HostConfig: &docker.HostConfig{
			NetworkMode:  "bridge",
			AutoRemove:   true,
			Binds:        binds,
			CPUPeriod:    cpulimit,
			Memory:       memlimit,
			PortBindings: portBinds,
		},
	}
	container, err := client.CreateContainer(opts)
	if err != nil {
		return err
	}
	d.Info("create container success", "id", container.ID)
	d.id = container.ID

	err = client.StartContainer(d.id, nil)
	if err != nil {
		return err
	}
	d.Info("start container success", "id", d.id)
	return nil
}
