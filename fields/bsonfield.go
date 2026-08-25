package fields

import (
	"github.com/xpwu/go-mongodb/field"
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Decimal128Field interface {
	field.Field
	filter.ComparableFilter[bson.Decimal128]
	updater.ComputableUpdater[bson.Decimal128, bson.Decimal128]
	index.BaseKey
}

func NewDecimal128Field(name string) Decimal128Field {
	return &BaseField[bson.Decimal128]{name}
}

type BinaryField interface {
	field.Field
	filter.ComparableFilter[bson.Binary]
	updater.BaseUpdater[bson.Binary]
	index.BaseKey
}

func NewBinaryField(name string) BinaryField {
	return &BaseField[bson.Binary]{name}
}

type ObjectIDField = ComparableField[bson.ObjectID]

var NewObjectIDField = NewComparableField[bson.ObjectID]

type RawField interface {
	field.Field
	filter.BaseFilter[bson.Raw]
	updater.BaseUpdater[bson.Raw]
}

func NewRawField(name string) RawField {
	return &BaseField[bson.Raw]{name}
}

type RawValueField interface {
	field.Field
	filter.BaseFilter[bson.RawValue]
	updater.BaseUpdater[bson.RawValue]
}

func NewRawValueField(name string) RawValueField {
	return &BaseField[bson.RawValue]{name}
}

type RawArrayField interface {
	field.Field
	filter.BaseFilter[bson.RawArray]
	updater.BaseUpdater[bson.RawArray]
}

func NewRawArrayField(name string) RawArrayField {
	return &BaseField[bson.RawArray]{name}
}

type RawElementField interface {
	field.Field
	filter.BaseFilter[bson.RawElement]
	updater.BaseUpdater[bson.RawElement]
}

func NewRawElementField(name string) RawElementField {
	return &BaseField[bson.RawElement]{name}
}

type DateTimeField interface {
	field.Field
	filter.ComparableFilter[bson.DateTime]
	updater.ComputableUpdater[bson.DateTime, bson.DateTime]
}

func NewDateTimeField(name string) DateTimeField {
	return &BaseField[bson.DateTime]{name}
}

type TimestampField interface {
	field.Field
	filter.ComparableFilterField[bson.Timestamp]
	updater.BaseUpdater[bson.Timestamp]
}

func NewTimestampField(name string) TimestampField {
	return &BaseField[bson.Timestamp]{name}
}

type BsonMField interface {
	field.Field
	filter.BaseFilter[bson.M]
	updater.BaseUpdater[bson.M]
}

func NewBsonMField(name string) BsonMField {
	return &BaseField[bson.M]{name}
}

type BsonAField interface {
	field.Field
	filter.BaseFilter[bson.A]
	updater.BaseUpdater[bson.A]
}

func NewBsonAField(name string) BsonAField {
	return &BaseField[bson.A]{name}
}
