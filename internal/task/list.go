package task

func NextID(tasks []Task) int {
	max := 0
	for _, t := range tasks {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}

// NextPosition は parentID を親に持つ兄弟タスクの中で次に割り当てるべき position を返す。
// 兄弟が居なければ 1 を返す。
func NextPosition(tasks []Task, parentID int) int {
	max := 0
	for _, t := range tasks {
		if t.ParentID != parentID {
			continue
		}
		if t.Position > max {
			max = t.Position
		}
	}
	return max + 1
}
