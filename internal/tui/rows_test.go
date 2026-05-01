package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/task"
)

func TestBuildRowsOrderAndGrouping(t *testing.T) {
	statuses := task.DefaultStatuses() // 1=todo seq1, 2=doing seq2, 3=done seq3
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1}, // todo
		{ID: 11, Title: "b", StatusID: 3}, // done
		{ID: 12, Title: "c", StatusID: 1}, // todo
		{ID: 13, Title: "d", StatusID: 2}, // doing
	}
	rows := buildRows(statuses, tasks, nil)

	// 期待: done ヘッダ, b, sep, doing ヘッダ, d, sep, todo ヘッダ, a, c
	wantKinds := []rowKind{
		rowStatus, rowTask, rowSeparator,
		rowStatus, rowTask, rowSeparator,
		rowStatus, rowTask, rowTask,
	}
	if len(rows) != len(wantKinds) {
		t.Fatalf("len=%d, want %d (rows=%+v)", len(rows), len(wantKinds), rows)
	}
	for i, k := range wantKinds {
		if rows[i].kind != k {
			t.Errorf("[%d]: kind=%v want %v", i, rows[i].kind, k)
		}
	}
	// done が先頭、todo が末尾
	if rows[0].statusID != 3 {
		t.Errorf("first status: got %d want 3 (done)", rows[0].statusID)
	}
	if rows[6].statusID != 1 {
		t.Errorf("last status: got %d want 1 (todo)", rows[6].statusID)
	}
	// done 配下のタスク
	if rows[1].taskIndex != 1 {
		t.Errorf("done task: got idx %d want 1", rows[1].taskIndex)
	}
	// todo 配下: a → c の順 (yaml 出現順)
	if rows[7].taskIndex != 0 || rows[8].taskIndex != 2 {
		t.Errorf("todo tasks: got idx %d,%d want 0,2", rows[7].taskIndex, rows[8].taskIndex)
	}
}

func TestBuildRowsCollapsed(t *testing.T) {
	statuses := task.DefaultStatuses()
	tasks := []task.Task{
		{ID: 1, Title: "a", StatusID: 1},
		{ID: 2, Title: "b", StatusID: 2},
	}
	collapsed := map[int]bool{2: true} // doing を折りたたみ
	rows := buildRows(statuses, tasks, collapsed)

	// 期待: done ヘッダ, sep, doing ヘッダ (タスクなし), sep, todo ヘッダ, a
	if len(rows) != 6 {
		t.Fatalf("len=%d, want 6", len(rows))
	}
	if rows[2].kind != rowStatus || rows[2].statusID != 2 {
		t.Errorf("doing header expected at idx 2, got %+v", rows[2])
	}
	// doing 配下にタスクが居ないこと
	if rows[3].kind != rowSeparator {
		t.Errorf("idx 3 should be separator (no task), got %+v", rows[3])
	}
}

func TestNavigableSkipsSeparator(t *testing.T) {
	statuses := task.DefaultStatuses()
	tasks := []task.Task{
		{ID: 1, Title: "a", StatusID: 3}, // done のみ
	}
	rows := buildRows(statuses, tasks, nil)
	// rows: [done, a, sep, doing, sep, todo]
	if got := nextNavigable(rows, 1); got != 3 {
		t.Errorf("nextNavigable from 1 = %d, want 3 (skip sep)", got)
	}
	if got := prevNavigable(rows, 3); got != 1 {
		t.Errorf("prevNavigable from 3 = %d, want 1 (skip sep)", got)
	}
}

func TestFirstNavigableEmpty(t *testing.T) {
	if got := firstNavigable(nil); got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}

func TestBuildRowsSubtaskNesting(t *testing.T) {
	statuses := task.DefaultStatuses()
	tasks := []task.Task{
		{ID: 1, Title: "parent1", StatusID: 1},
		{ID: 2, Title: "child1a", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "parent2", StatusID: 1},
		{ID: 4, Title: "child1b", StatusID: 1, ParentID: 1},
	}
	rows := buildRows(statuses, tasks, nil)
	var todoTasks []listRow
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			todoTasks = append(todoTasks, r)
		}
	}
	if len(todoTasks) != 4 {
		t.Fatalf("todo tasks: got %d, want 4 (rows=%+v)", len(todoTasks), todoTasks)
	}
	wantOrder := []struct {
		taskIndex int
		depth     int
	}{
		{0, 0},
		{1, 1},
		{3, 1},
		{2, 0},
	}
	for i, w := range wantOrder {
		if todoTasks[i].taskIndex != w.taskIndex || todoTasks[i].depth != w.depth {
			t.Errorf("todoTasks[%d] = (idx=%d depth=%d), want (idx=%d depth=%d)",
				i, todoTasks[i].taskIndex, todoTasks[i].depth, w.taskIndex, w.depth)
		}
	}
}

func TestBuildRowsMultiLevelNesting(t *testing.T) {
	statuses := task.DefaultStatuses()
	tasks := []task.Task{
		{ID: 1, Title: "l0", StatusID: 1},
		{ID: 2, Title: "l1", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "l2", StatusID: 1, ParentID: 2},
		{ID: 4, Title: "l3", StatusID: 1, ParentID: 3},
	}
	rows := buildRows(statuses, tasks, nil)
	var todoTasks []listRow
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			todoTasks = append(todoTasks, r)
		}
	}
	if len(todoTasks) != 4 {
		t.Fatalf("todo tasks: got %d, want 4", len(todoTasks))
	}
	for i, want := range []int{0, 1, 2, 3} {
		if todoTasks[i].depth != want {
			t.Errorf("todoTasks[%d].depth = %d, want %d", i, todoTasks[i].depth, want)
		}
	}
}

func TestTaskDepth(t *testing.T) {
	tasks := []task.Task{
		{ID: 1, Title: "l0", StatusID: 1},
		{ID: 2, Title: "l1", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "l2", StatusID: 1, ParentID: 2},
	}
	if got := taskDepth(tasks, 1); got != 0 {
		t.Errorf("depth(1)=%d want 0", got)
	}
	if got := taskDepth(tasks, 2); got != 1 {
		t.Errorf("depth(2)=%d want 1", got)
	}
	if got := taskDepth(tasks, 3); got != 2 {
		t.Errorf("depth(3)=%d want 2", got)
	}
}
