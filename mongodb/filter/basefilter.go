package filter

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type BaseFilterField[T any] interface {
	mongodb.Field
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
}
