package task

import (
	"errors"
	"testing"
)

func TestDefaultStatuses(t *testing.T) {
	sl := DefaultStatuses()
	if len(sl) != 3 {
		t.Fatalf("len=%d, want 3", len(sl))
	}
	want := []struct {
		id       int
		sequence int
		label    string
	}{
		{1, 1, "todo"},
		{2, 2, "doing"},
		{3, 3, "done"},
	}
	for i, w := range want {
		if sl[i].ID != w.id || sl[i].Sequence != w.sequence || sl[i].Label != w.label {
			t.Errorf("[%d]: got %+v, want id=%d sequence=%d label=%q", i, sl[i], w.id, w.sequence, w.label)
		}
		if sl[i].Color == "" {
			t.Errorf("[%d]: color must not be empty", i)
		}
	}
}

func TestStatusListSorted(t *testing.T) {
	sl := StatusList{
		{ID: 3, Sequence: 2, Label: "b"},
		{ID: 1, Sequence: 1, Label: "a"},
		{ID: 2, Sequence: 2, Label: "c"},
	}
	sorted := sl.Sorted()
	gotIDs := []int{sorted[0].ID, sorted[1].ID, sorted[2].ID}
	wantIDs := []int{1, 2, 3} // sequence 1, then sequence 2 with id 2 < 3
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("[%d]: got id=%d, want %d", i, gotIDs[i], wantIDs[i])
		}
	}
}

func TestStatusListNextPrev(t *testing.T) {
	sl := DefaultStatuses()
	if got := sl.NextID(1); got != 2 {
		t.Errorf("Next(1)=%d, want 2", got)
	}
	if got := sl.NextID(2); got != 3 {
		t.Errorf("Next(2)=%d, want 3", got)
	}
	if got := sl.NextID(3); got != 3 {
		t.Errorf("Next(3)=%d, want 3 (terminal)", got)
	}
	if got := sl.PrevID(1); got != 1 {
		t.Errorf("Prev(1)=%d, want 1 (terminal)", got)
	}
	if got := sl.PrevID(3); got != 2 {
		t.Errorf("Prev(3)=%d, want 2", got)
	}
}

func TestStatusListByID(t *testing.T) {
	sl := DefaultStatuses()
	s, ok := sl.ByID(2)
	if !ok || s.Label != "doing" {
		t.Errorf("ByID(2)=(%+v, %v), want doing,true", s, ok)
	}
	if _, ok := sl.ByID(99); ok {
		t.Error("ByID(99) should be not found")
	}
}

func TestStatusListAssignMissingIDs(t *testing.T) {
	sl := StatusList{
		{ID: 0, Sequence: 1, Label: "a"},
		{ID: 5, Sequence: 2, Label: "b"},
		{ID: 0, Sequence: 3, Label: "c"},
	}
	out, changed := sl.AssignMissingIDs()
	if !changed {
		t.Fatal("expected changed=true")
	}
	// max=5 -> 0番目=6, 2番目=7
	if out[0].ID != 6 {
		t.Errorf("[0]: got %d, want 6", out[0].ID)
	}
	if out[1].ID != 5 {
		t.Errorf("[1]: got %d, want 5 (unchanged)", out[1].ID)
	}
	if out[2].ID != 7 {
		t.Errorf("[2]: got %d, want 7", out[2].ID)
	}
}

func TestStatusListAssignMissingIDsNoChange(t *testing.T) {
	sl := DefaultStatuses()
	_, changed := sl.AssignMissingIDs()
	if changed {
		t.Error("expected changed=false when all ids are positive")
	}
}

func TestStatusListValidate(t *testing.T) {
	cases := []struct {
		name    string
		sl      StatusList
		wantErr error
	}{
		{"valid", DefaultStatuses(), nil},
		{"empty", StatusList{}, nil}, // 空リストの妥当性は呼び出し側 (storage 層) でチェック
		{"zero id", StatusList{{ID: 0, Label: "x"}}, ErrStatusInvalidID},
		{"empty label", StatusList{{ID: 1, Label: ""}}, ErrStatusEmptyLabel},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.sl.Validate()
			if c.wantErr == nil && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("expected %v, got %v", c.wantErr, err)
			}
		})
	}
}

