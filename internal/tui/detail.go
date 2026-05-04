package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

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

// detailFilesDividerRow は detailRows と「タスクが存在する」前提で、Files: 直下の罫線が
// 何行目に来るかを返す。物理レイアウト: ID(1) + Title(1) + Status(1) + N field rows + 空行(1)
// + Files: header(1) + 罫線(1) → 罫線位置 = 4 + N + 1 = 5 + N。
// 左右のペイン縦区切り線に T 字接合を入れるために使う。
func detailFilesDividerRow(rows []detailRow) int {
	n := 0
	for _, r := range rows {
		if r.kind == detailRowField {
			n++
		}
	}
	return 5 + n
}

// renderDetail は右ペインを描画する。
//   - rows: 詳細画面の論理行リスト (buildDetailRows の出力)
//   - cursor: rows のインデックス。focused 時にその行を反転表示する
//   - fileCursor: rows[cursor].kind == detailRowFiles のとき、ファイル一覧内の選択 index
func renderDetail(t *task.Task, statuses task.StatusList, fields task.FieldDefList, files []string, rows []detailRow, focused bool, cursor, fileCursor, width, height int) string {
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

	// ID は読み取り専用なのでカーソル対象外。
	idRow := "  " + styleLabel.Render("ID") + "     " + styleValueDim.Render(strconv.Itoa(t.ID))

	bodyLines := []string{idRow}
	var filesBlock string
	for i, r := range rows {
		hasCursor := focused && cursor == i
		switch r.kind {
		case detailRowTitle:
			bodyLines = append(bodyLines, renderDetailField("Title", t.Title, focused, hasCursor, statusStyleFor(status), false, width))
		case detailRowStatus:
			bodyLines = append(bodyLines, renderDetailField("Status", statusText, focused, hasCursor, statusStyleFor(status), true, width))
		case detailRowField:
			def, ok := fields.ByID(r.fieldID)
			if !ok {
				continue
			}
			value := ""
			if tf, ok := t.Fields.ByFieldID(def.ID); ok {
				value = tf.Value
			}
			bodyLines = append(bodyLines, renderDetailField(def.Name, value, focused, hasCursor, lipgloss.Style{}, false, width))
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
func renderDetailField(label, value string, focused, hasCursor bool, valueStatusStyle lipgloss.Style, useValueStatusStyle bool, width int) string {
	if hasCursor {
		raw := "  " + label + "  " + value
		return styleCursorRow.Width(width).Render(raw)
	}
	var labelRendered, valueRendered string
	if focused {
		labelRendered = styleLabel.Render(label)
		if useValueStatusStyle {
			valueRendered = valueStatusStyle.Render(value)
		} else {
			valueRendered = styleValue.Render(value)
		}
	} else {
		labelRendered = styleLabel.Render(label)
		valueRendered = styleValueDim.Render(value)
	}
	return "  " + labelRendered + "  " + valueRendered
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
