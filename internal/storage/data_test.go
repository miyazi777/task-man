package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTaskDataNoBase(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 1); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "task-1")
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
	if err := CreateTaskData(yamlDir, "datas", 2); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "datas", "task-2")
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
	taskDir := filepath.Join(yamlDir, "task-3")
	if err := os.Mkdir(taskDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := CreateTaskData(yamlDir, "", 3)
	if err == nil {
		t.Fatal("expected error on conflict")
	}
	if !errors.Is(err, ErrTaskDirExists) {
		t.Errorf("expected ErrTaskDirExists, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(taskDir, "memo.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("memo.md should not exist on conflict, stat err=%v", err)
	}
}

func TestCreateTaskDataRelativeBase(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "./datas", 4); err != nil {
		t.Fatalf("CreateTaskData: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "datas", "task-4")
	if info, err := os.Stat(taskDir); err != nil || !info.IsDir() {
		t.Errorf("task dir not created: %v", err)
	}
}

func TestListTaskFiles(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 5); err != nil {
		t.Fatalf("setup: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "task-5")
	for _, name := range []string{"zzz.md", "aaa.txt", "bbb.md"} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte{}, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(taskDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files, err := ListTaskFiles(yamlDir, "", 5)
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
	files, err := ListTaskFiles(t.TempDir(), "", 999)
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
	if err := CreateFile(yamlDir, "", 6, "report.md"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	full := filepath.Join(yamlDir, "task-6", "report.md")
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		t.Errorf("file not created: %v", err)
	}
}

func TestCreateFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 7, "x.md"); err != nil {
		t.Fatalf("first CreateFile: %v", err)
	}
	err := CreateFile(yamlDir, "", 7, "x.md")
	if err == nil {
		t.Fatal("expected error on conflict")
	}
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 8, "old.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := RenameFile(yamlDir, "", 8, "old.md", "new.md"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-8", "new.md")); err != nil {
		t.Errorf("new path not present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-8", "old.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old path should be gone: %v", err)
	}
}

func TestRenameFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 9, "a.md"); err != nil {
		t.Fatalf("setup1: %v", err)
	}
	if err := CreateFile(yamlDir, "", 9, "b.md"); err != nil {
		t.Fatalf("setup2: %v", err)
	}
	err := RenameFile(yamlDir, "", 9, "a.md", "b.md")
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFileMissingSource(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 10); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := RenameFile(yamlDir, "", 10, "ghost.md", "new.md")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 11, "x.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := DeleteFile(yamlDir, "", 11, "x.md"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-11", "x.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should be gone: %v", err)
	}
}

func TestDeleteFileMissing(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 12); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := DeleteFile(yamlDir, "", 12, "ghost.md")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}

// RemoveAllTaskData は task-<int> 命名のディレクトリだけを再帰的に削除し、
// 命名規則に合わない兄弟 (notes/, tasks.yaml など) には触れない。
func TestRemoveAllTaskData(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "datas", 1); err != nil {
		t.Fatalf("setup task-1: %v", err)
	}
	if err := CreateTaskData(yamlDir, "datas", 2); err != nil {
		t.Fatalf("setup task-2: %v", err)
	}
	// 命名規則外のディレクトリ・ファイル (掃除対象外)
	preserveDir := filepath.Join(yamlDir, "datas", "notes")
	if err := os.MkdirAll(preserveDir, 0o755); err != nil {
		t.Fatalf("setup preserve dir: %v", err)
	}
	preserveFile := filepath.Join(yamlDir, "datas", "README.md")
	if err := os.WriteFile(preserveFile, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("setup preserve file: %v", err)
	}

	removed, err := RemoveAllTaskData(yamlDir, "datas")
	if err != nil {
		t.Fatalf("RemoveAllTaskData: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("removed: got %d, want 2 (%v)", len(removed), removed)
	}

	for _, id := range []int{1, 2} {
		td := TaskDir(yamlDir, "datas", id)
		if _, err := os.Stat(td); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("task-%d still exists: err=%v", id, err)
		}
	}
	if _, err := os.Stat(preserveDir); err != nil {
		t.Errorf("preserve dir lost: %v", err)
	}
	if _, err := os.Stat(preserveFile); err != nil {
		t.Errorf("preserve file lost: %v", err)
	}
}

// RemoveAllTaskData はベースディレクトリが存在しない場合でもエラーにせず、
// 削除リストは空で返す (--init 直後の不在状態をそのまま許容するため)。
func TestRemoveAllTaskDataMissingRoot(t *testing.T) {
	yamlDir := t.TempDir()
	removed, err := RemoveAllTaskData(yamlDir, "does-not-exist")
	if err != nil {
		t.Fatalf("RemoveAllTaskData: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed: got %v, want empty", removed)
	}
}
