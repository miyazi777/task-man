package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/task"
)

// detailRowKind は詳細画面のカーソル可能行の種類。
type detailRowKind int

const (
	detailRowTitle detailRowKind = iota
	detailRowStatus
	detailRowField // 拡張項目 (FieldDef)。具体的な field は fieldID で識別
	detailRowFiles // Files セクション。fileCursor を別途持つ
)

// detailRow は詳細画面の論理行 (カーソル 1 ステップ単位)。
type detailRow struct {
	kind    detailRowKind
	fieldID int // detailRowField のときのみ意味がある
}

// buildDetailRows は詳細画面の論理行リストを Title → Status → fields (position 順) → Files の
// 順番で構築する。fields が空ならフィールド行は挟まれない。
func buildDetailRows(fields task.FieldDefList) []detailRow {
	sorted := fields.Sorted()
	rows := make([]detailRow, 0, 3+len(sorted))
	rows = append(rows, detailRow{kind: detailRowTitle})
	rows = append(rows, detailRow{kind: detailRowStatus})
	for _, f := range sorted {
		rows = append(rows, detailRow{kind: detailRowField, fieldID: f.ID})
	}
	rows = append(rows, detailRow{kind: detailRowFiles})
	return rows
}

// detailFilesDividerRow は detailRows と Tags 行の表示行数 (tagsLines) を受け取り、
// Files: 直下の罫線が何行目に来るかを返す。
// 物理レイアウト: ID(1) + Title(1) + Status(1) + Tags(L) + N field rows + 空行(1)
// + Files: header(1) + 罫線(1) → 罫線位置 = 5 + L + N。
// L (tags 行数) が 0 の場合は従来どおり 5 + N。
// 左右のペイン縦区切り線に T 字接合を入れるために使う。
func detailFilesDividerRow(rows []detailRow, tagsLines int) int {
	n := 0
	for _, r := range rows {
		if r.kind == detailRowField {
			n++
		}
	}
	return 5 + n + tagsLines
}

// renderDetail は右ペインを描画する。
//   - rows: 詳細画面の論理行リスト (buildDetailRows の出力)
//   - cursor: rows のインデックス。focused 時にその行を反転表示する
//   - fileCursor: rows[cursor].kind == detailRowFiles のとき、ファイル一覧内の選択 index
//   - tags: 全タグ集合 (t.Tags の id を解決するため)
func renderDetail(t *task.Task, statuses task.StatusList, fields task.FieldDefList, tags task.TagList, files []string, rows []detailRow, focused bool, cursor, fileCursor, width, height int) string {
	if width <= 0 {
		width = 40
	}
	if t == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("")
	}

	status, ok := statuses.ByID(t.StatusID)
	statusText := "?"
	if ok {
		statusText = status.Label
	}

	labelW := detailLabelWidth(fields)

	// ID は読み取り専用なのでカーソル対象外。
	idRow := "  " + styleLabel.Render(padDetailLabel("ID", labelW)) + " " + styleValueDim.Render(strconv.Itoa(t.ID))

	bodyLines := []string{idRow}
	var filesBlock string
	for i, r := range rows {
		hasCursor := focused && cursor == i
		switch r.kind {
		case detailRowTitle:
			bodyLines = append(bodyLines, renderDetailField("Title", t.Title, focused, hasCursor, statusStyleFor(status), false, labelW, width))
		case detailRowStatus:
			bodyLines = append(bodyLines, renderDetailField("Status", statusText, focused, hasCursor, statusStyleFor(status), true, labelW, width))
			// Status の直後に Tags 行を追加 (read-only、カーソル対象外)。タグ 0 件のときは行ごと省略。
			if tagsRow, _ := renderTagsRow(*t, tags, focused, labelW, width); tagsRow != "" {
				bodyLines = append(bodyLines, tagsRow)
			}
		case detailRowField:
			def, ok := fields.ByID(r.fieldID)
			if !ok {
				continue
			}
			value := ""
			if tf, ok := t.Fields.ByFieldID(def.ID); ok {
				value = tf.Value
			}
			// url 型は折り返しを避けるため、表示幅に収まらない場合は末尾を ... に置換する。
			// availW = ペイン幅 - leading "  " - ラベル幅 - separator " "
			if def.Type == task.FieldTypeURL {
				availW := width - 2 - labelW - 1
				if availW > 0 && ansi.StringWidth(value) > availW {
					value = ansi.Truncate(value, availW, "...")
				}
			}
			bodyLines = append(bodyLines, renderDetailField(def.Name, value, focused, hasCursor, lipgloss.Style{}, false, labelW, width))
		case detailRowFiles:
			// Files は専用ブロック。カーソル位置・focus 状態をブロック側に渡す。
			filesBlock = renderFilesBlock(files, focused, hasCursor, fileCursor, width)
		}
	}

	// Files ブロックは body 末尾に「空行 + ブロック」として配置する。
	bodyLines = append(bodyLines, "", filesBlock)

	body := strings.Join(bodyLines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
}

