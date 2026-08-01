package field

import "github.com/xpwu/go-db-mongo/mongodb/filter"

func (b *BaseField[T]) Eq(value T) filter.Filter {
	return filter.CompareByValue(b, filter.EQ, value)
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

func (b *BaseField[T]) Gte(value T) filter.Filter {
	return filter.CompareByValue(b, filter.GTE, value)
}

func (b *BaseField[T]) GteField(f filter.ComparableFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.GTE, f)
}

func (b *BaseField[T]) Lte(value T) filter.Filter {
	return filter.CompareByValue(b, filter.LTE, value)
}

func (b *BaseField[T]) LteField(f filter.ComparableFilterField[T]) filter.Filter {
	return filter.CompareByValue(b, filter.LTE, f)
}

func (b *BaseField[T]) In(values []T) filter.Filter {
	return filter.New(b, "$in", values)
}

func (b *BaseField[T]) Nin(values []T) filter.Filter {
	return filter.New(b, "$nin", values)
}

var (
	_ filter.ComparableFilterField[any] = &BaseField[any]{}
)
