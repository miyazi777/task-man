package task

import (
	"errors"
	"testing"
)

func TestTaskValidate(t *testing.T) {
	cases := []struct {
		name    string
		task    Task
		wantErr error
	}{
		{"valid", Task{ID: 1, Title: "x", Status: StatusTodo}, nil},
		{"empty title", Task{ID: 1, Title: "", Status: StatusTodo}, ErrEmptyTitle},
		{"zero id", Task{ID: 0, Title: "x", Status: StatusTodo}, ErrInvalidID},
		{"negative id", Task{ID: -1, Title: "x", Status: StatusTodo}, ErrInvalidID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.task.Validate()
			if c.wantErr == nil && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("expected %v, got %v", c.wantErr, err)
			}
		})
	}
}

func TestTaskValidateInvalidStatus(t *testing.T) {
	tk := Task{ID: 1, Title: "x", Status: "bogus"}
	if err := tk.Validate(); err == nil {
		t.Error("expected error for invalid status")
	}
}
