package filter

import "go.mongodb.org/mongo-driver/v2/bson"

func hasOperator2(d bson.D, op string) bool {
	for _, e := range d {
		if e.Key == op {
			return true
		}
	}
	return false
}

func getField(d bson.D, key string) (interface{}, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

func hasOperator(d bson.D, field, op string) bool {
	for _, e := range d {
		if e.Key != field {
			continue
		}
		sub, ok := e.Value.(bson.D)
		if !ok {
			return false
		}
		for _, se := range sub {
			if se.Key == op {
				return true
			}
		}
	}
	return false
}

func hasKey(d bson.D, key string) bool {
	for _, e := range d {
		if e.Key == key {
			return true
		}
	}
	return false
}

func extractOr(d bson.D) bson.A {
	for _, e := range d {
		if e.Key == "$or" {
			if arr, ok := e.Value.(bson.A); ok {
				return arr
			}
		}
	}
	return nil
}

func extractAnd(d bson.D) bson.A {
	for _, e := range d {
		if e.Key == "$and" {
			if arr, ok := e.Value.(bson.A); ok {
				return arr
			}
		}
	}
	return nil
}

func toBSON(d bson.D) string {
	raw, _ := bson.Marshal(d)
	var out bson.D
	_ = bson.Unmarshal(raw, &out)
	bs, _ := bson.MarshalExtJSON(out, false, false)
	return string(bs)
}
