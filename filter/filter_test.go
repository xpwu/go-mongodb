package filter

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ---------- mockField ----------
// 模拟 field.Field 接口的最小实现
// 如果外部包中 field.Field 接口方法不同，请调整

type mockField struct {
	name   string
	parent *mockField
}

func (m *mockField) FullName() string {
	if m.parent != nil {
		p := m.parent.FullName()
		if p != "" {
			return p + "." + m.name
		}
	}
	return m.name
}

func (m *mockField) Name() string {
	return m.name
}

func (m *mockField) Parent() *mockField {
	return m.parent
}

// ---------- TestCompareByValue ----------

func TestCompareByValue_EQ(t *testing.T) {
	f := &mockField{name: "a"}
	out := CompareByValue(f, EQ, 5).ToBsonD()

	if hasKey(out, "$eq") {
		t.Fatalf("EQ should not wrap $eq, got %s", toBSON(out))
	}
	val, ok := getField(out, "a")
	if !ok {
		t.Fatalf("missing a, got %s", toBSON(out))
	}
	if val != 5 {
		t.Fatalf("expected 5, got %#v", val)
	}
}

func TestCompareByValue_NonEQ(t *testing.T) {
	tests := []struct {
		name     string
		c        Comparer
		value    interface{}
		expected string
	}{
		{"GT", GT, 5, "$gt"},
		{"GTE", GTE, 5, "$gte"},
		{"LT", LT, 5, "$lt"},
		{"LTE", LTE, 5, "$lte"},
		{"NE", NE, 5, "$ne"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &mockField{name: "a"}
			out := CompareByValue(f, tt.c, tt.value).ToBsonD()
			if !hasOperator(out, "a", tt.expected) {
				t.Fatalf("expected a.%s, got %s", tt.expected, toBSON(out))
			}
		})
	}
}

func TestCompareByValue_EmptyField(t *testing.T) {
	f := &mockField{name: ""}
	out := CompareByValue(f, GT, 5).ToBsonD()
	// name 为空时，operator 直接作为 key
	if !hasKey(out, "$gt") {
		t.Fatalf("expected $gt as key when field name is empty, got %s", toBSON(out))
	}
}

func TestCompareByValue_NestedField(t *testing.T) {
	parent := &mockField{name: "parent"}
	child := &mockField{name: "child", parent: parent}
	out := CompareByValue(child, GT, 5).ToBsonD()

	if !hasOperator(out, "parent.child", "$gt") {
		t.Fatalf("expected parent.child.$gt, got %s", toBSON(out))
	}
}

// ---------- TestCompareByField (exprFilter) ----------

func TestCompareByField(t *testing.T) {
	f1 := &mockField{name: "a"}
	f2 := &mockField{name: "b"}
	out := CompareByField(f1, GT, f2).ToBsonD()

	if !hasKey(out, "$expr") {
		t.Fatalf("expected $expr, got %s", toBSON(out))
	}
	exprVal, _ := getField(out, "$expr")
	exprDoc, ok := exprVal.(bson.D)
	if !ok {
		t.Fatalf("expected bson.D inside $expr, got %T", exprVal)
	}
	if !hasOperator2(exprDoc, "$gt") {
		t.Fatalf("expected $gt inside $expr, got %s", toBSON(exprDoc))
	}
}

func TestCompareByField_AllComparers(t *testing.T) {
	tests := []struct {
		name string
		c    Comparer
		op   string
	}{
		{"EQ", EQ, "$eq"},
		{"GT", GT, "$gt"},
		{"GTE", GTE, "$gte"},
		{"LT", LT, "$lt"},
		{"LTE", LTE, "$lte"},
		{"NE", NE, "$ne"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f1 := &mockField{name: "x"}
			f2 := &mockField{name: "y"}
			out := CompareByField(f1, tt.c, f2).ToBsonD()
			exprVal, _ := getField(out, "$expr")
			exprDoc := exprVal.(bson.D)
			if !hasOperator2(exprDoc, tt.op) {
				t.Fatalf("expected %s inside $expr, got %s", tt.op, toBSON(exprDoc))
			}
			// 验证值是 [x, y]
			for _, e := range exprDoc {
				if e.Key == tt.op {
					arr, ok := e.Value.(bson.A)
					if !ok || len(arr) != 2 || arr[0] != "$x" || arr[1] != "$y" {
						t.Fatalf("expected [x,y], got %#v", e.Value)
					}
				}
			}
		})
	}
}

