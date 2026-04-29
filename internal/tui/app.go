package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

type Model struct {
	repo     storage.Repository
	tasks    []task.Task
	cursor   int
	mode     Mode
	prevMode Mode

	keys  keyMap
	input textinput.Model

	width  int
	height int

	saveErr error
}

func NewModel(repo storage.Repository, initial []task.Task) Model {
	return Model{
		repo:  repo,
		tasks: initial,
		mode:  ModeList,
		keys:  newKeyMap(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.mode == ModeNewTask {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeQuitConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			return m, tea.Quit
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = m.prevMode
			return m, nil
		}
		return m, nil

	case ModeNewTask:
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(m.input.Value())
			if title == "" {
				// 空タイトルは許可しない (バリデーションエラー回避)。
				return m, nil
			}
			t := task.Task{
				ID:     task.NextID(m.tasks),
				Title:  title,
				Status: task.StatusTodo,
			}
			if err := t.Validate(); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks = append(m.tasks, t)
			if err := m.repo.Save(m.tasks); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.cursor = len(m.tasks) - 1
			m.mode = ModeList
			m.input = textinput.Model{}
			return m, nil
		case "esc":
			m.mode = ModeList
			m.input = textinput.Model{}
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeDetail:
		if len(m.tasks) == 0 {
			m.mode = ModeList
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeList
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.tasks[m.cursor].Status = m.tasks[m.cursor].Status.Prev()
			if err := m.repo.Save(m.tasks); err != nil {
				m.saveErr = err
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.tasks[m.cursor].Status = m.tasks[m.cursor].Status.Next()
			if err := m.repo.Save(m.tasks); err != nil {
				m.saveErr = err
			}
			return m, nil
		}
		return m, nil

	case ModeList:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if len(m.tasks) > 0 {
				m.mode = ModeDetail
			}
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			// 入力フィールド値の最大幅 = contentW (= popupOuterW - 4) - prompt(2) - cursor(1)。
			// textinput.View() は m.Width + 3 cell を返すため、ここから 3 を差し引く。
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.mode = ModeNewTask
			return m, textinput.Blink
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	leftW := m.width / 3
	if leftW < 24 {
		leftW = 24
	}
	if leftW > m.width-20 {
		leftW = m.width - 20
	}
	rightW := m.width - leftW - 1 // divider 1 桁分
	bodyH := m.height - 1         // フッター 1 行分

	listFocused := m.mode == ModeList || m.mode == ModeQuitConfirm

	left := renderList(m.tasks, m.cursor, listFocused, leftW, bodyH)

	var current *task.Task
	if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
		t := m.tasks[m.cursor]
		current = &t
	}
	right := renderDetail(current, m.mode == ModeDetail, rightW, bodyH)

	divider := strings.Repeat("│\n", bodyH)
	divider = styleDivider.Render(strings.TrimRight(divider, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	footer := renderFooter(m.mode, m.width)

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	// 新規タスク入力中はポップアップを画面中央にオーバーレイ。
	if m.mode == ModeNewTask {
		view = overlayNewTaskPopup(view, m.input.View(), m.width, m.height-1)
	}

	if m.saveErr != nil {
		view += "\n" + lipgloss.NewStyle().Foreground(colorDanger).Render(fmt.Sprintf("save error: %v", m.saveErr))
	}
	return view
}

func overlayNewTaskPopup(bg, inputView string, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	// 内側コンテンツ幅 = 外形 - border(2) - padding(2)。
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	// 罫線の内側 (左右コーナー間) の cell 数。
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Title:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", stylePopupHint.Render("Enter:save  Esc:discard"), innerW)

	// 入力行: │ {input clamped/padded to contentW} │
	// textinput.View() の幅が contentW を超えないように切り詰め、不足分はポップアップ背景色で埋める。
	if w := ansi.StringWidth(inputView); w > contentW {
		inputView = ansi.Truncate(inputView, contentW, "")
	}
	inputPadded := stylePopupFill.Width(contentW).Render(inputView)
	inputRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		inputPadded +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	popup := lipgloss.JoinVertical(lipgloss.Left, topRow, inputRow, bottomRow)

	popupH := lipgloss.Height(popup)
	popupRenderedW := lipgloss.Width(popup)

	x := (screenW - popupRenderedW) / 2
	y := (screenH - popupH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return PlaceOverlay(x, y, popup, bg)
}

// buildBorderRow は両端のコーナー文字とラベル埋め込みからなる罫線行を構築する。
// パターン: {leftCorner}─{label}─...─{rightCorner}
// label は左寄せ (1 cell の罫線文字を挟んだ直後) で配置する。
func buildBorderRow(leftCorner, rightCorner, label string, innerW int) string {
	labelW := ansi.StringWidth(label)
	if labelW > innerW-2 {
		label = ansi.Truncate(label, innerW-2, "")
		labelW = ansi.StringWidth(label)
	}
	const leadDashes = 1
	tailDashes := innerW - leadDashes - labelW
	if tailDashes < 0 {
		tailDashes = 0
	}
	return stylePopupBorder.Render(leftCorner+strings.Repeat("─", leadDashes)) +
		label +
		stylePopupBorder.Render(strings.Repeat("─", tailDashes)+rightCorner)
}
