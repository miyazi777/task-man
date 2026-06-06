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

func TestListTaskFileTree(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 5); err != nil {
		t.Fatalf("setup: %v", err)
	}
	taskDir := filepath.Join(yamlDir, "task-5")
	// トップレベル: zzz.md, aaa.txt, bbb.md (memo.md は CreateTaskData が作成済)
	for _, name := range []string{"zzz.md", "aaa.txt", "bbb.md"} {
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte{}, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// サブディレクトリ: subdir/ と subdir/inner.md, subdir/empty/
	if err := os.Mkdir(filepath.Join(taskDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "subdir", "inner.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write inner.md: %v", err)
	}
	if err := os.Mkdir(filepath.Join(taskDir, "subdir", "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}

	tree, err := ListTaskFileTree(yamlDir, "", 5)
	if err != nil {
		t.Fatalf("ListTaskFileTree: %v", err)
	}
	// トップレベルは aaa.txt, bbb.md, memo.md, subdir, zzz.md の順 (Name 昇順)。
	wantTopNames := []string{"aaa.txt", "bbb.md", "memo.md", "subdir", "zzz.md"}
	if len(tree) != len(wantTopNames) {
		t.Fatalf("top entries: got %d, want %d (%v)", len(tree), len(wantTopNames), tree)
	}
	for i, want := range wantTopNames {
		if tree[i].Name != want {
			t.Errorf("top[%d].Name: got %q, want %q", i, tree[i].Name, want)
		}
		if tree[i].RelPath != want {
			t.Errorf("top[%d].RelPath: got %q, want %q", i, tree[i].RelPath, want)
		}
	}
	// subdir エントリの IsDir / Children 検証
	var sub FileEntry
	for _, e := range tree {
		if e.Name == "subdir" {
			sub = e
		}
	}
	if !sub.IsDir {
		t.Fatalf("subdir.IsDir = false, want true")
	}
	if len(sub.Children) != 2 {
		t.Fatalf("subdir children: got %d, want 2 (%v)", len(sub.Children), sub.Children)
	}
	// 子は empty (dir), inner.md (file) の順
	if sub.Children[0].Name != "empty" || !sub.Children[0].IsDir || sub.Children[0].RelPath != "subdir/empty" {
		t.Errorf("subdir/empty: %+v", sub.Children[0])
	}
	if sub.Children[1].Name != "inner.md" || sub.Children[1].IsDir || sub.Children[1].RelPath != "subdir/inner.md" {
		t.Errorf("subdir/inner.md: %+v", sub.Children[1])
	}
}

func TestListTaskFileTreeMissingDir(t *testing.T) {
	tree, err := ListTaskFileTree(t.TempDir(), "", 999)
	if err != nil {
		t.Fatalf("ListTaskFileTree: %v", err)
	}
	if len(tree) != 0 {
		t.Errorf("expected empty, got %v", tree)
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

func TestResolveTaskRelPathRejectsEscape(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "task-1")
	cases := []string{
		"../escape",
		"sub/../../escape",
		"/abs/path",
		"..",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := resolveTaskRelPath(taskDir, c); !errors.Is(err, ErrInvalidRelPath) {
				t.Errorf("resolveTaskRelPath(%q) err = %v, want ErrInvalidRelPath", c, err)
			}
		})
	}
}

func TestResolveTaskRelPathAcceptsNormal(t *testing.T) {
	taskDir := filepath.Join(t.TempDir(), "task-1")
	cases := map[string]string{
		"":             taskDir,
		".":            taskDir,
		"foo.md":       filepath.Join(taskDir, "foo.md"),
		"sub/foo.md":   filepath.Join(taskDir, "sub", "foo.md"),
		"./sub/foo.md": filepath.Join(taskDir, "sub", "foo.md"),
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := resolveTaskRelPath(taskDir, in)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestCreateFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 6, "", "report.md"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	full := filepath.Join(yamlDir, "task-6", "report.md")
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		t.Errorf("file not created: %v", err)
	}
}

func TestCreateFileInSubDir(t *testing.T) {
	yamlDir := t.TempDir()
	// サブディレクトリは事前に存在しなくても CreateFile が作る。
	if err := CreateFile(yamlDir, "", 6, "sub", "report.md"); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	full := filepath.Join(yamlDir, "task-6", "sub", "report.md")
	if _, err := os.Stat(full); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestCreateFileRejectsEscape(t *testing.T) {
	yamlDir := t.TempDir()
	err := CreateFile(yamlDir, "", 6, "../escape", "x.md")
	if !errors.Is(err, ErrInvalidRelPath) {
		t.Errorf("expected ErrInvalidRelPath, got %v", err)
	}
}

func TestCreateFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 7, "", "x.md"); err != nil {
		t.Fatalf("first CreateFile: %v", err)
	}
	err := CreateFile(yamlDir, "", 7, "", "x.md")
	if err == nil {
		t.Fatal("expected error on conflict")
	}
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 8, "", "old.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := RenameFile(yamlDir, "", 8, "", "old.md", "new.md"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-8", "new.md")); err != nil {
		t.Errorf("new path not present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-8", "old.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old path should be gone: %v", err)
	}
}

