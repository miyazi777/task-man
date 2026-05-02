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

	statuses := task.DefaultStatuses()
	in := []task.Task{
		{ID: 1, Title: "設計書を書く", StatusID: 1, Position: 1},
		{ID: 2, Title: "実装を進める", StatusID: 2, Position: 2},
		{ID: 3, Title: "仕様レビュー", StatusID: 3, Position: 3},
	}
	if err := repo.Save(in, statuses, AppConfig{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	outTasks, outStatuses, outCfg, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if outCfg.DataBaseDirectory != "" {
		t.Errorf("data_base_directory: got %q, want empty", outCfg.DataBaseDirectory)
	}
	if outCfg.Editor != "" {
		t.Errorf("editor: got %q, want empty", outCfg.Editor)
	}
	if len(outTasks) != len(in) {
		t.Fatalf("tasks len: got %d want %d", len(outTasks), len(in))
	}
	for i := range in {
		if in[i] != outTasks[i] {
			t.Errorf("tasks[%d]: got %+v want %+v", i, outTasks[i], in[i])
		}
	}
	if len(outStatuses) != len(statuses) {
		t.Fatalf("statuses len: got %d want %d", len(outStatuses), len(statuses))
	}
	for i := range statuses {
		if outStatuses[i] != statuses[i] {
			t.Errorf("statuses[%d]: got %+v want %+v", i, outStatuses[i], statuses[i])
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
	tasks, statuses, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected empty tasks, got %d items", len(tasks))
	}
	// statuses 欠落 → デフォルトが書き戻される
	if len(statuses) != 3 {
		t.Errorf("expected 3 default statuses, got %d", len(statuses))
	}
	// 再ロードで同じ statuses が返ること (= ファイルに書き戻されたこと)
	tasks2, statuses2, _, err := repo.Load()
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	if len(tasks2) != 0 {
		t.Errorf("tasks2 len: got %d, want 0", len(tasks2))
	}
	if len(statuses2) != 3 {
		t.Errorf("statuses2 len: got %d, want 3", len(statuses2))
	}
}

func TestYAMLStatusesEmptyArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses: []
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	_, statuses, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(statuses) != 3 {
		t.Errorf("expected 3 default statuses, got %d", len(statuses))
	}
}

func TestYAMLStatusAutoAssignID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      sequence: 1
      label: todo
      color: "#6c7086"
  - status:
      sequence: 2
      label: doing
      color: "#fab387"
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	_, statuses, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses[0].ID == 0 || statuses[1].ID == 0 {
		t.Errorf("ids should be auto-assigned, got %+v", statuses)
	}
	if statuses[0].ID == statuses[1].ID {
		t.Errorf("ids must be unique, got %d/%d", statuses[0].ID, statuses[1].ID)
	}
}

func TestYAMLStatusDuplicateID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: a
  - status:
      id: 1
      sequence: 2
      label: b
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for duplicated status id")
	}
}

func TestYAMLUnknownStatusID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: x
      status_id: 99
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for unknown status_id")
	}
}

func TestYAMLDuplicateTaskID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: a
      status_id: 1
  - task:
      id: 1
      title: b
      status_id: 1
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	_, _, _, err := repo.Load()
	if err == nil {
		t.Fatal("expected error for duplicated task id")
	}
	if !strings.Contains(err.Error(), "duplicated") {
		t.Errorf("expected 'duplicated' in error, got: %v", err)
	}
}

