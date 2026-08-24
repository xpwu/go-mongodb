package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ─── 辅助函数 ───────────────────────────────────────────────

func writeTempFile(t *testing.T, name, content string) (dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	return dir, func() {}
}

func parseTempFile(t *testing.T, dir, fileName string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, fileName), nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse file failed: %v", err)
	}
	return file
}

// ─── kindFromName ────────────────────────────────────────────

func TestKindFromName(t *testing.T) {
	tests := []struct {
		input    string
		expected reflect.Kind
	}{
		{"int", reflect.Int}, {"int8", reflect.Int8}, {"int16", reflect.Int16},
		{"int32", reflect.Int32}, {"int64", reflect.Int64},
		{"uint", reflect.Uint}, {"uint8", reflect.Uint8}, {"uint16", reflect.Uint16},
		{"uint32", reflect.Uint32}, {"uint64", reflect.Uint64},
		{"float32", reflect.Float32}, {"float64", reflect.Float64},
		{"string", reflect.String}, {"bool", reflect.Bool},
		{"any", reflect.Interface}, {"interface{}", reflect.Interface},
		{"byte", reflect.Uint8},
		{"rune", reflect.Int32},
		{"SomeStruct", reflect.Struct}, // 未知 → Struct
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := kindFromName(tt.input); got != tt.expected {
				t.Errorf("kindFromName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// ─── isBuiltinKind ───────────────────────────────────────────

func TestIsBuiltinKind(t *testing.T) {
	for _, k := range []reflect.Kind{
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Interface,
	} {
		if !isBuiltinKind(k) {
			t.Errorf("isBuiltinKind(%v) = false, want true", k)
		}
	}
	for _, k := range []reflect.Kind{
		reflect.Struct, reflect.Slice, reflect.Array, reflect.Ptr, reflect.Map, reflect.Chan, reflect.Func,
	} {
		if isBuiltinKind(k) {
			t.Errorf("isBuiltinKind(%v) = true, want false", k)
		}
	}
}

// ─── parseAstTypeWithLoader ──────────────────────────────────

func TestParseAstType_BuiltinIdent(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.Ident{Name: "string"}, map[string]string{}, "mypkg", GetLoader())
	if r.Name() != "string" || r.PkgPath() != "" || r.Kind() != reflect.String || !r.IsBuiltin() {
		t.Errorf("unexpected: %+v kind=%v builtin=%v", r.Name(), r.Kind(), r.IsBuiltin())
	}
}

func TestParseAstType_CustomIdent(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.Ident{Name: "MyStruct"}, map[string]string{}, "mypkg", GetLoader())
	if r.Name() != "MyStruct" || r.PkgPath() != "mypkg" || r.Kind() != reflect.Struct || r.IsBuiltin() {
		t.Errorf("unexpected: %+v kind=%v builtin=%v", r.Name(), r.Kind(), r.IsBuiltin())
	}
}

func TestParseAstType_InterfaceFromTarget(t *testing.T) {
	loader := GetLoader()
	pkg := &loadedPackage{
		fset: token.NewFileSet(), files: nil,
		importMap: make(map[string]string), typeElems: make(map[string]*astTypeSource),
		types: make(map[string]*ast.StructType), aliasTargets: make(map[string]*astTypeSource),
		typeDefTargets: make(map[string]string), interfaceTargets: map[string]bool{"MyIface": true},
		underlyingCache: make(map[string]*astTypeSource),
	}
	loader.loaded["p"] = pkg
	r := parseAstTypeWithLoader(&ast.Ident{Name: "MyIface"}, map[string]string{}, "p", loader)
	if r.Kind() != reflect.Interface || r.PkgPath() != "" {
		t.Errorf("unexpected: kind=%v pkg=%q", r.Kind(), r.PkgPath())
	}
}

func TestParseAstType_SelectorExpr(t *testing.T) {
	imp := map[string]string{"bson": "go.mongodb.org/mongo-driver/v2/bson"}
	r := parseAstTypeWithLoader(&ast.SelectorExpr{X: &ast.Ident{Name: "bson"}, Sel: &ast.Ident{Name: "ObjectID"}}, imp, "mypkg", GetLoader())
	if r.Name() != "ObjectID" || r.PkgPath() != "go.mongodb.org/mongo-driver/v2/bson" {
		t.Errorf("unexpected: %+v pkg=%q", r.Name(), r.PkgPath())
	}
}

func TestParseAstType_SelectorExpr_Any(t *testing.T) {
	imp := map[string]string{"bson": "go.mongodb.org/mongo-driver/v2/bson"}
	r := parseAstTypeWithLoader(&ast.SelectorExpr{X: &ast.Ident{Name: "bson"}, Sel: &ast.Ident{Name: "any"}}, imp, "mypkg", GetLoader())
	if r.Kind() != reflect.Interface || r.PkgPath() != "" {
		t.Errorf("any should be Interface with empty pkg, got kind=%v pkg=%q", r.Kind(), r.PkgPath())
	}
}

func TestParseAstType_StarExpr(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.StarExpr{X: &ast.Ident{Name: "string"}}, map[string]string{}, "mypkg", GetLoader())
	if r.Kind() != reflect.Ptr || r.Elem() == nil || r.Elem().Name() != "string" {
		t.Errorf("unexpected Ptr: kind=%v elem=%v", r.Kind(), r.Elem())
	}
}

func TestParseAstType_ArrayType(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.ArrayType{Elt: &ast.Ident{Name: "int"}}, map[string]string{}, "mypkg", GetLoader())
	if r.Kind() != reflect.Slice || r.Elem() == nil || r.Elem().Kind() != reflect.Int {
		t.Errorf("unexpected Slice: kind=%v elem=%v", r.Kind(), r.Elem())
	}
}

