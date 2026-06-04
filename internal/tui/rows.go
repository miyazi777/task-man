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
	depth       int  // task のときのネスト深さ (0=トップレベル)
	hasChildren bool // task が子タスクを持つか (マーカー表示用)
	collapsed   bool // task の collapsed 状態 (マーカー表示用)
}

type rowKind int

const (
	rowStatus rowKind = iota
	rowTask
	rowSeparator
)

// buildRows はステータスを sequence 昇順 (小さい順) で並べ (= yaml 上の sequence 順)、
// 各ステータス配下に該当タスクを position 昇順 (タイブレーカは id 昇順) で並べたフラットな行列を返す。
//
//   - showTrash=false: IsTrashBox=true のタスクを除外 (通常リスト)
//   - showTrash=true:  IsTrashBox=true のタスクのみを表示 (ゴミ箱ビュー)
//   - statusCollapsed[statusID]==true のステータスはタスク行を出さない (ヘッダのみ)
//   - taskCollapsed[taskID]==true のタスクは子孫を非表示にする (自身は表示)
//   - セクション間には空行 (rowSeparator) を 1 行はさむ。最後のセクション後には入れない。
//   - サブタスクは親タスクの直下に再帰的にネスト表示する (兄弟内は position → id 順)。
//     親が同じ status グループに居ない、または親のゴミ箱状態が異なる場合はサブタスクをトップレベル扱い (depth=0) で並べる。
func buildRows(statuses task.StatusList, tasks []task.Task, statusCollapsed, taskCollapsed map[int]bool, showTrash bool) []listRow {
	sorted := statuses.Sorted()

	// ビューに含まれるべき task インデックスかを判定 (IsTrashBox とビュー種別の一致)。
	visible := func(t task.Task) bool {
		return t.IsTrashBox == showTrash
	}

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
		if !visible(t) {
			continue
		}
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

	// emit は j 番タスクと、その全可視子孫を再帰的に行リストへ追加する。
	// 子孫は status_id を問わず親直下にネスト表示する (status はルートのみが管理する設計)。
	// statusID は所属グループ (描画上の親グループ) のステータス ID。
	var emit func(j, depth, statusID int)
	emit = func(j, depth, statusID int) {
		hasKids := false
		for _, ci := range childrenByParent[tasks[j].ID] {
			if visible(tasks[ci]) {
				hasKids = true
				break
			}
		}
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
			if !visible(tasks[ci]) {
				continue
			}
			emit(ci, depth+1, statusID)
		}
	}

	for i := 0; i < len(sorted); i++ {
		s := sorted[i]
		rows = append(rows, listRow{kind: rowStatus, statusID: s.ID})
		if !statusCollapsed[s.ID] {
			var topLevel []int
			for j, t := range tasks {
				if !visible(t) {
					continue
				}
				if t.StatusID != s.ID {
					continue
				}
				// 可視な親が居る場合は子として親に追従させ、ここでは top-level に積まない。
				// 親が存在しない / 親が非可視 (例: ゴミ箱) の場合のみ orphan として top-level 表示。
				if t.ParentID != 0 {
					if pi, ok := idToIndex[t.ParentID]; ok && visible(tasks[pi]) {
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
		if i < len(sorted)-1 {
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
// 見つからなければ from を返す。separator はスキップする。
func nextNavigable(rows []listRow, from int) int {
	for i := from + 1; i < len(rows); i++ {
		if isNavigable(rows[i].kind) {
			return i
		}
	}
	return from
}

// prevNavigable は from より前の (status か task の) 最後の行を返す。
// 見つからなければ from を返す。separator はスキップする。
func prevNavigable(rows []listRow, from int) int {
	for i := from - 1; i >= 0; i-- {
		if isNavigable(rows[i].kind) {
			return i
		}
	}
	return from
}

// firstNavigable は最初の navigable 行のインデックスを返す。空なら -1。
func firstNavigable(rows []listRow) int {
	for i, r := range rows {
		if isNavigable(r.kind) {
			return i
		}
	}
	return -1
}

func isNavigable(k rowKind) bool {
	return k == rowStatus || k == rowTask
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

// findRowForTaskID は taskID と一致するタスク行のインデックスを返す。見つからなければ -1。
// rows.taskIndex は tasks スライス上のインデックスなので、引き合わせのため tasks も受け取る。
// 起動時のカーソル復元 (storage.CursorState.TaskID → m.cursor) で利用する。
func findRowForTaskID(rows []listRow, tasks []task.Task, taskID int) int {
	if taskID == 0 {
		return -1
	}
	for i, r := range rows {
		if r.kind != rowTask {
			continue
		}
		if r.taskIndex < 0 || r.taskIndex >= len(tasks) {
			continue
		}
		if tasks[r.taskIndex].ID == taskID {
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