// ---------- TestExist / NotExist / Type ----------

func TestExist(t *testing.T) {
	f := &mockField{name: "a"}
	out := Exist(f).ToBsonD()

	if !hasOperator(out, "a", "$exists") {
		t.Fatalf("expected a.$exists, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$exists" {
			if b, ok := e.Value.(bool); !ok || !b {
				t.Fatalf("expected true, got %v", e.Value)
			}
		}
	}
	// 验证是 PartialIndexFilter
	_, ok := Exist(f).(PartialIndexFilter)
	if !ok {
		t.Fatalf("Exist should return PartialIndexFilter")
	}
}

func TestNotExist(t *testing.T) {
	f := &mockField{name: "a"}
	out := NotExist(f).ToBsonD()

	if !hasOperator(out, "a", "$exists") {
		t.Fatalf("expected a.$exists, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$exists" {
			if b, ok := e.Value.(bool); !ok || b {
				t.Fatalf("expected false, got %v", e.Value)
			}
		}
	}
}

func TestType(t *testing.T) {
	f := &mockField{name: "a"}
	out := Type(f, bson.TypeInt32).ToBsonD()

	if !hasOperator(out, "a", "$type") {
		t.Fatalf("expected a.$type, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$type" {
			if e.Value != bson.TypeInt32 {
				t.Fatalf("expected %v, got %v", bson.TypeInt32, e.Value)
			}
		}
	}
	_, ok := Type(f, bson.TypeString).(PartialIndexFilter)
	if !ok {
		t.Fatalf("Type should return PartialIndexFilter")
	}
}

// ---------- TestFromBsonD / bsonD ----------

func TestFromBsonD(t *testing.T) {
	d := bson.D{{"a", 1}, {"b", bson.D{{"$gt", 2}}}}
	f := FromBsonD(d)
	out := f.ToBsonD()

	if len(out) != 2 {
		t.Fatalf("expected 2 elements, got %d, out=%s", len(out), toBSON(out))
	}
	if !hasKey(out, "a") || !hasKey(out, "b") {
		t.Fatalf("missing a or b, got %s", toBSON(out))
	}
}

// ---------- TestSameElemMatch ----------

func TestSameElemMatch(t *testing.T) {
	f := &mockField{name: "arr"}
	subFilter := FromBsonD(bson.D{{"x", bson.D{{"$gt", 5}}}})
	out := SameElemMatch(f, subFilter).ToBsonD()

	if !hasOperator(out, "arr", "$elemMatch") {
		t.Fatalf("expected arr.$elemMatch, got %s", toBSON(out))
	}
	val, _ := getField(out, "arr")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$elemMatch" {
			inner, ok := e.Value.(bson.D)
			if !ok {
				t.Fatalf("expected bson.D inside $elemMatch, got %T", e.Value)
			}
			if !hasOperator(inner, "x", "$gt") {
				t.Fatalf("expected x.$gt inside $elemMatch, got %s", toBSON(inner))
			}
		}
	}
}

// ---------- TestNew (base constructor) ----------

func TestNew(t *testing.T) {
	f := &mockField{name: "a"}
	out := New(f, "$gt", 5).ToBsonD()

	if !hasOperator(out, "a", "$gt") {
		t.Fatalf("expected a.$gt, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$gt" && e.Value != 5 {
			t.Fatalf("expected 5, got %v", e.Value)
		}
	}
}

func TestNew_EmptyFieldName(t *testing.T) {
	f := &mockField{name: ""}
	out := New(f, "$gt", 5).ToBsonD()

	// name 为空时，operator 直接作为 key
	if !hasKey(out, "$gt") {
		t.Fatalf("expected $gt as key, got %s", toBSON(out))
	}
}

// ---------- TestBase_Not (base.not method) ----------

func TestBase_Not(t *testing.T) {
	b := &base{
		f:        &mockField{name: "a"},
		operator: "$gt",
		value:    5,
	}
	notB := b.not()

	if notB.operator != "$not" {
		t.Fatalf("expected operator $not, got %s", notB.operator)
	}
	val, ok := notB.value.(bson.M)
	if !ok {
		t.Fatalf("expected bson.M, got %T", notB.value)
	}
	if val["$gt"] != 5 {
		t.Fatalf("expected $gt:5 inside bson.M, got %#v", val)
	}
	if notB.f.FullName() != "a" {
		t.Fatalf("expected field name a, got %s", notB.f.FullName())
	}
}

// ---------- TestAnd / Or / Nor constructors ----------

func TestAnd_ToBsonD(t *testing.T) {
	f1 := FromBsonD(bson.D{{"a", 1}})
	f2 := FromBsonD(bson.D{{"b", bson.D{{"$gt", 2}}}})
	out := And(f1, f2).ToBsonD()

	if hasKey(out, "$and") {
		t.Fatalf("should not have $and after flatten, got %s", toBSON(out))
	}
	if !hasKey(out, "a") || !hasKey(out, "b") {
		t.Fatalf("missing a or b, got %s", toBSON(out))
	}
}

func TestAnd_SingleFilter(t *testing.T) {
	f := FromBsonD(bson.D{{"a", 1}})
	out := And(f).ToBsonD()

	if hasKey(out, "$and") {
		t.Fatalf("single filter should not wrap $and, got %s", toBSON(out))
	}
	if !hasKey(out, "a") {
		t.Fatalf("missing a, got %s", toBSON(out))
	}
}

func TestAnd_MultipleConflicting(t *testing.T) {
	// 两个冲突的 a 条件，应该保留 $and
	f1 := FromBsonD(bson.D{{"a", bson.D{{"$gt", 1}}}})
	f2 := FromBsonD(bson.D{{"a", bson.D{{"$gt", 3}}}})
	out := And(f1, f2).ToBsonD()

	if !hasKey(out, "$and") {
		t.Fatalf("conflicting conditions should keep $and, got %s", toBSON(out))
	}
	and := extractAnd(out)
	if len(and) != 2 {
		t.Fatalf("expected 2 clauses in $and, got %d", len(and))
	}
}

func TestAndPartial_ToBsonD(t *testing.T) {
	f1 := AsPartialIndexFilter(FromBsonD(bson.D{{"a", 1}}))
	f2 := AsPartialIndexFilter(FromBsonD(bson.D{{"b", bson.D{{"$gt", 2}}}}))
	out := AndPartial(f1, f2).ToBsonD()

	if hasKey(out, "$and") {
		t.Fatalf("should not have $and after flatten, got %s", toBSON(out))
	}
	if !hasKey(out, "a") || !hasKey(out, "b") {
		t.Fatalf("missing a or b, got %s", toBSON(out))
	}
	_, ok := AndPartial(f1).(PartialIndexFilter)
	if !ok {
		t.Fatalf("AndPartial should return PartialIndexFilter")
	}
}

func TestOr_ToBsonD(t *testing.T) {
	f1 := FromBsonD(bson.D{{"a", 1}})
	f2 := FromBsonD(bson.D{{"b", bson.D{{"$gt", 2}}}})
	out := Or(f1, f2).ToBsonD()

	if !hasKey(out, "$or") {
		t.Fatalf("expected $or, got %s", toBSON(out))
	}
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 clauses in $or, got %d", len(or))
	}
}

func TestOr_SingleFilter(t *testing.T) {
	f := FromBsonD(bson.D{{"a", 1}})
	out := Or(f).ToBsonD()

	if !hasKey(out, "$or") {
		t.Fatalf("single filter in Or should keep $or, got %s", toBSON(out))
	}
	or := extractOr(out)
	if len(or) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(or))
	}
}

