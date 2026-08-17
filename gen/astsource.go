package gen

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/xpwu/go-mongodb/x"
)

// ─── TypeLoader：负责从磁盘加载任意包的 AST ─────────────────

// globalLoader 全局单例
var globalLoader *TypeLoader

// GetLoader 获取全局 TypeLoader
func GetLoader() *TypeLoader {
	if globalLoader == nil {
		l := &TypeLoader{
			loaded: make(map[string]*loadedPackage),
		}
		// 尝试找到 go.mod
		dir, _ := os.Getwd()
		l.goModDir = findGoModDir(dir)
		if l.goModDir != "" {
			l.modulePath, _ = readModulePath(l.goModDir)
		}
		globalLoader = l
	}
	return globalLoader
}

// ResetLoader 重置全局 loader（主要用于测试）
func ResetLoader() {
	globalLoader = nil
}

// TypeLoader 管理所有已加载包的 AST 缓存
type TypeLoader struct {
	loaded     map[string]*loadedPackage // key: pkgPath
	modulePath string
	goModDir   string
}

// loadedPackage 一个已加载包的信息
type loadedPackage struct {
	fset      *token.FileSet
	files     []*ast.File
	importMap map[string]string          // alias → pkgPath
	aliases   map[string]reflect.Kind    // type alias: Name → kind
	types     map[string]*ast.StructType // type Name → StructType
}

// LoadPackage 加载指定包路径的所有 AST 信息（带缓存）
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

	pkg.importMap = make(map[string]string)
	pkg.aliases = make(map[string]reflect.Kind)
	pkg.types = make(map[string]*ast.StructType)

	for _, file := range pkg.files {
		// 收集 import
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if imp.Name != nil {
				pkg.importMap[imp.Name.Name] = path
			} else {
				parts := strings.Split(path, "/")
				pkg.importMap[parts[len(parts)-1]] = path
			}
		}
		// 收集类型别名和 struct 定义
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
				// 类型别名：type GPS float64
				if ident, ok := typeSpec.Type.(*ast.Ident); ok {
					pkg.aliases[typeSpec.Name.Name] = kindFromName(ident.Name)
				}
				// struct 定义
				if st, ok := typeSpec.Type.(*ast.StructType); ok {
					pkg.types[typeSpec.Name.Name] = st
				}
			}
		}
	}

	l.loaded[pkgPath] = pkg
	return pkg, nil
}

// parsePackageDir 解析目录下所有 .go 文件（排除测试和已生成的文件）
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
		fset:  fset,
		files: make([]*ast.File, 0),
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

