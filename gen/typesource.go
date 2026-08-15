package gen

import (
	"github.com/xpwu/go-mongodb/x"
	"reflect"
)

type TypeSource interface {
	Name() string
	PkgPath() string
	Kind() reflect.Kind
	NumField() int
	Field(i int) FieldSource
	Elem() TypeSource
	IsBuiltin() bool
}

type FieldSource interface {
	Name() string
	Type() TypeSource
	Tag() string
	StructTag() *x.StructTags
	IsExported() bool
}
