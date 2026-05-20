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
	lockfileExclusiveLock    = 0x02
	lockfileFailImmediately  = 0x01
	fileFlagOpenReparsePoint = 0x00200000
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

	path, err := syscall.UTF16PtrFromString(lockFile)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
	}
	h, err := syscall.CreateFile(
		path,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL|fileFlagOpenReparsePoint,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
	}
	f := os.NewFile(uintptr(h), lockFile)

	fi2, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	fi3, err := os.Lstat(lockFile)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("checking opened lock file: %w", err)
	}
	if !fi3.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("%s is not a regular file (symlinks/dirs not allowed)", lockFile)
	}

	if fi != nil && !os.SameFile(fi, fi2) {
		_ = f.Close()
		return nil, fmt.Errorf("lock file changed between stat and open")
	}
	if !os.SameFile(fi2, fi3) {
		_ = f.Close()
		return nil, fmt.Errorf("lock file changed between open and verification")
	}

	h = syscall.Handle(f.Fd())
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
