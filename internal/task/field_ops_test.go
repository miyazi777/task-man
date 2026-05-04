package task

import (
	"errors"
	"testing"
)

func TestPurgeRemovedFieldValues(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "t1", StatusID: 1, Fields: TaskFieldList{
			{ID: 1, FieldID: 10, Value: "a"},
			{ID: 2, FieldID: 20, Value: "b"},
		}},
		{ID: 2, Title: "t2", StatusID: 1, Fields: TaskFieldList{
			{ID: 1, FieldID: 10, Value: "c"},
			{ID: 2, FieldID: 99, Value: "orphan"}, // defs に無い
		}},
	}
	defs := FieldDefList{
		{ID: 10, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 20, Name: "b", Type: FieldTypeText, Position: 2},
	}
	out := PurgeRemovedFieldValues(tasks, defs)
	if len(out[0].Fields) != 2 {
		t.Errorf("task 1: kept %d fields, want 2", len(out[0].Fields))
	}
	if len(out[1].Fields) != 1 {
		t.Errorf("task 2: kept %d fields, want 1", len(out[1].Fields))
	}
	// 元の slice は不変
	if len(tasks[1].Fields) != 2 {
		t.Errorf("original tasks should be unchanged, got len=%d", len(tasks[1].Fields))
	}
}

func TestSetFieldValue(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "t1", StatusID: 1, Fields: nil},
		{ID: 2, Title: "t2", StatusID: 1, Fields: TaskFieldList{
			{ID: 1, FieldID: 10, Value: "old"},
		}},
	}
	defs := FieldDefList{
		{ID: 10, Name: "a", Type: FieldTypeText, Position: 1},
	}
	// 既存タスクへの新規 field 値追加
	out, err := SetFieldValue(tasks, defs, 1, 10, "new")
	if err != nil {
		t.Fatalf("set err: %v", err)
	}
	if len(out[0].Fields) != 1 || out[0].Fields[0].Value != "new" {
		t.Errorf("task 1 fields = %+v", out[0].Fields)
	}
	// 他タスクは不変
	if len(out[1].Fields) != 1 || out[1].Fields[0].Value != "old" {
		t.Errorf("task 2 should be unchanged, got %+v", out[1].Fields)
	}
	// 既存値の上書き
	out2, _ := SetFieldValue(tasks, defs, 2, 10, "updated")
	if out2[1].Fields[0].Value != "updated" {
		t.Errorf("task 2 update failed, got %+v", out2[1].Fields)
	}
	// 未知の field_id
	if _, err := SetFieldValue(tasks, defs, 1, 99, "x"); !errors.Is(err, ErrFieldUnknownFieldID) {
		t.Errorf("unknown field_id should be ErrFieldUnknownFieldID, got %v", err)
	}
	// 未知の task id
	if _, err := SetFieldValue(tasks, defs, 999, 10, "x"); err == nil {
		t.Errorf("unknown task id should error")
	}
}

func TestValidateAll(t *testing.T) {
	statuses := StatusList{{ID: 1, Sequence: 1, Label: "todo"}}
	defs := FieldDefList{
		{ID: 10, Name: "a", Type: FieldTypeText, Position: 1},
	}
	good := []Task{
		{ID: 1, Title: "t", StatusID: 1, Fields: TaskFieldList{
			{ID: 1, FieldID: 10, Value: "x"},
		}},
	}
	if err := ValidateAll(good, statuses, defs); err != nil {
		t.Errorf("good should be valid, got %v", err)
	}
	bad := []Task{
		{ID: 1, Title: "t", StatusID: 1, Fields: TaskFieldList{
			{ID: 1, FieldID: 99, Value: "x"},
		}},
	}
	if err := ValidateAll(bad, statuses, defs); err == nil {
		t.Errorf("bad should error")
	}
}
