package fields

import (
	"github.com/xpwu/go-mongodb/filter"
	"testing"

	"github.com/xpwu/go-mongodb/geo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ==================== SpherePointField 测试 ====================

func TestNewSpherePointField(t *testing.T) {
	f := NewSpherePointField("location")
	if f == nil {
		t.Fatal("NewSpherePointField: returned nil")
	}
	if f.FullName() != "location" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "location")
	}
}

func TestSpherePointField_Index2D(t *testing.T) {
	f := NewSpherePointField("location")
	key := f.Index2D()
	got := key.ToBsonD()

	want := bson.D{{"location", "2dsphere"}}
	if !bsonDEqual(got, want) {
		t.Errorf("Index2D: got %v, want %v", got, want)
	}
}

func TestSpherePointField_Index2DWith(t *testing.T) {
	f := NewSpherePointField("location")
	key := f.Index2DWith(3)
	got := key.ToBsonD()

	want := bson.D{{"location", "2dsphere"}}
	if !bsonDEqual(got, want) {
		t.Errorf("Index2DWith ToBsonD: got %v, want %v", got, want)
	}

	// 验证 options 包含 2dsphereIndexVersion
	opts := key.Options()
	found := false
	for _, e := range opts {
		if e.Key == "2dsphereIndexVersion" {
			found = true
			if e.Value != 3 {
				t.Errorf("2dsphereIndexVersion: got %v, want 3", e.Value)
			}
		}
	}
	if !found {
		t.Errorf("Index2DWith Options: expected 2dsphereIndexVersion, got %v", opts)
	}
}

func TestSpherePointField_WithinPoly(t *testing.T) {
	f := NewSpherePointField("location")
	polygon := geo.NewPolygon(
		geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}},
	)
	flt := f.WithinPoly(*polygon)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$geoWithin", bson.M{"$geometry": *polygon}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinPoly: \ngot  %v, \nwant %v", got, want)
	}
}

func TestSpherePointField_WithinMulPoly(t *testing.T) {
	f := NewSpherePointField("location")
	polygon1 := geo.NewPolygon(
		geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}},
	)
	polygon2 := geo.NewPolygon(
		geo.Ring{{20, 20}, {20, 30}, {30, 30}, {30, 20}, {20, 20}},
	)
	mp := geo.NewMultiPolygon([]geo.Polygon{*polygon1, *polygon2}...)

	flt := f.WithinMulPoly(*mp)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$geoWithin", bson.M{"$geometry": *mp}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinMulPoly: \ngot  %v, \nwant %v", got, want)
	}
}

func TestSpherePointField_WithinBigPoly(t *testing.T) {
	f := NewSpherePointField("location")
	ring := geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}

	flt := f.WithinBigPoly(ring)
	got := flt.ToBsonD()

	// 验证结构: location: {$geoWithin: {$geometry: {type:"Polygon", coordinates:[ring], crs:...}}}
	if len(got) != 1 || got[0].Key != "location" {
		t.Fatalf("WithinBigPoly: unexpected structure %v", got)
	}
	inner, ok := got[0].Value.(bson.D)
	if !ok || len(inner) != 1 || inner[0].Key != "$geoWithin" {
		t.Fatalf("WithinBigPoly: expected $geoWithin, got %v", got)
	}
	geoM, ok := inner[0].Value.(bson.M)
	if !ok {
		t.Fatalf("WithinBigPoly: expected bson.M, got %T", inner[0].Value)
	}
	v, ok := geoM["$geometry"].(bson.M)
	if !ok {
		t.Fatalf("WithinBigPoly: expected bson.M, got %T", geoM["$geometry"])
	}
	if v["type"] != "Polygon" {
		t.Errorf("WithinBigPoly type: got %v, want Polygon", geoM["type"])
	}
}

