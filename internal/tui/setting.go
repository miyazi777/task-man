package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
)

// 設定画面のメニュー項目 (左ペイン)。
const (
	settingMenuGeneral     = 0
	settingMenuStatus      = 1
	settingMenuField       = 2
	settingMenuApplication = 3
	settingMenuFileOpener  = 4
)

var settingMenuLabels = []string{"general", "status", "field", "application", "file_opener"}

// renderSettingStatus は設定画面の status 系モード用に左メニュー + 右 status ペインを描画する。
// menuFocused=true のとき左メニュー側にカーソル反転、=false なら右ペインの statusCursor 行に反転。
// inMoveMode=true のとき右ペインのカーソル色を黄 (移動中) に切り替える。
func renderSettingStatus(statuses task.StatusList, menuCursor, statusCursor int, menuFocused, inMoveMode bool, leftW, rightW, height int) (string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	right := renderSettingStatusPane(statuses, statusCursor, !menuFocused, inMoveMode, rightW, height)
	return left, right
}

// renderSettingGeneral は設定画面の general 系モード用に左メニュー + 右 general ペインを描画する。
// rightCursor は general ペイン内の編集行カーソル (0=data_base_directory)。
// rightFocused=true のとき、右ペインのカーソル行を反転表示する。
func renderSettingGeneral(yamlPath, dataBaseDir string, menuCursor, rightCursor int, menuFocused, rightFocused bool, leftW, rightW, height int) (string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	right := renderSettingGeneralPane(yamlPath, dataBaseDir, rightCursor, rightFocused, rightW, height)
	return left, right
}

// renderSettingApplication は設定画面の application 系モード用に 3 ペインを描画する。
func renderSettingApplication(apps []storage.Application, menuCursor, appCursor, attrCursor int, menuFocused, midFocused, rightFocused, inMoveMode bool, leftW, midW, rightW, height int) (string, string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	mid := renderSettingApplicationPane(apps, appCursor, midFocused, inMoveMode, midW, height)
	right := renderSettingApplicationAttributePane(apps, appCursor, attrCursor, rightFocused, rightW, height)
	return left, mid, right
}

