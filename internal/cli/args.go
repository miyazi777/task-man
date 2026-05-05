package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

const DefaultFileName = "tasks.yaml"

type Args struct {
	Path string
	// MustExist が true の場合、ファイルが存在しなければエラー終了。
	// false の場合は存在しなければ作成して良い。
	MustExist bool
}

// Parse は os.Args[1:] (またはテスト用の任意のスライス) を解析する。
// -t / --tasks で yaml パスを明示できる。alias 経由でクオート付きで渡された
// 場合のフォールバックとして、先頭の "~" / "~/" をホームディレクトリに展開する。
func Parse(argv []string) (*Args, error) {
	fs := pflag.NewFlagSet("task-man", pflag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	taskPath := fs.StringP("tasks", "t", "", "tasks yaml file path")
	if err := fs.Parse(argv); err != nil {
		return nil, err
	}

	if *taskPath != "" {
		expanded, err := expandHome(*taskPath)
		if err != nil {
			return nil, err
		}
		return &Args{Path: expanded, MustExist: true}, nil
	}
	return &Args{Path: DefaultFileName, MustExist: false}, nil
}

// expandHome は先頭が "~" または "~/" のパスをユーザのホームディレクトリに置換する。
// "~user" 形式 (他ユーザのホーム参照) は対応しない。
func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	if path != "~" && !strings.HasPrefix(path, "~/") {
		// "~user/foo" のようなパターンは未対応 (シェルに任せる)。
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve ~: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

// EnsureFile は引数の指針に従ってファイルの存在を保証する。
// MustExist=true で不在 → エラー。MustExist=false で不在 → 空ファイルを作成。
func EnsureFile(a *Args) error {
	_, err := os.Stat(a.Path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", a.Path, err)
	}
	if a.MustExist {
		return fmt.Errorf("file not found: %s", a.Path)
	}
	f, err := os.Create(a.Path)
	if err != nil {
		return fmt.Errorf("create %s: %w", a.Path, err)
	}
	return f.Close()
}
