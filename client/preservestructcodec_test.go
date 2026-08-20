package client

import (
	"bytes"
	"github.com/xpwu/go-mongodb/x"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ─────────────────────────────────────────────
// 辅助函数
// ─────────────────────────────────────────────

func encodeRaw(r *bson.Registry, val interface{}) ([]byte, error) {
	var buf bytes.Buffer
	vw := bson.NewDocumentWriter(&buf)
	enc := bson.NewEncoder(vw)
	enc.SetRegistry(r)
	if err := enc.Encode(val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bsonUnmarshal(raw []byte, val interface{}) error {
	return decodeRaw(bson.NewRegistry(), raw, val)
}

func decodeRaw(r *bson.Registry, raw []byte, val interface{}) error {
	vr := bson.NewDocumentReader(bytes.NewReader(raw))
	dec := bson.NewDecoder(vr)
	dec.DefaultDocumentM()
	dec.SetRegistry(r)
	return dec.Decode(val)
}

func roundTrip(r *bson.Registry, original interface{}, decoded interface{}) error {
	raw, err := encodeRaw(r, original)
	if err != nil {
		return err
	}
	return decodeRaw(r, raw, decoded)
}

// ─────────────────────────────────────────────
// 0. 核心语义验证：不写 tag 时自动保留大写
// ─────────────────────────────────────────────

func TestPreserveStructCodec_NoTag_PreservesOriginalCase(t *testing.T) {
	// 核心语义：没有 bson tag 时，字段名保持 Go 原始大小写
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		FirstName string // 没有 tag
		LastName  string // 没有 tag
		Age       int    // 没有 tag
		Score     float64
	}

	d := Doc{FirstName: "John", LastName: "Doe", Age: 30, Score: 95.5}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// 验证：没有 tag 时，字段名保持 Go 原始大小写
	if result["FirstName"] != "John" {
		t.Errorf("FirstName: got %v, want John", result["FirstName"])
	}
	if result["LastName"] != "Doe" {
		t.Errorf("LastName: got %v, want Doe", result["LastName"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age: got %v, want 30", result["Age"])
	}
	if result["Score"] != 95.5 {
		t.Errorf("Score: got %v, want 95.5", result["Score"])
	}

	// 确认没有小写版本
	if _, ok := result["firstname"]; ok {
		t.Error("firstname should not exist")
	}
	if _, ok := result["lastname"]; ok {
		t.Error("lastname should not exist")
	}
	if _, ok := result["age"]; ok {
		t.Error("age should not exist")
	}
	if _, ok := result["score"]; ok {
		t.Error("score should not exist")
	}
}

func TestPreserveStructCodec_NoTag_DecodeRoundTrip(t *testing.T) {
	// 不写 tag 时，编解码 round-trip 正确
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name  string
		Age   int
		Score float64
	}

	original := Doc{Name: "Alice", Age: 25, Score: 88.5}

	var decoded Doc
	if err := roundTrip(r, original, &decoded); err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Age != original.Age {
		t.Errorf("Age: got %d, want %d", decoded.Age, original.Age)
	}
	if decoded.Score != original.Score {
		t.Errorf("Score: got %f, want %f", decoded.Score, original.Score)
	}
}

func TestPreserveStructCodec_MixedTagAndNoTag(t *testing.T) {
	// 混合场景：有的字段有 tag（重命名），有的没有 tag（保留大写）
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		FirstName string `bson:"firstName"` // 有 tag，用 tag 名
		LastName  string // 没有 tag，保留原始大小写
		Age       int    // 没有 tag，保留原始大小写
		Email     string `bson:"email"` // 有 tag，用 tag 名
	}

	d := Doc{FirstName: "John", LastName: "Doe", Age: 30, Email: "john@example.com"}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// 有 tag 的字段：用 tag 里的名字
	if result["firstName"] != "John" {
		t.Errorf("firstName (from tag): got %v, want John", result["firstName"])
	}
	if result["email"] != "john@example.com" {
		t.Errorf("email (from tag): got %v, want john@example.com", result["email"])
	}

	// 没有 tag 的字段：保留 Go 原始大小写
	if result["LastName"] != "Doe" {
		t.Errorf("LastName (no tag): got %v, want Doe", result["LastName"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age (no tag): got %v, want 30", result["Age"])
	}

	// 确认没有小写版本
	if _, ok := result["lastname"]; ok {
		t.Error("lastname should not exist")
	}
	if _, ok := result["age"]; ok {
		t.Error("age should not exist")
	}
}

func TestPreserveStructCodec_NestedNoTag_PreservesCase(t *testing.T) {
	// 嵌套 struct 没有 tag 时，内层字段也保留原始大小写
	r := GetPreserveFieldRegistry(nil)

	type Address struct {
		Street string // 没有 tag
		City   string // 没有 tag
		Zip    int    // 没有 tag
	}
	type Person struct {
		Name    string  // 没有 tag
		Address Address // 没有 tag
	}

	p := Person{
		Name: "Bob",
		Address: Address{
			Street: "123 Main St",
			City:   "NYC",
			Zip:    10001,
		},
	}

	raw, err := encodeRaw(r, p)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Bob" {
		t.Errorf("Name: got %v, want Bob", result["Name"])
	}

	addr, ok := result["Address"].(bson.M)
	if !ok {
		t.Fatalf("Address should be a document, got %T", result["Address"])
	}
	if addr["Street"] != "123 Main St" {
		t.Errorf("Street: got %v, want '123 Main St'", addr["Street"])
	}
	if addr["City"] != "NYC" {
		t.Errorf("City: got %v, want NYC", addr["City"])
	}
	if addr["Zip"] != int32(10001) {
		t.Errorf("Zip: got %v, want 10001", addr["Zip"])
	}
}

// ─────────────────────────────────────────────
// 1. EncodeValue 基础测试（修正为不写 tag 验证核心语义）
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_BasicFields(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name  string
		Age   int
		Score float64
	}

	d := Doc{Name: "Alice", Age: 30, Score: 95.5}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age: got %v, want 30", result["Age"])
	}
	if result["Score"] != 95.5 {
		t.Errorf("Score: got %v, want 95.5", result["Score"])
	}
}

func TestPreserveStructCodec_Encode_BSONTag(t *testing.T) {
	// 有 tag 时，尊重 tag 里指定的名字（不管大小写）
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		FirstName string `bson:"firstName"`
		LastName  string `bson:"lastName"`
	}

	d := Doc{FirstName: "John", LastName: "Doe"}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["firstName"] != "John" {
		t.Errorf("firstName: got %v, want John", result["firstName"])
	}
	if result["lastName"] != "Doe" {
		t.Errorf("lastName: got %v, want Doe", result["lastName"])
	}
}

func TestPreserveStructCodec_Encode_OmitEmptyGlobal(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string
		Age  int
	}

	d := Doc{Name: "Alice"} // Age=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted (global OmitEmpty), got %v", result["Age"])
	}
}

func TestPreserveStructCodec_Encode_TagOmitEmptyBehavior(t *testing.T) {
	// 探测 tag 级 omitempty 的实际行为
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string `bson:",omitempty"`
		Age  int    `bson:",omitempty"`
	}

	d := Doc{Name: "Alice"} // Age=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	// 记录实际行为，不假设结果
	if _, ok := result["Age"]; ok {
		t.Errorf("tag omitempty NOT working: zero Age present = %v", result["Age"])
	}
}

func TestPreserveStructCodec_Encode_NestedStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Address struct {
		Street string
		City   string
		Zip    int
	}
	type Person struct {
		Name    string
		Address Address
	}

	p := Person{
		Name: "Bob",
		Address: Address{
			Street: "123 Main St",
			City:   "NYC",
			Zip:    10001,
		},
	}

	raw, err := encodeRaw(r, p)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Bob" {
		t.Errorf("Name: got %v, want Bob", result["Name"])
	}

	addr, ok := result["Address"].(bson.M)
	if !ok {
		t.Fatalf("Address should be a document, got %T", result["Address"])
	}
	if addr["Street"] != "123 Main St" {
		t.Errorf("Street: got %v, want '123 Main St'", addr["Street"])
	}
	if addr["City"] != "NYC" {
		t.Errorf("City: got %v, want NYC", addr["City"])
	}
	if addr["Zip"] != int32(10001) {
		t.Errorf("Zip: got %v, want 10001", addr["Zip"])
	}
}

func TestPreserveStructCodec_Encode_PointerToStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Age  int
	}

	d := &Doc{Name: "Alice", Age: 30}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age: got %v, want 30", result["Age"])
	}
}

