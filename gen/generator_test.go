package gen

import (
	"bytes"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ─── parseTypeRef 测试 ─────────────────────────────────────

func TestParseTypeRef_SimpleName(t *testing.T) {
	r := parseTypeRef("IntField")
	if r.name != "IntField" {
		t.Errorf("name = %s, want IntField", r.name)
	}
	if r.pkg != "" {
		t.Errorf("pkg = %s, want empty", r.pkg)
	}
}

func TestParseTypeRef_PackageQualified(t *testing.T) {
	r := parseTypeRef("fields.IntField")
	if r.name != "IntField" {
		t.Errorf("name = %s, want IntField", r.name)
	}
	if r.pkg != "fields" {
		t.Errorf("pkg = %s, want fields", r.pkg)
	}
}

func TestParseTypeRef_FullyQualified(t *testing.T) {
	r := parseTypeRef("github.com/xpwu/go-mongodb/fields.ObjectIDField")
	if r.name != "ObjectIDField" {
		t.Errorf("name = %s, want ObjectIDField", r.name)
	}
	if r.pkg != "github.com/xpwu/go-mongodb/fields" {
		t.Errorf("pkg = %s", r.pkg)
	}
}

func TestParseTypeRef_NoDot(t *testing.T) {
	r := parseTypeRef("JustName")
	if r.name != "JustName" {
		t.Errorf("name = %s", r.name)
	}
	if r.pkg != "" {
		t.Errorf("pkg should be empty")
	}
}

// ─── addDot 测试 ───────────────────────────────────────────

func TestAddDot_NonEmpty(t *testing.T) {
	result := addDot("fields")
	if result != "fields." {
		t.Errorf("addDot = %s, want fields.", result)
	}
}

func TestAddDot_Empty(t *testing.T) {
	result := addDot("")
	if result != "" {
		t.Errorf("addDot(\"\") = %s, want empty", result)
	}
}

// ─── indentLines 测试 ───────────────────────────────────────

func TestIndentLines_SingleLine(t *testing.T) {
	result := indentLines("hello", 2)
	if result != "hello" {
		t.Errorf("single line should not be indented, got %s", result)
	}
}

func TestIndentLines_MultiLine(t *testing.T) {
	input := "line1\nline2\nline3"
	result := indentLines(input, 1)
	lines := strings.Split(result, "\n")
	if lines[0] != "line1" {
		t.Errorf("line0 = %s", lines[0])
	}
	if lines[1] != "\tline2" {
		t.Errorf("line1 = %s, want \\tline2", lines[1])
	}
	if lines[2] != "\tline3" {
		t.Errorf("line2 = %s, want \\tline3", lines[2])
	}
}

func TestIndentLines_Empty(t *testing.T) {
	result := indentLines("", 3)
	if result != "" {
		t.Errorf("empty string should stay empty")
	}
}

// ─── extractBetweenFlexible 测试 ───────────────────────────

func TestExtractBetweenFlexible_BothFound(t *testing.T) {
	s := "LikeUserField[github.com/foo.User]"
	result := extractBetweenFlexible(s, "Like", "[")
	if result != "UserField" {
		t.Errorf("result = %s, want UserField", result)
	}
}

func TestExtractBetweenFlexible_StartNotFound(t *testing.T) {
	s := "UserField[github.com/foo.User]"
	result := extractBetweenFlexible(s, "Like", "[")
	if result != "UserField" {
		t.Errorf("result = %s", result)
	}
}

func TestExtractBetweenFlexible_EndNotFound(t *testing.T) {
	s := "LikeUserFieldNoBracket"
	result := extractBetweenFlexible(s, "Like", "[")
	if result != "UserFieldNoBracket" {
		t.Errorf("result = %s", result)
	}
}

func TestExtractBetweenFlexible_Empty(t *testing.T) {
	result := extractBetweenFlexible("", "Like", "[")
	if result != "" {
		t.Errorf("empty input should return empty")
	}
}

func TestExtractBetweenFlexible(t *testing.T) {
	tests := []struct {
		name, s, start, end, expected string
	}{
		{"both found", "LikeString[string]", "Like", "[", "String"},
		{"only start", "HelloWorld", "Hello", "XYZ", "World"},
		{"neither", "HelloWorld", "XYZ", "ABC", "HelloWorld"},
		{"empty end", "HelloWorld", "Hello", "", "World"},
		{"empty start", "HelloWorld", "", "World", "Hello"},
		{"start not found end found", "abcdef", "xyz", "def", "abc"},
		{"both empty", "abc", "", "", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBetweenFlexible(tt.s, tt.start, tt.end)
			if got != tt.expected {
				t.Errorf("extractBetweenFlexible(%q,%q,%q) = %q, want %q",
					tt.s, tt.start, tt.end, got, tt.expected)
			}
		})
	}
}

