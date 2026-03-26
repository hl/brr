//go:build !windows

package engine

import (
	"os/exec"
	"syscall"
)

// setProcAttr configures the command to run in its own process group
// so we can signal the entire child tree.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup sends a signal to the process group led by the given process.
func killGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, sig)
}
