package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

func newTitleInput(width int) textinput.Model {
	return newPopupInput(width, task.MaxTitleRunes)
}

func newFileNameInput(width int) textinput.Model {
	return newPopupInput(width, storage.MaxFileNameRunes)
}

// newFieldValueInput は拡張項目 (text 型) の値入力用 textinput を返す。
func newFieldValueInput(width int) textinput.Model {
	return newPopupInput(width, task.MaxFieldTextValueRunes)
}

// prevFieldType / nextFieldType は ModeSettingFieldAdd の type セレクター用。
// 現状は text のみだが将来追加に備えて循環できる構造にしておく。
func prevFieldType(cur task.FieldType) task.FieldType {
	all := task.AllFieldTypes
	if len(all) == 0 {
		return cur
	}
	idx := 0
	for i, ft := range all {
		if ft == cur {
			idx = i
			break
		}
	}
	idx = (idx - 1 + len(all)) % len(all)
	return all[idx]
}

func nextFieldType(cur task.FieldType) task.FieldType {
	all := task.AllFieldTypes
	if len(all) == 0 {
		return cur
	}
	idx := 0
	for i, ft := range all {
		if ft == cur {
			idx = i
			break
		}
	}
	idx = (idx + 1) % len(all)
	return all[idx]
}

func newPopupInput(width, charLimit int) textinput.Model {
	ti := textinput.New()
	ti.CharLimit = charLimit
	if width < 10 {
		width = 10
	}
	ti.Width = width

	bg := lipgloss.NewStyle().Background(colorPopupBg)
	ti.PromptStyle = bg.Foreground(colorAccent)
	ti.TextStyle = bg.Foreground(colorText)
	ti.PlaceholderStyle = bg.Foreground(colorDim)
	ti.Cursor.Style = bg.Foreground(colorText)
	ti.Cursor.TextStyle = bg.Foreground(colorText)

	ti.Focus()
	return ti
}

// popupWidth は画面幅から新規タスクポップアップの外形幅を返す。
func popupWidth(screenW int) int {
	w := screenW * 60 / 100
	if w < 24 {
		w = 24
	}
	if w > 60 {
		w = 60
	}
	if w > screenW-4 {
		w = screenW - 4
	}
	if w < 10 {
		w = 10
	}
	return w
}
