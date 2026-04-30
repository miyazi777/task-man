package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderDetail は右ペインを描画する。
// focused=true で詳細モード時のフィールド (Title/Status) が強調される。
// fieldCursor は 0=Title, 1=Status のどちらにカーソルがあるかを示す。
func renderDetail(t *task.Task, statuses task.StatusList, focused bool, fieldCursor, width, height int) string {
	if width <= 0 {
		width = 40
	}
	if t == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

	titleLabelText := "Title"
	statusLabelText := "Status"

	status, ok := statuses.ByID(t.StatusID)
	statusText := "?"
	if ok {
		statusText = status.Label
	}

	var titleValue, statusValue string
	if focused {
		titleValue = styleValue.Render(t.Title)
		statusValue = statusStyleFor(status).Render(statusText)
	} else {
		titleValue = styleValueDim.Render(t.Title)
		statusValue = styleValueDim.Render(statusText)
	}

	titleRow := renderDetailField(titleLabelText, titleValue, focused && fieldCursor == 0)
	statusRow := renderDetailField(statusLabelText, statusValue, focused && fieldCursor == 1)

	body := strings.Join([]string{titleRow, statusRow}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}

func renderDetailField(label, valueRendered string, hasCursor bool) string {
	if hasCursor {
		marker := lipgloss.NewStyle().Foreground(colorAccent).Render("│ ")
		return marker + styleLabelFocused.Render(label) + "  " + valueRendered
	}
	return "  " + styleLabel.Render(label) + "  " + valueRendered
}
