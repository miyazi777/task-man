# ドメイン仕様

`internal/task` パッケージに実装される純粋なビジネスルール。

## Task

- ID: 正の整数。`task.NextID(tasks)` は `max(id)+1` を返す (`internal/task/list.go`)。
- Title: 必須。`ValidateTitleChars` で長さと禁止文字をチェックする。
- StatusID: 必須。`statuses` に存在する id でなければ `Validate` がエラー。
- ParentID: 0 でトップレベル。0 以外なら循環なし & 親存在 & ネスト深さ <= 4 を満たす必要がある (`internal/storage/yaml.go` の `validateParents`、`internal/task/move.go` の `taskNestDepth` / `subtreeRelDepth`)。
- Position: 同一 (`ParentID`, トップレベル時の `StatusID`) グループ内で 1 始まりの昇順。`NextPosition(tasks, parentID)` で最後尾を計算する。
- Tags: 最大 `MaxTagsPerTask=5`。重複と未知 id は `Task.Validate` でエラー。
- Fields: タスク内で `id` 重複不可、`field_id` も重複不可。値は `FieldDef.Type` に応じてバリデートする (`TaskFieldList.Validate`)。

### 並び替えロジック (`internal/task/move.go`)

- `MoveTaskUp(tasks, statuses, id)`: 兄弟内で 1 つ上に移動。先頭で top-level の場合は sequence が 1 つ小さいステータスの末尾へ移動する。先頭で非 top-level の場合は no-op。
- `MoveTaskDown(tasks, statuses, id)`: 兄弟内で 1 つ下に移動。末尾で top-level の場合は sequence が 1 つ大きいステータスの先頭へ移動する。末尾で非 top-level の場合は no-op。
- `IndentTask(tasks, id)`: 直前の兄弟の子にする。直前の兄弟が無い or 最大ネスト深さを超える場合は no-op。新親の子末尾に配置する。
- `OutdentTask(tasks, id)`: 親の兄弟になる (1 階層上げる)。トップレベルなら no-op。新位置は元親の直後。
- `ReassignTasksToFallback(tasks, fromStatusID, toStatusID)`: ステータス削除時に参照していたタスクを別ステータスへ移動する。

### ネスト深さチェック

- `IndentTask` は `taskNestDepth(tasks, newParent) + 1 + subtreeRelDepth(tasks, target) > MaxNestDepth` のとき何もしない。
- yaml ロード時にも `validateParents` で `MaxNestDepth=4` を確認する。

## Status

- ユーザー定義可能なステータス。`StatusList` は `Sorted()` で sequence 昇順 (タイブレークは id 昇順) を返す。
- 最低 1 件は必要 (`DeleteByID` は最後の 1 件で `ErrCannotDeleteLastStatus`)。
- 削除時の fallback ステータス決定: Sorted で対象の次 (sequence が 1 大きい = 表示上 1 つ下) を採用。末尾なら 1 つ前 (sequence が 1 小さい = 表示上 1 つ上) を採用。
- 削除後は `sequence` を 1..N-1 に振り直す。
- `InsertAt(insertIdx, label, color)`: 指定位置に挿入。新 id は `max(id)+1`、sequence は 1..N に振り直す。
- `MoveStatusUp` / `MoveStatusDown` / `ResequenceByOrder`: 表示順を変更する。
- `RenameByID(id, newLabel)`: 空文字は `ErrStatusEmptyLabel`。
- `SetColorByID(id, newColor)`: 色文字列の検証は呼び出し側責任 (UI が `statusColorChoices()` から選ぶ前提)。

### デフォルト値 (`task.DefaultStatuses`)

| id | sequence | label | color |
|---|---|---|---|
| 1 | 1 | todo | `#6c7086` |
| 2 | 2 | doing | `#fab387` |
| 3 | 3 | done | `#a6e3a1` |

## Field (拡張項目)

### スキーマ (`FieldDef`)

- `ID`, `Name`, `Type`, `Position`
- `Type` の有効値: `text` / `date` / `url`。デフォルトは `text`。
- `AddDef(name, type)`: name 検証後に `max(id)+1` / `max(position)+1` で採番。
- `DeleteByID(id)`: 削除後に position を 1..N に振り直す。
- `RenameByID(id, newName)`: 空文字 / 18 rune 超 / 禁止文字はエラー。
- `MoveUp` / `MoveDown`: position を入れ替える。

