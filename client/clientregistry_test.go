package client

import (
	"bytes"
	"encoding/json"
	"github.com/xpwu/go-mongodb/x"
	"reflect"
	"testing"

	"github.com/xpwu/go-mongodb/fields"
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/projection"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ─────────────────────────────────────────────
// 辅助函数
// ─────────────────────────────────────────────

// encodeWithRegistry 通过自定义 registry 编码任意值为 bson.M
func encodeWithRegistry(r *bson.Registry, val interface{}) (bson.M, error) {
	var buf bytes.Buffer
	vw := bson.NewDocumentWriter(&buf)
	enc := bson.NewEncoder(vw)
	enc.SetRegistry(r)
	if err := enc.Encode(val); err != nil {
		return nil, err
	}
	var result bson.M
	if err := bson.Unmarshal(buf.Bytes(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// encodeRawWithRegistry 编码为原始 bson.Raw 字节
func encodeRawWithRegistry(r *bson.Registry, val interface{}) ([]byte, error) {
	var buf bytes.Buffer
	vw := bson.NewDocumentWriter(&buf)
	enc := bson.NewEncoder(vw)
	enc.SetRegistry(r)
	if err := enc.Encode(val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// JSONEqual 比较两个 JSON 字符串是否等价（忽略字段顺序）
func jsonEqual(a, b string) bool {
	var va, vb interface{}
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

// bsonMEqual 深度比较两个 bson.M
func bsonMEqual(got, want bson.M) bool {
	return jsonEqual(got.String(), filter.FlattenDoc(x.MtoDDeeply(want)).String())
}

// bsonDEqual 深度比较两个 bson.D
func bsonDEqual(a, b bson.D) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key {
			return false
		}
		if !bsonValueEqual(a[i].Value, b[i].Value) {
			return false
		}
	}
	return true
}

func bsonValueEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		switch bv := b.(type) {
		case int:
			return av == bv
		case int64:
			return int64(av) == bv
		}
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case bson.M:
		bv, ok := b.(bson.M)
		return ok && bsonMEqual(av, bv)
	case bson.D:
		bv, ok := b.(bson.D)
		return ok && bsonDEqual(av, bv)
	case bson.A:
		bv, ok := b.(bson.A)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !bsonValueEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case []string:
		bv, ok := b.([]string)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
		return true
	case []int:
		bv, ok := b.([]int)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
		return true
	}
	// 兜底：用 reflect.DeepEqual
	return reflect.DeepEqual(a, b)
}

// newTestField 创建一个测试用的 BaseField
func newTestField(name string) *fields.BaseField[any] {
	return fields.NewBaseField[any](name)
}

func TestGetLowerFieldRegistry_NotNil(t *testing.T) {
	registry := GetLowerFieldRegistry()

	if registry == nil {
		t.Errorf("GetLowerFieldRegistry: expected non-nil registry")
	}
}

func TestGetLowerFieldRegistry_HasUpdaterEncoder(t *testing.T) {
	registry := GetLowerFieldRegistry()

	// 验证注册了 updater.Updater 的 encoder
	updaterType := reflect.TypeOf((*updater.Updater)(nil)).Elem()
	enc, err := registry.LookupEncoder(updaterType)
	if err != nil {
		t.Errorf("Registry: updater.Updater encoder not found: %v", err)
	}
	if enc == nil {
		t.Errorf("Registry: updater.Updater encoder is nil")
	}
}

func TestGetLowerFieldRegistry_HasFilterEncoder(t *testing.T) {
	registry := GetLowerFieldRegistry()

	// 验证注册了 filter.Filter 的 encoder
	filterType := reflect.TypeOf((*filter.Filter)(nil)).Elem()
	enc, err := registry.LookupEncoder(filterType)
	if err != nil {
		t.Errorf("Registry: filter.Filter encoder not found: %v", err)
	}
	if enc == nil {
		t.Errorf("Registry: filter.Filter encoder is nil")
	}
}

// ─────────────────────────────────────────────
// GetLowerFieldRegistry 基础测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_IndependentInstances(t *testing.T) {
	r1 := GetLowerFieldRegistry()
	r2 := GetLowerFieldRegistry()
	if r1 == r2 {
		t.Error("GetLowerFieldRegistry() should return independent instances")
	}
}

// ─────────────────────────────────────────────
// Updater 编码器测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_Updater_Set(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("name")
	u := f.Set("Alice")

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$set": bson.M{"name": "Alice"}}
	if got.String() != want.String() {
		t.Errorf("Set: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_Inc(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	u := f.Inc(1)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$inc": bson.M{"age": 1}}
	if got.String() != want.String() {
		t.Errorf("Inc: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_Mul(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[float64]("score")
	u := f.Mul(2.5)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$mul": bson.M{"score": 2.5}}
	if got.String() != want.String() {
		t.Errorf("Mul: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_Unset(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("temp")
	u := f.Unset()

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$unset": bson.M{"temp": ""}}
	if got.String() != want.String() {
		t.Errorf("Unset: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_SetOnInsert(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("qty")
	u := f.SetOnInsert(100)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$setOnInsert": bson.M{"qty": 100}}
	if got.String() != want.String() {
		t.Errorf("SetOnInsert: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_SetMin(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("level")
	u := f.SetMin(5)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$min": bson.M{"level": 5}}
	if got.String() != want.String() {
		t.Errorf("SetMin: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_SetMax(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("level")
	u := f.SetMax(10)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$max": bson.M{"level": 10}}
	if got.String() != want.String() {
		t.Errorf("SetMax: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_Batch(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[int]("age")
	f2 := fields.NewBaseField[int]("score")
	u := updater.Batch(f1.Set(19), f2.Inc(100))

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{
		"$set": bson.M{"age": 19},
		"$inc": bson.M{"score": 100},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Batch: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_BatchMergeSameOp(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[int]("age")
	f2 := fields.NewBaseField[int]("score")
	// 两个都是 $set，应该合并到同一个 $set 下
	u := updater.Batch(f1.Set(19), f2.Set(100))

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{
		"$set": bson.M{"age": 19, "score": 100},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("BatchMergeSameOp: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_Push(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("tags", fields.NewIntField)
	u := f.Push([]int{1, 2, 3})

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$push": bson.M{"tags": bson.A{1, 2, 3}}}
	if got.String() != want.String() {
		t.Errorf("Push: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_PushWithModifier(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("quizzes", fields.NewIntField)
	mod := updater.NewModifier(updater.Desc(), updater.Slice(3))
	u := f.PushWith([]int{5, 6, 7}, func(elem fields.IntField) *updater.PushModifier {
		return mod
	})

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{
		"$push": bson.M{
			"quizzes": bson.M{
				"$each":  bson.A{5, 6, 7},
				"$sort":  -1,
				"$slice": 3,
			},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PushWithModifier: \ngot  %v, \nwant %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_PullByFilter(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("tags", fields.NewIntField)
	condField := fields.NewBaseField[int]("score")
	fil := condField.Gte(6)
	u := updater.PullByFilter(f, fil)

	got, err := encodeWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{
		"$pull": bson.M{
			"tags": bson.D{{"score", bson.D{{"$gte", 6}}}},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("PullByFilter: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Updater_WrongType(t *testing.T) {
	r := GetLowerFieldRegistry()
	// 传一个非 Updater 类型
	val := map[string]interface{}{"foo": "bar"}

	encoder := bson.NewEncoder(bson.NewDocumentWriter(&bytes.Buffer{}))
	encoder.SetRegistry(r)
	err := encoder.Encode(val)
	// 这个不应该报错（map 有默认编码器），但我们要确认不是走的 updater 编码器
	if err != nil {
		t.Logf("non-Updater encode returned (expected, map has default encoder): %v", err)
	}
}

// ─────────────────────────────────────────────
// Filter 编码器测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_Filter_Eq(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("name")
	fil := filter.CompareByValue(f, filter.EQ, "Alice")

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// EQ 走的是 FromBsonD 路径：{name: "Alice"}
	want := bson.M{"name": "Alice"}
	if !bsonMEqual(got, want) {
		t.Errorf("Eq: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Gte(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	fil := filter.CompareByValue(f, filter.GTE, 18)

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$gte": 18}}
	if !bsonMEqual(got, want) {
		t.Errorf("Gte: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Ne(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("status")
	fil := filter.CompareByValue(f, filter.NE, "deleted")

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"status": bson.M{"$ne": "deleted"}}
	if !bsonMEqual(got, want) {
		t.Errorf("Ne: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_In(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	fil := filter.New(f, "$in", bson.A{18, 19, 20})

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$in": bson.A{18, 19, 20}}}
	if !bsonMEqual(got, want) {
		t.Errorf("In: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_And(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[string]("name")
	f2 := fields.NewBaseField[int]("age")
	fil := filter.And(
		filter.CompareByValue(f1, filter.EQ, "Alice"),
		filter.CompareByValue(f2, filter.GTE, 18),
	)

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// And 两个不同字段 → $and 形式
	want := filter.FlattenDoc(bson.D{{
		"$and", bson.A{
			bson.D{{"name", "Alice"}},
			bson.D{{"age", bson.D{{"$gte", 18}}}},
		},
	}})
	if !bsonMEqual(got, x.DtoM(want)) {
		t.Errorf("And: \ngot  %v, \nwant %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Or(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[string]("name")
	f2 := fields.NewBaseField[string]("nickname")
	fil := filter.Or(
		filter.CompareByValue(f1, filter.EQ, "Alice"),
		filter.CompareByValue(f2, filter.EQ, "Bob"),
	)

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$or": bson.A{
		bson.M{"name": "Alice"},
		bson.M{"nickname": "Bob"},
	}}
	if !bsonMEqual(got, want) {
		t.Errorf("Or: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Not(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	fil := filter.Not(filter.CompareByValue(f, filter.GT, 5))

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	// Not(Gt) => {age: {$not: {$gt: 5}}}
	want := bson.M{"age": bson.M{"$not": bson.M{"$gt": 5}}}
	if !bsonMEqual(got, want) {
		t.Errorf("Not: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Exist(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	pif := f.Exist()

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"email": bson.M{"$exists": true}}
	if !bsonMEqual(got, want) {
		t.Errorf("Exist: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Filter_Nor(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[int]("age")
	f2 := fields.NewBaseField[string]("name")
	fil := filter.Nor(
		filter.CompareByValue(f1, filter.LT, 18),
		filter.CompareByValue(f2, filter.EQ, "Admin"),
	)

	got, err := encodeWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := bson.M{"$nor": bson.A{
		bson.M{"age": bson.M{"$lt": 18}},
		bson.M{"name": "Admin"},
	}}
	if !bsonMEqual(got, want) {
		t.Errorf("Nor: got %v, want %v", got, want)
	}
}

// ─────────────────────────────────────────────
// PartialIndexFilter 测试（重点新增）
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_PartialIndexFilter_Exist(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	pif := f.Exist()

	// PartialIndexFilter 也是 Filter，应该能被编码器处理
	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Exist failed: %v", err)
	}

	want := bson.M{"email": bson.M{"$exists": true}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Exist: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Type(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("score")
	pif := f.Type(bson.TypeInt32)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Type failed: %v", err)
	}

	want := bson.M{"score": bson.M{"$type": bson.TypeInt32}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Type: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Gt(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	pif := f.Gt(18)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Gt failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$gt": 18}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Gt: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Gte(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	pif := f.Gte(18)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Gte failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$gte": 18}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Gte: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Lt(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	pif := f.Lt(60)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Lt failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$lt": 60}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Lt: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Lte(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	pif := f.Lte(60)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Lte failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$lte": 60}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Lte: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_Eq(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("status")
	pif := f.Eq("active")

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.Eq failed: %v", err)
	}

	want := bson.M{"status": "active"}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter Eq: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_In(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	pif := f.In([]int{18, 21, 25})

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode PartialIndexFilter.In failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$in": bson.A{18, 21, 25}}}
	if !bsonMEqual(got, want) {
		t.Errorf("PartialIndexFilter In: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_AndPartial(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[int]("age")
	f2 := newTestField("email")
	pif := filter.AndPartial(
		f1.Gte(18),
		f2.Exist(),
	)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode AndPartial failed: %v", err)
	}

	// AndPartial 走 flattenAnd → $and 包裹两个不同字段
	want := bson.M{
		"$and": bson.A{
			bson.M{"age": bson.M{"$gte": 18}},
			bson.M{"email": bson.M{"$exists": true}},
		},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("AndPartial: \ngot  %v, \nwant %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_OrPartial(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := fields.NewBaseField[int]("age")
	f2 := fields.NewBaseField[int]("score")
	pif := filter.OrPartial(
		f1.Gte(18),
		f2.Gte(90),
	)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode OrPartial failed: %v", err)
	}

	want := bson.M{"$or": bson.A{
		bson.M{"age": bson.M{"$gte": 18}},
		bson.M{"score": bson.M{"$gte": 90}},
	}}
	if !bsonMEqual(got, want) {
		t.Errorf("OrPartial: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_AsPartialIndexFilter(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	// 用 AsPartialIndexFilter 包装一个普通 filter
	normalFilter := filter.CompareByValue(f, filter.GTE, 18)
	pif := filter.AsPartialIndexFilter(normalFilter)

	got, err := encodeWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode AsPartialIndexFilter failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$gte": 18}}
	if !bsonMEqual(got, want) {
		t.Errorf("AsPartialIndexFilter: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_PartialIndexFilter_UsedInIndexPartial(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	pif := f.Exist()

	// 模拟 index 的 Partial 选项
	key := index.NewKey(f, index.KeyTypeAscendingOrder, index.Partial(pif))

	// 验证 Key 的 ToBsonD
	kd := key.ToBsonD()
	wantD := bson.D{{"email", 1}}
	if !bsonDEqual(kd, wantD) {
		t.Errorf("Key ToBsonD: got %v, want %v", kd, wantD)
	}

	// 验证 Options 包含 partialFilterExpression
	opts := key.Options()
	// opts 应该包含 partialFilterExpression
	found := false
	for _, e := range opts {
		if e.Key == "partialFilterExpression" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Options should contain partialFilterExpression, got %v", opts)
	}

	// 通过 registry 编码整个 key
	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode key with partial failed: %v", err)
	}

	// 编码后应该包含 email:1 和 partialFilterExpression
	if v, ok := got["email"]; !ok || v.(int32) != 1 {
		t.Errorf("expected email:1 in encoded result, got %v", got)
	}
}

// ─────────────────────────────────────────────
// ArrayFilter / VirValue / VirPos 编码器测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_ArrayFilter(t *testing.T) {
	r := GetLowerFieldRegistry()
	// ArrayFilter 本质是 filter.Filter 的类型别名
	// 模拟数组元素条件：tags 数组中元素 >= 6
	elemField := fields.NewBaseField[int]("")
	fil := filter.CompareByValue(elemField, filter.GTE, 6)
	af := fields.ArrayFilter(fil)

	got, err := encodeWithRegistry(r, af)
	if err != nil {
		t.Fatalf("encode ArrayFilter failed: %v", err)
	}

	want := bson.M{"$gte": 6}
	if !bsonMEqual(got, want) {
		t.Errorf("ArrayFilter: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_VirValue(t *testing.T) {
	r := GetLowerFieldRegistry()
	elemField := fields.NewBaseField[string]("")
	vv := fields.VirValue(filter.CompareByValue(elemField, filter.EQ, "M"))

	// VirValue 也是 filter.Filter 的别名
	got, err := encodeWithRegistry(r, vv)
	if err != nil {
		t.Fatalf("encode VirValue failed: %v", err)
	}

	want := bson.M{"": "M"}
	if !bsonMEqual(got, want) {
		t.Errorf("VirValue: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_VirPos(t *testing.T) {
	r := GetLowerFieldRegistry()
	elemField := fields.NewBaseField[int]("")
	vp := fields.VirPos(filter.CompareByValue(elemField, filter.GTE, 8))

	got, err := encodeWithRegistry(r, vp)
	if err != nil {
		t.Fatalf("encode VirPos failed: %v", err)
	}

	want := bson.M{"$gte": 8}
	if !bsonMEqual(got, want) {
		t.Errorf("VirPos: got %v, want %v", got, want)
	}
}

// ─────────────────────────────────────────────
// Index Key 编码器测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_Key_Ascending(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("age")
	key := index.NewKey(f, index.KeyTypeAscendingOrder)

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key Ascending failed: %v", err)
	}

	want := bson.M{"age": 1}
	if !bsonMEqual(got, want) {
		t.Errorf("Key Ascending: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_Descending(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("age")
	key := index.NewKey(f, index.KeyTypeDescendingOrder)

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key Descending failed: %v", err)
	}

	want := bson.M{"age": -1}
	if !bsonMEqual(got, want) {
		t.Errorf("Key Descending: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_Text(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("content")
	key := index.NewKey(f, index.KeyTypeText)

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key Text failed: %v", err)
	}

	want := bson.M{"content": "text"}
	if !bsonMEqual(got, want) {
		t.Errorf("Key Text: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_2dSphere(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("location")
	key := index.NewKey(f, index.KeyType2dSphere)

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key 2dsphere failed: %v", err)
	}

	want := bson.M{"location": "2dsphere"}
	if !bsonMEqual(got, want) {
		t.Errorf("Key 2dsphere: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_2d(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("coords")
	key := index.NewKey(f, index.KeyType2d)

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key 2d failed: %v", err)
	}

	want := bson.M{"coords": "2d"}
	if !bsonMEqual(got, want) {
		t.Errorf("Key 2d: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_Compound(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := newTestField("name")
	f2 := fields.NewBaseField[int]("age")
	key := index.CompKeys([]index.Key{
		index.NewKey(f1, index.KeyTypeAscendingOrder),
		index.NewKey(f2, index.KeyTypeDescendingOrder),
	})

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Compound Key failed: %v", err)
	}

	want := bson.M{
		"name": 1,
		"age":  -1,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Compound Key: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_WithUnique(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	key := index.NewKey(f, index.KeyTypeAscendingOrder, index.Unique())

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key WithUnique failed: %v", err)
	}

	want := bson.M{
		"email": 1,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Key WithUnique: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_WithName(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	key := index.NewKey(f, index.KeyTypeAscendingOrder, index.Name("idx_email"))

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key WithName failed: %v", err)
	}

	want := bson.M{
		"email": 1,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Key WithName: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_Key_WithSparse(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	key := index.NewKey(f, index.KeyTypeAscendingOrder, index.Sparse())

	got, err := encodeWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode Key WithSparse failed: %v", err)
	}

	want := bson.M{
		"email": 1,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("Key WithSparse: got %v, want %v", got, want)
	}
}

// ─────────────────────────────────────────────
// IncludeBuilder / ExcludeBuilder 编码器测试
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_IncludeBuilder(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := newTestField("name")
	f2 := newTestField("email")
	inc := projection.Include(f1, f2)

	got, err := encodeWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode IncludeBuilder failed: %v", err)
	}

	want := bson.M{
		"name":  1,
		"email": 1,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("IncludeBuilder: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_IncludeBuilder_ExcludeID(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("name")
	inc := projection.Include(f)
	inc.Exclude_id()

	got, err := encodeWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode IncludeBuilder ExcludeID failed: %v", err)
	}

	want := bson.M{
		"name": 1,
		"_id":  0,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("IncludeBuilder ExcludeID: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_IncludeBuilder_WithSlice(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("comments", fields.NewIntField)
	inc := projection.IncludeWithSlice(f, 5)

	got, err := encodeWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode IncludeBuilder WithSlice failed: %v", err)
	}

	want := bson.M{
		"comments": bson.M{"$slice": 5},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("IncludeBuilder WithSlice: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_IncludeBuilder_WithSliceRange(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("comments", fields.NewIntField)
	inc := projection.IncludeWithSliceRange(f, 10, 5)

	got, err := encodeWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode IncludeBuilder WithSliceRange failed: %v", err)
	}

	want := bson.M{
		"comments": bson.M{"$slice": bson.A{10, 5}},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("IncludeBuilder WithSliceRange: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_IncludeBuilder_WithElemMatch(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewArrayField[int, fields.IntField]("scores", fields.NewIntField)
	//elemField := fields.NewBaseField[int]("")
	//fil := filter.CompareByValue(elemField, filter.GTE, 80)
	inc := f.ProjectWithElemMatch(func(theOne fields.IntField) filter.Filter {
		return theOne.Gte(80)
	})

	got, err := encodeWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode IncludeBuilder WithElemMatch failed: %v", err)
	}

	// IncludeWithElemMatch 用 fil.ToBsonD() 的第一个元素
	want := bson.M{
		"scores": bson.M{"$elemMatch": bson.M{"$gte": 80}},
	}
	if !bsonMEqual(got, want) {
		t.Errorf("IncludeBuilder WithElemMatch: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_ExcludeBuilder(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := newTestField("password")
	f2 := newTestField("secret")
	exc := projection.Exclude(f1, f2)

	got, err := encodeWithRegistry(r, exc)
	if err != nil {
		t.Fatalf("encode ExcludeBuilder failed: %v", err)
	}

	want := bson.M{
		"password": 0,
		"secret":   0,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("ExcludeBuilder: got %v, want %v", got, want)
	}
}

func TestGetLowerFieldRegistry_ExcludeBuilder_Dedup(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("password")
	// 重复添加同一个字段，应该只保留第一个
	exc := projection.Exclude(f, f)

	got, err := encodeWithRegistry(r, exc)
	if err != nil {
		t.Fatalf("encode ExcludeBuilder Dedup failed: %v", err)
	}

	want := bson.M{
		"password": 0,
	}
	if !bsonMEqual(got, want) {
		t.Errorf("ExcludeBuilder Dedup: got %v, want %v", got, want)
	}
}

// ─────────────────────────────────────────────
// 全注册验证
// ─────────────────────────────────────────────

func TestGetLowerFieldRegistry_AllEncodersRegistered(t *testing.T) {
	r := GetLowerFieldRegistry()

	// 验证所有 8 个类型都能找到编码器
	types := []struct {
		name string
		typ  reflect.Type
	}{
		{"Updater", reflect.TypeOf((*updater.Updater)(nil)).Elem()},
		{"Filter", reflect.TypeOf((*filter.Filter)(nil)).Elem()},
		{"PartialIndexFilter", reflect.TypeOf((*filter.PartialIndexFilter)(nil)).Elem()},
		{"ArrayFilter", reflect.TypeOf((*fields.ArrayFilter)(nil)).Elem()},
		{"VirValue", reflect.TypeOf((*fields.VirValue)(nil)).Elem()},
		{"VirPos", reflect.TypeOf((*fields.VirPos)(nil)).Elem()},
		{"IncludeBuilder", reflect.TypeOf((*projection.IncludeBuilder)(nil))},
		{"ExcludeBuilder", reflect.TypeOf((*projection.ExcludeBuilder)(nil))},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := r.LookupEncoder(tt.typ)
			if err != nil {
				t.Errorf("LookupEncoder(%s) failed: %v", tt.name, err)
			}
			if enc == nil {
				t.Errorf("LookupEncoder(%s) returned nil encoder", tt.name)
			}
		})
	}
}

// ─────────────────────────────────────────────
// GetPreserveFieldRegistry 测试
// ─────────────────────────────────────────────

func TestGetPreserveFieldRegistry_NotNil(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)
	//if err != nil {
	//	t.Fatalf("GetPreserveFieldRegistry(nil) failed: %v", err)
	//}
	if r == nil {
		t.Fatal("GetPreserveFieldRegistry returned nil")
	}
}

func TestGetPreserveFieldRegistry_WithBsonOpts(t *testing.T) {
	opts := &options.BSONOptions{
		OmitEmpty:               true,
		ZeroStructs:             true,
		UseJSONStructTags:       true,
		ErrorOnInlineDuplicates: true,
	}
	r := GetPreserveFieldRegistry(opts)
	if r == nil {
		t.Fatal("GetPreserveFieldRegistry with opts returned nil")
	}
}

func TestGetPreserveFieldRegistry_IndependentInstances(t *testing.T) {
	r1 := GetPreserveFieldRegistry(nil)
	r2 := GetPreserveFieldRegistry(nil)

	if r1 == r2 {
		t.Error("GetPreserveFieldRegistry should return independent instances")
	}
}

func TestGetPreserveFieldRegistry_StructCodecRegistered(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	// 验证 Struct 编解码器已注册
	enc, err := r.LookupEncoder(reflect.TypeOf(struct{}{}))
	if err != nil {
		t.Fatalf("LookupEncoder(struct) failed: %v", err)
	}
	if enc == nil {
		t.Fatal("Struct encoder is nil")
	}

	dec, err := r.LookupDecoder(reflect.TypeOf(struct{}{}))
	if err != nil {
		t.Fatalf("LookupDecoder(struct) failed: %v", err)
	}
	if dec == nil {
		t.Fatal("Struct decoder is nil")
	}
}

func TestGetPreserveFieldRegistry_IncludesLowerFieldEncoders(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	// Preserve registry 应该也包含 Lower registry 的所有编码器
	updaterType := reflect.TypeOf((*updater.Updater)(nil)).Elem()
	enc, err := r.LookupEncoder(updaterType)
	if err != nil {
		t.Errorf("Preserve registry missing Updater encoder: %v", err)
	}
	if enc == nil {
		t.Error("Preserve registry Updater encoder is nil")
	}
}

// TestPreserveStruct 是一个用于测试 PreserveStructCodec 的结构体
type TestPreserveStruct struct {
	FirstName string
	LastName  string `bson:"lastName"`
	Age       int    `bson:"Age"`
	Email     string `bson:"email"`
}

func TestGetPreserveFieldRegistry_PreservesFieldNames(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	s := TestPreserveStruct{
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		Email:     "john@example.com",
	}

	raw, err := encodeRawWithRegistry(r, s)
	if err != nil {
		t.Fatalf("encode struct failed: %v", err)
	}

	// 解析回来验证字段名保留了原始大小写
	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// 应该包含 "FirstName" 而不是 "firstname"
	if v, ok := result["FirstName"]; !ok || v != "John" {
		t.Errorf("Expected FirstName to be preserved, got %v", result)
	}
	if v, ok := result["lastName"]; !ok || v != "Doe" {
		t.Errorf("Expected LastName to be preserved, got %v", result)
	}
	if v, ok := result["Age"]; !ok || v.(int32) != 30 {
		t.Errorf("Expected Age to be preserved, got %v", result)
	}
	if v, ok := result["email"]; !ok || v != "john@example.com" {
		t.Errorf("Expected Email to be preserved, got %v", result)
	}

	// 确保没有小写版本
	if _, ok := result["firstname"]; ok {
		t.Error("Field name was lowercased to 'firstname', expected 'FirstName'")
	}
}

func TestGetPreserveFieldRegistry_PreservesNestedStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type InnerStruct struct {
		InnerField string
	}

	type OuterStruct struct {
		OuterField string
		Inner      InnerStruct `bson:"Inner"`
	}

	s := OuterStruct{
		OuterField: "outside",
		Inner: InnerStruct{
			InnerField: "inside",
		},
	}

	raw, err := encodeRawWithRegistry(r, s)
	if err != nil {
		t.Fatalf("encode nested struct failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if v, ok := result["OuterField"]; !ok || v != "outside" {
		t.Errorf("Expected OuterField preserved, got %v", result)
	}

	inner, ok := result["Inner"].(bson.D)
	if !ok {
		t.Fatalf("Expected Inner to be bson.D, got %T", result["Inner"])
	}
	innerM := x.DtoMDeeply(inner)
	if v, ok := innerM["InnerField"]; !ok || v != "inside" {
		t.Errorf("Expected Inner.InnerField preserved, got %v", inner)
	}
}

func TestGetPreserveFieldRegistry_PreservesInline(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type InnerStruct struct {
		InnerField string
	}

	type OuterStruct struct {
		OuterField string
		Inner      InnerStruct `bson:",inline"`
	}

	s := OuterStruct{
		OuterField: "outside",
		Inner: InnerStruct{
			InnerField: "inside",
		},
	}

	raw, err := encodeRawWithRegistry(r, s)
	if err != nil {
		t.Fatalf("encode nested struct failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if v, ok := result["OuterField"]; !ok || v != "outside" {
		t.Errorf("Expected OuterField preserved, got %v", result)
	}
	if v, ok := result["InnerField"]; !ok || v != "inside" {
		t.Errorf("Expected Inner.InnerField preserved, got %v", result)
	}
}

func TestGetPreserveFieldRegistry_DifferentFromLower(t *testing.T) {
	// Lower registry 会把字段名转小写
	// Preserve registry 保留原始大小写
	// 两者应该不同
	lowerR := GetLowerFieldRegistry()
	preserveR := GetPreserveFieldRegistry(nil)

	if lowerR == preserveR {
		t.Error("Lower and Preserve registries should be different")
	}

	// 两者都对 Updater 有编码器（共享 8 个自定义编码器）
	updaterType := reflect.TypeOf((*updater.Updater)(nil)).Elem()

	enc1, err1 := lowerR.LookupEncoder(updaterType)
	enc2, err2 := preserveR.LookupEncoder(updaterType)

	if err1 != nil || err2 != nil {
		t.Errorf("Both registries should have Updater encoder: %v, %v", err1, err2)
	}
	if enc1 == nil || enc2 == nil {
		t.Error("Both registries should have non-nil Updater encoder")
	}
}

func TestGetPreserveFieldRegistry_BsonOptsApplied(t *testing.T) {
	opts := &options.BSONOptions{
		OmitEmpty: true,
	}
	r := GetPreserveFieldRegistry(opts)

	s := TestPreserveStruct{
		FirstName: "Jane",
		// LastName 零值, Age 零值, Email 零值
	}

	raw, err := encodeRawWithRegistry(r, s)
	if err != nil {
		t.Fatalf("encode with OmitEmpty failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// FirstName 有值，应该存在
	if v, ok := result["FirstName"]; !ok || v != "Jane" {
		t.Errorf("Expected FirstName to be present, got %v", result)
	}

	// LastName 零值 + OmitEmpty，应该被省略
	if _, ok := result["LastName"]; ok {
		t.Errorf("Expected LastName to be omitted (OmitEmpty), got %v", result)
	}
}

// ─────────────────────────────────────────────
// Round-trip 端到端测试
// ─────────────────────────────────────────────

func TestRoundTrip_Updater(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("name")
	u := f.Set("Alice")

	raw, err := encodeRawWithRegistry(r, u)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	want := bson.M{"$set": bson.M{"name": "Alice"}}
	if !bsonMEqual(result, want) {
		t.Errorf("RoundTrip Updater: got %v, want %v", result, want)
	}
}

func TestRoundTrip_Filter(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := fields.NewBaseField[int]("age")
	fil := filter.CompareByValue(f, filter.GTE, 18)

	raw, err := encodeRawWithRegistry(r, fil)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	want := bson.M{"age": bson.M{"$gte": 18}}
	if !bsonMEqual(result, want) {
		t.Errorf("RoundTrip Filter: got %v, want %v", result, want)
	}
}

func TestRoundTrip_PartialIndexFilter(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("email")
	pif := f.Exist()

	raw, err := encodeRawWithRegistry(r, pif)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	want := bson.M{"email": bson.M{"$exists": true}}
	if !bsonMEqual(result, want) {
		t.Errorf("RoundTrip PartialIndexFilter: got %v, want %v", result, want)
	}
}

func TestRoundTrip_IndexKey(t *testing.T) {
	r := GetLowerFieldRegistry()
	f := newTestField("name")
	key := index.NewKey(f, index.KeyTypeAscendingOrder, index.Unique(), index.Name("idx_name"))

	raw, err := encodeRawWithRegistry(r, key)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// 验证 name:1 存在
	if v, ok := result["name"]; !ok {
		t.Errorf("Expected 'name' key in result, got %v", result)
	} else if v.(int32) != 1 {
		t.Errorf("Expected name:1, got name:%v", v)
	}
}

func TestRoundTrip_IncludeBuilder(t *testing.T) {
	r := GetLowerFieldRegistry()
	f1 := newTestField("name")
	f2 := newTestField("email")
	inc := projection.Include(f1, f2)

	raw, err := encodeRawWithRegistry(r, inc)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	want := bson.M{
		"name":  1,
		"email": 1,
	}
	if !bsonMEqual(result, want) {
		t.Errorf("RoundTrip IncludeBuilder: got %v, want %v", result, want)
	}
}

// ─────────────────────────────────────────────
// 错误语义测试
// ─────────────────────────────────────────────

func TestUpdaterEncoder_RejectsNonUpdater(t *testing.T) {
	r := GetLowerFieldRegistry()
	// 创建一个不满足 Updater 接口的值
	val := struct{ Foo string }{Foo: "bar"}

	encoder := bson.NewEncoder(bson.NewDocumentWriter(&bytes.Buffer{}))
	encoder.SetRegistry(r)
	err := encoder.Encode(val)
	// struct 有默认编码器，不会走 updater 编码器
	// 这里主要验证不会 panic
	if err != nil {
		t.Logf("struct encode with updater registry: %v (expected, uses struct encoder)", err)
	}
}
