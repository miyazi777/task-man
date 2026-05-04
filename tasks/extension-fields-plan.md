# 拡張項目機能 実装計画 (rev.2)

仕様: `private/issue.md` の「拡張項目機能」(更新版)。

## 0. 仕様改定で変わった点

旧版では各 task の下に name/type/position/value を全部持たせていた。
新版では:

- **スキーマ定義はトップレベル `fields:` に集約**: id / name / type / position
- **タスクは `field_id` で参照する値だけを持つ**: id (インスタンス id) / field_id / value
- **text の最大値長は 200 rune** (明示)

これにより 0 タスク時の問題が消え、スキーマは常に単一の source of truth になる。

## 1. 全体方針

- type は `text` のみ実装。enum で将来 url/date を追加できる構造に。
- スキーマは top-level、値はタスクごと。両者の整合性 (field_id が schema に存在) はロード時検証 + 削除時にカスケード。
- スキーマ操作 (追加/削除/リネーム/移動) は schema 1 配列に閉じる。値操作はタスク 1 件に閉じる。
- 詳細画面の表示: Status の下、Files の上に schema の position 昇順で並べる。値が無い field は空文字でレンダリング (ラベルだけ出る)。
- 値編集は Title/Status と同じ流儀: Enter で入力ポップアップ。**ラベルは `<field name>:` (例: `締切日:`) を採用**。
- 拡張項目の `value` 文字数上限 = 200 rune。`name` 長さ上限 = 18 rune (マルチバイト)。
- **拡張項目追加モーダルは 2 フィールド構成**: `name` (テキスト入力) と `type` (セレクター。現状 text のみだが将来拡張に備えセレクター UI で実装)。

## 2. yaml スキーマ拡張

```yaml
applications:
  editor: ...
data_base_directory: ...
statuses:
  - status: ...
fields:                              # ← 新規 (空配列 OK / 省略 OK)
  - field:
      id: 1
      type: text
      name: 締切日
      position: 1
  - field:
      id: 2
      type: text
      name: 開始日
      position: 2
tasks:
  - task:
      id: 1
      title: task1
      status_id: 3
      position: 1
      fields:                        # ← 新規 (省略 OK = 値なし)
        - field:
            id: 1                    # task 内のインスタンス id (一意)
            field_id: 1              # 上位 fields の id を参照
            value: "2025-01-01"
```

## 3. ドメイン層 (`internal/task`)

### 新規 `field.go` — スキーマ側

```go
type FieldType string
const (
    FieldTypeText FieldType = "text"
    // FieldTypeURL  FieldType = "url"
    // FieldTypeDate FieldType = "date"
)

const (
    MaxFieldNameRunes      = 18
    MaxFieldTextValueRunes = 200
)

// FieldDef はトップレベル fields の 1 件 (スキーマ定義)。
type FieldDef struct {
    ID       int
    Name     string
    Type     FieldType
    Position int
}

type FieldDefList []FieldDef

func (fl FieldDefList) Sorted() FieldDefList                     // position ASC, id ASC
func (fl FieldDefList) ByID(id int) (FieldDef, bool)
func (fl FieldDefList) Validate() error                          // 重複 id / 空 name / type 不明 / position<=0
func (fl FieldDefList) AssignMissingIDsAndPositions() (FieldDefList, bool)

// 編集 API (新しい FieldDefList を返す)
func (fl FieldDefList) AddDef(name string, ft FieldType) (FieldDefList, int, error)
func (fl FieldDefList) RenameByID(id int, newName string) (FieldDefList, error)
func (fl FieldDefList) DeleteByID(id int) (FieldDefList, error)
func (fl FieldDefList) MoveUp(id int) FieldDefList
func (fl FieldDefList) MoveDown(id int) FieldDefList

// バリデーション (ライブ入力用)
func ValidateFieldNameChars(s string) error                      // 18 rune 上限 + 禁止文字
func ValidateFieldTextValueChars(s string) error                 // 200 rune 上限 + 禁止文字
```

### 新規 `field.go` — タスク側 (同ファイル末尾 or 別ファイル)

```go
// TaskField は単一タスク内の拡張項目インスタンス (値ホルダー)。
type TaskField struct {
    ID      int    // task.fields 内で一意
    FieldID int    // FieldDef.ID への参照
    Value   string
}

type TaskFieldList []TaskField

func (tfl TaskFieldList) ByFieldID(fieldID int) (TaskField, bool)
func (tfl TaskFieldList) SetValue(fieldID int, value string) TaskFieldList   // 既存があれば置換、無ければ追加 (id 採番)
func (tfl TaskFieldList) RemoveByFieldID(fieldID int) TaskFieldList
func (tfl TaskFieldList) Validate(defs FieldDefList) error                   // id 重複 / field_id 未知 / value 長さ
```

