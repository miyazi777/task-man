package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

// 詳細画面でカーソルを当てる対象フィールド。
const (
	detailFieldTitle  = 0
	detailFieldStatus = 1
	detailFieldFiles  = 2
)

// editorFinishedMsg は外部エディタが終了したときに自身に通知する内部メッセージ。
type editorFinishedMsg struct {
	err error
}

type Model struct {
	repo     storage.Repository
	tasks    []task.Task
	statuses task.StatusList
	yamlDir  string            // tasks.yaml の置かれたディレクトリ
	cfg      storage.AppConfig // data_base_directory + editor
	cursor   int
	mode     Mode
	prevMode Mode

	keys     keyMap
	input    textinput.Model
	inputErr error // 入力ライブ検証用 (禁止文字・長さ超過)

	detailCursor       int      // 0=Title, 1=Status, 2=Files
	statusPickerCursor int      // sorted statuses のインデックス
	files              []string // 現タスクのディレクトリ内ファイル一覧
	fileCursor         int      // files のインデックス

	width  int
	height int

	saveErr error
}

func NewModel(repo storage.Repository, initial []task.Task, statuses task.StatusList, yamlDir string, cfg storage.AppConfig) Model {
	m := Model{
		repo:     repo,
		tasks:    initial,
		statuses: statuses,
		yamlDir:  yamlDir,
		cfg:      cfg,
		mode:     ModeList,
		keys:     newKeyMap(),
	}
	return m.withFilesRefreshed()
}

