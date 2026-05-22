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
	detailRowTags  // Tags 行 (常に存在、Enter でタグピッカー起動)
	detailRowField // 拡張項目 (FieldDef)。具体的な field は fieldID で識別
	detailRowFiles // Files セクション。fileCursor を別途持つ
)

// detailRow は詳細画面の論理行 (カーソル 1 ステップ単位)。
type detailRow struct {
	kind    detailRowKind
	fieldID int // detailRowField のときのみ意味がある
}

// buildDetailRows は詳細画面の論理行リストを Title → Status → Tags → fields (position 順) → Files の
// 順番で構築する。fields が空ならフィールド行は挟まれない。Tags 行はタスク有無に関わらず常に存在する
// (Enter でタグピッカーを起動する経路として cursor target を確保)。
func buildDetailRows(fields task.FieldDefList) []detailRow {
	sorted := fields.Sorted()
	rows := make([]detailRow, 0, 4+len(sorted))
	rows = append(rows, detailRow{kind: detailRowTitle})
	rows = append(rows, detailRow{kind: detailRowStatus})
	rows = append(rows, detailRow{kind: detailRowTags})
	for _, f := range sorted {
		rows = append(rows, detailRow{kind: detailRowField, fieldID: f.ID})
	}
	rows = append(rows, detailRow{kind: detailRowFiles})
	return rows
}

