package task

import "testing"

func TestStatusNext(t *testing.T) {
	cases := []struct {
		in   Status
		want Status
	}{
		{StatusTodo, StatusDoing},
		{StatusDoing, StatusDone},
		{StatusDone, StatusDone},
	}
	for _, c := range cases {
		if got := c.in.Next(); got != c.want {
			t.Errorf("Next(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStatusPrev(t *testing.T) {
	cases := []struct {
		in   Status
		want Status
	}{
		{StatusTodo, StatusTodo},
		{StatusDoing, StatusTodo},
		{StatusDone, StatusDoing},
	}
	for _, c := range cases {
		if got := c.in.Prev(); got != c.want {
			t.Errorf("Prev(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseStatus(t *testing.T) {
	valid := []string{"todo", "doing", "done"}
	for _, v := range valid {
		if _, err := ParseStatus(v); err != nil {
			t.Errorf("ParseStatus(%q) unexpected error: %v", v, err)
		}
	}
	invalid := []string{"", "TODO", "pending", "x"}
	for _, v := range invalid {
		if _, err := ParseStatus(v); err == nil {
			t.Errorf("ParseStatus(%q) expected error, got nil", v)
		}
	}
}
