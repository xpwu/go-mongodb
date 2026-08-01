package index

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Key interface {
	ToBsonD() bson.D
}

type base struct {
	f     mongodb.Field
	value interface{}
}

func (b *base) ToBsonD() bson.D {
	return bson.D{{b.f.FullName(), b.value}}
}

type KeyType = interface{}

const (
	KeyTypeDescendingOrder = -1
	KeyTypeAscendingOrder  = 1
	KeyTypeText            = "text"
	KeyType2d              = "2d"
	KeyType2dsphere        = "2dsphere"
)

// keyType 常用： 1 升序；-1 降序；"2dsphere"; "2d"; "text"。具体可以查阅mongodb文档
func NewKey(f mongodb.Field, keyType KeyType) Key {
	return &base{
		f:     f,
		value: keyType,
	}
}

type compound struct {
	keys []Key
}

func (c *compound) ToBsonD() bson.D {
	if c == nil || len(c.keys) == 0 {
		return bson.D{}
	}

	ret := make(bson.D, 0, len(c.keys))
	for _, k := range c.keys {
		d := k.ToBsonD()
		ret = append(ret, d...)
	}

	return ret
}

func Keys(k1 Key, keys ...Key) Key {
	r := make([]Key, 0, 1+len(keys))
	r = append(r, k1)
	return &compound{keys: append(r, keys...)}
}
