package filter

import (
	"bytes"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// filter: <field>: { <operator>: <value> }
// filter: <field>: <value>
// filter: field: { $not: { <operator>: <value> } }
// filter: $and: [filter]  /   {{field1: value1}, {field2:{$op2, value2}}, {field3:{$op3, value3}}}
// filter: $or: [filter]
// filter: $nor: [filter]

func flattenDoc(docs bson.D) bson.D {
	if len(docs) == 0 {
		return docs
	}

	// 隐式 $and
	if len(docs) != 1 {
		return flattenAnd(bson.A{docs})
	}

	key := docs[0].Key
	val := docs[0].Value

	switch v := val.(type) {
	case bson.D:
		return bson.D{{key, flattenDoc(v)}}
	case bson.A:
		switch key {
		case "$and":
			return flattenAnd(v)
		case "$or":
			return flattenOr(v)
		case "$nor":
			return flattenNor(v)
		}
	}

	return docs
}

// flattenAnd   bson.A{bson.D{}}  ==> 1、bson.D{bson.E{field, value}, bson.E{"$or", bson.A{xxx}}, ...}; 2、bson.D{bson.E{"$and", bson.A{xxx}}}
func flattenAnd(arr bson.A) bson.D {
	if len(arr) == 0 {
		return bson.D{{"$and", bson.A{}}}
	}

	keeps := bson.A{}

	// ---------- 1. 展开嵌套 $and ----------
	var docs []bson.E
	for _, item := range arr {
		d, ok := item.(bson.D)
		//  非 bson.D，直接进 keep，原样保留
		if !ok {
			keeps = append(keeps, item)
			continue
		}

		// d 的所有 bson.E 本就是隐式 $and
		for _, e := range d {
			if e.Key == "$and" {
				sub := flattenAnd(e.Value.(bson.A))
				for _, elem := range sub {
					docs = append(docs, elem)
				}
				continue
			}
			if e.Key == "$or" {
				orResult := flattenOr(e.Value.(bson.A))
				if len(orResult) == 1 {
					docs = append(docs, orResult[0])
				}
				continue
			}
			if e.Key == "$not" {
				norResult := flattenNor(e.Value.(bson.A))
				if len(norResult) == 1 {
					docs = append(docs, norResult[0])
				}
				continue
			}
			if d, ok := e.Value.(bson.D); ok {
				docs = append(docs, bson.E{
					Key:   e.Key,
					Value: flattenDoc(d),
				})
				continue
			}
			docs = append(docs, e)
		}
	}

	// ---------- 2. 逐条处理 ----------
	// 能在 $op 级别合并为一个bson.D 就尽可能的合并 $op, 减少 field 的数量
	// 能在 field 级别合并为一个bson.D 就合并，减少 bson.A 的数量，甚至没有bson.A
	type kind int
	const (
		kPure kind = iota
		kExpr
	)
	type field struct {
		kind  kind
		value interface{}
		expr  bson.D
	}
	fields := make([]map[string]*field, 0)

	for _, item := range docs {
		key := item.Key
		val := item.Value

		var valField field
		if d, ok := val.(bson.D); ok {
			valField.kind = kExpr
			valField.expr = d
		} else {
			valField.kind = kPure
			valField.value = val
		}

		placed := false
		for _, fm := range fields {
			f, ok := fm[key]
			if !ok {
				fm[key] = &valField
				placed = true
				break
			}

			if valField.kind == kPure && f.kind == kPure {
				if bsonDocEqual(bson.D{{key, valField.value}}, bson.D{{key, f.value}}) {
					placed = true
					break
				}
				continue
			}

			// merge $op
			var existOperators bson.D
			var newOperator bson.D
			if f.kind != valField.kind {
				if f.kind == kPure {
					existOperators = append(existOperators, bson.E{Key: "$eq", Value: f.value})
					newOperator = valField.expr
				} else {
					existOperators = f.expr
					newOperator = append(newOperator, bson.E{Key: "$eq", Value: valField.value})
				}
			} else {
				existOperators = f.expr
				newOperator = valField.expr
			}

			conflict := false
		conflictFlag:
			for _, e := range existOperators {
				for _, newOp := range newOperator {
					if e.Key == newOp.Key {
						conflict = true
						break conflictFlag
					}
				}
			}
			if conflict {
				continue
			}

			existOperators = append(existOperators, newOperator...)
			f.kind = kExpr
			f.expr = existOperators

			placed = true
			break
		}
		if !placed {
			f := make(map[string]*field)
			f[key] = &valField
			fields = append(fields, f)
		}
	}

	// ---------- 3. 重建 result ----------
	var result []bson.D
	for _, fs := range fields {
		var d bson.D
		for name, f := range fs {
			var val any
			if f.kind == kPure {
				val = f.value
			} else {
				val = f.expr
			}
			d = append(d, bson.E{Key: name, Value: val})
		}
		result = append(result, d)
	}

	if len(result) == 1 && len(keeps) == 0 {
		return result[0]
	}
	keeps = append(keeps, toBsonA(result)...)

	return bson.D{{"$and", keeps}}
}

// flattenOr   bson.A{bson.D{}}  ==> bson.D{bson.E{"$or", bson.A{...}}}
func flattenOr(arr bson.A) bson.D {
	if len(arr) == 0 {
		return bson.D{{"$or", bson.A{}}}
	}

	var keeps bson.A
	var out []bson.D
	for _, item := range arr {
		d, ok := item.(bson.D)
		if !ok {
			keeps = append(keeps, item)
			continue
		}
		// 不止一个 bson.E, 隐式 $and
		if len(d) != 1 {
			andRes := flattenAnd(bson.A{d})
			out = append(out, andRes)
			continue
		}

		key := d[0].Key
		val := d[0].Value

		if v, ok := val.(bson.A); ok {
			switch key {
			case "$or":
				// 嵌套 $or：展开
				sub := flattenOr(v)
				// 不会出现，但是防止有bug引起panic，加了这个逻辑
				if len(sub) != 1 || sub[0].Key != "$or" {
					out = append(out, d)
					continue
				}
				orRes, ok := sub[0].Value.(bson.A)
				if !ok {
					out = append(out, d)
					continue
				}

				for _, orRes1 := range orRes {
					if d, ok := orRes1.(bson.D); ok {
						out = append(out, d)
					} else {
						keeps = append(keeps, orRes1)
					}
				}
				continue

			case "$and":
				andRes := flattenAnd(v)
				out = append(out, andRes)
				continue

			case "$nor":
				norRes := flattenNor(v)
				out = append(out, norRes)
				continue

				// falling through: val.(bson.A) but key is else
			}
		}

		// 普通条件：递归 flattenDoc
		if v, ok := val.(bson.D); ok {
			out = append(out, bson.D{{key, flattenDoc(v)}})
			continue
		}

		out = append(out, d)
	}

	return bson.D{{"$or", append(keeps, toBsonA(out)...)}}
}

// flattenNor   bson.A{bson.D{}}  ==> bson.D{bson.E{"$nor", bson.A{...}}}
func flattenNor(arr bson.A) bson.D {
	if len(arr) == 0 {
		return bson.D{{"$nor", bson.A{}}}
	}

	var keeps bson.A
	var out []bson.D
	for _, item := range arr {
		d, ok := item.(bson.D)
		if !ok {
			keeps = append(keeps, item)
			continue
		}
		// 不止一个 bson.E, 隐式 $and
		if len(d) != 1 {
			andRes := flattenAnd(bson.A{d})
			out = append(out, andRes)
			continue
		}

		key := d[0].Key
		val := d[0].Value

		if v, ok := val.(bson.A); ok {
			switch key {
			case "$and":
				andRes := flattenAnd(v)
				out = append(out, andRes)
				continue

			case "$or":
				orRes := flattenOr(v)
				out = append(out, orRes)
				continue

			// 嵌套 $nor：也不能展开，意义会变，类似双重否定
			// 只能对 [] 中的每个 bson.D 递归 flattenDoc
			case "$nor":
				ret := bson.A{}
				for _, vv := range v {
					if d, ok := vv.(bson.D); ok {
						ret = append(ret, flattenDoc(d))
					} else {
						ret = append(ret, vv)
					}
				}
				out = append(out, bson.D{{"$nor", ret}})
				continue

				// falling through: val.(bson.A) but key is else
			}
		}

		// 普通条件：递归 flattenDoc
		if v, ok := val.(bson.D); ok {
			out = append(out, bson.D{{key, flattenDoc(v)}})
			continue
		}

		out = append(out, d)
	}

	return bson.D{{"$nor", append(keeps, toBsonA(out)...)}}
}

func toBsonA[T any](docs []T) bson.A {
	a := make(bson.A, len(docs))
	for i, d := range docs {
		a[i] = d
	}
	return a
}

func bsonDocEqual(a, b bson.D) bool {
	ba, err := bson.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := bson.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ba, bb)
}

