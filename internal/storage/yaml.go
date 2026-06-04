package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/miyazi777/task-man/internal/task"
)

type yamlStatus struct {
	ID        int    `yaml:"id"`
	Sequence  int    `yaml:"sequence"`
	Label     string `yaml:"label"`
	Color     string `yaml:"color,omitempty"`
	Collapsed bool   `yaml:"collapsed,omitempty"`
}

type yamlStatusEntry struct {
	Status yamlStatus `yaml:"status"`
}

// yamlField はトップレベル fields のスキーマ定義 1 件。
type yamlField struct {
	ID       int    `yaml:"id"`
	Type     string `yaml:"type,omitempty"`
	Name     string `yaml:"name"`
	Position int    `yaml:"position,omitempty"`
}

type yamlFieldEntry struct {
	Field yamlField `yaml:"field"`
}

// yamlTaskField は task の下に持たれる field の値。field_id で top-level fields を参照する。
type yamlTaskField struct {
	ID      int    `yaml:"id"`
	FieldID int    `yaml:"field_id"`
	Value   string `yaml:"value,omitempty"`
}

type yamlTaskFieldEntry struct {
	Field yamlTaskField `yaml:"field"`
}

// yamlTag はトップレベル tags の 1 件。
type yamlTag struct {
	ID    int    `yaml:"id"`
	Name  string `yaml:"name"`
	Color string `yaml:"color,omitempty"`
}

type yamlTagEntry struct {
	Tag yamlTag `yaml:"tag"`
}

type yamlTask struct {
	ID            int                  `yaml:"id"`
	Title         string               `yaml:"title"`
	StatusID      int                  `yaml:"status_id"`
	ParentID      int                  `yaml:"parent_id,omitempty"`
	Position      int                  `yaml:"position,omitempty"`
	Collapsed     bool                 `yaml:"collapsed,omitempty"`
	CollapsedDirs []string             `yaml:"collapsed_dirs,omitempty"`
	IsTrashBox    bool                 `yaml:"is_trash_box,omitempty"`
	Tags          []int                `yaml:"tags,omitempty"`
	Fields        []yamlTaskFieldEntry `yaml:"fields,omitempty"`
}

type yamlEntry struct {
	Task yamlTask `yaml:"task"`
}

// yamlApplication は applications 配列の 1 件分 (新スキーマ)。
type yamlApplication struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

type yamlApplicationEntry struct {
	Application yamlApplication `yaml:"application"`
}

// yamlFileOpener は file_opener 配列の 1 件分。
type yamlFileOpener struct {
	Extension    string `yaml:"extension"`
	Applications []int  `yaml:"applications"`
	DefaultApp   int    `yaml:"default_app,omitempty"`
}

type yamlFileOpenerEntry struct {
	Opener yamlFileOpener `yaml:"opener"`
}

// yamlLayoutValue は layout.main.<pane> の単一ペイン分。width / height は片方のみ使う。
type yamlLayoutValue struct {
	Width  *float64 `yaml:"width,omitempty"`
	Height *float64 `yaml:"height,omitempty"`
}

type yamlLayoutMain struct {
	TaskList    yamlLayoutValue `yaml:"task_list,omitempty"`
	TaskDetail  yamlLayoutValue `yaml:"task_detail,omitempty"`
	FileList    yamlLayoutValue `yaml:"file_list,omitempty"`
	FilePreview yamlLayoutValue `yaml:"file_preview,omitempty"`
}

type yamlLayout struct {
	Main yamlLayoutMain `yaml:"main,omitempty"`
}

// yamlCursor は ModeList の最終カーソル位置を yaml に保存するための値型。
// 両フィールドとも 0 のときは yamlFile.Cursor を nil にして cursor キーごと省略する。
type yamlCursor struct {
	TaskID   int `yaml:"task_id,omitempty"`
	StatusID int `yaml:"status_id,omitempty"`
}

type yamlFile struct {
	// Version は yaml スキーマのバージョン。task-man が読み書きする現行バージョンは
	// CurrentSchemaVersion。version キーが無い旧 yaml は v1 として扱い、Load 後の
	// 再 Save で version: 1 を補完する。
	Version           int                    `yaml:"version,omitempty"`
	Applications      []yamlApplicationEntry `yaml:"applications,omitempty"`
	DataBaseDirectory string                 `yaml:"data_base_directory,omitempty"`
	Layout            *yamlLayout            `yaml:"layout,omitempty"`
	Cursor            *yamlCursor            `yaml:"cursor,omitempty"`
	FileOpener        []yamlFileOpenerEntry  `yaml:"file_opener,omitempty"`
	Statuses          []yamlStatusEntry      `yaml:"statuses"`
	Fields            []yamlFieldEntry       `yaml:"fields,omitempty"`
	Tags              []yamlTagEntry         `yaml:"tags,omitempty"`
	Tasks             []yamlEntry            `yaml:"tasks"`
}

