package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// issue #34: ModeList の footer に R:refresh ヒントが含まれる。
// ゴミ箱ビュー (viewTrash=true) では編集系操作が無いため refresh ヒントも省く。
func TestRenderFooterModeListContainsRefreshHint(t *testing.T) {
	out := renderFooter(ModeList, ModeList, false, false, false, layoutFocusTaskList, 200)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "R:refresh") {
		t.Errorf("ModeList footer should contain R:refresh hint, got %q", plain)
	}

	trashOut := renderFooter(ModeList, ModeList, false, false, true, layoutFocusTaskList, 200)
	if strings.Contains(ansi.Strip(trashOut), "R:refresh") {
		t.Errorf("ModeList trash-view footer should NOT contain R:refresh, got %q", ansi.Strip(trashOut))
	}
}

// issue #34: ModeDetail の Files 行で R:refresh ヒントが見える。
// 非 Files 行 (Title/Status/Tags 等) ではヒントを出さない (キー自体は有効、UI 文脈で出さないだけ)。
func TestRenderFooterModeDetailFilesRowContainsRefreshHint(t *testing.T) {
	out := renderFooter(ModeDetail, ModeList, true, false, false, layoutFocusTaskList, 200)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "R:refresh") {
		t.Errorf("ModeDetail Files-row footer should contain R:refresh hint, got %q", plain)
	}

	nonFiles := renderFooter(ModeDetail, ModeList, false, false, false, layoutFocusTaskList, 200)
	if strings.Contains(ansi.Strip(nonFiles), "R:refresh") {
		t.Errorf("ModeDetail non-files-row footer should not surface R:refresh hint, got %q", ansi.Strip(nonFiles))
	}
}
