package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding // enter - タスクリストで詳細へ遷移
	Confirm    key.Binding // enter - 編集開始 / 保存決定
	Toggle     key.Binding // space - ModeMove での子モードトグル (タスクリストの開閉には未使用)
	Open       key.Binding // l/→ - タスクリストでステータス/タスクを展開
	Close      key.Binding // h/← - タスクリストでステータス/タスクを折りたたみ
	Move       key.Binding // x - カーソル位置のタスクの移動を開始
	Paste      key.Binding // p - 移動先で確定して貼り付け
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
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
		Confirm:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit")),
		Toggle:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		Open:       key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "open")),
		Close:      key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "close")),
		Move:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "move")),
		Paste:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "paste")),
		NewTask:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new")),
		AddFile:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		RenameFile: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		DeleteFile: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		ConfirmY:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		ConfirmN:   key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no")),
	}
}
