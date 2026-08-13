package projection

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ─── 普通 IncludeBuilder ───

type IncludeBuilder struct {
	proj bson.D
	seen map[string]int
}

// Include 创建一个不包含任何数组裁剪操作的 IncludeBuilder
func Include(fields ...string) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	for _, f := range fields {
		b.raw(f, 1)
	}
	return b
}

// ─── 带数组裁剪的构造函数 ───

// IncludeWithSlice 创建带 $slice 的 IncludeBuilder
func IncludeWithSlice(field string, n int) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	b.seen[field] = 0
	b.proj = append(b.proj, bson.E{Key: field, Value: bson.D{{"$slice", n}}})
	return b
}

// IncludeWithSliceRange 创建带 $slice(skip, limit) 的 IncludeBuilder
func IncludeWithSliceRange(field string, skip, limit int) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	b.seen[field] = 0
	b.proj = append(b.proj, bson.E{Key: field, Value: bson.D{{"$slice", bson.A{skip, limit}}}})
	return b
}

// IncludeWithElemMatch 创建带 $elemMatch 的 IncludeBuilder
func IncludeWithElemMatch(field string, condition bson.D) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	b.seen[field] = 0
	b.proj = append(b.proj, bson.E{Key: field, Value: bson.D{{"$elemMatch", condition}}})
	return b
}

// IncludeWithFirstMatch 创建带 $ 位置符的 IncludeBuilder
func IncludeWithFirstMatch(field string) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	path := fmt.Sprintf("%s.$", field)
	b.seen[path] = 0
	b.proj = append(b.proj, bson.E{Key: path, Value: 1})
	return b
}

// ─── IncludeBuilder 方法 ───

func (b *IncludeBuilder) Exclude_id() *IncludeBuilder {
	b.raw("_id", 0)
	return b
}

// WithSearchRelevance adds a field to the projection that returns the text search relevance score.
//
// The score is computed by MongoDB based on the $text search query and represents
// how well the document matches the search terms (similar to Elasticsearch's _score).
// This field does NOT exist in the stored document — MongoDB calculates it at query time.
//
//	name: the name of the field in the returned document that will hold the score.
//	      This is NOT a stored field name and NOT a MongoDB keyword.
//	      You can name it anything, e.g. "score", "relevance", "matchScore".
//
// Usage example:
//
//	Include("title").WithSearchRelevance("score").Build()
//	// → { title: 1, score: { $meta: "textScore" } }
//
// Note: Only meaningful when the query uses $text search. Without $text, the score is always 0.
func (b *IncludeBuilder) WithSearchRelevance(name string) *IncludeBuilder {
	b.raw(name, bson.D{{"$meta", "textScore"}})
	return b
}

// AtIndex 按数组下标投影
func (b *IncludeBuilder) AtIndex(field string, index int) *IncludeBuilder {
	path := fmt.Sprintf("%s.%d", field, index)
	b.raw(path, 1)
	return b
}

func (b *IncludeBuilder) Build() bson.D {
	return b.proj
}

// ─── 内部 ───

func (b *IncludeBuilder) raw(field string, value interface{}) {
	// 情况1：同名字段 → 覆盖
	if i, ok := b.seen[field]; ok {
		b.proj[i] = bson.E{Key: field, Value: value}
		return
	}

	// 情况2：当前字段是某个已存在字段的子路径 → 父赢，子跳过
	for existing := range b.seen {
		if isParentOrSame(existing, field) {
			return // 父已经包含了，直接忽略子
		}
	}

	// 情况3：当前字段是某个已存在字段的父路径 → 子被吞掉，清掉子
	for existing := range b.seen {
		if isParentOrSame(field, existing) {
			// 从 proj 里删掉子
			idx := b.seen[existing]
			b.proj = append(b.proj[:idx], b.proj[idx+1:]...)
			// 重新调整后面元素的 index
			for f, i := range b.seen {
				if i > idx {
					b.seen[f] = i - 1
				}
			}
			delete(b.seen, existing)
		}
	}

	// 写入自己
	b.seen[field] = len(b.proj)
	b.proj = append(b.proj, bson.E{Key: field, Value: value})
}

func isParentOrSame(parent, child string) bool {
	if parent == child {
		return true
	}
	if len(child) > len(parent) && child[len(parent)] == '.' && child[:len(parent)] == parent {
		return true
	}
	return false
}
