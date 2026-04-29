package task

import "fmt"

type Status string

const (
	StatusTodo  Status = "todo"
	StatusDoing Status = "doing"
	StatusDone  Status = "done"
)

var statusOrder = []Status{StatusTodo, StatusDoing, StatusDone}

func (s Status) Next() Status {
	for i, v := range statusOrder {
		if v == s && i+1 < len(statusOrder) {
			return statusOrder[i+1]
		}
	}
	return s
}

func (s Status) Prev() Status {
	for i, v := range statusOrder {
		if v == s && i-1 >= 0 {
			return statusOrder[i-1]
		}
	}
	return s
}

func ParseStatus(v string) (Status, error) {
	for _, s := range statusOrder {
		if string(s) == v {
			return s, nil
		}
	}
	return "", fmt.Errorf("invalid status: %q", v)
}
