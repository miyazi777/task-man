package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderList は左ペインを描画する。focused=true なら現在のカーソル行 (status / task) を反転表示。
// inMoveMode=true のときカーソル色を黄 (colorWarn) に切り替え、移動中であることを視覚的に区別する。
// rows は status ヘッダ + task + separator の混在で、上から順に描画する。
func renderList(tasks []task.Task, statuses task.StatusList, allTags task.TagList, rows []listRow, collapsed map[int]bool, cursor int, focused, inMoveMode bool, width, height int) string {
	if width <= 0 {
		width = 32
	}

	var lines []string
	for i, r := range rows {
		switch r.kind {
		case rowSeparator:
			lines = append(lines, "")
		case rowStatus:
			lines = append(lines, renderStatusHeader(statuses, r.statusID, collapsed[r.statusID], i == cursor, focused, inMoveMode, width))
		case rowTask:
			t := tasks[r.taskIndex]
			// 子タスクが所属グループと異なる status を持つ場合、視覚的に区別が付くよう
			// タイトル末尾にステータスラベルを付与する。同じ status のときは省略してクリーンに。
			statusBadge := ""
			if t.StatusID != r.statusID {
				if s, ok := statuses.ByID(t.StatusID); ok {
					statusBadge = s.Label
				}
			}
			lines = append(lines, renderTaskRow(t, statuses, allTags, r.depth, r.hasChildren, r.collapsed, i == cursor, focused, inMoveMode, statusBadge, width))
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

// renderStatusHeader は ▼/▶ + [label] のステータス見出し行を描画する。
// 通常時は status の色を背景にした反転表示 (黒抜き文字)。
// カーソル時はリスト共通のカーソル反転 (アクセント色背景、ModeMove なら警告色) を優先する。
func renderStatusHeader(statuses task.StatusList, statusID int, isCollapsed, isCursor, listFocused, inMoveMode bool, width int) string {
	status, _ := statuses.ByID(statusID)
	marker := "[-]"
	if isCollapsed {
		marker = "[+]"
	}

	if isCursor && listFocused {
		raw := " " + marker + "  " + status.Label + " "
		return cursorStyleFor(inMoveMode).Width(width).Render(raw)
	}

	prefix := " " + marker + " "
	labelPart := statusRowStyleFor(status).Render(" " + status.Label + " ")
	return lipgloss.NewStyle().Width(width).Render(prefix + labelPart)
}

// renderTaskRow はタスク行を描画する。先頭にインデント (depth に応じて 2 cell ずつ加算)
// を入れて status ヘッダ・サブタスクと階層感を出す。depth=0 は通常のタスク、depth>=1 はサブタスク。
// hasChildren=true のタスクは collapsed の有無に応じて "+ "/"- " のマーカーを付ける。
// 子を持たないタスクでもタイトル位置を揃えるため空白 2 cell を予約する。
// statusBadge が非空のときは行の右端に " <label>" を右寄せ表示する (子タスクの status が
// 親グループと異なる場合の視覚マーカー)。
// allTags は t.Tags の id 解決用。タイトル直後に "[tag1][tag2]" 形式でタグカラーの色文字を付与する。
func renderTaskRow(t task.Task, statuses task.StatusList, allTags task.TagList, depth int, hasChildren, collapsed, isCursor, listFocused, inMoveMode bool, statusBadge string, width int) string {
	const baseLeftPad, perDepth, markerW, rightPad = 2, 2, 2, 1
	leftPad := baseLeftPad + depth*perDepth

	marker := "  "
	if hasChildren {
		if collapsed {
			marker = "- "
		} else {
			marker = "+ "
		}
	}

	badgeW := 0
	if statusBadge != "" {
		badgeW = lipgloss.Width(statusBadge)
	}

	// 行幅見積もりに先立ち、タグバッジ群の利用幅を決める。
	// 利用可能幅は title の最小幅 (8 cell) を確保した上での残り。
	const minTitleW, gapMin = 8, 1
	tagsAvail := width - leftPad - markerW - minTitleW - 1 /*sp before tags*/ - badgeW - gapMin - rightPad
	if statusBadge == "" {
		tagsAvail = width - leftPad - markerW - minTitleW - 1 - rightPad
	}
	if tagsAvail < 0 {
		tagsAvail = 0
	}

	// 入る分だけタグを採用し、残りはドロップ。各タグは "[<name>]" で 表示幅 = name+2。
	type inlineTag struct {
		plain    string
		rendered string
		w        int
	}
	var tags []inlineTag
	tagsW := 0
	if len(t.Tags) > 0 && tagsAvail > 0 {
		for _, id := range t.Tags {
			tg, ok := allTags.ByID(id)
			if !ok {
				continue
			}
			plain := "[" + tg.Name + "]"
			w := lipgloss.Width(plain)
			if tagsW+w > tagsAvail {
				break
			}
			tags = append(tags, inlineTag{plain: plain, rendered: renderInlineTagBadge(tg, listFocused), w: w})
			tagsW += w
		}
	}

	// バッジはタイトル + タグ部分の右側に右寄せにする。
	// 残り幅でタイトルを切り詰める。タグが付くときはタイトル直後に空白 1 を入れる。
	gapTags := 0
	if tagsW > 0 {
		gapTags = 1
	}
	titleW := width - leftPad - markerW - gapTags - tagsW - badgeW - gapMin - rightPad
	if statusBadge == "" {
		titleW = width - leftPad - markerW - gapTags - tagsW - rightPad
	}
	if titleW < 4 {
		titleW = 4
	}
	title := truncate(t.Title, titleW)

	// 右端の status バッジまでの埋め空白幅を計算 (status badge 用の右寄せ)。
	gap := width - leftPad - markerW - lipgloss.Width(title) - gapTags - tagsW - badgeW - rightPad
	if gap < gapMin {
		gap = gapMin
	}
	if statusBadge == "" {
		gap = 0
	}

	if isCursor && listFocused {
		// カーソル行は反転表示で行全体が同じ bg になるため、タグもプレーン文字列で渡す。
		var tagsPlain string
		for _, tg := range tags {
			tagsPlain += tg.plain
		}
		var inlineTagsPart string
		if tagsW > 0 {
			inlineTagsPart = " " + tagsPlain
		}
		raw := strings.Repeat(" ", leftPad) + marker + title + inlineTagsPart + strings.Repeat(" ", gap) + statusBadge
		return cursorStyleFor(inMoveMode).Width(width).Render(raw)
	}

	var titleRendered string
	if listFocused {
		titleRendered = styleListItem.Inline(true).Render(title)
	} else {
		titleRendered = styleListItemDim.Inline(true).Render(title)
	}

	var tagsRendered string
	if tagsW > 0 {
		tagsRendered = " "
		for _, tg := range tags {
			tagsRendered += tg.rendered
		}
	}

	var badgeRendered string
	if statusBadge != "" {
		// バッジはその子タスクの実際の status の色で表示する (色一致で見分けやすく)。
		badgeStyle := lipgloss.NewStyle().Foreground(colorMuted)
		if s, ok := statuses.ByID(t.StatusID); ok && s.Color != "" {
			badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Color))
		}
		if !listFocused {
			badgeStyle = lipgloss.NewStyle().Foreground(colorDim)
		}
		badgeRendered = strings.Repeat(" ", gap) + badgeStyle.Render(statusBadge)
	}

	return strings.Repeat(" ", leftPad) + marker + titleRendered + tagsRendered + badgeRendered
}

// renderInlineTagBadge はタスクリストのタグ表示 "[name]" を描画する。
// foreground = タグ色 (Color 未指定なら colorMuted)、背景なし。listFocused=false のときは dim。
func renderInlineTagBadge(tg task.Tag, listFocused bool) string {
	style := lipgloss.NewStyle().Foreground(colorMuted)
	if !listFocused {
		style = lipgloss.NewStyle().Foreground(colorDim)
	} else if tg.Color != "" {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(tg.Color))
	}
	return style.Render("[" + tg.Name + "]")
}

// cursorStyleFor は ModeMove の有無で標準/警告色のどちらの反転スタイルを返すか切り替える。
func cursorStyleFor(inMoveMode bool) lipgloss.Style {
	if inMoveMode {
		return styleMoveCursorRow
	}
	return styleCursorRow
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// 表示幅で切り詰める単純実装 (CJK 全角は重み 2)。
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= w {
			return candidate
		}
	}
	return "…"
}
