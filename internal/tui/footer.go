package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type hintItem struct {
	key   string
	label string
}

func renderFooter(mode Mode, width int) string {
	var content string
	switch mode {
	case ModeQuitConfirm:
		content = renderQuitPrompt()
	case ModeList:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"l/→", "detail"}, {"a", "new"}, {"q", "quit"},
		})
	case ModeDetail:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"enter", "edit/open"}, {"h/←", "back"}, {"q", "quit"},
		})
	case ModeNewTask:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeEditTitle:
		content = renderHints([]hintItem{
			{"Enter", "save"}, {"Esc", "discard"},
		})
	case ModeEditStatus:
		content = renderHints([]hintItem{
			{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "discard"},
		})
	}

	bar := lipgloss.NewStyle().
		Background(colorFooterBg).
		Foreground(colorMuted).
		Width(width).
		Padding(0, 1).
		Render(content)
	return bar
}

func renderHints(items []hintItem) string {
	var parts []string
	for _, it := range items {
		k := styleFooterKey.Render(it.key)
		v := lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Render(":" + it.label)
		parts = append(parts, k+v)
	}
	sep := lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Render("  ")
	return strings.Join(parts, sep)
}

func renderQuitPrompt() string {
	return styleQuitPromptText.Render("quit?  ") +
		styleQuitPromptYes.Render("y") +
		styleQuitPromptText.Render(":quit  ") +
		styleQuitPromptNo.Render("n/esc") +
		styleQuitPromptText.Render(":cancel")
}
