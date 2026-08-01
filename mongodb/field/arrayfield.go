package field

import (
	"fmt"
	"github.com/xpwu/go-db-mongo/mongodb"
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/index"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
	"sync/atomic"
)

// VirPos is a virtual position described by filters.
//
// op: $[<identifier>]
//
//	eg: { "$elemMatch" : { size: "M", num: { $gt: 50} }
//	    { votes: { $gte: 6 } }
//	    { results: { score: 8 , item: "B" } }
//	    { answers: { $elemMatch: { q: 2, a: { $gte: 8 } } } }
// But cannot include the following query operators: $expr $text $where
//
// https://www.mongodb.com/docs/manual/reference/operator/update/positional-filtered/#definition
//
// https://www.mongodb.com/docs/manual/reference/operator/update/positional-filtered/#upsert
type VirPos filter.Filter

// ArrayFilter represents the type for the `arrayFilters` option in MongoDB update operations,
// such as db.collection.updateMany(), updateOne(), findOneAndUpdate(), etc.
//
// Example: db.collection.updateMany(filter, update, { arrayFilters: []ArrayFilter })
type ArrayFilter filter.Filter

// VirValue is a virtual value described by filters.
//
//	eg: { "$elemMatch" : { size: "M", num: { $gt: 50} }
//	    { votes: { $gte: 6 } }
//	    { results: { score: 8 , item: "B" } }
//	    { answers: { $elemMatch: { q: 2, a: { $gte: 8 } } } }
//
// https://www.mongodb.com/docs/manual/reference/operator/query/all/#use--all-with--elemmatch
type VirValue filter.Filter

type ArrayBaseField[T any, ElemField mongodb.Field] interface {
	mongodb.Field

	// AtPos returns the element at the given index pos.
	//
	// The returned elem can be used for both filter and updater creation.
	//
	// https://www.mongodb.com/docs/manual/core/document/#arrays
	//
	// https://www.mongodb.com/docs/manual/tutorial/query-arrays/#query-for-an-element-by-the-array-index-position
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/set/#set-elements-in-arrays
	AtPos(pos int) (elem ElemField)

	// Elems returns all elements of the array, used only for creating filters.
	//
	// 1. When followed by a negative condition (e.g., $ne, $not, $nin, or $nor), it behaves as "all elements";
	//
	// 2. When followed by a positive condition (e.g., $eq, $in, $gte), it behaves as "any element".
	//
	// https://www.mongodb.com/docs/manual/tutorial/query-arrays/#query-an-array-with-compound-filter-conditions-on-the-array-elements
	Elems() (elem ElemField)

	// AtVirPos returns the element at the position specified by the callback function.
	//
	// The returned elem is intended only for creating updaters.
	//
	// The returned arrayFilter is used in the `arrayFilters` option of MongoDB update operations.
	//
	// op: $[<identifier>]
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/positional-filtered/
	AtVirPos(f func(elem ElemField) VirPos) (elem ElemField, arrayFilter ArrayFilter)

	// FirstMatched returns the first element of the array that matches the filter document in updateXXX.
	//
	// The returned elem is intended only for creating updaters.
	//
	// The array field must appear as part of the filter document in updateXXX.
	//
	// eg:
	//    db.students.updateOne(
	//      { _id: 1, grades: 80 },
	//      { $set: { "grades.$" : 82 } }
	//    )
	//
	// Note the following restrictions:
	//  1. Do not use the updater created by the returned elem in upsert operations.
	//  2. Do not use the updater created by the returned elem for queries that traverse more than one array,
	//     such as queries that traverse arrays nested within other arrays.
	//  3. When used with the `Unset` method, the updater sets the element to null but does not remove
	//     the matching element from the array.
	//  4. Do not use the updater created by the returned elem if the filter matches the array using a
	//     negation operator ($ne, $not, $nin).
	//  5. The updater created by the returned elem behaves ambiguously when filtering on multiple array fields.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/positional/#behavior
	FirstMatched() (elem ElemField)

	// UpdateAll returns a delegated elem for updating all elements of the array.
	//
	// The returned elem is intended only for creating updaters.
	//
	//   Notes:
	//    1. If an upsert operation results in an insert,
	//       the filter must include an exact equality match on the array field.
	//    2. The returned elem can be used for queries that traverse more than one array and nested arrays.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/positional-all/#definition
	UpdateAll() (elems ElemField)
}

