package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDefault(t *testing.T) {
	a, err := Parse([]string{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.Path != DefaultFileName {
		t.Errorf("Path: got %q want %q", a.Path, DefaultFileName)
	}
	if a.MustExist {
		t.Error("MustExist: expected false for default")
	}
}

func TestParseWithTaskFlag(t *testing.T) {
	for _, args := range [][]string{
		{"-t", "/tmp/foo.yaml"},
		{"--tasks", "/tmp/foo.yaml"},
	} {
		a, err := Parse(args)
		if err != nil {
			t.Fatalf("Parse(%v): %v", args, err)
		}
		if a.Path != "/tmp/foo.yaml" {
			t.Errorf("Path: got %q want %q", a.Path, "/tmp/foo.yaml")
		}
		if !a.MustExist {
			t.Errorf("MustExist: expected true when flag given")
		}
	}
}

// "-t '~/...'" のようにシェルに展開されないクオート付きパスでも、
// task-man 側で先頭 ~/ をホームディレクトリへ置換する。
func TestParseExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	a, err := Parse([]string{"-t", "~/private/tasks.yaml"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := filepath.Join(home, "private/tasks.yaml")
	if a.Path != want {
		t.Errorf("Path: got %q want %q", a.Path, want)
	}
}

func TestEnsureFileMustExistMissing(t *testing.T) {
	dir := t.TempDir()
	a := &Args{Path: filepath.Join(dir, "nope.yaml"), MustExist: true}
	if err := EnsureFile(a); err == nil {
		t.Error("expected error")
	}
}

func TestEnsureFileCreatesWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	a := &Args{Path: path, MustExist: false}
	if err := EnsureFile(a); err != nil {
		t.Fatalf("EnsureFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestEnsureFileExistingOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(path, []byte("tasks: []\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	a := &Args{Path: path, MustExist: true}
	if err := EnsureFile(a); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
