# task-man 実装計画

## 概要
Go + Bubble Tea で TUI タスク管理ツール `task-man` を実装する。
仕様は `idea.md`、UI モックは `docs/mockups/*.svg` を参照。

## 設計判断 (確定済み)

1. **モジュールパス**: `github.com/miyazi777/task-man`
2. **配置**: 空の `task-man/` サブディレクトリは削除し、プロジェクトルートに Go モジュールを作成
3. **保存タイミング**: タスク追加・ステータス変更ごとに即 yaml に書き戻し
4. **タスク ID**: yaml に `id: int` フィールドを追加。タスク作成時に「既存最大 ID + 1」で採番 (空なら 1 から)
5. **空状態**: タスク 0 件時は何も表示しない

## yaml スキーマ (確定)

```yaml
tasks:
  - task:
      id: 1
      title: 設計書を書く
      status: todo
```

---

## フェーズ 1: プロジェクト初期化 ✅

- [x] 空の `task-man/` サブディレクトリを削除
- [x] `go mod init github.com/miyazi777/task-man` でモジュール初期化
- [x] `.gitignore` 作成 (Go 標準: `task-man` バイナリ, `*.test`, `vendor/` 等)
- [x] 依存追加
  - [x] `github.com/charmbracelet/bubbletea` v1.3.10
  - [x] `github.com/charmbracelet/bubbles` v1.0.0 (textinput, key)
  - [x] `github.com/charmbracelet/lipgloss` v1.1.0 (スタイリング・レイアウト)
  - [x] `gopkg.in/yaml.v3` v3.0.1 (yaml 読み書き)
  - [x] `github.com/spf13/pflag` v1.0.10 (`--task` 対応)
- [x] ディレクトリ雛形を作成
  ```
  cmd/task-man/
  internal/task/
  internal/storage/
  internal/cli/
  internal/tui/
  ```

## フェーズ 2: ドメイン層 (`internal/task`) ✅

- [x] `internal/task/status.go`
  - [x] `type Status string` 定義 (`StatusTodo`, `StatusDoing`, `StatusDone`)
  - [x] `(s Status) Next() Status` / `(s Status) Prev() Status` 実装 (端で止まる挙動)
  - [x] `ParseStatus(string) (Status, error)` (yaml ロード時の検証用)
- [x] `internal/task/task.go`
  - [x] `type Task struct { ID int; Title string; Status Status }`
  - [x] バリデーション関数 `(t Task) Validate() error` (空タイトル禁止、ID > 0)
- [x] `internal/task/list.go`
  - [x] ID 採番ヘルパ `NextID(tasks []Task) int` (max + 1、空なら 1)
- [x] テスト
  - [x] `status_test.go`: Next/Prev の境界、ParseStatus の不正値
  - [x] `task_test.go`: Validate
  - [x] `list_test.go`: NextID (空、連番、欠番、最大値ケース)

## フェーズ 3: 永続化層 (`internal/storage`) ✅

- [x] `internal/storage/repository.go`
  - [x] `type Repository interface { Load() ([]task.Task, error); Save([]task.Task) error }`
- [x] `internal/storage/yaml.go`
  - [x] yaml の構造マッピング型定義 (`tasks:` -> `- task: { id, title, status }`)
  - [x] `type YAMLRepository struct { Path string }` 実装
  - [x] `Load()`: ファイルなしならエラー (CLI 層で「作成 or 終了」を判断する)
  - [x] `Save()`: アトミック書き込み (一時ファイル → rename) で破損リスクを避ける
  - [x] ロード時に id 重複・欠落を検証 (重複ならエラー、欠落 (=0) は不正値として扱う)
- [x] テスト
  - [x] `yaml_test.go`: Load/Save ラウンドトリップ、不正な status 値、空ファイル、id 重複検出

## フェーズ 4: CLI 層 (`internal/cli`) ✅

- [x] `internal/cli/args.go`
  - [x] `pflag` で `-t` / `--task` 受付
  - [x] `Parse()` と `EnsureFile()` を分離
- [x] テスト: フラグ解析・存在チェック・自動作成の各分岐

## フェーズ 5: TUI 層 (`internal/tui`) ✅

### 5.1 基盤
- [x] `keys.go`: `bubbles/key` でキーマップ定義
- [x] `styles.go`: Lip Gloss スタイル + ステータス色 (todo: グレー / doing: オレンジ / done: グリーン)

### 5.2 ルートモデル
- [x] `mode.go`: `Mode` 列挙
- [x] `app.go`: `Model` の Init/Update/View、モード別キー処理、終了確認

### 5.3 各画面ビュー
- [x] `list.go`: 左ペイン描画 (カーソル、ステータス色、フォーカス状態の dim 表示、入力中プレースホルダ)
- [x] `detail.go`: 右ペイン (通常詳細・新規入力モードの両方)
- [x] `newtask.go`: textinput 初期化ヘルパ
- [x] `footer.go`: モード別ヒント、quit プロンプト
- [x] レイアウト: `app.go` の View で左右分割 + フッター + WindowSizeMsg 対応
  - [ ] 端末リサイズ (`tea.WindowSizeMsg`) に追従

## フェーズ 6: エントリポイント (`cmd/task-man/main.go`) ✅

- [x] `cli.Parse()` / `cli.EnsureFile()` 呼び出し
- [x] フラグ指定ファイルが存在しなければ `os.Exit(1)` でエラー終了
- [x] フラグ未指定で `tasks.yaml` がなければ空ファイル作成
- [x] `storage.YAMLRepository` を生成し、`tui.NewModel(repo, tasks)` で起動
- [x] エラー時の終了コード処理

## フェーズ 7: 動作検証

- [x] `go build ./...` 成功
- [x] `go vet ./...` / `go test ./...` パス
- [x] 非インタラクティブな確認
  - [x] `tasks.yaml` なしのディレクトリで起動 → 自動生成 (`/tmp/task-man-smoke` で確認)
  - [x] `-t /tmp/exists.yaml` (存在) → 読み込めること (TUI 直前まで通過)
  - [x] `-t /tmp/missing.yaml` (不在) → エラー終了 (`error: file not found: ...` exit 1)
  - [x] 不正 status の yaml → エラー終了 (`error: tasks[0]: invalid status: "invalid_status"`)
- [ ] **インタラクティブテスト (ユーザー確認待ち)** — 実 TTY が必要
  - [ ] `a` → タイトル入力 → `enter` で保存 → ファイル反映確認
  - [ ] カーソル移動 → `enter` → `j/k` でステータス変更 → ファイル反映確認
  - [ ] `q` → `y` で終了、`q` → `n` で復帰
  - [ ] 端末リサイズで崩れないこと
  - [ ] モック 01〜04 との見た目比較

## フェーズ 8: 仕上げ

- [ ] README.md (簡潔: 用途、ビルド方法、操作キー)
- [ ] `tasks/lessons.md` に今回得た教訓を追記 (該当あれば)

---

## 進め方

- 各フェーズ末で動かせる状態を保つ (フェーズ 2 → ドメインのテスト緑、フェーズ 3 → yaml ラウンドトリップ確認、フェーズ 5 → TUI 起動確認、というように段階的に)
- フェーズ 5 (TUI) は分量が多いので、`5.1 基盤` → `5.2 ルートモデル + 最小 List 表示` を先に通し、`5.3` を画面ごとに段階的に積む
- 詰まった箇所はサブエージェント (Explore / general-purpose) に Bubble Tea のパターン調査を委譲する

## レビュー (実装後に追記)

(空 — 実装完了後に振り返りを書く)
