package fields

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== Decimal128Field 测试 ====================

func TestNewDecimal128Field(t *testing.T) {
	f := NewDecimal128Field("price")
	if f == nil {
		t.Fatal("NewDecimal128Field: returned nil")
	}
	if f.FullName() != "price" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "price")
	}
}

func TestDecimal128Field_Set(t *testing.T) {
	f := NewDecimal128Field("price")
	val := bson.NewDecimal128(123, 456)
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"price": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("Decimal128 Set: got %v, want %v", got, want)
	}
}

func TestDecimal128Field_Gt(t *testing.T) {
	f := NewDecimal128Field("price")
	val := bson.NewDecimal128(0, 100)
	flt := f.Gt(val)
	got := flt.ToBsonD()

	want := bson.D{{"price", bson.D{{"$gt", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Decimal128 Gt: got %v, want %v", got, want)
	}
}

func TestDecimal128Field_Eq(t *testing.T) {
	f := NewDecimal128Field("price")
	val := bson.NewDecimal128(0, 100)
	flt := f.Eq(val)
	got := flt.ToBsonD()

	want := bson.D{{"price", val}}
	if !bsonDEqual(got, want) {
		t.Errorf("Decimal128 Eq: got %v, want %v", got, want)
	}
}

func TestDecimal128Field_In(t *testing.T) {
	f := NewDecimal128Field("price")
	v1 := bson.NewDecimal128(0, 100)
	v2 := bson.NewDecimal128(0, 200)
	flt := f.In([]bson.Decimal128{v1, v2})
	got := flt.ToBsonD()

	want := bson.D{{"price", bson.D{{"$in", bson.A{v1, v2}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Decimal128 In: got %v, want %v", got, want)
	}
}

func TestDecimal128Field_Ne(t *testing.T) {
	f := NewDecimal128Field("price")
	val := bson.NewDecimal128(0, 0)
	flt := f.Ne(val)
	got := flt.ToBsonD()

	want := bson.D{{"price", bson.D{{"$ne", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Decimal128 Ne: got %v, want %v", got, want)
	}
}

// ==================== BinaryField 测试 ====================

func TestNewBinaryField(t *testing.T) {
	f := NewBinaryField("data")
	if f == nil {
		t.Fatal("NewBinaryField: returned nil")
	}
	if f.FullName() != "data" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "data")
	}
}

func TestBinaryField_Set(t *testing.T) {
	f := NewBinaryField("data")
	val := bson.Binary{Subtype: 0x80, Data: []byte{0x01, 0x02, 0x03}}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"data": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("Binary Set: got %v, want %v", got, want)
	}
}

func TestBinaryField_Eq(t *testing.T) {
	f := NewBinaryField("data")
	val := bson.Binary{Subtype: 0x80, Data: []byte{0x01}}
	flt := f.Eq(val)
	got := flt.ToBsonD()

	want := bson.D{{"data", val}}
	if !bsonDEqual(got, want) {
		t.Errorf("Binary Eq: got %v, want %v", got, want)
	}
}

func TestBinaryField_Ne(t *testing.T) {
	f := NewBinaryField("data")
	val := bson.Binary{Subtype: 0x80, Data: []byte{0x01}}
	flt := f.Ne(val)
	got := flt.ToBsonD()

	want := bson.D{{"data", bson.D{{"$ne", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Binary Ne: got %v, want %v", got, want)
	}
}

func TestBinaryField_In(t *testing.T) {
	f := NewBinaryField("data")
	v1 := bson.Binary{Subtype: 0x80, Data: []byte{0x01}}
	v2 := bson.Binary{Subtype: 0x80, Data: []byte{0x02}}
	flt := f.In([]bson.Binary{v1, v2})
	got := flt.ToBsonD()

	want := bson.D{{"data", bson.D{{"$in", bson.A{v1, v2}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Binary In: got %v, want %v", got, want)
	}
}

// ==================== ObjectIDField 测试 ====================

func TestNewObjectIDField(t *testing.T) {
	f := NewObjectIDField("userId")
	if f == nil {
		t.Fatal("NewObjectIDField: returned nil")
	}
	if f.FullName() != "userId" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "userId")
	}
}

func TestObjectIDField_Eq(t *testing.T) {
	f := NewObjectIDField("userId")
	var oid bson.ObjectID
	oid[0] = 0x01
	flt := f.Eq(oid)
	got := flt.ToBsonD()

	want := bson.D{{"userId", oid}}
	if !bsonDEqual(got, want) {
		t.Errorf("ObjectID Eq: got %v, want %v", got, want)
	}
}

func TestObjectIDField_Ne(t *testing.T) {
	f := NewObjectIDField("userId")
	var oid bson.ObjectID
	flt := f.Ne(oid)
	got := flt.ToBsonD()

	want := bson.D{{"userId", bson.D{{"$ne", oid}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("ObjectID Ne: got %v, want %v", got, want)
	}
}

func TestObjectIDField_In(t *testing.T) {
	f := NewObjectIDField("userId")
	oids := []bson.ObjectID{{0x01}, {0x02}}
	flt := f.In(oids)
	got := flt.ToBsonD()

	want := bson.D{{"userId", bson.D{{"$in", bson.A{oids[0], oids[1]}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("ObjectID In: got %v, want %v", got, want)
	}
}

func TestObjectIDField_Set(t *testing.T) {
	f := NewObjectIDField("userId")
	var oid bson.ObjectID
	oid[5] = 0x42
	u := f.Set(oid)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"userId": oid}}
	if !bsonMEqual(got, want) {
		t.Errorf("ObjectID Set: got %v, want %v", got, want)
	}
}

func TestObjectIDField_Gt(t *testing.T) {
	f := NewObjectIDField("userId")
	var oid bson.ObjectID
	flt := f.Gt(oid)
	got := flt.ToBsonD()

	want := bson.D{{"userId", bson.D{{"$gt", oid}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("ObjectID Gt: got %v, want %v", got, want)
	}
}

// ==================== RawField 测试 ====================

func TestNewRawField(t *testing.T) {
	f := NewRawField("rawData")
	if f == nil {
		t.Fatal("NewRawField: returned nil")
	}
	if f.FullName() != "rawData" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "rawData")
	}
}

func TestRawField_Set(t *testing.T) {
	f := NewRawField("rawData")
	val := bson.Raw{0x05, 0x00, 0x00, 0x00}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"rawData": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("Raw Set: got %v, want %v", got, want)
	}
}

// ==================== RawValueField 测试 ====================

func TestNewRawValueField(t *testing.T) {
	f := NewRawValueField("rv")
	if f == nil {
		t.Fatal("NewRawValueField: returned nil")
	}
	if f.FullName() != "rv" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "rv")
	}
}

func TestRawValueField_Set(t *testing.T) {
	f := NewRawValueField("rv")
	val := bson.RawValue{Type: bson.TypeInt32, Value: []byte{0x01, 0x00, 0x00, 0x00}}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"rv": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("RawValue Set: got %v, want %v", got, want)
	}
}

// ==================== RawArrayField 测试 ====================

func TestNewRawArrayField(t *testing.T) {
	f := NewRawArrayField("ra")
	if f == nil {
		t.Fatal("NewRawArrayField: returned nil")
	}
	if f.FullName() != "ra" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "ra")
	}
}

func TestRawArrayField_Set(t *testing.T) {
	f := NewRawArrayField("ra")
	val := bson.RawArray{0x01, 0x00, 0x00, 0x00}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"ra": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("RawArray Set: got %v, want %v", got, want)
	}
}

// ==================== RawElementField 测试 ====================

func TestNewRawElementField(t *testing.T) {
	f := NewRawElementField("re")
	if f == nil {
		t.Fatal("NewRawElementField: returned nil")
	}
	if f.FullName() != "re" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "re")
	}
}

func TestRawElementField_Set(t *testing.T) {
	f := NewRawElementField("re")
	val := bson.RawElement{0x01, 0x00, 0x00, 0x00}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"re": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("RawElement Set: got %v, want %v", got, want)
	}
}

// ==================== DateTimeField 测试 ====================

func TestNewDateTimeField(t *testing.T) {
	f := NewDateTimeField("createdAt")
	if f == nil {
		t.Fatal("NewDateTimeField: returned nil")
	}
	if f.FullName() != "createdAt" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "createdAt")
	}
}

func TestDateTimeField_Set(t *testing.T) {
	f := NewDateTimeField("createdAt")
	val := bson.DateTime(1700000000000)
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"createdAt": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("DateTime Set: got %v, want %v", got, want)
	}
}

