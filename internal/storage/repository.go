package storage

import "github.com/miyazi777/task-man/internal/task"

// LoadResult はリポジトリから読み込まれた永続化データの集合。
// 設定 / ステータス / 拡張項目スキーマ / タグ / タスク を一つにまとめて返す。
type LoadResult struct {
	Tasks    []task.Task
	Statuses task.StatusList
	Fields   task.FieldDefList
	Tags     task.TagList
	Config   AppConfig
}

type Repository interface {
	Load() (LoadResult, error)
	Save(lr LoadResult) error
}
