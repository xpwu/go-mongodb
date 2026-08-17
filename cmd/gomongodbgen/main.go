package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xpwu/go-mongodb/gen"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var typeNames stringSlice
	flag.Var(&typeNames, "type", "struct name to generate (can be repeated, auto-scan if empty)")
	dir := flag.String("out-dir", ".", "output directory for generated files")
	pkg := flag.String("target-pkg", "", "target package path for output (empty = same as source)")
	useJSONTags := flag.Bool("use-json-tags", false, "use json tag as bson tag fallback")

	preserveField := flag.Bool("preserve-field", false,
		"use field name as bson tag when tag is empty (implies IgnoreTagErr=false)")
	preserveFieldIgnoreTagErr := flag.Bool("preserve-field-ignore-tag-err", false,
		"use field name as bson tag when tag is empty, and ignore unsupported tag errors (implies IgnoreTagErr=true)")

	var mapFlags stringSlice
	flag.Var(&mapFlags, "add-map",
		`custom type mapping: Type,FieldType,NewFunc,EqualAble (can be repeated)

Type:       fully qualified source type identifier
            e.g. "int" or "github.com/foo/bar.MyType"
FieldType:  fully qualified target field type
            e.g. "github.com/foo/fields.IntField"
NewFunc:    fully qualified constructor function
            e.g. "github.com/foo/fields.NewIntField"
EqualAble:  "true" or "false"
            true  -> FieldType implements/embeds github.com/xpwu/filter/ComparableFilter
            false -> otherwise

Example:
  -add-map=github.com/foo/bar.GPS,github.com/foo/fields.FloatField,github.com/foo/fields.NewFloatField,false
  -add-map=github.com/foo/bar.Address,github.com/foo/fields.StringField,github.com/foo/fields.NewStringField,true

NOTE: Generic types and generic functions are NOT supported.`)

	flag.Parse()

	config := gen.NewConfig()
	config.Dir = *dir
	config.Pkg = *pkg
	config.UseJSONTags = *useJSONTags

	config.PreserveField = *preserveField || *preserveFieldIgnoreTagErr
	config.IgnoreTagErr = *preserveFieldIgnoreTagErr

	// 解析 -add-map（支持多次）
	for _, raw := range mapFlags {
		parts := strings.SplitN(raw, ",", 4)
		if len(parts) != 4 {
			fmt.Fprintf(os.Stderr, "-add-map format: Type,FieldType,NewFunc,EqualAble\n  got: %s\n", raw)
			os.Exit(1)
		}
		equalAble, err := strconv.ParseBool(parts[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "-add-map EqualAble must be \"true\" or \"false\", got %q\n", parts[3])
			os.Exit(1)
		}
		config.AddMap(parts[0], parts[1], parts[2], equalAble)
	}

	// 确定源文件目录（不是输出目录）
	srcDir := determineSrcDir()

	// 确定要处理的 struct 列表
	var structNames []string

	if len(typeNames) > 0 {
		structNames = append(structNames, typeNames...)
	} else {
		// 自动扫描：用源文件目录 go:generate 注释下方的 struct
		absDir, _ := filepath.Abs(srcDir)
		scanResult, err := gen.ScanDir(absDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
		if scanResult == nil || len(scanResult.Structs) == 0 {
			fmt.Println("no struct found after //go:generate comment, nothing to do")
			return
		}
		for _, s := range scanResult.Structs {
			structNames = append(structNames, s.Name)
		}
	}

	// 逐个生成
	for _, name := range structNames {
		ts, err := gen.ParseStructFromFile(srcDir, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s error: %v\n", name, err)
			os.Exit(1)
		}
		if ts == nil {
			fmt.Fprintf(os.Stderr, "struct %s not found in %s\n", name, srcDir)
			os.Exit(1)
		}

		generator := gen.NewGenerator(config)
		generator.Generate(ts)
		fmt.Printf("generated: %s → z%sField.go\n", name, name)
	}
}

// determineSrcDir 确定源文件所在目录
func determineSrcDir() string {
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
	// fallback: 当前工作目录
	dir, _ := os.Getwd()
	return dir
}
