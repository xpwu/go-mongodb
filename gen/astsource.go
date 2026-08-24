package gen

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// ─── TypeLoader：负责从磁盘加载任意包的 AST ─────────────────

var globalLoader *TypeLoader

func GetLoader() *TypeLoader {
	if globalLoader == nil {
		l := &TypeLoader{
			loaded:      make(map[string]*loadedPackage),
			depVersions: make(map[string]string),
		}
		dir, _ := os.Getwd()
		l.goModDir = FindGoModDir(dir)
		if l.goModDir != "" {
			l.modulePath, _ = readModulePath(l.goModDir)
			goModPath := filepath.Join(l.goModDir, "go.mod")
			if data, err := os.ReadFile(goModPath); err == nil {
				if modFile, err := modfile.Parse(goModPath, data, nil); err == nil {
					for _, r := range modFile.Require {
						l.depVersions[r.Mod.Path] = r.Mod.Version
					}
				}
			}
		}
		l.gomodcache = os.Getenv("GOMODCACHE")
		if l.gomodcache == "" {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				if home, err := os.UserHomeDir(); err == nil {
					gopath = filepath.Join(home, "go")
				}
			}
			if gopath != "" {
				l.gomodcache = filepath.Join(gopath, "pkg", "mod")
			}
		}
		globalLoader = l
	}
	return globalLoader
}

func ResetLoader() {
	globalLoader = nil
}

type TypeLoader struct {
	loaded      map[string]*loadedPackage
	modulePath  string
	goModDir    string
	gomodcache  string
	depVersions map[string]string
}

type loadedPackage struct {
	fset             *token.FileSet
	files            []*ast.File
	importMap        map[string]string
	typeElems        map[string]*astTypeSource
	types            map[string]*ast.StructType
	aliasTargets     map[string]*astTypeSource
	typeDefTargets   map[string]string
	interfaceTargets map[string]bool
	underlyingCache  map[string]*astTypeSource
}

// ─── 核心加载逻辑 ────────────────────────────────────────────

func (l *TypeLoader) LoadPackage(pkgPath string) (*loadedPackage, error) {
	if pkgPath == "" {
		return nil, fmt.Errorf("empty pkgPath")
	}
	if pkg, ok := l.loaded[pkgPath]; ok {
		return pkg, nil
	}

	pkgDir := l.resolvePkgDir(pkgPath)
	if pkgDir == "" {
		return nil, fmt.Errorf("cannot find directory for package %s", pkgPath)
	}

	pkg, err := l.parsePackageDir(pkgDir)
	if err != nil {
		return nil, err
	}

	l.loaded[pkgPath] = pkg

	// 收集 import 和类型信息
	for _, file := range pkg.files {
		collectImports(file, pkg.importMap)
		collectTypeInfo(file, pkg, pkgPath, l)
	}

	return pkg, nil
}

func (l *TypeLoader) parsePackageDir(dir string) (*loadedPackage, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		if strings.HasPrefix(name, "z") && strings.HasSuffix(name, "Field.go") {
			return false
		}
		return true
	}, 0)
	if err != nil {
		return nil, err
	}

	pkg := &loadedPackage{
		fset:             fset,
		files:            make([]*ast.File, 0),
		importMap:        make(map[string]string),
		typeElems:        make(map[string]*astTypeSource),
		types:            make(map[string]*ast.StructType),
		aliasTargets:     make(map[string]*astTypeSource),
		typeDefTargets:   make(map[string]string),
		interfaceTargets: make(map[string]bool),
		underlyingCache:  make(map[string]*astTypeSource),
	}
	for _, p := range pkgs {
		for _, file := range p.Files {
			pkg.files = append(pkg.files, file)
		}
	}
	if len(pkg.files) == 0 {
		return nil, fmt.Errorf("no Go files found in %s", dir)
	}
	return pkg, nil
}

// ─── 三方库路径解析（不动）──────────────────────────────────

