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
	repo          storage.Repository
	tasks         []task.Task
	statuses      task.StatusList
	yamlDir       string            // tasks.yaml の置かれたディレクトリ
	cfg           storage.AppConfig // data_base_directory + editor
	rows          []listRow         // 表示用フラット行リスト (ステータスでグループ化)
	collapsed     map[int]bool      // statusID → 折りたたみ中
	taskCollapsed map[int]bool      // taskID → サブタスクを折りたたみ中
	moving        int               // ModeMove 中の移動対象タスク ID (0 なら移動中でない)
	moveSnapshot  []task.Task       // ModeMove 開始時の m.tasks のスナップショット (esc で復元)
	viewTrash     bool              // true: ゴミ箱ビュー (IsTrashBox=true のタスクのみ)、false: 通常リスト
	cursor        int               // m.rows のインデックス (旧: m.tasks のインデックス)
	mode          Mode
	prevMode      Mode

	keys     keyMap
	input    textinput.Model
	inputErr error // 入力ライブ検証用 (禁止文字・長さ超過)

	detailCursor       int      // 0=Title, 1=Status, 2=Files
	statusPickerCursor int      // sorted statuses のインデックス
	files              []string // 現タスクのディレクトリ内ファイル一覧
	fileCursor         int      // files のインデックス

	// 設定画面 (ModeSetting / ModeSettingStatus*) 用の状態
	settingMenuCursor   int      // 左メニュー (今は status のみ) のインデックス
	settingStatusCursor int      // 右ペイン: m.statuses.Sorted() のインデックス
	settingColorChoices []string // 色ピッカー候補 (#rrggbb 8 色)
	settingColorCursor  int      // 色ピッカー上のカーソル

	width  int
	height int

	saveErr error
}

func NewModel(repo storage.Repository, initial []task.Task, statuses task.StatusList, yamlDir string, cfg storage.AppConfig) Model {
	collapsed := make(map[int]bool)
	for _, s := range statuses {
		if s.Collapsed {
			collapsed[s.ID] = true
		}
	}
	taskCollapsed := make(map[int]bool)
	for _, t := range initial {
		if t.Collapsed {
			taskCollapsed[t.ID] = true
		}
	}
	m := Model{
		repo:          repo,
		tasks:         initial,
		statuses:      statuses,
		yamlDir:       yamlDir,
		cfg:           cfg,
		collapsed:     collapsed,
		taskCollapsed: taskCollapsed,
		mode:          ModeList,
		keys:          newKeyMap(),
	}
	m = m.withRowsRebuilt()
	if first := firstNavigable(m.rows); first >= 0 {
		m.cursor = first
	}
	return m.withFilesRefreshed()
}

// persist は m.collapsed / m.taskCollapsed を m.statuses / m.tasks に同期した上で yaml へ書き出す。
// 折りたたみ状態を含むあらゆる永続化はこの関数経由で行う。
func (m *Model) persist() error {
	for i := range m.statuses {
		m.statuses[i].Collapsed = m.collapsed[m.statuses[i].ID]
	}
	for i := range m.tasks {
		m.tasks[i].Collapsed = m.taskCollapsed[m.tasks[i].ID]
	}
	return m.repo.Save(m.tasks, m.statuses, m.cfg)
}

// withRowsRebuilt はステータスとタスクの構成・折りたたみ状態から rows を再構築し、
// cursor が範囲外/separator にいる場合は近接の navigable 行へ寄せる。
func (m Model) withRowsRebuilt() Model {
	m.rows = buildRows(m.statuses, m.tasks, m.collapsed, m.taskCollapsed, m.viewTrash)
	if len(m.rows) == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if !isNavigable(m.rows[m.cursor].kind) {
		// 直前/直後で navigable な行へ
		next := nextNavigable(m.rows, m.cursor)
		if next == m.cursor {
			next = prevNavigable(m.rows, m.cursor)
		}
		if next != m.cursor {
			m.cursor = next
		}
	}
	return m
}

// currentTask は cursor が指す行が task のときその index を返す。それ以外は ok=false。
func (m Model) currentTask() (task.Task, int, bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return task.Task{}, 0, false
	}
	r := m.rows[m.cursor]
	if r.kind != rowTask {
		return task.Task{}, 0, false
	}
	return m.tasks[r.taskIndex], r.taskIndex, true
}

