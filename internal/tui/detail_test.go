package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/task"
)

// fields 0 件のとき detailRows は Title / Status / Files の 3 行。
func TestBuildDetailRowsNoFields(t *testing.T) {
	rows := buildDetailRows(nil)
	if len(rows) != 3 {
		t.Fatalf("len=%d, want 3 (Title/Status/Files)", len(rows))
	}
	if rows[0].kind != detailRowTitle ||
		rows[1].kind != detailRowStatus ||
		rows[2].kind != detailRowFiles {
		t.Errorf("rows kinds = %+v, want [Title Status Files]", rows)
	}
}

// fields N 件のとき rows は Title, Status, field×N (position 順), Files の順。
func TestBuildDetailRowsWithFields(t *testing.T) {
	defs := task.FieldDefList{
		{ID: 2, Name: "second", Type: task.FieldTypeText, Position: 2},
		{ID: 1, Name: "first", Type: task.FieldTypeText, Position: 1},
		{ID: 3, Name: "third", Type: task.FieldTypeText, Position: 3},
	}
	rows := buildDetailRows(defs)
	if len(rows) != 6 {
		t.Fatalf("len=%d, want 6 (Title + Status + 3 fields + Files)", len(rows))
	}
	if rows[0].kind != detailRowTitle || rows[1].kind != detailRowStatus {
		t.Errorf("first two should be Title/Status, got %+v", rows[:2])
	}
	wantIDs := []int{1, 2, 3} // position 順
	for i, want := range wantIDs {
		r := rows[2+i]
		if r.kind != detailRowField || r.fieldID != want {
			t.Errorf("rows[%d] = %+v, want field id=%d", 2+i, r, want)
		}
	}
	if rows[5].kind != detailRowFiles {
		t.Errorf("last should be Files, got %+v", rows[5])
	}
}

// detailFilesDividerRow は fields 数 + tags 表示行数に応じて変動。
// tags 0 行のときは従来どおり 5 + N。tags 1 行以上で 5 + N + L。
func TestDetailFilesDividerRow(t *testing.T) {
	cases := []struct {
		fields    int
		tagsLines int
		want      int
	}{
		{0, 0, 5},
		{1, 0, 6},
		{3, 0, 8},
		{0, 1, 6},
		{2, 2, 9},
	}
	for _, c := range cases {
		defs := make(task.FieldDefList, c.fields)
		for i := 0; i < c.fields; i++ {
			defs[i] = task.FieldDef{ID: i + 1, Name: "x", Type: task.FieldTypeText, Position: i + 1}
		}
		rows := buildDetailRows(defs)
		if got := detailFilesDividerRow(rows, c.tagsLines); got != c.want {
			t.Errorf("fields=%d tags=%d: got %d, want %d", c.fields, c.tagsLines, got, c.want)
		}
	}
}
