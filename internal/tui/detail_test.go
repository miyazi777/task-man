package tui

import (
	"strings"
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

// renderFileNamesList はディレクトリ行に折りたたみマーカー (+ / -) と "/" 付き表示を出す。
// 葉ファイルには 2 cell の空白マーカーを使い、インデントは depth に応じて 2 cell ずつ加算する。
func TestRenderFileNamesListWithDirectories(t *testing.T) {
	rows := []fileRow{
		{name: "memo.md", relPath: "memo.md"},
		{name: "open", relPath: "open", isDir: true, hasChildren: true, collapsed: false},
		{name: "inner.md", relPath: "open/inner.md", depth: 1},
		{name: "closed", relPath: "closed", isDir: true, hasChildren: true, collapsed: true},
		{name: "empty", relPath: "empty", isDir: true, hasChildren: false},
	}
	out := renderFileNamesList(rows, true, false, 0, 40, 10)
	lines := strings.Split(out, "\n")
	if len(lines) < 5 {
		t.Fatalf("expected >=5 lines, got %d:\n%s", len(lines), out)
	}
	// 0: memo.md (葉ファイル、トップレベル)
	if !strings.Contains(lines[0], "memo.md") || strings.Contains(lines[0], "memo.md/") {
		t.Errorf("line0 want plain file 'memo.md', got %q", lines[0])
	}
	// 1: open/ + "- " マーカー
	if !strings.Contains(lines[1], "- open/") {
		t.Errorf("line1 want expanded dir '- open/', got %q", lines[1])
	}
	// 2: inner.md (depth=1 → 追加で 2 cell インデント)
	if !strings.Contains(lines[2], "inner.md") {
		t.Errorf("line2 want 'inner.md', got %q", lines[2])
	}
	if strings.Index(lines[2], "inner.md") <= strings.Index(lines[1], "open/") {
		t.Errorf("line2 inner.md should be indented further than line1 open/: %q vs %q", lines[2], lines[1])
	}
	// 3: closed/ + "+ " マーカー
	if !strings.Contains(lines[3], "+ closed/") {
		t.Errorf("line3 want collapsed dir '+ closed/', got %q", lines[3])
	}
	// 4: empty/ (子無し、マーカー無し)
	if !strings.Contains(lines[4], "empty/") || strings.Contains(lines[4], "+ empty/") || strings.Contains(lines[4], "- empty/") {
		t.Errorf("line4 want empty dir 'empty/' without marker, got %q", lines[4])
	}
}

// 0 件のとき "(no files)" を表示し、ディレクトリのときと識別できる。
func TestRenderFileNamesListEmpty(t *testing.T) {
	out := renderFileNamesList(nil, true, false, 0, 40, 3)
	if !strings.Contains(out, "(no files)") {
		t.Errorf("expected '(no files)' placeholder, got:\n%s", out)
	}
}
