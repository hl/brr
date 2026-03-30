//go:build !windows

package engine

import (
	"os/exec"
	"syscall"
	"time"
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

// reapGroup cleans up any orphaned processes remaining in the child's process
// group after the child has exited. AI agents often spawn subprocesses (MCP
// servers, language servers, etc.) that outlive the main agent process; without
// this cleanup they accumulate across iterations, leaking memory and file
// descriptors.
func reapGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	// SIGTERM: give orphans a chance to clean up
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return // group already empty
	}
	time.Sleep(100 * time.Millisecond)
	// SIGKILL: force-remove anything that ignored SIGTERM
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}
