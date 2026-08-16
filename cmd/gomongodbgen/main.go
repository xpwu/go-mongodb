package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xpwu/go-mongodb/gen"
)

// 用法：
//   go-mongodb-gen [-type=Name] [-dir=.] [-pkg=output/pkg/path]
//                  [-map=Key,FieldType,NewFunc] [-map-ext=PkgPath,TypeName,FieldType,NewFunc]
//                  [-preserve-field] [-use-json-tags] [-ignore-tag-err]

func main() {
	typeName := flag.String("type", "", "struct name to generate (optional, auto-scan if empty)")
	dir := flag.String("dir", ".", "source directory to scan")
	pkg := flag.String("pkg", "", "target package path for output (empty = same as source)")
	preserveField := flag.Bool("preserve-field", false, "use field name as bson tag when tag is empty")
	useJSONTags := flag.Bool("use-json-tags", false, "use json tag as bson tag fallback")
	ignoreTagErr := flag.Bool("ignore-tag-err", false, "ignore unsupported tag errors")

	// -map 可重复
	maps := flag.String("map", "", "type mapping: Key,FieldType,NewFunc (can be repeated)")
	mapExts := flag.String("map-ext", "", "extended type mapping: PkgPath,TypeName,FieldType,NewFunc (can be repeated)")

	flag.Parse()

	config := gen.NewConfig()
	config.Dir = *dir
	config.Pkg = *pkg
	config.PreserveField = *preserveField
	config.UseJSONTags = *useJSONTags
	config.IgnoreTagErr = *ignoreTagErr

	// 解析 -map（可重复）
	for _, m := range flag.Args() {
		_ = m
	}
	if *maps != "" {
		parseMap(config, *maps, false)
	}
	if *mapExts != "" {
		parseMap(config, *mapExts, true)
	}
	// 支持多次 -map
	for _, arg := range os.Args[1:] {
		if arg == "-map" || arg == "-map-ext" {
			break
		}
	}

	// 确定要处理的 struct 列表
	var structNames []string

	if *typeName != "" {
		// 显式指定
		structNames = append(structNames, *typeName)
	} else {
		// 自动扫描 //go:generate 注释下方的 struct
		absDir, _ := filepath.Abs(*dir)
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
		ts, err := gen.ParseStructFromFile(*dir, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse %s error: %v\n", name, err)
			os.Exit(1)
		}
		if ts == nil {
			fmt.Fprintf(os.Stderr, "struct %s not found in %s\n", name, *dir)
			os.Exit(1)
		}

		generator := gen.NewGenerator(config)
		generator.Generate(ts)
		fmt.Printf("generated: %s → z%sField.go\n", name, name)
	}
}

// parseMap 解析 -map / -map-ext 参数
// format: Key,FieldType,NewFunc 或 PkgPath,TypeName,FieldType,NewFunc
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