func TestSpherePointField_WithinCircle(t *testing.T) {
	f := NewSpherePointField("location")
	center := geo.Coordinate{116.46, 39.92}
	radians := 0.01

	flt := f.WithinCircle(center, radians)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$geoWithin",
		bson.M{"$centerSphere": bson.A{center, radians}}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinCircle: got %v, want %v", got, want)
	}
}

func TestSpherePointField_Near(t *testing.T) {
	f := NewSpherePointField("location")
	point := geo.NewSpherePoint(116.46, 39.92)
	maxDist := 1000.0

	flt := f.Near(*point, maxDist)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$nearSphere",
		bson.M{"$geometry": *point, "$maxDistance": maxDist}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Near: \ngot  %v, \nwant %v", got, want)
	}
}

func TestSpherePointField_Near_WithMinDistance(t *testing.T) {
	f := NewSpherePointField("location")
	point := geo.NewSpherePoint(116.46, 39.92)
	maxDist := 5000.0
	minDist := 1000.0

	flt := f.Near(*point, maxDist, minDist)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$nearSphere",
		bson.M{"$geometry": *point, "$maxDistance": maxDist, "$minDistance": minDist}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("Near with minDistance: got %v, want %v", got, want)
	}
}

func TestSpherePointField_Near_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Near: expected panic for too many args, got none")
		}
	}()
	f := NewSpherePointField("location")
	point := geo.NewSpherePoint(116.46, 39.92)
	_ = f.Near(*point, 1000.0, 500.0, 100.0) // 3 extra args, should panic
}

func TestSpherePointField_IntersectPoly(t *testing.T) {
	f := NewSpherePointField("location")
	polygon := geo.NewPolygon(
		geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}},
	)

	flt := f.IntersectPoly(*polygon)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$geoIntersects", bson.M{"$geometry": *polygon}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("IntersectPoly: got %v, want %v", got, want)
	}
}