### `task.go` 改修

- `Task.Fields TaskFieldList` を追加
- `Task.Validate(statuses)` は変更しない (FieldDef を引数に取らない設計)。Field 整合性はパッケージレベルの `ValidateAll` で扱う。

### 新規 `field_ops.go` — タスク群とスキーマの整合操作

```go
// PurgeRemovedFieldValues は defs に存在しなくなった field_id を全タスクから除去する。
// 主に FieldDef.DeleteByID 後に呼ぶ。
func PurgeRemovedFieldValues(tasks []Task, defs FieldDefList) []Task

// SetFieldValue は taskID の TaskField を fieldID に対して value で更新 (無ければ追加)。
func SetFieldValue(tasks []Task, taskID, fieldID int, value string) ([]Task, error)

// ValidateAll は (statuses, defs) の存在前提で全タスク + 各タスクの fields を検証。
func ValidateAll(tasks []Task, statuses StatusList, defs FieldDefList) error
```

### テスト

- `field_test.go`:
  - `Sorted` / `ByID` / `Validate`
  - `AddDef` / `RenameByID` / `DeleteByID` / `MoveUp/Down`: position が 1..N に振り直される
  - `TaskFieldList.SetValue` (新規追加 vs 既存更新)
  - 18 rune / 200 rune 境界
- `field_ops_test.go`:
  - `PurgeRemovedFieldValues`: schema から消えた field_id を持つ TaskField がすべて消える、他は残る
  - `SetFieldValue`: 該当タスクのみ変更、他タスク不変

## 4. ストレージ層 (`internal/storage`)

### `repository.go` シグネチャ変更

```go
type Repository interface {
    Load() (LoadResult, error)
    Save(LoadResult) error
}

type LoadResult struct {
    Tasks    []task.Task
    Statuses task.StatusList
    Fields   task.FieldDefList
    Config   AppConfig
}
```

(現行の `Load() ([]task.Task, task.StatusList, AppConfig, error)` を bundle 化。
呼び出し側 `cmd/task-man/main.go` と TUI Model を一斉更新。)

### `yaml.go` 改修

```go
type yamlField struct {
    ID       int    `yaml:"id"`
    Type     string `yaml:"type,omitempty"`
    Name     string `yaml:"name"`
    Position int    `yaml:"position,omitempty"`
}

type yamlFieldEntry struct {
    Field yamlField `yaml:"field"`
}

type yamlTaskField struct {
    ID      int    `yaml:"id"`
    FieldID int    `yaml:"field_id"`
    Value   string `yaml:"value,omitempty"`
}

type yamlTaskFieldEntry struct {
    Field yamlTaskField `yaml:"field"`
}

type yamlTask struct {
    // ... 既存
    Fields []yamlTaskFieldEntry `yaml:"fields,omitempty"`
}

type yamlFile struct {
    // ... 既存
    Fields []yamlFieldEntry `yaml:"fields,omitempty"`
}
```

ロード手順:
1. `yamlFile.Fields` → `task.FieldDefList`
2. `AssignMissingIDsAndPositions` で補完。補完が起きたら書き戻し対象。
3. `defs.Validate()`
4. tasks 読み込み中、各 task の TaskField を読み出し。task 内 id が 0 のものは max+1 で採番 (これも書き戻し対象)。
5. `task.ValidateAll(tasks, statuses, defs)` で field_id 整合性チェック。

セーブ手順:
- top-level fields は position 昇順で出力
- 各 task の fields は task 内 id 昇順で出力 (永続化キーとして安定)
- 値が空文字列の TaskField は省略しない (= ユーザーが意図的に空にしているケースを区別)

### テスト

- ラウンドトリップ (fields あり/なし)
- field_id が schema に存在しない → エラー
- field_id 重複 (同一タスク内で同じ field_id を持つ TaskField が 2 件) → エラー
- field.id=0 → 採番されて書き戻し
- task field.id=0 → 採番されて書き戻し
- type 欠落 → "text" 補完
- 既存 yaml (fields キー無し) のロードと再保存

## 5. TUI 層 (`internal/tui`)

### 5-1. Model 拡張 (`app.go`)

