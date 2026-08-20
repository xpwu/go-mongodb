package client

import (
	"bytes"
	"github.com/xpwu/go-mongodb/x"
	"reflect"
	"strings"
	"testing"

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

func decodeRaw(r *bson.Registry, raw []byte, val interface{}) error {
	vr := bson.NewDocumentReader(bytes.NewReader(raw))
	dec := bson.NewDecoder(vr)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
		Name string `bson:"Name,omitempty"`
		Age  int    `bson:"Age,omitempty"`
	}

	d := Doc{Name: "Alice"} // Age=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	// 记录实际行为，不假设结果
	if _, ok := result["Age"]; !ok {
		t.Log("✅ tag omitempty IS working: zero Age omitted")
	} else {
		t.Logf("❌ tag omitempty NOT working: zero Age present = %v", result["Age"])
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("Age: got %v, want 30", result["Age"])
	}
}

func TestPreserveStructCodec_Encode_NilPointer(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string
	}

	var d *Doc = nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("nil pointer should encode to empty document, got %v", result)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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

func TestPreserveStructCodec_Decode_ZeroMaps(t *testing.T) {
	opts := &options.BSONOptions{ZeroMaps: true}
	r := GetPreserveFieldRegistry(opts)

	type Doc struct {
		Name string
		Tags map[string]string
	}

	decoded := Doc{
		Name: "old",
		Tags: map[string]string{"old": "value"},
	}

	raw, err := encodeRaw(r, bson.M{
		"Name": "new",
		"Tags": bson.M{"new": "entry"},
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
	if decoded.Tags["old"] != "" {
		t.Errorf("Old map entry should be cleared, got %v", decoded.Tags["old"])
	}
	if decoded.Tags["new"] != "entry" {
		t.Errorf("New map entry: got %v, want 'entry'", decoded.Tags["new"])
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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

func TestPreserveStructCodec_DuplicateInline_OverwriteDefault(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

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
	if err != nil {
		t.Errorf("unexpected error (overwrite should be default): %v", err)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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

func TestPreserveStructCodec_Encode_WrongKind(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	val := "not a struct"

	var buf bytes.Buffer
	vw := bson.NewDocumentWriter(&buf)
	enc := bson.NewEncoder(vw)
	enc.SetRegistry(r)

	err := enc.Encode(val)
	if err != nil {
		t.Logf("encode string returned (expected, has default encoder): %v", err)
	}
}

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
	if err != nil {
		t.Logf("decode with value (not pointer) returned: %v", err)
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
		Val2  string
		Level Level3
	}
	type Level1 struct {
		Val1  string
		Level Level2
	}
	type Doc struct {
		Name  string
		Level Level1
	}

	d := Doc{
		Name: "root",
		Level: Level1{
			Val1: "v1",
			Level: Level2{
				Val2: "v2",
				Level: Level3{
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
		"Level1": bson.M{
			"Val1": "v1",
			"Level2": bson.M{
				"Val2": "v2",
				"Level3": bson.M{
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
// 12. OmitZeroStruct 测试
// ─────────────────────────────────────────────

func TestPreserveStructCodec_Encode_OmitZeroStruct(t *testing.T) {
	opts := &options.BSONOptions{OmitZeroStruct: true}
	r := GetPreserveFieldRegistry(opts)

	type Inner struct {
		Value int
	}
	type Doc struct {
		Name  string
		Inner Inner
	}

	d := Doc{Name: "Alice"}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name: got %v, want Alice", result["Name"])
	}
	if _, ok := result["Inner"]; ok {
		t.Log("Note: Inner was included (OmitZeroStruct behavior depends on implementation)")
	} else {
		t.Log("Note: Inner was omitted (OmitZeroStruct worked)")
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
	if err := bson.Unmarshal(raw, &result); err != nil {
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
// 17. 三个"不支持"属性的实际行为探测
// ─────────────────────────────────────────────

// === omitempty 组 ===

func TestPreserveStructCodec_OmitEmpty_TagLevel_String(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string `bson:"Name,omitempty"`
		Age  int    `bson:"Age,omitempty"`
	}

	d := Doc{} // 零值

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Name"]; !ok {
		t.Log("✅ tag omitempty IS working: zero string omitted")
	} else {
		t.Logf("❌ tag omitempty NOT working: zero string present = %v", result["Name"])
	}
	if _, ok := result["Age"]; !ok {
		t.Log("✅ tag omitempty IS working: zero int omitted")
	} else {
		t.Logf("❌ tag omitempty NOT working: zero int present = %v", result["Age"])
	}
}

func TestPreserveStructCodec_OmitEmpty_TagLevel_NonZero(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Name string `bson:"Name,omitempty"`
		Age  int    `bson:"Age,omitempty"`
	}

	d := Doc{Name: "Alice", Age: 30}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("non-zero Name should be present, got %v", result["Name"])
	}
	if result["Age"] != int32(30) {
		t.Errorf("non-zero Age should be present, got %v", result["Age"])
	}
	t.Log("✅ tag omitempty: non-zero values correctly preserved")
}

func TestPreserveStructCodec_OmitEmpty_TagLevel_Bool(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Active bool `bson:"Active,omitempty"`
	}

	d := Doc{} // Active=false

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Active"]; !ok {
		t.Log("✅ tag omitempty IS working: zero bool omitted")
	} else {
		t.Logf("❌ tag omitempty NOT working: zero bool present = %v", result["Active"])
	}
}

func TestPreserveStructCodec_OmitEmpty_TagLevel_Slice(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Tags []string `bson:"Tags,omitempty"`
	}

	d := Doc{} // Tags=nil

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := result["Tags"]; !ok {
		t.Log("✅ tag omitempty IS working: nil slice omitted")
	} else {
		t.Logf("❌ tag omitempty NOT working: nil slice present = %v", result["Tags"])
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
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result["Name"] != "Alice" {
		t.Errorf("Name should be present, got %v", result["Name"])
	}
	if _, ok := result["Age"]; !ok {
		t.Log("✅ global OmitEmpty overrides tag: zero Age omitted without tag")
	} else {
		t.Errorf("❌ global OmitEmpty should omit zero Age, got %v", result["Age"])
	}
}

// === minsize 组 ===

func TestPreserveStructCodec_MinSize_Int64_Small(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value int64 `bson:"Value,minsize"`
	}

	d := Doc{Value: 42}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("minsize int64(42) encoded as: %v (type: %T)", val, val)

	switch val.(type) {
	case int32:
		t.Log("✅ minsize IS working: int64(42) → int32")
	case int64:
		t.Log("❌ minsize NOT working: int64(42) → int64 (no truncation)")
	default:
		t.Logf("❓ minsize: encoded as unexpected type %T", val)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("minsize int64(1<<40) encoded as: %v (type: %T)", val, val)

	switch val.(type) {
	case int64:
		t.Log("✅ minsize IS working: large int64 preserved as int64")
	case int32:
		t.Errorf("❌ minsize NOT working correctly: large value truncated to int32")
	default:
		t.Logf("❓ minsize: encoded as unexpected type %T", val)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("minsize uint64(100) encoded as: %v (type: %T)", val, val)

	switch val.(type) {
	case int32:
		t.Log("✅ minsize IS working: uint64(100) → int32")
	case int64:
		t.Log("❌ minsize NOT working: uint64(100) → int64 (no truncation)")
	default:
		t.Logf("❓ minsize: encoded as unexpected type %T", val)
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
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("control int64(42) encoded as: %v (type: %T)", val, val)

	if _, ok := val.(int64); ok {
		t.Log("✅ control group correct: int64 preserved as int64")
	} else {
		t.Errorf("❌ control group wrong: expected int64, got %T", val)
	}
}

// === truncate 组 ===

func TestPreserveStructCodec_Truncate_Float64(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float64 `bson:"Value,truncate"`
	}

	d := Doc{Value: 3.99}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("truncate float64(3.99) encoded as: %v (type: %T)", val, val)

	switch v := val.(type) {
	case float64:
		if v == 3.99 {
			t.Log("❌ truncate NOT working: float64(3.99) → float64 (no truncation)")
		} else {
			t.Logf("❓ truncate: got float64 but value changed to %v", v)
		}
	case float32:
		t.Log("✅ truncate IS working: float64 → float32")
	default:
		t.Logf("❓ truncate: encoded as unexpected type %T", val)
	}
}

func TestPreserveStructCodec_Truncate_Float32(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float32 `bson:"Value,truncate"`
	}

	d := Doc{Value: 7.77}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("truncate float32(7.77) encoded as: %v (type: %T)", val, val)

	if _, ok := val.(float32); ok {
		t.Log("✅ truncate: float32 preserved as float32")
	} else {
		t.Logf("❓ truncate: float32 encoded as %T", val)
	}
}

func TestPreserveStructCodec_Truncate_NegativeFloat(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float64 `bson:"Value,truncate"`
	}

	d := Doc{Value: -2.99}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("truncate float64(-2.99) encoded as: %v (type: %T)", val, val)

	switch val.(type) {
	case float32:
		t.Log("✅ truncate IS working: negative float64 → float32")
	case float64:
		t.Log("❌ truncate NOT working: negative float64 remains float64")
	default:
		t.Logf("❓ truncate: encoded as unexpected type %T", val)
	}
}

func TestPreserveStructCodec_Truncate_ControlGroup(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		Value float64 // 无 truncate tag
	}

	d := Doc{Value: 3.99}

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	val := result["Value"]
	t.Logf("control float64(3.99) encoded as: %v (type: %T)", val, val)

	if _, ok := val.(float64); ok {
		t.Log("✅ control group correct: float64 preserved as float64")
	} else {
		t.Errorf("❌ control group wrong: expected float64, got %T", val)
	}
}

// === 组合测试 ===

func TestPreserveStructCodec_OmitEmpty_And_MinSize(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		A int64 `bson:"A,omitempty,minsize"`
		B int64 `bson:"B,omitempty,minsize"`
	}

	d := Doc{} // A=0, B=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(result) == 0 {
		t.Log("✅ omitempty+minsize: both zero fields omitted")
	} else {
		t.Logf("❌ omitempty+minsize: got %v", result)
	}

	// 非零值
	d2 := Doc{A: 42}
	raw2, err := encodeRaw(r, d2)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result2 bson.M
	if err := bson.Unmarshal(raw2, &result2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	t.Logf("omitempty+minsize non-zero: A=42 encoded as %v (type %T)", result2["A"], result2["A"])
}

func TestPreserveStructCodec_OmitEmpty_And_Truncate(t *testing.T) {
	r := GetPreserveFieldRegistry(nil)

	type Doc struct {
		A float64 `bson:"A,omitempty,truncate"`
		B float64 `bson:"B,omitempty,truncate"`
	}

	d := Doc{} // A=0, B=0

	raw, err := encodeRaw(r, d)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result bson.M
	if err := bson.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(result) == 0 {
		t.Log("✅ omitempty+truncate: both zero fields omitted")
	} else {
		t.Logf("❌ omitempty+truncate: got %v", result)
	}

	// 非零值
	d2 := Doc{A: 3.99}
	raw2, err := encodeRaw(r, d2)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var result2 bson.M
	if err := bson.Unmarshal(raw2, &result2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	t.Logf("omitempty+truncate non-zero: A=3.99 encoded as %v (type %T)", result2["A"], result2["A"])
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
		t.Error("❌ ParseStructTags did not parse omitempty")
	} else {
		t.Log("✅ ParseStructTags correctly parsed omitempty")
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
		t.Error("❌ ParseStructTags did not parse minsize")
	} else {
		t.Log("✅ ParseStructTags correctly parsed minsize")
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
		t.Error("❌ ParseStructTags did not parse truncate")
	} else {
		t.Log("✅ ParseStructTags correctly parsed truncate")
	}
}