func TestStatusListValidateDuplicateID(t *testing.T) {
	sl := StatusList{
		{ID: 1, Label: "a"},
		{ID: 1, Label: "b"},
	}
	err := sl.Validate()
	if err == nil {
		t.Fatal("expected error for duplicated id")
	}
}

func TestStatusListRenameByID(t *testing.T) {
	sl := DefaultStatuses()
	out, err := sl.RenameByID(2, "in-progress")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := out[1].Label; got != "in-progress" {
		t.Errorf("[1].Label=%q, want in-progress", got)
	}
	if sl[1].Label != "doing" {
		t.Error("source must remain unchanged")
	}
	if _, err := sl.RenameByID(99, "x"); err == nil {
		t.Error("expected error for missing id")
	}
	if _, err := sl.RenameByID(1, ""); err == nil {
		t.Error("expected error for empty label")
	}
}

func TestStatusListSetColorByID(t *testing.T) {
	sl := DefaultStatuses()
	out, err := sl.SetColorByID(1, "#abcdef")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got := out[0].Color; got != "#abcdef" {
		t.Errorf("[0].Color=%q, want #abcdef", got)
	}
	if _, err := sl.SetColorByID(99, "#000000"); err == nil {
		t.Error("expected error for missing id")
	}
}

func TestStatusListInsertAt(t *testing.T) {
	sl := DefaultStatuses()
	out, newID, err := sl.InsertAt(1, "review", "#fab387")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if newID != 4 {
		t.Errorf("newID=%d, want 4", newID)
	}
	// Sorted: todo(1), review(4), doing(2), done(3)、sequence は 1..4 に振り直し
	if out[0].Label != "todo" || out[0].Sequence != 1 {
		t.Errorf("[0]=%+v, want todo seq=1", out[0])
	}
	if out[1].Label != "review" || out[1].Sequence != 2 || out[1].ID != 4 {
		t.Errorf("[1]=%+v, want review id=4 seq=2", out[1])
	}
	if out[2].Label != "doing" || out[2].Sequence != 3 {
		t.Errorf("[2]=%+v, want doing seq=3", out[2])
	}
	if out[3].Label != "done" || out[3].Sequence != 4 {
		t.Errorf("[3]=%+v, want done seq=4", out[3])
	}
	if _, _, err := sl.InsertAt(0, "", "#000000"); err == nil {
		t.Error("expected error for empty label")
	}
}

func TestStatusListMoveStatusUp(t *testing.T) {
	sl := DefaultStatuses() // todo(1)/doing(2)/done(3)
	out := sl.MoveStatusUp(2)
	// Sorted: doing(2), todo(1), done(3)、sequence は 1..3 に再採番
	sorted := out.Sorted()
	if sorted[0].ID != 2 || sorted[0].Sequence != 1 {
		t.Errorf("[0]=%+v, want id=2 seq=1", sorted[0])
	}
	if sorted[1].ID != 1 || sorted[1].Sequence != 2 {
		t.Errorf("[1]=%+v, want id=1 seq=2", sorted[1])
	}
	if sorted[2].ID != 3 || sorted[2].Sequence != 3 {
		t.Errorf("[2]=%+v, want id=3 seq=3", sorted[2])
	}
}

func TestStatusListMoveStatusUpAtTopNoOp(t *testing.T) {
	sl := DefaultStatuses()
	out := sl.MoveStatusUp(1) // 先頭
	for i, s := range out.Sorted() {
		want := sl.Sorted()[i]
		if s.ID != want.ID || s.Sequence != want.Sequence {
			t.Errorf("[%d]: got %+v, want %+v", i, s, want)
		}
	}
}

