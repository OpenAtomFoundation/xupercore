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

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Volumes:    volumes,
			Env:        env,
			WorkingDir: d.basedir,
			// NetworkDisabled: true,
			Image: d.cfg.ImageName,
			Cmd:   cmd,
			User:  user,
		},
		HostConfig: &docker.HostConfig{
			NetworkMode: "host",
			AutoRemove:  true,
			Binds:       binds,
			CPUPeriod:   cpulimit,
			Memory:      memlimit,
			// PortBindings: portBinds,
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
