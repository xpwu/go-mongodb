package gen

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/xpwu/go-mongodb/x"
)

// ─── TypeInfo ───────────────────────────────────────────────

// TypeInfo 描述一个类型在生成代码中的映射信息
type TypeInfo struct {
	T         TypeSource
	Field     typeRef
	NewField  typeRef
	EqualAble bool
}

type typeRef struct {
	name string
	pkg  string
}

func (r typeRef) Name() string    { return r.name }
func (r typeRef) PkgPath() string { return r.pkg }

// ─── Generator ──────────────────────────────────────────────

// Generator 代码生成器
type Generator struct {
	config     *Config
	imports    *allImports
	outputDir  string
	targetPkg  string
	typeMap    map[string]TypeInfo
	pendingSts []TypeSource // 待处理的嵌套 struct 队列
	visited    map[string]bool
}

// NewGenerator 创建生成器
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config:  config,
		typeMap: make(map[string]TypeInfo),
		visited: make(map[string]bool),
	}
}

// Generate 从入口类型开始生成所有相关代码
// The returned subDir is both the subdirectory path and the sub-package name.
func (g *Generator) Generate(ts TypeSource) (subDir string) {
	g.imports = newAllImports()
	g.typeMap = make(map[string]TypeInfo)
	g.pendingSts = nil
	g.visited = make(map[string]bool)

	// 初始化 targetPkg
	if g.outputDir == "" {
		if g.config.Dir == "" {
			g.outputDir = "."
		} else {
			g.outputDir = g.config.Dir
		}
	}
	if g.targetPkg == "" {
		if g.config.Pkg == "" {
			g.targetPkg = ts.PkgPath()
		} else {
			g.targetPkg = g.config.Pkg
		}
	}

	// 先生成入口类型
	ret := g.build(ts)

	// 处理所有嵌套 struct
	g.processPending()

	if ret.Field.PkgPath() == g.targetPkg {
		return ""
	}

	return strings.TrimPrefix(ret.Field.PkgPath(), g.targetPkg+"/")
}

// processPending 循环处理嵌套 struct 队列
func (g *Generator) processPending() {
	for len(g.pendingSts) > 0 {
		// 取出队首
		ts := g.pendingSts[0]
		g.pendingSts = g.pendingSts[1:]

		key := ts.PkgPath() + "." + ts.Name()
		if g.visited[key] {
			continue
		}
		g.visited[key] = true

		// 防御性检查：只处理 struct 类型
		if ts.Kind() != reflect.Struct {
			continue
		}

		// 确保字段已加载（AST 版会懒加载，反射版空操作）
		ts.EnsureFields()

		// 生成该 struct 的代码
		g.buildStruct(ts)
	}
}

// build 分发到具体的 build 方法
func (g *Generator) build(ts TypeSource) TypeInfo {
	key := ts.PkgPath() + "." + ts.Name()
	if info, ok := g.typeMap[key]; ok {
		return info
	}

	if info, ok := g.lookupCustom(ts); ok {
		g.typeMap[key] = info
		return info
	}

	if info, ok := g.lookupBuiltinDirect(ts); ok {
		g.typeMap[key] = info
		return info
	}

	var info TypeInfo
	var ok bool

	switch ts.Kind() {
	case reflect.Struct:
		info, ok = g.buildStruct(ts)
	case reflect.Slice, reflect.Array:
		info, ok = g.buildSlice(ts)
	case reflect.Ptr:
		info, ok = g.buildPtr(ts)
	default:
		info, ok = g.buildKind(ts)
	}

	if !ok {
		panic(fmt.Errorf("not support type %s.%s (kind=%v)", ts.PkgPath(), ts.Name(), ts.Kind()))
	}

	g.typeMap[key] = info
	return info
}

func (g *Generator) buildPtr(ts TypeSource) (TypeInfo, bool) {
	elem := ts.Elem()
	if elem == nil {
		return TypeInfo{}, false
	}
	return g.build(elem), true
}

// ─── 自定义映射 ─────────────────────────────────────────────

