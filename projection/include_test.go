package projection

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ─── mock field.Field ───

type mockField struct {
	name string
}

func (m mockField) FullName() string {
	return m.name
}

func f(name string) mockField {
	return mockField{name: name}
}

// ─── mock filter.Filter ───

type mockFilter struct {
	key   string
	value bson.D
}

func (m mockFilter) ToBsonD() bson.D {
	return bson.D{{Key: m.key, Value: m.value}}
}

func elemMatchFilter(key string, cond bson.D) mockFilter {
	return mockFilter{key: key, value: bson.D{{"$elemMatch", cond}}}
}

// ─── Tests ───

func TestArrayRoot(t *testing.T) {
	tests := []struct {
		input string
		out   string
	}{
		{
			"a.b.0.c",
			"a.b",
		},
		{
			"a.b.0",
			"a.b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			a := assert.New(t)
			root := arrayRoot(tt.input)
			a.Equal(tt.out, root)
		})
	}
}

func TestInclude_Basic(t *testing.T) {
	p := Include(f("name"), f("email")).Build()
	if len(p) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(p))
	}
	if p[0].Key != "name" || p[0].Value != 1 {
		t.Errorf("unexpected name: %+v", p[0])
	}
	if p[1].Key != "email" || p[1].Value != 1 {
		t.Errorf("unexpected email: %+v", p[1])
	}
}

func TestInclude_ExcludeID(t *testing.T) {
	p := Include(f("name")).Exclude_id().Build()
	found := false
	for _, e := range p {
		if e.Key == "_id" && e.Value == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("_id:0 not found in projection")
	}
}

func TestInclude_WithSearchRelevance(t *testing.T) {
	p := Include(f("title")).WithSearchRelevance("score").Build()
	found := false
	for _, e := range p {
		if e.Key == "score" {
			found = true
			val, ok := e.Value.(bson.D)
			if !ok || len(val) != 1 || val[0].Key != "$meta" {
				t.Errorf("unexpected search relevance value: %+v", e.Value)
			}
		}
	}
	if !found {
		t.Fatal("search relevance field not found")
	}
}

func TestInclude_DuplicateField_FirstWins(t *testing.T) {
	p := Include(f("name"), f("name")).Build()
	count := 0
	for _, e := range p {
		if e.Key == "name" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected name to appear once, got %d", count)
	}
}

func TestInclude_WholeParentWholeChild_KeepParent(t *testing.T) {
	p := Include(f("a"), f("a.b")).Build()
	for _, e := range p {
		if e.Key == "a.b" {
			t.Fatal("a.b should be discarded, but found in projection")
		}
	}
	found := false
	for _, e := range p {
		if e.Key == "a" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a should be kept")
	}
}

func TestInclude_WholeChildWholeParent_KeepParent(t *testing.T) {
	p := Include(f("a.b"), f("a")).Build()
	for _, e := range p {
		if e.Key == "a.b" {
			t.Fatal("a.b should be discarded, but found in projection")
		}
	}
	found := false
	for _, e := range p {
		if e.Key == "a" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a should be kept")
	}
}

func TestInclude_PartialWinsOverWhole_SameField(t *testing.T) {
	p := Include(f("scores"))
	p.Field(f("scores.0"))
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "scores" && e.Value == 1 {
			t.Fatal("whole scores:1 should be removed")
		}
	}
	found := false
	for _, e := range proj {
		if e.Key == "scores.0" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("scores.0 should exist")
	}
}

func TestInclude_PartialWinsOverWhole_ReverseOrder(t *testing.T) {
	p := Include(f("a.b.0.c"), f("a.b")).Build()
	for _, e := range p {
		if e.Key == "a.b" && e.Value == 1 {
			t.Fatal("whole a.b:1 should be discarded when partial a.b.0.c exists")
		}
	}
	found := false
	for _, e := range p {
		if e.Key == "a.b.0.c" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a.b.0.c should be kept")
	}
}

