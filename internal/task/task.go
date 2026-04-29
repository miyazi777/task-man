package task

import "errors"

type Task struct {
	ID     int
	Title  string
	Status Status
}

var (
	ErrEmptyTitle  = errors.New("title must not be empty")
	ErrInvalidID   = errors.New("id must be greater than 0")
)

func (t Task) Validate() error {
	if t.Title == "" {
		return ErrEmptyTitle
	}
	if t.ID <= 0 {
		return ErrInvalidID
	}
	if _, err := ParseStatus(string(t.Status)); err != nil {
		return err
	}
	return nil
}
