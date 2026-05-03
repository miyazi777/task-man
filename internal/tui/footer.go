package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type hintItem struct {
	key   string
	label string
}

// renderFooter は画面下部のヒント帯を描画する。
// detailCursor は ModeDetail のときに参照され、Files セクションでは a/r/d を案内する。
// 確認モード (Quit/Delete) のときは prevMode のヒントを引き継ぎ、ポップアップにフォーカスを譲る。
// viewTrash は ModeList のときにヒントを通常用 / ゴミ箱用で切り替えるためのフラグ。
func renderFooter(mode, prevMode Mode, detailCursor int, viewTrash bool, width int) string {
	if mode == ModeQuitConfirm || mode == ModeDeleteFileConfirm || mode == ModeTrashConfirm || mode == ModeDeleteTaskConfirm {
		return renderFooter(prevMode, ModeList, detailCursor, viewTrash, width)
	}

	var content string
	switch mode {
	case ModeList:
		if viewTrash {
			content = renderHints([]hintItem{
				{"k/↑", "up"}, {"j/↓", "down"},
				{"l/→", "open"}, {"h/←", "close"},
				{"enter", "detail"},
				{"r", "restore"}, {"d", "delete"},
				{"T", "back"}, {"q", "quit"},
			})
			break
		}
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"l/→", "open"}, {"h/←", "close"},
			{"enter", "detail"},
			{"m", "move"},
			{"a", "new/subtask"},
			{"d", "delete"},
			{";", "prefix"},
			{"q", "quit"},
		})
	case ModeDetail:
		if detailCursor == detailFieldFiles {
			content = renderHints([]hintItem{
				{"k/↑", "up"}, {"j/↓", "down"}, {"enter", "open"},
				{"a", "add"}, {"r", "rename"}, {"d", "delete"},
				{"esc", "back"}, {"q", "quit"},
			})
		} else {
			content = renderHints([]hintItem{
				{"k/↑", "up"}, {"j/↓", "down"}, {"enter", "edit"}, {"esc", "back"}, {"q", "quit"},
			})
		}
	case ModeNewTask, ModeNewSubtask:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeEditTitle:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeEditStatus:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeAddFile, ModeRenameFile:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeMove:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"l/→", "indent"}, {"h/←", "outdent"},
			{"m", "confirm"}, {"Esc", "cancel"},
		})
	case ModePrefix:
		content = renderHints([]hintItem{
			{"t", "trash"}, {"s", "setting"}, {"esc", "back"},
		})
	case ModeSetting:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"enter", "detail"}, {"esc", "back"},
		})
	case ModeSettingStatus:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"r", "rename"}, {"c", "color"}, {"a", "add"},
			{"m", "move"}, {"d", "delete"},
			{"esc", "back"},
		})
	case ModeSettingStatusRename, ModeSettingStatusAdd:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeSettingStatusColor:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "cancel"},
		})
	case ModeSettingStatusMove:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"m", "confirm"}, {"Esc", "cancel"},
		})
	}

	bar := lipgloss.NewStyle().
		Background(colorFooterBg).
		Foreground(colorMuted).
		Width(width).
		Padding(0, 1).
		Render(content)
	return bar
}

func renderHints(items []hintItem) string {
	var parts []string
	for _, it := range items {
		k := styleFooterKey.Render(it.key)
		v := lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Render(":" + it.label)
		parts = append(parts, k+v)
	}
	sep := lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Render("  ")
	return strings.Join(parts, sep)
}

// renderPopupHints は ポップアップ下罫線用のヒント文字列を組み立てる。
// キー部分だけ太字 (stylePopupKey) で目立たせ、ラベル部分は muted italic (stylePopupHint) のまま。
func renderPopupHints(items []hintItem) string {
	var parts []string
	for _, it := range items {
		k := stylePopupKey.Render(it.key)
		v := stylePopupHint.Render(":" + it.label)
		parts = append(parts, k+v)
	}
	sep := stylePopupHint.Render("  ")
	return strings.Join(parts, sep)
}
