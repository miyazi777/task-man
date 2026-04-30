package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

func newTitleInput(width int) textinput.Model {
	ti := textinput.New()
	ti.CharLimit = task.MaxTitleRunes
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
