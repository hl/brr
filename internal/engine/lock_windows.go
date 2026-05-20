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

func AcquireLock() (*os.File, error) {
	fi, err := os.Lstat(lockFile)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file (symlinks/dirs not allowed)", lockFile)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking lock file: %w", err)
	}

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
	}

	fi2, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	if fi != nil && !os.SameFile(fi, fi2) {
		_ = f.Close()
		return nil, fmt.Errorf("lock file changed between stat and open")
	}

	h := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	r1, _, errno := procLockFileEx.Call(
		uintptr(h),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		_ = f.Close()
		// ERROR_LOCK_VIOLATION (33) means another process holds the lock
		if errno == syscall.Errno(33) {
			return nil, fmt.Errorf("another brr instance is already running in this directory")
		}
		return nil, fmt.Errorf("acquiring lock: %w", errno)
	}
	return f, nil
}

// releaseLock releases the advisory lock and closes the file handle.
// The lock file is intentionally kept on disk to prevent a race where
// another process acquires the old handle just before it is deleted.
func ReleaseLock(f *os.File) {
	h := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	_, _, _ = procUnlockFileEx.Call(
		uintptr(h),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	_ = f.Close()
}
