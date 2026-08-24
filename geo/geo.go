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

func NewLineString(c0, c1 Coordinate, c ...Coordinate) *LineString {
	cs := make([]Coordinate, 2, len(c)+2)
	cs[0] = c0
	cs[1] = c1
	return &LineString{
		Type: "LineString",
		C:    append(cs, c...),
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

//Ring the first and last coordinates in the array must be the same, and len(Coordinates) >= 4
type Ring = []Coordinate

func checkRing(r Ring) {
	if len(r) < 4 {
		panic("geo: Polygon ring must have at least 4 coordinates (min 3 unique + closing)")
	}
	if r[0][0] != r[len(r)-1][0] || r[0][1] != r[len(r)-1][1] {
		panic("geo: Polygon ring first and last coordinate must be identical")
	}
}

type Polygon struct {
	Type string `bson:"type"`
	C    []Ring `bson:"coordinates"`
}

func NewPolygon(r1 Ring, rs ...Ring) *Polygon {
	checkRing(r1)
	for _, r := range rs {
		checkRing(r)
	}

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

func (m MultiPolygon) Polygons() []Polygon {
	ret := make([]Polygon, len(m.C))

	for i, c := range m.C {
		ret[i] = *NewPolygon(c[0], c[1:]...)
	}

	return ret
}

// Geometry 是所有 GeoJSON 几何类型的标记接口。
// geoType() 不导出，确保只有本包内的类型能实现该接口。
type Geometry interface {
	geoType() string
}

func (s SpherePoint) geoType() string      { return s.Type }
func (m MultiSpherePoint) geoType() string { return m.Type }
func (l LineString) geoType() string       { return l.Type }
func (m MultiLineString) geoType() string  { return m.Type }
func (p Polygon) geoType() string          { return p.Type }
func (m MultiPolygon) geoType() string     { return m.Type }
func (c Collection) geoType() string       { return c.Type }

type Collection struct {
	Type string     `bson:"type"`
	C    []Geometry `bson:"geometries"`
}

func NewGeoCollection() *Collection {
	return &Collection{Type: "GeometryCollection"}
}

func (c *Collection) AddPoint(p SpherePoint) {
	c.C = append(c.C, p)
}

func (c *Collection) AddMultiPoint(p MultiSpherePoint) {
	c.C = append(c.C, p)
}

func (c *Collection) AddLineString(p LineString) {
	c.C = append(c.C, p)
}

func (c *Collection) AddPolygon(p Polygon) {
	c.C = append(c.C, p)
}

func (c *Collection) AddMultiPolygon(p MultiPolygon) {
	c.C = append(c.C, p)
}

func (c *Collection) AddCollection(p Collection) {
	c.C = append(c.C, p)
}
