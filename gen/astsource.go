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

// ─── AST 类型实现 TypeSource / FieldSource 接口 ─────────────

type astTypeSource struct {
	name    string
	pkgPath string
	kind    reflect.Kind
	fields  []*astFieldSource
	elem    *astTypeSource
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

type astFieldSource struct {
	name     string
	typ      *astTypeSource
	tag      string
	st       *x.StructTags
	exported bool
}

func (f *astFieldSource) Name() string             { return f.name }
func (f *astFieldSource) Type() TypeSource         { return f.typ }
func (f *astFieldSource) Tag() string              { return f.tag }
func (f *astFieldSource) StructTag() *x.StructTags { return f.st }
func (f *astFieldSource) IsExported() bool         { return f.exported }

// ─── 扫描结果 ───────────────────────────────────────────────

type ScanResult struct {
	Structs []ScannedStruct
}

type ScannedStruct struct {
	Name    string
	File    string
	Package string
}

// ─── ScanDir ────────────────────────────────────────────────

func ScanDir(dir string) (*ScanResult, error) {
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		return nil, fmt.Errorf("GOFILE not set, not running under go generate")
	}

	fullPath := filepath.Join(dir, goFile)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

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

		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			_, ok = typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			result.Structs = append(result.Structs, ScannedStruct{
				Name:    typeSpec.Name.Name,
				File:    fullPath,
				Package: file.Name.Name,
			})
			break
		}
	}

	return result, nil
}

// ─── ParseStructFromFile ────────────────────────────────────

func ParseStructFromFile(dir, structName string) (TypeSource, error) {
	goFile := os.Getenv("GOFILE")
	if goFile != "" {
		targetFile := filepath.Join(dir, goFile)
		return parseStructFromFile(targetFile, structName)
	}
	return parseStructFromDir(dir, structName)
}

func parseStructFromFile(filePath, structName string) (TypeSource, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	importMap := buildImportMap(file)

	// 收集同文件里的类型别名（type GPS float64 → GPS: reflect.Float64）
	typeAliases := collectTypeAliases(file)

	pkgPath := os.Getenv("GOPACKAGE")
	if pkgPath == "" {
		modulePath, _ := readModulePath(filepath.Dir(filePath))
		if modulePath != "" {
			rel, _ := filepath.Rel(findGoModDir(filepath.Dir(filePath)), filepath.Dir(filePath))
			if rel != "." {
				pkgPath = modulePath + "/" + filepath.ToSlash(rel)
			} else {
				pkgPath = modulePath
			}
		}
	}

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
			}
			for _, field := range structType.Fields.List {
				fs := parseAstField(field, importMap, pkgPath, typeAliases)
				ts.fields = append(ts.fields, fs)
			}
			return ts, nil
		}
	}
	return nil, nil
}

func parseStructFromDir(dir, structName string) (TypeSource, error) {
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

	pkgPath := os.Getenv("GOPACKAGE")
	if pkgPath == "" {
		modulePath, _ := readModulePath(dir)
		if modulePath != "" {
			pkgPath = modulePath
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

			// 收集类型别名
			typeAliases := collectTypeAliases(file)

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
					}
					for _, field := range structType.Fields.List {
						fs := parseAstField(field, importMap, pkgPath, typeAliases)
						ts.fields = append(ts.fields, fs)
					}
					return ts, nil
				}
			}
		}
	}
	return nil, nil
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

func parseAstField(f *ast.Field, importMap map[string]string, currentPkgPath string, typeAliases map[string]reflect.Kind) *astFieldSource {
	name := ""
	if len(f.Names) > 0 {
		name = f.Names[0].Name
	}
	tagStr := ""
	if f.Tag != nil {
		tagStr = strings.Trim(f.Tag.Value, "`")
	}

	var st *x.StructTags
	if tagStr != "" {
		sf := reflect.StructField{Name: name, Tag: reflect.StructTag(tagStr)}
		st, _ = x.ParseStructTags(sf)
	} else {
		sf := reflect.StructField{Name: name, Tag: ""}
		st, _ = x.ParseStructTags(sf)
	}

	return &astFieldSource{
		name:     name,
		typ:      parseAstType(f.Type, importMap, currentPkgPath, typeAliases),
		tag:      tagStr,
		st:       st,
		exported: name != "" && name[0] >= 'A' && name[0] <= 'Z',
	}
}

func parseAstType(expr ast.Expr, importMap map[string]string, currentPkgPath string, typeAliases map[string]reflect.Kind) *astTypeSource {
	switch t := expr.(type) {
	case *ast.Ident:
		k := kindFromName(t.Name)
		if k == reflect.Struct {
			// 查类型别名表，看是不是基本类型的别名（如 type GPS float64）
			if realKind, ok := typeAliases[t.Name]; ok {
				k = realKind
			}
		}
		return &astTypeSource{
			name:    t.Name,
			pkgPath: currentPkgPath,
			kind:    k,
		}
	case *ast.SelectorExpr:
		pkgAlias := ""
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgAlias = ident.Name
		}
		fullPkgPath := importMap[pkgAlias]
		return &astTypeSource{
			name:    t.Sel.Name,
			pkgPath: fullPkgPath,
			kind:    reflect.Struct,
		}
	case *ast.StarExpr:
		elem := parseAstType(t.X, importMap, currentPkgPath, typeAliases)
		return &astTypeSource{name: "*" + elem.name, kind: reflect.Ptr, elem: elem}
	case *ast.ArrayType:
		elem := parseAstType(t.Elt, importMap, currentPkgPath, typeAliases)
		return &astTypeSource{name: "[]" + elem.name, kind: reflect.Slice, elem: elem}
	default:
		return &astTypeSource{name: "unknown", kind: reflect.Interface}
	}
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
			return dir
		}
		current = parent
	}
}