// withFilesRefreshed は m.cursor が指すタスクのディレクトリ配下のファイル一覧を再読込する。
// タスクが無い・ディレクトリ無しの場合は空にする。エラーは握り潰し (UX 上は空表示で十分)。
func (m Model) withFilesRefreshed() Model {
	if len(m.tasks) == 0 || m.cursor >= len(m.tasks) {
		m.files = nil
		m.fileCursor = 0
		return m
	}
	files, _ := storage.ListTaskFiles(m.yamlDir, m.cfg.DataBaseDirectory, m.tasks[m.cursor].Title)
	m.files = files
	if m.fileCursor >= len(files) {
		m.fileCursor = 0
	}
	return m
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
	case editorFinishedMsg:
		if msg.err != nil {
			m.saveErr = msg.err
		}
		m = m.withFilesRefreshed()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// textinput の cursor blink などを反映するため、入力中のモードでは Update を委譲する。
	if m.mode == ModeNewTask || m.mode == ModeEditTitle || m.mode == ModeAddFile || m.mode == ModeRenameFile {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeAddFile:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" || m.inputErr != nil {
				return m, nil
			}
			title := m.tasks[m.cursor].Title
			if err := storage.CreateFile(m.yamlDir, m.cfg.DataBaseDirectory, title, name); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			// 追加ファイルにカーソルを合わせる
			for i, f := range m.files {
				if f == name {
					m.fileCursor = i
					break
				}
			}
			return m, nil
		case "esc":
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = storage.ValidateFileNameChars(m.input.Value())
		return m, cmd

	case ModeRenameFile:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" || m.inputErr != nil {
				return m, nil
			}
			title := m.tasks[m.cursor].Title
			oldName := m.files[m.fileCursor]
			if err := storage.RenameFile(m.yamlDir, m.cfg.DataBaseDirectory, title, oldName, name); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			for i, f := range m.files {
				if f == name {
					m.fileCursor = i
					break
				}
			}
			return m, nil
		case "esc":
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = storage.ValidateFileNameChars(m.input.Value())
		return m, cmd

	case ModeDeleteFileConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			title := m.tasks[m.cursor].Title
			name := m.files[m.fileCursor]
			if err := storage.DeleteFile(m.yamlDir, m.cfg.DataBaseDirectory, title, name); err != nil {
				m.saveErr = err
				m.mode = ModeDetail
				return m, nil
			}
			m.mode = ModeDetail
			m = m.withFilesRefreshed()
			// 削除位置に合わせてカーソルを調整
			if m.fileCursor >= len(m.files) {
				if len(m.files) == 0 {
					m.fileCursor = 0
					m.detailCursor = detailFieldStatus
				} else {
					m.fileCursor = len(m.files) - 1
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeDetail
			return m, nil
		}
		return m, nil

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
				return m, nil
			}
			if m.inputErr != nil {
				return m, nil
			}
			initialStatusID := m.firstStatusID()
			if initialStatusID == 0 {
				m.saveErr = fmt.Errorf("no statuses defined")
				return m, nil
			}
			t := task.Task{
				ID:       task.NextID(m.tasks),
				Title:    title,
				StatusID: initialStatusID,
			}
			if err := t.Validate(m.statuses); err != nil {
				m.saveErr = err
				return m, nil
			}
			// 情報格納用ディレクトリと memo.md を先に作る。衝突時はタスク自体を追加しない。
			if err := storage.CreateTaskData(m.yamlDir, m.cfg.DataBaseDirectory, t.Title); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks = append(m.tasks, t)
			if err := m.repo.Save(m.tasks, m.statuses, m.cfg); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.cursor = len(m.tasks) - 1
			m.mode = ModeList
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			return m, nil
		case "esc":
			m.mode = ModeList
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateTitleChars(m.input.Value())
		return m, cmd

	case ModeEditTitle:
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(m.input.Value())
			if title == "" {
				return m, nil
			}
			if m.inputErr != nil {
				return m, nil
			}
			updated := m.tasks[m.cursor]
			updated.Title = title
			if err := updated.Validate(m.statuses); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks[m.cursor] = updated
			if err := m.repo.Save(m.tasks, m.statuses, m.cfg); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			return m, nil
		case "esc":
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateTitleChars(m.input.Value())
		return m, cmd

	case ModeEditStatus:
		sorted := m.statuses.Sorted()
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.statusPickerCursor > 0 {
				m.statusPickerCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.statusPickerCursor < len(sorted)-1 {
				m.statusPickerCursor++
			}
			return m, nil
		}
		switch msg.String() {
		case "enter":
			if len(sorted) == 0 {
				m.mode = ModeDetail
				return m, nil
			}
			m.tasks[m.cursor].StatusID = sorted[m.statusPickerCursor].ID
			if err := m.repo.Save(m.tasks, m.statuses, m.cfg); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			return m, nil
		case "esc":
			m.mode = ModeDetail
			return m, nil
		}
		return m, nil

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
			if m.detailCursor == detailFieldFiles {
				if m.fileCursor > 0 {
					m.fileCursor--
				} else {
					m.detailCursor = detailFieldStatus
				}
			} else if m.detailCursor > 0 {
				m.detailCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			switch m.detailCursor {
			case detailFieldStatus:
				if len(m.files) > 0 {
					m.detailCursor = detailFieldFiles
					m.fileCursor = 0
				}
			case detailFieldFiles:
				if m.fileCursor < len(m.files)-1 {
					m.fileCursor++
				}
			default:
				m.detailCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Confirm):
			switch m.detailCursor {
			case detailFieldTitle:
				inputW := popupWidth(m.width) - 7
				if inputW < 1 {
					inputW = 1
				}
				m.input = newTitleInput(inputW)
				m.input.SetValue(m.tasks[m.cursor].Title)
				m.input.CursorEnd()
				m.inputErr = task.ValidateTitleChars(m.input.Value())
				m.mode = ModeEditTitle
				return m, textinput.Blink
			case detailFieldStatus:
				m.statusPickerCursor = sortedStatusIndex(m.statuses, m.tasks[m.cursor].StatusID)
				m.mode = ModeEditStatus
				return m, nil
			case detailFieldFiles:
				if len(m.files) == 0 {
					return m, nil
				}
				return m.openCurrentFile()
			}
			return m, nil
		case key.Matches(msg, m.keys.AddFile):
			if m.detailCursor != detailFieldFiles {
				return m, nil
			}
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newFileNameInput(inputW)
			m.inputErr = nil
			m.mode = ModeAddFile
			return m, textinput.Blink
		case key.Matches(msg, m.keys.RenameFile):
			if m.detailCursor != detailFieldFiles || len(m.files) == 0 {
				return m, nil
			}
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newFileNameInput(inputW)
			m.input.SetValue(m.files[m.fileCursor])
			m.input.CursorEnd()
			m.inputErr = storage.ValidateFileNameChars(m.input.Value())
			m.mode = ModeRenameFile
			return m, textinput.Blink
		case key.Matches(msg, m.keys.DeleteFile):
			if m.detailCursor != detailFieldFiles || len(m.files) == 0 {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeDeleteFileConfirm
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
				m = m.withFilesRefreshed()
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
				m = m.withFilesRefreshed()
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if len(m.tasks) > 0 {
				m.mode = ModeDetail
				m.detailCursor = detailFieldTitle
			}
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.inputErr = nil
			m.mode = ModeNewTask
			return m, textinput.Blink
		}
		return m, nil
	}
	return m, nil
}

