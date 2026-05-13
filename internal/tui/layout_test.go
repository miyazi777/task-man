package tui

import (
	"math"
	"testing"

	"github.com/miyazi777/task-man/internal/storage"
)

func fp(v float64) *float64 { return &v }

func TestEnsureLayoutRatiosFillsNil(t *testing.T) {
	out := ensureLayoutRatios(storage.LayoutConfig{})
	if !isLayoutComplete(out) {
		t.Fatal("expected complete config after ensure")
	}
	if *out.TaskListWidth != defaultTaskListRatio {
		t.Errorf("task_list: got %v want %v", *out.TaskListWidth, defaultTaskListRatio)
	}
	sum := *out.TaskDetailHeight + *out.FileListHeight + *out.FilePreviewHeight
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("vertical sum: got %v want 1.0", sum)
	}
}

func TestEnsureLayoutRatiosPreservesExisting(t *testing.T) {
	in := storage.LayoutConfig{TaskListWidth: fp(0.55)}
	out := ensureLayoutRatios(in)
	if *out.TaskListWidth != 0.55 {
		t.Errorf("expected to preserve 0.55, got %v", *out.TaskListWidth)
	}
}

func TestIsLayoutComplete(t *testing.T) {
	if isLayoutComplete(storage.LayoutConfig{}) {
		t.Error("empty should be incomplete")
	}
	if isLayoutComplete(storage.LayoutConfig{TaskListWidth: fp(0.5)}) {
		t.Error("partial should be incomplete")
	}
	full := ensureLayoutRatios(storage.LayoutConfig{})
	if !isLayoutComplete(full) {
		t.Error("filled should be complete")
	}
}

func TestClampLayoutRatios(t *testing.T) {
	in := storage.LayoutConfig{
		TaskListWidth:     fp(0.05),
		TaskDetailHeight:  fp(0.95),
		FileListHeight:    fp(0.0),
		FilePreviewHeight: fp(1.5),
	}
	out := clampLayoutRatios(in)
	if *out.TaskListWidth != layoutTaskListMin {
		t.Errorf("task_list lo: got %v want %v", *out.TaskListWidth, layoutTaskListMin)
	}
	if *out.TaskDetailHeight != layoutVerticalMax {
		t.Errorf("detail hi: got %v want %v", *out.TaskDetailHeight, layoutVerticalMax)
	}
	if *out.FileListHeight != layoutVerticalMin {
		t.Errorf("file lo: got %v want %v", *out.FileListHeight, layoutVerticalMin)
	}
	if *out.FilePreviewHeight != layoutVerticalMax {
		t.Errorf("preview hi: got %v want %v", *out.FilePreviewHeight, layoutVerticalMax)
	}
}

func TestNormalizeLayoutRatiosSumsTo1(t *testing.T) {
	in := storage.LayoutConfig{
		TaskListWidth:     fp(0.5),
		TaskDetailHeight:  fp(0.4),
		FileListHeight:    fp(0.4),
		FilePreviewHeight: fp(0.4),
	}
	out := normalizeLayoutRatios(in)
	sum := *out.TaskDetailHeight + *out.FileListHeight + *out.FilePreviewHeight
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("sum after normalize: got %v want 1.0", sum)
	}
}

func TestNormalizeLayoutRatiosPreservesProportions(t *testing.T) {
	// 元の比 2:1:1 を保つ
	in := storage.LayoutConfig{
		TaskListWidth:     fp(0.5),
		TaskDetailHeight:  fp(0.5),
		FileListHeight:    fp(0.25),
		FilePreviewHeight: fp(0.25),
	}
	out := normalizeLayoutRatios(in)
	td := *out.TaskDetailHeight
	fl := *out.FileListHeight
	fp := *out.FilePreviewHeight
	if math.Abs(td-0.5) > 0.05 {
		t.Errorf("td proportion drift: %v", td)
	}
	if math.Abs(fl-fp) > 1e-9 {
		t.Errorf("fl=%v should equal fp=%v", fl, fp)
	}
}

func TestComputeLayoutDelta(t *testing.T) {
	h, v := computeLayoutDelta(101, 30) // availW=100
	if math.Abs(h-1.0/100) > 1e-9 {
		t.Errorf("horiz: got %v want 0.01", h)
	}
	if math.Abs(v-1.0/30) > 1e-9 {
		t.Errorf("vert: got %v", v)
	}
	// 0 サイズでもパニックしない
	h, v = computeLayoutDelta(0, 0)
	if h <= 0 || v <= 0 {
		t.Errorf("must be positive, got %v / %v", h, v)
	}
}

