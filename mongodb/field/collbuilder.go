package field

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/xpwu/go-db-mongo/mongodb"
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/geo"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
	"github.com/xpwu/go-db-mongo/mongodb/x"
	"go.mongodb.org/mongo-driver/v2/bson"
	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/template"
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
	T     reflect.Type
	Field ReflectType
	// NewField func(name string) FieldType
	NewField  ReflectType
	EqualAble bool
}

func NewTypeInfo[T any, FieldType mongodb.Field](creator func(name string) FieldType) TypeInfo {
	name := runtime.FuncForPC(reflect.ValueOf(creator).Pointer()).Name()
	f := strings.FieldsFunc(name, func(r rune) bool {
		if r == '.' {
			return true
		}
		return false
	})

	funName := &reflectType{pkg: strings.Join(f[:len(f)-1], "."), name: f[len(f)-1]}
	equalAble := x.TypeFor[FieldType]().Implements(x.TypeFor[filter.ComparableFilterField[any]]())

	return TypeInfo{x.TypeFor[T](), x.TypeFor[FieldType](), funName, equalAble}
}

func typeFieldInfo[T any](field, creator string, equalAble bool) TypeInfo {
	return TypeInfo{x.TypeFor[T](),
		&reflectType{name: field, pkg: x.TypeFor[BaseField[any]]().PkgPath()},
		&reflectType{name: creator, pkg: x.TypeFor[BaseField[any]]().PkgPath()}, equalAble}
}

type structContext struct {
	imports *allImports
}

type CollBuilder struct {
	typeMap   map[reflect.Type]TypeInfo
	kindMap   map[reflect.Kind]func(reflect.Type) (TypeInfo, bool)
	dir       string
	targetPkg string

	structCtx *structContext

	opt *builderOption
}

type builderOption struct {
	useJSONStructTags bool
	lowerField        bool
	ignoreTagErr      bool
	dir               string
	targetPkg         string
}

type BuilderOption func(option *builderOption)

func UseJsonTag() BuilderOption {
	return func(option *builderOption) {
		option.useJSONStructTags = true
	}
}

func LowerField() BuilderOption {
	return func(option *builderOption) {
		option.lowerField = true
	}
}

// IgnoreTagErr 忽略 minsize & truncate & omitempty tag的报错
func IgnoreTagErr() BuilderOption {
	return func(option *builderOption) {
		option.ignoreTagErr = true
	}
}

//func WithDirAndPkg(dir, pkg string) BuilderOption {
//	return func(option *builderOption) {
//		option.dir = dir
//		option.targetPkg = pkg
//	}
//}

