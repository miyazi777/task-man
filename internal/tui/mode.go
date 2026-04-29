package tui

type Mode int

const (
	ModeList Mode = iota
	ModeDetail
	ModeNewTask
	ModeEditTitle
	ModeEditStatus
	ModeQuitConfirm
)

func (m Mode) String() string {
	switch m {
	case ModeList:
		return "list"
	case ModeDetail:
		return "detail"
	case ModeNewTask:
		return "newtask"
	case ModeEditTitle:
		return "edittitle"
	case ModeEditStatus:
		return "editstatus"
	case ModeQuitConfirm:
		return "quitconfirm"
	default:
		return "unknown"
	}
}
