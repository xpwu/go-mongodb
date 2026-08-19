package gen

import (
	"testing"
)

// ─── NewConfig ───────────────────────────────────────────────

func TestNewConfig_Defaults(t *testing.T) {
	c := NewConfig()
	if c == nil {
		t.Fatal("NewConfig returned nil")
	}
	if c.Maps == nil {
		t.Fatal("Maps should be initialized, got nil")
	}
	if len(c.Maps) != 0 {
		t.Errorf("Maps should be empty, got len=%d", len(c.Maps))
	}
	if c.PreserveField != false {
		t.Errorf("PreserveField default = %v, want false", c.PreserveField)
	}
	if c.UseJSONTags != false {
		t.Errorf("UseJSONTags default = %v, want false", c.UseJSONTags)
	}
	if c.IgnoreTagErr != false {
		t.Errorf("IgnoreTagErr default = %v, want false", c.IgnoreTagErr)
	}
	if c.Dir != "" {
		t.Errorf("Dir default = %q, want empty", c.Dir)
	}
	if c.Pkg != "" {
		t.Errorf("Pkg default = %q, want empty", c.Pkg)
	}
}

// ─── AddMap ──────────────────────────────────────────────────

func TestAddMap_Basic(t *testing.T) {
	c := NewConfig()
	c.AddMap("int", "fields.IntField", "fields.NewIntField", true)

	entry, ok := c.Maps["int"]
	if !ok {
		t.Fatal("AddMap did not add entry for 'int'")
	}
	if entry.Key != "int" {
		t.Errorf("Key = %q, want 'int'", entry.Key)
	}
	if entry.FieldType != "fields.IntField" {
		t.Errorf("FieldType = %q, want 'fields.IntField'", entry.FieldType)
	}
	if entry.NewFunc != "fields.NewIntField" {
		t.Errorf("NewFunc = %q, want 'fields.NewIntField'", entry.NewFunc)
	}
	if entry.EqualAble != true {
		t.Errorf("EqualAble = %v, want true", entry.EqualAble)
	}
}

func TestAddMap_Overwrite(t *testing.T) {
	c := NewConfig()
	c.AddMap("string", "fields.StringField", "fields.NewStringField", true)
	c.AddMap("string", "fields.MyStringField", "fields.NewMyStringField", false)

	entry := c.Maps["string"]
	if entry.FieldType != "fields.MyStringField" {
		t.Errorf("overwrite failed: FieldType = %q", entry.FieldType)
	}
	if entry.EqualAble != false {
		t.Errorf("overwrite failed: EqualAble = %v", entry.EqualAble)
	}
}

func TestAddMap_Multiple(t *testing.T) {
	c := NewConfig()
	c.AddMap("int", "fields.IntField", "fields.NewIntField", true)
	c.AddMap("string", "fields.StringField", "fields.NewStringField", true)
	c.AddMap("github.com/xpwu/go-mongodb/fields.ObjectID", "fields.ObjectIDField", "fields.NewObjectIDField", true)

	if len(c.Maps) != 3 {
		t.Errorf("expected 3 entries, got %d", len(c.Maps))
	}
	for _, key := range []string{"int", "string", "github.com/xpwu/go-mongodb/fields.ObjectID"} {
		if _, ok := c.Maps[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
}

func TestAddMap_EmptyKey(t *testing.T) {
	c := NewConfig()
	c.AddMap("", "fields.EmptyField", "fields.NewEmptyField", false)
	entry, ok := c.Maps[""]
	if !ok {
		t.Fatal("empty key should be stored")
	}
	if entry.FieldType != "fields.EmptyField" {
		t.Errorf("FieldType = %q", entry.FieldType)
	}
}

// ─── MapEntry 结构 ──────────────────────────────────────────

func TestMapEntry_StructFields(t *testing.T) {
	e := MapEntry{
		Key:       "time.Time",
		FieldType: "fields.TimeField",
		NewFunc:   "fields.NewTimeField",
		EqualAble: true,
	}
	if e.Key != "time.Time" {
		t.Errorf("Key = %q", e.Key)
	}
	if e.FieldType != "fields.TimeField" {
		t.Errorf("FieldType = %q", e.FieldType)
	}
	if e.NewFunc != "fields.NewTimeField" {
		t.Errorf("NewFunc = %q", e.NewFunc)
	}
	if !e.EqualAble {
		t.Error("EqualAble should be true")
	}
}

// ─── MapsSlice 排序 ──────────────────────────────────────────

func TestMapsSlice_Empty(t *testing.T) {
	c := NewConfig()
	slice := c.MapsSlice()
	if len(slice) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(slice))
	}
}

func TestMapsSlice_SortedByKey(t *testing.T) {
	c := NewConfig()
	c.AddMap("zebra", "fields.ZebraField", "fields.NewZebraField", true)
	c.AddMap("apple", "fields.AppleField", "fields.NewAppleField", true)
	c.AddMap("mango", "fields.MangoField", "fields.NewMangoField", true)

	slice := c.MapsSlice()
	if len(slice) != 3 {
		t.Fatalf("expected 3, got %d", len(slice))
	}
	expectedOrder := []string{"apple", "mango", "zebra"}
	for i, exp := range expectedOrder {
		if slice[i].Key != exp {
			t.Errorf("slice[%d].Key = %q, want %q", i, slice[i].Key, exp)
		}
	}
}

