package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/miyazi777/task-man/internal/cli"
	"github.com/miyazi777/task-man/internal/storage"
	"github.com/miyazi777/task-man/internal/task"
	"github.com/miyazi777/task-man/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	args, err := cli.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	// 相対パスで指定された場合でも、起動後の CWD 変化に依存しないよう絶対パスへ正規化する。
	absPath, err := filepath.Abs(args.Path)
	if err != nil {
		return err
	}

	if args.Init {
		return runInit(absPath, os.Stdin, os.Stdout)
	}

	if err := cli.EnsureFile(args); err != nil {
		return err
	}

	repo := storage.NewYAMLRepository(absPath)
	lr, err := repo.Load()
	if err != nil {
		return err
	}

	model := tui.NewModel(repo, lr.Tasks, lr.Statuses, lr.Fields, lr.Tags, absPath, lr.Config)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// runInit は -i / --init 指定時の初期化フロー。
//   - 既存 yaml が読み込めれば config (data_base_directory / editor) を保持
//   - data_base_directory 配下の task-<int> ディレクトリを全削除
//   - yaml をデフォルト 3 status のみで書き直し
//   - 実行前に y/N 確認 (no/空回答ならキャンセル)
func runInit(yamlPath string, in io.Reader, out io.Writer) error {
	yamlDir := filepath.Dir(yamlPath)

	// 既存 yaml をベストエフォートで読み込む。読めれば config を引き継ぐ。
	var existingCfg storage.AppConfig
	yamlExists := true
	if _, statErr := os.Stat(yamlPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			yamlExists = false
		} else {
			return fmt.Errorf("stat %s: %w", yamlPath, statErr)
		}
	}
	if yamlExists {
		repo := storage.NewYAMLRepository(yamlPath)
		if lr, lerr := repo.Load(); lerr == nil {
			existingCfg = lr.Config
		} else {
			return fmt.Errorf("load existing yaml (fix or remove it before --init): %w", lerr)
		}
	}

	dataRoot := yamlDir
	if existingCfg.DataBaseDirectory != "" {
		dataRoot = filepath.Join(yamlDir, existingCfg.DataBaseDirectory)
	}

	fmt.Fprintln(out, "task-man --init will:")
	fmt.Fprintf(out, "  - reset statuses to default (todo / doing / done)\n")
	fmt.Fprintf(out, "  - clear all tasks, fields, and tags\n")
	fmt.Fprintf(out, "  - delete every task-<id> directory under: %s\n", dataRoot)
	fmt.Fprintf(out, "  - rewrite yaml: %s\n", yamlPath)
	fmt.Fprint(out, "Are you sure? (y/N): ")

	reader := bufio.NewReader(in)
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(out, "aborted.")
		return nil
	}

	removed, err := storage.RemoveAllTaskData(yamlDir, existingCfg.DataBaseDirectory)
	if err != nil {
		return err
	}

	repo := storage.NewYAMLRepository(yamlPath)
	if err := repo.Save(storage.LoadResult{
		Tasks:    nil,
		Statuses: task.DefaultStatuses(),
		Fields:   nil,
		Tags:     nil,
		Config:   existingCfg,
	}); err != nil {
		return err
	}

	fmt.Fprintf(out, "removed %d task directory(ies).\n", len(removed))
	fmt.Fprintf(out, "wrote: %s\n", yamlPath)
	return nil
}
