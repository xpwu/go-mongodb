package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xpwu/go-mongodb/gen"
	"github.com/xpwu/go-mongodb/xopt"
)

// $GOMOD 是 go.mod 所在目录的占位符。
// 用户可以在 OutDir / -out-dir 里写 "$GOMOD/zgen"，
// 表示"项目根目录下的 zgen 子目录"。
const gomodPlaceholder = "$GOMOD"

// BuildConfig 是代码生成器的配置构造器。
// 用户通过链式调用设置参数，最后调用 Run() 执行生成。
type BuildConfig struct {
	config    *gen.Config
	outDirRaw string // 用户传入的原始路径字符串（可能是 $GOMOD/... 或 ./...）
	err       error  // 记录初始化阶段的错误，Run() 时统一处理
}

// NewBuildConfig 创建一个新的 BuildConfig。
// 可选传入 xopt.Option 来配置生成行为，这些 Option 必须与
// client 代码中使用的 xopt.Option 保持一致。
//
// Example:
//
//	cli.NewBuildConfig(
//	    xopt.WithPreserveField(true),
//	).OutDir("$GOMOD/zgen").Run()
func NewBuildConfig(opts ...xopt.Option) *BuildConfig {
	c := gen.NewConfig()

	applied := xopt.GetDefaultOpts()
	for _, o := range opts {
		o(applied)
	}
	c.PreserveField = applied.PreserveField
	if applied.BsonOpts != nil {
		c.UseJSONTags = applied.BsonOpts.UseJSONStructTags
	}

	return &BuildConfig{
		config:    c,
		outDirRaw: ".", // 默认当前目录
	}
}

// OutDir 设置生成文件的输出目录。
//
// 路径支持三种写法：
//   - 相对路径:  "./zgen" → 相对于书写此代码的 .go 文件所在目录
//   - $GOMOD 路径: "$GOMOD/zgen" → 相对于 go.mod 所在目录
//   - 磁盘绝对路径: 不支持，会报错退出
//
// 示例：
//
//	cli.NewBuildConfig().OutDir("./zgen")        // → mongodb_gen.go 旁边的 zgen/
//	cli.NewBuildConfig().OutDir("$GOMOD/zgen")  // → 项目根目录下的 zgen/
func (b *BuildConfig) OutDir(dir string) *BuildConfig {
	b.outDirRaw = dir
	return b
}

// TargetPkg 设置生成文件的 target package 路径。
func (b *BuildConfig) TargetPkg(pkg string) *BuildConfig {
	b.config.Pkg = pkg
	return b
}

// AddMap 添加自定义类型映射。
//
// typeIdent 是源类型的完全限定标识符：
//   - 内置类型:      "int"
//   - 同包类型:     "MyType"
//   - 外部包类型: "github.com/foo/bar.MyType"
//
// fieldType 是目标字段类型的完全限定标识符。
// newFunc 是构造函数名称。
//
// equalAble: true 表示 fieldType 实现了 github.com/xpwu/filter.ComparableFilter。
//
// 注意：不支持泛型类型。如果传入泛型，会记录错误，
// 在 Run() 执行时统一打印并退出（不会 panic，也不会打印堆栈）。
func (b *BuildConfig) AddMap(typeIdent, fieldType, newFunc string, equalAble bool) *BuildConfig {
	if b.err != nil {
		return b
	}
	for _, p := range []string{typeIdent, fieldType, newFunc} {
		if strings.Contains(p, "[") {
			b.err = fmt.Errorf("AddMap does not support generic types or functions: %s", p)
			return b
		}
	}
	b.config.AddMap(typeIdent, fieldType, newFunc, equalAble)
	return b
}

func toString(rt ReflectType) string {
	if rt.PkgPath() == "" {
		return rt.Name()
	}

	return fmt.Sprintf("%s.%s", rt.PkgPath(), rt.Name())
}

func (b *BuildConfig) AddMapByInfo(typeInfo TypeInfo) *BuildConfig {
	if typeInfo.Err != nil {
		b.err = typeInfo.Err
		return b
	}
	return b.AddMap(toString(typeInfo.T), toString(typeInfo.Field), toString(typeInfo.NewField), typeInfo.EqualAble)
}

