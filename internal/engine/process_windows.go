//go:build windows

package engine

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const createNewProcessGroup = 0x00000200

// setProcAttr configures the command to run in a new process group on Windows
// so that console control events and tree kills target the child tree only.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup,
	}
}

// killProcessTree recursively terminates all active descendants of parentPID.
// If killParent is true, it also terminates parentPID itself.
func killProcessTree(parentPID uint32, killParent bool) {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer syscall.CloseHandle(snapshot)

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = syscall.Process32First(snapshot, &pe)
	if err != nil {
		return
	}

	children := make(map[uint32][]uint32)
	for {
		children[pe.ParentProcessID] = append(children[pe.ParentProcessID], pe.ProcessID)
		err = syscall.Process32Next(snapshot, &pe)
		if err != nil {
			break
		}
	}

	var terminateTree func(uint32)
	terminateTree = func(pid uint32) {
		for _, childPID := range children[pid] {
			terminateTree(childPID)
		}
		if pid != parentPID || killParent {
			h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, pid)
			if err == nil {
				_ = syscall.TerminateProcess(h, 1)
				_ = syscall.CloseHandle(h)
			}
		}
	}

	terminateTree(parentPID)
}

// reapGroup cleans up any orphaned processes remaining in the child's process
// tree after the child has exited. Uses pure Go Toolhelp API traversal.
func reapGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// The parent process is already dead at this stage, so we only terminate descendants.
	killProcessTree(uint32(cmd.Process.Pid), false)
}

// killGroup sends a termination signal to the child process tree on Windows.
func killGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	pid := uint32(cmd.Process.Pid)

	switch sig {
	case syscall.SIGINT:
		// Attempt graceful shutdown by terminating children first, then parent
		killProcessTree(pid, false)
		return cmd.Process.Kill()
	default:
		// Force kill the entire process tree including the parent
		killProcessTree(pid, true)
		return cmd.Process.Kill()
	}
}
