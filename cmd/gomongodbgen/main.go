package main

import (
	"github.com/xpwu/go-mongodb/cmd/gomongodbgen/cli"
)

// gomongodbgen 是 go-mongodb 的代码生成器 CLI 入口。
//
// 使用方式：在 struct 上方写 //go:generate 注释
//
//	//go:generate gomongodbgen
//	type User struct {
//	    ID   string `bson:"_id"`
//	    Name string `bson:"name"`
//	}
//
// 可选参数：
//   - -out-dir <path>   输出目录（相对路径或 $GOMOD/...）
//   - -target-pkg <pkg>  目标包路径
//   - -add-map ...       自定义类型映射（可重复）
//   - -xopt.with-preserve-field
//   - -xopt.with-bson-options-use-json-tags
//
// 详细文档见 README 和 ./cli 包。
func main() {
	cli.RunFromArgs(cli.NewBuildConfig())
}
