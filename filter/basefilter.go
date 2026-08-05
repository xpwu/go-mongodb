package filter

import (
	"github.com/xpwu/go-mongodb/field"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type BaseFilterField[T any] interface {
	field.Field
	BaseFilter[T]
}

type BaseFilter[T any] interface {
	Exist() PartialIndexFilter
	NotExist() Filter
	Type(t bson.Type) PartialIndexFilter
	Gt(value T) PartialIndexFilter
	GtField(f BaseFilterField[T]) Filter
	Lt(value T) PartialIndexFilter
	LtField(f BaseFilterField[T]) Filter
	Gte(value T) PartialIndexFilter
	GteField(f BaseFilterField[T]) Filter
	Lte(value T) PartialIndexFilter
	LteField(f BaseFilterField[T]) Filter
}
