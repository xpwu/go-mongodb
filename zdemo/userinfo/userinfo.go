package userinfo

import (
	"github.com/xpwu/go-mongodb/geo"
	"github.com/xpwu/go-mongodb/zdemo/elsetype"
	"github.com/xpwu/go-mongodb/zdemo/userinfo/base"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Wx struct {
	Age    *bson.Decimal128
	Time   *int
	Order  base.Order
	Third  elsetype.ThirdParty
	Addr   geo.SpherePoint
	Player geo.FlatPoint
}

type UserInfo struct {
	ID    string `bson:"_id"`
	Login int    `bson:"Login"`
	Pass  []int
	Wx    *Wx
	Wx3   Wx
	Ws    []Wx
	Pass2 [][]int16
	InWx  Wx `bson:"inWx,inline"`
}

//var filter2 = UserInfoDoc.AgeF().
