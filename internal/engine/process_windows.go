//go:build windows

package engine

import (
	"os"
	"os/exec"
	"path/filepath"
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
// For SIGINT: attempts graceful tree kill via taskkill, then falls back to Process.Kill.
// For SIGKILL/SIGTERM: uses taskkill /T /F to force-terminate the entire process tree.
func killGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	var taskkill string
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		taskkill = filepath.Join(systemRoot, "System32", "taskkill.exe")
	} else {
		taskkill = "taskkill.exe"
	}

	switch sig {
	case syscall.SIGINT:
		// Graceful tree kill (no /F) — gives the child a chance to clean up
		kill := exec.Command(taskkill, "/T", "/PID", pid)
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	default:
		// Force kill the entire process tree
		kill := exec.Command(taskkill, "/T", "/F", "/PID", pid)
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
