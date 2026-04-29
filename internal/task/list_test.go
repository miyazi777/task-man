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
