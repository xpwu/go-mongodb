// Package gen 实现了 go-mongodb-gen 代码生成器的核心逻辑。
//
// # 设计目标
//
// 本包负责从 Go 源码的 AST 或 reflect.Type 中解析类型信息，并为每个 struct
// 及其嵌套类型生成对应的 Field 类型代码。以下是核心设计原则：
//
// ## 1. 保留用户定义的类型名
//
// 用户通过 type 关键字定义的命名类型，在生成代码时保留其名字，不展开为底层类型。
//
//	type GPS float64   → 生成 ComputableField[GPS]  （不是 ComputableField[float64]）
//	type MyInt int     → 生成 IntegerField[MyInt]    （不是 IntegerField[int]）
//
// 理由：用户定义的类型可能有自己的语义和方法，展开会丢失这些信息。
//
// ## 2. 切片类型不生成独立 Field 文件
//
// type A []E 这种切片类型定义，不生成新的 Field 文件，而是展开为对应的 ArrayField。
//
//	type GPSes []GPS  → 生成 ArrayField[GPS, ComputableField[GPS]]
//	                     不生成 zGPSesField.go
//
// 只有真正的 struct 类型才会生成独立的 Field 文件（如 Wx → zWxField.go）。
//
// ## 3. 类型别名声明（type A = C）直接穿透
//
// Go 1.9 引入的类型别名声明语法 type A = C，A 和 C 是完全等价的同一个类型。
// 生成代码时不应该为 A 创建新的 Field，而是直接穿透使用 C。
//
//	type GPSesA = GPSes  → 直接复用 GPSes 的逻辑
//	                        生成 ArrayField[GPS, ComputableField[GPS]]
//	                        不生成 zGPSesAField.go
//
// 实现方式：在 AST 解析阶段检查 typeSpec.Assign != token.NoPos，
// 将目标类型记录在 aliasTargets 映射中。Underlying() 优先查 aliasTargets 实现穿透。
//
// ## 4. 类型定义穿透（type A B）
//
// 对于 type A B（B 是另一个命名类型），Underlying() 查 typeDefTargets 返回下一层，
// isAlias = false。Generator 沿链路追到终点后，根据终点的 Kind 决定如何生成代码。
//
//	type A B; type B struct{}  → A 穿透到 B，B 是 struct，生成 LikeBField[A]
//	type A B; type B int       → A 穿透到 B，B 是 int，生成 IntegerField[A]
//
// ## 5. 复合类型穿透（type A []E）
//
// 对于 type A []E，Underlying() 查 typeElems 返回一个匿名 Slice TypeSource，
// Kind = reflect.Slice，Elem = E 的 TypeSource。Generator 拿到后走 buildSlice 分支。
//
//	type GPSes []GPS  → Underlying() 返回 Kind=Slice, Elem=GPS
//	                    Generator 生成 ArrayField[GPS, ComputableField[GPS]]
//
// ## 6. interface 类型的处理
//
// 对于 type A interface{ Do() } 这种用户定义的 interface，在 interfaceTargets
// 中记录其名字。parseAstTypeWithLoader 遇到标识符时查此表，将 Kind 修正为
// reflect.Interface。
//
// 对于 any / interface{} / bson.any 等，强制将 Kind 设为 reflect.Interface
// 并清空 pkgPath，防止生成错误的类型引用（如 bson.any）。
//
//	any / interface{}  → 生成 BaseStructField[any]（不带包路径前缀）
//	bson.any          → 生成 BaseStructField[any]（不是 bson.BaseStructField[any]）
//
// ## 7. 只有 struct 才生成独立的 Field 文件
//
// 在 build 函数中，根据终点类型的 Kind 分发：
//   - reflect.Struct → buildStruct，生成独立的 zXxxField.go 文件
//   - reflect.Slice / reflect.Array → buildSlice，返回 ArrayField 类型，不生成新文件
//   - reflect.Ptr → buildPtr，返回底层元素的 TypeInfo，不生成新文件
//   - 其他基本类型 / Interface → buildKind，返回对应的泛型 Field，不生成新文件
//
// 只有 struct 类型才会触发文件生成，其他类型只作为字段类型被引用。
//
// # AST 解析核心设计
//
// ## Underlying() 穿透机制
//
// TypeSource 接口提供 Underlying() 方法，返回类型链路的下一层：
//
//	next TypeSource, isAlias bool
//
// AST 层的实现按以下优先级查找：
//
//	1. aliasTargets  → 类型别名（type A = C），isAlias = true
//	2. typeDefTargets → 类型定义（type A B），isAlias = false
//	3. typeElems     → 复合类型（type A []E），返回匿名 Slice TypeSource
//	4. 都不命中      → 返回 nil, false（已是终点）
//
// Generator 只管沿着 Underlying() 追到 next == nil 的终点，然后看终点的 Kind 和 Elem。
// 中间节点的 Kind 不重要，只有终点的 Kind 才有意义。
//
// ## Kind() 的实现
//
// Kind() 始终穿透到终点拿 Kind：
//
//	func (a *astTypeSource) Kind() reflect.Kind {
//	    next, _ := a.Underlying()
//	    if next != nil {
//	        return next.Kind()
//	    }
//	    return a.kind
//	}
//
// 这样无论中间节点是什么，调用方拿到的都是终点类型的 Kind。
//
// ## 缓存策略
//
//   - LoadPackage 使用 loaded map 缓存已加载的包，避免重复解析文件。
//   - typeDefTargets 的解析结果通过 underlyingCache 缓存，避免重复调用
//     parseAstTypeWithLoader 创建对象。
//   - aliasTargets 本身存储的就是解析好的 TypeSource 对象，天然是缓存。
//
// # Generator 核心设计
//
// ## build 函数的穿透循环
//
// Generator 的 build 函数通过 Underlying() 追逐类型链路：
//
//  1. 将入口类型加入 underlays 数组
//  2. 循环调用 lastUnderlay.Underlying()
//  3. 每一层都检查：typeMap 缓存 → 自定义映射 → 内置映射
//  4. 直到 Underlying() 返回 nil（终点）
//  5. 根据终点的 Kind 分发到 buildStruct / buildSlice / buildKind
//
// ## 别名穿透的 toLike 机制
//
// 当 aliasType 为 false（链路中遇到过类型定义）时，Generator 使用 toLike
// 函数将终点 struct 的 Field 名拼接为 LikeXxxField[T] 形式：
//
//	type A B; type B struct{}  → LikeBField[A]
//
// 当 aliasType 为 true（全程都是别名）时，直接返回缓存的 TypeInfo，不生成新类型。
//
// ## 反射适配层（reflectsource.go）
//
// reflectsource.go 实现了 TypeSource / FieldSource 接口，把 reflect.Type
// 适配为与 AST 版相同的接口。反射版天然具备穿透能力（reflect.Type 的
// Elem() 直接返回目标类型），因此 Underlying() 始终返回 nil, false。
//
// # 使用方式
//
//	import "github.com/xpwu/go-mongodb/gen"
//	import "reflect"
//
//	// AST 版：从 Go 文件解析
//	ts, err := gen.ParseStructFromFile(".", "UserInfo")
//
//	// 反射版：从 reflect.Type 适配
//	ts := gen.ReflectTypeSource(reflect.TypeOf(UserInfo{}))
//
//	// 通过 generator 统一入口生成
//	g := gen.NewGenerator(&gen.Config{Dir: "output", Pkg: "mypkg"})
//	subDir := g.Generate(ts)
package gen
