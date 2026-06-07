package tui

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// defaultWatcherDebounce は外部からの touch/mv/rm 等が連続した場合に
// withFilesRefreshed をまとめて 1 回に抑える窓 (debounce)。
// vim の .swp などの atomic save パターンでも UI のチラつきが出ないよう
// 100ms に設定している。テスト時は構造体を直接書き換えて短縮する。
const defaultWatcherDebounce = 100 * time.Millisecond

// fsnotifyMsg は taskDirWatcher が debounce 済みイベントを Update に届けるシグナル。
// 中身は意図的に空: Update 側は受信した瞬間に withFilesRefreshed() を呼ぶだけでよい。
type fsnotifyMsg struct{}

// TaskDirWatcher は fsnotify をラップして「現在カーソル位置のタスクディレクトリ配下」
// のファイル変更を debounce 済みで bubbletea Update に届ける。
//
// 想定ユースケース:
//   - main.go で NewTaskDirWatcher() し Model.WithWatcher() で注入
//   - Model.Init() で初期 Switch + Wait() を発行
//   - withFilesRefreshed() の中で filesTaskID 変化時に Switch() を呼ぶ
//   - main.go の defer で Close()
//
// nil レシーバでも全メソッドが安全に呼べる (no-op)。Model 側で if 文を増やさないため。
type TaskDirWatcher struct {
	fsw      *fsnotify.Watcher
	events   chan struct{} // debounce 済みイベントを Wait Cmd へ渡すチャネル
	done     chan struct{} // Close 用シグナル
	debounce time.Duration

	mu      sync.Mutex
	watched map[string]struct{} // Add 済みディレクトリ絶対パス set (重複 Add 防止)
	closed  bool                // Close 冪等化
}

// NewTaskDirWatcher は fsnotify.Watcher を生成し、内部 goroutine を起動する。
// 失敗時 (権限不足 / inotify 上限超過 等) は (nil, err) を返す。呼び出し側は
// nil watcher を Model に注入してそのまま起動する (R キー fallback)。
func NewTaskDirWatcher() (*TaskDirWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &TaskDirWatcher{
		fsw:      fsw,
		events:   make(chan struct{}, 1),
		done:     make(chan struct{}),
		debounce: defaultWatcherDebounce,
		watched:  map[string]struct{}{},
	}
	go w.loop()
	return w, nil
}

// Switch は現在の watch 対象を taskDir 配下に切り替える。
//   - taskID == 0 || taskDir == "" の場合は既存 watch を全 Remove して終了 (status 行カーソル時など)
//   - そうでなければ filepath.Walk で全ディレクトリを Add する (Linux inotify が再帰非対応のため)
//
// 競合削除等で部分的に Add できなくても致命ではない (R キーで補える)。
func (w *TaskDirWatcher) Switch(taskID int, taskDir string) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}

	// 旧 watched を全 Remove。存在しないパスの Remove はエラーになり得るが無視。
	for p := range w.watched {
		_ = w.fsw.Remove(p)
	}
	w.watched = map[string]struct{}{}

	if taskID == 0 || taskDir == "" {
		return nil
	}

	return filepath.Walk(taskDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return nil // 他のエラーはスキップして継続
		}
		if !info.IsDir() {
			return nil
		}
		if _, ok := w.watched[path]; ok {
			return nil
		}
		if addErr := w.fsw.Add(path); addErr == nil {
			w.watched[path] = struct{}{}
		}
		return nil
	})
}

// Wait は debounce 済みファイル変更イベントを 1 件待つ tea.Cmd を返す。
// Update で fsnotifyMsg を処理したあと再度 Wait() を return することでループになる。
//   - events から受信したら fsnotifyMsg{} を返す
//   - done が閉じたら nil Msg を返す (bubbletea が無視するので静かに止まる)
//   - レシーバが nil なら nil Cmd を返す (Init で watcher 未注入の場合)
func (w *TaskDirWatcher) Wait() tea.Cmd {
	if w == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case <-w.events:
			return fsnotifyMsg{}
		case <-w.done:
			return nil
		}
	}
}

// Close は内部 goroutine を停止し fsnotify.Watcher を閉じる。冪等。
func (w *TaskDirWatcher) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()
	close(w.done)
	return w.fsw.Close()
}

// loop は内部 goroutine。raw イベントを受け取り、動的 Add/Remove と debounce を行う。
func (w *TaskDirWatcher) loop() {
	var timer *time.Timer
	var timerC <-chan time.Time
	// pending=true のときは debounce 中。タイマー満了で events に送る。
	pending := false

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleDynamic(ev)

			if !pending {
				pending = true
				if timer == nil {
					timer = time.NewTimer(w.currentDebounce())
				} else {
					timer.Reset(w.currentDebounce())
				}
				timerC = timer.C
			} else {
				// ハンマリング吸収: 既存タイマーを再 Reset
				if !timer.Stop() {
					// 既に発火している可能性がある。チャネルを排水してから Reset
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(w.currentDebounce())
				timerC = timer.C
			}

		case <-timerC:
			// debounce 満了。非ブロッキングで 1 件送る。
			// events に既に 1 件溜まっていれば送信は省略 (容量 1 のチャネル + select default)。
			select {
			case w.events <- struct{}{}:
			default:
			}
			pending = false
			timerC = nil

		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// 本リリースではエラーログは出さない (Issue #36 議論で「静かに諦める」合意)
		}
	}
}

// handleDynamic は Create/Remove/Rename イベントを見て watched set を維持する。
// Linux inotify は再帰 watch しないため、サブディレクトリ作成を見たら自分で Add する。
func (w *TaskDirWatcher) handleDynamic(ev fsnotify.Event) {
	if ev.Op&fsnotify.Create != 0 {
		info, err := os.Stat(ev.Name)
		if err == nil && info.IsDir() {
			w.mu.Lock()
			defer w.mu.Unlock()
			if w.closed {
				return
			}
			// 新規ディレクトリ + その配下を全部 Add (再帰的に作られたケースのため Walk)
			_ = filepath.Walk(ev.Name, func(path string, info os.FileInfo, werr error) error {
				if werr != nil {
					if os.IsNotExist(werr) {
						return filepath.SkipDir
					}
					return nil
				}
				if !info.IsDir() {
					return nil
				}
				if _, ok := w.watched[path]; ok {
					return nil
				}
				if addErr := w.fsw.Add(path); addErr == nil {
					w.watched[path] = struct{}{}
				}
				return nil
			})
		}
		return
	}

	if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.mu.Lock()
		defer w.mu.Unlock()
		if _, ok := w.watched[ev.Name]; ok {
			_ = w.fsw.Remove(ev.Name)
			delete(w.watched, ev.Name)
		}
	}
}

// currentDebounce はテスト時に外から debounce を書き換えても安全に拾えるよう
// ロック越しに値を返す。ロックは短時間なので loop の性能には影響しない。
func (w *TaskDirWatcher) currentDebounce() time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.debounce
}