// resolvePkgDir 将包路径解析为磁盘目录
func (l *TypeLoader) resolvePkgDir(pkgPath string) string {
	// 1. 当前 module 下的包
	if l.modulePath != "" && strings.HasPrefix(pkgPath, l.modulePath) {
		relPath := strings.TrimPrefix(pkgPath, l.modulePath+"/")
		candidate := filepath.Join(l.goModDir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// 2. GOPATH
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		candidate := filepath.Join(gopath, "src", pkgPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// 3. GOMODCACHE
	if gomodcache := os.Getenv("GOMODCACHE"); gomodcache != "" {
		parts := strings.Split(pkgPath, "/")
		if len(parts) >= 3 {
			modName := strings.Join(parts[:3], "/")
			modDir := filepath.Join(gomodcache, modName)
			if entries, err := os.ReadDir(modDir); err == nil {
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					fullModPath := filepath.Join(gomodcache, modName+"@"+e.Name())
					subRel := strings.Join(parts[3:], "/")
					candidate := filepath.Join(fullModPath, subRel)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
			}
		}
	}
	return ""
}

// ─── AST 类型实现 TypeSource / FieldSource 接口 ─────────────

type astTypeSource struct {
	name    string
	pkgPath string
	kind    reflect.Kind
	fields  []*astFieldSource
	elem    *astTypeSource
	loader  *TypeLoader
	loaded  bool // 字段是否已加载
}

func (a *astTypeSource) Name() string       { return a.name }
func (a *astTypeSource) PkgPath() string    { return a.pkgPath }
func (a *astTypeSource) Kind() reflect.Kind { return a.kind }
func (a *astTypeSource) NumField() int      { return len(a.fields) }
func (a *astTypeSource) Elem() TypeSource   { return a.elem }
func (a *astTypeSource) IsBuiltin() bool    { return a.pkgPath == "" }

func (a *astTypeSource) Field(i int) FieldSource {
	if i < 0 || i >= len(a.fields) {
		return nil
	}
	return a.fields[i]
}

// EnsureFields 懒加载字段信息：从磁盘加载该包的 AST 并解析字段
func (a *astTypeSource) EnsureFields() {
	if a.loaded || a.kind != reflect.Struct {
		return
	}
	if a.loader == nil {
		a.loader = GetLoader()
	}

	pkg, err := a.loader.LoadPackage(a.pkgPath)
	if err != nil {
		// 加载失败，当作空 struct 处理
		a.loaded = true
		return
	}

	st, ok := pkg.types[a.name]
	if !ok {
		// 可能是类型别名，检查别名表修正 kind
		if realKind, ok := pkg.aliases[a.name]; ok {
			a.kind = realKind
		}
		a.loaded = true
		return
	}

	// 解析字段
	a.fields = nil
	for _, field := range st.Fields.List {
		fs := parseAstFieldWithLoader(field, pkg.importMap, a.pkgPath, pkg.aliases, a.loader)
		a.fields = append(a.fields, fs)
	}
	a.loaded = true
}

type astFieldSource struct {
	name     string
	typ      *astTypeSource
	tag      string
	st       *x.StructTags
	exported bool
}

func (f *astFieldSource) Name() string     { return f.name }
func (f *astFieldSource) Type() TypeSource { return f.typ }
func (f *astFieldSource) Tag() string      { return f.tag }
func (f *astFieldSource) IsExported() bool { return f.exported }

// ─── 扫描结果 ───────────────────────────────────────────────

// ScanResult 扫描结果
type ScanResult struct {
	Structs []ScannedStruct
}

// ScannedStruct 一个被 go:generate 注释标记的 struct
type ScannedStruct struct {
	Name    string
	File    string
	Package string
}

// ─── ScanDir ────────────────────────────────────────────────
//
// 规则：
//   1. 找到 //go:generate 注释
//   2. 向下找【紧跟的下一个声明】（同一个 GenDecl 或下一个 GenDecl）
//   3. 中间遇到【其他有效代码】→ 停止，跳过（不报错）
//   4. 遇到文件结束 → 停止，跳过（不报错）
//   5. 找到 struct → 记录；没找到 → 静默跳过

// ScanDir 扫描目录下由 //go:generate 标记的 struct
func ScanDir(dir string) (*ScanResult, error) {
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		// 不依赖 GOFILE，直接扫描目录下所有文件
		return scanDirAll(dir)
	}
	fullPath := filepath.Join(dir, goFile)
	return scanFile(fullPath)
}

// scanDirAll 扫描目录下所有 Go 文件
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

// scanFile 扫描单个文件
func scanFile(filePath string) (*ScanResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return scanFileAST(filePath, file, fset), nil
}

// scanFileAST 从已解析的 AST 中扫描 go:generate 注释并提取 struct
func scanFileAST(filePath string, file *ast.File, fset *token.FileSet) *ScanResult {
	result := &ScanResult{}
	cmap := ast.NewCommentMap(fset, file, file.Comments)

	for _, decl := range file.Decls {
		comments := cmap.Filter(decl).Comments()
		if len(comments) == 0 {
			continue
		}

		// 检查是否有 //go:generate 注释
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

		// 情况 1：go:generate 和 struct 在同一个 GenDecl 里
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					result.Structs = append(result.Structs, ScannedStruct{
						Name:    typeSpec.Name.Name,
						File:    filePath,
						Package: file.Name.Name,
					})
					break
				}
			}
			continue
		}

		// 情况 2：go:generate 和 struct 不在同一个 GenDecl
		// 向下找【紧跟的下一个声明】，中间遇到其他有效代码就停止
		declIndex := -1
		for i, d := range file.Decls {
			if d == decl {
				declIndex = i
				break
			}
		}
		if declIndex == -1 || declIndex+1 >= len(file.Decls) {
			// 没有下一个声明了，跳过（不报错）
			continue
		}

		nextDecl := file.Decls[declIndex+1]
		genDecl, ok := nextDecl.(*ast.GenDecl)
		if !ok {
			// 下一个不是 GenDecl（比如 FuncDecl），跳过
			continue
		}

		// 检查 GenDecl 里有没有 struct
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				result.Structs = append(result.Structs, ScannedStruct{
					Name:    typeSpec.Name.Name,
					File:    filePath,
					Package: file.Name.Name,
				})
				// 如果有内嵌字段，也递归扫描
				scanInlineStructs(st, filePath, file.Name.Name, result)
				break
			}
		}
	}

	return result
}