func TestMapsSlice_StableOrder(t *testing.T) {
	c := NewConfig()
	keys := []string{"d", "a", "c", "b"}
	for _, k := range keys {
		c.AddMap(k, "fields.F", "fields.NewF", true)
	}
	slice := c.MapsSlice()
	for i := 1; i < len(slice); i++ {
		if slice[i].Key <= slice[i-1].Key {
			t.Errorf("not sorted: [%d]=%q vs [%d]=%q", i-1, slice[i-1].Key, i, slice[i].Key)
		}
	}
}

func TestMapsSlice_DoesNotMutateOriginal(t *testing.T) {
	c := NewConfig()
	c.AddMap("x", "fields.X", "fields.NewX", true)
	c.AddMap("y", "fields.Y", "fields.NewY", true)

	slice1 := c.MapsSlice()
	slice2 := c.MapsSlice()
	if &slice1[0] == &slice2[0] {
		t.Error("MapsSlice should return a new slice each time")
	}
	// 修改 slice1 不应影响 c.Maps
	slice1[0].Key = "mutated"
	if c.Maps["x"].Key != "x" {
		t.Errorf("original map mutated: %+v", c.Maps["x"])
	}
}

// ─── Config 字段赋值 ────────────────────────────────────────

func TestConfig_FieldAssignment(t *testing.T) {
	c := NewConfig()
	c.PreserveField = true
	c.UseJSONTags = true
	c.IgnoreTagErr = true
	c.Dir = "./zgen"
	c.Pkg = "mypkg"

	if !c.PreserveField {
		t.Error("PreserveField should be true")
	}
	if !c.UseJSONTags {
		t.Error("UseJSONTags should be true")
	}
	if !c.IgnoreTagErr {
		t.Error("IgnoreTagErr should be true")
	}
	if c.Dir != "./zgen" {
		t.Errorf("Dir = %q", c.Dir)
	}
	if c.Pkg != "mypkg" {
		t.Errorf("Pkg = %q", c.Pkg)
	}
}

// ─── 集成：Config + AddMap 完整场景 ────────────────────────

func TestConfig_Integration_TypicalUsage(t *testing.T) {
	c := NewConfig()
	c.PreserveField = true
	c.Dir = "./zgen"
	c.Pkg = "userinfo"

	c.AddMap("int", "fields.IntField", "fields.NewIntField", true)
	c.AddMap("string", "fields.StringField", "fields.NewStringField", true)
	c.AddMap("float64", "fields.Float64Field", "fields.NewFloat64Field", true)
	c.AddMap("github.com/xpwu/go-mongodb/fields.ObjectID", "fields.ObjectIDField", "fields.NewObjectIDField", true)
	c.AddMap("time.Time", "fields.TimeField", "fields.NewTimeField", false)

	slice := c.MapsSlice()
	if len(slice) != 5 {
		t.Fatalf("expected 5 maps, got %d", len(slice))
	}

	// 验证排序正确
	expectedKeys := []string{
		"float64",
		"github.com/xpwu/go-mongodb/fields.ObjectID",
		"int",
		"string",
		"time.Time",
	}
	for i, exp := range expectedKeys {
		if slice[i].Key != exp {
			t.Errorf("slice[%d].Key = %q, want %q", i, slice[i].Key, exp)
		}
	}

	// 验证 EqualAble 设置
	equalAbleMap := map[string]bool{}
	for _, e := range slice {
		equalAbleMap[e.Key] = e.EqualAble
	}
	if equalAbleMap["time.Time"] != false {
		t.Error("time.Time should be EqualAble=false")
	}
	if equalAbleMap["int"] != true {
		t.Error("int should be EqualAble=true")
	}
}

// ─── AddMap 不支持泛型的说明验证 ────────────────────────────

func TestAddMap_GenericNotSupported(t *testing.T) {
	// 文档说明泛型不支持，这里验证非泛型写法正常工作
	c := NewConfig()
	c.AddMap("MyType", "fields.MyField", "fields.NewMyField", true)

	entry := c.Maps["MyType"]
	if entry.FieldType != "fields.MyField" {
		t.Errorf("non-generic AddMap failed: %q", entry.FieldType)
	}
}

// ─── reflect.Kind 无关性验证 ───────────────────────────────

func TestConfig_NotDependentOnReflect(t *testing.T) {
	// Config 不依赖 reflect.Kind，它是纯数据配置
	// 验证不同类型标识字符串都能正常存储
	c := NewConfig()
	idents := []string{
		"int",
		"int8",
		"uint64",
		"float64",
		"string",
		"bool",
		"[]byte",
		"map[string]int",
		"github.com/foo/bar.Baz",
		"mypkg.InnerType",
	}
	for _, id := range idents {
		c.AddMap(id, "fields.F", "fields.NewF", true)
	}
	if len(c.Maps) != len(idents) {
		t.Errorf("expected %d entries, got %d", len(idents), len(c.Maps))
	}
	slice := c.MapsSlice()
	if len(slice) != len(idents) {
		t.Errorf("MapsSlice len = %d, want %d", len(slice), len(idents))
	}
	// 验证是字典序排序
	for i := 1; i < len(slice); i++ {
		if slice[i].Key < slice[i-1].Key {
			t.Errorf("not sorted at %d: %q < %q", i, slice[i].Key, slice[i-1].Key)
		}
	}
}
