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
