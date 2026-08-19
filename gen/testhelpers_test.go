package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// ─── TypeLoader 辅助 ──────────────────────────────────────

// newTestLoader 创建一个干净的 TypeLoader，不依赖全局状态。
// 它会写一个最小 go.mod 到 t.TempDir()，让 findGoModDir / readModulePath 能工作。
func newTestLoader(t *testing.T) *TypeLoader {
	t.Helper()
	dir := t.TempDir()

	// 写一个最小 go.mod，让 findGoModDir / readModulePath 能工作
	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module mypkg\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	l := &TypeLoader{
		loaded:      make(map[string]*loadedPackage),
		modulePath:  "mypkg",
		goModDir:    dir,
		depVersions: make(map[string]string),
		gomodcache:  filepath.Join(dir, "modcache"),
	}
	os.MkdirAll(l.gomodcache, 0755)

	// 预注册 mypkg 包
	pkg := &loadedPackage{
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
	l.loaded["mypkg"] = pkg

	return l
}

// registerTestFile 向 loader 的 mypkg 包注册一个 Go 源文件。
// 文件写入 loader 的 goModDir，并解析、收集 import 和类型信息。
func registerTestFile(t *testing.T, l *TypeLoader, name, content string) {
	t.Helper()
	dir := l.goModDir
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}

	pkg, ok := l.loaded["mypkg"]
	if !ok {
		t.Fatal("mypkg not pre-registered in loader")
	}
	pkg.files = append(pkg.files, file)
	collectImports(file, pkg.importMap)
	collectTypeInfo(file, pkg, "mypkg", l)
}
