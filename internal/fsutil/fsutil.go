package fsutil

import (
	"fmt"
	"io"
	"os"
)

// ErrNotRegularFile is returned when a path exists but is not a regular file.
var ErrNotRegularFile = fmt.Errorf("not a regular file")

// OpenRegularFile opens path only if it is a regular file (not a symlink, dir, etc.).
// Uses Lstat→Open→Fstat(SameFile) to prevent TOCTOU symlink-swap attacks.
func OpenRegularFile(path string) (*os.File, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, ErrNotRegularFile
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi2, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !os.SameFile(fi, fi2) {
		_ = f.Close()
		return nil, fmt.Errorf("%s: file changed between stat and open", path)
	}
	return f, nil
}

// ReadRegularFile reads path only if it is a regular file (not a symlink, dir, etc.).
func ReadRegularFile(path string) ([]byte, error) {
	f, err := OpenRegularFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(f)
}

// IsRegularFile returns true if path exists and is a regular file.
func IsRegularFile(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}
