package task

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// FieldType は拡張項目の値の型。
type FieldType string

const (
	FieldTypeText FieldType = "text"
	FieldTypeDate FieldType = "date" // yyyy-mm-dd の文字列を保持する。空文字列も許容 (未設定状態)。
	FieldTypeURL  FieldType = "url"  // 絶対 URL 文字列を保持する。空文字列も許容 (未設定状態)。
)

// FieldDateLayout は date 型 value の文字列フォーマット。
const FieldDateLayout = "2006-01-02"

// AllFieldTypes は UI セレクターでの選択肢順序を兼ねる。
var AllFieldTypes = []FieldType{FieldTypeText, FieldTypeDate, FieldTypeURL}

// IsKnownFieldType は ft が定義済みの FieldType か判定する。
func IsKnownFieldType(ft FieldType) bool {
	for _, t := range AllFieldTypes {
		if t == ft {
			return true
		}
	}
	return false
}

const (
	// MaxFieldNameRunes は拡張項目 name の最大文字数 (rune 単位)。仕様: マルチバイト 18 文字。
	MaxFieldNameRunes = 18
	// MaxFieldTextValueRunes は text 型 value の最大文字数 (rune 単位)。仕様: 200。
	MaxFieldTextValueRunes = 200
	// MaxFieldURLValueRunes は url 型 value の最大文字数 (rune 単位)。仕様: 320。
	MaxFieldURLValueRunes = 320
)

var (
	ErrFieldEmptyName       = errors.New("field name must not be empty")
	ErrFieldNameTooLong     = fmt.Errorf("field name must be at most %d characters", MaxFieldNameRunes)
	ErrFieldValueTooLong    = fmt.Errorf("field value must be at most %d characters", MaxFieldTextValueRunes)
	ErrFieldUnknownType     = errors.New("field type is unknown")
	ErrFieldInvalidID       = errors.New("field id must be greater than 0")
	ErrFieldUnknownFieldID  = errors.New("task field references unknown field_id")
	ErrFieldInvalidPosition = errors.New("field position must be greater than 0")
)

// FieldNameForbiddenCharError は name の禁止文字エラー。Title と同じ禁止文字 (NUL/`/`/`:`)。
type FieldNameForbiddenCharError struct {
	Char rune
}

func (e *FieldNameForbiddenCharError) Error() string {
	switch e.Char {
	case 0:
		return "null is not allowed in field name"
	case '/':
		return "slash (/) is not allowed in field name"
	case ':':
		return "colon (:) is not allowed in field name"
	default:
		return fmt.Sprintf("'%c' is not allowed in field name", e.Char)
	}
}

// FieldValueForbiddenCharError は value の禁止文字エラー。NUL のみ禁止。
type FieldValueForbiddenCharError struct {
	Char rune
}

func (e *FieldValueForbiddenCharError) Error() string {
	if e.Char == 0 {
		return "null is not allowed in field value"
	}
	return fmt.Sprintf("'%c' is not allowed in field value", e.Char)
}

// ValidateFieldNameChars はライブ入力検証。空文字は許容 (入力途中)。
func ValidateFieldNameChars(s string) error {
	if utf8.RuneCountInString(s) > MaxFieldNameRunes {
		return ErrFieldNameTooLong
	}
	for _, r := range s {
		switch r {
		case 0, '/', ':':
			return &FieldNameForbiddenCharError{Char: r}
		}
	}
	return nil
}

// ValidateFieldTextValueChars はライブ入力検証。空文字は許容。NUL のみ禁止。
func ValidateFieldTextValueChars(s string) error {
	if utf8.RuneCountInString(s) > MaxFieldTextValueRunes {
		return ErrFieldValueTooLong
	}
	for _, r := range s {
		if r == 0 {
			return &FieldValueForbiddenCharError{Char: r}
		}
	}
	return nil
}

// ErrFieldInvalidDateValue は date 型 value が yyyy-mm-dd でパースできない場合のエラー。
var ErrFieldInvalidDateValue = errors.New("field date value must be yyyy-mm-dd")