func (l *TypeLoader) resolvePkgDir(pkgPath string) string {
	// 1. 当前 module 下的包
	if l.modulePath != "" && strings.HasPrefix(pkgPath, l.modulePath) {
		relPath := strings.TrimPrefix(pkgPath, l.modulePath+"/")
		candidate := filepath.Join(l.goModDir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// 2. GOPATH/src
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath != "" {
		candidate := filepath.Join(gopath, "src", pkgPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// 3. GOMODCACHE
	gomodcache := l.gomodcache
	if gomodcache == "" {
		gomodcache = os.Getenv("GOMODCACHE")
	}
	if gomodcache == "" && gopath != "" {
		gomodcache = filepath.Join(gopath, "pkg", "mod")
	}
	if gomodcache != "" {
		for mod, ver := range l.depVersions {
			if pkgPath == mod || strings.HasPrefix(pkgPath, mod+"/") {
				subRel := strings.TrimPrefix(pkgPath, mod+"/")
				candidate := filepath.Join(l.gomodcache, mod+"@"+ver, subRel)
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
				break
			}
		}
	}
	return ""
}

// ─── 提取的辅助函数 ──────────────────────────────────────────

func collectImports(file *ast.File, importMap map[string]string) {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			importMap[imp.Name.Name] = path
		} else {
			parts := strings.Split(path, "/")
			importMap[parts[len(parts)-1]] = path
		}
	}
}

func collectTypeInfo(file *ast.File, pkg *loadedPackage, pkgPath string, loader *TypeLoader) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// 类型别名声明：type A = C
			if typeSpec.Assign != token.NoPos {
				elem := parseAstTypeWithLoader(typeSpec.Type, pkg.importMap, pkgPath, loader)
				pkg.aliasTargets[typeSpec.Name.Name] = elem
				continue
			}
			// 类型定义：type A B（右侧是 Ident）
			if ident, ok := typeSpec.Type.(*ast.Ident); ok {
				pkg.typeDefTargets[typeSpec.Name.Name] = ident.Name
			}
			// 类型定义：type A interface{Do()}
			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				pkg.interfaceTargets[typeSpec.Name.Name] = true
			}
			// 复合类型：type D []E
			if arrayType, ok := typeSpec.Type.(*ast.ArrayType); ok {
				elem := parseAstTypeWithLoader(arrayType.Elt, pkg.importMap, pkgPath, loader)
				pkg.typeElems[typeSpec.Name.Name] = elem
			}
			// struct 定义
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				pkg.types[typeSpec.Name.Name] = st
			}
		}
	}
}

// ─── AST 类型实现 ────────────────────────────────────────────

type astTypeSource struct {
	name    string
	pkgPath string
	kind    reflect.Kind
	fields  []*astFieldSource
	elem    *astTypeSource
	loader  *TypeLoader
	loaded  bool
}

func (a *astTypeSource) Name() string    { return a.name }
func (a *astTypeSource) PkgPath() string { return a.pkgPath }
func (a *astTypeSource) Kind() reflect.Kind {
	next, _ := a.Underlying()
	if next != nil {
		return next.Kind()
	}

	return a.kind
}
func (a *astTypeSource) NumField() int { return len(a.fields) }
func (a *astTypeSource) Elem() TypeSource {
	if a.elem == nil {
		return nil
	}
	return a.elem
}
func (a *astTypeSource) IsBuiltin() bool { return a.pkgPath == "" }

func (a *astTypeSource) Field(i int) FieldSource {
	if i < 0 || i >= len(a.fields) {
		return nil
	}
	return a.fields[i]
}

func (a *astTypeSource) EnsureFields() {
	if a.loaded {
		return
	}
	if a.kind != reflect.Struct {
		a.loaded = true
		return
	}
	if a.loader == nil {
		a.loader = GetLoader()
	}

	pkg, err := a.loader.LoadPackage(a.pkgPath)
	if err != nil {
		a.loaded = true
		return
	}

	st, ok := pkg.types[a.name]
	if !ok {
		a.loaded = true
		return
	}

	a.fields = nil
	for _, field := range st.Fields.List {
		fs := parseAstFieldWithLoader(field, pkg.importMap, a.pkgPath, a.loader)
		a.fields = append(a.fields, fs)
	}
	a.loaded = true
}

