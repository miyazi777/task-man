package task

import (
	"reflect"
	"testing"
)

func threeStatuses() StatusList {
	return StatusList{
		{ID: 1, Sequence: 1, Label: "todo"},
		{ID: 2, Sequence: 2, Label: "doing"},
		{ID: 3, Sequence: 3, Label: "done"},
	}
}

// posMap は (id -> position) のマップを作る。テストの期待値比較に使う。
func posMap(tasks []Task) map[int]int {
	out := map[int]int{}
	for _, t := range tasks {
		out[t.ID] = t.Position
	}
	return out
}

func parentMap(tasks []Task) map[int]int {
	out := map[int]int{}
	for _, t := range tasks {
		out[t.ID] = t.ParentID
	}
	return out
}

func statusMap(tasks []Task) map[int]int {
	out := map[int]int{}
	for _, t := range tasks {
		out[t.ID] = t.StatusID
	}
	return out
}

func TestMoveTaskUp_SwapsWithPreviousSibling(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
		{ID: 3, Title: "C", StatusID: 1, Position: 3},
	}
	out := MoveTaskUp(tasks, threeStatuses(), 3)
	want := map[int]int{1: 1, 2: 3, 3: 2}
	if got := posMap(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("positions = %v want %v", got, want)
	}
}

func TestMoveTaskUp_AtTopOfTopLevel_MovesToLowerSequenceStatus(t *testing.T) {
	// 表示は sequence 昇順 (todo が上、done が下)。視覚的「上」 = sequence が小さい方。
	// status 2 (doing) の先頭にある B を up → status 1 (todo) の末尾へ。
	// ルートのみ status 移動。子孫の status_id は据え置き (描画は親直下にネスト)。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 2, Position: 1},
		{ID: 3, Title: "B-child", StatusID: 2, ParentID: 2, Position: 1},
		{ID: 4, Title: "C", StatusID: 2, Position: 2},
	}
	out := MoveTaskUp(tasks, threeStatuses(), 2)
	if got := statusMap(out); got[2] != 1 {
		t.Fatalf("B status = %d want 1", got[2])
	}
	if got := statusMap(out); got[3] != 2 {
		t.Fatalf("B-child status = %d want 2 (unchanged)", got[3])
	}
	if got := posMap(out); got[1] != 1 || got[2] != 2 || got[4] != 1 {
		t.Fatalf("posMap = %v want A=1 B=2 C=1", got)
	}
}

func TestMoveTaskUp_AtTopOfLowestSequenceStatus_NoOp(t *testing.T) {
	// status 1 が視覚的に最上位 (sequence 昇順描画)。さらに上は無い。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
	}
	out := MoveTaskUp(tasks, threeStatuses(), 1)
	if got := statusMap(out); got[1] != 1 {
		t.Fatalf("statusMap changed: %v", got)
	}
}

func TestMoveTaskUp_NonTopLevelAtFirst_NoOp(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1},
		{ID: 2, Title: "C1", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "C2", StatusID: 1, ParentID: 1, Position: 2},
	}
	out := MoveTaskUp(tasks, threeStatuses(), 2)
	if got := posMap(out); got[2] != 1 || got[3] != 2 {
		t.Fatalf("positions changed unexpectedly: %v", got)
	}
}

func TestMoveTaskDown_SwapsWithNextSibling(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
	}
	out := MoveTaskDown(tasks, threeStatuses(), 1)
	want := map[int]int{1: 2, 2: 1}
	if got := posMap(out); !reflect.DeepEqual(got, want) {
		t.Fatalf("positions = %v want %v", got, want)
	}
}

func TestMoveTaskDown_AtBottomOfTopLevel_MovesToHigherSequenceStatus(t *testing.T) {
	// 視覚的「下」 = sequence が大きい方。status 2 (doing) の末尾にある B を down → status 3 (done) の先頭へ。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 2, Position: 1},
		{ID: 2, Title: "B", StatusID: 2, Position: 2},
		{ID: 3, Title: "X", StatusID: 3, Position: 1},
	}
	out := MoveTaskDown(tasks, threeStatuses(), 2)
	if got := statusMap(out); got[2] != 3 {
		t.Fatalf("B status = %d want 3", got[2])
	}
	if got := posMap(out); got[2] != 1 || got[3] != 2 {
		t.Fatalf("positions = %v want B=1 X=2", got)
	}
}

