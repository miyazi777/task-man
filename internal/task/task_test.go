package task

import (
	"errors"
	"strings"
	"testing"
)

func TestTaskValidate(t *testing.T) {
	sl := DefaultStatuses()
	cases := []struct {
		name    string
		task    Task
		wantErr error
	}{
		{"valid", Task{ID: 1, Title: "x", StatusID: 1}, nil},
		{"empty title", Task{ID: 1, Title: "", StatusID: 1}, ErrEmptyTitle},
		{"zero id", Task{ID: 0, Title: "x", StatusID: 1}, ErrInvalidID},
		{"negative id", Task{ID: -1, Title: "x", StatusID: 1}, ErrInvalidID},
		{"title too long", Task{ID: 1, Title: strings.Repeat("あ", MaxTitleRunes+1), StatusID: 1}, ErrTitleTooLong},
		{"title at limit", Task{ID: 1, Title: strings.Repeat("あ", MaxTitleRunes), StatusID: 1}, nil},
		{"unknown status_id", Task{ID: 1, Title: "x", StatusID: 99}, ErrUnknownStatusID},
		{"zero status_id", Task{ID: 1, Title: "x", StatusID: 0}, ErrUnknownStatusID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.task.Validate(sl)
			if c.wantErr == nil && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("expected %v, got %v", c.wantErr, err)
			}
		})
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

func TestHasDuplicateTitle(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "alpha"},
		{ID: 2, Title: "beta"},
		{ID: 3, Title: "gamma"},
	}
	if !HasDuplicateTitle(tasks, "alpha", 0) {
		t.Error("alpha should be detected as duplicate when no exclusion")
	}
	if HasDuplicateTitle(tasks, "delta", 0) {
		t.Error("delta should NOT be detected as duplicate")
	}
	// excludeID=1 のときは alpha のレコード自身を除外するので重複ではない
	if HasDuplicateTitle(tasks, "alpha", 1) {
		t.Error("alpha with excludeID=1 should NOT be duplicate (self)")
	}
	// excludeID=2 のときに alpha は依然として重複
	if !HasDuplicateTitle(tasks, "alpha", 2) {
		t.Error("alpha with excludeID=2 should still be duplicate")
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