func (a *astTypeSource) Underlying() (TypeSource, bool) {
	if a.loader == nil {
		return nil, false
	}
	pkg, err := a.loader.LoadPackage(a.pkgPath)
	if err != nil {
		return nil, false
	}
	// 2. 查 aliasTargets（自己是别名）
	if target, ok := pkg.aliasTargets[a.name]; ok {
		return target, true
	}
	// 3. 查 typeDefTargets（自己是类型定义）
	if next, ok := pkg.typeDefTargets[a.name]; ok && next != "" {
		if cached, ok := pkg.underlyingCache[a.name]; ok {
			return cached, false
		}
		nextTS := parseAstTypeWithLoader(&ast.Ident{Name: next}, pkg.importMap, a.pkgPath, a.loader)
		pkg.underlyingCache[a.name] = nextTS
		return nextTS, false
	}
	// 4. 查 typeElems（自己是复合类型）
	if elem, ok := pkg.typeElems[a.name]; ok {
		return &astTypeSource{
			name:   "[]" + elem.name,
			kind:   reflect.Slice,
			elem:   elem,
			loader: a.loader,
		}, false
	}
	return nil, false
}

type astFieldSource struct {
	name     string
	typ      *astTypeSource
	tag      string
	exported bool
}

func (f *astFieldSource) Name() string     { return f.name }
func (f *astFieldSource) Type() TypeSource { return f.typ }
func (f *astFieldSource) Tag() string      { return f.tag }
func (f *astFieldSource) IsExported() bool { return f.exported }

// ─── Scan ────────────────────────────────────────────────────

type ScanResult struct {
	Structs []ScannedStruct
}

type ScannedStruct struct {
	Name    string
	File    string
	Package string
}

func ScanDir(dir string) (*ScanResult, error) {
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		return scanDirAll(dir)
	}
	fullPath := filepath.Join(dir, goFile)
	return scanFile(fullPath)
}

func scanDirAll(dir string) (*ScanResult, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		if strings.HasPrefix(name, "z") && strings.HasSuffix(name, "Field.go") {
			return false
		}
		return true
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	result := &ScanResult{}
	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			r := scanFileAST(fileName, file, fset)
			result.Structs = append(result.Structs, r.Structs...)
		}
	}
	return result, nil
}

func scanFile(filePath string) (*ScanResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return scanFileAST(filePath, file, fset), nil
}

func scanFileAST(filePath string, file *ast.File, fset *token.FileSet) *ScanResult {
	result := &ScanResult{}
	cmap := ast.NewCommentMap(fset, file, file.Comments)

	for _, decl := range file.Decls {
		comments := cmap.Filter(decl).Comments()
		if len(comments) == 0 {
			continue
		}
		var hasGenerate bool
		for _, cg := range comments {
			for _, comment := range cg.List {
				if strings.Contains(comment.Text, "//go:generate") {
					hasGenerate = true
					break
				}
			}
			if hasGenerate {
				break
			}
		}
		if !hasGenerate {
			continue
		}

		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					result.Structs = append(result.Structs, ScannedStruct{
						Name: typeSpec.Name.Name, File: filePath, Package: file.Name.Name,
					})
					break
				}
			}
			continue
		}

		declIndex := -1
		for i, d := range file.Decls {
			if d == decl {
				declIndex = i
				break
			}
		}
		if declIndex == -1 || declIndex+1 >= len(file.Decls) {
			continue
		}
		nextDecl, ok := file.Decls[declIndex+1].(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range nextDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				result.Structs = append(result.Structs, ScannedStruct{
					Name: typeSpec.Name.Name, File: filePath, Package: file.Name.Name,
				})
				scanInlineStructs(st, filePath, file.Name.Name, result)
				break
			}
		}
	}
	return result
}

func scanInlineStructs(st *ast.StructType, filePath, pkgName string, result *ScanResult) {
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			if innerSt, ok := field.Type.(*ast.StructType); ok {
				name := fmt.Sprintf("inline_%d", len(result.Structs))
				result.Structs = append(result.Structs, ScannedStruct{
					Name: name, File: filePath, Package: pkgName,
				})
				scanInlineStructs(innerSt, filePath, pkgName, result)
			}
		}
	}
}

// ─── ParseStructFromFile ────────────────────────────────────

