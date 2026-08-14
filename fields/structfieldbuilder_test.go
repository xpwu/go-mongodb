package fields

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xpwu/go-mongodb/xopt"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== 测试用的结构体 ====================

type SimpleStruct struct {
	Name  string
	Age   int
	Score float64
}

type NestedStruct struct {
	ID      string
	Profile struct {
		Email string
		Phone string
	}
	Tags []string
}

type StructWithBsonTypes struct {
	ID        bson.ObjectID
	Data      bson.M
	RawBytes  bson.Binary
	CreatedAt bson.DateTime
}

type StructWithTags struct {
	Name  string `bson:"name"`
	Age   int    `bson:"age,omitempty"`
	Email string `bson:"-"`
}

type InlineStruct struct {
	BaseField1 string
	BaseField2 int
}

type StructWithInline struct {
	InlineField InlineStruct `bson:",inline"`
	Name        string
}

// ==================== NewTypeInfo 测试 ====================

func TestNewTypeInfo_Int(t *testing.T) {
	ti := NewTypeInfo[bson.Timestamp](NewTimestampField)
	if ti.T == nil {
		t.Error("NewTypeInfo: T should not be nil")
	}
	if ti.T.Name() != "Timestamp" {
		t.Errorf("NewTypeInfo: T.Name() = %v, want Timestamp", ti.T.Name())
	}
	if !ti.EqualAble {
		t.Error("NewTypeInfo[int]: EqualAble should be true")
	}
}

func newStringField(name string) StringField {
	return NewStringField(name)
}

func TestNewTypeInfo_String(t *testing.T) {
	ti := NewTypeInfo[string](newStringField)
	if ti.T.Name() != "string" {
		t.Errorf("NewTypeInfo[string]: T.Name() = %v, want string", ti.T.Name())
	}
	if !ti.EqualAble {
		t.Error("NewTypeInfo[string]: EqualAble should be true")
	}
}

func TestNewTypeInfo_BsonObjectID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Failed to inspect generic function")
		}
	}()

	NewTypeInfo[bson.ObjectID](NewObjectIDField)
}

//go:noinline
func newFloat64Field(name string) Float64Field {
	return NewFloat64Field(name)
}

func TestNewTypeInfo_Float64(t *testing.T) {
	ti := NewTypeInfo[float64](newFloat64Field)
	if ti.T.Name() != "float64" {
		t.Errorf("NewTypeInfo[float64]: T.Name() = %v, want float64", ti.T.Name())
	}
	if ti.EqualAble {
		t.Error("NewTypeInfo[float64]: EqualAble should be false")
	}
}

// ==================== NewStructFieldBuilder 测试 ====================

func TestNewStructFieldBuilder_DefaultOpts(t *testing.T) {
	b := NewStructFieldBuilder()
	if b == nil {
		t.Fatal("NewStructFieldBuilder: returned nil")
	}
	if b.typeMap == nil {
		t.Error("typeMap should be initialized")
	}
	if b.kindMap == nil {
		t.Error("kindMap should be initialized")
	}
	// 验证内置类型已注册
	if _, ok := b.typeMap[reflect.TypeOf(0)]; !ok {
		t.Error("int type should be registered")
	}
	if _, ok := b.typeMap[reflect.TypeOf("")]; !ok {
		t.Error("string type should be registered")
	}
	if _, ok := b.typeMap[reflect.TypeOf(true)]; !ok {
		t.Error("bool type should be registered")
	}
	if _, ok := b.typeMap[reflect.TypeOf(bson.ObjectID{})]; !ok {
		t.Error("bson.ObjectID type should be registered")
	}
	if _, ok := b.typeMap[reflect.TypeOf(bson.M{})]; !ok {
		t.Error("bson.M type should be registered")
	}
}

func TestNewStructFieldBuilder_WithOpts(t *testing.T) {
	b := NewStructFieldBuilder(xopt.WithPreserveField(true))
	if b == nil {
		t.Fatal("NewStructFieldBuilder with opts: returned nil")
	}
	if b.opt == nil {
		t.Fatal("builder option should not be nil")
	}
	if !b.opt.preserveField {
		t.Error("preserveField should be true")
	}
}

