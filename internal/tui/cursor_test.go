package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

// issue #41: storage.CursorState から起動時のカーソル位置を復元する。
//
//   - TaskID 指定 + 該当タスクが rows に存在 → そのタスク行へ
//   - StatusID 指定 + 該当ヘッダが存在 → そのヘッダ行へ
//   - 復元できないとき (削除済 / 未保存) → firstNavigable (= 先頭ステータスヘッダ)
func TestRestoreCursorByTaskID(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
		{ID: 11, Title: "b", StatusID: 2, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	got := restoreCursor(rows, tasks, storage.CursorState{TaskID: 11})
	// rows: [status1, task10, sep, status2, task11, sep, status3]
	want := 4
	if got != want {
		t.Errorf("restoreCursor(TaskID=11): got %d, want %d (rows=%+v)", got, want, rows)
	}
}

func TestRestoreCursorByStatusID(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	// status 3 (done) のヘッダ行に復元したい
	got := restoreCursor(rows, tasks, storage.CursorState{StatusID: 3})
	// rows: [status1, task10, sep, status2, sep, status3]
	want := 5
	if got != want {
		t.Errorf("restoreCursor(StatusID=3): got %d, want %d (rows=%+v)", got, want, rows)
	}
}

// TaskID が削除済 (rows に存在しない) のときは firstNavigable へフォールバック。
func TestRestoreCursorFallbackOnMissingTask(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	got := restoreCursor(rows, tasks, storage.CursorState{TaskID: 999})
	want := firstNavigable(rows)
	if got != want {
		t.Errorf("restoreCursor(missing TaskID): got %d, want firstNavigable=%d", got, want)
	}
}

// 未保存 (両 0) のときは firstNavigable に落ちる。
func TestRestoreCursorZeroValueFallback(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	got := restoreCursor(rows, tasks, storage.CursorState{})
	want := firstNavigable(rows)
	if got != want {
		t.Errorf("restoreCursor(zero): got %d, want firstNavigable=%d", got, want)
	}
}

// TaskID が指定されていて、その所属ステータスが折りたたみ中で row が出ていない場合、
// StatusID 側は 0 のままなので firstNavigable へ落ちる。
func TestRestoreCursorTaskInCollapsedStatusFallback(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
	}
	// status 1 を折りたたむと task 10 の行は生成されない。
	rows := buildRows(statuses, tasks, map[int]bool{1: true}, nil, false)

	got := restoreCursor(rows, tasks, storage.CursorState{TaskID: 10})
	want := firstNavigable(rows)
	if got != want {
		t.Errorf("restoreCursor(task in collapsed status): got %d, want firstNavigable=%d", got, want)
	}
}

// currentCursorState はタスク行・ステータス行・separator/範囲外で適切な CursorState を返す。
func TestCurrentCursorState(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1, Position: 1},
		{ID: 11, Title: "b", StatusID: 2, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
	m := Model{tasks: tasks, rows: rows}

	// rows: [0:status1, 1:task10, 2:sep, 3:status2, 4:task11, 5:sep, 6:status3]
	tests := []struct {
		name   string
		cursor int
		want   storage.CursorState
	}{
		{"task row", 1, storage.CursorState{TaskID: 10}},
		{"status row", 3, storage.CursorState{StatusID: 2}},
		{"separator row", 2, storage.CursorState{}},
		{"out of range high", 99, storage.CursorState{}},
		{"out of range low", -1, storage.CursorState{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m.cursor = tc.cursor
			got := m.currentCursorState()
			if got != tc.want {
				t.Errorf("cursor=%d: got %+v, want %+v", tc.cursor, got, tc.want)
			}
		})
	}
}

// findRowForTaskID は taskID == 0 のとき必ず -1 を返す (ゼロ値 fallthrough)。
func TestFindRowForTaskIDZero(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{{ID: 10, Title: "a", StatusID: 1, Position: 1}}
	rows := buildRows(statuses, tasks, nil, nil, false)
	if got := findRowForTaskID(rows, tasks, 0); got != -1 {
		t.Errorf("findRowForTaskID(0): got %d, want -1", got)
	}
}
