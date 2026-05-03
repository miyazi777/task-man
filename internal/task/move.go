package task

import "sort"

// taskIndexByID は id を持つタスクのスライスインデックスを返す。見つからなければ -1。
func taskIndexByID(tasks []Task, id int) int {
	for i, t := range tasks {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// taskNestDepth は id のネスト深さ (0 = top-level) を返す。親チェーンを辿って計測する。
func taskNestDepth(tasks []Task, id int) int {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return 0
	}
	depth := 0
	cur := tasks[idx]
	seen := map[int]bool{cur.ID: true}
	for cur.ParentID != 0 {
		if seen[cur.ParentID] {
			break
		}
		pi := taskIndexByID(tasks, cur.ParentID)
		if pi == -1 {
			break
		}
		seen[cur.ParentID] = true
		depth++
		cur = tasks[pi]
	}
	return depth
}

// subtreeRelDepth は id を起点とした最深子孫までの相対深さを返す (0 = 子孫なし)。
func subtreeRelDepth(tasks []Task, id int) int {
	children := map[int][]int{}
	for i, t := range tasks {
		children[t.ParentID] = append(children[t.ParentID], i)
	}
	var visit func(taskID, rel int) int
	visit = func(taskID, rel int) int {
		best := rel
		for _, ci := range children[taskID] {
			d := visit(tasks[ci].ID, rel+1)
			if d > best {
				best = d
			}
		}
		return best
	}
	return visit(id, 0)
}

// peerIndexes は parentID/statusID を持つ兄弟タスクのインデックスを
// position 昇順 (タイブレーカは id 昇順) で返す。
// parentID == 0 (top-level) のときは statusID で絞り込む (画面上の同じステータスグループ)。
// parentID != 0 のときは statusID 引数を無視して同じ親の子をすべて返す。
func peerIndexes(tasks []Task, parentID, statusID int) []int {
	var idxs []int
	for i, t := range tasks {
		if t.ParentID != parentID {
			continue
		}
		if parentID == 0 && t.StatusID != statusID {
			continue
		}
		idxs = append(idxs, i)
	}
	sortByPositionID(tasks, idxs)
	return idxs
}

func sortByPositionID(tasks []Task, idxs []int) {
	sort.SliceStable(idxs, func(a, b int) bool {
		ta, tb := tasks[idxs[a]], tasks[idxs[b]]
		if ta.Position != tb.Position {
			return ta.Position < tb.Position
		}
		return ta.ID < tb.ID
	})
}

// renumberPositions は idxs の順序で position を 1..N に振り直す。
func renumberPositions(tasks []Task, idxs []int) {
	for i, j := range idxs {
		tasks[j].Position = i + 1
	}
}

// setSubtreeStatusID は id のタスクおよびその全子孫の status_id を newStatusID に書き換える。
// 移動操作でステータスをまたぐ場合に使用する。
func setSubtreeStatusID(tasks []Task, id, newStatusID int) {
	children := map[int][]int{}
	for i, t := range tasks {
		children[t.ParentID] = append(children[t.ParentID], i)
	}
	var visit func(taskID int)
	visit = func(taskID int) {
		idx := taskIndexByID(tasks, taskID)
		if idx == -1 {
			return
		}
		tasks[idx].StatusID = newStatusID
		for _, ci := range children[taskID] {
			visit(tasks[ci].ID)
		}
	}
	visit(id)
}

// MoveTaskUp は id を兄弟内で 1 つ上へ移動する。先頭にいて top-level の場合は視覚的に上のステータス
// (= sequence が小さい方。rows.go は sequence 昇順で描画) の末尾へ移動する (子孫の status_id も追従)。
// 先頭・top-level かつそのステータスが無い、または非 top-level で先頭の場合は no-op。
// 子孫は parent_id 関係を保つので暗黙的に一緒に移動する。
func MoveTaskUp(tasks []Task, statuses StatusList, id int) []Task {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return tasks
	}
	t := tasks[idx]
	peers := peerIndexes(tasks, t.ParentID, t.StatusID)
	pos := indexOf(peers, idx)
	if pos == -1 {
		return tasks
	}
	if pos > 0 {
		peers[pos-1], peers[pos] = peers[pos], peers[pos-1]
		renumberPositions(tasks, peers)
		return tasks
	}
	if t.ParentID != 0 {
		return tasks
	}
	// 視覚的に上 = sequence が小さいステータス (rows.go は sequence 昇順で描画)。
	upperStatusID, ok := neighborStatusID(statuses, t.StatusID, -1)
	if !ok {
		return tasks
	}
	oldStatusID := t.StatusID
	setSubtreeStatusID(tasks, id, upperStatusID)
	// 元のステータスグループから idx が抜けたぶんを詰める。
	oldPeers := peerIndexes(tasks, 0, oldStatusID)
	renumberPositions(tasks, oldPeers)
	// 新しいステータスグループの末尾に追加。
	newPeers := peerIndexes(tasks, 0, upperStatusID)
	var ordered []int
	for _, j := range newPeers {
		if j == idx {
			continue
		}
		ordered = append(ordered, j)
	}
	ordered = append(ordered, idx)
	renumberPositions(tasks, ordered)
	return tasks
}

