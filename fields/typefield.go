package fields

import (
	"fmt"
	"github.com/xpwu/go-mongodb/field"
	filter2 "github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	updater2 "github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type IntegerFilter[T Integer] interface {
	filter2.ComparableFilter[T]
	Mod(divisor, remainder T) filter2.Filter
}

type IntegerField[T Integer] interface {
	field.Field
	IntegerFilter[T]
	updater2.ComputableUpdater[T, T]
	index.BaseKey
}

func (b *BaseField[T]) Mod(divisor, remainder T) filter2.Filter {
	return filter2.New(b, "$mod", bson.A{divisor, remainder})
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
	field.Field
	IntegerFilter[T]
	updater2.ComputableUpdater[T, VT]
	index.BaseKey
}

type unIntegerField[T UnInteger, VT Integer] struct {
	BaseField[T]
}

func (u *unIntegerField[T, VT]) Inc(num VT) updater2.Updater {
	return updater2.New(u, "$inc", num)
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
	Regex(regex bson.Regex) filter2.Filter
	filter2.ComparableFilter[string]
}

type StringField interface {
	field.Field
	StringFilter
	updater2.BaseUpdater[string]
	index.BaseKey
}

type stringField struct {
	BaseField[string]
}

func (s *stringField) Regex(regex bson.Regex) filter2.Filter {
	return filter2.New(s, "$regex", regex)
}

func NewStringField(name string) StringField {
	return &stringField{BaseField[string]{name}}
}

type ComparableField[T ~bool | bson.ObjectID] interface {
	field.Field
	filter2.ComparableFilter[T]
	updater2.BaseUpdater[T]
	index.BaseKey
}

func NewComparableField[T ~bool | bson.ObjectID](name string) ComparableField[T] {
	return &BaseField[T]{name}
}

type BoolField = ComparableField[bool]

var NewBoolField = NewComparableField[bool]

type ComputableField[T ~float32 | ~float64] interface {
	field.Field
	filter2.BaseFilter[T]
	updater2.ComputableUpdater[T, T]
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
	field.Field
	filter2.BaseFilter[T]
	updater2.BaseUpdater[T]
}

type ComparableStructField[T any] interface {
	field.Field
	filter2.ComparableFilter[T]
	updater2.BaseUpdater[T]
}
