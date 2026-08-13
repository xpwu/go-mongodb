package projection

import (
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type IncludeBuilder struct {
	proj bson.D
	seen map[string]int

	sliceFld string
	elemFld  string
	posFld   string
}

func Include(fields ...string) *IncludeBuilder {
	b := &IncludeBuilder{seen: make(map[string]int)}
	for _, f := range fields {
		b.raw(f, 1)
	}
	return b
}

func (b *IncludeBuilder) Exclude_id() *IncludeBuilder {
	return b.raw("_id", 0)
}

func (b *IncludeBuilder) Meta(field, metaType string) *IncludeBuilder {
	return b.raw(field, bson.D{{"$meta", metaType}})
}

func (b *IncludeBuilder) AtIndex(field string, index int) *IncludeBuilder {
	path := fmt.Sprintf("%s.%d", field, index)
	b.assertNoParentChild(field, path)
	return b.raw(path, 1)
}

func (b *IncludeBuilder) FirstMatch(field string) *IncludeBuilder {
	if b.posFld != "" {
		panic("can only use $ positional once per projection")
	}
	b.assertOneArrayOp(field, "$")
	path := fmt.Sprintf("%s.$", field)
	b.posFld = field
	b.assertNoParentChild(field, path)
	return b.raw(path, 1)
}

func (b *IncludeBuilder) Slice(field string, n int) *IncludeBuilder {
	b.assertOneArrayOp(field, "$slice")
	b.sliceFld = field
	b.assertNoParentChild(field, field)
	return b.raw(field, bson.D{{"$slice", n}})
}

func (b *IncludeBuilder) SliceRange(field string, skip, limit int) *IncludeBuilder {
	b.assertOneArrayOp(field, "$slice")
	b.sliceFld = field
	b.assertNoParentChild(field, field)
	return b.raw(field, bson.D{{"$slice", bson.A{skip, limit}}})
}

func (b *IncludeBuilder) ElemMatch(field string, condition bson.D) *IncludeBuilder {
	b.assertOneArrayOp(field, "$elemMatch")
	b.elemFld = field
	b.assertNoParentChild(field, field)
	return b.raw(field, bson.D{{"$elemMatch", condition}})
}

func (b *IncludeBuilder) Build() bson.D {
	return b.proj
}

func (b *IncludeBuilder) raw(field string, value interface{}) *IncludeBuilder {
	if i, ok := b.seen[field]; ok {
		b.proj[i] = bson.E{Key: field, Value: value}
	} else {
		b.seen[field] = len(b.proj)
		b.proj = append(b.proj, bson.E{Key: field, Value: value})
	}
	return b
}

func (b *IncludeBuilder) assertNoParentChild(parent, child string) {
	for fld := range b.seen {
		if fld == parent || fld == child {
			continue
		}
		if isParentOrSame(fld, parent) || isParentOrSame(fld, child) {
			if isParentOrSame(parent, fld) || isParentOrSame(child, fld) {
				panic("parent and child field cannot both be projected")
			}
		}
	}
}

func (b *IncludeBuilder) assertOneArrayOp(field, op string) {
	if b.sliceFld != "" && b.sliceFld != field {
		panic("can only apply $slice to one array field")
	}
	if b.elemFld != "" && b.elemFld != field {
		panic("can only apply $elemMatch to one array field")
	}
	if b.posFld != "" && b.posFld != field {
		if op == "$slice" {
			panic("cannot use $ positional and $slice on the same field")
		}
		if op == "$elemMatch" {
			panic("cannot use $ positional and $elemMatch on the same field")
		}
	}
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
