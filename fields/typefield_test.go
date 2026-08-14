package fields

import (
	"testing"

	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== IntegerField 测试 ====================

func TestNewIntegerField(t *testing.T) {
	f := NewIntegerField[int]("count")
	if f == nil {
		t.Fatal("NewIntegerField[int]: returned nil")
	}
	if f.FullName() != "count" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "count")
	}
}

func TestIntegerField_Mod(t *testing.T) {
	f := NewIntegerField[int]("count")
	flt := f.Mod(5, 0)
	got := flt.ToBsonD()

	want := bson.D{{"count", bson.D{{"$mod", bson.A{5, 0}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Mod: got %v, want %v", got, want)
	}
}

func TestIntegerField_Mod_Odd(t *testing.T) {
	f := NewIntegerField[int]("count")
	flt := f.Mod(2, 1)
	got := flt.ToBsonD()

	want := bson.D{{"count", bson.D{{"$mod", bson.A{2, 1}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Mod(odd): got %v, want %v", got, want)
	}
}

func TestIntegerField_Set(t *testing.T) {
	f := NewIntegerField[int]("count")
	u := f.Set(42)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"count": 42}}
	if !bsonMEqual(got, want) {
		t.Errorf("Integer Set: got %v, want %v", got, want)
	}
}

func TestIntegerField_Inc(t *testing.T) {
	f := NewIntegerField[int]("count")
	u := f.Inc(10)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"count": 10}}
	if !bsonMEqual(got, want) {
		t.Errorf("Integer Inc: got %v, want %v", got, want)
	}
}

func TestIntegerField_Mul(t *testing.T) {
	f := NewIntegerField[int]("count")
	u := f.Mul(3)
	got := u.ToBsonM()

	want := bson.M{"$mul": bson.M{"count": 3}}
	if !bsonMEqual(got, want) {
		t.Errorf("Integer Mul: got %v, want %v", got, want)
	}
}

func TestIntegerField_SetMin(t *testing.T) {
	f := NewIntegerField[int]("lowest")
	u := f.SetMin(5)
	got := u.ToBsonM()

	want := bson.M{"$min": bson.M{"lowest": 5}}
	if !bsonMEqual(got, want) {
		t.Errorf("Integer SetMin: got %v, want %v", got, want)
	}
}

func TestIntegerField_SetMax(t *testing.T) {
	f := NewIntegerField[int]("highest")
	u := f.SetMax(100)
	got := u.ToBsonM()

	want := bson.M{"$max": bson.M{"highest": 100}}
	if !bsonMEqual(got, want) {
		t.Errorf("Integer SetMax: got %v, want %v", got, want)
	}
}

func TestIntegerField_Gt_Gte_Lt_Lte(t *testing.T) {
	f := NewIntegerField[int]("age")

	got1 := f.Gt(18).ToBsonD()
	want1 := bson.D{{"age", bson.D{{"$gt", 18}}}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("Integer Gt: got %v, want %v", got1, want1)
	}

	got2 := f.Gte(18).ToBsonD()
	want2 := bson.D{{"age", bson.D{{"$gte", 18}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("Integer Gte: got %v, want %v", got2, want2)
	}

	got3 := f.Lt(60).ToBsonD()
	want3 := bson.D{{"age", bson.D{{"$lt", 60}}}}
	if !bsonDEqual(got3, want3) {
		t.Errorf("Integer Lt: got %v, want %v", got3, want3)
	}

	got4 := f.Lte(100).ToBsonD()
	want4 := bson.D{{"age", bson.D{{"$lte", 100}}}}
	if !bsonDEqual(got4, want4) {
		t.Errorf("Integer Lte: got %v, want %v", got4, want4)
	}
}

func TestIntegerField_Eq_Ne(t *testing.T) {
	f := NewIntegerField[int]("age")

	got1 := f.Eq(25).ToBsonD()
	want1 := bson.D{{"age", 25}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("Integer Eq: got %v, want %v", got1, want1)
	}

	got2 := f.Ne(0).ToBsonD()
	want2 := bson.D{{"age", bson.D{{"$ne", 0}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("Integer Ne: got %v, want %v", got2, want2)
	}
}

func TestIntegerField_In_Nin(t *testing.T) {
	f := NewIntegerField[int]("age")

	got1 := f.In([]int{18, 21, 25}).ToBsonD()
	want1 := bson.D{{"age", bson.D{{"$in", bson.A{18, 21, 25}}}}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("Integer In: got %v, want %v", got1, want1)
	}

	got2 := f.Nin([]int{0, -1}).ToBsonD()
	want2 := bson.D{{"age", bson.D{{"$nin", bson.A{0, -1}}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("Integer Nin: got %v, want %v", got2, want2)
	}
}

func TestIntegerField_Exist_NotExist(t *testing.T) {
	f := NewIntegerField[int]("age")

	got1 := f.Exist().ToBsonD()
	want1 := bson.D{{"age", bson.D{{"$exists", true}}}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("Integer Exist: got %v, want %v", got1, want1)
	}

	got2 := f.NotExist().ToBsonD()
	want2 := bson.D{{"age", bson.D{{"$exists", false}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("Integer NotExist: got %v, want %v", got2, want2)
	}
}

func TestIntegerField_Type(t *testing.T) {
	f := NewIntegerField[int]("age")

	got := f.Type(bson.TypeInt32).ToBsonD()
	want := bson.D{{"age", bson.D{{"$type", bson.TypeInt32}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Integer Type: got %v, want %v", got, want)
	}
}

// ==================== UnIntegerField 测试 ====================

func TestNewUnIntegerField(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	if f == nil {
		t.Fatal("NewUnIntegerField[uint,int]: returned nil")
	}
	if f.FullName() != "count" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "count")
	}
}

func TestUnIntegerField_Mod(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	flt := f.Mod(3, 0)
	got := flt.ToBsonD()

	want := bson.D{{"count", bson.D{{"$mod", bson.A{uint(3), uint(0)}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("UnInteger Mod: got %v, want %v", got, want)
	}
}

func TestUnIntegerField_Inc(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	u := f.Inc(5)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"count": 5}}
	if !bsonMEqual(got, want) {
		t.Errorf("UnInteger Inc: got %v, want %v", got, want)
	}
}

func TestUnIntegerField_Mul(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	u := f.Mul(2)
	got := u.ToBsonM()

	want := bson.M{"$mul": bson.M{"count": uint(2)}}
	if !bsonMEqual(got, want) {
		t.Errorf("UnInteger Mul: \ngot  %v, \nwant %v", got, want)
	}
}

func TestUnIntegerField_Set(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	u := f.Set(100)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"count": uint(100)}}
	if !bsonMEqual(got, want) {
		t.Errorf("UnInteger Set: got %v, want %v", got, want)
	}
}

func TestUnIntegerField_Gt(t *testing.T) {
	f := NewUnIntegerField[uint, int]("count")
	got := f.Gt(10).ToBsonD()
	want := bson.D{{"count", bson.D{{"$gt", uint(10)}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("UnInteger Gt: got %v, want %v", got, want)
	}
}

// ==================== LikeStringField 测试 ====================

func TestNewLikeStringField(t *testing.T) {
	f := NewLikeStringField[string]("name")
	if f == nil {
		t.Fatal("NewLikeStringField: returned nil")
	}
	if f.FullName() != "name" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "name")
	}
}

func TestLikeStringField_Regex(t *testing.T) {
	f := NewLikeStringField[string]("name")
	regex := bson.Regex{Pattern: "^A.*", Options: "i"}
	flt := f.Regex(regex)
	got := flt.ToBsonD()

	want := bson.D{{"name", regex}}
	if !bsonDEqual(got, want) {
		t.Errorf("Regex: got %v, want %v", got, want)
	}
}

func TestLikeStringField_Regex_CaseSensitive(t *testing.T) {
	f := NewLikeStringField[string]("email")
	regex := bson.Regex{Pattern: ".*@gmail\\.com$", Options: ""}
	flt := f.Regex(regex)
	got := flt.ToBsonD()

	want := bson.D{{"email", regex}}
	if !bsonDEqual(got, want) {
		t.Errorf("Regex(case sensitive): got %v, want %v", got, want)
	}
}

func TestLikeStringField_Set(t *testing.T) {
	f := NewLikeStringField[string]("name")
	u := f.Set("Alice")
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"name": "Alice"}}
	if !bsonMEqual(got, want) {
		t.Errorf("LikeString Set: got %v, want %v", got, want)
	}
}

func TestLikeStringField_Eq_Ne(t *testing.T) {
	f := NewLikeStringField[string]("status")

	got1 := f.Eq("active").ToBsonD()
	want1 := bson.D{{"status", "active"}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("LikeString Eq: got %v, want %v", got1, want1)
	}

	got2 := f.Ne("deleted").ToBsonD()
	want2 := bson.D{{"status", bson.D{{"$ne", "deleted"}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("LikeString Ne: got %v, want %v", got2, want2)
	}
}

func TestLikeStringField_In(t *testing.T) {
	f := NewLikeStringField[string]("role")
	got := f.In([]string{"admin", "user"}).ToBsonD()
	want := bson.D{{"role", bson.D{{"$in", bson.A{"admin", "user"}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("LikeString In: got %v, want %v", got, want)
	}
}

func TestLikeStringField_Gt_Lt(t *testing.T) {
	f := NewLikeStringField[string]("name")

	got1 := f.Gt("A").ToBsonD()
	want1 := bson.D{{"name", bson.D{{"$gt", "A"}}}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("LikeString Gt: got %v, want %v", got1, want1)
	}

	got2 := f.Lt("z").ToBsonD()
	want2 := bson.D{{"name", bson.D{{"$lt", "z"}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("LikeString Lt: got %v, want %v", got2, want2)
	}
}

// ==================== ComparableField 测试 ====================

func TestNewComparableField_Bool(t *testing.T) {
	f := NewComparableField[bool]("isActive")
	if f == nil {
		t.Fatal("NewComparableField[bool]: returned nil")
	}
	if f.FullName() != "isActive" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "isActive")
	}
}

func TestComparableField_Bool_Eq(t *testing.T) {
	f := NewComparableField[bool]("isActive")
	got := f.Eq(true).ToBsonD()
	want := bson.D{{"isActive", true}}
	if !bsonDEqual(got, want) {
		t.Errorf("Bool Eq(true): got %v, want %v", got, want)
	}
}

func TestComparableField_Bool_Ne(t *testing.T) {
	f := NewComparableField[bool]("isActive")
	got := f.Ne(false).ToBsonD()
	want := bson.D{{"isActive", bson.D{{"$ne", false}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Bool Ne: got %v, want %v", got, want)
	}
}

func TestComparableField_Bool_Set(t *testing.T) {
	f := NewComparableField[bool]("isActive")
	u := f.Set(false)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"isActive": false}}
	if !bsonMEqual(got, want) {
		t.Errorf("Bool Set: got %v, want %v", got, want)
	}
}

func TestNewComparableField_ObjectID(t *testing.T) {
	f := NewComparableField[bson.ObjectID]("ownerId")
	if f == nil {
		t.Fatal("NewComparableField[ObjectID]: returned nil")
	}
	if f.FullName() != "ownerId" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "ownerId")
	}
}

func TestComparableField_ObjectID_Eq(t *testing.T) {
	f := NewComparableField[bson.ObjectID]("ownerId")
	var oid bson.ObjectID
	oid[3] = 0x42
	got := f.Eq(oid).ToBsonD()
	want := bson.D{{"ownerId", oid}}
	if !bsonDEqual(got, want) {
		t.Errorf("ObjectID Eq: got %v, want %v", got, want)
	}
}

// ==================== ComputableField 测试 ====================

func TestNewComputableField_Float32(t *testing.T) {
	f := NewComputableField[float32]("ratio")
	if f == nil {
		t.Fatal("NewComputableField[float32]: returned nil")
	}
	if f.FullName() != "ratio" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "ratio")
	}
}

func TestComputableField_Float32_Inc_Mul(t *testing.T) {
	f := NewComputableField[float32]("ratio")

	u1 := f.Inc(1.5)
	got1 := u1.ToBsonM()
	want1 := bson.M{"$inc": bson.M{"ratio": float32(1.5)}}
	if !bsonMEqual(got1, want1) {
		t.Errorf("Float32 Inc: got %v, want %v", got1, want1)
	}

	u2 := f.Mul(2.0)
	got2 := u2.ToBsonM()
	want2 := bson.M{"$mul": bson.M{"ratio": float32(2.0)}}
	if !bsonMEqual(got2, want2) {
		t.Errorf("Float32 Mul: got %v, want %v", got2, want2)
	}
}

func TestComputableField_Float32_SetMin_SetMax(t *testing.T) {
	f := NewComputableField[float32]("ratio")

	u1 := f.SetMin(0.1)
	got1 := u1.ToBsonM()
	want1 := bson.M{"$min": bson.M{"ratio": float32(0.1)}}
	if !bsonMEqual(got1, want1) {
		t.Errorf("Float32 SetMin: got %v, want %v", got1, want1)
	}

	u2 := f.SetMax(1.0)
	got2 := u2.ToBsonM()
	want2 := bson.M{"$max": bson.M{"ratio": float32(1.0)}}
	if !bsonMEqual(got2, want2) {
		t.Errorf("Float32 SetMax: got %v, want %v", got2, want2)
	}
}

func TestNewComputableField_Float64(t *testing.T) {
	f := NewComputableField[float64]("price")
	if f == nil {
		t.Fatal("NewComputableField[float64]: returned nil")
	}
	if f.FullName() != "price" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "price")
	}
}

func TestComputableField_Float64_Gt_Lt(t *testing.T) {
	f := NewComputableField[float64]("price")

	got1 := f.Gt(99.99).ToBsonD()
	want1 := bson.D{{"price", bson.D{{"$gt", 99.99}}}}
	if !bsonDEqual(got1, want1) {
		t.Errorf("Float64 Gt: got %v, want %v", got1, want1)
	}

	got2 := f.Lt(0.01).ToBsonD()
	want2 := bson.D{{"price", bson.D{{"$lt", 0.01}}}}
	if !bsonDEqual(got2, want2) {
		t.Errorf("Float64 Lt: got %v, want %v", got2, want2)
	}
}

func TestComputableField_Float64_Set(t *testing.T) {
	f := NewComputableField[float64]("price")
	u := f.Set(199.99)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"price": 199.99}}
	if !bsonMEqual(got, want) {
		t.Errorf("Float64 Set: got %v, want %v", got, want)
	}
}

func TestComputableField_Float64_Inc(t *testing.T) {
	f := NewComputableField[float64]("price")
	u := f.Inc(0.5)
	got := u.ToBsonM()

	want := bson.M{"$inc": bson.M{"price": 0.5}}
	if !bsonMEqual(got, want) {
		t.Errorf("Float64 Inc: got %v, want %v", got, want)
	}
}

// ==================== SubField 测试 ====================

func TestSubField_BothNonEmpty(t *testing.T) {
	got := SubField("parent", "child")
	if got != "parent.child" {
		t.Errorf("SubField: got %v, want %v", got, "parent.child")
	}
}

func TestSubField_EmptyParent(t *testing.T) {
	got := SubField("", "child")
	if got != "child" {
		t.Errorf("SubField(empty parent): got %v, want %v", got, "child")
	}
}

func TestSubField_EmptyChild(t *testing.T) {
	got := SubField("parent", "")
	if got != "parent" {
		t.Errorf("SubField(empty child): got %v, want %v", got, "parent")
	}
}

func TestSubField_BothEmpty(t *testing.T) {
	got := SubField("", "")
	if got != "" {
		t.Errorf("SubField(both empty): got %v, want empty", got)
	}
}

func TestSubField_Nested(t *testing.T) {
	got := SubField("a.b", "c")
	if got != "a.b.c" {
		t.Errorf("SubField(nested): got %v, want %v", got, "a.b.c")
	}
}

// ==================== 类型别名测试 ====================

func TestTypeAliases_Int(t *testing.T) {
	var _ IntegerField[int] = NewIntField("x")
	var _ IntegerField[int8] = NewInt8Field("x")
	var _ IntegerField[int16] = NewInt16Field("x")
	var _ IntegerField[int32] = NewInt32Field("x")
	var _ IntegerField[int64] = NewInt64Field("x")
}

func TestTypeAliases_Uint(t *testing.T) {
	var _ UnIntegerField[uint, int] = NewUintField("x")
	var _ UnIntegerField[byte, int8] = NewByteField("x")
	var _ UnIntegerField[uint8, int8] = NewUint8Field("x")
	var _ UnIntegerField[uint16, int16] = NewUint16Field("x")
	var _ UnIntegerField[uint32, int32] = NewUint32Field("x")
	var _ UnIntegerField[uint64, int64] = NewUint64Field("x")
}

func TestTypeAliases_String(t *testing.T) {
	var _ LikeStringField[string] = NewStringField("x")
	var _ StringField = NewStringField("x")
}

func TestTypeAliases_Bool(t *testing.T) {
	var _ ComparableField[bool] = NewBoolField("x")
}

func TestTypeAliases_Float(t *testing.T) {
	var _ ComputableField[float32] = NewFloat32Field("x")
	var _ ComputableField[float64] = NewFloat64Field("x")
}

// ==================== Integer 类型约束测试 ====================

func TestIntegerTypeConstraint_AllTypes(t *testing.T) {
	// 验证所有整数类型都能创建 IntegerField
	_ = NewIntegerField[int]("i")
	_ = NewIntegerField[int8]("i8")
	_ = NewIntegerField[int16]("i16")
	_ = NewIntegerField[int32]("i32")
	_ = NewIntegerField[int64]("i64")
	_ = NewIntegerField[uint]("u")
	_ = NewIntegerField[uint8]("u8")
	_ = NewIntegerField[uint16]("u16")
	_ = NewIntegerField[uint32]("u32")
	_ = NewIntegerField[uint64]("u64")
}

func TestUnIntegerTypeConstraint_AllTypes(t *testing.T) {
	_ = NewUnIntegerField[uint, int]("u")
	_ = NewUnIntegerField[uint8, int8]("u8")
	_ = NewUnIntegerField[uint16, int16]("u16")
	_ = NewUnIntegerField[uint32, int32]("u32")
	_ = NewUnIntegerField[uint64, int64]("u64")
}

// ==================== 接口实现验证 ====================

func TestIntegerField_ImplementsFilter(t *testing.T) {
	f := NewIntegerField[int]("test")
	// 验证实现了 filter.ComparableFilter[int]
	_ = filter.ComparableFilter[int](f)
}

func TestIntegerField_ImplementsUpdater(t *testing.T) {
	f := NewIntegerField[int]("test")
	// 验证实现了 updater.ComputableUpdater[int, int]
	_ = updater.ComputableUpdater[int, int](f)
}

func TestStringField_ImplementsFilter(t *testing.T) {
	f := NewStringField("test")
	_ = filter.ComparableFilter[string](f)
}

func TestStringField_ImplementsUpdater(t *testing.T) {
	f := NewStringField("test")
	_ = updater.BaseUpdater[string](f)
}
