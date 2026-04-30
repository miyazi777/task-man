package storage

import "github.com/miyazi777/task-man/internal/task"

type Repository interface {
	Load() ([]task.Task, task.StatusList, AppConfig, error)
	Save(tasks []task.Task, statuses task.StatusList, cfg AppConfig) error
}
