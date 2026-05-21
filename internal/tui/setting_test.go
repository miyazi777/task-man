package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

// #18 の回帰テスト: 設定画面 status pane のカーソル行が narrow width でも分裂しない。
// 元コードでは cursorStyleFor(...).Width(width).Render(raw) が長いラベルを word-wrap
// していた。renderStatusSettingRow を直接呼んで結果が単一行であることを確認する。
func TestRenderStatusSettingRowNarrowWidthSingleLine(t *testing.T) {
	s := task.Status{ID: 1, Sequence: 1, Label: "long-label-that-overflows-narrow-width", Color: "#888888"}
	for _, w := range []int{30, 20, 14, 8} {
		out := renderStatusSettingRow(s, true, false, w)
		stripped := strings.TrimRight(ansi.Strip(out), " \n")
		lines := strings.Split(stripped, "\n")
		if len(lines) != 1 {
			t.Errorf("highlight width=%d: should stay single-line, got %d:\n%s", w, len(lines), stripped)
		}
		if got := ansi.StringWidth(stripped); got > w {
			t.Errorf("highlight width=%d: result width %d exceeds limit (stripped=%q)", w, got, stripped)
		}
	}
	// 非ハイライト行も同様にチェック。
	for _, w := range []int{30, 20, 14, 8} {
		out := renderStatusSettingRow(s, false, false, w)
		stripped := strings.TrimRight(ansi.Strip(out), " \n")
		lines := strings.Split(stripped, "\n")
		if len(lines) != 1 {
			t.Errorf("non-highlight width=%d: should stay single-line, got %d:\n%s", w, len(lines), stripped)
		}
	}
}

// #18 の回帰テスト: 設定画面の左メニューが narrow width でも分裂しない。
func TestRenderSettingMenuNarrowWidthSingleLines(t *testing.T) {
	for _, w := range []int{20, 12, 8, 4} {
		out := renderSettingMenu(0, true, w, 10)
		// menu は複数行 (各設定項目 1 行ずつ) なので各行の幅を個別に確認。
		for i, line := range strings.Split(ansi.Strip(out), "\n") {
			if got := ansi.StringWidth(line); got > w {
				t.Errorf("width=%d line[%d]: cell width %d exceeds limit (line=%q)", w, i, got, line)
			}
		}
	}
}

// #18 の回帰テスト: file_opener attribute pane のカーソル行が narrow width で
// 分裂しないこと。長い application 名で applications 行が押し出されるケースを想定。
func TestRenderSettingFileOpenerAttributePaneNarrowWidth(t *testing.T) {
	apps := []storage.Application{
		{ID: 1, Name: "very-long-application-name", Run: "/usr/local/bin/very-long-application-name"},
	}
	openers := []storage.FileOpener{
		{Extension: "md", ApplicationIDs: []int{1}, DefaultApp: 1},
	}
	for _, w := range []int{40, 24, 16, 10} {
		out := renderSettingFileOpenerAttributePane(openers, apps, 0, 1, true, w, 8)
		for i, line := range strings.Split(ansi.Strip(out), "\n") {
			if got := ansi.StringWidth(line); got > w {
				t.Errorf("width=%d line[%d]: cell width %d exceeds limit (line=%q)", w, i, got, line)
			}
		}
	}
}
