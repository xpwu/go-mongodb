package fields

import (
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/updater"
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

func (b *BaseField[T]) Exist() filter.PartialIndexFilter {
	return filter.Exist(b)
}

func (b *BaseField[T]) NotExist() filter.Filter {
	return filter.NotExist(b)
}

func (b *BaseField[T]) Type(t bson.Type) filter.PartialIndexFilter {
	return filter.Type(b, t)
}

func (b *BaseField[T]) Gt(value T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.CompareByValue(b, filter.GT, value))
}

func (b *BaseField[T]) GtField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.GT, f)
}

func (b *BaseField[T]) Lt(value T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.CompareByValue(b, filter.LT, value))
}

func (b *BaseField[T]) LtField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.LT, f)
}

func (b *BaseField[T]) Gte(value T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.CompareByValue(b, filter.GTE, value))
}

func (b *BaseField[T]) GteField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.GTE, f)
}

func (b *BaseField[T]) Lte(value T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.CompareByValue(b, filter.LTE, value))
}

func (b *BaseField[T]) LteField(f filter.BaseFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.LTE, f)
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

func (b *BaseField[T]) AscIndex(opts ...index.Option) index.Key {
	return index.NewKey(b, index.KeyTypeAscendingOrder, opts...)
}

func (b *BaseField[T]) DescIndex(opts ...index.Option) index.Key {
	return index.NewKey(b, index.KeyTypeDescendingOrder, opts...)
}

func (b *BaseField[T]) Eq(value T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.CompareByValue(b, filter.EQ, value))
}

func (b *BaseField[T]) EqField(f filter.ComparableFilterField[T]) filter.Filter {
	return filter.CompareByField(b, filter.EQ, f)
}

func (b *BaseField[T]) Ne(value T) filter.Filter {
	return filter.CompareByValue(b, filter.NE, value)
}

func (b *BaseField[T]) NeField(f filter.ComparableFilterField[T]) filter.Filter {
	return filter.CompareByField(b, filter.NE, f)
}

func (b *BaseField[T]) In(values []T) filter.PartialIndexFilter {
	return filter.AsPartialIndexFilter(filter.New(b, "$in", values))
}

func (b *BaseField[T]) Nin(values []T) filter.Filter {
	return filter.New(b, "$nin", values)
}

var (
	_ filter.ComparableFilterField[any] = &BaseField[any]{}
)
