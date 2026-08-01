package filter

import (
	"fmt"
	"github.com/xpwu/go-db-mongo/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"reflect"
)

type Filter interface {
	ToBsonD() *bson.D
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
	f        mongodb.Field
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

func (b *base) ToBsonD() *bson.D {
	name := b.f.FullName()
	if name == "" {
		return &bson.D{{b.operator, b.value}}
	}

	return &bson.D{{name, bson.D{{b.operator, b.value}}}}
}

func New(f mongodb.Field, operator string, value interface{}) Filter {
	return &base{
		f:        f,
		operator: operator,
		value:    value,
	}
}

func Exist(f mongodb.Field) Filter {
	return &base{
		f:        f,
		operator: `$exists`,
		value:    true,
	}
}

func NotExist(f mongodb.Field) Filter {
	return &base{
		f:        f,
		operator: `$exists`,
		value:    false,
	}
}

func Type(f mongodb.Field, t bson.Type) Filter {
	return &base{
		f:        f,
		operator: `$type`,
		value:    t,
	}
}

func CompareByValue(f mongodb.Field, c Comparer, value interface{}) Filter {
	return &base{
		f:        f,
		operator: c.String(),
		value:    value,
	}
}

type exprFilter struct {
	f1       mongodb.Field
	operator string
	f2       mongodb.Field
}

func (e *exprFilter) ToBsonD() *bson.D {
	return &bson.D{{"$expr", bson.D{{e.operator,
		[]string{e.f1.FullName(), e.f2.FullName()}}}}}
}

// CompareByField compare fields from the same document.
// https://www.mongodb.com/docs/manual/reference/operator/query/expr/#compare-two-fields-from-a-single-document
func CompareByField(f1 mongodb.Field, c Comparer, f2 mongodb.Field) Filter {
	return &exprFilter{
		f1:       f1,
		operator: c.String(),
		f2:       f2,
	}
}

func SameElemMatch(f mongodb.Field, filter Filter) Filter {
	return &base{
		f:        f,
		operator: `$elemMatch`,
		value:    filter.ToBsonD(),
	}
}

// Not selects the documents
//
// 1. that do not match the <operator-expression>.
//
// 2.This includes documents that do not contain the field
// https://www.mongodb.com/docs/manual/reference/operator/query/not/#mongodb-query-op.-not
func Not(filter Filter) Filter {
	if b, ok := filter.(*base); ok {
		return b.not()
	}

	panic(fmt.Sprintf("%s has not `not` method ", reflect.TypeOf(filter).Elem().Name()))
}

func And(filter1, filter2 Filter, filters ...Filter) Filter {
	return newLogic(and, filter1, filter2, filters...)
}

func Or(filter1, filter2 Filter, filters ...Filter) Filter {
	return newLogic(or, filter1, filter2, filters...)
}

// Nor selects the documents that fail all the query predicates in the array,
// including those documents that do not contain these field(s).
//
// NOTE THAT: The exception in returning documents that do not contain the field in the $nor expression
//is when the $nor operator is used with the $exists operator.
// https://www.mongodb.com/docs/manual/reference/operator/query/nor/#-nor-and--exists
func Nor(filter1, filter2 Filter, filters ...Filter) Filter {
	return newNor(filter1, filter2, filters...)
}
