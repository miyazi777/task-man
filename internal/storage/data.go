package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

// MaxFileNameRunes は単一ファイル名 (basename) として許容する最大長。
// Linux の NAME_MAX (255) を rune 単位で適用 (UTF-8 だとバイト長は超える可能性があるが UX 重視)。
const MaxFileNameRunes = 255

// ErrTaskDirExists はタスク用ディレクトリ (もしくは memo.md) がすでに存在することを示す。
var ErrTaskDirExists = errors.New("task data directory already exists")

var (
	ErrFileNameEmpty   = errors.New("filename must not be empty")
	ErrFileNameTooLong = fmt.Errorf("filename must be at most %d characters", MaxFileNameRunes)
	ErrFileExists      = errors.New("file already exists")
	ErrFileNotFoundIn  = errors.New("file not found in task directory")
)

// FileNameForbiddenCharError は使用できない文字がファイル名に含まれていることを示す。
type FileNameForbiddenCharError struct {
	Char rune
}

func (e *FileNameForbiddenCharError) Error() string {
	switch e.Char {
	case 0:
		return "null is not allowed in filename"
	case '/':
		return "slash (/) is not allowed in filename"
	default:
		return fmt.Sprintf("'%c' is not allowed in filename", e.Char)
	}
}

// ValidateFileNameChars はライブ入力検証用。空文字は許容 (入力途中の状態)。
// 禁止: \0 (POSIX), / (パス区切り)。長さ超過もここで弾く。
func ValidateFileNameChars(name string) error {
	if utf8.RuneCountInString(name) > MaxFileNameRunes {
		return ErrFileNameTooLong
	}
	for _, r := range name {
		switch r {
		case 0, '/':
			return &FileNameForbiddenCharError{Char: r}
		}
	}
	return nil
}

// ValidateFileName は確定時に呼ぶ。空文字も拒否する点が ValidateFileNameChars との違い。
func ValidateFileName(name string) error {
	if name == "" {
		return ErrFileNameEmpty
	}
	return ValidateFileNameChars(name)
}

// CreateTaskData は新規タスクの情報格納先を作成する。
//   - yamlDir: tasks.yaml が置かれているディレクトリ (絶対 or 起動時の作業ディレクトリ基準)
//   - dataBaseDir: yaml の data_base_directory 値。空文字なら yamlDir 直下にタスクディレクトリを作る。
//   - taskID: タスク ID (ディレクトリ名は task-{id})
//
// 構造: <yamlDir>[/<dataBaseDir>]/task-<id>/memo.md
//
// 既存ディレクトリ・ファイルとの衝突は ErrTaskDirExists を返し、何も作成しない。
func CreateTaskData(yamlDir, dataBaseDir string, taskID int) error {
	if taskID <= 0 {
		return errors.New("task id must be positive")
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
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

// TaskDir は yamlDir / dataBaseDir / task-{id} を組み合わせたタスクディレクトリのパスを返す。
func TaskDir(yamlDir, dataBaseDir string, taskID int) string {
	root := yamlDir
	if dataBaseDir != "" {
		root = filepath.Join(yamlDir, dataBaseDir)
	}
	return filepath.Join(root, fmt.Sprintf("task-%d", taskID))
}

// ListTaskFiles はタスクディレクトリ内の通常ファイル名 (basename) をアルファベット順で返す。
// ディレクトリ自体が無い場合は空スライスを返し、エラーにはしない (旧タスクや手動配置の許容)。
func ListTaskFiles(yamlDir, dataBaseDir string, taskID int) ([]string, error) {
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
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

// CreateFile はタスクディレクトリ内に空ファイルを新規作成する。
// 同名が既にあれば ErrFileExists を返す。タスクディレクトリが無ければ作る。
func CreateFile(yamlDir, dataBaseDir string, taskID int, fileName string) error {
	if err := ValidateFileName(fileName); err != nil {
		return err
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return fmt.Errorf("ensure task dir %s: %w", taskDir, err)
	}
	full := filepath.Join(taskDir, fileName)
	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", ErrFileExists, full)
		}
		return fmt.Errorf("create %s: %w", full, err)
	}
	return f.Close()
}

// RenameFile はタスクディレクトリ内のファイル名を変更する。
// oldName が存在しなければ ErrFileNotFoundIn、newName が既存なら ErrFileExists。
func RenameFile(yamlDir, dataBaseDir string, taskID int, oldName, newName string) error {
	if err := ValidateFileName(newName); err != nil {
		return err
	}
	if oldName == newName {
		return nil
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	oldPath := filepath.Join(taskDir, oldName)
	newPath := filepath.Join(taskDir, newName)

	if _, err := os.Stat(oldPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrFileNotFoundIn, oldPath)
		}
		return fmt.Errorf("stat %s: %w", oldPath, err)
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("%w: %s", ErrFileExists, newPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", newPath, err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}
	return nil
}

// DeleteFile はタスクディレクトリ内のファイルを削除する。
// 不在なら ErrFileNotFoundIn を返す。ディレクトリやその他特殊ファイルは対象外。
func DeleteFile(yamlDir, dataBaseDir string, taskID int, fileName string) error {
	if fileName == "" {
		return ErrFileNameEmpty
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	full := filepath.Join(taskDir, fileName)

	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrFileNotFoundIn, full)
		}
		return fmt.Errorf("stat %s: %w", full, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", full)
	}
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("remove %s: %w", full, err)
	}
	return nil
}
