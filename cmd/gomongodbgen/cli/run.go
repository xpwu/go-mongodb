package cli

import (
	"fmt"

	"github.com/xpwu/go-mongodb/gen"
)

var globalConfig = gen.NewConfig()

func Type(name string)         { globalConfig.AddType(name) }
func Map(t, f, n string)       { globalConfig.AddMap(t, f, n) }
func MapExt(p, t, f, n string) { globalConfig.AddMapExt(p, t, f, n) }
func PreserveField()           { globalConfig.PreserveField = true }
func UseJSONTags()             { globalConfig.UseJSONTags = true }
func IgnoreTagErr()            { globalConfig.IgnoreTagErr = true }

func Run(options ...func()) {
	for _, opt := range options {
		opt()
	}
	Main()
}

func Main() {
	pkg, err := gen.InferPackagePath(".")
	if err != nil {
		panic(err)
	}
	globalConfig.Pkg = pkg

	g := gen.NewGenerator(globalConfig)
	for _, typeName := range globalConfig.Types {
		ts, err := gen.ParseStructFromFile(".", typeName)
		if err != nil {
			panic(err)
		}
		if ts == nil {
			panic(fmt.Errorf("struct %s not found", typeName))
		}
		g.Generate(ts)
	}
}