func TestParseAstType_InterfaceType(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.InterfaceType{}, map[string]string{}, "mypkg", GetLoader())
	if r.Kind() != reflect.Interface {
		t.Errorf("InterfaceType kind=%v, want Interface", r.Kind())
	}
}

// ─── parseAstFieldWithLoader ────────────────────────────────

func TestParseAstField_Basic(t *testing.T) {
	f := &ast.Field{
		Names: []*ast.Ident{{Name: "UserName"}},
		Type:  &ast.Ident{Name: "string"},
		Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`bson:\"user_name\"`"},
	}
	fs := parseAstFieldWithLoader(f, map[string]string{}, "mypkg", GetLoader())
	if fs.Name() != "UserName" || fs.Type() == nil || fs.Type().Name() != "string" ||
		fs.Tag() != "bson:\"user_name\"" || !fs.IsExported() {
		t.Errorf("unexpected field: name=%q tag=%q exported=%v", fs.Name(), fs.Tag(), fs.IsExported())
	}
}

func TestParseAstField_NotExported(t *testing.T) {
	f := &ast.Field{Names: []*ast.Ident{{Name: "private"}}, Type: &ast.Ident{Name: "int"}}
	if parseAstFieldWithLoader(f, map[string]string{}, "mypkg", GetLoader()).IsExported() {
		t.Error("private field should not be exported")
	}
}

func TestParseAstField_AnonymousStruct(t *testing.T) {
	f := &ast.Field{Type: &ast.StructType{
		Fields: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "X"}}, Type: &ast.Ident{Name: "int"}}}},
	}}
	fs := parseAstFieldWithLoader(f, map[string]string{}, "mypkg", GetLoader())
	if fs.Name() != "" {
		t.Errorf("anonymous struct field name = %q, want empty", fs.Name())
	}
}

// ─── collectImports ──────────────────────────────────────────

