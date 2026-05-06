package tui

import (
	"testing"

	"github.com/miyazi777/task-man/internal/storage"
)

func TestResolveFileOpenerCandidatesMatch(t *testing.T) {
	apps := []storage.Application{
		{ID: 1, Name: "editor", Run: "$EDITOR"},
		{ID: 2, Name: "less", Run: "less"},
	}
	openers := []storage.FileOpener{
		{Extension: "md", ApplicationIDs: []int{1, 2}},
	}
	got := resolveFileOpenerCandidates("memo.md", apps, openers)
	if len(got) != 2 {
		t.Fatalf("len: got %d want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("order: got %+v want [1,2]", got)
	}
}

func TestResolveFileOpenerCandidatesCaseInsensitive(t *testing.T) {
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}}}
	got := resolveFileOpenerCandidates("README.MD", apps, openers)
	if len(got) != 1 {
		t.Fatalf("expected 1 match for uppercase extension, got %d", len(got))
	}
}

func TestResolveFileOpenerCandidatesNoOpener(t *testing.T) {
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}}}
	got := resolveFileOpenerCandidates("foo.txt", apps, openers)
	if len(got) != 0 {
		t.Errorf("expected fallback (empty), got %+v", got)
	}
}

func TestResolveFileOpenerCandidatesNoExtension(t *testing.T) {
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}}}
	got := resolveFileOpenerCandidates("Makefile", apps, openers)
	if len(got) != 0 {
		t.Errorf("expected empty for no extension, got %+v", got)
	}
}

func TestResolveFileOpenerCandidatesUnknownAppID(t *testing.T) {
	// loadFileOpeners で弾く想定だが、念のためここでも未知 id は無視する。
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1, 99}}}
	got := resolveFileOpenerCandidates("a.md", apps, openers)
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("expected only id=1, got %+v", got)
	}
}

func TestResolveDefaultAppMatch(t *testing.T) {
	apps := []storage.Application{
		{ID: 1, Name: "editor", Run: "$EDITOR"},
		{ID: 2, Name: "less", Run: "less"},
	}
	openers := []storage.FileOpener{
		{Extension: "md", ApplicationIDs: []int{1, 2}, DefaultApp: 2},
	}
	got, ok := resolveDefaultApp("a.md", apps, openers)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.ID != 2 {
		t.Errorf("got id=%d want 2", got.ID)
	}
}

func TestResolveDefaultAppMissingDefault(t *testing.T) {
	// default_app 未指定 (=0) は ok=false ($EDITOR フォールバック)
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}}}
	if _, ok := resolveDefaultApp("a.md", apps, openers); ok {
		t.Error("expected ok=false when default_app is 0")
	}
}

func TestResolveDefaultAppNoOpener(t *testing.T) {
	// 対象拡張子の opener が無い → ok=false
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}, DefaultApp: 1}}
	if _, ok := resolveDefaultApp("foo.txt", apps, openers); ok {
		t.Error("expected ok=false for unmatched extension")
	}
}

func TestResolveDefaultAppCaseInsensitive(t *testing.T) {
	apps := []storage.Application{{ID: 1, Name: "ed", Run: "vi"}}
	openers := []storage.FileOpener{{Extension: "md", ApplicationIDs: []int{1}, DefaultApp: 1}}
	got, ok := resolveDefaultApp("README.MD", apps, openers)
	if !ok || got.ID != 1 {
		t.Errorf("expected match for uppercase ext, got ok=%v app=%+v", ok, got)
	}
}