### 値 (`TaskField`)

- 各 task の `Fields` 配下にぶら下がる。`FieldDef.ID` を `FieldID` で参照する。
- 同一 task 内で `id` と `field_id` の重複は不可。
- 型ごとのバリデーション (`TaskFieldList.Validate(defs)`):
  - `text`: 200 rune 上限、NUL 不可。
  - `date`: 空文字 OK、それ以外は `yyyy-mm-dd` で `time.Parse` 通過必須。
  - `url`: 空文字 OK、320 rune 上限、`url.Parse` が成功し scheme と host を持つこと。scheme に空白文字は含めない。
- `SetValue(fieldID, value)`: 既存があれば上書き、無ければ末尾に追加 (新 id は `max(id)+1`)。
- `RemoveByFieldID(fieldID)`: 値を削除する。フィールド定義削除時に `task.PurgeRemovedFieldValues` (`internal/task/field_ops.go`) で全タスクから孤児値を除去する。

### Date 形式

- レイアウト定数: `FieldDateLayout = "2006-01-02"`
- UI のカレンダーで選択した日付を `yyyy-mm-dd` 文字列として保存する。

## Tag

- `Name`: 必須、8 rune 上限、NUL 不可、全タグ内で重複不可。
- `Color`: HEX (例: `#f38ba8`)。表示時に foreground のみ着色 (status の塗りチップとの差別化のため `#name` 形式で表示)。
- `AddTag(name, color)`: 重複名は `ErrTagDuplicateName`。`max(id)+1` で採番。
- `RenameByID(id, newName)`: 自分自身との完全一致は許容、他タグとの重複はエラー。
- `DeleteByID(id)`: 該当 id が無ければエラー。UI 側でタスクの `Tags` 配列からも削除する責務がある (`internal/tui/app.go` の `ModeTagPickerDeleteConfirm`)。

## ゴミ箱 (Trash)

`internal/task/trash.go`。

- `SubtreeIDs(tasks, rootID)`: rootID とその全子孫を深さ優先 / 行順で列挙する。循環は検出して打ち切る。
- `TrashTask(tasks, rootID)`: rootID と子孫すべてに `IsTrashBox=true` を立てる。`status_id` / `parent_id` / `position` は保持する。
- `RestoreTask(tasks, rootID)`: rootID と子孫すべての `IsTrashBox` を `false` に戻す。rootID 自体がゴミ箱に居なければ no-op。
- `TrashRootID(tasks, id)`: ゴミ箱ビュー上でサブタスクから restore する際に「trashed としての根」を求める補助。
- `DeleteTaskSubtree(tasks, rootID)`: rootID とその子孫を tasks から取り除き、元グループの position を再採番する。削除された id 配列を返す (添付ディレクトリの後処理用)。

## 入力検証 (`Validate`) のエラー一覧

| 関数 | エラー |
|---|---|
| `Task.Validate` | `ErrEmptyTitle` / `ErrTitleTooLong` / `ErrInvalidID` / `ErrUnknownStatusID` / `ErrTaskTooManyTags` / `ErrTagUnknownID` / 重複 tag |
| `StatusList.Validate` | `ErrStatusInvalidID` / `ErrStatusEmptyLabel` / 重複 id |
| `FieldDefList.Validate` | `ErrFieldInvalidID` / `ErrFieldEmptyName` / `ErrFieldNameTooLong` / `ErrFieldUnknownType` / `ErrFieldInvalidPosition` / 重複 id / 禁止文字 |
| `TaskFieldList.Validate` | `ErrFieldInvalidID` / `ErrFieldUnknownFieldID` / `ErrFieldValueTooLong` / 重複 id / 重複 field_id / 型別エラー (`ErrFieldInvalidDateValue` / `ErrFieldInvalidURLValue` / `ErrFieldURLValueTooLong`) |
| `TagList.Validate` | `ErrTagInvalidID` / `ErrTagEmptyName` / `ErrTagNameTooLong` / `ErrTagDuplicateName` / 重複 id |
