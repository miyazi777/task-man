package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// openInOSFileManager は OS のデフォルトファイラーで path を開く。
// バックグラウンドで cmd.Start() を呼ぶだけで Wait はしない (TUI を奪わない)。
//
//   - darwin (Finder):
//     ディレクトリ → そのフォルダを開く (`open <dir>`)
//     ファイル     → 親フォルダを開きファイルを選択 (`open -R <file>`)
//   - windows (Explorer):
//     ディレクトリ → そのフォルダを開く (`explorer <dir>`)
//     ファイル     → 親フォルダを開きファイルを選択 (`explorer /select,<file>`)
//   - linux/freebsd 等:
//     xdg-open はファイル選択を伴う reveal をサポートしないため、ファイルなら
//     親ディレクトリを開く統一動作にする (`xdg-open <dir>`)。
func openInOSFileManager(path string, isDir bool) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		if isDir {
			cmd = exec.Command("open", path)
		} else {
			cmd = exec.Command("open", "-R", path)
		}
	case "windows":
		if isDir {
			cmd = exec.Command("explorer", path)
		} else {
			cmd = exec.Command("explorer", "/select,"+path)
		}
	default:
		target := path
		if !isDir {
			target = filepath.Dir(path)
		}
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file manager: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