// MoveTaskDown は id を兄弟内で 1 つ下へ移動する。末尾にいて top-level の場合は視覚的に下のステータス
// (= sequence が大きい方) の先頭へ移動する (子孫の status_id も追従)。
func MoveTaskDown(tasks []Task, statuses StatusList, id int) []Task {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return tasks
	}
	t := tasks[idx]
	peers := peerIndexes(tasks, t.ParentID, t.StatusID)
	pos := indexOf(peers, idx)
	if pos == -1 {
		return tasks
	}
	if pos < len(peers)-1 {
		peers[pos], peers[pos+1] = peers[pos+1], peers[pos]
		renumberPositions(tasks, peers)
		return tasks
	}
	if t.ParentID != 0 {
		return tasks
	}
	// 視覚的に下 = sequence が大きいステータス。
	lowerStatusID, ok := neighborStatusID(statuses, t.StatusID, +1)
	if !ok {
		return tasks
	}
	oldStatusID := t.StatusID
	setSubtreeStatusID(tasks, id, lowerStatusID)
	oldPeers := peerIndexes(tasks, 0, oldStatusID)
	renumberPositions(tasks, oldPeers)
	newPeers := peerIndexes(tasks, 0, lowerStatusID)
	ordered := []int{idx}
	for _, j := range newPeers {
		if j == idx {
			continue
		}
		ordered = append(ordered, j)
	}
	renumberPositions(tasks, ordered)
	return tasks
}

// IndentTask は id を直前の兄弟の子にする (= 1 階層深くする)。直前の兄弟が無い、または深さ上限を
// 超える場合は no-op。新しい親の子末尾に追加する。
func IndentTask(tasks []Task, id int) []Task {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return tasks
	}
	t := tasks[idx]
	peers := peerIndexes(tasks, t.ParentID, t.StatusID)
	pos := indexOf(peers, idx)
	if pos <= 0 {
		return tasks
	}
	prevSibIdx := peers[pos-1]
	newParentID := tasks[prevSibIdx].ID
	if taskNestDepth(tasks, newParentID)+1+subtreeRelDepth(tasks, id) > MaxNestDepth {
		return tasks
	}
	var oldOrder []int
	for _, j := range peers {
		if j == idx {
			continue
		}
		oldOrder = append(oldOrder, j)
	}
	renumberPositions(tasks, oldOrder)
	tasks[idx].ParentID = newParentID
	var newSibs []int
	for i, tt := range tasks {
		if tt.ParentID == newParentID {
			newSibs = append(newSibs, i)
		}
	}
	sortByPositionID(tasks, newSibs)
	var ordered []int
	for _, j := range newSibs {
		if j == idx {
			continue
		}
		ordered = append(ordered, j)
	}
	ordered = append(ordered, idx)
	renumberPositions(tasks, ordered)
	return tasks
}

// OutdentTask は id を 1 階層上に出す (= 親の兄弟にする)。トップレベルなら no-op。
// 移動先での position は元の親の直後。
func OutdentTask(tasks []Task, id int) []Task {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return tasks
	}
	t := tasks[idx]
	if t.ParentID == 0 {
		return tasks
	}
	parentIdx := taskIndexByID(tasks, t.ParentID)
	if parentIdx == -1 {
		return tasks
	}
	parent := tasks[parentIdx]
	var oldChildren []int
	for i, tt := range tasks {
		if tt.ParentID == parent.ID {
			oldChildren = append(oldChildren, i)
		}
	}
	sortByPositionID(tasks, oldChildren)
	var oldOrder []int
	for _, j := range oldChildren {
		if j == idx {
			continue
		}
		oldOrder = append(oldOrder, j)
	}
	renumberPositions(tasks, oldOrder)
	tasks[idx].ParentID = parent.ParentID
	var newSibs []int
	if parent.ParentID == 0 {
		newSibs = peerIndexes(tasks, 0, tasks[idx].StatusID)
	} else {
		for i, tt := range tasks {
			if tt.ParentID == parent.ParentID {
				newSibs = append(newSibs, i)
			}
		}
		sortByPositionID(tasks, newSibs)
	}
	var ordered []int
	inserted := false
	for _, j := range newSibs {
		if j == idx {
			continue
		}
		ordered = append(ordered, j)
		if j == parentIdx {
			ordered = append(ordered, idx)
			inserted = true
		}
	}
	if !inserted {
		ordered = append(ordered, idx)
	}
	renumberPositions(tasks, ordered)
	return tasks
}

// ReassignTasksToFallback は status_id == fromStatusID のタスクの StatusID を toStatusID に
// 書き換え、移動先 (toStatusID) の top-level グループの position を 1..N に再採番した新しい tasks を返す。
// status 削除時にそのステータスを参照していたタスクを別ステータスへ寄せる用途を想定。
// 子タスク (parent_id != 0) の position は同じ親の中で完結しているため触らない。
// 元の tasks スライスは変更しない。
func ReassignTasksToFallback(tasks []Task, fromStatusID, toStatusID int) []Task {
	out := make([]Task, len(tasks))
	copy(out, tasks)
	for i := range out {
		if out[i].StatusID == fromStatusID {
			out[i].StatusID = toStatusID
		}
	}
	peers := peerIndexes(out, 0, toStatusID)
	renumberPositions(out, peers)
	return out
}

func indexOf(idxs []int, target int) int {
	for i, j := range idxs {
		if j == target {
			return i
		}
	}
	return -1
}

// neighborStatusID は statuses を Sorted 順に並べたとき、currentID の direction (-1 / +1) 隣の
// status id を返す。端を超える場合は ok=false。
func neighborStatusID(statuses StatusList, currentID, direction int) (int, bool) {
	sorted := statuses.Sorted()
	cur := -1
	for i, s := range sorted {
		if s.ID == currentID {
			cur = i
			break
		}
	}
	if cur == -1 {
		return 0, false
	}
	next := cur + direction
	if next < 0 || next >= len(sorted) {
		return 0, false
	}
	return sorted[next].ID, true
}