func TestSpherePointField_IntersectLineString(t *testing.T) {
	f := NewSpherePointField("location")
	ls := geo.NewLineString(
		[]geo.Coordinate{{0, 0}, {10, 10}, {20, 5}}...,
	)

	flt := f.IntersectLineString(*ls)
	got := flt.ToBsonD()

	want := bson.D{{"location", bson.D{{"$geoIntersects", bson.M{"$geometry": *ls}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("IntersectLineString: got %v, want %v", got, want)
	}
}

func TestSpherePointField_IntersectBigPoly(t *testing.T) {
	f := NewSpherePointField("location")
	ring := geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}

	flt := f.IntersectBigPoly(ring)
	got := flt.ToBsonD()

	if len(got) != 1 || got[0].Key != "location" {
		t.Fatalf("IntersectBigPoly: got %v", got)
	}
	inner := got[0].Value.(bson.D)
	if inner[0].Key != "$geoIntersects" {
		t.Errorf("IntersectBigPoly: expected $geoIntersects, got %v", inner[0].Key)
	}
}

func TestSpherePointField_Coordinate(t *testing.T) {
	f := NewSpherePointField("location")
	flatPoint := f.Coordinate()

	if flatPoint == nil {
		t.Fatal("Coordinate: returned nil")
	}
	got := flatPoint.FullName()
	if got != "location.coordinates" {
		t.Errorf("Coordinate FullName: got %v, want %v", got, "location.coordinates")
	}
}

// ==================== FlatPointField 测试 ====================

func TestNewFlatPointField(t *testing.T) {
	f := NewFlatPointField("pos")
	if f == nil {
		t.Fatal("NewFlatPointField: returned nil")
	}
	if f.FullName() != "pos" {
		t.Errorf("FullName: got %v, want %v", f.FullName(), "pos")
	}
}

func TestFlatPointField_Index2D(t *testing.T) {
	f := NewFlatPointField("pos")
	key := f.Index2D()
	got := key.ToBsonD()

	want := bson.D{{"pos", "2d"}}
	if !bsonDEqual(got, want) {
		t.Errorf("FlatPoint Index2D: got %v, want %v", got, want)
	}
}

func TestFlatPointField_Index2DWith(t *testing.T) {
	f := NewFlatPointField("pos")
	key := f.Index2DWith(30)
	got := key.ToBsonD()

	want := bson.D{{"pos", "2d"}}
	if !bsonDEqual(got, want) {
		t.Errorf("FlatPoint Index2DWith ToBsonD: got %v, want %v", got, want)
	}

	opts := key.Options()
	found := false
	for _, e := range opts {
		if e.Key == "bits" {
			found = true
			if e.Value != 30 {
				t.Errorf("bits: got %v, want 30", e.Value)
			}
		}
	}
	if !found {
		t.Errorf("FlatPoint Index2DWith Options: expected bits, got %v", opts)
	}
}

func TestFlatPointField_Index2DWithRange(t *testing.T) {
	f := NewFlatPointField("pos")
	key := f.Index2DWithRange(-100, 100)
	got := key.ToBsonD()

	want := bson.D{{"pos", "2d"}}
	if !bsonDEqual(got, want) {
		t.Errorf("FlatPoint Index2DWithRange ToBsonD: got %v, want %v", got, want)
	}

	opts := key.Options()
	hasMin, hasMax := false, false
	for _, d := range opts {
		if d.Key == "min" {
			hasMin = true
		}
		if d.Key == "max" {
			hasMax = true
		}
	}
	if !hasMin || !hasMax {
		t.Errorf("FlatPoint Index2DWithRange Options: expected min and max, got %v", opts)
	}
}

func TestFlatPointField_WithinBox(t *testing.T) {
	f := NewFlatPointField("pos")
	bottomLeft := geo.Coordinate{0, 0}
	upperRight := geo.Coordinate{10, 10}

	flt := f.WithinBox(bottomLeft, upperRight)
	got := flt.ToBsonD()

	want := bson.D{{"pos", bson.D{{"$geoWithin",
		bson.M{"$box": bson.A{bottomLeft, upperRight}}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinBox: got %v, want %v", got, want)
	}
}

func TestFlatPointField_WithinRing(t *testing.T) {
	f := NewFlatPointField("pos")
	ring := geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}

	flt := f.WithinRing(ring)
	got := flt.ToBsonD()

	want := bson.D{{"pos", bson.D{{"$geoWithin",
		bson.M{"$polygon": ring}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinRing: got %v, want %v", got, want)
	}
}

func TestFlatPointField_WithinCircle(t *testing.T) {
	f := NewFlatPointField("pos")
	center := geo.Coordinate{50, 50}
	radius := float32(10.5)

	flt := f.WithinCircle(center, radius)
	got := flt.ToBsonD()

	want := bson.D{{"pos", bson.D{{"$geoWithin",
		bson.M{"$center": bson.A{center, radius}}}}}}
	if !bsonDEqual(got, want) {
		t.Errorf("WithinCircle: got %v, want %v", got, want)
	}
}

func TestFlatPointField_Near(t *testing.T) {
	f := NewFlatPointField("pos")
	point := geo.Coordinate{50, 50}
	maxDist := 100.0

	flt := f.Near(point, maxDist)
	got := flt.ToBsonD()

	// FlatPoint Near = And(filter.New(f, "$near", point), filter.New(f, "$maxDistance", maxDist))
	// filter.New with name="pos", operator="$near", value=point
	//   → bson.D{{"pos", bson.D{{"$near", point}}}}
	// filter.New with name="pos", operator="$maxDistance", value=100
	//   → bson.D{{"pos", bson.D{{"$maxDistance", 100}}}}
	// And → flattenDoc(bson.D{{"$and", bson.A{...}}})
	want := bson.D{{"$and", bson.A{
		bson.D{{"pos", bson.D{{"$near", point}}}},
		bson.D{{"pos", bson.D{{"$maxDistance", maxDist}}}},
	}}}
	if !bsonDEqual(got, filter.FlattenDoc(want)) {
		t.Errorf("FlatPoint Near: \ngot  %v, \nwant %v", got, want)
	}
}

// ==================== NewSpherePoint 边界测试 ====================

func TestNewSpherePoint_Valid(t *testing.T) {
	p := geo.NewSpherePoint(116.46, 39.92)
	if p.Type != "Point" {
		t.Errorf("Type: got %v, want Point", p.Type)
	}
	if p.C[0] != 116.46 || p.C[1] != 39.92 {
		t.Errorf("Coordinates: got %v, want [116.46, 39.92]", p.C)
	}
}

func TestNewSpherePoint_Panic_LngTooHigh(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for lng > 180")
		}
	}()
	_ = geo.NewSpherePoint(181, 0)
}

func TestNewSpherePoint_Panic_LngTooLow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for lng < -180")
		}
	}()
	_ = geo.NewSpherePoint(-181, 0)
}

func TestNewSpherePoint_Panic_LatTooHigh(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for lat > 90")
		}
	}()
	_ = geo.NewSpherePoint(0, 91)
}

