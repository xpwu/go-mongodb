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

有两种使用方式，选择其中一种即可：

**方式一：go install（推荐，全局可用）**

```bash
go install github.com/xpwu/go-mongodb/cmd/gomongodbgen@latest
```

确保 `$GOBIN`（默认 `$HOME/go/bin`）在 `PATH` 中，然后验证：

```bash
gomongodbgen -h
```

**方式二：go run（无需安装，直接使用）**

```go
// 在 //go:generate 注释中直接引用远程模块路径
//go:generate go run github.com/xpwu/go-mongodb/cmd/gomongodbgen
```

两种方式功能完全等价，都是执行同一个 CLI 入口。

> **两种 go run 的区别：**
> - `go run github.com/xpwu/...` → 直接运行远程模块（无需安装，下载即用）
> - `go run ./cmd/mongodb_gen.go` → 运行你项目里自己写的 main 文件（集中管理方式，见下文）

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

如果你没有 `go install`，用方式二：

```go
//go:generate go run github.com/xpwu/go-mongodb/cmd/gomongodbgen

type UserInfo struct {
    ID     string  `bson:"_id"`
    Name   string  `bson:"name"`
    Age    int     `bson:"age"`
    Amount float64 `bson:"amount"`
}
```

代码生成器会**自动找到紧邻的 struct 定义**，无需指定类型名。每个需要生成 Field 的结构体都需要单独一行 `//go:generate`。

### 3. 运行代码生成

```bash
go generate ./...
```
或者点击 IDE 在 `//go:generate` 左侧显示的运行按钮（具体是否支持与 IDE 有关）。

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
//go:generate gomongodbgen -out-dir ./zgen -xopt.with-preserve-field
```

| 参数 | 说明 | 对应代码 API | 默认值 |
|------|------|-------------|--------|
| `-out-dir` | 生成文件输出目录（相对路径基于当前 `.go` 文件所在目录） | `cli.OutDir()` | 当前 `.go` 文件所在目录 |
| `-target-pkg` | 生成文件的 `package` 声明名 | `cli.TargetPkg()` | 与源文件同包 |
| `-add-map` | 自定义类型映射，格式：`Type,FieldType,NewFunc,EqualAble`（可重复） | `cli.AddMap()` | 无 |
| `-xopt.with-preserve-field` | 保留原名（不转小写），原始 bson tag 完整保留（含 `omitempty`/`minsize`/`truncate`），本级生效，不传递到嵌套字段 | `xopt.WithPreserveField()` | 关闭 |
| `-xopt.with-bson-options-use-json-tags` | 使用 JSON tag 替代 bson tag | `xopt.WithBsonOptions()` + JSON 配置 | 关闭 |

> 生成器自动扫描 `//go:generate` 注释下方紧邻的 struct 定义。

### 路径规则

所有路径参数（如 `-out-dir`）遵循以下规则：

| 写法 | 含义 | 示例 |
|------|------|------|
| `./zgen` | 相对路径，基于**书写 `//go:generate` 的 `.go` 文件所在目录** | `model/zgen/` |
| `$GOMOD/zgen` | 基于项目根目录（go.mod 所在位置）的路径 | `project-root/zgen/` |
| 磁盘绝对路径（如 `/home/user/zgen`） | **报错，不支持** | — |

**核心原则：哪里写的路径，就相对于哪里。**

- 在 `//go:generate` 注释里写的 `./zgen` → 相对于该 `.go` 文件目录
- 在 `cli` API 里写的 `OutDir("./zgen")` → 相对于 `mongodb_gen.go` 文件目录
- 使用 `$GOMOD/zgen` → 相对于 go.mod 所在目录（找不到 go.mod 直接报错退出）

### 集中管理：cli API

当项目中有多个结构体需要生成时，逐个写 `//go:generate` 难以统一参数。使用 `cli` 包可以在一个地方集中管理：

**mongodb_gen.go（放在你的项目中任意位置）：**

```go
//go:build ignore

package main

import (
    "github.com/xpwu/go-mongodb/cmd/gomongodbgen/cli"
    "github.com/xpwu/go-mongodb/xopt"
)

func main() {
    cli.RunFromArgs(
        cli.NewBuildConfig(
            xopt.WithPreserveField(),
        ).OutDir("$GOMOD/zgen").
            AddMap("github.com/xpwu/go-mongodb/fields.ObjectID",
                "github.com/xpwu/go-mongodb/fields.ObjectIDField",
                "github.com/xpwu/go-mongodb/fields.NewObjectIDField",
                false).
            AddMap("time.Time",
                "github.com/xpwu/go-mongodb/fields.TimeField",
                "github.com/xpwu/go-mongodb/fields.NewTimeField",
                false),
    )
}
```

**在结构体文件中引用：**

```go
// model/userinfo.go

// 替换 ./cmd/ 为你实际存放 mongodb_gen.go 的目录
//go:generate go run ./cmd/mongodb_gen.go

type UserInfo struct {
    ID   string `bson:"_id"`
    Name string `bson:"name"`
    Age  int    `bson:"age"`
}
```

**运行：**

```bash
go generate ./...
# 或者直接运行
go run ./cmd/mongodb_gen.go
```
也可以点击 IDE 在 `//go:generate` 左侧显示的运行按钮（具体是否支持与 IDE 有关）。

