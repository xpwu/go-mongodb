package userinfo

import (
	"github.com/xpwu/go-mongodb/geo"
	"github.com/xpwu/go-mongodb/zdemo/elsetype"
	"github.com/xpwu/go-mongodb/zdemo/userinfo/base"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GPS float64

type GPSes []GPS

type GPSesA = GPSes

type BSOND = bson.D

type BSOND2 BSOND

type Wx struct {
	Decimal128                       *bson.Decimal128
	IntPtr                           *int
	Order                            base.Order
	ThirdParty                       elsetype.ThirdParty
	SpherePoint                      geo.SpherePoint
	FlatPoint                        geo.FlatPoint
	LikeStringName_elsetype_FullName elsetype.FullName
}

type WxAlias = Wx
type WxDef Wx
type WXDef2 WxAlias
type AE = bson.E
type DefAE AE
type AliasDefAE = DefAE

//go:generate go run github.com/xpwu/go-mongodb/cmd/gomongodbgen

type UserInfo struct {
	StringID     string `bson:"_id"`
	IntLogin     int    `bson:"Login"`
	IntArrayPass []int
	WxPtr        *Wx
	Wx           Wx
	WsArray      []Wx
	Int16Array2  [][]int16
	//InWx                   Wx `bson:"inWx,inline"`
	LikeFloat64_GPS        GPS
	LikeFloat64Array_GPSes GPSes
	AliasGPSes             GPSesA
	BsonD                  bson.D
	AliasDefAE             AliasDefAE
	AalisBsonD             BSOND
	AalisBsonDDef          BSOND2
	AliasWx                WxAlias
	LikeWx_WxDef           WxDef
	InWxDef                WxDef `bson:",inline"`
	DefAliasWx_WXDef2      WXDef2
	Byte                   byte
	Bytes                  []byte
	Rune                   rune
	Runes                  []rune
}

var collName = UserInfoColl.DefaultName
var fil = UserInfoColl.AliasWxF().IntPtrF().Eq(5)

//go:generate go run github.com/xpwu/go-mongodb/cmd/gomongodbgen

type OrgInfo struct {
	Admin UserInfo
}
