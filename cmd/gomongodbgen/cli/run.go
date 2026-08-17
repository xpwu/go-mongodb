package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xpwu/go-mongodb/gen"
	"github.com/xpwu/go-mongodb/xopt"
)

type BuildConfig struct {
	config *gen.Config
	types  []string
}

// NewBuildConfig receives xopt.Option to configure generation behavior.
// These Options MUST be consistent with the xopt.Option used in your client code.
//
// Example:
//
//	  cli.NewBuildConfig(
//	    xopt.WithPreserveField(true),
//	  ).OutDir(".").Type("Order").Run()
//
// NewBuildConfig 接收 xopt.Option 来配置生成行为。
// 这些 Option 必须与 client 代码中使用的 xopt.Option 保持一致。
//
// 示例：
//
//	  cli.NewBuildConfig(
//	    xopt.WithPreserveField(true),
//	  ).OutDir(".").Type("Order").Run()
func NewBuildConfig(opts ...xopt.Option) *BuildConfig {
	c := gen.NewConfig()

	// apply xopt.Option → 翻译到 gen.Config
	applied := xopt.GetDefaultOpts()
	for _, o := range opts {
		o(applied)
	}
	c.PreserveField = applied.PreserveField
	c.IgnoreTagErr = applied.IgnoreTagErr
	if applied.BsonOpts != nil {
		c.UseJSONTags = applied.BsonOpts.UseJSONStructTags
	}

	return &BuildConfig{
		config: c,
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
	// 校验：不支持泛型（通过 [] 判断）
	for _, p := range []string{typeIdent, fieldType, newFunc} {
		if strings.Contains(p, "[") {
			fmt.Fprintf(os.Stderr, "AddMap does not support generic types or functions: %s\n", p)
			os.Exit(1)
		}
	}

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
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s error: %v\n", name, err)
			os.Exit(1)
		}
		if ts == nil {
			fmt.Fprintf(os.Stderr, "struct %s not found in %s\n", name, srcDir)
			os.Exit(1)
		}
		g := gen.NewGenerator(b.config)
		g.Generate(ts)
	}
}
