package index

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

func bsonDEqual(a, b bson.D) bool {
	return reflect.DeepEqual(a, b)
}

// --- single-field key tests ---

func TestIndex_SingleField_Ascending(t *testing.T) {
	f := &mockField{"age"}
	key := NewKey(f, 1)
	got := key.ToBsonD()

	want := bson.D{{"age", 1}}
	if !bsonDEqual(got, want) {
		t.Errorf("Ascending: got %#v, want %#v", got, want)
	}
}

func TestIndex_SingleField_Descending(t *testing.T) {
	f := &mockField{"age"}
	key := NewKey(f, -1)
	got := key.ToBsonD()

	want := bson.D{{"age", -1}}
	if !bsonDEqual(got, want) {
		t.Errorf("Descending: got %#v, want %#v", got, want)
	}
}

func TestIndex_SingleField_Text(t *testing.T) {
	f := &mockField{"content"}
	key := NewKey(f, "text")
	got := key.ToBsonD()

	want := bson.D{{"content", "text"}}
	if !bsonDEqual(got, want) {
		t.Errorf("Text: got %#v, want %#v", got, want)
	}
}

func TestIndex_SingleField_2dsphere(t *testing.T) {
	f := &mockField{"location"}
	key := NewKey(f, "2dsphere")
	got := key.ToBsonD()

	want := bson.D{{"location", "2dsphere"}}
	if !bsonDEqual(got, want) {
		t.Errorf("2dsphere: got %#v, want %#v", got, want)
	}
}

func TestIndex_SingleField_2d(t *testing.T) {
	f := &mockField{"coords"}
	key := NewKey(f, "2d")
	got := key.ToBsonD()

	want := bson.D{{"coords", "2d"}}
	if !bsonDEqual(got, want) {
		t.Errorf("2d: got %#v, want %#v", got, want)
	}
}

// --- compound key tests ---

func TestIndex_Compound_TwoFields(t *testing.T) {
	f1 := &mockField{"name"}
	f2 := &mockField{"age"}

	compound := CompKeys([]Key{
		NewKey(f1, 1),
		NewKey(f2, -1),
	})
	got := compound.ToBsonD()

	want := bson.D{
		{"name", 1},
		{"age", -1},
	}
	if !bsonDEqual(got, want) {
		t.Errorf("Compound 2 fields: got %#v, want %#v", got, want)
	}
}

func TestIndex_Compound_ThreeFields(t *testing.T) {
	f1 := &mockField{"a"}
	f2 := &mockField{"b"}
	f3 := &mockField{"c"}

	compound := CompKeys([]Key{
		NewKey(f1, 1),
		NewKey(f2, -1),
		NewKey(f3, "text"),
	})
	got := compound.ToBsonD()

	want := bson.D{
		{"a", 1},
		{"b", -1},
		{"c", "text"},
	}
	if !bsonDEqual(got, want) {
		t.Errorf("Compound 3 fields: got %#v, want %#v", got, want)
	}
}

func TestIndex_Compound_Empty(t *testing.T) {
	compound := CompKeys(nil)
	got := compound.ToBsonD()

	want := bson.D{}
	if !bsonDEqual(got, want) {
		t.Errorf("Compound empty: got %#v, want %#v", got, want)
	}
}

// --- options tests ---

func TestIndex_Options_Unique(t *testing.T) {
	f := &mockField{"email"}
	key := NewKey(f, 1, Unique())
	opts := key.Options()

	if len(opts) != 1 || opts[0].Key != "unique" {
		t.Errorf("Options Unique: got %#v, want key 'unique'", opts)
	}
	if v, ok := opts[0].Value.(bool); !ok || !v {
		t.Errorf("Options Unique: expected true, got %#v", opts[0].Value)
	}
}

func TestIndex_Options_Sparse(t *testing.T) {
	f := &mockField{"email"}
	key := NewKey(f, 1, Sparse())
	opts := key.Options()

	if len(opts) != 1 || opts[0].Key != "sparse" {
		t.Errorf("Options Sparse: got %#v, want key 'sparse'", opts)
	}
}

func TestIndex_Options_Name(t *testing.T) {
	f := &mockField{"email"}
	key := NewKey(f, 1, Name("idx_email"))
	opts := key.Options()

	found := false
	for _, o := range opts {
		if o.Key == "name" && o.Value == "idx_email" {
			found = true
		}
	}
	if !found {
		t.Errorf("Options Name: got %#v, want name='idx_email'", opts)
	}
}

func TestIndex_Options_Combined(t *testing.T) {
	f := &mockField{"email"}
	key := NewKey(f, 1,
		Unique(),
		Sparse(),
		Name("idx_email"),
	)
	opts := key.Options()

	if len(opts) != 3 {
		t.Errorf("Options Combined: expected 3 options, got %d: %#v", len(opts), opts)
	}
}

func TestIndex_Compound_Options(t *testing.T) {
	f1 := &mockField{"name"}
	f2 := &mockField{"age"}

	compound := CompKeys(
		[]Key{NewKey(f1, 1), NewKey(f2, -1)},
		Unique(),
		Name("idx_name_age"),
	)
	got := compound.ToBsonD()
	opts := compound.Options()
	
	if len(opts) == 0 {
		t.Errorf("Compound Options: expected options, got none")
	}

	want := bson.D{
		{"name", 1},
		{"age", -1},
	}
	if !bsonDEqual(got, want) {
		t.Errorf("Compound Options ToBsonD: got %#v, want %#v", got, want)
	}
}

// --- mock filter.PartialIndexFilter ---

//type mockPartialFilter struct {
//	filter bson.D
//}
//
//func (m *mockPartialFilter) ToBsonD() bson.D {
//	return m.filter
//}
//
//// partialAble marks this as a PartialIndexFilter
//func (m *mockPartialFilter) partialAble() {}