// CurrentSchemaVersion は task-man が読み書きする tasks.yaml のスキーマバージョン。
// 互換性を破る変更を加えるたびに 1 ずつ増やし、Load 側で旧版からの移行ロジックを
// 追加していく (現時点では v1 しか存在しないため移行ロジックは未実装)。
const CurrentSchemaVersion = 1

// ErrSchemaVersionUnsupported は yaml の version が現行 (CurrentSchemaVersion) より
// 大きい場合に返される。新しいバージョンの task-man で書かれた yaml を古いバイナリで
// 開いた可能性が高く、データ破壊を避けるため起動を中断する。
var ErrSchemaVersionUnsupported = errors.New("tasks.yaml schema version is newer than this binary supports")

// YAMLRepository は tasks.yaml をバックエンドにする Repository 実装。
type YAMLRepository struct {
	Path string
}

// NewYAMLRepository は指定 path の yaml をバックエンドにする Repository を返す。
// 実際の I/O は Load / Save 呼び出し時に発生する。
func NewYAMLRepository(path string) *YAMLRepository {
	return &YAMLRepository{Path: path}
}

// Load は yaml を読み込み LoadResult を返す。id / position / sequence の欠落を
// 補完した場合は内部で Save し直して yaml の表現を正規化する。
func (r *YAMLRepository) Load() (LoadResult, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return LoadResult{}, fmt.Errorf("read %s: %w", r.Path, err)
	}

	var f yamlFile
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &f); err != nil {
			return LoadResult{}, fmt.Errorf("parse %s: %w", r.Path, err)
		}
	}

	// スキーマバージョン検証。
	//   - version > CurrentSchemaVersion: 未来バージョンの yaml を読み書きするとデータ破壊の恐れ
	//     があるため、起動を拒否する。
	//   - version == 0: yaml に version キーが無い (旧 yaml)。現行バージョンとして扱い、
	//     Load 後に再 Save して version: 1 を補完する。
	//   - 1 <= version <= CurrentSchemaVersion: そのまま受け入れる (将来 version < CurrentSchemaVersion
	//     になったら移行ロジックを挟む箇所)。
	if f.Version > CurrentSchemaVersion {
		return LoadResult{}, fmt.Errorf("%w: %s has version %d, supported up to %d", ErrSchemaVersionUnsupported, r.Path, f.Version, CurrentSchemaVersion)
	}
	versionChanged := f.Version != CurrentSchemaVersion

	statuses, statusesChanged := loadStatuses(f.Statuses)
	if err := statuses.Validate(); err != nil {
		return LoadResult{}, err
	}

	defs, defsChanged := loadFieldDefs(f.Fields)
	if err := defs.Validate(); err != nil {
		return LoadResult{}, err
	}

	tags, tagsChanged := loadTags(f.Tags)
	if err := tags.Validate(); err != nil {
		return LoadResult{}, err
	}

	tasks, tasksChanged, err := loadTasks(f.Tasks, statuses, defs, tags)
	if err != nil {
		return LoadResult{}, err
	}

	apps, err := loadApplications(f.Applications)
	if err != nil {
		return LoadResult{}, err
	}
	openers, err := loadFileOpeners(f.FileOpener, apps)
	if err != nil {
		return LoadResult{}, err
	}

	cfg := AppConfig{
		DataBaseDirectory: f.DataBaseDirectory,
		Layout:            loadLayout(f.Layout),
		Applications:      apps,
		FileOpeners:       openers,
		Cursor:            loadCursor(f.Cursor),
	}

	lr := LoadResult{
		Tasks:    tasks,
		Statuses: statuses,
		Fields:   defs,
		Tags:     tags,
		Config:   cfg,
	}

	if statusesChanged || defsChanged || tagsChanged || tasksChanged || versionChanged {
		if err := r.Save(lr); err != nil {
			return LoadResult{}, fmt.Errorf("write back defaults: %w", err)
		}
	}

	return lr, nil
}

