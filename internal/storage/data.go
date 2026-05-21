package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxFileNameRunes は単一ファイル名 (basename) として許容する最大長。
// Linux の NAME_MAX (255) を rune 単位で適用 (UTF-8 だとバイト長は超える可能性があるが UX 重視)。
const MaxFileNameRunes = 255

// ErrTaskDirExists はタスク用ディレクトリ (もしくは memo.md) がすでに存在することを示す。
var ErrTaskDirExists = errors.New("task data directory already exists")

// 添付ファイル名のバリデーションエラー。AddTaskFile / RenameTaskFile が返す。
var (
	ErrFileNameEmpty   = errors.New("filename must not be empty")
	ErrFileNameTooLong = fmt.Errorf("filename must be at most %d characters", MaxFileNameRunes)
	ErrFileExists      = errors.New("file already exists")
	ErrFileNotFoundIn  = errors.New("file not found in task directory")
	// ErrInvalidRelPath はタスクディレクトリ外を参照する相対パス (絶対パス指定や
	// 親 (`..`) によるエスケープ等) を渡された場合に返される。
	ErrInvalidRelPath = errors.New("invalid relative path")
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

// taskDirNamePattern は CreateTaskData が作る "task-<int>" ディレクトリ名にマッチする。
var taskDirNamePattern = regexp.MustCompile(`^task-\d+$`)

// RemoveAllTaskData は data_base_directory 配下にある task-N (整数 N) ディレクトリを
// すべて削除する。ロード済みの id を引数に取らないことで、yaml に載っていない孤立
// ディレクトリも掃除できるようにしている。
//
//   - root が存在しない: 何もしない (エラーにしない)
//   - 命名規則 (task-<int>) に合わない子は触らない
//   - 通常ファイルや他ディレクトリは触らない
//
// 戻り値の removed には削除したディレクトリの絶対パス相当を入れる (--init の事後表示用)。
func RemoveAllTaskData(yamlDir, dataBaseDir string) (removed []string, err error) {
	root := yamlDir
	if dataBaseDir != "" {
		root = filepath.Join(yamlDir, dataBaseDir)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", root, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !taskDirNamePattern.MatchString(name) {
			continue
		}
		// 念のため数値部が int としてパース可能であることも確認 (regexp で保証されているがフェイルセーフ)。
		idStr := name[len("task-"):]
		if _, perr := strconv.Atoi(idStr); perr != nil {
			continue
		}
		full := filepath.Join(root, name)
		if rmErr := os.RemoveAll(full); rmErr != nil {
			return removed, fmt.Errorf("remove %s: %w", full, rmErr)
		}
		removed = append(removed, full)
	}
	sort.Strings(removed)
	return removed, nil
}

// FileEntry は ListTaskFileTree が返す再帰的なファイルツリーのノード。
// RelPath はタスクディレクトリからの相対パスで、セパレータは常に "/" (POSIX 形式) で保持する。
// Children は IsDir==true のときのみ意味を持ち、空ディレクトリは len(Children)==0 となる。
type FileEntry struct {
	Name     string
	RelPath  string
	IsDir    bool
	Children []FileEntry
}

// ListTaskFileTree はタスクディレクトリ配下を再帰的に走査して FileEntry の木を返す。
// 各階層内は Name 昇順 (大文字小文字は string 比較に従う)。通常ファイル / ディレクトリ以外
// (symlink, FIFO, ソケット等) は表示対象外として無視する。タスクディレクトリ自体が存在
// しない場合は (nil, nil) を返し、エラーにはしない。
func ListTaskFileTree(yamlDir, dataBaseDir string, taskID int) ([]FileEntry, error) {
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	return readDirAsTree(taskDir, "")
}

func readDirAsTree(absDir, relPrefix string) ([]FileEntry, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", absDir, err)
	}
	out := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		typ := e.Type()
		isDir := typ.IsDir()
		isReg := typ.IsRegular()
		if !isDir && !isReg {
			continue
		}
		name := e.Name()
		rel := name
		if relPrefix != "" {
			rel = relPrefix + "/" + name
		}
		fe := FileEntry{Name: name, RelPath: rel, IsDir: isDir}
		if isDir {
			children, err := readDirAsTree(filepath.Join(absDir, name), rel)
			if err != nil {
				return nil, err
			}
			fe.Children = children
		}
		out = append(out, fe)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// resolveTaskRelPath は relPath がタスクディレクトリ内に閉じていることを検証し、
// 絶対パスに変換する。空文字は relPath==taskDir 相当として扱う ("." 指定と同じ)。
// 絶対パス指定や `..` での親ディレクトリ脱出は ErrInvalidRelPath を返す。
func resolveTaskRelPath(taskDir, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("%w: %s", ErrInvalidRelPath, relPath)
	}
	// path.Clean (POSIX) で正規化することで、"sub/" や "sub//foo" を吸収する。
	// "" は "." に正規化される。
	cleaned := path.Clean(relPath)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: %s", ErrInvalidRelPath, relPath)
	}
	if cleaned == "." || cleaned == "" {
		return taskDir, nil
	}
	// filepath.FromSlash で OS 依存セパレータに戻す。
	return filepath.Join(taskDir, filepath.FromSlash(cleaned)), nil
}

