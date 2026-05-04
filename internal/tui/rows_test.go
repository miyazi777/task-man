package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/task"
)

// threeStatusesNoTrash は trash を含まない 3 ステータスのテスト用フィクスチャ。
// 行レイアウト系のテストでは trash の存在で row 数が増えてしまうため、テストの簡潔さのために使う。
func threeStatusesNoTrash() task.StatusList {
	return task.StatusList{
		{ID: 1, Sequence: 1, Label: "todo"},
		{ID: 2, Sequence: 2, Label: "doing"},
		{ID: 3, Sequence: 3, Label: "done"},
	}
}

func TestBuildRowsOrderAndGrouping(t *testing.T) {
	statuses := threeStatusesNoTrash() // 1=todo seq1, 2=doing seq2, 3=done seq3
	tasks := []task.Task{
		{ID: 10, Title: "a", StatusID: 1}, // todo
		{ID: 11, Title: "b", StatusID: 3}, // done
		{ID: 12, Title: "c", StatusID: 1}, // todo
		{ID: 13, Title: "d", StatusID: 2}, // doing
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	// 期待: todo ヘッダ, a, c, sep, doing ヘッダ, d, sep, done ヘッダ, b
	wantKinds := []rowKind{
		rowStatus, rowTask, rowTask, rowSeparator,
		rowStatus, rowTask, rowSeparator,
		rowStatus, rowTask,
	}
	if len(rows) != len(wantKinds) {
		t.Fatalf("len=%d, want %d (rows=%+v)", len(rows), len(wantKinds), rows)
	}
	for i, k := range wantKinds {
		if rows[i].kind != k {
			t.Errorf("[%d]: kind=%v want %v", i, rows[i].kind, k)
		}
	}
	// todo が先頭、done が末尾 (sequence 昇順 = yaml 順)
	if rows[0].statusID != 1 {
		t.Errorf("first status: got %d want 1 (todo)", rows[0].statusID)
	}
	if rows[7].statusID != 3 {
		t.Errorf("last status: got %d want 3 (done)", rows[7].statusID)
	}
	// todo 配下のタスク: a → c の順 (yaml 出現順)
	if rows[1].taskIndex != 0 || rows[2].taskIndex != 2 {
		t.Errorf("todo tasks: got idx %d,%d want 0,2", rows[1].taskIndex, rows[2].taskIndex)
	}
	// done 配下のタスク
	if rows[8].taskIndex != 1 {
		t.Errorf("done task: got idx %d want 1", rows[8].taskIndex)
	}
}

func TestBuildRowsCollapsed(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "a", StatusID: 1},
		{ID: 2, Title: "b", StatusID: 2},
	}
	collapsed := map[int]bool{2: true} // doing を折りたたみ
	rows := buildRows(statuses, tasks, collapsed, nil, false)

	// 期待: todo ヘッダ, a, sep, doing ヘッダ (タスクなし), sep, done ヘッダ
	if len(rows) != 6 {
		t.Fatalf("len=%d, want 6", len(rows))
	}
	if rows[3].kind != rowStatus || rows[3].statusID != 2 {
		t.Errorf("doing header expected at idx 3, got %+v", rows[3])
	}
	// doing 配下にタスクが居ないこと (即 separator)
	if rows[4].kind != rowSeparator {
		t.Errorf("idx 4 should be separator (no task), got %+v", rows[4])
	}
}

func TestNavigableSkipsSeparator(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "a", StatusID: 1}, // todo のみ
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
	// rows: [todo, a, sep, doing, sep, done]
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
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "parent1", StatusID: 1},
		{ID: 2, Title: "child1a", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "parent2", StatusID: 1},
		{ID: 4, Title: "child1b", StatusID: 1, ParentID: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
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

func TestBuildRowsSubtaskWithDifferentStatusStaysNested(t *testing.T) {
	// ルートは status 2 (doing)、子は status 1 (todo)。新挙動では status を問わず
	// 子は親直下にネスト表示する。子が独立した todo グループの top-level には現れない。
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "parent", StatusID: 2},
		{ID: 2, Title: "child", StatusID: 1, ParentID: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)

	// 子の行は親グループ (status 2) の中にネスト (depth=1) で出現する。
	var nestedFound bool
	for _, r := range rows {
		if r.kind != rowTask {
			continue
		}
		if tasks[r.taskIndex].ID == 2 {
			if r.statusID != 2 || r.depth != 1 {
				t.Errorf("child row got statusID=%d depth=%d, want 2/1", r.statusID, r.depth)
			}
			nestedFound = true
		}
	}
	if !nestedFound {
		t.Error("child row not found in any status group")
	}
	// status 1 (todo) のタスク行は 0 件 (子は親直下にネストされたので独立しない)。
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			t.Errorf("unexpected task in status 1 group: idx=%d", r.taskIndex)
		}
	}
}

