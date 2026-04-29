package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	NewTask   key.Binding
	Quit      key.Binding
	Back      key.Binding
	ConfirmY  key.Binding
	ConfirmN  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Enter:    key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "detail")),
		NewTask:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new")),
		Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Back:     key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "back")),
		ConfirmY: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		ConfirmN: key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no")),
	}
}