func TestCollectImports(t *testing.T) {
	dir, _ := writeTempFile(t, "imports.go", `package mypkg
import (
	"fmt"
	bson "go.mongodb.org/mongo-driver/v2/bson"
	"github.com/xpwu/go-mongodb/fields"
)`)
	defer os.RemoveAll(dir)
	file := parseTempFile(t, dir, "imports.go")
	imp := make(map[string]string)
	collectImports(file, imp)
	if imp["fmt"] != "fmt" {
		t.Errorf("fmt = %q", imp["fmt"])
	}
	if imp["bson"] != "go.mongodb.org/mongo-driver/v2/bson" {
		t.Errorf("bson = %q", imp["bson"])
	}
	if imp["fields"] != "github.com/xpwu/go-mongodb/fields" {
		t.Errorf("fields = %q", imp["fields"])
	}
}

// ─── collectTypeInfo ──────────────────────────────────────────

func newTestPkg() *loadedPackage {
	return &loadedPackage{
		fset: token.NewFileSet(), files: nil,
		importMap: make(map[string]string), typeElems: make(map[string]*astTypeSource),
		types: make(map[string]*ast.StructType), aliasTargets: make(map[string]*astTypeSource),
		typeDefTargets: make(map[string]string), interfaceTargets: make(map[string]bool),
		underlyingCache: make(map[string]*astTypeSource),
	}
}

func TestCollectTypeInfo_Struct(t *testing.T) {
	dir, _ := writeTempFile(t, "u.go", `package mypkg
type UserInfo struct {
	ID   string `+"`bson:\"_id\"`"+`
	Name string `+"`bson:\"name\"`"+`
}`)
	defer os.RemoveAll(dir)
	pkg := newTestPkg()
	file := parseTempFile(t, dir, "u.go")
	collectTypeInfo(file, pkg, "mypkg", GetLoader())
	if _, ok := pkg.types["UserInfo"]; !ok {
		t.Fatal("UserInfo not collected")
	}
	if len(pkg.types["UserInfo"].Fields.List) != 2 {
		t.Errorf("field count = %d, want 2", len(pkg.types["UserInfo"].Fields.List))
	}
}

func TestCollectTypeInfo_Alias(t *testing.T) {
	dir, _ := writeTempFile(t, "a.go", `package mypkg
type Age = int`)
	defer os.RemoveAll(dir)
	pkg := newTestPkg()
	file := parseTempFile(t, dir, "a.go")
	collectTypeInfo(file, pkg, "mypkg", GetLoader())
	if len(pkg.aliasTargets) != 1 {
		t.Fatalf("alias count = %d, want 1", len(pkg.aliasTargets))
	}
	if pkg.aliasTargets["Age"].Name() != "int" {
		t.Errorf("Age → %q, want int", pkg.aliasTargets["Age"].Name())
	}
}

func TestCollectTypeInfo_TypeDef(t *testing.T) {
	dir, _ := writeTempFile(t, "t.go", `package mypkg
type MyInt int`)
	defer os.RemoveAll(dir)
	pkg := newTestPkg()
	file := parseTempFile(t, dir, "t.go")
	collectTypeInfo(file, pkg, "mypkg", GetLoader())
	if v := pkg.typeDefTargets["MyInt"]; v != "int" {
		t.Errorf("typeDef[MyInt] = %q, want int", v)
	}
}

func TestCollectTypeInfo_Slice(t *testing.T) {
	dir, _ := writeTempFile(t, "s.go", `package mypkg
type IntSlice []int`)
	defer os.RemoveAll(dir)
	pkg := newTestPkg()
	file := parseTempFile(t, dir, "s.go")
	collectTypeInfo(file, pkg, "mypkg", GetLoader())
	elem, ok := pkg.typeElems["IntSlice"]
	if !ok || elem.Name() != "int" {
		t.Errorf("IntSlice elem = %+v", elem)
	}
}

func TestCollectTypeInfo_Interface(t *testing.T) {
	dir, _ := writeTempFile(t, "i.go", `package mypkg
type MyIface interface {
	Do() error
}`)
	defer os.RemoveAll(dir)
	pkg := newTestPkg()
	file := parseTempFile(t, dir, "i.go")
	collectTypeInfo(file, pkg, "mypkg", GetLoader())
	if !pkg.interfaceTargets["MyIface"] {
		t.Error("MyIface not in interfaceTargets")
	}
}

