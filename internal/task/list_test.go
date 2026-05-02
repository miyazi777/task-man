package task

import "testing"

func TestNextID(t *testing.T) {
	cases := []struct {
		name string
		in   []Task
		want int
	}{
		{"empty", nil, 1},
		{"single", []Task{{ID: 1}}, 2},
		{"sequential", []Task{{ID: 1}, {ID: 2}, {ID: 3}}, 4},
		{"with gaps", []Task{{ID: 1}, {ID: 3}, {ID: 7}}, 8},
		{"unordered", []Task{{ID: 5}, {ID: 2}, {ID: 9}, {ID: 1}}, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextID(c.in); got != c.want {
				t.Errorf("NextID(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestNextPosition(t *testing.T) {
	tasks := []Task{
		{ID: 1, ParentID: 0, Position: 1},
		{ID: 2, ParentID: 0, Position: 3},
		{ID: 3, ParentID: 1, Position: 1},
		{ID: 4, ParentID: 1, Position: 2},
		{ID: 5, ParentID: 2, Position: 1},
	}
	cases := []struct {
		name     string
		parentID int
		want     int
	}{
		{"root: max=3 -> 4", 0, 4},
		{"sub of 1: max=2 -> 3", 1, 3},
		{"sub of 2: max=1 -> 2", 2, 2},
		{"sub of unknown parent: 1", 99, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NextPosition(tasks, c.parentID); got != c.want {
				t.Errorf("NextPosition(parentID=%d) = %d, want %d", c.parentID, got, c.want)
			}
		})
	}
}