// renderDetailField は label と value の 1 行を描画する。
// hasCursor=true なら行幅いっぱいを反転背景にする (yazi 風)。
// useValueStatusStyle は Status 行の値だけに適用する色 (それ以外は無視)。
// labelW は値の左端を揃えるためのラベル幅 (ID/Title/Status/拡張項目名のうち最大表示幅)。
func renderDetailField(label, value string, focused, hasCursor bool, valueStatusStyle lipgloss.Style, useValueStatusStyle bool, labelW, width int) string {
	padded := padDetailLabel(label, labelW)
	if hasCursor {
		raw := "  " + padded + " " + value
		return styleCursorRow.Width(width).Render(raw)
	}
	var labelRendered, valueRendered string
	if focused {
		labelRendered = styleLabel.Render(padded)
		if useValueStatusStyle {
			valueRendered = valueStatusStyle.Render(value)
		} else {
			valueRendered = styleValue.Render(value)
		}
	} else {
		labelRendered = styleLabel.Render(padded)
		valueRendered = styleValueDim.Render(value)
	}
	return "  " + labelRendered + " " + valueRendered
}

// renderTagsRow は Tags 行を構築する。
// タグ 0 件のときは空文字列 + 0 行を返し、呼び出し側で行ごと省略してもらう。
// 1 行に並びきらないときは折り返す (継続行は label 幅と同じだけインデント)。
// 第二戻り値は実際の表示行数。
func renderTagsRow(t task.Task, tags task.TagList, focused bool, labelW, width int) (string, int) {
	if len(t.Tags) == 0 {
		return "", 0
	}
	// "[<tag>]" のトークン群を組み立てる (未知 ID は単純にスキップ)。
	tokens := make([]string, 0, len(t.Tags))
	for _, id := range t.Tags {
		if tg, ok := tags.ByID(id); ok {
			tokens = append(tokens, "["+tg.Name+"]")
		}
	}
	if len(tokens) == 0 {
		return "", 0
	}

	leadW := 2 + labelW + 1 // "  " + label + " "
	availW := width - leadW
	if availW < 4 {
		availW = 4
	}

	// 折り返しレイアウト: トークン間のセパレータは半角スペース 1。
	var lines []string
	var cur strings.Builder
	curW := 0
	for i, tok := range tokens {
		tokW := ansi.StringWidth(tok)
		sep := 0
		if i > 0 && curW > 0 {
			sep = 1
		}
		if curW+sep+tokW > availW && curW > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
			sep = 0
		}
		if sep > 0 {
			cur.WriteString(" ")
			curW++
		}
		cur.WriteString(tok)
		curW += tokW
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}

	paddedLabel := padDetailLabel("Tags", labelW)
	labelRendered := styleLabel.Render(paddedLabel)
	indent := strings.Repeat(" ", leadW)

	tokenStyle := styleValue
	if !focused {
		tokenStyle = styleValueDim
	}

	out := make([]string, 0, len(lines))
	for i, line := range lines {
		styled := tokenStyle.Render(line)
		if i == 0 {
			out = append(out, "  "+labelRendered+" "+styled)
		} else {
			out = append(out, indent+styled)
		}
	}
	return strings.Join(out, "\n"), len(out)
}

// tagsRowLineCount は renderTagsRow が出力する表示行数を計算だけする (描画はしない)。
// dividerRow 計算で使う。
func tagsRowLineCount(t *task.Task, tags task.TagList, labelW, width int) int {
	if t == nil {
		return 0
	}
	_, n := renderTagsRow(*t, tags, true, labelW, width)
	return n
}

// detailLabelWidth は ID/Title/Status と全 field 名のうち、表示幅の最大値を返す。
// 値の左端を揃えるためにラベル列の幅として使う。
func detailLabelWidth(fields task.FieldDefList) int {
	w := 6 // "Status" (6 cells) が ID/Title/Status の最大
	for _, f := range fields.Sorted() {
		if v := ansi.StringWidth(f.Name); v > w {
			w = v
		}
	}
	return w
}

// padDetailLabel は label を w cell 幅まで右側を空白でパディングする。
// 既に w 以上ならそのまま返す。
func padDetailLabel(label string, w int) string {
	diff := w - ansi.StringWidth(label)
	if diff <= 0 {
		return label
	}
	return label + strings.Repeat(" ", diff)
}

// renderFilesBlock は Files: セクションをヘッダ + 区切り線 + ファイル行で描画する。
//   - blockFocused: detailCursor が Files セクションを指しているか
//   - fileCursor: Files 内の選択行
//
// ファイルが 0 件のときは "(no files)" を 1 行表示する。
func renderFilesBlock(files []string, focused, blockFocused bool, fileCursor, width int) string {
	header := "  " + styleLabel.Render("Files:")
	// 区切り線はペイン全幅にして、左右のペイン縦区切り線 (├ / ┤) と
	// つながる横一文字に見えるようにする。
	dividerWidth := width
	if dividerWidth < 1 {
		dividerWidth = 1
	}
	divider := styleDivider.Render(strings.Repeat("─", dividerWidth))

	var rows []string
	rows = append(rows, header, divider)

	if len(files) == 0 {
		rows = append(rows, "    "+styleValueDim.Render("(no files)"))
		return strings.Join(rows, "\n")
	}

	for i, name := range files {
		isCursor := blockFocused && focused && i == fileCursor
		if isCursor {
			rows = append(rows, styleCursorRow.Width(width).Render("    "+name))
			continue
		}
		var line string
		if focused {
			line = "    " + styleValue.Render(name)
		} else {
			line = "    " + styleValueDim.Render(name)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}
