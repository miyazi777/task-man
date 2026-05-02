package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

type yamlTask struct {
	ID        int    `yaml:"id"`
	Title     string `yaml:"title"`
	StatusID  int    `yaml:"status_id"`
	ParentID  int    `yaml:"parent_id,omitempty"`
	Position  int    `yaml:"position,omitempty"`
	Collapsed bool   `yaml:"collapsed,omitempty"`
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
	Tasks             []yamlEntry       `yaml:"tasks"`
}

type YAMLRepository struct {
	Path string
}

func NewYAMLRepository(path string) *YAMLRepository {
	return &YAMLRepository{Path: path}
}

func (r *YAMLRepository) Load() ([]task.Task, task.StatusList, AppConfig, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return nil, nil, AppConfig{}, fmt.Errorf("read %s: %w", r.Path, err)
	}

	var f yamlFile
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, nil, AppConfig{}, fmt.Errorf("parse %s: %w", r.Path, err)
		}
	}

	statuses, statusesChanged := loadStatuses(f.Statuses)
	if err := statuses.Validate(); err != nil {
		return nil, nil, AppConfig{}, err
	}

	tasks, tasksChanged, err := loadTasks(f.Tasks, statuses)
	if err != nil {
		return nil, nil, AppConfig{}, err
	}

	cfg := AppConfig{
		DataBaseDirectory: f.DataBaseDirectory,
		Editor:            f.Applications.Editor,
	}

	if statusesChanged || tasksChanged {
		if err := r.Save(tasks, statuses, cfg); err != nil {
			return nil, nil, AppConfig{}, fmt.Errorf("write back defaults: %w", err)
		}
	}

	return tasks, statuses, cfg, nil
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

func loadTasks(entries []yamlEntry, statuses task.StatusList) ([]task.Task, bool, error) {
	seen := make(map[int]struct{}, len(entries))
	tasks := make([]task.Task, 0, len(entries))
	for i, e := range entries {
		if e.Task.ID <= 0 {
			return nil, false, fmt.Errorf("tasks[%d]: invalid id %d", i, e.Task.ID)
		}
		if _, dup := seen[e.Task.ID]; dup {
			return nil, false, fmt.Errorf("tasks[%d]: duplicated id %d", i, e.Task.ID)
		}
		seen[e.Task.ID] = struct{}{}

		t := task.Task{
			ID:        e.Task.ID,
			Title:     e.Task.Title,
			StatusID:  e.Task.StatusID,
			ParentID:  e.Task.ParentID,
			Position:  e.Task.Position,
			Collapsed: e.Task.Collapsed,
		}
		if err := t.Validate(statuses); err != nil {
			return nil, false, fmt.Errorf("tasks[%d]: %w", i, err)
		}
		tasks = append(tasks, t)
	}
	if err := validateParents(tasks); err != nil {
		return nil, false, err
	}
	changed := assignMissingPositions(tasks)
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

func (r *YAMLRepository) Save(tasks []task.Task, statuses task.StatusList, cfg AppConfig) error {
	sortedStatuses := statuses.Sorted()
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

	taskEntries := make([]yamlEntry, 0, len(tasks))
	for _, t := range tasks {
		taskEntries = append(taskEntries, yamlEntry{
			Task: yamlTask{
				ID:        t.ID,
				Title:     t.Title,
				StatusID:  t.StatusID,
				ParentID:  t.ParentID,
				Position:  t.Position,
				Collapsed: t.Collapsed,
			},
		})
	}
	data, err := yaml.Marshal(yamlFile{
		Applications:      yamlApplications{Editor: cfg.Editor},
		DataBaseDirectory: cfg.DataBaseDirectory,
		Statuses:          statusEntries,
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