// ─── astTypeSource 方法 ──────────────────────────────────────

func TestAstTypeSource_Basic(t *testing.T) {
	ts := &astTypeSource{
		name: "T", pkgPath: "mypkg/sub", kind: reflect.Struct,
		fields: []*astFieldSource{{name: "A", exported: true}, {name: "b", exported: false}},
		loader: GetLoader(),
	}
	if ts.Name() != "T" || ts.PkgPath() != "mypkg/sub" || ts.Kind() != reflect.Struct {
		t.Errorf("basic attrs wrong: %+v", ts)
	}
	if ts.NumField() != 2 || ts.Elem() != nil || ts.IsBuiltin() {
		t.Errorf("fields/elem/builtin wrong")
	}
	if ts.Field(0).Name() != "A" || ts.Field(1).Name() != "b" {
		t.Errorf("Field(i) wrong")
	}
	if ts.Field(99) != nil {
		t.Errorf("Field(99) should be nil")
	}
}

func TestAstTypeSource_Kind_Underlying(t *testing.T) {
	loader := GetLoader()
	intT := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int, loader: loader}
	pkg := newTestPkg()
	pkg.aliasTargets = map[string]*astTypeSource{"MyAlias": intT}
	loader.loaded["p"] = pkg
	ts := &astTypeSource{name: "MyAlias", pkgPath: "p", kind: reflect.Struct, loader: loader}
	if ts.Kind() != reflect.Int {
		t.Errorf("Kind should penetrate to int, got %v", ts.Kind())
	}
}

// ─── astFieldSource ───────────────────────────────────────────

func TestAstFieldSource_Basic(t *testing.T) {
	typ := &astTypeSource{name: "string", pkgPath: "", kind: reflect.String, loader: GetLoader()}
	fs := &astFieldSource{name: "N", typ: typ, tag: "bson:\"n\"", exported: true}
	if fs.Name() != "N" || fs.Type() != typ || fs.Tag() != "bson:\"n\"" || !fs.IsExported() {
		t.Errorf("field attrs wrong: %+v", fs)
	}
}

// ─── Underlying 穿透 ──────────────────────────────────────────

func TestUnderlying_Alias(t *testing.T) {
	loader := GetLoader()
	intT := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int, loader: loader}
	pkg := newTestPkg()
	pkg.aliasTargets = map[string]*astTypeSource{"MyInt": intT}
	loader.loaded["p"] = pkg
	ts := &astTypeSource{name: "MyInt", pkgPath: "p", kind: reflect.Int, loader: loader}
	next, alias := ts.Underlying()
	if next == nil || !alias || next.Name() != "int" {
		t.Errorf("alias穿透失败: next=%+v alias=%v", next, alias)
	}
}

func TestUnderlying_TypeDef(t *testing.T) {
	loader := GetLoader()
	pkg := newTestPkg()
	pkg.typeDefTargets = map[string]string{"MyType": "int"}
	loader.loaded["p"] = pkg
	ts := &astTypeSource{name: "MyType", pkgPath: "p", kind: reflect.Int, loader: loader}
	next, alias := ts.Underlying()
	if next == nil || alias || next.Name() != "int" {
		t.Errorf("typedef穿透失败: next=%+v alias=%v", next, alias)
	}
}

func TestUnderlying_Slice(t *testing.T) {
	loader := GetLoader()
	elemT := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int, loader: loader}
	pkg := newTestPkg()
	pkg.typeElems = map[string]*astTypeSource{"IntList": elemT}
	loader.loaded["p"] = pkg
	ts := &astTypeSource{name: "IntList", pkgPath: "p", kind: reflect.Slice, loader: loader}
	next, alias := ts.Underlying()
	if next == nil || alias || next.Kind() != reflect.Slice || next.Elem() == nil || next.Elem().Name() != "int" {
		t.Errorf("slice穿透失败: next=%+v alias=%v", next, alias)
	}
}

