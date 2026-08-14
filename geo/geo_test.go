package geo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- helpers ---

func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// --- SpherePoint tests ---

func TestSpherePoint_New(t *testing.T) {
	p := NewSpherePoint(116.46, 39.92)

	want := &SpherePoint{
		Type: "Point",
		C:    Coordinate{116.46, 39.92},
	}
	if !deepEqual(p, want) {
		t.Errorf("NewSpherePoint: got %+v, want %+v", p, want)
	}
}

func TestSpherePoint_LngOutOfRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for lng > 180")
		}
	}()
	_ = NewSpherePoint(200, 39.92)
}

func TestSpherePoint_LngNegativeOutOfRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for lng < -180")
		}
	}()
	_ = NewSpherePoint(-200, 39.92)
}

func TestSpherePoint_LatOutOfRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for lat > 90")
		}
	}()
	_ = NewSpherePoint(116.46, 100)
}

func TestSpherePoint_LatNegativeOutOfRange(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for lat < -90")
		}
	}()
	_ = NewSpherePoint(116.46, -100)
}

func TestSpherePoint_BsonTag(t *testing.T) {
	p := NewSpherePoint(116.46, 39.92)

	// 验证 bson tag 映射
	b, err := bson.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: unexpected error: %v", err)
	}

	var m bson.M
	err = bson.Unmarshal(b, &m)
	if err != nil {
		t.Fatalf("Unmarshal: unexpected error: %v", err)
	}

	if m["type"] != "Point" {
		t.Errorf("BsonTag type: got %v, want 'Point'", m["type"])
	}

	coords, ok := m["coordinates"].(bson.A)
	if !ok || len(coords) != 2 {
		t.Fatalf("BsonTag coordinates: got %v", m["coordinates"])
	}
	if coords[0] != 116.46 || coords[1] != 39.92 {
		t.Errorf("BsonTag coordinates: got %v, want [116.46, 39.92]", coords)
	}
}

// --- FlatPoint tests ---

func TestFlatPoint_New(t *testing.T) {
	p := NewFlatPoint(10.5, 20.3)

	if p.X() != 10.5 {
		t.Errorf("FlatPoint X: got %f, want 10.5", p.X())
	}
	if p.Y() != 20.3 {
		t.Errorf("FlatPoint Y: got %f, want 20.3", p.Y())
	}
}

func TestFlatPoint_Coordinate(t *testing.T) {
	p := Coordinate{10.5, 20.3}

	if p[0] != 10.5 || p[1] != 20.3 {
		t.Errorf("Coordinate: got %v, want [10.5, 20.3]", p)
	}
}

// --- MultiSpherePoint tests ---

func TestMultiSpherePoint_New(t *testing.T) {
	p1 := *NewSpherePoint(116.46, 39.92)
	p2 := *NewSpherePoint(121.47, 31.23)

	mp := NewMultiSpherePoint(p1, p2)

	if mp.Type != "MultiPoint" {
		t.Errorf("MultiSpherePoint Type: got %s, want 'MultiPoint'", mp.Type)
	}
	if len(mp.C) != 2 {
		t.Fatalf("MultiSpherePoint C: got %d coords, want 2", len(mp.C))
	}
	if mp.C[0][0] != 116.46 || mp.C[0][1] != 39.92 {
		t.Errorf("MultiSpherePoint C[0]: got %v", mp.C[0])
	}
	if mp.C[1][0] != 121.47 || mp.C[1][1] != 31.23 {
		t.Errorf("MultiSpherePoint C[1]: got %v", mp.C[1])
	}
}

// --- LineString tests ---

func TestLineString_New(t *testing.T) {
	coords := []Coordinate{{0, 0}, {1, 1}, {2, 2}}
	ls := NewLineString(coords)

	if ls.Type != "LineString" {
		t.Errorf("LineString Type: got %s, want 'LineString'", ls.Type)
	}
	if len(ls.C) != 3 {
		t.Fatalf("LineString C: got %d coords, want 3", len(ls.C))
	}
}

// --- MultiLineString tests ---

func TestMultiLineString_New(t *testing.T) {
	ls1 := *NewLineString([]Coordinate{{0, 0}, {1, 1}})
	ls2 := *NewLineString([]Coordinate{{2, 2}, {3, 3}})

	mls := NewMultiLineString(ls1, ls2)

	if mls.Type != "MultiLineString" {
		t.Errorf("MultiLineString Type: got %s, want 'MultiLineString'", mls.Type)
	}
	if len(mls.C) != 2 {
		t.Fatalf("MultiLineString C: got %d lines, want 2", len(mls.C))
	}
}