func TestOrPartial_ToBsonD(t *testing.T) {
	f1 := AsPartialIndexFilter(FromBsonD(bson.D{{"a", 1}}))
	f2 := AsPartialIndexFilter(FromBsonD(bson.D{{"b", bson.D{{"$gt", 2}}}}))
	out := OrPartial(f1, f2).ToBsonD()

	if !hasKey(out, "$or") {
		t.Fatalf("expected $or, got %s", toBSON(out))
	}
	or := extractOr(out)
	if len(or) != 2 {
		t.Fatalf("expected 2 clauses in $or, got %d", len(or))
	}
	_, ok := OrPartial(f1).(PartialIndexFilter)
	if !ok {
		t.Fatalf("OrPartial should return PartialIndexFilter")
	}
}

func TestNor_ToBsonD(t *testing.T) {
	f1 := FromBsonD(bson.D{{"a", 1}})
	f2 := FromBsonD(bson.D{{"b", bson.D{{"$gt", 2}}}})
	out := Nor(f1, f2).ToBsonD()

	if !hasKey(out, "$nor") {
		t.Fatalf("expected $nor, got %s", toBSON(out))
	}
	nor := extractNor(out)
	if len(nor) != 2 {
		t.Fatalf("expected 2 clauses in $nor, got %d", len(nor))
	}
}