// loadApplications は yaml の applications 配列を Application 型へ変換する。
//   - id, name, path のいずれかが欠落していたらエラー
//   - id の重複もエラー
func loadApplications(entries []yamlApplicationEntry) ([]Application, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	seen := make(map[int]struct{}, len(entries))
	out := make([]Application, 0, len(entries))
	for i, e := range entries {
		if e.Application.ID <= 0 {
			return nil, fmt.Errorf("applications[%d]: id must be positive (got %d)", i, e.Application.ID)
		}
		if _, dup := seen[e.Application.ID]; dup {
			return nil, fmt.Errorf("applications[%d]: duplicated id %d", i, e.Application.ID)
		}
		seen[e.Application.ID] = struct{}{}
		if e.Application.Name == "" {
			return nil, fmt.Errorf("applications[%d]: name must not be empty", i)
		}
		if e.Application.Run == "" {
			return nil, fmt.Errorf("applications[%d]: run must not be empty", i)
		}
		out = append(out, Application{
			ID:   e.Application.ID,
			Name: e.Application.Name,
			Run:  e.Application.Run,
		})
	}
	return out, nil
}

// loadFileOpeners は yaml の file_opener 配列を FileOpener 型へ変換する。
//   - extension は空文字エラー、先頭の "." は除去、小文字化
//   - 同一 extension の重複は最後に書かれたものを採用 (yaml の上書き直感に合わせる)
//   - applications の各 ID が apps に存在することを検証
func loadFileOpeners(entries []yamlFileOpenerEntry, apps []Application) ([]FileOpener, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	knownIDs := make(map[int]struct{}, len(apps))
	for _, a := range apps {
		knownIDs[a.ID] = struct{}{}
	}
	seen := make(map[string]int) // ext -> index in out
	out := make([]FileOpener, 0, len(entries))
	for i, e := range entries {
		ext := strings.ToLower(strings.TrimPrefix(e.Opener.Extension, "."))
		if ext == "" {
			return nil, fmt.Errorf("file_opener[%d]: extension must not be empty", i)
		}
		ids := make([]int, 0, len(e.Opener.Applications))
		for _, id := range e.Opener.Applications {
			if _, ok := knownIDs[id]; !ok {
				return nil, fmt.Errorf("file_opener[%d]: applications references unknown id %d", i, id)
			}
			ids = append(ids, id)
		}
		if e.Opener.DefaultApp != 0 {
			if _, ok := knownIDs[e.Opener.DefaultApp]; !ok {
				return nil, fmt.Errorf("file_opener[%d]: default_app references unknown id %d", i, e.Opener.DefaultApp)
			}
		}
		op := FileOpener{Extension: ext, ApplicationIDs: ids, DefaultApp: e.Opener.DefaultApp}
		if idx, ok := seen[ext]; ok {
			out[idx] = op
		} else {
			seen[ext] = len(out)
			out = append(out, op)
		}
	}
	return out, nil
}

// loadLayout は yaml の layout セクションを LayoutConfig に変換する。
// セクション欠落 / 該当 pane なし / 個別フィールドなし、いずれも nil ポインタとして扱う。
func loadLayout(yl *yamlLayout) LayoutConfig {
	if yl == nil {
		return LayoutConfig{}
	}
	return LayoutConfig{
		TaskListWidth:     yl.Main.TaskList.Width,
		TaskDetailHeight:  yl.Main.TaskDetail.Height,
		FileListHeight:    yl.Main.FileList.Height,
		FilePreviewHeight: yl.Main.FilePreview.Height,
	}
}

// loadCursor は yaml の cursor セクションを CursorState に変換する。
// セクション欠落 / 両フィールド 0 はゼロ値の CursorState (TaskID=0, StatusID=0) を返す。
// 復元の優先度 (task_id > status_id) は TUI 層側で解釈する。
func loadCursor(yc *yamlCursor) CursorState {
	if yc == nil {
		return CursorState{}
	}
	return CursorState{TaskID: yc.TaskID, StatusID: yc.StatusID}
}

// marshalCursor は CursorState を yaml 出力用にシリアライズする。
// 両フィールド 0 なら nil を返す (= cursor キーごと省略)。
func marshalCursor(cs CursorState) *yamlCursor {
	if cs.TaskID == 0 && cs.StatusID == 0 {
		return nil
	}
	return &yamlCursor{TaskID: cs.TaskID, StatusID: cs.StatusID}
}