// renderSettingFileOpener は設定画面の file_opener 系モード用に 3 ペインを描画する。
func renderSettingFileOpener(openers []storage.FileOpener, apps []storage.Application, menuCursor, openerCursor, attrCursor int, menuFocused, midFocused, rightFocused, inMoveMode bool, leftW, midW, rightW, height int) (string, string, string) {
	left := renderSettingMenu(menuCursor, menuFocused, leftW, height)
	mid := renderSettingFileOpenerPane(openers, openerCursor, midFocused, inMoveMode, midW, height)
	right := renderSettingFileOpenerAttributePane(openers, apps, openerCursor, attrCursor, rightFocused, rightW, height)
	return left, mid, right
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

// renderSettingGeneralPane は設定画面の general 詳細を描画する。
//   - 1 行目: ヘッダ "-- general setting --"
//   - 2 行目: 現在対象の yaml ファイルパス (label "yaml: " 付き、read-only)
//   - 3 行目: data_base_directory の値 (cursor=0、focused=true でカーソル反転、編集可能)
//
// 表示幅を超える場合は末尾を ... に切り詰める。
func renderSettingGeneralPane(yamlPath, dataBaseDir string, cursor int, focused bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- general setting --")

	// yaml 行 (read-only)
	const yamlLabel = "  yaml: "
	yamlAvail := width - ansi.StringWidth(yamlLabel)
	if yamlAvail < 1 {
		yamlAvail = 1
	}
	yamlDisplay := yamlPath
	if ansi.StringWidth(yamlDisplay) > yamlAvail {
		yamlDisplay = ansi.Truncate(yamlDisplay, yamlAvail, "...")
	}
	yamlRow := lipgloss.NewStyle().Foreground(colorMuted).Render(yamlLabel) +
		lipgloss.NewStyle().Foreground(colorText).Render(yamlDisplay)

	// data_base_directory 行 (editable)
	const dbdLabel = "  data_base_directory: "
	dbdValue := dataBaseDir
	if dbdValue == "" {
		dbdValue = "(empty)"
	}
	dbdAvail := width - ansi.StringWidth(dbdLabel)
	if dbdAvail < 1 {
		dbdAvail = 1
	}
	dbdDisplay := dbdValue
	if ansi.StringWidth(dbdDisplay) > dbdAvail {
		dbdDisplay = ansi.Truncate(dbdDisplay, dbdAvail, "...")
	}
	var dbdRow string
	if focused && cursor == 0 {
		// 行全体をカーソル反転で塗る。プレーン文字列で安定描画。
		raw := dbdLabel + dbdDisplay
		dbdRow = styleCursorRow.Width(width).Render(raw)
	} else {
		valueStyle := lipgloss.NewStyle().Foreground(colorText)
		if dataBaseDir == "" {
			valueStyle = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
		}
		dbdRow = lipgloss.NewStyle().Foreground(colorMuted).Render(dbdLabel) +
			valueStyle.Render(dbdDisplay)
	}

	body := strings.Join([]string{header, yamlRow, dbdRow}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(body)
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
		{"Tab", "focus"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	// 行を contentW 幅で組み立てる小ヘルパー。
	wrap := func(content string) string {
		used := ansi.StringWidth(ansi.Strip(content))
		if used < contentW {
			content += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
		}
		return stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			content +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
	}
	emptyRow := wrap(stylePopupFill.Render(strings.Repeat(" ", contentW)))

	// ----- name 行 -----
	nameLabel := stylePopupFill.Foreground(colorAccent).Render("name: ")
	if focus != 0 {
		nameLabel = stylePopupFill.Foreground(colorMuted).Render("name: ")
	}
	if w := ansi.StringWidth(nameInputView); w > contentW-ansi.StringWidth(ansi.Strip(nameLabel)) {
		nameInputView = ansi.Truncate(nameInputView, contentW-ansi.StringWidth(ansi.Strip(nameLabel)), "")
	}
	nameRow := wrap(nameLabel + nameInputView)

	// ----- type ラベル + 縦並び選択肢 (1 行目に type: ラベルを同居) -----
	typeLabel := stylePopupFill.Foreground(colorAccent).Render("type: ")
	if focus != 1 {
		typeLabel = stylePopupFill.Foreground(colorMuted).Render("type: ")
	}
	// 2 行目以降のインデント。1 行目の "type: " (6 cell) と揃える。
	typeIndent := stylePopupFill.Render("      ")

	var typeOptionRows []string
	for i, ft := range types {
		isCurrent := ft == curType
		var marker string
		if focus == 1 && isCurrent {
			marker = stylePopupFill.Foreground(colorAccent).Bold(true).Render("> ")
		} else {
			marker = stylePopupFill.Render("  ")
		}
		var label string
		switch {
		case focus == 1 && isCurrent:
			label = stylePopupFill.Foreground(colorText).Bold(true).Render(string(ft))
		case isCurrent:
			label = stylePopupFill.Foreground(colorText).Render(string(ft))
		default:
			label = stylePopupFill.Foreground(colorMuted).Render(string(ft))
		}
		var prefix string
		if i == 0 {
			prefix = typeLabel // "type: "
		} else {
			prefix = typeIndent // "      "
		}
		typeOptionRows = append(typeOptionRows, wrap(prefix+marker+label))
	}

	rows := []string{topRow, nameRow, emptyRow}
	rows = append(rows, typeOptionRows...)
	if nameErr != nil {
		errMsg := stylePopupError.Render("! " + nameErr.Error())
		if w := ansi.StringWidth(errMsg); w > contentW {
			errMsg = ansi.Truncate(errMsg, contentW, "")
		}
		errPadded := stylePopupFill.Width(contentW).Render(errMsg)
		rows = append(rows, wrap(errPadded))
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// overlayColorPicker は色変更用ピッカーをポップアップ表示する。
// labelText はモーダル上罫線に出すラベル (例: "Status Color:" / "Tag Color:")。
// grid[row][col] が #rrggbb 形式の色 (行 = 色相、列 = 明度)。
// (curRow, curCol) のセルだけ [██] で囲んで強調、それ以外は  ██  で揃えて配置する。
//
// ポップアップ幅は「グリッド幅 / ラベル幅 / ヒント幅」のうち最大値に合わせて
// コンテンツに過不足ない大きさになるよう動的算出する。
func overlayColorPicker(bg, labelText string, grid [][]string, curRow, curCol, screenW, screenH int) string {
	cols := 0
	if len(grid) > 0 {
		cols = len(grid[0])
	}
	// セル幅 4 (左 1 + ██ + 右 1)。グリッド幅 = cols * 4。
	gridW := cols * 4
	if gridW < 8 {
		gridW = 8
	}

	labelW := ansi.StringWidth(labelText)

	// カーソルの上下左右は明らかなのでモーダル内ヒントからは省略する。
	hints := []hintItem{
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
		// グリッドを contentW 内で中央寄せにする (左右にパディング)。
		used := ansi.StringWidth(ansi.Strip(line))
		leftPad := 0
		rightPad := 0
		if used < contentW {
			leftPad = (contentW - used) / 2
			rightPad = contentW - used - leftPad
		}
		row := stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			stylePopupFill.Render(strings.Repeat(" ", leftPad)) +
			line +
			stylePopupFill.Render(strings.Repeat(" ", rightPad)) +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
		rows = append(rows, row)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// 色ピッカーのグリッド寸法 (縦長レイアウト)。
//
//	行 = 固定パレットの 12 色 (Google 風カラーパレットの上段 8 + 下段 4 を上から並べる)
//	列 = 明度 3 段階 (列ごとに V から 0.25 を差し引く)
const (
	colorPickerRows  = 12
	colorPickerCols  = 3
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

// statusColorChoices は固定 12 色パレット × 明度 3 段階の色グリッド (#rrggbb) を返す。
// grid[row][col] でアクセス。row はパレット順 (purple, indigo, blue, ...)、
// col=0 が各色のベース、col 増加で明度が 0.25 ずつ低下。
func statusColorChoices() [][]string {
	grid := make([][]string, colorPickerRows)
	for r, hex := range colorPickerBaseHexes {
		grid[r] = make([]string, colorPickerCols)
		h, s, v := hexToHSV(hex)
		for c := 0; c < colorPickerCols; c++ {
			colV := v - colorPickerVStep*float64(c)
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
	hex = strings.TrimPrefix(hex, "#")
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

// renderSettingApplicationPane は中央ペイン (application 一覧) を描画する。
func renderSettingApplicationPane(apps []storage.Application, cursor int, focused, inMoveMode bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- application setting --")
	lines := []string{header}
	if len(apps) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  (no applications)"))
	}
	for i, a := range apps {
		highlight := i == cursor && focused
		raw := "  " + a.Name
		if highlight {
			lines = append(lines, cursorStyleFor(inMoveMode).Width(width).Render(raw))
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorText).Render(raw))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// renderSettingApplicationAttributePane は右ペイン (id/name/run の 3 行) を描画する。
// id 行は read-only。
func renderSettingApplicationAttributePane(apps []storage.Application, appCursor, attrCursor int, focused bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- attributes --")
	lines := []string{header}
	if appCursor < 0 || appCursor >= len(apps) {
		return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
	}
	a := apps[appCursor]
	rows := [][2]string{
		{"id", fmt.Sprintf("%d", a.ID)},
		{"name", a.Name},
		{"run", a.Run},
	}
	const idReadonlyNote = "(read-only)"
	for i, kv := range rows {
		leftPart := "  " + kv[0] + ": " + kv[1]
		isID := kv[0] == "id"
		isCursor := focused && i == attrCursor
		if isID {
			leftW := ansi.StringWidth(leftPart)
			noteW := ansi.StringWidth(idReadonlyNote)
			padLen := width - leftW - noteW
			if padLen < 1 {
				padLen = 1
			}
			if isCursor {
				raw := leftPart + strings.Repeat(" ", padLen) + idReadonlyNote
				lines = append(lines, styleCursorRow.Width(width).Render(raw))
			} else {
				leftStyled := lipgloss.NewStyle().Foreground(colorText).Render(leftPart)
				noteStyled := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(idReadonlyNote)
				lines = append(lines, lipgloss.NewStyle().Width(width).Render(leftStyled+strings.Repeat(" ", padLen)+noteStyled))
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

// renderSettingFileOpenerPane は中央ペイン (file_opener 一覧、行は extension) を描画する。
func renderSettingFileOpenerPane(openers []storage.FileOpener, cursor int, focused, inMoveMode bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- file_opener setting --")
	lines := []string{header}
	if len(openers) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  (no openers)"))
	}
	for i, op := range openers {
		highlight := i == cursor && focused
		raw := "  ." + op.Extension
		if highlight {
			lines = append(lines, cursorStyleFor(inMoveMode).Width(width).Render(raw))
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorText).Render(raw))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// renderSettingFileOpenerAttributePane は右ペイン (extension/applications/default_app) を描画する。
// applications は app.Name のカンマ区切り、default_app は ID と name を併記する。
func renderSettingFileOpenerAttributePane(openers []storage.FileOpener, apps []storage.Application, openerCursor, attrCursor int, focused bool, width, height int) string {
	header := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("-- attributes --")
	lines := []string{header}
	if openerCursor < 0 || openerCursor >= len(openers) {
		return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
	}
	op := openers[openerCursor]
	byID := make(map[int]storage.Application, len(apps))
	for _, a := range apps {
		byID[a.ID] = a
	}
	appsLabel := joinAppNames(op.ApplicationIDs, byID)
	if appsLabel == "" {
		appsLabel = "(none)"
	}
	defLabel := "(none)"
	if op.DefaultApp != 0 {
		if a, ok := byID[op.DefaultApp]; ok {
			defLabel = fmt.Sprintf("%d:%s", a.ID, a.Name)
		} else {
			defLabel = fmt.Sprintf("%d (unknown)", op.DefaultApp)
		}
	}
	rows := [][2]string{
		{"extension", op.Extension},
		{"applications", appsLabel},
		{"default_app", defLabel},
	}
	for i, kv := range rows {
		leftPart := "  " + kv[0] + ": " + kv[1]
		if w := ansi.StringWidth(leftPart); w > width {
			leftPart = ansi.Truncate(leftPart, width, "...")
		}
		isCursor := focused && i == attrCursor
		if isCursor {
			lines = append(lines, styleCursorRow.Width(width).Render(leftPart))
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Foreground(colorText).Render(leftPart))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

// overlayApplicationAddPopup は application 追加用の 2 行モーダル (name + run) を描画する。
//   - focus=0: name 行 (textinput) にフォーカス、run 行は退避バッファを read-only 表示
//   - focus=1: run 行 (textinput) にフォーカス、name 行は退避バッファを read-only 表示
//
// inputView は textinput.Model.View() の結果。focus 側に表示する。
// nameBuf/runBuf は非フォーカス側の現在値。
func overlayApplicationAddPopup(bg, inputView string, inputErr error, focus int, nameBuf, runBuf string, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 12 {
		contentW = 12
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Add application:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"Tab", "focus"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	wrap := func(content string) string {
		used := ansi.StringWidth(ansi.Strip(content))
		if used < contentW {
			content += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
		}
		return stylePopupBorder.Render("│") +
			stylePopupFill.Render(" ") +
			content +
			stylePopupFill.Render(" ") +
			stylePopupBorder.Render("│")
	}

	mkRow := func(label string, isFocused bool, value string) string {
		var lbl string
		if isFocused {
			lbl = stylePopupFill.Foreground(colorAccent).Render(label)
		} else {
			lbl = stylePopupFill.Foreground(colorMuted).Render(label)
		}
		var valStr string
		if isFocused {
			valStr = inputView
		} else {
			if value == "" {
				value = "(empty)"
			}
			if w := ansi.StringWidth(value); w > contentW-ansi.StringWidth(ansi.Strip(lbl)) {
				value = ansi.Truncate(value, contentW-ansi.StringWidth(ansi.Strip(lbl)), "")
			}
			if value == "(empty)" {
				valStr = stylePopupFill.Foreground(colorDim).Italic(true).Render(value)
			} else {
				valStr = stylePopupFill.Foreground(colorMuted).Render(value)
			}
		}
		if w := ansi.StringWidth(valStr); w > contentW-ansi.StringWidth(ansi.Strip(lbl)) {
			valStr = ansi.Truncate(valStr, contentW-ansi.StringWidth(ansi.Strip(lbl)), "")
		}
		return wrap(lbl + valStr)
	}

	nameRow := mkRow("name: ", focus == 0, nameBuf)
	runRow := mkRow("run:  ", focus == 1, runBuf)

	rows := []string{topRow, nameRow, runRow}
	if inputErr != nil {
		errRow := wrap(stylePopupError.Render(inputErr.Error()))
		rows = append(rows, errRow)
	}
	rows = append(rows, bottomRow)

	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// overlayFileOpenerAppsPicker は applications multi-select 用モーダル。
// space で対象 toggle、enter で確定、esc でキャンセル。selected は現在選択中の ID 配列。
func overlayFileOpenerAppsPicker(bg string, apps []storage.Application, selected []int, cursor, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 12 {
		contentW = 12
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Applications:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"space", "toggle"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	selSet := make(map[int]struct{}, len(selected))
	for _, id := range selected {
		selSet[id] = struct{}{}
	}

	rows := []string{topRow}
	if len(apps) == 0 {
		rows = append(rows, lineWrap(stylePopupFill.Foreground(colorMuted).Italic(true).Render("(no applications)"), contentW))
	}
	for i, a := range apps {
		mark := " "
		if _, ok := selSet[a.ID]; ok {
			mark = "x"
		}
		raw := fmt.Sprintf("[%s] %s", mark, a.Name)
		if w := ansi.StringWidth(raw); w > contentW {
			raw = ansi.Truncate(raw, contentW, "")
		}
		var rendered string
		if i == cursor {
			rendered = stylePopupCursorRow.Width(contentW).Render(raw)
		} else {
			rendered = stylePopupFill.Foreground(colorText).Width(contentW).Render(raw)
		}
		rows = append(rows, stylePopupBorder.Render("│")+stylePopupFill.Render(" ")+rendered+stylePopupFill.Render(" ")+stylePopupBorder.Render("│"))
	}
	rows = append(rows, bottomRow)
	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// overlayFileOpenerDefaultPicker は default_app 選択用モーダル。
// 0 番は "(none)"、それ以降は applications。
func overlayFileOpenerDefaultPicker(bg string, apps []storage.Application, cursor, screenW, screenH int) string {
	popupOuterW := popupWidth(screenW)
	contentW := popupOuterW - 4
	if contentW < 12 {
		contentW = 12
	}
	innerW := popupOuterW - 2

	topRow := buildBorderRow("╭", "╮", stylePopupLabel.Render("Default app:"), innerW)
	bottomRow := buildBorderRow("╰", "╯", renderPopupHints([]hintItem{
		{"k/↑", "up"}, {"j/↓", "down"}, {"Enter", "save"}, {"Esc", "cancel"},
	}), innerW)

	rowEntries := []string{"(none)"}
	for _, a := range apps {
		rowEntries = append(rowEntries, fmt.Sprintf("%d: %s", a.ID, a.Name))
	}

	rows := []string{topRow}
	for i, label := range rowEntries {
		raw := "  " + label
		if w := ansi.StringWidth(raw); w > contentW {
			raw = ansi.Truncate(raw, contentW, "")
		}
		var rendered string
		if i == cursor {
			rendered = stylePopupCursorRow.Width(contentW).Render(raw)
		} else {
			rendered = stylePopupFill.Foreground(colorText).Width(contentW).Render(raw)
		}
		rows = append(rows, stylePopupBorder.Render("│")+stylePopupFill.Render(" ")+rendered+stylePopupFill.Render(" ")+stylePopupBorder.Render("│"))
	}
	rows = append(rows, bottomRow)
	popup := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return centerOverlay(popup, bg, screenW, screenH)
}

// lineWrap はポップアップ 1 行を contentW 幅で組み立てる小ヘルパー。
func lineWrap(content string, contentW int) string {
	used := ansi.StringWidth(ansi.Strip(content))
	if used < contentW {
		content += stylePopupFill.Render(strings.Repeat(" ", contentW-used))
	}
	return stylePopupBorder.Render("│") + stylePopupFill.Render(" ") + content + stylePopupFill.Render(" ") + stylePopupBorder.Render("│")
}

// joinAppNames は ID 配列を "name1, name2" のカンマ区切り表示に整形する。未知 ID は "id?(N)" 表示。
func joinAppNames(ids []int, byID map[int]storage.Application) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			parts = append(parts, a.Name)
		} else {
			parts = append(parts, fmt.Sprintf("id?(%d)", id))
		}
	}
	return strings.Join(parts, ", ")
}