**参数优先级：命令行 > cli API 设置 > 默认值。** 例如：

```bash
# cli 里写了 OutDir("$GOMOD/zgen")，但这次临时改输出位置：
go run ./cmd/mongodb_gen.go -out-dir ./other
```

### 自定义类型

#### 编解码自定义类型

有两种方式让 MongoDB 正确编解码你的自定义类型：

- **通过 Registry 注册**：使用 `client.GetLowerFieldRegistry()` 或 `client.GetPreserveFieldRegistry()` 获取内置的 `*bson.Registry`，通过官方 driver 的注册接口添加自定义的 `ValueEncoder` / `ValueDecoder`，最后通过 `xopt.WithRegistry()` 设置到 client 中。
- **实现 Marshaler 接口**：在自定义类型上直接实现 `bson.Marshaler` 和 `bson.Unmarshaler` 接口，自行管理编解码逻辑，无需任何额外注册。如果同时使用了两种方式，接口方式的优先级更高。

#### 生成自定义字段 Field

如果你需要为自定义类型生成特定的 `Field` 类型及其查询/更新方法（例如自定义的时间范围类型、枚举类型），则需要编写对应的 `Field` 结构体及构造函数，然后使用 `-add-map` 参数或者 `cli.AddMap()` 方法注册该 Field。注册后，生成的代码中对应字段就会使用你定义的 `Field` 类型。

### 代码生成与运行时的一致性

`xopt.Option`（包括 `WithPreserveField`）**同时影响代码生成阶段和运行时 MongoDB 客户端编解码阶段**。两者必须使用相同的 Option 配置，否则字段名映射不一致，导致查询或更新失败。

```go
// 代码生成阶段（cli API 或 go:generate）
cli.RunFromArgs(
    cli.NewBuildConfig(
        xopt.WithPreserveField(), // 与运行时保持一致
    ).OutDir("$GOMOD/zgen"),
)

// 运行时创建 MongoDB Client
cfg := client.Config{URI: "mongodb://localhost:27017/xxxx"}
opt := xopt.WithPreserveField() // ← 必须与生成阶段一致
cli := client.MustGet(cfg, opt)
```

#### WithPreserveField 的 bson tag 说明

使用 `WithPreserveField` 时，原始 bson tag 会被完整保留并透传到生成的代码中。三个属性 `omitempty`、`minsize`、`truncate` **在当前字段上生效，但不会传递到嵌套 struct 字段**（这是 MongoDB Go Driver v2 的限制，无法改变）。

**支持的 tag 写法：**
- `omitempty` — ✅ 本级生效，零值字段在编解码时被省略
- `minsize` — ✅ 本级生效，小整数编码为 int32
- `truncate` — ✅ 本级生效（decode 阶段：BSON double 截断后赋值给整型/float32 字段）
- `inline` — ✅ 支持，内嵌字段会被展开处理
- 重命名（如 `bson:"my_name"`）— ✅ 支持，生成的 Field 使用重命名后的字段名
- `bson:"-"`（skip）— ✅ 支持，该字段会被跳过不生成 Field

示例：

```go
type Profile struct {
	Bio  string `bson:"bio,omitempty"` // ✅ omitempty 本级生效：Bio 为零值时省略
	Rank int    `bson:"rank,minsize"` // ✅ minsize 本级生效：Rank 小值时编码为 int32
}

type User struct {
	ID       string  `bson:"_id"`
	Name     string  `bson:"name"`                // ✅ 支持重命名
	Password string  `bson:"-"`                   // ✅ 支持 skip
	Profile  Profile `bson:"profile,inline"`      // ✅ 支持 inline

	SecretKey string  `bson:"secret,omitempty"`   // ✅ omitempty 本级生效（string 零值省略）
	Count     int     `bson:"count,minsize"`      // ✅ minsize 本级生效（小值编码为 int32）
	Score     float64 `bson:"score,truncate"`     // ✅ truncate 本级生效（double→整型截断，double→float64 不截断）

	Extra Profile `bson:"extra,omitempty"`        // ⚠️ omitempty 只判断 Extra 整体是否零值（Profile{}），不会传递到 Profile 内部的 Bio、Rank 字段
}
```

**说明：**
- `SecretKey`、`Count`、`Score` 是标量字段，`omitempty` / `minsize` / `truncate` 直接在本字段上生效。
- `Extra` 是嵌套 struct 字段，`omitempty` 只判断 `Extra` 整体是否为零值（`Profile{}`），**不会**自动让 `Profile.Bio` 或 `Profile.Rank` 的 tag 也获得 `omitempty` / `minsize`——这就是"不传递到嵌套字段"的含义。

```go
// 运行时也可以通过 BSONOptions 统一配置（效果相同，作用于所有字段）：
bsonOpts := &options.BSONOptions{
	OmitEmpty:              true,
	IntMinSize:             true,
	AllowTruncatingDoubles: false,
}
opt := xopt.WithBsonOptions(bsonOpts)
cli := client.MustGet(cfg, opt)
```

> ⚠️ **规则：生成阶段和运行时阶段的 `xopt.Option` 必须完全一致。**

---

## License

[MIT](LICENSE)
