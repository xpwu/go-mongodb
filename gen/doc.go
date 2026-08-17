// Package gen 实现了 go-mongodb-gen 代码生成器的核心逻辑。
//
// # 设计目标
//
// 本包负责从 Go 源码的 AST 中解析类型信息，并为每个 struct 及其嵌套类型
// 生成对应的 Field 类型代码。以下是核心设计原则：
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
// ## 2. 切片别名不生成独立 Field 文件
//
// type X []Y 这种切片别名，不生成新的 Field 文件，而是展开为对应的 ArrayField。
//
//	type GPSes []GPS  → 生成 ArrayComparableField[GPS, ComputableField[GPS]]
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
//	                        生成 ArrayComparableField[GPS, ComputableField[GPS]]
//	                        不生成 zGPSesAField.go
//
// 实现方式：在 AST 解析阶段检查 typeSpec.Assign != token.NoPos，
// 将目标类型记录在 aliasTargets 映射中，遇到标识符时直接返回目标类型。
//
// ## 4. 复合类型别名的穿透
//
// 对于跨包的复合类型别名（如 bson.D），需要穿透别名找到底层元素类型：
//
//	type D []E        → bson.D 的 Kind 是 reflect.Slice，Elem 是 bson.E
//	type E struct{...} → 生成 EField，bson.D 生成 ArrayField[bson.E, EField]
//
// ## 5. interface{} / any 的特殊处理
//
// any 和 interface{} 是 Go 内置类型，不属于任何包：
//
//	any / interface{}  → 生成 BaseStructField[any]（不带包路径前缀）
//
// 用户定义的 interface 类型（如 type A interface{ Do() }）同样走 Interface 分支，
// 生成 BaseStructField[A]（带包路径）。
//
// 对于 bson.any 这种通过 SelectorExpr 引用的形式，强制将 Kind 设为
// reflect.Interface 并清空 pkgPath，防止生成 bson.any 这种错误引用。
//
// ## 6. 只有 struct 才进入待处理队列
//
// 在生成过程中，只有 Kind == reflect.Struct 的类型才会被加入 pendingSts 队列，
// 触发独立的 Field 文件生成。Slice、Ptr、基本类型、Interface 都不生成新文件。
//
// # 反射适配层（reflectsource.go）
//
// reflectsource.go 实现了 TypeSource / FieldSource 接口，把 reflect.Type
// 适配为与 AST 版相同的接口，使得 generator.go 的 Generate(ts TypeSource)
// 方法可以统一接受两种输入源。
//
// ## 使用方式
//
//	import "github.com/xpwu/go-mongodb/gen"
//	import "reflect"
//
//	// 把 reflect.Type 适配成 TypeSource
//	ts := gen.ReflectTypeSource(reflect.TypeOf(UserInfo{}))
//
//	// 通过 generator 统一入口生成
//	g := gen.NewGenerator(&gen.Config{Dir: "output", Pkg: "mypkg"})
//	subDir := g.Generate(ts)
//
// ## 反射版天然具备的能力
//
//   - type A = C 穿透：reflect.Type 中 A 和 C 是同一个对象，Elem() 直接返回目标类型
//   - type D []E 切片展开：reflect.Type.Elem() 直接返回 E 的 reflect.Type
//   - interface 识别：reflect.Type.Kind() == reflect.Interface 直接可用
//
// 因此 reflectsource.go 本身不需要额外的别名解析逻辑，只需要正确实现
// TypeSource 接口的 6 个方法即可。
//
// # AST 解析关键设计
//
// ## 递归加载的缓存策略
//
// LoadPackage 在 parsePackageDir 完成之后、填充 types/aliases 之前，
// 将 pkg 放入 loaded 缓存。这样在填充过程中如果遇到需要递归加载同一包
// 的情况（如解析 type D []E 时需要加载 bson 包），能命中缓存而不会死循环。
//
// ## parsePackageDir 统一初始化
//
// parsePackageDir 负责创建完整初始化的 loadedPackage（所有 map 字段都在
// parsePackageDir 里 make），LoadPackage 不再重复 make。这样不会重复分配
// 内存，也不会有人忘记加新字段。
//
// ## typeElems 的作用
//
// 对于 type D []E 这种复合类型别名，除了在 aliases 里记录 kind=Slice，
// 还需要在 typeElems 里记录 E 的 astTypeSource，这样 Elem() 才能返回正确的元素类型。
//
// ## aliasTargets 的作用
//
// 对于 type A = C 这种类型别名声明，在 aliasTargets 里记录 C 的 astTypeSource。
// 当 parseAstTypeWithLoader 遇到标识符 A 时，先查 aliasTargets，
// 如果找到则直接返回 C 的 astTypeSource，实现完全穿透。
//
// ## SelectorExpr 的 any 防御
//
// 当遇到 bson.any 或 bson.interface{} 这种 SelectorExpr 时，
// 强制将 Kind 设为 reflect.Interface 并清空 pkgPath，
// 防止生成 bson.any 这种错误的类型引用。
//
// ## kindFromName 的定位
//
// kindFromName 只是一个"快速路径"初步猜测，后面会被 aliases 和 pkg.types
// 二次确认修正。它不会造成错误判断，但需要在 isBuiltinKind 里覆盖所有
// 需要清空 pkgPath 的 Kind（包括 reflect.Interface）。
package gen
