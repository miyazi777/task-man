package tui

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// ErrEditorNotConfigured はアプリケーションパスも $EDITOR も空のときに返るエラー。
var ErrEditorNotConfigured = errors.New("no application configured: add an application to tasks.yaml or set $EDITOR env var")

// buildAppCmd はアプリケーションパスとファイルパスから *exec.Cmd を組み立てる。
// appPath は os.ExpandEnv で展開され ("$EDITOR" → "vim" など)、
// 展開後が空なら $EDITOR を直接フォールバック先として使う。
// それでも空なら ErrEditorNotConfigured を返す。
// 引数を含むコマンド ("nvim --noplugin") も空白区切りで扱う。
func buildAppCmd(appPath, filePath string) (*exec.Cmd, error) {
	expanded := os.ExpandEnv(strings.TrimSpace(appPath))
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