// RunFromArgs 解析 os.Args 中的命令行参数，叠加到 BuildConfig 上，然后执行生成。
//
// 参数优先级：命令行 > API 设置。
// 每次调用独立，不污染全局状态。
//
// 用户在自己写的 main 函数中调用此方法：
//
//	func main() {
//	    cli.RunFromArgs(
//	        cli.NewBuildConfig(xopt.WithPreserveField(true)).
//	            OutDir("$GOMOD/zgen"),
//	    )
//	}
//
// 然后在 struct 上方写：
//
//	//go:generate go run ./path/to/mongodb_gen.go
func RunFromArgs(b *BuildConfig) {
	// 使用 ContinueOnError 而非 ExitOnError，
	// 这样在测试中可以捕获解析错误，而不是 os.Exit。
	fs := flag.NewFlagSet("gomongodbgen", flag.ContinueOnError)

	var mapFlags stringSlice
	fs.Var(&mapFlags, "add-map",
		"custom type mapping: Type,FieldType,NewFunc,EqualAble (can be repeated)\n\n"+
			"Type:       fully qualified source type identifier\n"+
			"            e.g. \"int\" or \"github.com/foo/bar.MyType\"\n"+
			"FieldType:  fully qualified target field type\n"+
			"            e.g. \"github.com/foo/fields.IntField\"\n"+
			"NewFunc:    fully qualified constructor function\n"+
			"            e.g. \"github.com/foo/fields.NewIntField\"\n"+
			"EqualAble:  \"true\" or \"false\"\n"+
			"            true  -> FieldType implements/embeds github.com/xpwu/filter.ComparableFilter\n"+
			"            false -> otherwise\n\n"+
			"Example:\n"+
			"  -add-map=github.com/foo/bar.GPS,github.com/foo/fields.FloatField,github.com/foo/fields.NewFloatField,false\n\n"+
			"NOTE: Generic types and generic functions are NOT supported.")

	dir := fs.String("out-dir", b.outDirRaw, "output directory (relative to source file, or use $GOMOD/zgen)")
	pkg := fs.String("target-pkg", b.config.Pkg, "target package path for output (empty = same as source)")

	useJSONTags := fs.Bool("xopt.with-bson-options-use-json-tags", b.config.UseJSONTags,
		"equivalent to xopt.WithBsonOptions(&mongo/options.BSONOptions{UseJSONStructTags: true}).\n"+
			"MUST match the xopt.Options used in your go-mongodb/client code.")
	preserveField := fs.Bool("xopt.with-preserve-field", b.config.PreserveField,
		"equivalent to xopt.WithPreserveField().\n"+
			"MUST match the xopt.Options used in your go-mongodb/client code.")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "parse args error: %v\n", err)
		os.Exit(1)
	}

	// 命令行覆盖 API 设置
	b.outDirRaw = *dir
	b.config.Pkg = *pkg
	b.config.UseJSONTags = *useJSONTags
	b.config.PreserveField = *preserveField

	// -add-map 追加（key 相同覆盖）
	for _, raw := range mapFlags {
		parts := strings.SplitN(raw, ",", 4)
		if len(parts) != 4 {
			fmt.Fprintf(os.Stderr, "-add-map format: Type,FieldType,NewFunc,EqualAble\n  got: %s\n", raw)
			os.Exit(1)
		}
		for _, p := range parts[:3] {
			if strings.Contains(p, "[") {
				fmt.Fprintf(os.Stderr, "-add-map does not support generic types or functions: %s\n", p)
				os.Exit(1)
			}
		}
		equalAble, err := strconv.ParseBool(parts[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "-add-map EqualAble must be \"true\" or \"false\", got %q\n", parts[3])
			os.Exit(1)
		}
		// 使用 b.config.AddMap 直接添加（已通过校验）
		b.config.AddMap(parts[0], parts[1], parts[2], equalAble)
	}

	b.Run()
}

