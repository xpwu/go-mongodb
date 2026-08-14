package updater

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- mock field.Field ---

type mockField struct {
	name string
}

func (m *mockField) FullName() string {
	return m.name
}

// --- helpers ---

func bsonMEqual(a, b bson.M) bool {
	return reflect.DeepEqual(a, b)
}

// --- tests ---

func TestUpdater_Set(t *testing.T) {
	f := &mockField{"name"}
	u := New(f, "$set", "Bob")
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"name": "Bob"}}
	if !bsonMEqual(got, want) {
		t.Errorf("Set: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Unset(t *testing.T) {
	f := &mockField{"name"}
	u := New(f, "$unset", "")
	got := u.ToBsonM()

	want := bson.M{"$unset": bson.M{"name": ""}}
	if !bsonMEqual(got, want) {
		t.Errorf("Unset: got %#v, want %#v", got, want)
	}
}

func TestUpdater_SetOnInsert(t *testing.T) {
	f := &mockField{"qty"}
	u := New(f, "$setOnInsert", 100)
	got := u.ToBsonM()

	want := bson.M{"$setOnInsert": bson.M{"qty": 100}}
	if !bsonMEqual(got, want) {
		t.Errorf("SetOnInsert: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Inc(t *testing.T) {
	f := &mockField{"age"}
	u := New(f, "$inc", 1)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"age": 1}}
	if !bsonMEqual(got, want) {
		t.Errorf("Inc: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Inc_Int64(t *testing.T) {
	f := &mockField{"count"}
	u := New(f, "$inc", int64(100))
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"count": int64(100)}}
	if !bsonMEqual(got, want) {
		t.Errorf("Inc Int64: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Mul(t *testing.T) {
	f := &mockField{"price"}
	u := New(f, "$mul", 2.5)
	got := u.ToBsonM()

	want := bson.M{"$mul": bson.M{"price": 2.5}}
	if !bsonMEqual(got, want) {
		t.Errorf("Mul: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Min(t *testing.T) {
	f := &mockField{"score"}
	u := New(f, "$min", 50)
	got := u.ToBsonM()

	want := bson.M{"$min": bson.M{"score": 50}}
	if !bsonMEqual(got, want) {
		t.Errorf("Min: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Max(t *testing.T) {
	f := &mockField{"score"}
	u := New(f, "$max", 100)
	got := u.ToBsonM()

	want := bson.M{"$max": bson.M{"score": 100}}
	if !bsonMEqual(got, want) {
		t.Errorf("Max: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Batch(t *testing.T) {
	f1 := &mockField{"age"}
	f2 := &mockField{"amount"}

	u1 := New(f1, "$set", 19)
	u2 := New(f2, "$inc", int64(100))
	u := Batch(u1, u2)
	got := u.ToBsonM()

	want := bson.M{
		"$set": bson.M{"age": 19},
		"$inc": bson.M{"amount": int64(100)},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Batch_Single(t *testing.T) {
	f := &mockField{"age"}
	u := Batch(New(f, "$set", 20))
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"age": 20}}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch single: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Batch_MergeSameOperator(t *testing.T) {
	// 两个 $set 应该合并到同一个 $set map 里
	f1 := &mockField{"name"}
	f2 := &mockField{"age"}

	u1 := New(f1, "$set", "Alice")
	u2 := New(f2, "$set", 30)
	u := Batch(u1, u2)
	got := u.ToBsonM()

	want := bson.M{
		"$set": bson.M{
			"name": "Alice",
			"age":  30,
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch merge same op: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Batch_Empty(t *testing.T) {
	u := Batch()
	got := u.ToBsonM()

	want := bson.M{}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch empty: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Push(t *testing.T) {
	f := &mockField{"tags"}
	u := New(f, "$push", "vip")
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"tags": "vip"}}
	if !bsonMEqual(got, want) {
		t.Errorf("Push: got %#v, want %#v", got, want)
	}
}

func TestUpdater_Push_WithEach(t *testing.T) {
	f := &mockField{"tags"}

	// 模拟 PushByModifier 的 value 构造
	each := bson.M{"$each": []string{"vip", "premium"}}
	u := New(f, "$push", each)
	got := u.ToBsonM()

	want := bson.M{"$push": bson.M{"tags": bson.M{"$each": []string{"vip", "premium"}}}}
	if !bsonMEqual(got, want) {
		t.Errorf("Push with $each: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_WithPosition(t *testing.T) {
	f := &mockField{"scores"}

	modifier := NewModifier(Position(0))
	values := []int{99, 100}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"scores": bson.M{
				"$each":     []int{99, 100},
				"$position": 0,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier Position: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_WithSlice(t *testing.T) {
	f := &mockField{"scores"}

	modifier := NewModifier(Slice(5))
	values := []int{1, 2, 3}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"scores": bson.M{
				"$each":  []int{1, 2, 3},
				"$slice": 5,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier Slice: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_WithSortAsc(t *testing.T) {
	f := &mockField{"scores"}

	modifier := NewModifier(Asc())
	values := []int{10, 20, 30}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"scores": bson.M{
				"$each": []int{10, 20, 30},
				"$sort": 1,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier Sort Asc: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_WithSortDesc(t *testing.T) {
	f := &mockField{"scores"}

	modifier := NewModifier(Desc())
	values := []int{10, 20, 30}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"scores": bson.M{
				"$each": []int{10, 20, 30},
				"$sort": -1,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier Sort Desc: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_WithSortAscWithField(t *testing.T) {
	f := &mockField{"quizzes"}
	sortField := &mockField{"score"}

	modifier := NewModifier(AscWith(sortField))
	values := []bson.M{{"wk": 5, "score": 8}}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"quizzes": bson.M{
				"$each": []bson.M{{"wk": 5, "score": 8}},
				"$sort": bson.M{"score": 1},
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier Sort AscWith: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushByModifier_AllOptions(t *testing.T) {
	f := &mockField{"quizzes"}

	modifier := NewModifier(
		Position(0),
		Slice(3),
		Desc(),
	)
	values := []bson.M{{"wk": 5, "score": 8}}

	u := PushByModifier(f, modifier, values)
	got := u.ToBsonM()

	want := bson.M{
		"$push": bson.M{
			"quizzes": bson.M{
				"$each":     []bson.M{{"wk": 5, "score": 8}},
				"$position": 0,
				"$slice":    3,
				"$sort":     -1,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushByModifier AllOptions: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PullByFilter(t *testing.T) {
	f := &mockField{"tags"}

	// 模拟 filter.Filter，用 bson.D 表示
	mockFilter := mockFilterBsonD(bson.D{{"age", bson.D{{"$gte", 18}}}})

	u := PullByFilter(f, mockFilter)
	got := u.ToBsonM()

	want := bson.M{
		"$pull": bson.M{
			"tags": bson.D{{"age", bson.D{{"$gte", 18}}}},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PullByFilter: got %#v, want %#v", got, want)
	}
}

func TestUpdater_PushModifier_DuplicateSortPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate sort, got none")
		}
	}()

	_ = NewModifier(Asc(), Desc())
}

// --- mock filter.Filter ---

type mockFilter struct {
	d bson.D
}

func mockFilterBsonD(d bson.D) *mockFilter {
	return &mockFilter{d: d}
}

func (m *mockFilter) ToBsonD() bson.D {
	return m.d
}
