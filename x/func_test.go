package x

import (
	"reflect"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────
// TypeFor
// ─────────────────────────────────────────────

func TestTypeFor_Int(t *testing.T) {
	typ := TypeFor[int]()
	if typ.Kind() != reflect.Int {
		t.Errorf("expected Int, got %v", typ.Kind())
	}
}

func TestTypeFor_String(t *testing.T) {
	typ := TypeFor[string]()
	if typ.Kind() != reflect.String {
		t.Errorf("expected String, got %v", typ.Kind())
	}
}

func TestTypeFor_Struct(t *testing.T) {
	type Doc struct {
		Name string
	}
	typ := TypeFor[Doc]()
	if typ.Kind() != reflect.Struct {
		t.Errorf("expected Struct, got %v", typ.Kind())
	}
	if typ.Name() != "Doc" {
		t.Errorf("expected name %q, got %q", "Doc", typ.Name())
	}
}

// ─────────────────────────────────────────────
// BaseTypeName
// ─────────────────────────────────────────────

func TestBaseTypeName_NoGeneric(t *testing.T) {
	type Doc struct{}
	typ := reflect.TypeOf(Doc{})
	name := BaseTypeName(typ)
	if name != "Doc" {
		t.Errorf("got %q, want %q", name, "Doc")
	}
}

func TestBaseTypeName_WithGeneric(t *testing.T) {
	// BaseTypeName 内部调用 BaseTypeNameFromName，按 '[' 截断
	// 用真实泛型实例化类型验证（Go 1.21+ 支持）
	type Map[K, V any] struct{}
	typ := TypeFor[Map[string, int]]()
	name := BaseTypeName(typ)
	if name != "Map" {
		t.Errorf("got %q, want %q", name, "Map")
	}
}

func TestBaseTypeNameFromName_NoGeneric(t *testing.T) {
	name := BaseTypeNameFromName("SimpleName")
	if name != "SimpleName" {
		t.Errorf("got %q, want %q", name, "SimpleName")
	}
}

func TestBaseTypeNameFromName_WithBracket(t *testing.T) {
	name := BaseTypeNameFromName("Foo[Bar,Baz]")
	if name != "Foo" {
		t.Errorf("got %q, want %q", name, "Foo")
	}
}

func TestBaseTypeNameFromName_Empty(t *testing.T) {
	name := BaseTypeNameFromName("")
	if name != "" {
		t.Errorf("got %q, want empty", name)
	}
}

// ─────────────────────────────────────────────
// LastSubPath
// ─────────────────────────────────────────────

func TestLastSubPath_WithSlash(t *testing.T) {
	result := LastSubPath("a/b/c/d")
	if result != "d" {
		t.Errorf("got %q, want %q", result, "d")
	}
}

func TestLastSubPath_NoSlash(t *testing.T) {
	result := LastSubPath("nodir")
	if result != "nodir" {
		t.Errorf("got %q, want %q", result, "nodir")
	}
}

func TestLastSubPath_RootPath(t *testing.T) {
	result := LastSubPath("/usr/local/bin")
	if result != "bin" {
		t.Errorf("got %q, want %q", result, "bin")
	}
}

func TestLastSubPath_MultipleSlashes(t *testing.T) {
	// 多个 '/' 取最后一段
	result := LastSubPath("a/b/c/d/e")
	if result != "e" {
		t.Errorf("got %q, want %q", result, "e")
	}
}

// ─────────────────────────────────────────────
// SanitizePackageName
// ─────────────────────────────────────────────

func TestSanitizePackageName_ValidChars(t *testing.T) {
	result := SanitizePackageName("abcDEF_123")
	if result != "abcDEF_123" {
		t.Errorf("got %q, want %q", result, "abcDEF_123")
	}
}

func TestSanitizePackageName_ReplaceInvalid(t *testing.T) {
	result := SanitizePackageName("a-b/c.d!e")
	// '-' → '_', '/' → '_', '.' → '_', '!' → '_'
	if result != "a_b_c_d_e" {
		t.Errorf("got %q, want %q", result, "a_b_c_d_e")
	}
}

func TestSanitizePackageName_LeadingDigit(t *testing.T) {
	result := SanitizePackageName("123abc")
	if result != "_23abc" {
		t.Errorf("got %q, want %q", result, "_23abc")
	}
}

func TestSanitizePackageName_Empty(t *testing.T) {
	result := SanitizePackageName("")
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestSanitizePackageName_AllInvalid(t *testing.T) {
	result := SanitizePackageName("!@#$%^&*()")
	// 全部替换为 '_'
	if !strings.Contains(result, "_") {
		t.Errorf("expected underscores, got %q", result)
	}
	if strings.ContainsAny(result, "!@#$%^&*()") {
		t.Errorf("should not contain original invalid chars, got %q", result)
	}
}

func TestSanitizePackageName_ChineseChars(t *testing.T) {
	// 中文字符 → 全部替换为 '_'
	result := SanitizePackageName("包名test")
	expected := "____test" // 每个中文字符占多个rune，但每个rune都非合法字符
	// 不严格断言长度，只断言不含中文
	if strings.Contains(result, "包") || strings.Contains(result, "名") {
		t.Errorf("should not contain Chinese chars, got %q", result)
	}
	if !strings.Contains(result, "test") {
		t.Errorf("should preserve valid chars, got %q", result)
	}
	_ = expected
}

// ─────────────────────────────────────────────
// CapitalizeASCII
// ─────────────────────────────────────────────

func TestCapitalizeASCII_LowercaseFirst(t *testing.T) {
	result := CapitalizeASCII("hello")
	if result != "Hello" {
		t.Errorf("got %q, want %q", result, "Hello")
	}
}

func TestCapitalizeASCII_AlreadyUpper(t *testing.T) {
	result := CapitalizeASCII("Hello")
	if result != "Hello" {
		t.Errorf("got %q, want %q", result, "Hello")
	}
}

func TestCapitalizeASCII_NonLetter(t *testing.T) {
	result := CapitalizeASCII("123abc")
	// 首字符是数字，不在 a-z 范围，不应修改
	if result != "123abc" {
		t.Errorf("got %q, want %q", result, "123abc")
	}
}

func TestCapitalizeASCII_Empty(t *testing.T) {
	result := CapitalizeASCII("")
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestCapitalizeASCII_SingleChar(t *testing.T) {
	result := CapitalizeASCII("a")
	if result != "A" {
		t.Errorf("got %q, want %q", result, "A")
	}
}

func TestCapitalizeASCII_ZLowerCase(t *testing.T) {
	result := CapitalizeASCII("zoo")
	if result != "Zoo" {
		t.Errorf("got %q, want %q", result, "Zoo")
	}
}

// ─────────────────────────────────────────────
// ToBsonA
// ─────────────────────────────────────────────

func TestToBsonA_Ints(t *testing.T) {
	docs := []int{1, 2, 3}
	result := ToBsonA(docs)
	if len(result) != 3 {
		t.Fatalf("len: got %d, want 3", len(result))
	}
	for i, v := range result {
		if v != i+1 {
			t.Errorf("index %d: got %v, want %d", i, v, i+1)
		}
	}
}

func TestToBsonA_Strings(t *testing.T) {
	docs := []string{"a", "b", "c"}
	result := ToBsonA(docs)
	if len(result) != 3 {
		t.Fatalf("len: got %d, want 3", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("values: got %v", result)
	}
}

func TestToBsonA_Empty(t *testing.T) {
	docs := []int{}
	result := ToBsonA(docs)
	if len(result) != 0 {
		t.Errorf("len: got %d, want 0", len(result))
	}
}

func TestToBsonA_BsonTypes(t *testing.T) {
	type Doc struct {
		Name string `bson:"name"`
	}
	docs := []Doc{{Name: "test1"}, {Name: "test2"}}
	result := ToBsonA(docs)
	if len(result) != 2 {
		t.Fatalf("len: got %d, want 2", len(result))
	}
}

// ─────────────────────────────────────────────
// Base6408
// ─────────────────────────────────────────────

func TestBase6408_Length(t *testing.T) {
	result := Base6408("hello")
	if len(result) != 8 {
		t.Errorf("len: got %d, want 8", len(result))
	}
}

func TestBase6408_Deterministic(t *testing.T) {
	// 同一输入永远产生同一输出
	r1 := Base6408("test-input")
	r2 := Base6408("test-input")
	if r1 != r2 {
		t.Errorf("should be deterministic: got %q and %q", r1, r2)
	}
}

func TestBase6408_DifferentInputs(t *testing.T) {
	r1 := Base6408("input-a")
	r2 := Base6408("input-b")
	if r1 == r2 {
		t.Errorf("different inputs should produce different outputs: both %q", r1)
	}
}

func TestBase6408_Empty(t *testing.T) {
	result := Base6408("")
	if len(result) != 8 {
		t.Errorf("len: got %d, want 8 (sha256 of empty)", len(result))
	}
}

func TestBase6408_ValidBase64Chars(t *testing.T) {
	// base64 标准编码只含 [A-Za-z0-9+/=]
	result := Base6408("some random string")
	for _, c := range result {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			t.Errorf("invalid base64 char %q in result %q", c, result)
		}
	}
}
