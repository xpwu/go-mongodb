package x

import (
	"reflect"
	"testing"
)

// ─────────────────────────────────────────────
// ParseStructTags
// ─────────────────────────────────────────────

func TestParseStructTags_NoTag(t *testing.T) {
	type Doc struct {
		Name string
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Name")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "Name" {
		t.Errorf("Name: got %q, want %q", st.Name, "Name")
	}
	if st.OmitEmpty || st.MinSize || st.Truncate || st.Inline || st.Skip {
		t.Errorf("all flags should be false, got %+v", st)
	}
}

func TestParseStructTags_BasicName(t *testing.T) {
	type Doc struct {
		Name string `bson:"name"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Name")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "name" {
		t.Errorf("Name: got %q, want %q", st.Name, "name")
	}
}

func TestParseStructTags_PreserveOriginalCase(t *testing.T) {
	// 关键测试：不转小写
	type Doc struct {
		UserName string `bson:"UserName"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "UserName" {
		t.Errorf("Name should preserve original case, got %q", st.Name)
	}
}

func TestParseStructTags_OmitEmpty(t *testing.T) {
	type Doc struct {
		Field string `bson:"field,omitempty"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
	if st.Name != "field" {
		t.Errorf("Name: got %q, want %q", st.Name, "field")
	}
}

func TestParseStructTags_MinSize(t *testing.T) {
	type Doc struct {
		Field int64 `bson:"field,minsize"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.MinSize {
		t.Error("MinSize should be true")
	}
}

func TestParseStructTags_Truncate(t *testing.T) {
	type Doc struct {
		Field float64 `bson:"field,truncate"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Truncate {
		t.Error("Truncate should be true")
	}
}

func TestParseStructTags_Inline(t *testing.T) {
	type Doc struct {
		Field interface{} `bson:"field,inline"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Inline {
		t.Error("Inline should be true")
	}
}

func TestParseStructTags_Skip(t *testing.T) {
	type Doc struct {
		Field string `bson:"-"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Skip {
		t.Error("Skip should be true")
	}
}

func TestParseStructTags_AllFlags(t *testing.T) {
	type Doc struct {
		Field string `bson:"myfield,omitempty,minsize,truncate,inline"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "myfield" {
		t.Errorf("Name: got %q, want %q", st.Name, "myfield")
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
	if !st.MinSize {
		t.Error("MinSize should be true")
	}
	if !st.Truncate {
		t.Error("Truncate should be true")
	}
	if !st.Inline {
		t.Error("Inline should be true")
	}
}

func TestParseStructTags_EmptyTag(t *testing.T) {
	type Doc struct {
		Field string `bson:""`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 空 tag → 用字段名
	if st.Name != "Field" {
		t.Errorf("Name: got %q, want %q", st.Name, "Field")
	}
}

func TestParseStructTags_TagWithoutKey(t *testing.T) {
	type Doc struct {
		Field int `bson:",omitempty"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 无 key → 用字段名（不转小写）
	if st.Name != "Field" {
		t.Errorf("Name: got %q, want %q", st.Name, "Field")
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
}

// ─────────────────────────────────────────────
// ParseStructTagsToLower
// ─────────────────────────────────────────────

func TestParseStructTagsToLower_True(t *testing.T) {
	type Doc struct {
		UserName string `bson:"username"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStructTagsToLower(sf, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "username" {
		t.Errorf("toLower=true: Name should be %q, got %q", "username", st.Name)
	}
}

func TestParseStructTagsToLower_False(t *testing.T) {
	type Doc struct {
		UserName string `bson:"UserName"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStructTagsToLower(sf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "UserName" {
		t.Errorf("toLower=false: Name should preserve %q, got %q", "UserName", st.Name)
	}
}

func TestParseStructTagsToLower_NoTagToLower(t *testing.T) {
	type Doc struct {
		UserName string
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStructTagsToLower(sf, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "username" {
		t.Errorf("toLower=true, no tag: Name should be %q, got %q", "username", st.Name)
	}
}

// ─────────────────────────────────────────────
// ParseJSONStructTags / ParseJSONStructTagsToLower
// ─────────────────────────────────────────────

func TestParseJSONStructTags_UseBsonFirst(t *testing.T) {
	type Doc struct {
		Field string `bson:"from_bson" json:"from_json"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseJSONStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "from_bson" {
		t.Errorf("should prefer bson tag, got %q", st.Name)
	}
}

func TestParseJSONStructTags_FallbackToJSON(t *testing.T) {
	type Doc struct {
		Field string `json:"from_json"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseJSONStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "from_json" {
		t.Errorf("should fallback to json tag, got %q", st.Name)
	}
}

func TestParseJSONStructTagsToLower_True(t *testing.T) {
	type Doc struct {
		UserName string `json:"user_name"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseJSONStructTagsToLower(sf, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "user_name" {
		t.Errorf("json tag value should be %q, got %q", "user_name", st.Name)
	}
}

func TestParseJSONStructTagsToLower_False_NoTag(t *testing.T) {
	type Doc struct {
		UserName string
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseJSONStructTagsToLower(sf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 无 json tag，无 bson tag → 用字段名，不转小写
	if st.Name != "UserName" {
		t.Errorf("should use field name %q, got %q", "UserName", st.Name)
	}
}

// ─────────────────────────────────────────────
// ParseStruct (统一入口)
// ─────────────────────────────────────────────

func TestParseStruct_NoToLowerNoJson(t *testing.T) {
	type Doc struct {
		UserName string `bson:"UserName,omitempty,minsize"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStruct(sf, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "UserName" {
		t.Errorf("Name should preserve case, got %q", st.Name)
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
	if !st.MinSize {
		t.Error("MinSize should be true")
	}
}

func TestParseStruct_ToLowerNoJson(t *testing.T) {
	type Doc struct {
		UserName string `bson:"uname,omitempty"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStruct(sf, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "uname" {
		t.Errorf("Name should be %q, got %q", "uname", st.Name)
	}
}

func TestParseStruct_NoToLowerWithJson(t *testing.T) {
	type Doc struct {
		UserName string `json:"user_name,omitempty"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStruct(sf, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "user_name" {
		t.Errorf("Name should be %q (from json), got %q", "user_name", st.Name)
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true (from json tag)")
	}
}

func TestParseStruct_ToLowerWithJson_NoTag(t *testing.T) {
	type Doc struct {
		UserName string
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("UserName")

	st, err := ParseStruct(sf, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 无 tag，toLower=true → 转小写
	if st.Name != "username" {
		t.Errorf("Name should be %q, got %q", "username", st.Name)
	}
}

// ─────────────────────────────────────────────
// 边界情况
// ─────────────────────────────────────────────

func TestParseStructTags_NonBsonTagIgnored(t *testing.T) {
	// 有非 bson tag，不应影响解析
	type Doc struct {
		Field string `json:"something" xml:"else"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 无 bson tag → 用字段名
	if st.Name != "Field" {
		t.Errorf("Name should be %q, got %q", "Field", st.Name)
	}
}

func TestParseStructTags_BsonWithOtherTags(t *testing.T) {
	type Doc struct {
		Field string `bson:"myfield,omitempty" json:"other"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "myfield" {
		t.Errorf("Name should be %q, got %q", "myfield", st.Name)
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
}

func TestParseStructTags_UnknownFlagIgnored(t *testing.T) {
	// 未知 flag 不应报错（静默忽略）
	type Doc struct {
		Field string `bson:"field,unknownflag"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "field" {
		t.Errorf("Name: got %q, want %q", st.Name, "field")
	}
	// 未知 flag 不影响其他字段
	if st.OmitEmpty || st.MinSize || st.Truncate || st.Inline || st.Skip {
		t.Errorf("unknown flag should not set any bool field, got %+v", st)
	}
}

func TestParseStructTags_DashOnly(t *testing.T) {
	type Doc struct {
		Field string `bson:"-"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Skip {
		t.Error("Skip should be true for '-' tag")
	}
}

func TestParseStructTags_ComplexCombination(t *testing.T) {
	type Doc struct {
		Field int64 `bson:"renamed,inline,minsize,truncate,omitempty"`
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "renamed" {
		t.Errorf("Name: got %q, want %q", st.Name, "renamed")
	}
	if !st.Inline {
		t.Error("Inline should be true")
	}
	if !st.MinSize {
		t.Error("MinSize should be true")
	}
	if !st.Truncate {
		t.Error("Truncate should be true")
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
}

// ─────────────────────────────────────────────
// Fallback 分支：tag 里没有 ':' 时整段当作 bson tag
// ─────────────────────────────────────────────

func TestParseStructTags_FallbackWholeTag(t *testing.T) {
	// 无 `bson:""` 格式，整段 tag 不含 ':' → 回退到整段作为 tag
	type Doc struct {
		Field string "myname,omitempty"
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "myname" {
		t.Errorf("Name: got %q, want %q", st.Name, "myname")
	}
	if !st.OmitEmpty {
		t.Error("OmitEmpty should be true")
	}
}

func TestParseStructTags_FallbackOnlyName(t *testing.T) {
	// 整段 tag 不含 ':'，也没有逗号 → 整个作为 key
	type Doc struct {
		Field string "justname"
	}
	sf, _ := reflect.TypeOf(Doc{}).FieldByName("Field")

	st, err := ParseStructTags(sf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Name != "justname" {
		t.Errorf("Name: got %q, want %q", st.Name, "justname")
	}
}
