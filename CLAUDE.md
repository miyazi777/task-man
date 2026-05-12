# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## はじめに読む

- 仕様の一次ソースは `docs/spec/*.md`（`README.md` 配下に overview / architecture / data-model / domain / storage / cli / tui / layout / file-opener）。実装と齟齬があれば実装を優先し、spec を追従更新する。
- TUI 周りの非自明な落とし穴は `tasks/lessons.md` を最初に確認する。

## Build / Test

`Makefile` 経由が基本。バイナリは `./task-man`、メインパッケージは `./cmd/task-man`。Go の要求バージョンは `go.mod` 参照。

| Command | Purpose |
|---|---|
| `make build` | `./task-man` をビルド（`-ldflags "-s -w"`）|
| `make run` | ビルド後、CWD の `tasks.yaml` を開いて起動 |
| `make test` | `go test ./...` |
| `make vet` / `make fmt` / `make tidy` | `go vet` / `go fmt` / `go mod tidy` |
| `make install` | `$GOPATH/bin` にインストール |

単一テスト / サブテストを走らせる:

```bash
go test ./internal/task/...
go test -run TestTaskValidate ./internal/task
go test -run TestTaskValidate/valid ./internal/task
```

`internal/tui` には View 文字列の差分を見るレンダリング系テスト（`popup_test.go`, `overlay_test.go`, `rows_test.go`, `detail_test.go` ほか）がある。TUI を触ったら `go test ./internal/tui/...` を必ず通す。

## 起動

`./task-man` は CWD の `tasks.yaml` を読む（無ければ空ファイルを自動生成）。フラグ:

- `-t, --tasks <path>`: 別の yaml を使う。`~` と `~/...` のみ展開（`~user` 形式は非対応）。
- `-i, --init`: yaml をデフォルト 3 ステータスのみで再生成し、`data_base_directory` 配下の `task-<id>/` を全削除。`y/N` 確認あり。既存 `AppConfig`（特に `data_base_directory` / file-opener / layout）は引き継がれる。

## アーキテクチャの骨格

詳細は `docs/spec/architecture.md`。要点だけ:

- **単一 yaml 永続化**: タスク・ステータス・拡張項目・タグ・タグ・applications・file_opener・layout・`data_base_directory` がすべて 1 ファイルに収まる。`storage.LoadResult` がそのスナップショット。
- **Repository 抽象**: `storage.Repository` の唯一の実装は `YAMLRepository`。`Load` 時に id / position 等が補完された場合は即 `Save` を呼んで yaml を整える。
- **依存方向** (片方向):
  ```
  cmd/task-man → cli, storage, task, tui
  internal/tui → storage, task
  internal/storage → task
  internal/task : 他 internal に依存しない純粋ドメイン
  internal/cli  : 他 internal に依存しない
  ```
  新しいバリデーション / 並べ替えロジックは `internal/task` に置けるか最初に検討する。
- **TUI モードマシン**: `internal/tui/mode.go` の `Mode` 列挙で画面・入力フェーズを切替。同じキーがモードごとに別の意味を持つので、キーバインドを追加するときは必ず `keys.go` と `app.go` の該当 `case` を両方確認する。
- **書き込みは必ず `Model.persist()` 経由**: collapse 状態を同期したうえで `repo.Save()` を呼ぶ。`Save` は `atomicWrite`（`.task-man-*.tmp` → `os.Rename`）で原子的に書く。
- **添付ファイル**: `<yamlDir>/<data_base_directory>/task-<id>/` 配下。FS 操作は `internal/storage/data.go` の関数群に集約されているので、新規追加もここに足す。

## i18n / ファイル言語

- README は `README.md` (英) と `README.ja.md` (日) の 2 本立て。`.githooks/pre-commit` が片方だけステージされた状態で警告を出す（ブロックはしない）。クローンごとに有効化:
  ```bash
  git config core.hooksPath .githooks
  ```
- リポジトリ内の Go コメント / Makefile help / `runInit` の stdout は日本語と英語が混在する。**新規コメント・メッセージは編集対象ファイル既存の言語に合わせる**（ファイル単位で一貫しているため）。

## ローカル除外

`private/`, `tasks.yaml`, `tasks.back.yaml`, `task2.yaml` は `.gitignore` 済みの個人データ。テストフィクスチャや再現環境として参照しない。