func TestPreserveStructCodec_Encode_SkipTag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name   string `bson:"Name"`
		Secret string `bson:"-"`
	}

	d := Doc{Name: "Alice", Secret: "hidden"}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["Secret"]; ok {
		t.Errorf("Secret should be skipped (bson:\"-\"), got %v", result["Secret"])
	}
}

func TestPreserveStructCodec_Encode_PrivateFieldsIgnored(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name    string
		private string
	}

	d := Doc{Name: "Alice", private: "secret"}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["private"]; ok {
		t.Errorf("private field should be ignored, got %v", result["private"])
	}
}

// ─────────────────────────────────────────────
// 2. DecodeValue 基础测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Decode_BasicFields(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Age  int
	}

	raw, err := encodeRaw(r, bson.M{"Name": "Alice", "Age": int32(30)})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
	if decoded.Age != 30 {
		t.Errorf("Age: got %d, want 30", decoded.Age)
	}
}

func TestPreserveStructCodec_Decode_NestedStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Address struct {
		Street string
		City   string
	}
	type Person struct {
		Name    string
		Address Address
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": "Bob",
		"Address": bson.M{
			"Street": "123 Main St",
			"City":   "NYC",
		},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Person
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Bob" {
		t.Errorf("Name: got %q, want Bob", decoded.Name)
	}
	if decoded.Address.Street != "123 Main St" {
		t.Errorf("Street: got %q, want '123 Main St'", decoded.Address.Street)
	}
	if decoded.Address.City != "NYC" {
		t.Errorf("City: got %q, want NYC", decoded.Address.City)
	}
}

func TestPreserveStructCodec_Decode_PointerFieldAutoAlloc(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Inner *Inner
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": "Alice",
		"Inner": bson.M{
			"Value": int32(42),
		},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
	if decoded.Inner == nil {
		t.Fatal("Inner should be auto-allocated, got nil")
	}
	if decoded.Inner.Value != 42 {
		t.Errorf("Inner.Value: got %d, want 42", decoded.Inner.Value)
	}
}

func TestPreserveStructCodec_Decode_UnknownFieldsIgnored(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":    "Alice",
		"Unknown": "value",
		"Age":     int32(30),
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
}

func TestPreserveStructCodec_Decode_NullValue(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Age  int
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": nil,
		"Age":  nil,
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "" {
		t.Errorf("Name should be zero value, got %q", decoded.Name)
	}
	if decoded.Age != 0 {
		t.Errorf("Age should be zero value, got %d", decoded.Age)
	}
}

func TestPreserveStructCodec_Decode_UndefinedValue(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
	}

	raw, err := encodeRaw(r, bson.M{"Name": nil})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "" {
		t.Errorf("Name should be zero after null, got %q", decoded.Name)
	}
}

// ─────────────────────────────────────────────
// 3. ZeroStructs / ZeroMaps 选项测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Decode_ZeroStructsOption(t *testing.T) {
	opts := &options.BSONOptions{ZeroStructs: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string
		Age  int
	}

	decoded := Doc{Name: "old", Age: 99}

	raw, err := encodeRaw(r, bson.M{"Name": "new", "Age": int32(50)})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "new" {
		t.Errorf("Name: got %q, want new", decoded.Name)
	}
	if decoded.Age != 50 {
		t.Errorf("Age: got %d, want 50", decoded.Age)
	}
}

func TestPreserveStructCodec_Decode_NoZeroStructOption(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Age  int
	}

	decoded := Doc{Name: "old", Age: 99}

	raw, err := encodeRaw(r, bson.M{"Name": "new"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "new" {
		t.Errorf("Name: got %q, want new", decoded.Name)
	}
	if decoded.Age != 99 {
		t.Errorf("Age should keep old value 99, got %d", decoded.Age)
	}
}

func TestPreserveStructCodec_Decode_DeepZeroInline(t *testing.T) {
	opts := &options.BSONOptions{ZeroStructs: true}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Inner Inner
	}

	decoded := Doc{
		Name:  "old",
		Inner: Inner{Value: 999},
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": "new",
		"Inner": bson.M{
			"Value": int32(42),
		},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "new" {
		t.Errorf("Name: got %q, want new", decoded.Name)
	}
	if decoded.Inner.Value != 42 {
		t.Errorf("Inner.Value: got %d, want 42", decoded.Inner.Value)
	}
}

// ─────────────────────────────────────────────
// 4. Inline 展开测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_InlineStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID   string
		Type string
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	d := Doc{
		Embedded: Embedded{ID: "abc123", Type: "user"},
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["ID"] != "abc123" {
		t.Errorf("ID (inlined): got %v, want abc123", result["ID"])
	}
	if result["Type"] != "user" {
		t.Errorf("Type (inlined): got %v, want user", result["Type"])
	}
	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
}

func TestPreserveStructCodec_Decode_InlineStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID   string
		Type string
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	raw, err := encodeRaw(r, bson.M{
		"ID":   "abc123",
		"Type": "user",
		"Name": "Alice",
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.ID != "abc123" {
		t.Errorf("ID: got %q, want abc123", decoded.ID)
	}
	if decoded.Type != "user" {
		t.Errorf("Type: got %q, want user", decoded.Type)
	}
	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
}

func TestPreserveStructCodec_Encode_InlinePointer(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID string
	}
	type Doc struct {
		*Embedded `bson:",inline"`
		Name      string
	}

	d := Doc{
		Embedded: &Embedded{ID: "ptr123"},
		Name:     "Bob",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["ID"] != "ptr123" {
		t.Errorf("ID (inlined pointer): got %v, want ptr123", result["ID"])
	}
	if result["Name"] != "Bob" {
		t.Errorf("Name: got %v, want Bob", result["Name"])
	}
}

func TestPreserveStructCodec_Decode_InlinePointerAutoAlloc(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID string
	}
	type Doc struct {
		*Embedded `bson:",inline"`
		Name      string
	}

	raw, err := encodeRaw(r, bson.M{
		"ID":   "ptr123",
		"Name": "Bob",
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Embedded == nil {
		t.Fatal("Embedded pointer should be auto-allocated")
	}
	if decoded.ID != "ptr123" {
		t.Errorf("ID: got %q, want ptr123", decoded.ID)
	}
	if decoded.Name != "Bob" {
		t.Errorf("Name: got %q, want Bob", decoded.Name)
	}
}

func TestPreserveStructCodec_Encode_InlineMap(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Ext  map[string]interface{} `bson:",inline"`
	}

	d := Doc{
		Name: "Alice",
		Ext: map[string]interface{}{
			"Custom1": "value1",
			"Custom2": int32(42),
		},
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if result["Custom1"] != "value1" {
		t.Errorf("Custom1 (from inline map): got %v, want value1", result["Custom1"])
	}
	if result["Custom2"] != int32(42) {
		t.Errorf("Custom2 (from inline map): got %v, want 42", result["Custom2"])
	}
}

func TestPreserveStructCodec_Decode_InlineMap(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Ext  map[string]interface{} `bson:",inline"`
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":    "Alice",
		"Custom1": "value1",
		"Custom2": int32(42),
		"Custom3": true,
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
	if decoded.Ext == nil {
		t.Fatal("Ext map should be auto-created")
	}
	if decoded.Ext["Custom1"] != "value1" {
		t.Errorf("Ext[Custom1]: got %v, want value1", decoded.Ext["Custom1"])
	}
	if decoded.Ext["Custom2"] != int32(42) {
		t.Errorf("Ext[Custom2]: got %v, want 42", decoded.Ext["Custom2"])
	}
	if decoded.Ext["Custom3"] != true {
		t.Errorf("Ext[Custom3]: got %v, want true", decoded.Ext["Custom3"])
	}
}

func TestPreserveStructCodec_Decode_InlineMapConflictWithField(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Ext  map[string]interface{} `bson:",inline"`
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":    "Alice",
		"NameDup": "should go to map",
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", decoded.Name)
	}
	if decoded.Ext["NameDup"] != "should go to map" {
		t.Errorf("Ext[NameDup]: got %v, want 'should go to map'", decoded.Ext["NameDup"])
	}
}

// ─────────────────────────────────────────────
// 5. Duplicate inline 冲突测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_DuplicateInline_Error(t *testing.T) {
	opts := &options.BSONOptions{ErrorOnInlineDuplicates: true}
	r := GetPreserveFieldRegistry(opts)

	type A struct {
		Field string
	}
	type B struct {
		Field int
	}
	type Conflict struct {
		A `bson:",inline"`
		B `bson:",inline"`
	}

	c := Conflict{}
	raw, err := encodeRaw(r, bson.M{"Field": "test"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	err = decodeRaw(r, raw, &c)
	if err == nil {
		t.Error("expected error for duplicate inline fields, got nil")
	}
	if !strings.Contains(err.Error(), "duplicated key") {
		t.Errorf("error = %v, should mention 'duplicated key'", err)
	}
}

// ─────────────────────────────────────────────
// 6. UseJSONStructTags 测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_UseJSONStructTags(t *testing.T) {
	opts := &options.BSONOptions{UseJSONStructTags: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Age       int    `json:"age"`
	}

	d := Doc{FirstName: "John", LastName: "Doe", Age: 30}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["first_name"] != "John" {
		t.Errorf("first_name (from json tag): got %v, want John", result["first_name"])
	}
	if result["last_name"] != "Doe" {
		t.Errorf("last_name (from json tag): got %v, want Doe", result["last_name"])
	}
	if result["age"] != int32(30) {
		t.Errorf("age (from json tag): got %v, want 30", result["age"])
	}
}

func TestPreserveStructCodec_Decode_UseJSONStructTags(t *testing.T) {
	opts := &options.BSONOptions{UseJSONStructTags: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		FirstName string `json:"first_name"`
		Age       int    `json:"age"`
	}

	raw, err := encodeRaw(r, bson.M{
		"first_name": "Jane",
		"age":        int32(25),
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.FirstName != "Jane" {
		t.Errorf("FirstName: got %q, want Jane", decoded.FirstName)
	}
	if decoded.Age != 25 {
		t.Errorf("Age: got %d, want 25", decoded.Age)
	}
}

// ─────────────────────────────────────────────
// 7. 各种数据类型 Round-Trip 测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_RoundTrip_AllTypes(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		S   string
		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64
		U   uint
		F32 float32
		F64 float64
		B   bool
		BS  []byte
		Arr []int
		M   bson.M
	}

	original := Doc{
		S:   "hello",
		I:   42,
		I8:  8,
		I16: 16,
		I32: 32,
		I64: 64,
		U:   100,
		F32: 3.14,
		F64: 2.718,
		B:   true,
		BS:  []byte{0x01, 0x02, 0x03},
		Arr: []int{1, 2, 3},
		M:   bson.M{"key": "value"},
	}

	var decoded Doc
	if err := roundTrip(r, original, &decoded); err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}

	if decoded.S != original.S {
		t.Errorf("S: got %q, want %q", decoded.S, original.S)
	}
	if decoded.I != original.I {
		t.Errorf("I: got %d, want %d", decoded.I, original.I)
	}
	if decoded.I64 != original.I64 {
		t.Errorf("I64: got %d, want %d", decoded.I64, original.I64)
	}
	if decoded.U != original.U {
		t.Errorf("U: got %d, want %d", decoded.U, original.U)
	}
	if decoded.F32 != original.F32 {
		t.Errorf("F32: got %f, want %f", decoded.F32, original.F32)
	}
	if decoded.F64 != original.F64 {
		t.Errorf("F64: got %f, want %f", decoded.F64, original.F64)
	}
	if decoded.B != original.B {
		t.Errorf("B: got %v, want %v", decoded.B, original.B)
	}
	if !bytes.Equal(decoded.BS, original.BS) {
		t.Errorf("BS: got %v, want %v", decoded.BS, original.BS)
	}
	if !reflect.DeepEqual(decoded.Arr, original.Arr) {
		t.Errorf("Arr: got %v, want %v", decoded.Arr, original.Arr)
	}
	if !reflect.DeepEqual(decoded.M, original.M) {
		t.Errorf("M: got %v, want %v", decoded.M, original.M)
	}
}

// ─────────────────────────────────────────────
// 8. 接口字段解码测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Decode_InterfaceField(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name  string
		Value interface{}
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":  "test",
		"Value": int32(42),
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "test" {
		t.Errorf("Name: got %q, want test", decoded.Name)
	}
	if decoded.Value != int32(42) {
		t.Errorf("Value: got %v (%T), want 42 (int32)", decoded.Value, decoded.Value)
	}
}

