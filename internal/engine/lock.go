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
func acquireLock() (*os.File, error) {
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating lock file: %w", err)
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
func releaseLock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