func TestUnderlying_Terminal(t *testing.T) {
	loader := GetLoader()
	pkg := newTestPkg()
	loader.loaded["p"] = pkg
	ts := &astTypeSource{name: "Plain", pkgPath: "p", kind: reflect.Struct, loader: loader}
	next, alias := ts.Underlying()
	if next != nil || alias {
		t.Errorf("terminal应为nil/false: next=%+v alias=%v", next, alias)
	}
}

// ─── EnsureFields ──────────────────────────────────────────────

func TestEnsureFields_Loads(t *testing.T) {
	dir, _ := writeTempFile(t, "p.go", `package testpkg
type Person struct {
	Name string `+"`bson:\"name\"`"+`
	Age  int    `+"`bson:\"age\"`"+`
}`)
	defer os.RemoveAll(dir)
	loader := GetLoader()
	pkg, _ := loader.parsePackageDir(dir)
	collectTypeInfo(parseTempFile(t, dir, "p.go"), pkg, "testpkg", loader)
	loader.loaded["testpkg"] = pkg
	ts := &astTypeSource{name: "Person", pkgPath: "testpkg", kind: reflect.Struct, loader: loader}
	ts.EnsureFields()
	if ts.NumField() != 2 || !ts.loaded {
		t.Errorf("EnsureFields failed: n=%d loaded=%v", ts.NumField(), ts.loaded)
	}
	ts.EnsureFields() // 幂等
}

func TestEnsureFields_NonStruct(t *testing.T) {
	ts := &astTypeSource{name: "int", pkgPath: "", kind: reflect.Int, loader: GetLoader()}
	ts.EnsureFields()
	if !ts.loaded || ts.NumField() != 0 {
		t.Errorf("non-struct EnsureFields: loaded=%v n=%d", ts.loaded, ts.NumField())
	}
}

// ─── findStructInFile ──────────────────────────────────────────

func TestFindStructInFile(t *testing.T) {
	dir, _ := writeTempFile(t, "m.go", `package mypkg
type Foo struct { X int }
type Bar struct { Y string }`)
	defer os.RemoveAll(dir)
	file := parseTempFile(t, dir, "m.go")
	imp := map[string]string{}
	if ts, _ := findStructInFile(file, "Foo", imp, "mypkg", GetLoader()); ts == nil || ts.Name() != "Foo" {
		t.Error("Foo not found")
	}
	if ts, _ := findStructInFile(file, "Bar", imp, "mypkg", GetLoader()); ts == nil || ts.Name() != "Bar" {
		t.Error("Bar not found")
	}
	if ts, _ := findStructInFile(file, "Nope", imp, "mypkg", GetLoader()); ts != nil {
		t.Error("Nope should be nil")
	}
}

// ─── buildStructTypeSource ────────────────────────────────────

func TestBuildStructTypeSource(t *testing.T) {
	st := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{
		{Names: []*ast.Ident{{Name: "A"}}, Type: &ast.Ident{Name: "int"}},
		{Names: []*ast.Ident{{Name: "B"}}, Type: &ast.Ident{Name: "string"}},
	}}}
	ts := buildStructTypeSource("S", "mypkg", st, map[string]string{}, GetLoader())
	if ts.Name() != "S" || ts.Kind() != reflect.Struct || ts.NumField() != 2 {
		t.Errorf("buildStructTypeSource wrong: %+v", ts)
	}
}

// ─── ParseStructFromFile 集成 ─────────────────────────────────

func TestParseStructFromFile_Integration(t *testing.T) {
	dir, _ := writeTempFile(t, "u.go", `package mypkg
type User struct {
	ID   string `+"`bson:\"_id\"`"+`
	Name string `+"`bson:\"name\"`"+`
	Age  int    `+"`bson:\"age\"`"+`
}`)
	defer os.RemoveAll(dir)
	ResetLoader()
	ts, err := ParseStructFromFile(dir, "User")
	if err != nil || ts == nil || ts.Name() != "User" {
		t.Fatalf("ParseStructFromFile failed: err=%v ts=%+v", err, ts)
	}
	ts.EnsureFields()
	if ts.NumField() != 3 {
		t.Errorf("NumField=%d, want 3", ts.NumField())
	}
}

