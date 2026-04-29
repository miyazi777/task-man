package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// ラベル/ヒントが上下の罫線にそれぞれ埋め込まれていること、
// および左寄せで配置されていることを検証する。
func TestPopupEmbedsLabelInBorder(t *testing.T) {
	const screenW = 100
	const screenH = 30
	bg := strings.Repeat(strings.Repeat(" ", screenW)+"\n", screenH)
	bg = strings.TrimSuffix(bg, "\n")
	overlaid := overlayNewTaskPopup(bg, "> sample", screenW, screenH)

	var topRow, bottomRow string
	for _, line := range strings.Split(overlaid, "\n") {
		stripped := ansi.Strip(line)
		if strings.ContainsRune(stripped, '╭') {
			topRow = stripped
		}
		if strings.ContainsRune(stripped, '╰') {
			bottomRow = stripped
		}
	}
	if topRow == "" || bottomRow == "" {
		t.Fatalf("could not find top/bottom rows; top=%q bottom=%q", topRow, bottomRow)
	}

	// 上罫線: ╭─New task─...─╮
	if !strings.Contains(topRow, "New task") {
		t.Errorf("top border missing label: %q", topRow)
	}
	if !strings.Contains(topRow, "╮") {
		t.Errorf("top border missing right corner: %q", topRow)
	}
	// 左寄せ: ╭ の直後の ─ から数えてすぐにラベルが現れること。
	startIdx := strings.Index(topRow, "╭")
	labelIdx := strings.Index(topRow, "New task")
	// runes ベースで距離を確認。╭ から最大 2 文字以内 (= ╭─ の直後) に label があれば左寄せ。
	prefix := topRow[startIdx:labelIdx]
	if got := len([]rune(prefix)); got > 2 {
		t.Errorf("label not left-aligned: %d runes between ╭ and label, prefix=%q", got, prefix)
	}

	// 下罫線: ╰─Enter:save  Esc:discard─...─╯
	if !strings.Contains(bottomRow, "Enter:save") {
		t.Errorf("bottom border missing hint: %q", bottomRow)
	}
	if !strings.Contains(bottomRow, "╯") {
		t.Errorf("bottom border missing right corner: %q", bottomRow)
	}
	startIdxB := strings.Index(bottomRow, "╰")
	hintIdx := strings.Index(bottomRow, "Enter:save")
	prefixB := bottomRow[startIdxB:hintIdx]
	if got := len([]rune(prefixB)); got > 2 {
		t.Errorf("hint not left-aligned: %d runes between ╰ and hint, prefix=%q", got, prefixB)
	}
}

// 実際の textinput.View() を入力にした場合でもポップアップ幅が揃うことを検証する。
// (ユーザー報告の「背景が乱れる」現象の再発防止用。)
func TestPopupWithRealTextInput(t *testing.T) {
	const screenW = 120
	const screenH = 30
	ti := newTitleInput(popupWidth(screenW) - 6)

	bg := strings.Repeat(strings.Repeat(" ", screenW)+"\n", screenH)
	bg = strings.TrimSuffix(bg, "\n")

	overlaid := overlayNewTaskPopup(bg, ti.View(), screenW, screenH)

	for i, line := range strings.Split(overlaid, "\n") {
		if got := ansi.StringWidth(line); got != screenW {
			t.Errorf("line %d width = %d, want %d, content=%q", i, got, screenW, ansi.Strip(line))
		}
	}
}

// ポップアップ単体の各行幅が外形幅で揃っていることを検証する。
// 揃っていないと背景色が歯抜けになり、画面が乱れて見える原因になる。
func TestPopupHasUniformInnerWidth(t *testing.T) {
	const screenW = 100
	const screenH = 30
	bg := strings.Repeat(strings.Repeat(" ", screenW)+"\n", screenH)
	bg = strings.TrimSuffix(bg, "\n")
	overlaid := overlayNewTaskPopup(bg, "> sample", screenW, screenH)

	wantOuter := popupWidth(screenW)

	// オーバーレイ済みの行から、ポップアップの中身行 (左右端が背景空白の中に乗っている部分) を検査する。
	lines := strings.Split(overlaid, "\n")
	for _, line := range lines {
		stripped := ansi.Strip(line)
		if !strings.ContainsRune(stripped, '│') && !strings.ContainsRune(stripped, '╭') && !strings.ContainsRune(stripped, '╰') {
			continue
		}
		// 背景連続スペースを除いた、ポップアップ部分だけの幅を計測する。
		// 行全体は screenW 幅で揃っているはずなので、ここでは ANSI Strip 結果に
		// ポップアップ罫線が wantOuter 幅にわたって出現することを確認する。
		first := strings.IndexAny(stripped, "│╭╰")
		last := strings.LastIndexAny(stripped, "│╮╯")
		if first < 0 || last < 0 {
			continue
		}
		// last は rune インデックスではなくバイトインデックスだが、対象文字は 3 バイト。
		// 文字数ベースの幅として再計算する。
		runes := []rune(stripped)
		var firstR, lastR int = -1, -1
		for i, r := range runes {
			if firstR == -1 && (r == '│' || r == '╭' || r == '╰') {
				firstR = i
			}
			if r == '│' || r == '╮' || r == '╯' {
				lastR = i
			}
		}
		gotOuter := ansi.StringWidth(string(runes[firstR : lastR+1]))
		if gotOuter != wantOuter {
			t.Errorf("popup line outer width = %d, want %d, content=%q", gotOuter, wantOuter, stripped)
		}
	}
}

// ポップアップ各行の表示幅が揃っていることを検証する。
// 揃っていないと背景色が歯抜けになり、画面が乱れて見える原因になる。
func TestPopupLinesHaveUniformWidth(t *testing.T) {
	const screenW = 100
	const screenH = 30

	// 簡易な textinput 相当の入力ビュー文字列。
	inputView := "> sample"

	bg := strings.Repeat(strings.Repeat(" ", screenW)+"\n", screenH)
	bg = strings.TrimSuffix(bg, "\n")

	overlaid := overlayNewTaskPopup(bg, inputView, screenW, screenH)

	wantW := popupWidth(screenW)

	// オーバーレイ後の各行から、ポップアップの上端〜下端を抽出して幅を確認する。
	lines := strings.Split(overlaid, "\n")
	popupLines := 0
	for _, line := range lines {
		stripped := ansi.Strip(line)
		// ポップアップの境界文字を含む行を対象にする。
		if !strings.ContainsAny(stripped, "╭╮╯╰│") {
			continue
		}
		popupLines++
		// 行全体の表示幅は背景幅と一致する。
		if got := ansi.StringWidth(line); got != screenW {
			t.Errorf("line width != screen width: got %d want %d, line=%q", got, screenW, stripped)
		}
	}

	if popupLines < 2 {
		t.Fatalf("expected at least 2 popup lines, found %d", popupLines)
	}
	t.Logf("popup outer width = %d, content lines = %d", wantW, popupLines)
}