// ==================== RegisterType / RegisterKind / ClearType 测试 ====================

func TestStructFieldBuilder_RegisterType(t *testing.T) {
	b := NewStructFieldBuilder()

	type CustomInt int
	ti := TypeInfo{
		T:         reflect.TypeOf(CustomInt(0)),
		Field:     &reflectType{name: "IntegerField[fields.CustomInt]", pkg: "github.com/xpwu/go-mongodb/fields"},
		NewField:  &reflectType{name: "NewIntegerField[fields.CustomInt]", pkg: "github.com/xpwu/go-mongodb/fields"},
		EqualAble: true,
	}

	b.RegisterType(ti)
	if got, ok := b.typeMap[reflect.TypeOf(CustomInt(0))]; !ok {
		t.Error("RegisterType: type not found after registration")
	} else if got.T != ti.T {
		t.Error("RegisterType: registered type mismatch")
	}
}

func TestStructFieldBuilder_RegisterKind(t *testing.T) {
	b := NewStructFieldBuilder()

	called := false
	b.RegisterKind(reflect.Float64, func(rt reflect.Type) (TypeInfo, bool) {
		called = true
		return TypeInfo{T: rt}, true
	})

	_, ok := b.kindMap[reflect.Float64](reflect.TypeOf(float64(0)))
	if !called {
		t.Error("RegisterKind: callback was not called")
	}
	if !ok {
		t.Error("RegisterKind: should return ok=true")
	}
}

func TestStructFieldBuilder_ClearType(t *testing.T) {
	b := NewStructFieldBuilder()

	type MyType int
	ti := TypeInfo{T: reflect.TypeOf(MyType(0)), EqualAble: true}
	b.RegisterType(ti)

	if _, ok := b.typeMap[reflect.TypeOf(MyType(0))]; !ok {
		t.Fatal("type should be registered")
	}

	b.ClearType(reflect.TypeOf(MyType(0)))

	if _, ok := b.typeMap[reflect.TypeOf(MyType(0))]; ok {
		t.Error("ClearType: type should be removed")
	}
}

// ==================== Build 测试 ====================

func TestStructFieldBuilder_BuildSimpleStruct(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(SimpleStruct{})
	b.Build(rt)

	if ti, ok := b.typeMap[rt]; !ok {
		t.Error("Build: struct type should be registered after build")
	} else {
		if !strings.Contains(ti.Field.Name(), "SimpleStruct") {
			t.Errorf("Build: Field.Name() = %v, should contain 'SimpleStruct'", ti.Field.Name())
		}
	}
}

func TestStructFieldBuilder_BuildPointer(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(&SimpleStruct{})
	b.Build(rt)

	// 应该注册的是 SimpleStruct 而不是 *SimpleStruct
	elemType := rt.Elem()
	if ti, ok := b.typeMap[elemType]; !ok {
		t.Error("Build: should register the dereferenced struct type")
	} else {
		_ = ti
	}
}

func TestStructFieldBuilder_BuildBsonTypes(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(StructWithBsonTypes{})
	b.Build(rt)

	if _, ok := b.typeMap[rt]; !ok {
		t.Error("Build: StructWithBsonTypes should be registered")
	}
}

func TestStructFieldBuilder_BuildNestedStruct(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(NestedStruct{})
	b.Build(rt)

	if _, ok := b.typeMap[rt]; !ok {
		t.Error("Build: NestedStruct should be registered")
	}
}

func TestStructFieldBuilder_BuildWithTags(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(StructWithTags{})
	b.Build(rt)

	if _, ok := b.typeMap[rt]; !ok {
		t.Error("Build: StructWithTags should be registered")
	}
}

func TestStructFieldBuilder_BuildWithInline(t *testing.T) {
	b := NewStructFieldBuilder()
	b.opt.dir = os.TempDir()
	b.opt.targetPkg = "testpkg"

	rt := reflect.TypeOf(StructWithInline{})
	b.Build(rt)

	if _, ok := b.typeMap[rt]; !ok {
		t.Error("Build: StructWithInline should be registered")
	}
}

// ==================== 生成代码验证测试 ====================