type ArrayBaseFilter[T any, ElemField mongodb.Field] interface {
	filter.BaseFilter[[]T]
	Size(sz int) filter.Filter

	// AnyElemMeet Different elements can satisfy different conditions,
	// or a single element can satisfy all conditions.
	//
	// NOTE THAT: If using a Negation operator, such as $ne, $not, or $nin, 'anyElem' is 'allElem'.
	// In other words, none of elem meets the positive operator (the counterpart of the negation operator).
	//
	//	op: { dim_cm: { $gt: 15, $lt: 20 } }
	//
	// https://www.mongodb.com/docs/manual/tutorial/query-arrays/#query-an-array-with-compound-filter-conditions-on-the-array-elements
	//AnyElemMeet(f func(anyElem *ElemField) filter.Filter) filter.Filter

	// SameElemMeet requires that the same element satisfies all filters.
	//
	// However, when used with a negation operator ($ne, $not, $nin, $nor), the behavior changes: 'theOne' becomes 'someOne'.
	// That is, if any element in the array meets the negation operator, the document is selected.
	//
	// In other words, under negation, the requirement is equivalent to: "not all elements satisfy the positive operator"
	// (where the positive operator is the counterpart of the negation operator).
	//
	//	op: { dim_cm: { $elemMatch: { $gt: 22, $lt: 30 } } }
	//
	// https://www.mongodb.com/docs/manual/tutorial/query-arrays/#query-for-an-array-element-that-meets-multiple-criteria
	SameElemMeet(f func(theOne ElemField) filter.Filter) filter.Filter

	// PosElemMeet Element at a fixed index must satisfy all filters.
	//
	//	op: { 'dim_cm.1': { $gt: 25 } }
	//
	// https://www.mongodb.com/docs/manual/tutorial/query-arrays/#query-for-an-element-by-the-array-index-position
	//PosElemMeet(pos int, f func(atPosElem *ElemField) filter.Filter) filter.Filter

	// CoverVirValues checks whether the array covers the virtual values specified by the callback function.
	// Coverage is satisfied if a single element covers all VirValues,
	// or if multiple elements collectively cover them (order irrelevant).
	//
	//		op: { tags {
	//	         $all: [
	//	            { "$elemMatch" : { size: "M", num: { $gt: 50} } },
	//	            { "$elemMatch" : { num : 100, color: "green" } }
	//	         ]}
	//	     }
	//
	// eg: Both document
	//
	//	{ tags: [
	//	   { size: "M", num: 100, color: "green" }
	//	] }
	//
	// and document
	//
	//	{ tags: [
	//	   { size: "S", num: 10, color: "blue" },
	//	   { size: "M", num: 100, color: "blue" },
	//	   { size: "L", num: 100, color: "green" }
	//	] }
	//
	// can cover these VirValues
	//
	//	[
	//	  { "$elemMatch" : { size: "M", num: { $gt: 50} } },
	//	  { "$elemMatch" : { num : 100, color: "green" } }
	//	]
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/all/#use--all-with--elemmatch
	CoverVirValues(f func(sameElem ElemField) []VirValue) filter.Filter
}

// ArrayComparableFilter T ~ comparable | EqualAble
type ArrayComparableFilter[T any, ElemField mongodb.Field] interface {
	ArrayBaseFilter[T, ElemField]
	filter.ComparableFilter[[]T]

	// CoverValues checks whether the array covers the given values.
	//
	// op: {tags: { $all: ['red', 'blank'] }}
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/all/#use--all-to-match-values
	CoverValues(values []T) filter.Filter
}

