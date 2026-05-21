# データモデル (`tasks.yaml`)

`tasks.yaml` は task-man の唯一の永続化先である。トップレベルのキーは `version` / `applications` / `file_opener` / `data_base_directory` / `layout` / `statuses` / `fields` / `tags` / `tasks` の 9 種類で、`statuses` 以外は省略可能 (空配列 / 未定義どちらでも可)。

実装は `internal/storage/yaml.go` の `yamlFile` 型と `Load` / `Save` で行う。

## スキーマバージョン

`version` キーは yaml スキーマのバージョンを示す。

- 現行バージョンは `internal/storage/yaml.go` の `CurrentSchemaVersion`（執筆時点で `1`）。
- 互換性を破る変更を加えるたびに 1 ずつ増やす。
- Load 時の挙動:
  - `version > CurrentSchemaVersion` → `ErrSchemaVersionUnsupported` で起動を拒否。古いバイナリで新しい yaml を上書きしてデータを破壊することを防ぐ。
  - `version` 未指定（旧 yaml）→ 現行バージョンとして解釈し、Load 後の再 Save で `version: 1` を補完する。
  - `1 <= version <= CurrentSchemaVersion` → そのまま受け入れる。将来 `version < CurrentSchemaVersion` のケースが発生したら、ここに移行ロジックを挟む（現時点では v1 しか存在しないため未実装）。
- Save は常に `CurrentSchemaVersion` を書き出す。

```yaml
version: 1
```

## トップレベルの例

```yaml
version: 1

applications:
  - application:
      id: 1
      name: editor
      run: $EDITOR
  - application:
      id: 2
      name: md-viewer
      run: md-viewer

file_opener:
  - opener:
      extension: "md"
      applications: [1, 2]
      default_app: 1

data_base_directory: ./tasks_data

layout:
  main:
    task_list:
      width: 0.6
    task_detail:
      height: 0.4
    file_list:
      height: 0.3
    file_preview:
      height: 0.3

statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
      color: "#6c7086"
  - status:
      id: 2
      sequence: 2
      label: doing
      color: "#fab387"
  - status:
      id: 3
      sequence: 3
      label: done
      color: "#a6e3a1"

fields:
  - field:
      id: 1
      type: text
      name: due_date
      position: 1

tags:
  - tag:
      id: 1
      name: urgent
      color: "#f38ba8"

tasks:
  - task:
      id: 1
      title: Task 1
      status_id: 1
      position: 1
      tags: [1]
      fields:
        - field:
            id: 1
            field_id: 1
            value: "2026-05-01"
```

## `applications`

外部アプリケーションの一覧。`internal/storage/yaml.go` の `yamlApplicationEntry`。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `application.id` | int (>0) | 必須 | 一意の ID。重複・0 以下はロードエラー。 |
| `application.name` | string | 必須 | 表示名。空文字はロードエラー。 |
| `application.run` | string | 必須 | 起動コマンド。`$EDITOR` 等の環境変数記法 / PATH 上のコマンド / `nvim --noplugin` のような引数付き文字列に対応。空文字はロードエラー。 |

## `file_opener`

拡張子ごとの起動アプリ対応表。`yamlFileOpenerEntry`。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `opener.extension` | string | 必須 | ドット無しの拡張子 (例: `"md"`)。先頭 `.` は除去、小文字に正規化。空文字はロードエラー。 |
| `opener.applications` | []int | 必須 | 候補となる `application.id` の配列。順序が UI モーダルの表示順。存在しない id はロードエラー。 |
| `opener.default_app` | int | 任意 | enter 押下時に起動するアプリの `application.id`。`0` または未指定なら `$EDITOR` フォールバック。存在しない id はロードエラー。 |

同一拡張子が複数現れた場合、最後に書かれたものが採用される (yaml の上書きの直感に合わせる)。

## `data_base_directory`

タスク添付ファイル群を置くベースディレクトリ。`tasks.yaml` の置かれたディレクトリからの相対パス。空文字なら yaml と同じディレクトリを使う。タスク本体ディレクトリは `<yamlDir>/<data_base_directory>/task-<id>/` (例: `./tasks_data/task-1/`)。実装: `storage.TaskDir`。

## `layout`

タスクリスト画面のペイン比率。`yamlLayoutMain` / `yamlLayoutValue`。詳細は [layout.md](./layout.md)。

| キー | 値 | 範囲 |
|---|---|---|
| `main.task_list.width` | float | 0.1 ～ 0.9 |
| `main.task_detail.height` | float | 0.1 ～ 0.8 |
| `main.file_list.height` | float | 0.1 ～ 0.8 |
| `main.file_preview.height` | float | 0.1 ～ 0.8 (3 値の合計が 1.0 になるよう正規化) |

未設定の場合は描画時にデフォルト比率 (横 2/3、縦 各 1/3) を用いる。

## `statuses`