// scanInlineStructs 扫描内嵌 struct 的字段（匿名 struct 类型）
func scanInlineStructs(st *ast.StructType, filePath, pkgName string, result *ScanResult) {
	for _, field := range st.Fields.List {
		// 匿名 struct 类型内嵌
		if len(field.Names) == 0 {
			if innerSt, ok := field.Type.(*ast.StructType); ok {
				// 生成一个合成名称
				name := fmt.Sprintf("inline_%d", len(result.Structs))
				result.Structs = append(result.Structs, ScannedStruct{
					Name:    name,
					File:    filePath,
					Package: pkgName,
				})
				// 递归
				scanInlineStructs(innerSt, filePath, pkgName, result)
			}
		}
	}
}

// ─── ParseStructFromFile ────────────────────────────────────

// ParseStructFromFile 从文件中解析指定 struct 的 AST TypeSource
func ParseStructFromFile(dir, structName string) (TypeSource, error) {
	goFile := os.Getenv("GOFILE")
	loader := GetLoader()

	if goFile != "" {
		targetFile := filepath.Join(dir, goFile)
		return parseStructFromFile(targetFile, structName, loader)
	}
	return parseStructFromDir(dir, structName, loader)
}

func parseStructFromFile(filePath, structName string, loader *TypeLoader) (TypeSource, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	importMap := buildImportMap(file)
	typeAliases := collectTypeAliases(file)

	// 获取完整 pkgPath
	pkgPath := os.Getenv("GOPACKAGE")
	if pkgPath == "" || !strings.Contains(pkgPath, "/") {
		// 先尝试从 go.mod 推导
		modulePath, _ := readModulePath(filepath.Dir(filePath))
		if modulePath != "" {
			goModDir := findGoModDir(filepath.Dir(filePath))
			if goModDir != "" {
				rel, _ := filepath.Rel(goModDir, filepath.Dir(filePath))
				if rel != "." {
					pkgPath = modulePath + "/" + filepath.ToSlash(rel)
				} else {
					pkgPath = modulePath
				}
			} else {
				pkgPath = modulePath
			}
		}

		// 如果还是没拼出来（go.mod 找不到），用 packages 包推断
		if pkgPath == "" || !strings.Contains(pkgPath, "/") {
			if inferred, err := InferPackagePath(filepath.Dir(filePath)); err == nil && inferred != "" {
				pkgPath = inferred
			}
		}
	}

	// 同时把当前文件注册到 loader 中
	registerFileToLoader(loader, pkgPath, file, filePath)

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
			if typeSpec.Name.Name != structName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			ts := &astTypeSource{
				name:    structName,
				pkgPath: pkgPath,
				kind:    reflect.Struct,
				loader:  loader,
			}
			for _, field := range structType.Fields.List {
				fs := parseAstFieldWithLoader(field, importMap, pkgPath, typeAliases, loader)
				ts.fields = append(ts.fields, fs)
			}
			ts.loaded = true
			return ts, nil
		}
	}
	return nil, nil
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

	// 获取完整 pkgPath（和 parseStructFromFile 同样的逻辑）
	pkgPath := os.Getenv("GOPACKAGE")
	if pkgPath == "" || !strings.Contains(pkgPath, "/") {
		modulePath, _ := readModulePath(dir)
		if modulePath != "" {
			goModDir := findGoModDir(dir)
			if goModDir != "" {
				rel, _ := filepath.Rel(goModDir, dir)
				if rel != "." && !strings.HasPrefix(rel, "..") {
					pkgPath = modulePath + "/" + filepath.ToSlash(rel)
				} else {
					pkgPath = modulePath
				}
			} else {
				pkgPath = modulePath
			}
		}

		// fallback：用 packages 推断
		if pkgPath == "" || !strings.Contains(pkgPath, "/") {
			if inferred, err := InferPackagePath(dir); err == nil && inferred != "" {
				pkgPath = inferred
			}
		}
	}

	importMap := make(map[string]string)

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			if strings.HasSuffix(fileName, "_test.go") {
				continue
			}
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, "\"")
				if imp.Name != nil {
					importMap[imp.Name.Name] = path
				} else {
					parts := strings.Split(path, "/")
					importMap[parts[len(parts)-1]] = path
				}
			}

			typeAliases := collectTypeAliases(file)
			registerFileToLoader(loader, pkgPath, file, fileName)

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
					if typeSpec.Name.Name != structName {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					ts := &astTypeSource{
						name:    structName,
						pkgPath: pkgPath,
						kind:    reflect.Struct,
						loader:  loader,
					}
					for _, field := range structType.Fields.List {
						fs := parseAstFieldWithLoader(field, importMap, pkgPath, typeAliases, loader)
						ts.fields = append(ts.fields, fs)
					}
					ts.loaded = true
					return ts, nil
				}
			}
		}
	}
	return nil, nil
}

