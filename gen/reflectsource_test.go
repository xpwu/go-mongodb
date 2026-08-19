package gen

import (
	"reflect"
	"testing"
)

// ─── ReflectTypeSource 基本属性 ───────────────────────────────

func TestReflectTypeSource_Basic(t *testing.T) {
	ts := ReflectTypeSource(reflect.TypeOf(int(0)))
	if ts.Name() != "int" {
		t.Fatalf("Name() = %q, want int", ts.Name())
	}
	if ts.PkgPath() != "" {
		t.Fatalf("PkgPath() = %q, want empty", ts.PkgPath())
	}
	if ts.Kind() != reflect.Int {
		t.Fatalf("Kind() = %v, want reflect.Int", ts.Kind())
	}
	if !ts.IsBuiltin() {
		t.Fatal("IsBuiltin() = false, want true")
	}
	if ts.Elem() != nil {
		t.Fatal("Elem() should be nil for non-composite type")
	}
}

func TestReflectTypeSource_Struct(t *testing.T) {
	type User struct {
		Name string `bson:"name"`
		Age  int    `bson:"age"`
	}
	ts := ReflectTypeSource(reflect.TypeOf(User{}))
	if ts.Name() != "User" {
		t.Fatalf("Name() = %q, want User", ts.Name())
	}
	if ts.Kind() != reflect.Struct {
		t.Fatalf("Kind() = %v, want reflect.Struct", ts.Kind())
	}
	if ts.IsBuiltin() {
		t.Fatal("IsBuiltin() = true, want false")
	}
	if ts.NumField() != 2 {
		t.Fatalf("NumField() = %d, want 2", ts.NumField())
	}
	f0 := ts.Field(0)
	if f0.Name() != "Name" {
		t.Errorf("Field(0).Name() = %q, want Name", f0.Name())
	}
	if f0.Tag() != "bson:\"name\"" {
		t.Errorf("Field(0).Tag() = %q, want bson:\"name\"", f0.Tag())
	}
	if !f0.IsExported() {
		t.Error("Field(0).IsExported() = false, want true")
	}
	f1 := ts.Field(1)
	if f1.Name() != "Age" {
		t.Errorf("Field(1).Name() = %q, want Age", f1.Name())
	}
	if f1.Type().Kind() != reflect.Int {
		t.Errorf("Field(1).Type().Kind() = %v, want reflect.Int", f1.Type().Kind())
	}
}

func TestReflectTypeSource_Ptr(t *testing.T) {
	type User struct{ Name string }
	ts := ReflectTypeSource(reflect.TypeOf(&User{}))
	if ts.Kind() != reflect.Ptr {
		t.Fatalf("Kind() = %v, want reflect.Ptr", ts.Kind())
	}
	elem := ts.Elem()
	if elem == nil {
		t.Fatal("Elem() is nil")
	}
	if elem.Name() != "User" || elem.Kind() != reflect.Struct {
		t.Errorf("Elem() = %+v, want User struct", elem)
	}
}

func TestReflectTypeSource_Slice(t *testing.T) {
	ts := ReflectTypeSource(reflect.TypeOf([]string{}))
	if ts.Kind() != reflect.Slice {
		t.Fatalf("Kind() = %v, want reflect.Slice", ts.Kind())
	}
	elem := ts.Elem()
	if elem == nil || elem.Name() != "string" || elem.Kind() != reflect.String {
		t.Errorf("Elem() = %+v, want string", elem)
	}
}

func TestReflectTypeSource_Map(t *testing.T) {
	ts := ReflectTypeSource(reflect.TypeOf(map[string]int{}))
	if ts.Kind() != reflect.Map {
		t.Fatalf("Kind() = %v, want reflect.Map", ts.Kind())
	}
	if ts.Elem() == nil || ts.Elem().Kind() != reflect.Int {
		t.Error("Elem() should be int")
	}
}

func TestReflectTypeSource_FieldIndexOutOfRange(t *testing.T) {
	type User struct{ Name string }
	ts := ReflectTypeSource(reflect.TypeOf(User{}))
	if ts.Field(-1) != nil {
		t.Error("Field(-1) should be nil")
	}
	if ts.Field(99) != nil {
		t.Error("Field(99) should be nil")
	}
}

// ─── Underlying() 始终返回 nil ────────────────────────────────