タスクの状態の集合。`yamlStatusEntry`。空または未定義の場合、ロード時に `task.DefaultStatuses()` が注入され、即時に yaml へ書き戻される。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `status.id` | int (>0) | 必須 (未採番なら自動採番) | 一意の ID。0 以下 / 重複はロードエラー。 |
| `status.sequence` | int | 任意 | 表示順 (昇順)。タイブレークは id 昇順。 |
| `status.label` | string | 必須 | 表示名。空文字はロードエラー。 |
| `status.color` | string | 任意 | HEX カラー (例 `"#fab387"`)。空文字なら表示側で `colorMuted` にフォールバック。 |
| `status.collapsed` | bool | 任意 | タスクリストのグループ折りたたみ状態。 |

## `fields`

拡張項目のスキーマ定義。`yamlFieldEntry`。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `field.id` | int (>0) | 必須 (未採番なら自動採番) | 一意の ID。0 以下 / 重複はロードエラー。 |
| `field.type` | string | 任意 | `text` / `date` / `url`。空文字なら `text` を補完。未知の値はロードエラー。 |
| `field.name` | string | 必須 | 表示名。空文字 / 18 rune 超 / 禁止文字 (`NUL`, `/`, `:`) はロードエラー。 |
| `field.position` | int (>0) | 任意 | 表示順 (昇順)。未指定なら自動採番。 |

## `tags`

タグの集合。`yamlTagEntry`。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `tag.id` | int (>0) | 必須 (未採番なら自動採番) | 一意の ID。0 以下 / 重複はロードエラー。 |
| `tag.name` | string | 必須 | 表示名。空文字 / 8 rune 超 / 重複 / NUL 含みはロードエラー。 |
| `tag.color` | string | 任意 | HEX カラー。空文字なら `colorMuted` にフォールバック。 |

## `tasks`

タスクの集合。`yamlEntry` (`yamlTask`)。

| キー | 型 | 必須 | 説明 |
|---|---|---|---|
| `task.id` | int (>0) | 必須 | 一意の ID。重複 / 0 以下はロードエラー。 |
| `task.title` | string | 必須 | 空文字 / 60 rune 超 / 禁止文字 (`NUL`, `/`, `:`) はロードエラー。 |
| `task.status_id` | int | 必須 | `statuses` に存在する id を指す。 |
| `task.parent_id` | int | 任意 | 親タスクの id。`0` または未指定でトップレベル。存在しない id はロードエラー。循環は検出してエラー。 |
| `task.position` | int | 任意 | 同一 parent / status グループ内での昇順表示順。0 のタスクは出現順に自動採番。 |
| `task.collapsed` | bool | 任意 | サブタスクのリスト上折りたたみ状態。 |
| `task.is_trash_box` | bool | 任意 | true ならゴミ箱に入っている扱い。 |
| `task.tags` | []int | 任意 | 紐付けられたタグ id 配列。最大 5、重複不可、未知 id はロードエラー。 |
| `task.fields` | array | 任意 | タスク内の拡張項目値。 |
| `task.fields[].field.id` | int (>0) | 必須 (未採番なら自動採番) | task 内で一意な ID。 |
| `task.fields[].field.field_id` | int | 必須 | `fields` 内のスキーマ ID。存在しない id はロードエラー。同一 task 内で重複不可。 |
| `task.fields[].field.value` | string | 任意 | 型に応じてバリデーション (詳細は [domain.md](./domain.md))。 |

### ネスト制約

- 親チェーンの最大ネスト深さは `task.MaxNestDepth=4` (= top-level + 4 階層、合計 5 階層)。
- 親 id の循環は禁止。
- `task.parent_id` がトップレベル (=0) でないとき、子タスクの `status_id` は描画上は親 (= ルート祖先) の status グループ配下にネスト表示される (`internal/tui/rows.go`)。

### ゴミ箱との関係

- `is_trash_box=true` のタスクは通常リストでは非表示、ゴミ箱ビュー (`;`→`t`) でのみ表示される。
- 親をゴミ箱に入れると、`TrashTask` (`internal/task/trash.go`) によって子孫もすべて `is_trash_box=true` になる。
- 復元 (`r` キー) では同様に子孫まで `is_trash_box=false` に戻す (`RestoreTask`)。
- 完全削除 (`DeleteTaskSubtree`) では、対象タスクとその子孫を tasks から取り除き、添付ディレクトリも `storage.DeleteTaskData` で削除する。

## 入力検証一覧

| 対象 | 上限 | 禁止文字 | 備考 |
|---|---|---|---|
| タスクタイトル | 60 rune | `NUL`, `/`, `:` | `task.ValidateTitleChars` |
| 拡張項目 name | 18 rune | `NUL`, `/`, `:` | `task.ValidateFieldNameChars` |
| text 型 value | 200 rune | `NUL` | `task.ValidateFieldTextValueChars` |
| date 型 value | `yyyy-mm-dd` 形式 (空文字許容) | - | `task.ValidateFieldDateValue` |
| url 型 value | 320 rune / scheme と host を持つ絶対 URL | `NUL` | `task.ValidateFieldURLValue` |
| タグ name | 8 rune | `NUL` | `task.ValidateTagNameChars` |
| ファイル名 | 255 rune | `NUL`, `/` | `storage.ValidateFileNameChars` |
| 1 タスクあたりタグ数 | 5 | - | `task.MaxTagsPerTask` |