func TestApplyLayoutToScreenSumsToBody(t *testing.T) {
	in := ensureLayoutRatios(storage.LayoutConfig{})
	leftW, rightW, detailH, fileAreaH, previewAreaH := applyLayoutToScreen(in, 121, 30)
	if leftW+rightW != 120 { // availW = 121 - 1
		t.Errorf("widths sum: got %d want 120", leftW+rightW)
	}
	if detailH+fileAreaH+previewAreaH != 30 {
		t.Errorf("heights sum: got %d want 30", detailH+fileAreaH+previewAreaH)
	}
}

func TestApplyLayoutToScreenSmallTerm(t *testing.T) {
	in := ensureLayoutRatios(storage.LayoutConfig{})
	// 極小サイズでもパニックせず最低値を維持する。
	leftW, rightW, detailH, fileAreaH, previewAreaH := applyLayoutToScreen(in, 10, 5)
	if leftW < 1 || rightW < 1 {
		t.Errorf("widths < 1: %d / %d", leftW, rightW)
	}
	if detailH < 3 || fileAreaH < 3 || previewAreaH < 2 {
		t.Errorf("min heights violated: %d/%d/%d", detailH, fileAreaH, previewAreaH)
	}
}

func TestAdjustLayoutVerticalDetailGrowsFromPreview(t *testing.T) {
	in := ensureLayoutRatios(storage.LayoutConfig{})
	delta := 0.1
	out := adjustLayoutVertical(in, layoutFocusTaskDetail, delta)
	if math.Abs(*out.TaskDetailHeight-(*in.TaskDetailHeight+delta)) > 1e-9 {
		t.Errorf("detail not increased: %v -> %v", *in.TaskDetailHeight, *out.TaskDetailHeight)
	}
	if math.Abs(*out.FilePreviewHeight-(*in.FilePreviewHeight-delta)) > 1e-9 {
		t.Errorf("preview not decreased: %v -> %v", *in.FilePreviewHeight, *out.FilePreviewHeight)
	}
	if *out.FileListHeight != *in.FileListHeight {
		t.Errorf("file_list should be untouched: %v -> %v", *in.FileListHeight, *out.FileListHeight)
	}
}

func TestAdjustLayoutVerticalDetailGrowsBeyondPreview(t *testing.T) {
	// preview を下限に達するまで使い切ったら file_list から相殺
	in := storage.LayoutConfig{
		TaskListWidth:     fp(0.5),
		TaskDetailHeight:  fp(0.5),
		FileListHeight:    fp(0.4),
		FilePreviewHeight: fp(layoutVerticalMin), // 0.1
	}
	delta := 0.1 // preview は下限なので全量 file から
	out := adjustLayoutVertical(in, layoutFocusTaskDetail, delta)
	if *out.FilePreviewHeight != layoutVerticalMin {
		t.Errorf("preview should stay at min: %v", *out.FilePreviewHeight)
	}
	if math.Abs(*out.FileListHeight-(*in.FileListHeight-delta)) > 1e-9 {
		t.Errorf("file_list should absorb all: %v -> %v", *in.FileListHeight, *out.FileListHeight)
	}
}

func TestAdjustLayoutVerticalNoOpForTaskList(t *testing.T) {
	in := ensureLayoutRatios(storage.LayoutConfig{})
	out := adjustLayoutVertical(in, layoutFocusTaskList, 0.1)
	if *out.TaskDetailHeight != *in.TaskDetailHeight ||
		*out.FileListHeight != *in.FileListHeight ||
		*out.FilePreviewHeight != *in.FilePreviewHeight {
		t.Error("task_list focus should not change vertical ratios")
	}
}

func TestAdjustLayoutHorizontal(t *testing.T) {
	in := ensureLayoutRatios(storage.LayoutConfig{})
	out := adjustLayoutHorizontal(in, 0.1)
	if math.Abs(*out.TaskListWidth-(*in.TaskListWidth+0.1)) > 1e-9 {
		t.Errorf("task_list not increased: %v -> %v", *in.TaskListWidth, *out.TaskListWidth)
	}
}
