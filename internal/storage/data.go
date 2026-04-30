package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ErrTaskDirExists はタスク用ディレクトリ (もしくは memo.md) がすでに存在することを示す。
var ErrTaskDirExists = errors.New("task data directory already exists")

// CreateTaskData は新規タスクの情報格納先を作成する。
//   - yamlDir: tasks.yaml が置かれているディレクトリ (絶対 or 起動時の作業ディレクトリ基準)
//   - dataBaseDir: yaml の data_base_directory 値。空文字なら yamlDir 直下にタスクディレクトリを作る。
//   - title: タスクタイトル (= ディレクトリ名)
//
// 構造: <yamlDir>[/<dataBaseDir>]/<title>/memo.md
//
// 既存ディレクトリ・ファイルとの衝突は ErrTaskDirExists を返し、何も作成しない。
func CreateTaskData(yamlDir, dataBaseDir, title string) error {
	if title == "" {
		return errors.New("title must not be empty")
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, title)
	root := filepath.Dir(taskDir)
	memoPath := filepath.Join(taskDir, "memo.md")

	// 衝突チェック: タスクディレクトリ自体が既に存在しているなら作成しない。
	if _, err := os.Stat(taskDir); err == nil {
		return fmt.Errorf("%w: %s", ErrTaskDirExists, taskDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", taskDir, err)
	}

	// 親ディレクトリ (data_base_directory) は無ければ作る。
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create base dir %s: %w", root, err)
	}
	if err := os.Mkdir(taskDir, 0o755); err != nil {
		return fmt.Errorf("create task dir %s: %w", taskDir, err)
	}

	f, err := os.OpenFile(memoPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		// memo.md 作成失敗時はディレクトリも巻き戻す (作成直後で空なので安全)。
		_ = os.Remove(taskDir)
		return fmt.Errorf("create memo %s: %w", memoPath, err)
	}
	return f.Close()
}

// TaskDir は yamlDir / dataBaseDir / title を組み合わせたタスクディレクトリのパスを返す。
func TaskDir(yamlDir, dataBaseDir, title string) string {
	root := yamlDir
	if dataBaseDir != "" {
		root = filepath.Join(yamlDir, dataBaseDir)
	}
	return filepath.Join(root, title)
}

// ListTaskFiles はタスクディレクトリ内の通常ファイル名 (basename) をアルファベット順で返す。
// ディレクトリ自体が無い場合は空スライスを返し、エラーにはしない (旧タスクや手動配置の許容)。
func ListTaskFiles(yamlDir, dataBaseDir, title string) ([]string, error) {
	taskDir := TaskDir(yamlDir, dataBaseDir, title)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", taskDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