func ParseStructFromFile(dir, structName string) (TypeSource, error) {
	loader := GetLoader()
	goFile := os.Getenv("GOFILE")
	if goFile != "" {
		return parseStructFromFile(filepath.Join(dir, goFile), structName, loader)
	}
	return parseStructFromDir(dir, structName, loader)
}

func parseStructFromFile(filePath, structName string, loader *TypeLoader) (TypeSource, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	importMap := make(map[string]string)
	collectImports(file, importMap)
	pkgPath := resolvePkgPath(filepath.Dir(filePath))
	registerFileToLoader(loader, pkgPath, file, filePath)

	return findStructInFile(file, structName, importMap, pkgPath, loader)
}

func parseStructFromDir(dir, structName string, loader *TypeLoader) (TypeSource, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		if strings.HasPrefix(name, "z") && strings.HasSuffix(name, "Field.go") {
			return false
		}
		return true
	}, 0)
	if err != nil {
		return nil, err
	}

	pkgPath := resolvePkgPath(dir)
	importMap := make(map[string]string)

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			if strings.HasSuffix(fileName, "_test.go") {
				continue
			}
			collectImports(file, importMap)
			registerFileToLoader(loader, pkgPath, file, fileName)

			if ts, err := findStructInFile(file, structName, importMap, pkgPath, loader); ts != nil || err != nil {
				return ts, err
			}
		}
	}
	return nil, nil
}

// ─── 提取的 struct 查找函数 ─────────────────────────────────

func findStructInFile(file *ast.File, structName string, importMap map[string]string, pkgPath string, loader *TypeLoader) (TypeSource, error) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return buildStructTypeSource(structName, pkgPath, structType, importMap, loader), nil
		}
	}
	return nil, nil
}

func buildStructTypeSource(structName, pkgPath string, structType *ast.StructType, importMap map[string]string, loader *TypeLoader) *astTypeSource {
	ts := &astTypeSource{
		name:    structName,
		pkgPath: pkgPath,
		kind:    reflect.Struct,
		loader:  loader,
	}
	for _, field := range structType.Fields.List {
		fs := parseAstFieldWithLoader(field, importMap, pkgPath, loader)
		ts.fields = append(ts.fields, fs)
	}
	return ts
}

// ─── registerFileToLoader ───────────────────────────────────

func registerFileToLoader(loader *TypeLoader, pkgPath string, file *ast.File, fileName string) {
	if loader == nil || pkgPath == "" {
		return
	}
	pkg, ok := loader.loaded[pkgPath]
	if !ok {
		pkg = &loadedPackage{
			fset:             token.NewFileSet(),
			files:            make([]*ast.File, 0),
			importMap:        make(map[string]string),
			typeElems:        make(map[string]*astTypeSource),
			types:            make(map[string]*ast.StructType),
			aliasTargets:     make(map[string]*astTypeSource),
			typeDefTargets:   make(map[string]string),
			interfaceTargets: make(map[string]bool),
			underlyingCache:  make(map[string]*astTypeSource),
		}
		loader.loaded[pkgPath] = pkg
	}
	for _, f := range pkg.files {
		if f == file {
			return
		}
	}
	pkg.files = append(pkg.files, file)
	collectImports(file, pkg.importMap)
	collectTypeInfo(file, pkg, pkgPath, loader)
}

// ─── AST 解析辅助 ───────────────────────────────────────────

func parseAstFieldWithLoader(f *ast.Field, importMap map[string]string, currentPkgPath string, loader *TypeLoader) *astFieldSource {
	name := ""
	if len(f.Names) > 0 {
		name = f.Names[0].Name
	}
	tagStr := ""
	if f.Tag != nil {
		tagStr = strings.Trim(f.Tag.Value, "`")
	}
	return &astFieldSource{
		name:     name,
		typ:      parseAstTypeWithLoader(f.Type, importMap, currentPkgPath, loader),
		tag:      tagStr,
		exported: name != "" && name[0] >= 'A' && name[0] <= 'Z',
	}
}

