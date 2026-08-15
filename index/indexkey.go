package index

import (
	"github.com/xpwu/go-mongodb/field"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Key interface {
	ToBsonD() bson.D
	Options() bson.D
}

type base struct {
	f     field.Field
	value interface{}
	opts  options
}

func (b *base) ToBsonD() bson.D {
	return bson.D{{b.f.FullName(), b.value}}
}

func (b *base) Options() bson.D {
	return b.opts.Build()
}

type KeyType = interface{}

const (
	KeyTypeDescendingOrder = -1
	KeyTypeAscendingOrder  = 1
	KeyTypeText            = "text"
	KeyType2d              = "2d"
	KeyType2dSphere        = "2dsphere"
)

func NewKey(f field.Field, keyType KeyType, opts ...Option) Key {
	ret := &base{
		f:     f,
		value: keyType,
	}
	for _, o := range opts {
		o(&ret.opts)
	}

	return ret
}

type compKey struct {
	keys []Key
	opts options
}

func (c *compKey) ToBsonD() bson.D {
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

func (c *compKey) Options() bson.D {
	return c.opts.Build()
}

func CompKeys(keys []Key, opts ...Option) Key {
	ret := &compKey{keys: keys}

	for _, o := range opts {
		o(&ret.opts)
	}

	return ret
}