func TestNewSpherePoint_Panic_LatTooLow(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for lat < -90")
		}
	}()
	_ = geo.NewSpherePoint(0, -91)
}

// ==================== Geo 类型构造测试 ====================

func TestNewLineString(t *testing.T) {
	ls := geo.NewLineString(
		[]geo.Coordinate{{0, 0}, {10, 10}, {20, 5}}...,
	)
	if ls.Type != "LineString" {
		t.Errorf("Type: got %v, want LineString", ls.Type)
	}
	if len(ls.C) != 3 {
		t.Errorf("Coordinates length: got %v, want 3", len(ls.C))
	}
}

func TestNewPolygon(t *testing.T) {
	ring := geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}
	p := geo.NewPolygon(ring)
	if p.Type != "Polygon" {
		t.Errorf("Type: got %v, want Polygon", p.Type)
	}
	if len(p.C) != 1 {
		t.Errorf("Rings: got %v, want 1", len(p.C))
	}
}

func TestNewMultiPolygon(t *testing.T) {
	r1 := geo.Ring{{0, 0}, {0, 10}, {10, 10}, {10, 0}, {0, 0}}
	r2 := geo.Ring{{20, 20}, {20, 30}, {30, 30}, {30, 20}, {20, 20}}
	p1 := *geo.NewPolygon(r1)
	p2 := *geo.NewPolygon(r2)
	mp := geo.NewMultiPolygon(p1, p2)
	if mp.Type != "MultiPolygon" {
		t.Errorf("Type: got %v, want MultiPolygon", mp.Type)
	}
}

func TestNewMultiSpherePoint(t *testing.T) {
	p1 := *geo.NewSpherePoint(0, 0)
	p2 := *geo.NewSpherePoint(1, 1)
	msp := geo.NewMultiSpherePoint(p1, p2)
	if msp.Type != "MultiPoint" {
		t.Errorf("Type: got %v, want MultiPoint", msp.Type)
	}
	if len(msp.C) != 2 {
		t.Errorf("Coordinates: got %v, want 2", len(msp.C))
	}
}

func TestNewGeoCollection(t *testing.T) {
	gc := geo.NewGeoCollection()
	if gc.Type != "GeometryCollection" {
		t.Errorf("Type: got %v, want GeometryCollection", gc.Type)
	}
	if len(gc.C) != 0 {
		t.Errorf("Geometries: got %v, want empty", len(gc.C))
	}
}

func TestGeoCollection_AddPoint(t *testing.T) {
	gc := geo.NewGeoCollection()
	p := *geo.NewSpherePoint(10, 20)
	gc.AddPoint(p)
	if len(gc.C) != 1 {
		t.Errorf("AddPoint: got %v items, want 1", len(gc.C))
	}
}
