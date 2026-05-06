package tui

import (
	"math"

	"github.com/miyazi777/task-man/internal/storage"
)

// レイアウト比率の制約。範囲外の値は clampLayoutRatios で範囲内に丸める。
const (
	layoutTaskListMin   = 0.1
	layoutTaskListMax   = 0.9
	layoutVerticalMin   = 0.1
	layoutVerticalMax   = 0.8
	layoutVerticalCount = 3 // task_detail / file_list / file_preview
)

// デフォルト比率。LayoutConfig のいずれかが未設定 (nil) の時にフォールバックする。
//   - 横: 左 2/3, 右 1/3 (= twoPaneWidths と一致)
//   - 縦: 各 1/3
const (
	defaultTaskListRatio    = 2.0 / 3.0
	defaultTaskDetailRatio  = 1.0 / 3.0
	defaultFileListRatio    = 1.0 / 3.0
	defaultFilePreviewRatio = 1.0 / 3.0
)

// レイアウトモードのフォーカス位置。task list / task detail / file list の 3 値。
type layoutFocus int

const (
	layoutFocusTaskList layoutFocus = iota
	layoutFocusTaskDetail
	layoutFocusFileList
)

// ensureLayoutRatios は LayoutConfig のいずれかが nil の場合にデフォルト比率で埋める。
// 戻り値は 4 値とも非 nil の LayoutConfig。元の非 nil 値はそのまま保持される。
func ensureLayoutRatios(in storage.LayoutConfig) storage.LayoutConfig {
	out := in
	if out.TaskListWidth == nil {
		v := defaultTaskListRatio
		out.TaskListWidth = &v
	}
	if out.TaskDetailHeight == nil {
		v := defaultTaskDetailRatio
		out.TaskDetailHeight = &v
	}
	if out.FileListHeight == nil {
		v := defaultFileListRatio
		out.FileListHeight = &v
	}
	if out.FilePreviewHeight == nil {
		v := defaultFilePreviewRatio
		out.FilePreviewHeight = &v
	}
	return out
}

// isLayoutComplete は 4 値とも非 nil の場合のみ true。View 側がフォールバックを判定するのに使う。
func isLayoutComplete(in storage.LayoutConfig) bool {
	return in.TaskListWidth != nil && in.TaskDetailHeight != nil &&
		in.FileListHeight != nil && in.FilePreviewHeight != nil
}

// clampLayoutRatios は各値を範囲内に丸めるだけ (正規化はしない)。
// 操作中 (h/l/j/k 押下時) に 1 セル増減した直後の clamp 用。
// 入力は 4 値とも非 nil である前提。
func clampLayoutRatios(in storage.LayoutConfig) storage.LayoutConfig {
	out := in
	tl := clampFloat(*out.TaskListWidth, layoutTaskListMin, layoutTaskListMax)
	td := clampFloat(*out.TaskDetailHeight, layoutVerticalMin, layoutVerticalMax)
	fl := clampFloat(*out.FileListHeight, layoutVerticalMin, layoutVerticalMax)
	fp := clampFloat(*out.FilePreviewHeight, layoutVerticalMin, layoutVerticalMax)
	out.TaskListWidth = &tl
	out.TaskDetailHeight = &td
	out.FileListHeight = &fl
	out.FilePreviewHeight = &fp
	return out
}

// normalizeLayoutRatios は縦 3 値の合計が 1.0 になるよう正規化する。
// clamp 後に和が崩れたぶんを比例配分で再スケールする。enter 確定時 / ロード時に使う。
// 入力は 4 値とも非 nil である前提。
func normalizeLayoutRatios(in storage.LayoutConfig) storage.LayoutConfig {
	out := clampLayoutRatios(in)
	td := *out.TaskDetailHeight
	fl := *out.FileListHeight
	fp := *out.FilePreviewHeight
	sum := td + fl + fp
	if sum <= 0 {
		// 異常値: デフォルトに戻す
		td, fl, fp = defaultTaskDetailRatio, defaultFileListRatio, defaultFilePreviewRatio
	} else {
		td /= sum
		fl /= sum
		fp /= sum
	}
	// 正規化後に範囲外になる可能性があるので再 clamp + 残差を preview に押しつける
	td = clampFloat(td, layoutVerticalMin, layoutVerticalMax)
	fl = clampFloat(fl, layoutVerticalMin, layoutVerticalMax)
	fp = 1.0 - td - fl
	if fp < layoutVerticalMin {
		fp = layoutVerticalMin
		// 不足分を file_list → task_detail の順に減らす
		over := td + fl + fp - 1.0
		if over > 0 {
			if fl-over >= layoutVerticalMin {
				fl -= over
			} else {
				fl = layoutVerticalMin
				td = 1.0 - fl - fp
			}
		}
	} else if fp > layoutVerticalMax {
		fp = layoutVerticalMax
	}
	out.TaskDetailHeight = &td
	out.FileListHeight = &fl
	out.FilePreviewHeight = &fp
	return out
}