func TestRenameFileInSubDir(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 8, "sub", "old.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := RenameFile(yamlDir, "", 8, "sub", "old.md", "new.md"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-8", "sub", "new.md")); err != nil {
		t.Errorf("new path not present: %v", err)
	}
}

func TestRenameFileConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 9, "", "a.md"); err != nil {
		t.Fatalf("setup1: %v", err)
	}
	if err := CreateFile(yamlDir, "", 9, "", "b.md"); err != nil {
		t.Fatalf("setup2: %v", err)
	}
	err := RenameFile(yamlDir, "", 9, "", "a.md", "b.md")
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestRenameFileMissingSource(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 10); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := RenameFile(yamlDir, "", 10, "", "ghost.md", "new.md")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 11, "", "x.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := DeleteFile(yamlDir, "", 11, "x.md"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-11", "x.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should be gone: %v", err)
	}
}

func TestDeleteFileInSubDir(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 11, "sub", "x.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := DeleteFile(yamlDir, "", 11, "sub/x.md"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-11", "sub", "x.md")); !errors.Is(err, os.ErrNotExist) {
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

func TestDeleteFileRejectsDirectory(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 13, "sub", "memo.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := DeleteFile(yamlDir, "", 13, "sub")
	if err == nil {
		t.Fatal("expected error when deleting a directory")
	}
}

func TestCreateDir(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateDir(yamlDir, "", 14, "", "new-dir"); err != nil {
		t.Fatalf("CreateDir: %v", err)
	}
	full := filepath.Join(yamlDir, "task-14", "new-dir")
	info, err := os.Stat(full)
	if err != nil || !info.IsDir() {
		t.Errorf("directory not created: err=%v info=%+v", err, info)
	}
}

func TestCreateDirNested(t *testing.T) {
	yamlDir := t.TempDir()
	// parent ディレクトリが無くても作る。
	if err := CreateDir(yamlDir, "", 14, "parent", "child"); err != nil {
		t.Fatalf("CreateDir: %v", err)
	}
	full := filepath.Join(yamlDir, "task-14", "parent", "child")
	if info, err := os.Stat(full); err != nil || !info.IsDir() {
		t.Errorf("nested dir not created: err=%v", err)
	}
}

func TestCreateDirConflict(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateDir(yamlDir, "", 15, "", "dir"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := CreateDir(yamlDir, "", 15, "", "dir")
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists, got %v", err)
	}
}

func TestCreateDirConflictWithFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 15, "", "name"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := CreateDir(yamlDir, "", 15, "", "name")
	if !errors.Is(err, ErrFileExists) {
		t.Errorf("expected ErrFileExists when file with same name exists, got %v", err)
	}
}

func TestCreateDirRejectsEscape(t *testing.T) {
	yamlDir := t.TempDir()
	err := CreateDir(yamlDir, "", 15, "../escape", "name")
	if !errors.Is(err, ErrInvalidRelPath) {
		t.Errorf("expected ErrInvalidRelPath, got %v", err)
	}
}

func TestDeleteDir(t *testing.T) {
	yamlDir := t.TempDir()
	// dir に中身ありで作って、再帰削除できることを確認。
	if err := CreateFile(yamlDir, "", 16, "sub", "memo.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := DeleteDir(yamlDir, "", 16, "sub"); err != nil {
		t.Fatalf("DeleteDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-16", "sub")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("sub should be gone: %v", err)
	}
}

func TestDeleteDirRejectsRoot(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 16); err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, p := range []string{"", "."} {
		if err := DeleteDir(yamlDir, "", 16, p); err == nil {
			t.Errorf("expected error for root delete (%q)", p)
		}
	}
	if _, err := os.Stat(TaskDir(yamlDir, "", 16)); err != nil {
		t.Errorf("task root should still exist: %v", err)
	}
}