```go
type Model struct {
    // ... 既存
    fields  task.FieldDefList   // top-level スキーマ
    
    // 設定画面 / field
    settingFieldCursor       int       // 中央ペイン (field 一覧)
    settingFieldAttrCursor   int       // 右ペイン (attributes 行)
    settingFieldMoveSnapshot task.FieldDefList
    settingFieldMoving       int       // ModeSettingFieldMove 中の対象 field id
}
```

`NewModel` シグネチャに `fields task.FieldDefList` を追加。

### 5-2. 詳細画面 (`detail.go`) 改修

- `detailFieldTitle/Status/Files` の整数定数の代わりに、動的な「detail row 配列」を導入。
  ```go
  type detailRowKind int
  const (
      detailRowTitle detailRowKind = iota
      detailRowStatus
      detailRowField     // FieldID で識別
      detailRowFiles     // (Files セクション全体を 1 行として扱うのは現状どおり)
  )
  type detailRow struct {
      kind    detailRowKind
      fieldID int
  }
  ```
- `m.detailRows []detailRow` を Model に持たせ、タスク or fields 変更時に再構築。
- `m.detailCursor` は `m.detailRows` のインデックス。
- カーソル順: Title → Status → field1 → ... → fieldN → Files (内部で fileCursor) と直線的に移動。
- `detailFilesDividerRow` (Files の罫線位置) は detailRows と Files 内の構造から動的に算出。pane 区切り線の T 字接合に渡す。

レイアウト例:
```
  ID        1
  Title     task1
  Status    完了
  締切日    2025-01-01           ← FieldDef.Name + TaskField.Value
  開始日    2024-12-25
  
  Files:
  ─────────────────
    memo.md
```

### 5-3. 設定画面 (`setting.go`) — 動的ペイン構成

**status 選択中は現行の 2 ペインレイアウトを維持。field 選択中のみ 3 ペインに切り替え**。

- 左メニュー (`settingMenuLabels`) を `["status", "field"]` に拡張。
- レイアウト分岐:
  - **status 系モード** (`ModeSetting` で menu cursor が status / `ModeSettingStatus*`): 現行の 2 ペイン (左メニュー + 右 status 詳細) を変更せず維持。
  - **field 系モード** (`ModeSetting` で menu cursor が field / `ModeSettingField*`): 3 ペイン (左メニュー + 中央 field 一覧 + 右 attributes)。
- 中央ペイン (field 選択時): `renderSettingFieldPane`
- 右ペイン (field 選択時): `renderSettingFieldAttributePane` — `name: <値>` と `type: <値>` の 2 行
- 幅配分:
  - 2 ペイン時 (status): 既存どおり (`leftW = m.width / 3`、残りを右へ)
  - 3 ペイン時 (field): `leftW = 12 cell` 固定、残りを中央 / 右で 1/2 ずつ

### 5-4. 新規モード (`mode.go`)

```go
ModeSettingField                 // 中央 (field 一覧)
ModeSettingFieldAttribute        // 右 (attributes)
ModeSettingFieldAdd              // 追加入力ポップアップ (name 入力)
ModeSettingFieldRename           // name 変更入力ポップアップ
ModeSettingFieldMove             // 位置変更
ModeSettingFieldDeleteConfirm    // 削除確認
ModeEditFieldValue               // 詳細画面で field 値編集
```

### 5-5. キー設計

中央 (ModeSettingField):
- `k/↑` `j/↓`: カーソル
- `a`: 追加 → ModeSettingFieldAdd (multi-row popup)
- `r`: rename → ModeSettingFieldRename (single-row popup)
- `m`: 位置変更 → ModeSettingFieldMove
- `d`: 削除 → ModeSettingFieldDeleteConfirm
- `enter`: 右ペインへ → ModeSettingFieldAttribute
- `esc`: 左メニューへ → ModeSetting

右 (ModeSettingFieldAttribute):
- `k/↑` `j/↓`: name / type 行を上下 (type 行は read-only だが移動可)
- `enter`: name 行のみ ModeSettingFieldRename
- `esc`: 中央へ → ModeSettingField

追加モーダル (ModeSettingFieldAdd) — 2 フィールド構成:
- `Tab` / `j` / `↓` / `k` / `↑`: name / type 行間で focus 移動
- name 行 focus 中: 通常の文字入力 (textinput)
- type 行 focus 中: `h/←` / `l/→` でタイプ選択肢を循環
- `Enter`: 両方バリデート OK なら確定 → 新規 field を追加
- `Esc`: キャンセル

詳細 (ModeDetail):
- field 行で `enter` → ModeEditFieldValue (入力ポップアップ。ラベルは `<field name>:`)
- 既存の Title/Status/Files 操作は維持

### 5-6. footer ヒント (`footer.go`)

