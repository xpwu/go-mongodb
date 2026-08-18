package gen

import (
	"sort"
)

// MapEntry 自定义类型映射条目，都是全限定符
//	"IntField"
//	"fields.IntField"
//	"github.com/foo/fields.IntField"
type MapEntry struct {
	Key       string
	FieldType string
	NewFunc   string
	EqualAble bool
}

// Config 生成器配置
type Config struct {
	Maps          map[string]MapEntry
	PreserveField bool
	UseJSONTags   bool
	IgnoreTagErr  bool
	Dir           string
	Pkg           string
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		Maps: make(map[string]MapEntry),
	}
}

// AddMap adds a custom type mapping.
//
// typeIdent is the fully qualified type identifier of the SOURCE type:
//   - Builtin:      "int"
//   - Same pkg:     "MyType"
//   - External pkg: "github.com/foo/bar.MyType"
//
// fieldType is the fully qualified type identifier of the TARGET field type.
// newFunc is the fully qualified constructor function name.
//
// equalAble should be true if fieldType implements or embeds
// github.com/xpwu/filter/ComparableFilter (directly or transitively through
// embedded interfaces/structs). Otherwise, set to false.
//
// NOTE: Generic types and generic functions are NOT supported.
// Not supported:
//
//	AddMap("User", "fields.StringField[User]", "fields.NewStringField[User]", true) // generic
func (c *Config) AddMap(typeIdent, fieldType, newFunc string, equalAble bool) {
	c.Maps[typeIdent] = MapEntry{
		Key:       typeIdent,
		FieldType: fieldType,
		NewFunc:   newFunc,
		EqualAble: equalAble,
	}
}

// MapsSlice 返回排序后的映射列表
func (c *Config) MapsSlice() []MapEntry {
	keys := make([]string, 0, len(c.Maps))
	for k := range c.Maps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]MapEntry, 0, len(keys))
	for _, k := range keys {
		result = append(result, c.Maps[k])
	}
	return result
}
