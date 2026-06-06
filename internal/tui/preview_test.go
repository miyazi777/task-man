package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 対象拡張子 (.md / .txt) ではファイル内容が描画結果に含まれること、
// それ以外では "Preview not available" が表示されることを検証する。
func TestRenderPreview(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mustWrite := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(taskDir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	mustWrite("memo.md", "hello world\nsecond line")
	mustWrite("note.txt", "plain text content")
	mustWrite("data.csv", "a,b,c")
	mustWrite("script.go", "package main")

	// 80x10 のペインで描画。プレビュー対象は中身を含む。
	if got := renderPreview(dir, "", 1, "memo.md", 80, 10); !strings.Contains(got, "hello world") {
		t.Errorf(".md preview should contain content, got:\n%s", got)
	}
	if got := renderPreview(dir, "", 1, "note.txt", 80, 10); !strings.Contains(got, "plain text content") {
		t.Errorf(".txt preview should contain content, got:\n%s", got)
	}
	// 対象外: メッセージのみ。
	if got := renderPreview(dir, "", 1, "data.csv", 80, 10); !strings.Contains(got, previewNotAvailableMessage) {
		t.Errorf(".csv preview should show not-available message, got:\n%s", got)
	}
	if got := renderPreview(dir, "", 1, "script.go", 80, 10); !strings.Contains(got, previewNotAvailableMessage) {
		t.Errorf(".go preview should show not-available message, got:\n%s", got)
	}
	// 大文字拡張子も対象扱い。
	mustWrite("UPPER.MD", "uppercase ext")
	if got := renderPreview(dir, "", 1, "UPPER.MD", 80, 10); !strings.Contains(got, "uppercase ext") {
		t.Errorf("uppercase .MD preview should contain content, got:\n%s", got)
	}
	// 空ファイル名 / taskID=0 は何も出さない (スタイル分の空白のみ)。
	got := renderPreview(dir, "", 0, "", 80, 10)
	if strings.Contains(got, previewNotAvailableMessage) {
		t.Errorf("empty preview should not show not-available message, got:\n%s", got)
	}

	// サブディレクトリ配下のファイルも relPath ("sub/inner.md") で表示できる。
	subDir := filepath.Join(taskDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inner.md"), []byte("nested content"), 0o644); err != nil {
		t.Fatalf("write inner.md: %v", err)
	}
	if got := renderPreview(dir, "", 1, "sub/inner.md", 80, 10); !strings.Contains(got, "nested content") {
		t.Errorf("nested preview should contain content, got:\n%s", got)
	}
}

// renderDirPreview の Top-Level 表示: 直下のファイルとサブディレクトリが描画され、
// サブディレクトリは末尾スラッシュ、ファイルはそのまま、Name 昇順で並ぶ。
func TestRenderDirPreviewContents(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"alpha.md", "zebra.txt"} {
		if err := os.WriteFile(filepath.Join(taskDir, name), nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(taskDir, "mid"), 0o755); err != nil {
		t.Fatalf("mkdir mid: %v", err)
	}

	got := renderDirPreview(dir, "", 1, ".", 80, 10)
	for _, want := range []string{"alpha.md", "mid/", "zebra.txt"} {
		if !strings.Contains(got, want) {
			t.Errorf("preview should contain %q, got:\n%s", want, got)
		}
	}
	// Name 昇順: alpha < mid < zebra。strings.Index で前後関係を確認。
	if strings.Index(got, "alpha.md") > strings.Index(got, "mid/") {
		t.Errorf("alpha.md should appear before mid/, got:\n%s", got)
	}
	if strings.Index(got, "mid/") > strings.Index(got, "zebra.txt") {
		t.Errorf("mid/ should appear before zebra.txt, got:\n%s", got)
	}
}

// サブディレクトリを指定したとき、直下のエントリのみが描画される (再帰しない)。
func TestRenderDirPreviewSubDir(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task-1")
	subDir := filepath.Join(taskDir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "top.md"), nil, 0o644); err != nil {
		t.Fatalf("write top: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "inner.md"), nil, 0o644); err != nil {
		t.Fatalf("write inner: %v", err)
	}

	got := renderDirPreview(dir, "", 1, "sub", 80, 10)
	if !strings.Contains(got, "inner.md") {
		t.Errorf("preview should contain inner.md, got:\n%s", got)
	}
	if strings.Contains(got, "top.md") {
		t.Errorf("preview should not contain top.md, got:\n%s", got)
	}
}

// 空ディレクトリでは "(empty)" が表示される。
func TestRenderDirPreviewEmpty(t *testing.T) {
	dir := t.TempDir()
	taskDir := filepath.Join(dir, "task-1", "emptydir")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got := renderDirPreview(dir, "", 1, "emptydir", 80, 10)
	if !strings.Contains(got, "(empty)") {
		t.Errorf("preview should contain (empty), got:\n%s", got)
	}
}

// 存在しない relPath では "(read error)" が表示される。
func TestRenderDirPreviewReadError(t *testing.T) {
	dir := t.TempDir()
	got := renderDirPreview(dir, "", 1, "missing", 80, 10)
	if !strings.Contains(got, "(read error)") {
		t.Errorf("preview should contain (read error), got:\n%s", got)
	}
}

// taskID=0 / relPath="" は空ペイン: "(empty)" も "(read error)" も含まない。
func TestRenderDirPreviewEmptyArgs(t *testing.T) {
	dir := t.TempDir()
	got := renderDirPreview(dir, "", 0, "", 80, 10)
	if strings.Contains(got, "(empty)") || strings.Contains(got, "(read error)") {
		t.Errorf("empty args should render empty pane, got:\n%s", got)
	}
}

// previewLines が height 行で切り、tab を 4 スペース展開し、width で切り詰めることを検証。
func TestPreviewLines(t *testing.T) {
	content := "abcdefghij\n\tindented\nthird\nfourth"
	lines := previewLines(content, 5, 3)
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d (%v)", len(lines), lines)
	}
	if lines[0] != "abcde" {
		t.Errorf("line0 want %q, got %q", "abcde", lines[0])
	}
	// "\tindented" → "    indented" → width=5 で "    i"
	if lines[1] != "    i" {
		t.Errorf("line1 want %q, got %q", "    i", lines[1])
	}
	if lines[2] != "third" {
		t.Errorf("line2 want %q, got %q", "third", lines[2])
	}
}