// parseTypeRef splits a fully qualified identifier into pkgPath + name.
//
//	"IntField"                       -> {pkgPath:"", name:"IntField"}
//	"fields.IntField"                 -> {pkgPath:"fields", name:"IntField"}
//	"github.com/foo/fields.IntField"  -> {pkgPath:"github.com/foo/fields", name:"IntField"}
func parseTypeRef(s string) typeRef {
	if strings.Contains(s, "/") {
		lastSlash := strings.LastIndex(s, "/")
		afterSlash := s[lastSlash+1:]
		if dotIdx := strings.Index(afterSlash, "."); dotIdx != -1 {
			return typeRef{
				pkg:  s[:lastSlash] + "/" + afterSlash[:dotIdx],
				name: afterSlash[dotIdx+1:],
			}
		}
		return typeRef{
			pkg:  s[:lastSlash],
			name: afterSlash,
		}
	}
	if dotIdx := strings.Index(s, "."); dotIdx != -1 {
		return typeRef{
			pkg:  s[:dotIdx],
			name: s[dotIdx+1:],
		}
	}
	return typeRef{name: s}
}

func (g *Generator) lookupCustom(ts TypeSource) (TypeInfo, bool) {
	key := ts.PkgPath() + "." + ts.Name()
	entry, ok := g.config.Maps[key]
	if !ok {
		entry, ok = g.config.Maps[ts.Name()]
		if !ok {
			return TypeInfo{}, false
		}
	}
	return TypeInfo{
		T:         ts,
		Field:     parseTypeRef(entry.FieldType),
		NewField:  parseTypeRef(entry.NewFunc),
		EqualAble: entry.EqualAble,
	}, true
}

// ─── 内置类型硬编码 ─────────────────────────────────────────

func (g *Generator) lookupBuiltinDirect(ts TypeSource) (TypeInfo, bool) {
	fieldsPkg := "github.com/xpwu/go-mongodb/fields"

	// 无包路径 = 内置类型
	if ts.PkgPath() == "" {
		builtins := map[string]TypeInfo{
			"int":     {T: ts, Field: typeRef{name: "IntField", pkg: fieldsPkg}, NewField: typeRef{name: "NewIntField", pkg: fieldsPkg}, EqualAble: true},
			"int8":    {T: ts, Field: typeRef{name: "Int8Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewInt8Field", pkg: fieldsPkg}, EqualAble: true},
			"int16":   {T: ts, Field: typeRef{name: "Int16Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewInt16Field", pkg: fieldsPkg}, EqualAble: true},
			"int32":   {T: ts, Field: typeRef{name: "Int32Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewInt32Field", pkg: fieldsPkg}, EqualAble: true},
			"int64":   {T: ts, Field: typeRef{name: "Int64Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewInt64Field", pkg: fieldsPkg}, EqualAble: true},
			"uint":    {T: ts, Field: typeRef{name: "UintField", pkg: fieldsPkg}, NewField: typeRef{name: "NewUintField", pkg: fieldsPkg}, EqualAble: true},
			"uint8":   {T: ts, Field: typeRef{name: "Uint8Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewUint8Field", pkg: fieldsPkg}, EqualAble: true},
			"uint16":  {T: ts, Field: typeRef{name: "Uint16Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewUint16Field", pkg: fieldsPkg}, EqualAble: true},
			"uint32":  {T: ts, Field: typeRef{name: "Uint32Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewUint32Field", pkg: fieldsPkg}, EqualAble: true},
			"uint64":  {T: ts, Field: typeRef{name: "Uint64Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewUint64Field", pkg: fieldsPkg}, EqualAble: true},
			"float32": {T: ts, Field: typeRef{name: "Float32Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewFloat32Field", pkg: fieldsPkg}, EqualAble: false},
			"float64": {T: ts, Field: typeRef{name: "Float64Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewFloat64Field", pkg: fieldsPkg}, EqualAble: false},
			"string":  {T: ts, Field: typeRef{name: "StringField", pkg: fieldsPkg}, NewField: typeRef{name: "NewStringField", pkg: fieldsPkg}, EqualAble: true},
			"bool":    {T: ts, Field: typeRef{name: "BoolField", pkg: fieldsPkg}, NewField: typeRef{name: "NewBoolField", pkg: fieldsPkg}, EqualAble: true},
		}
		if info, ok := builtins[ts.Name()]; ok {
			return info, true
		}
		return TypeInfo{}, false
	}

	// bson 包类型
	bsonPkg := "go.mongodb.org/mongo-driver/v2/bson"
	if ts.PkgPath() == bsonPkg {
		bsonTypes := map[string]TypeInfo{
			"ObjectID":   {T: ts, Field: typeRef{name: "ObjectIDField", pkg: fieldsPkg}, NewField: typeRef{name: "NewObjectIDField", pkg: fieldsPkg}, EqualAble: true},
			"Binary":     {T: ts, Field: typeRef{name: "BinaryField", pkg: fieldsPkg}, NewField: typeRef{name: "NewBinaryField", pkg: fieldsPkg}, EqualAble: true},
			"Decimal128": {T: ts, Field: typeRef{name: "Decimal128Field", pkg: fieldsPkg}, NewField: typeRef{name: "NewDecimal128Field", pkg: fieldsPkg}, EqualAble: true},
			"Raw":        {T: ts, Field: typeRef{name: "RawField", pkg: fieldsPkg}, NewField: typeRef{name: "NewRawField", pkg: fieldsPkg}, EqualAble: true},
			"RawValue":   {T: ts, Field: typeRef{name: "RawValueField", pkg: fieldsPkg}, NewField: typeRef{name: "NewRawValueField", pkg: fieldsPkg}, EqualAble: true},
			"RawArray":   {T: ts, Field: typeRef{name: "RawArrayField", pkg: fieldsPkg}, NewField: typeRef{name: "NewRawArrayField", pkg: fieldsPkg}, EqualAble: true},
			"RawElement": {T: ts, Field: typeRef{name: "RawElementField", pkg: fieldsPkg}, NewField: typeRef{name: "NewRawElementField", pkg: fieldsPkg}, EqualAble: true},
			"DateTime":   {T: ts, Field: typeRef{name: "DateTimeField", pkg: fieldsPkg}, NewField: typeRef{name: "NewDateTimeField", pkg: fieldsPkg}, EqualAble: true},
			"Timestamp":  {T: ts, Field: typeRef{name: "TimestampField", pkg: fieldsPkg}, NewField: typeRef{name: "NewTimestampField", pkg: fieldsPkg}, EqualAble: true},
			"M":          {T: ts, Field: typeRef{name: "BsonMField", pkg: fieldsPkg}, NewField: typeRef{name: "NewBsonMField", pkg: fieldsPkg}, EqualAble: true},
			"A":          {T: ts, Field: typeRef{name: "BsonAField", pkg: fieldsPkg}, NewField: typeRef{name: "NewBsonAField", pkg: fieldsPkg}, EqualAble: true},
		}
		if info, ok := bsonTypes[ts.Name()]; ok {
			return info, true
		}
	}

	// geo 包类型
	geoPkg := "github.com/xpwu/go-mongodb/geo"
	if ts.PkgPath() == geoPkg {
		geoTypes := map[string]TypeInfo{
			"SpherePoint": {T: ts, Field: typeRef{name: "SpherePointField", pkg: fieldsPkg}, NewField: typeRef{name: "NewSpherePointField", pkg: fieldsPkg}, EqualAble: true},
			"FlatPoint":   {T: ts, Field: typeRef{name: "FlatPointField", pkg: fieldsPkg}, NewField: typeRef{name: "NewFlatPointField", pkg: fieldsPkg}, EqualAble: true},
		}
		if info, ok := geoTypes[ts.Name()]; ok {
			return info, true
		}
	}

	return TypeInfo{}, false
}

