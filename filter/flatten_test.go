package filter

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ---------- TestFlattenDoc_Additional ----------

func TestFlattenDoc_EmptyInput(t *testing.T) {
	out := flattenDoc(bson.D{})
	if len(out) != 0 {
		t.Fatalf("expected empty output, got %s", toBSON(out))
	}
}

func TestFlattenDoc_SingleFieldNoOp(t *testing.T) {
	input := bson.D{{"a", 5}}
	out := flattenDoc(input)
	if hasKey(out, "$and") {
		t.Fatalf("should not have $and, got %s", toBSON(out))
	}
	val, ok := getField(out, "a")
	if !ok || val != 5 {
		t.Fatalf("expected a=5, got %s", toBSON(out))
	}
}

func TestFlattenDoc_SingleFieldWithOp(t *testing.T) {
	input := bson.D{{"a", bson.D{{"$gt", 5}}}}
	out := flattenDoc(input)
	val, ok := getField(out, "a")
	if !ok {
		t.Fatalf("missing a, got %s", toBSON(out))
	}
	valDoc, ok := val.(bson.D)
	if !ok || !hasOperator2(valDoc, "$gt") {
		t.Fatalf("expected a.$gt, got %s", toBSON(out))
	}
}

func TestFlattenDoc_ArrayButNotLogical(t *testing.T) {
	input := bson.D{{"a", bson.A{1, 2, 3}}}
	out := flattenDoc(input)
	val, ok := getField(out, "a")
	if !ok || len(val.(bson.A)) != 3 {
		t.Fatalf("expected a=[1,2,3], got %s", toBSON(out))
	}
}

// ---------- TestFlattenAnd_Additional ----------

func TestFlattenAnd_Empty(t *testing.T) {
	out := flattenAnd(bson.A{})
	if !hasKey(out, "$and") {
		t.Fatalf("expected $and for empty input, got %s", toBSON(out))
	}
	and := extractAnd(out)
	if len(and) != 0 {
		t.Fatalf("expected empty $and, got %s", toBSON(out))
	}
}

func TestFlattenAnd_ThreeFieldsNoConflict(t *testing.T) {
	input := bson.A{
		bson.D{{"a", 1}},
		bson.D{{"b", 2}},
		bson.D{{"c", 3}},
	}
	out := flattenAnd(input)
	if hasKey(out, "$and") {
		t.Fatalf("should not have $and, got %s", toBSON(out))
	}
	if !hasKey(out, "a") || !hasKey(out, "b") || !hasKey(out, "c") {
		t.Fatalf("missing fields, got %s", toBSON(out))
	}
}

func TestFlattenAnd_RegexAndPureValue(t *testing.T) {
	input := bson.A{
		bson.D{{"name", bson.Regex{Pattern: "^a", Options: ""}}},
		bson.D{{"name", "exact"}},
	}
	out := flattenAnd(input)
	if hasKey(out, "$and") {
		t.Fatalf("should merge regex+value, got %s", toBSON(out))
	}
	val, _ := getField(out, "name")
	valDoc := val.(bson.D)
	if !hasOperator2(valDoc, "$regex") || !hasOperator2(valDoc, "$eq") {
		t.Fatalf("expected $regex+$eq, got %s", toBSON(valDoc))
	}
}

func TestFlattenAnd_NonDocElement(t *testing.T) {
	input := bson.A{
		"just a string",
		bson.D{{"a", 1}},
		nil,
	}
	out := flattenAnd(input)
	and := extractAnd(out)
	if len(and) != 3 {
		t.Fatalf("expected 3 elements in $and, got %d, out=%s", len(and), toBSON(out))
	}
}

