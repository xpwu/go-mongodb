package gen

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ─── newAllImports ──────────────────────────────────────

func TestNewAllImports(t *testing.T) {
	ai := newAllImports()
	if ai == nil {
		t.Fatal("newAllImports returned nil")
	}
	if ai.data == nil {
		t.Error("data map should be initialized")
	}
	if ai.alias == nil {
		t.Error("alias map should be initialized")
	}
	if ai.exc == nil {
		t.Error("exc map should be initialized")
	}
	// all() on empty should return empty slice
	got := ai.all()
	if len(got) != 0 {
		t.Errorf("all() on empty = %d entries, want 0", len(got))
	}
}

// ─── add / all ──────────────────────────────────────────

func TestAdd_Basic(t *testing.T) {
	ai := newAllImports()

	alias := ai.add("github.com/xpwu/go-mongodb/fields")
	if alias == "" {
		t.Fatal("add returned empty alias")
	}
	// path.Base("fields") = "fields"
	if alias != "fields" {
		t.Errorf("alias = %q, want %q", alias, "fields")
	}

	// all() should contain one entry
	entries := ai.all()
	if len(entries) != 1 {
		t.Fatalf("all() len = %d, want 1", len(entries))
	}
	if entries[0].Import != "github.com/xpwu/go-mongodb/fields" {
		t.Errorf("Import = %q, want %q", entries[0].Import, "github.com/xpwu/go-mongodb/fields")
	}
	if entries[0].Alias != "fields" {
		t.Errorf("Alias = %q, want %q", entries[0].Alias, "fields")
	}
}

func TestAdd_Duplicate(t *testing.T) {
	ai := newAllImports()

	a1 := ai.add("github.com/xpwu/go-mongodb/fields")
	a2 := ai.add("github.com/xpwu/go-mongodb/fields")

	if a1 != a2 {
		t.Errorf("duplicate add: %q vs %q", a1, a2)
	}

	// still only one entry
	if len(ai.all()) != 1 {
		t.Errorf("all() len = %d, want 1", len(ai.all()))
	}
}

func TestAdd_EmptyPath(t *testing.T) {
	ai := newAllImports()
	alias := ai.add("")
	if alias != "" {
		t.Errorf("add empty path = %q, want empty", alias)
	}
	if len(ai.all()) != 0 {
		t.Errorf("all() len = %d, want 0", len(ai.all()))
	}
}

func TestAdd_MultiplePaths(t *testing.T) {
	ai := newAllImports()

	a1 := ai.add("github.com/xpwu/go-mongodb/fields")
	a2 := ai.add("go.mongodb.org/mongo-driver/v2/bson")

	if a1 == "" || a2 == "" {
		t.Fatal("aliases should not be empty")
	}
	if a1 == a2 {
		t.Errorf("two different paths got same alias: %q", a1)
	}

	entries := ai.all()
	if len(entries) != 2 {
		t.Fatalf("all() len = %d, want 2", len(entries))
	}

	// verify sorted by Import path
	if entries[0].Import > entries[1].Import {
		t.Error("entries should be sorted by Import path")
	}

	// check both paths present
	imports := map[string]string{}
	for _, e := range entries {
		imports[e.Import] = e.Alias
	}
	if imports["github.com/xpwu/go-mongodb/fields"] != "fields" {
		t.Errorf("fields alias = %q, want %q", imports["github.com/xpwu/go-mongodb/fields"], "fields")
	}
	if imports["go.mongodb.org/mongo-driver/v2/bson"] != "bson" {
		t.Errorf("bson alias = %q, want %q", imports["go.mongodb.org/mongo-driver/v2/bson"], "bson")
	}
}

func TestAdd_ConflictDifferentPathsSameBase(t *testing.T) {
	ai := newAllImports()

	// both paths have base "fields" → second gets "fields2"
	a1 := ai.add("github.com/xpwu/go-mongodb/fields")
	a2 := ai.add("github.com/someone-else/fields")

	if a1 == "" || a2 == "" {
		t.Fatal("both aliases should be non-empty")
	}
	if a1 == a2 {
		t.Errorf("conflicting bases should differ: %q vs %q", a1, a2)
	}
	// a1 = "fields", a2 = "fields2"
	if a2 != "fields2" {
		t.Errorf("a2 = %q, want %q", a2, "fields2")
	}

	entries := ai.all()
	if len(entries) != 2 {
		t.Errorf("all() len = %d, want 2", len(entries))
	}
}

