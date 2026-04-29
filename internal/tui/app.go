package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
			m.input = newTitleInput()
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

	var pendingPlaceholder *string
	if m.mode == ModeNewTask {
		s := ""
		pendingPlaceholder = &s
	}

	left := renderList(m.tasks, m.cursor, listFocused, leftW, bodyH, pendingPlaceholder)

	var right string
	switch m.mode {
	case ModeNewTask:
		right = renderNewTaskDetail(m.input.View(), rightW, bodyH)
	default:
		var current *task.Task
		if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
			t := m.tasks[m.cursor]
			current = &t
		}
		right = renderDetail(current, m.mode == ModeDetail, rightW, bodyH)
	}

	divider := strings.Repeat("│\n", bodyH)
	divider = styleDivider.Render(strings.TrimRight(divider, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	footer := renderFooter(m.mode, m.width)

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	if m.saveErr != nil {
		view += "\n" + lipgloss.NewStyle().Foreground(colorDanger).Render(fmt.Sprintf("save error: %v", m.saveErr))
	}
	return view
}
