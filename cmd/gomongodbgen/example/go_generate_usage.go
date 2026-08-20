//go:build ignore

package example

// 本文件展示 go-mongodb-gen 的各种使用方式。
// 实际使用时，把 struct 定义和 go:generate 注释放到你的业务代码中。

// ─── 方式一：默认使用 ────────────────────────
//
// 在 struct 上方写一行 //go:generate gomongodbgen
// 不需要任何参数，生成器自动找到紧邻的 struct。
//
//go:generate gomongodbgen

type User struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}

// ─── 方式二：指定输出目录 ────────────────────────────────
//
// -out-dir 是相对于当前 .go 文件所在目录的。
// 下面会生成到当前目录的 zgen/ 子目录下。
//
//go:generate gomongodbgen -out-dir ./zgen

type Order struct {
	ID     string  `bson:"_id"`
	UserID string  `bson:"user_id"`
	Total  float64 `bson:"total"`
}

// ─── 方式三：使用 JSON tag ────────────────────────────────
//
// 如果你的项目用 json tag 而不是 bson tag：
//
//go:generate gomongodbgen -out-dir ./zgen -xopt.with-bson-options-use-json-tags

type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ─── 方式四：保留原始字段名 ──────────────────────────────
//
// 默认会转小写（MyField → myField），加 -xopt.with-preserve-field 保留原样。
//
//go:generate gomongodbgen -out-dir ./zgen -xopt.with-preserve-field

type Config struct {
	APIKey     string `bson:"api_key"`
	MaxRetries int    `bson:"max_retries"`
}

// ─── 方式五：自定义类型映射 ──────────────────────────────
//
// 把自定义类型映射到对应的 Field 实现。
//
//go:generate gomongodbgen -out-dir ./zgen \
//  -add-map=github.com/foo/bar.GPS,github.com/foo/fields.FloatField,github.com/foo/fields.NewFloatField,false \
//  -add-map=time.Time,github.com/xpwu/go-mongodb/fields.TimeField,github.com/xpwu/go-mongodb/fields.NewTimeField,false

type Location struct {
	Name string    `bson:"name"`
	Pos  GPS       `bson:"pos"` // 自定义 GPS 类型
	At   time.Time `bson:"at"`  // time.Time
}

// ─── 方式六：集中管理（cli API）──────────────────────────
//
// 在 path/to/mongodb_gen.go 里写：（mongodb_gen.go 可以放你项目下合适的任何路径中）
//
//	func main() {
//	    cli.RunFromArgs(
//	        cli.NewBuildConfig(xopt.WithPreserveField()).
//	            OutDir("$GOMOD/zgen"),
//	    )
//	}
//
// 然后每个 struct 上方只需写：
//
//	//go:generate go run ./path/to/mongodb_gen.go
//
// 所有生成物统一输出到项目根的 zgen/ 目录。
