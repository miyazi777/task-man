package tui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openURLInBrowser は OS のデフォルトハンドラで rawURL を開く。
// バックグラウンドで cmd.Start() を呼ぶだけで Wait はしない (TUI を奪わない)。
//   - linux/freebsd 等: xdg-open
//   - darwin: open
//   - windows: rundll32 url.dll,FileProtocolHandler
func openURLInBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open url: %w", err)
	}
	// Linux 系では子プロセスを Reap しないと <defunct> として残るため Release で切り離す。
	go func() { _ = cmd.Wait() }()
	return nil
}
