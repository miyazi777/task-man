# 永続化仕様

`internal/storage` パッケージ。`Repository` インタフェースの唯一の実装は `YAMLRepository`。

## ロード (`YAMLRepository.Load`)

1. `os.ReadFile(path)` でファイル全体を読む。
2. 空でなければ `yaml.Unmarshal` で `yamlFile` 構造体にデコードする。
3. セクションごとに以下の順序で正規化と検証を行う:
   1. `loadStatuses`: 空または未定義なら `task.DefaultStatuses()` を注入 (要書き戻し)。id 未採番は `AssignMissingIDs` で `max+1` から採番。`Validate` で id 重複 / 0 以下 / label 空 を検出。
   2. `loadFieldDefs`: 各 field の id/position/type 補完を `AssignMissingIDsAndPositions` で行い、`Validate` で id 重複 / name 検証 / type 検証 / position>0 を確認。
   3. `loadTags`: id 未採番を補完して `Validate` する。
   4. `loadTasks`: id 重複・id 不正を弾く。task 内 fields の id 補完。`Task.Validate` (status_id 参照 / tags 参照) と `TaskFieldList.Validate(defs)` を実行。`validateParents` で循環 / 親不在 / ネスト超過を検出。`assignMissingPositions` で position=0 のタスクに兄弟内 `max+1` から採番。
   5. `loadApplications`: id>0 / 重複なし / name/run 非空を確認。
   6. `loadFileOpeners`: extension 正規化、applications 内 id の存在確認。
4. いずれかの補完が起きた場合 (`statusesChanged || defsChanged || tagsChanged || tasksChanged`) は、ロード直後に `Save(lr)` を呼んで yaml を整える。

ロード時の自動補完一覧:
- `statuses` 空 → デフォルト 3 種注入
- `statuses[].id <= 0` → `max+1` から採番
- `fields[].id <= 0` / `fields[].position <= 0` / `fields[].type == ""` → 補完
- `tags[].id <= 0` → 採番
- `tasks[].fields[].id <= 0` → 採番
- `tasks[].position == 0` → 兄弟内末尾に採番

## 保存 (`YAMLRepository.Save`)

1. `LoadResult` から `yamlFile` を組み立てる。
2. `statuses` は `Sorted()` (sequence 昇順) で出力。`fields` は `Sorted()` (position 昇順)。`tags` は `Sorted()` (id 昇順)。各タスク内の fields は id 昇順。
3. `yaml.Marshal` でバイト列化し、`atomicWrite(path, data)` で書き出す。

### `atomicWrite`

1. `os.CreateTemp(dir, ".task-man-*.tmp")` で一時ファイルを作成。
2. `Write` → `Sync` → `Close` の順に実行。
3. `os.Rename(tmp, path)` で原子的に置換。
4. 途中で失敗した場合は一時ファイルを `os.Remove` で掃除する。

## `LoadResult` / `AppConfig`

```go
// internal/storage/repository.go
type LoadResult struct {
    Tasks    []task.Task
    Statuses task.StatusList
    Fields   task.FieldDefList
    Tags     task.TagList
    Config   AppConfig
}

// internal/storage/config.go
type AppConfig struct {
    DataBaseDirectory string
    Layout            LayoutConfig
    Applications      []Application
    FileOpeners       []FileOpener
}
```

## タスク添付ファイル (`internal/storage/data.go`)

タスクごとのファイル群は `<yamlDir>[/data_base_directory]/task-<id>/` 配下に置かれる。

| 関数 | 動作 |
|---|---|
| `TaskDir(yamlDir, dataBaseDir, taskID)` | ディレクトリパスを組み立てる。`dataBaseDir` 空文字なら `yamlDir` 直下。 |
| `CreateTaskData(yamlDir, dataBaseDir, taskID)` | `task-<id>/memo.md` を作成。既存衝突は `ErrTaskDirExists` (no-op、何も作らない)。memo.md 作成失敗時はディレクトリも巻き戻し。 |
| `ListTaskFiles(yamlDir, dataBaseDir, taskID)` | タスクディレクトリ内の通常ファイルを basename のアルファベット順で返す。ディレクトリ不在は空スライスを返す。 |
| `CreateFile(yamlDir, dataBaseDir, taskID, name)` | 空ファイル作成。同名既存は `ErrFileExists`。タスクディレクトリが無ければ作る。 |
| `RenameFile(yamlDir, dataBaseDir, taskID, oldName, newName)` | リネーム。元不在は `ErrFileNotFoundIn`、先既存は `ErrFileExists`。 |
| `DeleteFile(yamlDir, dataBaseDir, taskID, name)` | 通常ファイルのみ削除。ディレクトリや特殊ファイルは拒否。 |
| `ReadTaskFile(yamlDir, dataBaseDir, taskID, name, maxBytes)` | プレビュー用に先頭 `maxBytes` バイトまで読む。 |
| `DeleteTaskData(yamlDir, dataBaseDir, taskID)` | `task-<id>/` を再帰削除。不在は no-op。 |
| `RemoveAllTaskData(yamlDir, dataBaseDir)` | data_base_directory 配下にある `task-<int>` 名のディレクトリをすべて削除。整数でない子や通常ファイルには触れない。`--init` で利用される。 |

### ファイル名検証

- 空文字: `ErrFileNameEmpty`。
- 255 rune 超: `ErrFileNameTooLong`。
- 禁止文字 (`NUL`, `/`): `FileNameForbiddenCharError`。
- ライブ入力検証は `ValidateFileNameChars` (空文字は許容)、確定時検証は `ValidateFileName` (空文字も拒否)。

## エラー定義

| 変数 | 意味 |
|---|---|
| `ErrTaskDirExists` | `CreateTaskData` 時にタスクディレクトリが既存 |
| `ErrFileNameEmpty` | ファイル名が空 |
| `ErrFileNameTooLong` | ファイル名が 255 rune 超 |
| `ErrFileExists` | 同名ファイルが既に存在 |
| `ErrFileNotFoundIn` | 対象ファイルがタスクディレクトリ内に無い |
| `ErrFileNotFound` | tasks ファイル自体が無い |