func NewCollBuilder(opts ...BuilderOption) *CollBuilder {
	op := &builderOption{
		useJSONStructTags: false,
		lowerField:        false,
		ignoreTagErr:      false,
	}
	for _, f := range opts {
		f(op)
	}

	b := &CollBuilder{
		typeMap: make(map[reflect.Type]TypeInfo),
		kindMap: make(map[reflect.Kind]func(reflect.Type) (TypeInfo, bool)),
		opt:     op,
	}

	b.RegisterType(typeFieldInfo[int]("IntField", "NewIntField", true))
	b.RegisterType(typeFieldInfo[int8]("Int8Field", "NewInt8Field", true))
	b.RegisterType(typeFieldInfo[int16]("Int16Field", "NewInt16Field", true))
	b.RegisterType(typeFieldInfo[int32]("Int32Field", "NewInt32Field", true))
	b.RegisterType(typeFieldInfo[int64]("Int64Field", "NewInt64Field", true))
	b.RegisterType(typeFieldInfo[uint]("UintField", "NewUintField", true))
	b.RegisterType(typeFieldInfo[uint8]("Uint8Field", "NewUint8Field", true))
	b.RegisterType(typeFieldInfo[uint16]("Uint16Field", "NewUint16Field", true))
	b.RegisterType(typeFieldInfo[uint32]("Uint32Field", "NewUint32Field", true))
	b.RegisterType(typeFieldInfo[uint64]("Uint64Field", "NewUint64Field", true))
	b.RegisterType(typeFieldInfo[float32]("Float32Field", "NewFloat32Field", false))
	b.RegisterType(typeFieldInfo[float64]("Float64Field", "NewFloat64Field", false))
	b.RegisterType(typeFieldInfo[string]("StringField", "NewStringField", true))
	b.RegisterType(typeFieldInfo[bool]("BoolField", "NewBoolField", true))
	b.RegisterType(typeFieldInfo[bson.Binary]("BinaryField", "NewBinaryField", true))
	b.RegisterType(typeFieldInfo[bson.Decimal128]("Decimal128Field", "NewDecimal128Field", true))
	b.RegisterType(typeFieldInfo[bson.ObjectID]("ObjectIDField", "NewObjectIDField", true))

	b.RegisterType(NewTypeInfo[geo.SpherePoint](NewSpherePointField))
	b.RegisterType(NewTypeInfo[geo.FlatPoint](NewFlatPointField))

	b.RegisterKind(reflect.Struct, b.buildStruct)
	b.RegisterKind(reflect.Slice, b.buildSlice)
	b.RegisterKind(reflect.Array, b.buildSlice)
	b.RegisterKind(reflect.Ptr, b.buildPtr)

	return b
}

func (b *CollBuilder) ClearType(rt reflect.Type) *CollBuilder {
	delete(b.typeMap, rt)
	return b
}

func (b *CollBuilder) RegisterType(info TypeInfo) *CollBuilder {
	b.typeMap[info.T] = info
	return b
}

func (b *CollBuilder) RegisterKind(k reflect.Kind, f func(rt reflect.Type) (TypeInfo, bool)) *CollBuilder {
	b.kindMap[k] = f
	return b
}

func (b *CollBuilder) buildPtr(t reflect.Type) (ft TypeInfo, ok bool) {
	return b.build(t.Elem())
}

func (b *CollBuilder) build(rt reflect.Type) (ft TypeInfo, ok bool) {
	ft, ok = b.typeMap[rt]
	if ok {
		return
	}

	f := b.kindMap[rt.Kind()]
	ft, ok = f(rt)
	if ok {
		b.typeMap[rt] = ft
	}

	return
}

func getRuntimeInfo() (pkg, dir string) {
	pc, file, _, ok := runtime.Caller(3)
	if ok {
		fName := runtime.FuncForPC(pc).Name()
		f := strings.FieldsFunc(fName, func(r rune) bool {
			if r == '.' {
				return true
			}
			return false
		})

		pkg = strings.Join(f[:len(f)-1], ".")

		dir = path.Dir(file)
	}
	return
}

func BuildColl[T any](builder *CollBuilder) {
	if builder.opt.dir == "" || builder.opt.targetPkg == "" {
		builder.targetPkg, builder.dir = getRuntimeInfo()
	} else {
		builder.targetPkg = builder.opt.targetPkg
		builder.dir = builder.opt.dir
	}

	builder.Build(x.TypeFor[T]())
}

func (b *CollBuilder) Build(rt reflect.Type) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if b.opt.dir == "" || b.opt.targetPkg == "" {
		b.targetPkg, b.dir = getRuntimeInfo()
	} else {
		b.targetPkg = b.opt.targetPkg
		b.dir = b.opt.dir
	}

	//b.targetPkg = rt.PkgPath()
	//if b.targetPkg == "" {
	//	// 基本类型 就取当前的pkg, 基本类型都是预生成在此pkg中
	//	// 对于其他pkg为空的类型 暂不支持
	//	b.targetPkg = reflect.TypeOf(b).Elem().PkgPath()
	//}

	b.build(rt)
}

func firstToLower(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func indentLines(s string, indents int) string {
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat("\t", indents) + lines[i]
	}
	return strings.Join(lines, "\n")
}