func TestReflectTypeSource_Underlying_AlwaysNil(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
	}{
		{"builtin int", reflect.TypeOf(int(0))},
		{"builtin string", reflect.TypeOf("")},
		{"struct", reflect.TypeOf(struct{ X int }{})},
		{"ptr", reflect.TypeOf(&struct{}{})},
		{"slice", reflect.TypeOf([]int{})},
		{"map", reflect.TypeOf(map[string]int{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := ReflectTypeSource(tt.typ)
			next, alias := ts.Underlying()
			if next != nil {
				t.Errorf("Underlying() next = %+v, want nil", next)
			}
			if alias {
				t.Error("Underlying() alias = true, want false")
			}
		})
	}
}

// ─── EnsureFields 幂等 ────────────────────────────────────────

func TestReflectTypeSource_EnsureFields_Idempotent(t *testing.T) {
	type User struct{ Name string }
	ts := ReflectTypeSource(reflect.TypeOf(User{}))
	ts.EnsureFields()
	n1 := ts.NumField()
	ts.EnsureFields()
	n2 := ts.NumField()
	if n1 != n2 {
		t.Errorf("not idempotent: first=%d second=%d", n1, n2)
	}
}

// ─── 非导出字段不标记 IsExported ─────────────────────────────

func TestReflectTypeSource_UnexportedField(t *testing.T) {
	type hidden struct {
		Name string
		age  int
	}
	ts := ReflectTypeSource(reflect.TypeOf(hidden{}))
	if ts.NumField() != 2 {
		t.Fatalf("NumField() = %d, want 2", ts.NumField())
	}
	if !ts.Field(0).IsExported() {
		t.Error("Name should be exported")
	}
	if ts.Field(1).IsExported() {
		t.Error("age should not be exported")
	}
}

// ─── 匿名结构体字段 ───────────────────────────────────────────

func TestReflectTypeSource_AnonymousStructField(t *testing.T) {
	type Outer struct {
		Inner struct {
			X int
			Y string
		}
	}
	ts := ReflectTypeSource(reflect.TypeOf(Outer{}))
	if ts.NumField() != 1 {
		t.Fatalf("NumField() = %d, want 1", ts.NumField())
	}
	f := ts.Field(0)
	if f.Name() != "Inner" {
		t.Errorf("Field(0).Name() = %q, want Inner", f.Name())
	}
	inner := f.Type()
	if inner.Kind() != reflect.Struct {
		t.Fatalf("Inner type kind = %v, want Struct", inner.Kind())
	}
	if inner.NumField() != 2 {
		t.Fatalf("Inner.NumField() = %d, want 2", inner.NumField())
	}
	if inner.Field(0).Name() != "X" || inner.Field(1).Name() != "Y" {
		t.Errorf("Inner fields wrong: %+v", inner.Field(0).Name())
	}
}

// ─── 接口类型 ─────────────────────────────────────────────────

func TestReflectTypeSource_Interface(t *testing.T) {
	var x any
	ts := ReflectTypeSource(reflect.TypeOf(&x).Elem())
	if ts.Kind() != reflect.Interface {
		t.Fatalf("Kind() = %v, want reflect.Interface", ts.Kind())
	}
	if ts.NumField() != 0 {
		t.Errorf("Interface NumField() = %d, want 0", ts.NumField())
	}
}

// ─── 类型一致性：reflect 和 AST 对同一个 struct 的描述一致 ──

func TestReflectTypeSource_ConsistentWithAST(t *testing.T) {
	type User struct {
		ID   string `bson:"_id"`
		Name string `bson:"name"`
		Age  int    `bson:"age"`
	}
	ts := ReflectTypeSource(reflect.TypeOf(User{}))

	if ts.Name() != "User" {
		t.Errorf("Name() = %q", ts.Name())
	}
	if ts.Kind() != reflect.Struct {
		t.Errorf("Kind() = %v", ts.Kind())
	}
	if ts.NumField() != 3 {
		t.Fatalf("NumField() = %d, want 3", ts.NumField())
	}

	// 验证字段名和类型
	expected := []struct {
		name string
		kind reflect.Kind
		tag  string
	}{
		{"ID", reflect.String, "bson:\"_id\""},
		{"Name", reflect.String, "bson:\"name\""},
		{"Age", reflect.Int, "bson:\"age\""},
	}
	for i, exp := range expected {
		f := ts.Field(i)
		if f.Name() != exp.name {
			t.Errorf("Field(%d).Name() = %q, want %q", i, f.Name(), exp.name)
		}
		if f.Type().Kind() != exp.kind {
			t.Errorf("Field(%d).Type().Kind() = %v, want %v", i, f.Type().Kind(), exp.kind)
		}
		if f.Tag() != exp.tag {
			t.Errorf("Field(%d).Tag() = %q, want %q", i, f.Tag(), exp.tag)
		}
	}
}
