# go-mongodb
类型安全的 Go MongoDB 辅助层（微型 ORM）

## 为什么需要它

在 Go 语言中使用官方 `go.mongodb.org/mongo-driver` 时，开发者经常需要手写大量的 `bson.M{"fieldName": bson.M{"$gte": value}}` 这样的魔法字符串。

这会带来三个明显的问题：

- **容易拼错**：字段名是字符串，写错了编译器不会报错，只能在运行时暴露。
- **失去类型安全**：值的类型没有约束，传错类型同样只能运行时发现。
- **IDE 无法自动补全**：字段名、操作符全靠记忆，开发体验差。

`go-mongodb` 应运而生——它是一个**轻量级、类型安全的 MongoDB 辅助层（微型 ORM）**，通过 AST 静态分析自动生成 Field 代码，让你用 Go 代码本身来描述和构建所有 MongoDB 操作。

## 核心理念

- **零魔法字符串**：通过 AST 静态分析，将结构体字段映射为类型安全的查询对象，不依赖反射。
- **编译期报错**：字段名拼写错误、类型不匹配，在编译阶段直接暴露，而不是在运行时引发 bug。
- **极致的 IDE 支持**：享受完整的自动补全体验，大幅提升开发效率。
- **轻量且灵活**：不屏蔽 MongoDB 的原生特性，底层依然对接官方驱动，保留最大的灵活性。

## 安装

```bash
go get github.com/xpwu/go-mongodb
go install github.com/xpwu/go-mongodb/cmd/gomongodbgen@latest
```

确保 `$GOBIN`（默认 `$HOME/go/bin`）在 `PATH` 中，然后验证：

```bash
gomongodbgen -h
```

## 快速开始

### 1. 定义你的业务结构体

```go
package userinfo

type UserInfo struct {
    ID     string  `bson:"_id"`
    Name   string  `bson:"name"`
    Age    int     `bson:"age"`
    Amount float64 `bson:"amount"`
}
```

### 2. 添加 go:generate 注释

在结构体**紧邻的上方**添加 `//go:generate` 注释（中间可以有其他注释或空行）：

```go
//go:generate gomongodbgen

type UserInfo struct {
    ID     string  `bson:"_id"`
    Name   string  `bson:"name"`
    Age    int     `bson:"age"`
    Amount float64 `bson:"amount"`
}
```

代码生成器会**自动找到紧邻的 struct 定义**，无需指定类型名。每个需要生成 Field 的结构体都需要单独一行 `//go:generate gomongodbgen`。

### 3. 运行代码生成

```bash
go generate ./...
```

运行完毕后，同包目录下会自动生成 `zUserInfoField.go`（以及子包对应的生成文件）。

> **建议**：生成的代码也需要加入 git 等代码仓库中。

### 4. 开始类型安全的查询

```go
package main

import (
    "context"
    "github.com/xpwu/go-mongodb/client"
    "github.com/xpwu/go-mongodb/filter"
    "github.com/xpwu/go-mongodb/updater"
    "zdemo/userinfo"
)

func main() {
    cfg := client.Config{URI: "mongodb://localhost:27017/xxxx"}
    cli := client.MustGet(cfg)
    coll := cli.Database("market").Collection("userinfo")

    // 类型安全过滤：等同于 bson.M{"age": bson.M{"$gte": 18}, "name": "Alice"}
    f := filter.And(
        userinfo.UserInfoDoc.AgeF().Gte(18),
        userinfo.UserInfoDoc.NameF().Eq("Alice"),
    )

    var result userinfo.UserInfo
    _ = coll.FindOne(context.TODO(), f.ToBson()).Decode(&result)

    // 类型安全更新
    u := updater.Batch(
        userinfo.UserInfoDoc.AgeF().Set(19),
        userinfo.UserInfoDoc.AmountF().Inc(1),
    )
    _, _ = coll.UpdateOne(context.TODO(), f.ToBson(), u.ToBson())
}
```

---

## 核心特性详解

### 1. AST 静态分析

代码生成器基于 Go AST（抽象语法树）解析源码，不依赖反射、不依赖运行时类型信息。

优点：
- **编译期完成**：生成阶段就能发现类型错误
- **完整支持三方库**：`bson.ObjectID`、`bson.Decimal128` 等外部包类型正确解析
- **类型穿透**：`type A = B`、`type A B`、`type A []B` 都能正确追踪到底层类型
- **零运行时开销**：生成的代码就是普通的 Go 类型定义和泛型实例化

### 2. 类型穿透规则

| 定义 | 生成行为 |
|------|---------|
| `type A struct{...}` | 生成 `zAField.go`，Field 类型引用 struct 的字段 |
| `type A B; type B struct{...}` | A 穿透到 B，生成 `LikeBField[A]` |
| `type A = B; type B struct{...}` | A 直接复用 B 的 Field，不生成新文件 |
| `type A []B` | 展开为 `ArrayField[B, FieldOfB]`，不生成新文件 |
| `type A interface{...}` | 生成 `BaseStructField[A]` |

### 3. 自动代码生成与第三方包支持

- **同包生成**：生成的 `zxxxField.go` 文件直接存放在原结构体所在的包内，调用极其方便，无需额外导入。
- **外部类型隔离**：如果结构体定义在第三方包（如 `elsetype.ThirdParty`），生成器会自动创建带有哈希后缀的独立包（如 `elsetype_eLWi9M5n`），完美解决命名冲突和循环依赖问题。

### 4. 强大的查询构造器（filter 包）

告别魔法字符串，支持所有常用的 MongoDB 比较和逻辑操作符：

