package gen

import (
	"reflect"
)

// TypeSource 描述一个类型的元信息，支持反射和 AST 两种实现
type TypeSource interface {
	Name() string
	PkgPath() string
	Kind() reflect.Kind
	NumField() int
	Field(i int) FieldSource
	Elem() TypeSource
	IsBuiltin() bool
	// EnsureFields 确保字段信息已加载（AST 版会懒加载，反射版空实现）
	EnsureFields()
}

// FieldSource 描述结构体一个字段的元信息
type FieldSource interface {
	Name() string
	Type() TypeSource
	Tag() string
	IsExported() bool
}
