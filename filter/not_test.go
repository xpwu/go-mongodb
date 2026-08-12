package filter

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ---------- TestRunNot_Additional ----------

func TestRunNot_EmptyInput(t *testing.T) {
	out := runNot(bson.D{})
	if !hasKey(out, "$not") {
		t.Fatalf("expected $not for empty input, got %s", toBSON(out))
	}
}

func TestRunNot_EmptyDInside(t *testing.T) {
	input := bson.D{{"a", bson.D{}}}
	out := runNot(input)

	if !hasOperator(out, "a", "$not") {
		t.Fatalf("expected a.$not, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$not" {
			notVal := e.Value.(bson.D)
			if len(notVal) != 0 {
				t.Fatalf("expected empty $not value, got %s", toBSON(notVal))
			}
		}
	}
}

func TestRunNot_NotOnNotOnComplex(t *testing.T) {
	// Not(Not({a: {$gt:5, $lt:10}})) => {a: {$gt:5, $lt:10}}
	input := bson.D{{"a", bson.D{{"$not", bson.D{{"$gt", 5}, {"$lt", 10}}}}}}
	out := runNot(input)

	if hasKey(out, "$not") {
		t.Fatalf("double negation should be eliminated, got %s", toBSON(out))
	}
	val, ok := getField(out, "a")
	if !ok {
		t.Fatalf("missing a, got %s", toBSON(out))
	}
	valDoc, ok := val.(bson.D)
	if !ok {
		t.Fatalf("a should be bson.D, got %T", val)
	}
	if !hasOperator(out, "a", "$gt") || !hasOperator(out, "a", "$lt") {
		t.Fatalf("expected $gt and $lt, got %s", toBSON(valDoc))
	}
}

func TestRunNot_ImplicitAndWithThreeFields(t *testing.T) {
	input := bson.D{
		{"a", 1},
		{"b", 2},
		{"c", bson.D{{"$gt", 3}}},
	}
	out := runNot(input)

	if !hasKey(out, "$nor") {
		t.Fatalf("expected $nor, got %s", toBSON(out))
	}
	nor := extractNor(out)
	if len(nor) != 1 {
		t.Fatalf("expected 1 clause in $nor, got %d", len(nor))
	}
}

func TestRunNot_NotWithRegex(t *testing.T) {
	input := bson.D{{"name", bson.Regex{Pattern: "test", Options: ""}}}
	out := runNot(input)

	if !hasOperator(out, "name", "$not") {
		t.Fatalf("expected name.$not, got %s", toBSON(out))
	}
	val, _ := getField(out, "name")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$not" {
			notVal := e.Value.(bson.D)
			if !hasOperator2(notVal, "$regex") {
				t.Fatalf("expected $regex inside $not, got %s", toBSON(notVal))
			}
		}
	}
}

func TestRunNot_NotWithArrayValue(t *testing.T) {
	input := bson.D{{"a", bson.A{1, 2, 3}}}
	out := runNot(input)

	if !hasOperator(out, "a", "$ne") {
		t.Fatalf("expected a.$ne, got %s", toBSON(out))
	}
}

func TestRunNot_NotWithNilValue(t *testing.T) {
	input := bson.D{{"a", nil}}
	out := runNot(input)

	if !hasOperator(out, "a", "$ne") {
		t.Fatalf("expected a.$ne, got %s", toBSON(out))
	}
}

func TestRunNot_NotWithStringValue(t *testing.T) {
	input := bson.D{{"status", "active"}}
	out := runNot(input)

	if !hasOperator(out, "status", "$ne") {
		t.Fatalf("expected status.$ne, got %s", toBSON(out))
	}
	val, _ := getField(out, "status")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$ne" && e.Value != "active" {
			t.Fatalf("expected 'active', got %v", e.Value)
		}
	}
}

// ---------- TestNotOp_Additional ----------

func TestNotOp_UnknownOperator(t *testing.T) {
	out := notOp(bson.E{Key: "$mod", Value: bson.D{{"divisor", 4}, {"remainder", 0}}})
	if len(out) != 1 || out[0].Key != "$not" {
		t.Fatalf("expected $not wrapper, got %#v", out)
	}
}

func TestNotOp_ExistsNonBool(t *testing.T) {
	out := notOp(bson.E{Key: "$exists", Value: "yes"})
	if len(out) != 1 || out[0].Key != "$not" {
		t.Fatalf("expected $not wrapper for non-bool $exists, got %#v", out)
	}
}

func TestNotOp_DoubleNotElimination(t *testing.T) {
	inner := bson.D{{"$eq", 5}}
	out := notOp(bson.E{Key: "$not", Value: inner})
	// 应该返回 flattenDoc(inner) = {$eq: 5}
	if len(out) != 1 || out[0].Key != "$eq" {
		t.Fatalf("expected $eq after double negation, got %#v", out)
	}
	if out[0].Value != 5 {
		t.Fatalf("expected 5, got %v", out[0].Value)
	}
}
