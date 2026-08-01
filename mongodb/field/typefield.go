package field

import (
	"fmt"
	"github.com/xpwu/go-db-mongo/mongodb"
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/index"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type IntegerFilter[T Integer] interface {
	filter.ComparableFilter[T]
	Mod(divisor, remainder T) filter.Filter
}

type IntegerField[T Integer] interface {
	mongodb.Field
	IntegerFilter[T]
	updater.ComputableUpdater[T, T]
	index.BaseKey
}

func (b *BaseField[T]) Mod(divisor, remainder T) filter.Filter {
	return filter.New(b, "$mod", bson.A{divisor, remainder})
}

func NewIntegerField[T Integer](name string) IntegerField[T] {
	return &BaseField[T]{name}
}

type IntField = IntegerField[int]
type Int8Field = IntegerField[int8]
type Int16Field = IntegerField[int16]
type Int32Field = IntegerField[int32]
type Int64Field = IntegerField[int64]

var (
	NewIntField   = NewIntegerField[int]
	NewInt8Field  = NewIntegerField[int8]
	NewInt16Field = NewIntegerField[int16]
	NewInt32Field = NewIntegerField[int32]
	NewInt64Field = NewIntegerField[int64]
)

type UnInteger interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type UnIntegerField[T UnInteger, VT Integer] interface {
	mongodb.Field
	IntegerFilter[T]
	updater.ComputableUpdater[T, VT]
	index.BaseKey
}

type unIntegerField[T UnInteger, VT Integer] struct {
	BaseField[T]
}

func (u *unIntegerField[T, VT]) Inc(num VT) updater.Updater {
	return updater.New(u, "$inc", num)
}

func NewUnIntegerField[T UnInteger, VT Integer](name string) UnIntegerField[T, VT] {
	return &unIntegerField[T, VT]{BaseField[T]{name}}
}

type UintField = UnIntegerField[uint, int]
type ByteField = UnIntegerField[byte, int8]
type Uint8Field = UnIntegerField[uint8, int8]
type Uint16Field = UnIntegerField[uint16, int16]
type Uint32Field = UnIntegerField[uint32, int32]
type Uint64Field = UnIntegerField[uint64, int64]

var (
	NewUintField   = NewUnIntegerField[uint, int]
	NewByteField   = NewUnIntegerField[byte, int8]
	NewUint8Field  = NewUnIntegerField[uint8, int8]
	NewUint16Field = NewUnIntegerField[uint16, int16]
	NewUint32Field = NewUnIntegerField[uint32, int32]
	NewUint64Field = NewUnIntegerField[uint64, int64]
)

type StringFilter interface {
	Regex(regex bson.Regex) filter.Filter
	filter.ComparableFilter[string]
}

type StringField interface {
	mongodb.Field
	StringFilter
	updater.BaseUpdater[string]
	index.BaseKey
}

type stringField struct {
	BaseField[string]
}

func (s *stringField) Regex(regex bson.Regex) filter.Filter {
	return filter.New(s, "$regex", regex)
}

func NewStringField(name string) StringField {
	return &stringField{BaseField[string]{name}}
}

type ComparableField[T ~bool | bson.ObjectID] interface {
	mongodb.Field
	filter.ComparableFilter[T]
	updater.BaseUpdater[T]
	index.BaseKey
}

func NewComparableField[T ~bool | bson.ObjectID](name string) ComparableField[T] {
	return &BaseField[T]{name}
}

type BoolField = ComparableField[bool]
type ObjectIDField = ComparableField[bson.ObjectID]

var (
	NewBoolField     = NewComparableField[bool]
	NewObjectIDField = NewComparableField[bson.ObjectID]
)

type ComputableField[T ~float32 | ~float64] interface {
	mongodb.Field
	filter.BaseFilter[T]
	updater.ComputableUpdater[T, T]
	index.BaseKey
}

func NewComputableField[T ~float32 | ~float64](name string) ComputableField[T] {
	return &BaseField[T]{name}
}

type Float32Field = ComputableField[float32]
type Float64Field = ComputableField[float64]

var (
	NewFloat32Field = NewComputableField[float32]
	NewFloat64Field = NewComputableField[float64]
)

type Decimal128Field interface {
	mongodb.Field
	filter.ComparableFilter[bson.Decimal128]
	updater.ComputableUpdater[bson.Decimal128, bson.Decimal128]
	index.BaseKey
}

func NewDecimal128Field(name string) Decimal128Field {
	return &BaseField[bson.Decimal128]{name}
}

type BinaryField interface {
	mongodb.Field
	filter.ComparableFilter[bson.Binary]
	updater.BaseUpdater[bson.Binary]
	index.BaseKey
}

func NewBinaryField(name string) BinaryField {
	return &BaseField[bson.Binary]{name}
}

func SubField(selfName, fieldName string) string {
	if selfName == "" {
		return fieldName
	}
	if fieldName == "" {
		return selfName
	}

	return fmt.Sprintf("%s.%s", selfName, fieldName)
}

type BaseStructField[T any] interface {
	mongodb.Field
	filter.BaseFilter[T]
	updater.BaseUpdater[T]
}

type ComparableStructField[T any] interface {
	mongodb.Field
	filter.ComparableFilter[T]
	updater.BaseUpdater[T]
}
