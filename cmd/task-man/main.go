package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/miyazi777/task-man/internal/cli"
	"github.com/miyazi777/task-man/internal/storage"
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
	if err := cli.EnsureFile(args); err != nil {
		return err
	}

	repo := storage.NewYAMLRepository(args.Path)
	lr, err := repo.Load()
	if err != nil {
		return err
	}

	yamlDir := filepath.Dir(args.Path)
	model := tui.NewModel(repo, lr.Tasks, lr.Statuses, lr.Fields, yamlDir, lr.Config)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