func TestStatusListMoveStatusDown(t *testing.T) {
	sl := DefaultStatuses()
	out := sl.MoveStatusDown(2) // doing → done と入れ替え
	sorted := out.Sorted()
	if sorted[0].ID != 1 || sorted[0].Sequence != 1 {
		t.Errorf("[0]=%+v, want id=1 seq=1", sorted[0])
	}
	if sorted[1].ID != 3 || sorted[1].Sequence != 2 {
		t.Errorf("[1]=%+v, want id=3 seq=2", sorted[1])
	}
	if sorted[2].ID != 2 || sorted[2].Sequence != 3 {
		t.Errorf("[2]=%+v, want id=2 seq=3", sorted[2])
	}
}

func TestStatusListMoveStatusDownAtBottomNoOp(t *testing.T) {
	sl := DefaultStatuses()
	out := sl.MoveStatusDown(3) // 末尾
	for i, s := range out.Sorted() {
		want := sl.Sorted()[i]
		if s.ID != want.ID || s.Sequence != want.Sequence {
			t.Errorf("[%d]: got %+v, want %+v", i, s, want)
		}
	}
}

func TestStatusListDeleteByID_MiddleFallsBackToNext(t *testing.T) {
	sl := DefaultStatuses() // 1=todo seq1, 2=doing seq2, 3=done seq3
	out, fb, err := sl.DeleteByID(2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// doing が消えて、fallback は sequence が 1 つ大きい (= 表示上 1 つ下) done (id 3)
	if fb != 3 {
		t.Errorf("fallback=%d, want 3 (done)", fb)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	// 残りは todo(seq=1), done(seq=2) に再採番
	sorted := out.Sorted()
	if sorted[0].ID != 1 || sorted[0].Sequence != 1 {
		t.Errorf("[0]=%+v, want id=1 seq=1", sorted[0])
	}
	if sorted[1].ID != 3 || sorted[1].Sequence != 2 {
		t.Errorf("[1]=%+v, want id=3 seq=2", sorted[1])
	}
}

func TestStatusListDeleteByID_TopFallsBackToNext(t *testing.T) {
	sl := DefaultStatuses()
	out, fb, err := sl.DeleteByID(1) // 最小 sequence の todo を削除
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// fallback は sequence が 1 つ大きい doing (id 2)
	if fb != 2 {
		t.Errorf("fallback=%d, want 2 (doing)", fb)
	}
	sorted := out.Sorted()
	if sorted[0].ID != 2 || sorted[0].Sequence != 1 {
		t.Errorf("[0]=%+v, want id=2 seq=1", sorted[0])
	}
	if sorted[1].ID != 3 || sorted[1].Sequence != 2 {
		t.Errorf("[1]=%+v, want id=3 seq=2", sorted[1])
	}
}

func TestStatusListDeleteByID_BottomFallsBackToPrev(t *testing.T) {
	sl := DefaultStatuses()
	// 最大 sequence の done を削除すると次が無いので、1 つ前 (sequence 値が小さい) の doing にフォールバック。
	out, fb, err := sl.DeleteByID(3)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fb != 2 {
		t.Errorf("fallback=%d, want 2 (doing)", fb)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
}

func TestStatusListDeleteByID_LastReturnsError(t *testing.T) {
	sl := StatusList{{ID: 1, Sequence: 1, Label: "todo"}}
	_, _, err := sl.DeleteByID(1)
	if !errors.Is(err, ErrCannotDeleteLastStatus) {
		t.Errorf("expected ErrCannotDeleteLastStatus, got %v", err)
	}
}

func TestStatusListDeleteByID_NotFound(t *testing.T) {
	sl := DefaultStatuses()
	_, _, err := sl.DeleteByID(99)
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestStatusListInsertAtEmpty(t *testing.T) {
	sl := StatusList{}
	out, newID, err := sl.InsertAt(0, "todo", "#6c7086")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if newID != 1 {
		t.Errorf("newID=%d, want 1", newID)
	}
	if len(out) != 1 || out[0].Label != "todo" || out[0].Sequence != 1 {
		t.Errorf("got %+v, want single todo seq=1", out)
	}
}