type and struct {
	filters []Filter
}

func (a *and) ToBsonD() bson.D {
	val := make(bson.A, 0, len(a.filters))
	for _, f := range a.filters {
		val = append(val, f.ToBsonD())
	}
	return flattenDoc(bson.D{{"$and", val}})
}

func And(filters ...Filter) Filter {
	return &and{filters: filters}
}

func AndPartial(filters ...PartialIndexFilter) PartialIndexFilter {
	fil := make([]Filter, len(filters))
	for i, a := range filters {
		fil[i] = a
	}
	return AsPartialIndexFilter(&and{filters: fil})
}

type or struct {
	filters []Filter
}

func (a *or) ToBsonD() bson.D {
	val := make(bson.A, 0, len(a.filters))
	for _, f := range a.filters {
		val = append(val, f.ToBsonD())
	}
	return flattenDoc(bson.D{{"$or", val}})
}

func Or(filters ...Filter) Filter {
	return &or{filters: filters}
}

func OrPartial(filter1, filter2 PartialIndexFilter, filters ...PartialIndexFilter) PartialIndexFilter {
	fil := make([]Filter, len(filters))
	for i, a := range filters {
		fil[i] = a
	}
	return AsPartialIndexFilter(&or{filters: fil})
}

type nor struct {
	filters []Filter
}

func (l *nor) ToBsonD() bson.D {

	val := make(bson.A, 0, len(l.filters))
	for _, f := range l.filters {
		val = append(val, f.ToBsonD())
	}
	return flattenDoc(bson.D{{"$nor", val}})
}

// Nor selects the documents that fail all the query predicates in the array,
// including those documents that do not contain these field(s).
//
// NOTE THAT: The exception in returning documents that do not contain the field in the $nor expression
// is when the $nor operator is used with the $exists operator.
// https://www.mongodb.com/docs/manual/reference/operator/query/nor/#-nor-and--exists
func Nor(filter1, filter2 Filter, filters ...Filter) Filter {
	return &nor{filters: filters}
}
