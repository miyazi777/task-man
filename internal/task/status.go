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

// RenameByID は id を持つ status の Label を newLabel に置き換えた新しい StatusList を返す。
// 該当 id が無い、または label が空のときはエラー。元のリストは変更しない。
func (sl StatusList) RenameByID(id int, newLabel string) (StatusList, error) {
	if newLabel == "" {
		return nil, ErrStatusEmptyLabel
	}
	out := make(StatusList, len(sl))
	copy(out, sl)
	for i := range out {
		if out[i].ID == id {
			out[i].Label = newLabel
			return out, nil
		}
	}
	return nil, fmt.Errorf("status id %d not found", id)
}

// SetColorByID は id を持つ status の Color を newColor に置き換えた新しい StatusList を返す。
// 該当 id が無いときはエラー。色フォーマットの妥当性検証は呼び出し側責任。
func (sl StatusList) SetColorByID(id int, newColor string) (StatusList, error) {
	out := make(StatusList, len(sl))
	copy(out, sl)
	for i := range out {
		if out[i].ID == id {
			out[i].Color = newColor
			return out, nil
		}
	}
	return nil, fmt.Errorf("status id %d not found", id)
}

// InsertAt は sl を Sorted 順とみなし、insertIdx の位置 (0..len(sl)) に label/color の
// 新規 status を挿入する。新規 id は既存最大 +1 で採番し、Sequence は 1..N で振り直す。
// 戻り値は (新しい StatusList, 新規 status の id, error)。
func (sl StatusList) InsertAt(insertIdx int, label, color string) (StatusList, int, error) {
	if label == "" {
		return nil, 0, ErrStatusEmptyLabel
	}
	if insertIdx < 0 {
		insertIdx = 0
	}
	if insertIdx > len(sl) {
		insertIdx = len(sl)
	}
	max := 0
	for _, s := range sl {
		if s.ID > max {
			max = s.ID
		}
	}
	newID := max + 1
	newStatus := Status{ID: newID, Label: label, Color: color}

	sorted := sl.Sorted()
	merged := make(StatusList, 0, len(sorted)+1)
	merged = append(merged, sorted[:insertIdx]...)
	merged = append(merged, newStatus)
	merged = append(merged, sorted[insertIdx:]...)
	for i := range merged {
		merged[i].Sequence = i + 1
	}
	return merged, newID, nil
}

// MoveStatusUp は id を持つ status を Sorted 順で 1 つ上に移動した新しい StatusList を返す。
// 既に先頭または id が無い場合はコピーを返すだけ。
func (sl StatusList) MoveStatusUp(id int) StatusList {
	sorted := sl.Sorted()
	idx := -1
	for i, s := range sorted {
		if s.ID == id {
			idx = i
			break
		}
	}
	out := make(StatusList, len(sl))
	copy(out, sl)
	if idx <= 0 {
		return out
	}
	ids := make([]int, len(sorted))
	for i, s := range sorted {
		ids[i] = s.ID
	}
	ids[idx-1], ids[idx] = ids[idx], ids[idx-1]
	return sl.ResequenceByOrder(ids)
}

// MoveStatusDown は id を持つ status を Sorted 順で 1 つ下に移動した新しい StatusList を返す。
// 既に末尾または id が無い場合はコピーを返すだけ。
func (sl StatusList) MoveStatusDown(id int) StatusList {
	sorted := sl.Sorted()
	idx := -1
	for i, s := range sorted {
		if s.ID == id {
			idx = i
			break
		}
	}
	out := make(StatusList, len(sl))
	copy(out, sl)
	if idx == -1 || idx >= len(sorted)-1 {
		return out
	}
	ids := make([]int, len(sorted))
	for i, s := range sorted {
		ids[i] = s.ID
	}
	ids[idx], ids[idx+1] = ids[idx+1], ids[idx]
	return sl.ResequenceByOrder(ids)
}

// ResequenceByOrder は orderedIDs の並び順で sequence を 1..N に振り直した新しい StatusList を返す。
// orderedIDs に含まれない status はそのまま末尾扱い (sequence は付与されない) になるが、
// 通常は全 status を含めて呼ぶ前提。
func (sl StatusList) ResequenceByOrder(orderedIDs []int) StatusList {
	out := make(StatusList, len(sl))
	copy(out, sl)
	rank := make(map[int]int, len(orderedIDs))
	for i, id := range orderedIDs {
		rank[id] = i + 1
	}
	for i := range out {
		if r, ok := rank[out[i].ID]; ok {
			out[i].Sequence = r
		}
	}
	return out
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
