package projection

import (
	"fmt"
	"github.com/xpwu/go-mongodb/field"
	"github.com/xpwu/go-mongodb/filter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Inclusion provides type-safe projection builders for array fields.
// Implementers of this interface (e.g., ArrayBaseField) should be the primary
// way users construct array projections. The package-level IncludeWithXxx
// functions are lower-level alternatives for dynamic field paths.
type Inclusion[ElemField field.Field] interface {
	ProjectWithSlice(n int) *IncludeBuilder
	ProjectWithSliceRange(skip, limit int) *IncludeBuilder
	ProjectWithElemMatch(f func(theOne ElemField) filter.Filter) *IncludeBuilder
	ProjectWithFirstMatch() *IncludeBuilder
}

// ─── IncludeBuilder ───

type IncludeBuilder struct {
	proj bson.D
	seen map[string]int
}

// Include creates an IncludeBuilder for normal field inclusion without any
// array slicing operators ($slice, $elemMatch, $).
func Include(fields ...field.Field) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	for _, f := range fields {
		b.Field(f)
	}
	return b
}

// ─── Array partial constructors ───

// IncludeWithSlice creates a projection for an array field with the $slice operator.
//
// Most callers should NOT use this function directly. If you already have an
// ArrayField, prefer field.ProjectWithSlice instead—it knows the field path
// and returns an IncludeBuilder in one step.
//
// This function exists for advanced use cases where the field path is built
// dynamically or the projection is constructed outside the fields package.
//
// IncludeWithSlice should be avoided in favor of ArrayBaseField.ProjectWithSlice.
// It is kept for use cases where the fields package is not in scope.
func IncludeWithSlice(field field.Field, n int) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	b.seen[field.FullName()] = 0
	b.proj = append(b.proj, bson.E{Key: field.FullName(), Value: bson.D{{"$slice", n}}})
	return b
}

// IncludeWithSliceRange creates a projection for an array field with the $slice operator (skip + limit).
//
// Most callers should NOT use this function directly. If you already have an
// ArrayField, prefer field.ProjectWithSliceRange instead—it knows the field path
// and returns an IncludeBuilder in one step.
//
// This function exists for advanced use cases where the field path is built
// dynamically or the projection is constructed outside the fields package.
//
// IncludeWithSlice should be avoided in favor of ArrayBaseField.ProjectWithSliceRange.
// It is kept for use cases where the fields package is not in scope.
func IncludeWithSliceRange(field field.Field, skip, limit int) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	b.seen[field.FullName()] = 0
	b.proj = append(b.proj, bson.E{Key: field.FullName(), Value: bson.D{{"$slice", bson.A{skip, limit}}}})
	return b
}

// IncludeWithElemMatch creates a projection for an array field with the $elemMatch operator.
//
// Most callers should NOT use this function directly. If you already have an
// ArrayField, prefer field.ProjectWithElemMatch instead—it provides type-safe
// element access and handles the $elemMatch wrapper automatically.
//
// This function exists for advanced use cases where the field path is built
// dynamically or the projection is constructed outside the fields package.
//
// IncludeWithSlice should be avoided in favor of ArrayBaseField.ProjectWithElemMatch.
// It is kept for use cases where the fields package is not in scope.
func IncludeWithElemMatch(fil filter.Filter) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	fd := fil.ToBsonD()
	b.seen[fd[0].Key] = 0
	b.proj = append(b.proj, fd[0])
	return b
}

// IncludeWithFirstMatch creates a projection for an array field with the positional $ operator.
//
// Most callers should NOT use this function directly. If you already have an
// ArrayField, prefer field.ProjectWithFirstMatch instead—it knows the field path
// and returns an IncludeBuilder in one step.
//
// This function exists for advanced use cases where the field path is built
// dynamically or the projection is constructed outside the fields package.
//
// IncludeWithSlice should be avoided in favor of ArrayBaseField.ProjectWithFirstMatch.
// It is kept for use cases where the fields package is not in scope.
func IncludeWithFirstMatch(field field.Field) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	path := fmt.Sprintf("%s.$", field)
	b.seen[path] = 0
	b.proj = append(b.proj, bson.E{Key: path, Value: 1})
	return b
}

// ─── Methods ───

func (b *IncludeBuilder) Exclude_id() *IncludeBuilder {
	if i, ok := b.seen["_id"]; ok {
		b.proj[i] = bson.E{Key: "_id", Value: 0}
	} else {
		b.seen["_id"] = len(b.proj)
		b.proj = append(b.proj, bson.E{Key: "_id", Value: 0})
	}
	return b
}

func (b *IncludeBuilder) WithSearchRelevance(name string) *IncludeBuilder {
	b.addField(name, bson.D{{"$meta", "textScore"}})
	return b
}