// registerFileToLoader 把单个文件的 struct 定义和别名注册到 loader 缓存中
func registerFileToLoader(loader *TypeLoader, pkgPath string, file *ast.File, fileName string) {
	if loader == nil || pkgPath == "" {
		return
	}
	pkg, ok := loader.loaded[pkgPath]
	if !ok {
		// 创建一个空的 loadedPackage 占位
		pkg = &loadedPackage{
			fset:      token.NewFileSet(),
			files:     make([]*ast.File, 0),
			importMap: make(map[string]string),
			aliases:   make(map[string]reflect.Kind),
			types:     make(map[string]*ast.StructType),
		}
		loader.loaded[pkgPath] = pkg
	}

	// 避免重复注册
	for _, f := range pkg.files {
		if f == file {
			return
		}
	}
	pkg.files = append(pkg.files, file)

	// 收集 import
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			pkg.importMap[imp.Name.Name] = path
		} else {
			parts := strings.Split(path, "/")
			pkg.importMap[parts[len(parts)-1]] = path
		}
	}

	// 收集类型别名和 struct 定义
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
			if ident, ok := typeSpec.Type.(*ast.Ident); ok {
				pkg.aliases[typeSpec.Name.Name] = kindFromName(ident.Name)
			}
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				pkg.types[typeSpec.Name.Name] = st
			}
		}
	}
}

// ─── 收集类型别名 ───────────────────────────────────────────

func collectTypeAliases(file *ast.File) map[string]reflect.Kind {
	aliases := make(map[string]reflect.Kind)
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
			if ident, ok := typeSpec.Type.(*ast.Ident); ok {
				aliases[typeSpec.Name.Name] = kindFromName(ident.Name)
			}
		}
	}
	return aliases
}