func TestPreserveStructCodec_Decode_InterfaceFieldString(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value interface{}
	}

	raw, err := encodeRaw(r, bson.M{"Value": "hello"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Value != "hello" {
		t.Errorf("Value: got %v (%T), want hello (string)", decoded.Value, decoded.Value)
	}
}

// ─────────────────────────────────────────────
// 9. 错误路径测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Decode_WrongKind(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	raw, err := encodeRaw(r, bson.M{"key": "value"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var s string
	err = decodeRaw(r, raw, &s)
	if err == nil {
		t.Error("expected error decoding document into string, got nil")
	}
}

func TestPreserveStructCodec_Decode_NonSettableValue(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
	}

	raw, err := encodeRaw(r, bson.M{"Name": "test"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	d := Doc{}
	err = decodeRaw(r, raw, d)
	if err == nil {
		t.Fatalf("decode with value (not pointer) should returned: `argument to Decode must be a pointer or a map, but got {}`")
	}
}

// ─────────────────────────────────────────────
// 10. 缓存测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_CacheWorks(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
		Age  int
	}

	d := Doc{Name: "Alice", Age: 30}

	raw1, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("first encode failed: %v", err)
	}

	raw2, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("second encode failed: %v", err)
	}

	if !bytes.Equal(raw1, raw2) {
		t.Error("cached encode should produce identical output")
	}
}

func TestPreserveStructCodec_CacheIndependentInstances(t *testing.T) {
	r1 := GetPreserveFieldRegistry(nil)
	r2 := GetPreserveFieldRegistry(nil)

	if r1 == r2 {
		t.Error("GetPreserveFieldRegistry should return independent instances")
	}

	type Doc struct {
		Name string
	}

	d := Doc{Name: "test"}

	raw1, err := encodeRaw(r1, d)
	if err != nil {
		t.Fatalf("r1 encode failed: %v", err)
	}
	raw2, err := encodeRaw(r2, d)
	if err != nil {
		t.Fatalf("r2 encode failed: %v", err)
	}

	if !bytes.Equal(raw1, raw2) {
		t.Error("independent instances should produce same output")
	}
}

// ─────────────────────────────────────────────
// 11. 深嵌套 + inline 组合测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_DeepNested(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Level3 struct {
		Val3 string
	}
	type Level2 struct {
		Val2   string
		Level3 Level3
	}
	type Level1 struct {
		Val1   string
		Level2 Level2
	}
	type Doc struct {
		Name   string
		Level1 Level1
	}

	d := Doc{
		Name: "root",
		Level1: Level1{
			Val1: "v1",
			Level2: Level2{
				Val2: "v2",
				Level3: Level3{
					Val3: "v3",
				},
			},
		},
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "root" {
		t.Errorf("Name: got %v, want root", result["Name"])
	}

	lvl1, ok := result["Level1"].(bson.M)
	if !ok {
		t.Fatalf("Level1 should be a document")
	}
	if lvl1["Val1"] != "v1" {
		t.Errorf("Level1.Val1: got %v, want v1", lvl1["Val1"])
	}

	lvl2, ok := lvl1["Level2"].(bson.M)
	if !ok {
		t.Fatalf("Level2 should be a document")
	}
	if lvl2["Val2"] != "v2" {
		t.Errorf("Level2.Val2: got %v, want v2", lvl2["Val2"])
	}

	lvl3, ok := lvl2["Level3"].(bson.M)
	if !ok {
		t.Fatalf("Level3 should be a document")
	}
	if lvl3["Val3"] != "v3" {
		t.Errorf("Level3.Val3: got %v, want v3", lvl3["Val3"])
	}
}

