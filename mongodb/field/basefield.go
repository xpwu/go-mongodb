package field

import (
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/index"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type BaseField[T any] struct {
	name string
}

func NewBaseField[T any](name string) *BaseField[T] {
	return &BaseField[T]{name: name}
}

func (b *BaseField[T]) FullName() string {
	return b.name
}

func (b *BaseField[T]) Exist() filter.Filter {
	return filter.Exist(b)
}

func (b *BaseField[T]) NotExist() filter.Filter {
	return filter.NotExist(b)
}

func (b *BaseField[T]) Type(t bson.Type) filter.Filter {
	return filter.Type(b, t)
}

func (b *BaseField[T]) Gt(value T) filter.Filter {
	return filter.CompareByValue(b, filter.GT, value)
}

func (b *BaseField[T]) GtField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.GT, f)
}

func (b *BaseField[T]) Lt(value T) filter.Filter {
	return filter.CompareByValue(b, filter.LT, value)
}

func (b *BaseField[T]) LtField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.LT, f)
}

var (
	_ filter.BaseFilterField[any] = &BaseField[any]{}
)

func (b *BaseField[T]) Unset() updater.Updater {
	return updater.New(b, `$unset`, "")
}

func (b *BaseField[T]) Set(value T) updater.Updater {
	return updater.New(b, `$set`, value)
}

func (b *BaseField[T]) SetOnInsert(value T) updater.Updater {
	return updater.New(b, `$setOnInsert`, value)
}

// SetMin finalValue = min(value, nowValue) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) SetMin(value T) updater.Updater {
	return updater.New(b, "$min", value)
}

// SetMax finalValue = max(value, nowValue) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) SetMax(value T) updater.Updater {
	return updater.New(b, "$max", value)
}

var (
	_ updater.BaseUpdater[any] = &BaseField[any]{}
)

// Inc finalValue = T(nowValue + num) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) Inc(num T) updater.Updater {
	return updater.New(b, "$inc", num)
}

// Mul finalValue = nowValue * num or finalValue = 0 (if nowValue is Not exist)
func (b *BaseField[T]) Mul(num T) updater.Updater {
	return updater.New(b, "$mul", num)
}

func (b *BaseField[T]) AscIndex() index.Key {
	return index.NewKey(b, 1)
}

func (b *BaseField[T]) DescIndex() index.Key {
	return index.NewKey(b, -1)
}