// CreateFile はタスクディレクトリ内に空ファイルを新規作成する。
// relDir はタスクディレクトリからの相対パス (例: "sub" / "sub/inner")。空文字 or "." はタスクディレクトリ直下。
// 同名が既にあれば ErrFileExists を返す。タスクディレクトリや relDir が無ければ作る。
func CreateFile(yamlDir, dataBaseDir string, taskID int, relDir, fileName string) error {
	if err := ValidateFileName(fileName); err != nil {
		return err
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	dirAbs, err := resolveTaskRelPath(taskDir, relDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirAbs, 0o755); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dirAbs, err)
	}
	full := filepath.Join(dirAbs, fileName)
	f, err := os.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", ErrFileExists, full)
		}
		return fmt.Errorf("create %s: %w", full, err)
	}
	return f.Close()
}

// RenameFile はタスクディレクトリ内 (relDir 配下) のエントリ名を変更する。同一ディレクトリ内のみ。
// oldName が存在しなければ ErrFileNotFoundIn、newName が既存なら ErrFileExists。
// ファイル / ディレクトリ どちらにも適用できる (内部で `os.Rename` を使う)。
func RenameFile(yamlDir, dataBaseDir string, taskID int, relDir, oldName, newName string) error {
	if err := ValidateFileName(newName); err != nil {
		return err
	}
	if oldName == newName {
		return nil
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	dirAbs, err := resolveTaskRelPath(taskDir, relDir)
	if err != nil {
		return err
	}
	oldPath := filepath.Join(dirAbs, oldName)
	newPath := filepath.Join(dirAbs, newName)

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

// DeleteTaskData はタスクディレクトリを丸ごと削除する。
// ディレクトリが存在しない場合は no-op (エラーにしない)。trash からの完全削除で使用。
func DeleteTaskData(yamlDir, dataBaseDir string, taskID int) error {
	if taskID <= 0 {
		return errors.New("task id must be positive")
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	if _, err := os.Stat(taskDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", taskDir, err)
	}
	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("remove dir %s: %w", taskDir, err)
	}
	return nil
}

// ReadTaskFile はタスクディレクトリ内の relPath が指すファイルを先頭 maxBytes バイトまで読み込む。
// プレビュー用。ファイルが存在しない場合は os.ErrNotExist でラップしたエラーを返す。
func ReadTaskFile(yamlDir, dataBaseDir string, taskID int, relPath string, maxBytes int) (string, error) {
	if relPath == "" {
		return "", ErrFileNameEmpty
	}
	if maxBytes <= 0 {
		return "", nil
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	full, err := resolveTaskRelPath(taskDir, relPath)
	if err != nil {
		return "", err
	}
	f, err := os.Open(full)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CreateDir はタスクディレクトリ内 (relDir 配下) に新しいサブディレクトリを作成する。
// relDir 空文字 / "." はタスク直下。途中の階層も無ければ作る。
// 同名のファイル / ディレクトリが既にあれば ErrFileExists を返す。
func CreateDir(yamlDir, dataBaseDir string, taskID int, relDir, dirName string) error {
	if err := ValidateFileName(dirName); err != nil {
		return err
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	parentAbs, err := resolveTaskRelPath(taskDir, relDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(parentAbs, 0o755); err != nil {
		return fmt.Errorf("ensure dir %s: %w", parentAbs, err)
	}
	full := filepath.Join(parentAbs, dirName)
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%w: %s", ErrFileExists, full)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", full, err)
	}
	if err := os.Mkdir(full, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", full, err)
	}
	return nil
}

// DeleteDir はタスクディレクトリ内 relPath が指すディレクトリを配下ごと再帰削除する。
// タスクディレクトリ自体 (relPath="" / ".") の指定は安全のため拒否する。
// 対象が通常ファイルのときは error、不在なら ErrFileNotFoundIn を返す。
func DeleteDir(yamlDir, dataBaseDir string, taskID int, relPath string) error {
	if relPath == "" || relPath == "." {
		return errors.New("cannot delete task root directory")
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	full, err := resolveTaskRelPath(taskDir, relPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrFileNotFoundIn, full)
		}
		return fmt.Errorf("stat %s: %w", full, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", full)
	}
	if err := os.RemoveAll(full); err != nil {
		return fmt.Errorf("remove dir %s: %w", full, err)
	}
	return nil
}

// DeleteFile はタスクディレクトリ内 (relPath が指す) ファイルを削除する。
// 不在なら ErrFileNotFoundIn を返す。ディレクトリやその他特殊ファイルは対象外。
func DeleteFile(yamlDir, dataBaseDir string, taskID int, relPath string) error {
	if relPath == "" {
		return ErrFileNameEmpty
	}
	taskDir := TaskDir(yamlDir, dataBaseDir, taskID)
	full, err := resolveTaskRelPath(taskDir, relPath)
	if err != nil {
		return err
	}

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
