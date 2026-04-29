package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const ansiReset = "\x1b[0m"

// PlaceOverlay は背景文字列 bg の (x, y) 位置に前景 fg を重ねた文字列を返す。
// ANSI エスケープと表示幅 (CJK 全角) を考慮した cell 単位の合成。
// 各セグメントの境界で ANSI リセットを挿入し、隣接スタイルの漏れを防ぐ。
func PlaceOverlay(x, y int, fg, bg string) string {
	if fg == "" {
		return bg
	}
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	for i, fgLine := range fgLines {
		ty := y + i
		if ty >= len(bgLines) {
			break
		}
		bgLine := bgLines[ty]
		bgW := ansi.StringWidth(bgLine)
		fgW := ansi.StringWidth(fgLine)

		// 左側: 0 から x までを切り出す。bg が短ければ空白でパディング。
		var left string
		if x <= bgW {
			left = ansi.Cut(bgLine, 0, x)
			if w := ansi.StringWidth(left); w < x {
				left += strings.Repeat(" ", x-w)
			}
		} else {
			left = bgLine + strings.Repeat(" ", x-bgW)
		}

		// 右側: x+fgW から bg の末尾まで。範囲外なら空。
		var right string
		if x+fgW < bgW {
			right = ansi.Cut(bgLine, x+fgW, bgW)
		}

		bgLines[ty] = left + ansiReset + fgLine + ansiReset + right
	}
	return strings.Join(bgLines, "\n")
}
