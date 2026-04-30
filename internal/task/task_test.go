package task

import (
	"errors"
	"strings"
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
		{"title too long", Task{ID: 1, Title: strings.Repeat("あ", MaxTitleRunes+1), Status: StatusTodo}, ErrTitleTooLong},
		{"title at limit", Task{ID: 1, Title: strings.Repeat("あ", MaxTitleRunes), Status: StatusTodo}, nil},
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

func TestValidateTitleCharsForbidden(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want rune
	}{
		{"slash", "foo/bar", '/'},
		{"colon", "foo:bar", ':'},
		{"null", "foo\x00bar", 0},
		{"japanese with colon", "タスク:追加", ':'},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateTitleChars(c.s)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var fce *ForbiddenCharError
			if !errors.As(err, &fce) {
				t.Fatalf("expected *ForbiddenCharError, got %T", err)
			}
			if fce.Char != c.want {
				t.Errorf("expected char %q, got %q", c.want, fce.Char)
			}
		})
	}
}

func TestValidateTitleCharsLength(t *testing.T) {
	if err := ValidateTitleChars(""); err != nil {
		t.Errorf("empty string should be allowed in live validation, got %v", err)
	}
	if err := ValidateTitleChars(strings.Repeat("a", MaxTitleRunes)); err != nil {
		t.Errorf("at-limit should be allowed, got %v", err)
	}
	if err := ValidateTitleChars(strings.Repeat("a", MaxTitleRunes+1)); !errors.Is(err, ErrTitleTooLong) {
		t.Errorf("over-limit should return ErrTitleTooLong, got %v", err)
	}
	// 日本語でも rune 単位で 60 文字まで許容されること。
	if err := ValidateTitleChars(strings.Repeat("漢", MaxTitleRunes)); err != nil {
		t.Errorf("60 Japanese chars should be allowed, got %v", err)
	}
	if err := ValidateTitleChars(strings.Repeat("漢", MaxTitleRunes+1)); !errors.Is(err, ErrTitleTooLong) {
		t.Errorf("61 Japanese chars should return ErrTitleTooLong, got %v", err)
	}
}
