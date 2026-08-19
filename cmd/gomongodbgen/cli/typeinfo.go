package cli

import (
	"errors"
	"reflect"
	"runtime"
	"strings"

	"github.com/xpwu/go-mongodb/field"
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/x"
)

type ReflectType interface {
	Name() string
	PkgPath() string
}

type reflectType struct {
	name string
	pkg  string
}

func (r *reflectType) Name() string {
	return r.name
}

func (r *reflectType) PkgPath() string {
	return r.pkg
}

type TypeInfo struct {
	T     ReflectType
	Field ReflectType
	// NewField func(name string) FieldType
	NewField  ReflectType
	EqualAble bool
	Err       error
}

// NewTypeInfo creates a TypeInfo for the given type T and its field type FieldType.
//
// # Requirements for creator
//
// creator MUST be a named, non-generic, package-level function.
// Method values (e.g. receiver.method) and anonymous functions are NOT supported.
//
// Correct usage:
//
//	func newNameField(name string) field.Field {
//	    return fields.NewStringField(name)
//	}
//	cli.NewTypeInfo[User](newNameField)
//
// The following are NOT supported and will return a TypeInfo with Err set:
//   - Anonymous functions: cli.NewTypeInfo[User](func(name string) field.Field { ... })
//   - Generic functions:   cli.NewTypeInfo[User](NewStringField[User])
//   - Method values:       cli.NewTypeInfo[User](myReceiver.newField)
//
// NewTypeInfo 为给定的类型 T 及其字段类型 FieldType 创建 TypeInfo。
//
// # creator 的要求
//
// creator 必须是一个命名的、非泛型的包级函数。
// 不支持方法值（如 receiver.method）和匿名函数。
//
// 正确用法：
//
//	func newNameField(name string) field.Field {
//	    return fields.NewStringField(name)
//	}
//	cli.NewTypeInfo[User](newNameField)
//
// 以下用法不受支持，会返回带有 Err 的 TypeInfo：
//   - 匿名函数：cli.NewTypeInfo[User](func(name string) field.Field { ... })
//   - 泛型函数：cli.NewTypeInfo[User](NewStringField[User])
//   - 方法值：  cli.NewTypeInfo[User](myReceiver.newField)
func NewTypeInfo[T any, FieldType field.Field](creator func(name string) FieldType) TypeInfo {
	name := runtime.FuncForPC(reflect.ValueOf(creator).Pointer()).Name()

	// 泛型函数：名字以 ] 结尾（如 package.NewField[github.com/foo/bar.User]）
	if strings.HasSuffix(name, "]") {
		return TypeInfo{Err: errors.New("TypeInfo does not support generic functions")}
	}

	// 方法值：名字包含 -fm（Go 编译器为方法值生成的闭包，如 (*MyType).Method-fm）
	if strings.Contains(name, "-fm") {
		return TypeInfo{Err: errors.New("TypeInfo does not support method values, use a package-level function")}
	}

	// 匿名函数：名字包含 .func（Go 编译器为匿名函数生成，如 package.func1）
	if strings.Contains(name, ".func") {
		return TypeInfo{Err: errors.New("TypeInfo does not support anonymous functions, use a named function")}
	}

	// 以下为正常解析逻辑...
	f := strings.FieldsFunc(name, func(r rune) bool {
		return r == '.'
	})
	if len(f) < 2 {
		return TypeInfo{Err: errors.New("TypeInfo: cannot parse function name")}
	}

	funName := &reflectType{pkg: strings.Join(f[:len(f)-1], "."), name: f[len(f)-1]}
	equalAble := x.TypeFor[FieldType]().Implements(x.TypeFor[filter.ComparableFilterField[T]]())

	return TypeInfo{x.TypeFor[T](), x.TypeFor[FieldType](), funName, equalAble, nil}
}
