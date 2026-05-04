package task

import (
	"errors"
	"strings"
	"testing"
)

func TestTagListSorted(t *testing.T) {
	tl := TagList{{ID: 3, Name: "c"}, {ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	out := tl.Sorted()
	want := []int{1, 2, 3}
	for i, w := range want {
		if out[i].ID != w {
			t.Errorf("[%d]: got id=%d, want %d", i, out[i].ID, w)
		}
	}
}

func TestTagListByID(t *testing.T) {
	tl := TagList{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	if tag, ok := tl.ByID(2); !ok || tag.Name != "b" {
		t.Errorf("ByID(2) = (%+v, %v)", tag, ok)
	}
	if _, ok := tl.ByID(99); ok {
		t.Error("ByID(99) should not be found")
	}
}

func TestTagListByName(t *testing.T) {
	tl := TagList{{ID: 1, Name: "テスト"}, {ID: 2, Name: "test"}}
	if tag, ok := tl.ByName("テスト"); !ok || tag.ID != 1 {
		t.Errorf("ByName(テスト) = (%+v, %v)", tag, ok)
	}
	if _, ok := tl.ByName("not-found"); ok {
		t.Error("unexpected match")
	}
}

func TestTagListAddTag(t *testing.T) {
	tl := TagList{{ID: 5, Name: "exists"}}
	out, id, err := tl.AddTag("new")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 6 {
		t.Errorf("newID=%d, want 6", id)
	}
	if len(out) != 2 || out[1].Name != "new" {
		t.Errorf("got %+v", out)
	}
	if _, _, err := tl.AddTag("exists"); !errors.Is(err, ErrTagDuplicateName) {
		t.Errorf("expected ErrTagDuplicateName, got %v", err)
	}
	if _, _, err := tl.AddTag(""); !errors.Is(err, ErrTagEmptyName) {
		t.Errorf("expected ErrTagEmptyName, got %v", err)
	}
	long := strings.Repeat("あ", MaxTagNameRunes+1)
	if _, _, err := tl.AddTag(long); !errors.Is(err, ErrTagNameTooLong) {
		t.Errorf("expected ErrTagNameTooLong, got %v", err)
	}
}

func TestTagListAssignMissingIDs(t *testing.T) {
	tl := TagList{{ID: 0, Name: "a"}, {ID: 5, Name: "b"}, {ID: 0, Name: "c"}}
	out, changed := tl.AssignMissingIDs()
	if !changed {
		t.Fatal("expected changed=true")
	}
	if out[0].ID != 6 || out[1].ID != 5 || out[2].ID != 7 {
		t.Errorf("got ids=%d,%d,%d", out[0].ID, out[1].ID, out[2].ID)
	}
}

func TestTagListValidate(t *testing.T) {
	cases := []struct {
		name    string
		tl      TagList
		wantErr error
	}{
		{"valid", TagList{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, nil},
		{"empty list", TagList{}, nil},
		{"id<=0", TagList{{ID: 0, Name: "a"}}, ErrTagInvalidID},
		{"empty name", TagList{{ID: 1, Name: ""}}, ErrTagEmptyName},
		{"name too long", TagList{{ID: 1, Name: strings.Repeat("a", MaxTagNameRunes+1)}}, ErrTagNameTooLong},
		{"duplicate name", TagList{{ID: 1, Name: "x"}, {ID: 2, Name: "x"}}, ErrTagDuplicateName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.tl.Validate()
			if c.wantErr == nil && err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("expected %v, got %v", c.wantErr, err)
			}
		})
	}
}

func TestTagListValidateDuplicateID(t *testing.T) {
	tl := TagList{{ID: 1, Name: "a"}, {ID: 1, Name: "b"}}
	if err := tl.Validate(); err == nil {
		t.Error("expected error for duplicated id")
	}
}

func TestValidateTagNameChars(t *testing.T) {
	if err := ValidateTagNameChars(""); err != nil {
		t.Errorf("empty allowed live: %v", err)
	}
	if err := ValidateTagNameChars(strings.Repeat("a", MaxTagNameRunes)); err != nil {
		t.Errorf("at-limit allowed: %v", err)
	}
	if err := ValidateTagNameChars(strings.Repeat("a", MaxTagNameRunes+1)); !errors.Is(err, ErrTagNameTooLong) {
		t.Errorf("expected too-long, got %v", err)
	}
	if err := ValidateTagNameChars("a\x00b"); err == nil {
		t.Error("expected NUL forbidden")
	}
}