func TestPreserveStructCodec_Decode_DeepNestedWithPointer(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Level3 struct {
		Val3 string
	}
	type Level2 struct {
		Val2  string
		Level *Level3
	}
	type Level1 struct {
		Val1  string
		Level Level2
	}
	type Doc struct {
		Name  string
		Level Level1
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": "root",
		"Level": bson.M{
			"Val1": "v1",
			"Level": bson.M{
				"Val2": "v2",
				"Level": bson.M{
					"Val3": "v3",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "root" {
		t.Errorf("Name: got %q, want root", decoded.Name)
	}
	if decoded.Level.Val1 != "v1" {
		t.Errorf("Level1.Val1: got %q, want v1", decoded.Level.Val1)
	}
	if decoded.Level.Level.Val2 != "v2" {
		t.Errorf("Level2.Val2: got %q, want v2", decoded.Level.Level.Val2)
	}
	if decoded.Level.Level.Level == nil {
		t.Fatal("Level3 pointer should be auto-allocated")
	}
	if decoded.Level.Level.Level.Val3 != "v3" {
		t.Errorf("Level3.Val3: got %q, want v3", decoded.Level.Level.Level.Val3)
	}
}

// ─────────────────────────────────────────────
// 13. 混合 inline + 普通字段 + map 的复杂场景
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_ComplexMix(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Meta struct {
		CreatedAt string
		UpdatedAt string
	}
	type Doc struct {
		Meta   `bson:",inline"`
		Name   string
		Tags   []string
		Extras map[string]interface{} `bson:",inline"`
	}

	d := Doc{
		Meta: Meta{
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-06-01",
		},
		Name: "Complex",
		Tags: []string{"go", "mongo"},
		Extras: map[string]interface{}{
			"Priority": int32(1),
			"Active":   true,
		},
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["CreatedAt"] != "2024-01-01" {
		t.Errorf("CreatedAt: got %v, want 2024-01-01", result["CreatedAt"])
	}
	if result["UpdatedAt"] != "2024-06-01" {
		t.Errorf("UpdatedAt: got %v, want 2024-06-01", result["UpdatedAt"])
	}
	if result["Name"] != "Complex" {
		t.Errorf("Name: got %v, want Complex", result["Name"])
	}

	tags, ok := result["Tags"].(bson.A)
	if !ok {
		t.Fatalf("Tags should be array, got %T", result["Tags"])
	}
	if len(tags) != 2 || tags[0] != "go" || tags[1] != "mongo" {
		t.Errorf("Tags: got %v, want [go mongo]", tags)
	}

	if result["Priority"] != int32(1) {
		t.Errorf("Priority: got %v, want 1", result["Priority"])
	}
	if result["Active"] != true {
		t.Errorf("Active: got %v, want true", result["Active"])
	}
}

func TestPreserveStructCodec_Decode_ComplexMix(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Meta struct {
		CreatedAt string
		UpdatedAt string
	}
	type Doc struct {
		Meta   `bson:",inline"`
		Name   string
		Tags   []string
		Extras map[string]interface{} `bson:",inline"`
	}

	raw, err := encodeRaw(r, bson.M{
		"CreatedAt": "2024-01-01",
		"UpdatedAt": "2024-06-01",
		"Name":      "Complex",
		"Tags":      bson.A{"go", "mongo"},
		"Priority":  int32(1),
		"Active":    true,
		"Score":     99.5,
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.CreatedAt != "2024-01-01" {
		t.Errorf("CreatedAt: got %q, want 2024-01-01", decoded.CreatedAt)
	}
	if decoded.UpdatedAt != "2024-06-01" {
		t.Errorf("UpdatedAt: got %q, want 2024-06-01", decoded.UpdatedAt)
	}
	if decoded.Name != "Complex" {
		t.Errorf("Name: got %q, want Complex", decoded.Name)
	}
	if !reflect.DeepEqual(decoded.Tags, []string{"go", "mongo"}) {
		t.Errorf("Tags: got %v, want [go mongo]", decoded.Tags)
	}
	if decoded.Extras == nil {
		t.Fatal("Extras map should be auto-created")
	}
	if decoded.Extras["Priority"] != int32(1) {
		t.Errorf("Extras[Priority]: got %v, want 1", decoded.Extras["Priority"])
	}
	if decoded.Extras["Active"] != true {
		t.Errorf("Extras[Active]: got %v, want true", decoded.Extras["Active"])
	}
	if decoded.Extras["Score"] != 99.5 {
		t.Errorf("Extras[Score]: got %v, want 99.5", decoded.Extras["Score"])
	}
}

// ─────────────────────────────────────────────
// 14. 切片和数组测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_RoundTrip_Slices(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Ints    []int
		Strings []string
		Floats  []float64
		Bytes   []byte
	}

	original := Doc{
		Ints:    []int{1, 2, 3, 4, 5},
		Strings: []string{"a", "b", "c"},
		Floats:  []float64{1.1, 2.2, 3.3},
		Bytes:   []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}

	var decoded Doc
	if err := roundTrip(r, original, &decoded); err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}

	if !reflect.DeepEqual(decoded.Ints, original.Ints) {
		t.Errorf("Ints: got %v, want %v", decoded.Ints, original.Ints)
	}
	if !reflect.DeepEqual(decoded.Strings, original.Strings) {
		t.Errorf("Strings: got %v, want %v", decoded.Strings, original.Strings)
	}
	if !reflect.DeepEqual(decoded.Floats, original.Floats) {
		t.Errorf("Floats: got %v, want %v", decoded.Floats, original.Floats)
	}
	if !bytes.Equal(decoded.Bytes, original.Bytes) {
		t.Errorf("Bytes: got %v, want %v", decoded.Bytes, original.Bytes)
	}
}

// ─────────────────────────────────────────────
// 15. 嵌套指针链解码测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Decode_NestedPointerChain(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Level3 struct {
		Data string
	}
	type Level2 struct {
		L3 *Level3
	}
	type Level1 struct {
		L2 *Level2
	}
	type Doc struct {
		L1 *Level1
	}

	raw, err := encodeRaw(r, bson.M{
		"L1": bson.M{
			"L2": bson.M{
				"L3": bson.M{
					"Data": "deep",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.L1 == nil {
		t.Fatal("L1 should be auto-allocated")
	}
	if decoded.L1.L2 == nil {
		t.Fatal("L2 should be auto-allocated")
	}
	if decoded.L1.L2.L3 == nil {
		t.Fatal("L3 should be auto-allocated")
	}
	if decoded.L1.L2.L3.Data != "deep" {
		t.Errorf("L3.Data: got %q, want deep", decoded.L1.L2.L3.Data)
	}
}

// ─────────────────────────────────────────────
// 16. Empty struct 测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_EmptyStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Empty struct{}
	type Doc struct {
		Name  string
		Empty Empty
	}

	d := Doc{Name: "test", Empty: Empty{}}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "test" {
		t.Errorf("Name: got %v, want test", result["Name"])
	}

	empty, ok := result["Empty"].(bson.M)
	if !ok {
		t.Fatalf("Empty should be a document, got %T", result["Empty"])
	}
	if len(empty) != 0 {
		t.Errorf("Empty struct should be empty document, got %v", empty)
	}
}

func TestPreserveStructCodec_Decode_EmptyStruct(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Empty struct{}
	type Doc struct {
		Name  string
		Empty Empty
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":  "test",
		"Empty": bson.M{},
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "test" {
		t.Errorf("Name: got %q, want test", decoded.Name)
	}
}

// ─────────────────────────────────────────────
// 17. 特殊属性的实际行为探测
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitEmpty_TagLevel_NonZero(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string `bson:",omitempty"`
		Age  int    `bson:",omitempty"`
	}

	d := Doc{Name: "Alice", Age: 30}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("non-zero Name should be present, got %v", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("non-zero Age should be present, got %v", result["Age"])
	}
}

func TestPreserveStructCodec_OmitEmpty_TagLevel_Bool(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Active bool `bson:",omitempty"`
	}

	d := Doc{} // Active=false

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Active"]; ok {
		t.Errorf("Tag omitempty NOT working: zero bool present = %v", result["Active"])
	}
}

func TestPreserveStructCodec_OmitEmpty_TagLevel_Slice(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Tags []string `bson:",omitempty"`
	}

	d := Doc{} // Tags=nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Tags"]; ok {
		t.Errorf("Tag omitempty NOT working: nil slice present = %v", result["Tags"])
	}
}

func TestPreserveStructCodec_OmitEmpty_GlobalOverridesTag(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string // 没有 omitempty tag
		Age  int
	}

	d := Doc{Name: "Alice"} // Age=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if _, ok := result["Age"]; ok {
		t.Errorf("Global OmitEmpty should omit zero Age, got %v", result["Age"])
	}
}

// === minsize 组 ===

func TestPreserveStructCodec_MinSize_Int64_Small(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 `bson:",minsize"`
	}

	d := Doc{Value: 42}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]

	switch val.(type) {
	case int32:
		// ok
	case int64:
		t.Errorf("Minsize NOT working: int64(42) → int64 (no truncation)")
	default:
		t.Errorf("Minsize: encoded as unexpected type %T", val)
	}
}

func TestPreserveStructCodec_MinSize_Int64_Large(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 `bson:"Value,minsize"`
	}

	d := Doc{Value: 1 << 40}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]

	switch val.(type) {
	case int64:
		// ok
	case int32:
		t.Errorf("Minsize NOT working correctly: large value truncated to int32")
	default:
		t.Errorf("Minsize: encoded as unexpected type %T", val)
	}
}

