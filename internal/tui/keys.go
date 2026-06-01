package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding // enter - タスクリストで詳細へ遷移
	Confirm       key.Binding // enter - 編集開始 / 保存決定
	Open          key.Binding // l/→ - タスクリストでステータス/タスクを展開 / ModeMove でインデント
	Close         key.Binding // h/← - タスクリストでステータス/タスクを折りたたみ / ModeMove でアウトデント
	Move          key.Binding // m - ModeMove の開始 / 確定
	NewTask       key.Binding // a - 新規タスク (status 行) / サブタスク (task 行)
	DeleteTask    key.Binding // d - タスクをゴミ箱へ / ゴミ箱ビューでは完全削除
	RestoreTask   key.Binding // r - ゴミ箱ビューのタスクを通常リストへ復帰
	ToggleTrash   key.Binding // T - 通常リスト ↔ ゴミ箱ビューのトグル
	Prefix        key.Binding // ; - prefix モードへ遷移
	PrefixTrash   key.Binding // t - prefix 中: ゴミ箱表示トグル
	PrefixSetting key.Binding // s - prefix 中: 設定画面へ遷移
	PrefixLayout  key.Binding // l - prefix 中: レイアウト調整モードへ遷移
	Color         key.Binding // c - status 設定で色変更
	AddFile       key.Binding // a (Files セクション)
	RenameFile    key.Binding // r (Files セクション)
	DeleteFile    key.Binding // d (Files セクション)
	CopyPath      key.Binding // p - カーソル位置のタスク / ファイルの絶対パスをクリップボードへコピー
	Refresh       key.Binding // R - ファイル一覧を再読込 (外部での mv / 追加 / 削除を反映)
	Quit          key.Binding
	Back          key.Binding
	ConfirmY      key.Binding
	ConfirmN      key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
		Confirm:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit")),
		Open:          key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "open")),
		Close:         key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "close")),
		Move:          key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),
		NewTask:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "new")),
		DeleteTask:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		RestoreTask:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore")),
		ToggleTrash:   key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "trash")),
		Prefix:        key.NewBinding(key.WithKeys(";"), key.WithHelp(";", "prefix")),
		PrefixTrash:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "trash")),
		PrefixSetting: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "setting")),
		PrefixLayout:  key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "layout")),
		Color:         key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "color")),
		AddFile:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		RenameFile:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		DeleteFile:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		CopyPath:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "path")),
		Refresh:       key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Quit:          key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		ConfirmY:      key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		ConfirmN:      key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no")),
	}
}
