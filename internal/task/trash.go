package task

// SubtreeIDs は rootID とその子孫すべての ID を返す (深さ優先・行順)。
// rootID 自体が tasks に存在しない場合は空スライス。循環は検出して打ち切る。
func SubtreeIDs(tasks []Task, rootID int) []int {
	idIndex := make(map[int]int, len(tasks))
	for i, t := range tasks {
		idIndex[t.ID] = i
	}
	if _, ok := idIndex[rootID]; !ok {
		return nil
	}
	children := make(map[int][]int)
	for i, t := range tasks {
		if t.ParentID != 0 {
			children[t.ParentID] = append(children[t.ParentID], i)
		}
	}
	var out []int
	seen := map[int]bool{}
	var visit func(id int)
	visit = func(id int) {
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
		for _, ci := range children[id] {
			visit(tasks[ci].ID)
		}
	}
	visit(rootID)
	return out
}

// TrashTask は rootID とその子孫すべてに IsTrashBox=true を設定する。
// status_id / parent_id / position はそのまま保持されるため、復帰時 (RestoreTask) は
// 元のステータスグループへフラグを下ろすだけで戻る。
func TrashTask(tasks []Task, rootID int) []Task {
	for _, id := range SubtreeIDs(tasks, rootID) {
		idx := taskIndexByID(tasks, id)
		if idx == -1 {
			continue
		}
		tasks[idx].IsTrashBox = true
	}
	return tasks
}

// RestoreTask は rootID とその子孫すべての IsTrashBox を false に戻す。
// rootID 自体がゴミ箱に入っていない場合は no-op。
func RestoreTask(tasks []Task, rootID int) []Task {
	rootIdx := taskIndexByID(tasks, rootID)
	if rootIdx == -1 || !tasks[rootIdx].IsTrashBox {
		return tasks
	}
	for _, id := range SubtreeIDs(tasks, rootID) {
		idx := taskIndexByID(tasks, id)
		if idx == -1 {
			continue
		}
		tasks[idx].IsTrashBox = false
	}
	return tasks
}

// TrashRootID は id を起点に、IsTrashBox=true な親をたどって辿り着ける最上位の trashed 祖先 id を返す。
// id 自体が IsTrashBox=false なら id を返す。これにより、ゴミ箱ビュー上でサブタスクから restore を
// トリガしても、その「trashed としての根」から restore できる。
func TrashRootID(tasks []Task, id int) int {
	idx := taskIndexByID(tasks, id)
	if idx == -1 {
		return id
	}
	cur := tasks[idx]
	if !cur.IsTrashBox {
		return id
	}
	seen := map[int]bool{cur.ID: true}
	for cur.ParentID != 0 {
		pi := taskIndexByID(tasks, cur.ParentID)
		if pi == -1 {
			break
		}
		parent := tasks[pi]
		if !parent.IsTrashBox {
			break
		}
		if seen[parent.ID] {
			break
		}
		seen[parent.ID] = true
		cur = parent
	}
	return cur.ID
}

// DeleteTaskSubtree は rootID とその子孫すべてを tasks から取り除いた新しいスライスを返す。
// 元グループの position は再採番する。削除された ID リストを第二戻り値で返す
// (呼び出し側でデータディレクトリ等の後処理に使う)。
func DeleteTaskSubtree(tasks []Task, rootID int) ([]Task, []int) {
	subtree := SubtreeIDs(tasks, rootID)
	if len(subtree) == 0 {
		return tasks, nil
	}
	removeSet := make(map[int]bool, len(subtree))
	for _, id := range subtree {
		removeSet[id] = true
	}
	rootIdx := taskIndexByID(tasks, rootID)
	rootParent, rootStatus := 0, 0
	if rootIdx != -1 {
		rootParent = tasks[rootIdx].ParentID
		rootStatus = tasks[rootIdx].StatusID
	}
	out := make([]Task, 0, len(tasks)-len(subtree))
	for _, t := range tasks {
		if removeSet[t.ID] {
			continue
		}
		out = append(out, t)
	}
	var groupIdx []int
	for i, t := range out {
		if t.ParentID != rootParent {
			continue
		}
		if rootParent == 0 && t.StatusID != rootStatus {
			continue
		}
		groupIdx = append(groupIdx, i)
	}
	sortByPositionID(out, groupIdx)
	renumberPositions(out, groupIdx)
	return out, subtree
}
