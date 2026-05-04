package task

import "fmt"

// PurgeRemovedFieldValues は defs に存在しなくなった field_id を全タスクから除去する。
// FieldDefList.DeleteByID 後に必ず呼んで孤児 TaskField を残さないようにする。
// 元の tasks スライスは変更せず、新しいスライスを返す。
func PurgeRemovedFieldValues(tasks []Task, defs FieldDefList) []Task {
	out := make([]Task, len(tasks))
	copy(out, tasks)
	for i := range out {
		if len(out[i].Fields) == 0 {
			continue
		}
		filtered := make(TaskFieldList, 0, len(out[i].Fields))
		for _, tf := range out[i].Fields {
			if _, ok := defs.ByID(tf.FieldID); ok {
				filtered = append(filtered, tf)
			}
		}
		out[i].Fields = filtered
	}
	return out
}

// SetFieldValue は taskID の TaskField を fieldID に対して value で更新する。
// 該当タスクが無い / fieldID が defs に無い場合はエラー。
// 値文字数 / 禁止文字バリデーションは呼び出し側責任 (ライブ入力検証側でも捕捉する想定)。
func SetFieldValue(tasks []Task, defs FieldDefList, taskID, fieldID int, value string) ([]Task, error) {
	if _, ok := defs.ByID(fieldID); !ok {
		return nil, fmt.Errorf("%w: %d", ErrFieldUnknownFieldID, fieldID)
	}
	out := make([]Task, len(tasks))
	copy(out, tasks)
	for i := range out {
		if out[i].ID == taskID {
			out[i].Fields = out[i].Fields.SetValue(fieldID, value)
			return out, nil
		}
	}
	return nil, fmt.Errorf("task id %d not found", taskID)
}

// ValidateAll は (statuses, defs, tags) の存在前提で全タスク + 各タスクの fields を検証する。
func ValidateAll(tasks []Task, statuses StatusList, defs FieldDefList, tags TagList) error {
	if err := defs.Validate(); err != nil {
		return err
	}
	if err := tags.Validate(); err != nil {
		return err
	}
	for i, t := range tasks {
		if err := t.Validate(statuses, tags); err != nil {
			return fmt.Errorf("tasks[%d]: %w", i, err)
		}
		if err := t.Fields.Validate(defs); err != nil {
			return fmt.Errorf("tasks[%d]: %w", i, err)
		}
	}
	return nil
}
