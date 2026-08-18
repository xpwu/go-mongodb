package gen

import (
	"reflect"
)

type reflectTypeSource struct {
	t reflect.Type
}

func (r *reflectTypeSource) Name() string       { return r.t.Name() }
func (r *reflectTypeSource) PkgPath() string    { return r.t.PkgPath() }
func (r *reflectTypeSource) Kind() reflect.Kind { return r.t.Kind() }
func (r *reflectTypeSource) NumField() int {
	if r.t.Kind() != reflect.Struct {
		return 0
	}
	return r.t.NumField()
}
func (r *reflectTypeSource) Elem() TypeSource {
	if r.t.Kind() == reflect.Ptr || r.t.Kind() == reflect.Slice || r.t.Kind() == reflect.Array {
		return &reflectTypeSource{t: r.t.Elem()}
	}
	return nil
}
func (r *reflectTypeSource) IsBuiltin() bool { return r.t.PkgPath() == "" }

// EnsureFields 反射版不需要懒加载，空实现
func (r *reflectTypeSource) EnsureFields() {}

func (r *reflectTypeSource) Field(i int) FieldSource {
	return &reflectFieldSource{f: r.t.Field(i)}
}

func (r *reflectTypeSource) Underlying() (TypeSource, bool) {
	return nil, false
}

type reflectFieldSource struct {
	f reflect.StructField
}

func (f *reflectFieldSource) Name() string     { return f.f.Name }
func (f *reflectFieldSource) Type() TypeSource { return &reflectTypeSource{t: f.f.Type} }
func (f *reflectFieldSource) Tag() string      { return string(f.f.Tag) }
func (f *reflectFieldSource) IsExported() bool { return f.f.PkgPath == "" }

func ReflectTypeSource(t reflect.Type) TypeSource {
	return &reflectTypeSource{t: t}
}
