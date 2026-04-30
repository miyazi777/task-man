package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTaskDataNoBase(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", "タスクA"); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "タスクA")
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		t.Errorf("task dir not created: %v", err)
	}
	memoPath := filepath.Join(taskDir, "memo.md")
	if info, err := os.Stat(memoPath); err != nil || info.IsDir() {
		t.Errorf("memo.md not created: %v", err)
	}
}

func TestCreateTaskDataWithBase(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "datas", "タスクB"); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "datas", "タスクB")
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		t.Errorf("task dir not created: %v", err)
	}
	memoPath := filepath.Join(taskDir, "memo.md")
	if info, err := os.Stat(memoPath); err != nil || info.IsDir() {
		t.Errorf("memo.md not created: %v", err)
	}
}

func TestCreateTaskDataConflict(t *testing.T) {
	yamlDir := t.TempDir()
	// 先に同名ディレクトリを作っておく
	taskDir := filepath.Join(yamlDir, "既存")
	if err := os.Mkdir(taskDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := CreateTaskData(yamlDir, "", "既存")
	if err == nil {
		t.Fatal("expected error on conflict")
	}
	if !errors.Is(err, ErrTaskDirExists) {
		t.Errorf("expected ErrTaskDirExists, got %v", err)
	}
	// memo.md は作られていないこと
	if _, err := os.Stat(filepath.Join(taskDir, "memo.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("memo.md should not exist on conflict, stat err=%v", err)
	}
}

func TestCreateTaskDataRelativeBase(t *testing.T) {
	yamlDir := t.TempDir()
	// "./datas" のような相対表記もサポート
	if err := CreateTaskData(yamlDir, "./datas", "タスクC"); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "datas", "タスクC")
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		t.Errorf("task dir not created: %v", err)
	}
}

func TestListTaskFiles(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", "タスクD"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "タスクD")
	// 追加ファイル + サブディレクトリ (除外されること)
	for _, name := range []string{"zzz.md", "aaa.txt", "bbb.md"} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte{}, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(taskDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files, err := ListTaskFiles(yamlDir, "", "タスクD")
	if err != nil {
		t.Fatalf("ListTaskFiles: %v", err)
	}
	want := []string{"aaa.txt", "bbb.md", "memo.md", "zzz.md"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, files[i], want[i])
		}
	}
}

func TestListTaskFilesMissingDir(t *testing.T) {
	files, err := ListTaskFiles(t.TempDir(), "", "存在しないタスク")
	if err != nil {
		t.Fatalf("ListTaskFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestValidateFileName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"normal", "memo.md", false},
		{"japanese", "メモ.md", false},
		{"empty", "", true},
		{"slash", "a/b.md", true},
		{"null", "a\x00b.md", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateFileName(c.input)
			if c.wantErr && err == nil {
				t.Errorf("expected error for %q", c.input)
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", c.input, err)
			}
		})
	}
}

func TestValidateFileNameCharsAllowsEmpty(t *testing.T) {
	if err := ValidateFileNameChars(""); err != nil {
		t.Errorf("empty should be allowed for live validation: %v", err)
	}
}

func TestCreateFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", "タスクE", "report.md"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	full := filepath.Join(yamlDir, "タスクE", "report.md")
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		t.Errorf("file not created: %v", err)
	}
}

func TestCreateFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", "タスクF", "x.md"); err != nil {
		t.Fatalf("first CreateFile: %v", err)
	}
	err := CreateFile(yamlDir, "", "タスクF", "x.md")
	if err == nil {
		t.Fatal("expected error on conflict")
	}
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", "タスクG", "old.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := RenameFile(yamlDir, "", "タスクG", "old.md", "new.md"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "タスクG", "new.md")); err != nil {
		t.Errorf("new path not present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "タスクG", "old.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old path should be gone: %v", err)
	}
}

func TestRenameFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", "タスクH", "a.md"); err != nil {
		t.Fatalf("setup1: %v", err)
	}
	if err := CreateFile(yamlDir, "", "タスクH", "b.md"); err != nil {
		t.Fatalf("setup2: %v", err)
	}
	err := RenameFile(yamlDir, "", "タスクH", "a.md", "b.md")
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFileMissingSource(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", "タスクI"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := RenameFile(yamlDir, "", "タスクI", "ghost.md", "new.md")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", "タスクJ", "x.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := DeleteFile(yamlDir, "", "タスクJ", "x.md"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "タスクJ", "x.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should be gone: %v", err)
	}
}

func TestDeleteFileMissing(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", "タスクK"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := DeleteFile(yamlDir, "", "タスクK", "ghost.md")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}
