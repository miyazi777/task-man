# アーキテクチャ

## ディレクトリ構成

```
.
├── cmd/task-man          # エントリポイント (main.go)
├── internal/cli          # CLI 引数パース
├── internal/storage      # tasks.yaml の読み書き / 添付ファイル操作
├── internal/task         # ドメイン (Task / Status / Field / Tag / Trash / Move)
├── internal/tui          # Bubble Tea Model/View/Update
├── docs                  # README 用画像 / 仕様書 (本ディレクトリ)
├── tasks                 # 開発タスクメモ (ユーザー用)
├── .githooks             # pre-commit フック (README 同期チェック)
├── Makefile
├── go.mod / go.sum
├── README.md / README.ja.md
```

## レイヤと責務

- `cmd/task-man`: 起動・初期化フロー (`run()` / `runInit()`)。CLI 解析、YAML ロード、TUI モデル生成、Bubble Tea プログラムの起動を担当する (`cmd/task-man/main.go`)。
- `internal/cli`: `-t` / `--tasks` 等のフラグ解析、`~` 展開、ファイル存在チェックと自動生成 (`internal/cli/args.go`)。
- `internal/storage`: yaml の読み書き (`yaml.go`)、`AppConfig` / `Application` / `FileOpener` / `LayoutConfig` の型定義 (`config.go`)、`Repository` インタフェース (`repository.go`)、タスク添付ファイル管理 (`data.go`)。
- `internal/task`: ドメインモデルとビジネスルール。`Task` (`task.go`)、`Status` (`status.go`)、`FieldDef` / `TaskField` (`field.go`)、`Tag` (`tag.go`)、ゴミ箱操作 (`trash.go`)、並び替え操作 (`list.go`, `move.go`)、フィールド一括操作 (`field_ops.go`)。
- `internal/tui`: Bubble Tea の Model / View / Update を実装。モード列挙 (`mode.go`)、キーバインド (`keys.go`)、リスト描画 (`browser.go` / `list.go` / `rows.go`)、詳細描画 (`detail.go`)、設定画面 (`setting.go`)、各種ポップアップ (`overlay.go`, `popup_test.go`)、ファイルプレビュー (`preview.go`)、レイアウト調整 (`layout.go`)、タグピッカー (`tagpicker.go`)、ファイルオープナー (`fileopener.go`)、エディタ起動 (`editor.go`)、フッター (`footer.go`)、カラー (`styles.go`)、カレンダー (`calendar.go`)。

## 依存方向

- `cmd/task-man` → `internal/cli`, `internal/storage`, `internal/task`, `internal/tui`
- `internal/tui` → `internal/storage`, `internal/task`
- `internal/storage` → `internal/task`
- `internal/task`: 他 internal パッケージに依存しない (純粋なドメイン)
- `internal/cli`: 他 internal パッケージに依存しない

## Repository 抽象

```go
// internal/storage/repository.go
type Repository interface {
    Load() (LoadResult, error)
    Save(lr LoadResult) error
}
```

- 唯一の実装は `YAMLRepository` (`internal/storage/yaml.go`)。
- 永続化単位は `LoadResult` 1 つ (タスク・ステータス・フィールド・タグ・設定をまとめた構造体)。

## 起動フロー

1. `cli.Parse(os.Args[1:])` で引数を解析する。
2. `filepath.Abs` で yaml パスを絶対化する (起動後の CWD 変化に影響されないため)。
3. `--init` 指定時は `runInit()` を実行し、確認プロンプト・データディレクトリ削除・yaml 初期化を行ってから終了する。
4. 通常起動では `cli.EnsureFile` でファイルを保証し、`storage.NewYAMLRepository(absPath)` → `repo.Load()` で `LoadResult` を取得する。
5. `tui.NewModel(...)` でモデルを構築し、`tea.NewProgram(..., tea.WithAltScreen())` を起動する。

## 永続化の同期方針

- 編集系の操作はモデル経由で `Model.persist()` を呼び、`Status.Collapsed` / `Task.Collapsed` を同期したうえで `repo.Save()` で書き戻す (`internal/tui/app.go`)。
- 書き戻しは `atomicWrite()` (`internal/storage/yaml.go`) で `<dir>/.task-man-*.tmp` を作って `os.Rename` で原子的に置換する。
- 読み込み時に補完 (id/position の自動採番、status の自動補完等) が発生した場合、`Load()` の中で即座に `Save()` を呼んで yaml を整える。
