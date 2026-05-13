package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireLock(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(yamlPath, []byte{}, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	f, err := acquireLock(yamlPath)
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	defer releaseLock(f)

	// ロックファイルが作成されていること
	lp := lockFilePath(yamlPath)
	if _, err := os.Stat(lp); err != nil {
		t.Errorf("lock file should exist: %v", err)
	}
}

func TestAcquireLockExclusive(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(yamlPath, []byte{}, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// 1 つ目のロックを取得
	f1, err := acquireLock(yamlPath)
	if err != nil {
		t.Fatalf("first acquireLock: %v", err)
	}
	defer releaseLock(f1)

	// 2 つ目のロック取得は ErrAlreadyLocked で失敗すること
	_, err = acquireLock(yamlPath)
	if err == nil {
		t.Fatal("second acquireLock should fail")
	}
	if !errors.Is(err, ErrAlreadyLocked) {
		t.Errorf("expected ErrAlreadyLocked, got: %v", err)
	}
}

func TestAcquireLockAfterRelease(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(yamlPath, []byte{}, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// ロックを取得して解放
	f1, err := acquireLock(yamlPath)
	if err != nil {
		t.Fatalf("first acquireLock: %v", err)
	}
	if err := releaseLock(f1); err != nil {
		t.Fatalf("releaseLock: %v", err)
	}

	// 解放後は再取得できること
	f2, err := acquireLock(yamlPath)
	if err != nil {
		t.Fatalf("second acquireLock after release: %v", err)
	}
	defer releaseLock(f2)
}

func TestReleaseLockNil(t *testing.T) {
	// nil を渡しても安全であること
	if err := releaseLock(nil); err != nil {
		t.Errorf("releaseLock(nil) should return nil, got: %v", err)
	}
}

func TestYAMLRepositoryLockClose(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(yamlPath, []byte{}, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	repo := NewYAMLRepository(yamlPath)
	if err := repo.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	// 別のリポジトリが同じパスをロックしようとすると失敗
	repo2 := NewYAMLRepository(yamlPath)
	if err := repo2.Lock(); err == nil {
		t.Fatal("second Lock should fail")
	} else if !errors.Is(err, ErrAlreadyLocked) {
		t.Errorf("expected ErrAlreadyLocked, got: %v", err)
	}

	// Close 後は再取得可能
	if err := repo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := repo2.Lock(); err != nil {
		t.Fatalf("Lock after Close: %v", err)
	}
	defer repo2.Close()
}

func TestYAMLRepositoryCloseWithoutLock(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "tasks.yaml")
	repo := NewYAMLRepository(yamlPath)

	// Lock を呼ばずに Close しても安全であること
	if err := repo.Close(); err != nil {
		t.Errorf("Close without Lock should return nil, got: %v", err)
	}
}
