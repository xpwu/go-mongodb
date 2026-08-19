//go:build ignore

// 示例：使用 cli API 集中管理代码生成
//
// 使用方式：
//  //go:generate go run ./cmd/gomongodbgen/cli/example
//
// 这个文件展示如何在统一入口管理多个 struct 的代码生成。
// 实际使用时，复制此文件到你的项目中，修改配置后运行。

package main

import (
	"github.com/xpwu/go-mongodb/cmd/gomongodbgen/cli"
	"github.com/xpwu/go-mongodb/xopt"
)

func main() {
	// 方式一：相对路径（相对于此 mongodb_gen.go 文件所在目录）
	// 生成物会放到 ./zgen/ 目录下
	cli.RunFromArgs(
		cli.NewBuildConfig(
			xopt.WithPreserveField(true),
		).OutDir("./zgen"),
	)
}
