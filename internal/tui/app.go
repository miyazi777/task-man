package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)


// editorFinishedMsg は外部エディタが終了したときに自身に通知する内部メッセージ。
type editorFinishedMsg struct {
	err error
}

type Model struct {
	repo          storage.Repository
	tasks         []task.Task
	statuses      task.StatusList
	fields        task.FieldDefList // 拡張項目スキーマ (top-level)
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

	detailRows         []detailRow // 詳細画面の論理行リスト (Title/Status/field×N/Files)
	detailCursor       int         // detailRows のインデックス
	statusPickerCursor int         // sorted statuses のインデックス
	files              []string    // 現タスクのディレクトリ内ファイル一覧
	fileCursor         int         // files のインデックス

	// 設定画面 (ModeSetting / ModeSettingStatus* / ModeSettingField*) 用の状態
	settingMenuCursor   int              // 左メニュー (status / field) のインデックス
	settingStatusCursor int              // 右ペイン: m.statuses.Sorted() のインデックス
	settingColorChoices [][]string       // 色ピッカー候補グリッド (#rrggbb)。grid[row][col]
	settingColorRow     int              // 色ピッカー上の行カーソル (色相)
	settingColorCol     int              // 色ピッカー上の列カーソル (明度)
	settingMovingStatus int              // ModeSettingStatusMove 中の対象 status ID (0 なら未選択)
	settingMoveSnapshot task.StatusList  // ModeSettingStatusMove 開始時のスナップショット (esc 用)

	// ModeSettingField* 用の状態
	settingFieldCursor       int               // 中央ペイン: m.fields.Sorted() のインデックス
	settingFieldAttrCursor   int               // 右ペイン: 0=name, 1=type
	settingFieldMoving       int               // ModeSettingFieldMove 中の対象 field ID
	settingFieldMoveSnapshot task.FieldDefList // ModeSettingFieldMove 開始時のスナップショット (esc 用)

	// ModeSettingFieldAdd 入力モーダル用の状態
	addFieldFocus int            // 0=name 行 (textinput), 1=type 行 (selector)
	addFieldType  task.FieldType // 現在選択中の type

	// ModeEditFieldValue / ModeEditFieldDateValue 用の状態
	editingFieldID int       // 編集中の FieldDef.ID
	calendarCursor time.Time // ModeEditFieldDateValue のカーソル日付

	width  int
	height int

	saveErr error
}

func NewModel(repo storage.Repository, initial []task.Task, statuses task.StatusList, fields task.FieldDefList, yamlDir string, cfg storage.AppConfig) Model {
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
		fields:        fields,
		yamlDir:       yamlDir,
		cfg:           cfg,
		collapsed:     collapsed,
		taskCollapsed: taskCollapsed,
		mode:          ModeList,
		keys:          newKeyMap(),
	}
	m = m.withRowsRebuilt()
	m = m.withDetailRowsRebuilt()
	if first := firstNavigable(m.rows); first >= 0 {
		m.cursor = first
	}
	return m.withFilesRefreshed()
}

// withDetailRowsRebuilt は m.fields から詳細画面の論理行リストを再構築する。
// detailCursor が範囲外になった場合は末尾に寄せる。
func (m Model) withDetailRowsRebuilt() Model {
	m.detailRows = buildDetailRows(m.fields)
	if m.detailCursor < 0 {
		m.detailCursor = 0
	}
	if m.detailCursor >= len(m.detailRows) {
		m.detailCursor = len(m.detailRows) - 1
	}
	return m
}

