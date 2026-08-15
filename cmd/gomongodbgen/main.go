package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xpwu/go-mongodb/gen"
)

func main() {
	var (
		typeFlag      = flag.String("type", "", "comma-separated struct type names")
		mapFlag       = flag.String("map", "", "type mapping: Type:FieldType:NewFunc")
		mapExtFlag    = flag.String("map-ext", "", "external: pkg.Type:FieldType:NewFunc")
		preserveField = flag.Bool("preserve-field", false, "preserve original field names")
		useJSONTags   = flag.Bool("use-json-tags", false, "use json tags")
		ignoreTagErr  = flag.Bool("ignore-tag-err", false, "ignore tag errors")
		dirFlag       = flag.String("dir", ".", "output directory")
		pkgFlag       = flag.String("pkg", "", "target package path")
	)

	flag.Parse()

	config := gen.NewConfig()
	config.Dir = *dirFlag
	config.Pkg = *pkgFlag
	config.PreserveField = *preserveField
	config.UseJSONTags = *useJSONTags
	config.IgnoreTagErr = *ignoreTagErr

	// 解析 -type
	if *typeFlag != "" {
		config.SetTypes(strings.Split(*typeFlag, ","))
	}

	// 解析 -map
	if *mapFlag != "" {
		for _, m := range strings.Split(*mapFlag, ",") {
			parts := strings.SplitN(m, ":", 3)
			if len(parts) == 3 {
				config.AddMap(parts[0], parts[1], parts[2])
			}
		}
	}

	// 解析 -map-ext
	if *mapExtFlag != "" {
		for _, m := range strings.Split(*mapExtFlag, ",") {
			parts := strings.SplitN(m, ":", 3)
			if len(parts) == 3 {
				pkgAndType := strings.SplitN(parts[0], ".", 2)
				if len(pkgAndType) == 2 {
					config.AddMapExt(pkgAndType[0], pkgAndType[1], parts[1], parts[2])
				}
			}
		}
	}

	// 如果没指定 -type，从 GOFILE 扫描 //go:generate 注释
	if len(config.Types) == 0 {
		scanResult, err := gen.ScanDir(*dirFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
			os.Exit(1)
		}
		for _, s := range scanResult.Structs {
			config.AddType(s.Name)
		}
	}

	if len(config.Types) == 0 {
		fmt.Fprintln(os.Stderr, "no struct to generate")
		os.Exit(1)
	}

	g := gen.NewGenerator(config)
	for _, typeName := range config.Types {
		ts, err := gen.ParseStructFromFile(*dirFlag, typeName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			os.Exit(1)
		}
		if ts == nil {
			fmt.Fprintf(os.Stderr, "struct %s not found\n", typeName)
			os.Exit(1)
		}
		g.Generate(ts)
	}

	fmt.Println("generation complete")
}
