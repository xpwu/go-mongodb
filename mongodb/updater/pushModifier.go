package updater

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Option func(p *PushModifier)

func Asc() Option {
	return func(p *PushModifier) {
		if p.sort != nil {
			panic("PushModifier sort has set")
		}
		p.sort = 1
	}
}

func AscWith(f mongodb.Field) Option {
	return func(p *PushModifier) {
		if p.sort != nil {
			panic("PushModifier sort has set")
		}
		p.sort = bson.M{f.FullName(): 1}
	}
}

func Desc() Option {
	return func(p *PushModifier) {
		if p.sort != nil {
			panic("PushModifier sort has set")
		}
		p.sort = -1
	}
}

func DescWith(f mongodb.Field) Option {
	return func(p *PushModifier) {
		if p.sort != nil {
			panic("PushModifier sort has set")
		}
		p.sort = bson.M{f.FullName(): -1}
	}
}

// Position A negative number corresponds to the position in the array, counting from (but not including) the last element
//of the array. For example, -1 indicates the position just before the last element in the array.
//
// https://www.mongodb.com/docs/manual/reference/operator/update/position/#mongodb-update-up.-position
func Position(pos int) Option {
	return func(p *PushModifier) {
		p.position = &pos
	}
}

// Slice Limits the number of array elements during a Push operation.
//   n == 0 To update the array <field> to an empty array.
//   n < 0  To update the array <field> to contain only the last <num> elements.
//   n > 0  To update the array <field> contain only the first <num> elements.
//
// https://www.mongodb.com/docs/manual/reference/operator/update/slice/#mongodb-update-up.-slice
func Slice(n int) Option {
	return func(p *PushModifier) {
		p.slice = &n
	}
}

type PushModifier struct {
	position *int
	slice    *int
	sort     interface{}
}

func NewModifier(options ...Option) *PushModifier {
	ret := &PushModifier{}
	for _, o := range options {
		o(ret)
	}

	return ret
}

func (p *PushModifier) toBsonM() bson.M {
	ret := bson.M{}
	if p.position != nil {
		ret[`$position`] = *p.position
	}
	if p.slice != nil {
		ret[`$slice`] = *p.slice
	}
	if p.sort != nil {
		ret[`$sort`] = p.sort
	}

	return ret
}
