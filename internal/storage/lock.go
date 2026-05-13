package storage

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// ErrAlreadyLocked は別プロセスがすでにロックを保持していることを示す。
var ErrAlreadyLocked = errors.New("another process is already using this file")

// lockFilePath はロックファイルのパスを返す。yaml ファイルと同じディレクトリに
// ".task-man.lock" を配置する。
func lockFilePath(yamlPath string) string {
	return yamlPath + ".lock"
}

// acquireLock は yamlPath に対応するロックファイルを作成し、flock(LOCK_EX|LOCK_NB) で
// 排他ロックをノンブロッキングで取得する。
// 成功時はロックファイルの *os.File を返す (呼び出し側が Close するまでロックを保持)。
// 別プロセスがロックを保持している場合は ErrAlreadyLocked を返す。
func acquireLock(yamlPath string) (*os.File, error) {
	lp := lockFilePath(yamlPath)
	f, err := os.OpenFile(lp, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lp, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyLocked, yamlPath)
		}
		return nil, fmt.Errorf("flock %s: %w", lp, err)
	}
	return f, nil
}

// releaseLock はロックファイルを閉じてロックを解放する。
// f が nil の場合は何もしない (ロック未取得時の安全な呼び出し)。
func releaseLock(f *os.File) error {
	if f == nil {
		return nil
	}
	return f.Close()
}
