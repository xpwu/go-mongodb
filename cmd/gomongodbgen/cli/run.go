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
}

func NewBuildConfig() *BuildConfig {
	return &BuildConfig{
		config: gen.NewConfig(),
	}
}

func (b *BuildConfig) Type(name string) *BuildConfig {
	b.config.AddType(name)
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

func (b *BuildConfig) AddMap(key, fieldType, newFunc string) *BuildConfig {
	b.config.AddMap(key, fieldType, newFunc)
	return b
}

func (b *BuildConfig) AddMapExt(pkgPath, typeName, fieldType, newFunc string) *BuildConfig {
	b.config.AddMapExt(pkgPath, typeName, fieldType, newFunc)
	return b
}

// determineSrcDir 确定源文件所在目录
// 通过调用栈找到用户的 mongodb_gen.go 所在目录
func determineSrcDir() string {
	_, file, _, ok := runtime.Caller(2)
	if ok {
		dir := filepath.Dir(file)
		if strings.HasSuffix(dir, "gen/cmd/go-mongodb-gen/cli") {
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

	if len(b.config.Types) == 0 {
		// 自动扫描：用源文件目录
		absDir, _ := filepath.Abs(srcDir)
		scanResult, err := gen.ScanDir(absDir)
		if err != nil || scanResult == nil || len(scanResult.Structs) == 0 {
			return
		}
		for _, s := range scanResult.Structs {
			b.config.AddType(s.Name)
		}
	}

	for _, name := range b.config.Types {
		ts, err := gen.ParseStructFromFile(srcDir, name) // ← 用 srcDir，不是 b.config.Dir
		if err != nil || ts == nil {
			continue
		}
		g := gen.NewGenerator(b.config)
		g.Generate(ts)
	}
}
