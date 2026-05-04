package task

import (
	"errors"
	"strings"
	"testing"
)

func TestFieldDefListSorted(t *testing.T) {
	fl := FieldDefList{
		{ID: 3, Name: "c", Type: FieldTypeText, Position: 2},
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 2, Name: "b", Type: FieldTypeText, Position: 1}, // タイブレーク
	}
	got := fl.Sorted()
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	// position 1 のうち id 昇順で a, b、その後 c。
	if got[0].ID != 1 || got[1].ID != 2 || got[2].ID != 3 {
		t.Errorf("sorted ids = [%d %d %d], want [1 2 3]", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestFieldDefListAddDef(t *testing.T) {
	fl := FieldDefList{
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 5, Name: "b", Type: FieldTypeText, Position: 3},
	}
	out, newID, err := fl.AddDef("c", FieldTypeText)
	if err != nil {
		t.Fatalf("AddDef err: %v", err)
	}
	if newID != 6 {
		t.Errorf("newID=%d, want 6 (max+1)", newID)
	}
	added, ok := out.ByID(newID)
	if !ok {
		t.Fatalf("new field not found")
	}
	if added.Position != 4 {
		t.Errorf("position=%d, want 4 (maxPos+1)", added.Position)
	}
	if added.Name != "c" || added.Type != FieldTypeText {
		t.Errorf("unexpected added=%+v", added)
	}

	// 空 name はエラー
	if _, _, err := fl.AddDef("", FieldTypeText); !errors.Is(err, ErrFieldEmptyName) {
		t.Errorf("empty name should be ErrFieldEmptyName, got %v", err)
	}
	// 18 rune 超過
	long := strings.Repeat("あ", 19)
	if _, _, err := fl.AddDef(long, FieldTypeText); !errors.Is(err, ErrFieldNameTooLong) {
		t.Errorf("too long name should be ErrFieldNameTooLong, got %v", err)
	}
	// 18 rune ぴったりは OK
	exact := strings.Repeat("あ", 18)
	if _, _, err := fl.AddDef(exact, FieldTypeText); err != nil {
		t.Errorf("exact 18 should be ok, got %v", err)
	}
	// 不明 type
	if _, _, err := fl.AddDef("x", FieldType("date")); !errors.Is(err, ErrFieldUnknownType) {
		t.Errorf("unknown type should be ErrFieldUnknownType, got %v", err)
	}
}

func TestFieldDefListRenameByID(t *testing.T) {
	fl := FieldDefList{
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
	}
	out, err := fl.RenameByID(1, "renamed")
	if err != nil {
		t.Fatalf("rename err: %v", err)
	}
	if out[0].Name != "renamed" {
		t.Errorf("name not changed, got %q", out[0].Name)
	}
	// 存在しない id
	if _, err := fl.RenameByID(99, "x"); err == nil {
		t.Errorf("rename nonexistent should fail")
	}
	// 空 name
	if _, err := fl.RenameByID(1, ""); !errors.Is(err, ErrFieldEmptyName) {
		t.Errorf("empty name should be ErrFieldEmptyName, got %v", err)
	}
}

func TestFieldDefListDeleteByID(t *testing.T) {
	fl := FieldDefList{
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 2, Name: "b", Type: FieldTypeText, Position: 2},
		{ID: 3, Name: "c", Type: FieldTypeText, Position: 3},
	}
	out, err := fl.DeleteByID(2)
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	// position が 1..N に振り直されている
	if out[0].Position != 1 || out[1].Position != 2 {
		t.Errorf("positions = [%d %d], want [1 2]", out[0].Position, out[1].Position)
	}
	// 残った id は 1, 3
	if out[0].ID != 1 || out[1].ID != 3 {
		t.Errorf("ids = [%d %d], want [1 3]", out[0].ID, out[1].ID)
	}
}

func TestFieldDefListMoveUpDown(t *testing.T) {
	fl := FieldDefList{
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 2, Name: "b", Type: FieldTypeText, Position: 2},
		{ID: 3, Name: "c", Type: FieldTypeText, Position: 3},
	}
	// id=2 を上へ → b, a, c
	up := fl.MoveUp(2)
	upSorted := up.Sorted()
	if upSorted[0].ID != 2 || upSorted[1].ID != 1 || upSorted[2].ID != 3 {
		t.Errorf("after MoveUp(2), sorted ids = [%d %d %d], want [2 1 3]",
			upSorted[0].ID, upSorted[1].ID, upSorted[2].ID)
	}
	// 先頭は no-op
	noop := fl.MoveUp(1)
	noopSorted := noop.Sorted()
	if noopSorted[0].ID != 1 {
		t.Errorf("MoveUp on head should be no-op")
	}
	// id=2 を下へ → a, c, b
	down := fl.MoveDown(2)
	downSorted := down.Sorted()
	if downSorted[0].ID != 1 || downSorted[1].ID != 3 || downSorted[2].ID != 2 {
		t.Errorf("after MoveDown(2), sorted ids = [%d %d %d], want [1 3 2]",
			downSorted[0].ID, downSorted[1].ID, downSorted[2].ID)
	}
	// 末尾は no-op
	noop2 := fl.MoveDown(3)
	noop2Sorted := noop2.Sorted()
	if noop2Sorted[2].ID != 3 {
		t.Errorf("MoveDown on tail should be no-op")
	}
}