func TestDateTimeField_Gt(t *testing.T) {
	f := NewDateTimeField("createdAt")
	val := bson.DateTime(1600000000000)
	flt := f.Gt(val)
	got := flt.ToBsonD()

	want := bson.D{{"createdAt", bson.D{{"$gt", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("DateTime Gt: got %v, want %v", got, want)
	}
}

func TestDateTimeField_Gte(t *testing.T) {
	f := NewDateTimeField("createdAt")
	val := bson.DateTime(1600000000000)
	flt := f.Gte(val)
	got := flt.ToBsonD()

	want := bson.D{{"createdAt", bson.D{{"$gte", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("DateTime Gte: got %v, want %v", got, want)
	}
}

func TestDateTimeField_Lte(t *testing.T) {
	f := NewDateTimeField("createdAt")
	val := bson.DateTime(1700000000000)
	flt := f.Lte(val)
	got := flt.ToBsonD()

	want := bson.D{{"createdAt", bson.D{{"$lte", val}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("DateTime Lte: got %v, want %v", got, want)
	}
}

func TestDateTimeField_Eq(t *testing.T) {
	f := NewDateTimeField("createdAt")
	val := bson.DateTime(1700000000000)
	flt := f.Eq(val)
	got := flt.ToBsonD()

	want := bson.D{{"createdAt", val}}
	if !bsonDEqual(got, want) {
		t.Errorf("DateTime Eq: got %v, want %v", got, want)
	}
}

// ==================== TimestampField 测试 ====================

func TestNewTimestampField(t *testing.T) {
	f := NewTimestampField("ts")
	if f == nil {
		t.Fatal("NewTimestampField: returned nil")
	}
	if f.FullName() != "ts" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "ts")
	}
}

func TestTimestampField_Set(t *testing.T) {
	f := NewTimestampField("ts")
	val := bson.Timestamp{T: 1234, I: 1}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"ts": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("Timestamp Set: got %v, want %v", got, want)
	}
}

func TestTimestampField_Eq(t *testing.T) {
	f := NewTimestampField("ts")
	val := bson.Timestamp{T: 1234, I: 1}
	flt := f.Eq(val)
	got := flt.ToBsonD()

	want := bson.D{{"ts", val}}
	if !bsonDEqual(got, want) {
		t.Errorf("Timestamp Eq: got %v, want %v", got, want)
	}
}

// ==================== BsonMField 测试 ====================

func TestNewBsonMField(t *testing.T) {
	f := NewBsonMField("metadata")
	if f == nil {
		t.Fatal("NewBsonMField: returned nil")
	}
	if f.FullName() != "metadata" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "metadata")
	}
}

func TestBsonMField_Set(t *testing.T) {
	f := NewBsonMField("metadata")
	val := bson.M{"key": "value", "count": 42}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"metadata": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("BsonM Set: got %v, want %v", got, want)
	}
}

// ==================== BsonAField 测试 ====================

func TestNewBsonAField(t *testing.T) {
	f := NewBsonAField("items")
	if f == nil {
		t.Fatal("NewBsonAField: returned nil")
	}
	if f.FullName() != "items" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "items")
	}
}

func TestBsonAField_Set(t *testing.T) {
	f := NewBsonAField("items")
	val := bson.A{"a", "b", "c"}
	u := f.Set(val)
	got := u.ToBsonM()

	want := bson.M{"$set": bson.M{"items": val}}
	if !bsonMEqual(got, want) {
		t.Errorf("BsonA Set: got %v, want %v", got, want)
	}
}