func TestFlattenAnd_NestedOrAndNor(t *testing.T) {
	input := bson.A{
		bson.D{{"$or", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
		bson.D{{"$nor", bson.A{
			bson.D{{"c", 3}},
			bson.D{{"d", 4}},
		}}},
	}
	out := flattenAnd(input)
	if !hasKey(out, "$or") {
		t.Fatalf("missing $or, got %s", toBSON(out))
	}
	if !hasKey(out, "$nor") {
		t.Fatalf("missing $nor, got %s", toBSON(out))
	}
}

func TestFlattenAnd_GteAndLte(t *testing.T) {
	input := bson.A{
		bson.D{{"a", bson.D{{"$gte", 1}}}},
		bson.D{{"a", bson.D{{"$lte", 10}}}},
	}
	out := flattenAnd(input)
	if hasKey(out, "$and") {
		t.Fatalf("should merge $gte+$lte, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	if !hasOperator2(valDoc, "$gte") || !hasOperator2(valDoc, "$lte") {
		t.Fatalf("expected $gte+$lte, got %s", toBSON(valDoc))
	}
}

func TestFlattenAnd_GtAndLt(t *testing.T) {
	input := bson.A{
		bson.D{{"a", bson.D{{"$gt", 0}}}},
		bson.D{{"a", bson.D{{"$lt", 100}}}},
	}
	out := flattenAnd(input)
	if hasKey(out, "$and") {
		t.Fatalf("should merge $gt+$lt, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	if !hasOperator2(valDoc, "$gt") || !hasOperator2(valDoc, "$lt") {
		t.Fatalf("expected $gt+$lt, got %s", toBSON(valDoc))
	}
}

// ---------- TestFlattenOr_Additional ----------

func TestFlattenOr_Empty(t *testing.T) {
	out := flattenOr(bson.A{})
	if !hasKey(out, "$or") {
		t.Fatalf("expected $or for empty input, got %s", toBSON(out))
	}
	or := extractOr(out)
	if len(or) != 0 {
		t.Fatalf("expected empty $or, got %s", toBSON(out))
	}
}

func TestFlattenOr_NonDocElement(t *testing.T) {
	input := bson.A{
		"invalid",
		bson.D{{"a", 1}},
	}
	out := flattenOr(input)
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 elements, got %d, out=%s", len(or), toBSON(out))
	}
}

func TestFlattenOr_NestedAnd(t *testing.T) {
	input := bson.A{
		bson.D{{"$and", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
		bson.D{{"c", 3}},
	}
	out := flattenOr(input)
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 clauses, got %d, out=%s", len(or), toBSON(out))
	}
	first, ok := or[0].(bson.D)
	if !ok || len(first) != 2 {
		t.Fatalf("first clause should be flattened to 2 fields, got %s", toBSON(first))
	}
}

func TestFlattenOr_NestedNor(t *testing.T) {
	input := bson.A{
		bson.D{{"$nor", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
		bson.D{{"c", 3}},
	}
	out := flattenOr(input)
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 clauses, got %d, out=%s", len(or), toBSON(out))
	}
}

func TestFlattenOr_ImplicitAndInside(t *testing.T) {
	input := bson.A{
		bson.D{{"a", 1}, {"b", 2}},
		bson.D{{"c", 3}},
	}
	out := flattenOr(input)
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 clauses, got %d, out=%s", len(or), toBSON(out))
	}
}

// ---------- TestFlattenNor_Additional ----------

func TestFlattenNor_Empty(t *testing.T) {
	out := flattenNor(bson.A{})
	if !hasKey(out, "$nor") {
		t.Fatalf("expected $nor for empty input, got %s", toBSON(out))
	}
	nor := extractNor(out)
	if len(nor) != 0 {
		t.Fatalf("expected empty $nor, got %s", toBSON(out))
	}
}

func TestFlattenNor_NonDocElement(t *testing.T) {
	input := bson.A{
		"invalid",
		bson.D{{"a", 1}},
	}
	out := flattenNor(input)
	nor := extractNor(out)
	if len(nor) != 2 {
		t.Fatalf("expected 2 elements, got %d, out=%s", len(nor), toBSON(out))
	}
}

func TestFlattenNor_NestedNorRecurses(t *testing.T) {
	input := bson.A{
		bson.D{{"$nor", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
	}
	out := flattenNor(input)
	nor := extractNor(out)
	if len(nor) != 1 {
		t.Fatalf("expected 1 clause, got %d, out=%s", len(nor), toBSON(out))
	}
	clause, ok := nor[0].(bson.D)
	if !ok || !hasKey(clause, "$nor") {
		t.Fatalf("nested $nor should be preserved, got %s", toBSON(out))
	}
}

func TestFlattenNor_NestedAnd(t *testing.T) {
	input := bson.A{
		bson.D{{"$and", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
	}
	out := flattenNor(input)
	nor := extractNor(out)
	if len(nor) != 1 {
		t.Fatalf("expected 1 clause, got %d, out=%s", len(nor), toBSON(out))
	}
	clause, ok := nor[0].(bson.D)
	if !ok || len(clause) != 2 {
		t.Fatalf("clause should be flattened to 2 fields, got %s", toBSON(clause))
	}
}

func TestFlattenNor_NestedOr(t *testing.T) {
	input := bson.A{
		bson.D{{"$or", bson.A{
			bson.D{{"a", 1}},
			bson.D{{"b", 2}},
		}}},
	}
	out := flattenNor(input)
	nor := extractNor(out)
	if len(nor) != 1 {
		t.Fatalf("expected 1 clause, got %d, out=%s", len(nor), toBSON(out))
	}
	clause, ok := nor[0].(bson.D)
	if !ok || !hasKey(clause, "$or") {
		t.Fatalf("nested $or should be preserved, got %s", toBSON(out))
	}
}

// ---------- TestToBsonA ----------

func TestToBsonA(t *testing.T) {
	docs := []bson.D{
		{{"a", 1}},
		{{"b", 2}},
	}
	out := toBsonA(docs)
	if len(out) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(out))
	}
	first := out[0].(bson.D)
	if first[0].Key != "a" {
		t.Fatalf("expected a, got %s", toBSON(first))
	}
}

// ---------- TestBsonDocEqual ----------

func TestBsonDocEqual_Identical(t *testing.T) {
	a := bson.D{{"a", 1}, {"b", bson.D{{"$gt", 2}}}}
	b := bson.D{{"a", 1}, {"b", bson.D{{"$gt", 2}}}}
	if !bsonDocEqual(a, b) {
		t.Fatalf("identical docs should be equal")
	}
}

func TestBsonDocEqual_Different(t *testing.T) {
	a := bson.D{{"a", 1}}
	b := bson.D{{"a", 2}}
	if bsonDocEqual(a, b) {
		t.Fatalf("different docs should not be equal")
	}
}

func TestBsonDocEqual_Reordered(t *testing.T) {
	a := bson.D{{"a", 1}, {"b", 2}}
	b := bson.D{{"b", 2}, {"a", 1}}
	if bsonDocEqual(a, b) {
		t.Fatalf("reordered docs should not be equal via bson.M")
	}
}
