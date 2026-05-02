package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// renderList は左ペインを描画する。focused=true なら現在のカーソル行 (status / task) を反転表示。
// rows は status ヘッダ + task + separator の混在で、上から順に描画する。
func renderList(tasks []task.Task, statuses task.StatusList, rows []listRow, collapsed map[int]bool, cursor int, focused bool, width, height int) string {
	if width <= 0 {
		width = 32
	}

	var lines []string
	for i, r := range rows {
		switch r.kind {
		case rowSeparator:
			lines = append(lines, "")
		case rowStatus:
			lines = append(lines, renderStatusHeader(statuses, r.statusID, collapsed[r.statusID], i == cursor, focused, width))
		case rowTask:
			t := tasks[r.taskIndex]
			lines = append(lines, renderTaskRow(t, statuses, r.depth, r.hasChildren, r.collapsed, i == cursor, focused, width))
		case rowMovePlaceholder:
			lines = append(lines, renderMovePlaceholder(r.depth, width))
		}
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

// renderStatusHeader は ▼/▶ + [label] のステータス見出し行を描画する。
// 通常時は status の色を背景にした反転表示 (黒抜き文字)。
// カーソル時はリスト共通のカーソル反転 (アクセント色背景) を優先する。
func renderStatusHeader(statuses task.StatusList, statusID int, isCollapsed, isCursor, listFocused bool, width int) string {
	status, _ := statuses.ByID(statusID)
	marker := "[-]"
	if isCollapsed {
		marker = "[+]"
	}

	if isCursor && listFocused {
		raw := " " + marker + "  " + status.Label + " "
		return styleCursorRow.Width(width).Render(raw)
	}

	prefix := " " + marker + " "
	labelPart := statusRowStyleFor(status).Render(" " + status.Label + " ")
	return lipgloss.NewStyle().Width(width).Render(prefix + labelPart)
}

// renderTaskRow はタスク行を描画する。先頭にインデント (depth に応じて 2 cell ずつ加算)
// を入れて status ヘッダ・サブタスクと階層感を出す。depth=0 は通常のタスク、depth>=1 はサブタスク。
// hasChildren=true のタスクは collapsed の有無に応じて "+ "/"- " のマーカーを付ける。
// 子を持たないタスクでもタイトル位置を揃えるため空白 2 cell を予約する。
func renderTaskRow(t task.Task, statuses task.StatusList, depth int, hasChildren, collapsed, isCursor, listFocused bool, width int) string {
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

	titleW := width - leftPad - markerW - rightPad
	if titleW < 4 {
		titleW = 4
	}
	title := truncate(t.Title, titleW)

	if isCursor && listFocused {
		raw := strings.Repeat(" ", leftPad) + marker + title
		return styleCursorRow.Width(width).Render(raw)
	}

	var titleRendered string
	if listFocused {
		titleRendered = styleListItem.Inline(true).Render(title)
	} else {
		titleRendered = styleListItemDim.Inline(true).Render(title)
	}
	return strings.Repeat(" ", leftPad) + marker + titleRendered
}

// renderMovePlaceholder は ModeMove 時にカーソルタスク直下に挿入される【移動先】行を描画する。
// depth は親タスクの depth + 1 を想定し、左パディングで階層感を出す。
// 視認性のため黄色 (colorWarn) で太字。
func renderMovePlaceholder(depth, width int) string {
	const baseLeftPad, perDepth = 2, 2
	leftPad := baseLeftPad + depth*perDepth
	body := "+ 【移動先】"
	style := lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
	return strings.Repeat(" ", leftPad) + style.Render(body)
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