// ValidateFieldDateValue は date 型 value を検証する。空文字列は未設定として許容する。
func ValidateFieldDateValue(s string) error {
	if s == "" {
		return nil
	}
	if _, err := time.Parse(FieldDateLayout, s); err != nil {
		return fmt.Errorf("%w: %q", ErrFieldInvalidDateValue, s)
	}
	return nil
}

// ErrFieldURLValueTooLong は url 型 value が 320 rune を超えた場合のエラー。
var ErrFieldURLValueTooLong = fmt.Errorf("field url value must be at most %d characters", MaxFieldURLValueRunes)

// ErrFieldInvalidURLValue は url 型 value が URL 形式 (scheme + host) として解釈できない場合のエラー。
var ErrFieldInvalidURLValue = errors.New("field url value must be a valid URL with scheme and host (e.g. https://example.com)")

// ValidateFieldURLValueChars は url 型 value のライブ入力検証。
// 320 rune 上限と NUL のみチェックし、入力途中の不完全な文字列は許容する。
func ValidateFieldURLValueChars(s string) error {
	if utf8.RuneCountInString(s) > MaxFieldURLValueRunes {
		return ErrFieldURLValueTooLong
	}
	for _, r := range s {
		if r == 0 {
			return &FieldValueForbiddenCharError{Char: r}
		}
	}
	return nil
}

// ValidateFieldURLValue は url 型 value の保存時検証。
// 空文字列は未設定として許容する。それ以外は scheme と host を持つ絶対 URL であることを要求する。
func ValidateFieldURLValue(s string) error {
	if s == "" {
		return nil
	}
	if err := ValidateFieldURLValueChars(s); err != nil {
		return err
	}
	// url.Parse は許容度が高く相対参照もパースしてしまうため、scheme/host の存在を明示的にチェックする。
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFieldInvalidURLValue, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return ErrFieldInvalidURLValue
	}
	// scheme は英字始まりの ASCII で構成されることを要求 (url.Parse がほぼ担保するが念のため)。
	if strings.ContainsAny(u.Scheme, " \t\r\n") {
		return ErrFieldInvalidURLValue
	}
	return nil
}

// FieldDef はトップレベル fields の 1 件 (スキーマ定義)。
type FieldDef struct {
	ID       int
	Name     string
	Type     FieldType
	Position int
}

// FieldDefList は fields スキーマ全体。
type FieldDefList []FieldDef