// marshalLayout は LayoutConfig を yaml 出力用にシリアライズする。
// 4 値とも nil なら nil を返す (= layout キーごと省略)。
func marshalLayout(lc LayoutConfig) *yamlLayout {
	if lc.TaskListWidth == nil && lc.TaskDetailHeight == nil &&
		lc.FileListHeight == nil && lc.FilePreviewHeight == nil {
		return nil
	}
	return &yamlLayout{
		Main: yamlLayoutMain{
			TaskList:    yamlLayoutValue{Width: lc.TaskListWidth},
			TaskDetail:  yamlLayoutValue{Height: lc.TaskDetailHeight},
			FileList:    yamlLayoutValue{Height: lc.FileListHeight},
			FilePreview: yamlLayoutValue{Height: lc.FilePreviewHeight},
		},
	}
}

// loadStatuses は yaml の statuses をドメイン型に変換し、欠落・空・id 未採番に対する
// 自動補完を行う。第二戻り値は補完によりファイルへの書き戻しが必要かどうか。
func loadStatuses(entries []yamlStatusEntry) (task.StatusList, bool) {
	if len(entries) == 0 {
		return task.DefaultStatuses(), true
	}
	sl := make(task.StatusList, 0, len(entries))
	for _, e := range entries {
		sl = append(sl, task.Status{
			ID:        e.Status.ID,
			Sequence:  e.Status.Sequence,
			Label:     e.Status.Label,
			Color:     e.Status.Color,
			Collapsed: e.Status.Collapsed,
		})
	}
	assigned, changed := sl.AssignMissingIDs()
	return assigned, changed
}

// loadFieldDefs は yaml の fields をドメイン型に変換し、id<=0 / position<=0 / type 空への補完を行う。
// 第二戻り値は補完が起きたか。fields 欠落 / 空配列ともに空 FieldDefList を返す (タスクが参照してなければ valid)。
func loadFieldDefs(entries []yamlFieldEntry) (task.FieldDefList, bool) {
	if len(entries) == 0 {
		return task.FieldDefList{}, false
	}
	fl := make(task.FieldDefList, 0, len(entries))
	for _, e := range entries {
		ft := task.FieldType(e.Field.Type)
		if ft == "" {
			ft = task.FieldTypeText
		}
		fl = append(fl, task.FieldDef{
			ID:       e.Field.ID,
			Name:     e.Field.Name,
			Type:     ft,
			Position: e.Field.Position,
		})
	}
	assigned, changed := fl.AssignMissingIDsAndPositions()
	// type 空文字をデフォルト補完したかどうかも changed に含める
	for i := range entries {
		if entries[i].Field.Type == "" {
			changed = true
			break
		}
	}
	return assigned, changed
}

// loadTags は yaml の tags をドメイン型に変換する。
// id<=0 を採番し、第二戻り値は補完が起きたか。tags 欠落 / 空配列ともに空 TagList を返す。
func loadTags(entries []yamlTagEntry) (task.TagList, bool) {
	if len(entries) == 0 {
		return task.TagList{}, false
	}
	tl := make(task.TagList, 0, len(entries))
	for _, e := range entries {
		tl = append(tl, task.Tag{
			ID:    e.Tag.ID,
			Name:  e.Tag.Name,
			Color: e.Tag.Color,
		})
	}
	assigned, changed := tl.AssignMissingIDs()
	return assigned, changed
}

