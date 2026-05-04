package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// detailFilesDividerRow は renderDetail の出力 (タスク存在時) における Files: 区切り線の行番号 (0-origin)。
// 並び: 0=ID, 1=Title, 2=Status, 3=空行, 4=Files:, 5=区切り線, 6+=ファイル行。
// 左右のペイン縦区切り線にこの行で T 字接合を入れて視覚的につなげるために使う。
const detailFilesDividerRow = 5

// renderDetail は右ペインを描画する。
// focused=true で詳細モード時のフィールド (Title/Status/Files) が反転カーソルで強調される。
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

	// ID は読み取り専用なのでカーソル対象外。常に muted で表示する。
	idRow := "  " + styleLabel.Render("ID") + "     " + styleValueDim.Render(strconv.Itoa(t.ID))
	titleRow := renderDetailField("Title", t.Title, focused, focused && fieldCursor == detailFieldTitle, statusStyleFor(status), false, width)
	statusRow := renderDetailField("Status", statusText, focused, focused && fieldCursor == detailFieldStatus, statusStyleFor(status), true, width)
	filesBlock := renderFilesBlock(files, focused, fieldCursor == detailFieldFiles, fileCursor, width)

	body := strings.Join([]string{idRow, titleRow, statusRow, "", filesBlock}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}

// renderDetailField は label と value の 1 行を描画する。
// hasCursor=true なら行幅いっぱいを反転背景にする (yazi 風)。
// valueStatusStyle は Status 行の値だけに適用する色 (それ以外は無視)。
func renderDetailField(label, value string, focused, hasCursor bool, valueStatusStyle lipgloss.Style, useValueStatusStyle bool, width int) string {
	if hasCursor {
		raw := "  " + label + "  " + value
		return styleCursorRow.Width(width).Render(raw)
	}
	var labelRendered, valueRendered string
	if focused {
		labelRendered = styleLabel.Render(label)
		if useValueStatusStyle {
			valueRendered = valueStatusStyle.Render(value)
		} else {
			valueRendered = styleValue.Render(value)
		}
	} else {
		labelRendered = styleLabel.Render(label)
		valueRendered = styleValueDim.Render(value)
	}
	return "  " + labelRendered + "  " + valueRendered
}

// renderFilesBlock は Files: セクションをヘッダ + 区切り線 + ファイル行で描画する。
//   - blockFocused: detailCursor が Files セクションを指しているか
//   - fileCursor: Files 内の選択行
//
// ファイルが 0 件のときは "(no files)" を 1 行表示する。
func renderFilesBlock(files []string, focused, blockFocused bool, fileCursor, width int) string {
	header := "  " + styleLabel.Render("Files:")
	// 区切り線はペイン全幅にして、左右のペイン縦区切り線 (├ / ┤) と
	// つながる横一文字に見えるようにする。
	dividerWidth := width
	if dividerWidth < 1 {
		dividerWidth = 1
	}
	divider := styleDivider.Render(strings.Repeat("─", dividerWidth))

	var rows []string
	rows = append(rows, header, divider)

	if len(files) == 0 {
		rows = append(rows, "    "+styleValueDim.Render("(no files)"))
		return strings.Join(rows, "\n")
	}

	for i, name := range files {
		isCursor := blockFocused && focused && i == fileCursor
		if isCursor {
			rows = append(rows, styleCursorRow.Width(width).Render("    "+name))
			continue
		}
		var line string
		if focused {
			line = "    " + styleValue.Render(name)
		} else {
			line = "    " + styleValueDim.Render(name)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}
