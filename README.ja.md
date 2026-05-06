[English](./README.md) | [日本語](./README.ja.md)

# task-man

ターミナルで動くタスク管理 TUI アプリケーション。すべてのデータを単一の `tasks.yaml` に保存し、タスク本文や添付ファイルは隣接するディレクトリで管理します。

[Bubble Tea](https://github.com/charmbracelet/bubbletea) / [Lip Gloss](https://github.com/charmbracelet/lipgloss) を利用した日本語フレンドリーな UI。

## 特徴

- **シングルファイル永続化**: タスク・ステータス・拡張項目・タグ・レイアウトをすべて 1 つの `tasks.yaml` に書き出し
- **カスタムステータス**: `todo / doing / done` 既定に加え、yaml で自由にステータス・色を定義可能
- **サブタスク**: 最大 5 階層のネスト
- **タグ**: タスクに最大 5 個まで付与、設定画面でタグ管理
- **拡張項目 (fields)**: text / date / URL の各タイプを task に追加可能
- **ゴミ箱機能**: 削除タスクの一時保管・復元・完全削除
- **ファイルプレビュー**: タスク添付ファイル (md / txt / csv) を右ペインに表示
- **ファイルオープナー**: 拡張子ごとに起動するアプリを yaml で設定、ファイルから直接起動
- **レイアウト調整**: タスクリスト画面の各ペインの比率をインタラクティブに変更し永続化
- **複数ワークスペース**: `-t` オプションで起動時に異なる `tasks.yaml` を指定可能

## スクリーンショット

`docs/mockups/` にモックアップ SVG (`01-list-focused.svg` など) を収録しています。

## 動作要件

- Go 1.26 以上 (`go.mod` 参照)
- 動作確認済み環境: Linux

## ビルド・インストール

```bash
git clone https://github.com/<your-account>/task-man.git
cd task-man

# ローカルビルド (./task-man が生成される)
make build

# $GOPATH/bin にインストール
make install
```

主な Make ターゲット:

| ターゲット | 動作 |
|---|---|
| `make build` | `./task-man` バイナリをビルド |
| `make run` | ビルドして起動 |
| `make test` | 全パッケージのテスト実行 |
| `make vet` | `go vet ./...` |
| `make fmt` | `go fmt ./...` |
| `make tidy` | `go mod tidy` |
| `make clean` | バイナリ削除 |

## 起動

カレントディレクトリの `tasks.yaml` を読みます。存在しなければ自動で空ファイルを作成します。

```bash
./task-man
```

### 起動オプション

| フラグ | 内容 |
|---|---|
| `-t`, `--tasks <path>` | 任意の `tasks.yaml` を指定 (`~/...` の展開対応) |
| `-i`, `--init` | yaml をデフォルト 3 ステータス (todo/doing/done) のみで初期化し、タスクデータディレクトリを全削除 (要 y/N 確認) |

#### 例: 共有 tasks.yaml をエイリアスから参照する

```bash
# 任意の作業ディレクトリから ~/private/tasks.yaml を使う
alias tm='task-man -t ~/private/tasks.yaml'
```

#### 例: 状態のリセット

```bash
./task-man -i        # 確認ありで初期化
./task-man -t ~/private/tasks.yaml -i
```

## キー操作

### タスクリスト画面

| キー | 動作 |
|---|---|
| `k` / `↑` , `j` / `↓` | カーソル上下 |
| `l` / `→` | ステータス・タスクの展開 |
| `h` / `←` | 折りたたみ |
| `enter` | タスク詳細へ遷移 |
| `a` | 新規タスク (ステータス行) / サブタスク (タスク行) |
| `d` | タスクをゴミ箱へ (ゴミ箱ビューでは完全削除) |
| `m` | 移動モード開始 / 確定 |
| `o` | operation モード (`t`=title, `s`=status, `g`=tag, `f`=files) |
| `;` | prefix モード |
| `q` | 終了 |

### prefix モード (`;` の後)

| キー | 動作 |
|---|---|
| `t` | ゴミ箱ビュー表示トグル |
| `s` | 設定画面へ遷移 |
| `l` | レイアウト調整モードへ遷移 |
| `esc` | キャンセル |

### タスク詳細画面

| キー | 動作 |
|---|---|
| `k` / `j` | 行カーソル上下 |
| `enter` | 各行の編集ポップアップを開く / Files 行ではファイルオープナー起動 (default_app) |
| `o` | url 型項目: ブラウザで開く / Files 行: 拡張子に応じたアプリ選択モーダル |
| `a` | Files セクションでファイル新規作成 |
| `r` | Files セクションでファイルリネーム |
| `d` | Files セクションでファイル削除 |
| `;` | prefix モード |
| `esc` | リストへ戻る |

### レイアウト調整モード (`;` → `l`)

突入時のフォーカス (タスクリスト / タスク詳細 / ファイルリスト) によって縦操作の対象が決まります。

| キー | 動作 |
|---|---|
| `h` / `l` | タスクリスト幅 縮小 / 拡大 |
| `j` / `k` (詳細 / ファイルリストフォーカス時) | 高さ拡大 / 縮小 |
| `enter` | 確定 (yaml に保存) |
| `esc` | 突入前の値に戻して終了 |

### 設定画面 (`;` → `s`)

左メニューで `general` / `status` / `field` / `application` / `file_opener` を切り替え。

- **general**: yaml パスの確認、`data_base_directory` の編集
- **status**: ステータス追加・名称変更・色変更・削除・並べ替え
- **field**: 拡張項目の追加・編集・並べ替え・削除
- **application**: ファイルオープナー機能で利用するアプリの登録・編集
- **file_opener**: 拡張子ごとに使用する application と default_app の対応を編集

## tasks.yaml の構造

`task-man` は以下の構造で yaml を読み書きします。各セクションは省略可能 (Statuses 以外は空でも OK)。

```yaml
applications:
  - application:
      id: 1
      name: editor
      run: $EDITOR        # 環境変数 or PATH 上のコマンド
  - application:
      id: 2
      name: md-viewer
      run: md-viewer

file_opener:
  - opener:
      extension: "md"
      applications: [1, 2]
      default_app: 1     # enter 押下時に起動するアプリ (省略時は $EDITOR)

data_base_directory: ./tasks_data   # タスク添付ディレクトリの基準 (省略時は yaml と同階層)

layout:
  main:
    task_list:
      width: 0.6      # 0.0〜1.0 の比率 (画面横幅に対する task list の占有比)
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
      name: 締切日
      position: 1

tags:
  - tag:
      id: 1
      name: urgent
      color: "#f38ba8"

tasks:
  - task:
      id: 1
      title: タスク 1
      status_id: 1
      position: 1
      tags: [1]
      fields:
        - field:
            id: 1
            field_id: 1
            value: "2026-05-01"
```

### data_base_directory とタスク添付

タスクごとに `task-<id>/` ディレクトリが生成され、その中に `memo.md` 等の添付ファイルが置かれます。
配置先は `data_base_directory` 設定に従います (未設定なら yaml と同じ階層)。

例: `data_base_directory: ./tasks_data` のとき、ID=1 のタスクの添付は `./tasks_data/task-1/` 以下。

## ディレクトリ構成

```
.
├── cmd/task-man         # エントリポイント (main.go)
├── internal/cli         # CLI 引数パース
├── internal/storage     # tasks.yaml の読み書き / 添付ファイル操作
├── internal/task        # ドメイン (Task / Status / Field / Tag)
├── internal/tui         # Bubble Tea Model/View/Update
├── docs/mockups         # 画面モックアップ (SVG)
├── Makefile
├── go.mod
└── README.md
```

## 開発者向け

`.githooks/` に `README.md` / `README.ja.md` の片方だけがステージされた状態でコミットしようとした際に警告を出す pre-commit フックを同梱しています (警告のみでコミットは中断しません)。クローンごとに一度有効化してください:

```bash
git config core.hooksPath .githooks
```

## 謝辞

- TUI 基盤: [Bubble Tea](https://github.com/charmbracelet/bubbletea) / [Bubbles](https://github.com/charmbracelet/bubbles) / [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- カラーテーマ: [Catppuccin](https://catppuccin.com/) ベース
