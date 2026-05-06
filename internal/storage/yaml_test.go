package storage

import (
	"os"
	"path/filepath"
	"reflect"
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
	if err := repo.Save(LoadResult{Tasks: in, Statuses: statuses}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lr.Config.DataBaseDirectory != "" {
		t.Errorf("data_base_directory: got %q, want empty", lr.Config.DataBaseDirectory)
	}
	if lr.Config.Editor != "" {
		t.Errorf("editor: got %q, want empty", lr.Config.Editor)
	}
	if len(lr.Tasks) != len(in) {
		t.Fatalf("tasks len: got %d want %d", len(lr.Tasks), len(in))
	}
	for i := range in {
		if !reflect.DeepEqual(in[i], lr.Tasks[i]) {
			t.Errorf("tasks[%d]: got %+v want %+v", i, lr.Tasks[i], in[i])
		}
	}
	if len(lr.Statuses) != len(statuses) {
		t.Fatalf("statuses len: got %d want %d", len(lr.Statuses), len(statuses))
	}
	for i := range statuses {
		if lr.Statuses[i] != statuses[i] {
			t.Errorf("statuses[%d]: got %+v want %+v", i, lr.Statuses[i], statuses[i])
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
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Tasks) != 0 {
		t.Errorf("expected empty tasks, got %d items", len(lr.Tasks))
	}
	if len(lr.Statuses) != 3 {
		t.Errorf("expected 3 default statuses, got %d", len(lr.Statuses))
	}
	lr2, err := repo.Load()
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	if len(lr2.Tasks) != 0 {
		t.Errorf("tasks2 len: got %d, want 0", len(lr2.Tasks))
	}
	if len(lr2.Statuses) != 3 {
		t.Errorf("statuses2 len: got %d, want 3", len(lr2.Statuses))
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
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Statuses) != 3 {
		t.Errorf("expected 3 default statuses, got %d", len(lr.Statuses))
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
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(lr.Statuses))
	}
	if lr.Statuses[0].ID == 0 || lr.Statuses[1].ID == 0 {
		t.Errorf("ids should be auto-assigned, got %+v", lr.Statuses)
	}
	if lr.Statuses[0].ID == lr.Statuses[1].ID {
		t.Errorf("ids must be unique, got %d/%d", lr.Statuses[0].ID, lr.Statuses[1].ID)
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
	if _, err := repo.Load(); err == nil {
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
	if _, err := repo.Load(); err == nil {
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
	_, err := repo.Load()
	if err == nil {
		t.Fatal("expected error for duplicated task id")
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

func TestYAMLDataBaseDirectoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	if err := repo.Save(LoadResult{
		Statuses: task.DefaultStatuses(),
		Config:   AppConfig{DataBaseDirectory: "./datas", Editor: "$EDITOR"},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lr.Config.DataBaseDirectory != "./datas" {
		t.Errorf("data_base_directory: got %q, want %q", lr.Config.DataBaseDirectory, "./datas")
	}
	if lr.Config.Editor != "$EDITOR" {
		t.Errorf("editor: got %q, want %q", lr.Config.Editor, "$EDITOR")
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
	if err := repo.Save(LoadResult{Statuses: statuses}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	out := lr.Statuses
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
	if err := repo.Save(LoadResult{Tasks: in, Statuses: statuses}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	out := lr.Tasks
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
	if err := repo.Save(LoadResult{Tasks: in, Statuses: statuses}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	out := lr.Tasks
	if len(out) != len(in) {
		t.Fatalf("tasks len: got %d want %d", len(out), len(in))
	}
	for i := range in {
		if !reflect.DeepEqual(out[i], in[i]) {
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
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for unknown parent_id")
	}
}

func TestYAMLSubtaskNestedAllowed(t *testing.T) {
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
	if _, err := repo.Load(); err != nil {
		t.Errorf("unexpected error for 5-level nest: %v", err)
	}
}

func TestYAMLSubtaskDepthExceeded(t *testing.T) {
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
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for nesting depth exceeded")
	}
}

func TestYAMLPositionAutoAssign(t *testing.T) {
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
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[int]int{10: 1, 11: 2, 12: 1, 13: 2}
	for _, tk := range lr.Tasks {
		if tk.Position != want[tk.ID] {
			t.Errorf("task id=%d: got Position=%d want %d", tk.ID, tk.Position, want[tk.ID])
		}
	}

	lr2, err := repo.Load()
	if err != nil {
		t.Fatalf("re-Load: %v", err)
	}
	for _, tk := range lr2.Tasks {
		if tk.Position != want[tk.ID] {
			t.Errorf("re-loaded task id=%d: got Position=%d want %d", tk.ID, tk.Position, want[tk.ID])
		}
	}
}

func TestYAMLPositionPartialAutoAssign(t *testing.T) {
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
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[int]int{1: 5, 2: 6, 3: 2}
	for _, tk := range lr.Tasks {
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
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for task id <= 0")
	}
}

// ---- 拡張項目 (fields) のテスト ----

func TestYAMLFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	statuses := task.DefaultStatuses()
	defs := task.FieldDefList{
		{ID: 1, Name: "締切日", Type: task.FieldTypeText, Position: 1},
		{ID: 2, Name: "開始日", Type: task.FieldTypeText, Position: 2},
	}
	in := []task.Task{
		{ID: 1, Title: "task1", StatusID: 1, Position: 1, Fields: task.TaskFieldList{
			{ID: 1, FieldID: 1, Value: "2025-01-01"},
			{ID: 2, FieldID: 2, Value: "2024-12-25"},
		}},
		{ID: 2, Title: "task2", StatusID: 1, Position: 2, Fields: task.TaskFieldList{
			{ID: 1, FieldID: 1, Value: "2025-02-15"},
		}},
	}
	if err := repo.Save(LoadResult{Tasks: in, Statuses: statuses, Fields: defs}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(lr.Fields, defs) {
		t.Errorf("fields mismatch: got %+v, want %+v", lr.Fields, defs)
	}
	if !reflect.DeepEqual(lr.Tasks, in) {
		t.Errorf("tasks mismatch: got %+v, want %+v", lr.Tasks, in)
	}
}

func TestYAMLFieldsAutoAssignID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
fields:
  - field:
      type: text
      name: 締切日
  - field:
      type: text
      name: 備考
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(lr.Fields))
	}
	if lr.Fields[0].ID == 0 || lr.Fields[1].ID == 0 {
		t.Errorf("ids should be assigned, got %+v", lr.Fields)
	}
	if lr.Fields[0].Position == 0 || lr.Fields[1].Position == 0 {
		t.Errorf("positions should be assigned, got %+v", lr.Fields)
	}
}

func TestYAMLFieldsTypeDefaultsToText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
fields:
  - field:
      id: 1
      name: メモ
      position: 1
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lr.Fields[0].Type != task.FieldTypeText {
		t.Errorf("type should default to text, got %q", lr.Fields[0].Type)
	}
}

func TestYAMLFieldsUnknownFieldID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
fields:
  - field:
      id: 1
      type: text
      name: 締切日
      position: 1
tasks:
  - task:
      id: 1
      title: x
      status_id: 1
      fields:
        - field:
            id: 1
            field_id: 99
            value: bad
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for unknown field_id reference")
	}
}

func TestYAMLFieldsDuplicateFieldIDInTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
fields:
  - field:
      id: 1
      type: text
      name: 締切日
      position: 1
tasks:
  - task:
      id: 1
      title: x
      status_id: 1
      fields:
        - field:
            id: 1
            field_id: 1
            value: a
        - field:
            id: 2
            field_id: 1
            value: b
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	if _, err := repo.Load(); err == nil {
		t.Error("expected error for duplicated field_id within a task")
	}
}

func TestYAMLFieldsTaskFieldAutoAssignID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
fields:
  - field:
      id: 1
      type: text
      name: 締切日
      position: 1
tasks:
  - task:
      id: 1
      title: x
      status_id: 1
      fields:
        - field:
            field_id: 1
            value: a
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Tasks[0].Fields) != 1 {
		t.Fatalf("len fields=%d, want 1", len(lr.Tasks[0].Fields))
	}
	if lr.Tasks[0].Fields[0].ID == 0 {
		t.Errorf("task field id should be auto-assigned, got %+v", lr.Tasks[0].Fields[0])
	}
}

// ---- Layout のテスト ----

// floatPtr は *float64 リテラルを書きやすくするヘルパ (テスト専用)。
func floatPtr(v float64) *float64 { return &v }

// layout キーが無い既存 yaml を読んでも cfg.Layout は全 nil。
func TestYAMLLayoutAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	lc := lr.Config.Layout
	if lc.TaskListWidth != nil || lc.TaskDetailHeight != nil ||
		lc.FileListHeight != nil || lc.FilePreviewHeight != nil {
		t.Errorf("expected all nil, got %+v", lc)
	}
}

// 4 値が揃った yaml をロードすると LayoutConfig がポインタ越しで一致する。
func TestYAMLLayoutRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)

	in := AppConfig{
		Layout: LayoutConfig{
			TaskListWidth:     floatPtr(0.6),
			TaskDetailHeight:  floatPtr(0.4),
			FileListHeight:    floatPtr(0.3),
			FilePreviewHeight: floatPtr(0.3),
		},
	}
	if err := repo.Save(LoadResult{
		Statuses: task.DefaultStatuses(),
		Config:   in,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := lr.Config.Layout
	checks := []struct {
		name string
		got  *float64
		want *float64
	}{
		{"task_list.width", got.TaskListWidth, in.Layout.TaskListWidth},
		{"task_detail.height", got.TaskDetailHeight, in.Layout.TaskDetailHeight},
		{"file_list.height", got.FileListHeight, in.Layout.FileListHeight},
		{"file_preview.height", got.FilePreviewHeight, in.Layout.FilePreviewHeight},
	}
	for _, c := range checks {
		if c.got == nil || c.want == nil || *c.got != *c.want {
			t.Errorf("%s: got %v want %v", c.name, c.got, c.want)
		}
	}
}

// 一部のフィールドだけ書かれた yaml をロード → 該当だけ非 nil、残りは nil。
func TestYAMLLayoutPartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	body := `layout:
  main:
    task_list:
      width: 0.7
statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
tasks: []
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	lc := lr.Config.Layout
	if lc.TaskListWidth == nil || *lc.TaskListWidth != 0.7 {
		t.Errorf("task_list.width: got %v want 0.7", lc.TaskListWidth)
	}
	if lc.TaskDetailHeight != nil || lc.FileListHeight != nil || lc.FilePreviewHeight != nil {
		t.Errorf("expected other fields nil, got %+v", lc)
	}
}

// 全 nil の LayoutConfig で Save → yaml に layout キーが書かれない。
func TestYAMLLayoutMarshalEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(path)
	if err := repo.Save(LoadResult{Statuses: task.DefaultStatuses()}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "layout:") {
		t.Errorf("expected no 'layout:' key in output, got:\n%s", string(data))
	}
}

func TestYAMLNoFieldsKeyOK(t *testing.T) {
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
      status_id: 1
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	repo := NewYAMLRepository(path)
	lr, err := repo.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lr.Fields) != 0 {
		t.Errorf("expected empty fields, got %+v", lr.Fields)
	}
	if len(lr.Tasks[0].Fields) != 0 {
		t.Errorf("expected empty task fields, got %+v", lr.Tasks[0].Fields)
	}
}