func TestYAMLMissingFile(t *testing.T) {
	repo := NewYAMLRepository(filepath.Join(t.TempDir(), "nope.yaml"))
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestYAMLDataBaseDirectoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	if err := repo.Save(nil, task.DefaultStatuses(), AppConfig{DataBaseDirectory: "./datas", Editor: "$EDITOR"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, _, cfg, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataBaseDirectory != "./datas" {
		t.Errorf("data_base_directory: got %q, want %q", cfg.DataBaseDirectory, "./datas")
	}
	if cfg.Editor != "$EDITOR" {
		t.Errorf("editor: got %q, want %q", cfg.Editor, "$EDITOR")
	}
}

func TestYAMLStatusCollapsedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)

	statuses := task.StatusList{
		{ID: 1, Sequence: 1, Label: "todo", Color: "#6c7086", Collapsed: false},
		{ID: 2, Sequence: 2, Label: "doing", Color: "#fab387", Collapsed: true},
		{ID: 3, Sequence: 3, Label: "done", Color: "#a6e3a1", Collapsed: true},
	}
	if err := repo.Save(nil, statuses, AppConfig{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, out, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out) != len(statuses) {
		t.Fatalf("statuses len: got %d, want %d", len(out), len(statuses))
	}
	for i := range statuses {
		if out[i].Collapsed != statuses[i].Collapsed {
			t.Errorf("statuses[%d].Collapsed: got %v, want %v", i, out[i].Collapsed, statuses[i].Collapsed)
		}
	}
}

func TestYAMLTaskCollapsedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	statuses := task.DefaultStatuses()
	in := []task.Task{
		{ID: 1, Title: "親", StatusID: 1, Collapsed: true},
		{ID: 2, Title: "子", StatusID: 1, ParentID: 1, Collapsed: false},
	}
	if err := repo.Save(in, statuses, AppConfig{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, _, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for i := range in {
		if out[i].Collapsed != in[i].Collapsed {
			t.Errorf("tasks[%d].Collapsed: got %v, want %v", i, out[i].Collapsed, in[i].Collapsed)
		}
	}
}

func TestYAMLSubtaskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	statuses := task.DefaultStatuses()
	in := []task.Task{
		{ID: 1, Title: "親", StatusID: 1, Position: 1},
		{ID: 2, Title: "子1", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "子2", StatusID: 1, ParentID: 1, Position: 2},
	}
	if err := repo.Save(in, statuses, AppConfig{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, _, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("tasks len: got %d want %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("tasks[%d]: got %+v want %+v", i, out[i], in[i])
		}
	}
}

func TestYAMLSubtaskUnknownParent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: c
      status_id: 1
      parent_id: 99
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for unknown parent_id")
	}
}

func TestYAMLSubtaskNestedAllowed(t *testing.T) {
	// MaxNestDepth=4 のとき depth 0..4 まで許容される (5 階層)。
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: l0
      status_id: 1
  - task:
      id: 2
      title: l1
      status_id: 1
      parent_id: 1
  - task:
      id: 3
      title: l2
      status_id: 1
      parent_id: 2
  - task:
      id: 4
      title: l3
      status_id: 1
      parent_id: 3
  - task:
      id: 5
      title: l4
      status_id: 1
      parent_id: 4
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err != nil {
		t.Errorf("unexpected error for 5-level nest: %v", err)
	}
}

func TestYAMLSubtaskDepthExceeded(t *testing.T) {
	// 6 階層は MaxNestDepth=4 を超えるためエラーになる。
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: l0
      status_id: 1
  - task:
      id: 2
      title: l1
      status_id: 1
      parent_id: 1
  - task:
      id: 3
      title: l2
      status_id: 1
      parent_id: 2
  - task:
      id: 4
      title: l3
      status_id: 1
      parent_id: 3
  - task:
      id: 5
      title: l4
      status_id: 1
      parent_id: 4
  - task:
      id: 6
      title: l5
      status_id: 1
      parent_id: 5
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for nesting depth exceeded")
	}
}

func TestYAMLPositionAutoAssign(t *testing.T) {
	// position 未指定のタスクが yaml 出現順に 1 から採番され、書き戻される。
	// 兄弟スコープごとに独立して採番される (ルートとサブタスクは別スコープ)。
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 10
      title: r1
      status_id: 1
  - task:
      id: 11
      title: r2
      status_id: 1
  - task:
      id: 12
      title: c1
      status_id: 1
      parent_id: 10
  - task:
      id: 13
      title: c2
      status_id: 1
      parent_id: 10
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	tasks, _, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[int]int{10: 1, 11: 2, 12: 1, 13: 2}
	for _, tk := range tasks {
		if tk.Position != want[tk.ID] {
			t.Errorf("task id=%d: got Position=%d want %d", tk.ID, tk.Position, want[tk.ID])
		}
	}

	// 再ロードで補完済みの値が保持されること (= 書き戻されたこと)。
	tasks2, _, _, err := repo.Load()
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	for _, tk := range tasks2 {
		if tk.Position != want[tk.ID] {
			t.Errorf("re-loaded task id=%d: got Position=%d want %d", tk.ID, tk.Position, want[tk.ID])
		}
	}
}

func TestYAMLPositionPartialAutoAssign(t *testing.T) {
	// 一部のみ position を持つ場合、未指定のタスクは max+1 から採番される。
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 1
      title: a
      status_id: 1
      position: 5
  - task:
      id: 2
      title: b
      status_id: 1
  - task:
      id: 3
      title: c
      status_id: 1
      position: 2
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	tasks, _, _, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[int]int{1: 5, 2: 6, 3: 2}
	for _, tk := range tasks {
		if tk.Position != want[tk.ID] {
			t.Errorf("task id=%d: got Position=%d want %d", tk.ID, tk.Position, want[tk.ID])
		}
	}
}

func TestYAMLZeroTaskID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks:
  - task:
      id: 0
      title: x
      status_id: 1
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, _, _, err := repo.Load(); err == nil {
		t.Error("expected error for task id <= 0")
	}
}
