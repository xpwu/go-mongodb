package updater

import "github.com/xpwu/go-db-mongo/mongodb"

type BaseUpdaterField[T any] interface {
	mongodb.Field
	BaseUpdater[T]
}

type BaseUpdater[T any] interface {
	// Unset deletes a particular field
	//
	// NOTE THAT: When used with $ to match an array element, $unset replaces the matching element with null rather
	//than removing the matching element from the array. This behavior keeps consistent the array size and
	//element positions.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/unset/#behavior
	Unset() Updater
	// Set replaces the value of a field with the specified value.
	// If the field does not exist, $set adds a new field with the specified value
	// https://www.mongodb.com/docs/manual/reference/operator/update/set/#behavior
	Set(value T) Updater
	// SetOnInsert If an update operation with 'upsert: true' results in an insert of a document,
	//then $setOnInsert assigns the specified values to the fields in the document.
	//If the update operation does not result in an insert, $setOnInsert does nothing.
	//
	// op: { $setOnInsert: { defaultQty: 100 } }
	//
	// When the upsert parameter is true db.collection.updateOne():
	//    1. creates a new document
	//    2. applies the $set operation
	//    3. applies the $setOnInsert operation
	//
	// If db.collection.updateOne() matches an existing document, MongoDB only applies the $set operation.
	// https://www.mongodb.com/docs/manual/reference/operator/update/setOnInsert/#example
	SetOnInsert(value T) Updater
	// SetMin finalValue = min(value, nowValue) or finalValue = value (if nowValue is Not exist).
	// https://www.mongodb.com/docs/manual/reference/operator/update/min/#definition
	SetMin(value T) Updater
	// SetMax finalValue = max(value, nowValue) or finalValue = value (if nowValue is Not exist).
	//
	// https://www.mongodb.com/docs/manual/reference/operator/update/max/
	SetMax(value T) Updater
}
