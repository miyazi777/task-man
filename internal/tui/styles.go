package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/miyazi777/task-man/internal/task"
)

// モック (docs/mockups/*.svg) の Catppuccin 系カラーをベースに調整。
var (
	colorText     = lipgloss.Color("#cdd6f4")
	colorMuted    = lipgloss.Color("#a6adc8")
	colorDim      = lipgloss.Color("#7f849c")
	colorSubtle   = lipgloss.Color("#6c7086")
	colorAccent   = lipgloss.Color("#89b4fa") // フォーカス・カーソル
	colorBase     = lipgloss.Color("#1e1e2e") // カーソル反転時の前景 (= 通常背景色)
	colorWarn     = lipgloss.Color("#f9e2af") // ModeMove のカーソル / バナー
	colorDanger   = lipgloss.Color("#f38ba8") // y:quit
	colorOK       = lipgloss.Color("#a6e3a1") // n:cancel
	colorDivider  = lipgloss.Color("#313244")
	colorFooterBg = lipgloss.Color("#313244")
)

var (
	styleListItem    = lipgloss.NewStyle().Foreground(colorText).Padding(0, 0)
	styleListItemDim = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 0)
	styleLabel       = lipgloss.NewStyle().Foreground(colorSubtle)
	styleValue       = lipgloss.NewStyle().Foreground(colorText)
	styleValueDim    = lipgloss.NewStyle().Foreground(colorSubtle)
	styleDivider     = lipgloss.NewStyle().Foreground(colorDivider)
	styleFooter      = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorMuted).Padding(0, 1).Width(0)
	styleFooterKey   = lipgloss.NewStyle().Background(colorFooterBg).Foreground(colorText).Bold(true)
	colorPopupBg     = lipgloss.Color("#11111b")

	stylePopupLabel  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorPopupBg)
	stylePopupHint   = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Background(colorPopupBg)
	stylePopupKey    = lipgloss.NewStyle().Foreground(colorText).Bold(true).Background(colorPopupBg)
	stylePopupFill   = lipgloss.NewStyle().Background(colorPopupBg)
	stylePopupBorder = lipgloss.NewStyle().Foreground(colorAccent).Background(colorPopupBg)
	stylePopupError  = lipgloss.NewStyle().Foreground(colorDanger).Background(colorPopupBg).Bold(true)

	// styleCursorRow は yazi 風の反転カーソル: 行全体の背景をアクセント色、文字を base 色 (= 通常背景) で塗る。
	// フォーカス中のリスト/詳細/Files/ピッカーで共通利用する。角は丸めない。
	styleCursorRow = lipgloss.NewStyle().Background(colorAccent).Foreground(colorBase)
	// styleMoveCursorRow は ModeMove 中の反転カーソル: 黄 (colorWarn) 背景で「移動中」を強調する。
	styleMoveCursorRow = lipgloss.NewStyle().Background(colorWarn).Foreground(colorBase)
	// styleMoveBanner はリスト右上に表示する移動モードバナー。
	styleMoveBanner = lipgloss.NewStyle().Background(colorWarn).Foreground(colorBase).Bold(true).Padding(0, 1)
	// styleOperationBanner はリスト右上に表示する operation モードバナー (黒抜き + アクセント色背景)。
	styleOperationBanner = lipgloss.NewStyle().Background(colorAccent).Foreground(colorBase).Bold(true).Padding(0, 1)
	// styleTrashHeader はゴミ箱ビュー時にリスト最上部 1 行を占有するヘッダ「-- TRASH BOX --」。
	// 黒抜き (colorBase) + 赤背景 (colorDanger)、太字、左ペイン全幅で中央寄せ。
	styleTrashHeader = lipgloss.NewStyle().Background(colorDanger).Foreground(colorBase).Bold(true).Align(lipgloss.Center)
	// stylePopupCursorRow はポップアップ背景 (colorPopupBg) 上で同じ反転表現を出すための変種。
	stylePopupCursorRow = lipgloss.NewStyle().Background(colorAccent).Foreground(colorBase)
)

var _ = styleFooter // app.go 直接組み立てているが将来用に保持

// statusStyleFor は Status の色設定を反映した強調スタイルを返す。
// Color 未指定 (空文字) のときは muted にフォールバック。
func statusStyleFor(s task.Status) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	if s.Color != "" {
		return base.Foreground(lipgloss.Color(s.Color))
	}
	return base.Foreground(colorMuted)
}

// statusRowStyleFor は Status の色を背景にした反転スタイル (黒抜き文字) を返す。
// リスト画面のステータス見出し行で利用する。
func statusRowStyleFor(s task.Status) lipgloss.Style {
	bg := colorMuted
	if s.Color != "" {
		bg = lipgloss.Color(s.Color)
	}
	return lipgloss.NewStyle().Background(bg).Foreground(colorBase)
}

// tagChipStyleFor はタグの色を背景にした反転スタイル (黒抜き文字) を返す。
// Color 未指定なら colorMuted にフォールバック。
func tagChipStyleFor(tg task.Tag) lipgloss.Style {
	bg := colorMuted
	if tg.Color != "" {
		bg = lipgloss.Color(tg.Color)
	}
	return lipgloss.NewStyle().Background(bg).Foreground(colorBase)
}

// renderTagChip は " <name> " 形式 (両端 1 cell パッド) のカラーチップを返す。
func renderTagChip(tg task.Tag) string {
	return tagChipStyleFor(tg).Render(" " + tg.Name + " ")
}