func TestIndentLines(t *testing.T) {
	tests := []struct {
		name, input string
		indents     int
		expected    string
	}{
		{"single line", "hello", 1, "hello"},
		{"two lines", "line1\nline2", 1, "line1\n\tline2"},
		{"double indent", "a\nb\nc", 2, "a\n\t\tb\n\t\tc"},
		{"zero indent", "x\ny", 0, "x\ny"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentLines(tt.input, tt.indents)
			if got != tt.expected {
				t.Errorf("indentLines(%q,%d) = %q, want %q", tt.input, tt.indents, got, tt.expected)
			}
		})
	}
}

// ─── NewGenerator 测试 ──────────────────────────────────────

func TestNewGenerator_Defaults(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if g.config != c {
		t.Error("config not set correctly")
	}
	if g.typeMap == nil {
		t.Error("typeMap should be initialized")
	}
	if g.likeStruct == nil {
		t.Error("likeStruct should be initialized")
	}
}

// ─── lookupPrimitive 测试 ──────────────────────────────────

func TestLookupPrimitive_BuiltinInt(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "int"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(int) should succeed")
	}
	if !info.EqualAble {
		t.Error("int should be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "Int") {
		t.Errorf("Field name = %s, should contain Int", info.Field.Name())
	}
}

func TestLookupPrimitive_BuiltinString(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "string"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(string) should succeed")
	}
	if !info.EqualAble {
		t.Error("string should be EqualAble")
	}
}

func TestLookupPrimitive_BuiltinFloat64(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "float64"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(float64) should succeed")
	}
	if info.EqualAble {
		t.Error("float64 should NOT be EqualAble")
	}
}

func TestLookupPrimitive_Bool(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "bool"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(bool) should succeed")
	}
	if !info.EqualAble {
		t.Error("bool should be EqualAble")
	}
}

func TestLookupPrimitive_Byte(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "byte"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(byte) should succeed")
	}
	if !info.EqualAble {
		t.Error("byte should be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "Byte") {
		t.Errorf("Field name = %s, should contain Byte", info.Field.Name())
	}
}

func TestLookupPrimitive_Rune(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "rune"}, nil, "", nil)
	info, ok := g.lookupPrimitive(ts)
	if !ok {
		t.Fatal("lookupPrimitive(rune) should succeed")
	}
	if !info.EqualAble {
		t.Error("rune should be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "Rune") {
		t.Errorf("Field name = %s, should contain Rune", info.Field.Name())
	}
}

func TestLookupPrimitive_NotBuiltin(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	// 有 PkgPath 的 = 非内置
	ts := parseAstTypeWithLoader(&ast.Ident{Name: "UserInfo"}, nil, "mypkg", nil)
	_, ok := g.lookupPrimitive(ts)
	if ok {
		t.Error("lookupPrimitive should fail for non-builtin type")
	}
}

// ─── lookupCustom 测试 ──────────────────────────────────────

func TestLookupCustom_FoundByFullName(t *testing.T) {
	c := NewConfig()
	c.AddMap("mypkg.MyType", "MyField", "NewMyField", true)
	g := NewGenerator(c)

	ts := &astTypeSource{name: "MyType", pkgPath: "mypkg"}
	info, ok := g.lookupCustom(ts)
	if !ok {
		t.Fatal("lookupCustom should find mypkg.MyType")
	}
	if info.Field.Name() != "MyField" {
		t.Errorf("Field = %s, want MyField", info.Field.Name())
	}
}