| 类型 | 支持的操作 |
|------|-----------|
| **比较** | `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte`, `In`, `Nin` |
| **逻辑** | `And`, `Or`, `Nor`, `Not` |
| **数组/元素** | `ElemMatch`, `All`, `Size` |

```go
f := filter.And(
    userinfo.UserInfoDoc.AgeF().Gte(18),
    userinfo.UserInfoDoc.NameF().Eq("Alice"),
)
```

### 5. 灵活的更新构造器（updater 包）

| 类型 | 支持的操作 |
|------|-----------|
| **字段更新** | `$set`, `$unset` |
| **计算更新** | `$inc`, `$mul`, `$min`, `$max` |
| **数组操作** | `$push`, `$pull`, `$addToSet` |
| **数组修饰符** | `Filter`, `Position`, `Sort`, `Slice` 等高级修饰符 |

```go
u := updater.Batch(
    userinfo.UserInfoDoc.AgeF().Set(19),
    userinfo.UserInfoDoc.AmountF().Inc(100),
)
```

### 6. 客户端连接池缓存（client 包）

- 内置基于 `sync.Map` 的客户端缓存机制 `GetFromCache` / `MustGet`。
- **相同的 `Config` 值**（而非指针地址）会复用同一个 `*mongo.Client` 实例，避免频繁创建连接池。
- 支持 `xopt.Option` 进行初始化配置，首次创建后忽略后续不一致的 Option，保证运行时行为稳定。

```go
cfg := client.Config{URI: "mongodb://localhost:27017/xxxx"}
cli := client.MustGet(cfg) // 相同 Config 值复用同一个连接池
```

### 7. 索引与地理信息支持（index / geo 包）

- **索引构建器**：类型安全的索引定义方式，方便在代码中统一管理索引。
- **Geo 查询**：内置 `$near`、`$geoWithin`、`$geoIntersects` 等常用地理位置查询构造器。

---

## 高级用法

### go:generate 参数详解

```go
//go:generate gomongodbgen -out-dir ./zgen -preserve-field
```

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-out-dir` | 生成文件输出目录 | 结构体所在包的目录下 |
| `-preserve-field` | 无 bson tag 时保留 Go 原始字段名（不转小写） | 关闭（默认转小写） |

> 不需要 `-type` 参数。生成器自动找到紧邻 `//go:generate` 注释下方的 struct 定义。

### 集中管理：cli.Run

当项目中有多个结构体需要生成时，逐个写 `//go:generate` 难以统一参数。使用 `cli.Run` 可以在一个地方集中管理所有类型和参数：

// mongodb_gen.go（放在你的业务包中）
```go

//go:build ignore

package userinfo

import (
    "github.com/xpwu/go-mongodb/gen/cli"
)

func init() {
    cli.Run(
        cli.NewBuildConfig().
            SetOutDir("./zgen").
            AddType("UserInfo").
            AddType("Order").
            AddType("Product").
            AddMap("github.com/xpwu/go-mongodb/fields.ObjectID", "fields.ObjectIDField").
            AddMap("time.Time", "fields.TimeField").
            WithPreserveField(),
    )
}
```

运行：

```bash
go generate ./mongodb_gen.go
# 或直接运行测试触发
go test ./mongodb_gen.go
```

`cli.Run` 通过调用栈自动定位源文件所在目录，无需手动指定包路径。

### 自定义类型

#### 编解码自定义类型

有两种方式让 MongoDB 正确编解码你的自定义类型：

- **通过 Registry 注册**：使用 `client.GetLowerFieldRegistry()` 或 `client.GetPreserveFieldRegistry()` 获取内置的 `*bson.Registry`，通过官方 driver 的注册接口添加自定义的 `ValueEncoder` / `ValueDecoder`，最后通过 `xopt.WithRegistry()` 设置到 client 中。
- **实现 Marshaler 接口**：在自定义类型上直接实现 `bson.Marshaler` 和 `bson.Unmarshaler` 接口，自行管理编解码逻辑，无需任何额外注册。如果同时使用了两种方式，接口方式的优先级更高。

#### 生成自定义字段 Field

如果你需要为自定义类型生成特定的 `Field` 类型及其查询/更新方法（例如自定义的时间范围类型、枚举类型），则需要编写对应的 `Field` 结构体及构造函数，然后在 `StructFieldBuilder` 中通过 `RegisterType` 注册该 Field。注册后，生成的代码中对应字段就会使用你定义的 `Field` 类型。

### 代码生成与运行时的一致性

`xopt.Option`（包括 `WithPreserveField`）**同时影响代码生成阶段和运行时 MongoDB 客户端编解码阶段**。两者必须使用相同的 Option 配置，否则字段名映射不一致，导致查询或更新失败。

```go
// 代码生成阶段（cli.Run 或 go:generate）
cli.Run(
    cli.NewBuildConfig().
        AddType("UserInfo").
        WithPreserveField(),  // ← 生成代码时保留原名
)

// 运行时创建 MongoDB Client
opt := xopt.WithPreserveField() // ← 运行时也必须保留原名，否则对不上
cli := client.MustGet(cfg, opt)
```

`WithPreserveField` 的语义：
- **没有 bson tag 的情况下**，保留 Go 结构体字段的原始名称（不做小写转换）
- MongoDB Go Driver 官方默认行为是将字段名转为小写
- 如果生成阶段用了 `WithPreserveField` 但运行时没用（或反过来），字段名会不匹配

> ⚠️ **规则：生成阶段和运行时阶段的 `xopt.Option` 必须完全一致。**

---

## License

[MIT](LICENSE)
