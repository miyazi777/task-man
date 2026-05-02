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
	ModeTrashConfirm      // タスクをゴミ箱へ移動するときの確認
	ModeDeleteTaskConfirm // ゴミ箱内タスクの完全削除確認
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
	default:
		return "unknown"
	}
}
