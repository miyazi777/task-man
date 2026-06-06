package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/miyazi777/task-man/internal/storage"
)

// previewableExtensions はプレビュー対象とする拡張子 (小文字、ドット込み)。
var previewableExtensions = map[string]struct{}{
	".md":  {},
	".txt": {},
}

// previewMaxBytes はプレビューでファイル先頭から読み込む最大バイト数。
const previewMaxBytes = 256 * 1024

// previewNotAvailableMessage は対象外拡張子のときに表示する英語メッセージ。
const previewNotAvailableMessage = "Preview not available"

// renderPreview は cursor が指すファイルの内容をプレビューペインとして描画する。
//   - taskID==0 / fileName=="" : 空のペインを返す (プレビュー対象なし)。
//   - 拡張子が previewableExtensions に含まれない : "Preview not available" を表示。
//   - 対象拡張子 : ファイル先頭 previewMaxBytes バイトを読み込み、各行を width に切り詰めて表示。
//
// 描画結果は必ず指定の width / height ぴったりのブロックになる (lipgloss の Width/Height で揃える)。
func renderPreview(yamlDir, dataBaseDir string, taskID int, fileName string, width, height int) string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	frame := lipgloss.NewStyle().Width(width).Height(height)

	if taskID == 0 || fileName == "" {
		return frame.Render("")
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if _, ok := previewableExtensions[ext]; !ok {
		body := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(previewNotAvailableMessage)
		return frame.Render(body)
	}

	content, err := storage.ReadTaskFile(yamlDir, dataBaseDir, taskID, fileName, previewMaxBytes)
	if err != nil {
		body := lipgloss.NewStyle().Foreground(colorDanger).Render("(read error)")
		return frame.Render(body)
	}

	lines := previewLines(content, width, height)
	body := strings.Join(lines, "\n")
	return frame.Render(body)
}

// previewLines は content を行で区切り、各行を width に切り詰め、最大 height 行に
// 制限したスライスを返す。タブはスペース 4 個に展開する。
func previewLines(content string, width, height int) []string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		return nil
	}
	raw := strings.Split(content, "\n")
	if len(raw) > height {
		raw = raw[:height]
	}
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.ReplaceAll(line, "\t", "    ")
		// 制御文字 (ESC など) はそのまま流すと表示が壊れるので除去する。
		line = stripControlChars(line)
		if w := ansi.StringWidth(line); w > width {
			line = ansi.Truncate(line, width, "")
		}
		out = append(out, line)
	}
	return out
}

// renderDirPreview は cursor が指すディレクトリの直下エントリ一覧をプレビューペイン
// として描画する。
//   - taskID==0 / relPath=="" : 空のペインを返す (プレビュー対象なし)。
//   - 対象が空ディレクトリ : "(empty)" を表示。
//   - 読み込みエラー : "(read error)" を表示 (renderPreview と統一)。
//   - 通常時 : ListTaskDirChildren が返した順 (Name 昇順) に、ディレクトリは末尾 "/" 付き、
//     ファイルは name のみで 1 行ずつ並べる。
//
// 行数が height を超える場合は単純に末尾を切り詰める (renderPreview の previewLines と
// 同じ方針)。各行の幅は ansi.Truncate で末尾 "…" 省略。
func renderDirPreview(yamlDir, dataBaseDir string, taskID int, relPath string, width, height int) string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	frame := lipgloss.NewStyle().Width(width).Height(height)

	if taskID == 0 || relPath == "" {
		return frame.Render("")
	}

	entries, err := storage.ListTaskDirChildren(yamlDir, dataBaseDir, taskID, relPath)
	if err != nil {
		body := lipgloss.NewStyle().Foreground(colorDanger).Render("(read error)")
		return frame.Render(body)
	}
	if len(entries) == 0 {
		body := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("(empty)")
		return frame.Render(body)
	}

	limit := len(entries)
	if limit > height {
		limit = height
	}
	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		name := entries[i].Name
		if entries[i].IsDir {
			name += "/"
		}
		if w := ansi.StringWidth(name); w > width {
			name = ansi.Truncate(name, width, "…")
		}
		lines = append(lines, name)
	}
	return frame.Render(strings.Join(lines, "\n"))
}

// stripControlChars は表示を破壊しうる制御文字を空文字に置換する。
// 改行 (\n) は呼び出し側が事前に分割しているのでここでは現れない想定。
func stripControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
