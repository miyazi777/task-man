package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/task"
)

// 設定画面のメニュー項目 (左ペイン)。
const (
	settingMenuStatus = 0
	settingMenuField  = 1
)

var settingMenuLabels = []string{"status", "field"}

// renderSettingStatus は設定画面の status 系モード用に左メニュー + 右 status ペインを描画する。
// menuFocused=true のとき左メニュー側にカーソル反転、=false なら右ペインの statusCursor 行に反転。
// inMoveMode=true のとき右ペインのカーソル色を黄 (移動中) に切り替える。
func renderSettingStatus(statuses task.StatusList, menuCursor, statusCursor int, menuFocused, inMoveMode bool, leftW, rightW, height int) (string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	right := renderSettingStatusPane(statuses, statusCursor, !menuFocused, inMoveMode, rightW, height)
	return left, right
}

// renderSettingField は設定画面の field 系モード用に左メニュー + 中央 field 一覧 + 右 attributes ペインを描画する。
// fieldFocus が ModeSetting のとき menuFocused=true、ModeSettingField のとき midFocused=true、
// ModeSettingFieldAttribute のとき rightFocused=true となる。
// inMoveMode=true のとき中央ペインのカーソル色を黄 (移動中) に切り替える。
func renderSettingField(fields task.FieldDefList, menuCursor, fieldCursor, attrCursor int, menuFocused, midFocused, rightFocused, inMoveMode bool, leftW, midW, rightW, height int) (string, string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	mid := renderSettingFieldPane(fields, fieldCursor, midFocused, inMoveMode, midW, height)
	right := renderSettingFieldAttributePane(fields, fieldCursor, attrCursor, rightFocused, rightW, height)
	return left, mid, right
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

// renderSettingFieldPane は設定画面 field モード時の中央ペインを描画する。
// fields は position 昇順で並べる。fieldCursor は Sorted 後のインデックス。
func renderSettingFieldPane(fields task.FieldDefList, fieldCursor int, focused, inMoveMode bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- field setting --")
	sorted := fields.Sorted()

	lines := []string{header}
	if len(sorted) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  (no fields)"))
	}
	for i, f := range sorted {
		highlight := i == fieldCursor && focused
		lines = append(lines, renderFieldSettingRow(f, highlight, inMoveMode, width))
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// renderFieldSettingRow は field 一行を描画する。highlight=true で行全体を反転、
// inMoveMode=true なら黄 (移動中) に切り替える。
func renderFieldSettingRow(f task.FieldDef, highlight, inMoveMode bool, width int) string {
	if highlight {
		raw := "  " + f.Name
		return cursorStyleFor(inMoveMode).Width(width).Render(raw)
	}
	return lipgloss.NewStyle().Width(width).Foreground(colorText).Render("  " + f.Name)
}

// renderSettingFieldAttributePane は設定画面 field モード時の右ペインを描画する。
// 行は固定で 0=name, 1=type の 2 行。focused=true のとき attrCursor 行を反転表示。
func renderSettingFieldAttributePane(fields task.FieldDefList, fieldCursor, attrCursor int, focused bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- attributes --")
	sorted := fields.Sorted()

	lines := []string{header}
	if fieldCursor < 0 || fieldCursor >= len(sorted) {
		// 対応する field が無いときは空ペイン (header のみ)
		return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
	}
	f := sorted[fieldCursor]

	rows := [][2]string{
		{"name", f.Name},
		{"type", string(f.Type)},
	}
	// type 行は read-only である旨を英文で右詰め表示する。
	const typeReadonlyNote = "(read-only)"
	for i, kv := range rows {
		leftPart := "  " + kv[0] + ": " + kv[1]
		isType := kv[0] == "type"
		isCursor := focused && i == attrCursor

		if isType {
			leftW := ansi.StringWidth(leftPart)
			noteW := ansi.StringWidth(typeReadonlyNote)
			padLen := width - leftW - noteW
			if padLen < 1 {
				padLen = 1
			}
			if isCursor {
				// カーソル行: 行全体を反転背景にして注釈もまとめて反転させる。
				raw := leftPart + strings.Repeat(" ", padLen) + typeReadonlyNote
				lines = append(lines, styleCursorRow.Width(width).Render(raw))
			} else {
				leftStyled := lipgloss.NewStyle().Foreground(colorText).Render(leftPart)
				noteStyled := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(typeReadonlyNote)
				line := leftStyled + strings.Repeat(" ", padLen) + noteStyled
				lines = append(lines, lipgloss.NewStyle().Width(width).Render(line))
			}
			continue
		}

		if isCursor {
			lines = append(lines, styleCursorRow.Width(width).Render(leftPart))
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorText).Render(leftPart))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// overlayFieldAddPopup は field 追加用の 2 行モーダルを描画する。
// focus=0: name 行 (textinput) にフォーカス。focus=1: type 行 (selector) にフォーカス。
// nameInputView は textinput.Model.View() の結果。nameErr は入力検証エラー (nil なら表示しない)。
// curType は現在選択中の FieldType。types は選択肢全体 (左右で循環する)。
func overlayFieldAddPopup(bg, nameInputView string, nameErr error, focus int, curType task.FieldType, types []task.FieldType, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 12 {
		contentW = 12
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Add field:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"Tab", "focus"}, {"←/→", "type"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	// ----- name 行 -----
	nameLabel := stylePopupFill.Foreground(colorAccent).Render("name: ")
	if focus != 0 {
		nameLabel = stylePopupFill.Foreground(colorMuted).Render("name: ")
	}
	if w := ansi.StringWidth(nameInputView); w > contentW-ansi.StringWidth(ansi.Strip(nameLabel)) {
		nameInputView = ansi.Truncate(nameInputView, contentW-ansi.StringWidth(ansi.Strip(nameLabel)), "")
	}
	namePart := nameLabel + nameInputView
	used := ansi.StringWidth(ansi.Strip(namePart))
	if used < contentW {
		namePart += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	nameRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		namePart +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	// ----- type 行 -----
	typeLabel := stylePopupFill.Foreground(colorAccent).Render("type: ")
	if focus != 1 {
		typeLabel = stylePopupFill.Foreground(colorMuted).Render("type: ")
	}
	var typeValue string
	if focus == 1 {
		typeValue = stylePopupFill.Foreground(colorText).Bold(true).Render("< " + string(curType) + " >")
	} else {
		typeValue = stylePopupFill.Foreground(colorText).Render("  " + string(curType) + "  ")
	}
	typePart := typeLabel + typeValue
	used = ansi.StringWidth(ansi.Strip(typePart))
	if used < contentW {
		typePart += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	typeRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		typePart +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	// ----- 空行 (見やすさのためのセパレータ) -----
	emptyContent := stylePopupFill.Render(strings.Repeat(" ", contentW))
	emptyRow := stylePopupBorder.Render("│") +
		stylePopupFill.Render(" ") +
		emptyContent +
		stylePopupFill.Render(" ") +
		stylePopupBorder.Render("│")

	rows := []string{topRow, nameRow, emptyRow, typeRow}
	if nameErr != nil {
		errMsg := stylePopupError.Render("! " + nameErr.Error())
		if w := ansi.StringWidth(errMsg); w > contentW {
			errMsg = ansi.Truncate(errMsg, contentW, "")
		}
		errPadded := stylePopupFill.Width(contentW).Render(errMsg)
		errRow := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			errPadded +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, errRow)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	_ = types // 将来 types を受けて長さチェックなどに使う想定
	return centerOverlay(popup, bg, screenW, screenH)
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