func loadTasks(entries []yamlEntry, statuses task.StatusList, defs task.FieldDefList, tags task.TagList) ([]task.Task, bool, error) {
	seen := make(map[int]struct{}, len(entries))
	tasks := make([]task.Task, 0, len(entries))
	changed := false
	for i, e := range entries {
		if e.Task.ID <= 0 {
			return nil, false, fmt.Errorf("tasks[%d]: invalid id %d", i, e.Task.ID)
		}
		if _, dup := seen[e.Task.ID]; dup {
			return nil, false, fmt.Errorf("tasks[%d]: duplicated id %d", i, e.Task.ID)
		}
		seen[e.Task.ID] = struct{}{}

		// task 内 fields をドメイン型へ。entry が無ければ nil のままにして
		// 「fields キー無し / 空配列」のラウンドトリップを安定させる。id<=0 は採番。
		var tfl task.TaskFieldList
		if len(e.Task.Fields) > 0 {
			tfl = make(task.TaskFieldList, 0, len(e.Task.Fields))
			for _, fe := range e.Task.Fields {
				tfl = append(tfl, task.TaskField{
					ID:      fe.Field.ID,
					FieldID: fe.Field.FieldID,
					Value:   fe.Field.Value,
				})
			}
		}
		var assignedTFL task.TaskFieldList
		if tfl != nil {
			out, tflChanged := tfl.AssignMissingIDs()
			assignedTFL = out
			if tflChanged {
				changed = true
			}
		}

		// tags は ID 配列をそのまま保持。Validate が tag 存在チェックを担当する。
		var taskTags []int
		if len(e.Task.Tags) > 0 {
			taskTags = make([]int, len(e.Task.Tags))
			copy(taskTags, e.Task.Tags)
		}

		// collapsed_dirs は順序を安定させるため Load 時に sort して保持する。
		// 既に存在しないディレクトリ relPath が混ざっていても TUI 側の flatten で
		// 自然に無視されるため、ロード時には検証しない。
		var collapsedDirs []string
		if len(e.Task.CollapsedDirs) > 0 {
			collapsedDirs = make([]string, len(e.Task.CollapsedDirs))
			copy(collapsedDirs, e.Task.CollapsedDirs)
			sort.Strings(collapsedDirs)
		}

		t := task.Task{
			ID:            e.Task.ID,
			Title:         e.Task.Title,
			StatusID:      e.Task.StatusID,
			ParentID:      e.Task.ParentID,
			Position:      e.Task.Position,
			Collapsed:     e.Task.Collapsed,
			CollapsedDirs: collapsedDirs,
			IsTrashBox:    e.Task.IsTrashBox,
			Tags:          taskTags,
			Fields:        assignedTFL,
		}
		if err := t.Validate(statuses, tags); err != nil {
			return nil, false, fmt.Errorf("tasks[%d]: %w", i, err)
		}
		if err := t.Fields.Validate(defs); err != nil {
			return nil, false, fmt.Errorf("tasks[%d]: %w", i, err)
		}
		tasks = append(tasks, t)
	}
	if err := validateParents(tasks); err != nil {
		return nil, false, err
	}
	if assignMissingPositions(tasks) {
		changed = true
	}
	return tasks, changed, nil
}

// assignMissingPositions は同じ ParentID を持つ兄弟内で position=0 のタスクに対し、
// その兄弟群の現在の max(position)+1 から始めて yaml 出現順に採番する。
// 1 件でも補完が発生したら true を返す (書き戻し用)。
func assignMissingPositions(tasks []task.Task) bool {
	maxByParent := make(map[int]int)
	for _, t := range tasks {
		if t.Position > maxByParent[t.ParentID] {
			maxByParent[t.ParentID] = t.Position
		}
	}
	changed := false
	for i := range tasks {
		if tasks[i].Position == 0 {
			maxByParent[tasks[i].ParentID]++
			tasks[i].Position = maxByParent[tasks[i].ParentID]
			changed = true
		}
	}
	return changed
}

// validateParents は parent_id の存在・循環の有無・ネスト深さを検証する。
// ネスト深さは task.MaxNestDepth まで許容する (深さ 0 = トップレベル)。
func validateParents(tasks []task.Task) error {
	idIndex := make(map[int]int, len(tasks))
	for i, t := range tasks {
		idIndex[t.ID] = i
	}
	for i, t := range tasks {
		if t.ParentID == 0 {
			continue
		}
		seen := map[int]bool{t.ID: true}
		depth := 0
		cur := t
		for cur.ParentID != 0 {
			if seen[cur.ParentID] {
				return fmt.Errorf("tasks[%d]: parent chain has cycle at id %d", i, cur.ParentID)
			}
			seen[cur.ParentID] = true
			depth++
			if depth > task.MaxNestDepth {
				return fmt.Errorf("tasks[%d]: nesting depth exceeds limit (%d)", i, task.MaxNestDepth)
			}
			pi, ok := idIndex[cur.ParentID]
			if !ok {
				return fmt.Errorf("tasks[%d]: parent_id %d does not exist", i, cur.ParentID)
			}
			cur = tasks[pi]
		}
	}
	return nil
}

