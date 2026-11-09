package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samznd/layrgo/internal/fileutil"
)

func TestCreateFile_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "a", "b", "c", "file.go")
	if err := fileutil.CreateFile(path, "package main\n"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package main\n" {
		t.Errorf("content = %q", got)
	}
}

func TestCreateFile_NoParentDir(t *testing.T) {
	t.Parallel()

	// A bare filename has no directory component; filepath.Dir returns ".",
	// which must not be treated as an error. The hand-rolled dirOf this
	// replaced returned "." too, but via a manual separator scan.
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := fileutil.CreateFile("bare.txt", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bare.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestCreateFile_Overwrites(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "f.txt")
	if err := fileutil.CreateFile(path, "first"); err != nil {
		t.Fatal(err)
	}
	if err := fileutil.CreateFile(path, "second"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Errorf("content = %q, want %q", got, "second")
	}
}

func TestCreateFile_ErrorsWhenParentIsAFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// "blocker" is a file, so it cannot also be a parent directory.
	if err := fileutil.CreateFile(filepath.Join(blocker, "child.txt"), "y"); err == nil {
		t.Fatal("expected an error when the parent path is a file")
	}
}
