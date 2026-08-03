package fields

import (
	"github.com/xpwu/go-mongodb/field"
	filter2 "github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/geo"
	index2 "github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/updater"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SpherePointField interface {
	field.Field
	filter2.BaseFilter[geo.SpherePoint]
	updater.BaseUpdater[geo.SpherePoint]
	index2.BaseKey
	Index2D() index2.Key
	// Index2DWith overrides the default version and specify a different version for your 2dsphere index.
	//
	// https://www.mongodb.com/docs/manual/core/indexes/index-types/geospatial/2dsphere/2dsphere-index-versions/#2dsphere-index-versions
	Index2DWith(version int) index2.Key
	WithinPoly(polygon geo.Polygon) filter2.Filter
	WithinMulPoly(polygon geo.MultiPolygon) filter2.Filter
	WithinBigPoly(ring geo.Ring) filter2.Filter
	WithinCircle(center geo.Coordinate, radians float64) filter2.Filter
	// Near returns the documents from nearest to farthest.
	// Near 根据分析，GeoJson 是按照WGS84的标准来计算的，定死半径为地球半径，所以maxDistance与minDistance使用了单位"米"
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/near/#mongodb-query-op.-near
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/maxDistance/#mongodb-query-op.-maxDistance
	Near(point geo.SpherePoint, maxDistance float64, minDistance ...float64) filter2.Filter
	Coordinate() FlatPointField
}

type spherePointField struct {
	BaseField[geo.SpherePoint]
}

func NewSpherePointField(name string) SpherePointField {
	return &spherePointField{BaseField[geo.SpherePoint]{name: name}}
}

func (p *spherePointField) Index2D() index2.Key {
	return index2.NewKey(p, index2.KeyType2dSphere)
}

type versionKey struct {
	fieldName string
	version   int
}

func (vk *versionKey) ToBsonD() bson.D {
	return bson.D{{vk.fieldName, index2.KeyType2dSphere}}
}

func (vk *versionKey) Options() bson.D {
	return bson.D{{"2dsphereIndexVersion", vk.version}}
}

func (p *spherePointField) Index2DWith(version int) index2.Key {
	return &versionKey{
		fieldName: p.FullName(),
		version:   version,
	}
}

func (p *spherePointField) WithinPoly(polygon geo.Polygon) filter2.Filter {
	return filter2.New(p, "$geoWithin", bson.M{"$geometry": polygon})
}

func (p *spherePointField) WithinMulPoly(polygon geo.MultiPolygon) filter2.Filter {
	return filter2.New(p, "$geoWithin", bson.M{"$geometry": polygon})
}

func bigPolyQuery(ring geo.Ring) bson.M {
	return bson.M{
		"type":        "Polygon",
		"coordinates": []geo.Ring{ring},
		"crs": bson.M{
			"type":       "name",
			"properties": bson.M{"name": "urn:x-mongodb:crs:strictwinding:EPSG:4326"}}}
}

func (p *spherePointField) WithinBigPoly(ring geo.Ring) filter2.Filter {
	return filter2.New(p, "$geoWithin",
		bson.M{"$geometry": bigPolyQuery(ring)})
}

// WithinCircle defines a circle for a geospatial query that uses spherical geometry.
// The circle's radius measured in radians
//
// https://www.mongodb.com/docs/manual/reference/operator/query/centerSphere/#example
func (p *spherePointField) WithinCircle(center geo.Coordinate, radians float64) filter2.Filter {
	return filter2.New(p, "$geoWithin",
		bson.M{"$centerSphere": bson.A{center, radians}})
}

// Near GeoJson 是按照WGS84的标准来计算的，定死半径为地球半径，所以 maxDistance minDistance 使用了单位"米"
//
// https://docs.mongodb.com/manual/reference/operator/query/maxDistance/
func (p *spherePointField) Near(point geo.SpherePoint, maxDistance float64, minDistance ...float64) filter2.Filter {
	var value interface{}
	if len(minDistance) == 0 {
		value = bson.M{"$geometry": point, "$maxDistance": maxDistance}
	} else if len(minDistance) == 1 {
		value = bson.M{"$geometry": point, "$maxDistance": maxDistance, "$minDistance": minDistance[0]}
	} else {
		panic("args error")
	}

	return filter2.New(p, "$nearSphere", value)
}

func (p *spherePointField) IntersectPoly(polygon geo.Polygon) filter2.Filter {
	return filter2.New(p, "$geoIntersects", bson.M{"$geometry": polygon})
}

func (p *spherePointField) IntersectLineString(lineString geo.LineString) filter2.Filter {
	return filter2.New(p, "$geoIntersects", bson.M{"$geometry": lineString})
}

func (p *spherePointField) IntersectBigPoly(ring geo.Ring) filter2.Filter {
	return filter2.New(p, "$geoIntersects",
		bson.M{"$geometry": bigPolyQuery(ring)})
}

func (p *spherePointField) Coordinate() FlatPointField {
	return NewFlatPointField(SubField(p.FullName(), "coordinates"))
}

type FlatPointField interface {
	field.Field
	filter2.BaseFilter[geo.FlatPoint]
	updater.BaseUpdater[geo.FlatPoint]
	index2.BaseKey
	Index2D() index2.Key
	// Index2DWith changes the default precision.
	// You can specify a bits value between 1 and 32, inclusive.
	// By default, 2d indexes use 26 bits of precision, which is equivalent to approximately two feet (60 centimeters).
	//
	//https://www.mongodb.com/docs/manual/core/indexes/index-types/geospatial/2d/create/define-location-precision/#define-location-precision-for-a-2d-index
	Index2DWith(precision int) index2.Key
	// Index2DWithRange changes the location range of a 2d index, specify the min and max options when you create the index.
	//
	// https://www.mongodb.com/docs/manual/core/indexes/index-types/geospatial/2d/create/define-location-range/#define-location-range-for-a-2d-index
	Index2DWithRange(min, max float32) index2.Key
	WithinBox(bottomLeft, upperRight geo.Coordinate) filter2.Filter
	WithinRing(ring geo.Ring) filter2.Filter
	// WithinCircle returns the flat point that are within the bounds of the circle.
	//The circle's radius, as measured in the units used by the coordinate system.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/center/#mongodb-query-op.-center
	// https://www.mongodb.com/docs/manual/reference/operator/query/center/#behavior
	WithinCircle(center geo.Coordinate, radiusNoUnit float32) filter2.Filter
	// Near returns the documents from nearest to farthest.
	// The measuring units for the maximum distance are determined by the coordinate system in use.
	//
	// https://www.mongodb.com/docs/manual/reference/operator/query/maxDistance/#mongodb-query-op.-maxDistance
	Near(point geo.Coordinate, maxDistance float64) filter2.Filter
}

type flatPointField struct {
	BaseField[geo.FlatPoint]
}

func NewFlatPointField(name string) FlatPointField {
	return &flatPointField{BaseField[geo.FlatPoint]{name: name}}
}

func (f *flatPointField) Index2D() index2.Key {
	return index2.NewKey(f, index2.KeyType2d)
}

type precisionKey struct {
	fieldName string
	precision int
}

func (vk *precisionKey) ToBsonD() bson.D {
	return bson.D{{vk.fieldName, index2.KeyType2d}}
}

func (vk *precisionKey) Options() bson.D {
	return bson.D{{"bits", vk.precision}}
}

func (f *flatPointField) Index2DWith(precision int) index2.Key {
	return &precisionKey{
		fieldName: f.FullName(),
		precision: precision,
	}
}

type rangeKey struct {
	fieldName string
	min       float32
	max       float32
}

func (vk *rangeKey) ToBsonD() bson.D {
	return bson.D{{vk.fieldName, index2.KeyType2d}}
}

func (vk *rangeKey) Options() bson.D {
	return bson.D{{"min", vk.min}, {"max", vk.max}}
}

func (f *flatPointField) Index2DWithRange(min, max float32) index2.Key {
	return &rangeKey{
		fieldName: f.FullName(),
		max:       max,
		min:       min,
	}
}

func (f *flatPointField) WithinBox(bottomLeft, upperRight geo.Coordinate) filter2.Filter {
	return filter2.New(f, "$geoWithin",
		bson.M{"$box": bson.A{bottomLeft, upperRight}})
}

func (f *flatPointField) WithinRing(ring geo.Ring) filter2.Filter {
	return filter2.New(f, "$geoWithin", bson.M{"$polygon": ring})
}

// WithinCircle returns the flat point that are within the bounds of the circle.
// The circle's radius, as measured in the units used by the coordinate system.
//
// https://www.mongodb.com/docs/manual/reference/operator/query/center/#mongodb-query-op.-center
// https://www.mongodb.com/docs/manual/reference/operator/query/center/#behavior
func (f *flatPointField) WithinCircle(center geo.Coordinate, radiusNoUnit float32) filter2.Filter {
	return filter2.New(f, "$geoWithin",
		bson.M{"$center": bson.A{center, radiusNoUnit}})
}

// Near returns the documents from nearest to farthest.
// The measuring units for the maximum distance are determined by the coordinate system in use.
//
// https://www.mongodb.com/docs/manual/reference/operator/query/maxDistance/#mongodb-query-op.-maxDistance
func (f *flatPointField) Near(point geo.Coordinate, maxDistance float64) filter2.Filter {
	return filter2.And(filter2.New(f, "$near", point),
		filter2.New(f, "$maxDistance", maxDistance))
}
