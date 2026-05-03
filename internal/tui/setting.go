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
// grid[row][col] が #rrggbb 形式の色 (行 = 色相、列 = 明度)。
// (curRow, curCol) のセルだけ [██] で囲んで強調、それ以外は  ██  で揃えて配置する。
//
// ポップアップ幅は「グリッド幅 / ラベル幅 / ヒント幅」のうち最大値に合わせて
// コンテンツに過不足ない大きさになるよう動的算出する。
func overlayColorPicker(bg string, grid [][]string, curRow, curCol, screenW, screenH int) string {
	cols := 0
	if len(grid) > 0 {
		cols = len(grid[0])
	}
	// セル幅 4 (左 1 + ██ + 右 1)。グリッド幅 = cols * 4。
	gridW := cols * 4
	if gridW < 8 {
		gridW = 8
	}

	labelText := "Status Color:"
	labelW := ansi.StringWidth(labelText)

	hints := []hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"h/←", "left"}, {"l/→", "right"},
		{"Enter", "save"}, {"Esc", "cancel"},
	}
	hintRendered := renderPopupHints(hints)
	hintW := ansi.StringWidth(ansi.Strip(hintRendered))

	contentW := gridW
	if labelW > contentW {
		contentW = labelW
	}
	if hintW > contentW {
		contentW = hintW
	}
	popupOuterW := contentW + 4 // 左右の border (2) + 余白 (2)
	if popupOuterW > screenW {
		popupOuterW = screenW
		contentW = popupOuterW - 4
		if contentW < gridW {
			contentW = gridW
		}
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render(labelText), innerW)
	bottomRow := buildBorderRow("╰", "╯", hintRendered, innerW)

	rows := []string{topRow}
	for r, rowCells := range grid {
		// 1 行 = cols * (左マーカー1 + 色2 + 右マーカー1) = cols * 4 cell
		var line string
		for c, hex := range rowCells {
			active := r == curRow && c == curCol
			swatch := lipgloss.NewStyle().Background(colorPopupBg).Foreground(lipgloss.Color(hex)).Render("██")
			var leftMk, rightMk string
			if active {
				leftMk = stylePopupFill.Foreground(colorText).Bold(true).Render("[")
				rightMk = stylePopupFill.Foreground(colorText).Bold(true).Render("]")
			} else {
				leftMk = stylePopupFill.Render(" ")
				rightMk = stylePopupFill.Render(" ")
			}
			line += leftMk + swatch + rightMk
		}
		used := ansi.StringWidth(ansi.Strip(line))
		if used < contentW {
			line += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
		}
		row := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			line +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, row)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// 色ピッカーのグリッド寸法。
//   行 = 明度 3 段階 (行ごとに V から 0.25 を差し引く)
//   列 = 固定パレットの 12 色 (Google 風カラーパレットの上段 8 + 下段 4 を左から順に並べたもの)
const (
	colorPickerRows  = 3
	colorPickerCols  = 12
	colorPickerVStep = 0.25
)

// colorPickerBaseHexes は色ピッカーのベース色 (列 0)。
// 画像参照 — 1 段目左→右: purple, indigo, blue, cyan, green, lime, yellow, orange、
// 2 段目左→右: red, pink, brown, grey の順。これらをこの画面では上→下に展開する。
var colorPickerBaseHexes = []string{
	"#a855f7", // purple
	"#6366f1", // indigo
	"#3b82f6", // blue
	"#0ea5e9", // sky / cyan
	"#22c55e", // green
	"#84cc16", // lime
	"#eab308", // yellow
	"#f97316", // orange
	"#ef4444", // red
	"#ec4899", // pink
	"#a16207", // brown
	"#9ca3af", // grey
}

// statusColorChoices は明度 3 段階 × 固定 12 色パレットの色グリッド (#rrggbb) を返す。
// grid[row][col] でアクセス。row=0 が各色のベース、row 増加で明度が 0.25 ずつ低下。
// col はパレット順 (purple, indigo, blue, ...)。
func statusColorChoices() [][]string {
	grid := make([][]string, colorPickerRows)
	for r := 0; r < colorPickerRows; r++ {
		grid[r] = make([]string, colorPickerCols)
		for c, hex := range colorPickerBaseHexes {
			h, s, v := hexToHSV(hex)
			colV := v - colorPickerVStep*float64(r)
			if colV < 0 {
				colV = 0
			}
			grid[r][c] = hsvToHex(h, s, colV)
		}
	}
	return grid
}

// nearestColorChoiceCell は grid 内で currentColor に最も近い (row, col) を返す。
// 距離は hue 差 + S 差 + V 差で評価する。S が 0 に近い (= グレー) 同士なら hue 差は無視。
// grid が空なら (0, 0)。
func nearestColorChoiceCell(grid [][]string, currentColor string) (int, int) {
	if len(grid) == 0 {
		return 0, 0
	}
	curH, curS, curV := hexToHSV(currentColor)
	bestRow, bestCol := 0, 0
	bestDiff := math.MaxFloat64
	for r, row := range grid {
		for c, hex := range row {
			h, s, v := hexToHSV(hex)
			dH := math.Abs(h - curH)
			if dH > 180 {
				dH = 360 - dH
			}
			// グレー同士は hue 比較しない (グレーでは hue が定義されない)。
			if s < 0.05 || curS < 0.05 {
				dH = 0
			}
			// hue は 0..360、S/V は 0..1。S/V を 360 倍してスケールを揃える。
			dS := math.Abs(s-curS) * 360
			dV := math.Abs(v-curV) * 360
			diff := dH + dS + dV
			if diff < bestDiff {
				bestDiff = diff
				bestRow, bestCol = r, c
			}
		}
	}
	return bestRow, bestCol
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
