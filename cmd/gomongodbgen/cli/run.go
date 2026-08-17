package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xpwu/go-mongodb/gen"
)

type BuildConfig struct {
	config *gen.Config
	types  []string
}

func NewBuildConfig() *BuildConfig {
	return &BuildConfig{
		config: gen.NewConfig(),
	}
}

func (b *BuildConfig) AddType(name string) *BuildConfig {
	b.types = append(b.types, name)
	return b
}

func (b *BuildConfig) OutDir(dir string) *BuildConfig {
	b.config.Dir = dir
	return b
}

func (b *BuildConfig) TargetPkg(pkg string) *BuildConfig {
	b.config.Pkg = pkg
	return b
}

func (b *BuildConfig) PreserveField(ignoreTagErr bool) *BuildConfig {
	b.config.PreserveField = true
	b.config.IgnoreTagErr = ignoreTagErr
	return b
}

// AddMap adds a custom type mapping.
//
// typeIdent is the fully qualified type identifier of the SOURCE type:
//   - Builtin:      "int"
//   - Same pkg:     "MyType"
//   - External pkg: "github.com/foo/bar.MyType"
//
// fieldType is the fully qualified type identifier of the TARGET field type.
// newFunc is the fully qualified constructor function name.
//
// equalAble should be true if fieldType implements or embeds
// github.com/xpwu/filter/ComparableFilter (directly or transitively through
// embedded interfaces/structs). Otherwise, set to false.
//
// Example:
//
//	cli.NewBuildConfig().
//	    Type("Order").
//	    AddMap("github.com/foo/bar.GPS", "github.com/foo/fields.FloatField", "github.com/foo/fields.NewFloatField", false).
//	    Run()
//
// NOTE: Generic types and generic functions are NOT supported.
func (b *BuildConfig) AddMap(typeIdent, fieldType, newFunc string, equalAble bool) *BuildConfig {
	b.config.AddMap(typeIdent, fieldType, newFunc, equalAble)
	return b
}

// determineSrcDir 确定源文件所在目录
// 通过调用栈找到用户的 mongodb_gen.go 所在目录
func determineSrcDir() string {
	_, file, _, ok := runtime.Caller(2)
	if ok {
		dir := filepath.Dir(file)
		if strings.HasSuffix(dir, "cmd/go-mongodb-gen/cli") {
			// 可能被间接调用，再往上找一层
			dir = filepath.Dir(filepath.Dir(filepath.Dir(dir)))
		}
		return dir
	}
	dir, _ := os.Getwd()
	return dir
}

func (b *BuildConfig) Run() {
	srcDir := determineSrcDir()

	names := b.types // 用户显式指定的

	if len(names) == 0 {
		// 自动扫描
		absDir, _ := filepath.Abs(srcDir)
		scanResult, err := gen.ScanDir(absDir)
		if err != nil || scanResult == nil || len(scanResult.Structs) == 0 {
			return
		}
		for _, s := range scanResult.Structs {
			names = append(names, s.Name)
		}
	}

	for _, name := range names {
		ts, err := gen.ParseStructFromFile(srcDir, name)
		if err != nil || ts == nil {
			continue
		}
		g := gen.NewGenerator(b.config)
		g.Generate(ts)
	}
}
