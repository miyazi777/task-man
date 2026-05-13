package task

// NextID は tasks の中の最大 ID + 1 を返す。新規タスクの id 採番に使う。
// 空のスライスを渡した場合は 1 を返す。
func NextID(tasks []Task) int {
	maxID := 0
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	return maxID + 1
}

// NextPosition は parentID を親に持つ兄弟タスクの中で次に割り当てるべき position を返す。
// 兄弟が居なければ 1 を返す。
func NextPosition(tasks []Task, parentID int) int {
	maxPos := 0
	for _, t := range tasks {
		if t.ParentID != parentID {
			continue
		}
		if t.Position > maxPos {
			maxPos = t.Position
		}
	}
	return maxPos + 1
}
