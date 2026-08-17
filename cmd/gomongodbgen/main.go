package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xpwu/go-mongodb/gen"
)

func main() {
	typeName := flag.String("type", "", "struct name to generate (optional, auto-scan if empty)")
	dir := flag.String("out-dir", ".", "output directory for generated files")
	pkg := flag.String("target-pkg", "", "target package path for output (empty = same as source)")
	useJSONTags := flag.Bool("use-json-tags", false, "use json tag as bson tag fallback")

	preserveField := flag.Bool("preserve-field", false,
		"use field name as bson tag when tag is empty (implies IgnoreTagErr=false)")
	preserveFieldIgnoreTagErr := flag.Bool("preserve-field-ignore-tag-err", false,
		"use field name as bson tag when tag is empty, and ignore unsupported tag errors (implies IgnoreTagErr=true)")

	maps := flag.String("map", "", "type mapping: Key,FieldType,NewFunc")
	mapExts := flag.String("map-ext", "", "extended type mapping: PkgPath,TypeName,FieldType,NewFunc")

	flag.Parse()

	config := gen.NewConfig()
	config.Dir = *dir
	config.Pkg = *pkg
	config.UseJSONTags = *useJSONTags

	config.PreserveField = *preserveField || *preserveFieldIgnoreTagErr
	config.IgnoreTagErr = *preserveFieldIgnoreTagErr

	// 解析 -map / -map-ext
	if *maps != "" {
		parseMap(config, *maps, false)
	}
	if *mapExts != "" {
		parseMap(config, *mapExts, true)
	}

	// 确定源文件目录（不是输出目录）
	srcDir := determineSrcDir()

	// 确定要处理的 struct 列表
	var structNames []string

	if *typeName != "" {
		structNames = append(structNames, *typeName)
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
		ts, err := gen.ParseStructFromFile(srcDir, name) // ← 用 srcDir，不是 *dir
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

func parseMap(config *gen.Config, s string, ext bool) {
	parts := splitComma(s)
	if ext {
		if len(parts) != 4 {
			fmt.Fprintf(os.Stderr, "-map-ext format: PkgPath,TypeName,FieldType,NewFunc\n")
			os.Exit(1)
		}
		config.AddMapExt(parts[0], parts[1], parts[2], parts[3])
	} else {
		if len(parts) != 3 {
			fmt.Fprintf(os.Stderr, "-map format: Key,FieldType,NewFunc\n")
			os.Exit(1)
		}
		config.AddMap(parts[0], parts[1], parts[2])
	}
}

func splitComma(s string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
