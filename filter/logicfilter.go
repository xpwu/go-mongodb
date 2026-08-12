package filter

import (
	"bytes"
	"go.mongodb.org/mongo-driver/v2/bson"
	"strings"
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

func extractAndToEs(elem bson.E) (ret []bson.E, ok bool) {
	if elem.Key != "$and" {
		return nil, false
	}
	a, ok := elem.Value.(bson.A)
	if !ok {
		return nil, false
	}
	for _, a1 := range a {
		d, ok := a1.(bson.D)
		if !ok {
			return nil, false
		}
		for _, e := range d {
			ret = append(ret, e)
		}
	}

	return ret, true
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
					if elem.Key == "$and" {
						if es, ok := extractAndToEs(elem); ok {
							docs = append(docs, es...)
						} else {
							keeps = append(keeps, bson.D{elem})
						}
					} else {
						docs = append(docs, elem)
					}
				}
				continue
			}
			if e.Key == "$or" {
				orResult := flattenOr(e.Value.(bson.A))
				if len(orResult) == 1 {
					docs = append(docs, orResult[0])
				}
				// todo len(orResult) !=1 ? 正常情况不会出现，是否需要防御性代码
				continue
			}
			if e.Key == "$nor" {
				norResult := flattenNor(e.Value.(bson.A))
				if len(norResult) == 1 {
					docs = append(docs, norResult[0])
				}
				// todo len(norResult) !=1 ? 正常情况不会出现，是否需要防御性代码
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

			// $or  $nor 等操作符 是不能合并的，否则不满足逻辑本身
			// 为了 bson.A 尽可能的少，分散到不同的 fields
			// 之所以没有直接加入 keeps，是因为如果只有一个 $or 或者 $nor 就省去了 $and
			if strings.HasPrefix(key, "$") {
				continue
			}

			if valField.kind == kPure && f.kind == kPure {
				if bsonDocEqual(bson.D{{key, valField.value}}, bson.D{{key, f.value}}) {
					placed = true
					break
				}
				_, fok := f.value.(bson.Regex)
				_, vok := valField.value.(bson.Regex)

				if fok == vok {
					continue
				}
				// falling through: kind = kPure && the one is $regex, the other is $eq
			}

			// merge $op
			var existOperators bson.D
			var newOperator bson.D
			if f.kind != valField.kind {
				if f.kind == kPure {
					if r, ok := f.value.(bson.Regex); ok {
						existOperators = append(existOperators, bson.E{Key: "$regex", Value: r})
					} else {
						existOperators = append(existOperators, bson.E{Key: "$eq", Value: f.value})
					}
					newOperator = valField.expr
				} else {
					existOperators = f.expr
					if r, ok := valField.value.(bson.Regex); ok {
						newOperator = append(newOperator, bson.E{Key: "$regex", Value: r})
					} else {
						newOperator = append(newOperator, bson.E{Key: "$eq", Value: valField.value})
					}
				}
			} else if f.kind == kExpr {
				existOperators = f.expr
				newOperator = valField.expr
			} else {
				// f.kind == valField.kind == kPure && the one is $regex, the other is $eq
				if fr, fok := f.value.(bson.Regex); fok {
					existOperators = append(existOperators, bson.E{Key: "$regex", Value: fr})
					newOperator = append(newOperator, bson.E{Key: "$eq", Value: valField.value})
				} else {
					existOperators = append(existOperators, bson.E{Key: "$eq", Value: f.value})
					newOperator = append(newOperator, bson.E{Key: "$regex", Value: valField.value})
				}
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

func OrPartial(filters ...PartialIndexFilter) PartialIndexFilter {
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
func Nor(filters ...Filter) Filter {
	return &nor{filters: filters}
}

func runNot(inner bson.D) bson.D {
	if len(inner) == 0 {
		return bson.D{{"$not", bson.D{}}}
	}

	// 隐式 $and（多字段）
	if len(inner) != 1 {
		// Not({a:1, b:{$gt:2}}) => {$nor: [{a:1, b:{$gt:2}}]}
		return flattenNor(bson.A{inner})
	}

	key := inner[0].Key
	val := inner[0].Value

	if arr, ok := val.(bson.A); ok {
		switch key {
		case "$and":
			// Not($and:[f1, f2])
			// 德摩根：¬(f1 ∧ f2) = ¬f1 ∨ ¬f2
			// => {$or: [{field1: {$not: f1_val}}, {field2: {$not: f2_val}}]}
			var clauses []bson.D
			for _, item := range arr {
				if d, ok := item.(bson.D); ok {
					clauses = append(clauses, runNot(d))
				} else {
					clauses = append(clauses, bson.D{{"$not", item}})
				}
			}
			return flattenOr(toBsonA(clauses))

		case "$or":
			// Not($or:[f1, f2])
			// 德摩根：¬(f1 ∨ f2) = ¬f1 ∧ ¬f2
			// => {$and: [{field1: {$not: f1_val}}, {field2: {$not: f2_val}}]}
			var clauses []bson.D
			for _, item := range arr {
				if d, ok := item.(bson.D); ok {
					clauses = append(clauses, runNot(d))
				} else {
					clauses = append(clauses, bson.D{{"$not", item}})
				}
			}
			return flattenAnd(toBsonA(clauses))

		case "$nor":
			// Not($nor:[f1, f2])
			// $nor = ¬f1 ∧ ¬f2，所以 ¬(¬f1 ∧ ¬f2) = f1 ∨ f2
			// => {$or: [f1, f2]}
			return flattenOr(arr)

			// falling through: val.(bson.A) but key is else
		}
	}

	// 普通字段：{field: {$not: condition}}
	// 例如 Not(Gt("a", 5)) => {a: {$not: {$gt: 5}}}
	if d, ok := val.(bson.D); ok {
		if len(d) == 0 {
			return bson.D{{key, bson.D{{"$not", bson.D{}}}}}
		}
		if len(d) == 1 {
			return bson.D{{key, notOp(d[0])}}
		}
		// { a: { $gt: 5, $lt: 7 } } => {"$and":[{a:{$gt:5}}, {a:{$lt:7}]}
		var clauses []bson.D
		for _, dn := range d {
			clauses = append(clauses, bson.D{{key, bson.D{dn}}})
		}
		return runNot(bson.D{{"$and", toBsonA(clauses)}})
	}

	// 普通字段：{field: bson.Regex{}}
	// 例如 Not({field: bson.Regex{}}) => {a: {$not: {$regex: bson.Regex{}}}}
	if r, ok := val.(bson.Regex); ok {
		return bson.D{{key, bson.D{{"$not", bson.D{{"$regex", r}}}}}}
	}

	// 普通字段：{field: value} or {field: []values}
	// Not({field: value}) => {field: {"$ne": value}}
	return bson.D{{key, bson.D{{"$ne", val}}}}
}

// $not 取反等价替换规则：
// $eq  ↔  $ne    // {a: {$not: {$eq: 5}}}    => {a: {$ne: 5}}
// $ne  ↔  $eq    // {a: {$not: {$ne: 5}}}    => {a: {$eq: 5}}
// $in  ↔  $nin   // {a: {$not: {$in: [1,2]}}} => {a: {$nin: [1,2]}}
// $nin ↔  $in    // {a: {$not: {$nin: [1,2]}}} => {a: {$in: [1,2]}}
// $exists: true  ↔  $exists: false
// $exists: false ↔  $exists: true
//
// 以下不能直接替换（字段不存在的语义不同）：
// $gt  → $lte   (不等价)
// $gte → $lt    (不等价)
// $lt  → $gte   (不等价)
// $lte → $gt    (不等价)
func notOp(op bson.E) bson.D {
	// 已经是 $not，双重否定消除：Not(Not(f)) => f
	if op.Key == "$not" {
		if d, ok := op.Value.(bson.D); ok {
			return flattenDoc(d)
		}
	}
	switch op.Key {
	case "$eq":
		return bson.D{{"$ne", op.Value}}
	case "$ne":
		return bson.D{{"$eq", op.Value}}
	case "$in":
		return bson.D{{"$nin", op.Value}}
	case "$nin":
		return bson.D{{"$in", op.Value}}
	case "$exists":
		if b, ok := op.Value.(bool); ok {
			return bson.D{{"$exists", !b}}
		}
	}

	return bson.D{{"$not", bson.D{op}}}
}

type not struct {
	filter Filter
}

func (n *not) ToBsonD() bson.D {
	return runNot(n.filter.ToBsonD())
}

// Not returns a Filter that negates the given filter.
//
// 1. that do not match the <operator-expression>.
//
// 2.This includes documents that do not contain the field
//
// https://www.mongodb.com/docs/manual/reference/operator/query/not/#mongodb-query-op.-not
//
// Conversion rules (De Morgan's laws):
//
//   - Not(single field condition)  => {field: {$not: condition}}
//     e.g. Not(Gt("a", 5))         => {a: {$not: {$gt: 5}}}
//
//   - Not(And(f1, f2))             => {$or: [{f1_field: {$not: f1_val}}, {f2_field: {$not: f2_val}}]}
//     ¬(f1 ∧ f2) = ¬f1 ∨ ¬f2
//
//   - Not(Or(f1, f2))              => {$and: [{f1_field: {$not: f1_val}}, {f2_field: {$not: f2_val}}]}
//     ¬(f1 ∨ f2) = ¬f1 ∧ ¬f2
//
//   - Not(Nor(f1, f2))             => {$or: [f1, f2]}
//     ¬(¬f1 ∧ ¬f2) = f1 ∨ f2
//
//   - Not(Not(f))                  => f (double negation elimination)
//
//   - Not(implicit $and)           => {$nor: [implicit_and_doc]}
//     e.g. Not(Filter{a:1, b:{$gt:2}}) => {$nor: [{a:1, b:{$gt:2}}]}
//
// IMPORTANT: $not is a field-level operator and must always be placed inside
// the field's value. It must never wrap a logical operator ($and/$or/$nor)
// because MongoDB does not support {$not: {$or: [...]}}.
func Not(filter Filter) Filter {
	return &not{filter: filter}
}
