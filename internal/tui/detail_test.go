package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/task"
)

// fields 0 件のとき detailRows は Title / Status / Tags / Files の 4 行。
func TestBuildDetailRowsNoFields(t *testing.T) {
	rows := buildDetailRows(nil)
	if len(rows) != 4 {
		t.Fatalf("len=%d, want 4 (Title/Status/Tags/Files)", len(rows))
	}
	if rows[0].kind != detailRowTitle ||
		rows[1].kind != detailRowStatus ||
		rows[2].kind != detailRowTags ||
		rows[3].kind != detailRowFiles {
		t.Errorf("rows kinds = %+v, want [Title Status Tags Files]", rows)
	}
}

// fields N 件のとき rows は Title, Status, Tags, field×N (position 順), Files の順。
func TestBuildDetailRowsWithFields(t *testing.T) {
	defs := task.FieldDefList{
		{ID: 2, Name: "second", Type: task.FieldTypeText, Position: 2},
		{ID: 1, Name: "first", Type: task.FieldTypeText, Position: 1},
		{ID: 3, Name: "third", Type: task.FieldTypeText, Position: 3},
	}
	rows := buildDetailRows(defs)
	if len(rows) != 7 {
		t.Fatalf("len=%d, want 7 (Title + Status + Tags + 3 fields + Files)", len(rows))
	}
	if rows[0].kind != detailRowTitle || rows[1].kind != detailRowStatus || rows[2].kind != detailRowTags {
		t.Errorf("first three should be Title/Status/Tags, got %+v", rows[:3])
	}
	wantIDs := []int{1, 2, 3} // position 順
	for i, want := range wantIDs {
		r := rows[3+i]
		if r.kind != detailRowField || r.fieldID != want {
			t.Errorf("rows[%d] = %+v, want field id=%d", 3+i, r, want)
		}
	}
	if rows[6].kind != detailRowFiles {
		t.Errorf("last should be Files, got %+v", rows[6])
	}
}