// openCurrentFile は現在のファイルカーソルが指すファイルを外部エディタで開く tea.Cmd を返す。
func (m Model) openCurrentFile() (Model, tea.Cmd) {
	taskTitle := m.tasks[m.cursor].Title
	taskDir := storage.TaskDir(m.yamlDir, m.cfg.DataBaseDirectory, taskTitle)
	filePath := filepath.Join(taskDir, m.files[m.fileCursor])

	cmd, err := buildEditorCmd(m.cfg.Editor, filePath)
	if err != nil {
		m.saveErr = err
		return m, nil
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// firstStatusID は sequence/id 順で先頭の status id を返す。statuses が空なら 0。
func (m Model) firstStatusID() int {
	sorted := m.statuses.Sorted()
	if len(sorted) == 0 {
		return 0
	}
	return sorted[0].ID
}

func sortedStatusIndex(sl task.StatusList, id int) int {
	sorted := sl.Sorted()
	for i, s := range sorted {
		if s.ID == id {
			return i
		}
	}
	return 0
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
	rightW := m.width - leftW - 1
	bodyH := m.height - 1

	listFocused := m.mode == ModeList || m.mode == ModeQuitConfirm
	detailFocused := m.mode == ModeDetail || m.mode == ModeEditTitle || m.mode == ModeEditStatus

	left := renderList(m.tasks, m.statuses, m.cursor, listFocused, leftW, bodyH)

	var current *task.Task
	if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
		t := m.tasks[m.cursor]
		current = &t
	}
	right := renderDetail(current, m.statuses, m.files, detailFocused, m.detailCursor, m.fileCursor, rightW, bodyH)

	divider := strings.Repeat("│\n", bodyH)
	divider = styleDivider.Render(strings.TrimRight(divider, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	footer := renderFooter(m.mode, m.detailCursor, m.width)

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	switch m.mode {
	case ModeNewTask, ModeEditTitle:
		view = overlayInputPopup(view, "Title:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeAddFile:
		view = overlayInputPopup(view, "Filename:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeRenameFile:
		view = overlayInputPopup(view, "Rename:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeEditStatus:
		view = overlayStatusPicker(view, m.statuses.Sorted(), m.statusPickerCursor, m.width, m.height-1)
	}

	if m.saveErr != nil {
		view += "\n" + lipgloss.NewStyle().Foreground(colorDanger).Render(fmt.Sprintf("save error: %v", m.saveErr))
	}
	return view
}

// overlayInputPopup は単一行入力ポップアップを画面中央にオーバーレイする。
// inputErr が non-nil のときは入力行の下にエラー行を追加表示する。
func overlayInputPopup(bg, label, inputView string, inputErr error, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render(label), innerW)
	bottomRow := buildBorderRow("╰", "╯", stylePopupHint.Render("Enter:save  Esc:discard"), innerW)

	if w := ansi.StringWidth(inputView); w > contentW {
		inputView = ansi.Truncate(inputView, contentW, "")
	}
	inputPadded := stylePopupFill.Width(contentW).Render(inputView)
	inputRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		inputPadded +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	rows := []string{topRow, inputRow}
	if inputErr != nil {
		errMsg := stylePopupError.Render("! " + inputErr.Error())
		if w := ansi.StringWidth(errMsg); w > contentW {
			errMsg = ansi.Truncate(errMsg, contentW, "")
		}
		errPadded := stylePopupFill.Width(contentW).Render(errMsg)
		errRow := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			errPadded +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, errRow)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// overlayStatusPicker は status の選択肢リストをポップアップとして中央オーバーレイする。
// sortedStatuses は sequence 昇順の statuses。currentIdx は選択中インデックス。
func overlayStatusPicker(bg string, sortedStatuses task.StatusList, currentIdx, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Status:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", stylePopupHint.Render("Enter:save  Esc:discard"), innerW)

	rows := []string{topRow}
	for i, s := range sortedStatuses {
		raw := "  " + s.Label
		if w := ansi.StringWidth(raw); w > contentW {
			raw = ansi.Truncate(raw, contentW, "")
		}
		var padded string
		if i == currentIdx {
			padded = stylePopupCursorRow.Width(contentW).Render(raw)
		} else {
			padded = stylePopupFill.Foreground(colorText).Width(contentW).Render(raw)
		}
		row := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			padded +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, row)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

func centerOverlay(popup, bg string, screenW, screenH int) string {
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