// Field adds a field to the projection with value 1 (inclusion).
//
// The field path supports dot notation for nested fields and array index access.
// Examples:
//   - Nested field:  Field("comments.user")
//   - Array element: Field("scores.0")   — equivalent to selecting scores[0]
//   - Deep path:     Field("a.b.2.c")    — the "2" is treated as array index
//
// Conflict resolution rules (priority from high to low):
//
// 1. Partial wins over whole (部分永远赢，整体永远被抛弃)
//    If a whole field (value=1) and a partial (array index, $slice, $elemMatch, $)
//    target the same field, the partial wins regardless of write order.
//    Reason: returning more array elements than intended causes hidden bugs;
//    returning less is easy to detect.
//    如果整体字段（value=1）和部分操作（数组索引、$slice、$elemMatch、$）
//    指向同一字段，部分永远赢，不管写入顺序。原因：返回超出预期的数组元素会导致隐蔽
//    的 bug；返回过少则容易被发现。
//
// 2. Partial + partial on same ancestor → first wins (同字段部分之间，先写赢)
//    If two partial operations have an ancestor relationship, the first write wins.
//    如果两个部分操作存在祖先关系，先写的保留，后写的被抛弃。
//
// 3. Duplicate field: later discarded, first wins (同名字段，后写抛弃)
//    If the exact same field is specified twice, the first write wins.
//    同一字段被指定两次，先写的保留，后写的被抛弃。
//
// 4. Whole parent + whole child: keep parent, discard child (整体的父 + 整体的子，去子留父)
//    If a parent field (value=1, not array partial) and its child path (value=1) both exist,
//    the parent wins because it already contains all child data.
//    如果两个都是整体（value=1 且不是数组部分），父赢子抛弃，因为父已包含子的全部数据。
//
// 5. Partial vs whole child on same array (部分 vs 同数组上的整体子字段)
//    If one path is a partial (e.g. "a.b.0") and the other is a whole-child access
//    on the same array (e.g. "a.b.c"), the partial wins regardless of write order.
//    如果一条路径是部分（如 "a.b.0"），另一条是同数组上的整体子字段（如 "a.b.c"），
//    部分永远赢，不管写入顺序。
//
// Field 向投影中添加字段，值为 1（包含）。
// 字段路径支持点号表示法，包括数组索引访问如 "scores.0"。
func (b *IncludeBuilder) Field(field field.Field) {
	b.addField(field.FullName(), 1)
}

// ─── Internal ───

func (b *IncludeBuilder) addField(field string, value interface{}) {
	// Same field: partial overrides whole, whole is discarded when existing is partial
	if i, ok := b.seen[field]; ok {
		if value != 1 {
			b.proj[i] = bson.E{Key: field, Value: value}
		}
		return
	}

	// Rule 1: Partial wins over whole (ancestor check)
	if isPartialOrIndex(field, value) {
		for existing, idx := range b.seen {
			if isAncestor(existing, field) && b.proj[idx].Value == 1 {
				b.proj = append(b.proj[:idx], b.proj[idx+1:]...)
				for f, i := range b.seen {
					if i > idx {
						b.seen[f] = i - 1
					}
				}
				delete(b.seen, existing)
				break
			}
		}
	}

	// Rule 2: Partial + partial on same ancestor → first wins
	for existing := range b.seen {
		if isAncestor(existing, field) {
			return
		}
	}

	// Rule 4: Whole parent + whole child → keep parent, discard child
	// Only when existing is also a whole (not an array partial)
	if value == 1 {
		for existing, idx := range b.seen {
			if isAncestor(field, existing) && b.proj[idx].Value == 1 && !isPartialOrIndex(existing, b.proj[idx].Value) {
				b.proj = append(b.proj[:idx], b.proj[idx+1:]...)
				for f, i := range b.seen {
					if i > idx {
						b.seen[f] = i - 1
					}
				}
				delete(b.seen, existing)
			}
		}
	}

	// Rule 5: Partial vs whole child on same array → partial wins regardless of order
	if value == 1 {
		for existing := range b.seen {
			if sameArrayField(existing, field) && isPartialOrIndex(existing, b.proj[b.seen[existing]].Value) {
				return
			}
		}
	} else {
		for existing, idx := range b.seen {
			if sameArrayField(existing, field) && !isPartialOrIndex(existing, b.proj[idx].Value) && b.proj[idx].Value == 1 {
				b.proj = append(b.proj[:idx], b.proj[idx+1:]...)
				for f, i := range b.seen {
					if i > idx {
						b.seen[f] = i - 1
					}
				}
				delete(b.seen, existing)
				break
			}
		}
	}

	b.seen[field] = len(b.proj)
	b.proj = append(b.proj, bson.E{Key: field, Value: value})
}

func (b *IncludeBuilder) Build() bson.D {
	return b.proj
}

// ─── Helpers ───

func isPartialOrIndex(field string, value interface{}) bool {
	if value != 1 {
		return true
	}
	lastDot := -1
	for i := len(field) - 1; i >= 0; i-- {
		if field[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot == -1 {
		return false
	}
	segment := field[lastDot+1:]
	for _, c := range segment {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(segment) > 0
}

func isAncestor(parent, child string) bool {
	if parent == child {
		return false
	}
	return len(child) > len(parent) && child[len(parent)] == '.' && child[:len(parent)] == parent
}

func sameArrayField(a, b string) bool {
	baseA := arrayFieldBase(a)
	baseB := arrayFieldBase(b)
	return baseA != "" && baseA == baseB
}

func arrayFieldBase(path string) string {
	lastDot := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot == -1 {
		return ""
	}
	segment := path[lastDot+1:]
	if segment == "$" {
		return path[:lastDot]
	}
	for _, c := range segment {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return path[:lastDot]
}
