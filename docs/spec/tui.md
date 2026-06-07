# TUI 仕様

`internal/tui` パッケージ。Bubble Tea の Model/View/Update をベースに、`Mode` 列挙でスクリーンと入力フェーズを切り替える。

## 画面構成

起動直後はタスクリスト画面 (左ペイン) と詳細＋ファイルプレビュー画面 (右ペイン) の 2 ペイン構造。

```
┌────────────────────────┬──────────────────────┐
│ Status / Task list     │ Task detail          │
│  (左ペイン)            │  (上段)              │
│                        ├──────────────────────┤
│                        │ File list            │
│                        │  (中段)              │
│                        ├──────────────────────┤
│                        │ File preview         │
│                        │  (下段)              │
└────────────────────────┴──────────────────────┘
│ Footer (key hints / saveErr 表示)              │
```

- 左ペイン: ステータス見出し + 配下のタスクをツリー状に描画 (`internal/tui/rows.go`, `list.go`, `browser.go`)。
- 右ペイン: Title / Status / Tags / 拡張項目 / Files セクションを縦に並べる (`internal/tui/detail.go`)。Files 行下に file list と file preview を表示。
- Files セクション見出しは `Files: <タスクディレクトリ絶対パス>` の形式 (`renderFilesHeader`)。パスはカーソル下のタスクのデータディレクトリ (`storage.TaskDir`) で、右ペイン幅に収まらない場合は `ansi.Truncate` で末尾を `…` に省略する。
- File preview (下段): カーソルがファイル行のときは内容を先頭 256KiB まで表示する (`.md` / `.txt` のみ。それ以外は `Preview not available`)。カーソルがディレクトリ行のときはそのディレクトリ直下のエントリ一覧 (Name 昇順、サブディレクトリは末尾 `/` 付き) を表示する (`renderDirPreview`)。空ディレクトリは `(empty)`、読み込み失敗は `(read error)` を表示する。折りたたみ状態とは独立で、`h` で閉じたディレクトリでもプレビューには直下の全エントリが出る。

## モード一覧 (`internal/tui/mode.go`)

主なモードと役割:

- `ModeList`: タスクリスト画面のデフォルト。
- `ModeDetail`: タスク詳細画面。
- `ModeNewTask` / `ModeNewSubtask`: 新規タスク / サブタスク入力ポップアップ。
- `ModeEditTitle` / `ModeEditStatus`: 詳細または operation 経由の編集。
- `ModeAddFile` / `ModeRenameFile` / `ModeDeleteFileConfirm`: Files セクション操作。
- `ModeTrashConfirm` / `ModeDeleteTaskConfirm`: 削除確認 (通常 / 完全)。
- `ModeQuitConfirm`: 終了確認。
- `ModeMove`: タスク移動モード。`m` または `o`→`m`、確定 = `enter` / 取消 = `esc`。
- `ModePrefix`: `;` 押下後の二段階キー入力。
- `ModeSetting*`: 設定画面。サブメニューは `general` / `status` / `field` / `application` / `file_opener`。
- `ModeEditFieldValue` / `ModeEditFieldDateValue`: 拡張項目値の編集。
- `ModeOperation`: タスクリストで `o` を押した直後の operation 入力待ち (`t`/`s`/`g`/`f` のいずれか)。
- `ModeTagPicker` / `ModeTagColorPicker` / `ModeTagPickerRename` / `ModeTagPickerDeleteConfirm`: タグピッカー関連モーダル。
- `ModeLayout`: レイアウト調整。
- `ModeFileOpener`: ファイルオープナー候補から選ぶモーダル。

文字列表現は `Mode.String()` で取得可能。

## キーバインド

定義は `internal/tui/keys.go`。`q` で終了確認、`esc` で戻る、`enter` で確定 / 詳細表示が基本。同じキーでもモードによって役割が変わる。

### タスクリスト画面 (`ModeList`)

| キー | 動作 |
|---|---|
| `k` / `↑`, `j` / `↓` | カーソル上下 |
| `l` / `→` | ステータス / タスクの展開 (`applyCollapseChange(false)`) |
| `h` / `←` | 折りたたみ |
| `enter` | タスク詳細画面へ遷移 (`ModeDetail`) |
| `a` | 新規タスク (ステータス行のとき) / サブタスク (タスク行のとき) |
| `d` | ゴミ箱へ移動 (通常リスト) / 完全削除 (ゴミ箱ビュー) |
| `r` | ゴミ箱ビューでタスクを復元 |
| `m` | 移動モードの開始 / 確定 (`ModeMove`) |
|| `o` | operation モード (`r`=rename, `s`=status, `g`=tag, `f`=files) |
| `p` | カーソル位置のタスクのデータディレクトリ絶対パスをクリップボードへコピー |
| `R` | 現在カーソル行のタスクのファイル一覧を再読込 (`withFilesRefreshed`)。外部 (Finder / mv) で発生した変更を反映するためのキー |
| `;` | prefix モード (`ModePrefix`) |
| `q` | 終了確認 |

