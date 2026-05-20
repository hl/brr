//go:build !windows

package engine

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const lockFile = ".brr.lock"

// acquireLock attempts to acquire an exclusive lock on .brr.lock in the working
// directory. Returns the lock file handle (caller must defer releaseLock) or an
// error if another brr instance is already running in this directory.
func AcquireLock() (*os.File, error) {
	fi, err := os.Lstat(lockFile)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file (symlinks/dirs not allowed)", lockFile)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking lock file: %w", err)
	}

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
	}

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

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("another brr instance is already running in this directory")
		}
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}
	return f, nil
}

// releaseLock releases the advisory lock and closes the file handle.
// The lock file is intentionally kept on disk to prevent a race where
// another process acquires the old inode just before it is unlinked.
func ReleaseLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
