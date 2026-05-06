package tui

import (
	"errors"
	"testing"
)

func TestBuildEditorCmdLiteral(t *testing.T) {
	cmd, err := buildAppCmd("vim", "/tmp/foo.md")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cmd.Path == "" {
		t.Errorf("cmd.Path empty")
	}
	if got := cmd.Args; len(got) != 2 || got[0] != "vim" || got[1] != "/tmp/foo.md" {
		t.Errorf("args: %v, want [vim /tmp/foo.md]", got)
	}
}

func TestBuildEditorCmdWithArgs(t *testing.T) {
	cmd, err := buildAppCmd("nvim --noplugin", "/tmp/x.md")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := cmd.Args; len(got) != 3 || got[0] != "nvim" || got[1] != "--noplugin" || got[2] != "/tmp/x.md" {
		t.Errorf("args: %v, want [nvim --noplugin /tmp/x.md]", got)
	}
}

func TestBuildEditorCmdEnvVar(t *testing.T) {
	t.Setenv("EDITOR", "vi")
	cmd, err := buildAppCmd("$EDITOR", "/tmp/x.md")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := cmd.Args; len(got) != 2 || got[0] != "vi" || got[1] != "/tmp/x.md" {
		t.Errorf("args: %v, want [vi /tmp/x.md]", got)
	}
}

func TestBuildEditorCmdFallbackToEnv(t *testing.T) {
	t.Setenv("EDITOR", "code")
	cmd, err := buildAppCmd("", "/tmp/x.md")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := cmd.Args; len(got) != 2 || got[0] != "code" {
		t.Errorf("args: %v, want first=code", got)
	}
}

func TestBuildEditorCmdNoneConfigured(t *testing.T) {
	t.Setenv("EDITOR", "")
	_, err := buildAppCmd("", "/tmp/x.md")
	if err == nil {
		t.Fatal("expected error when neither yaml nor env is set")
	}
	if !errors.Is(err, ErrEditorNotConfigured) {
		t.Errorf("expected ErrEditorNotConfigured, got %v", err)
	}
}
