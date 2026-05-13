package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/miyazi777/task-man/internal/task"
)

// TestRenderListNarrowWidthKeepsStatusHeadersOnSingleLine は #15 の回帰テスト。
// 横幅が狭い (status ラベルが入りきらない幅) ときに、ステータス見出しが
// 自動 word-wrap で 2 行に分裂し最上段が「marker だけの行」と
// 「ラベルだけの行」に割れる挙動を防ぐ。
//
// 期待: どの widths でも各ステータスは 1 行に収まり、line 0 は最上段の
// ステータス見出し (truncated でも marker + 何らかのラベル文字または "…")。
func TestRenderListNarrowWidthKeepsStatusHeadersOnSingleLine(t *testing.T) {
	statuses := task.StatusList{
		{ID: 1, Sequence: 1, Label: "リリース待ち"},
		{ID: 2, Sequence: 2, Label: "レビュー中"},
		{ID: 3, Sequence: 3, Label: "PR済"},
	}
	tasks := []task.Task{}
	rows := buildRows(statuses, tasks, nil, nil, false)

	// status 数 (3) + separator 数 (2) = 5 行が期待値。height はそれを十分に超える 10。
	const height = 10

	for _, w := range []int{30, 20, 14, 10, 8} {
		t.Run("width="+itoa(w), func(t *testing.T) {
			// cursor=0 (最上段) で focused、move/trash 無し。
			out := renderList(tasks, statuses, nil, rows, nil, 0, true, false, w, height)
			stripped := ansi.Strip(out)
			lines := strings.Split(stripped, "\n")

			// 行数は status(3) + sep(2) + height 不足分の padding なので、ちょうど 10 行になるはず。
			if len(lines) != height {
				t.Fatalf("expected %d lines, got %d:\n%s", height, len(lines), strings.Join(lines, "\n"))
			}

			// line 0 は最上段の status header。少なくともラベル先頭か "…" を含むこと。
			// 自動 wrap で marker だけの行になる回帰を防ぐ。
			line0 := strings.TrimRight(lines[0], " ")
			if !strings.ContainsAny(line0, "リ…") {
				t.Errorf("line 0 lost top status label (width=%d): %q", w, lines[0])
			}

			// 2 番目の status は line 2 にあるはず (line 1 は separator)。
			line2 := strings.TrimRight(lines[2], " ")
			if !strings.ContainsAny(line2, "レ…") {
				t.Errorf("line 2 lost second status label (width=%d): %q", w, lines[2])
			}
		})
	}
}

// itoa は依存を減らすための簡易整数→文字列変換。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
