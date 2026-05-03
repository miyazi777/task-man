package tui

type Mode int

const (
	ModeList Mode = iota
	ModeDetail
	ModeNewTask
	ModeNewSubtask
	ModeEditTitle
	ModeEditStatus
	ModeAddFile
	ModeRenameFile
	ModeDeleteFileConfirm
	ModeQuitConfirm
	ModeMove
	ModeTrashConfirm           // タスクをゴミ箱へ移動するときの確認
	ModeDeleteTaskConfirm      // ゴミ箱内タスクの完全削除確認
	ModePrefix                 // ; を押した直後の prefix 入力待ち状態
	ModeSetting                // 設定画面 (左メニュー側にフォーカス)
	ModeSettingStatus          // 設定画面 status サブ (右ペイン側にフォーカス)
	ModeSettingStatusRename    // status のラベル変更入力
	ModeSettingStatusAdd       // status の新規追加入力
	ModeSettingStatusColor     // status の色選択ピッカー
)

func (m Mode) String() string {
	switch m {
	case ModeList:
		return "list"
	case ModeDetail:
		return "detail"
	case ModeNewTask:
		return "newtask"
	case ModeNewSubtask:
		return "newsubtask"
	case ModeEditTitle:
		return "edittitle"
	case ModeEditStatus:
		return "editstatus"
	case ModeAddFile:
		return "addfile"
	case ModeRenameFile:
		return "renamefile"
	case ModeDeleteFileConfirm:
		return "deletefileconfirm"
	case ModeQuitConfirm:
		return "quitconfirm"
	case ModeMove:
		return "move"
	case ModeTrashConfirm:
		return "trashconfirm"
	case ModeDeleteTaskConfirm:
		return "deletetaskconfirm"
	case ModePrefix:
		return "prefix"
	case ModeSetting:
		return "setting"
	case ModeSettingStatus:
		return "settingstatus"
	case ModeSettingStatusRename:
		return "settingstatusrename"
	case ModeSettingStatusAdd:
		return "settingstatusadd"
	case ModeSettingStatusColor:
		return "settingstatuscolor"
	default:
		return "unknown"
	}
}
