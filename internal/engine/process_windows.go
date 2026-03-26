//go:build windows

package engine

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const createNewProcessGroup = 0x00000200

// setProcAttr configures the command to run in a new process group on Windows
// so that console control events and tree kills target the child tree only.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup,
	}
}

// killGroup sends a termination signal to the child process tree on Windows.
// For SIGINT: sends CTRL_BREAK_EVENT via os.Interrupt to the child's process group.
// For SIGKILL: uses taskkill /T /F to terminate the entire process tree.
func killGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	switch sig {
	case syscall.SIGINT:
		// os.Interrupt sends CTRL_BREAK_EVENT to the child's process group
		return cmd.Process.Signal(os.Interrupt)
	default:
		// Kill the entire process tree
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		if err := kill.Run(); err != nil {
			// Fallback: kill just the direct process
			return cmd.Process.Kill()
		}
		return nil
	}
}
