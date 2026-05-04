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
	ModeTrashConfirm                // タスクをゴミ箱へ移動するときの確認
	ModeDeleteTaskConfirm           // ゴミ箱内タスクの完全削除確認
	ModePrefix                      // ; を押した直後の prefix 入力待ち状態
	ModeSetting                     // 設定画面 (左メニュー側にフォーカス)
	ModeSettingStatus               // 設定画面 status サブ (右ペイン側にフォーカス)
	ModeSettingStatusRename         // status のラベル変更入力
	ModeSettingStatusAdd            // status の新規追加入力
	ModeSettingStatusColor          // status の色選択ピッカー
	ModeSettingStatusMove           // status の位置変更モード
	ModeSettingStatusDeleteConfirm  // status 削除確認
	ModeSettingField                // 設定画面 field サブ (中央ペインにフォーカス)
	ModeSettingFieldAttribute       // field の属性 (右ペインにフォーカス)
	ModeSettingFieldAdd             // field の新規追加 (name + type 2 行モーダル)
	ModeSettingFieldRename          // field の name 変更入力
	ModeSettingFieldMove            // field の position 変更モード
	ModeSettingFieldDeleteConfirm   // field 削除確認
	ModeEditFieldValue              // 詳細画面で text 型 field の値編集 (textinput)
	ModeEditFieldDateValue          // 詳細画面で date 型 field の値編集 (calendar)
	ModeOperation                   // タスクリストで o を押した直後の operation 入力待ち状態
	ModeTagPicker                   // タグ追加/解除モーダル (create input + 既存タグリスト)
	ModeTagColorPicker              // タグの色変更ピッカー (ModeTagPicker から c キーで遷移)
	ModeTagPickerRename             // タグ名変更入力 (ModeTagPicker から r キーで遷移)
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
	case ModeSettingStatusMove:
		return "settingstatusmove"
	case ModeSettingStatusDeleteConfirm:
		return "settingstatusdeleteconfirm"
	case ModeSettingField:
		return "settingfield"
	case ModeSettingFieldAttribute:
		return "settingfieldattribute"
	case ModeSettingFieldAdd:
		return "settingfieldadd"
	case ModeSettingFieldRename:
		return "settingfieldrename"
	case ModeSettingFieldMove:
		return "settingfieldmove"
	case ModeSettingFieldDeleteConfirm:
		return "settingfielddeleteconfirm"
	case ModeEditFieldValue:
		return "editfieldvalue"
	case ModeEditFieldDateValue:
		return "editfielddatevalue"
	case ModeOperation:
		return "operation"
	case ModeTagPicker:
		return "tagpicker"
	case ModeTagColorPicker:
		return "tagcolorpicker"
	case ModeTagPickerRename:
		return "tagpickerrename"
	default:
		return "unknown"
	}
}
