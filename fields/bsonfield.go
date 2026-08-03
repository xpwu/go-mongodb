package fields

import (
	"github.com/xpwu/go-mongodb/field"
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	updater2 "github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Decimal128Field interface {
	field.Field
	filter.ComparableFilter[bson.Decimal128]
	updater2.ComputableUpdater[bson.Decimal128, bson.Decimal128]
	index.BaseKey
}

func NewDecimal128Field(name string) Decimal128Field {
	return &BaseField[bson.Decimal128]{name}
}

type BinaryField interface {
	field.Field
	filter.ComparableFilter[bson.Binary]
	updater2.BaseUpdater[bson.Binary]
	index.BaseKey
}

func NewBinaryField(name string) BinaryField {
	return &BaseField[bson.Binary]{name}
}

type ObjectIDField = ComparableField[bson.ObjectID]

var NewObjectIDField = NewComparableField[bson.ObjectID]

// todo Raw RawValue
