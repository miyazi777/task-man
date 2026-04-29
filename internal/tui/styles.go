package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// モック (docs/mockups/*.svg) の Catppuccin 系カラーをベースに調整。
var (
	colorText      = lipgloss.Color("#cdd6f4")
	colorMuted     = lipgloss.Color("#a6adc8")
	colorDim       = lipgloss.Color("#7f849c")
	colorSubtle    = lipgloss.Color("#6c7086")
	colorAccent    = lipgloss.Color("#89b4fa") // フォーカス・カーソル
	colorTodo      = lipgloss.Color("#6c7086") // todo: グレー
	colorDoing     = lipgloss.Color("#fab387") // doing: オレンジ
	colorDone      = lipgloss.Color("#a6e3a1") // done: グリーン
	colorWarn      = lipgloss.Color("#f9e2af") // 入力中・終了確認アクセント
	colorDanger    = lipgloss.Color("#f38ba8") // y:quit
	colorOK        = lipgloss.Color("#a6e3a1") // n:cancel
	colorDivider   = lipgloss.Color("#313244")
	colorFooterBg  = lipgloss.Color("#313244")
)

var (
	styleListItem        = lipgloss.NewStyle().Foreground(colorText).Padding(0, 1)
	styleListItemDim     = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 1)
	styleListItemCursor  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Padding(0, 1)
	styleCursorMarker    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleLabel           = lipgloss.NewStyle().Foreground(colorSubtle)
	styleLabelFocused    = lipgloss.NewStyle().Foreground(colorAccent)
	styleValue           = lipgloss.NewStyle().Foreground(colorText)
	styleValueDim        = lipgloss.NewStyle().Foreground(colorSubtle)
	styleDivider         = lipgloss.NewStyle().Foreground(colorDivider)
	styleFooter          = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Padding(0, 1).Width(0)
	styleFooterKey       = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorText).Bold(true)
	styleQuitPromptText  = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorText)
	styleQuitPromptYes   = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorDanger).Bold(true)
	styleQuitPromptNo    = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorOK).Bold(true)
	colorPopupBg = lipgloss.Color("#11111b")

	stylePopupLabel  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorPopupBg)
	stylePopupHint   = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Background(colorPopupBg)
	stylePopupFill   = lipgloss.NewStyle().Background(colorPopupBg)
	stylePopupBorder = lipgloss.NewStyle().Foreground(colorAccent).Background(colorPopupBg)
)

func statusStyle(s task.Status) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	switch s {
	case task.StatusTodo:
		return base.Foreground(colorTodo)
	case task.StatusDoing:
		return base.Foreground(colorDoing)
	case task.StatusDone:
		return base.Foreground(colorDone)
	default:
		return base.Foreground(colorMuted)
	}
}
