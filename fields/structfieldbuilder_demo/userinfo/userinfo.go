package userinfo

import (
	"github.com/xpwu/go-mongodb/fields/structfieldbuilder_demo/elsetype"
	"github.com/xpwu/go-mongodb/fields/structfieldbuilder_demo/userinfo/base"
	"github.com/xpwu/go-mongodb/geo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GPS float64

type Wx struct {
	Decimal128                       *bson.Decimal128
	IntPtr                           *int
	Order                            base.Order
	ThirdParty                       elsetype.ThirdParty
	SpherePoint                      geo.SpherePoint
	FlatPoint                        geo.FlatPoint
	LikeStringName_elsetype_FullName elsetype.FullName
}

type UserInfo struct {
	StringID     string `bson:"_id"`
	IntLogin     int    `bson:"Login"`
	IntArrayPass []int
	WxPtr        *Wx
	Wx           Wx
	WsArray      []Wx
	Int16Array2  [][]int16
	//InWx                   Wx `bson:"inWx,inline"`
	LikeFloat64_GPS GPS
	BsonD           bson.D
}