func TestAdd_ThreeWayConflict(t *testing.T) {
	ai := newAllImports()

	a1 := ai.add("a/b/fields")
	a2 := ai.add("c/d/fields")
	a3 := ai.add("e/f/fields")

	// a1="fields", a2="fields2", a3="fields3"
	if a1 != "fields" {
		t.Errorf("a1 = %q, want %q", a1, "fields")
	}
	if a2 != "fields2" {
		t.Errorf("a2 = %q, want %q", a2, "fields2")
	}
	if a3 != "fields3" {
		t.Errorf("a3 = %q, want %q", a3, "fields3")
	}
}

// ─── exclude ───────────────────────────────────────────

func TestExclude(t *testing.T) {
	ai := newAllImports()

	// exclude first, then add → should be ignored
	ai.exclude("github.com/xpwu/go-mongodb/fields")
	alias := ai.add("github.com/xpwu/go-mongodb/fields")
	if alias != "" {
		t.Errorf("add excluded path = %q, want empty", alias)
	}
	if len(ai.all()) != 0 {
		t.Errorf("all() len = %d, want 0", len(ai.all()))
	}
}

func TestExclude_AfterAdd(t *testing.T) {
	ai := newAllImports()

	// add first
	alias := ai.add("github.com/xpwu/go-mongodb/fields")
	if alias == "" {
		t.Fatal("add should succeed before exclude")
	}

	// exclude doesn't remove existing entries
	ai.exclude("github.com/xpwu/go-mongodb/fields")
	entries := ai.all()
	if len(entries) != 1 {
		t.Errorf("exclude after add: all() len = %d, want 1 (exclude only affects future adds)", len(entries))
	}
}

func TestExclude_Multiple(t *testing.T) {
	ai := newAllImports()

	ai.exclude("path/a")
	ai.exclude("path/b")
	ai.exclude("path/a") // duplicate exclude, no-op

	// both should be excluded
	if ai.add("path/a") != "" {
		t.Error("path/a should be excluded")
	}
	if ai.add("path/b") != "" {
		t.Error("path/b should be excluded")
	}
	// non-excluded path works
	if ai.add("path/c") == "" {
		t.Error("path/c should not be excluded")
	}
	if len(ai.all()) != 1 {
		t.Errorf("all() len = %d, want 1", len(ai.all()))
	}
}

// ─── all / sorting ──────────────────────────────────────

func TestAll_SortedByImportPath(t *testing.T) {
	ai := newAllImports()

	paths := []string{
		"zebra.com/z",
		"apple.com/a",
		"mango.com/m",
	}
	for _, p := range paths {
		ai.add(p)
	}

	entries := ai.all()
	if len(entries) != 3 {
		t.Fatalf("all() len = %d, want 3", len(entries))
	}

	// verify sorted
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Import > entries[i].Import {
			t.Errorf("not sorted: %q > %q", entries[i-1].Import, entries[i].Import)
		}
	}

	// exact order
	expected := []string{"apple.com/a", "mango.com/m", "zebra.com/z"}
	for i, e := range entries {
		if e.Import != expected[i] {
			t.Errorf("entries[%d].Import = %q, want %q", i, e.Import, expected[i])
		}
	}
}

func TestAll_Empty(t *testing.T) {
	ai := newAllImports()
	entries := ai.all()
	if entries == nil {
		t.Error("all() should return empty slice, not nil")
	}
	if len(entries) != 0 {
		t.Errorf("all() len = %d, want 0", len(entries))
	}
}

// ─── addDot ─────────────────────────────────────────────

