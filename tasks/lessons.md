# Lessons

## bubbles/textinput.View() の幅は `Width + 3`

`textinput.Model.View()` の戻り値の表示幅は **`PromptStyle.Render(Prompt)` の幅 + 1 (カーソル) + `m.Width`** になる (内部で `m.Width` セル分のスペース pad を追加するため)。デフォルト Prompt = `"> "` (2 cell) のとき、`View()` 全体の幅は **`m.Width + 3`** cell。

ポップアップ枠内に置く場合、内側コンテンツ幅 `contentW` に揃えるには次の式で `m.Width` を決める:

```
m.Width = contentW - promptWidth - 1
       = contentW - 3   (デフォルト prompt 時)
```

合致しないと:

- `View()` が `contentW` を超え、囲い (`│ ... │`) が枠の上下行より広くなる
- `lipgloss.JoinVertical(Left, ...)` が短い行を素のスペースで右パディング (背景色なし) し、ポップアップ右端に「歯抜け」が出て見栄えが崩れる

### 防御パターン

1. `m.Width = contentW - 3` を起点に計算する
2. `View()` を使う側でも `ansi.Truncate(view, contentW, "")` で上限クランプ
3. テストで実 textinput を使ったポップアップ外形幅 (`popupWidth(screenW)`) の一致を assert する (画面幅一致だけだと `JoinVertical` の素パディングで隠れて検出できない)