func TestLookupCustom_FoundByShortName(t *testing.T) {
	c := NewConfig()
	c.AddMap("MyType", "MyField", "NewMyField", false)
	g := NewGenerator(c)

	ts := &astTypeSource{name: "MyType", pkgPath: "otherpkg"}
	info, ok := g.lookupCustom(ts)
	if !ok {
		t.Fatal("lookupCustom should find by short name")
	}
	if info.EqualAble {
		t.Error("EqualAble should be false")
	}
}

func TestLookupCustom_NotFound(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "NonExistent", pkgPath: "mypkg"}
	_, ok := g.lookupCustom(ts)
	if ok {
		t.Error("lookupCustom should fail for unknown type")
	}
}

// ─── lookupBuiltinDirect 测试 ───────────────────────────────

func TestLookupBuiltinDirect_BSONObjectID(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	bsonPkg := "go.mongodb.org/mongo-driver/v2/bson"
	ts := &astTypeSource{name: "ObjectID", pkgPath: bsonPkg}
	info, ok := g.lookupBuiltinDirect(ts)
	if !ok {
		t.Fatal("should find ObjectID")
	}
	if !strings.Contains(info.Field.Name(), "ObjectID") {
		t.Errorf("Field = %s, should contain ObjectID", info.Field.Name())
	}
}

func TestLookupBuiltinDirect_BSONDecimal128(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	bsonPkg := "go.mongodb.org/mongo-driver/v2/bson"
	ts := &astTypeSource{name: "Decimal128", pkgPath: bsonPkg}
	info, ok := g.lookupBuiltinDirect(ts)
	if !ok {
		t.Fatal("should find Decimal128")
	}
	if !info.EqualAble {
		t.Error("Decimal128 should be EqualAble")
	}
}

func TestLookupBuiltinDirect_BSONAny(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	bsonPkg := "go.mongodb.org/mongo-driver/v2/bson"
	ts := &astTypeSource{name: "M", pkgPath: bsonPkg}
	info, ok := g.lookupBuiltinDirect(ts)
	if !ok {
		t.Fatal("should find bson.M")
	}
	if !info.EqualAble {
		t.Error("bson.M should be EqualAble")
	}
}

func TestLookupBuiltinDirect_WrongPkg(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "ObjectID", pkgPath: "wrong/pkg"}
	_, ok := g.lookupBuiltinDirect(ts)
	if ok {
		t.Error("should not find ObjectID in wrong package")
	}
}

func TestLookupBuiltinDirect_GeoPoint(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	geoPkg := "github.com/xpwu/go-mongodb/geo"
	ts := &astTypeSource{name: "SpherePoint", pkgPath: geoPkg}
	info, ok := g.lookupBuiltinDirect(ts)
	if !ok {
		t.Fatal("should find SpherePoint")
	}
	if !info.EqualAble {
		t.Error("SpherePoint should be EqualAble")
	}
}

func TestLookupBuiltinDirect_TimeTime(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	timePkg := "time"
	ts := &astTypeSource{name: "Time", pkgPath: timePkg}
	info, ok := g.lookupBuiltinDirect(ts)
	if !ok {
		t.Fatal("should find time.Time")
	}
	if !info.EqualAble {
		t.Error("time.Time should be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "Time") {
		t.Errorf("Field = %s, should contain Time", info.Field.Name())
	}
}

// ─── buildKind 测试 ─────────────────────────────────────────

func TestBuildKind_Int(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "MyInt", pkgPath: "mypkg", kind: reflect.Int}
	realTs := &astTypeSource{name: "MyInt", pkgPath: "mypkg"}
	info, ok := g.buildKind(ts, realTs)
	if !ok {
		t.Fatal("buildKind(int) should succeed")
	}
	if !info.EqualAble {
		t.Error("int should be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "Integer") {
		t.Errorf("Field = %s, should contain Integer", info.Field.Name())
	}
}

func TestBuildKind_String(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "MyString", pkgPath: "mypkg", kind: reflect.String}
	realTs := &astTypeSource{name: "MyString", pkgPath: "mypkg"}
	info, ok := g.buildKind(ts, realTs)
	if !ok {
		t.Fatal("buildKind(string) should succeed")
	}
	if !info.EqualAble {
		t.Error("string should be EqualAble")
	}
}

