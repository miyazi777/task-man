package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderList は左ペインを描画する。focused=true なら現在のカーソル行 (status / task) を反転表示。
// inMoveMode=true のときカーソル色を黄 (colorWarn) に切り替え、移動中であることを視覚的に区別する。
// rows は status ヘッダ + task + separator の混在で、上から順に描画する。
func renderList(tasks []task.Task, statuses task.StatusList, rows []listRow, collapsed map[int]bool, cursor int, focused, inMoveMode bool, width, height int) string {
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
			lines = append(lines, renderTaskRow(t, statuses, r.depth, r.hasChildren, r.collapsed, i == cursor, focused, inMoveMode, statusBadge, width))
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
// statusBadge が非空のときはタイトル末尾に " <label>" を付与する (子タスクの status が
// 親グループと異なる場合の視覚マーカー)。
func renderTaskRow(t task.Task, statuses task.StatusList, depth int, hasChildren, collapsed, isCursor, listFocused, inMoveMode bool, statusBadge string, width int) string {
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

	// バッジはタイトルから少なくとも 1 cell 離して右寄せにする。
	// gapMin = 1 を確保しつつ、残りを空白で埋めて右端に貼り付ける。
	const gapMin = 1
	titleW := width - leftPad - markerW - badgeW - gapMin - rightPad
	if titleW < 4 {
		titleW = 4
	}
	title := truncate(t.Title, titleW)

	gap := width - leftPad - markerW - lipgloss.Width(title) - badgeW - rightPad
	if gap < gapMin {
		gap = gapMin
	}
	if statusBadge == "" {
		gap = 0
	}

	if isCursor && listFocused {
		raw := strings.Repeat(" ", leftPad) + marker + title + strings.Repeat(" ", gap) + statusBadge
		return cursorStyleFor(inMoveMode).Width(width).Render(raw)
	}

	var titleRendered string
	if listFocused {
		titleRendered = styleListItem.Inline(true).Render(title)
	} else {
		titleRendered = styleListItemDim.Inline(true).Render(title)
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

	return strings.Repeat(" ", leftPad) + marker + titleRendered + badgeRendered
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
