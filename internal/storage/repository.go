package storage

import "github.com/miyazi777/task-man/internal/task"

type Repository interface {
	Load() ([]task.Task, task.StatusList, error)
	Save(tasks []task.Task, statuses task.StatusList) error
}