func TestPreserveStructCodec_MinSize_Uint64_Small(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value uint64 `bson:"Value,minsize"`
	}

	d := Doc{Value: 100}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]

	switch val.(type) {
	case int32:
		// ok
	case int64:
		t.Errorf("Minsize NOT working: uint64(100) → int64 (no truncation)")
	default:
		t.Errorf("Minsize: encoded as unexpected type %T", val)
	}
}

func TestPreserveStructCodec_MinSize_ControlGroup(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 // 无 minsize tag
	}

	d := Doc{Value: 42}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]

	if _, ok := val.(int64); !ok {
		t.Errorf("Control group wrong: expected int64, got %T", val)
	}
}

// === 组合测试 ===

func TestPreserveStructCodec_OmitEmpty_And_MinSize(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		A int64 `bson:",omitempty,minsize"`
		B int64 `bson:",omitempty,minsize"`
	}

	d := Doc{} // A=0, B=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(result) != 0 {
		t.Logf("omitempty+minsize: got %v", result)
	}

	// 非零值
	d2 := Doc{A: 42}
	raw2, err := encodeRaw(r, d2)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result2 bson.M
	if err := bsonUnmarshal(raw2, &result2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

// === ParseStructTags 解析验证 ===

func TestPreserveStructCodec_ParseStructTags_OmitEmpty(t *testing.T) {
	st, err := x.ParseStructTags(reflect.StructField{
		Name: "Test",
		Tag:  `bson:"test,omitempty"`,
	})
	if err != nil {
		t.Fatalf("ParseStructTags failed: %v", err)
	}

	if !st.OmitEmpty {
		t.Error("ParseStructTags did not parse omitempty")
	}
}

func TestPreserveStructCodec_ParseStructTags_MinSize(t *testing.T) {
	st, err := x.ParseStructTags(reflect.StructField{
		Name: "Test",
		Tag:  `bson:"test,minsize"`,
	})
	if err != nil {
		t.Fatalf("ParseStructTags failed: %v", err)
	}

	if !st.MinSize {
		t.Error("ParseStructTags did not parse minsize")
	}
}

func TestPreserveStructCodec_ParseStructTags_Truncate(t *testing.T) {
	st, err := x.ParseStructTags(reflect.StructField{
		Name: "Test",
		Tag:  `bson:"test,truncate"`,
	})
	if err != nil {
		t.Fatalf("ParseStructTags failed: %v", err)
	}

	if !st.Truncate {
		t.Error("ParseStructTags did not parse truncate")
	}
}

// ─────────────────────────────────────────────
// 12b. OmitZeroStruct 第二种情况（tag omitempty，不设全局）
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_OmitZeroStruct_TagOmitEmpty(t *testing.T) {
	// 情况2: 只给 struct 字段加 omitempty tag，同时设置 OmitZeroStruct
	opts := &options.BSONOptions{OmitZeroStruct: true}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Inner Inner `bson:",omitempty"`
	}

	d := Doc{Name: "Alice"} // Inner 是零值

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["Inner"]; ok {
		t.Errorf("Inner should be omitted (tag omitempty + OmitZeroStruct), got %v", result["Inner"])
	}
}

// ─────────────────────────────────────────────
// 17b. 全局 BSONOptions 的 omitempty / minsize 测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_GlobalOmitEmpty_AllFields(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string
		Age  int
		City string
	}

	d := Doc{Name: "Alice"} // Age=0, City=""

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted (global OmitEmpty), got %v", result["Age"])
	}
	if _, ok := result["City"]; ok {
		t.Errorf("City should be omitted (global OmitEmpty), got %v", result["City"])
	}
}

func TestPreserveStructCodec_GlobalIntMinSize_Int64Small(t *testing.T) {
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Value int64
	}

	d := Doc{Value: 42}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	switch val.(type) {
	case int32:
		// ✅ 全局 IntMinSize 生效
	case int64:
		t.Errorf("Global IntMinSize NOT working: int64(42) → int64 (no truncation)")
	default:
		t.Errorf("Unexpected type: %T", val)
	}
}

func TestPreserveStructCodec_GlobalIntMinSize_Int64Large(t *testing.T) {
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Value int64
	}

	d := Doc{Value: 1 << 40}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	switch val.(type) {
	case int64:
		// ✅ 大值保持 int64
	case int32:
		t.Errorf("Global IntMinSize incorrectly truncated large value to int32")
	default:
		t.Errorf("Unexpected type: %T", val)
	}
}

func TestPreserveStructCodec_GlobalIntMinSize_Uint64Small(t *testing.T) {
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Value uint64
	}

	d := Doc{Value: 100}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	switch val.(type) {
	case int32:
		// ✅ 全局 IntMinSize 生效
	case int64:
		t.Errorf("Global IntMinSize NOT working: uint64(100) → int64 (no truncation)")
	default:
		t.Errorf("Unexpected type: %T", val)
	}
}

// ─────────────────────────────────────────────
// 18. omitempty / minsize 传递性测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_MinSize_InnerTagWorksIndependently(t *testing.T) {
	// 验证：内层字段自己有 minsize tag → 独立生效（这不是传递性）
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int64 `bson:",minsize"` // 自己有 tag
	}
	type Doc struct {
		Inner Inner `bson:",minsize"`
	}

	d := Doc{Inner: Inner{Value: 42}}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	inner := result["Inner"].(bson.M)
	innerVal := inner["Value"]
	if _, ok := innerVal.(int32); !ok {
		t.Errorf("Inner.Value should be int32 (own tag), got %T", innerVal)
	}
}

func TestPreserveStructCodec_OmitEmpty_InnerTagWorksIndependently(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int    `bson:",omitempty"`
		Name  string `bson:",omitempty"`
	}
	type Doc struct {
		Inner Inner `bson:",omitempty"`
	}

	d := Doc{Inner: Inner{Value: 42}} // Name 是零值

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	inner := result["Inner"].(bson.M)
	if _, ok := inner["Name"]; ok {
		t.Errorf("Inner.Name should be omitted (own omitempty tag), got %v", inner["Name"])
	}
}

