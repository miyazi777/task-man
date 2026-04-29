package storage

import "github.com/miyazi777/task-man/internal/task"

type Repository interface {
	Load() ([]task.Task, error)
	Save([]task.Task) error
}
