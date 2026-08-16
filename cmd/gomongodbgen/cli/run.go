package cli

import (
	"os"
	"path/filepath"

	"github.com/xpwu/go-mongodb/gen"
)

// BuildConfig 用户通过 mongodb_gen.go 调用的构建配置
type BuildConfig struct {
	config *gen.Config
}

// NewBuildConfig 创建构建配置
func NewBuildConfig() *BuildConfig {
	return &BuildConfig{
		config: gen.NewConfig(),
	}
}

// Type 添加要生成的类型名
func (b *BuildConfig) Type(name string) *BuildConfig {
	b.config.AddType(name)
	return b
}

// Dir 设置源目录
func (b *BuildConfig) Dir(dir string) *BuildConfig {
	b.config.Dir = dir
	return b
}

// Pkg 设置输出包路径
func (b *BuildConfig) Pkg(pkg string) *BuildConfig {
	b.config.Pkg = pkg
	return b
}

// PreserveField 设置是否保留字段名
func (b *BuildConfig) PreserveField(v bool) *BuildConfig {
	b.config.PreserveField = v
	return b
}

// IgnoreTagErr 设置是否忽略 tag 错误
func (b *BuildConfig) IgnoreTagErr(v bool) *BuildConfig {
	b.config.IgnoreTagErr = v
	return b
}

// Map 添加自定义类型映射
func (b *BuildConfig) Map(key, fieldType, newFunc string) *BuildConfig {
	b.config.AddMap(key, fieldType, newFunc)
	return b
}

// MapExt 添加跨包自定义类型映射
func (b *BuildConfig) MapExt(pkgPath, typeName, fieldType, newFunc string) *BuildConfig {
	b.config.AddMapExt(pkgPath, typeName, fieldType, newFunc)
	return b
}

// Run 执行生成
func (b *BuildConfig) Run() {
	if len(b.config.Types) == 0 {
		// 自动扫描
		absDir, _ := filepath.Abs(b.config.Dir)
		scanResult, err := gen.ScanDir(absDir)
		if err != nil || scanResult == nil || len(scanResult.Structs) == 0 {
			return
		}
		for _, s := range scanResult.Structs {
			b.config.AddType(s.Name)
		}
	}

	for _, name := range b.config.Types {
		ts, err := gen.ParseStructFromFile(b.config.Dir, name)
		if err != nil || ts == nil {
			continue
		}
		g := gen.NewGenerator(b.config)
		g.Generate(ts)
	}
}

// 确保 main 包可以用 go run 执行
func main() {
	// 此文件不应被直接执行，仅为 mongodb_gen.go 提供库
	os.Exit(0)
}