### Prefix モード (`;` の後)

| キー | 動作 |
|---|---|
| `t` | ゴミ箱ビューのトグル (`viewTrash`) |
| `s` | 設定画面へ遷移 (`ModeSetting`) |
| `l` | レイアウト調整モードへ遷移 (`ModeLayout`) |
| `esc` | キャンセル |

### タスク詳細画面 (`ModeDetail`)

| キー | 動作 |
|---|---|
| `k` / `j` | 行カーソル上下。Files 行内ではファイルカーソルが上下する。 |
| `l` / `→` | Files 行のディレクトリを展開 (タスクリストと同じ表現)。葉ファイルでは no-op。開閉状態は `task.collapsed_dirs` として yaml に永続化される (再起動・別タスクからの戻りでも復元)。 |
| `h` / `←` | Files 行のディレクトリを折りたたみ。葉ファイルでは no-op。`l` と同じく yaml に永続化される。 |
| `enter` | Title/Status は編集ポップアップ、Tags 行はタグピッカー、Field 行は型別ポップアップ (text/url) または日付カレンダー (date)、Files 行はファイルなら `default_app` で開き、ディレクトリなら折りたたみトグル |
| `o` | url 型項目をブラウザで開く (`openURLInBrowser`)、Files 行 (ファイル) は拡張子に応じた候補モーダルを開く。ディレクトリ行では no-op |
| `f` | Files 行カーソル時、OS のデフォルトファイラーで該当ファイル / ディレクトリを開く (`openInOSFileManager`)。macOS / Windows ではファイル選択状態 (reveal) で親フォルダを開き、Linux 系はファイルの場合 `xdg-open` で親ディレクトリを開く |
| `a` | Files 行で新規エントリ作成 (`ModeAddFile`)。カーソルがディレクトリ行ならその直下、ファイル行ならその親ディレクトリ。入力末尾が `/` ならディレクトリとして作成、それ以外はファイル |
| `r` | Files 行でリネーム (`ModeRenameFile`)。ファイル / ディレクトリ どちらも対象 |
| `d` | Files 行で削除確認 (`ModeDeleteFileConfirm`)。ファイルは単体削除、ディレクトリは再帰削除 (確認モーダルに配下も消える旨を明示) |
| `p` | Files 行ならカーソル位置のファイル / サブディレクトリの絶対パス、それ以外の行ならタスクのデータディレクトリ絶対パスをクリップボードへコピー |
| `R` | ファイル一覧を再読込 (`withFilesRefreshed`)。Files 行以外でも有効。外部 (Finder / mv) で発生した変更を反映するためのキー |
| `;` | prefix モードへ |
| `esc` | リストへ戻る |

ファイルリストはタスクディレクトリ配下を再帰的に表示し、ディレクトリの折りたたみ状態はセッション内・現在タスクに限り保持される (タスク切り替えでリセット)。マーカーは hasChildren を持つディレクトリのみ `+ `/`- ` を出し、葉ファイルや空ディレクトリは空白 2 cell。

Files セクションは fsnotify で現在カーソル位置のタスクディレクトリを監視し、外部からの `touch` / `mv` / `rm` を 100ms の debounce 後に自動反映する (`internal/tui/watcher.go` の `TaskDirWatcher`)。タスク切替時 (`withFilesRefreshed` 内の `filesTaskID` 変化検出) には旧 watch を解放して新ディレクトリへ Add し直し、status 行カーソル時には watch を完全に解放する。Linux の inotify が再帰 watch をサポートしないため、初回 Switch と Create イベント時にサブディレクトリを `filepath.Walk` で個別に Add する。watcher の作成 / Add に失敗した場合は静かに諦め、`R` キーでの手動 refresh が引き続き fallback として機能する。アプリ終了時には `main.go` の defer で watcher を Close する。

### 移動モード (`ModeMove`)

- 開始: `m` で対象タスクを選択。スナップショットを `moveSnapshot` に退避する。
- `k` / `j`: タスクを 1 行上下に移動 (`task.MoveTaskUp` / `MoveTaskDown`)。先頭/末尾を超えると上下のステータスへ移動。
- `l` / `h`: タスクをインデント / アウトデント (`IndentTask` / `OutdentTask`)。ネスト深さ上限を超える操作は無視。
- `enter`: 確定 (`persist()`)。
- `esc`: スナップショットから復元してキャンセル。

