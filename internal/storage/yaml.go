package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type yamlTagEntry struct {
	Tag yamlTag `yaml:"tag"`
}

type yamlTask struct {
	ID         int                  `yaml:"id"`
	Title      string               `yaml:"title"`
	StatusID   int                  `yaml:"status_id"`
	ParentID   int                  `yaml:"parent_id,omitempty"`
	Position   int                  `yaml:"position,omitempty"`
	Collapsed  bool                 `yaml:"collapsed,omitempty"`
	IsTrashBox bool                 `yaml:"is_trash_box,omitempty"`
	Tags       []int                `yaml:"tags,omitempty"`
	Fields     []yamlTaskFieldEntry `yaml:"fields,omitempty"`
}

type yamlEntry struct {
	Task yamlTask `yaml:"task"`
}

type yamlApplications struct {
	Editor string `yaml:"editor,omitempty"`
}

type yamlFile struct {
	Applications      yamlApplications  `yaml:"applications,omitempty"`
	DataBaseDirectory string            `yaml:"data_base_directory,omitempty"`
	Statuses          []yamlStatusEntry `yaml:"statuses"`
	Fields            []yamlFieldEntry  `yaml:"fields,omitempty"`
	Tags              []yamlTagEntry    `yaml:"tags,omitempty"`
	Tasks             []yamlEntry       `yaml:"tasks"`
}

type YAMLRepository struct {
	Path string
}

func NewYAMLRepository(path string) *YAMLRepository {
	return &YAMLRepository{Path: path}
}

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

	cfg := AppConfig{
		DataBaseDirectory: f.DataBaseDirectory,
		Editor:            f.Applications.Editor,
	}

	lr := LoadResult{
		Tasks:    tasks,
		Statuses: statuses,
		Fields:   defs,
		Tags:     tags,
		Config:   cfg,
	}

	if statusesChanged || defsChanged || tagsChanged || tasksChanged {
		if err := r.Save(lr); err != nil {
			return LoadResult{}, fmt.Errorf("write back defaults: %w", err)
		}
	}

	return lr, nil
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
			ID:   e.Tag.ID,
			Name: e.Tag.Name,
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

		t := task.Task{
			ID:         e.Task.ID,
			Title:      e.Task.Title,
			StatusID:   e.Task.StatusID,
			ParentID:   e.Task.ParentID,
			Position:   e.Task.Position,
			Collapsed:  e.Task.Collapsed,
			IsTrashBox: e.Task.IsTrashBox,
			Tags:       taskTags,
			Fields:     assignedTFL,
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
				ID:   tg.ID,
				Name: tg.Name,
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

		taskEntries = append(taskEntries, yamlEntry{
			Task: yamlTask{
				ID:         t.ID,
				Title:      t.Title,
				StatusID:   t.StatusID,
				ParentID:   t.ParentID,
				Position:   t.Position,
				Collapsed:  t.Collapsed,
				IsTrashBox: t.IsTrashBox,
				Tags:       tagsCopy,
				Fields:     fieldEntriesPerTask,
			},
		})
	}
	data, err := yaml.Marshal(yamlFile{
		Applications:      yamlApplications{Editor: lr.Config.Editor},
		DataBaseDirectory: lr.Config.DataBaseDirectory,
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

var ErrFileNotFound = errors.New("tasks file not found")