新規モード分のヒント帯を追加。

### 5-7. その他の変更

- `app.go`:
  - `isSettingMode` に新規モードを追加
  - 設定画面のレイアウト分岐 (status 系: 2 ペイン維持 / field 系: 3 ペイン) を追加
  - 各モードのキーハンドラを追加
  - field 削除時に `task.PurgeRemovedFieldValues` を呼んで永続化
- 新規 overlay 関数 `overlayFieldAddPopup` (multi-row 入力):
  - name 行 (textinput) + type 行 (selector) + バリデーション + ヒント
  - 既存の `overlayInputPopup` は流用せず、別関数として実装
- `cmd/task-man/main.go`:
  - `repo.Load()` の戻り値変更に追従
  - `tui.NewModel(repo, lr.Tasks, lr.Statuses, lr.Fields, yamlDir, lr.Config)` に
- `tasks.yaml` (実機サンプル):
  - `fields:` セクションと、いくつかのタスクへの値設定を追加

### 5-8. テスト

- `detailRows` の構築テスト: fields 0 件 / N 件 / Files との並び
- `app.go` のキーハンドラユニットテストはほぼ無いが、可能な範囲で field 追加・削除でカーソル位置が正しく動くことを確認 (既存の rows_test.go 流儀)
- ライブ検証関数 (`ValidateFieldNameChars` / `ValidateFieldTextValueChars`)

## 6. 実装順序 (フェーズ)

各フェーズ完了時点で `go vet ./... && go test ./... && go build ./...` をパス。

1. **ドメイン層**: `FieldDef` / `FieldDefList` / `TaskField` / `TaskFieldList` / `task.Fields` 追加 / `field_ops.go` + テスト。
2. **ストレージ層**: yaml 拡張 + `LoadResult` 化 + 補完 + 検証 + テスト。CLI 層 (main.go) の追従。
3. **詳細画面 (read-only)**: 動的 `detailRows` を導入し fields を表示するだけ。Files 区切り線位置の動的化。
4. **設定画面 3 ペイン化 + 追加 / 削除 / リネーム** (移動はまだ).
5. **field 移動 (m モード)** + cursor 追従.
6. **詳細画面で field 値編集** (`ModeEditFieldValue`).
7. **footer ヒント整備 + サンプル yaml 更新**.

## 7. リスク / 確認事項 (確定済み)

| 項目 | 決定内容 |
|---|---|
| Repository インターフェース変更 | `LoadResult` で bundle 化、既存呼び出しを一斉更新 |
| 値編集ポップアップのラベル | `<field name>:` (例: `締切日:`) |
| 追加時モーダル | 2 フィールド (name + type selector)。type は現状 text のみだが UI はセレクター実装 |
| status 設定中の画面レイアウト | 現行の 2 ペインレイアウトを維持 (3 ペイン化は field 選択時のみ) |
| 詳細画面で値が空の field | 空文字で表示 (ラベルだけが見える) |
| field name の使用可能文字 | タイトルと同じ禁止文字 (NUL/`/`/`:`) を流用 |
| field value (text) の禁止文字 | NUL のみ禁止。スラッシュ・コロンは許可 |

## 8. 影響を受けるファイル一覧

| ファイル | 変更概要 |
|---|---|
| `internal/task/field.go` | 新規: FieldDef / FieldDefList / TaskField / TaskFieldList |
| `internal/task/field_ops.go` | 新規: tasks 横断のスキーマ削除カスケード等 |
| `internal/task/field_test.go` | 新規 |
| `internal/task/field_ops_test.go` | 新規 |
| `internal/task/task.go` | `Task.Fields` 追加 |
| `internal/storage/repository.go` | `Repository` シグネチャ + `LoadResult` |
| `internal/storage/yaml.go` | top-level fields + per-task fields のシリアライズ・整合検証 |
| `internal/storage/yaml_test.go` | 新ケース追加 |
| `cmd/task-man/main.go` | LoadResult への追従 |
| `internal/tui/mode.go` | 新規モード 7 個追加 |
| `internal/tui/app.go` | Model 拡張 / ハンドラ / 設定画面 3 ペイン分岐 / `NewModel` シグネチャ |
| `internal/tui/detail.go` | detailRows 動的化 / field 行追加 / Files 罫線行を動的化 |
| `internal/tui/setting.go` | menu 2 項目化 / field/attribute ペイン |
| `internal/tui/footer.go` | 新規モードのヒント |
| `tasks.yaml` (サンプル) | top-level fields + 各タスクの値 |

---

承認後、フェーズ 1 (ドメイン層) から順に実装します。