// ─── buildKind ──────────────────────────────────────────────

func (g *Generator) buildKind(ts TypeSource) (TypeInfo, bool) {
	fieldsPkg := "github.com/xpwu/go-mongodb/fields"

	typeAlias := ""
	if ts.PkgPath() != "" {
		typeAlias = g.imports.add(ts.PkgPath())
	}
	typeName := addDot(typeAlias) + ts.Name()

	switch ts.Kind() {
	case reflect.Bool:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("ComparableField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewComparableField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Int:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("IntegerField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewIntegerField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Int8:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("IntegerField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewIntegerField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Int16:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("IntegerField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewIntegerField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Int32:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("IntegerField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewIntegerField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Int64:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("IntegerField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewIntegerField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Uint:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("UnIntegerField[%s, int]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewUnIntegerField[%s, int]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Uint8:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("UnIntegerField[%s, int8]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewUnIntegerField[%s, int8]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Uint16:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("UnIntegerField[%s, int16]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewUnIntegerField[%s, int16]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Uint32:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("UnIntegerField[%s, int32]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewUnIntegerField[%s, int32]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Uint64:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("UnIntegerField[%s, int64]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewUnIntegerField[%s, int64]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Float32:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("ComputableField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewComputableField[%s]", typeName), pkg: fieldsPkg}, EqualAble: false}, true
	case reflect.Float64:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("ComputableField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewComputableField[%s]", typeName), pkg: fieldsPkg}, EqualAble: false}, true
	case reflect.String:
		return TypeInfo{T: ts, Field: typeRef{name: fmt.Sprintf("LikeStringField[%s]", typeName), pkg: fieldsPkg}, NewField: typeRef{name: fmt.Sprintf("NewLikeStringField[%s]", typeName), pkg: fieldsPkg}, EqualAble: true}, true
	case reflect.Interface:
		// any / interface{} → 使用 BaseStructField
		// 防御性说明：interface 类型无法在编译期确定具体类型，
		// 因此只能生成最通用的 BaseStructField，不支持 EqualAble 比较操作。
		return TypeInfo{
			T:         ts,
			Field:     typeRef{name: fmt.Sprintf("BaseStructField[%s]", typeName), pkg: fieldsPkg},
			NewField:  typeRef{name: fmt.Sprintf("NewBaseStructField[%s]", typeName), pkg: fieldsPkg},
			EqualAble: false,
		}, true
	default:
		return TypeInfo{}, false
	}
}

// ─── buildSlice ─────────────────────────────────────────────

func (g *Generator) buildSlice(ts TypeSource) (TypeInfo, bool) {
	// 计算维度，同时找到最内层元素类型
	dim := 1
	current := ts.Elem()
	for current != nil && (current.Kind() == reflect.Slice || current.Kind() == reflect.Array) {
		dim++
		current = current.Elem()
	}
	innermost := current // 最内层元素类型，如 int16

	// build 最内层元素类型（和反射版一致）
	eft := g.build(innermost)

	arrPkg := g.imports.add("github.com/xpwu/go-mongodb/fields")

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

	thisNewFieldTempl := template.Must(template.New("newField").Parse(
		`func(name string) {{.ThisField}} {
	newElem := {{.NewElemField}}
	return {{.ArrNewField}}[{{.ElemT}}, {{.ElemField}}](name, newElem)
}`))

	type templData struct {
		ThisField    string
		NewElemField string
		ArrNewField  string
		ElemT        string
		ElemField    string
	}

	newTemplData := func(thisField, newElemField, elemT, elemField string) *templData {
		return &templData{
			ArrNewField:  arrNewField,
			ThisField:    thisField,
			NewElemField: newElemField,
			ElemField:    elemField,
			ElemT:        elemT,
		}
	}

	// elemT 从最内层类型名开始（和反射版逐行对应）
	elemT := addDot(g.imports.add(eft.T.PkgPath())) + eft.T.Name()
	elemField := addDot(g.imports.add(eft.Field.PkgPath())) + eft.Field.Name()
	newElemField := addDot(g.imports.add(eft.NewField.PkgPath())) + eft.NewField.Name()

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

	return TypeInfo{
		T:         ts,
		Field:     typeRef{name: elemField, pkg: ""},
		NewField:  typeRef{name: newElemField, pkg: ""},
		EqualAble: eft.EqualAble,
	}, true
}

// ─── buildStruct ────────────────────────────────────────────

func (g *Generator) buildStruct(ts TypeSource) (TypeInfo, bool) {
	// 确保字段已加载（AST 版关键！）
	ts.EnsureFields()

	key := ts.PkgPath() + "." + ts.Name()
	if info, ok := g.typeMap[key]; ok {
		return info, true
	}

	oldImports := g.imports
	g.imports = newAllImports()
	thisImports := g.imports

	thisPkg := g.targetPkg
	thisDir := g.outputDir
	thisName := x.BaseTypeNameFromName(ts.Name())

	if g.targetPkg != "" && ts.PkgPath() != "" && g.targetPkg != ts.PkgPath() {
		subDir := x.SanitizePackageName(x.LastSubPath(ts.PkgPath()) + "_" + x.Base6408(ts.PkgPath()))
		if strings.HasPrefix(ts.PkgPath(), g.targetPkg+"/") {
			subDir = strings.TrimPrefix(ts.PkgPath(), g.targetPkg+"/")
		}
		thisPkg = path.Join(g.targetPkg, subDir)
		thisDir = path.Join(g.outputDir, subDir)
	}

	thisImports.exclude(thisPkg)
	thisImports.alias.get(thisPkg)

	fieldsPkg := "github.com/xpwu/go-mongodb/fields"
	filterPkg := "github.com/xpwu/go-mongodb/filter"
	updaterPkg := "github.com/xpwu/go-mongodb/updater"
	mongoPkg := "github.com/xpwu/go-mongodb/field"

	s := &templateData{
		Pkg:          path.Base(thisPkg),
		TypePkg:      addDot(thisImports.add(ts.PkgPath())),
		Name:         thisName,
		FilterAlias:  addDot(thisImports.add(filterPkg)),
		FieldAlias:   addDot(thisImports.add(fieldsPkg)),
		MongoAlias:   addDot(thisImports.add(mongoPkg)),
		UpdaterAlias: addDot(thisImports.add(updaterPkg)),
		Inlines:      make([]templateInline, 0),
		Fields:       make([]templateField, 0),
	}

	equalAble := true
	for i := 0; i < ts.NumField(); i++ {
		fs := ts.Field(i)
		if !fs.IsExported() {
			continue
		}

		sf := reflect.StructField{Name: fs.Name(), Tag: reflect.StructTag(fs.Tag())}
		tag, _ := x.ParseStruct(sf, !g.config.PreserveField, g.config.UseJSONTags)

		if tag == nil || tag.Skip {
			continue
		}
		if g.config.PreserveField && (tag.OmitEmpty || tag.MinSize || tag.Truncate) && !g.config.IgnoreTagErr {
			if !g.config.IgnoreTagErr {
				panic(fmt.Errorf(
					"NOT supported tag: minsize & truncate & omitempty are used in %s.%s.%s. \n"+
						"Using IgnoreTagErr() can ignore the error",
					ts.PkgPath(), ts.Name(), fs.Name()))
			} else {
				println(fmt.Sprintf(
					"NOT supported tag: minsize & truncate & omitempty are used in %s.%s.%s.",
					ts.PkgPath(), ts.Name(), fs.Name))
			}
		}

		fd := templateField{}
		fd.MethodName = fs.Name()
		fd.TagName = tag.Name

		subFt := g.build(fs.Type())
		equalAble = equalAble && subFt.EqualAble

		// 关键：如果子类型是 struct，加入待处理队列
		g.maybeEnqueue(fs.Type())

		subFName := addDot(thisImports.add(subFt.Field.PkgPath())) + subFt.Field.Name()
		subNewF := addDot(thisImports.add(subFt.NewField.PkgPath())) + subFt.NewField.Name()

		if tag.Inline && fs.Type().Kind() == reflect.Struct {
			s.Inlines = append(s.Inlines, templateInline{subFName, subNewF})
		} else {
			fd.FieldName = subFName
			fd.NewField = indentLines(subNewF, 2)
			s.Fields = append(s.Fields, fd)
		}
	}

	s.Imports = thisImports.all()
	s.EqualAble = equalAble

	if err := os.MkdirAll(thisDir, 0755); err != nil {
		panic(err)
	}

	outputPath := filepath.Join(thisDir, "z"+ts.Name()+"Field.go")
	file, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if err := structTemplate.Execute(file, s); err != nil {
		panic(err)
	}

	g.imports = oldImports

	info := TypeInfo{
		T:         ts,
		Field:     typeRef{name: thisName + "Field", pkg: thisPkg},
		NewField:  typeRef{name: "New" + thisName + "Field", pkg: thisPkg},
		EqualAble: equalAble,
	}
	g.typeMap[key] = info
	return info, true
}

// maybeEnqueue 如果类型是 struct（或解引用后是 struct），加入待处理队列
func (g *Generator) maybeEnqueue(ts TypeSource) {
	if ts == nil {
		return
	}

	// 解引用 Ptr
	actual := ts
	if ts.Kind() == reflect.Ptr {
		actual = ts.Elem()
	}
	if actual == nil {
		return
	}

	// 解引用 Slice/Array（只解一层用于判断）
	elem := actual
	if actual.Kind() == reflect.Slice || actual.Kind() == reflect.Array {
		elem = actual.Elem()
	}
	if elem == nil {
		return
	}

	// 只处理 struct 类型
	if elem.Kind() != reflect.Struct {
		return
	}

	// 跳过内置包（fields 自身、bson、geo 等已在 lookupBuiltinDirect 中处理）
	if elem.PkgPath() == "" {
		return
	}
	skipPrefixes := []string{
		"github.com/xpwu/go-mongodb/fields",
		"github.com/xpwu/go-mongodb/filter",
		"github.com/xpwu/go-mongodb/updater",
		"github.com/xpwu/go-mongodb/field",
		"go.mongodb.org/mongo-driver/v2/bson",
		"github.com/xpwu/go-mongodb/geo",
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(elem.PkgPath(), p) {
			return
		}
	}

	key := elem.PkgPath() + "." + elem.Name()
	if g.visited[key] {
		return
	}

	// 确保该类型可以加载（AST 版需要）
	elem.EnsureFields()

	g.pendingSts = append(g.pendingSts, elem)
}

// ─── 工具函数 ───────────────────────────────────────────────

func indentLines(s string, indents int) string {
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat("\t", indents) + lines[i]
	}
	return strings.Join(lines, "\n")
}