// ─────────────────────────────────────────────
// 标量字段 tag omitempty 基础覆盖
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_String(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string `bson:",omitempty"`
		Age  int    `bson:",omitempty"`
	}

	d := Doc{} // Name="", Age=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Name"]; ok {
		t.Errorf("Name should be omitted (zero string + tag omitempty), got %v", result["Name"])
	}
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted (zero int + tag omitempty), got %v", result["Age"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_Bool(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Active bool `bson:",omitempty"`
		Admin  bool `bson:",omitempty"`
	}

	d := Doc{} // Active=false, Admin=false

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Active"]; ok {
		t.Errorf("Active should be omitted (zero bool + tag omitempty), got %v", result["Active"])
	}
	if _, ok := result["Admin"]; ok {
		t.Errorf("Admin should be omitted (zero bool + tag omitempty), got %v", result["Admin"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_Float(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Score float64 `bson:",omitempty"`
		Rate  float32 `bson:",omitempty"`
	}

	d := Doc{} // Score=0, Rate=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Score"]; ok {
		t.Errorf("Score should be omitted (zero float64 + tag omitempty), got %v", result["Score"])
	}
	if _, ok := result["Rate"]; ok {
		t.Errorf("Rate should be omitted (zero float32 + tag omitempty), got %v", result["Rate"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_Slice(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Tags []string `bson:",omitempty"`
		Nums []int    `bson:",omitempty"`
	}

	d := Doc{} // Tags=nil, Nums=nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Tags"]; ok {
		t.Errorf("Tags should be omitted (nil slice + tag omitempty), got %v", result["Tags"])
	}
	if _, ok := result["Nums"]; ok {
		t.Errorf("Nums should be omitted (nil slice + tag omitempty), got %v", result["Nums"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_Map(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Meta map[string]string `bson:",omitempty"`
	}

	d := Doc{} // Meta=nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Meta"]; ok {
		t.Errorf("Meta should be omitted (nil map + tag omitempty), got %v", result["Meta"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_Pointer(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Inner struct{ Value int }
	type Doc struct {
		Ref *Inner `bson:",omitempty"`
	}

	d := Doc{} // Ref=nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Ref"]; ok {
		t.Errorf("Ref should be omitted (nil pointer + tag omitempty), got %v", result["Ref"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Scalar_NonZeroKept(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name   string   `bson:",omitempty"`
		Age    int      `bson:",omitempty"`
		Score  float64  `bson:",omitempty"`
		Active bool     `bson:",omitempty"`
		Tags   []string `bson:",omitempty"`
	}

	d := Doc{
		Name: "Alice", Age: 30, Score: 95.5, Active: true,
		Tags: []string{"go"},
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age should be present, got %v", result["Age"])
	}
	if result["Score"] != 95.5 {
		t.Errorf("Score should be present, got %v", result["Score"])
	}
	if result["Active"] != true {
		t.Errorf("Active should be present, got %v", result["Active"])
	}
	tags, ok := result["Tags"].(bson.A)
	if !ok || len(tags) != 1 || tags[0] != "go" {
		t.Errorf("Tags should be present, got %v", result["Tags"])
	}
}

func TestPreserveStructCodec_Truncate_Tag_DoubleToInt(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 3.99})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// 3.99 截断为 3
	if decoded.Value != 3 {
		t.Errorf("Value: got %d, want 3 (truncated from 3.99)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_Tag_DoubleToInt64(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 `bson:"Value,truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 42.7})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Value != 42 {
		t.Errorf("Value: got %d, want 42 (truncated from 42.7)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_Tag_DoubleToUint(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value uint `bson:"Value,truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 100.4})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Value != 100 {
		t.Errorf("Value: got %d, want 100 (truncated from 100.4)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_Tag_DoubleToFloat32(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float32 `bson:"Value,truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 3.14159})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// float32 精度截断
	if math.Abs(float64(decoded.Value)-3.14159) > 0.00001 {
		t.Errorf("Value: got %f, want ~3.14159 (float32 precision)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_NoTag_DoubleToInt_Error(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int // 没有 truncate tag
	}

	raw, err := encodeRaw(r, bson.M{"Value": 3.99})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected error decoding double into int without truncate tag, got nil")
	}
}

func TestPreserveStructCodec_Truncate_Global_AllowTruncatingDoubles(t *testing.T) {
	opts := &options.BSONOptions{AllowTruncatingDoubles: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		A int
		B int64
		C uint
	}

	raw, err := encodeRaw(r, bson.M{"A": 1.9, "B": 2.1, "C": 3.7})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.A != 1 || decoded.B != 2 || decoded.C != 3 {
		t.Errorf("A=%d, B=%d, C=%d, want 1, 2, 3", decoded.A, decoded.B, decoded.C)
	}
}

func TestPreserveStructCodec_Truncate_And_OmitEmpty(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int `bson:",omitempty,truncate"`
	}

	// 零值 BSON double 0.0 → 截断为 0 → omitempty 判断为零 → 省略
	raw, err := encodeRaw(r, bson.M{"Value": 0.0})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
}

// ─────────────────────────────────────────────
// omitempty minsize truncate 传递性差异记录
// 官方 driver：外层这三个属性 会传递给内层 struct 的所有字段
// 本 codec：这三个属性 只在当前字段生效，不传递
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Truncate_NoTransitivity_InnerHasNoTag(t *testing.T) {
	// 验证：外层字段 truncate 不传递给内层
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int // 没有 truncate tag
	}
	type Doc struct {
		Inner Inner `bson:",truncate"` // 外层有 truncate
		A     int   `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{
		"Inner": bson.M{"Value": 5.9},
		"A":     2.3,
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err == nil {
		t.Errorf("truncate should Not has transitivity, Inner.Value=%d", decoded.Inner.Value)
	}
	t.Logf("✅ confirmed: truncate has NO transitivity in this codec")
}

func TestPreserveStructCodec_OmitEmpty_NoTransitivity_InnerHasNoTag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int    // 没有 omitempty
		Name  string // 没有 omitempty
	}
	type Doc struct {
		Inner Inner `bson:",omitempty"`
		Age   int   `bson:",omitempty"`
	}

	d := Doc{Inner: Inner{Value: 42}, Age: 0}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Age 被省略（自己有 omitempty）
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted, got %v", result["Age"])
	}

	// Inner 存在
	inner, ok := result["Inner"].(bson.M)
	if !ok {
		t.Fatalf("Inner should exist, got %T", result["Inner"])
	}

	// Inner.Value 存在（非零）
	if inner["Value"] != int32(42) {
		t.Errorf("Inner.Value: got %v, want 42", inner["Value"])
	}

	// Inner.Name 被编码为空字符串（没有 omitempty tag，不传递）
	if inner["Name"] != "" {
		t.Errorf("Inner.Name should be empty string (no omitempty tag), got %v", inner["Name"])
	}
	t.Log("✅ confirmed: omitempty has NO transitivity in this codec")
}

func TestPreserveStructCodec_MinSize_NoTransitivity_InnerHasNoTag(t *testing.T) {
	// 验证：外层有 minsize，内层字段【没有】minsize tag → 内层保持 int64
	r := GetPreserveFieldRegistry(nil)

	type Inner struct {
		Value int64 // 没有 minsize tag
	}
	type Doc struct {
		Inner Inner `bson:",minsize"` // 外层有 minsize
		Count int64 `bson:",minsize"`
	}

	d := Doc{Inner: Inner{Value: 42}, Count: 100}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Count 自己有 tag → int32
	count := result["Count"]
	if _, ok := count.(int32); !ok {
		t.Errorf("Count should be int32 (own tag), got %T", count)
	}

	// Inner.Value 没有 tag → 保持 int64（没有传递性）
	inner := result["Inner"].(bson.M)
	innerVal := inner["Value"]
	if _, ok := innerVal.(int64); !ok {
		t.Errorf("Inner.Value should be int64 (no transitivity), got %T", innerVal)
	}
	t.Log("✅ confirmed: minsize has NO transitivity in this codec")
}

// ─────────────────────────────────────────────
// 19. 全局 + Tag 组合覆盖补充
// ─────────────────────────────────────────────

// --- omitempty 组合 ---

func TestPreserveStructCodec_OmitEmpty_GlobalAndTag_BothZero(t *testing.T) {
	// 全局 OmitEmpty + tag omitempty，零值 → 省略
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		A string `bson:",omitempty"` // 有 tag
		B int    // 无 tag，靠全局
	}

	d := Doc{} // A="", B=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["A"]; ok {
		t.Errorf("A should be omitted (tag omitempty), got %v", result["A"])
	}
	if _, ok := result["B"]; ok {
		t.Errorf("B should be omitted (global OmitEmpty), got %v", result["B"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Global_NonZeroKept(t *testing.T) {
	// 全局 OmitEmpty，非零值字段不应被省略
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name   string
		Age    int
		Score  float64
		Active bool
	}

	d := Doc{Name: "Alice", Age: 30, Score: 1.5, Active: true}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age should be present, got %v", result["Age"])
	}
	if result["Score"] != 1.5 {
		t.Errorf("Score should be present, got %v", result["Score"])
	}
	if result["Active"] != true {
		t.Errorf("Active should be present, got %v", result["Active"])
	}
}

// --- minsize 组合 ---

func TestPreserveStructCodec_MinSize_GlobalAndTag_BothPresent(t *testing.T) {
	// 全局 IntMinSize=true + tag minsize，小值 → int32（OR 关系，两者任一为 true 即可）
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		A int64 `bson:",minsize"` // tag + 全局
		B int64 // 只有全局
	}

	d := Doc{A: 42, B: 100}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["A"].(int32); !ok {
		t.Errorf("A should be int32 (tag+global), got %T", result["A"])
	}
	if _, ok := result["B"].(int32); !ok {
		t.Errorf("B should be int32 (global only), got %T", result["B"])
	}
}

func TestPreserveStructCodec_MinSize_Global_LargeUint64_WithinInt64(t *testing.T) {
	// 全局 IntMinSize，uint64 大值但在 int64 范围内 → int64
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Value uint64
	}

	d := Doc{Value: 1 << 40} // > MaxInt32, < MaxInt64

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	if _, ok := val.(int64); !ok {
		t.Errorf("uint64 large value should be int64, got %T", val)
	}
}

func TestPreserveStructCodec_Truncate_Tag_DoubleToFloat64(t *testing.T) {
	// truncate tag 对 float64 目标字段——不应该截断，直接赋值
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float64 `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 3.14159})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// float64 → float64，truncate 不改变精度
	if decoded.Value != 3.14159 {
		t.Errorf("Value: got %f, want 3.14159", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_Tag_NegativeDoubleToInt(t *testing.T) {
	// 负数 double → int（有 truncate tag）
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": -3.7})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// -3.7 截断为 -3
	if decoded.Value != -3 {
		t.Errorf("Value: got %d, want -3 (truncated from -3.7)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_Global_NoTag_Works(t *testing.T) {
	// 全局 AllowTruncatingDoubles，字段没有 truncate tag → 也应该截断
	opts := &options.BSONOptions{AllowTruncatingDoubles: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		A int // 没有 truncate tag
		B int64
	}

	raw, err := encodeRaw(r, bson.M{"A": 5.6, "B": 9.1})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.A != 5 {
		t.Errorf("A: got %d, want 5 (global truncate), got %v", decoded.A, decoded.A)
	}
	if decoded.B != 9 {
		t.Errorf("B: got %d, want 9 (global truncate)", decoded.B)
	}
}

// --- omitempty + minsize 全局同时开启 ---

func TestPreserveStructCodec_Global_OmitEmptyAndMinSize_Combined(t *testing.T) {
	// 同时开启 OmitEmpty + IntMinSize
	opts := &options.BSONOptions{
		OmitEmpty:  true,
		IntMinSize: true,
	}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name  string
		Age   int
		Count int64
		Total uint64
	}

	d := Doc{Name: "Alice", Age: 0, Count: 42, Total: 100}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Name 非零 → 保留
	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}

	// Age 零值 → 省略（全局 OmitEmpty）
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted, got %v", result["Age"])
	}

	// Count 非零 + minsize → int32
	if _, ok := result["Count"].(int32); !ok {
		t.Errorf("Count should be int32 (minsize), got %T", result["Count"])
	}

	// Total 非零 + minsize → int32
	if _, ok := result["Total"].(int32); !ok {
		t.Errorf("Total should be int32 (minsize), got %T", result["Total"])
	}
}

// --- OmitZeroStruct 全局（无 tag）---

func TestPreserveStructCodec_OmitZeroStruct_Global_NoTag(t *testing.T) {
	// 全局 OmitZeroStruct=true，struct 字段没有 omitempty tag
	opts := &options.BSONOptions{OmitZeroStruct: true, OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Inner Inner // 没有 omitempty tag
	}

	d := Doc{Name: "Alice"} // Inner 是零值

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}

	// Inner 是零值 struct，OmitZeroStruct=true → 应该被省略
	if _, ok := result["Inner"]; ok {
		t.Errorf("Inner should be omitted (global OmitZeroStruct), got %v", result["Inner"])
	}
}

func TestPreserveStructCodec_OmitZeroStruct_GlobalAndOmitEmpty_Combined(t *testing.T) {
	// 同时开启 OmitZeroStruct + 全局 OmitEmpty
	opts := &options.BSONOptions{
		OmitZeroStruct: true,
		OmitEmpty:      true,
	}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Age   int
		Inner Inner
	}

	d := Doc{Name: "Alice"} // Age=0, Inner={}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Name 非零 → 保留
	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}

	// Age 零值 → 省略（全局 OmitEmpty）
	if _, ok := result["Age"]; ok {
		t.Errorf("Age should be omitted, got %v", result["Age"])
	}

	// Inner 零值 struct → 省略（全局 OmitZeroStruct）
	if _, ok := result["Inner"]; ok {
		t.Errorf("Inner should be omitted (OmitZeroStruct), got %v", result["Inner"])
	}
}

// ─────────────────────────────────────────────
// 20. P0 — 潜在 bug 暴露：uint64 溢出 + truncate 溢出
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_Uint64Overflow_NoMinSize(t *testing.T) {
	// uint64 > MaxInt64，无 minsize tag，无全局 IntMinSize
	// integerEncodeValue 里 u64 > MaxInt64 应返回 overflow 错误
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value uint64
	}

	d := Doc{Value: uint64(1<<63) + 1} // 大于 MaxInt64

	_, err := encodeRaw(r, d)
	if err == nil {
		t.Error("expected overflow error for uint64 > MaxInt64 without minsize, got nil")
	}
}

func TestPreserveStructCodec_Encode_Uint64Overflow_WithMinSize(t *testing.T) {
	// uint64 > MaxInt64，有 minsize tag
	// integerEncodeValue 里同样应返回 overflow 错误
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value uint64 `bson:",minsize"`
	}

	d := Doc{Value: uint64(1<<63) + 1}

	_, err := encodeRaw(r, d)
	if err == nil {
		t.Error("expected overflow error for uint64 > MaxInt64 with minsize tag, got nil")
	}
}

func TestPreserveStructCodec_Encode_Uint64Overflow_GlobalMinSize(t *testing.T) {
	// uint64 > MaxInt64，全局 IntMinSize=true
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Value uint64
	}

	d := Doc{Value: uint64(1<<63) + 1}

	_, err := encodeRaw(r, d)
	if err == nil {
		t.Error("expected overflow error for uint64 > MaxInt64 with global IntMinSize, got nil")
	}
}

func TestPreserveStructCodec_Truncate_DoubleOverflowToInt64(t *testing.T) {
	// double 超出 int64 范围 → int64 字段 + truncate tag → 应报错
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": 1e100})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected overflow error decoding 1e100 into int64 with truncate, got nil")
	}
}