### 設定画面 (`;` → `s`)

設定画面は左メニュー (`general` / `status` / `field` / `application` / `file_opener`) と詳細ペインの 2 カラム。

- `ModeSetting`: 左メニューにフォーカス。`j/k` で項目移動、`enter` でサブモードへ。
- `ModeSettingGeneral`: yaml パスを参照表示、`data_base_directory` を編集 (`enter` でテキスト入力モーダル)。
- `ModeSettingStatus`: ステータス一覧操作。`a` 追加 / `r` リネーム / `c` 色変更 / `m` 並べ替え / `d` 削除確認。
- `ModeSettingField`: 拡張項目一覧操作。`a` 追加 / `r` リネーム / `m` 並べ替え / `d` 削除確認。`enter` で属性ペインに移動。
- `ModeSettingApplication`: アプリ一覧。`a` 追加 / `m` 並べ替え / `d` 削除確認。`enter` で属性ペイン (name/run 編集)。
- `ModeSettingFileOpener`: 拡張子 → アプリ対応一覧。`a` 追加 / `m` 並べ替え / `d` 削除確認。属性ペインで applications (multi-select) / default_app を編集。

### レイアウト調整モード (`ModeLayout`)

- フォーカス対象は突入時に決定される:
  - `ModeList` から突入 → `task_list`
  - `ModeDetail` で Files 行 → `file_list`
  - `ModeDetail` でそれ以外 → `task_detail`
- `h`/`l`: タスクリスト幅を縮小/拡大 (フォーカスを問わない)。
- `j`/`k`: 対象ペインの高さを拡大/縮小 (`task_list` フォーカス時は no-op)。
- `enter`: 縦 3 値を `normalizeLayoutRatios` で 1.0 に正規化し、`cfg.Layout` を更新して yaml に書き戻す。
- `esc`: `layoutBackup` から復元してキャンセル。

詳細は [layout.md](./layout.md)。

### タグピッカー (`ModeTagPicker`)

- 表示: 1 行目に検索/作成入力欄 (`textinput`)、以降の行に既存タグを `Sorted()` 順で表示。
- `j`/`k`: カーソル移動。
- 入力行で `enter`: 完全一致するタグがあればタスクへの付与/解除を toggle、無ければ新規作成して付与する。新規タグの色は 12 色パレットから round-robin (`nextTagColor`)。
- 既存タグ行で `enter`: タスクへ付与/解除を toggle。
- 既存タグ行で `c`: 色変更ピッカー (`ModeTagColorPicker`)。
- 既存タグ行で `r`: 名前変更 (`ModeTagPickerRename`)。検索文字列は退避され、戻る際に復元される。
- 既存タグ行で `d`: 削除確認 (`ModeTagPickerDeleteConfirm`)。確定で全タスクの `Tags` 配列からも削除する。

### Operation モード (`ModeOperation`)

タスクリストで `o` を押した直後に遷移する一回限りのキー入力モード。

| キー | 動作 |
|---|---|
|| `r` | タイトル編集 (`ModeEditTitle`) |
| `s` | ステータス変更 (`ModeEditStatus`) |
| `g` | タグピッカー (`ModeTagPicker`) |
| `f` | 詳細画面に遷移し Files セクション先頭にカーソル合わせ |
| `esc` | キャンセル |

## 入力検証 UI

- 入力モーダル中はテキストの長さ / 禁止文字を `m.inputErr` に格納し、ポップアップ下部にエラー表示する。
- `enter` で確定するとき `inputErr != nil` なら受け付けない (`internal/tui/app.go` の各 `case` の `enter` 分岐)。

## エラー表示

- `m.saveErr` は画面下部にエラーメッセージを表示する。`esc` で消える (各モード固有の `esc` 動作より先に処理される)。
- 永続化エラー (`m.persist()` が返したエラー) は `saveErr` に格納してそのまま画面に出す。

## 終了確認 (`ModeQuitConfirm`)

- `q` キーで遷移。`y` で `tea.Quit`、`n`/`esc` で元のモードに戻る。

## 外部プロセス起動

- ファイルオープナーや `$EDITOR` の起動は `tea.ExecProcess` を使い、alt screen を抜けた状態で実行する。終了時に `editorFinishedMsg` を model に通知する (`internal/tui/editor.go`)。
- URL を開く処理は `openURLInBrowser` (`internal/tui/editor.go` 近辺) で `xdg-open` / `open` / `start` 系を OS に応じて起動する。
