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
	Exist() Filter
	NotExist() Filter
	Type(t bson.Type) Filter
	Gt(value T) Filter
	GtField(f BaseFilterField[T]) Filter
	Lt(value T) Filter
	LtField(f BaseFilterField[T]) Filter
	Gte(value T) Filter
	GteField(f BaseFilterField[T]) Filter
	Lte(value T) Filter
	LteField(f BaseFilterField[T]) Filter
}
