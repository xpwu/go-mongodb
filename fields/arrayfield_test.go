package fields

import (
	"github.com/xpwu/go-mongodb/x"
	"testing"

	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== 辅助类型 ====================

// mockElem 模拟一个 int 类型的 ElemField
type mockElem struct {
	name string
}

func (m *mockElem) FullName() string { return m.name }

func newTestArrayField[T any](name string) *arrayBaseField[T, *mockElem] {
	return &arrayBaseField[T, *mockElem]{
		BaseField: BaseField[[]T]{name: name},
		newElemField: func(n string) *mockElem {
			return &mockElem{name: n}
		},
	}
}

// ==================== AtPos 测试 ====================

func TestArrayField_AtPos_Zero(t *testing.T) {
	af := newTestArrayField[int]("scores")

	elem := af.AtPos(0)
	got := elem.FullName()

	if got != "scores.0" {
		t.Errorf("AtPos(0): got %v, want %v", got, "scores.0")
	}
}

func TestArrayField_AtPos_Negative(t *testing.T) {
	af := newTestArrayField[int]("scores")

	elem := af.AtPos(-1)
	got := elem.FullName()

	if got != "scores.-1" {
		t.Errorf("AtPos(-1): got %v, want %v", got, "scores.-1")
	}
}

func TestArrayField_AtPos_Middle(t *testing.T) {
	af := newTestArrayField[int]("tags")

	elem := af.AtPos(5)
	got := elem.FullName()

	if got != "tags.5" {
		t.Errorf("AtPos(5): got %v, want %v", got, "tags.5")
	}
}

// ==================== Size 测试 ====================

func TestArrayField_Size(t *testing.T) {
	af := newTestArrayField[int]("tags")

	f := af.Size(3)
	got := f.ToBsonD()

	want := bson.D{{"tags", bson.D{{"$size", 3}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Size: got %v, want %v", got, want)
	}
}

func TestArrayField_Size_Zero(t *testing.T) {
	af := newTestArrayField[int]("tags")

	f := af.Size(0)
	got := f.ToBsonD()

	want := bson.D{{"tags", bson.D{{"$size", 0}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Size(0): got %v, want %v", got, want)
	}
}

// ==================== PopFirst / PopLast 测试 ====================

func TestArrayField_PopFirst(t *testing.T) {
	af := newTestArrayField[int]("items")

	u := af.PopFirst()
	got := u.ToBsonM()

	want := bson.M{"$pop": bson.M{"items": -1}}
	if !bsonMEqual(got, want) {
		t.Errorf("PopFirst: got %v, want %v", got, want)
	}
}

func TestArrayField_PopLast(t *testing.T) {
	af := newTestArrayField[int]("items")

	u := af.PopLast()
	got := u.ToBsonM()

	want := bson.M{"$pop": bson.M{"items": 1}}
	if !bsonMEqual(got, want) {
		t.Errorf("PopLast: got %v, want %v", got, want)
	}
}

// ==================== AddEach 测试 ====================

func TestArrayField_AddEach(t *testing.T) {
	af := newTestArrayField[int]("tags")

	u := af.AddEach([]int{1, 2, 3})
	got := u.ToBsonM()

	want := bson.M{"$addToSet": bson.M{"tags": bson.M{"$each": x.ToBsonA([]int{1, 2, 3})}}}
	if !bsonMEqual(got, want) {
		t.Errorf("AddEach: got %v, want %v", got, want)
	}
}

func TestArrayField_AddEach_String(t *testing.T) {
	af := newTestArrayField[string]("tags")

	u := af.AddEach([]string{"a", "b"})
	got := u.ToBsonM()

	want := bson.M{"$addToSet": bson.M{"tags": bson.M{"$each": x.ToBsonA([]string{"a", "b"})}}}
	if !bsonMEqual(got, want) {
		t.Errorf("AddEach(string): got %v, want %v", got, want)
	}
}

func TestArrayField_AddEach_Empty(t *testing.T) {
	af := newTestArrayField[int]("tags")

	u := af.AddEach([]int{})
	got := u.ToBsonM()

	want := bson.M{"$addToSet": bson.M{"tags": bson.M{"$each": x.ToBsonA([]int{})}}}
	if !bsonMEqual(got, want) {
		t.Errorf("AddEach(empty): got %v, want %v", got, want)
	}
}

// ==================== RemoveValues 测试 ====================

func TestArrayField_RemoveValues(t *testing.T) {
	af := newTestArrayField[int]("scores")

	u := af.RemoveValues([]int{0, 5})
	got := u.ToBsonM()

	want := bson.M{"$pullAll": bson.M{"scores": x.ToBsonA([]int{0, 5})}}
	if !bsonMEqual(got, want) {
		t.Errorf("RemoveValues: got %v, want %v", got, want)
	}
}

func TestArrayField_RemoveValues_String(t *testing.T) {
	af := newTestArrayField[string]("tags")

	u := af.RemoveValues([]string{"deleted", "removed"})
	got := u.ToBsonM()

	want := bson.M{"$pullAll": bson.M{"tags": x.ToBsonA([]string{"deleted", "removed"})}}
	if !bsonMEqual(got, want) {
		t.Errorf("RemoveValues(string): got %v, want %v", got, want)
	}
}

// ==================== Push 测试 ====================

func TestArrayField_Push(t *testing.T) {
	af := newTestArrayField[int]("scores")

	u := af.Push([]int{90, 85, 88})
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"scores": x.ToBsonA([]int{90, 85, 88})}}
	if !bsonMEqual(got, want) {
		t.Errorf("Push: got %v, want %v", got, want)
	}
}

func TestArrayField_Push_Single(t *testing.T) {
	af := newTestArrayField[string]("tags")

	u := af.Push([]string{"newTag"})
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"tags": x.ToBsonA([]string{"newTag"})}}
	if !bsonMEqual(got, want) {
		t.Errorf("Push(single): got %v, want %v", got, want)
	}
}

// ==================== RemoveVirValue 测试 ====================

func TestArrayField_RemoveVirValue(t *testing.T) {
	af := newTestArrayField[int]("votes")

	u := af.RemoveVirValue(func(elem *mockElem) VirValue {
		f := filter.New(elem, "$gte", 6)
		return VirValue(f)
	})
	got := u.ToBsonM()

	// PullByFilter: &base{f, "$pull", filter.ToBsonD()}
	// filter.ToBsonD() of filter.New(elem, "$gte", 6) where elem.FullName() = ""
	// When name is "", base.ToBsonD() returns bson.D{{"$gte", 6}}
	want := bson.M{"$pull": bson.M{"votes": bson.D{{"$gte", 6}}}}
	if !bsonMEqual(got, want) {
		t.Errorf("RemoveVirValue: got %v, want %v", got, want)
	}
}

// ==================== PushWith 测试 ====================

func TestArrayField_PushWith_SortDesc(t *testing.T) {
	af := newTestArrayField[int]("scores")

	u := af.PushWith([]int{8, 7, 6}, func(elem *mockElem) *updater.PushModifier {
		return updater.NewModifier(updater.DescWith(elem))
	})
	got := u.ToBsonM()

	// elem.FullName() = "" when name is ""
	// DescWith: p.sort = bson.M{"": -1}
	want := bson.M{"$push": bson.M{"scores": bson.M{
		"$each": x.ToBsonA([]int{8, 7, 6}),
		"$sort": bson.M{"": -1},
	}}}
	if !bsonMEqual(got, want) {
		t.Errorf("PushWith(SortDesc): got %v, want %v", got, want)
	}
}

func TestArrayField_PushWith_Position(t *testing.T) {
	af := newTestArrayField[int]("scores")

	u := af.PushWith([]int{1, 2}, func(elem *mockElem) *updater.PushModifier {
		return updater.NewModifier(updater.Position(0))
	})
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"scores": bson.M{
		"$each":     x.ToBsonA([]int{1, 2}),
		"$position": 0,
	}}}
	if !bsonMEqual(got, want) {
		t.Errorf("PushWith(Position): got %v, want %v", got, want)
	}
}