func TestPreserveStructCodec_Truncate_NegativeDoubleToInt8_Overflow(t *testing.T) {
	// -200.0 → int8 + truncate → 应报溢出错误
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int8 `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": -200.0})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected overflow error decoding -200 into int8 with truncate, got nil")
	}
}

func TestPreserveStructCodec_Truncate_NegativeDoubleToInt16_Overflow(t *testing.T) {
	// -40000.0 → int16 + truncate → 应报溢出错误
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int16 `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": -40000.0})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected overflow error decoding -40000 into int16 with truncate, got nil")
	}
}

func TestPreserveStructCodec_Decode_LowercaseFallback_Mixed(t *testing.T) {
	// 混合：BSON 里有的 key 大写、有的小写 → 都能匹配
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name  string
		Age   int
		Email string `bson:"email"` // tag 指定了小写
	}

	raw, err := encodeRaw(r, bson.M{
		"Name":  "Bob",     // 大写 → 直接匹配
		"age":   int32(25), // 小写 → 不匹配
		"email": "b@x.com", // tag 匹配
	})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Name != "Bob" {
		t.Errorf("Name: got %q, want Bob", decoded.Name)
	}
	if decoded.Age != 0 {
		t.Errorf("Age: got %d, want 0", decoded.Age)
	}
	if decoded.Email != "b@x.com" {
		t.Errorf("Email: got %q, want b@x.com", decoded.Email)
	}
}

// ─────────────────────────────────────────────
// 22. P1 — time.Time + OmitZeroStruct / isEmpty
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitZeroStruct_TimeZero(t *testing.T) {
	opts := &options.BSONOptions{OmitZeroStruct: true, OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		T    time.Time
		Name string
	}

	d := Doc{Name: "test"} // T 是零值 time.Time{}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "test" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if _, ok := result["T"]; ok {
		t.Errorf("zero time.Time should be omitted (OmitZeroStruct), got %v", result["T"])
	}
}

func TestPreserveStructCodec_OmitZeroStruct_TimeNonZero(t *testing.T) {
	opts := &options.BSONOptions{OmitZeroStruct: true, OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		T time.Time
	}

	d := Doc{T: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["T"] == nil {
		t.Error("non-zero time.Time should NOT be omitted")
	}
}

// ─────────────────────────────────────────────
// 23. P1 — Zeroer 接口
// ─────────────────────────────────────────────

type zeroerTestStruct struct {
	Val int
}

func (z zeroerTestStruct) IsZero() bool {
	return z.Val == 0
}

func TestPreserveStructCodec_OmitEmpty_Zeroer_Zero(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Z zeroerTestStruct `bson:",omitempty"`
	}

	d := Doc{} // Z.Val=0 → IsZero()=true

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Z"]; ok {
		t.Errorf("Zeroer zero value should be omitted, got %v", result["Z"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Zeroer_NonZero(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Z zeroerTestStruct `bson:",omitempty"`
	}

	d := Doc{Z: zeroerTestStruct{Val: 42}} // IsZero()=false

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Z"]; !ok {
		t.Error("Zeroer non-zero value should be present")
	}
}

// ─────────────────────────────────────────────
// 24. P1 — encoder/decoder 不存在的错误路径
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_NoEncoder_Chan(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		F chan int // chan 没有注册的 encoder
	}

	d := Doc{F: make(chan int)}

	_, err := encodeRaw(r, d)
	if err == nil {
		t.Error("expected errNoEncoder for chan field, got nil")
	}
}

func TestPreserveStructCodec_Decode_NoDecoder_Chan(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		F chan int
	}

	raw, err := encodeRaw(r, bson.M{"F": "unexpected"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected errNoDecoder for chan field, got nil")
	}
}

// ─────────────────────────────────────────────
// 25. P1 — interface 字段 encode 时 IsNil 判断
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitEmpty_InterfaceNil(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		V interface{}
	}

	d := Doc{V: nil} // interface 是 nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["V"]; ok {
		t.Errorf("nil interface should be omitted (global OmitEmpty), got %v", result["V"])
	}
}