func TestBuildKind_Bool(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "MyBool", pkgPath: "mypkg", kind: reflect.Bool}
	realTs := &astTypeSource{name: "MyBool", pkgPath: "mypkg"}
	info, ok := g.buildKind(ts, realTs)
	if !ok {
		t.Fatal("buildKind(bool) should succeed")
	}
	if !info.EqualAble {
		t.Error("bool should be EqualAble")
	}
}

func TestBuildKind_Float64(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "MyFloat", pkgPath: "mypkg", kind: reflect.Float64}
	realTs := &astTypeSource{name: "MyFloat", pkgPath: "mypkg"}
	info, ok := g.buildKind(ts, realTs)
	if !ok {
		t.Fatal("buildKind(float64) should succeed")
	}
	if info.EqualAble {
		t.Error("float64 should NOT be EqualAble")
	}
}

func TestBuildKind_Interface(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "any", pkgPath: "", kind: reflect.Interface}
	realTs := &astTypeSource{name: "any", pkgPath: "", kind: reflect.Interface}
	info, ok := g.buildKind(ts, realTs)
	if !ok {
		t.Fatal("buildKind(interface) should succeed")
	}
	if info.EqualAble {
		t.Error("interface should NOT be EqualAble")
	}
	if !strings.Contains(info.Field.Name(), "BaseStruct") {
		t.Errorf("Field = %s, should contain BaseStruct", info.Field.Name())
	}
}

func TestBuildKind_Unsupported(t *testing.T) {
	tests := []struct {
		name string
		kind reflect.Kind
	}{
		{"complex64", reflect.Complex64},
		{"complex128", reflect.Complex128},
		{"chan", reflect.Chan},
		{"func", reflect.Func},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						want := "complex64/complex128/chan/func are not supported by MongoDB/BSON"
						if err.Error() != want {
							t.Errorf("panic message = %q, want %q", err.Error(), want)
						}
					} else {
						panic(r)
					}
				} else {
					t.Error("expected panic, got none")
				}
			}()

			c := NewConfig()
			g := NewGenerator(c)
			ts := &astTypeSource{name: tt.name, pkgPath: "mypkg", kind: tt.kind}
			realTs := &astTypeSource{name: tt.name, pkgPath: "mypkg", kind: tt.kind}
			_, ok := g.buildKind(ts, realTs)
			if ok {
				t.Error("buildKind should not return ok for unsupported kind")
			}
		})
	}
}

// ─── buildPtr 测试 ──────────────────────────────────────────

func TestBuildPtr_IntPtr(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	elem := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int}
	ts := &astTypeSource{name: "*int", kind: reflect.Ptr, elem: elem}
	info, ok := g.buildPtr(ts)
	if !ok {
		t.Fatal("buildPtr should succeed")
	}
	if !info.EqualAble {
		t.Error("int ptr should be EqualAble")
	}
}

func TestBuildPtr_NilElem(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "*nil", kind: reflect.Ptr, elem: nil}
	_, ok := g.buildPtr(ts)
	if ok {
		t.Error("buildPtr with nil elem should fail")
	}
}

// ─── buildSlice 测试 ────────────────────────────────────────

func TestBuildSlice_IntSlice(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	elem := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int}
	ts := &astTypeSource{name: "[]int", kind: reflect.Slice, elem: elem}
	info, ok := g.buildSlice(ts)
	if !ok {
		t.Fatal("buildSlice([]int) should succeed")
	}
	if !info.EqualAble {
		t.Error("[]int should be EqualAble")
	}
}

