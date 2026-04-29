package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

func newTitleInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "タスク名を入力"
	ti.CharLimit = 200
	ti.Width = 60
	ti.Focus()
	return ti
}