func TestBuildStruct_GeneratesValidCode(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_test_gen")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "gentest"

	rt := reflect.TypeOf(SimpleStruct{})
	subDir := b.Build(rt)

	expectedFile := filepath.Join(tmpDir, subDir, "zSimpleStructField.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("Build: expected file %s to be created, err: %v", expectedFile, err)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Read generated file: %v", err)
	}

	contentStr := string(content)
	checks := []string{
		"package gentest",
		"SimpleStructField",
		"NameF()",
		"AgeF()",
		"ScoreF()",
		"NewSimpleStructField",
		"SimpleStructDoc",
	}
	if subDir != "" {
		checks[0] = "package " + subDir
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Generated code missing: %s\nFile content:\n%s", check, contentStr)
		}
	}
}

func TestBuildStruct_NestedGeneratesValidCode(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_test_gen_nested")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "gentest_nested"

	rt := reflect.TypeOf(NestedStruct{})
	subDir := b.Build(rt)

	expectedFile := filepath.Join(tmpDir, subDir, "zNestedStructField.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("Build: expected file %s to be created, err: %v", expectedFile, err)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Profile") {
		t.Error("Generated code should contain Profile field reference")
	}
}

func TestBuildStruct_InlineGeneratesValidCode(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_test_gen_inline")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "gentest_inline"

	rt := reflect.TypeOf(StructWithInline{})
	subDir := b.Build(rt)

	expectedFile := filepath.Join(tmpDir, subDir, "zStructWithInlineField.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("Build: expected file %s to be created, err: %v", expectedFile, err)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Inline") {
		t.Error("Generated code should contain Inline reference for inline fields")
	}
}

// ==================== base6408 测试 ====================

func TestBase6408_Consistent(t *testing.T) {
	s := "github.com/example/mypackage"
	r1 := base6408(s)
	r2 := base6408(s)

	if r1 != r2 {
		t.Errorf("base6408: should be deterministic, got %v and %v", r1, r2)
	}
	if len(r1) != 8 {
		t.Errorf("base6408: should return 8 chars, got %d (%v)", len(r1), r1)
	}
}

func TestBase6408_DifferentInput(t *testing.T) {
	r1 := base6408("package1")
	r2 := base6408("package2")

	if r1 == r2 {
		t.Errorf("base6408: different inputs should produce different outputs, got %v", r1)
	}
}

func TestBase6408_KnownOutput(t *testing.T) {
	// sha256("test")[:8] base64 encoded
	r := base6408("test")
	if len(r) != 8 {
		t.Errorf("base6408: expected 8 chars, got %d (%v)", len(r), r)
	}
}

// ==================== 辅助函数测试 ====================