func TestBuildSlice_StringSlice(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	elem := &astTypeSource{name: "string", pkgPath: "", kind: reflect.String}
	ts := &astTypeSource{name: "[]string", kind: reflect.Slice, elem: elem}
	info, ok := g.buildSlice(ts)
	if !ok {
		t.Fatal("buildSlice([]string) should succeed")
	}
	if !info.EqualAble {
		t.Error("[]string should be EqualAble")
	}
}

func TestBuildSlice_NilElem(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	ts := &astTypeSource{name: "[]nil", kind: reflect.Slice, elem: nil}
	_, ok := g.buildSlice(ts)
	if ok {
		t.Error("buildSlice with nil elem should fail")
	}
}

func TestBuildSlice_2DArray(t *testing.T) {
	c := NewConfig()
	g := NewGenerator(c)
	inner := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int}
	middle := &astTypeSource{name: "[]int", kind: reflect.Slice, elem: inner}
	ts := &astTypeSource{name: "[][]int", kind: reflect.Slice, elem: middle}
	info, ok := g.buildSlice(ts)
	if !ok {
		t.Fatal("buildSlice([][]int) should succeed")
	}
	if !info.EqualAble {
		t.Error("[][]int should be EqualAble")
	}
}

// ─── buildStruct 集成测试 ──────────────────────────────────

