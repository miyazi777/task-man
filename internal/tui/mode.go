package tui

// Mode は TUI のスクリーン・入力フェーズの識別子。同じキーでも Mode によって意味が変わる。
type Mode int

// TUI のモード列挙。値は iota ベースなので順序は安定だが yaml には永続化しない。
const (
	// ModeList はタスクリスト画面 (起動直後のデフォルトモード)。
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
	ModeTrashConfirm               // タスクをゴミ箱へ移動するときの確認
	ModeDeleteTaskConfirm          // ゴミ箱内タスクの完全削除確認
	ModePrefix                     // ; を押した直後の prefix 入力待ち状態
	ModeSetting                    // 設定画面 (左メニュー側にフォーカス)
	ModeSettingGeneral             // 設定画面 general サブ (yaml パスなど読み取り専用情報)
	ModeSettingGeneralEdit         // 設定画面 general の data_base_directory 編集モーダル
	ModeSettingStatus              // 設定画面 status サブ (右ペイン側にフォーカス)
	ModeSettingStatusRename        // status のラベル変更入力
	ModeSettingStatusAdd           // status の新規追加入力
	ModeSettingStatusColor         // status の色選択ピッカー
	ModeSettingStatusMove          // status の位置変更モード
	ModeSettingStatusDeleteConfirm // status 削除確認
	ModeSettingField               // 設定画面 field サブ (中央ペインにフォーカス)
	ModeSettingFieldAttribute      // field の属性 (右ペインにフォーカス)
	ModeSettingFieldAdd            // field の新規追加 (name + type 2 行モーダル)
	ModeSettingFieldRename         // field の name 変更入力
	ModeSettingFieldMove           // field の position 変更モード
	ModeSettingFieldDeleteConfirm  // field 削除確認
	ModeEditFieldValue             // 詳細画面で text 型 field の値編集 (textinput)
	ModeEditFieldDateValue         // 詳細画面で date 型 field の値編集 (calendar)
	ModeOperation                  // タスクリストで o を押した直後の operation 入力待ち状態
	ModeTagPicker                  // タグ追加/解除モーダル (create input + 既存タグリスト)
	ModeTagColorPicker             // タグの色変更ピッカー (ModeTagPicker から c キーで遷移)
	ModeTagPickerRename            // タグ名変更入力 (ModeTagPicker から r キーで遷移)
	ModeTagPickerDeleteConfirm     // タグ削除確認 (y/n オーバーレイ)
	ModeLayout                     // タスクリスト画面のレイアウト調整 (;→l で突入)
	ModeFileOpener                 // 詳細画面 file list で o/enter 押下時のアプリ選択モーダル

	// ModeSettingApplication 系: 設定画面 application サブモード群。
	ModeSettingApplication              // 中央ペイン (application 一覧) にフォーカス
	ModeSettingApplicationAttribute     // 右ペイン (id/name/run) にフォーカス
	ModeSettingApplicationAdd           // 新規追加 (name + run の 2 行モーダル)
	ModeSettingApplicationEditName      // name の編集 (textinput)
	ModeSettingApplicationEditRun       // run の編集 (textinput)
	ModeSettingApplicationMove          // 順序変更モード
	ModeSettingApplicationDeleteConfirm // 削除確認

	// ModeSettingFileOpener 系: 設定画面 file_opener サブモード群。
	ModeSettingFileOpener              // 中央ペイン (opener 一覧) にフォーカス
	ModeSettingFileOpenerAttribute     // 右ペイン (extension/applications/default_app) にフォーカス
	ModeSettingFileOpenerAdd           // 新規追加 (extension モーダル)
	ModeSettingFileOpenerEditExtension // extension の編集
	ModeSettingFileOpenerEditApps      // applications 編集 (multi-select サブモーダル)
	ModeSettingFileOpenerEditDefault   // default_app picker
	ModeSettingFileOpenerMove          // 順序変更モード
	ModeSettingFileOpenerDeleteConfirm // 削除確認
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
	case ModeSettingGeneral:
		return "settinggeneral"
	case ModeSettingGeneralEdit:
		return "settinggeneraledit"
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
	case ModeTagPickerDeleteConfirm:
		return "tagpickerdeleteconfirm"
	case ModeLayout:
		return "layout"
	case ModeFileOpener:
		return "fileopener"
	case ModeSettingApplication:
		return "settingapplication"
	case ModeSettingApplicationAttribute:
		return "settingapplicationattribute"
	case ModeSettingApplicationAdd:
		return "settingapplicationadd"
	case ModeSettingApplicationEditName:
		return "settingapplicationeditname"
	case ModeSettingApplicationEditRun:
		return "settingapplicationeditrun"
	case ModeSettingApplicationMove:
		return "settingapplicationmove"
	case ModeSettingApplicationDeleteConfirm:
		return "settingapplicationdeleteconfirm"
	case ModeSettingFileOpener:
		return "settingfileopener"
	case ModeSettingFileOpenerAttribute:
		return "settingfileopenerattribute"
	case ModeSettingFileOpenerAdd:
		return "settingfileopeneradd"
	case ModeSettingFileOpenerEditExtension:
		return "settingfileopenereditextension"
	case ModeSettingFileOpenerEditApps:
		return "settingfileopenereditapps"
	case ModeSettingFileOpenerEditDefault:
		return "settingfileopenereditdefault"
	case ModeSettingFileOpenerMove:
		return "settingfileopenermove"
	case ModeSettingFileOpenerDeleteConfirm:
		return "settingfileopenerdeleteconfirm"
	default:
		return "unknown"
	}
}
