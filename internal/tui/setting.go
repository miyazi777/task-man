package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/task"
)

// 設定画面のメニュー項目 (左ペイン)。今は status のみ。
const (
	settingMenuStatus = 0
)

var settingMenuLabels = []string{"status"}

// renderSetting は設定画面を描画する。
// menuFocused=true のとき左メニュー側にカーソル反転、=false なら右ペインの statusCursor 行に反転。
// inMoveMode=true のとき右ペインのカーソル色を黄 (移動中) に切り替える。
// statusCursor は m.statuses.Sorted() のインデックス。
func renderSetting(statuses task.StatusList, menuCursor, statusCursor int, menuFocused, inMoveMode bool, leftW, rightW, height int) (string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	right := renderSettingStatusPane(statuses, statusCursor, !menuFocused, inMoveMode, rightW, height)
	return left, right
}

func renderSettingMenu(menuCursor int, focused bool, width, height int) string {
	var lines []string
	for i, label := range settingMenuLabels {
		row := " " + label + " "
		if i == menuCursor && focused {
			lines = append(lines, styleCursorRow.Width(width).Render(row))
		} else if i == menuCursor {
			// 非フォーカス時もカーソル行は控えめに見せる
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorText).Render(row))
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorMuted).Render(row))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func renderSettingStatusPane(statuses task.StatusList, statusCursor int, focused, inMoveMode bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- status setting --")
	sorted := statuses.Sorted()

	lines := []string{header}
	for i, s := range sorted {
		highlight := i == statusCursor && focused
		lines = append(lines, renderStatusSettingRow(s, highlight, inMoveMode, width))
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// renderStatusSettingRow は status 一行をタスクリスト内のステータス表示 (statusRowStyleFor)
// と同じく「カラー背景に黒抜きラベル」で描画する。
// highlight=true のときは行全体にカーソル背景を敷き、inMoveMode=true なら黄 (移動中) に
// 切り替える。
func renderStatusSettingRow(s task.Status, highlight, inMoveMode bool, width int) string {
	if highlight {
		raw := "  " + s.Label + " "
		return cursorStyleFor(inMoveMode).Width(width).Render(raw)
	}
	prefix := " "
	labelPart := statusRowStyleFor(s).Render(" " + s.Label + " ")
	return lipgloss.NewStyle().Width(width).Render(prefix + labelPart)
}

// renderColorSwatch は ██ 2 cell の色見本を生成する (ピッカー用、背景は素のまま)。
func renderColorSwatch(hex string) string {
	if hex == "" {
		return "  "
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
}

// overlayColorPicker は status の色変更用ピッカーをポップアップ表示する。
// choices は #rrggbb 形式の 8 色 (色相順)。currentIdx は選択中インデックス。
func overlayColorPicker(bg string, choices []string, currentIdx, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 8 {
		contentW = 8
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Status Color:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	rows := []string{topRow}
	for i, hex := range choices {
		marker := "  "
		if i == currentIdx {
			marker = "> "
		}
		swatch := lipgloss.NewStyle().Background(colorPopupBg).Foreground(lipgloss.Color(hex)).Render("██")
		mk := stylePopupFill.Foreground(colorText).Render(marker)
		raw := mk + swatch
		// pad to contentW (背景色を popup bg に揃える)
		used := ansi.StringWidth(ansi.Strip(raw))
		if used < contentW {
			raw += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
		}
		row := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			raw +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, row)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// statusColorChoices は currentColor の S/V を保持しつつ、Hue を 0,45,...,315 に振った
// 8 色 (#rrggbb) を色相順で返す。currentColor が無効/グレーなら適当な S/V を補う。
func statusColorChoices(currentColor string) []string {
	_, s, v := hexToHSV(currentColor)
	if s < 0.2 {
		s = 0.6
	}
	if v < 0.2 {
		v = 0.85
	}
	out := make([]string, 8)
	for i := 0; i < 8; i++ {
		hue := float64(i) * 45.0
		out[i] = hsvToHex(hue, s, v)
	}
	return out
}

// nearestColorChoiceIndex は choices の中から currentColor に最も近い hue のインデックスを返す。
func nearestColorChoiceIndex(choices []string, currentColor string) int {
	curH, _, _ := hexToHSV(currentColor)
	best := 0
	bestDiff := 360.0
	for i, hex := range choices {
		h, _, _ := hexToHSV(hex)
		d := math.Abs(h - curH)
		if d > 180 {
			d = 360 - d
		}
		if d < bestDiff {
			bestDiff = d
			best = i
		}
	}
	return best
}

// hsvToHex は HSV (h: 0..360, s/v: 0..1) を #rrggbb に変換する。
func hsvToHex(h, s, v float64) string {
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	mm := v - c
	var rf, gf, bf float64
	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}
	r := uint8(math.Round((rf + mm) * 255))
	g := uint8(math.Round((gf + mm) * 255))
	b := uint8(math.Round((bf + mm) * 255))
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// hexToHSV は #rrggbb を HSV (h: 0..360, s/v: 0..1) に変換する。
// 解釈失敗時は (0, 0, 0) を返す。
func hexToHSV(hex string) (float64, float64, float64) {
	hex = strings.TrimSpace(hex)
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 0, 0, 0
	}
	var ri, gi, bi int
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &ri, &gi, &bi); err != nil {
		return 0, 0, 0
	}
	rf := float64(ri) / 255.0
	gf := float64(gi) / 255.0
	bf := float64(bi) / 255.0
	maxv := math.Max(rf, math.Max(gf, bf))
	minv := math.Min(rf, math.Min(gf, bf))
	delta := maxv - minv
	v := maxv
	var s float64
	if maxv == 0 {
		s = 0
	} else {
		s = delta / maxv
	}
	var h float64
	if delta == 0 {
		h = 0
	} else if maxv == rf {
		h = 60 * math.Mod((gf-bf)/delta, 6)
	} else if maxv == gf {
		h = 60 * ((bf-rf)/delta + 2)
	} else {
		h = 60 * ((rf-gf)/delta + 4)
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}
