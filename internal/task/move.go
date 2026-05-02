package task

import "sort"

// MoveDestination は MoveTasks における移動先の指定。
//
//   - ParentID: 新しい親タスク ID (ルートに置くなら 0)
//   - StatusID: 新しいステータス ID
//   - InsertAt: 移動対象が新グループ内で取る最初の Position (1 始まり)。
//     新グループの既存タスクを Position 昇順に並べたとき、index = InsertAt-1 の位置に挿入される。
//     値が新グループの長さ+1 を超える場合は末尾に追加される。
type MoveDestination struct {
	ParentID int
	StatusID int
	InsertAt int
}

// MoveTasks は ids に含まれるタスクを dst に移動した新しいタスクスライスを返す。
//
// 仕様:
//   - 移動対象タスクの ParentID, StatusID は dst に従って更新される。
//   - 移動対象タスクの子孫 (= ParentID が移動対象のタスク) は ParentID/StatusID を変更しない。
//     これにより親が別ステータスに移ると、子孫が元のグループに孤立する状態を許容する。
//   - 移動順は (旧 ParentID, 旧 Position, ID) の昇順で安定。
//   - 同一グループ内移動 (旧グループ == 新グループ) もサポート。穴はカウントせず再採番される。
//   - 各グループの Position は最後に 1 から連番で振り直される (グループ内で密に詰める)。
//
// 戻り値の長さは入力 tasks と同じ。元のスライスは変更しない。
func MoveTasks(tasks []Task, ids map[int]bool, dst MoveDestination) []Task {
	out := make([]Task, len(tasks))
	copy(out, tasks)

	movingIdx := make([]int, 0, len(ids))
	for i, t := range out {
		if ids[t.ID] {
			movingIdx = append(movingIdx, i)
		}
	}
	sort.SliceStable(movingIdx, func(a, b int) bool {
		ia, ib := movingIdx[a], movingIdx[b]
		if out[ia].ParentID != out[ib].ParentID {
			return out[ia].ParentID < out[ib].ParentID
		}
		if out[ia].Position != out[ib].Position {
			return out[ia].Position < out[ib].Position
		}
		return out[ia].ID < out[ib].ID
	})

	// 旧グループ (ParentID, StatusID) 集合 — 後で再採番するため記録。
	type group struct {
		parent int
		status int
	}
	oldGroups := make(map[group]bool)
	for _, i := range movingIdx {
		oldGroups[group{out[i].ParentID, out[i].StatusID}] = true
		out[i].ParentID = dst.ParentID
		out[i].StatusID = dst.StatusID
	}
	dstGroup := group{dst.ParentID, dst.StatusID}
	oldGroups[dstGroup] = true

	// 新グループ内に移動対象を InsertAt の位置に挿入するため、
	// 移動対象は一時的に新 Position の中間値 (大きめにシフトした値) を持たせ、
	// その後グループ単位で再採番する。
	const moveBase = 1_000_000
	for k, i := range movingIdx {
		out[i].Position = moveBase + k
	}

	// 新グループの既存 (移動対象以外) タスクは、InsertAt-1 番目までは現状維持、
	// InsertAt 以降は移動対象群より後ろに来るよう Position を持ち上げる。
	// 単純化のため: 既存タスクの Position をベース順位に詰め直し、その後 InsertAt 以降を 2*moveBase へ持ち上げる。
	{
		// dstGroup に属する既存タスク (= 移動対象でないもの) を集める
		var existing []int
		for i := range out {
			if ids[out[i].ID] {
				continue
			}
			if out[i].ParentID == dst.ParentID && out[i].StatusID == dst.StatusID {
				existing = append(existing, i)
			}
		}
		sort.SliceStable(existing, func(a, b int) bool {
			ia, ib := existing[a], existing[b]
			if out[ia].Position != out[ib].Position {
				return out[ia].Position < out[ib].Position
			}
			return out[ia].ID < out[ib].ID
		})
		// 1..InsertAt-1: そのまま (連番再採番)
		// InsertAt..end: 2*moveBase 以降に押しやる
		insertIdx := dst.InsertAt - 1
		if insertIdx < 0 {
			insertIdx = 0
		}
		if insertIdx > len(existing) {
			insertIdx = len(existing)
		}
		for k, ei := range existing {
			if k < insertIdx {
				out[ei].Position = k + 1
			} else {
				out[ei].Position = 2*moveBase + k
			}
		}
	}

	// 影響を受ける各グループ (旧グループ + 新グループ) で Position を 1 から連番で振り直す。
	for g := range oldGroups {
		var members []int
		for i := range out {
			if out[i].ParentID == g.parent && out[i].StatusID == g.status {
				members = append(members, i)
			}
		}
		sort.SliceStable(members, func(a, b int) bool {
			ia, ib := members[a], members[b]
			if out[ia].Position != out[ib].Position {
				return out[ia].Position < out[ib].Position
			}
			return out[ia].ID < out[ib].ID
		})
		for k, mi := range members {
			out[mi].Position = k + 1
		}
	}

	return out
}