func TestArrayField_PushWith_Slice(t *testing.T) {
	af := newTestArrayField[int]("scores")

	u := af.PushWith([]int{1, 2, 3}, func(elem *mockElem) *updater.PushModifier {
		return updater.NewModifier(updater.Slice(5))
	})
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"scores": bson.M{
		"$each":  x.ToBsonA([]int{1, 2, 3}),
		"$slice": 5,
	}}}
	if !bsonMEqual(got, want) {
		t.Errorf("PushWith(Slice): got %v, want %v", got, want)
	}
}

// ==================== AnyElemMeet 测试 ====================

func TestArrayField_AnyElemMeet(t *testing.T) {
	af := newTestArrayField[int]("scores")

	f := af.AnyElemMeet(func(anyElem *mockElem) filter.Filter {
		return filter.New(anyElem, "$gte", 90)
	})
	got := f.ToBsonD()

	// anyElem.FullName() = "scores" (因为传入的是 af.FullName())
	want := bson.D{{"scores", bson.D{{"$gte", 90}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("AnyElemMeet: got %v, want %v", got, want)
	}
}

// ==================== SameElemMeet 测试 ====================

func TestArrayField_SameElemMeet_And(t *testing.T) {
	af := newTestArrayField[int]("dim_cm")

	// 用 And 组合多个条件
	f := af.SameElemMeet(func(theOne *mockElem) filter.Filter {
		return filter.And(
			filter.New(theOne, "$gt", 22),
			filter.New(theOne, "$lt", 30),
		)
	})
	got := f.ToBsonD()

	want := bson.D{{"dim_cm", bson.D{{"$elemMatch",
		bson.D{{"$gt", 22}, {"$lt", 30}},
	}}}}
	if !bsonMEqual(DtoMDeeply(got), DtoMDeeply(want)) {
		t.Errorf("SameElemMeet(And): \ngot  %v, \nwant %v", got, want)
	}
}

func TestArrayField_SameElemMeet_Simple(t *testing.T) {
	af := newTestArrayField[int]("dim_cm")

	// 用单个 filter 而不是 And，更简单
	f := af.SameElemMeet(func(theOne *mockElem) filter.Filter {
		return filter.New(theOne, "$gt", 22)
	})
	got := f.ToBsonD()

	// SameElemMatch: &base{f: af, operator: "$elemMatch", value: filter.ToBsonD()}
	// theOne.FullName() = "" → filter.New("") → bson.D{{"$gt", 22}}
	want := bson.D{{"dim_cm", bson.D{{"$elemMatch", bson.D{{"$gt", 22}}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("SameElemMeet(simple): got %v, want %v", got, want)
	}
}

// ==================== Elems 测试 ====================

func TestArrayField_Elems(t *testing.T) {
	af := newTestArrayField[int]("tags")

	elem := af.Elems()
	got := elem.FullName()

	if got != "tags" {
		t.Errorf("Elems: got %v, want %v", got, "tags")
	}
}

// ==================== CoverValues 测试 ====================

func TestArrayField_CoverValues(t *testing.T) {
	af := newTestArrayField[int]("tags")

	f := af.CoverValues([]int{1, 2})
	got := f.ToBsonD()

	want := bson.D{{"tags", bson.D{{"$all", x.ToBsonA([]int{1, 2})}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("CoverValues: got %v, want %v", got, want)
	}
}

// ==================== CoverVirValues 测试 ====================

func TestArrayField_CoverVirValues(t *testing.T) {
	af := newTestArrayField[int]("results")

	f := af.CoverVirValues(func(sameElem *mockElem) []VirValue {
		v1 := VirValue(filter.New(sameElem, "$gte", 50))
		v2 := VirValue(filter.New(sameElem, "$lt", 100))
		return []VirValue{v1, v2}
	})
	got := f.ToBsonD()

	// sameElem.FullName() = "" → filter.New → bson.D{{"$gte", 50}} and bson.D{{"$lt", 100}}
	want := bson.D{{"results", bson.D{{"$all", bson.A{
		bson.D{{"$gte", 50}},
		bson.D{{"$lt", 100}},
	}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("CoverVirValues: \ngot  %v, \nwant %v", got, want)
	}
}

// ==================== AtVirPos 测试 ====================

func TestArrayField_AtVirPos(t *testing.T) {
	af := newTestArrayField[int]("grades")

	elem, arrayFilter := af.AtVirPos(func(elem *mockElem) VirPos {
		f := filter.New(elem, "$gte", 80)
		return VirPos(f)
	})

	// elem 的 FullName 应该是 grades.$[idN]
	elemName := elem.FullName()
	if elemName == "grades" || elemName == "" {
		t.Errorf("AtVirPos elem name: got %v, expected grades.$[idN]", elemName)
	}
	if len(elemName) < 10 || elemName[:9] != "grades.$[" {
		t.Errorf("AtVirPos elem name format: got %v, expected grades.$[idN]", elemName)
	}

	// arrayFilter 应该是一个有效的 filter
	if arrayFilter == nil {
		t.Error("AtVirPos: arrayFilter should not be nil")
	}

	// 验证 arrayFilter 的内容
	af2 := arrayFilter.ToBsonD()
	// elem.FullName() = "" → filter.New → bson.D{{"$gte", 80}}
	want := bson.D{{"$gte", 80}}
	if !bsonDEqual(af2[0].Value.(bson.D), want) {
		t.Errorf("AtVirPos filter: got %v, want %v", af2, want)
	}
}

// ==================== FirstMatched 测试 ====================

func TestArrayField_FirstMatched(t *testing.T) {
	af := newTestArrayField[int]("grades")

	elem := af.FirstMatched()
	got := elem.FullName()

	if got != "grades.$" {
		t.Errorf("FirstMatched: got %v, want %v", got, "grades.$")
	}
}

// ==================== UpdateAll 测试 ====================

func TestArrayField_UpdateAll(t *testing.T) {
	af := newTestArrayField[int]("scores")

	elem := af.UpdateAll()
	got := elem.FullName()

	if got != "scores.$[]" {
		t.Errorf("UpdateAll: got %v, want %v", got, "scores.$[]")
	}
}

// ==================== FullName 测试 ====================

func TestArrayField_FullName(t *testing.T) {
	af := newTestArrayField[int]("myArray")

	got := af.FullName()
	if got != "myArray" {
		t.Errorf("FullName: got %v, want %v", got, "myArray")
	}
}

// ==================== NewArrayField / NewArrayComparableField 测试 ====================

func TestNewArrayField(t *testing.T) {
	af := NewArrayField[int, *mockElem]("scores", func(n string) *mockElem {
		return &mockElem{name: n}
	})
	if af == nil {
		t.Fatal("NewArrayField: returned nil")
	}
	if af.FullName() != "scores" {
		t.Errorf("FullName: got %v, want %v", af.FullName(), "scores")
	}
}

func TestNewArrayComparableField(t *testing.T) {
	af := NewArrayComparableField[int, *mockElem]("tags", func(n string) *mockElem {
		return &mockElem{name: n}
	})
	if af == nil {
		t.Fatal("NewArrayComparableField: returned nil")
	}
	if af.FullName() != "tags" {
		t.Errorf("FullName: got %v, want %v", af.FullName(), "tags")
	}
}

// ==================== VirPos / VirValue / ArrayFilter 类型测试 ====================

func TestVirPos_IsFilter(t *testing.T) {
	// VirPos 是 filter.Filter 的别名
	var _ filter.Filter = VirPos(filter.New(&mockElem{name: "test"}, "$eq", 1))
}

func TestArrayFilter_IsFilter(t *testing.T) {
	// ArrayFilter 是 filter.Filter 的别名
	var _ filter.Filter = ArrayFilter(filter.New(&mockElem{name: "test"}, "$gte", 5))
}
