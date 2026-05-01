package task

import (
	"errors"
	"fmt"
	"sort"
)

// Status はユーザー定義のタスク状態を表す。tasks.yaml の statuses セクションでカスタマイズ可能。
type Status struct {
	ID        int
	Sequence  int
	Label     string
	Color     string // 例: "#fab387"。空文字なら表示側でフォールバック。
	Collapsed bool   // タスクリスト画面でのグループ折りたたみ状態 (永続化対象)
}

// StatusList は同一 yaml ファイル内で有効なステータス集合。
type StatusList []Status

var (
	ErrStatusEmptyLabel = errors.New("status label must not be empty")
	ErrStatusInvalidID  = errors.New("status id must be greater than 0")
)

// DefaultStatuses は statuses 未定義時に注入されるデフォルト集合。
// 色は styles.go の Catppuccin 系に揃える: todo=グレー / doing=オレンジ / done=グリーン。
func DefaultStatuses() StatusList {
	return StatusList{
		{ID: 1, Sequence: 1, Label: "todo", Color: "#6c7086"},
		{ID: 2, Sequence: 2, Label: "doing", Color: "#fab387"},
		{ID: 3, Sequence: 3, Label: "done", Color: "#a6e3a1"},
	}
}

// ByID は id に一致する status を返す。見つからなければ ok=false。
func (sl StatusList) ByID(id int) (Status, bool) {
	for _, s := range sl {
		if s.ID == id {
			return s, true
		}
	}
	return Status{}, false
}

// Sorted は sequence 昇順 (タイブレークは id 昇順) でコピーを返す。
func (sl StatusList) Sorted() StatusList {
	out := make(StatusList, len(sl))
	copy(out, sl)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sequence != out[j].Sequence {
			return out[i].Sequence < out[j].Sequence
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// NextID は currentID の次 (Sorted 上で +1) の status id を返す。端なら currentID を返す。
func (sl StatusList) NextID(currentID int) int {
	sorted := sl.Sorted()
	for i, s := range sorted {
		if s.ID == currentID && i+1 < len(sorted) {
			return sorted[i+1].ID
		}
	}
	return currentID
}

// PrevID は currentID の前 (Sorted 上で -1) の status id を返す。端なら currentID を返す。
func (sl StatusList) PrevID(currentID int) int {
	sorted := sl.Sorted()
	for i, s := range sorted {
		if s.ID == currentID && i-1 >= 0 {
			return sorted[i-1].ID
		}
	}
	return currentID
}

// AssignMissingIDs は id<=0 の要素に対し、既存最大 id+1 から連番で採番した新しいリストを返す。
// 元のリストは変更しない。第二戻り値は採番が発生したかどうか。
func (sl StatusList) AssignMissingIDs() (StatusList, bool) {
	out := make(StatusList, len(sl))
	copy(out, sl)
	max := 0
	for _, s := range out {
		if s.ID > max {
			max = s.ID
		}
	}
	changed := false
	for i := range out {
		if out[i].ID <= 0 {
			max++
			out[i].ID = max
			changed = true
		}
	}
	return out, changed
}

// Validate は id 重複・id<=0・label 空のチェックを行う。
func (sl StatusList) Validate() error {
	seen := make(map[int]struct{}, len(sl))
	for i, s := range sl {
		if s.ID <= 0 {
			return fmt.Errorf("statuses[%d]: %w", i, ErrStatusInvalidID)
		}
		if _, dup := seen[s.ID]; dup {
			return fmt.Errorf("statuses[%d]: duplicated id %d", i, s.ID)
		}
		seen[s.ID] = struct{}{}
		if s.Label == "" {
			return fmt.Errorf("statuses[%d]: %w", i, ErrStatusEmptyLabel)
		}
	}
	return nil
}