func (b *CollBuilder) buildSlice(t reflect.Type) (ft TypeInfo, ok bool) {
	elem := t.Elem()
	dim := 1
	for elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array {
		elem = elem.Elem()
		dim++
	}

	eft, eok := b.build(elem)
	if !eok {
		panic(fmt.Errorf("not support %v", elem))
	}

	thisImports := b.structCtx.imports

	// Array 的代码都在同一 package 下
	arrPkg := thisImports.add(x.TypeFor[ArrayField[any, mongodb.Field]]().PkgPath())

	arrField := ""
	if !eft.EqualAble {
		arrField = arrPkg + ".ArrayField"
	} else {
		arrField = arrPkg + ".ArrayComparableField"
	}

	arrNewField := arrPkg + ".NewArrayField"
	if eft.EqualAble {
		arrNewField = arrPkg + ".NewArrayAnyComparableField"
	}
	//newElem := func(name string) ArrayField[[][]int, ArrayField[[]int, ArrayField[int, IntField]]] {
	//	newElem := func(name string) ArrayField[[]int, ArrayField[int, IntField]] {
	//		newElem := func(name string) ArrayField[int, IntField] {
	//			newElem := NewIntField
	//			return NewArrayField[int, IntField](name, newElem)
	//		}
	//		return NewArrayField[[]int, ArrayField[int, IntField]](name, newElem)
	//	}
	//	return NewArrayField[[][]int, ArrayField[[]int, ArrayField[int, IntField]]](name, newElem)
	//}
	thisNewFieldTempl := template.Must(template.New("newField").Parse(
		`func(name string) {{.ThisField}} {
	newElem := {{.NewElemField}}
	return {{.ArrNewField}}[{{.ElemT}}, {{.ElemField}}](name, newElem)
}`))
	type TemplData struct {
		ThisField    string
		NewElemField string
		ArrNewField  string
		ElemT        string
		ElemField    string
	}

	newTemplData := func(thisField, newElemField, elemT, elemField string) *TemplData {
		return &TemplData{
			ArrNewField:  arrNewField,
			ThisField:    thisField,
			NewElemField: newElemField,
			ElemField:    elemField,
			ElemT:        elemT,
		}
	}

	elemT := addDot(thisImports.add(eft.T.PkgPath())) + eft.T.Name()
	elemField := addDot(thisImports.add(eft.Field.PkgPath())) + eft.Field.Name()
	newElemField := addDot(thisImports.add(eft.NewField.PkgPath())) + eft.NewField.Name()

	for i := 0; i < dim; i++ {
		newElemField = indentLines(newElemField, 1)

		thisT := fmt.Sprintf("[]%s", elemT)
		thisField := fmt.Sprintf("%s[%s, %s]", arrField, elemT, elemField)
		thisData := newTemplData(thisField, newElemField, elemT, elemField)
		buf := bytes.Buffer{}
		if err := thisNewFieldTempl.Execute(&buf, thisData); err != nil {
			panic(err)
		}
		thisNewField := buf.String()

		elemT = thisT
		elemField = thisField
		newElemField = thisNewField
	}

	// pkg 已经加入 imports, 所以不需要再返回 pkg
	ft = TypeInfo{
		T: t,
		Field: &reflectType{
			name: elemField,
			pkg:  "",
		},
		NewField: &reflectType{
			name: newElemField,
			pkg:  "",
		},
		EqualAble: eft.EqualAble,
	}

	return ft, true
}

