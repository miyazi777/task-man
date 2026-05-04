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
// onFilesRow は ModeDetail のときカーソルが Files 行を指しているかを示す (a/r/d ヒント切替用)。
// onURLRow は ModeDetail のときカーソルが url 型項目行を指しているかを示す (enter:open / e:edit ヒント用)。
// 確認モード (Quit/Delete) のときは prevMode のヒントを引き継ぎ、ポップアップにフォーカスを譲る。
// viewTrash は ModeList のときにヒントを通常用 / ゴミ箱用で切り替えるためのフラグ。
func renderFooter(mode, prevMode Mode, onFilesRow bool, onURLRow bool, viewTrash bool, width int) string {
	if mode == ModeQuitConfirm || mode == ModeDeleteFileConfirm || mode == ModeTrashConfirm || mode == ModeDeleteTaskConfirm || mode == ModeSettingStatusDeleteConfirm {
		return renderFooter(prevMode, ModeList, onFilesRow, onURLRow, viewTrash, width)
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
			{"o", "operation"},
			{";", "prefix"},
			{"q", "quit"},
		})
	case ModeDetail:
		switch {
		case onFilesRow:
			content = renderHints([]hintItem{
				{"k/↑", "up"}, {"j/↓", "down"}, {"enter", "open"},
				{"a", "add"}, {"r", "rename"}, {"d", "delete"},
				{"esc", "back"}, {"q", "quit"},
			})
		case onURLRow:
			content = renderHints([]hintItem{
				{"k/↑", "up"}, {"j/↓", "down"},
				{"enter", "edit"}, {"o", "open"},
				{"esc", "back"}, {"q", "quit"},
			})
		default:
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
			{"Enter", "confirm"}, {"Esc", "cancel"},
		})
	case ModePrefix:
		content = renderHints([]hintItem{
			{"t", "trash"}, {"s", "setting"}, {"esc", "back"},
		})
	case ModeOperation:
		content = renderHints([]hintItem{
			{"t", "title"}, {"s", "status"}, {"g", "tags"}, {"esc", "back"},
		})
	case ModeTagPicker:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"Enter", "add/toggle"}, {"c", "color"}, {"Esc", "close"},
		})
	case ModeTagColorPicker:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"h/←", "left"}, {"l/→", "right"},
			{"Enter", "save"}, {"Esc", "cancel"},
		})
	case ModeSetting:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"enter", "detail"}, {"esc", "back"},
		})
	case ModeSettingField:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"r", "rename"}, {"a", "add"},
			{"m", "move"}, {"d", "delete"},
			{"enter", "detail"}, {"esc", "back"},
		})
	case ModeSettingFieldAttribute:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"enter", "edit"}, {"esc", "back"},
		})
	case ModeSettingFieldAdd:
		content = renderHints([]hintItem{
			{"Tab", "focus"},
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeSettingFieldRename:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeSettingFieldMove:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"Enter", "confirm"}, {"Esc", "cancel"},
		})
	case ModeEditFieldValue:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeEditFieldDateValue:
		content = renderHints([]hintItem{
			{"h/←", "prev"}, {"l/→", "next"},
			{"j/↓", "down"}, {"k/↑", "up"},
			{"p", "prev mo"}, {"n", "next mo"},
			{"Enter", "save"}, {"Esc", "cancel"},
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
			{"k/↑", "up"}, {"j/↓", "down"}, {"h/←", "left"}, {"l/→", "right"},
			{"Enter", "save"}, {"Esc", "cancel"},
		})
	case ModeSettingStatusMove:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"},
			{"Enter", "confirm"}, {"Esc", "cancel"},
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
