package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRegularFileRejectsSymlink(t *testing.T) {
	t.Chdir(t.TempDir())

	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, "link"); err != nil {
		t.Skip("symlinks not supported")
	}

	_, err := OpenRegularFile("link")
	if err == nil {
		t.Error("expected error when opening symlink")
	}
}

func TestOpenRegularFileRejectsDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.Mkdir("dir", 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := OpenRegularFile("dir")
	if err == nil {
		t.Error("expected error when opening directory")
	}
}

func TestOpenRegularFileSuccess(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("regular.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := OpenRegularFile("regular.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = f.Close()
}

func TestReadRegularFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("data.txt", []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := ReadRegularFile("data.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("expected 'content', got %q", string(data))
	}
}

func TestReadRegularFileRejectsSymlink(t *testing.T) {
	t.Chdir(t.TempDir())

	target := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, "link.txt"); err != nil {
		t.Skip("symlinks not supported")
	}

	_, err := ReadRegularFile("link.txt")
	if err == nil {
		t.Error("expected error when reading symlink")
	}
}

func TestIsRegularFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("file.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsRegularFile("file.txt") {
		t.Error("expected true for regular file")
	}
	if IsRegularFile("nonexistent") {
		t.Error("expected false for nonexistent file")
	}
	if err := os.Mkdir("dir", 0o755); err != nil {
		t.Fatal(err)
	}
	if IsRegularFile("dir") {
		t.Error("expected false for directory")
	}
}
