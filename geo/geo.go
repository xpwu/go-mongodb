package geo

import "fmt"

type FlatPoint []float64

func (p FlatPoint) X() float64 {
	return p[0]
}

func (p FlatPoint) Y() float64 {
	return p[1]
}

func NewFlatPoint(x, y float64) *FlatPoint {
	return &FlatPoint{x, y}
}

type Coordinate = FlatPoint

type SpherePoint struct {
	Type string     `bson:"type"`
	C    Coordinate `bson:"coordinates"`
}

func NewSpherePoint(lng float64, lat float64) *SpherePoint {
	if lng > 180 || lng < -180 || lat > 90 || lat < -90 {
		panic(fmt.Sprintf("lng(%f) lat(%f) error", lng, lat))
	}

	return &SpherePoint{
		Type: `Point`,
		C:    Coordinate{lng, lat},
	}
}

type MultiSpherePoint struct {
	Type string       `bson:"type"`
	C    []Coordinate `bson:"coordinates"`
}

func NewMultiSpherePoint(s1 SpherePoint, ss ...SpherePoint) *MultiSpherePoint {
	ret := &MultiSpherePoint{
		Type: "MultiPoint",
		C:    make([]Coordinate, 1, len(ss)+1),
	}

	ret.C[0] = s1.C
	for _, s := range ss {
		ret.C = append(ret.C, s.C)
	}

	return ret
}

type LineString struct {
	Type string       `bson:"type"`
	C    []Coordinate `bson:"coordinates"`
}

func NewLineString(c ...Coordinate) *LineString {
	return &LineString{
		Type: "LineString",
		C:    c,
	}
}

type LintStringC = []Coordinate

type MultiLineString struct {
	Type string        `bson:"type"`
	C    []LintStringC `bson:"coordinates"`
}

func NewMultiLineString(ls ...LineString) *MultiLineString {
	ret := &MultiLineString{
		Type: "MultiLineString",
		C:    []LintStringC{},
	}
	for _, l := range ls {
		ret.C = append(ret.C, l.C)
	}

	return ret
}

//Ring the first and last coordinates in the array must be the same
type Ring = []Coordinate

type Polygon struct {
	Type string `bson:"type"`
	C    []Ring `bson:"coordinates"`
}

func NewPolygon(r1 Ring, rs ...Ring) *Polygon {
	ret := &Polygon{
		Type: "Polygon",
		C:    make([]Ring, 1, len(rs)+1),
	}

	ret.C[0] = r1
	ret.C = append(ret.C, rs...)

	return ret
}

type PolygonC = []Ring

type MultiPolygon struct {
	Type string     `bson:"type"`
	C    []PolygonC `bson:"coordinates"`
}

func NewMultiPolygon(pgs ...Polygon) *MultiPolygon {
	ret := &MultiPolygon{
		Type: "MultiPolygon",
		C:    make([]PolygonC, len(pgs)),
	}
	for i, pg := range pgs {
		ret.C[i] = pg.C
	}

	return ret
}

func (m *MultiPolygon) Polygons() []Polygon {
	ret := make([]Polygon, len(m.C))

	for i, c := range m.C {
		ret[i] = *NewPolygon(c[0], c[1:]...)
	}

	return ret
}

type Collection struct {
	Type string `bson:"type"`
	C    []any  `bson:"geometries"`
}

func NewGeoCollection() *Collection {
	return &Collection{Type: "GeometryCollection"}
}

func (g *Collection) AddPoint(p SpherePoint) {
	g.C = append(g.C, p)
}

func (g *Collection) AddMultiPoint(p MultiSpherePoint) {
	g.C = append(g.C, p)
}

func (g *Collection) AddLineString(p LineString) {
	g.C = append(g.C, p)
}

func (g *Collection) AddPolygon(p Polygon) {
	g.C = append(g.C, p)
}

func (g *Collection) AddMultiPolygon(p MultiPolygon) {
	g.C = append(g.C, p)
}