// applyCollapseChange は ModeList でカーソル位置の status / task の折りたたみ状態を
// 指定値 (collapse=true なら閉じる、false なら開く) に変更する。
// 既に同じ状態のとき、子を持たないタスクのとき、対象外の行のときは no-op。
// 変更があれば rows をリビルドし、cursor を元の対象行に追従させて永続化する。
func (m Model) applyCollapseChange(collapse bool) (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	r := m.rows[m.cursor]
	switch r.kind {
	case rowStatus:
		sid := r.statusID
		if m.collapsed[sid] == collapse {
			return m, nil
		}
		if collapse {
			m.collapsed[sid] = true
		} else {
			delete(m.collapsed, sid)
		}
		m = m.withRowsRebuilt()
		if rr := findRowForStatus(m.rows, sid); rr >= 0 {
			m.cursor = rr
		}
		m = m.withFilesRefreshed()
		if err := m.persist(); err != nil {
			m.saveErr = err
		}
		return m, nil
	case rowTask:
		cur := m.tasks[r.taskIndex]
		if !taskHasChildren(m.tasks, cur.ID) {
			return m, nil
		}
		if m.taskCollapsed[cur.ID] == collapse {
			return m, nil
		}
		if collapse {
			m.taskCollapsed[cur.ID] = true
		} else {
			delete(m.taskCollapsed, cur.ID)
		}
		taskIdx := r.taskIndex
		m = m.withRowsRebuilt()
		if rr := findRowForTask(m.rows, taskIdx); rr >= 0 {
			m.cursor = rr
		}
		m = m.withFilesRefreshed()
		if err := m.persist(); err != nil {
			m.saveErr = err
		}
		return m, nil
	}
	return m, nil
}

// clampCursor はタスク削除等で rows が縮んだ後、cursor を範囲内に収め、
// separator 等の非 navigable 行に居る場合は近接の navigable 行へ寄せる。
func clampCursor(m Model) Model {
	if len(m.rows) == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if !isNavigable(m.rows[m.cursor].kind) {
		if next := nextNavigable(m.rows, m.cursor); next != m.cursor {
			m.cursor = next
		} else if prev := prevNavigable(m.rows, m.cursor); prev != m.cursor {
			m.cursor = prev
		}
	}
	return m
}

// cursorFollowMovingTask は m.moving が指すタスクの行へカーソルを合わせる。
// rebuild 直後に呼び出して移動対象を視覚的に追従させるための補助関数。
func (m Model) cursorFollowMovingTask() Model {
	if m.moving == 0 {
		return m
	}
	for i, t := range m.tasks {
		if t.ID == m.moving {
			if r := findRowForTask(m.rows, i); r >= 0 {
				m.cursor = r
			}
			return m
		}
	}
	return m
}

// currentStatusID は cursor が指す行が status (or 該当行のグループ) のステータス ID を返す。
// 行が separator で statusID 不明な場合などは 0 を返す。
func (m Model) currentStatusID() int {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return 0
	}
	return m.rows[m.cursor].statusID
}

