package fields

import (
	"testing"

	"github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== Filter 相关测试 ====================

func TestBaseField_Exist(t *testing.T) {
	b := NewBaseField[string]("name")
	f := b.Exist()
	got := f.ToBsonD()

	want := bson.D{{"name", bson.D{{"$exists", true}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Exist: got %v, want %v", got, want)
	}
}

func TestBaseField_NotExist(t *testing.T) {
	b := NewBaseField[string]("name")
	f := b.NotExist()
	got := f.ToBsonD()

	want := bson.D{{"name", bson.D{{"$exists", false}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("NotExist: got %v, want %v", got, want)
	}
}

func TestBaseField_Type(t *testing.T) {
	b := NewBaseField[string]("name")
	f := b.Type(bson.TypeString)
	got := f.ToBsonD()

	want := bson.D{{"name", bson.D{{"$type", bson.TypeString}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Type: got %v, want %v", got, want)
	}
}

func TestBaseField_Gt(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Gt(18)
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$gt", 18}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Gt: got %v, want %v", got, want)
	}
}

func TestBaseField_GtField(t *testing.T) {
	b1 := NewBaseField[int]("age")
	b2 := NewBaseField[int]("minAge")
	f := b1.GtField(b2)
	got := f.ToBsonD()

	// CompareByField 产生 exprFilter: {$expr: [{$gt: ["$age", "$minAge"]}]}
	want := bson.D{{"$expr", bson.D{{"$gt", bson.A{"$age", "$minAge"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("GtField: got %v, want %v", got, want)
	}
}

func TestBaseField_Lt(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Lt(60)
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$lt", 60}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Lt: got %v, want %v", got, want)
	}
}

func TestBaseField_LtField(t *testing.T) {
	b1 := NewBaseField[int]("maxAge")
	b2 := NewBaseField[int]("age")
	f := b1.LtField(b2)
	got := f.ToBsonD()

	want := bson.D{{"$expr", bson.D{{"$lt", bson.A{"$maxAge", "$age"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("LtField: got %v, want %v", got, want)
	}
}

func TestBaseField_Gte(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Gte(18)
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$gte", 18}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Gte: got %v, want %v", got, want)
	}
}

func TestBaseField_Lte(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Lte(100)
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$lte", 100}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Lte: got %v, want %v", got, want)
	}
}

func TestBaseField_Eq(t *testing.T) {
	b := NewBaseField[string]("name")
	f := b.Eq("Alice")
	got := f.ToBsonD()

	// EQ 在 CompareByValue 中直接用 FromBsonD(bson.D{{f.FullName(), value}})
	want := bson.D{{"name", "Alice"}}
	if !bsonDEqual(got, want) {
		t.Errorf("Eq(string): got %v, want %v", got, want)
	}
}

func TestBaseField_EqInt(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Eq(25)
	got := f.ToBsonD()

	want := bson.D{{"age", 25}}
	if !bsonDEqual(got, want) {
		t.Errorf("Eq(int): got %v, want %v", got, want)
	}
}

func TestBaseField_EqField(t *testing.T) {
	b1 := NewBaseField[int]("age")
	b2 := NewBaseField[int]("birthYear")
	f := b1.EqField(b2)
	got := f.ToBsonD()

	want := bson.D{{"$expr", bson.D{{"$eq", bson.A{"$age", "$birthYear"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("EqField: got %v, want %v", got, want)
	}
}

func TestBaseField_Ne(t *testing.T) {
	b := NewBaseField[string]("status")
	f := b.Ne("deleted")
	got := f.ToBsonD()

	want := bson.D{{"status", bson.D{{"$ne", "deleted"}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Ne: got %v, want %v", got, want)
	}
}

func TestBaseField_NeField(t *testing.T) {
	b1 := NewBaseField[int]("a")
	b2 := NewBaseField[int]("b")
	f := b1.NeField(b2)
	got := f.ToBsonD()

	want := bson.D{{"$expr", bson.D{{"$ne", bson.A{"$a", "$b"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("NeField: got %v, want %v", got, want)
	}
}

func TestBaseField_In(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.In([]int{18, 19, 20})
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$in", bson.A{18, 19, 20}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("In: got %v, want %v", got, want)
	}
}

func TestBaseField_In_StringSlice(t *testing.T) {
	b := NewBaseField[string]("name")
	f := b.In([]string{"Alice", "Bob"})
	got := f.ToBsonD()

	want := bson.D{{"name", bson.D{{"$in", bson.A{"Alice", "Bob"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("In(string): got %v, want %v", got, want)
	}
}

func TestBaseField_Nin(t *testing.T) {
	b := NewBaseField[int]("age")
	f := b.Nin([]int{0, 1})
	got := f.ToBsonD()

	want := bson.D{{"age", bson.D{{"$nin", bson.A{0, 1}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Nin: got %v, want %v", got, want)
	}
}

// ==================== Updater 相关测试 ====================

func TestBaseField_Set(t *testing.T) {
	b := NewBaseField[string]("name")
	u := b.Set("Bob")
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"name": "Bob"}}
	if !bsonMEqual(got, want) {
		t.Errorf("Set: got %v, want %v", got, want)
	}
}

func TestBaseField_SetInt(t *testing.T) {
	b := NewBaseField[int]("age")
	u := b.Set(30)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"age": 30}}
	if !bsonMEqual(got, want) {
		t.Errorf("Set(int): got %v, want %v", got, want)
	}
}

func TestBaseField_SetOnInsert(t *testing.T) {
	b := NewBaseField[string]("name")
	u := b.SetOnInsert("default")
	got := u.ToBsonM()

	want := bson.M{"$setOnInsert": bson.M{"name": "default"}}
	if !bsonMEqual(got, want) {
		t.Errorf("SetOnInsert: got %v, want %v", got, want)
	}
}

func TestBaseField_Unset(t *testing.T) {
	b := NewBaseField[string]("name")
	u := b.Unset()
	got := u.ToBsonM()

	want := bson.M{"$unset": bson.M{"name": ""}}
	if !bsonMEqual(got, want) {
		t.Errorf("Unset: got %v, want %v", got, want)
	}
}

func TestBaseField_Inc(t *testing.T) {
	b := NewBaseField[int]("count")
	u := b.Inc(5)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"count": 5}}
	if !bsonMEqual(got, want) {
		t.Errorf("Inc: got %v, want %v", got, want)
	}
}

func TestBaseField_IncFloat(t *testing.T) {
	b := NewBaseField[float64]("amount")
	u := b.Inc(1.5)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"amount": 1.5}}
	if !bsonMEqual(got, want) {
		t.Errorf("Inc(float): got %v, want %v", got, want)
	}
}

func TestBaseField_Mul(t *testing.T) {
	b := NewBaseField[float64]("price")
	u := b.Mul(2.0)
	got := u.ToBsonM()

	want := bson.M{"$mul": bson.M{"price": 2.0}}
	if !bsonMEqual(got, want) {
		t.Errorf("Mul: got %v, want %v", got, want)
	}
}

func TestBaseField_MulInt(t *testing.T) {
	b := NewBaseField[int]("count")
	u := b.Mul(3)
	got := u.ToBsonM()

	want := bson.M{"$mul": bson.M{"count": 3}}
	if !bsonMEqual(got, want) {
		t.Errorf("Mul(int): got %v, want %v", got, want)
	}
}

func TestBaseField_SetMin(t *testing.T) {
	b := NewBaseField[int]("lowest")
	u := b.SetMin(10)
	got := u.ToBsonM()

	want := bson.M{"$min": bson.M{"lowest": 10}}
	if !bsonMEqual(got, want) {
		t.Errorf("SetMin: got %v, want %v", got, want)
	}
}

func TestBaseField_SetMax(t *testing.T) {
	b := NewBaseField[int]("highest")
	u := b.SetMax(100)
	got := u.ToBsonM()

	want := bson.M{"$max": bson.M{"highest": 100}}
	if !bsonMEqual(got, want) {
		t.Errorf("SetMax: got %v, want %v", got, want)
	}
}

// ==================== Index 相关测试 ====================

func TestBaseField_AscIndex(t *testing.T) {
	b := NewBaseField[string]("name")
	key := b.AscIndex()
	got := key.ToBsonD()

	want := bson.D{{"name", 1}}
	if !bsonDEqual(got, want) {
		t.Errorf("AscIndex: got %v, want %v", got, want)
	}
}

func TestBaseField_DescIndex(t *testing.T) {
	b := NewBaseField[int]("age")
	key := b.DescIndex()
	got := key.ToBsonD()

	want := bson.D{{"age", -1}}
	if !bsonDEqual(got, want) {
		t.Errorf("DescIndex: got %v, want %v", got, want)
	}
}

func TestBaseField_AscIndex_Unique(t *testing.T) {
	b := NewBaseField[string]("email")
	key := b.AscIndex(index.Unique())
	opts := key.Options()
	found := false
	for _, e := range opts {
		if e.Key == "unique" {
			found = true
			if e.Value != true {
				t.Errorf("Unique: expected true, got %v", e.Value)
			}
		}
	}
	if !found {
		t.Errorf("Options: expected unique option, got %v", opts)
	}
}

func TestBaseField_AscIndex_Name(t *testing.T) {
	b := NewBaseField[string]("email")
	key := b.AscIndex(index.Name("idx_email"))
	opts := key.Options()
	found := false
	for _, e := range opts {
		if e.Key == "name" {
			found = true
			if e.Value != "idx_email" {
				t.Errorf("Name: expected idx_email, got %v", e.Value)
			}
		}
	}
	if !found {
		t.Errorf("Options: expected name option, got %v", opts)
	}
}

// ==================== FullName 测试 ====================

func TestBaseField_FullName(t *testing.T) {
	b := NewBaseField[string]("userName")
	got := b.FullName()

	if got != "userName" {
		t.Errorf("FullName: got %v, want %v", got, "userName")
	}
}

func TestBaseField_FullName_Nested(t *testing.T) {
	b := NewBaseField[string]("address.city")
	got := b.FullName()

	if got != "address.city" {
		t.Errorf("FullName(nested): got %v, want %v", got, "address.city")
	}
}

// ==================== Batch 合并测试 ====================

func TestBaseField_Batch_Merge(t *testing.T) {
	b1 := NewBaseField[string]("name")
	b2 := NewBaseField[int]("age")

	u := updater.Batch(b1.Set("Bob"), b2.Inc(1))
	got := u.ToBsonM()

	want := bson.M{
		"$set": bson.M{"name": "Bob"},
		"$inc": bson.M{"age": 1},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch: got %v, want %v", got, want)
	}
}

func TestBaseField_Batch_SameOp(t *testing.T) {
	b1 := NewBaseField[string]("name")
	b2 := NewBaseField[string]("email")

	u := updater.Batch(b1.Set("Bob"), b2.Set("bob@example.com"))
	got := u.ToBsonM()

	want := bson.M{
		"$set": bson.M{
			"name":  "Bob",
			"email": "bob@example.com",
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch(same op): got %v, want %v", got, want)
	}
}