func TestFieldDefListValidate(t *testing.T) {
	good := FieldDefList{
		{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
		{ID: 2, Name: "b", Type: FieldTypeText, Position: 2},
	}
	if err := good.Validate(); err != nil {
		t.Errorf("good should be valid, got %v", err)
	}
	cases := []struct {
		name string
		fl   FieldDefList
	}{
		{"id<=0", FieldDefList{{ID: 0, Name: "a", Type: FieldTypeText, Position: 1}}},
		{"dup id", FieldDefList{
			{ID: 1, Name: "a", Type: FieldTypeText, Position: 1},
			{ID: 1, Name: "b", Type: FieldTypeText, Position: 2},
		}},
		{"empty name", FieldDefList{{ID: 1, Name: "", Type: FieldTypeText, Position: 1}}},
		{"unknown type", FieldDefList{{ID: 1, Name: "a", Type: FieldType("date"), Position: 1}}},
		{"invalid pos", FieldDefList{{ID: 1, Name: "a", Type: FieldTypeText, Position: 0}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fl.Validate(); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func TestFieldDefListAssignMissingIDsAndPositions(t *testing.T) {
	fl := FieldDefList{
		{ID: 0, Name: "a", Type: "", Position: 0},
		{ID: 5, Name: "b", Type: FieldTypeText, Position: 3},
		{ID: 0, Name: "c", Type: FieldTypeText, Position: 0},
	}
	out, changed := fl.AssignMissingIDsAndPositions()
	if !changed {
		t.Errorf("changed should be true")
	}
	// id=0 だった a, c が採番される (max=5 → 6, 7)
	if out[0].ID != 6 || out[2].ID != 7 {
		t.Errorf("ids = [%d _ %d], want [6 _ 7]", out[0].ID, out[2].ID)
	}
	// position=0 だった a, c が採番される (maxPos=3 → 4, 5)
	if out[0].Position != 4 || out[2].Position != 5 {
		t.Errorf("positions = [%d _ %d], want [4 _ 5]", out[0].Position, out[2].Position)
	}
	// type 空だった a が text 補完
	if out[0].Type != FieldTypeText {
		t.Errorf("type = %q, want text", out[0].Type)
	}
}

func TestTaskFieldListSetValue(t *testing.T) {
	tfl := TaskFieldList{
		{ID: 1, FieldID: 10, Value: "old"},
	}
	// 既存 field_id への上書き
	out := tfl.SetValue(10, "new")
	if len(out) != 1 || out[0].Value != "new" {
		t.Errorf("update existing failed, got %+v", out)
	}
	// 新規 field_id への追加 (id は max+1 で採番)
	out2 := tfl.SetValue(20, "x")
	if len(out2) != 2 {
		t.Fatalf("len=%d", len(out2))
	}
	if out2[1].ID != 2 || out2[1].FieldID != 20 || out2[1].Value != "x" {
		t.Errorf("new entry = %+v, want id=2 field_id=20 value=x", out2[1])
	}
}

func TestTaskFieldListRemoveByFieldID(t *testing.T) {
	tfl := TaskFieldList{
		{ID: 1, FieldID: 10, Value: "a"},
		{ID: 2, FieldID: 20, Value: "b"},
	}
	out := tfl.RemoveByFieldID(10)
	if len(out) != 1 || out[0].FieldID != 20 {
		t.Errorf("remove failed, got %+v", out)
	}
}

func TestTaskFieldListValidate(t *testing.T) {
	defs := FieldDefList{
		{ID: 10, Name: "a", Type: FieldTypeText, Position: 1},
	}
	good := TaskFieldList{
		{ID: 1, FieldID: 10, Value: "ok"},
	}
	if err := good.Validate(defs); err != nil {
		t.Errorf("good should be valid, got %v", err)
	}
	// 未知の field_id
	bad := TaskFieldList{
		{ID: 1, FieldID: 99, Value: "x"},
	}
	if err := bad.Validate(defs); !errors.Is(err, ErrFieldUnknownFieldID) {
		t.Errorf("unknown field_id should be ErrFieldUnknownFieldID, got %v", err)
	}
	// id 重複
	dup := TaskFieldList{
		{ID: 1, FieldID: 10, Value: "x"},
		{ID: 1, FieldID: 10, Value: "y"},
	}
	if err := dup.Validate(defs); err == nil {
		t.Errorf("duplicate id should error")
	}
	// value 長さ超過
	long := strings.Repeat("あ", MaxFieldTextValueRunes+1)
	tooLong := TaskFieldList{
		{ID: 1, FieldID: 10, Value: long},
	}
	if err := tooLong.Validate(defs); !errors.Is(err, ErrFieldValueTooLong) {
		t.Errorf("too long value should be ErrFieldValueTooLong, got %v", err)
	}
}