var structCode2 = template.Must(template.New("structCode2").Funcs(template.FuncMap{
	"firstToLower": firstToLower,
}).Parse(`
// Code generated by collbuilder; DO NOT EDIT.

package {{.Pkg}}

import ({{- range .Imports}}
  {{.Alias}} "{{.Import}}"
{{- end}}
)

type {{.Name}}Field interface {
	{{.MongoAlias}}Field
	{{.FilterAlias}}ComparableFilter[{{.TypePkg}}{{.Name}}]
	{{.UpdaterAlias}}BaseUpdater[{{.TypePkg}}{{.Name}}]
{{- range .Fields}}
	{{.MethodName}}F() {{.FieldName}}
{{- end}}
{{- range .Inlines}}
	{{.FiledName}}Inline
{{- end}}
}

type {{.Name}}FieldInline interface {
{{- range .Fields}}
	{{.MethodName}}F() {{.FieldName}}
{{- end}}
{{- range .Inlines}}
	{{.FiledName}}Inline
{{- end}}
}

type {{.Name|firstToLower}}Field struct {
	{{.FieldAlias}}BaseField[{{.TypePkg}}{{.Name}}]
{{- range .Inlines}}
	{{.FiledName}}Inline
{{- end}}
}

var {{.Name}}Doc = New{{.Name}}Field("")

func New{{.Name}}Field(name string) {{.Name}}Field {
	return &{{.Name|firstToLower}}Field{
		*{{.FieldAlias}}NewBaseField[{{.TypePkg}}{{.Name}}](name),
{{- range .Inlines}}
		{{.NewField}}Inline(name),
{{- end}}
	}
}

func New{{.Name}}FieldInline(name string) {{.Name}}FieldInline {
	return &{{.Name|firstToLower}}Field{
		*{{.FieldAlias}}NewBaseField[{{.TypePkg}}{{.Name}}](name),
{{- range .Inlines}}
		{{.NewField}}Inline(name),
{{- end}}
	}
}

{{- range .Fields}}

func (s *{{$.Name|firstToLower}}Field) {{.MethodName}}F() {{.FieldName}} {
	return {{.NewField}}({{$.FieldAlias}}SubField(s.FullName(), "{{.TagName}}"))
}
{{- end}}
`))

type aliasNames map[string]bool

func (a aliasNames) get(expect string) string {
	test := expect
	num := 1
	for (a)[test] {
		num++
		test = fmt.Sprintf("%s%d", expect, num)
	}
	(a)[test] = true

	return test
}

type allImports struct {
	data  map[string]string
	alias aliasNames
	exc   map[string]bool
}

func newAllImports() *allImports {
	return &allImports{
		data:  make(map[string]string),
		alias: aliasNames{},
		exc:   make(map[string]bool),
	}
}

func (m *allImports) exclude(paths string) {
	m.exc[paths] = true
}

// return  `alias`  or “
func (m *allImports) add(paths string) (alias string) {
	if paths == "" || m.exc[paths] {
		return ""
	}

	if a, ok := m.data[paths]; ok {
		return a
	}

	a := m.alias.get(path.Base(paths))
	m.data[paths] = a

	return a
}

func addDot(s string) string {
	if len(s) == 0 {
		return s
	}
	return s + "."
}

type importTemp struct {
	Alias  string
	Import string
}

func (m *allImports) all() []importTemp {
	ret := make([]importTemp, len(m.data))

	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for j, k := range keys {
		ret[j] = importTemp{
			Alias:  m.data[k],
			Import: k,
		}
	}

	return ret
}

