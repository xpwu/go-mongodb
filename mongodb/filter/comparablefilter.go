package filter

import "github.com/xpwu/go-db-mongo/mongodb"

type ComparableFilterField[T any] interface {
	mongodb.Field
	ComparableFilter[T]
}

// ComparableFilter T ~ comparable | EqualAble
type ComparableFilter[T any] interface {
	BaseFilter[T]
	Eq(value T) Filter
	EqField(f ComparableFilterField[T]) Filter
	
	// Ne selects documents
	//
	// 1. where the value of the field is not equal to the specified value.
	//
	// 2. This includes documents that do not contain the field.
	// https://www.mongodb.com/docs/manual/reference/operator/query/ne/
	Ne(value T) Filter
	NeField(f ComparableFilterField[T]) Filter
	Gte(value T) Filter
	GteField(f ComparableFilterField[T]) Filter
	Lte(value T) Filter
	LteField(f ComparableFilterField[T]) Filter
	In(values []T) Filter

	// Nin selects documents where:
	//
	// 1. the specified field value is not in the specified array or
	//
	// 2. the specified field does not exist.
	// https://www.mongodb.com/docs/manual/reference/operator/query/nin/
	Nin(values []T) Filter
}

type EqualAble[T EqualAble[T]] interface {
	Equal(T) bool
}
