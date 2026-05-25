package tui

import (
	"errors"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

// nopRepo は handleKey 経由のテストで Save が呼ばれた場合に no-op で済ませるための
// 最小実装。pathForCopy + copy 経路では Save に到達しないが、interface 充足のため用意。
type nopRepo struct{}

func (nopRepo) Load() (storage.LoadResult, error) { return storage.LoadResult{}, nil }
func (nopRepo) Save(storage.LoadResult) error     { return nil }

// withFakeClipboard は copyToClipboard を差し替えて、コピー文字列を捕捉する。
// 戻り値の cleanup を defer 呼び出しすると元に戻る。
func withFakeClipboard(t *testing.T, target *string, retErr error) func() {
	t.Helper()
	orig := copyToClipboard
	copyToClipboard = func(s string) error {
		*target = s
		return retErr
	}
	return func() { copyToClipboard = orig }
}

// buildModelForPathTest は path 計算に必要最低限の状態を持つ Model を組み立てる。
func buildModelForPathTest(yamlDir, dataBaseDir string, tasks []task.Task, statuses task.StatusList) Model {
	m := Model{
		repo:     nopRepo{},
		tasks:    tasks,
		statuses: statuses,
		yamlDir:  yamlDir,
		cfg:      storage.AppConfig{DataBaseDirectory: dataBaseDir},
		mode:     ModeList,
		keys:     newKeyMap(),
	}
	m = m.withRowsRebuilt()
	m = m.withDetailRowsRebuilt()
	if first := firstNavigable(m.rows); first >= 0 {
		m.cursor = first
	}
	return m
}

// TestPathForCopyListModeTaskRow はタスクリストでカーソルがタスク行にあるとき、
// タスクのデータディレクトリ絶対パスが返ることを検証する。
func TestPathForCopyListModeTaskRow(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"data",
		[]task.Task{{ID: 7, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	// カーソルを task 行に合わせる
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}
	got, ok := m.pathForCopy()
	if !ok {
		t.Fatalf("pathForCopy returned ok=false")
	}
	want := filepath.Join("/tmp/wsp", "data", "task-7")
	if got != want {
		t.Errorf("pathForCopy = %q, want %q", got, want)
	}
}

// TestPathForCopyListModeStatusRow はタスクリストでカーソルがステータス行 (タスクなし) のとき
// ok=false が返ることを検証する。
func TestPathForCopyListModeStatusRow(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"data",
		nil,
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	if _, ok := m.pathForCopy(); ok {
		t.Errorf("expected ok=false when no task under cursor")
	}
}

// TestPathForCopyDetailModeFilesRow は詳細画面の Files 行でカーソル位置のファイル
// 絶対パス (タスクディレクトリ + relPath) が返ることを検証する。
func TestPathForCopyDetailModeFilesRow(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"",
		[]task.Task{{ID: 3, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	// 詳細画面に入る前にタスク行へ cursor を進める (currentTask は m.rows[m.cursor] を見る)
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}
	// 詳細画面 + Files 行へ
	m.mode = ModeDetail
	m.detailCursor = len(m.detailRows) - 1
	if m.detailRows[m.detailCursor].kind != detailRowFiles {
		t.Fatalf("expected last detail row to be Files")
	}
	m.files = []fileRow{
		{name: "memo.md", relPath: "memo.md"},
		{name: "sub", relPath: "sub", isDir: true, hasChildren: true},
		{name: "note.md", relPath: "sub/note.md", depth: 1},
	}
	m.filesTaskID = 3
	m.fileCursor = 2

	got, ok := m.pathForCopy()
	if !ok {
		t.Fatalf("pathForCopy returned ok=false")
	}
	want := filepath.Join("/tmp/wsp", "task-3", "sub", "note.md")
	if got != want {
		t.Errorf("pathForCopy = %q, want %q", got, want)
	}
}

// TestPathForCopyDetailModeNonFilesRow は詳細画面の Files 以外の行 (Title 等) の
// ときタスクディレクトリが返ることを検証する。
func TestPathForCopyDetailModeNonFilesRow(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"data",
		[]task.Task{{ID: 9, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}
	m.mode = ModeDetail
	m.detailCursor = 0 // Title 行

	got, ok := m.pathForCopy()
	if !ok {
		t.Fatalf("pathForCopy returned ok=false")
	}
	want := filepath.Join("/tmp/wsp", "data", "task-9")
	if got != want {
		t.Errorf("pathForCopy = %q, want %q", got, want)
	}
}

// TestHandleKeyCopyPathListMode は ModeList で p を押した時に copyToClipboard が
// 期待文字列で呼ばれることを検証する。
func TestHandleKeyCopyPathListMode(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"data",
		[]task.Task{{ID: 11, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}

	var got string
	cleanup := withFakeClipboard(t, &got, nil)
	defer cleanup()

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	res := out.(Model)

	want := filepath.Join("/tmp/wsp", "data", "task-11")
	if got != want {
		t.Errorf("clipboard got %q, want %q", got, want)
	}
	if res.saveErr != nil {
		t.Errorf("unexpected saveErr: %v", res.saveErr)
	}
}

// TestHandleKeyCopyPathClipboardError は copy 失敗時に saveErr に伝播することを検証する。
func TestHandleKeyCopyPathClipboardError(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"data",
		[]task.Task{{ID: 12, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}

	var got string
	wantErr := errors.New("boom")
	cleanup := withFakeClipboard(t, &got, wantErr)
	defer cleanup()

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	res := out.(Model)
	if !errors.Is(res.saveErr, wantErr) {
		t.Errorf("saveErr = %v, want %v", res.saveErr, wantErr)
	}
}

// TestHandleKeyCopyPathDetailFilesRow は ModeDetail かつ Files 行カーソル時、
// ファイルの絶対パスがコピーされることを検証する。
func TestHandleKeyCopyPathDetailFilesRow(t *testing.T) {
	m := buildModelForPathTest(
		"/tmp/wsp",
		"",
		[]task.Task{{ID: 4, Title: "a", StatusID: 1, Position: 1}},
		task.StatusList{{ID: 1, Sequence: 1, Label: "todo"}},
	)
	for i, r := range m.rows {
		if r.kind == rowTask {
			m.cursor = i
			break
		}
	}
	m.mode = ModeDetail
	m.detailCursor = len(m.detailRows) - 1
	m.files = []fileRow{{name: "memo.md", relPath: "memo.md"}}
	m.filesTaskID = 4
	m.fileCursor = 0

	var got string
	cleanup := withFakeClipboard(t, &got, nil)
	defer cleanup()

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	res := out.(Model)
	want := filepath.Join("/tmp/wsp", "task-4", "memo.md")
	if got != want {
		t.Errorf("clipboard got %q, want %q", got, want)
	}
	if res.saveErr != nil {
		t.Errorf("unexpected saveErr: %v", res.saveErr)
	}
}