// ─── AST 解析辅助 ───────────────────────────────────────────

func buildImportMap(file *ast.File) map[string]string {
	importMap := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			importMap[imp.Name.Name] = path
		} else {
			parts := strings.Split(path, "/")
			importMap[parts[len(parts)-1]] = path
		}
	}
	return importMap
}

// parseAstFieldWithLoader 带 loader 的字段解析
func parseAstFieldWithLoader(f *ast.Field, importMap map[string]string, currentPkgPath string, typeAliases map[string]reflect.Kind, loader *TypeLoader) *astFieldSource {
	name := ""
	if len(f.Names) > 0 {
		name = f.Names[0].Name
	}
	tagStr := ""
	if f.Tag != nil {
		tagStr = strings.Trim(f.Tag.Value, "`")
	}

	return &astFieldSource{
		name: name,
		typ:  parseAstTypeWithLoader(f.Type, importMap, currentPkgPath, typeAliases, loader),
		tag:  tagStr,
		//st:       st,
		exported: name != "" && name[0] >= 'A' && name[0] <= 'Z',
	}
}

// parseAstTypeWithLoader 带 loader 的类型表达式解析
func parseAstTypeWithLoader(expr ast.Expr, importMap map[string]string, currentPkgPath string, typeAliases map[string]reflect.Kind, loader *TypeLoader) *astTypeSource {
	switch t := expr.(type) {
	case *ast.Ident:
		k := kindFromName(t.Name)
		// 关键：如果 kindFromName 直接识别为内置类型（非 Struct），
		// 说明这是 Go 原生类型名（int/string/bool 等），pkgPath 必须为空
		resolvedPkgPath := currentPkgPath
		if k != reflect.Struct && isBuiltinKind(k) {
			resolvedPkgPath = ""
		}

		// 查类型别名表
		if realKind, ok := typeAliases[t.Name]; ok {
			k = realKind
		}

		// 如果是 struct 且在当前包的 loader 缓存中存在，kind 保持 Struct
		if k == reflect.Struct {
			if loader != nil {
				if pkg, err := loader.LoadPackage(currentPkgPath); err == nil {
					if _, ok := pkg.types[t.Name]; ok {
						k = reflect.Struct
					}
				}
			}
		}
		return &astTypeSource{
			name:    t.Name,
			pkgPath: resolvedPkgPath,
			kind:    k,
			loader:  loader,
		}
	case *ast.SelectorExpr:
		pkgAlias := ""
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgAlias = ident.Name
		}
		fullPkgPath := importMap[pkgAlias]

		// 尝试解析为内置类型（如 bson.ObjectID）
		k := reflect.Struct // 默认当作 struct
		if loader != nil {
			if pkg, err := loader.LoadPackage(fullPkgPath); err == nil {
				if realKind, ok := pkg.aliases[t.Sel.Name]; ok {
					k = realKind
				}
			}
		}

		return &astTypeSource{
			name:    t.Sel.Name,
			pkgPath: fullPkgPath,
			kind:    k,
			loader:  loader,
		}
	case *ast.StarExpr:
		elem := parseAstTypeWithLoader(t.X, importMap, currentPkgPath, typeAliases, loader)
		return &astTypeSource{name: "*" + elem.name, kind: reflect.Ptr, elem: elem, loader: loader}
	case *ast.ArrayType:
		elem := parseAstTypeWithLoader(t.Elt, importMap, currentPkgPath, typeAliases, loader)
		return &astTypeSource{name: "[]" + elem.name, kind: reflect.Slice, elem: elem, loader: loader}
	default:
		return &astTypeSource{name: "unknown", kind: reflect.Interface, loader: loader}
	}
}

// ─── 工具函数 ───────────────────────────────────────────────

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

func findGoModDir(dir string) string {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// isBuiltinKind 判断是否为 Go 内置基本类型
func isBuiltinKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String, reflect.Bool:
		return true
	}
	return false
}
