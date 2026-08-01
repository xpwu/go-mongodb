package updater

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Updater interface {
	ToBsonM() bson.M
}

type base struct {
	f     mongodb.Field
	op    string
	value interface{}
}

func (b *base) ToBsonM() bson.M {
	// {op: {f:value, ...}, ...}
	return bson.M{b.op: bson.M{b.f.FullName(): b.value}}
}

func New(f mongodb.Field, op string, value interface{}) Updater {
	return &base{
		f:     f,
		op:    op,
		value: value,
	}
}

type batch struct {
	updaters []Updater
}

func (b *batch) ToBsonM() bson.M {
	if b == nil || len(b.updaters) == 0 {
		return bson.M{}
	}

	// {op: {f:value, ...}, ...}

	ret := b.updaters[0].ToBsonM()
	if len(b.updaters) == 1 {
		return ret
	}

	for i := 1; i < len(b.updaters); i++ {
		m := b.updaters[i].ToBsonM()
		for k, v := range m {
			old, ok := ret[k]
			if !ok {
				old = v
			} else {
				o := old.(bson.M)
				vM := v.(bson.M)
				// merge vM and old
				for vMk, vMv := range vM {
					o[vMk] = vMv
				}
				old = o
			}
			ret[k] = old
		}
	}

	return ret
}

func Batch(u1 Updater, updaters ...Updater) Updater {
	return &batch{updaters: append([]Updater{u1}, updaters...)}
}

func PullByFilter(f mongodb.Field, filter filter.Filter) Updater {
	return &base{
		f:     f,
		op:    `$pull`,
		value: filter.ToBsonD(),
	}
}

func PushByModifier(f mongodb.Field, modifier PushModifier, each interface{}) Updater {
	val := modifier.toBsonM()
	val[`$each`] = each

	return &base{
		f:     f,
		op:    `$push`,
		value: val,
	}
}
