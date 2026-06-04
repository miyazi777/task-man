package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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

// #18 の回帰テスト: 詳細画面 Title 行カーソルが narrow width でも分裂しない。
// 元コードでは styleCursorRow.Width(width).Render(raw) が長いタイトルを word-wrap
// して詳細ペイン全体が下方向に押し下げられる現象があった。
func TestRenderDetailTitleCursorRowNarrowWidthSingleLine(t *testing.T) {
	statuses := task.DefaultStatuses()
	tk := task.Task{ID: 1, Title: "this-is-a-very-long-title-that-easily-overflows-narrow-pane-widths", StatusID: 1}
	rows := buildDetailRows(nil)
	// Title 行は detailRows[0]。focused かつ cursor=0 でカーソル反転表示。
	for _, w := range []int{40, 24, 16, 10} {
		out := renderDetail(&tk, statuses, nil, nil, rows, true, 0, w, 4)
		stripped := strings.TrimRight(ansi.Strip(out), " \n")
		// 1 行が w を超える幅で書かれた場合 lipgloss が wrap し、行数が増える。
		// 期待: ちょうど 4 (height) 行に収まる。
		lines := strings.Split(stripped, "\n")
		if len(lines) > 4 {
			t.Errorf("width=%d: lines=%d > height=4 (word-wrap regression):\n%s", w, len(lines), stripped)
		}
	}
}

// issue #40: Files ヘッダに taskDir を渡すとパスが "Files:" の右側に並ぶ。
func TestRenderFilesHeaderShowsTaskDir(t *testing.T) {
	taskDir := "/tmp/wsp/data/task-7"
	out := renderFilesHeader(taskDir, 80)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Files:") {
		t.Errorf("output should contain 'Files:' label, got %q", plain)
	}
	if !strings.Contains(plain, taskDir) {
		t.Errorf("output should contain task dir %q, got %q", taskDir, plain)
	}
	// "Files:" がパスの左にあること (ラベル → パスの順)。
	labelIdx := strings.Index(plain, "Files:")
	pathIdx := strings.Index(plain, taskDir)
	if labelIdx < 0 || pathIdx < 0 || labelIdx >= pathIdx {
		t.Errorf("'Files:' should appear before the path, got %q", plain)
	}
}

// issue #40: taskDir が空文字 (= カーソル下にタスクが無い) なら path は表示せず "Files:" のみ。
func TestRenderFilesHeaderEmptyTaskDir(t *testing.T) {
	out := renderFilesHeader("", 80)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Files:") {
		t.Errorf("output should still contain 'Files:' label, got %q", plain)
	}
	// "Files:" の右側 (trim 後) には空白しか残らない。
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(plain), "Files:"))
	if rest != "" {
		t.Errorf("expected nothing after 'Files:', got %q", rest)
	}
}

// issue #40: 横幅を超える長いパスは ansi.Truncate("…") で右端が省略される。
// 1 行に収まり (= word-wrap しない) かつ末尾が "…" になることを確認する。
func TestRenderFilesHeaderTruncatesLongPath(t *testing.T) {
	taskDir := "/tmp/very/long/path/that/easily/overflows/the/narrow/right/pane/width/task-9999"
	width := 40
	out := renderFilesHeader(taskDir, width)
	plain := ansi.Strip(out)
	// 1 行に収まる (width 強制で改行は起きない)。
	lines := strings.Split(strings.TrimRight(plain, " \n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d:\n%s", len(lines), plain)
	}
	// 描画幅が width に丸まる。
	if got := ansi.StringWidth(plain); got != width {
		t.Errorf("rendered width=%d, want %d (plain=%q)", got, width, plain)
	}
	// 末尾は ellipsis (renderSingleLineRow が "…" でクランプするため)。
	if !strings.HasSuffix(strings.TrimRight(plain, " "), "…") {
		t.Errorf("expected suffix '…' after truncation, got %q", plain)
	}
}

// #18 の回帰テスト: file 行カーソルが narrow width でも 2 行に分裂しない。
// 元コードでは styleCursorRow.Width(width).Render(raw) が word-wrap し、
// 後続行が押し下げられて height で切れる現象があった。
func TestRenderFileNamesListNarrowWidthSingleLine(t *testing.T) {
	rows := []fileRow{
		{name: "very-long-filename-that-overflows-narrow-width.md", relPath: "very-long-filename-that-overflows-narrow-width.md"},
	}
	for _, w := range []int{30, 20, 14, 10, 6} {
		out := renderFileNamesList(rows, true, true, 0, w, 1)
		stripped := strings.TrimRight(ansi.Strip(out), " \n")
		lines := strings.Split(stripped, "\n")
		if len(lines) != 1 {
			t.Errorf("width=%d: cursor row should stay on 1 line, got %d:\n%s", w, len(lines), stripped)
		}
	}
}
