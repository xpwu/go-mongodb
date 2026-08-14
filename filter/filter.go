package filter

import (
	"github.com/xpwu/go-mongodb/field"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Filter interface {
	ToBsonD() bson.D
}

type Comparer byte

const (
	EQ Comparer = iota
	GT
	GTE
	LT
	LTE
	NE
)

func (c Comparer) String() string {
	str := []string{`$eq`, `$gt`, `$gte`, `$lt`, `$lte`, `$ne`}
	return str[c]
}

type base struct {
	f        field.Field
	operator string
	value    interface{}
}

func (b *base) not() *base {
	return &base{
		f:        b.f,
		operator: `$not`,
		value:    bson.M{b.operator: b.value},
	}
}

func (b *base) ToBsonD() bson.D {
	name := b.f.FullName()
	if name == "" {
		return bson.D{{b.operator, b.value}}
	}

	return bson.D{{name, bson.D{{b.operator, b.value}}}}
}

func New(f field.Field, operator string, value interface{}) Filter {
	return &base{
		f:        f,
		operator: operator,
		value:    value,
	}
}

type bsonD bson.D

func (b bsonD) ToBsonD() bson.D {
	return bson.D(b)
}

func FromBsonD(d bson.D) Filter {
	return bsonD(d)
}

func Exist(f field.Field) PartialIndexFilter {
	return AsPartialIndexFilter(&base{
		f:        f,
		operator: `$exists`,
		value:    true,
	})
}

func NotExist(f field.Field) Filter {
	return &base{
		f:        f,
		operator: `$exists`,
		value:    false,
	}
}

func Type(f field.Field, t bson.Type) PartialIndexFilter {
	return AsPartialIndexFilter(&base{
		f:        f,
		operator: `$type`,
		value:    t,
	})
}

func CompareByValue(f field.Field, c Comparer, value interface{}) Filter {
	if c == EQ {
		return FromBsonD(bson.D{{f.FullName(), value}})
	}
	return &base{
		f:        f,
		operator: c.String(),
		value:    value,
	}
}

type exprFilter struct {
	f1       field.Field
	operator string
	f2       field.Field
}

func (e *exprFilter) ToBsonD() bson.D {
	return bson.D{{"$expr", bson.D{{e.operator,
		bson.A{"$" + e.f1.FullName(), "$" + e.f2.FullName()}}}}}
}

// CompareByField compare fields from the same document.
// https://www.mongodb.com/docs/manual/reference/operator/query/expr/#compare-two-fields-from-a-single-document
func CompareByField(f1 field.Field, c Comparer, f2 field.Field) Filter {
	return &exprFilter{
		f1:       f1,
		operator: c.String(),
		f2:       f2,
	}
}

func SameElemMatch(f field.Field, filter Filter) Filter {
	return &base{
		f:        f,
		operator: `$elemMatch`,
		value:    filter.ToBsonD(),
	}
}

type PartialIndexFilter interface {
	Filter
	partialAble()
}

type partialFilter struct {
	Filter
}

func (p partialFilter) partialAble() {

}

func AsPartialIndexFilter(filter Filter) PartialIndexFilter {
	return partialFilter{filter}
}
