# 概要

## プロダクトの目的

task-man は、単一の YAML ファイル (`tasks.yaml`) にすべての状態を保存するターミナル UI のタスク管理アプリケーションである。日本語入力に対応した UI を [Bubble Tea](https://github.com/charmbracelet/bubbletea) / [Lip Gloss](https://github.com/charmbracelet/lipgloss) で構築し、Vim ライクなキー操作で運用できる。

## 主な特徴

- シングルファイル永続化: タスク・ステータス・拡張項目・タグ・レイアウトをすべて 1 つの `tasks.yaml` に書き出す (`internal/storage/yaml.go`)。
- カスタムステータス: デフォルトの `todo / doing / done` に加え、yaml で自由にステータスと色を定義できる (`internal/task/status.go`)。
- サブタスク: 最大 5 階層 (top-level + 4) のネスト構造を許容する。`MaxNestDepth=4` で制御される (`internal/task/task.go`)。
- タグ: タスクあたり最大 5 個。色 (HEX) を持ち、`#name` 形式で表示される (`internal/task/tag.go`)。
- 拡張項目 (fields): `text` / `date` / `url` の 3 種類の型に対応 (`internal/task/field.go`)。
- ゴミ箱: 削除タスクを `is_trash_box=true` で論理削除し、復元または完全削除する (`internal/task/trash.go`)。
- ファイルプレビュー: タスクの添付ファイル (md / txt / csv) を右ペインで表示する (`internal/tui/preview.go`)。
- ファイルオープナー: 拡張子ごとに任意の外部アプリケーションを起動する (`internal/tui/fileopener.go`, `internal/storage/config.go`)。
- レイアウト調整: タスクリスト画面のペイン比率をインタラクティブに編集して永続化できる (`internal/tui/layout.go`)。
- マルチワークスペース: `-t` フラグで任意の `tasks.yaml` を指定し、ホームディレクトリ展開 (`~/...`) に対応する (`internal/cli/args.go`)。

## 動作要件

- Go 1.26.2 以上 (`go.mod`)
- Linux で動作確認済み (`README.md`)
- 外部アプリケーションを起動するため、ユーザー側で対応するコマンド (例: `$EDITOR`, ブラウザ) が PATH 上に存在する必要がある。

## 配布物

- バイナリ名: `task-man`
- メインパッケージ: `cmd/task-man`
- Make ターゲット: `build` / `run` / `test` / `fmt` / `vet` / `tidy` / `install` / `clean` (`Makefile`)
- 依存関係: `bubbletea v1.3.10`, `bubbles v1.0.0`, `lipgloss v1.1.0`, `pflag v1.0.10`, `yaml.v3 v3.0.1` (`go.mod`)
