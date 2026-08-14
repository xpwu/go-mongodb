package fields

import (
	"github.com/xpwu/go-mongodb/geo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"math"
	"reflect"
)

// ==================== 辅助函数 ====================

func bsonDEqual(a, b bson.D) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key {
			return false
		}
		if !bsonValueEqual(a[i].Value, b[i].Value) {
			return false
		}
	}
	return true
}

func bsonMEqual(a, b bson.M) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok || !bsonValueEqual(v, bv) {
			return false
		}
	}
	return true
}

func bsonValueEqual(a, b any) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		switch bv := b.(type) {
		case int:
			return av == bv
		case int64:
			return int64(av) == bv
		}
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && math.Abs(av-bv) < 0.0000000001
	case float32:
		bv, ok := b.(float32)
		return ok && math.Abs(float64(av-bv)) < 0.0000000001
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case bson.A:
		bv, ok := b.(bson.A)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !bsonValueEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case bson.M:
		bv, ok := b.(bson.M)
		return ok && bsonMEqual(av, bv)
	case bson.D:
		bv, ok := b.(bson.D)
		return ok && bsonDEqual(av, bv)
	case bson.Type:
		bv, ok := b.(bson.Type)
		return ok && av == bv
	case bson.ObjectID:
		bv, ok := b.(bson.ObjectID)
		if !ok {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
		return true
	case bson.Binary:
		bv, ok := b.(bson.Binary)
		if !ok || av.Subtype != bv.Subtype || len(av.Data) != len(bv.Data) {
			return false
		}
		for i := range av.Data {
			if av.Data[i] != bv.Data[i] {
				return false
			}
		}
		return true
	case bson.Decimal128:
		bv, ok := b.(bson.Decimal128)
		return ok && av == bv
	case bson.DateTime:
		bv, ok := b.(bson.DateTime)
		return ok && av == bv
	case bson.Timestamp:
		bv, ok := b.(bson.Timestamp)
		return ok && av == bv
	case bson.Regex:
		bv, ok := b.(bson.Regex)
		return ok && av == bv
	case bson.Raw:
		bv, ok := b.(bson.Raw)
		return ok && arrayEqual(av, bv)
	case bson.RawValue:
		bv, ok := b.(bson.RawValue)
		return ok && av.Type == bv.Type && arrayEqual(av.Value, bv.Value)
	case bson.RawArray:
		bv, ok := b.(bson.RawArray)
		return ok && arrayEqual(av, bv)
	case bson.RawElement:
		bv, ok := b.(bson.RawElement)
		return ok && arrayEqual(av, bv)
	case geo.Polygon:
		bv, ok := b.(geo.Polygon)
		return ok && av.Type == bv.Type && arrayEqual(av.C, bv.C)
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Array:
		return reflect.Array == reflect.TypeOf(b).Kind() && reflect.DeepEqual(a, b)
	case reflect.Slice:
		return reflect.Slice == reflect.TypeOf(b).Kind() && reflect.DeepEqual(a, b)
	case reflect.Struct:
		return reflect.Struct == reflect.TypeOf(b).Kind() && reflect.DeepEqual(a, b)
	}
	return a == b
}

func arrayEqual[T any](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i, aa := range a {
		if !bsonValueEqual(aa, b[i]) {
			return false
		}
	}

	return true
}