func TestInclude_PartialVsWholeChildSameArray_PartialWins(t *testing.T) {
	p := Include(f("a.b.0"), f("a.b.c")).Build()
	for _, e := range p {
		if e.Key == "a.b.c" {
			t.Fatal("a.b.c should be discarded")
		}
	}
	found := false
	for _, e := range p {
		if e.Key == "a.b.0" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a.b.0 should be kept")
	}
}

func TestInclude_PartialVsWholeChildSameArray_ReverseOrder(t *testing.T) {
	p := Include(f("a.b.c"), f("a.b.0")).Build()
	for _, e := range p {
		if e.Key == "a.b.c" {
			t.Fatal("a.b.c should be discarded")
		}
	}
	found := false
	for _, e := range p {
		if e.Key == "a.b.0" && e.Value == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("a.b.0 should be kept")
	}
}

func TestInclude_PartialPlusPartialSameAncestor_FirstWins(t *testing.T) {
	p := Include(f("scores.0"), f("scores.1")).Build()
	count := 0
	for _, e := range p {
		if e.Key == "scores.0" || e.Key == "scores.1" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 partial, got %d", count)
	}
	for _, e := range p {
		if e.Key == "scores.1" {
			t.Fatal("scores.1 should be discarded")
		}
	}
}

func TestIncludeWithSlice_Basic(t *testing.T) {
	p := IncludeWithSlice(f("comments"), 5).Build()
	if len(p) != 1 {
		t.Fatalf("expected 1 field, got %d", len(p))
	}
	if p[0].Key != "comments" {
		t.Fatalf("unexpected key: %s", p[0].Key)
	}
	val, ok := p[0].Value.(bson.D)
	if !ok || len(val) != 1 || val[0].Key != "$slice" {
		t.Fatalf("unexpected value: %+v", p[0].Value)
	}
}

func TestIncludeWithSliceRange_Basic(t *testing.T) {
	p := IncludeWithSliceRange(f("comments"), 2, 5).Build()
	if len(p) != 1 {
		t.Fatalf("expected 1 field, got %d", len(p))
	}
	val, ok := p[0].Value.(bson.D)
	if !ok || len(val) != 1 || val[0].Key != "$slice" {
		t.Fatalf("unexpected value: %+v", p[0].Value)
	}
}

func TestIncludeWithElemMatch_Basic(t *testing.T) {
	cond := bson.D{{"$gt", 80}}
	p := IncludeWithElemMatch(elemMatchFilter("scores", cond)).Build()
	if len(p) != 1 {
		t.Fatalf("expected 1 field, got %d", len(p))
	}
	if p[0].Key != "scores" {
		t.Fatalf("unexpected key: %s", p[0].Key)
	}
	val, ok := p[0].Value.(bson.D)
	if !ok || len(val) != 1 || val[0].Key != "$elemMatch" {
		t.Fatalf("unexpected value: %+v", p[0].Value)
	}
}

func TestIncludeWithFirstMatch_Basic(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores")).Build()
	if len(p) != 1 {
		t.Fatalf("expected 1 field, got %d", len(p))
	}
	if p[0].Key != "scores.$" {
		t.Fatalf("unexpected key: %s", p[0].Key)
	}
	if p[0].Value != 1 {
		t.Fatalf("unexpected value: %+v", p[0].Value)
	}
}

func TestIncludeWithSlice_ThenFieldSameField_Discarded(t *testing.T) {
	p := IncludeWithSlice(f("scores"), 5)
	p.Field(f("scores"))
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "scores" && e.Value == 1 {
			t.Fatal("whole scores:1 should be discarded")
		}
	}
	val, ok := proj[0].Value.(bson.D)
	if !ok || val[0].Key != "$slice" {
		t.Fatal("slice should be preserved")
	}
}

func TestIncludeWithSlice_ThenFieldDifferentField_Allowed(t *testing.T) {
	p := IncludeWithSlice(f("comments"), 5)
	p.Field(f("tags.0"))
	proj := p.Build()
	if len(proj) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(proj))
	}
}

func TestIncludeWithFirstMatch_ThenFieldSameArray_Discarded(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("scores.0"))
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "scores.0" {
			t.Fatal("scores.0 should be discarded")
		}
	}
	if proj[0].Key != "scores.$" {
		t.Fatalf("expected scores.$, got %s", proj[0].Key)
	}
}