// withFilesRefreshed は cursor が指すタスクのディレクトリ配下のファイル一覧を再読込する。
// status 行・separator 行や、ディレクトリ無しの場合は空にする。エラーは握り潰し (UX 上は空表示で十分)。
func (m Model) withFilesRefreshed() Model {
	t, _, ok := m.currentTask()
	if !ok {
		m.files = nil
		m.fileCursor = 0
		return m
	}
	files, _ := storage.ListTaskFiles(m.yamlDir, m.cfg.DataBaseDirectory, t.ID)
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
	if m.mode == ModeNewTask || m.mode == ModeNewSubtask || m.mode == ModeEditTitle ||
		m.mode == ModeAddFile || m.mode == ModeRenameFile ||
		m.mode == ModeSettingStatusRename || m.mode == ModeSettingStatusAdd {
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
			t, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			if err := storage.CreateFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, name); err != nil {
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
			t, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			oldName := m.files[m.fileCursor]
			if err := storage.RenameFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, oldName, name); err != nil {
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

	case ModeTrashConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			cur, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeList
				return m, nil
			}
			m.tasks = task.TrashTask(m.tasks, cur.ID)
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeList
			m = m.withRowsRebuilt()
			m = clampCursor(m)
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeList
			return m, nil
		}
		return m, nil

	case ModeDeleteTaskConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			cur, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeList
				return m, nil
			}
			newTasks, removedIDs := task.DeleteTaskSubtree(m.tasks, cur.ID)
			m.tasks = newTasks
			for _, id := range removedIDs {
				if err := storage.DeleteTaskData(m.yamlDir, m.cfg.DataBaseDirectory, id); err != nil {
					m.saveErr = err
				}
			}
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeList
			m = m.withRowsRebuilt()
			m = clampCursor(m)
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeList
			return m, nil
		}
		return m, nil

	case ModeDeleteFileConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			t, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			name := m.files[m.fileCursor]
			if err := storage.DeleteFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, name); err != nil {
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
			// 新規タスクの所属ステータス: カーソルが当たっている行の statusID を採用。
			// status 行 / task 行どちらでも m.rows[cursor].statusID で取得できる。
			// 該当しなければ (空など) sequence/id 順の先頭にフォールバック。
			initialStatusID := m.currentStatusID()
			if initialStatusID == 0 {
				initialStatusID = m.firstStatusID()
			}
			if initialStatusID == 0 {
				m.saveErr = fmt.Errorf("no statuses defined")
				return m, nil
			}
			t := task.Task{
				ID:       task.NextID(m.tasks),
				Title:    title,
				StatusID: initialStatusID,
				Position: task.NextPosition(m.tasks, 0),
			}
			if err := t.Validate(m.statuses); err != nil {
				m.saveErr = err
				return m, nil
			}
			// 情報格納用ディレクトリと memo.md を先に作る。衝突時はタスク自体を追加しない。
			if err := storage.CreateTaskData(m.yamlDir, m.cfg.DataBaseDirectory, t.ID); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks = append(m.tasks, t)
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			// 新規タスクの行へカーソル移動 (折りたたみ中なら展開してから)
			delete(m.collapsed, initialStatusID)
			m = m.withRowsRebuilt()
			if newRow := findRowForTask(m.rows, len(m.tasks)-1); newRow >= 0 {
				m.cursor = newRow
			}
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

	case ModeNewSubtask:
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(m.input.Value())
			if title == "" {
				return m, nil
			}
			if m.inputErr != nil {
				return m, nil
			}
			// カーソルが指すタスク自身を親としてサブタスクを作成する。
			cur, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeList
				m.input = textinput.Model{}
				m.inputErr = nil
				return m, nil
			}
			if taskDepth(m.tasks, cur.ID) >= task.MaxNestDepth {
				m.saveErr = fmt.Errorf("nesting depth limit (%d) reached", task.MaxNestDepth)
				return m, nil
			}
			t := task.Task{
				ID:       task.NextID(m.tasks),
				Title:    title,
				StatusID: cur.StatusID,
				ParentID: cur.ID,
				Position: task.NextPosition(m.tasks, cur.ID),
			}
			if err := t.Validate(m.statuses); err != nil {
				m.saveErr = err
				return m, nil
			}
			if err := storage.CreateTaskData(m.yamlDir, m.cfg.DataBaseDirectory, t.ID); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks = append(m.tasks, t)
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			delete(m.collapsed, cur.StatusID)
			m = m.withRowsRebuilt()
			if newRow := findRowForTask(m.rows, len(m.tasks)-1); newRow >= 0 {
				m.cursor = newRow
			}
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
			_, taskIdx, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			updated := m.tasks[taskIdx]
			updated.Title = title
			if err := updated.Validate(m.statuses); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks[taskIdx] = updated
			if err := m.persist(); err != nil {
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
			_, taskIdx, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			m.tasks[taskIdx].StatusID = sorted[m.statusPickerCursor].ID
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			// ステータス変更によりタスクのグループが変わるので rows を rebuild。
			// カーソルを移動先のタスク行に追従させる。
			m = m.withRowsRebuilt()
			if newRow := findRowForTask(m.rows, taskIdx); newRow >= 0 {
				m.cursor = newRow
			}
			m.mode = ModeDetail
			return m, nil
		case "esc":
			m.mode = ModeDetail
			return m, nil
		}
		return m, nil

	case ModeDetail:
		if _, _, ok := m.currentTask(); !ok {
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
			t, _, _ := m.currentTask()
			switch m.detailCursor {
			case detailFieldTitle:
				inputW := popupWidth(m.width) - 7
				if inputW < 1 {
					inputW = 1
				}
				m.input = newTitleInput(inputW)
				m.input.SetValue(t.Title)
				m.input.CursorEnd()
				m.inputErr = task.ValidateTitleChars(m.input.Value())
				m.mode = ModeEditTitle
				return m, textinput.Blink
			case detailFieldStatus:
				m.statusPickerCursor = sortedStatusIndex(m.statuses, t.StatusID)
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

	case ModeMove:
		switch {
		case key.Matches(msg, m.keys.Back):
			// esc: スナップショットから復元してキャンセル
			if m.moveSnapshot != nil {
				m.tasks = m.moveSnapshot
			}
			movedID := m.moving
			m.moveSnapshot = nil
			m.moving = 0
			m.mode = ModeList
			m = m.withRowsRebuilt()
			for i, t := range m.tasks {
				if t.ID == movedID {
					if r := findRowForTask(m.rows, i); r >= 0 {
						m.cursor = r
					}
					break
				}
			}
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.Move):
			// m: 確定 → 永続化して ModeList へ
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			movedID := m.moving
			m.moveSnapshot = nil
			m.moving = 0
			m.mode = ModeList
			m = m.withRowsRebuilt()
			for i, t := range m.tasks {
				if t.ID == movedID {
					if r := findRowForTask(m.rows, i); r >= 0 {
						m.cursor = r
					}
					break
				}
			}
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.tasks = task.MoveTaskUp(m.tasks, m.statuses, m.moving)
			m = m.withRowsRebuilt().cursorFollowMovingTask()
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.tasks = task.MoveTaskDown(m.tasks, m.statuses, m.moving)
			m = m.withRowsRebuilt().cursorFollowMovingTask()
			return m, nil
		case key.Matches(msg, m.keys.Open):
			m.tasks = task.IndentTask(m.tasks, m.moving)
			m = m.withRowsRebuilt().cursorFollowMovingTask()
			return m, nil
		case key.Matches(msg, m.keys.Close):
			m.tasks = task.OutdentTask(m.tasks, m.moving)
			m = m.withRowsRebuilt().cursorFollowMovingTask()
			return m, nil
		}
		return m, nil

	case ModePrefix:
		switch {
		case key.Matches(msg, m.keys.Back):
			// esc: prefix 入力をキャンセルし元のモードへ戻る。
			m.mode = m.prevMode
			return m, nil
		case key.Matches(msg, m.keys.PrefixTrash):
			// t: ゴミ箱ビューのトグル (タスクリストでの T と同じ動作)。
			m.viewTrash = !m.viewTrash
			m = m.withRowsRebuilt()
			if first := firstNavigable(m.rows); first >= 0 {
				m.cursor = first
			} else {
				m.cursor = 0
			}
			m = m.withFilesRefreshed()
			m.mode = m.prevMode
			return m, nil
		case key.Matches(msg, m.keys.PrefixSetting):
			// s: 設定画面へ遷移。左メニューにフォーカス。
			m.mode = ModeSetting
			m.settingMenuCursor = settingMenuStatus
			m.settingStatusCursor = 0
			return m, nil
		}
		return m, nil

	case ModeSetting:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// esc: 設定画面を抜けてタスクリストへ戻る。
			m.mode = ModeList
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingMenuCursor > 0 {
				m.settingMenuCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingMenuCursor < len(settingMenuLabels)-1 {
				m.settingMenuCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// enter: 詳細ペインへフォーカス移動。
			if m.settingMenuCursor == settingMenuStatus {
				m.mode = ModeSettingStatus
				if m.settingStatusCursor >= len(m.statuses) {
					m.settingStatusCursor = 0
				}
			}
			return m, nil
		}
		return m, nil

	case ModeSettingStatus:
		sorted := m.statuses.Sorted()
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// esc: 左メニュー側にフォーカスを戻す。
			m.mode = ModeSetting
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingStatusCursor > 0 {
				m.settingStatusCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingStatusCursor < len(sorted)-1 {
				m.settingStatusCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.RenameFile):
			// r: ステータス名変更
			if len(sorted) == 0 {
				return m, nil
			}
			cur := sorted[m.settingStatusCursor]
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.input.SetValue(cur.Label)
			m.input.CursorEnd()
			m.inputErr = task.ValidateTitleChars(m.input.Value())
			m.mode = ModeSettingStatusRename
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Color):
			// c: 色変更ピッカー
			if len(sorted) == 0 {
				return m, nil
			}
			cur := sorted[m.settingStatusCursor]
			m.settingColorChoices = statusColorChoices(cur.Color)
			m.settingColorCursor = nearestColorChoiceIndex(m.settingColorChoices, cur.Color)
			m.mode = ModeSettingStatusColor
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			// a: 新規ステータス追加 (カーソル位置に挿入)
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.inputErr = nil
			m.mode = ModeSettingStatusAdd
			return m, textinput.Blink
		}
		return m, nil

	case ModeSettingStatusRename:
		switch msg.String() {
		case "enter":
			label := strings.TrimSpace(m.input.Value())
			if label == "" || m.inputErr != nil {
				return m, nil
			}
			sorted := m.statuses.Sorted()
			if m.settingStatusCursor >= len(sorted) {
				m.mode = ModeSettingStatus
				return m, nil
			}
			id := sorted[m.settingStatusCursor].ID
			renamed, err := m.statuses.RenameByID(id, label)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.statuses = renamed
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeSettingStatus
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "esc":
			m.mode = ModeSettingStatus
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateTitleChars(m.input.Value())
		return m, cmd

	case ModeSettingStatusAdd:
		switch msg.String() {
		case "enter":
			label := strings.TrimSpace(m.input.Value())
			if label == "" || m.inputErr != nil {
				return m, nil
			}
			// カーソル位置に挿入。空リスト時は先頭。
			insertIdx := m.settingStatusCursor
			if len(m.statuses) == 0 {
				insertIdx = 0
			}
			defaultColor := statusColorChoices("")[0]
			inserted, newID, err := m.statuses.InsertAt(insertIdx, label, defaultColor)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.statuses = inserted
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			// カーソルを新規ステータス行に合わせる
			sorted := m.statuses.Sorted()
			for i, s := range sorted {
				if s.ID == newID {
					m.settingStatusCursor = i
					break
				}
			}
			m.mode = ModeSettingStatus
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "esc":
			m.mode = ModeSettingStatus
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateTitleChars(m.input.Value())
		return m, cmd

	case ModeSettingStatusColor:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.settingColorCursor > 0 {
				m.settingColorCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingColorCursor < len(m.settingColorChoices)-1 {
				m.settingColorCursor++
			}
			return m, nil
		}
		switch msg.String() {
		case "enter":
			sorted := m.statuses.Sorted()
			if m.settingStatusCursor >= len(sorted) || m.settingColorCursor >= len(m.settingColorChoices) {
				m.mode = ModeSettingStatus
				return m, nil
			}
			id := sorted[m.settingStatusCursor].ID
			newColor := m.settingColorChoices[m.settingColorCursor]
			updated, err := m.statuses.SetColorByID(id, newColor)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.statuses = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeSettingStatus
			return m, nil
		case "esc":
			m.mode = ModeSettingStatus
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
			if next := prevNavigable(m.rows, m.cursor); next != m.cursor {
				m.cursor = next
				m = m.withFilesRefreshed()
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if next := nextNavigable(m.rows, m.cursor); next != m.cursor {
				m.cursor = next
				m = m.withFilesRefreshed()
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// タスク行: 詳細遷移。status 行では何もしない (開閉は space に集約)。
			if _, _, ok := m.currentTask(); ok {
				m.mode = ModeDetail
				m.detailCursor = detailFieldTitle
			}
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// 開閉は space に集約。タスクリスト内では何もしない。
			return m, nil
		case key.Matches(msg, m.keys.Open):
			// l/→: カーソル位置の status 行 / task 行を展開する (既に展開済みなら no-op)。
			return m.applyCollapseChange(false)
		case key.Matches(msg, m.keys.Close):
			// h/←: カーソル位置の status 行 / task 行を折りたたむ (既に折りたたみ済みなら no-op)。
			return m.applyCollapseChange(true)
		case key.Matches(msg, m.keys.Move):
			// m: カーソル位置がタスク行なら ModeMove へ遷移。ゴミ箱ビューでは無効。
			if m.viewTrash {
				return m, nil
			}
			cur, _, ok := m.currentTask()
			if !ok {
				return m, nil
			}
			snapshot := make([]task.Task, len(m.tasks))
			copy(snapshot, m.tasks)
			m.moveSnapshot = snapshot
			m.moving = cur.ID
			m.mode = ModeMove
			m = m.withRowsRebuilt()
			return m, nil
		case key.Matches(msg, m.keys.ToggleTrash):
			// T: ゴミ箱ビュー → 通常リストへ戻る (ゴミ箱への遷移は ; t prefix に集約)。
			if !m.viewTrash {
				return m, nil
			}
			m.viewTrash = false
			m = m.withRowsRebuilt()
			if first := firstNavigable(m.rows); first >= 0 {
				m.cursor = first
			} else {
				m.cursor = 0
			}
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.Prefix):
			// ;: prefix モードへ遷移。次のキー入力を待つ。
			m.prevMode = m.mode
			m.mode = ModePrefix
			return m, nil
		case key.Matches(msg, m.keys.DeleteTask):
			// d: タスク行のとき、通常リスト → ゴミ箱へ移動、ゴミ箱ビュー → 完全削除。
			// 確認ポップアップを挟む。
			if _, _, ok := m.currentTask(); !ok {
				return m, nil
			}
			m.prevMode = m.mode
			if m.viewTrash {
				m.mode = ModeDeleteTaskConfirm
			} else {
				m.mode = ModeTrashConfirm
			}
			return m, nil
		case key.Matches(msg, m.keys.RestoreTask):
			// r: ゴミ箱ビューのタスク行のみ有効。trashed root から subtree を一括 restore。
			if !m.viewTrash {
				return m, nil
			}
			cur, _, ok := m.currentTask()
			if !ok {
				return m, nil
			}
			rootID := task.TrashRootID(m.tasks, cur.ID)
			m.tasks = task.RestoreTask(m.tasks, rootID)
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withRowsRebuilt()
			// 復帰したタスクは現在のビュー (trash) には居ない。近接の navigable 行に寄せる。
			m = clampCursor(m)
			m = m.withFilesRefreshed()
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			// status 行: その status 配下にトップレベルの新規タスクを作成
			// task 行: そのタスクの直下にサブタスクを作成 (深さ上限まで)
			// ゴミ箱ビューでは新規作成不可。
			if m.viewTrash {
				return m, nil
			}
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			if m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowStatus {
				m.input = newTitleInput(inputW)
				m.inputErr = nil
				m.mode = ModeNewTask
				return m, textinput.Blink
			}
			cur, _, ok := m.currentTask()
			if !ok {
				return m, nil
			}
			if taskDepth(m.tasks, cur.ID) >= task.MaxNestDepth {
				m.saveErr = fmt.Errorf("nesting depth limit (%d) reached", task.MaxNestDepth)
				return m, nil
			}
			m.input = newTitleInput(inputW)
			m.inputErr = nil
			m.mode = ModeNewSubtask
			return m, textinput.Blink
		}
		return m, nil
	}
	return m, nil
}

// openCurrentFile は現在のファイルカーソルが指すファイルを外部エディタで開く tea.Cmd を返す。
func (m Model) openCurrentFile() (Model, tea.Cmd) {
	t, _, ok := m.currentTask()
	if !ok {
		return m, nil
	}
	taskDir := storage.TaskDir(m.yamlDir, m.cfg.DataBaseDirectory, t.ID)
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

// isSettingMode は現在のモードが設定画面 (左/右ペイン or 設定画面内のサブモード) かを返す。
// View 切替の判定に使う。
func isSettingMode(m Mode) bool {
	switch m {
	case ModeSetting, ModeSettingStatus, ModeSettingStatusRename, ModeSettingStatusAdd, ModeSettingStatusColor:
		return true
	}
	return false
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

	listFocused := m.mode == ModeList || m.mode == ModeQuitConfirm || m.mode == ModeMove || m.mode == ModePrefix
	detailFocused := m.mode == ModeDetail || m.mode == ModeEditTitle || m.mode == ModeEditStatus

	inMoveMode := m.mode == ModeMove
	listH := bodyH
	if m.viewTrash {
		// ゴミ箱ビューでは最上部 1 行をヘッダで占有するので、リスト本体の高さを 1 減らす。
		listH = bodyH - 1
		if listH < 1 {
			listH = 1
		}
	}

	var left, right string
	if isSettingMode(m.mode) {
		menuFocused := m.mode == ModeSetting
		left, right = renderSetting(m.statuses, m.settingMenuCursor, m.settingStatusCursor, menuFocused, leftW, rightW, bodyH)
	} else {
		left = renderList(m.tasks, m.statuses, m.rows, m.collapsed, m.cursor, listFocused, inMoveMode, leftW, listH)
		if inMoveMode {
			banner := styleMoveBanner.Render("-- MOVE MODE --")
			bannerW := lipgloss.Width(banner)
			x := leftW - bannerW
			if x < 0 {
				x = 0
			}
			left = PlaceOverlay(x, 0, banner, left)
		}
		if m.viewTrash {
			// 左ペインの最上部に「-- TRASH BOX --」ヘッダ行 (黒抜き赤背景) を 1 行追加。
			header := styleTrashHeader.Width(leftW).Render("-- TRASH BOX --")
			left = lipgloss.JoinVertical(lipgloss.Left, header, left)
		}

		var current *task.Task
		if t, _, ok := m.currentTask(); ok {
			current = &t
		}
		right = renderDetail(current, m.statuses, m.files, detailFocused, m.detailCursor, m.fileCursor, rightW, bodyH)
	}

	divider := strings.Repeat("│\n", bodyH)
	divider = styleDivider.Render(strings.TrimRight(divider, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	footer := renderFooter(m.mode, m.prevMode, m.detailCursor, m.viewTrash, m.width)

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	switch m.mode {
	case ModeNewTask, ModeEditTitle:
		view = overlayInputPopup(view, "Title:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeNewSubtask:
		view = overlayInputPopup(view, "Subtask:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeAddFile:
		view = overlayInputPopup(view, "Filename:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeRenameFile:
		view = overlayInputPopup(view, "Rename:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeEditStatus:
		view = overlayStatusPicker(view, m.statuses.Sorted(), m.statusPickerCursor, m.width, m.height-1)
	case ModeSettingStatusRename:
		view = overlayInputPopup(view, "Rename status:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingStatusAdd:
		view = overlayInputPopup(view, "Add status:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingStatusColor:
		view = overlayColorPicker(view, m.settingColorChoices, m.settingColorCursor, m.width, m.height-1)
	case ModeQuitConfirm:
		view = overlayConfirmPopup(view, "Quit?", "are you sure?",
			[]hintItem{{"y", "quit"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeDeleteFileConfirm:
		msg := "delete file?"
		if m.fileCursor < len(m.files) {
			msg = "delete \"" + m.files[m.fileCursor] + "\" ?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeTrashConfirm:
		msg := "move task to trash?"
		if t, _, ok := m.currentTask(); ok {
			msg = "move \"" + t.Title + "\" to trash?"
		}
		view = overlayConfirmPopup(view, "Trash?", msg,
			[]hintItem{{"y", "trash"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeDeleteTaskConfirm:
		msg := "delete task permanently?"
		if t, _, ok := m.currentTask(); ok {
			msg = "permanently delete \"" + t.Title + "\" and its subtasks?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
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
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"Enter", "save"}, {"Esc", "discard"},
	}), innerW)

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
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "discard"},
	}), innerW)

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

// overlayConfirmPopup は y/n 確認用の中央オーバーレイを描画する。
// レイアウトは入力ポップアップと同じ (上下罫線 + 中央 1 行) で、
// 中央にメッセージ、下罫線に hints (キー太字) を埋め込む。
func overlayConfirmPopup(bg, label, message string, hints []hintItem, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render(label), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints(hints), innerW)

	if w := ansi.StringWidth(message); w > contentW {
		message = ansi.Truncate(message, contentW, "")
	}
	body := stylePopupFill.Foreground(colorText).Render(message)
	padded := stylePopupFill.Width(contentW).Render(body)
	msgRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		padded +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	popup := lipgloss.JoinVertical(lipgloss.Left, topRow, msgRow, bottomRow)
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
