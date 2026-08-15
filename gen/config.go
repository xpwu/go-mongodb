package gen

import (
	"sort"
)

type MapEntry struct {
	Key       string
	FieldType string
	NewFunc   string
}

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

func NewConfig() *Config {
	return &Config{
		typeSet: make(map[string]bool),
		Maps:    make(map[string]MapEntry),
		Dir:     ".",
	}
}

func (c *Config) AddType(name string) {
	if !c.typeSet[name] {
		c.typeSet[name] = true
		c.Types = append(c.Types, name)
	}
}

func (c *Config) SetTypes(names []string) {
	c.typeSet = make(map[string]bool)
	c.Types = nil
	for _, n := range names {
		c.AddType(n)
	}
}

func (c *Config) AddMap(key, fieldType, newFunc string) {
	c.Maps[key] = MapEntry{Key: key, FieldType: fieldType, NewFunc: newFunc}
}

func (c *Config) AddMapExt(pkgPath, typeName, fieldType, newFunc string) {
	key := pkgPath + "." + typeName
	c.Maps[key] = MapEntry{Key: key, FieldType: fieldType, NewFunc: newFunc}
}

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
