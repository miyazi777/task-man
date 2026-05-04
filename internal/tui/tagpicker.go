package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/task"
)

// overlayTagPicker は対象タスク (assignedIDs) と全タグ集合 (allTags) を元にタグピッカーを描画する。
// cursor=0 は上部の "create tag" 入力行、cursor=i (i>=1) は allTags.Sorted() の i-1 番目を指す。
// inputValue / inputCursorPos は textinput の値・カーソル位置。inputErr が non-nil ならエラー行を入れる。
func overlayTagPicker(bg string, allTags task.TagList, assignedIDs []int, cursor int, inputValue string, inputCursorPos int, inputErr error, screenW, screenH int) string {
	sorted := allTags.Sorted()

	// 描画上は label / hint / 各タグ行 / create input の幅から popup 幅を決める。
	labelText := "Tags:"
	labelW := ansi.StringWidth(labelText)

	hints := []hintItem{
		{"k/↑", "up"}, {"j/↓", "down"},
		{"Enter", "add/toggle"}, {"Esc", "close"},
	}
	hintRendered := renderPopupHints(hints)
	hintW := ansi.StringWidth(ansi.Strip(hintRendered))

	// 行コンテンツ最大幅: 各タグ名の最大、create input ヒント文の幅
	contentW := labelW
	if hintW > contentW {
		contentW = hintW
	}
	for _, tg := range sorted {
		// "  ✓ <name>" の形式
		w := ansi.StringWidth("  ✓ ") + ansi.StringWidth(tg.Name)
		if w > contentW {
			contentW = w
		}
	}
	// create input は "> create tag" の placeholder 幅も考慮 (最低でも 24 cell くらい確保)
	const minInputW = 24
	if contentW < minInputW {
		contentW = minInputW
	}

	popupOuterW := contentW + 4 // 左右 border (2) + 内側余白 (2)
	if popupOuterW > screenW {
		popupOuterW = screenW
		contentW = popupOuterW - 4
		if contentW < 4 {
			contentW = 4
		}
	}
	innerW := popupOuterW - 2

	// --- 上罫線 / 下罫線 ---
	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render(labelText), innerW)
	bottomRow := buildBorderRow("╰", "╯", hintRendered, innerW)

	rows := []string{topRow}

	// --- 1 行目: create input ---
	rows = append(rows, tagPickerInputRow(cursor == 0, inputValue, inputCursorPos, contentW))
	if inputErr != nil {
		// エラー行を入力行直下に挟む。
		rows = append(rows, tagPickerErrorRow(inputErr.Error(), contentW))
	}
	// 区切り罫線
	rows = append(rows, tagPickerDivider(contentW))

	// --- 既存タグ行 ---
	assignedSet := make(map[int]bool, len(assignedIDs))
	for _, id := range assignedIDs {
		assignedSet[id] = true
	}
	for i, tg := range sorted {
		isCursor := cursor == i+1
		isAssigned := assignedSet[tg.ID]
		rows = append(rows, tagPickerListRow(tg, isCursor, isAssigned, contentW))
	}
	if len(sorted) == 0 {
		rows = append(rows, tagPickerEmptyListRow(contentW))
	}

	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// tagPickerInputRow は create input 行を自前描画する。
// 行全体を popup 背景色で統一するために textinput.View() は使わず、
// 値・カーソル位置から直接組み立てる。focused=true で先頭に "> " マーカー。
// 値が空なら "create tag" の placeholder を dim 色で表示し、focused 時は先頭文字をカーソルブロック表示。
func tagPickerInputRow(focused bool, value string, cursorPos, contentW int) string {
	rowFg := lipgloss.NewStyle().Background(colorPopupBg).Foreground(colorText)
	rowDim := lipgloss.NewStyle().Background(colorPopupBg).Foreground(colorDim)
	cursorBlock := lipgloss.NewStyle().Background(colorAccent).Foreground(colorBase)

	prefix := "  "
	if focused {
		prefix = "> "
	}

	var body string
	if value == "" {
		placeholder := "create tag"
		runes := []rune(placeholder)
		if focused && len(runes) > 0 {
			body = cursorBlock.Render(string(runes[0])) + rowDim.Render(string(runes[1:]))
		} else {
			body = rowDim.Render(placeholder)
		}
	} else {
		runes := []rune(value)
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos > len(runes) {
			cursorPos = len(runes)
		}
		before := string(runes[:cursorPos])
		if focused {
			if cursorPos < len(runes) {
				ch := string(runes[cursorPos])
				rest := string(runes[cursorPos+1:])
				body = rowFg.Render(before) + cursorBlock.Render(ch) + rowFg.Render(rest)
			} else {
				body = rowFg.Render(value) + cursorBlock.Render(" ")
			}
		} else {
			body = rowFg.Render(value)
		}
	}

	raw := rowFg.Render(prefix) + body
	used := ansi.StringWidth(ansi.Strip(raw))
	if used < contentW {
		raw += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		raw +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}

// tagPickerErrorRow は inputErr を danger 色で 1 行表示する。
func tagPickerErrorRow(msg string, contentW int) string {
	prefix := "  ! "
	full := prefix + msg
	if ansi.StringWidth(full) > contentW {
		full = ansi.Truncate(full, contentW, "")
	}
	rendered := stylePopupError.Render(full)
	used := ansi.StringWidth(ansi.Strip(rendered))
	if used < contentW {
		rendered += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		rendered +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}

// tagPickerDivider は input と list の境界に入れる横罫線 1 行。
func tagPickerDivider(contentW int) string {
	line := stylePopupBorder.Render(strings.Repeat("─", contentW))
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		line +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}

// tagPickerListRow は既存タグの 1 行。assigned=true で先頭に ✓ マーカー。
func tagPickerListRow(tg task.Tag, focused, assigned bool, contentW int) string {
	cur := "  "
	if focused {
		cur = "> "
	}
	mark := "  "
	if assigned {
		mark = "✓ "
	}
	raw := cur + mark + tg.Name
	if w := ansi.StringWidth(raw); w > contentW {
		raw = ansi.Truncate(raw, contentW, "")
	}
	rendered := stylePopupFill.Foreground(colorText).Render(raw)
	used := ansi.StringWidth(ansi.Strip(rendered))
	if used < contentW {
		rendered += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		rendered +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}

// tagPickerEmptyListRow は既存タグ 0 件のときの (no tags) 行。
func tagPickerEmptyListRow(contentW int) string {
	raw := "  (no tags yet)"
	if w := ansi.StringWidth(raw); w > contentW {
		raw = ansi.Truncate(raw, contentW, "")
	}
	rendered := stylePopupHint.Render(raw)
	used := ansi.StringWidth(ansi.Strip(rendered))
	if used < contentW {
		rendered += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		rendered +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}
