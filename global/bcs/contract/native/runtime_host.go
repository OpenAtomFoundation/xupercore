package native

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	log "github.com/xuperchain/log15"
)

// HostProcess is the process running as a native process
type HostProcess struct {
	basedir  string
	startcmd *exec.Cmd
	envs     []string

	cmd *exec.Cmd
	log.Logger
}

// Start implements process interface
func (h *HostProcess) Start() error {
	cmd := h.startcmd
	cmd.Dir = h.basedir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
		Pgid:   0,
	}
	cmd.Env = []string{"XCHAIN_PING_TIMEOUT=" + strconv.Itoa(pingTimeoutSecond)}
	cmd.Env = append(cmd.Env, h.envs...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}
	h.Info("start command success", "pid", cmd.Process.Pid)
	h.cmd = cmd
	return nil
}

func processExists(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}

// Stop implements process interface
func (h *HostProcess) Stop(timeout time.Duration) error {
	h.cmd.Process.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(h.cmd.Process.Pid) {
			break
		}
		time.Sleep(time.Second)
	}
	// force kill if timeout
	if !time.Now().Before(deadline) {
		h.cmd.Process.Kill()
	}
	h.Info("stop command success", "pid", h.cmd.Process.Pid)
	return h.cmd.Wait()
}
