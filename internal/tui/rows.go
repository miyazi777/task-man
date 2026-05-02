package tui

import (
	"sort"

	"github.com/miyazi777/task-man/internal/task"
)

// listRow はリスト画面の 1 行を表す。種別は status ヘッダ / task / 空区切り のいずれか。
type listRow struct {
	kind        rowKind
	statusID    int  // status / task の所属
	taskIndex   int  // task のときの m.tasks 上のインデックス
	depth       int  // task のときのネスト深さ (0=トップレベル, 1=サブタスク...)
	hasChildren bool // task が子タスクを持つか (マーカー表示用)
	collapsed   bool // task の collapsed 状態 (マーカー表示用)
}

type rowKind int

const (
	rowStatus rowKind = iota
	rowTask
	rowSeparator
)

// buildRows はステータスを sequence 逆順 (大きい順) で並べ、
// 各ステータス配下に該当タスクを position 昇順 (タイブレーカは id 昇順) で並べたフラットな行列を返す。
//
//   - statusCollapsed[statusID]==true のステータスはタスク行を出さない (ヘッダのみ)
//   - taskCollapsed[taskID]==true のタスクは子孫を非表示にする (自身は表示)
//   - セクション間には空行 (rowSeparator) を 1 行はさむ。最後のセクション後には入れない。
//   - サブタスクは親タスクの直下に再帰的にネスト表示する (兄弟内は position → id 順)。
//     親が同じ status グループに居ない場合はサブタスクをトップレベル扱い (depth=0) で並べる。
func buildRows(statuses task.StatusList, tasks []task.Task, statusCollapsed, taskCollapsed map[int]bool) []listRow {
	sorted := statuses.Sorted()

	// 子インデックスを position 昇順 → id 昇順で保持する。
	lessByPosID := func(a, b int) bool {
		if tasks[a].Position != tasks[b].Position {
			return tasks[a].Position < tasks[b].Position
		}
		return tasks[a].ID < tasks[b].ID
	}

	childrenByParent := make(map[int][]int)
	idToIndex := make(map[int]int, len(tasks))
	for j, t := range tasks {
		idToIndex[t.ID] = j
		if t.ParentID != 0 {
			childrenByParent[t.ParentID] = append(childrenByParent[t.ParentID], j)
		}
	}
	for pid := range childrenByParent {
		sort.SliceStable(childrenByParent[pid], func(a, b int) bool {
			return lessByPosID(childrenByParent[pid][a], childrenByParent[pid][b])
		})
	}

	var rows []listRow

	// emit はタスク j を depth で出力し、collapsed でなければ子孫を再帰的に出力する。
	var emit func(j, depth, statusID int)
	emit = func(j, depth, statusID int) {
		hasKids := len(childrenByParent[tasks[j].ID]) > 0
		isCollapsed := taskCollapsed[tasks[j].ID]
		rows = append(rows, listRow{
			kind:        rowTask,
			statusID:    statusID,
			taskIndex:   j,
			depth:       depth,
			hasChildren: hasKids,
			collapsed:   isCollapsed,
		})
		if isCollapsed {
			return
		}
		for _, ci := range childrenByParent[tasks[j].ID] {
			if tasks[ci].StatusID != statusID {
				continue
			}
			emit(ci, depth+1, statusID)
		}
	}

	for i := len(sorted) - 1; i >= 0; i-- {
		s := sorted[i]
		rows = append(rows, listRow{kind: rowStatus, statusID: s.ID})
		if !statusCollapsed[s.ID] {
			// status グループのトップレベル: ParentID==0 (もしくは親が別 status グループに居る孤立サブタスク)。
			var topLevel []int
			for j, t := range tasks {
				if t.StatusID != s.ID {
					continue
				}
				if t.ParentID != 0 {
					if pi, ok := idToIndex[t.ParentID]; ok && tasks[pi].StatusID == s.ID {
						continue
					}
				}
				topLevel = append(topLevel, j)
			}
			sort.SliceStable(topLevel, func(a, b int) bool {
				return lessByPosID(topLevel[a], topLevel[b])
			})
			for _, j := range topLevel {
				emit(j, 0, s.ID)
			}
		}
		if i > 0 {
			rows = append(rows, listRow{kind: rowSeparator})
		}
	}
	return rows
}

// taskHasChildren は id を親に持つタスクが tasks 内に存在するかを返す。
func taskHasChildren(tasks []task.Task, id int) bool {
	for _, t := range tasks {
		if t.ParentID == id {
			return true
		}
	}
	return false
}

// taskDepth は tasks の中で id のネスト深さ (0 = トップレベル) を返す。
// 親チェーンを辿って数える。循環や行方不明な親があれば打ち切る。
func taskDepth(tasks []task.Task, id int) int {
	idIndex := make(map[int]int, len(tasks))
	for i, t := range tasks {
		idIndex[t.ID] = i
	}
	idx, ok := idIndex[id]
	if !ok {
		return 0
	}
	cur := tasks[idx]
	depth := 0
	seen := map[int]bool{cur.ID: true}
	for cur.ParentID != 0 {
		pi, ok := idIndex[cur.ParentID]
		if !ok {
			break
		}
		if seen[cur.ParentID] {
			break
		}
		seen[cur.ParentID] = true
		depth++
		cur = tasks[pi]
	}
	return depth
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
