package updater

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Computable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
	~float32 | ~float64 | bson.Decimal128
}

type ComputableUpdaterField[T Computable, VT Computable] interface {
	mongodb.Field
	ComputableUpdater[T, VT]
}

type ComputableUpdater[T Computable, VT Computable] interface {
	BaseUpdater[T]
	// Inc finalValue = T(nowValue + num) or finalValue = value (if nowValue is Not exist)
	Inc(num VT) Updater
	// Mul finalValue = nowValue * num or finalValue = 0 (if nowValue is Not exist)
	Mul(num T) Updater
}
