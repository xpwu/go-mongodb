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

func flattenDoc(doc bson.D) bson.D {
	var out bson.D
	for _, e := range doc {
		switch v := e.Value.(type) {
		case bson.D:
			out = append(out, bson.E{
				Key:   e.Key,
				Value: flattenDoc(v),
			})
		case bson.A:
			if e.Key == "$and" {
				arr := flattenAnd(v)
				// 关键：允许空 $and 透传
				if len(arr) == 0 {
					out = append(out, bson.E{Key: "$and", Value: bson.A{}})
					continue
				}
				for _, item := range arr {
					// 必须断言成 bson.E
					if elem, ok := item.(bson.E); ok {
						out = append(out, elem)
					}
				}
			} else if e.Key == "$or" || e.Key == "$nor" {
				newArr := make(bson.A, 0, len(v))
				for _, item := range v {
					if d, ok := item.(bson.D); ok {
						newArr = append(newArr, flattenDoc(d))
					} else {
						newArr = append(newArr, item)
					}
				}
				out = append(out, bson.E{Key: e.Key, Value: newArr})
			} else {
				out = append(out, e)
			}
		default:
			out = append(out, e)
		}
	}
	return out
}

func flattenAnd(arr bson.A) bson.A {
	type kind int
	const (
		kEmpty kind = iota
		kPure
		kExpr
		kConflict
	)

	type field struct {
		kind  kind
		value interface{}
		expr  bson.D
	}

	fields := make(map[string]*field)
	var keep []interface{}

	// ---------- 1. 展开嵌套 $and ----------
	var docs []interface{}
	hasInvalid := false
	for _, item := range arr {
		d, ok := item.(bson.D)
		//  非 bson.D，直接进 keep，原样保留
		if !ok {
			//  遇到非 bson.D，标记并停止拆分
			hasInvalid = true
			keep = append(keep, item)
			continue
		}

		if len(d) != 1 {
			hasInvalid = true
			docs = append(docs, item)
			continue
		}

		if d[0].Key == "$and" {
			sub := flattenAnd(d[0].Value.(bson.A))
			for _, elem := range sub {
				if e, ok := elem.(bson.E); ok {
					docs = append(docs, bson.D{e})
				}
			}
			continue
		}

		//  如果已经遇到非法条件，后续所有条件都不能平铺
		if hasInvalid {
			keep = append(keep, d)
			continue
		}

		docs = append(docs, d)
	}

	// ---------- 2. 逐条处理 ----------
	for _, item := range docs {
		d, ok := item.(bson.D)
		if !ok || len(d) != 1 {
			keep = append(keep, item)
			continue
		}

		key := d[0].Key
		val := d[0].Value

		f, ok := fields[key]
		if !ok {
			f = &field{}
			fields[key] = f
		}

		if f.kind == kConflict {
			keep = append(keep, d)
			continue
		}

		// -------- operator --------
		if expr, ok := val.(bson.D); ok {
			if f.kind == kPure {
				keep = append(keep, bson.D{{key, f.value}})
				keep = append(keep, d)
				f.kind = kConflict
				continue
			}

			if f.kind == kExpr {
				conflict := false
				for _, e := range expr {
					for _, ex := range f.expr {
						if e.Key == ex.Key {
							conflict = true
							break
						}
					}
				}
				if conflict {
					keep = append(keep, bson.D{{key, f.expr}})
					keep = append(keep, d)
					f.kind = kConflict
					continue
				}
				f.expr = append(f.expr, expr...)
				continue
			}

			f.kind = kExpr
			f.expr = expr
			continue
		}

		// -------- pure value --------
		if f.kind == kExpr {
			keep = append(keep, bson.D{{key, f.expr}})
			keep = append(keep, d)
			f.kind = kConflict
			continue
		}

		if f.kind == kPure {
			prev := bson.D{{key, f.value}}
			if !bsonDocEqual(prev, d) {
				keep = append(keep, prev)
				keep = append(keep, d)
				f.kind = kConflict
				continue
			}
			continue
		}

		f.kind = kPure
		f.value = val
	}

	// ---------- 3. 重建 result ----------
	var result bson.A
	for name, f := range fields {
		if f.kind == kConflict || f.kind == kEmpty {
			continue
		}
		if f.kind == kPure {
			result = append(result, bson.E{Key: name, Value: f.value})
			continue
		}
		if f.kind == kExpr {
			result = append(result, bson.E{Key: name, Value: f.expr})
		}
	}

	// ---------- 4. 处理 keep ----------
	if len(keep) > 0 {
		var realKeep []interface{}
		realKeep = append(realKeep, keep...)

		if len(realKeep) == 0 {
			return bson.A{bson.E{Key: "$and", Value: bson.A{}}}
		}
		and := bson.E{Key: "$and", Value: toBsonA(realKeep)}
		if len(result) == 0 {
			return bson.A{and}
		}
		result = append(result, and)
	}

	return result
}

func toBsonA(docs []interface{}) bson.A {
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

type logic struct {
	operator string
	filters  []Filter
}

func (l *logic) ToBsonD() bson.D {

	var value bson.A

	for _, filter := range l.filters {
		b, ok := filter.(*logic)
		if !ok || b.operator != l.operator {
			value = append(value, filter.ToBsonD())
			continue
		}

		// merge operator
		for _, filter := range b.filters {
			value = append(value, filter.ToBsonD())
		}
	}

	return bson.D{{l.operator, value}}
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

func newLogic(operator string, filter1, filter2 Filter, filters ...Filter) *logic {
	f := make([]Filter, 2, 2+len(filters))
	f[0] = filter1
	f[1] = filter2
	f = append(f, filters...)

	return &logic{
		operator: operator,
		filters:  f,
	}
}

type nor struct {
	filters []Filter
}

func (l *nor) ToBsonD() bson.D {

	var value []bson.D

	for _, filter := range l.filters {
		value = append(value, filter.ToBsonD())
	}

	return bson.D{{`$nor`, value}}
}

func newNor(filter1, filter2 Filter, filters ...Filter) *nor {
	f := make([]Filter, 2, 2+len(filters))
	f[0] = filter1
	f[1] = filter2
	f = append(f, filters...)

	return &nor{
		filters: f,
	}
}
