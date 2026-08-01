package newfield

import (
	"github.com/xpwu/go-db-mongo/mongodb/field/elsejson"
	"github.com/xpwu/go-db-mongo/mongodb/field/newfield/base"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Wx struct {
	Age   *bson.Decimal128
	Time  *int
	Order base.Order
	Third elsejson.ThirdParty
}

type UserInfo struct {
	ID    string `bson:"_id"`
	Login int    `bson:"login"`
	Pass  []int
	Wx    *Wx
	Wx3   Wx
	Ws    []Wx
	Pass2 [][]int16
	InWx  Wx `bson:"inWx,inline"`
}

//var filter2 = UserInfoDoc.AgeF().