func TestParseStructFromFile_NotFound(t *testing.T) {
	dir, _ := writeTempFile(t, "u.go", `package mypkg
type User struct { Name string }`)
	defer os.RemoveAll(dir)
	ResetLoader()
	ts, err := ParseStructFromFile(dir, "Nope")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ts != nil {
		t.Error("expected nil for non-existent struct")
	}
}

// ─── ScanDir / ScanFile ───────────────────────────────────────

func TestScanDir_WithGenerate(t *testing.T) {
	dir, _ := writeTempFile(t, "u.go", `package mypkg
//go:generate gomongodbgen
type User struct { Name string }`)
	defer os.RemoveAll(dir)
	ResetLoader()
	res, err := ScanDir(dir)
	if err != nil || len(res.Structs) != 1 || res.Structs[0].Name != "User" {
		t.Fatalf("ScanDir failed: err=%v res=%+v", err, res)
	}
}

func TestScanDir_WithoutGenerate(t *testing.T) {
	dir, _ := writeTempFile(t, "u.go", `package mypkg
type User struct { Name string }`)
	defer os.RemoveAll(dir)
	ResetLoader()
	res, err := ScanDir(dir)
	if err != nil || len(res.Structs) != 0 {
		t.Fatalf("expected 0 structs, got err=%v res=%+v", err, res)
	}
}

func TestScanFile_InlineStruct(t *testing.T) {
	dir, _ := writeTempFile(t, "o.go", `package mypkg
//go:generate gomongodbgen
type Outer struct {
	Inner struct { X int }
}`)
	defer os.RemoveAll(dir)
	res, err := scanFile(filepath.Join(dir, "o.go"))
	if err != nil || len(res.Structs) == 0 {
		t.Fatalf("scanFile inline failed: err=%v res=%+v", err, res)
	}
}

// ─── TypeLoader ────────────────────────────────────────────────

func TestTypeLoader_EmptyPkgPath(t *testing.T) {
	_, err := GetLoader().LoadPackage("")
	if err == nil {
		t.Error("empty pkgPath should error")
	}
}

// ─── readModulePath / findGoModDir ────────────────────────────

func TestReadModulePath(t *testing.T) {
	dir, _ := writeTempFile(t, "go.mod", "module github.com/test/mymod\n\ngo 1.21\n")
	defer os.RemoveAll(dir)
	mp, err := readModulePath(dir)
	if err != nil || mp != "github.com/test/mymod" {
		t.Errorf("readModulePath = %q err=%v, want github.com/test/mymod", mp, err)
	}
}

func TestReadModulePath_NotFound(t *testing.T) {
	dir := t.TempDir()
	if _, err := readModulePath(dir); err == nil {
		t.Error("expected error for dir without go.mod")
	}
}

func TestFindGoModDir(t *testing.T) {
	dir, _ := writeTempFile(t, "go.mod", "module test\n")
	defer os.RemoveAll(dir)
	if found := FindGoModDir(dir); found == "" {
		t.Error("findGoModDir should find dir")
	}
}

// ─── resolvePkgPath ────────────────────────────────────────────

func TestResolvePkgPath_WithGoMod(t *testing.T) {
	dir, _ := writeTempFile(t, "go.mod", "module github.com/test/proj\n\ngo 1.21\n")
	defer os.RemoveAll(dir)
	if got := resolvePkgPath(dir); got != "github.com/test/proj" {
		t.Errorf("resolvePkgPath = %q, want github.com/test/proj", got)
	}
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	if got := resolvePkgPath(sub); got != "github.com/test/proj/sub" {
		t.Errorf("resolvePkgPath(sub) = %q, want .../sub", got)
	}
}

// ─── parsePackageDir ──────────────────────────────────────────