// Sorted は position 昇順 (タイブレークは id 昇順) でコピーを返す。
func (fl FieldDefList) Sorted() FieldDefList {
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ByID は id に一致する FieldDef を返す。
func (fl FieldDefList) ByID(id int) (FieldDef, bool) {
	for _, f := range fl {
		if f.ID == id {
			return f, true
		}
	}
	return FieldDef{}, false
}

// Validate は id 重複・id<=0・name 空・name 長すぎ・type 不明・position<=0 を検出する。
func (fl FieldDefList) Validate() error {
	seen := make(map[int]struct{}, len(fl))
	for i, f := range fl {
		if f.ID <= 0 {
			return fmt.Errorf("fields[%d]: %w", i, ErrFieldInvalidID)
		}
		if _, dup := seen[f.ID]; dup {
			return fmt.Errorf("fields[%d]: duplicated id %d", i, f.ID)
		}
		seen[f.ID] = struct{}{}
		if f.Name == "" {
			return fmt.Errorf("fields[%d]: %w", i, ErrFieldEmptyName)
		}
		if err := ValidateFieldNameChars(f.Name); err != nil {
			return fmt.Errorf("fields[%d]: %w", i, err)
		}
		if !IsKnownFieldType(f.Type) {
			return fmt.Errorf("fields[%d]: %w: %q", i, ErrFieldUnknownType, f.Type)
		}
		if f.Position <= 0 {
			return fmt.Errorf("fields[%d]: %w", i, ErrFieldInvalidPosition)
		}
	}
	return nil
}

// AssignMissingIDsAndPositions は id<=0 / position<=0 の field を採番し直す。
// 元のリストは変更せず、新しいリストを返す。第二戻り値は補完が起きたか。
func (fl FieldDefList) AssignMissingIDsAndPositions() (FieldDefList, bool) {
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	maxID := 0
	maxPos := 0
	for _, f := range out {
		if f.ID > maxID {
			maxID = f.ID
		}
		if f.Position > maxPos {
			maxPos = f.Position
		}
	}
	changed := false
	for i := range out {
		if out[i].ID <= 0 {
			maxID++
			out[i].ID = maxID
			changed = true
		}
		if out[i].Position <= 0 {
			maxPos++
			out[i].Position = maxPos
			changed = true
		}
		if out[i].Type == "" {
			out[i].Type = FieldTypeText
			changed = true
		}
	}
	return out, changed
}

// AddDef は新規 FieldDef を末尾 (position = max+1) に追加した新しいリストを返す。
// id は既存 max+1 で採番。第二戻り値は新規 id。
func (fl FieldDefList) AddDef(name string, ft FieldType) (FieldDefList, int, error) {
	if name == "" {
		return nil, 0, ErrFieldEmptyName
	}
	if err := ValidateFieldNameChars(name); err != nil {
		return nil, 0, err
	}
	if !IsKnownFieldType(ft) {
		return nil, 0, fmt.Errorf("%w: %q", ErrFieldUnknownType, ft)
	}
	maxID := 0
	maxPos := 0
	for _, f := range fl {
		if f.ID > maxID {
			maxID = f.ID
		}
		if f.Position > maxPos {
			maxPos = f.Position
		}
	}
	newID := maxID + 1
	newPos := maxPos + 1
	out := make(FieldDefList, len(fl), len(fl)+1)
	copy(out, fl)
	out = append(out, FieldDef{ID: newID, Name: name, Type: ft, Position: newPos})
	return out, newID, nil
}

// RenameByID は id を持つ FieldDef の Name を newName に置き換えた新しいリストを返す。
func (fl FieldDefList) RenameByID(id int, newName string) (FieldDefList, error) {
	if newName == "" {
		return nil, ErrFieldEmptyName
	}
	if err := ValidateFieldNameChars(newName); err != nil {
		return nil, err
	}
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	for i := range out {
		if out[i].ID == id {
			out[i].Name = newName
			return out, nil
		}
	}
	return nil, fmt.Errorf("field id %d not found", id)
}

// DeleteByID は id を持つ FieldDef を削除し、残った position を 1..N に振り直した新しいリストを返す。
func (fl FieldDefList) DeleteByID(id int) (FieldDefList, error) {
	idx := -1
	sorted := fl.Sorted()
	for i, f := range sorted {
		if f.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("field id %d not found", id)
	}
	out := make(FieldDefList, 0, len(sorted)-1)
	for _, f := range sorted {
		if f.ID == id {
			continue
		}
		out = append(out, f)
	}
	for i := range out {
		out[i].Position = i + 1
	}
	return out, nil
}

// MoveUp は id を持つ FieldDef を Sorted 順で 1 つ上に移動した新しいリストを返す。
func (fl FieldDefList) MoveUp(id int) FieldDefList {
	sorted := fl.Sorted()
	idx := -1
	for i, f := range sorted {
		if f.ID == id {
			idx = i
			break
		}
	}
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	if idx <= 0 {
		return out
	}
	ids := make([]int, len(sorted))
	for i, f := range sorted {
		ids[i] = f.ID
	}
	ids[idx-1], ids[idx] = ids[idx], ids[idx-1]
	return fl.resequenceByOrder(ids)
}

// MoveDown は id を持つ FieldDef を Sorted 順で 1 つ下に移動した新しいリストを返す。
func (fl FieldDefList) MoveDown(id int) FieldDefList {
	sorted := fl.Sorted()
	idx := -1
	for i, f := range sorted {
		if f.ID == id {
			idx = i
			break
		}
	}
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	if idx == -1 || idx >= len(sorted)-1 {
		return out
	}
	ids := make([]int, len(sorted))
	for i, f := range sorted {
		ids[i] = f.ID
	}
	ids[idx], ids[idx+1] = ids[idx+1], ids[idx]
	return fl.resequenceByOrder(ids)
}

// resequenceByOrder は orderedIDs の並び順で position を 1..N に振り直す。
func (fl FieldDefList) resequenceByOrder(orderedIDs []int) FieldDefList {
	out := make(FieldDefList, len(fl))
	copy(out, fl)
	rank := make(map[int]int, len(orderedIDs))
	for i, id := range orderedIDs {
		rank[id] = i + 1
	}
	for i := range out {
		if r, ok := rank[out[i].ID]; ok {
			out[i].Position = r
		}
	}
	return out
}

// TaskField は単一タスク内の拡張項目インスタンス (値ホルダー)。
type TaskField struct {
	ID      int    // task.fields 内で一意
	FieldID int    // FieldDef.ID への参照
	Value   string
}

// TaskFieldList は task が持つ拡張項目値の集合。
type TaskFieldList []TaskField

// ByFieldID は field_id に一致する TaskField を返す。
func (tfl TaskFieldList) ByFieldID(fieldID int) (TaskField, bool) {
	for _, f := range tfl {
		if f.FieldID == fieldID {
			return f, true
		}
	}
	return TaskField{}, false
}

// SetValue は fieldID に対する value を更新する。既存があれば上書き、無ければ末尾に追加。
// 追加時の id は既存 max+1 で採番。元のリストは変更せず新しいリストを返す。
func (tfl TaskFieldList) SetValue(fieldID int, value string) TaskFieldList {
	out := make(TaskFieldList, len(tfl))
	copy(out, tfl)
	for i := range out {
		if out[i].FieldID == fieldID {
			out[i].Value = value
			return out
		}
	}
	maxID := 0
	for _, f := range out {
		if f.ID > maxID {
			maxID = f.ID
		}
	}
	out = append(out, TaskField{ID: maxID + 1, FieldID: fieldID, Value: value})
	return out
}

// RemoveByFieldID は field_id を持つ TaskField を削除した新しいリストを返す。
func (tfl TaskFieldList) RemoveByFieldID(fieldID int) TaskFieldList {
	out := make(TaskFieldList, 0, len(tfl))
	for _, f := range tfl {
		if f.FieldID == fieldID {
			continue
		}
		out = append(out, f)
	}
	return out
}

// AssignMissingIDs は id<=0 の TaskField に対し既存 max+1 から採番した新しいリストを返す。
// 第二戻り値は補完が起きたか。
func (tfl TaskFieldList) AssignMissingIDs() (TaskFieldList, bool) {
	out := make(TaskFieldList, len(tfl))
	copy(out, tfl)
	maxID := 0
	for _, f := range out {
		if f.ID > maxID {
			maxID = f.ID
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

// Validate は id 重複・id<=0・field_id が defs に存在するか・value 長さを検証する。
// type は defs に紐づく FieldDef の型でバリデートする。
func (tfl TaskFieldList) Validate(defs FieldDefList) error {
	seen := make(map[int]struct{}, len(tfl))
	seenFieldID := make(map[int]struct{}, len(tfl))
	for i, f := range tfl {
		if f.ID <= 0 {
			return fmt.Errorf("fields[%d]: %w", i, ErrFieldInvalidID)
		}
		if _, dup := seen[f.ID]; dup {
			return fmt.Errorf("fields[%d]: duplicated id %d", i, f.ID)
		}
		seen[f.ID] = struct{}{}
		if _, dup := seenFieldID[f.FieldID]; dup {
			return fmt.Errorf("fields[%d]: duplicated field_id %d", i, f.FieldID)
		}
		seenFieldID[f.FieldID] = struct{}{}
		def, ok := defs.ByID(f.FieldID)
		if !ok {
			return fmt.Errorf("fields[%d]: %w: %d", i, ErrFieldUnknownFieldID, f.FieldID)
		}
		switch def.Type {
		case FieldTypeText:
			if err := ValidateFieldTextValueChars(f.Value); err != nil {
				return fmt.Errorf("fields[%d]: %w", i, err)
			}
		case FieldTypeDate:
			if err := ValidateFieldDateValue(f.Value); err != nil {
				return fmt.Errorf("fields[%d]: %w", i, err)
			}
		case FieldTypeURL:
			if err := ValidateFieldURLValue(f.Value); err != nil {
				return fmt.Errorf("fields[%d]: %w", i, err)
			}
		}
	}
	return nil
}