func TestBuildStruct_GeneratesFile(t *testing.T) {
	src := `
package mypkg

type User struct {
	Name string ` + "`bson:\"name\"`" + `
	Age  int    ` + "`bson:\"age\"`" + `
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "user.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "User"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	info, ok := g.buildStruct(ts)
	if !ok {
		t.Fatal("buildStruct should succeed")
	}
	if info.Field.Name() != "UserField" {
		t.Errorf("Field = %s, want UserField", info.Field.Name())
	}

	// 验证文件是否生成
	expectedFile := filepath.Join(dir, "zUserField.go")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("generated file not found: %s", expectedFile)
	}
}

func TestBuildStruct_WithPreserveField(t *testing.T) {
	src := `
package mypkg

type Product struct {
	ProductName string
	Price       float64
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "product.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "Product"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	c.PreserveField = true
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	info, ok := g.buildStruct(ts)
	if !ok {
		t.Fatal("buildStruct should succeed with PreserveField")
	}
	if info.EqualAble {
		t.Error("Product should be NotEqualAble (string is but float64 is not, mixed)")
	}
}

func TestBuildStruct_CachesResult(t *testing.T) {
	src := `
package mypkg

type Item struct {
	Value int
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "item.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "Item"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	info1, _ := g.buildStruct(ts)
	info2, _ := g.buildStruct(ts)
	if info1.Field.Name() != info2.Field.Name() {
		t.Error("cached result should be consistent")
	}
}

func TestBuildStruct_WithTimeField(t *testing.T) {
	src := `
package mypkg

import "time"

type Event struct {
	CreatedAt time.Time ` + "`bson:\"created_at\"`" + `
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "event.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "Event"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	info, ok := g.buildStruct(ts)
	if !ok {
		t.Fatal("buildStruct should succeed")
	}
	if !info.EqualAble {
		t.Error("Event should be EqualAble (time.Time is EqualAble)")
	}

	// 验证生成的文件内容包含 TimeField
	expectedFile := filepath.Join(dir, "zEventField.go")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "TimeField") {
		t.Error("generated file should contain TimeField")
	}
	if !strings.Contains(contentStr, "NewTimeField") {
		t.Error("generated file should contain NewTimeField")
	}
}

// ─── Generate 端到端测试 ───────────────────────────────────

func TestGenerate_SimpleStruct(t *testing.T) {
	src := `
package mypkg

type Simple struct {
	ID   string ` + "`bson:\"_id\"`" + `
	Data string ` + "`bson:\"data\"`" + `
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "simple.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "Simple"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	subDir := g.Generate(ts)
	if subDir != "" {
		t.Errorf("subDir = %s, want empty (same package)", subDir)
	}

	// 验证生成的文件内容
	expectedFile := filepath.Join(dir, "zSimpleField.go")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "SimpleField") {
		t.Error("generated file should contain SimpleField")
	}
	if !strings.Contains(contentStr, "NewSimpleField") {
		t.Error("generated file should contain NewSimpleField")
	}
	if !strings.Contains(contentStr, "package mypkg") {
		t.Error("generated file should be in package mypkg")
	}
}

func TestGenerate_WithCustomMap(t *testing.T) {
	src := `
package mypkg

type CustomType int

type Container struct {
	ID   string      ` + "`bson:\"_id\"`" + `
	Data CustomType ` + "`bson:\"data\"`" + `
}
`
	loader := newTestLoader(t)
	registerTestFile(t, loader, "container.go", src)

	ts := parseAstTypeWithLoader(&ast.Ident{Name: "Container"}, nil, "mypkg", loader)
	_, err := loader.LoadPackage("mypkg")
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	ts.EnsureFields()

	dir := loader.goModDir
	c := NewConfig()
	c.Dir = dir
	c.Pkg = "mypkg"
	c.AddMap("mypkg.CustomType", "IntField", "NewIntField", true)
	g := NewGenerator(c)
	g.outputDir = dir
	g.targetPkg = "mypkg"

	subDir := g.Generate(ts)
	_ = subDir

	expectedFile := filepath.Join(dir, "zContainerField.go")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "ContainerField") {
		t.Error("should contain ContainerField")
	}
}

// ─── TypeInfo 测试 ──────────────────────────────────────────

func TestTypeInfo_FieldAccessors(t *testing.T) {
	tr := typeRef{name: "TestField", pkg: "testpkg"}
	if tr.Name() != "TestField" {
		t.Errorf("Name = %s", tr.Name())
	}
	if tr.PkgPath() != "testpkg" {
		t.Errorf("PkgPath = %s", tr.PkgPath())
	}
}

func TestTypeInfo_EmptyPkg(t *testing.T) {
	tr := typeRef{name: "BuiltinField", pkg: ""}
	if tr.PkgPath() != "" {
		t.Errorf("PkgPath should be empty")
	}
}

// ─── template 数据验证 ──────────────────────────────────────

func TestTemplateData_StructFields(t *testing.T) {
	td := templateData{
		Pkg:          "mypkg",
		TypePkg:      "mypkg.",
		Name:         "User",
		FieldAlias:   "fields.",
		MongoAlias:   "field.",
		FilterAlias:  "filter.",
		UpdaterAlias: "updater.",
		Imports:      []importTemp{{Alias: "fields", Import: "github.com/xpwu/go-mongodb/fields"}},
		Fields: []templateField{
			{MethodName: "Name", FieldName: "StringField", TagName: "name"},
			{MethodName: "Age", FieldName: "IntField", TagName: "age"},
		},
		Inlines:   []templateInline{},
		EqualAble: true,
	}

	var buf bytes.Buffer
	if err := structTemplate.Execute(&buf, td); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "type UserField") {
		t.Error("output should contain UserField type")
	}
	if !strings.Contains(output, "NewUserField") {
		t.Error("output should contain NewUserField")
	}
	if !strings.Contains(output, "NameF()") {
		t.Error("output should contain NameF method")
	}
}

// ─── reflect.Kind 对照测试 ─────────────────────────────────

func TestKind_ConsistencyWithReflect(t *testing.T) {
	tests := map[string]reflect.Kind{
		"int":     reflect.Int,
		"int8":    reflect.Int8,
		"int16":   reflect.Int16,
		"int32":   reflect.Int32,
		"int64":   reflect.Int64,
		"uint":    reflect.Uint,
		"uint8":   reflect.Uint8,
		"uint16":  reflect.Uint16,
		"uint32":  reflect.Uint32,
		"uint64":  reflect.Uint64,
		"float32": reflect.Float32,
		"float64": reflect.Float64,
		"string":  reflect.String,
		"bool":    reflect.Bool,
		"byte":    reflect.Uint8,
		"rune":    reflect.Int32,
	}
	for name, expected := range tests {
		got := kindFromName(name)
		if got != expected {
			t.Errorf("kindFromName(%s) = %v, want %v", name, got, expected)
		}
	}
}
