package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding // l/→ - 前進ナビゲーション (一覧→詳細)
	Confirm    key.Binding // enter - 編集開始 / 保存決定
	NewTask    key.Binding // a - 新規タスク (status 行) / サブタスク (task 行)
	AddFile    key.Binding // a (Files セクション)
	RenameFile key.Binding // r (Files セクション)
	DeleteFile key.Binding // d (Files セクション)
	Quit       key.Binding
	Back       key.Binding
	ConfirmY   key.Binding
	ConfirmN   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Enter:      key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "detail")),
		Confirm:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit")),
		NewTask:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new")),
		AddFile:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		RenameFile: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		DeleteFile: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Back:       key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "back")),
		ConfirmY:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		ConfirmN:   key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no")),
	}
}
