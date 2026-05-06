package tui

import (
	"path/filepath"
	"strings"

	"github.com/miyazi777/task-man/internal/storage"
)

// resolveDefaultApp は fileName の拡張子に対応する file_opener.default_app の Application を返す。
// 該当拡張子の opener が無い、または default_app 未指定 (0)、または存在しない id の場合は ok=false。
// ok=false の場合、呼び出し側は $EDITOR フォールバックを使う。
func resolveDefaultApp(fileName string, apps []storage.Application, openers []storage.FileOpener) (storage.Application, bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		return storage.Application{}, false
	}
	defaultID := 0
	for _, op := range openers {
		if op.Extension == ext {
			defaultID = op.DefaultApp
			break
		}
	}
	if defaultID == 0 {
		return storage.Application{}, false
	}
	for _, a := range apps {
		if a.ID == defaultID {
			return a, true
		}
	}
	return storage.Application{}, false
}

// resolveFileOpenerCandidates は fileName の拡張子に対応する application 候補を返す。
// 見つからない場合は空スライス (= 呼び出し側で $EDITOR フォールバック)。
//
//   - extension は大文字小文字無視 (foo.MD と foo.md は同一拡張子)
//   - file_opener に該当 extension が無ければ candidates は空
//   - candidates の順序は file_opener.applications の指定順。存在しない id は無視。
func resolveFileOpenerCandidates(fileName string, apps []storage.Application, openers []storage.FileOpener) []storage.Application {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		return nil
	}
	var ids []int
	for _, op := range openers {
		if op.Extension == ext {
			ids = op.ApplicationIDs
			break
		}
	}
	if len(ids) == 0 {
		return nil
	}
	byID := make(map[int]storage.Application, len(apps))
	for _, a := range apps {
		byID[a.ID] = a
	}
	out := make([]storage.Application, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out
}
