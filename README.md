[English](./README.md) | [日本語](./README.ja.md)

# task-man

A terminal-based task management TUI application. All data is stored in a single `tasks.yaml`, while task contents and attached files live in an adjacent directory.

A Japanese-friendly UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) / [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Single-file persistence**: Tasks, statuses, custom fields, tags, and layout are all written to a single `tasks.yaml`
- **Custom statuses**: In addition to the default `todo / doing / done`, you can freely define statuses and colors in yaml
- **Subtasks**: Up to 5 levels of nesting
- **Tags**: Up to 5 tags per task; manage tags from the settings screen
- **Custom fields**: Add `text` / `date` / `URL` typed fields to a task
- **Trash**: Temporarily store deleted tasks, restore them, or permanently delete
- **File preview**: Show task-attached files (md / txt / csv) in the right pane
- **File opener**: Configure per-extension launch applications in yaml; launch directly from a file
- **Layout adjustment**: Interactively resize each pane on the task list screen and persist it
- **Multiple workspaces**: Specify a different `tasks.yaml` at startup with the `-t` option

## Screenshots

Mockup SVGs (such as `01-list-focused.svg`) are included in `docs/mockups/`.

## Requirements

- Go 1.26 or later (see `go.mod`)
- Tested environments: Linux

## Build & Install

```bash
git clone https://github.com/<your-account>/task-man.git
cd task-man

# Local build (produces ./task-man)
make build

# Install to $GOPATH/bin
make install
```

Main Make targets:

| Target | Action |
|---|---|
| `make build` | Build the `./task-man` binary |
| `make run` | Build and run |
| `make test` | Run tests for all packages |
| `make vet` | `go vet ./...` |
| `make fmt` | `go fmt ./...` |
| `make tidy` | `go mod tidy` |
| `make clean` | Remove the binary |

## Launch

Reads `tasks.yaml` from the current directory. If it does not exist, an empty file is created automatically.

```bash
./task-man
```

### Launch options

| Flag | Description |
|---|---|
| `-t`, `--tasks <path>` | Specify any `tasks.yaml` (supports `~/...` expansion) |
| `-i`, `--init` | Initialize the yaml with only the default 3 statuses (todo/doing/done) and remove all task data directories (requires y/N confirmation) |

#### Example: Reference a shared tasks.yaml via an alias

```bash
# Use ~/private/tasks.yaml from any working directory
alias tm='task-man -t ~/private/tasks.yaml'
```

#### Example: Reset state

```bash
./task-man -i        # Initialize (with confirmation)
./task-man -t ~/private/tasks.yaml -i
```

## Key bindings

### Task list screen

| Key | Action |
|---|---|
| `k` / `↑` , `j` / `↓` | Move cursor up / down |
| `l` / `→` | Expand status / task |
| `h` / `←` | Collapse |
| `enter` | Open task detail |
| `a` | New task (on a status row) / subtask (on a task row) |
| `d` | Move task to trash (in trash view: permanently delete) |
| `m` | Start / confirm move mode |
| `o` | Operation mode (`t`=title, `s`=status, `g`=tag, `f`=files) |
| `;` | Prefix mode |
| `q` | Quit |

### Prefix mode (after `;`)

| Key | Action |
|---|---|
| `t` | Toggle trash view |
| `s` | Open settings screen |
| `l` | Enter layout adjustment mode |
| `esc` | Cancel |

### Task detail screen

| Key | Action |
|---|---|
| `k` / `j` | Move row cursor up / down |
| `enter` | Open the edit popup for each row / on the Files row, launch the file opener (default_app) |
| `o` | URL field: open in browser / Files row: extension-based application selection modal |
| `a` | Create a new file in the Files section |
| `r` | Rename a file in the Files section |
| `d` | Delete a file in the Files section |
| `;` | Prefix mode |
| `esc` | Return to the list |

### Layout adjustment mode (`;` → `l`)

The vertical operation target depends on the focus when entering the mode (task list / task detail / file list).

| Key | Action |
|---|---|
| `h` / `l` | Shrink / expand task list width |
| `j` / `k` (when detail / file list focused) | Expand / shrink height |
| `enter` | Confirm (save to yaml) |
| `esc` | Revert to pre-entry values and exit |

### Settings screen (`;` → `s`)

Switch between `general` / `status` / `field` / `application` / `file_opener` from the left menu.

- **general**: Check yaml path, edit `data_base_directory`
- **status**: Add / rename / change color / delete / reorder statuses
- **field**: Add / edit / reorder / delete custom fields
- **application**: Register / edit applications used by the file opener
- **file_opener**: Edit the mapping of extensions to applications and `default_app`

## Structure of tasks.yaml

`task-man` reads and writes yaml with the following structure. Each section is optional (everything except `statuses` may be empty).

```yaml
applications:
  - application:
      id: 1
      name: editor
      run: $EDITOR        # env var or command on PATH
  - application:
      id: 2
      name: md-viewer
      run: md-viewer

file_opener:
  - opener:
      extension: "md"
      applications: [1, 2]
      default_app: 1     # app launched when enter is pressed (defaults to $EDITOR)

data_base_directory: ./tasks_data   # base for task attachment dirs (defaults to the yaml's dir)

layout:
  main:
    task_list:
      width: 0.6      # ratio 0.0–1.0 (task list's share of screen width)
    task_detail:
      height: 0.4
    file_list:
      height: 0.3
    file_preview:
      height: 0.3

statuses:
  - status:
      id: 1
      sequence: 1
      label: todo
      color: "#6c7086"
  - status:
      id: 2
      sequence: 2
      label: doing
      color: "#fab387"
  - status:
      id: 3
      sequence: 3
      label: done
      color: "#a6e3a1"

fields:
  - field:
      id: 1
      type: text
      name: due_date
      position: 1

tags:
  - tag:
      id: 1
      name: urgent
      color: "#f38ba8"

tasks:
  - task:
      id: 1
      title: Task 1
      status_id: 1
      position: 1
      tags: [1]
      fields:
        - field:
            id: 1
            field_id: 1
            value: "2026-05-01"
```

### data_base_directory and task attachments

A `task-<id>/` directory is generated per task, and attached files such as `memo.md` are placed inside it.
The location follows the `data_base_directory` setting (defaults to the same directory as the yaml).

Example: When `data_base_directory: ./tasks_data`, attachments for the task with ID=1 live under `./tasks_data/task-1/`.

## Directory structure

```
.
├── cmd/task-man         # Entry point (main.go)
├── internal/cli         # CLI argument parsing
├── internal/storage     # tasks.yaml read/write / attachment operations
├── internal/task        # Domain (Task / Status / Field / Tag)
├── internal/tui         # Bubble Tea Model/View/Update
├── docs/mockups         # Screen mockups (SVG)
├── Makefile
├── go.mod
└── README.md
```

## Contributing

This repo ships a git pre-commit hook under `.githooks/` that warns when only one of `README.md` / `README.ja.md` is staged (the warning does not block the commit). Enable it once per clone:

```bash
git config core.hooksPath .githooks
```

## Acknowledgments

- TUI foundation: [Bubble Tea](https://github.com/charmbracelet/bubbletea) / [Bubbles](https://github.com/charmbracelet/bubbles) / [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Color theme: based on [Catppuccin](https://catppuccin.com/)
