package filter

import (
	"bytes"
	"go.mongodb.org/mongo-driver/v2/bson"
	"math/rand"
	"sort"
	"testing"
)

func toBSON(d bson.D) string {
	raw, _ := bson.Marshal(d)
	var out bson.D
	_ = bson.Unmarshal(raw, &out)
	bs, _ := bson.MarshalExtJSON(out, false, false)
	return string(bs)
}

func normalize(d bson.D) bson.D {
	sort.Slice(d, func(i, j int) bool {
		return d[i].Key < d[j].Key
	})
	for i := range d {
		if sub, ok := d[i].Value.(bson.D); ok {
			d[i].Value = normalize(sub)
		}
	}
	return d
}

func assertFilterEqual(t *testing.T, actual, expected bson.D) {
	a := normalize(actual)
	b := normalize(expected)
	ba, _ := bson.Marshal(a)
	bb, _ := bson.Marshal(b)
	if !bytes.Equal(ba, bb) {
		t.Fatalf("\nExpected: %s\nActual: %s", toBSON(b), toBSON(a))
	}
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

func hasKey(d bson.D, key string) bool {
	for _, e := range d {
		if e.Key == key {
			return true
		}
	}
	return false
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

func getField(d bson.D, key string) (interface{}, bool) {
	for _, e := range d {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

func andContains(d bson.D, target bson.D) bool {
	and := extractAnd(d)
	for _, item := range and {
		if doc, ok := item.(bson.D); ok {
			if bsonDocEqual(doc, target) {
				return true
			}
		}
	}
	return false
}

func countConditions(d bson.D) int {
	count := 0
	var walk func(bson.D)
	walk = func(d bson.D) {
		for _, e := range d {
			count++
			switch v := e.Value.(type) {
			case bson.D:
				walk(v)
			case bson.A:
				for _, item := range v {
					if sub, ok := item.(bson.D); ok {
						walk(sub)
					}
				}
			}
		}
	}
	walk(d)
	return count
}

func TestFlattenDoc(t *testing.T) {
	tests := []struct {
		name     string
		input    bson.D
		expected bson.D
		check    func(t *testing.T, out bson.D)
	}{
		{
			name: "simple $and flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$lt", 5}}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
			},
		},
		{
			name: "different fields flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"b", 2}},
				}},
			},
			expected: bson.D{
				{"a", 1},
				{"b", 2},
			},
		},
		{
			name: "same operator conflict -> keep $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
		},
		{
			name: "pure value vs operator conflict",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", bson.D{{"$gt", 0}}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$eq", 1}, {"$gt", 0}}},
			},
		},
		{
			name: "nested $and flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"$and", bson.A{
						bson.D{{"a", bson.D{{"$gt", 1}}}},
						bson.D{{"a", bson.D{{"$lt", 5}}}},
					}}},
					bson.D{{"b", 2}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
				{"b", 2},
			},
		},
		{
			name: "$or inside $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"$or", bson.A{
						bson.D{{"b", 1}},
						bson.D{{"c", 2}},
					}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}}},
				{"$or", bson.A{
					bson.D{{"b", 1}},
					bson.D{{"c", 2}},
				}},
			},
		},
		{
			name: "mixed flattenable and non-flushable",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$lt", 5}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}}, // conflict
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}, {"$lt", 5}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
		},
		{
			name: "pure value duplicate -> flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 1}},
				}},
			},
			expected: bson.D{
				{"a", 1},
			},
		},
		{
			name: "pure value different -> conflict",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
		},
		{
			name: "no $and at all",
			input: bson.D{
				{"a", 1},
				{"b", bson.D{{"$gt", 2}}},
			},
			expected: bson.D{
				{"a", 1},
				{"b", bson.D{{"$gt", 2}}},
			},
		},
		{
			name: "$nor preserved",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
			expected: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
		},
		{
			name: "deep nested $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"$and", bson.A{
						bson.D{{"$and", bson.A{
							bson.D{{"a", bson.D{{"$gt", 1}}}},
							bson.D{{"a", bson.D{{"$lt", 5}}}},
						}}},
					}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
			},
		},
		// ---------- 基础 ----------
		{
			name: "no $and",
			input: bson.D{
				{"a", 1},
				{"b", bson.D{{"$gt", 2}}},
			},
			expected: bson.D{
				{"a", 1},
				{"b", bson.D{{"$gt", 2}}},
			},
		},
		{
			name: "simple $and flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$lt", 5}}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
			},
		},
		{
			name: "different fields flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"b", 2}},
				}},
			},
			expected: bson.D{
				{"a", 1},
				{"b", 2},
			},
		},
		{
			name: "pure_value_multiple_conflicts",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
					bson.D{{"a", 3}},
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
					bson.D{{"a", 3}},
				}},
			},
		},

		// ---------- operator conflicts ----------
		{
			name: "same_operator_conflict -> keep $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
		},
		{
			name: "operator_conflict_mixed_order",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 3}}}},
					bson.D{{"a", bson.D{{"$gt", 1}}}},
				}},
			},
			expected: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 3}}}},
					bson.D{{"a", bson.D{{"$gt", 1}}}},
				}},
			},
		},

		// ---------- pure value vs operator ----------
		{
			name: "operator_vs_pure_value_conflict",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 0}}}},
					bson.D{{"a", 1}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$eq", 1}, {"$gt", 0}}},
			},
		},

		// ---------- mixed flattenable + conflict ----------
		{
			name: "mixed_flattenable_and_non_flushable",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$lt", 5}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				and := extractAnd(out)
				if len(and) != 2 {
					t.Fatalf("expected 2 conditions in $and, got %d", len(and))
				}
				// 一个包含 $gt + $lt，一个包含 $gt
			},
		},

		// ---------- $or / $nor ----------
		{
			name: "$or inside $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"$or", bson.A{
						bson.D{{"b", 1}},
						bson.D{{"c", 2}},
					}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "a") {
					t.Fatalf("missing a")
				}
				if !hasKey(out, "$or") {
					t.Fatalf("missing $or")
				}
				or := extractOr(out)
				if len(or) != 2 {
					t.Fatalf("expected 2 clauses in $or")
				}
			},
		},
		{
			name: "$nor preserved",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
			expected: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
				}},
			},
		},

		// ---------- nested $and ----------
		{
			name: "nested $and flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"$and", bson.A{
						bson.D{{"a", bson.D{{"$gt", 1}}}},
						bson.D{{"a", bson.D{{"$lt", 5}}}},
					}}},
					bson.D{{"b", 2}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
				{"b", 2},
			},
		},
		{
			name: "deep nested $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"$and", bson.A{
						bson.D{{"$and", bson.A{
							bson.D{{"a", bson.D{{"$gt", 1}}}},
							bson.D{{"a", bson.D{{"$lt", 5}}}},
						}}},
					}}},
				}},
			},
			expected: bson.D{
				{"a", bson.D{{"$gt", 1}, {"$lt", 5}}},
			},
		},

		// ---------- edge cases ----------
		{
			name: "non-doc in $and array",
			input: bson.D{
				{"$and", bson.A{
					"invalid",
					bson.D{{"a", 1}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				and := extractAnd(out)
				if len(and) != 2 {
					t.Fatalf("expected 2 elements in $and")
				}
				if and[0] != "invalid" {
					t.Fatalf("first element should be 'invalid', got %#v", and[0])
				}
				if !andContains(out, bson.D{{"a", 1}}) {
					t.Fatalf("missing {a:1} in $and")
				}
			},
		},
		{
			name: "empty $and",
			input: bson.D{
				{"$and", bson.A{}},
			},
			expected: bson.D{
				{"$and", bson.A{}},
			},
		},
		{
			name: "single element $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
				}},
			},
			expected: bson.D{
				{"a", 1},
			},
		},
		{
			name: "invariant: condition count never decreases",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"a", 2}},
					bson.D{{"b", 3}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if countConditions(out) < 3 {
					t.Fatalf("condition count decreased")
				}
			},
		},
		{
			name: "invalid $and is never repaired",
			input: bson.D{
				{"$and", bson.A{
					"invalid",
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					nil,
				}},
			},
			check: func(t *testing.T, out bson.D) {
				and := extractAnd(out)
				if len(and) != 3 {
					t.Fatalf("expected 3 elements, got %d", len(and))
				}
			},
		},
		{
			name: "$elemMatch must not flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$elemMatch", bson.D{{"x", 1}}}}}},
					bson.D{{"a", bson.D{{"$elemMatch", bson.D{{"y", 2}}}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				and := extractAnd(out)
				if len(and) != 2 {
					t.Fatalf("$elemMatch must not be merged")
				}
			},
		},
		{
			name: "$all must not flatten",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$all", bson.A{1, 2}}}}},
					bson.D{{"a", bson.D{{"$all", bson.A{3, 4}}}},
					}},
				}},
			check: func(t *testing.T, out bson.D) {
				if len(extractAnd(out)) != 2 {
					t.Fatalf("$all must not be merged")
				}
			},
		},
		{
			name: "$or with nested $or",
			input: bson.D{
				{"$or", bson.A{
					bson.D{{"$or", bson.A{
						bson.D{{"a", 1}},
						bson.D{{"b", 2}},
					}}},
					bson.D{{"c", 3}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				or := extractOr(out)
				if len(or) != 3 {
					t.Fatalf("expected 3 clauses after flatten")
				}
			},
		},
		{
			name: "$or single clause preserved",
			input: bson.D{
				{"$or", bson.A{
					bson.D{{"a", 1}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$or") {
					t.Fatalf("$or must be preserved")
				}
				or := extractOr(out)
				if len(or) != 1 {
					t.Fatalf("expected 1 clause")
				}
			},
		},
		{
			name: "$or conflict preserved",
			input: bson.D{
				{"$or", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				or := extractOr(out)
				if len(or) != 2 {
					t.Fatalf("$or conflict must be preserved")
				}
			},
		},
		{
			name: "$nor nested $nor preserved (double negation)",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
					bson.D{{"$nor", bson.A{
						bson.D{{"b", 2}},
						bson.D{{"c", 3}},
					}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$nor") {
					t.Fatalf("missing $nor")
				}
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
				}
				// 第二个 clause 应该还是 {$nor: [{b:2}, {c:3}]}
				secondClause, ok := nor[1].(bson.D)
				if !ok || len(secondClause) != 1 || secondClause[0].Key != "$nor" {
					t.Fatalf("nested $nor should be preserved, got %#v", nor[1])
				}
			},
		},
		{
			name: "$nor with $and inside",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}, {"b", 2}},
					bson.D{{"c", 3}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$nor") {
					t.Fatalf("missing $nor")
				}
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
				}
				// 第一个 clause 是隐式 $and，应该被 flattenAnd 处理
				firstClause, ok := nor[0].(bson.D)
				if !ok {
					t.Fatalf("first clause should be bson.D")
				}
				// flattenAnd 应该把 {a:1, b:2} 合并成一个 bson.D
				if len(firstClause) != 2 {
					t.Fatalf("first clause should have 2 fields, got %d", len(firstClause))
				}
			},
		},
		{
			name: "$nor with $or inside",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"$or", bson.A{
						bson.D{{"a", 1}},
						bson.D{{"b", 2}},
					}}},
					bson.D{{"c", 3}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$nor") {
					t.Fatalf("missing $nor")
				}
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
				}
				// 第一个 clause 应该包含 $or
				firstClause, ok := nor[0].(bson.D)
				if !ok || !hasKey(firstClause, "$or") {
					t.Fatalf("first clause should contain $or")
				}
			},
		},
		{
			name: "$nor single clause preserved",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", 1}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$nor") {
					t.Fatalf("$nor must be preserved")
				}
				nor := extractNor(out)
				if len(nor) != 1 {
					t.Fatalf("expected 1 clause, got %d", len(nor))
				}
			},
		},
		{
			name: "$nor conflict preserved (same field, same operator)",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"a", bson.D{{"$gt", 3}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("$nor conflict must be preserved, got %d clauses", len(nor))
				}
			},
		},
		{
			name: "$and with $nor inside",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", bson.D{{"$gt", 1}}}},
					bson.D{{"$nor", bson.A{
						bson.D{{"b", 1}},
						bson.D{{"c", 2}},
					}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "a") {
					t.Fatalf("missing a")
				}
				if !hasKey(out, "$nor") {
					t.Fatalf("missing $nor")
				}
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
				}
			},
		},
		{
			name: "empty $nor",
			input: bson.D{
				{"$nor", bson.A{}},
			},
			expected: bson.D{
				{"$nor", bson.A{}},
			},
		},
		{
			name: "$nor with non-doc element",
			input: bson.D{
				{"$nor", bson.A{
					"invalid",
					bson.D{{"a", 1}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 elements in $nor, got %d", len(nor))
				}
				if nor[0] != "invalid" {
					t.Fatalf("first element should be 'invalid', got %#v", nor[0])
				}
			},
		},
		{
			name: "regex implicit converted to $regex and merged with pure value",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"a", 7}},
					bson.D{{"a", bson.Regex{Pattern: "^abc", Options: "i"}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if hasKey(out, "$and") {
					t.Fatalf("should not have $and")
				}
				val, ok := getField(out, "a")
				if !ok {
					t.Fatalf("missing a")
				}
				_, ok = val.(bson.D)
				if !ok {
					t.Fatalf("a value should be bson.D")
				}
				if !hasOperator(out, "a", "$eq") {
					t.Fatalf("missing $eq in a\nflat=%s", toBSON(out))
				}
				if !hasOperator(out, "a", "$regex") {
					t.Fatalf("missing $regex in a")
				}
			},
		},
		{
			name: "regex explicit $regex with bson.Regex merged with $gt",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"name", bson.D{{"$regex", bson.Regex{Pattern: "^abc", Options: "i"}}}}},
					bson.D{{"name", bson.D{{"$gt", 18}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if hasKey(out, "$and") {
					t.Fatalf("should not have $and")
				}
				val, ok := getField(out, "name")
				if !ok {
					t.Fatalf("missing name")
				}
				_, ok = val.(bson.D)
				if !ok {
					t.Fatalf("name value should be bson.D")
				}
				if !hasOperator(out, "name", "$regex") {
					t.Fatalf("missing $regex in name")
				}
				if !hasOperator(out, "name", "$gt") {
					t.Fatalf("missing $gt in name")
				}
			},
		},
		{
			name: "regex explicit $regex string with $options merged with $gt",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"name", bson.D{{"$regex", "^abc"}, {"$options", "i"}}}},
					bson.D{{"name", bson.D{{"$gt", 18}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if hasKey(out, "$and") {
					t.Fatalf("should not have $and")
				}
				val, ok := getField(out, "name")
				if !ok {
					t.Fatalf("missing name")
				}
				_, ok = val.(bson.D)
				if !ok {
					t.Fatalf("name value should be bson.D")
				}
				if !hasOperator(out, "name", "$regex") {
					t.Fatalf("missing $regex in name")
				}
				if !hasOperator(out, "name", "$gt") {
					t.Fatalf("missing $gt in name")
				}
			},
		},
		{
			name: "same field two $regex conflict keeps $and",
			input: bson.D{
				{"$and", bson.A{
					bson.D{{"name", bson.D{{"$regex", "^abc"}}}},
					bson.D{{"name", bson.D{{"$regex", "^def"}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				and := extractAnd(out)
				if len(and) != 2 {
					t.Fatalf("expected 2 clauses in $and due to $regex conflict, got %d", len(and))
				}
			},
		},
		{
			name: "regex in $or preserved",
			input: bson.D{
				{"$or", bson.A{
					bson.D{{"name", bson.Regex{Pattern: "^abc", Options: "i"}}},
					bson.D{{"age", bson.D{{"$gt", 18}}}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$or") {
					t.Fatalf("missing $or")
				}
				or := extractOr(out)
				if len(or) != 2 {
					t.Fatalf("expected 2 clauses in $or, got %d", len(or))
				}
			},
		},
		{
			name: "regex in $nor preserved",
			input: bson.D{
				{"$nor", bson.A{
					bson.D{{"name", bson.Regex{Pattern: "^admin", Options: "i"}}},
					bson.D{{"status", "banned"}},
				}},
			},
			check: func(t *testing.T, out bson.D) {
				if !hasKey(out, "$nor") {
					t.Fatalf("missing $nor")
				}
				nor := extractNor(out)
				if len(nor) != 2 {
					t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := flattenDoc(tt.input)
			if tt.expected != nil {
				assertFilterEqual(t, out, tt.expected)
			}
			if tt.check != nil {
				tt.check(t, out)
			}
		})
	}
}

func extractNor(d bson.D) bson.A {
	for _, e := range d {
		if e.Key == "$nor" {
			if arr, ok := e.Value.(bson.A); ok {
				return arr
			}
		}
	}
	return nil
}

func randomFilter(maxDepth int) bson.D {
	if maxDepth <= 0 {
		return randomLeaf()
	}
	return randomNode(maxDepth)
}

func randomNode(depth int) bson.D {
	if rand.Float32() < 0.3 && depth > 1 {
		switch rand.Intn(2) {
		case 0:
			return bson.D{{"$and", randomArray(depth - 1)}}
		case 1:
			return bson.D{{"$or", randomArray(depth - 1)}}
		}
	}
	return randomLeaf()
}

func randomArray(depth int) bson.A {
	n := rand.Intn(3) + 1
	arr := make(bson.A, n)
	for i := 0; i < n; i++ {
		arr[i] = randomNode(depth)
	}
	return arr
}

func randomLeaf() bson.D {
	field := randomField()
	switch rand.Intn(3) {
	case 0:
		return bson.D{{field, randomValue()}}
	case 1:
		return bson.D{{field, bson.D{
			{randomOperator(), randomValue()},
		}}}
	default:
		return bson.D{{field, bson.D{
			{"$gt", rand.Intn(10)},
			{"$lt", rand.Intn(20) + 10},
		}}}
	}
}

func randomField() string {
	return []string{"a", "b", "c", "x", "y", "z"}[rand.Intn(6)]
}

func randomOperator() string {
	return []string{"$gt", "$lt", "$gte", "$lte", "$ne"}[rand.Intn(5)]
}

func randomValue() interface{} {
	switch rand.Intn(4) {
	case 0:
		return rand.Intn(100)
	case 1:
		return rand.Float64() * 100
	case 2:
		return []string{"foo", "bar", "baz"}[rand.Intn(3)]
	default:
		return rand.Intn(2) == 0
	}
}

func randomFilterWithRand(r *rand.Rand, maxDepth int) bson.D {
	if maxDepth <= 0 {
		return randomLeafWithRand(r)
	}
	return randomNodeWithRand(r, maxDepth)
}

func randomNodeWithRand(r *rand.Rand, depth int) bson.D {
	if r.Float32() < 0.3 && depth > 1 {
		switch r.Intn(3) { // 3 种逻辑算子
		case 0:
			return bson.D{{"$and", randomArrayWithRand(r, depth-1)}}
		case 1:
			return bson.D{{"$or", randomArrayWithRand(r, depth-1)}}
		case 2:
			return bson.D{{"$nor", randomArrayWithRand(r, depth-1)}}
		}
	}
	return randomLeafWithRand(r)
}

func randomArrayWithRand(r *rand.Rand, depth int) bson.A {
	n := r.Intn(3) + 1
	arr := make(bson.A, n)
	for i := 0; i < n; i++ {
		arr[i] = randomNodeWithRand(r, depth)
	}
	return arr
}

func randomLeafWithRand(r *rand.Rand) bson.D {
	field := randomFieldWithRand(r)
	switch r.Intn(3) {
	case 0:
		return bson.D{{field, randomValueWithRand(r)}}
	case 1:
		return bson.D{{field, bson.D{
			{randomOperatorWithRand(r), randomValueWithRand(r)},
		}}}
	default:
		return bson.D{{field, bson.D{
			{"$gt", r.Intn(10)},
			{"$lt", r.Intn(20) + 10},
		}}}
	}
}

func randomFieldWithRand(r *rand.Rand) string {
	return []string{"a", "b", "c", "x", "y", "z"}[r.Intn(6)]
}

func randomOperatorWithRand(r *rand.Rand) string {
	return []string{"$gt", "$lt", "$gte", "$lte", "$ne"}[r.Intn(5)]
}

func randomValueWithRand(r *rand.Rand) interface{} {
	switch r.Intn(5) { // 5 种
	case 0:
		return r.Intn(100)
	case 1:
		return r.Float64() * 100
	case 2:
		return []string{"foo", "bar", "baz"}[r.Intn(3)]
	case 3:
		return r.Intn(2) == 0
	case 4:
		// 脏数据
		if r.Float32() < 0.5 {
			return nil
		}
		return "invalid"
	default:
		return nil
	}
}

func countClauses(d bson.D) int {
	count := 0
	var walk func(bson.D)
	walk = func(d bson.D) {
		for _, e := range d {
			count++
			switch v := e.Value.(type) {
			case bson.D:
				walk(v)
			case bson.A:
				for _, item := range v {
					if sub, ok := item.(bson.D); ok {
						walk(sub)
					} else {
						count++ // 非 doc（invalid / nil / primitive）
					}
				}
			}
		}
	}
	walk(d)
	return count
}

func TestFlattenDoc_NoPanic(t *testing.T) {
	r := rand.New(rand.NewSource(42)) // 固定 seed，方便复现
	for i := 0; i < 1000; i++ {
		input := randomFilterWithRand(r, 4)
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Fatalf(
						"panic on iteration %d\nInput: %s\nError: %v",
						i,
						toBSON(input),
						rec,
					)
				}
			}()
			before := countClauses(input)
			flat := flattenDoc(input)
			after := countClauses(flat)
			if after > before {
				t.Fatalf(
					"condition count decreased\nbefore=%d after=%d\ninput=%s\nflat=%s",
					before, after, toBSON(input), toBSON(flat),
				)
			}
			// 结构合法（能 Marshal）
			if _, err := bson.Marshal(flat); err != nil {
				t.Fatalf("invalid bson after flatten\ninput=%s\nflat=%s\nerr=%v", toBSON(input), toBSON(flat), err)
			}
		}()
	}
}

//func TestFlattenDoc_RoundTrip(t *testing.T) {
//	r := rand.New(rand.NewSource(123))
//	for i := 0; i < 100; i++ {
//		input := randomFilterWithRand(r, 3)
//		flat := flattenDoc(input)
//
//		// Marshal / Unmarshal 消除顺序差异
//		raw1, _ := bson.Marshal(input)
//		raw2, _ := bson.Marshal(flat)
//
//		var d1, d2 bson.D
//		_ = bson.Unmarshal(raw1, &d1)
//		_ = bson.Unmarshal(raw2, &d2)
//
//		if !bsonDocEqual(d1, d2) {
//			t.Fatalf(
//				"round-trip mismatch\ninput=%s\nflat=%s",
//				toBSON(d1), toBSON(d2),
//			)
//		}
//	}
//}