// Save は LoadResult を yaml に書き出す。書き込みは atomicWrite で原子的に行われる。
func (r *YAMLRepository) Save(lr LoadResult) error {
	sortedStatuses := lr.Statuses.Sorted()
	statusEntries := make([]yamlStatusEntry, 0, len(sortedStatuses))
	for _, s := range sortedStatuses {
		statusEntries = append(statusEntries, yamlStatusEntry{
			Status: yamlStatus{
				ID:        s.ID,
				Sequence:  s.Sequence,
				Label:     s.Label,
				Color:     s.Color,
				Collapsed: s.Collapsed,
			},
		})
	}

	sortedFields := lr.Fields.Sorted()
	fieldEntries := make([]yamlFieldEntry, 0, len(sortedFields))
	for _, f := range sortedFields {
		fieldEntries = append(fieldEntries, yamlFieldEntry{
			Field: yamlField{
				ID:       f.ID,
				Type:     string(f.Type),
				Name:     f.Name,
				Position: f.Position,
			},
		})
	}

	sortedTags := lr.Tags.Sorted()
	tagEntries := make([]yamlTagEntry, 0, len(sortedTags))
	for _, tg := range sortedTags {
		tagEntries = append(tagEntries, yamlTagEntry{
			Tag: yamlTag{
				ID:    tg.ID,
				Name:  tg.Name,
				Color: tg.Color,
			},
		})
	}

	taskEntries := make([]yamlEntry, 0, len(lr.Tasks))
	for _, t := range lr.Tasks {
		// 各 task の fields は安定した出力のため id 昇順で出力する。
		tfields := make(task.TaskFieldList, len(t.Fields))
		copy(tfields, t.Fields)
		sort.SliceStable(tfields, func(i, j int) bool {
			return tfields[i].ID < tfields[j].ID
		})
		fieldEntriesPerTask := make([]yamlTaskFieldEntry, 0, len(tfields))
		for _, tf := range tfields {
			fieldEntriesPerTask = append(fieldEntriesPerTask, yamlTaskFieldEntry{
				Field: yamlTaskField{
					ID:      tf.ID,
					FieldID: tf.FieldID,
					Value:   tf.Value,
				},
			})
		}

		var tagsCopy []int
		if len(t.Tags) > 0 {
			tagsCopy = make([]int, len(t.Tags))
			copy(tagsCopy, t.Tags)
		}

		// collapsed_dirs は yaml 上の diff を安定させるため出力時にも sort してコピーする。
		var collapsedDirsCopy []string
		if len(t.CollapsedDirs) > 0 {
			collapsedDirsCopy = make([]string, len(t.CollapsedDirs))
			copy(collapsedDirsCopy, t.CollapsedDirs)
			sort.Strings(collapsedDirsCopy)
		}

		taskEntries = append(taskEntries, yamlEntry{
			Task: yamlTask{
				ID:            t.ID,
				Title:         t.Title,
				StatusID:      t.StatusID,
				ParentID:      t.ParentID,
				Position:      t.Position,
				Collapsed:     t.Collapsed,
				CollapsedDirs: collapsedDirsCopy,
				IsTrashBox:    t.IsTrashBox,
				Tags:          tagsCopy,
				Fields:        fieldEntriesPerTask,
			},
		})
	}
	appEntries := make([]yamlApplicationEntry, 0, len(lr.Config.Applications))
	for _, a := range lr.Config.Applications {
		appEntries = append(appEntries, yamlApplicationEntry{
			Application: yamlApplication(a),
		})
	}

	openerEntries := make([]yamlFileOpenerEntry, 0, len(lr.Config.FileOpeners))
	for _, op := range lr.Config.FileOpeners {
		idsCopy := make([]int, len(op.ApplicationIDs))
		copy(idsCopy, op.ApplicationIDs)
		openerEntries = append(openerEntries, yamlFileOpenerEntry{
			Opener: yamlFileOpener{
				Extension:    op.Extension,
				Applications: idsCopy,
				DefaultApp:   op.DefaultApp,
			},
		})
	}

	data, err := yaml.Marshal(yamlFile{
		Version:           CurrentSchemaVersion,
		Applications:      appEntries,
		DataBaseDirectory: lr.Config.DataBaseDirectory,
		Layout:            marshalLayout(lr.Config.Layout),
		Cursor:            marshalCursor(lr.Config.Cursor),
		FileOpener:        openerEntries,
		Statuses:          statusEntries,
		Fields:            fieldEntries,
		Tags:              tagEntries,
		Tasks:             taskEntries,
	})
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	return atomicWrite(r.Path, data)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".task-man-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ErrFileNotFound は Load 対象の yaml が存在しないことを示すセンチネルエラー。
var ErrFileNotFound = errors.New("tasks file not found")