type ArrayBaseUpdater[T any, ElemField mongodb.Field] interface {
	updater.BaseUpdater[[]T]

	PopFirst() updater.Updater
	PopLast() updater.Updater

	// AddEach adds each value of values to an array unless the value is already present,
	// in which case the value isn't added to that array.
	//
	// op: { $addToSet: { tags: { $each: [ "camera", "electronics", "accessories" ] } }
	//
	// document:
	//
	//	{ _id: 2, item: "cable",
	//	  tags: [ "electronics", "supplies" ]
	//	}
	//
	// only adds "camera" and "accessories" to the tags array. "electronics" was already in the array.
	//
	// result:
	//
	//	{
	//	  _id: 2,
	//	  item: "cable",
	//	  tags: [ "electronics", "supplies", "camera", "accessories" ]
	//	}
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/addToSet/#examples
	AddEach(values []T) updater.Updater

	// RemoveVirValue removes all instances that match the specified VirValue from an existing array.
	//
	// op:
	//
	//		 { $pull: { votes: { $gte: 6 } } }
	//	  { $pull: { results: { score: 8 , item: "B" } } }
	//	  { $pull: { results: { answers: { $elemMatch: { q: 2, a: { $gte: 8 } } } } } }
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/pull/#examples
	RemoveVirValue(func(elem ElemField) VirValue) updater.Updater

	// Push appends multiple values to the array field
	//
	// op: { $push: { genres: { $each: [ "Modern Classic", "Award-Winning" ] } } }
	// https://www.mongodb.com/docs/manual/reference/operator/update/push/#append-multiple-values-to-an-array
	Push(values []T) updater.Updater

	// PushWith appends multiple values to the array field with the PushModifier
	//
	//   op: { $push: {
	//         quizzes: {
	//           $each: [ { wk: 5, score: 8 }, { wk: 6, score: 7 }, { wk: 7, score: 6 } ],
	//           $sort: { score: -1 },
	//           $slice: 3
	//         }
	//       } }
	//
	// Operation with modifiers occur in the following order, regardless of the order in which the modifiers appear:
	//
	// 1. Update array to add elements in the correct position.
	//
	// 2. Apply sort, if specified.
	//
	// 3. Slice the array, if specified.
	//
	// 4. Store the array.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/push/#use--push-operator-with-multiple-modifiers
	PushWith(values []T, f func(elem ElemField) updater.PushModifier) updater.Updater
}

// ArrayComparableUpdater T ~ comparable | EqualAble
type ArrayComparableUpdater[T any, ElemField mongodb.Field] interface {
	ArrayBaseUpdater[T, ElemField]

	// RemoveValues removes all instances of the specified values from an existing array.
	//
	// op: { $pullAll: { scores: [ 0, 5 ] } }
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/pullAll/#examples
	RemoveValues(values []T) updater.Updater
}

type ArrayField[T any, ElemField mongodb.Field] interface {
	ArrayBaseField[T, ElemField]
	ArrayBaseFilter[T, ElemField]
	ArrayBaseUpdater[T, ElemField]
	index.BaseKey
}

// ArrayComparableField T ~ comparable | EqualAble
type ArrayComparableField[T any, ElemField mongodb.Field] interface {
	ArrayBaseField[T, ElemField]
	ArrayComparableFilter[T, ElemField]
	ArrayComparableUpdater[T, ElemField]
	index.BaseKey
}

type arrayBaseField[T any, ElemField mongodb.Field] struct {
	BaseField[[]T]
	newElemField func(name string) ElemField
}

func NewArrayField[T any, ElemField mongodb.Field](name string,
	newElem func(name string) ElemField) ArrayField[T, ElemField] {

	return &arrayBaseField[T, ElemField]{BaseField[[]T]{name: name}, newElem}
}

func NewArrayComparableField[T comparable, ElemField mongodb.Field](name string,
	newElem func(name string) ElemField) ArrayComparableField[T, ElemField] {

	return &arrayBaseField[T, ElemField]{BaseField[[]T]{name: name}, newElem}
}