func TestAddDot2(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"fields", "fields."},
		{"", ""},
		{"bson", "bson."},
		{"a", "a."},
	}
	for _, tt := range tests {
		got := addDot(tt.in)
		if got != tt.want {
			t.Errorf("addDot(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// ─── aliasNames.get ─────────────────────────────────────

func TestAliasNames_Get(t *testing.T) {
	an := aliasNames{}

	a1 := an.get("fields")
	if a1 != "fields" {
		t.Errorf("first get = %q, want %q", a1, "fields")
	}

	a2 := an.get("fields")
	if a2 != "fields2" {
		t.Errorf("second get = %q, want %q", a2, "fields2")
	}

	a3 := an.get("fields")
	if a3 != "fields3" {
		t.Errorf("third get = %q, want %q", a3, "fields3")
	}

	// different base unaffected
	b1 := an.get("bson")
	if b1 != "bson" {
		t.Errorf("bson get = %q, want %q", b1, "bson")
	}
}

func TestAliasNames_Get_SequentialGaps(t *testing.T) {
	an := aliasNames{}

	an.get("x") // x
	an.get("x") // x2
	an.get("x") // x3

	// verify internal state
	if !an["x"] || !an["x2"] || !an["x3"] {
		t.Error("aliasNames should have x, x2, x3 set to true")
	}
}

// ─── importTemp struct ──────────────────────────────────

func TestImportTemp_Struct(t *testing.T) {
	it := importTemp{
		Alias:  "fields",
		Import: "github.com/xpwu/go-mongodb/fields",
	}
	if it.Alias != "fields" {
		t.Errorf("Alias = %q", it.Alias)
	}
	if it.Import != "github.com/xpwu/go-mongodb/fields" {
		t.Errorf("Import = %q", it.Import)
	}

	// zero value
	var zero importTemp
	if zero.Alias != "" || zero.Import != "" {
		t.Error("zero importTemp should have empty fields")
	}
}

// ─── 集成 ───────────────────────────────────────────────

func TestAllImports_Integration(t *testing.T) {
	ai := newAllImports()

	// exclude a path
	ai.exclude("exclude.me/pkg")

	// add several paths
	paths := []string{
		"github.com/xpwu/go-mongodb/fields",
		"go.mongodb.org/mongo-driver/v2/bson",
		"github.com/another/pkg",
	}
	for _, p := range paths {
		alias := ai.add(p)
		if alias == "" {
			t.Errorf("add(%q) returned empty", p)
		}
	}

	// excluded path returns empty
	if ai.add("exclude.me/pkg") != "" {
		t.Error("excluded path should return empty")
	}

	// all() returns 3 entries, sorted
	entries := ai.all()
	if len(entries) != 3 {
		t.Fatalf("all() len = %d, want 3", len(entries))
	}

	// verify sorting
	sorted := sort.SliceIsSorted(entries, func(i, j int) bool {
		return entries[i].Import < entries[j].Import
	})
	if !sorted {
		t.Error("entries not sorted by Import")
	}

	// verify alias values
	aliasMap := map[string]string{}
	for _, e := range entries {
		aliasMap[e.Import] = e.Alias
	}
	if aliasMap["github.com/xpwu/go-mongodb/fields"] != "fields" {
		t.Errorf("fields alias wrong: %q", aliasMap["github.com/xpwu/go-mongodb/fields"])
	}
	if aliasMap["go.mongodb.org/mongo-driver/v2/bson"] != "bson" {
		t.Errorf("bson alias wrong: %q", aliasMap["go.mongodb.org/mongo-driver/v2/bson"])
	}
	if aliasMap["github.com/another/pkg"] != "pkg" {
		t.Errorf("pkg alias wrong: %q", aliasMap["github.com/another/pkg"])
	}

	// addDot on aliases
	for _, e := range entries {
		dotted := addDot(e.Alias)
		if !strings.HasSuffix(dotted, ".") {
			t.Errorf("addDot(%q) = %q, should end with '.'", e.Alias, dotted)
		}
	}
}

// ─── 类型一致性 ─────────────────────────────────────────

func TestAllReturnsImportTempSlice(t *testing.T) {
	ai := newAllImports()
	ai.add("a/b/c")
	ai.add("d/e/f")

	entries := ai.all()
	// compile-time type check
	var _ []importTemp = entries

	// each entry has non-empty Import
	for i, e := range entries {
		if e.Import == "" {
			t.Errorf("entries[%d].Import is empty", i)
		}
	}
}

func TestDataMapIndependent(t *testing.T) {
	// modifying returned slice doesn't affect internal map
	ai := newAllImports()
	ai.add("p/q/r")
	ai.add("s/t/u")

	entries := ai.all()
	entries[0].Alias = "hacked"

	// internal state should be unchanged
	again := ai.all()
	if again[0].Alias == "hacked" {
		t.Error("all() should return a copy, not reference to internal data")
	}
}

// ─── 辅助：验证 aliasNames 是 map 类型 ──────────────────

func TestAliasNames_IsMap(t *testing.T) {
	var an aliasNames = aliasNames{}
	an["test"] = true

	// should be usable as map
	if !an["test"] {
		t.Error("aliasNames should behave like map")
	}
	if an["nope"] {
		t.Error("unset key should be false")
	}

	// len works
	if len(an) != 1 {
		t.Errorf("len(an) = %d, want 1", len(an))
	}
}

// ─── 边界：路径末尾带斜杠 ────────────────────────────────

func TestAdd_PathEdgeCases(t *testing.T) {
	ai := newAllImports()

	// path.Base("a/b/c/") = "c" (Go's path.Base strips trailing slash)
	alias := ai.add("a/b/c/")
	if alias == "" {
		t.Fatal("add with trailing slash returned empty")
	}
	// alias will be "c"
	if alias != "c" {
		t.Errorf("alias for 'a/b/c/' = %q, want %q", alias, "c")
	}
}

// ─── 完整工作流模拟代码生成 ─────────────────────────────

func TestAllImports_CodeGenWorkflow(t *testing.T) {
	ai := newAllImports()

	// 模拟生成器收集 import
	typeRefs := []string{
		"github.com/xpwu/go-mongodb/fields",
		"go.mongodb.org/mongo-driver/v2/bson",
		"context",
	}
	for _, p := range typeRefs {
		ai.add(p)
	}

	// 排除不需要的
	ai.exclude("unwanted/pkg")

	entries := ai.all()

	// 生成 import 块
	var sb strings.Builder
	sb.WriteString("import (\n")
	for _, e := range entries {
		if e.Alias != "" {
			sb.WriteString("\t" + e.Alias + " \"" + e.Import + "\"\n")
		} else {
			sb.WriteString("\t\"" + e.Import + "\"\n")
		}
	}
	sb.WriteString(")")

	output := sb.String()
	expect := `import (
	context "context"
	fields "github.com/xpwu/go-mongodb/fields"
	bson "go.mongodb.org/mongo-driver/v2/bson"
)`
	if output != expect {
		t.Errorf("generated import block:\n%s \nwant=%s", output, expect)
	}

	// 验证输出包含关键路径
	for _, p := range typeRefs {
		if !strings.Contains(output, p) {
			t.Errorf("output missing %q", p)
		}
	}
	// 验证不包含被排除的
	if strings.Contains(output, "unwanted/pkg") {
		t.Error("output should not contain excluded path")
	}
	// 验证别名带点
	if !strings.Contains(output, "fields") {
		t.Error("output should contain fields alias")
	}
}

// ─── 反射验证内部结构（防御性） ─────────────────────────

func TestAllImports_InternalStruct(t *testing.T) {
	ai := newAllImports()

	// data is map[string]string
	if reflect.TypeOf(ai.data).Kind() != reflect.Map {
		t.Error("data should be a map")
	}

	// exc is map[string]bool
	if reflect.TypeOf(ai.exc).Kind() != reflect.Map {
		t.Error("exc should be a map")
	}

	// alias is aliasNames (map[string]bool)
	if reflect.TypeOf(ai.alias).Kind() != reflect.Map {
		t.Error("alias should be a map")
	}
}

func TestAddDot(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		{"non-empty", "fields", "fields."},
		{"single char", "f", "f."},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addDot(tt.input)
			if got != tt.expected {
				t.Errorf("addDot(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAddDot_Integration(t *testing.T) {
	all := newAllImports()
	all.add("github.com/xpwu/go-mongodb/fields")
	// 拿到别名（不管具体是 fields 还是 fields2）
	for k, alias := range all.data {
		result := addDot(alias)
		if result == "" || !strings.HasSuffix(result, ".") {
			t.Errorf("addDot(%q) for %q = %q, should be non-empty ending with '.'", alias, k, result)
		}
	}
}
