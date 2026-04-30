package tui

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// ErrEditorNotConfigured は applications.editor も $EDITOR も空のときに返るエラー。
var ErrEditorNotConfigured = errors.New("editor is not configured: set applications.editor in tasks.yaml or $EDITOR env var")

// buildEditorCmd は applications.editor の値とファイルパスから *exec.Cmd を組み立てる。
// editor 文字列は os.ExpandEnv で展開され ("$EDITOR" → "vim" など)、
// 展開後が空なら $EDITOR を直接フォールバック先として使う。
// それでも空なら ErrEditorNotConfigured を返す。
// 引数を含むコマンド ("nvim --noplugin") も空白区切りで扱う。
func buildEditorCmd(editor, filePath string) (*exec.Cmd, error) {
	expanded := os.ExpandEnv(strings.TrimSpace(editor))
	if expanded == "" {
		expanded = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if expanded == "" {
		return nil, ErrEditorNotConfigured
	}
	parts := strings.Fields(expanded)
	bin := parts[0]
	args := append([]string{}, parts[1:]...)
	args = append(args, filePath)
	return exec.Command(bin, args...), nil
}
