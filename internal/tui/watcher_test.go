package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestWatcher は debounce を短縮した TaskDirWatcher を作る。
// テスト失敗時に確実に Close するため cleanup を登録する。
func newTestWatcher(t *testing.T) *TaskDirWatcher {
	t.Helper()
	w, err := NewTaskDirWatcher()
	if err != nil {
		t.Fatalf("NewTaskDirWatcher: %v", err)
	}
	w.mu.Lock()
	w.debounce = 20 * time.Millisecond
	w.mu.Unlock()
	t.Cleanup(func() { _ = w.Close() })
	return w
}

// waitForEvent は w.events から最大 timeout 待つ。受信できれば true、タイムアウトなら false。
func waitForEvent(t *testing.T, w *TaskDirWatcher, timeout time.Duration) bool {
	t.Helper()
	select {
	case <-w.events:
		return true
	case <-time.After(timeout):
		return false
	}
}

// expectNoEvent は timeout 期間中に w.events に何も来ないことを確認する。
func expectNoEvent(t *testing.T, w *TaskDirWatcher, timeout time.Duration) {
	t.Helper()
	select {
	case <-w.events:
		t.Fatalf("expected no event, but received one")
	case <-time.After(timeout):
	}
}

// drainEvents は debounce 直後のバースト残響をクリアする。
// テスト中の Switch / Mkdir などで生まれた副次的なイベントを取り除いて、
// 後続の検証を安定させる。
func drainEvents(w *TaskDirWatcher, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-w.events:
			// 受け取れたら次へ
		case <-time.After(10 * time.Millisecond):
			// 一定時間来なければ抜ける
			return
		}
	}
}

// TestSwitchAndCreate は Switch 後にファイル作成すると events が届くこと。
func TestSwitchAndCreate(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	if err := os.WriteFile(filepath.Join(dir, "foo.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after WriteFile, none received")
	}
}

// TestSwitchAndRemove は Switch 後にファイル削除でも events が届くこと。
func TestSwitchAndRemove(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "to-remove.md")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	if err := os.Remove(target); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after Remove, none received")
	}
}

// TestSwitchAndRename は Switch 後にファイル名変更で events が届くこと。
func TestSwitchAndRename(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "src.md")
	if err := os.WriteFile(src, nil, 0o644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	if err := os.Rename(src, filepath.Join(dir, "dst.md")); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after Rename, none received")
	}
}

// TestDynamicAddSubdir は Switch 後に作成されたサブディレクトリ配下の変更も
// 動的 Add で拾われること (Linux inotify が再帰非対応のための要件)。
func TestDynamicAddSubdir(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	// Mkdir 自体で 1 イベント発生する。debounce 経過待ち + drain。
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after Mkdir, none received")
	}
	drainEvents(w, 60*time.Millisecond)

	// 動的 Add が効いていれば、サブディレクトリ配下のファイル作成も検知される。
	if err := os.WriteFile(filepath.Join(sub, "x.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile in sub: %v", err)
	}
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after WriteFile in subdir, none received (dynamic Add broken?)")
	}
}

// TestSwitchAwayIgnoresOldDir は旧 Switch 対象での変更が events に来ないこと、
// 新 Switch 対象では届くことを検証する。
func TestSwitchAwayIgnoresOldDir(t *testing.T) {
	w := newTestWatcher(t)
	d1 := t.TempDir()
	d2 := t.TempDir()

	if err := w.Switch(1, d1); err != nil {
		t.Fatalf("Switch d1: %v", err)
	}
	if err := w.Switch(2, d2); err != nil {
		t.Fatalf("Switch d2: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	// d1 で変更 → 来ないことを期待
	if err := os.WriteFile(filepath.Join(d1, "ignored.md"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile d1: %v", err)
	}
	expectNoEvent(t, w, 200*time.Millisecond)

	// d2 で変更 → 来ることを期待
	if err := os.WriteFile(filepath.Join(d2, "watched.md"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile d2: %v", err)
	}
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected event after WriteFile in d2, none received")
	}
}

// TestDebounceCollapsesBurst は連続イベントが 1 件にまとめられること。
func TestDebounceCollapsesBurst(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	// debounce=20ms 以内に複数の WriteFile を撒く
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "burst-"+string(rune('a'+i))+".md")
		if err := os.WriteFile(name, nil, 0o644); err != nil {
			t.Fatalf("WriteFile %d: %v", i, err)
		}
	}

	// debounce 満了で 1 件だけ届くはず
	if !waitForEvent(t, w, 500*time.Millisecond) {
		t.Fatalf("expected one event after burst, none received")
	}
	// 追加で来ないこと (バーストが 1 件に collapse されている)
	expectNoEvent(t, w, 200*time.Millisecond)
}

// TestCloseStopsEvents は Close 後に変更があっても events が来ないこと、
// Wait Cmd が nil Msg を返してループが静かに止まること。
func TestCloseStopsEvents(t *testing.T) {
	w, err := NewTaskDirWatcher()
	if err != nil {
		t.Fatalf("NewTaskDirWatcher: %v", err)
	}
	w.mu.Lock()
	w.debounce = 20 * time.Millisecond
	w.mu.Unlock()

	dir := t.TempDir()
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// 二重 Close も安全
	if err := w.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	// Wait() が返す Cmd を実行すると nil Msg が返るはず
	cmd := w.Wait()
	if cmd == nil {
		t.Fatalf("Wait() should return a non-nil Cmd even after Close")
	}
	msg := cmd()
	if msg != nil {
		t.Fatalf("expected nil Msg after Close, got %T", msg)
	}
}

// TestNilSafe は (*TaskDirWatcher)(nil) でも全メソッドが panic しないこと。
func TestNilSafe(t *testing.T) {
	var w *TaskDirWatcher

	if err := w.Switch(1, "/tmp/nonexistent"); err != nil {
		t.Errorf("nil Switch: unexpected err %v", err)
	}
	if cmd := w.Wait(); cmd != nil {
		t.Errorf("nil Wait: expected nil Cmd, got non-nil")
	}
	if err := w.Close(); err != nil {
		t.Errorf("nil Close: unexpected err %v", err)
	}
}

// TestSwitchEmptyClearsWatch は Switch(0, "") で旧 watch を解放すること。
// 解放後に旧ディレクトリで変更しても events が来ないことで検証する。
func TestSwitchEmptyClearsWatch(t *testing.T) {
	w := newTestWatcher(t)
	dir := t.TempDir()
	if err := w.Switch(1, dir); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	drainEvents(w, 60*time.Millisecond)

	if err := w.Switch(0, ""); err != nil {
		t.Fatalf("Switch(0, \"\"): %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "ignored.md"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	expectNoEvent(t, w, 200*time.Millisecond)
}