func NewArrayEqualAbleField[T filter.EqualAble[T], ElemField mongodb.Field](name string,
	newElem func(name string) ElemField) ArrayComparableField[T, ElemField] {

	return &arrayBaseField[T, ElemField]{BaseField[[]T]{name: name}, newElem}
}

func NewArrayAnyComparableField[T any, ElemField mongodb.Field](name string,
	newElem func(name string) ElemField) ArrayComparableField[T, ElemField] {

	return &arrayBaseField[T, ElemField]{BaseField[[]T]{name: name}, newElem}
}

func (a *arrayBaseField[T, ElemField]) AtPos(pos int) ElemField {
	return a.newElemField(fmt.Sprintf("%s.%d", a.FullName(), pos))
}

func (a *arrayBaseField[T, ElemField]) Size(sz int) filter.Filter {
	return filter.New(a, `$size`, sz)
}

func (a *arrayBaseField[T, ElemField]) PopFirst() updater.Updater {
	return updater.New(a, `$pop`, -1)
}

func (a *arrayBaseField[T, ElemField]) PopLast() updater.Updater {
	return updater.New(a, `$pop`, 1)
}

func (a *arrayBaseField[T, ElemField]) AddEach(values []T) updater.Updater {
	return updater.New(a, "$addToSet", bson.M{"$each": values})
}

func (a *arrayBaseField[T, ElemField]) RemoveValues(values []T) updater.Updater {
	return updater.New(a, "$pullAll", values)
}

func (a *arrayBaseField[T, ElemField]) RemoveVirValue(f func(sameElem ElemField) VirValue) updater.Updater {
	fil := f(a.newElemField(""))
	return updater.PullByFilter(a, fil)
}

func (a *arrayBaseField[T, ElemField]) Push(values []T) updater.Updater {
	return updater.New(a, "$push", values)
}

func (a *arrayBaseField[T, ElemField]) PushWith(values []T,
	f func(elem ElemField) updater.PushModifier) updater.Updater {

	return updater.PushByModifier(a, f(a.newElemField("")), values)
}

func (a *arrayBaseField[T, ElemField]) AnyElemMeet(f func(anyElem ElemField) filter.Filter) filter.Filter {
	return f(a.newElemField(a.FullName()))
}

func (a *arrayBaseField[T, ElemField]) Elems() ElemField {
	return a.newElemField(a.FullName())
}

func (a *arrayBaseField[T, ElemField]) SameElemMeet(f func(theOne ElemField) filter.Filter) filter.Filter {
	fil := f(a.newElemField(""))
	return filter.SameElemMatch(a, fil)
}

func (a *arrayBaseField[T, ElemField]) CoverValues(values []T) filter.Filter {
	return filter.New(a, "$all", values)
}

func (a *arrayBaseField[T, ElemField]) CoverVirValues(f func(sameElem ElemField) []VirValue) filter.Filter {
	virValues := f(a.newElemField(""))
	return filter.New(a, "$all", virValues)
}

var counter atomic.Uint64

func (a *arrayBaseField[T, ElemField]) AtVirPos(f func(elem ElemField) VirPos) (elem ElemField, arrayFilter ArrayFilter) {
	// The <identifier> must begin with a lowercase letter and contain only alphanumeric characters.
	id := fmt.Sprintf("id%d", counter.Add(1))
	flt := f(a.newElemField(id))
	return a.newElemField(fmt.Sprintf("%s.$[%s]", a.FullName(), id)), flt
}

func (a *arrayBaseField[T, ElemField]) FirstMatched() (elem ElemField) {
	return a.newElemField(fmt.Sprintf("%s.$", a.FullName()))
}

func (a *arrayBaseField[T, ElemField]) UpdateAll() (elem ElemField) {
	return a.newElemField(fmt.Sprintf("%s.$[]", a.FullName()))
}