func TestParsePackageDir_ExcludesTestAndGen(t *testing.T) {
	dir, _ := writeTempFile(t, "real.go", `package mypkg
type Real struct { X int }`)
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "real_test.go"), []byte(`package mypkg
type TestOnly struct { Y string }`), 0644)
	os.WriteFile(filepath.Join(dir, "zRealField.go"), []byte(`package mypkg
type Gen struct { Z int }`), 0644)

	pkg, err := GetLoader().parsePackageDir(dir)
	if err != nil {
		t.Fatalf("parsePackageDir err: %v", err)
	}

	// 只应该收录 real.go，排除 _test.go 和 z*Field.go
	if len(pkg.files) != 1 {
		t.Errorf("expected 1 file, got %d", len(pkg.files))
	}
}

func TestParsePackageDir_Empty(t *testing.T) {
	dir := t.TempDir()
	if _, err := GetLoader().parsePackageDir(dir); err == nil {
		t.Error("empty dir should error")
	}
}

// ─── registerFileToLoader ─────────────────────────────────────

func TestRegisterFileToLoader(t *testing.T) {
	dir, _ := writeTempFile(t, "a.go", `package p
type A struct { X int }`)
	defer os.RemoveAll(dir)
	file := parseTempFile(t, dir, "a.go")
	loader := GetLoader()
	registerFileToLoader(loader, "p", file, filepath.Join(dir, "a.go"))
	pkg, ok := loader.loaded["p"]
	if !ok || len(pkg.files) != 1 {
		if _, ok := pkg.types["A"]; !ok {
			t.Errorf("register failed: ok=%v files=%d types_A=%v", ok, len(pkg.files), pkg.types["A"])
		}
	}

	// 重复注册不重复添加
	registerFileToLoader(loader, "p", file, filepath.Join(dir, "a.go"))
	if len(pkg.files) != 1 {
		t.Errorf("duplicate register: files=%d, want 1", len(pkg.files))
	}
}

// ─── byte / rune 内置别名 ────────────────────────────────────

func TestKindFromName_ByteAndRune(t *testing.T) {
	if got := kindFromName("byte"); got != reflect.Uint8 {
		t.Errorf("kindFromName(\"byte\") = %v, want Uint8", got)
	}
	if got := kindFromName("rune"); got != reflect.Int32 {
		t.Errorf("kindFromName(\"rune\") = %v, want Int32", got)
	}
}

func TestParseAstType_Byte(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.Ident{Name: "byte"}, map[string]string{}, "mypkg", GetLoader())
	if r.Name() != "byte" || r.PkgPath() != "" || r.Kind() != reflect.Uint8 || !r.IsBuiltin() {
		t.Errorf("byte: name=%q pkg=%q kind=%v builtin=%v, want byte/empty/Uint8/true",
			r.Name(), r.PkgPath(), r.Kind(), r.IsBuiltin())
	}
}

func TestParseAstType_Rune(t *testing.T) {
	r := parseAstTypeWithLoader(&ast.Ident{Name: "rune"}, map[string]string{}, "mypkg", GetLoader())
	if r.Name() != "rune" || r.PkgPath() != "" || r.Kind() != reflect.Int32 || !r.IsBuiltin() {
		t.Errorf("rune: name=%q pkg=%q kind=%v builtin=%v, want rune/empty/Int32/true",
			r.Name(), r.PkgPath(), r.Kind(), r.IsBuiltin())
	}
}

func TestParseAstField_ByteField(t *testing.T) {
	f := &ast.Field{
		Names: []*ast.Ident{{Name: "Data"}},
		Type:  &ast.Ident{Name: "byte"},
		Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`bson:\"data\"`"},
	}
	fs := parseAstFieldWithLoader(f, map[string]string{}, "mypkg", GetLoader())
	if fs.Type() == nil || fs.Type().Kind() != reflect.Uint8 || !fs.Type().IsBuiltin() {
		t.Errorf("byte field: kind=%v builtin=%v, want Uint8/true",
			fs.Type().Kind(), fs.Type().IsBuiltin())
	}
}