func TestDeleteDirRejectsFile(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateFile(yamlDir, "", 16, "", "file.md"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := DeleteDir(yamlDir, "", 16, "file.md")
	if err == nil {
		t.Fatal("expected error when deleting a regular file via DeleteDir")
	}
}

func TestDeleteDirMissing(t *testing.T) {
	yamlDir := t.TempDir()
	if err := CreateTaskData(yamlDir, "", 17); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := DeleteDir(yamlDir, "", 17, "ghost")
	if !errors.Is(err, ErrFileNotFoundIn) {
		t.Errorf("expected ErrFileNotFoundIn, got %v", err)
	}
}

func TestRenameDirectory(t *testing.T) {
	// RenameFile はディレクトリにも適用できる (os.Rename 経由)。
	yamlDir := t.TempDir()
	if err := CreateDir(yamlDir, "", 18, "", "old-dir"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := CreateFile(yamlDir, "", 18, "old-dir", "inner.md"); err != nil {
		t.Fatalf("setup inner: %v", err)
	}
	if err := RenameFile(yamlDir, "", 18, "", "old-dir", "new-dir"); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(yamlDir, "task-18", "new-dir", "inner.md")); err != nil {
		t.Errorf("renamed dir should keep content: %v", err)
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

// setupListDirChildrenFixture は ListTaskDirChildren テスト用の共通フィクスチャを
// 構築する。task-1 配下に複数ファイルと subdir/ を作成する。
func setupListDirChildrenFixture(t *testing.T) string {
	t.Helper()
	yamlDir := t.TempDir()
	taskDir := filepath.Join(yamlDir, "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	for _, name := range []string{"aaa.txt", "bbb.md", "zzz.md", "memo.md"} {
		if err := os.WriteFile(filepath.Join(taskDir, name), nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	subDir := filepath.Join(taskDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inner.md"), nil, 0o644); err != nil {
		t.Fatalf("write inner: %v", err)
	}
	emptyDir := filepath.Join(taskDir, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	return yamlDir
}

func TestListTaskDirChildrenTopLevel(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	got, err := ListTaskDirChildren(yamlDir, "", 1, "")
	if err != nil {
		t.Fatalf("ListTaskDirChildren: %v", err)
	}
	want := []FileEntry{
		{Name: "aaa.txt", RelPath: "aaa.txt", IsDir: false},
		{Name: "bbb.md", RelPath: "bbb.md", IsDir: false},
		{Name: "empty", RelPath: "empty", IsDir: true},
		{Name: "memo.md", RelPath: "memo.md", IsDir: false},
		{Name: "subdir", RelPath: "subdir", IsDir: true},
		{Name: "zzz.md", RelPath: "zzz.md", IsDir: false},
	}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].RelPath != want[i].RelPath || got[i].IsDir != want[i].IsDir {
			t.Errorf("entry[%d]: got %+v, want %+v", i, got[i], want[i])
		}
		if got[i].Children != nil {
			t.Errorf("entry[%d].Children should be nil (shallow), got %v", i, got[i].Children)
		}
	}
}

func TestListTaskDirChildrenSubDir(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	got, err := ListTaskDirChildren(yamlDir, "", 1, "subdir")
	if err != nil {
		t.Fatalf("ListTaskDirChildren: %v", err)
	}
	if len(got) != 1 || got[0].Name != "inner.md" || got[0].RelPath != "subdir/inner.md" || got[0].IsDir {
		t.Errorf("got %+v, want one entry inner.md", got)
	}
}

func TestListTaskDirChildrenDotAndEmptyEquivalent(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	a, errA := ListTaskDirChildren(yamlDir, "", 1, "")
	b, errB := ListTaskDirChildren(yamlDir, "", 1, ".")
	if errA != nil || errB != nil {
		t.Fatalf("err A=%v B=%v", errA, errB)
	}
	if len(a) != len(b) {
		t.Fatalf("len mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].RelPath != b[i].RelPath || a[i].IsDir != b[i].IsDir {
			t.Errorf("entry[%d]: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestListTaskDirChildrenEmptyDir(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	got, err := ListTaskDirChildren(yamlDir, "", 1, "empty")
	if err != nil {
		t.Fatalf("ListTaskDirChildren: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestListTaskDirChildrenNotADirectory(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	_, err := ListTaskDirChildren(yamlDir, "", 1, "memo.md")
	if !errors.Is(err, ErrNotADirectory) {
		t.Errorf("expected ErrNotADirectory, got %v", err)
	}
}

func TestListTaskDirChildrenNotFound(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	_, err := ListTaskDirChildren(yamlDir, "", 1, "does-not-exist")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestListTaskDirChildrenInvalidRelPath(t *testing.T) {
	yamlDir := setupListDirChildrenFixture(t)
	_, err := ListTaskDirChildren(yamlDir, "", 1, "../escape")
	if !errors.Is(err, ErrInvalidRelPath) {
		t.Errorf("expected ErrInvalidRelPath, got %v", err)
	}
}
