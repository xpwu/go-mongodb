package fields

import (
	filter2 "github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	updater2 "github.com/xpwu/go-mongodb/updater"
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

func (b *BaseField[T]) Exist() filter2.Filter {
	return filter2.Exist(b)
}

func (b *BaseField[T]) NotExist() filter2.Filter {
	return filter2.NotExist(b)
}

func (b *BaseField[T]) Type(t bson.Type) filter2.Filter {
	return filter2.Type(b, t)
}

func (b *BaseField[T]) Gt(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.GT, value)
}

func (b *BaseField[T]) GtField(f filter2.BaseFilterField[T]) filter2.Filter {
	return filter2.CompareByValue(b, filter2.GT, f)
}

func (b *BaseField[T]) Lt(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.LT, value)
}

func (b *BaseField[T]) LtField(f filter2.BaseFilterField[T]) filter2.Filter {
	return filter2.CompareByValue(b, filter2.LT, f)
}

var (
	_ filter2.BaseFilterField[any] = &BaseField[any]{}
)

func (b *BaseField[T]) Unset() updater2.Updater {
	return updater2.New(b, `$unset`, "")
}

func (b *BaseField[T]) Set(value T) updater2.Updater {
	return updater2.New(b, `$set`, value)
}

func (b *BaseField[T]) SetOnInsert(value T) updater2.Updater {
	return updater2.New(b, `$setOnInsert`, value)
}

// SetMin finalValue = min(value, nowValue) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) SetMin(value T) updater2.Updater {
	return updater2.New(b, "$min", value)
}

// SetMax finalValue = max(value, nowValue) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) SetMax(value T) updater2.Updater {
	return updater2.New(b, "$max", value)
}

var (
	_ updater2.BaseUpdater[any] = &BaseField[any]{}
)

// Inc finalValue = T(nowValue + num) or finalValue = value (if nowValue is Not exist)
func (b *BaseField[T]) Inc(num T) updater2.Updater {
	return updater2.New(b, "$inc", num)
}

// Mul finalValue = nowValue * num or finalValue = 0 (if nowValue is Not exist)
func (b *BaseField[T]) Mul(num T) updater2.Updater {
	return updater2.New(b, "$mul", num)
}

func (b *BaseField[T]) AscIndex() index.Key {
	return index.NewKey(b, 1)
}

func (b *BaseField[T]) DescIndex() index.Key {
	return index.NewKey(b, -1)
}

func (b *BaseField[T]) Eq(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.EQ, value)
}

func (b *BaseField[T]) EqField(f filter2.ComparableFilterField[T]) filter2.Filter {
	return filter2.CompareByField(b, filter2.EQ, f)
}

func (b *BaseField[T]) Ne(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.NE, value)
}

func (b *BaseField[T]) NeField(f filter2.ComparableFilterField[T]) filter2.Filter {
	return filter2.CompareByField(b, filter2.NE, f)
}

func (b *BaseField[T]) Gte(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.GTE, value)
}

func (b *BaseField[T]) GteField(f filter2.ComparableFilterField[T]) filter2.Filter {
	return filter2.CompareByValue(b, filter2.GTE, f)
}

func (b *BaseField[T]) Lte(value T) filter2.Filter {
	return filter2.CompareByValue(b, filter2.LTE, value)
}

func (b *BaseField[T]) LteField(f filter2.ComparableFilterField[T]) filter2.Filter {
	return filter2.CompareByValue(b, filter2.LTE, f)
}

func (b *BaseField[T]) In(values []T) filter2.Filter {
	return filter2.New(b, "$in", values)
}

func (b *BaseField[T]) Nin(values []T) filter2.Filter {
	return filter2.New(b, "$nin", values)
}

var (
	_ filter2.ComparableFilterField[any] = &BaseField[any]{}
)