func TestReassignTasksToFallback_RewritesStatusAndRenumbers(t *testing.T) {
	// status 1 は元から A,B が居る (pos 1,2)。status 2 を削除して status 1 へ寄せる。
	tasks := []Task{
		{ID: 10, Title: "A", StatusID: 1, Position: 1},
		{ID: 11, Title: "B", StatusID: 1, Position: 2},
		{ID: 20, Title: "X", StatusID: 2, Position: 1},
		{ID: 21, Title: "Y", StatusID: 2, Position: 2},
	}
	out := ReassignTasksToFallback(tasks, 2, 1)
	// すべて status 1 になる
	if got := statusMap(out); got[10] != 1 || got[11] != 1 || got[20] != 1 || got[21] != 1 {
		t.Fatalf("statusMap = %v, want all in status 1", got)
	}
	// position は (oldPos, id) 昇順で再採番される。
	// A(pos1,id10), X(pos1,id20), B(pos2,id11), Y(pos2,id21) → 1,2,3,4
	want := map[int]int{10: 1, 20: 2, 11: 3, 21: 4}
	if got := posMap(out); !reflect.DeepEqual(got, want) {
		t.Errorf("posMap = %v, want %v", got, want)
	}
	// 元の slice は変更されない
	if tasks[2].StatusID != 2 {
		t.Error("source tasks must remain unchanged")
	}
}

func TestReassignTasksToFallback_SubtaskFollowsByStatusOnly(t *testing.T) {
	// 親が status 2 / 子も status 2 → 親子ともに status 1 へ。
	// 子の position は同じ親の中で完結しているのでそのまま。
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 2, Position: 1},
		{ID: 2, Title: "C", StatusID: 2, ParentID: 1, Position: 1},
	}
	out := ReassignTasksToFallback(tasks, 2, 1)
	if got := statusMap(out); got[1] != 1 || got[2] != 1 {
		t.Fatalf("statusMap = %v, want both in status 1", got)
	}
	if out[1].Position != 1 {
		t.Errorf("child position = %d, want 1", out[1].Position)
	}
}

func TestIndentTask_BecomesChildOfPreviousSibling(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
	}
	out := IndentTask(tasks, 2)
	if got := parentMap(out); got[2] != 1 {
		t.Fatalf("parent of 2 = %d want 1", got[2])
	}
	if got := posMap(out); got[1] != 1 || got[2] != 1 {
		t.Fatalf("positions = %v want A=1 B(child)=1", got)
	}
}

func TestIndentTask_NoPreviousSibling_NoOp(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
	}
	out := IndentTask(tasks, 1)
	if got := parentMap(out); got[1] != 0 {
		t.Fatalf("parent changed: %v", got)
	}
}

func TestIndentTask_DepthLimit_NoOp(t *testing.T) {
	// 5 levels = root + 4 children. MaxNestDepth = 4 (depths 0..4).
	tasks := []Task{
		{ID: 1, Title: "L0", StatusID: 1, Position: 1},
		{ID: 2, Title: "L0sib", StatusID: 1, Position: 2},
		{ID: 3, Title: "L1", StatusID: 1, ParentID: 2, Position: 1},
		{ID: 4, Title: "L2", StatusID: 1, ParentID: 3, Position: 1},
		{ID: 5, Title: "L3", StatusID: 1, ParentID: 4, Position: 1},
		{ID: 6, Title: "L4", StatusID: 1, ParentID: 5, Position: 1},
	}
	// Indent ID=2 (L0sib) into ID=1: subtree relative depth of 2 = 4 (down to L4).
	// New depth = 1 + 4 = 5 > MaxNestDepth(4) → no-op.
	out := IndentTask(tasks, 2)
	if got := parentMap(out); got[2] != 0 {
		t.Fatalf("indent should have been blocked but parent of 2 = %d", got[2])
	}
}

func TestOutdentTask_TopLevel_NoOp(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
	}
	out := OutdentTask(tasks, 1)
	if got := parentMap(out); got[1] != 0 {
		t.Fatalf("unexpected parent: %v", got)
	}
}

func TestOutdentTask_BecomesSiblingOfParent(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1},
		{ID: 2, Title: "C1", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "C2", StatusID: 1, ParentID: 1, Position: 2},
		{ID: 4, Title: "After", StatusID: 1, Position: 2},
	}
	out := OutdentTask(tasks, 2)
	// C1 outdents to top-level, position right after P (= 2). After becomes 3.
	if got := parentMap(out); got[2] != 0 {
		t.Fatalf("parent of 2 = %d want 0", got[2])
	}
	if got := posMap(out); got[1] != 1 || got[2] != 2 || got[4] != 3 || got[3] != 1 {
		t.Fatalf("positions = %v unexpected", got)
	}
}

func TestOutdentTask_DescendantsFollow(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "GC", StatusID: 1, ParentID: 2, Position: 1},
	}
	out := OutdentTask(tasks, 2)
	// C outdents to top-level. GC's parent is still C (so GC follows C automatically).
	if got := parentMap(out); got[2] != 0 || got[3] != 2 {
		t.Fatalf("parent map = %v want 2:0 3:2", got)
	}
	if got := posMap(out); got[1] != 1 || got[2] != 2 {
		t.Fatalf("positions = %v want P=1 C=2", got)
	}
}