func TestIncludeWithFirstMatch_ThenFieldDifferentArray_Allowed(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("tags.0"))
	proj := p.Build()
	if len(proj) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(proj))
	}
}

func TestInclude_ExcludeID_ThenIncludeID_Discarded(t *testing.T) {
	p := Include(f("name")).Exclude_id()
	p.Field(f("_id"))
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "_id" && e.Value == 1 {
			t.Fatal("_id:1 should be discarded")
		}
	}
	found := false
	for _, e := range proj {
		if e.Key == "_id" && e.Value == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("_id:0 should be preserved")
	}
}

func TestInclude_WithSearchRelevance_OverridesWhole(t *testing.T) {
	p := Include(f("score"))
	p.WithSearchRelevance("score")
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "score" {
			val, ok := e.Value.(bson.D)
			if !ok || val[0].Key != "$meta" {
				t.Fatalf("score should be $meta, got %+v", e.Value)
			}
		}
	}
}

func TestInclude_WithSearchRelevance_ThenWhole_Discarded(t *testing.T) {
	p := Include(f("score")).WithSearchRelevance("score")
	p.Field(f("score"))
	proj := p.Build()
	for _, e := range proj {
		if e.Key == "score" && e.Value == 1 {
			t.Fatal("score:1 should be discarded when $meta exists")
		}
	}
}

func toBSON(d bson.D) string {
	raw, _ := bson.Marshal(d)
	var out bson.D
	_ = bson.Unmarshal(raw, &out)
	bs, _ := bson.MarshalExtJSON(out, false, false)
	return string(bs)
}

func TestInclude_ChainedUsage(t *testing.T) {
	p := Include(f("name"), f("email")).
		Exclude_id().
		WithSearchRelevance("score").
		Build()
	if len(p) != 4 {
		t.Fatalf("expected 4 fields, got %d\nret=%s", len(p), toBSON(p))
	}
}

func TestIncludeWithSlice_ThenFirstMatch_SameArray_Discarded(t *testing.T) {
	p := IncludeWithSlice(f("scores"), 5)
	p.Field(f("scores.$"))
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	val, ok := proj[0].Value.(bson.D)
	if !ok || val[0].Key != "$slice" {
		t.Fatal("$slice should be preserved, scores.$ should be discarded")
	}
}

func TestIncludeWithFirstMatch_ThenSlice_SameArray_Discarded(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("scores")) // whole, should be discarded
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	if proj[0].Key != "scores.$" {
		t.Fatalf("expected scores.$, got %s", proj[0].Key)
	}
}

func TestIncludeWithElemMatch_ThenFirstMatch_SameArray_Discarded(t *testing.T) {
	cond := bson.D{{"$gt", 80}}
	p := IncludeWithElemMatch(elemMatchFilter("scores", cond))
	p.Field(f("scores.$"))
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	val, ok := proj[0].Value.(bson.D)
	if !ok || val[0].Key != "$elemMatch" {
		t.Fatal("$elemMatch should be preserved, scores.$ should be discarded")
	}
}

func TestIncludeWithFirstMatch_ThenWhole_Discarded(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("scores"))
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	if proj[0].Key != "scores.$" {
		t.Fatalf("expected scores.$, got %s", proj[0].Key)
	}
}

func TestIncludeWithFirstMatch_ThenIndex_SameArray_Discarded(t *testing.T) {
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("scores.0"))
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	if proj[0].Key != "scores.$" {
		t.Fatalf("expected scores.$, got %s", proj[0].Key)
	}
}

func TestInclude_PartialWithDollarVsPartialWithIndex_SameArray(t *testing.T) {
	// scores.$ written first, scores.1 written second → $ wins
	p := IncludeWithFirstMatch(f("scores"))
	p.Field(f("scores.1"))
	proj := p.Build()
	if len(proj) != 1 {
		t.Fatalf("expected 1 field, got %d", len(proj))
	}
	if proj[0].Key != "scores.$" {
		t.Fatalf("expected scores.$, got %s", proj[0].Key)
	}
}
