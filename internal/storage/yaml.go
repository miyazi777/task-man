package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/miyazi777/task-man/internal/task"
)

type yamlTask struct {
	ID     int    `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
}

type yamlEntry struct {
	Task yamlTask `yaml:"task"`
}

type yamlFile struct {
	Tasks []yamlEntry `yaml:"tasks"`
}

type YAMLRepository struct {
	Path string
}

func NewYAMLRepository(path string) *YAMLRepository {
	return &YAMLRepository{Path: path}
}

func (r *YAMLRepository) Load() ([]task.Task, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", r.Path, err)
	}
	if len(data) == 0 {
		return []task.Task{}, nil
	}

	var f yamlFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", r.Path, err)
	}

	seen := make(map[int]struct{}, len(f.Tasks))
	tasks := make([]task.Task, 0, len(f.Tasks))
	for i, e := range f.Tasks {
		status, err := task.ParseStatus(e.Task.Status)
		if err != nil {
			return nil, fmt.Errorf("tasks[%d]: %w", i, err)
		}
		if e.Task.ID <= 0 {
			return nil, fmt.Errorf("tasks[%d]: invalid id %d", i, e.Task.ID)
		}
		if _, dup := seen[e.Task.ID]; dup {
			return nil, fmt.Errorf("tasks[%d]: duplicated id %d", i, e.Task.ID)
		}
		seen[e.Task.ID] = struct{}{}

		t := task.Task{
			ID:     e.Task.ID,
			Title:  e.Task.Title,
			Status: status,
		}
		if err := t.Validate(); err != nil {
			return nil, fmt.Errorf("tasks[%d]: %w", i, err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *YAMLRepository) Save(tasks []task.Task) error {
	entries := make([]yamlEntry, 0, len(tasks))
	for _, t := range tasks {
		entries = append(entries, yamlEntry{
			Task: yamlTask{
				ID:     t.ID,
				Title:  t.Title,
				Status: string(t.Status),
			},
		})
	}
	data, err := yaml.Marshal(yamlFile{Tasks: entries})
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