// Run 执行代码生成。
// 自动扫描当前源文件目录中 //go:generate 下方的 struct 并生成 Field 文件。
func (b *BuildConfig) Run() {
	// 检查初始化阶段是否有错误（如 AddMap 传入泛型）
	if b.err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", b.err)
		os.Exit(1)
	}

	srcDir := determineSrcDir()

	// 解析输出目录
	outDir, err := resolveOutDir(b.outDirRaw, srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "out-dir error: %v\n", err)
		os.Exit(1)
	}
	b.config.Dir = outDir

	// 自动扫描：找 //go:generate 下方的 struct
	absDir, _ := filepath.Abs(srcDir)
	scanResult, err := gen.ScanDir(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}
	if scanResult == nil || len(scanResult.Structs) == 0 {
		if goFile := os.Getenv("GOFILE"); goFile != "" {
			absGoFile := goFile
			if !filepath.IsAbs(goFile) {
				absGoFile = filepath.Join(srcDir, goFile)
			}
			fmt.Printf("no struct found after //go:generate in %s\n", absGoFile)
		} else {
			fmt.Printf("no struct found after //go:generate in %s\n", srcDir)
		}
		return
	}

	for _, s := range scanResult.Structs {
		ts, err := gen.ParseStructFromFile(srcDir, s.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s error: %v\n", s.Name, err)
			os.Exit(1)
		}
		if ts == nil {
			fmt.Fprintf(os.Stderr, "struct %s not found in %s\n", s.Name, srcDir)
			os.Exit(1)
		}
		g := gen.NewGenerator(b.config)
		g.Generate(ts)
		fmt.Printf("generated: %s → %s\n", s.Name, filepath.Join(outDir, "z"+s.Name+"Field.go"))
	}
}

// ─── 内部函数 ────────────────────────────────────────────────

// determineSrcDir 确定源文件所在目录。
//
// 优先级：
//  1. GOFILE 环境变量（go generate 设置时存在）
//  2. runtime.Caller 找到调用者的 .go 文件
//  3. os.Getwd() 兜底
func determineSrcDir() string {
	// 优先级 1：GOFILE 环境变量
	if goFile := os.Getenv("GOFILE"); goFile != "" {
		dir := filepath.Dir(goFile)
		if filepath.IsAbs(dir) {
			return dir
		}
		abs, err := filepath.Abs(dir)
		if err == nil {
			return abs
		}
	}

	// 优先级 2：runtime.Caller 找到用户的 .go 文件
	// 跳过 runtime 内部帧和 go-mongodb 项目内部的调用
	for i := 1; i < 20; i++ {
		_, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// 跳过标准库
		if strings.Contains(file, "runtime/") {
			continue
		}
		// 跳过 go-mongodb 项目内部的调用（gen 包、cli 包等）
		// 但不跳过用户自己的 cmd/ 目录
		if strings.Contains(file, "github.com/xpwu/go-mongodb/") {
			continue
		}
		return filepath.Dir(file)
	}

	// 优先级 3：当前工作目录兜底
	dir, _ := os.Getwd()
	return dir
}

// resolveOutDir 解析输出目录路径。
//
// 规则：
//   - 磁盘绝对路径（如 /home/user/...）→ 报错，不支持
//   - $GOMOD/... → 替换为 go.mod 所在目录（只允许出现在路径开头）
//   - $gomod（小写）→ 报错，提示是否想写 $GOMOD
//   - ./... 或普通相对路径 → 基于 anchorDir 解析
func resolveOutDir(rawPath, anchorDir string) (string, error) {
	// 检查是否为磁盘绝对路径
	if filepath.IsAbs(rawPath) {
		return "", fmt.Errorf("absolute disk path %q is not allowed; use relative path or $GOMOD/...", rawPath)
	}
	// Windows 绝对路径（如 C:\...）
	if len(rawPath) >= 3 && rawPath[1] == ':' && (rawPath[2] == '/' || rawPath[2] == '\\') {
		return "", fmt.Errorf("absolute disk path %q is not allowed; use relative path or $GOMOD/...", rawPath)
	}

	// 检查用户是否手误写了 $gomod（大小写敏感），提前拦截
	if strings.Contains(rawPath, "$gomod") {
		return "", fmt.Errorf("$gomod is not a valid placeholder, did you mean $GOMOD?")
	}

	// 处理 $GOMOD 占位符：只允许出现在路径开头，找不到 go.mod 同样报错
	if strings.Contains(rawPath, gomodPlaceholder) {
		if !strings.HasPrefix(rawPath, gomodPlaceholder+"/") && rawPath != gomodPlaceholder {
			return "", fmt.Errorf("$GOMOD must appear at the beginning of the path, got %q", rawPath)
		}
		modDir := gen.FindGoModDir(anchorDir)
		if modDir == "" {
			return "", fmt.Errorf("$GOMOD referenced but go.mod not found (searched from %s)", anchorDir)
		}
		resolved := strings.ReplaceAll(rawPath, gomodPlaceholder, modDir)
		return filepath.Clean(resolved), nil
	}

	// 普通相对路径：基于 anchorDir 解析
	return filepath.Clean(filepath.Join(anchorDir, rawPath)), nil
}

// stringSlice 实现 flag.Value 接口，支持重复 flag。
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