// currentDetailRow はカーソルが指す詳細行を返す。範囲外なら ok=false。
func (m Model) currentDetailRow() (detailRow, bool) {
	if m.detailCursor < 0 || m.detailCursor >= len(m.detailRows) {
		return detailRow{}, false
	}
	return m.detailRows[m.detailCursor], true
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
	return m.repo.Save(storage.LoadResult{
		Tasks:    m.tasks,
		Statuses: m.statuses,
		Fields:   m.fields,
		Config:   m.cfg,
	})
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
		m.mode == ModeSettingStatusRename || m.mode == ModeSettingStatusAdd ||
		m.mode == ModeSettingFieldRename || m.mode == ModeEditFieldValue ||
		(m.mode == ModeSettingFieldAdd && m.addFieldFocus == 0) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 画面下に表示中のエラーメッセージは esc で消せる。
	// 各モード固有の esc 動作より先に処理する (エラー表示中は最初の esc が「エラー消去」、
	// 次の esc から通常のモード戻り)。
	if m.saveErr != nil && msg.String() == "esc" {
		m.saveErr = nil
		return m, nil
	}
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
			// 削除位置に合わせてカーソルを調整。0 件になった場合は Files 行から
			// 1 つ前の論理行 (= 末尾の field 行 or Status) に戻す。
			if m.fileCursor >= len(m.files) {
				if len(m.files) == 0 {
					m.fileCursor = 0
					if m.detailCursor > 0 {
						m.detailCursor--
					}
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
		// 編集後の戻り先は m.prevMode に応じて切り替える (ModeDetail / ModeList のいずれか)。
		// 期待外の値が入っている場合は安全のため ModeDetail にフォールバック。
		ret := editReturnMode(m.prevMode)
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
				m.mode = ret
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
			m.mode = ret
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			return m, nil
		case "esc":
			m.mode = ret
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
		ret := editReturnMode(m.prevMode)
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
				m.mode = ret
				return m, nil
			}
			_, taskIdx, ok := m.currentTask()
			if !ok {
				m.mode = ret
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
			m.mode = ret
			return m, nil
		case "esc":
			m.mode = ret
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
			row, ok := m.currentDetailRow()
			if !ok {
				return m, nil
			}
			if row.kind == detailRowFiles {
				if m.fileCursor > 0 {
					m.fileCursor--
				} else if m.detailCursor > 0 {
					m.detailCursor--
				}
			} else if m.detailCursor > 0 {
				m.detailCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			row, ok := m.currentDetailRow()
			if !ok {
				return m, nil
			}
			if row.kind == detailRowFiles {
				if m.fileCursor < len(m.files)-1 {
					m.fileCursor++
				}
				return m, nil
			}
			nextIdx := m.detailCursor + 1
			if nextIdx >= len(m.detailRows) {
				return m, nil
			}
			nextRow := m.detailRows[nextIdx]
			if nextRow.kind == detailRowFiles && len(m.files) == 0 {
				// Files 行はファイル 0 件のときカーソルを置かない (既存 UX)。
				return m, nil
			}
			m.detailCursor = nextIdx
			if nextRow.kind == detailRowFiles {
				m.fileCursor = 0
			}
			return m, nil
		case key.Matches(msg, m.keys.Confirm):
			t, _, _ := m.currentTask()
			row, ok := m.currentDetailRow()
			if !ok {
				return m, nil
			}
			switch row.kind {
			case detailRowTitle:
				inputW := popupWidth(m.width) - 7
				if inputW < 1 {
					inputW = 1
				}
				m.input = newTitleInput(inputW)
				m.input.SetValue(t.Title)
				m.input.CursorEnd()
				m.inputErr = task.ValidateTitleChars(m.input.Value())
				m.prevMode = ModeDetail
				m.mode = ModeEditTitle
				return m, textinput.Blink
			case detailRowStatus:
				m.statusPickerCursor = sortedStatusIndex(m.statuses, t.StatusID)
				m.prevMode = ModeDetail
				m.mode = ModeEditStatus
				return m, nil
			case detailRowField:
				def, ok := m.fields.ByID(row.fieldID)
				if !ok {
					return m, nil
				}
				m.editingFieldID = def.ID
				switch def.Type {
				case task.FieldTypeDate:
					// date 型: カレンダーモーダル。既存値があればその日付、無ければ今日。
					var existing string
					if tf, ok := t.Fields.ByFieldID(def.ID); ok {
						existing = tf.Value
					}
					m.calendarCursor = parseFieldDateOrToday(existing)
					m.mode = ModeEditFieldDateValue
					return m, nil
				default:
					// text / url 型: 入力ポップアップ。ラベルは <field name>:。
					// url 型でブラウザを開くのは o キーに割り当てており、enter は常に編集。
					var existing string
					if tf, ok := t.Fields.ByFieldID(def.ID); ok {
						existing = tf.Value
					}
					return m.openFieldEditPopup(def, existing)
				}
			case detailRowFiles:
				if len(m.files) == 0 {
					return m, nil
				}
				return m.openCurrentFile()
			}
			return m, nil
		case key.Matches(msg, m.keys.AddFile):
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles {
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
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles || len(m.files) == 0 {
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
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles || len(m.files) == 0 {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeDeleteFileConfirm
			return m, nil
		case msg.String() == "o":
			// o: url 型項目で値を OS のデフォルトブラウザで開く。
			// enter は編集ポップアップに統一しているため、開く操作は別キーに分けている。
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowField {
				return m, nil
			}
			def, ok := m.fields.ByID(row.fieldID)
			if !ok || def.Type != task.FieldTypeURL {
				return m, nil
			}
			t, _, ok := m.currentTask()
			if !ok {
				return m, nil
			}
			tf, ok := t.Fields.ByFieldID(def.ID)
			if !ok || tf.Value == "" {
				return m, nil
			}
			if err := task.ValidateFieldURLValue(tf.Value); err != nil {
				m.saveErr = err
				return m, nil
			}
			if err := openURLInBrowser(tf.Value); err != nil {
				m.saveErr = err
			}
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
		case key.Matches(msg, m.keys.Enter):
			// enter: 確定 → 永続化して ModeList へ
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

	case ModeOperation:
		// タスクリスト上で o を押した直後の operation 入力待ち状態。
		// t = title 編集 / s = status 編集 / esc = キャンセル。
		t, _, ok := m.currentTask()
		if !ok {
			m.mode = ModeList
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeList
			return m, nil
		}
		switch msg.String() {
		case "t":
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.input.SetValue(t.Title)
			m.input.CursorEnd()
			m.inputErr = task.ValidateTitleChars(m.input.Value())
			m.prevMode = ModeList
			m.mode = ModeEditTitle
			return m, textinput.Blink
		case "s":
			m.statusPickerCursor = sortedStatusIndex(m.statuses, t.StatusID)
			m.prevMode = ModeList
			m.mode = ModeEditStatus
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
			switch m.settingMenuCursor {
			case settingMenuStatus:
				m.mode = ModeSettingStatus
				if m.settingStatusCursor >= len(m.statuses) {
					m.settingStatusCursor = 0
				}
			case settingMenuField:
				m.mode = ModeSettingField
				if m.settingFieldCursor >= len(m.fields) {
					m.settingFieldCursor = 0
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
			m.settingColorChoices = statusColorChoices()
			m.settingColorRow, m.settingColorCol = nearestColorChoiceCell(m.settingColorChoices, cur.Color)
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
		case key.Matches(msg, m.keys.Move):
			// m: ステータス位置変更モードへ。スナップショットを取得して開始。
			if len(sorted) == 0 {
				return m, nil
			}
			cur := sorted[m.settingStatusCursor]
			snapshot := make(task.StatusList, len(m.statuses))
			copy(snapshot, m.statuses)
			m.settingMoveSnapshot = snapshot
			m.settingMovingStatus = cur.ID
			m.mode = ModeSettingStatusMove
			return m, nil
		case key.Matches(msg, m.keys.DeleteTask):
			// d: ステータス削除確認モーダルへ遷移。
			// 1 つしか無い場合は遷移せずエラー表示のみ。
			if len(sorted) == 0 {
				return m, nil
			}
			if len(m.statuses) <= 1 {
				m.saveErr = task.ErrCannotDeleteLastStatus
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeSettingStatusDeleteConfirm
			return m, nil
		}
		return m, nil

	case ModeSettingStatusDeleteConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			sorted := m.statuses.Sorted()
			if m.settingStatusCursor >= len(sorted) {
				m.mode = ModeSettingStatus
				return m, nil
			}
			cur := sorted[m.settingStatusCursor]
			newStatuses, fallbackID, err := m.statuses.DeleteByID(cur.ID)
			if err != nil {
				m.saveErr = err
				m.mode = ModeSettingStatus
				return m, nil
			}
			m.tasks = task.ReassignTasksToFallback(m.tasks, cur.ID, fallbackID)
			m.statuses = newStatuses
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withRowsRebuilt()
			sortedAfter := m.statuses.Sorted()
			if m.settingStatusCursor >= len(sortedAfter) {
				m.settingStatusCursor = len(sortedAfter) - 1
			}
			if m.settingStatusCursor < 0 {
				m.settingStatusCursor = 0
			}
			m.mode = ModeSettingStatus
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeSettingStatus
			return m, nil
		}
		return m, nil

	case ModeSettingStatusMove:
		switch {
		case key.Matches(msg, m.keys.Back):
			// esc: スナップショットから復元してキャンセル
			if m.settingMoveSnapshot != nil {
				m.statuses = m.settingMoveSnapshot
			}
			movedID := m.settingMovingStatus
			m.settingMoveSnapshot = nil
			m.settingMovingStatus = 0
			m.mode = ModeSettingStatus
			m = m.withRowsRebuilt()
			// カーソルを移動対象の現位置に合わせる
			sorted := m.statuses.Sorted()
			for i, s := range sorted {
				if s.ID == movedID {
					m.settingStatusCursor = i
					break
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// enter: 確定 → 永続化して ModeSettingStatus へ
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.settingMoveSnapshot = nil
			m.settingMovingStatus = 0
			m.mode = ModeSettingStatus
			m = m.withRowsRebuilt()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.statuses = m.statuses.MoveStatusUp(m.settingMovingStatus)
			// カーソルを移動対象の新位置に追従
			sorted := m.statuses.Sorted()
			for i, s := range sorted {
				if s.ID == m.settingMovingStatus {
					m.settingStatusCursor = i
					break
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.statuses = m.statuses.MoveStatusDown(m.settingMovingStatus)
			sorted := m.statuses.Sorted()
			for i, s := range sorted {
				if s.ID == m.settingMovingStatus {
					m.settingStatusCursor = i
					break
				}
			}
			return m, nil
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
			m = m.withRowsRebuilt()
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
			// 新規ステータスのデフォルト色: グリッド先頭セル (パレット先頭 = purple のベース)。
			defaultColor := statusColorChoices()[0][0]
			inserted, newID, err := m.statuses.InsertAt(insertIdx, label, defaultColor)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.statuses = inserted
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withRowsRebuilt()
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
		grid := m.settingColorChoices
		rows, cols := len(grid), 0
		if rows > 0 {
			cols = len(grid[0])
		}
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.settingColorRow > 0 {
				m.settingColorRow--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingColorRow < rows-1 {
				m.settingColorRow++
			}
			return m, nil
		case key.Matches(msg, m.keys.Close):
			// h/← で 1 列左へ。先頭列なら no-op (esc とは別役割)。
			if m.settingColorCol > 0 {
				m.settingColorCol--
			}
			return m, nil
		case key.Matches(msg, m.keys.Open):
			// l/→ で 1 列右へ。
			if m.settingColorCol < cols-1 {
				m.settingColorCol++
			}
			return m, nil
		}
		switch msg.String() {
		case "enter":
			sorted := m.statuses.Sorted()
			if m.settingStatusCursor >= len(sorted) ||
				m.settingColorRow >= rows || m.settingColorCol >= cols {
				m.mode = ModeSettingStatus
				return m, nil
			}
			id := sorted[m.settingStatusCursor].ID
			newColor := grid[m.settingColorRow][m.settingColorCol]
			updated, err := m.statuses.SetColorByID(id, newColor)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.statuses = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withRowsRebuilt()
			m.mode = ModeSettingStatus
			return m, nil
		case "esc":
			m.mode = ModeSettingStatus
			return m, nil
		}
		return m, nil

	case ModeSettingField:
		sorted := m.fields.Sorted()
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// esc: 左メニューへフォーカスを戻す。
			m.mode = ModeSetting
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFieldCursor > 0 {
				m.settingFieldCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFieldCursor < len(sorted)-1 {
				m.settingFieldCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// enter: 右ペイン (attributes) へ
			if len(sorted) == 0 {
				return m, nil
			}
			m.settingFieldAttrCursor = 0
			m.mode = ModeSettingFieldAttribute
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			// a: 新規 field 追加モーダル
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.inputErr = nil
			m.addFieldFocus = 0
			m.addFieldType = task.FieldTypeText
			m.mode = ModeSettingFieldAdd
			return m, textinput.Blink
		case key.Matches(msg, m.keys.RenameFile):
			// r: name 変更
			if len(sorted) == 0 {
				return m, nil
			}
			cur := sorted[m.settingFieldCursor]
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.input.SetValue(cur.Name)
			m.input.CursorEnd()
			m.inputErr = task.ValidateFieldNameChars(m.input.Value())
			m.mode = ModeSettingFieldRename
			return m, textinput.Blink
		case key.Matches(msg, m.keys.Move):
			// m: 位置変更モード
			if len(sorted) == 0 {
				return m, nil
			}
			cur := sorted[m.settingFieldCursor]
			snapshot := make(task.FieldDefList, len(m.fields))
			copy(snapshot, m.fields)
			m.settingFieldMoveSnapshot = snapshot
			m.settingFieldMoving = cur.ID
			m.mode = ModeSettingFieldMove
			return m, nil
		case key.Matches(msg, m.keys.DeleteTask):
			// d: 削除確認モーダルへ
			if len(sorted) == 0 {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeSettingFieldDeleteConfirm
			return m, nil
		}
		return m, nil

	case ModeSettingFieldAttribute:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// esc: 中央ペインへ
			m.mode = ModeSettingField
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFieldAttrCursor > 0 {
				m.settingFieldAttrCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFieldAttrCursor < 1 {
				m.settingFieldAttrCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// enter: name 行のみ rename へ。type 行は read-only。
			if m.settingFieldAttrCursor != 0 {
				return m, nil
			}
			sorted := m.fields.Sorted()
			if m.settingFieldCursor >= len(sorted) {
				return m, nil
			}
			cur := sorted[m.settingFieldCursor]
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newTitleInput(inputW)
			m.input.SetValue(cur.Name)
			m.input.CursorEnd()
			m.inputErr = task.ValidateFieldNameChars(m.input.Value())
			m.mode = ModeSettingFieldRename
			return m, textinput.Blink
		}
		return m, nil

	case ModeSettingFieldAdd:
		// 2 行モーダル: focus=0 → name (textinput), focus=1 → type (selector)
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" || m.inputErr != nil {
				return m, nil
			}
			updated, newID, err := m.fields.AddDef(name, m.addFieldType)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.fields = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withDetailRowsRebuilt()
			// 新規 field にカーソルを合わせる
			sorted := m.fields.Sorted()
			for i, f := range sorted {
				if f.ID == newID {
					m.settingFieldCursor = i
					break
				}
			}
			m.mode = ModeSettingField
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "esc":
			m.mode = ModeSettingField
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "tab":
			m.addFieldFocus = (m.addFieldFocus + 1) % 2
			return m, nil
		}
		// focus=0 (name) のときの上下キー: type 行へ移動。それ以外は textinput に委譲。
		// focus=1 (type) のときの上下キー: name 行へ。h/l で type を循環。
		if m.addFieldFocus == 0 {
			switch {
			case key.Matches(msg, m.keys.Down):
				m.addFieldFocus = 1
				return m, nil
			case key.Matches(msg, m.keys.Up):
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.inputErr = task.ValidateFieldNameChars(m.input.Value())
			return m, cmd
		}
		// focus=1 (type 縦並び)
		switch {
		case key.Matches(msg, m.keys.Up):
			// k/↑ : 先頭選択肢で押されたら name にフォーカスを戻す。それ以外は前の type へ。
			if isFirstFieldType(m.addFieldType) {
				m.addFieldFocus = 0
			} else {
				m.addFieldType = prevFieldType(m.addFieldType)
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			// j/↓ : 末尾選択肢ではこれ以上動かない。それ以外は次の type へ。
			if !isLastFieldType(m.addFieldType) {
				m.addFieldType = nextFieldType(m.addFieldType)
			}
			return m, nil
		}
		return m, nil

	case ModeSettingFieldRename:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" || m.inputErr != nil {
				return m, nil
			}
			sorted := m.fields.Sorted()
			if m.settingFieldCursor >= len(sorted) {
				m.mode = ModeSettingField
				return m, nil
			}
			id := sorted[m.settingFieldCursor].ID
			updated, err := m.fields.RenameByID(id, name)
			if err != nil {
				m.saveErr = err
				return m, nil
			}
			m.fields = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withDetailRowsRebuilt()
			m.mode = ModeSettingField
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "esc":
			m.mode = ModeSettingField
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateFieldNameChars(m.input.Value())
		return m, cmd

	case ModeSettingFieldMove:
		switch {
		case key.Matches(msg, m.keys.Back):
			// esc: スナップショットから復元
			if m.settingFieldMoveSnapshot != nil {
				m.fields = m.settingFieldMoveSnapshot
			}
			movedID := m.settingFieldMoving
			m.settingFieldMoveSnapshot = nil
			m.settingFieldMoving = 0
			m.mode = ModeSettingField
			m = m.withDetailRowsRebuilt()
			sorted := m.fields.Sorted()
			for i, f := range sorted {
				if f.ID == movedID {
					m.settingFieldCursor = i
					break
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			// enter: 確定 → 永続化
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.settingFieldMoveSnapshot = nil
			m.settingFieldMoving = 0
			m.mode = ModeSettingField
			m = m.withDetailRowsRebuilt()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.fields = m.fields.MoveUp(m.settingFieldMoving)
			sorted := m.fields.Sorted()
			for i, f := range sorted {
				if f.ID == m.settingFieldMoving {
					m.settingFieldCursor = i
					break
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.fields = m.fields.MoveDown(m.settingFieldMoving)
			sorted := m.fields.Sorted()
			for i, f := range sorted {
				if f.ID == m.settingFieldMoving {
					m.settingFieldCursor = i
					break
				}
			}
			return m, nil
		}
		return m, nil

	case ModeSettingFieldDeleteConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			sorted := m.fields.Sorted()
			if m.settingFieldCursor >= len(sorted) {
				m.mode = ModeSettingField
				return m, nil
			}
			cur := sorted[m.settingFieldCursor]
			updated, err := m.fields.DeleteByID(cur.ID)
			if err != nil {
				m.saveErr = err
				m.mode = ModeSettingField
				return m, nil
			}
			m.fields = updated
			// 全タスクから孤児 TaskField を除去
			m.tasks = task.PurgeRemovedFieldValues(m.tasks, m.fields)
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m = m.withDetailRowsRebuilt()
			sortedAfter := m.fields.Sorted()
			if m.settingFieldCursor >= len(sortedAfter) {
				m.settingFieldCursor = len(sortedAfter) - 1
			}
			if m.settingFieldCursor < 0 {
				m.settingFieldCursor = 0
			}
			m.mode = ModeSettingField
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeSettingField
			return m, nil
		}
		return m, nil

	case ModeEditFieldValue:
		def, _ := m.fields.ByID(m.editingFieldID)
		switch msg.String() {
		case "enter":
			value := m.input.Value()
			if m.inputErr != nil {
				return m, nil
			}
			// url 型は保存時に scheme + host を要求する形式チェックを行う。
			if def.Type == task.FieldTypeURL {
				if err := task.ValidateFieldURLValue(value); err != nil {
					m.inputErr = err
					return m, nil
				}
			}
			_, taskIdx, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			fieldID := m.editingFieldID
			t := m.tasks[taskIdx]
			t.Fields = t.Fields.SetValue(fieldID, value)
			if err := t.Fields.Validate(m.fields); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks[taskIdx] = t
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		case "esc":
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = validateFieldValueLiveByType(def.Type, m.input.Value())
		return m, cmd

	case ModeEditFieldDateValue:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeDetail
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			_, taskIdx, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			value := formatFieldDate(m.calendarCursor)
			t := m.tasks[taskIdx]
			t.Fields = t.Fields.SetValue(m.editingFieldID, value)
			if err := t.Fields.Validate(m.fields); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.tasks[taskIdx] = t
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			return m, nil
		case key.Matches(msg, m.keys.Close):
			// h/← : 前日
			m.calendarCursor = m.calendarCursor.AddDate(0, 0, -1)
			return m, nil
		case key.Matches(msg, m.keys.Open):
			// l/→ : 翌日
			m.calendarCursor = m.calendarCursor.AddDate(0, 0, 1)
			return m, nil
		case key.Matches(msg, m.keys.Up):
			// k/↑ : 前週
			m.calendarCursor = m.calendarCursor.AddDate(0, 0, -7)
			return m, nil
		case key.Matches(msg, m.keys.Down):
			// j/↓ : 翌週
			m.calendarCursor = m.calendarCursor.AddDate(0, 0, 7)
			return m, nil
		}
		switch msg.String() {
		case "p":
			// 前月。カーソル日が新月で存在しない場合 (例: 3/31 → 2/28) は末日にクランプ。
			m.calendarCursor = shiftMonth(m.calendarCursor, -1)
			return m, nil
		case "n":
			m.calendarCursor = shiftMonth(m.calendarCursor, 1)
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
				m.detailCursor = 0 // detailRows[0] = Title
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
		case msg.String() == "o":
			// o: operation モードへ遷移 (タスク行のとき)。t: title / s: status を選べる。
			if _, _, ok := m.currentTask(); !ok {
				return m, nil
			}
			if m.viewTrash {
				return m, nil
			}
			m.mode = ModeOperation
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

// openFieldEditPopup は拡張項目 (text/url) の値編集ポップアップを開く。
// type に応じて入力欄の charLimit と live バリデーションを切り替える。
// editingFieldID は呼び出し側が事前にセットしておく前提。
func (m Model) openFieldEditPopup(def task.FieldDef, existing string) (Model, tea.Cmd) {
	inputW := popupWidth(m.width) - 7
	if inputW < 1 {
		inputW = 1
	}
	switch def.Type {
	case task.FieldTypeURL:
		m.input = newFieldURLValueInput(inputW)
	default:
		m.input = newFieldValueInput(inputW)
	}
	if existing != "" {
		m.input.SetValue(existing)
		m.input.CursorEnd()
	}
	m.inputErr = validateFieldValueLiveByType(def.Type, m.input.Value())
	m.mode = ModeEditFieldValue
	return m, textinput.Blink
}

// editReturnMode は ModeEditTitle / ModeEditStatus 終了時に戻るべきモードを返す。
// 呼び出し側が prevMode に設定した「編集を呼び出した側のモード」を尊重しつつ、
// 期待外の値が入っていた場合は安全のため ModeDetail にフォールバックする。
func editReturnMode(prev Mode) Mode {
	switch prev {
	case ModeList, ModeDetail:
		return prev
	default:
		return ModeDetail
	}
}

// validateFieldValueLiveByType はライブ入力検証を type で切り替える。
// 入力途中は形式チェックは行わず、長さ・禁止文字のみを評価する。
func validateFieldValueLiveByType(ft task.FieldType, value string) error {
	switch ft {
	case task.FieldTypeURL:
		return task.ValidateFieldURLValueChars(value)
	default:
		return task.ValidateFieldTextValueChars(value)
	}
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

// buildPaneDivider は高さ bodyH のペイン縦区切り線を組み立てる。
// junctionRow が [0, bodyH) にあり junctionChar が空でない場合、その行のみ
// "│" を junctionChar に差し替える (T 字接合用)。それ以外は "│" のみ。
func buildPaneDivider(bodyH int, junctionChar string, junctionRow int) string {
	if bodyH <= 0 {
		return ""
	}
	rows := make([]string, bodyH)
	useJunction := junctionChar != "" && junctionRow >= 0 && junctionRow < bodyH
	for i := 0; i < bodyH; i++ {
		if useJunction && i == junctionRow {
			rows[i] = junctionChar
		} else {
			rows[i] = "│"
		}
	}
	return styleDivider.Render(strings.Join(rows, "\n"))
}

// threePaneWidths は通常画面 (list / detail / preview) の各ペイン幅を返す。
// 区切り線 2 本 (1 cell × 2) を差し引いた残りを 1/3 ずつ割り当てる。剰余は preview に寄せる。
// 画面が狭すぎて preview に充分な幅が取れない場合は最低 1 cell を確保しつつ list/mid を圧縮する。
func threePaneWidths(screenW int) (leftW, midW, previewW int) {
	avail := screenW - 2 // 区切り線 2 本ぶん
	if avail < 3 {
		// 極端に狭いケースのフォールバック。各ペイン 1 cell ずつ。
		return 1, 1, 1
	}
	leftW = avail / 3
	midW = avail / 3
	previewW = avail - leftW - midW
	return
}

// isSettingMode は現在のモードが設定画面 (左/中/右ペイン or 設定画面内のサブモード) かを返す。
// View 切替の判定に使う。
func isSettingMode(m Mode) bool {
	switch m {
	case ModeSetting,
		ModeSettingStatus, ModeSettingStatusRename, ModeSettingStatusAdd,
		ModeSettingStatusColor, ModeSettingStatusMove, ModeSettingStatusDeleteConfirm,
		ModeSettingField, ModeSettingFieldAttribute,
		ModeSettingFieldAdd, ModeSettingFieldRename,
		ModeSettingFieldMove, ModeSettingFieldDeleteConfirm:
		return true
	}
	return false
}

// isSettingFieldFocus は設定画面で「field」側を見ている状態かを返す (3 ペインレイアウト用)。
// ModeSetting (メニュー) 中は cursor が field を指しているかで判断する。
func (m Model) isSettingFieldFocus() bool {
	switch m.mode {
	case ModeSettingField, ModeSettingFieldAttribute,
		ModeSettingFieldAdd, ModeSettingFieldRename,
		ModeSettingFieldMove, ModeSettingFieldDeleteConfirm:
		return true
	case ModeSetting:
		return m.settingMenuCursor == settingMenuField
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

	bodyH := m.height - 1
	divider := buildPaneDivider(bodyH, "", -1)

	listFocused := m.mode == ModeList || m.mode == ModeQuitConfirm || m.mode == ModeMove || m.mode == ModePrefix || m.mode == ModeOperation
	detailFocused := m.mode == ModeDetail || m.mode == ModeEditTitle || m.mode == ModeEditStatus

	inMoveMode := m.mode == ModeMove

	var body string
	if isSettingMode(m.mode) {
		menuFocused := m.mode == ModeSetting
		if m.isSettingFieldFocus() {
			// field 系: 3 ペイン (menu 12cell + 中央 + 右 attributes)。
			leftW := 12
			if leftW > m.width-2 {
				leftW = m.width - 2
				if leftW < 1 {
					leftW = 1
				}
			}
			remain := m.width - leftW - 2 // 区切り線 2 本
			if remain < 2 {
				remain = 2
			}
			midW := remain / 2
			rightW := remain - midW
			midFocused := m.mode == ModeSettingField || m.mode == ModeSettingFieldAdd ||
				m.mode == ModeSettingFieldRename || m.mode == ModeSettingFieldMove ||
				m.mode == ModeSettingFieldDeleteConfirm
			rightFocused := m.mode == ModeSettingFieldAttribute
			inFieldMove := m.mode == ModeSettingFieldMove
			left, mid, right := renderSettingField(m.fields, m.settingMenuCursor, m.settingFieldCursor, m.settingFieldAttrCursor,
				menuFocused, midFocused, rightFocused, inFieldMove, leftW, midW, rightW, bodyH)
			if inFieldMove {
				banner := styleMoveBanner.Render("-- MOVE MODE --")
				bannerW := lipgloss.Width(banner)
				x := midW - bannerW
				if x < 0 {
					x = 0
				}
				mid = PlaceOverlay(x, 0, banner, mid)
			}
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, mid, divider, right)
		} else {
			// status 系: 左メニュー + 右詳細の 2 ペイン。
			// 左メニュー幅は field 系と揃えて 12 cell 固定にする。
			leftW := 12
			if leftW > m.width-2 {
				leftW = m.width - 2
				if leftW < 1 {
					leftW = 1
				}
			}
			rightW := m.width - leftW - 1
			if rightW < 1 {
				rightW = 1
			}
			inSettingMove := m.mode == ModeSettingStatusMove
			left, right := renderSettingStatus(m.statuses, m.settingMenuCursor, m.settingStatusCursor, menuFocused, inSettingMove, leftW, rightW, bodyH)
			if inSettingMove {
				banner := styleMoveBanner.Render("-- MOVE MODE --")
				bannerW := lipgloss.Width(banner)
				x := rightW - bannerW
				if x < 0 {
					x = 0
				}
				right = PlaceOverlay(x, 0, banner, right)
			}
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
		}
	} else {
		// 通常画面: タスクリスト 1/3 + 詳細 1/3 + プレビュー 1/3 (区切り線 2 本)。
		leftW, midW, previewW := threePaneWidths(m.width)

		listH := bodyH
		if m.viewTrash {
			// ゴミ箱ビューでは最上部 1 行をヘッダで占有するので、リスト本体の高さを 1 減らす。
			listH = bodyH - 1
			if listH < 1 {
				listH = 1
			}
		}

		left := renderList(m.tasks, m.statuses, m.rows, m.collapsed, m.cursor, listFocused, inMoveMode, leftW, listH)
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
		mid := renderDetail(current, m.statuses, m.fields, m.files, m.detailRows, detailFocused, m.detailCursor, m.fileCursor, midW, bodyH)

		// プレビュー: 現在タスクの fileCursor が指すファイルを対象にする。
		// 現在タスク無し / ファイル無し / カーソル範囲外 のいずれでも空ペイン。
		var previewFile string
		var previewTaskID int
		if current != nil && len(m.files) > 0 && m.fileCursor >= 0 && m.fileCursor < len(m.files) {
			previewFile = m.files[m.fileCursor]
			previewTaskID = current.ID
		}
		right := renderPreview(m.yamlDir, m.cfg.DataBaseDirectory, previewTaskID, previewFile, previewW, bodyH)

		// 詳細ペインの Files: 下の罫線と視覚的につなげるため、左右のペイン縦区切り線に
		// T 字接合 (├ / ┤) を入れる。タスクが選択されておらず Files: ブロックが描画
		// されないときは通常の │ のままにする。
		// 罫線の行位置は detailRows に含まれる field 数によって変動するため
		// detailFilesDividerRow(rows) で動的に算出する。
		junctionRow := -1
		if current != nil {
			junctionRow = detailFilesDividerRow(m.detailRows)
		}
		leftDivider := buildPaneDivider(bodyH, "├", junctionRow)
		rightDivider := buildPaneDivider(bodyH, "┤", junctionRow)

		body = lipgloss.JoinHorizontal(lipgloss.Top, left, leftDivider, mid, rightDivider, right)
	}

	onFilesRow := false
	onURLRow := false
	if row, ok := m.currentDetailRow(); ok {
		switch row.kind {
		case detailRowFiles:
			onFilesRow = true
		case detailRowField:
			if def, ok := m.fields.ByID(row.fieldID); ok && def.Type == task.FieldTypeURL {
				onURLRow = true
			}
		}
	}
	footer := renderFooter(m.mode, m.prevMode, onFilesRow, onURLRow, m.viewTrash, m.width)

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
		view = overlayColorPicker(view, m.settingColorChoices, m.settingColorRow, m.settingColorCol, m.width, m.height-1)
	case ModeSettingFieldAdd:
		// name 行にフォーカスが無いときは textinput の prompt "> " を非表示にする。
		// 横位置を維持するため空白 2 cell に置換 (View 用ローカルコピーのみ変更)。
		if m.addFieldFocus != 0 {
			m.input.Prompt = "  "
		} else {
			m.input.Prompt = "> "
		}
		view = overlayFieldAddPopup(view, m.input.View(), m.inputErr, m.addFieldFocus, m.addFieldType, task.AllFieldTypes, m.width, m.height-1)
	case ModeSettingFieldRename:
		view = overlayInputPopup(view, "Rename field:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeEditFieldValue:
		label := "Value:"
		if def, ok := m.fields.ByID(m.editingFieldID); ok {
			label = def.Name + ":"
		}
		view = overlayInputPopup(view, label, m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeEditFieldDateValue:
		view = overlayCalendarPopup(view, m.calendarCursor, m.width, m.height-1)
	case ModeSettingFieldDeleteConfirm:
		msg := "delete field?"
		if sorted := m.fields.Sorted(); m.settingFieldCursor < len(sorted) {
			msg = "delete field \"" + sorted[m.settingFieldCursor].Name + "\" ?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeSettingStatusDeleteConfirm:
		msg := "delete status?"
		if sorted := m.statuses.Sorted(); m.settingStatusCursor < len(sorted) {
			msg = "delete status \"" + sorted[m.settingStatusCursor].Label + "\" ?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
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
		view = overlayErrorPopup(view, m.saveErr.Error(), m.width, m.height-1)
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

// overlayErrorPopup はエラーメッセージ用の中央オーバーレイを描画する。
// 上罫線にラベル "Error"、本文 1 行、下罫線に "esc:dismiss" ヒント。
// 本文は danger 色 (赤) で強調する。
func overlayErrorPopup(bg, message string, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupError.Render("Error"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"esc", "dismiss"},
	}), innerW)

	if w := ansi.StringWidth(message); w > contentW {
		message = ansi.Truncate(message, contentW, "")
	}
	body := stylePopupFill.Foreground(colorDanger).Render(message)
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
