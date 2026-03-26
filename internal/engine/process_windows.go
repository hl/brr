//go:build windows

package engine

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcAttr is a no-op on Windows; process groups work differently.
func setProcAttr(cmd *exec.Cmd) {
	// On Windows, CREATE_NEW_PROCESS_GROUP is set via SysProcAttr.CreationFlags
	// but Go's windows syscall doesn't expose Setpgid. Child processes are
	// terminated via cmd.Process.Kill() instead of group signals.
}

// killGroup sends a termination signal to the child process on Windows.
// Windows doesn't support Unix process groups, so we kill the process directly.
func killGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	// On Windows, the only reliable way to stop a child is Process.Kill or
	// Process.Signal. SIGINT/SIGTERM are approximated by os.Kill / os.Interrupt.
	switch sig {
	case syscall.SIGINT:
		return cmd.Process.Signal(os.Interrupt)
	default:
		return cmd.Process.Kill()
	}
}
