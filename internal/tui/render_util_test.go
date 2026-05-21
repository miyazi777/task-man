package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// renderSingleLineRow の基本動作:
//   - 短い → 末尾 padding で width に揃う
//   - 同じ → そのまま
//   - 長い → ansi.Truncate で width に切り詰め (末尾は "…")
//
// いずれのケースも結果が 1 行 (= width cell ぴったり) であることを確認する。
func TestRenderSingleLineRow_PlainText(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		width int
	}{
		{"short", "abc", 10},
		{"equal", "abcde", 5},
		{"long", "abcdefghij", 5},
		{"single rune", "a", 5},
		{"japanese short", "あい", 10},
		{"japanese long", "あいうえおか", 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := renderSingleLineRow(lipgloss.NewStyle(), c.raw, c.width)
			stripped := ansi.Strip(out)
			if strings.Contains(stripped, "\n") {
				t.Fatalf("output should be single-line, got %q", stripped)
			}
			if got := ansi.StringWidth(stripped); got != c.width {
				t.Errorf("width: got %d, want %d (out=%q)", got, c.width, stripped)
			}
		})
	}
}

// raw に ANSI escape sequence が含まれている場合でも、結果は単一行で正確な cell 幅。
// (truncate が rune 単位で切る前提だと escape を破壊する可能性があり、ANSI 認識の
// ansi.Truncate を使っていることを担保するための回帰テスト。)
func TestRenderSingleLineRow_ANSI(t *testing.T) {
	colored := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("RED-CONTENT")
	raw := "prefix " + colored + " suffix"
	out := renderSingleLineRow(lipgloss.NewStyle(), raw, 10)
	stripped := ansi.Strip(out)
	if strings.Contains(stripped, "\n") {
		t.Fatalf("ANSI input should not produce multi-line output, got %q", stripped)
	}
	if got := ansi.StringWidth(stripped); got != 10 {
		t.Errorf("width: got %d, want 10 (stripped=%q)", got, stripped)
	}
}

// width <= 0 は空文字を返す (描画範囲が無い)。
func TestRenderSingleLineRow_ZeroWidth(t *testing.T) {
	if out := renderSingleLineRow(lipgloss.NewStyle(), "anything", 0); out != "" {
		t.Errorf("zero width: want empty, got %q", out)
	}
	if out := renderSingleLineRow(lipgloss.NewStyle(), "anything", -3); out != "" {
		t.Errorf("negative width: want empty, got %q", out)
	}
}