func TestBuildRowsMultiLevelNesting(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "l0", StatusID: 1},
		{ID: 2, Title: "l1", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "l2", StatusID: 1, ParentID: 2},
		{ID: 4, Title: "l3", StatusID: 1, ParentID: 3},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
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

func TestBuildRowsTaskCollapsed(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "p", StatusID: 1},
		{ID: 2, Title: "c1", StatusID: 1, ParentID: 1},
		{ID: 3, Title: "c2", StatusID: 1, ParentID: 1},
		{ID: 4, Title: "gc", StatusID: 1, ParentID: 2},
	}
	taskCollapsed := map[int]bool{1: true} // p を折りたたみ
	rows := buildRows(statuses, tasks, nil, taskCollapsed, false)
	var todoTasks []listRow
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			todoTasks = append(todoTasks, r)
		}
	}
	// p のみが表示され、c1/c2/gc は非表示。マーカー: p は collapsed=true, hasChildren=true
	if len(todoTasks) != 1 {
		t.Fatalf("expected only parent visible when collapsed, got %d rows", len(todoTasks))
	}
	if !todoTasks[0].hasChildren || !todoTasks[0].collapsed {
		t.Errorf("parent row should have hasChildren=true, collapsed=true, got %+v", todoTasks[0])
	}
}

func TestTaskHasChildren(t *testing.T) {
	tasks := []task.Task{
		{ID: 1, Title: "p"},
		{ID: 2, Title: "c", ParentID: 1},
		{ID: 3, Title: "lonely"},
	}
	if !taskHasChildren(tasks, 1) {
		t.Error("id=1 should have children")
	}
	if taskHasChildren(tasks, 3) {
		t.Error("id=3 should NOT have children")
	}
	if taskHasChildren(tasks, 999) {
		t.Error("nonexistent id should NOT have children")
	}
}

func TestBuildRowsSortedByPosition(t *testing.T) {
	statuses := threeStatusesNoTrash()
	// 同じ status (todo) のルートタスクを yaml 順 (a,b,c) と異なる position 順 (3,1,2) で並べる。
	tasks := []task.Task{
		{ID: 1, Title: "a", StatusID: 1, Position: 3},
		{ID: 2, Title: "b", StatusID: 1, Position: 1},
		{ID: 3, Title: "c", StatusID: 1, Position: 2},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
	var todoTaskIdx []int
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			todoTaskIdx = append(todoTaskIdx, r.taskIndex)
		}
	}
	// 期待: position 昇順なので b(idx=1), c(idx=2), a(idx=0)
	want := []int{1, 2, 0}
	if len(todoTaskIdx) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", todoTaskIdx, want)
	}
	for i := range want {
		if todoTaskIdx[i] != want[i] {
			t.Errorf("[%d]: got idx=%d want %d", i, todoTaskIdx[i], want[i])
		}
	}
}

func TestBuildRowsSubtaskSortedByPosition(t *testing.T) {
	statuses := threeStatusesNoTrash()
	tasks := []task.Task{
		{ID: 1, Title: "p", StatusID: 1, Position: 1},
		{ID: 2, Title: "c1", StatusID: 1, ParentID: 1, Position: 3},
		{ID: 3, Title: "c2", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 4, Title: "c3", StatusID: 1, ParentID: 1, Position: 2},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
	var sub []int
	for _, r := range rows {
		if r.kind == rowTask && r.depth == 1 {
			sub = append(sub, r.taskIndex)
		}
	}
	// 期待: position 昇順なので c2(idx=2), c3(idx=3), c1(idx=1)
	want := []int{2, 3, 1}
	if len(sub) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", sub, want)
	}
	for i := range want {
		if sub[i] != want[i] {
			t.Errorf("[%d]: got idx=%d want %d", i, sub[i], want[i])
		}
	}
}

func TestBuildRowsPositionTieBreakerByID(t *testing.T) {
	statuses := threeStatusesNoTrash()
	// position が同じ場合は id 昇順。
	tasks := []task.Task{
		{ID: 5, Title: "a", StatusID: 1, Position: 1},
		{ID: 2, Title: "b", StatusID: 1, Position: 1},
		{ID: 9, Title: "c", StatusID: 1, Position: 1},
	}
	rows := buildRows(statuses, tasks, nil, nil, false)
	var ids []int
	for _, r := range rows {
		if r.kind == rowTask && r.statusID == 1 {
			ids = append(ids, tasks[r.taskIndex].ID)
		}
	}
	want := []int{2, 5, 9}
	if len(ids) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("[%d]: got id=%d want %d", i, ids[i], want[i])
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
