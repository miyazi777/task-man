package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderList は左ペインを描画する。focused=true なら現在のカーソル行を強調表示。
// pendingNewTask が非 nil の場合、最下段に「(入力中…) [todo]」のプレースホルダを差し込む。
func renderList(tasks []task.Task, cursor int, focused bool, width, height int, pendingNewTask *string) string {
	if width <= 0 {
		width = 32
	}

	var lines []string
	for i, t := range tasks {
		lines = append(lines, renderRow(t, i == cursor, focused, width))
	}
	if pendingNewTask != nil {
		lines = append(lines, renderPendingRow(width))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

func renderRow(t task.Task, isCursor, listFocused bool, width int) string {
	statusLabel := fmt.Sprintf("[%s]", t.Status)

	// 内部余白(padding 1)とマーカー領域 2、ステータス、間隔を考慮し title 部の幅を決める。
	const markerW = 2
	statusW := lipgloss.Width(statusLabel)
	titleW := width - markerW - statusW - 4
	if titleW < 4 {
		titleW = 4
	}

	title := truncate(t.Title, titleW)

	var marker, titleRendered, statusRendered string
	switch {
	case isCursor && listFocused:
		marker = styleCursorMarker.Render("▶ ")
		titleRendered = styleListItemCursor.Inline(true).Render(title)
		statusRendered = statusStyle(t.Status).Render(statusLabel)
	case isCursor && !listFocused:
		// 詳細フォーカス時はリスト全体を dim 表示。カーソル行も色控えめに。
		marker = lipgloss.NewStyle().Foreground(colorDivider).Render("│ ")
		titleRendered = styleListItemDim.Inline(true).Render(title)
		statusRendered = lipgloss.NewStyle().Foreground(colorDim).Bold(true).Render(statusLabel)
	default:
		marker = "  "
		if listFocused {
			titleRendered = styleListItem.Inline(true).Render(title)
			statusRendered = statusStyle(t.Status).Render(statusLabel)
		} else {
			titleRendered = styleListItemDim.Inline(true).Render(title)
			statusRendered = lipgloss.NewStyle().Foreground(colorDim).Bold(true).Render(statusLabel)
		}
	}

	left := marker + titleRendered
	leftW := lipgloss.Width(left)
	gap := width - leftW - lipgloss.Width(statusRendered) - 1
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + statusRendered
}

func renderPendingRow(width int) string {
	title := styleNewTaskTitle.Render("(入力中…)")
	status := lipgloss.NewStyle().Foreground(colorWarn).Bold(true).Render("[todo]")
	left := "  " + title
	leftW := lipgloss.Width(left)
	gap := width - leftW - lipgloss.Width(status) - 1
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + status
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
