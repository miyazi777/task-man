package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPlaceOverlayPlain(t *testing.T) {
	bg := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbbbbbbb",
		"cccccccccc",
		"dddddddddd",
	}, "\n")
	fg := strings.Join([]string{
		"XX",
		"YY",
	}, "\n")

	got := PlaceOverlay(2, 1, fg, bg)
	// ANSI エスケープを取り除いた表示内容で比較する。
	gotPlain := ansi.Strip(got)
	want := strings.Join([]string{
		"aaaaaaaaaa",
		"bbXXbbbbbb",
		"ccYYcccccc",
		"dddddddddd",
	}, "\n")
	if gotPlain != want {
		t.Errorf("\ngot (stripped):\n%q\nwant:\n%q", gotPlain, want)
	}
}

func TestPlaceOverlayBeyondBg(t *testing.T) {
	bg := "abc"
	fg := "X"
	got := PlaceOverlay(5, 0, fg, bg)
	if ansi.StringWidth(got) != 6 {
		t.Errorf("expected width 6 (3 + 2 padding + 1 X), got %d (%q)", ansi.StringWidth(got), got)
	}
}

func TestPlaceOverlayPreservesUntouchedLines(t *testing.T) {
	bg := strings.Join([]string{"line1", "line2", "line3"}, "\n")
	fg := "X"
	got := PlaceOverlay(2, 1, fg, bg)
	lines := strings.Split(got, "\n")
	if lines[0] != "line1" || lines[2] != "line3" {
		t.Errorf("untouched lines modified: %q", lines)
	}
}
