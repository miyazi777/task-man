package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderDetail は右ペインを描画する。
// focused=true で詳細モード時のフィールド (Title/Status/Files) が強調される。
// fieldCursor は 0=Title, 1=Status, 2=Files。fileCursor は Files 内のインデックス。
func renderDetail(t *task.Task, statuses task.StatusList, files []string, focused bool, fieldCursor, fileCursor, width, height int) string {
	if width <= 0 {
		width = 40
	}
	if t == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

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

	titleRow := renderDetailField("Title", titleValue, focused && fieldCursor == detailFieldTitle)
	statusRow := renderDetailField("Status", statusValue, focused && fieldCursor == detailFieldStatus)
	filesBlock := renderFilesBlock(files, focused, fieldCursor == detailFieldFiles, fileCursor)

	body := strings.Join([]string{titleRow, statusRow, "", filesBlock}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}

func renderDetailField(label, valueRendered string, hasCursor bool) string {
	if hasCursor {
		marker := lipgloss.NewStyle().Foreground(colorAccent).Render("│ ")
		return marker + styleLabelFocused.Render(label) + "  " + valueRendered
	}
	return "  " + styleLabel.Render(label) + "  " + valueRendered
}

// renderFilesBlock は Files: セクションをヘッダ + 区切り線 + ファイル行で描画する。
//   - blockFocused: detailCursor が Files セクションを指しているか
//   - fileCursor: Files 内の選択行
//
// ファイルが 0 件のときは "(no files)" を 1 行表示する。
func renderFilesBlock(files []string, focused, blockFocused bool, fileCursor int) string {
	header := "  " + styleLabel.Render("Files:")
	if blockFocused && focused {
		marker := lipgloss.NewStyle().Foreground(colorAccent).Render("│ ")
		header = marker + styleLabelFocused.Render("Files:")
	}
	divider := "  " + styleDivider.Render(strings.Repeat("─", 14))

	var rows []string
	rows = append(rows, header, divider)

	if len(files) == 0 {
		rows = append(rows, "    "+styleValueDim.Render("(no files)"))
		return strings.Join(rows, "\n")
	}

	for i, name := range files {
		isCursor := blockFocused && focused && i == fileCursor
		var line string
		switch {
		case isCursor:
			line = styleCursorMarker.Render("  > ") + styleValue.Render(name)
		case focused:
			line = "    " + styleValue.Render(name)
		default:
			line = "    " + styleValueDim.Render(name)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}