func TestPreserveStructCodec_OmitEmpty_InterfaceNonNil(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		V interface{}
	}

	d := Doc{V: "hello"} // interface 非 nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["V"] != "hello" {
		t.Errorf("non-nil interface should be present, got %v", result["V"])
	}
}

// ─────────────────────────────────────────────
// 26. P1 — array（非 slice）零值 omitempty
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitEmpty_Tag_Array(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Arr []int `bson:",omitempty"`
	}

	d := Doc{} // [] 零值

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Arr"]; ok {
		t.Errorf("zero array should be omitted, got %v", result["Arr"])
	}
}

func TestPreserveStructCodec_OmitEmpty_Tag_Array_NonZero(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Arr [3]int `bson:",omitempty"`
	}

	d := Doc{Arr: [3]int{1, 0, 0}} // 非零

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Arr"]; !ok {
		t.Error("non-zero array should be present")
	}
}

// ─────────────────────────────────────────────
// 27. P1 — truncate + 非 double 类型（processed=false 路径）
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Truncate_NonDouble_BsonInt32(t *testing.T) {
	// BSON 里是 int32，不是 double → truncate 不应介入，正常 decode
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": int32(42)})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Value != 42 {
		t.Errorf("Value: got %d, want 42 (non-double, normal decode)", decoded.Value)
	}
}

func TestPreserveStructCodec_Truncate_NonDouble_BsonString(t *testing.T) {
	// BSON 里是 string，不是 double → truncate 不介入，int 字段 decode string 应报错
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int `bson:",truncate"`
	}

	raw, err := encodeRaw(r, bson.M{"Value": "hello"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	err = decodeRaw(r, raw, &decoded)
	if err == nil {
		t.Error("expected error decoding string into int even with truncate tag")
	}
}

// ─────────────────────────────────────────────
// 28. P2 — 裸 int（非 int64）大值 encode
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_IntLarge_NoMinSize(t *testing.T) {
	// 裸 int 大值（> MaxInt32），无 minsize → 应编码为 int64
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int
	}

	// 用一个确定大于 MaxInt32 的值
	bigInt := int(1 << 40)
	if bigInt <= math.MaxInt32 {
		t.Skip("platform int is 32-bit, skipping")
	}

	d := Doc{Value: bigInt}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	if _, ok := val.(int64); !ok {
		t.Errorf("large int should encode as int64, got %T", val)
	}
}

// ─────────────────────────────────────────────
// 29. P2 — 嵌套 struct + OmitZeroStruct 非零值保留
// ─────────────────────────────────────────────

func TestPreserveStructCodec_OmitZeroStruct_NonZeroStruct_Preserved(t *testing.T) {
	opts := &options.BSONOptions{OmitZeroStruct: true}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Inner Inner
	}

	d := Doc{Inner: Inner{Value: 42}} // 非零 struct

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	inner, ok := result["Inner"].(bson.M)
	if !ok {
		t.Fatalf("Inner should exist, got %T", result["Inner"])
	}
	if inner["Value"] != int32(42) {
		t.Errorf("Inner.Value: got %v, want 42", inner["Value"])
	}
}

// ─────────────────────────────────────────────
// 21. inline + omitempty / minsize / truncate 组合
// ─────────────────────────────────────────────

// --- inline + omitempty tag ---

func TestPreserveStructCodec_Inline_OmitEmpty_Tag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID   string `bson:",omitempty"`
		Type string `bson:",omitempty"`
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	d := Doc{
		Embedded: Embedded{}, // ID="", Type=""
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if _, ok := result["ID"]; ok {
		t.Errorf("ID should be omitted (inline field + omitempty), got %v", result["ID"])
	}
	if _, ok := result["Type"]; ok {
		t.Errorf("Type should be omitted (inline field + omitempty), got %v", result["Type"])
	}
}

func TestPreserveStructCodec_Inline_OmitEmpty_Global(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Embedded struct {
		ID   string
		Type string
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	d := Doc{
		Embedded: Embedded{}, // ID="", Type=""
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if _, ok := result["ID"]; ok {
		t.Errorf("ID should be omitted (global OmitEmpty + inline), got %v", result["ID"])
	}
	if _, ok := result["Type"]; ok {
		t.Errorf("Type should be omitted (global OmitEmpty + inline), got %v", result["Type"])
	}
}

func TestPreserveStructCodec_Inline_OmitEmpty_NonZeroKept(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		ID   string `bson:",omitempty"`
		Type string `bson:",omitempty"`
	}
	type Doc struct {
		Embedded `bson:",inline"`
	}

	d := Doc{Embedded: Embedded{ID: "abc", Type: "user"}}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["ID"] != "abc" {
		t.Errorf("ID should be present, got %v", result["ID"])
	}
	if result["Type"] != "user" {
		t.Errorf("Type should be present, got %v", result["Type"])
	}
}

// --- inline + minsize ---

func TestPreserveStructCodec_Inline_MinSize_Tag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		Count int64  `bson:",minsize"`
		Total uint64 `bson:",minsize"`
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	d := Doc{
		Embedded: Embedded{Count: 42, Total: 100},
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Count"].(int32); !ok {
		t.Errorf("Count should be int32 (inline + minsize), got %T", result["Count"])
	}
	if _, ok := result["Total"].(int32); !ok {
		t.Errorf("Total should be int32 (inline + minsize), got %T", result["Total"])
	}
}

func TestPreserveStructCodec_Inline_MinSize_Global(t *testing.T) {
	opts := &options.BSONOptions{IntMinSize: true}
	r := GetPreserveFieldRegistry(opts)

	type Embedded struct {
		Count int64
		Total uint64
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	d := Doc{
		Embedded: Embedded{Count: 42, Total: 100},
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Count"].(int32); !ok {
		t.Errorf("Count should be int32 (global IntMinSize + inline), got %T", result["Count"])
	}
	if _, ok := result["Total"].(int32); !ok {
		t.Errorf("Total should be int32 (global IntMinSize + inline), got %T", result["Total"])
	}
}

func TestPreserveStructCodec_Inline_MinSize_LargeValue(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		Count int64 `bson:",minsize"`
	}
	type Doc struct {
		Embedded `bson:",inline"`
	}

	d := Doc{Embedded: Embedded{Count: 1 << 40}}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Count"].(int64); !ok {
		t.Errorf("Count should be int64 (large value + inline minsize), got %T", result["Count"])
	}
}

// --- inline + truncate (decode) ---

func TestPreserveStructCodec_Inline_Truncate_Tag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		Count int `bson:",truncate"`
	}
	type Doc struct {
		Embedded `bson:",inline"`
		Name     string
	}

	// 用 bson.M 构造 BSON，Count 是 double
	raw, err := encodeRaw(r, bson.M{"Count": 3.99, "Name": "Alice"})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Embedded.Count != 3 {
		t.Errorf("Count should be 3 (truncated from 3.99), got %d", decoded.Embedded.Count)
	}
	if decoded.Name != "Alice" {
		t.Errorf("Name should be Alice, got %q", decoded.Name)
	}
}

func TestPreserveStructCodec_Inline_Truncate_Global(t *testing.T) {
	opts := &options.BSONOptions{AllowTruncatingDoubles: true}
	r := GetPreserveFieldRegistry(opts)

	type Embedded struct {
		Count int
	}
	type Doc struct {
		Embedded `bson:",inline"`
	}

	raw, err := encodeRaw(r, bson.M{"Count": 7.3})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded Doc
	if err := decodeRaw(r, raw, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Embedded.Count != 7 {
		t.Errorf("Count should be 7 (global truncate + inline), got %d", decoded.Embedded.Count)
	}
}

// --- inline pointer + minsize ---

func TestPreserveStructCodec_InlinePointer_MinSize_Tag(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Embedded struct {
		Count int64 `bson:",minsize"`
	}
	type Doc struct {
		*Embedded `bson:",inline"`
		Name      string
	}

	d := Doc{
		Embedded: &Embedded{Count: 42},
		Name:     "Alice",
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Count"].(int32); !ok {
		t.Errorf("Count should be int32 (inline pointer + minsize), got %T", result["Count"])
	}
}

// --- inline map + 全局 OmitEmpty（验证不崩溃、行为正确）---

func TestPreserveStructCodec_InlineMap_GlobalOmitEmpty(t *testing.T) {
	opts := &options.BSONOptions{OmitEmpty: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string
		Ext  map[string]interface{} `bson:",inline"`
	}

	d := Doc{
		Name: "Alice",
		Ext: map[string]interface{}{
			"Custom": "value",
		},
	}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bsonUnmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if result["Custom"] != "value" {
		t.Errorf("Custom should be present from inline map, got %v", result["Custom"])
	}
}
