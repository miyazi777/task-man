package tui

import (
	"github.com/miyazi777/task-man/internal/task"
)

// listRow はリスト画面の 1 行を表す。種別は status ヘッダ / task / 空区切り のいずれか。
type listRow struct {
	kind      rowKind
	statusID  int // status / task の所属
	taskIndex int // task のときの m.tasks 上のインデックス
}

type rowKind int

const (
	rowStatus rowKind = iota
	rowTask
	rowSeparator
)

// buildRows はステータスを sequence 逆順 (大きい順) で並べ、
// 各ステータス配下に該当タスクを yaml の出現順で並べたフラットな行列を返す。
//
//   - collapsed[statusID]==true のステータスはタスク行を出さない (ヘッダのみ)
//   - セクション間には空行 (rowSeparator) を 1 行はさむ。最後のセクション後には入れない。
func buildRows(statuses task.StatusList, tasks []task.Task, collapsed map[int]bool) []listRow {
	sorted := statuses.Sorted()
	var rows []listRow
	for i := len(sorted) - 1; i >= 0; i-- {
		s := sorted[i]
		rows = append(rows, listRow{kind: rowStatus, statusID: s.ID})
		if !collapsed[s.ID] {
			for j, t := range tasks {
				if t.StatusID == s.ID {
					rows = append(rows, listRow{kind: rowTask, statusID: s.ID, taskIndex: j})
				}
			}
		}
		if i > 0 {
			rows = append(rows, listRow{kind: rowSeparator})
		}
	}
	return rows
}

// nextNavigable は from より後ろの (status か task の) 最初の行を返す。
// 見つからなければ from を返す。
func nextNavigable(rows []listRow, from int) int {
	for i := from + 1; i < len(rows); i++ {
		if rows[i].kind != rowSeparator {
			return i
		}
	}
	return from
}

// prevNavigable は from より前の (status か task の) 最後の行を返す。
// 見つからなければ from を返す。
func prevNavigable(rows []listRow, from int) int {
	for i := from - 1; i >= 0; i-- {
		if rows[i].kind != rowSeparator {
			return i
		}
	}
	return from
}

// firstNavigable は最初の navigable 行のインデックスを返す。空なら -1。
func firstNavigable(rows []listRow) int {
	for i, r := range rows {
		if r.kind != rowSeparator {
			return i
		}
	}
	return -1
}

// findRowForTask は taskIndex を含む行のインデックスを返す。見つからなければ -1。
func findRowForTask(rows []listRow, taskIndex int) int {
	for i, r := range rows {
		if r.kind == rowTask && r.taskIndex == taskIndex {
			return i
		}
	}
	return -1
}

// findRowForStatus は statusID のヘッダ行を返す。見つからなければ -1。
func findRowForStatus(rows []listRow, statusID int) int {
	for i, r := range rows {
		if r.kind == rowStatus && r.statusID == statusID {
			return i
		}
	}
	return -1
}