// renderDetail は詳細ペイン (上半分) を描画する。Files ブロックは含めない。
//   - rows: 詳細画面の論理行リスト (buildDetailRows の出力)
//   - cursor: rows のインデックス。focused 時にその行を反転表示する
//   - tags: 全タグ集合 (t.Tags の id を解決するため)
//
// 新レイアウトでは Files ブロックは右ペインの中段、Preview は下段に分割表示するため、
// この関数では描画しない。Files カーソルの取り扱いは呼び出し側 (renderFilesBlock 直接呼び出し) に委ねる。
func renderDetail(t *task.Task, statuses task.StatusList, fields task.FieldDefList, tags task.TagList, rows []detailRow, focused bool, cursor, width, height int) string {
	if width <= 0 {
		width = 40
	}
	if t == nil {
		return renderPaneBlock("", width, height)
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
	for i, r := range rows {
		hasCursor := focused && cursor == i
		switch r.kind {
		case detailRowTitle:
			bodyLines = append(bodyLines, renderDetailField("Title", t.Title, focused, hasCursor, statusStyleFor(status), false, labelW, width))
		case detailRowStatus:
			bodyLines = append(bodyLines, renderDetailField("Status", statusText, focused, hasCursor, statusStyleFor(status), true, labelW, width))
		case detailRowTags:
			tagsRow, _ := renderTagsRow(*t, tags, focused, hasCursor, labelW, width)
			bodyLines = append(bodyLines, tagsRow)
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
			if def.Type == task.FieldTypeURL {
				availW := width - 2 - labelW - 1
				if availW > 0 && ansi.StringWidth(value) > availW {
					value = ansi.Truncate(value, availW, "...")
				}
			}
			bodyLines = append(bodyLines, renderDetailField(def.Name, value, focused, hasCursor, lipgloss.Style{}, false, labelW, width))
		case detailRowFiles:
			// Files 行はこのペインでは描画しない (右ペイン中段で描画される)。
		}
	}

	body := strings.Join(bodyLines, "\n")
	return renderPaneBlock(body, width, height)
}

// renderDetailField は label と value の 1 行を描画する。
// hasCursor=true なら行幅いっぱいを反転背景にする (yazi 風)。
// useValueStatusStyle は Status 行の値だけに適用する色 (それ以外は無視)。
// labelW は値の左端を揃えるためのラベル幅 (ID/Title/Status/拡張項目名のうち最大表示幅)。
func renderDetailField(label, value string, focused, hasCursor bool, valueStatusStyle lipgloss.Style, useValueStatusStyle bool, labelW, width int) string {
	padded := padDetailLabel(label, labelW)
	if hasCursor {
		raw := "  " + padded + " " + value
		return renderSingleLineRow(styleCursorRow, raw, width)
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
// タグは "#<name>" を foreground 着色して描画し、間に半角スペース 1 を挟む。
// タグ 0 件でもラベルだけは 1 行表示する (Enter でタグピッカーを起動するための cursor target)。
// 1 行に並びきらないときは折り返す (継続行は label 幅と同じだけインデント)。
// hasCursor=true のときは先頭行に反転スタイル (styleCursorRow) を適用する。
// 第二戻り値は実際の表示行数。
func renderTagsRow(t task.Task, tags task.TagList, focused, hasCursor bool, labelW, width int) (string, int) {
	leadW := 2 + labelW + 1 // "  " + label + " "
	availW := width - leadW
	if availW < 6 {
		availW = 6
	}

	// 解決済みのタグ集合を作る (未知 ID はスキップ)。
	resolved := make([]task.Tag, 0, len(t.Tags))
	for _, id := range t.Tags {
		if tg, ok := tags.ByID(id); ok {
			resolved = append(resolved, tg)
		}
	}

	// 各タグの "#<name>" を構築し、availW に収まるよう折り返してラインに分ける。
	type chip struct {
		w        int    // 表示幅
		plain    string // "#<name>" (反転背景時の素描画用)
		rendered string // 通常表示用 (foreground 着色)
	}
	chips := make([]chip, 0, len(resolved))
	for _, tg := range resolved {
		plain := "#" + tg.Name
		chips = append(chips, chip{
			w:        ansi.StringWidth(plain),
			plain:    plain,
			rendered: renderTagChip(tg),
		})
	}

	type line struct {
		plain    string
		rendered string
	}
	var lines []line
	var curPlain strings.Builder
	var curRendered strings.Builder
	curW := 0
	for i, ch := range chips {
		sep := 0
		if i > 0 && curW > 0 {
			sep = 1
		}
		if curW+sep+ch.w > availW && curW > 0 {
			lines = append(lines, line{plain: curPlain.String(), rendered: curRendered.String()})
			curPlain.Reset()
			curRendered.Reset()
			curW = 0
			sep = 0
		}
		if sep > 0 {
			curPlain.WriteString(" ")
			curRendered.WriteString(" ")
			curW++
		}
		curPlain.WriteString(ch.plain)
		curRendered.WriteString(ch.rendered)
		curW += ch.w
	}
	if curW > 0 {
		lines = append(lines, line{plain: curPlain.String(), rendered: curRendered.String()})
	}
	// タグ 0 件のときは空のラインを 1 つ用意してラベルだけ表示する。
	if len(lines) == 0 {
		lines = []line{{plain: "", rendered: ""}}
	}

	paddedLabel := padDetailLabel("Tags", labelW)
	indent := strings.Repeat(" ", leadW)

	out := make([]string, 0, len(lines))
	for i, ln := range lines {
		if i == 0 {
			if hasCursor && focused {
				// 先頭行はカーソル反転で行全体を塗る。チップ色は失うがプレーン文字列で安定描画。
				raw := "  " + paddedLabel + " " + ln.plain
				out = append(out, renderSingleLineRow(styleCursorRow, raw, width))
			} else {
				out = append(out, "  "+styleLabel.Render(paddedLabel)+" "+ln.rendered)
			}
		} else {
			out = append(out, indent+ln.rendered)
		}
	}
	_ = focused // チップ自体は常に色付き表示 (非カーソル時)
	return strings.Join(out, "\n"), len(out)
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

// fileRow はファイルリストの 1 行分の表示モデル。
// withFilesRefreshed で構築され renderFileNamesList が消費する。
//   - relPath: タスクディレクトリからの相対パス ("/" 区切り)。各種ファイル操作で使う。
//   - hasChildren: ディレクトリかつ子を 1 つ以上含む場合のみ true。マーカー表示の有無を決める。
//   - collapsed: ディレクトリが折りたたまれていれば true。
type fileRow struct {
	name        string
	relPath     string
	isDir       bool
	depth       int
	hasChildren bool
	collapsed   bool
}

// renderFileNamesList はファイル一覧を width × height の領域に描画する。
// Files: ヘッダや罫線は含まない (新レイアウトで右ペインを上下分割するため、ヘッダ/罫線は呼び出し側で組み立てる)。
//   - blockFocused: detailCursor が Files セクションを指しているか
//   - fileCursor: Files 内の選択行
//
// 行レイアウトは "<indent><marker><name>"。
//   - indent: fileBaseLeftPad + depth*filePerDepth
//   - marker: タスクリストと揃え、hasChildren && collapsed なら "+ "、!collapsed なら "- "、それ以外は "  "
//   - name: ディレクトリの場合は末尾に "/" を付ける
//
// ファイルが 0 件のときは "(no files)" を 1 行表示し、残りは空行で埋める。
// ファイル数が height を超える場合は fileCursor が見える範囲を表示するスクロールを行う。
func renderFileNamesList(files []fileRow, focused, blockFocused bool, fileCursor, width, height int) string {
	if height <= 0 {
		return ""
	}
	if len(files) == 0 {
		empty := strings.Repeat(" ", fileBaseLeftPad) + styleValueDim.Render("(no files)")
		return renderPaneBlock(empty, width, height)
	}

	// fileCursor を表示範囲に含めるためのオフセット計算 (シンプルなウィンドウスクロール)。
	startIdx := 0
	if fileCursor >= height {
		startIdx = fileCursor - height + 1
	}
	endIdx := startIdx + height
	if endIdx > len(files) {
		endIdx = len(files)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		row := files[i]
		indent := strings.Repeat(" ", fileBaseLeftPad+row.depth*filePerDepth)
		marker := "  "
		if row.hasChildren {
			if row.collapsed {
				marker = "+ "
			} else {
				marker = "- "
			}
		}
		name := row.name
		if row.isDir {
			name += "/"
		}
		raw := indent + marker + name

		isCursor := blockFocused && focused && i == fileCursor
		if isCursor {
			lines = append(lines, renderSingleLineRow(styleCursorRow, raw, width))
			continue
		}
		if focused {
			lines = append(lines, indent+marker+styleValue.Render(name))
		} else {
			lines = append(lines, indent+marker+styleValueDim.Render(name))
		}
	}
	return renderPaneBlock(strings.Join(lines, "\n"), width, height)
}

// ファイルリストのインデント定数。タスクリストと同じ階層 2 cell で揃える。
// ベース 4 cell は "  " (2 cell, marker と同じ幅) + 詳細ペインの行頭余白 2 cell 相当。
const (
	fileBaseLeftPad = 4
	filePerDepth    = 2
)
