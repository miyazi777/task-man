package task

import (
	"reflect"
	"sort"
	"testing"
)

func TestSubtreeIDs(t *testing.T) {
	tasks := []Task{
		{ID: 1, StatusID: 1, Position: 1},
		{ID: 2, StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, StatusID: 1, ParentID: 2, Position: 1},
		{ID: 4, StatusID: 1, Position: 2},
	}
	got := SubtreeIDs(tasks, 1)
	sort.Ints(got)
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SubtreeIDs(1) = %v want %v", got, want)
	}
}

func TestTrashTask_FlagsSubtreeKeepingStatus(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "Sib", StatusID: 1, Position: 2},
	}
	out := TrashTask(tasks, 1)
	// P と C は IsTrashBox=true、status_id は変わらず。Sib は影響なし。
	for _, id := range []int{1, 2} {
		idx := taskIndexByID(out, id)
		if !out[idx].IsTrashBox {
			t.Errorf("task %d should be trashed", id)
		}
		if out[idx].StatusID != 1 {
			t.Errorf("task %d status changed: got %d want 1", id, out[idx].StatusID)
		}
	}
	if out[taskIndexByID(out, 3)].IsTrashBox {
		t.Errorf("Sib should not be trashed")
	}
}

func TestRestoreTask_ClearsSubtreeFlag(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1, IsTrashBox: true},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, Position: 1, IsTrashBox: true},
	}
	out := RestoreTask(tasks, 1)
	for _, id := range []int{1, 2} {
		idx := taskIndexByID(out, id)
		if out[idx].IsTrashBox {
			t.Errorf("task %d should be restored", id)
		}
	}
}

func TestRestoreTask_NotInTrash_NoOp(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1, Position: 1},
	}
	out := RestoreTask(tasks, 1)
	if out[0].IsTrashBox {
		t.Errorf("flag should remain false")
	}
}

func TestTrashRootID_FindsTopmostTrashedAncestor(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, IsTrashBox: true},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, IsTrashBox: true},
		{ID: 3, Title: "GC", StatusID: 1, ParentID: 2, IsTrashBox: true},
	}
	if got := TrashRootID(tasks, 3); got != 1 {
		t.Errorf("TrashRootID(3) = %d want 1 (P is the topmost trashed)", got)
	}
}

func TestTrashRootID_StopsAtNonTrashedParent(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, IsTrashBox: false},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, IsTrashBox: true},
	}
	// P は trash 外なので C が trash root。
	if got := TrashRootID(tasks, 2); got != 2 {
		t.Errorf("TrashRootID(2) = %d want 2", got)
	}
}

func TestTrashRootID_NonTrashedTaskReturnsItself(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "A", StatusID: 1},
	}
	if got := TrashRootID(tasks, 1); got != 1 {
		t.Errorf("TrashRootID(1) = %d want 1", got)
	}
}

func TestDeleteTaskSubtree_RemovesAllDescendants(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "P", StatusID: 1, Position: 1},
		{ID: 2, Title: "C", StatusID: 1, ParentID: 1, Position: 1},
		{ID: 3, Title: "Sib", StatusID: 1, Position: 2},
	}
	out, removed := DeleteTaskSubtree(tasks, 1)
	if len(out) != 1 {
		t.Fatalf("len(out) = %d want 1", len(out))
	}
	if out[0].ID != 3 || out[0].Position != 1 {
		t.Fatalf("remaining = %+v want Sib at pos 1", out[0])
	}
	sort.Ints(removed)
	if !reflect.DeepEqual(removed, []int{1, 2}) {
		t.Fatalf("removed = %v want [1 2]", removed)
	}
}
