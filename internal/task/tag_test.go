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
	out, id, err := tl.AddTag("new", "#abcdef")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != 6 {
		t.Errorf("newID=%d, want 6", id)
	}
	if len(out) != 2 || out[1].Name != "new" || out[1].Color != "#abcdef" {
		t.Errorf("got %+v", out)
	}
	if _, _, err := tl.AddTag("exists", ""); !errors.Is(err, ErrTagDuplicateName) {
		t.Errorf("expected ErrTagDuplicateName, got %v", err)
	}
	if _, _, err := tl.AddTag("", ""); !errors.Is(err, ErrTagEmptyName) {
		t.Errorf("expected ErrTagEmptyName, got %v", err)
	}
	long := strings.Repeat("あ", MaxTagNameRunes+1)
	if _, _, err := tl.AddTag(long, ""); !errors.Is(err, ErrTagNameTooLong) {
		t.Errorf("expected ErrTagNameTooLong, got %v", err)
	}
}

func TestTagListRenameByID(t *testing.T) {
	tl := TagList{{ID: 1, Name: "old"}, {ID: 2, Name: "other"}}
	out, err := tl.RenameByID(1, "new")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out[0].Name != "new" {
		t.Errorf("[0].Name=%q, want new", out[0].Name)
	}
	if tl[0].Name != "old" {
		t.Error("source must remain unchanged")
	}
	// 自分自身との一致は OK (no-op 同等)
	if _, err := tl.RenameByID(1, "old"); err != nil {
		t.Errorf("rename to same name should not error: %v", err)
	}
	// 他のタグと重複はエラー
	if _, err := tl.RenameByID(1, "other"); !errors.Is(err, ErrTagDuplicateName) {
		t.Errorf("expected ErrTagDuplicateName, got %v", err)
	}
	// 存在しない id
	if _, err := tl.RenameByID(99, "x"); err == nil {
		t.Error("expected error for missing id")
	}
	// 空 / 長すぎ
	if _, err := tl.RenameByID(1, ""); !errors.Is(err, ErrTagEmptyName) {
		t.Errorf("expected ErrTagEmptyName, got %v", err)
	}
	long := strings.Repeat("あ", MaxTagNameRunes+1)
	if _, err := tl.RenameByID(1, long); !errors.Is(err, ErrTagNameTooLong) {
		t.Errorf("expected ErrTagNameTooLong, got %v", err)
	}
}

func TestTagListDeleteByID(t *testing.T) {
	tl := TagList{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}, {ID: 3, Name: "c"}}
	out, err := tl.DeleteByID(2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	if out[0].ID != 1 || out[1].ID != 3 {
		t.Errorf("got ids=%d,%d, want 1,3", out[0].ID, out[1].ID)
	}
	// source unchanged
	if len(tl) != 3 {
		t.Error("source must remain unchanged")
	}
	if _, err := tl.DeleteByID(99); err == nil {
		t.Error("expected error for missing id")
	}
}

func TestTagListSetColorByID(t *testing.T) {
	tl := TagList{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	out, err := tl.SetColorByID(2, "#ff00ff")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out[1].Color != "#ff00ff" {
		t.Errorf("[1].Color=%q, want #ff00ff", out[1].Color)
	}
	// source unchanged
	if tl[1].Color != "" {
		t.Error("source must remain unchanged")
	}
	if _, err := tl.SetColorByID(99, "#000000"); err == nil {
		t.Error("expected error for missing id")
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