func parseAstTypeWithLoader(expr ast.Expr, importMap map[string]string, currentPkgPath string, loader *TypeLoader) *astTypeSource {
	switch t := expr.(type) {
	case *ast.Ident:
		k := kindFromName(t.Name)
		pkgPath := currentPkgPath
		if isBuiltinKind(k) {
			pkgPath = ""
		}
		// 检查是不是用户定义的 interface 类型
		if loader != nil {
			if pkg, err := loader.LoadPackage(currentPkgPath); err == nil {
				if pkg.interfaceTargets[t.Name] {
					k = reflect.Interface
					pkgPath = ""
				}
			}
		}
		return &astTypeSource{name: t.Name, pkgPath: pkgPath, kind: k, loader: loader}

	case *ast.SelectorExpr:
		pkgAlias := ""
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgAlias = ident.Name
		}
		fullPkgPath := importMap[pkgAlias]

		k := reflect.Struct
		if t.Sel.Name == "any" || t.Sel.Name == "interface{}" {
			k = reflect.Interface
			fullPkgPath = ""
		}

		var elem *astTypeSource
		if loader != nil {
			if pkg, err := loader.LoadPackage(fullPkgPath); err == nil {
				elem = pkg.typeElems[t.Sel.Name]
			}
		}

		return &astTypeSource{name: t.Sel.Name, pkgPath: fullPkgPath, kind: k, elem: elem, loader: loader}

	case *ast.StarExpr:
		elem := parseAstTypeWithLoader(t.X, importMap, currentPkgPath, loader)
		return &astTypeSource{name: "*" + elem.name, kind: reflect.Ptr, elem: elem, loader: loader}

	case *ast.ArrayType:
		elem := parseAstTypeWithLoader(t.Elt, importMap, currentPkgPath, loader)
		return &astTypeSource{name: "[]" + elem.name, kind: reflect.Slice, elem: elem, loader: loader}

	case *ast.InterfaceType:
		return &astTypeSource{name: "", pkgPath: "", kind: reflect.Interface, loader: loader}

	default:
		// 保底：未知类型当作 interface（任意类型），如果用户发现生成不对，说明该类型还没被支持
		return &astTypeSource{name: "unknown", kind: reflect.Interface, loader: loader}
	}
}

// ─── 工具函数 ───────────────────────────────────────────────

func resolvePkgPath(dir string) string {
	pkgPath := os.Getenv("GOPACKAGE")
	if pkgPath != "" && strings.Contains(pkgPath, "/") {
		return pkgPath
	}
	modulePath, _ := readModulePath(dir)
	if modulePath != "" {
		goModDir := FindGoModDir(dir)
		if goModDir != "" {
			rel, _ := filepath.Rel(goModDir, dir)
			if rel != "." && !strings.HasPrefix(rel, "..") {
				return modulePath + "/" + filepath.ToSlash(rel)
			}
		}
		return modulePath
	}
	if inferred, err := InferPackagePath(dir); err == nil && inferred != "" {
		return inferred
	}
	return ""
}

func kindFromName(name string) reflect.Kind {
	switch name {
	case "int":
		return reflect.Int
	case "int8":
		return reflect.Int8
	case "int16":
		return reflect.Int16
	case "int32":
		return reflect.Int32
	case "int64":
		return reflect.Int64
	case "uint":
		return reflect.Uint
	case "uint8":
		return reflect.Uint8
	case "uint16":
		return reflect.Uint16
	case "uint32":
		return reflect.Uint32
	case "uint64":
		return reflect.Uint64
	case "float32":
		return reflect.Float32
	case "float64":
		return reflect.Float64
	case "string":
		return reflect.String
	case "bool":
		return reflect.Bool
	case "any", "interface{}":
		return reflect.Interface
	case "byte":
		return reflect.Uint8
	case "rune":
		return reflect.Int32
	default:
		return reflect.Struct
	}
}

func readModulePath(dir string) (string, error) {
	current := dir
	for {
		gomod := filepath.Join(current, "go.mod")
		f, err := os.Open(gomod)
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
				}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

// FindGoModDir 从 start 目录开始向上查找包含 go.mod 的目录。
// 找不到返回空字符串。
func FindGoModDir(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	dir = filepath.Clean(dir)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isBuiltinKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String, reflect.Bool, reflect.Interface:
		return true
	}
	return false
}