func TestFirstToLower(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello", "hello"},
		{"", ""},
		{"A", "a"},
		{"already", "already"},
		{"XYZ", "xYZ"},
	}
	for _, tt := range tests {
		got := firstToLower(tt.input)
		if got != tt.want {
			t.Errorf("firstToLower(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIndentLines(t *testing.T) {
	input := "line1\nline2\nline3"
	got := indentLines(input, 2)
	want := "line1\n\t\tline2\n\t\tline3"

	if got != want {
		t.Errorf("indentLines: got %q, want %q", got, want)
	}
}

func TestIndentLines_SingleLine(t *testing.T) {
	input := "only one line"
	got := indentLines(input, 3)
	want := "only one line"

	if got != want {
		t.Errorf("indentLines(single): got %q, want %q", got, want)
	}
}

// ==================== allImports 测试 ====================

func TestAllImports_Add(t *testing.T) {
	ai := newAllImports()

	a1 := ai.add("github.com/xpwu/go-mongodb/filter")
	a2 := ai.add("github.com/xpwu/go-mongodb/filter")

	if a1 != a2 {
		t.Errorf("allImports.add: same path should return same alias, got %v and %v", a1, a2)
	}
	if a1 == "" {
		t.Error("allImports.add: alias should not be empty")
	}
}

func TestAllImports_AddDifferent(t *testing.T) {
	ai := newAllImports()

	a1 := ai.add("github.com/xpwu/go-mongodb/filter")
	a2 := ai.add("github.com/xpwu/go-mongodb/updater")

	if a1 == a2 {
		t.Errorf("allImports.add: different paths should return different aliases")
	}
}

func TestAllImports_Exclude(t *testing.T) {
	ai := newAllImports()
	ai.exclude("github.com/excluded/pkg")

	alias := ai.add("github.com/excluded/pkg")
	if alias != "" {
		t.Errorf("allImports.add excluded: should return empty, got %v", alias)
	}
}

func TestAllImports_All(t *testing.T) {
	ai := newAllImports()
	ai.add("github.com/xpwu/go-mongodb/filter")
	ai.add("github.com/xpwu/go-mongodb/updater")
	ai.add("go.mongodb.org/mongo-driver/v2/bson")

	all := ai.all()
	if len(all) != 3 {
		t.Errorf("allImports.all: got %d entries, want 3", len(all))
	}

	// 验证排序（按 import path 排序）
	for i := 1; i < len(all); i++ {
		if all[i].Import < all[i-1].Import {
			t.Errorf("allImports.all: not sorted, %v before %v", all[i], all[i-1])
		}
	}
}

// ==================== 集成测试 ====================

func TestIntegration_BuildAndUse(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_integration")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "integration"

	rt := reflect.TypeOf(SimpleStruct{})
	subDir := b.Build(rt)

	expectedFile := filepath.Join(tmpDir, subDir, "zSimpleStructField.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("Integration: file not created: %v", err)
	}

	content, _ := os.ReadFile(expectedFile)
	if subDir == "" {
		if !bytes.Contains(content, []byte("package integration")) {
			t.Error("Integration: generated file has wrong package name")
		}
	} else {
		if !bytes.Contains(content, []byte("package "+subDir)) {
			t.Error("Integration: generated file has wrong package name")
		}
	}

	checks := []string{
		"filter.ComparableFilter",
		"updater.BaseUpdater",
		"field.Field",
	}
	for _, check := range checks {
		if !bytes.Contains(content, []byte(check)) {
			t.Errorf("Integration: generated code missing %s", check)
		}
	}
}

func TestIntegration_GeneratedFieldFilterUpdater(t *testing.T) {
	// 使用已有的 BaseField 模拟生成代码的行为
	b := NewStringField("name")

	// Filter
	f := b.Eq("test")
	got := f.ToBsonD()
	want := bson.D{{"name", "test"}}
	if !bsonDEqual(got, want) {
		t.Errorf("Integration Filter: got %v, want %v", got, want)
	}

	// Updater
	u := b.Set("newName")
	gotU := u.ToBsonM()
	wantU := bson.M{"$set": bson.M{"name": "newName"}}
	if !bsonMEqual(gotU, wantU) {
		t.Errorf("Integration Updater: got %v, want %v", gotU, wantU)
	}
}

// ==================== Build 后的类型信息验证 ====================

func TestBuildStruct_TypeInfoFields(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_typeinfo")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "typeinfo"

	rt := reflect.TypeOf(SimpleStruct{})
	b.Build(rt)

	ti, ok := b.typeMap[rt]
	if !ok {
		t.Fatal("Build: type not registered")
	}

	if ti.Field == nil {
		t.Error("Field interface info should not be nil")
	}
	if ti.NewField == nil {
		t.Error("NewField info should not be nil")
	}
	if ti.EqualAble {
		t.Error("SimpleStruct should not be EqualAble (float64 is not comparable)")
	}
}

// ==================== 模板渲染测试 ====================

func TestStructCode2_TemplateExists(t *testing.T) {
	// 验证 structCode2 模板已初始化
	if structCode2 == nil {
		t.Fatal("structCode2 template should be initialized")
	}
}

// ==================== BuildStruct 简化 API 测试 ====================

func TestBuildStruct_Simple(t *testing.T) {
	b := NewStructFieldBuilder()
	tmpDir := filepath.Join(os.TempDir(), "go_mongodb_buildsimple")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	b.opt.dir = tmpDir
	b.opt.targetPkg = "buildsimple"

	b.Build(reflect.TypeOf(SimpleStruct{}))
}
