package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderDetail は右ペインを描画する。focused=true で Detail モード時のステータス行を強調。
func renderDetail(t *task.Task, focused bool, width, height int) string {
	if width <= 0 {
		width = 40
	}
	if t == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

	titleLabel := styleLabel.Render("Title")
	titleValue := styleValue.Render(t.Title)
	titleRow := titleLabel + "  " + titleValue

	statusLabelText := "Status"
	statusValueText := string(t.Status)

	var statusRow string
	if focused {
		left := lipgloss.NewStyle().Foreground(colorAccent).Render("│ ") + styleLabelFocused.Render(statusLabelText)
		statusRow = left + "  " + statusStyle(t.Status).Render(statusValueText)
	} else {
		statusRow = "  " + styleLabel.Render(statusLabelText) + "  " + styleValueDim.Render(statusValueText)
	}

	body := strings.Join([]string{
		"  " + titleRow,
		statusRow,
	}, "\n")

	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}

// renderNewTaskDetail は新規タスク作成中の右ペイン。
// inputView は textinput.View() の結果。
func renderNewTaskDetail(inputView string, width, height int) string {
	if width <= 0 {
		width = 40
	}
	titleLabel := styleLabel.Render("Title")
	statusLabel := styleLabel.Render("Status")
	statusValue := styleValueDim.Render("todo") + " " + lipgloss.NewStyle().Foreground(colorDim).Italic(true).Render("(default)")

	body := strings.Join([]string{
		"  " + titleLabel,
		"  " + styleInputBox.Render(inputView),
		"",
		"  " + statusLabel + "  " + statusValue,
	}, "\n")

	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}
