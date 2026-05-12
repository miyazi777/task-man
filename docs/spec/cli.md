# CLI 仕様

`internal/cli` パッケージ。`pflag` を利用したシンプルなフラグ解析。

## フラグ

| フラグ | 短縮 | 型 | デフォルト | 説明 |
|---|---|---|---|---|
| `--tasks` | `-t` | string | `"tasks.yaml"` | 使用する yaml のパス。`~` / `~/...` を `os.UserHomeDir()` で展開する (`~user` 形式は非対応)。 |
| `--init` | `-i` | bool | `false` | yaml をデフォルト 3 ステータスのみで再生成し、対応する `task-<id>/` ディレクトリを全削除する初期化フロー。実行前に `y/N` 確認プロンプトを表示。 |

## デフォルトファイル

- 定数: `cli.DefaultFileName = "tasks.yaml"` (`internal/cli/args.go`)。
- `-t` 未指定時は CWD の `tasks.yaml` を見る。

## 起動時の振る舞い

1. `Parse(argv)` は `Args` 構造体を返す:
   - `Path`: 指定または `tasks.yaml`。`~` 展開済み。
   - `MustExist`: `-t` が指定され、かつ `-i` が指定されていない場合 `true`。
   - `Init`: `-i` の真偽。
2. メインフロー (`cmd/task-man/main.go`) は `filepath.Abs(args.Path)` で絶対化する。
3. `args.Init` が真なら `runInit(absPath, os.Stdin, os.Stdout)` を実行して終了する。
4. それ以外は `cli.EnsureFile(args)` を呼び、不在で `MustExist=false` の場合は空ファイルを `os.Create` で作る。`MustExist=true` で不在なら `"file not found: <path>"` のエラー。
5. `storage.NewYAMLRepository(absPath).Load()` を実行して `LoadResult` を取得し、`tui.NewModel` で UI を構築する。

## `--init` の挙動 (`runInit`)

1. 既存 yaml をベストエフォートで読み込む (`storage.YAMLRepository.Load`)。読めれば `AppConfig` (主に `data_base_directory`) を引き継ぐ。読み込みエラー時は `--init` を中断 (yaml を修正または削除してから再実行を要求)。
2. 削除対象ディレクトリのパスを表示し、`Are you sure? (y/N): ` プロンプトで確認 (空回答 / `n` でキャンセル)。
3. `storage.RemoveAllTaskData(yamlDir, cfg.DataBaseDirectory)` で `task-<id>/` ディレクトリを再帰削除。
4. `repo.Save(LoadResult{Statuses: task.DefaultStatuses(), Config: existingCfg})` で yaml を再生成。タスク・拡張項目・タグはすべて空。
5. 削除件数と保存先パスを stdout に出力して終了。

## 出力

- エラーは `fmt.Fprintln(os.Stderr, "error:", err)` の形式で stderr に出力し、`os.Exit(1)` で終了する (`cmd/task-man/main.go` の `main()`)。

## 設定例

`alias` で共有 yaml を参照する想定:

```bash
alias tm='task-man -t ~/private/tasks.yaml'
```

リセット:

```bash
./task-man -i                            # ./tasks.yaml を初期化 (確認あり)
./task-man -t ~/private/tasks.yaml -i    # 指定 yaml を初期化
```
