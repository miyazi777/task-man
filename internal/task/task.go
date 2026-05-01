package task

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// MaxTitleRunes はタスクタイトルの最大文字数 (rune 単位)。
// Linux/macOS のファイル名上限を考慮し、日本語でも安全な値として 60 を採用。
const MaxTitleRunes = 60

// MaxNestDepth はサブタスクのネスト最大深さ (0 = トップレベル)。
// 値 4 のとき、トップレベル + 4 階層 = 計 5 階層まで許容する。
const MaxNestDepth = 4

type Task struct {
	ID        int
	Title     string
	StatusID  int
	ParentID  int  // 親タスクの ID。0 ならトップレベル。
	Collapsed bool // サブタスクをタスクリスト上で折りたたんでいるか (永続化対象)
}

var (
	ErrEmptyTitle      = errors.New("title must not be empty")
	ErrInvalidID       = errors.New("id must be greater than 0")
	ErrTitleTooLong    = fmt.Errorf("title must be at most %d characters", MaxTitleRunes)
	ErrUnknownStatusID = errors.New("status_id does not match any defined status")
)

// ForbiddenCharError は使用できない文字がタイトルに含まれていることを示すエラー。
type ForbiddenCharError struct {
	Char rune
}

func (e *ForbiddenCharError) Error() string {
	var name string
	switch e.Char {
	case 0:
		name = "null"
	case '/':
		name = "slash (/)"
	case ':':
		name = "colon (:)"
	default:
		name = fmt.Sprintf("'%c'", e.Char)
	}
	return name + " is not allowed"
}

// ValidateTitleChars はタイトル文字列の長さと禁止文字をチェックする。
// 空文字列はここではエラーとしない (入力途中の状態を許容するため)。
// 禁止文字: \0 (POSIX), / (POSIX パス区切り), : (macOS Finder のパス区切り扱い)。
func ValidateTitleChars(s string) error {
	if utf8.RuneCountInString(s) > MaxTitleRunes {
		return ErrTitleTooLong
	}
	for _, r := range s {
		switch r {
		case 0, '/', ':':
			return &ForbiddenCharError{Char: r}
		}
	}
	return nil
}

// Validate はタスク自身の整合性と、status_id が statuses に存在するかをチェックする。
func (t Task) Validate(statuses StatusList) error {
	if t.Title == "" {
		return ErrEmptyTitle
	}
	if err := ValidateTitleChars(t.Title); err != nil {
		return err
	}
	if t.ID <= 0 {
		return ErrInvalidID
	}
	if _, ok := statuses.ByID(t.StatusID); !ok {
		return fmt.Errorf("%w: %d", ErrUnknownStatusID, t.StatusID)
	}
	return nil
}
