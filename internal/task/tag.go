package task

import (
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"
)

// Tag はタスクに紐付ける単純なラベル。tasks.yaml の tags セクションで一元管理し、
// task.tags は ID 参照で関連付ける。
type Tag struct {
	ID    int
	Name  string
	Color string // 例: "#fab387"。空文字なら表示側でフォールバック (colorMuted)。
}

// TagList はファイル全体で有効なタグの集合。
type TagList []Tag

const (
	// MaxTagsPerTask は 1 タスクに付与できるタグ数の上限。
	MaxTagsPerTask = 5
	// MaxTagNameRunes は tag name の最大文字数 (rune 単位)。
	MaxTagNameRunes = 8
)

// タグ操作のバリデーションエラー。Tag の Validate / TagList の操作が返す。
var (
	ErrTagEmptyName     = errors.New("tag name must not be empty")
	ErrTagNameTooLong   = fmt.Errorf("tag name must be at most %d characters", MaxTagNameRunes)
	ErrTagDuplicateName = errors.New("tag name already exists")
	ErrTagInvalidID     = errors.New("tag id must be greater than 0")
	ErrTaskTooManyTags  = fmt.Errorf("task can have at most %d tags", MaxTagsPerTask)
	ErrTagUnknownID     = errors.New("task references unknown tag_id")
)

// TagNameForbiddenCharError は tag 名の禁止文字エラー。
type TagNameForbiddenCharError struct {
	Char rune
}

func (e *TagNameForbiddenCharError) Error() string {
	if e.Char == 0 {
		return "null is not allowed in tag name"
	}
	return fmt.Sprintf("'%c' is not allowed in tag name", e.Char)
}

// ValidateTagNameChars はライブ入力検証用。空文字は許容 (入力途中)。
// 8 rune 上限と NUL のみ禁止する。
func ValidateTagNameChars(s string) error {
	if utf8.RuneCountInString(s) > MaxTagNameRunes {
		return ErrTagNameTooLong
	}
	for _, r := range s {
		if r == 0 {
			return &TagNameForbiddenCharError{Char: r}
		}
	}
	return nil
}

// Sorted は id 昇順でコピーを返す。
func (tl TagList) Sorted() TagList {
	out := make(TagList, len(tl))
	copy(out, tl)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// ByID は id に一致する Tag を返す。
func (tl TagList) ByID(id int) (Tag, bool) {
	for _, t := range tl {
		if t.ID == id {
			return t, true
		}
	}
	return Tag{}, false
}

// ByName は name に一致する Tag を返す (重複チェック用、完全一致)。
func (tl TagList) ByName(name string) (Tag, bool) {
	for _, t := range tl {
		if t.Name == name {
			return t, true
		}
	}
	return Tag{}, false
}

// AssignMissingIDs は id<=0 の要素に対し既存 max+1 から連番採番した新しいリストを返す。
// 第二戻り値は採番が発生したか。
func (tl TagList) AssignMissingIDs() (TagList, bool) {
	out := make(TagList, len(tl))
	copy(out, tl)
	maxID := 0
	for _, t := range out {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	changed := false
	for i := range out {
		if out[i].ID <= 0 {
			maxID++
			out[i].ID = maxID
			changed = true
		}
	}
	return out, changed
}

// Validate は id<=0 / id 重複 / name 空 / name 長すぎ / name 重複を検出する。
func (tl TagList) Validate() error {
	seenID := make(map[int]struct{}, len(tl))
	seenName := make(map[string]struct{}, len(tl))
	for i, t := range tl {
		if t.ID <= 0 {
			return fmt.Errorf("tags[%d]: %w", i, ErrTagInvalidID)
		}
		if _, dup := seenID[t.ID]; dup {
			return fmt.Errorf("tags[%d]: duplicated id %d", i, t.ID)
		}
		seenID[t.ID] = struct{}{}
		if t.Name == "" {
			return fmt.Errorf("tags[%d]: %w", i, ErrTagEmptyName)
		}
		if err := ValidateTagNameChars(t.Name); err != nil {
			return fmt.Errorf("tags[%d]: %w", i, err)
		}
		if _, dup := seenName[t.Name]; dup {
			return fmt.Errorf("tags[%d]: %w: %q", i, ErrTagDuplicateName, t.Name)
		}
		seenName[t.Name] = struct{}{}
	}
	return nil
}

// AddTag は新規 Tag を追加した新しいリストを返す。新規 id は max+1 で採番。
// name が空・長すぎ・既存と重複の場合はエラー。第二戻り値は新規 id。
// color は #rrggbb の表示色 (空文字を許容、表示側でフォールバック)。
func (tl TagList) AddTag(name, color string) (TagList, int, error) {
	if name == "" {
		return nil, 0, ErrTagEmptyName
	}
	if err := ValidateTagNameChars(name); err != nil {
		return nil, 0, err
	}
	if _, dup := tl.ByName(name); dup {
		return nil, 0, fmt.Errorf("%w: %q", ErrTagDuplicateName, name)
	}
	maxID := 0
	for _, t := range tl {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	newID := maxID + 1
	out := make(TagList, len(tl), len(tl)+1)
	copy(out, tl)
	out = append(out, Tag{ID: newID, Name: name, Color: color})
	return out, newID, nil
}

// RenameByID は id を持つ Tag の Name を newName に置き換えた新しいリストを返す。
// name が空・長すぎ・他のタグと重複する場合はエラー (自分自身との一致は OK = no-op 同等)。
func (tl TagList) RenameByID(id int, newName string) (TagList, error) {
	if newName == "" {
		return nil, ErrTagEmptyName
	}
	if err := ValidateTagNameChars(newName); err != nil {
		return nil, err
	}
	// 重複チェック (自分自身は除外)。
	for _, t := range tl {
		if t.ID != id && t.Name == newName {
			return nil, fmt.Errorf("%w: %q", ErrTagDuplicateName, newName)
		}
	}
	out := make(TagList, len(tl))
	copy(out, tl)
	for i := range out {
		if out[i].ID == id {
			out[i].Name = newName
			return out, nil
		}
	}
	return nil, fmt.Errorf("tag id %d not found", id)
}

// DeleteByID は id を持つ Tag を削除した新しいリストを返す。
// 該当 id が無いときはエラー。
func (tl TagList) DeleteByID(id int) (TagList, error) {
	out := make(TagList, 0, len(tl))
	found := false
	for _, t := range tl {
		if t.ID == id {
			found = true
			continue
		}
		out = append(out, t)
	}
	if !found {
		return nil, fmt.Errorf("tag id %d not found", id)
	}
	return out, nil
}

// SetColorByID は id を持つ Tag の Color を新しい値に置き換えた新しいリストを返す。
// 該当 id が無いときはエラー。
func (tl TagList) SetColorByID(id int, color string) (TagList, error) {
	out := make(TagList, len(tl))
	copy(out, tl)
	for i := range out {
		if out[i].ID == id {
			out[i].Color = color
			return out, nil
		}
	}
	return nil, fmt.Errorf("tag id %d not found", id)
}
