// Package tui は Bubble Tea ベースの TUI 層。Model / Update / View を提供し、
// Mode 列挙で画面・入力フェーズを切り替える。
package tui

import (
	"errors"
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

// Model は TUI のアプリケーション状態。Bubble Tea の Model 規約 (Init / Update / View) を実装する。
type Model struct {
	repo          storage.Repository
	tasks         []task.Task
	statuses      task.StatusList
	fields        task.FieldDefList // 拡張項目スキーマ (top-level)
	tags          task.TagList      // タグ集合 (top-level)
	yamlPath      string            // tasks.yaml の絶対パス (general 設定画面で表示)
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

	detailRows         []detailRow     // 詳細画面の論理行リスト (Title/Status/field×N/Files)
	detailCursor       int             // detailRows のインデックス
	statusPickerCursor int             // sorted statuses のインデックス
	files              []fileRow       // 現タスクのディレクトリ内ファイル一覧 (DFS フラット化済)
	fileCursor         int             // files のインデックス
	fileCollapsed      map[string]bool // ディレクトリ relPath -> 折りたたみ中
	filesTaskID        int             // m.files の元になったタスク ID。切り替え検出に使う
	addFileRelDir      string          // ModeAddFile 突入時に決めた作成先の親ディレクトリ (relPath、空文字 = タスク直下)

	// 設定画面 (ModeSetting / ModeSettingStatus* / ModeSettingField*) 用の状態
	settingMenuCursor    int             // 左メニュー (general / status / field) のインデックス
	settingGeneralCursor int             // general ペイン: 編集対象行のインデックス (現状は 0=data_base_directory のみ)
	settingStatusCursor  int             // 右ペイン: m.statuses.Sorted() のインデックス
	settingColorChoices  [][]string      // 色ピッカー候補グリッド (#rrggbb)。grid[row][col]
	settingColorRow      int             // 色ピッカー上の行カーソル (色相)
	settingColorCol      int             // 色ピッカー上の列カーソル (明度)
	settingMovingStatus  int             // ModeSettingStatusMove 中の対象 status ID (0 なら未選択)
	settingMoveSnapshot  task.StatusList // ModeSettingStatusMove 開始時のスナップショット (esc 用)

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

	// ModeTagPicker 用の状態
	// tagPickerTaskID = 対象タスク ID (モーダルが開いている間にカーソルが動いても固定)
	// tagPickerCursor = 0 が create input、1..N が既存タグ (Sorted 順) のインデックス
	tagPickerTaskID int
	tagPickerCursor int

	// ModeTagColorPicker 用の状態 (背後の ModeTagPicker は維持される)
	tagColorPickerTagID int

	// ModeTagPickerRename 用の状態。リネーム中は m.input を流用し、
	// 背後の検索フィルタ値を復元できるよう退避する。
	tagPickerRenameTagID int
	tagPickerSearchSaved string

	// ModeLayout 用の状態。
	// layout は編集中の比率セット (突入時に ensureLayoutRatios で全 nil を埋める)。
	// layoutBackup は esc で巻き戻すためのスナップショット (突入直前の cfg.Layout の値)。
	// layoutFocus は task_list / task_detail / file_list の 3 値。
	layout       storage.LayoutConfig
	layoutBackup storage.LayoutConfig
	layoutFocus  layoutFocus

	// ModeFileOpener 用の状態 (file_opener モーダル)。
	// 候補が 2 件以上の時にモーダルを開いて選ばせる。1 件以下は openCurrentFile 内で即起動。
	// モーダル表示中はタスクリスト/ファイルカーソルが動かない前提なので、対象ファイルを文字列で固定保持する。
	fileOpenerCandidates []storage.Application
	fileOpenerCursor     int
	fileOpenerTaskID     int
	fileOpenerFile       string

	// ModeSettingApplication 系の状態。
	settingApplicationCursor     int                   // 中央ペイン: applications 内のインデックス
	settingApplicationAttrCursor int                   // 右ペイン: 0=id, 1=name, 2=run
	settingApplicationMovingID   int                   // 移動中の application ID (0=未選択)
	settingApplicationMoveBackup []storage.Application // esc で復元する移動前スナップショット
	addApplicationFocus          int                   // 追加モーダル: 0=name, 1=run
	addApplicationNameBuf        string                // 追加モーダルで focus 切替時に name を退避するバッファ
	addApplicationRunBuf         string                // 追加モーダルで focus 切替時に run を退避するバッファ

	// ModeSettingFileOpener 系の状態。
	settingFileOpenerCursor     int                  // 中央ペイン: file_openers 内のインデックス
	settingFileOpenerAttrCursor int                  // 右ペイン: 0=extension, 1=applications, 2=default_app
	settingFileOpenerMovingExt  string               // 移動中の opener.Extension
	settingFileOpenerMoveBackup []storage.FileOpener // esc で復元する移動前スナップショット
	// applications multi-select / default_app picker 中の表示インデックス
	settingFileOpenerAppsCursor    int   // multi-select 中のカーソル (apps 配列の index)
	settingFileOpenerAppsSelected  []int // 編集中の applications ID 配列 (作業用バッファ)
	settingFileOpenerDefaultCursor int   // default_app picker 中のカーソル

	width  int
	height int

	saveErr error
}

// NewModel は初期データから Model を組み立てて返す。Bubble Tea program に渡す Model はここで作る。
func NewModel(repo storage.Repository, initial []task.Task, statuses task.StatusList, fields task.FieldDefList, tags task.TagList, yamlPath string, cfg storage.AppConfig) Model {
	yamlDir := filepath.Dir(yamlPath)
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
		tags:          tags,
		yamlPath:      yamlPath,
		yamlDir:       yamlDir,
		cfg:           cfg,
		collapsed:     collapsed,
		taskCollapsed: taskCollapsed,
		mode:          ModeList,
		keys:          newKeyMap(),
		layout:        cfg.Layout,
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
		Tags:     m.tags,
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
//
// タスク ID が前回読み込み時から変わっていた場合は折りたたみ状態をリセットする
// (フォルダ構成が大きく異なるタスク間で持ち越すと混乱の元になるため)。
func (m Model) withFilesRefreshed() Model {
	t, _, ok := m.currentTask()
	if !ok {
		m.files = nil
		m.fileCursor = 0
		m.filesTaskID = 0
		return m
	}
	if t.ID != m.filesTaskID {
		m.fileCollapsed = nil
		m.filesTaskID = t.ID
	}
	tree, _ := storage.ListTaskFileTree(m.yamlDir, m.cfg.DataBaseDirectory, t.ID)
	m.files = flattenFileTree(tree, 0, m.fileCollapsed)
	if m.fileCursor >= len(m.files) {
		m.fileCursor = 0
	}
	return m
}

// flattenFileTree は FileEntry ツリーを DFS でフラット化する。
// collapsed[relPath]==true のディレクトリでは子を出さない (ディレクトリ自身は表示)。
func flattenFileTree(tree []storage.FileEntry, depth int, collapsed map[string]bool) []fileRow {
	var out []fileRow
	for _, e := range tree {
		row := fileRow{
			name:        e.Name,
			relPath:     e.RelPath,
			isDir:       e.IsDir,
			depth:       depth,
			hasChildren: e.IsDir && len(e.Children) > 0,
			collapsed:   e.IsDir && collapsed[e.RelPath],
		}
		out = append(out, row)
		if row.hasChildren && !row.collapsed {
			out = append(out, flattenFileTree(e.Children, depth+1, collapsed)...)
		}
	}
	return out
}

// currentFileRow は m.fileCursor が指す fileRow を返す。範囲外なら ok=false。
func (m Model) currentFileRow() (fileRow, bool) {
	if m.fileCursor < 0 || m.fileCursor >= len(m.files) {
		return fileRow{}, false
	}
	return m.files[m.fileCursor], true
}

// toggleFileCollapse はディレクトリ row の折りたたみ状態を反転して再フラット化する。
// row.hasChildren==false のときは状態を持たないため何もしない。
// カーソルはトグル対象のディレクトリ自身に追従する (折りたたみで子が消えた / 展開で子が現れた場合の安定性のため)。
func (m Model) toggleFileCollapse(row fileRow) Model {
	if !row.isDir || !row.hasChildren {
		return m
	}
	if m.fileCollapsed == nil {
		m.fileCollapsed = map[string]bool{}
	}
	if m.fileCollapsed[row.relPath] {
		delete(m.fileCollapsed, row.relPath)
	} else {
		m.fileCollapsed[row.relPath] = true
	}
	m = m.withFilesRefreshed()
	for i, f := range m.files {
		if f.relPath == row.relPath {
			m.fileCursor = i
			break
		}
	}
	return m
}

// joinRelPath は相対パスを "/" 区切りで結合する。空のセグメントは無視する。
// 結果が空のときは "" を返す (タスクディレクトリ直下を意味する)。
func joinRelPath(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "/")
}

// parentRelPath は relPath の親ディレクトリの相対パスを返す。
// "foo.md" → ""、"sub/foo.md" → "sub"、"" / "." → ""。
func parentRelPath(relPath string) string {
	if relPath == "" || relPath == "." {
		return ""
	}
	idx := strings.LastIndex(relPath, "/")
	if idx < 0 {
		return ""
	}
	return relPath[:idx]
}

// pathForCopy はカーソル位置に応じてクリップボードへ送るファイルシステム上の
// 絶対パスを返す。対象が無い (status 行・空のファイルリスト等) 場合は ok=false。
//
//   - ModeList でタスク行 → そのタスクのデータディレクトリ (<yamlDir>/<dataBaseDir>/task-<id>)
//   - ModeDetail で Files 行 → カーソルが指すファイル / サブディレクトリの絶対パス
//   - ModeDetail のその他の行 → タスクのデータディレクトリ
func (m Model) pathForCopy() (string, bool) {
	t, _, ok := m.currentTask()
	if !ok {
		return "", false
	}
	taskDir := storage.TaskDir(m.yamlDir, m.cfg.DataBaseDirectory, t.ID)
	if m.mode == ModeDetail {
		if row, ok := m.currentDetailRow(); ok && row.kind == detailRowFiles {
			cur, ok := m.currentFileRow()
			if !ok {
				return "", false
			}
			return filepath.Join(taskDir, filepath.FromSlash(cur.relPath)), true
		}
	}
	return taskDir, true
}

// Init は Bubble Tea プログラム開始時に呼ばれる。本 TUI は起動コマンドを持たないので nil を返す。
func (m Model) Init() tea.Cmd {
	return nil
}

// Update は Bubble Tea のメッセージを受け取り、新しい Model と次に発行するコマンドを返す。
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
		(m.mode == ModeSettingFieldAdd && m.addFieldFocus == 0) ||
		(m.mode == ModeTagPicker && m.tagPickerCursor == 0) ||
		m.mode == ModeTagPickerRename {
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
			raw := strings.TrimSpace(m.input.Value())
			if raw == "" || m.inputErr != nil {
				return m, nil
			}
			// 末尾 "/" は「ディレクトリとして作成」のマーカー。
			asDir := strings.HasSuffix(raw, "/")
			name := strings.TrimSuffix(raw, "/")
			if name == "" {
				return m, nil
			}
			t, _, ok := m.currentTask()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			relDir := m.addFileRelDir
			var createErr error
			if asDir {
				createErr = storage.CreateDir(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, relDir, name)
			} else {
				createErr = storage.CreateFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, relDir, name)
			}
			if createErr != nil {
				m.saveErr = createErr
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			// 追加ファイルがある親ディレクトリは折りたたみ解除して、新規エントリを可視化する。
			if relDir != "" && relDir != "." && m.fileCollapsed != nil {
				delete(m.fileCollapsed, relDir)
			}
			m = m.withFilesRefreshed()
			// 追加した entry にカーソルを合わせる
			createdRel := joinRelPath(relDir, name)
			for i, f := range m.files {
				if f.relPath == createdRel {
					m.fileCursor = i
					break
				}
			}
			m.addFileRelDir = ""
			return m, nil
		case "esc":
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			m.addFileRelDir = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		// 末尾 "/" は「ディレクトリ作成」のマーカーとして許容するため、validation 時は剥がす。
		m.inputErr = storage.ValidateFileNameChars(strings.TrimSuffix(m.input.Value(), "/"))
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
			cur, ok := m.currentFileRow()
			if !ok {
				// モーダル中に状況が変わった (本来到達不能)。
				m.mode = ModeDetail
				m.input = textinput.Model{}
				m.inputErr = nil
				return m, nil
			}
			relDir := parentRelPath(cur.relPath)
			if err := storage.RenameFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, relDir, cur.name, name); err != nil {
				m.saveErr = err
				return m, nil
			}
			m.mode = ModeDetail
			m.input = textinput.Model{}
			m.inputErr = nil
			m = m.withFilesRefreshed()
			newRel := joinRelPath(relDir, name)
			for i, f := range m.files {
				if f.relPath == newRel {
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
			// yaml 保存を先に行い、失敗時はディレクトリ削除をスキップする。
			// これにより「yaml にタスクが残っているがディレクトリだけ消えた」半削除状態を防ぐ。
			if err := m.persist(); err != nil {
				m.saveErr = err
				m.mode = ModeList
				m = m.withRowsRebuilt()
				m = clampCursor(m)
				m = m.withFilesRefreshed()
				return m, nil
			}
			var errs []error
			for _, id := range removedIDs {
				if err := storage.DeleteTaskData(m.yamlDir, m.cfg.DataBaseDirectory, id); err != nil {
					errs = append(errs, err)
				}
			}
			if err := errors.Join(errs...); err != nil {
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
			cur, ok := m.currentFileRow()
			if !ok {
				m.mode = ModeDetail
				return m, nil
			}
			var delErr error
			if cur.isDir {
				delErr = storage.DeleteDir(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, cur.relPath)
			} else {
				delErr = storage.DeleteFile(m.yamlDir, m.cfg.DataBaseDirectory, t.ID, cur.relPath)
			}
			if delErr != nil {
				m.saveErr = delErr
				m.mode = ModeDetail
				return m, nil
			}
			// ディレクトリを消した場合、その配下を fileCollapsed に記憶していた key も無効化する。
			if cur.isDir && m.fileCollapsed != nil {
				prefix := cur.relPath + "/"
				delete(m.fileCollapsed, cur.relPath)
				for k := range m.fileCollapsed {
					if strings.HasPrefix(k, prefix) {
						delete(m.fileCollapsed, k)
					}
				}
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
			if err := t.Validate(m.statuses, m.tags); err != nil {
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
			if err := t.Validate(m.statuses, m.tags); err != nil {
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
			if err := updated.Validate(m.statuses, m.tags); err != nil {
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
		case key.Matches(msg, m.keys.Prefix):
			// ;: 詳細画面でも prefix モードに入れる (;→l でレイアウト調整など)。
			m.prevMode = m.mode
			m.mode = ModePrefix
			return m, nil
		case key.Matches(msg, m.keys.Refresh):
			// R: 現在タスクのファイル一覧を再読込する。Files 行以外にカーソルがあっても受け付ける。
			// 外部 (Finder / mv / 他プロセス) で発生したファイル変更を取り込むためのキー。
			m = m.withFilesRefreshed()
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
			case detailRowTags:
				// Tags 行 enter: タグピッカーを開く。閉じたら ModeDetail に戻る。
				m.prevMode = ModeDetail
				return m.openTagPicker(t.ID)
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
				cur, ok := m.currentFileRow()
				if !ok {
					return m, nil
				}
				if cur.isDir {
					// ディレクトリ行で enter は折りたたみトグル。
					return m.toggleFileCollapse(cur), nil
				}
				return m.openCurrentFileWithDefault()
			}
			return m, nil
		case key.Matches(msg, m.keys.Open):
			// Files 行のディレクトリで右キーは展開。葉ファイルでは何もしない。
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles {
				return m, nil
			}
			cur, ok := m.currentFileRow()
			if !ok || !cur.isDir || !cur.hasChildren || !cur.collapsed {
				return m, nil
			}
			return m.toggleFileCollapse(cur), nil
		case key.Matches(msg, m.keys.Close):
			// Files 行のディレクトリで左キーは折りたたみ。葉ファイルでは何もしない。
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles {
				return m, nil
			}
			cur, ok := m.currentFileRow()
			if !ok || !cur.isDir || !cur.hasChildren || cur.collapsed {
				return m, nil
			}
			return m.toggleFileCollapse(cur), nil
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
			// 作成先ディレクトリ: カーソルがディレクトリ行ならその直下、ファイル行ならその親、
			// カーソルが無ければタスク直下。
			m.addFileRelDir = ""
			if cur, ok := m.currentFileRow(); ok {
				if cur.isDir {
					m.addFileRelDir = cur.relPath
				} else {
					m.addFileRelDir = parentRelPath(cur.relPath)
				}
			}
			m.mode = ModeAddFile
			return m, textinput.Blink
		case key.Matches(msg, m.keys.RenameFile):
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles || len(m.files) == 0 {
				return m, nil
			}
			cur, ok := m.currentFileRow()
			if !ok {
				return m, nil
			}
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newFileNameInput(inputW)
			m.input.SetValue(cur.name)
			m.input.CursorEnd()
			m.inputErr = storage.ValidateFileNameChars(m.input.Value())
			m.mode = ModeRenameFile
			return m, textinput.Blink
		case key.Matches(msg, m.keys.DeleteFile):
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles || len(m.files) == 0 {
				return m, nil
			}
			if _, ok := m.currentFileRow(); !ok {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeDeleteFileConfirm
			return m, nil
		case key.Matches(msg, m.keys.CopyPath):
			// p: カーソル位置に応じた絶対パスをクリップボードへ。
			//   Files 行: 当該ファイル / サブディレクトリの絶対パス
			//   それ以外: タスクのデータディレクトリ
			path, ok := m.pathForCopy()
			if !ok {
				return m, nil
			}
			if err := copyToClipboard(path); err != nil {
				m.saveErr = err
			}
			return m, nil
		case msg.String() == "f":
			// f: Files 行カーソル時、OS のデフォルトファイラーで該当ファイル / ディレクトリを開く。
			//   ファイル     : 親フォルダを開き、可能なら選択状態にする (macOS=open -R / windows=/select,)
			//   ディレクトリ : そのフォルダを開く
			row, ok := m.currentDetailRow()
			if !ok || row.kind != detailRowFiles {
				return m, nil
			}
			cur, ok := m.currentFileRow()
			if !ok {
				return m, nil
			}
			t, _, ok := m.currentTask()
			if !ok {
				return m, nil
			}
			taskDir := storage.TaskDir(m.yamlDir, m.cfg.DataBaseDirectory, t.ID)
			target := filepath.Join(taskDir, filepath.FromSlash(cur.relPath))
			if err := openInOSFileManager(target, cur.isDir); err != nil {
				m.saveErr = err
			}
			return m, nil
		case msg.String() == "o":
			// o:
			//   - url 型項目: 値を OS のデフォルトブラウザで開く。
			//   - Files 行: ファイルオープナーを起動 (拡張子に対応する applications を選択)。
			//   - enter は編集ポップアップに統一しているため、開く操作は別キーに分けている。
			row, ok := m.currentDetailRow()
			if !ok {
				return m, nil
			}
			switch row.kind {
			case detailRowFiles:
				if len(m.files) == 0 {
					return m, nil
				}
				if cur, ok := m.currentFileRow(); ok && cur.isDir {
					return m, nil
				}
				return m.openCurrentFileWithPicker()
			case detailRowField:
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
			// s: 設定画面へ遷移。左メニューにフォーカス。最初は general を指す。
			m.mode = ModeSetting
			m.settingMenuCursor = settingMenuGeneral
			m.settingStatusCursor = 0
			return m, nil
		case key.Matches(msg, m.keys.PrefixLayout):
			// l: レイアウト調整モードへ遷移。突入時に未設定値はデフォルト比率で埋め、
			// esc 用に layoutBackup へスナップショットを退避する。
			// フォーカス対象は prevMode (= ; を押した瞬間のモード) と詳細カーソル位置から決定する。
			//   ModeList → task_list
			//   ModeDetail でカーソルが Files 行 → file_list
			//   ModeDetail でカーソルが Files 行以外 → task_detail
			m.layoutBackup = m.layout
			m.layout = ensureLayoutRatios(m.layout)
			m.layoutFocus = layoutFocusTaskList
			if m.prevMode == ModeDetail {
				if row, ok := m.currentDetailRow(); ok && row.kind == detailRowFiles {
					m.layoutFocus = layoutFocusFileList
				} else {
					m.layoutFocus = layoutFocusTaskDetail
				}
			}
			m.mode = ModeLayout
			return m, nil
		}
		return m, nil

	case ModeFileOpener:
		// file_opener モーダル: 候補から選んでアプリ起動。
		switch {
		case key.Matches(msg, m.keys.Back):
			m.mode = m.prevMode
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.fileOpenerCursor > 0 {
				m.fileOpenerCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.fileOpenerCursor < len(m.fileOpenerCandidates)-1 {
				m.fileOpenerCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Confirm):
			if len(m.fileOpenerCandidates) == 0 {
				m.mode = m.prevMode
				return m, nil
			}
			app := m.fileOpenerCandidates[m.fileOpenerCursor]
			taskID := m.fileOpenerTaskID
			fileName := m.fileOpenerFile
			// モーダルを閉じてから外部プロセスを起動する (alt screen を抜けるため)。
			m.mode = m.prevMode
			return m.launchAppForFile(app.Run, taskID, fileName)
		}
		return m, nil

	case ModeLayout:
		// レイアウト調整モード: 比率の編集と確定/キャンセル。フォーカスは突入時に固定。
		horiz, vert := computeLayoutDelta(m.width, m.height-1)
		switch {
		case key.Matches(msg, m.keys.Confirm):
			// enter: 確定。縦 3 値を 1.0 に正規化してから cfg に反映し yaml に書き戻し、
			// レイアウトモード突入前のモードに戻る。
			m.layout = normalizeLayoutRatios(m.layout)
			m.cfg.Layout = m.layout
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = m.prevMode
			return m, nil
		case key.Matches(msg, m.keys.Back):
			// esc: 突入直前の値に戻して終了。yaml には触れない。元のモードへ戻る。
			m.layout = m.layoutBackup
			m.mode = m.prevMode
			return m, nil
		case key.Matches(msg, m.keys.Close):
			// h/←: タスクリスト幅を縮小。フォーカスを問わず横操作は常に有効。
			m.layout = clampLayoutRatios(adjustLayoutHorizontal(m.layout, -horiz))
			return m, nil
		case key.Matches(msg, m.keys.Open):
			// l/→: タスクリスト幅を拡大。
			m.layout = clampLayoutRatios(adjustLayoutHorizontal(m.layout, horiz))
			return m, nil
		case key.Matches(msg, m.keys.Up):
			// k/↑: 対象ペインの高さを縮小。task_list フォーカス時は no-op。
			m.layout = clampLayoutRatios(adjustLayoutVertical(m.layout, m.layoutFocus, -vert))
			return m, nil
		case key.Matches(msg, m.keys.Down):
			// j/↓: 対象ペインの高さを拡大。
			m.layout = clampLayoutRatios(adjustLayoutVertical(m.layout, m.layoutFocus, vert))
			return m, nil
		}
		return m, nil

	case ModeOperation:
		// タスクリスト上で o を押した直後の operation 入力待ち状態。
		// r = rename (title 編集) / s = status 編集 / esc = キャンセル。
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
		case "r":
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
		case "g":
			// g: タグ追加/解除モーダルを開く。
			m.prevMode = ModeList
			return m.openTagPicker(t.ID)
		case "f":
			// f: 該当タスクの Files 先頭ファイルへカーソルジャンプ (詳細モードへ遷移)。
			//    ファイルが 0 件なら no-op。
			if len(m.files) == 0 {
				return m, nil
			}
			filesRow := -1
			for i, r := range m.detailRows {
				if r.kind == detailRowFiles {
					filesRow = i
					break
				}
			}
			if filesRow < 0 {
				return m, nil
			}
			m.detailCursor = filesRow
			m.fileCursor = 0
			m.mode = ModeDetail
			return m, nil
		}
		return m, nil

	case ModeTagPicker:
		// 対象タスクを毎回引き直し (削除されていたら閉じる)。
		taskIdx := -1
		for i, tt := range m.tasks {
			if tt.ID == m.tagPickerTaskID {
				taskIdx = i
				break
			}
		}
		if taskIdx == -1 {
			m.mode = ModeList
			m.input = textinput.Model{}
			m.inputErr = nil
			return m, nil
		}
		// 入力欄の値で既存タグを絞り込む (検索)。空のときは全件。
		filtered := filterTags(m.tags.Sorted(), m.input.Value())
		listLen := len(filtered)

		// タイプ操作で list 件数が縮んだ場合、list 行カーソルが範囲外にならないように clamp。
		if m.tagPickerCursor > listLen {
			m.tagPickerCursor = listLen
		}

		switch {
		case key.Matches(msg, m.keys.Back):
			// 戻り先は呼び出し元が prevMode に設定 (ModeList or ModeDetail)。
			m.mode = editReturnMode(m.prevMode)
			m.input = textinput.Model{}
			m.inputErr = nil
			m.tagPickerCursor = 0
			m.tagPickerTaskID = 0
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.tagPickerCursor > 0 {
				m.tagPickerCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.tagPickerCursor < listLen { // 0..listLen (list 末端まで)
				m.tagPickerCursor++
			}
			return m, nil
		}

		switch msg.String() {
		case "c":
			// 既存タグ行 (cursor>=1) のときだけ色変更ピッカーへ遷移。
			// 入力行 (cursor=0) では typing として "c" を取り込ませるため透過。
			if m.tagPickerCursor == 0 {
				break
			}
			idx := m.tagPickerCursor - 1
			if idx < 0 || idx >= listLen {
				return m, nil
			}
			tg := filtered[idx]
			m.tagColorPickerTagID = tg.ID
			m.settingColorChoices = statusColorChoices()
			m.settingColorRow, m.settingColorCol = nearestColorChoiceCell(m.settingColorChoices, tg.Color)
			m.mode = ModeTagColorPicker
			return m, nil
		case "d":
			// 既存タグ行 (cursor>=1) のときだけ削除確認モーダルへ遷移。
			// 入力行 (cursor=0) では typing として "d" を取り込ませるため透過。
			if m.tagPickerCursor == 0 {
				break
			}
			idx := m.tagPickerCursor - 1
			if idx < 0 || idx >= listLen {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeTagPickerDeleteConfirm
			return m, nil
		case "r":
			// 既存タグ行 (cursor>=1) のときだけ rename ポップアップへ遷移。
			// 入力行 (cursor=0) では typing として "r" を取り込ませるため透過。
			if m.tagPickerCursor == 0 {
				break
			}
			idx := m.tagPickerCursor - 1
			if idx < 0 || idx >= listLen {
				return m, nil
			}
			tg := filtered[idx]
			// 検索フィルタ値を退避し、入力欄を rename 用に切り替える。
			m.tagPickerSearchSaved = m.input.Value()
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newPopupInput(inputW, task.MaxTagNameRunes)
			m.input.SetValue(tg.Name)
			m.input.CursorEnd()
			m.inputErr = task.ValidateTagNameChars(m.input.Value())
			m.tagPickerRenameTagID = tg.ID
			m.mode = ModeTagPickerRename
			return m, textinput.Blink
		case "enter":
			if m.tagPickerCursor == 0 {
				// 入力行 enter: 既存と完全一致なら toggle、そうでなければ新規作成 + 付与。
				name := strings.TrimSpace(m.input.Value())
				if name == "" || m.inputErr != nil {
					return m, nil
				}
				if existing, ok := m.tags.ByName(name); ok {
					m = m.toggleTaskTag(taskIdx, existing.ID)
					return m, nil
				}
				// 新規作成。色は 12 色パレットから round-robin で自動採番。
				autoColor := nextTagColor(m.tags)
				newTags, newID, err := m.tags.AddTag(name, autoColor)
				if err != nil {
					m.inputErr = err
					return m, nil
				}
				m.tags = newTags
				if len(m.tasks[taskIdx].Tags) >= task.MaxTagsPerTask {
					m.saveErr = task.ErrTaskTooManyTags
					if err := m.persist(); err != nil {
						m.saveErr = err
					}
					return m, nil
				}
				m.tasks[taskIdx].Tags = append(m.tasks[taskIdx].Tags, newID)
				if err := m.persist(); err != nil {
					m.saveErr = err
					return m, nil
				}
				// 入力クリア、フィルタも解除。カーソルは新規タグ位置 (Sorted 内の i+1)。
				m.input.SetValue("")
				m.inputErr = nil
				newSorted := m.tags.Sorted()
				for i, tg := range newSorted {
					if tg.ID == newID {
						m.tagPickerCursor = i + 1
						break
					}
				}
				return m, nil
			}
			// 既存タグ: cursor=1..listLen → filtered の i-1 番目。
			idx := m.tagPickerCursor - 1
			if idx < 0 || idx >= listLen {
				return m, nil
			}
			tagID := filtered[idx].ID
			m = m.toggleTaskTag(taskIdx, tagID)
			return m, nil
		}

		// cursor=0 (create input) のときはタイピングを textinput に委譲。
		// list 行 (cursor>=1) のときは何も受け付けない。
		if m.tagPickerCursor == 0 {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.inputErr = task.ValidateTagNameChars(m.input.Value())
			return m, cmd
		}
		return m, nil

	case ModeTagColorPicker:
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
			if m.settingColorCol > 0 {
				m.settingColorCol--
			}
			return m, nil
		case key.Matches(msg, m.keys.Open):
			if m.settingColorCol < cols-1 {
				m.settingColorCol++
			}
			return m, nil
		}
		switch msg.String() {
		case "enter":
			if rows == 0 || cols == 0 ||
				m.settingColorRow >= rows || m.settingColorCol >= cols {
				m.mode = ModeTagPicker
				return m, nil
			}
			newColor := grid[m.settingColorRow][m.settingColorCol]
			updated, err := m.tags.SetColorByID(m.tagColorPickerTagID, newColor)
			if err != nil {
				m.saveErr = err
				m.mode = ModeTagPicker
				return m, nil
			}
			m.tags = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeTagPicker
			return m, nil
		case "esc":
			m.mode = ModeTagPicker
			return m, nil
		}
		return m, nil

	case ModeTagPickerRename:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" || m.inputErr != nil {
				return m, nil
			}
			updated, err := m.tags.RenameByID(m.tagPickerRenameTagID, name)
			if err != nil {
				m.inputErr = err
				return m, nil
			}
			m.tags = updated
			if err := m.persist(); err != nil {
				m.saveErr = err
				return m, nil
			}
			// 検索入力欄を復元して ModeTagPicker へ戻る。
			m = m.restoreTagPickerSearch()
			m.mode = ModeTagPicker
			return m, nil
		case "esc":
			// キャンセル: 検索入力を復元してそのまま戻る。
			m = m.restoreTagPickerSearch()
			m.mode = ModeTagPicker
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.inputErr = task.ValidateTagNameChars(m.input.Value())
		return m, cmd

	case ModeTagPickerDeleteConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			// カーソル位置のタグを削除し、全タスクの Tags 配列からも除去する。
			filtered := filterTags(m.tags.Sorted(), m.input.Value())
			idx := m.tagPickerCursor - 1
			if idx < 0 || idx >= len(filtered) {
				m.mode = ModeTagPicker
				return m, nil
			}
			targetID := filtered[idx].ID
			newTags, err := m.tags.DeleteByID(targetID)
			if err != nil {
				m.saveErr = err
				m.mode = ModeTagPicker
				return m, nil
			}
			m.tags = newTags
			// 全タスクの Tags 配列から targetID を除去 (新しい slice で置換)。
			for i := range m.tasks {
				if len(m.tasks[i].Tags) == 0 {
					continue
				}
				kept := make([]int, 0, len(m.tasks[i].Tags))
				for _, id := range m.tasks[i].Tags {
					if id != targetID {
						kept = append(kept, id)
					}
				}
				m.tasks[i].Tags = kept
			}
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			// カーソルを範囲内に収める。
			newFiltered := filterTags(m.tags.Sorted(), m.input.Value())
			if m.tagPickerCursor > len(newFiltered) {
				m.tagPickerCursor = len(newFiltered)
			}
			m.mode = ModeTagPicker
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeTagPicker
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
			case settingMenuGeneral:
				m.mode = ModeSettingGeneral
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
			case settingMenuApplication:
				m.mode = ModeSettingApplication
				if m.settingApplicationCursor >= len(m.cfg.Applications) {
					m.settingApplicationCursor = 0
				}
			case settingMenuFileOpener:
				m.mode = ModeSettingFileOpener
				if m.settingFileOpenerCursor >= len(m.cfg.FileOpeners) {
					m.settingFileOpenerCursor = 0
				}
			}
			return m, nil
		}
		return m, nil

	case ModeSettingGeneral:
		// general 詳細: 0=data_base_directory のみ編集可能 (yaml は read-only)。
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSetting
			return m, nil
		case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
			// 現状は編集可能行が 1 行だけなので no-op。将来追加した場合の足掛かり。
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if m.settingGeneralCursor == 0 {
				inputW := popupWidth(m.width) - 7
				if inputW < 1 {
					inputW = 1
				}
				m.input = newPopupInput(inputW, 1024)
				m.input.SetValue(m.cfg.DataBaseDirectory)
				m.input.CursorEnd()
				m.inputErr = nil
				m.mode = ModeSettingGeneralEdit
				return m, textinput.Blink
			}
			return m, nil
		}
		return m, nil

	case ModeSettingGeneralEdit:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			m.cfg.DataBaseDirectory = value
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingGeneral
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingGeneral
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

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

	case ModeSettingApplication:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSetting
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingApplicationCursor > 0 {
				m.settingApplicationCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingApplicationCursor < len(m.cfg.Applications)-1 {
				m.settingApplicationCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if len(m.cfg.Applications) == 0 {
				return m, nil
			}
			m.settingApplicationAttrCursor = 1 // name から開始 (id は read-only)
			m.mode = ModeSettingApplicationAttribute
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			// a: 新規追加モーダル (name + run の 2 行入力)。run は textinput を遅延初期化するため
			// 突入時には name 用の textinput のみを構築し、focus 移動時に初期化を切り替える。
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newPopupInput(inputW, 256)
			m.input.Focus()
			m.inputErr = nil
			m.addApplicationFocus = 0
			m.mode = ModeSettingApplicationAdd
			return m, textinput.Blink
		case key.Matches(msg, m.keys.DeleteTask):
			if len(m.cfg.Applications) == 0 {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeSettingApplicationDeleteConfirm
			return m, nil
		case key.Matches(msg, m.keys.Move):
			if len(m.cfg.Applications) == 0 {
				return m, nil
			}
			m.settingApplicationMovingID = m.cfg.Applications[m.settingApplicationCursor].ID
			m.settingApplicationMoveBackup = append([]storage.Application{}, m.cfg.Applications...)
			m.mode = ModeSettingApplicationMove
			return m, nil
		}
		return m, nil

	case ModeSettingApplicationAttribute:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSettingApplication
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingApplicationAttrCursor > 0 {
				m.settingApplicationAttrCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingApplicationAttrCursor < 2 {
				m.settingApplicationAttrCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if m.settingApplicationCursor < 0 || m.settingApplicationCursor >= len(m.cfg.Applications) {
				return m, nil
			}
			a := m.cfg.Applications[m.settingApplicationCursor]
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			switch m.settingApplicationAttrCursor {
			case 0:
				// id は read-only
				return m, nil
			case 1:
				m.input = newPopupInput(inputW, 256)
				m.input.SetValue(a.Name)
				m.input.CursorEnd()
				m.input.Focus()
				m.inputErr = nil
				m.mode = ModeSettingApplicationEditName
				return m, textinput.Blink
			case 2:
				m.input = newPopupInput(inputW, 1024)
				m.input.SetValue(a.Run)
				m.input.CursorEnd()
				m.input.Focus()
				m.inputErr = nil
				m.mode = ModeSettingApplicationEditRun
				return m, textinput.Blink
			}
			return m, nil
		}
		return m, nil

	case ModeSettingApplicationEditName:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.inputErr = fmt.Errorf("name must not be empty")
				return m, nil
			}
			if m.settingApplicationCursor >= 0 && m.settingApplicationCursor < len(m.cfg.Applications) {
				m.cfg.Applications[m.settingApplicationCursor].Name = value
				if err := m.persist(); err != nil {
					m.saveErr = err
				}
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingApplicationAttribute
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingApplicationAttribute
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeSettingApplicationEditRun:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.inputErr = fmt.Errorf("run must not be empty")
				return m, nil
			}
			if m.settingApplicationCursor >= 0 && m.settingApplicationCursor < len(m.cfg.Applications) {
				m.cfg.Applications[m.settingApplicationCursor].Run = value
				if err := m.persist(); err != nil {
					m.saveErr = err
				}
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingApplicationAttribute
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingApplicationAttribute
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeSettingApplicationAdd:
		// 2 行モーダル: focus=0 (name) / focus=1 (run)
		switch msg.String() {
		case "tab":
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			if m.addApplicationFocus == 0 {
				m.addApplicationNameBuf = m.input.Value()
				m.input = newPopupInput(inputW, 1024)
				m.input.SetValue(m.addApplicationRunBuf)
				m.input.CursorEnd()
				m.input.Focus()
				m.addApplicationFocus = 1
			} else {
				m.addApplicationRunBuf = m.input.Value()
				m.input = newPopupInput(inputW, 256)
				m.input.SetValue(m.addApplicationNameBuf)
				m.input.CursorEnd()
				m.input.Focus()
				m.addApplicationFocus = 0
			}
			return m, textinput.Blink
		case "enter":
			var nameVal, runVal string
			if m.addApplicationFocus == 0 {
				nameVal = m.input.Value()
				runVal = m.addApplicationRunBuf
			} else {
				nameVal = m.addApplicationNameBuf
				runVal = m.input.Value()
			}
			nameVal = strings.TrimSpace(nameVal)
			runVal = strings.TrimSpace(runVal)
			if nameVal == "" || runVal == "" {
				m.inputErr = fmt.Errorf("name and run must not be empty")
				return m, nil
			}
			newID := nextApplicationID(m.cfg.Applications)
			m.cfg.Applications = append(m.cfg.Applications, storage.Application{ID: newID, Name: nameVal, Run: runVal})
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.addApplicationNameBuf = ""
			m.addApplicationRunBuf = ""
			m.settingApplicationCursor = len(m.cfg.Applications) - 1
			m.mode = ModeSettingApplication
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.addApplicationNameBuf = ""
			m.addApplicationRunBuf = ""
			m.mode = ModeSettingApplication
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeSettingApplicationMove:
		switch {
		case key.Matches(msg, m.keys.Back):
			// esc: スナップショットから復元してキャンセル
			m.cfg.Applications = m.settingApplicationMoveBackup
			m.settingApplicationMovingID = 0
			m.settingApplicationMoveBackup = nil
			m.mode = ModeSettingApplication
			return m, nil
		case key.Matches(msg, m.keys.Move), key.Matches(msg, m.keys.Confirm):
			// m / enter: 確定
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.settingApplicationMovingID = 0
			m.settingApplicationMoveBackup = nil
			m.mode = ModeSettingApplication
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingApplicationCursor > 0 {
				m.cfg.Applications[m.settingApplicationCursor], m.cfg.Applications[m.settingApplicationCursor-1] =
					m.cfg.Applications[m.settingApplicationCursor-1], m.cfg.Applications[m.settingApplicationCursor]
				m.settingApplicationCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingApplicationCursor < len(m.cfg.Applications)-1 {
				m.cfg.Applications[m.settingApplicationCursor], m.cfg.Applications[m.settingApplicationCursor+1] =
					m.cfg.Applications[m.settingApplicationCursor+1], m.cfg.Applications[m.settingApplicationCursor]
				m.settingApplicationCursor++
			}
			return m, nil
		}
		return m, nil

	case ModeSettingApplicationDeleteConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			if m.settingApplicationCursor >= 0 && m.settingApplicationCursor < len(m.cfg.Applications) {
				deletedID := m.cfg.Applications[m.settingApplicationCursor].ID
				// applications から削除
				m.cfg.Applications = append(m.cfg.Applications[:m.settingApplicationCursor], m.cfg.Applications[m.settingApplicationCursor+1:]...)
				// FileOpeners から該当 ID への参照を除去
				for i := range m.cfg.FileOpeners {
					filtered := make([]int, 0, len(m.cfg.FileOpeners[i].ApplicationIDs))
					for _, id := range m.cfg.FileOpeners[i].ApplicationIDs {
						if id != deletedID {
							filtered = append(filtered, id)
						}
					}
					m.cfg.FileOpeners[i].ApplicationIDs = filtered
					if m.cfg.FileOpeners[i].DefaultApp == deletedID {
						m.cfg.FileOpeners[i].DefaultApp = 0
					}
				}
				if err := m.persist(); err != nil {
					m.saveErr = err
				}
				if m.settingApplicationCursor >= len(m.cfg.Applications) {
					m.settingApplicationCursor = len(m.cfg.Applications) - 1
				}
				if m.settingApplicationCursor < 0 {
					m.settingApplicationCursor = 0
				}
			}
			m.mode = ModeSettingApplication
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeSettingApplication
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpener:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSetting
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFileOpenerCursor > 0 {
				m.settingFileOpenerCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFileOpenerCursor < len(m.cfg.FileOpeners)-1 {
				m.settingFileOpenerCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if len(m.cfg.FileOpeners) == 0 {
				return m, nil
			}
			m.settingFileOpenerAttrCursor = 0
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		case key.Matches(msg, m.keys.NewTask):
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			m.input = newPopupInput(inputW, 32)
			m.input.Focus()
			m.inputErr = nil
			m.mode = ModeSettingFileOpenerAdd
			return m, textinput.Blink
		case key.Matches(msg, m.keys.DeleteTask):
			if len(m.cfg.FileOpeners) == 0 {
				return m, nil
			}
			m.prevMode = m.mode
			m.mode = ModeSettingFileOpenerDeleteConfirm
			return m, nil
		case key.Matches(msg, m.keys.Move):
			if len(m.cfg.FileOpeners) == 0 {
				return m, nil
			}
			m.settingFileOpenerMovingExt = m.cfg.FileOpeners[m.settingFileOpenerCursor].Extension
			m.settingFileOpenerMoveBackup = append([]storage.FileOpener{}, m.cfg.FileOpeners...)
			m.mode = ModeSettingFileOpenerMove
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpenerAttribute:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.prevMode = m.mode
			m.mode = ModeQuitConfirm
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSettingFileOpener
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFileOpenerAttrCursor > 0 {
				m.settingFileOpenerAttrCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFileOpenerAttrCursor < 2 {
				m.settingFileOpenerAttrCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			if m.settingFileOpenerCursor < 0 || m.settingFileOpenerCursor >= len(m.cfg.FileOpeners) {
				return m, nil
			}
			op := m.cfg.FileOpeners[m.settingFileOpenerCursor]
			inputW := popupWidth(m.width) - 7
			if inputW < 1 {
				inputW = 1
			}
			switch m.settingFileOpenerAttrCursor {
			case 0:
				m.input = newPopupInput(inputW, 32)
				m.input.SetValue(op.Extension)
				m.input.CursorEnd()
				m.input.Focus()
				m.inputErr = nil
				m.mode = ModeSettingFileOpenerEditExtension
				return m, textinput.Blink
			case 1:
				// applications multi-select: 編集用バッファに現在の選択を複製
				m.settingFileOpenerAppsSelected = append([]int{}, op.ApplicationIDs...)
				m.settingFileOpenerAppsCursor = 0
				m.mode = ModeSettingFileOpenerEditApps
				return m, nil
			case 2:
				m.settingFileOpenerDefaultCursor = 0
				// default_app picker は applications 配列 (ApplicationIDs に限定せず全体)
				// + "(none)" 行を先頭にもつ。現在値があればそこに合わせる。
				if op.DefaultApp != 0 {
					for i, a := range m.cfg.Applications {
						if a.ID == op.DefaultApp {
							m.settingFileOpenerDefaultCursor = i + 1 // index 0 は (none)
							break
						}
					}
				}
				m.mode = ModeSettingFileOpenerEditDefault
				return m, nil
			}
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpenerAdd:
		switch msg.String() {
		case "enter":
			value := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(m.input.Value()), "."))
			if value == "" {
				m.inputErr = fmt.Errorf("extension must not be empty")
				return m, nil
			}
			if hasFileOpenerExtension(m.cfg.FileOpeners, value) {
				m.inputErr = fmt.Errorf("extension %q already registered", value)
				return m, nil
			}
			m.cfg.FileOpeners = append(m.cfg.FileOpeners, storage.FileOpener{Extension: value})
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.settingFileOpenerCursor = len(m.cfg.FileOpeners) - 1
			m.mode = ModeSettingFileOpener
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingFileOpener
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeSettingFileOpenerEditExtension:
		switch msg.String() {
		case "enter":
			value := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(m.input.Value()), "."))
			if value == "" {
				m.inputErr = fmt.Errorf("extension must not be empty")
				return m, nil
			}
			cur := m.cfg.FileOpeners[m.settingFileOpenerCursor]
			if value != cur.Extension && hasFileOpenerExtension(m.cfg.FileOpeners, value) {
				m.inputErr = fmt.Errorf("extension %q already registered", value)
				return m, nil
			}
			m.cfg.FileOpeners[m.settingFileOpenerCursor].Extension = value
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		case "esc":
			m.input = textinput.Model{}
			m.inputErr = nil
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case ModeSettingFileOpenerEditApps:
		// 編集用バッファ (m.settingFileOpenerAppsSelected) に対し、space でトグル、enter で確定。
		switch {
		case key.Matches(msg, m.keys.Back):
			m.settingFileOpenerAppsSelected = nil
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFileOpenerAppsCursor > 0 {
				m.settingFileOpenerAppsCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFileOpenerAppsCursor < len(m.cfg.Applications)-1 {
				m.settingFileOpenerAppsCursor++
			}
			return m, nil
		case msg.String() == " ":
			if m.settingFileOpenerAppsCursor < 0 || m.settingFileOpenerAppsCursor >= len(m.cfg.Applications) {
				return m, nil
			}
			id := m.cfg.Applications[m.settingFileOpenerAppsCursor].ID
			// toggle
			found := -1
			for i, x := range m.settingFileOpenerAppsSelected {
				if x == id {
					found = i
					break
				}
			}
			if found >= 0 {
				m.settingFileOpenerAppsSelected = append(m.settingFileOpenerAppsSelected[:found], m.settingFileOpenerAppsSelected[found+1:]...)
			} else {
				m.settingFileOpenerAppsSelected = append(m.settingFileOpenerAppsSelected, id)
			}
			return m, nil
		case key.Matches(msg, m.keys.Confirm):
			m.cfg.FileOpeners[m.settingFileOpenerCursor].ApplicationIDs = append([]int{}, m.settingFileOpenerAppsSelected...)
			// default_app が新 applications に含まれなくなった場合は 0 に戻す
			defID := m.cfg.FileOpeners[m.settingFileOpenerCursor].DefaultApp
			if defID != 0 {
				stillIn := false
				for _, x := range m.cfg.FileOpeners[m.settingFileOpenerCursor].ApplicationIDs {
					if x == defID {
						stillIn = true
						break
					}
				}
				if !stillIn {
					m.cfg.FileOpeners[m.settingFileOpenerCursor].DefaultApp = 0
				}
			}
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.settingFileOpenerAppsSelected = nil
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpenerEditDefault:
		// 候補は (none) + applications 全体。enter で確定、esc で戻る。
		switch {
		case key.Matches(msg, m.keys.Back):
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFileOpenerDefaultCursor > 0 {
				m.settingFileOpenerDefaultCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFileOpenerDefaultCursor < len(m.cfg.Applications) {
				m.settingFileOpenerDefaultCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Confirm):
			if m.settingFileOpenerDefaultCursor == 0 {
				m.cfg.FileOpeners[m.settingFileOpenerCursor].DefaultApp = 0
			} else {
				app := m.cfg.Applications[m.settingFileOpenerDefaultCursor-1]
				m.cfg.FileOpeners[m.settingFileOpenerCursor].DefaultApp = app.ID
			}
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.mode = ModeSettingFileOpenerAttribute
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpenerMove:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.cfg.FileOpeners = m.settingFileOpenerMoveBackup
			m.settingFileOpenerMovingExt = ""
			m.settingFileOpenerMoveBackup = nil
			m.mode = ModeSettingFileOpener
			return m, nil
		case key.Matches(msg, m.keys.Move), key.Matches(msg, m.keys.Confirm):
			if err := m.persist(); err != nil {
				m.saveErr = err
			}
			m.settingFileOpenerMovingExt = ""
			m.settingFileOpenerMoveBackup = nil
			m.mode = ModeSettingFileOpener
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.settingFileOpenerCursor > 0 {
				m.cfg.FileOpeners[m.settingFileOpenerCursor], m.cfg.FileOpeners[m.settingFileOpenerCursor-1] =
					m.cfg.FileOpeners[m.settingFileOpenerCursor-1], m.cfg.FileOpeners[m.settingFileOpenerCursor]
				m.settingFileOpenerCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.settingFileOpenerCursor < len(m.cfg.FileOpeners)-1 {
				m.cfg.FileOpeners[m.settingFileOpenerCursor], m.cfg.FileOpeners[m.settingFileOpenerCursor+1] =
					m.cfg.FileOpeners[m.settingFileOpenerCursor+1], m.cfg.FileOpeners[m.settingFileOpenerCursor]
				m.settingFileOpenerCursor++
			}
			return m, nil
		}
		return m, nil

	case ModeSettingFileOpenerDeleteConfirm:
		switch {
		case key.Matches(msg, m.keys.ConfirmY):
			if m.settingFileOpenerCursor >= 0 && m.settingFileOpenerCursor < len(m.cfg.FileOpeners) {
				m.cfg.FileOpeners = append(m.cfg.FileOpeners[:m.settingFileOpenerCursor], m.cfg.FileOpeners[m.settingFileOpenerCursor+1:]...)
				if err := m.persist(); err != nil {
					m.saveErr = err
				}
				if m.settingFileOpenerCursor >= len(m.cfg.FileOpeners) {
					m.settingFileOpenerCursor = len(m.cfg.FileOpeners) - 1
				}
				if m.settingFileOpenerCursor < 0 {
					m.settingFileOpenerCursor = 0
				}
			}
			m.mode = ModeSettingFileOpener
			return m, nil
		case key.Matches(msg, m.keys.ConfirmN):
			m.mode = ModeSettingFileOpener
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
		case key.Matches(msg, m.keys.Refresh):
			// R: 現在カーソルが指すタスクのファイル一覧を再読込する。
			// 外部 (Finder / mv / 他プロセス) で発生したファイル変更を取り込むためのキー。
			m = m.withFilesRefreshed()
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
		case key.Matches(msg, m.keys.CopyPath):
			// p: カーソル位置のタスクのデータディレクトリ絶対パスをクリップボードへ。
			path, ok := m.pathForCopy()
			if !ok {
				return m, nil
			}
			if err := copyToClipboard(path); err != nil {
				m.saveErr = err
			}
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
// restoreTagPickerSearch は ModeTagPickerRename で退避した検索フィルタ文字列を
// ModeTagPicker の入力欄に書き戻し、rename 用の状態をクリアする。
func (m Model) restoreTagPickerSearch() Model {
	inputW := popupWidth(m.width) - 7
	if inputW < 1 {
		inputW = 1
	}
	m.input = newPopupInput(inputW, task.MaxTagNameRunes)
	m.input.Placeholder = "Search tag or Create tag"
	m.input.SetValue(m.tagPickerSearchSaved)
	m.input.CursorEnd()
	m.tagPickerSearchSaved = ""
	m.tagPickerRenameTagID = 0
	m.inputErr = task.ValidateTagNameChars(m.input.Value())
	return m
}

// nextTagColor は既存タグ数を元にパレットから次の色を round-robin で返す。
// 12 色パレット (status と共用) の i = len(tags) % 12 番目。
func nextTagColor(tags task.TagList) string {
	palette := colorPickerBaseHexes
	if len(palette) == 0 {
		return ""
	}
	return palette[len(tags)%len(palette)]
}

// toggleTaskTag は taskIdx 番目のタスクの Tags に tagID を toggle する。
// 付与済みなら外し、未付与なら追加 (上限到達時は saveErr に表示)。
// 永続化エラーも saveErr へ。
func (m Model) toggleTaskTag(taskIdx, tagID int) Model {
	tags := m.tasks[taskIdx].Tags
	pos := -1
	for i, id := range tags {
		if id == tagID {
			pos = i
			break
		}
	}
	if pos >= 0 {
		m.tasks[taskIdx].Tags = append(tags[:pos], tags[pos+1:]...)
	} else {
		if len(tags) >= task.MaxTagsPerTask {
			m.saveErr = task.ErrTaskTooManyTags
			return m
		}
		m.tasks[taskIdx].Tags = append(tags, tagID)
	}
	if err := m.persist(); err != nil {
		m.saveErr = err
	}
	return m
}

// openTagPicker は taskID を対象にタグピッカーモーダルを開く。
// 入力欄を初期化し、cursor を 0 (create input) に置く。
func (m Model) openTagPicker(taskID int) (Model, tea.Cmd) {
	inputW := popupWidth(m.width) - 7
	if inputW < 1 {
		inputW = 1
	}
	m.input = newPopupInput(inputW, task.MaxTagNameRunes)
	m.input.Placeholder = "Search tag or Create tag"
	m.inputErr = nil
	m.tagPickerTaskID = taskID
	m.tagPickerCursor = 0
	m.mode = ModeTagPicker
	return m, textinput.Blink
}

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

// openCurrentFileWithDefault は file list で enter が押されたときの起動フロー。
// file_opener.default_app が指定されていればそのアプリ、未指定なら $EDITOR フォールバック。
// モーダル表示はせず必ず即起動する (`o` キーとの差別化)。ディレクトリ行は呼び出し側で除外する前提。
func (m Model) openCurrentFileWithDefault() (Model, tea.Cmd) {
	t, _, ok := m.currentTask()
	if !ok {
		return m, nil
	}
	cur, ok := m.currentFileRow()
	if !ok || cur.isDir {
		return m, nil
	}
	if app, ok := resolveDefaultApp(cur.name, m.cfg.Applications, m.cfg.FileOpeners); ok {
		return m.launchAppForFile(app.Run, t.ID, cur.relPath)
	}
	return m.launchAppForFile("", t.ID, cur.relPath)
}

// openCurrentFileWithPicker は file list で o が押されたときの起動フロー。
//
//   - file_opener に該当拡張子の設定がある: 候補数で分岐
//     候補 1 件 → そのアプリで即起動
//     候補 2 件以上 → モーダル (ModeFileOpener) を開いてユーザーに選択させる
//   - 該当拡張子設定なし / 候補 0 件 → $EDITOR でフォールバック起動
func (m Model) openCurrentFileWithPicker() (Model, tea.Cmd) {
	t, _, ok := m.currentTask()
	if !ok {
		return m, nil
	}
	cur, ok := m.currentFileRow()
	if !ok || cur.isDir {
		return m, nil
	}
	candidates := resolveFileOpenerCandidates(cur.name, m.cfg.Applications, m.cfg.FileOpeners)
	switch len(candidates) {
	case 0:
		return m.launchAppForFile("", t.ID, cur.relPath)
	case 1:
		return m.launchAppForFile(candidates[0].Run, t.ID, cur.relPath)
	default:
		// 2 件以上はモーダルで選択させる。
		m.fileOpenerCandidates = candidates
		m.fileOpenerCursor = 0
		m.fileOpenerTaskID = t.ID
		m.fileOpenerFile = cur.relPath
		m.prevMode = m.mode
		m.mode = ModeFileOpener
		return m, nil
	}
}

// launchAppForFile は appPath (空文字なら $EDITOR フォールバック) で taskID 配下の
// relPath (タスクディレクトリからの相対パス) を開く tea.Cmd を返す。
func (m Model) launchAppForFile(appPath string, taskID int, relPath string) (Model, tea.Cmd) {
	taskDir := storage.TaskDir(m.yamlDir, m.cfg.DataBaseDirectory, taskID)
	filePath := filepath.Join(taskDir, filepath.FromSlash(relPath))
	cmd, err := buildAppCmd(appPath, filePath)
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

// twoPaneWidths は通常画面 (list 2/3 / detail+files+preview 1/3) の各ペイン幅を返す。
// 区切り線 1 本 (1 cell) を差し引いた残りを 2:1 で割り当てる。
// 画面が狭すぎる場合は両ペイン 1 cell ずつにフォールバック。
func twoPaneWidths(screenW int) (leftW, rightW int) {
	avail := screenW - 1 // 区切り線 1 本ぶん
	if avail < 2 {
		return 1, 1
	}
	leftW = avail * 2 / 3
	if leftW < 1 {
		leftW = 1
	}
	rightW = avail - leftW
	if rightW < 1 {
		rightW = 1
		if leftW > 1 {
			leftW = avail - rightW
		}
	}
	return
}

// isSettingMode は現在のモードが設定画面 (左/中/右ペイン or 設定画面内のサブモード) かを返す。
// View 切替の判定に使う。
func isSettingMode(m Mode) bool {
	switch m {
	case ModeSetting,
		ModeSettingGeneral, ModeSettingGeneralEdit,
		ModeSettingStatus, ModeSettingStatusRename, ModeSettingStatusAdd,
		ModeSettingStatusColor, ModeSettingStatusMove, ModeSettingStatusDeleteConfirm,
		ModeSettingField, ModeSettingFieldAttribute,
		ModeSettingFieldAdd, ModeSettingFieldRename,
		ModeSettingFieldMove, ModeSettingFieldDeleteConfirm,
		ModeSettingApplication, ModeSettingApplicationAttribute,
		ModeSettingApplicationAdd, ModeSettingApplicationEditName, ModeSettingApplicationEditRun,
		ModeSettingApplicationMove, ModeSettingApplicationDeleteConfirm,
		ModeSettingFileOpener, ModeSettingFileOpenerAttribute,
		ModeSettingFileOpenerAdd, ModeSettingFileOpenerEditExtension,
		ModeSettingFileOpenerEditApps, ModeSettingFileOpenerEditDefault,
		ModeSettingFileOpenerMove, ModeSettingFileOpenerDeleteConfirm:
		return true
	}
	return false
}

// isSettingApplicationFocus は設定画面で「application」側を見ている状態かを返す (3 ペインレイアウト用)。
func (m Model) isSettingApplicationFocus() bool {
	switch m.mode {
	case ModeSettingApplication, ModeSettingApplicationAttribute,
		ModeSettingApplicationAdd, ModeSettingApplicationEditName, ModeSettingApplicationEditRun,
		ModeSettingApplicationMove, ModeSettingApplicationDeleteConfirm:
		return true
	case ModeSetting:
		return m.settingMenuCursor == settingMenuApplication
	}
	return false
}

// isSettingFileOpenerFocus は設定画面で「file_opener」側を見ている状態かを返す (3 ペインレイアウト用)。
func (m Model) isSettingFileOpenerFocus() bool {
	switch m.mode {
	case ModeSettingFileOpener, ModeSettingFileOpenerAttribute,
		ModeSettingFileOpenerAdd, ModeSettingFileOpenerEditExtension,
		ModeSettingFileOpenerEditApps, ModeSettingFileOpenerEditDefault,
		ModeSettingFileOpenerMove, ModeSettingFileOpenerDeleteConfirm:
		return true
	case ModeSetting:
		return m.settingMenuCursor == settingMenuFileOpener
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

// isSettingGeneralFocus は設定画面で「general」側を見ている状態かを返す。
// ModeSetting (メニュー) 中は cursor が general を指しているかで判断する。
func (m Model) isSettingGeneralFocus() bool {
	if m.mode == ModeSettingGeneral || m.mode == ModeSettingGeneralEdit {
		return true
	}
	if m.mode == ModeSetting {
		return m.settingMenuCursor == settingMenuGeneral
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

// View は現在の Model をレンダリング済み文字列にして返す。Bubble Tea がこれを毎フレーム呼ぶ。
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	bodyH := m.height - 1
	divider := buildPaneDivider(bodyH, "", -1)

	listFocused := m.mode == ModeList || m.mode == ModeQuitConfirm || m.mode == ModeMove || m.mode == ModeOperation
	detailFocused := m.mode == ModeDetail || m.mode == ModeEditTitle || m.mode == ModeEditStatus
	// ModePrefix は元々タスクリストからしか入れなかったため list フォーカス固定で良かったが、
	// 詳細画面からも ; が入れるようになったので prevMode に応じて維持する。
	if m.mode == ModePrefix {
		switch m.prevMode {
		case ModeDetail, ModeEditTitle, ModeEditStatus:
			detailFocused = true
		default:
			listFocused = true
		}
	}
	// レイアウトモード中は突入時のフォーカスに応じて、視覚的なフォーカス表現を維持する。
	if m.mode == ModeLayout {
		switch m.layoutFocus {
		case layoutFocusTaskList:
			listFocused = true
		case layoutFocusTaskDetail, layoutFocusFileList:
			detailFocused = true
		}
	}

	inMoveMode := m.mode == ModeMove
	inLayoutMode := m.mode == ModeLayout

	var body string
	if isSettingMode(m.mode) {
		menuFocused := m.mode == ModeSetting
		if m.isSettingApplicationFocus() {
			// application 系: 3 ペイン (menu 12cell + 中央 + 右 attributes)。
			leftW := 12
			if leftW > m.width-2 {
				leftW = m.width - 2
				if leftW < 1 {
					leftW = 1
				}
			}
			remain := m.width - leftW - 2
			if remain < 2 {
				remain = 2
			}
			midW := remain / 2
			rightW := remain - midW
			midFocused := m.mode == ModeSettingApplication || m.mode == ModeSettingApplicationAdd ||
				m.mode == ModeSettingApplicationMove || m.mode == ModeSettingApplicationDeleteConfirm
			rightFocused := m.mode == ModeSettingApplicationAttribute ||
				m.mode == ModeSettingApplicationEditName || m.mode == ModeSettingApplicationEditRun
			inAppMove := m.mode == ModeSettingApplicationMove
			left, mid, right := renderSettingApplication(m.cfg.Applications, m.settingMenuCursor, m.settingApplicationCursor, m.settingApplicationAttrCursor,
				menuFocused, midFocused, rightFocused, inAppMove, leftW, midW, rightW, bodyH)
			if inAppMove {
				banner := styleMoveBanner.Render("-- MOVE MODE --")
				bannerW := lipgloss.Width(banner)
				x := midW - bannerW
				if x < 0 {
					x = 0
				}
				mid = PlaceOverlay(x, 0, banner, mid)
			}
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, mid, divider, right)
		} else if m.isSettingFileOpenerFocus() {
			leftW := 12
			if leftW > m.width-2 {
				leftW = m.width - 2
				if leftW < 1 {
					leftW = 1
				}
			}
			remain := m.width - leftW - 2
			if remain < 2 {
				remain = 2
			}
			midW := remain / 2
			rightW := remain - midW
			midFocused := m.mode == ModeSettingFileOpener || m.mode == ModeSettingFileOpenerAdd ||
				m.mode == ModeSettingFileOpenerMove || m.mode == ModeSettingFileOpenerDeleteConfirm
			rightFocused := m.mode == ModeSettingFileOpenerAttribute ||
				m.mode == ModeSettingFileOpenerEditExtension || m.mode == ModeSettingFileOpenerEditApps ||
				m.mode == ModeSettingFileOpenerEditDefault
			inOpenerMove := m.mode == ModeSettingFileOpenerMove
			left, mid, right := renderSettingFileOpener(m.cfg.FileOpeners, m.cfg.Applications, m.settingMenuCursor, m.settingFileOpenerCursor, m.settingFileOpenerAttrCursor,
				menuFocused, midFocused, rightFocused, inOpenerMove, leftW, midW, rightW, bodyH)
			if inOpenerMove {
				banner := styleMoveBanner.Render("-- MOVE MODE --")
				bannerW := lipgloss.Width(banner)
				x := midW - bannerW
				if x < 0 {
					x = 0
				}
				mid = PlaceOverlay(x, 0, banner, mid)
			}
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, mid, divider, right)
		} else if m.isSettingFieldFocus() {
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
			// general / status 系: 左メニュー + 右詳細の 2 ペイン。
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
			var left, right string
			if m.isSettingGeneralFocus() {
				rightFocused := m.mode == ModeSettingGeneral || m.mode == ModeSettingGeneralEdit
				left, right = renderSettingGeneral(m.yamlPath, m.cfg.DataBaseDirectory, m.settingMenuCursor, m.settingGeneralCursor, menuFocused, rightFocused, leftW, rightW, bodyH)
			} else {
				inSettingMove := m.mode == ModeSettingStatusMove
				left, right = renderSettingStatus(m.statuses, m.settingMenuCursor, m.settingStatusCursor, menuFocused, inSettingMove, leftW, rightW, bodyH)
				if inSettingMove {
					banner := styleMoveBanner.Render("-- MOVE MODE --")
					bannerW := lipgloss.Width(banner)
					x := rightW - bannerW
					if x < 0 {
						x = 0
					}
					right = PlaceOverlay(x, 0, banner, right)
				}
			}
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
		}
	} else {
		// 通常画面: 左 タスクリスト | 右 (上から detail / files / preview)。
		// レイアウト保存値が完全 (4 値とも非 nil) のときはそれを反映、未設定なら従来計算。
		var leftW, rightW, detailH, fileAreaH, previewAreaH int
		if isLayoutComplete(m.layout) {
			leftW, rightW, detailH, fileAreaH, previewAreaH = applyLayoutToScreen(m.layout, m.width, bodyH)
		} else {
			leftW, rightW = twoPaneWidths(m.width)
			detailH = bodyH / 3
			fileAreaH = bodyH / 3
			previewAreaH = bodyH - detailH - fileAreaH
			if detailH < 3 {
				detailH = 3
			}
			if fileAreaH < 3 {
				fileAreaH = 3
			}
			if previewAreaH < 2 {
				previewAreaH = 2
			}
			if total := detailH + fileAreaH + previewAreaH; total > bodyH {
				over := total - bodyH
				if previewAreaH-over >= 2 {
					previewAreaH -= over
				} else {
					previewAreaH = 2
				}
			}
		}

		listH := bodyH
		if m.viewTrash {
			// ゴミ箱ビューでは最上部 1 行をヘッダで占有するので、リスト本体の高さを 1 減らす。
			listH = bodyH - 1
			if listH < 1 {
				listH = 1
			}
		}

		left := renderList(m.tasks, m.statuses, m.tags, m.rows, m.collapsed, m.cursor, listFocused, inMoveMode, leftW, listH)
		if inMoveMode {
			banner := styleMoveBanner.Render("-- MOVE MODE --")
			bannerW := lipgloss.Width(banner)
			x := leftW - bannerW
			if x < 0 {
				x = 0
			}
			left = PlaceOverlay(x, 0, banner, left)
		}
		if m.mode == ModeOperation {
			banner := styleOperationBanner.Render("-- OPERATION MODE --")
			bannerW := lipgloss.Width(banner)
			x := leftW - bannerW
			if x < 0 {
				x = 0
			}
			left = PlaceOverlay(x, 0, banner, left)
		}
		if inLayoutMode {
			label := "-- LAYOUT --"
			switch m.layoutFocus {
			case layoutFocusTaskDetail:
				label = "-- LAYOUT (detail) --"
			case layoutFocusFileList:
				label = "-- LAYOUT (files) --"
			}
			banner := styleLayoutBanner.Render(label)
			bannerW := lipgloss.Width(banner)
			x := leftW - bannerW
			if x < 0 {
				x = 0
			}
			left = PlaceOverlay(x, 0, banner, left)
		}
		if m.viewTrash {
			// 左ペインの最上部に「-- TRASH BOX --」ヘッダ行 (黒抜き赤背景) を 1 行追加。
			header := renderSingleLineRow(styleTrashHeader, "-- TRASH BOX --", leftW)
			left = lipgloss.JoinVertical(lipgloss.Left, header, left)
		}

		var current *task.Task
		if t, _, ok := m.currentTask(); ok {
			current = &t
		}

		namesH := fileAreaH - 2 // header + top divider
		if namesH < 1 {
			namesH = 1
		}
		previewH := previewAreaH - 1 // bottom divider
		if previewH < 1 {
			previewH = 1
		}

		// 詳細ペイン (Files なし)。
		detailBlock := renderDetail(current, m.statuses, m.fields, m.tags, m.detailRows, detailFocused, m.detailCursor, rightW, detailH)

		// Files: header + 罫線 + 名前リスト。
		filesHeader := renderSingleLineRow(lipgloss.NewStyle(), "  "+styleLabel.Render("Files:"), rightW)
		hDivider := styleDivider.Render(strings.Repeat("─", rightW))
		hasCursorOnFiles := false
		if row, ok := m.currentDetailRow(); ok && row.kind == detailRowFiles {
			hasCursorOnFiles = detailFocused
		}
		fileNamesBlock := renderFileNamesList(m.files, detailFocused, hasCursorOnFiles, m.fileCursor, rightW, namesH)

		// プレビュー: 現在タスクの fileCursor が指す通常ファイルを対象にする。
		// カーソルがディレクトリ行のときは空ペインを返す (renderPreview の空指定経路)。
		var previewFile string
		var previewTaskID int
		if current != nil && len(m.files) > 0 && m.fileCursor >= 0 && m.fileCursor < len(m.files) {
			if cur := m.files[m.fileCursor]; !cur.isDir {
				previewFile = cur.relPath
				previewTaskID = current.ID
			}
		}
		previewBlock := renderPreview(m.yamlDir, m.cfg.DataBaseDirectory, previewTaskID, previewFile, rightW, previewH)

		right := lipgloss.JoinVertical(lipgloss.Left,
			detailBlock,
			filesHeader,
			hDivider,
			fileNamesBlock,
			hDivider,
			previewBlock,
		)

		divider := buildPaneDivider(bodyH, "", -1)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
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
	footer := renderFooter(m.mode, m.prevMode, onFilesRow, onURLRow, m.viewTrash, m.layoutFocus, m.width)

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
	case ModeFileOpener:
		view = overlayFileOpenerPicker(view, m.fileOpenerCandidates, m.fileOpenerCursor, m.width, m.height-1)
	case ModeTagPicker, ModeTagColorPicker, ModeTagPickerRename, ModeTagPickerDeleteConfirm:
		var assigned []int
		for _, tt := range m.tasks {
			if tt.ID == m.tagPickerTaskID {
				assigned = tt.Tags
				break
			}
		}
		// 背後の tagpicker を描画する。Rename 中は m.input がリネーム用に占有されているので、
		// 退避済みの検索フィルタ値を描画値として使う。
		var pickerInputValue string
		var pickerCursorPos int
		var pickerInputErr error
		if m.mode == ModeTagPickerRename {
			pickerInputValue = m.tagPickerSearchSaved
			pickerCursorPos = len([]rune(pickerInputValue))
			pickerInputErr = nil
		} else {
			pickerInputValue = m.input.Value()
			pickerCursorPos = m.input.Position()
			pickerInputErr = m.inputErr
		}
		// 行全体の背景色を popup bg に統一するため、textinput.View() ではなく値とカーソル位置を渡して自前描画する。
		view = overlayTagPicker(view, m.tags, assigned, m.tagPickerCursor, pickerInputValue, pickerCursorPos, pickerInputErr, m.width, m.height-1)
		switch m.mode {
		case ModeTagColorPicker:
			// タグピッカーの上に色ピッカーをさらに重ねる。
			view = overlayColorPicker(view, "Tag Color:", m.settingColorChoices, m.settingColorRow, m.settingColorCol, m.width, m.height-1)
		case ModeTagPickerRename:
			// タグピッカーの上に rename 入力モーダルを重ねる。
			view = overlayInputPopup(view, "Rename tag:", m.input.View(), m.inputErr, m.width, m.height-1)
		case ModeTagPickerDeleteConfirm:
			// 削除確認モーダルを重ねる。対象タグ名を表示。
			confMsg := "delete tag?"
			filtered := filterTags(m.tags.Sorted(), m.input.Value())
			idx := m.tagPickerCursor - 1
			if idx >= 0 && idx < len(filtered) {
				confMsg = "delete tag \"" + filtered[idx].Name + "\" ? (also removes from all tasks)"
			}
			view = overlayConfirmPopup(view, "Delete?", confMsg,
				[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
				m.width, m.height-1)
		}
	case ModeSettingGeneralEdit:
		view = overlayInputPopup(view, "data_base_directory:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingStatusRename:
		view = overlayInputPopup(view, "Rename status:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingStatusAdd:
		view = overlayInputPopup(view, "Add status:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingStatusColor:
		view = overlayColorPicker(view, "Status Color:", m.settingColorChoices, m.settingColorRow, m.settingColorCol, m.width, m.height-1)
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
	case ModeSettingApplicationEditName:
		view = overlayInputPopup(view, "Rename application:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingApplicationEditRun:
		view = overlayInputPopup(view, "Edit run:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingApplicationAdd:
		view = overlayApplicationAddPopup(view, m.input.View(), m.inputErr, m.addApplicationFocus, m.addApplicationNameBuf, m.addApplicationRunBuf, m.width, m.height-1)
	case ModeSettingApplicationDeleteConfirm:
		msg := "delete application?"
		if m.settingApplicationCursor >= 0 && m.settingApplicationCursor < len(m.cfg.Applications) {
			msg = "delete application \"" + m.cfg.Applications[m.settingApplicationCursor].Name + "\" ?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeSettingFileOpenerAdd:
		view = overlayInputPopup(view, "Add file_opener (extension):", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingFileOpenerEditExtension:
		view = overlayInputPopup(view, "Edit extension:", m.input.View(), m.inputErr, m.width, m.height-1)
	case ModeSettingFileOpenerEditApps:
		view = overlayFileOpenerAppsPicker(view, m.cfg.Applications, m.settingFileOpenerAppsSelected, m.settingFileOpenerAppsCursor, m.width, m.height-1)
	case ModeSettingFileOpenerEditDefault:
		view = overlayFileOpenerDefaultPicker(view, m.cfg.Applications, m.settingFileOpenerDefaultCursor, m.width, m.height-1)
	case ModeSettingFileOpenerDeleteConfirm:
		msg := "delete file_opener?"
		if m.settingFileOpenerCursor >= 0 && m.settingFileOpenerCursor < len(m.cfg.FileOpeners) {
			msg = "delete file_opener \"." + m.cfg.FileOpeners[m.settingFileOpenerCursor].Extension + "\" ?"
		}
		view = overlayConfirmPopup(view, "Delete?", msg,
			[]hintItem{{"y", "delete"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeQuitConfirm:
		view = overlayConfirmPopup(view, "Quit?", "are you sure?",
			[]hintItem{{"y", "quit"}, {"n/esc", "cancel"}},
			m.width, m.height-1)
	case ModeDeleteFileConfirm:
		title := "Delete?"
		msg := "delete file?"
		if cur, ok := m.currentFileRow(); ok {
			if cur.isDir {
				// ディレクトリ削除は再帰削除なので、配下も消えることを明示する。
				title = "Delete directory?"
				msg = "delete directory \"" + cur.relPath + "/\" and all files inside?"
			} else {
				msg = "delete \"" + cur.relPath + "\" ?"
			}
		}
		view = overlayConfirmPopup(view, title, msg,
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
	inputPadded := renderSingleLineRow(stylePopupFill, inputView, contentW)
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
		errPadded := renderSingleLineRow(stylePopupFill, errMsg, contentW)
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

// overlayFileOpenerPicker は file_opener の application 候補をポップアップとして中央オーバーレイする。
// 描画は overlayStatusPicker と同形式 (Label = "Open with:")。
func overlayFileOpenerPicker(bg string, candidates []storage.Application, currentIdx, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 4 {
		contentW = 4
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Open with:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "open"}, {"Esc", "cancel"},
	}), innerW)

	rows := []string{topRow}
	for i, app := range candidates {
		raw := "  " + app.Name
		if w := ansi.StringWidth(raw); w > contentW {
			raw = ansi.Truncate(raw, contentW, "")
		}
		var padded string
		if i == currentIdx {
			padded = renderSingleLineRow(stylePopupCursorRow, raw, contentW)
		} else {
			padded = renderSingleLineRow(stylePopupFill.Foreground(colorText), raw, contentW)
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
			padded = renderSingleLineRow(stylePopupCursorRow, raw, contentW)
		} else {
			padded = renderSingleLineRow(stylePopupFill.Foreground(colorText), raw, contentW)
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
	padded := renderSingleLineRow(stylePopupFill, body, contentW)
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
	padded := renderSingleLineRow(stylePopupFill, body, contentW)
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
