package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// calendarDayLabels は曜日ラベル (sun-first)。仕様の表記に合わせて全て小文字。
var calendarDayLabels = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

const (
	// calendarCellW は 1 日のセル幅 (3 cell: " 1 " など)。
	calendarCellW = 3
	// calendarMonthW は 1 ヶ月分の表示幅 (7 列 × 3 cell + 6 区切り = 27 cell)。
	calendarMonthW = 7*calendarCellW + 6
	// calendarWeeksFixed は常時表示する週数。月によって変わらないように 6 週固定にする。
	calendarWeeksFixed = 6
)

// overlayCalendarPopup は date 型 field 値編集用のカレンダーモーダルを中央オーバーレイで描画する。
// 1 ヶ月分のカレンダーをモーダル中央に表示し、カーソル日のみ反転表示する。
func overlayCalendarPopup(bg string, cursor time.Time, screenW, screenH int) string {
	// hint 行幅も考慮して contentW を決める。
	// カーソルの上下左右は明らかなのでモーダル内ヒントからは省略する。
	hints := []hintItem{
		{"p", "prev mo"}, {"n", "next mo"},
		{"Enter", "save"}, {"Esc", "cancel"},
	}
	hintRendered := renderPopupHints(hints)
	hintW := ansi.StringWidth(ansi.Strip(hintRendered))

	contentW := calendarMonthW
	if hintW > contentW {
		contentW = hintW
	}

	popupOuterW := contentW + 4
	if popupOuterW > screenW {
		popupOuterW = screenW
		contentW = popupOuterW - 4
		if contentW < calendarMonthW {
			contentW = calendarMonthW
		}
	}
	innerW := popupOuterW - 2

	// 1 ヶ月分の論理行 (header + dow + 6 weeks = 8 行)。
	monthLines := renderSingleMonth(cursor, cursor, true)

	// グリッドを contentW 内で中央寄せにするための左右パディング。
	leftPad := (contentW - calendarMonthW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := contentW - calendarMonthW - leftPad
	if rightPad < 0 {
		rightPad = 0
	}
	leftPadStr := stylePopupFill.Render(strings.Repeat(" ", leftPad))
	rightPadStr := stylePopupFill.Render(strings.Repeat(" ", rightPad))

	rows := []string{}
	rows = append(rows, buildBorderRow("╭", "╮", stylePopupLabel.Render("Date:"), innerW))
	for _, line := range monthLines {
		rows = append(rows, wrapPopupContentRow(leftPadStr+line+rightPadStr))
	}
	rows = append(rows, buildBorderRow("╰", "╯", hintRendered, innerW))

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// wrapPopupContentRow は 1 行分のコンテンツを左右の罫線でくくる。
func wrapPopupContentRow(content string) string {
	return stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		content +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")
}

// renderSingleMonth は 1 ヶ月分のカレンダー行 (header + dow + 6 週 = 計 8 行) を返す。
// isCurrent=true のときのみ cursor.Day() のセルを反転表示する。
// 各行は calendarMonthW cell ぴったりにパディングされる。
func renderSingleMonth(month time.Time, cursor time.Time, isCurrent bool) []string {
	year := month.Year()
	mo := month.Month()
	loc := month.Location()
	firstOfMonth := time.Date(year, mo, 1, 0, 0, 0, 0, loc)
	daysInMonth := firstOfMonth.AddDate(0, 1, -1).Day()
	startDow := int(firstOfMonth.Weekday())

	lines := make([]string, 0, 2+calendarWeeksFixed)

	// ----- header -----
	// 表記は yyyy/M 形式 (月はゼロ埋めなし)。例: 2026/8
	header := fmt.Sprintf("%d/%d", year, int(mo))
	headerLine := stylePopupFill.Foreground(colorText).Bold(true).Render(header)
	if used := ansi.StringWidth(header); used < calendarMonthW {
		headerLine += stylePopupFill.Render(strings.Repeat(" ", calendarMonthW-used))
	}
	lines = append(lines, headerLine)

	// ----- 曜日ラベル -----
	var dowLine string
	for i, lbl := range calendarDayLabels {
		dowLine += stylePopupFill.Foreground(colorMuted).Render(lbl)
		if i < len(calendarDayLabels)-1 {
			dowLine += stylePopupFill.Render(" ")
		}
	}
	if used := ansi.StringWidth(ansi.Strip(dowLine)); used < calendarMonthW {
		dowLine += stylePopupFill.Render(strings.Repeat(" ", calendarMonthW-used))
	}
	lines = append(lines, dowLine)

	// ----- 6 週固定の日付グリッド -----
	day := 1
	for week := 0; week < calendarWeeksFixed; week++ {
		var line string
		for c := 0; c < 7; c++ {
			showDay := -1
			switch {
			case week == 0 && c < startDow:
				// 月初前の空セル
			case day > daysInMonth:
				// 月末以降の空セル
			default:
				showDay = day
				day++
			}
			if showDay < 0 {
				line += stylePopupFill.Render(strings.Repeat(" ", calendarCellW))
			} else {
				cell := fmt.Sprintf("%2d ", showDay)
				if isCurrent && showDay == cursor.Day() {
					line += stylePopupCursorRow.Render(cell)
				} else {
					line += stylePopupFill.Foreground(colorText).Render(cell)
				}
			}
			if c < 6 {
				line += stylePopupFill.Render(" ")
			}
		}
		if used := ansi.StringWidth(ansi.Strip(line)); used < calendarMonthW {
			line += stylePopupFill.Render(strings.Repeat(" ", calendarMonthW-used))
		}
		lines = append(lines, line)
	}
	return lines
}

// parseFieldDateOrToday は date 型 value の文字列から time.Time を返す。
// 空文字 / パース失敗時は今日 (ローカルタイムゾーンの 00:00) を返す。
func parseFieldDateOrToday(value string) time.Time {
	if value != "" {
		if t, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
			return t
		}
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}

// formatFieldDate は date 型 value として永続化する yyyy-mm-dd 文字列を返す。
func formatFieldDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// shiftMonth は t の年月を delta カ月ずらし、日が新しい月の末日を超える場合は末日にクランプする。
// 例: 2026-03-31 + (-1) → 2026-02-28
func shiftMonth(t time.Time, delta int) time.Time {
	loc := t.Location()
	year, month := t.Year(), int(t.Month())+delta
	for month > 12 {
		month -= 12
		year++
	}
	for month < 1 {
		month += 12
		year--
	}
	day := t.Day()
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, loc).Day()
	if day > last {
		day = last
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}