func (b *CollBuilder) buildStruct(t reflect.Type) (ft TypeInfo, ok bool) {
	type Field struct {
		MethodName string
		FieldName  string
		TagName    string
		NewField   string
	}

	type Inline struct {
		FiledName string
		NewField  string
	}

	type st struct {
		Pkg          string
		TypePkg      string
		Name         string
		FieldAlias   string
		MongoAlias   string
		FilterAlias  string
		UpdaterAlias string
		Imports      []importTemp
		Fields       []Field
		Inlines      []Inline
	}

	oldCtx := b.structCtx
	defer func() {
		b.structCtx = oldCtx
	}()

	b.structCtx = &structContext{imports: newAllImports()}
	thisImports := b.structCtx.imports

	thisPkg := b.targetPkg
	thisDir := b.dir
	thisName := x.BaseTypeName(t)

	if b.targetPkg != t.PkgPath() {
		subDir := x.SanitizePackageName(t.PkgPath()) + base6408(t.PkgPath())
		if strings.HasPrefix(t.PkgPath(), b.targetPkg+"/") {
			subDir = strings.TrimPrefix(t.PkgPath(), b.targetPkg+"/")
		}
		thisPkg = path.Join(b.targetPkg, subDir)
		thisDir = path.Join(b.dir, subDir)
	}

	thisImports.exclude(thisPkg)

	// firstly, put this pkg into alias
	thisImports.alias.get(thisPkg)

	s := &st{
		Pkg:          path.Base(thisPkg),
		TypePkg:      addDot(thisImports.add(t.PkgPath())),
		Name:         thisName,
		FilterAlias:  addDot(thisImports.add(x.TypeFor[filter.ComparableFilter[any]]().PkgPath())),
		FieldAlias:   addDot(thisImports.add(x.TypeFor[BaseField[any]]().PkgPath())),
		MongoAlias:   addDot(thisImports.add(x.TypeFor[mongodb.Field]().PkgPath())),
		UpdaterAlias: addDot(thisImports.add(x.TypeFor[updater.BaseUpdater[any]]().PkgPath())),
		Inlines:      make([]Inline, 0),
	}

	allSubName := make(map[string]bool)
	equalAble := true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// unexported
		if f.PkgPath != "" {
			continue
		}
		tag, _ := x.ParseStruct(f, b.opt.lowerField, b.opt.useJSONStructTags)
		if tag.Skip {
			continue
		}
		if !b.opt.lowerField && (tag.OmitEmpty || tag.MinSize || tag.Truncate) && !b.opt.ignoreTagErr {
			if !b.opt.ignoreTagErr {
				panic(errors.New(fmt.Sprintf(
					"NOT supported tag: minsize & truncate & omitempty are used in %s.%s.%s. \n"+
						"Using IgnoreTagErr() can ignore the error",
					t.PkgPath(), t.Name(), f.Name)))
			} else {
				println(fmt.Sprintf(
					"NOT supported tag: minsize & truncate & omitempty are used in %s.%s.%s.",
					t.PkgPath(), t.Name(), f.Name))
			}
		}
		fd := Field{}
		fd.MethodName = f.Name
		fd.TagName = tag.Name
		subFt, subOk := b.build(f.Type)
		if !subOk {
			panic(fmt.Errorf("not support %v", f.Type))
		}
		equalAble = equalAble && subFt.EqualAble
		subFName := addDot(thisImports.add(subFt.Field.PkgPath())) + subFt.Field.Name()
		subNewF := addDot(thisImports.add(subFt.NewField.PkgPath())) + subFt.NewField.Name()
		// inline
		if tag.Inline && f.Type.Kind() == reflect.Struct {
			s.Inlines = append(s.Inlines, Inline{subFName, subNewF})
		} else {
			fd.FieldName = subFName
			fd.NewField = indentLines(subNewF, 2)

			s.Fields = append(s.Fields, fd)
		}

		allSubName[fd.MethodName] = true
	}

	s.Imports = thisImports.all()

	if err := os.MkdirAll(thisDir, 0755); err != nil {
		panic(err)
	}

	file, err := os.Create(fmt.Sprintf("%s/z%sField.go", thisDir, t.Name()))
	if err != nil {
		panic(err)
	}

	err = structCode2.Execute(file, s)
	if err != nil {
		panic(err)
	}

	ft = TypeInfo{
		T:         t,
		Field:     &reflectType{name: thisName + "Field", pkg: thisPkg},
		NewField:  &reflectType{name: "New" + thisName + "Field", pkg: thisPkg},
		EqualAble: equalAble,
	}

	return ft, true
}

func base6408(s string) string {
	sha256v := sha256.Sum256([]byte(s))
	r := base64.StdEncoding.EncodeToString(sha256v[:])
	return r[0:8]
}
