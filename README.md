# go-mongodb
类型安全的 Go MongoDB 辅助层(微型ORM)

## 为什么需要它

在 Go 语言中使用官方 `go.mongodb.org/mongo-driver` 时，开发者经常需要手写大量的 `bson.M{"fieldName": bson.M{"$gte": value}}` 这样的魔法字符串。

这会带来三个明显的问题：

- **容易拼错**：字段名是字符串，写错了编译器不会报错，只能在运行时暴露。
- **失去类型安全**：值的类型没有约束，传错类型同样只能运行时发现。
- **IDE 无法自动补全**：字段名、操作符全靠记忆，开发体验差。

`go-mongodb` 应运而生——它是一个**轻量级、类型安全的 MongoDB 辅助层（微型 ORM）**，通过自动代码生成，让你用 Go 代码本身来描述和构建所有 MongoDB 操作。

## 核心理念

- **零魔法字符串**：通过自动代码生成技术，将结构体字段映射为类型安全的查询对象。
- **编译期报错**：字段名拼写错误、类型不匹配，在编译阶段直接暴露，而不是在运行时引发 bug。
- **极致的 IDE 支持**：享受完整的自动补全体验，大幅提升开发效率。
- **轻量且灵活**：不屏蔽 MongoDB 的原生特性，底层依然对接官方驱动，保留最大的灵活性。

## 安装

`go get github.com/xpwu/go-mongodb`

## 快速开始

1. 定义你的业务结构体   
在你的项目中正常定义带有 bson tag 的 Go 结构体。
```go
package userinfo

type UserInfo struct {
  ID     string  `bson:"_id"`
  Name   string  `bson:"name"`
  Age    int     `bson:"age"`
  Amount float64 `bson:"amount"`
}
```
2. 触发代码生成   
编写 Example 测试函数来触发字段代码的自动生成，userinfo_test.go
```go
package userinfo

import (
  "github.com/xpwu/go-mongodb/fields"
)

// 注意：这是一个 Example 函数，运行测试时会触发代码生成
func ExampleStructFieldBuilder() {
  builder := fields.NewStructFieldBuilder()
  // 为当前包中的 UserInfo 生成字段描述符
  fields.BuildStruct[UserInfo](builder)

  fmt.Println(true)
  // Output:
  // true
}
```
执行生成命令：
在项目根目录下运行测试（或专门的生成命令）：
```bash
go test ./...
```
或者 golang IDE上直接点左侧运行图标
运行完毕后，同包目录下会自动生成 zUserInfoField.go（以及子包对应的生成文件）。   
>***建议*** 生成的代码也需要加入git等代码仓库中。

3. 开始类型安全的查询    
现在你可以利用生成的字段对象，配合 filter 和 updater 包进行查询。
```go
package main

import (
  "context"
  "fmt"
  "github.com/xpwu/go-mongodb/client"
  "github.com/xpwu/go-mongodb/filter"
  "github.com/xpwu/go-mongodb/updater"

  // 引入你自己的业务包
  "zdemo/userinfo"
)

func main() {
  // 1. 获取缓存的 MongoDB 客户端
  cfg := client.Config{URI: "mongodb://localhost:27017/xxxx",}
  cli := client.MustGet(cfg)

  // 2. 获取集合
  coll := cli.Database("market").Collection("userinfo")

  // 3. 使用生成的字段构建类型安全的过滤器
  // 等同于 bson.M{"age": bson.M{"$gte": 18}, "name": "Alice"}
  f := filter.And(
    userinfo.UserInfoDoc.AgeF().Gte(18),
    userinfo.UserInfoDoc.NameF().Eq("Alice"),
  )

  // 4. 执行查询
  var result userinfo.UserInfo
  _ = coll.FindOne(context.TODO(), f.ToBson()).Decode(&result)
  fmt.Println(result)

  // 5. 使用 updater 进行类型安全的更新
  u := updater.Batch(userinfo.UserInfoDoc.AgeF().Set(19), userinfo.UserInfoDoc.AmountF().Inc(1))
  _, _ = coll.UpdateOne(context.TODO(), f.ToBson(), u.ToBson())
}
```
https://github.com/xpwu/go-mongodb/tree/master/zdemo 有更详细的例子

## 核心特性详解

### 1. 自动代码生成与第三方包支持

- **同包生成**：生成的 `zxxxField.go` 文件直接存放在原结构体所在的包内，调用极其方便，无需额外导入。
- **外部类型隔离**：如果结构体定义在第三方包（如 `elsetype.ThirdParty`），生成器会自动创建带有哈希后缀的独立包（如 `elsetype_eLWi9M5n`），完美解决命名冲突和循环依赖问题。

### 2. 强大的查询构造器（`filter` 包）

告别魔法字符串，支持所有常用的 MongoDB 比较和逻辑操作符：

| 类型 | 支持的操作                                             |
|------|---------------------------------------------------|
| **比较** | `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte`, `In`, `Nin` |
| **逻辑** | `And`, `Or`, `Nor`, `Not`                         |
| **数组/元素** | `ElemMatch`, `All`, `Size`                        |

```go
// 示例：构建类型安全的复合查询

f := filter.And(
  userinfo.UserInfoDoc.AgeF().Gte(18),
  userinfo.UserInfoDoc.NameF().Eq("Alice"),
)
```

### 3. 灵活的更新构造器（`updater` 包）

支持 MongoDB 的各种更新操作符，同样完全类型安全：

| 类型 | 支持的操作 |
|------|-----------|
| **字段更新** | `$set`, `$unset` |
| **计算更新** | `$inc`, `$mul`, `$min`, `$max` |
| **数组操作** | `$push`, `$pull`, `$addToSet` |
| **数组修饰符** | `Filter`, `Position`, `Sort`, `Slice` 等高级修饰符 |

```go
// 示例：批量更新

u := updater.Batch(
  userinfo.UserInfoDoc.AgeF().Set(19),
  userinfo.UserInfoDoc.AmountF().Inc(100),
)

```

### 4. 客户端连接池缓存（`client` 包）

- 内置了基于 `sync.Map` 的客户端缓存机制 `GetFromCache` / `MustGet`。
- **相同的 `Config` 值**（而非指针地址）会复用同一个 `*mongo.Client` 实例，避免频繁创建连接池带来的性能开销。
- 支持 `xopt.Option` 进行初始化配置，且首次创建后忽略后续不一致的 Option，保证运行时行为的稳定性。

```go
cfg := client.Config{URI: "mongodb://localhost:27017/xxxx"}
cli := client.MustGet(cfg) // 相同 Config 值复用同一个连接池
```

### 5. 索引与地理信息支持（`index` / `geo` 包）

- **索引构建器**：提供了类型安全的索引定义方式，方便在代码中统一管理索引，避免手写 `bson.D`。
- **Geo 查询**：内置了常用的地理位置查询构造器，如 `$near`、`$geoWithin`、`$geoIntersects` 等，轻松处理位置数据。

### 6. 灵活的大小写控制

默认情况下，MongoDB 官方驱动会将结构体字段名转为小写作为数据库字段名。你可以通过 `xopt.WithPreserveField()` 保留原始字段名：

```go
opt := xopt.WithPreserveField()
builder := fields.NewStructFieldBuilder(opt)
cli := client.MustGet(cfg, opt) // Builder 和 client 必须使用同一个 option
```

> ⚠️ **注意**：如果使用了 `xopt.Option`，`StructFieldBuilder` 和 `client` 必须使用同一个 option 实例，否则字段名可能不匹配，导致查询或更新失败。


## License

[MIT](LICENSE)