// --- Polygon tests ---

func TestPolygon_New(t *testing.T) {
	ring1 := Ring{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	ring2 := Ring{{0.2, 0.2}, {0.2, 0.8}, {0.8, 0.8}, {0.8, 0.2}, {0.2, 0.2}}

	p := NewPolygon(ring1, ring2)

	if p.Type != "Polygon" {
		t.Errorf("Polygon Type: got %s, want 'Polygon'", p.Type)
	}
	if len(p.C) != 2 {
		t.Fatalf("Polygon C: got %d rings, want 2", len(p.C))
	}
}

// --- MultiPolygon tests ---

func TestMultiPolygon_New(t *testing.T) {
	ring := Ring{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	p1 := *NewPolygon(ring)
	ring2 := Ring{{2, 2}, {2, 3}, {3, 3}, {3, 2}, {2, 2}}
	p2 := *NewPolygon(ring2)

	mp := NewMultiPolygon(p1, p2)

	if mp.Type != "MultiPolygon" {
		t.Errorf("MultiPolygon Type: got %s, want 'MultiPolygon'", mp.Type)
	}
	if len(mp.C) != 2 {
		t.Fatalf("MultiPolygon C: got %d polygons, want 2", len(mp.C))
	}
}

func TestMultiPolygon_Polygons(t *testing.T) {
	ring := Ring{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	p1 := *NewPolygon(ring)

	mp := NewMultiPolygon(p1)
	polygons := mp.Polygons()

	if len(polygons) != 1 {
		t.Fatalf("Polygons: got %d, want 1", len(polygons))
	}
	if polygons[0].Type != "Polygon" {
		t.Errorf("Polygons[0] Type: got %s, want 'Polygon'", polygons[0].Type)
	}
}

// --- Collection tests ---

func TestCollection_New(t *testing.T) {
	gc := NewGeoCollection()

	if gc.Type != "GeometryCollection" {
		t.Errorf("Collection Type: got %s, want 'GeometryCollection'", gc.Type)
	}
	if len(gc.C) != 0 {
		t.Errorf("Collection C: got %d, want empty", len(gc.C))
	}
}

func TestCollection_AddPoint(t *testing.T) {
	gc := NewGeoCollection()
	p := *NewSpherePoint(116.46, 39.92)

	gc.AddPoint(p)

	if len(gc.C) != 1 {
		t.Fatalf("AddPoint: got %d, want 1", len(gc.C))
	}
}

func TestCollection_AddMultiPoint(t *testing.T) {
	gc := NewGeoCollection()
	p1 := *NewSpherePoint(116.46, 39.92)
	p2 := *NewSpherePoint(121.47, 31.23)
	mp := *NewMultiSpherePoint(p1, p2)

	gc.AddMultiPoint(mp)

	if len(gc.C) != 1 {
		t.Fatalf("AddMultiPoint: got %d, want 1", len(gc.C))
	}
}

func TestCollection_AddLineString(t *testing.T) {
	gc := NewGeoCollection()
	ls := *NewLineString([]Coordinate{{0, 0}, {1, 1}})

	gc.AddLineString(ls)

	if len(gc.C) != 1 {
		t.Fatalf("AddLineString: got %d, want 1", len(gc.C))
	}
}

func TestCollection_AddPolygon(t *testing.T) {
	gc := NewGeoCollection()
	ring := Ring{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	p := *NewPolygon(ring)

	gc.AddPolygon(p)

	if len(gc.C) != 1 {
		t.Fatalf("AddPolygon: got %d, want 1", len(gc.C))
	}
}

func TestCollection_AddMultiPolygon(t *testing.T) {
	gc := NewGeoCollection()
	ring := Ring{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}
	mp := *NewMultiPolygon(*NewPolygon(ring))

	gc.AddMultiPolygon(mp)

	if len(gc.C) != 1 {
		t.Fatalf("AddMultiPolygon: got %d, want 1", len(gc.C))
	}
}

func TestCollection_MultipleAdds(t *testing.T) {
	gc := NewGeoCollection()
	p := *NewSpherePoint(116.46, 39.92)
	ls := *NewLineString([]Coordinate{{0, 0}, {1, 1}})

	gc.AddPoint(p)
	gc.AddLineString(ls)

	if len(gc.C) != 2 {
		t.Fatalf("MultipleAdds: got %d, want 2", len(gc.C))
	}
}
