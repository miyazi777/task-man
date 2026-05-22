package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// renderSingleLineRow は raw を width cell 幅にクランプしてから style で描画する。
//
// なぜこれが必要か:
//
//	lipgloss の Style に Width(w) を設定したまま Render(raw) を呼ぶと、raw の
//	表示幅が w を超えるケースで自動的に word-wrap し、結果が 2 行以上に分裂する。
//	タスクリスト / 詳細画面 / 設定画面のような縦に行を積むレイアウトでは、ある 1 行
//	が 2 行に分裂すると後続行が押し下げられ、最上段や末尾が画面外に消える原因になる
//	(issue #15 / #18)。
//
// このヘルパは:
//   - raw を ansi.Truncate (ANSI escape 認識) で w 以下に切り詰め
//   - 短い場合は末尾に半角スペースで pad (style の Width 指定は使わない)
//   - その後で style.Render を呼ぶので、style に Width が設定されていれば結局
//     冪等だが、いずれにせよ raw 側で長さを担保するため wrap は起こらない。
//
// raw に ANSI escape sequence が含まれていても安全に扱える点が `clampLineWidth`
// (rune 単位の truncate を前提とする plain-text 限定) との違い。
func renderSingleLineRow(style lipgloss.Style, raw string, width int) string {
	if width <= 0 {
		return ""
	}
	return style.Render(clampLine(raw, width))
}

// clampLine は raw を厳密に width cell 幅にクランプする。
// renderSingleLineRow の素描画版で、style を後段で適用したい / 既に適用済の文字列を
// 揃えたい場合に使う。短ければ末尾を半角スペースで pad、長ければ ansi.Truncate で
// "…" 付き切り詰めを行う (ANSI escape sequence 認識)。
func clampLine(raw string, width int) string {
	if width <= 0 {
		return ""
	}
	sw := lipgloss.Width(raw)
	switch {
	case sw == width:
		return raw
	case sw < width:
		return raw + strings.Repeat(" ", width-sw)
	default:
		return ansi.Truncate(raw, width, "…")
	}
}

// renderPaneBlock は複数行 body を width x height のブロックに整形する。
//
// 各行を clampLine で width cell ぴったりに揃え、足りない行は下に空行 pad を追加、
// 余分な行は捨てる。`lipgloss.NewStyle().Width(w).Height(h).Render(body)` の代替で、
// body 中のどの行が width を超えていても lipgloss の word-wrap を起こさない (issue #18)。
//
// 各行が事前に「width 以内に収まる」ことが保証されているなら従来の `.Width().Height().Render()`
// と等価だが、保証が崩れた将来の改修に対しても安全側に倒すために使う。
func renderPaneBlock(body string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	var lines []string
	if body != "" {
		lines = strings.Split(body, "\n")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = clampLine(line, width)
	}
	pad := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, pad)
	}
	return strings.Join(lines, "\n")
}
