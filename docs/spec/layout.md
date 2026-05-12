# レイアウト仕様

`internal/tui/layout.go`。タスクリスト画面のペイン比率に関する計算とモード操作を担う。

## 保存スキーマ

`tasks.yaml` の `layout.main` に保存される。各値は `0.0 ～ 1.0` の比率。

```yaml
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
```

対応する Go 型:

```go
// internal/storage/config.go
type LayoutConfig struct {
    TaskListWidth     *float64
    TaskDetailHeight  *float64
    FileListHeight    *float64
    FilePreviewHeight *float64
}
```

ポインタ型なので 4 値とも独立に未設定 (nil) を区別できる。

## デフォルト比率

| ペイン | 比率 |
|---|---|
| `task_list.width` | `2/3` (= `defaultTaskListRatio`) |
| `task_detail.height` | `1/3` |
| `file_list.height` | `1/3` |
| `file_preview.height` | `1/3` |

## 範囲制約

| 項目 | 下限 | 上限 |
|---|---|---|
| `task_list.width` | `0.1` | `0.9` |
| `task_detail.height` | `0.1` | `0.8` |
| `file_list.height` | `0.1` | `0.8` |
| `file_preview.height` | `0.1` | `0.8` |

## 主要関数

- `ensureLayoutRatios(in)`: nil 値をデフォルト比率で埋める。
- `isLayoutComplete(in)`: 4 値とも非 nil の場合のみ true。レンダリング側で「完全な比率があるか」を判定し、無ければ従来計算 (twoPaneWidths 相当) にフォールバックする。
- `clampLayoutRatios(in)`: 各値を範囲に丸める。操作 1 ステップごとの clamp 用。
- `normalizeLayoutRatios(in)`: 縦 3 値の合計が `1.0` になるよう比例配分で正規化。範囲外に押し出された分は `preview → file_list → task_detail` の順に相殺する。`enter` 確定時に呼ぶ。
- `computeLayoutDelta(screenW, bodyH)`: 「画面 1 セルぶん」を比率に変換した値を返す (`1.0 / availW`, `1.0 / bodyH`)。
- `applyLayoutToScreen(in, screenW, bodyH)`: 比率を実セル数に変換。`leftW + 区切り 1 + rightW = screenW`、`detailH + fileAreaH + previewAreaH = bodyH` を満たすように分配。最低高さ (detail >= 3, fileArea >= 3, previewArea >= 2) を確保し、超過分は preview → file → detail の順で縮め、不足分は detail に足す。
- `adjustLayoutVertical(in, focus, delta)`: フォーカスペインの高さを `delta` 増減し、不足分を `preview → 他ペイン` から相殺する。`task_list` フォーカス時は no-op (横操作専用)。
- `adjustLayoutHorizontal(in, delta)`: `task_list.width` を `delta` 増減する。clamp は呼び出し側で行う。

## 操作フロー

1. `;` → `l` で `ModeLayout` に突入する。
2. 突入時に `layoutBackup` へ現在値を退避し、`layout` を `ensureLayoutRatios` で埋める。
3. フォーカスは突入元から決定する:
   - `ModeList` から → `layoutFocusTaskList`
   - `ModeDetail` で Files 行カーソル → `layoutFocusFileList`
   - `ModeDetail` でそれ以外 → `layoutFocusTaskDetail`
4. キー操作 (`h/l/j/k`) で `clampLayoutRatios(adjust...)` を適用しながら比率を編集する。
5. `enter` で `normalizeLayoutRatios` → `cfg.Layout = m.layout` → `persist()` で yaml へ保存し、元のモードへ戻る。
6. `esc` で `layoutBackup` から復元し、保存せずに元のモードへ戻る。

## 描画への反映

- 比率が「完全」(4 値すべて非 nil) のときは `applyLayoutToScreen` の結果を用いる。
- それ以外は従来計算 (画面横の 2/3 を左ペイン、縦 3 等分) を使う。
