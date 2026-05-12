# ファイルオープナー仕様

タスクごとの添付ファイル (`task-<id>/<file>`) を、拡張子に応じて外部アプリケーションで開く仕組み。設定は `tasks.yaml` の `applications` と `file_opener` で行い、UI 操作は `internal/tui/fileopener.go` と `internal/tui/app.go` の `ModeFileOpener` で行う。

## 設定モデル

### `applications`

外部アプリケーションのレジストリ。1 件あたり id / name / run を持つ。

```yaml
applications:
  - application:
      id: 1
      name: editor
      run: $EDITOR
  - application:
      id: 2
      name: md-viewer
      run: glow
```

- `run` は `os.ExpandEnv` で環境変数を展開する。
- 空白区切りでコマンドと引数を分けるため、`nvim --noplugin` のような表記も使える。

### `file_opener`

拡張子から `applications` への対応表。

```yaml
file_opener:
  - opener:
      extension: "md"
      applications: [1, 2]
      default_app: 1
```

- `extension`: ドット無し / 小文字。先頭の `.` は除去される。空文字はロードエラー。
- `applications`: 候補となる `application.id` の配列。順序が UI モーダルでの候補表示順。
- `default_app`: `enter` 押下時の即時起動に使う `application.id`。`0` または未指定なら `$EDITOR` フォールバック。

## 起動キー

- 詳細画面の Files 行で `enter`: `default_app` (もしくは `$EDITOR`) を即起動 (`openCurrentFileWithDefault`)。
- 詳細画面の Files 行で `o`: 候補から選ぶモーダル (`openCurrentFileWithPicker` → `ModeFileOpener`)。
- 候補が 1 件以下なら、モーダルを開かずそのまま起動する (`openCurrentFile` の中で分岐)。

## 候補解決ロジック (`internal/tui/fileopener.go`)

- `resolveDefaultApp(fileName, apps, openers)`: 拡張子に該当する `default_app` を返す。該当無し / `default_app == 0` / id 存在しない場合は ok=false。
- `resolveFileOpenerCandidates(fileName, apps, openers)`: 拡張子に対応する `applications` 配列を、`apps` 内で実在する id だけに絞って返す。順序は `file_opener.applications` の指定順。
- 大文字小文字は無視 (`foo.MD` と `foo.md` は同じ)。

## フォールバック

- 該当拡張子のエントリが無い、または `default_app` が解決できない場合は `$EDITOR` を起動する。
- `$EDITOR` 未設定で解決失敗した場合は `m.saveErr` にエラーメッセージを格納して UI に表示する。

## ModeFileOpener の操作

- `j`/`k`: 候補カーソル移動。
- `enter`: 選択したアプリケーションでファイルを起動。alt screen を抜けて `tea.ExecProcess` を実行する。
- `esc`: モーダルを閉じて元のモード (`prevMode`) に戻る。

## 設定画面での編集

設定画面 (`;` → `s`) の `application` / `file_opener` サブメニューで以下が編集できる:

- application の追加 / 編集 (name / run) / 並べ替え / 削除。
- file_opener の追加 / 拡張子変更 / applications の multi-select 編集 / default_app の選択 / 並べ替え / 削除。

application 削除時は `FileOpeners` 内で参照していた id を自動的に除去し、`default_app == 削除 id` の場合は `0` に戻す (`internal/tui/app.go` の `ModeSettingApplicationDeleteConfirm`)。

## 関連実装

| ファイル | 役割 |
|---|---|
| `internal/tui/fileopener.go` | 候補解決ロジック (`resolveFileOpenerCandidates` / `resolveDefaultApp`) |
| `internal/tui/editor.go` | `tea.ExecProcess` ベースのアプリ起動 / `$EDITOR` フォールバック / URL ブラウザ起動 |
| `internal/storage/yaml.go` | `loadApplications` / `loadFileOpeners` での YAML マッピングと検証 |
| `internal/storage/config.go` | `Application` / `FileOpener` 型定義 |