// computeLayoutDelta は「画面 1 セルぶん」を比率に変換した値を返す。
// 0 除算を避けるため最低 1 セル相当として扱う。
func computeLayoutDelta(screenW, bodyH int) (horiz, vert float64) {
	availW := screenW - 1 // 区切り線 1 本ぶん
	if availW < 1 {
		availW = 1
	}
	if bodyH < 1 {
		bodyH = 1
	}
	return 1.0 / float64(availW), 1.0 / float64(bodyH)
}

// applyLayoutToScreen は LayoutConfig を実セル数に変換する。
//   - leftW + (区切り線 1) + rightW = screenW
//   - detailH + fileAreaH + previewAreaH = bodyH
//
// 入力は 4 値とも非 nil で、縦 3 値の和が 1.0 である前提 (= normalizeLayoutRatios 経由)。
// fileAreaH / previewAreaH は header / divider を含んだペイン全体の高さ。
func applyLayoutToScreen(in storage.LayoutConfig, screenW, bodyH int) (leftW, rightW, detailH, fileAreaH, previewAreaH int) {
	availW := screenW - 1
	if availW < 2 {
		availW = 2
	}

	leftW = int(math.Round(*in.TaskListWidth * float64(availW)))
	if leftW < 1 {
		leftW = 1
	}
	if leftW > availW-1 {
		leftW = availW - 1
	}
	rightW = availW - leftW

	if bodyH < 3 {
		bodyH = 3
	}
	detailH = int(math.Round(*in.TaskDetailHeight * float64(bodyH)))
	fileAreaH = int(math.Round(*in.FileListHeight * float64(bodyH)))
	previewAreaH = bodyH - detailH - fileAreaH

	// 最低高さ確保: detail >= 3, fileArea >= 3, previewArea >= 2
	if detailH < 3 {
		detailH = 3
	}
	if fileAreaH < 3 {
		fileAreaH = 3
	}
	if previewAreaH < 2 {
		previewAreaH = 2
	}

	// 合計が bodyH を超えた場合は preview → file → detail の順に縮める。
	if total := detailH + fileAreaH + previewAreaH; total > bodyH {
		over := total - bodyH
		shrink := func(target *int, floor int) {
			if over <= 0 {
				return
			}
			room := *target - floor
			if room <= 0 {
				return
			}
			if room >= over {
				*target -= over
				over = 0
			} else {
				*target = floor
				over -= room
			}
		}
		shrink(&previewAreaH, 2)
		shrink(&fileAreaH, 3)
		shrink(&detailH, 3)
	}

	// 合計が bodyH に満たない場合は detail に足す (残りの吸収先)。
	if total := detailH + fileAreaH + previewAreaH; total < bodyH {
		detailH += bodyH - total
	}
	return
}

// clampFloat は v を [lo, hi] に丸める。
func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// adjustLayoutVertical は「フォーカスペインの高さを delta 分増減し、preview から相殺する」
// レイアウトモード操作の中核ロジック。preview が下限に達した場合は file_list から相殺する。
// focus が task_list の場合は何もしない (横操作は別)。clamp は呼び出し側で行う。
func adjustLayoutVertical(in storage.LayoutConfig, focus layoutFocus, delta float64) storage.LayoutConfig {
	if focus == layoutFocusTaskList {
		return in
	}
	out := in
	td := *out.TaskDetailHeight
	fl := *out.FileListHeight
	fp := *out.FilePreviewHeight

	switch focus {
	case layoutFocusTaskDetail:
		td += delta
		// 相殺: preview から先に減らす。下限に達したら file_list から。
		fp -= delta
		if fp < layoutVerticalMin {
			short := layoutVerticalMin - fp
			fp = layoutVerticalMin
			fl -= short
		}
	case layoutFocusFileList:
		fl += delta
		fp -= delta
		if fp < layoutVerticalMin {
			short := layoutVerticalMin - fp
			fp = layoutVerticalMin
			td -= short
		}
	}
	out.TaskDetailHeight = &td
	out.FileListHeight = &fl
	out.FilePreviewHeight = &fp
	return out
}

// adjustLayoutHorizontal は task_list の比率を delta 分増減する。
// clamp は呼び出し側。
func adjustLayoutHorizontal(in storage.LayoutConfig, delta float64) storage.LayoutConfig {
	out := in
	v := *out.TaskListWidth + delta
	out.TaskListWidth = &v
	return out
}
