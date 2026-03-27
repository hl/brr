//go:build windows

package engine

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const lockFile = ".brr.lock"

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x02
	lockfileFailImmediately = 0x01
)

func acquireLock() (*os.File, error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
	}
	h := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	r1, _, _ := procLockFileEx.Call(
		uintptr(h),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		_ = f.Close()
		return nil, fmt.Errorf("another brr instance is already running in this directory")
	}
	return f, nil
}

func releaseLock(f *os.File) {
	h := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	_, _, _ = procUnlockFileEx.Call(
		uintptr(h),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	_ = f.Close()
	_ = os.Remove(lockFile)
}
