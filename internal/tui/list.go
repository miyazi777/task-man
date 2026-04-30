package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderList は左ペインを描画する。focused=true なら現在のカーソル行を反転表示。
func renderList(tasks []task.Task, statuses task.StatusList, cursor int, focused bool, width, height int) string {
	if width <= 0 {
		width = 32
	}

	var lines []string
	for i, t := range tasks {
		lines = append(lines, renderRow(t, statuses, i == cursor, focused, width))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

func renderRow(t task.Task, statuses task.StatusList, isCursor, listFocused bool, width int) string {
	status, ok := statuses.ByID(t.StatusID)
	statusText := "?"
	if ok {
		statusText = status.Label
	}
	statusLabel := fmt.Sprintf("[%s]", statusText)

	// 左 padding 2 cell + 右 padding 1 cell + ラベル幅 を引いた残りが title の使用可能幅。
	const leftPad, rightPad = 2, 1
	statusW := lipgloss.Width(statusLabel)
	titleW := width - leftPad - rightPad - statusW - 1
	if titleW < 4 {
		titleW = 4
	}
	title := truncate(t.Title, titleW)

	// カーソル行 (フォーカス中): 行全体を反転背景にする。
	if isCursor && listFocused {
		left := strings.Repeat(" ", leftPad) + title
		gap := width - lipgloss.Width(left) - statusW - rightPad
		if gap < 1 {
			gap = 1
		}
		raw := left + strings.Repeat(" ", gap) + statusLabel + strings.Repeat(" ", rightPad)
		return styleCursorRow.Width(width).Render(raw)
	}

	// それ以外 (非カーソル / フォーカス外): マーカーは置かず padding のみで揃える。
	var titleRendered, statusRendered string
	if listFocused {
		titleRendered = styleListItem.Inline(true).Render(title)
		statusRendered = statusStyleFor(status).Render(statusLabel)
	} else {
		titleRendered = styleListItemDim.Inline(true).Render(title)
		statusRendered = lipgloss.NewStyle().Foreground(colorDim).Render(statusLabel)
	}

	left := strings.Repeat(" ", leftPad) + titleRendered
	leftW := lipgloss.Width(left)
	gap := width - leftW - lipgloss.Width(statusRendered) - rightPad
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + statusRendered
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// 表示幅で切り詰める単純実装 (CJK 全角は重み 2)。
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= w {
			return candidate
		}
	}
	return "…"
}