func TestNor_SingleFilter(t *testing.T) {
	f := FromBsonD(bson.D{{"a", 1}})
	out := Nor(f).ToBsonD()

	if !hasKey(out, "$nor") {
		t.Fatalf("single filter in Nor should keep $nor, got %s", toBSON(out))
	}
	nor := extractNor(out)
	if len(nor) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(nor))
	}
}

// ---------- TestNot (not struct + ToBsonD) ----------

func TestNot_ToBsonD(t *testing.T) {
	f := FromBsonD(bson.D{{"a", bson.D{{"$gt", 5}}}})
	out := Not(f).ToBsonD()

	if !hasOperator(out, "a", "$not") {
		t.Fatalf("expected a.$not, got %s", toBSON(out))
	}
	val, _ := getField(out, "a")
	valDoc := val.(bson.D)
	for _, e := range valDoc {
		if e.Key == "$not" {
			notVal := e.Value.(bson.D)
			if !hasOperator2(notVal, "$gt") {
				t.Fatalf("expected $gt inside $not, got %s", toBSON(notVal))
			}
		}
	}
}

func TestNot_DoubleNegation(t *testing.T) {
	// Not(Not(f)) 应该等于 f
	f := FromBsonD(bson.D{{"a", bson.D{{"$gt", 5}}}})
	out := Not(Not(f)).ToBsonD()

	if hasKey(out, "$not") {
		t.Fatalf("double negation should be eliminated, got %s", toBSON(out))
	}
	if !hasOperator(out, "a", "$gt") {
		t.Fatalf("expected a.$gt after double negation, got %s", toBSON(out))
	}
}

// ---------- TestAsPartialIndexFilter ----------

func TestAsPartialIndexFilter(t *testing.T) {
	f := FromBsonD(bson.D{{"a", 1}})
	p := AsPartialIndexFilter(f)

	_, ok := p.(PartialIndexFilter)
	if !ok {
		t.Fatalf("should be PartialIndexFilter")
	}
	out := p.ToBsonD()
	if !hasKey(out, "a") {
		t.Fatalf("expected a, got %s", toBSON(out))
	}
}

// ---------- TestComparer_String ----------

func TestComparer_String(t *testing.T) {
	tests := []struct {
		c    Comparer
		want string
	}{
		{EQ, "$eq"},
		{GT, "$gt"},
		{GTE, "$gte"},
		{LT, "$lt"},
		{LTE, "$lte"},
		{NE, "$ne"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Fatalf("Comparer(%d).String() = %s, want %s", tt.c, got, tt.want)
			}
		})
	}
}
