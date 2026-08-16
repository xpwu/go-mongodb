package gen

import (
	"sort"
)

// MapEntry 自定义类型映射条目
type MapEntry struct {
	Key       string
	FieldType string
	NewFunc   string
}

// Config 生成器配置
type Config struct {
	Types         []string
	typeSet       map[string]bool
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
		typeSet: make(map[string]bool),
		Maps:    make(map[string]MapEntry),
		Dir:     ".",
	}
}

// AddType 添加要生成的类型名
func (c *Config) AddType(name string) {
	if !c.typeSet[name] {
		c.typeSet[name] = true
		c.Types = append(c.Types, name)
	}
}

// SetTypes 设置要生成的类型名列表
func (c *Config) SetTypes(names []string) {
	c.typeSet = make(map[string]bool)
	c.Types = nil
	for _, n := range names {
		c.AddType(n)
	}
}

// AddMap 添加自定义类型映射（同包用 key=TypeName）
func (c *Config) AddMap(key, fieldType, newFunc string) {
	c.Maps[key] = MapEntry{Key: key, FieldType: fieldType, NewFunc: newFunc}
}

// AddMapExt 添加自定义类型映射（跨包用 key=pkgPath.TypeName）
func (c *Config) AddMapExt(pkgPath, typeName, fieldType, newFunc string) {
	key := pkgPath + "." + typeName
	c.Maps[key] = MapEntry{Key: key, FieldType: fieldType, NewFunc: newFunc}
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
