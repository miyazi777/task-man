package task

import (
	"sort"
	"testing"
)

// findByID はテスト用ヘルパ。tasks から ID で 1 件取得する。
func findByID(tasks []Task, id int) Task {
	for _, t := range tasks {
		if t.ID == id {
			return t
		}
	}
	return Task{}
}

// groupedPositions は (parentID, statusID) ごとに ID を Position 昇順で返す。
func groupedPositions(tasks []Task, parentID, statusID int) []int {
	type pair struct{ id, pos int }
	var pairs []pair
	for _, t := range tasks {
		if t.ParentID == parentID && t.StatusID == statusID {
			pairs = append(pairs, pair{t.ID, t.Position})
		}
	}
	sort.SliceStable(pairs, func(a, b int) bool { return pairs[a].pos < pairs[b].pos })
	ids := make([]int, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
}

func TestMoveTasksAfterSibling(t *testing.T) {
	// 同一ステータス・親内で並び替え: [A,B,C,D] のうち D を A の次へ移動。
	// 期待: [A, D, B, C] → Position は 1,2,3,4 で連番採番される。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
		{ID: 3, Title: "C", StatusID: 1, Position: 3},
		{ID: 4, Title: "D", StatusID: 1, Position: 4},
	}
	out := MoveTasks(tasks, map[int]bool{4: true}, MoveDestination{ParentID: 0, StatusID: 1, InsertAt: 2})
	got := groupedPositions(out, 0, 1)
	want := []int{1, 4, 2, 3}
	if !equalInts(got, want) {
		t.Errorf("got order %v, want %v", got, want)
	}
}

func TestMoveTasksToStatusFirst(t *testing.T) {
	// 異ステータスへの引っ越し: B (status=1) を status=2 の先頭へ。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
		{ID: 3, Title: "X", StatusID: 2, Position: 1},
		{ID: 4, Title: "Y", StatusID: 2, Position: 2},
	}
	out := MoveTasks(tasks, map[int]bool{2: true}, MoveDestination{ParentID: 0, StatusID: 2, InsertAt: 1})

	// status=2 の先頭が B になり、X, Y が後ろに続く。
	got2 := groupedPositions(out, 0, 2)
	want2 := []int{2, 3, 4}
	if !equalInts(got2, want2) {
		t.Errorf("status=2 order: got %v, want %v", got2, want2)
	}
	// status=1 には A だけが残り Position=1 に詰める。
	got1 := groupedPositions(out, 0, 1)
	want1 := []int{1}
	if !equalInts(got1, want1) {
		t.Errorf("status=1 order: got %v, want %v", got1, want1)
	}
	// B のステータスが更新されている。
	if findByID(out, 2).StatusID != 2 {
		t.Errorf("B.StatusID: got %d, want 2", findByID(out, 2).StatusID)
	}
}

func TestMoveTasksAsFirstChild(t *testing.T) {
	// A の最初の子として C, D を挿入 (元はルート、status=1)。
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
		{ID: 3, Title: "C", StatusID: 1, Position: 3},
		{ID: 4, Title: "D", StatusID: 1, Position: 4},
	}
	out := MoveTasks(tasks, map[int]bool{3: true, 4: true}, MoveDestination{ParentID: 1, StatusID: 1, InsertAt: 1})

	// A の子が [C, D] の順 (元の Position 順を保つ)。
	gotChildren := groupedPositions(out, 1, 1)
	wantChildren := []int{3, 4}
	if !equalInts(gotChildren, wantChildren) {
		t.Errorf("children of A: got %v, want %v", gotChildren, wantChildren)
	}
	// ルートの status=1 には A, B のみ。
	gotRoot := groupedPositions(out, 0, 1)
	wantRoot := []int{1, 2}
	if !equalInts(gotRoot, wantRoot) {
		t.Errorf("root status=1: got %v, want %v", gotRoot, wantRoot)
	}
	// C, D の ParentID が A.ID (=1) に更新されている。
	for _, id := range []int{3, 4} {
		if findByID(out, id).ParentID != 1 {
			t.Errorf("task %d ParentID: got %d, want 1", id, findByID(out, id).ParentID)
		}
	}
}

func TestMoveTasksDoesNotTouchDescendants(t *testing.T) {
	// 親 P1 (status=1) を status=2 へ移動するとき、子 C1 (status=1, parent=P1) は触らない。
	tasks := []Task{
		{ID: 1, Title: "P1", StatusID: 1, Position: 1},
		{ID: 2, Title: "C1", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "X", StatusID: 2, Position: 1},
	}
	out := MoveTasks(tasks, map[int]bool{1: true}, MoveDestination{ParentID: 0, StatusID: 2, InsertAt: 1})

	// P1 は status=2、parent=0 に。
	p1 := findByID(out, 1)
	if p1.StatusID != 2 || p1.ParentID != 0 {
		t.Errorf("P1: got status=%d parent=%d, want 2/0", p1.StatusID, p1.ParentID)
	}
	// C1 は触られず status=1, parent=1 のまま。
	c1 := findByID(out, 2)
	if c1.StatusID != 1 || c1.ParentID != 1 {
		t.Errorf("C1: got status=%d parent=%d, want 1/1", c1.StatusID, c1.ParentID)
	}
}

func TestMoveTasksMultipleAtMiddle(t *testing.T) {
	// [A,B,C,D,E] のうち C, E を B の次 (InsertAt=2) へ。
	// 期待 (元の position 順を保つ): [A, B, C, E, D] → 1..5
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
		{ID: 2, Title: "B", StatusID: 1, Position: 2},
		{ID: 3, Title: "C", StatusID: 1, Position: 3},
		{ID: 4, Title: "D", StatusID: 1, Position: 4},
		{ID: 5, Title: "E", StatusID: 1, Position: 5},
	}
	out := MoveTasks(tasks, map[int]bool{3: true, 5: true}, MoveDestination{ParentID: 0, StatusID: 1, InsertAt: 3})
	got := groupedPositions(out, 0, 1)
	want := []int{1, 2, 3, 5, 4}
	if !equalInts(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
