package index

import (
	"github.com/xpwu/go-mongodb/filter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type options struct {
	partialFilterExpression filter.PartialIndexFilter
	unique                  bool
	sparse                  bool
	name                    string
}

func (opts *options) ToBsonD() (r bson.D, err error) {
	ret := bson.D{}
	if opts.unique {
		ret = append(ret, bson.E{Key: "unique", Value: true})
	}
	if opts.partialFilterExpression != nil && opts.sparse {
		return nil, ErrOptions
	}
	if opts.partialFilterExpression != nil {
		ret = append(ret, bson.E{
			Key:   "partialFilterExpression",
			Value: opts.partialFilterExpression.ToBsonD(),
		})
	}
	if opts.sparse {
		ret = append(ret, bson.E{Key: "sparse", Value: true})
	}
	if opts.name != "" {
		ret = append(ret, bson.E{Key: "name", Value: opts.name})
	}

	return ret, nil
}

type Option func(opt *options)

// Partial sets a partial index filter.
// Only one of Partial or Sparse may be set; setting both will cause
// index creation to fail.
func Partial(p filter.PartialIndexFilter) Option {
	return func(opt *options) {
		opt.partialFilterExpression = p
	}
}

// Sparse enables the sparse property for the index.
// Only one of Partial or Sparse may be set; setting both will cause
// index creation to fail.
func Sparse() Option {
	return func(opt *options) {
		opt.sparse = true
	}
}

func Unique() Option {
	return func(opt *options) {
		opt.unique = true
	}
}

func Name(n string) Option {
	return func(opt *options) {
		opt.name = n
	}
}
