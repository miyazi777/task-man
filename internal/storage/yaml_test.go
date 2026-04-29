package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miyazi777/task-man/internal/task"
)

func TestYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)

	in := []task.Task{
		{ID: 1, Title: "設計書を書く", Status: task.StatusTodo},
		{ID: 2, Title: "実装を進める", Status: task.StatusDoing},
		{ID: 3, Title: "仕様レビュー", Status: task.StatusDone},
	}
	if err := repo.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("len: got %d want %d", len(out), len(in))
	}
	for i := range in {
		if in[i] != out[i] {
			t.Errorf("[%d]: got %+v want %+v", i, out[i], in[i])
		}
	}
}

func TestYAMLEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	out, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty, got %d items", len(out))
	}
}

func TestYAMLInvalidStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `tasks:
  - task:
      id: 1
      title: x
      status: bogus
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestYAMLDuplicateID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `tasks:
  - task:
      id: 1
      title: a
      status: todo
  - task:
      id: 1
      title: b
      status: doing
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	_, err := repo.Load()
	if err == nil {
		t.Fatal("expected error for duplicated id")
	}
	if !strings.Contains(err.Error(), "duplicated") {
		t.Errorf("expected 'duplicated' in error, got: %v", err)
	}
}

func TestYAMLMissingFile(t *testing.T) {
	repo := NewYAMLRepository(filepath.Join(t.TempDir(), "nope.yaml"))
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestYAMLZeroID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `tasks:
  - task:
      id: 0
      title: x
      status: todo
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for id <= 0")
	}
}
