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

func (opts *options) Build() bson.D {
	ret := bson.D{}
	if opts.unique {
		ret = append(ret, bson.E{Key: "unique", Value: true})
	}
	if opts.partialFilterExpression != nil {
		ret = append(ret, bson.E{
			Key:   "partialFilterExpression",
			Value: opts.partialFilterExpression.ToBsonD(),
		})
	}
	if opts.partialFilterExpression == nil && opts.sparse {
		ret = append(ret, bson.E{Key: "sparse", Value: true})
	}
	if opts.name != "" {
		ret = append(ret, bson.E{Key: "name", Value: opts.name})
	}

	return ret
}

type Option func(opt *options)

// Partial sets a partial index filter.
// Only one of Partial or Sparse may be set; setting both will cause
// index creation to fail.
//
// If both Partial and Sparse are configured, Partial takes precedence
// and the sparse option is ignored.
//
// （如果同时设置了 Partial 和 Sparse，Partial 会覆盖 Sparse 的设置，sparse 选项将被忽略。）
func Partial(p filter.PartialIndexFilter) Option {
	return func(opt *options) {
		opt.partialFilterExpression = p
	}
}

// Sparse enables the sparse property for the index.
// Only one of Partial or Sparse may be set; setting both will cause
// index creation to fail.
//
// If both Partial and Sparse are configured, Partial takes precedence
// and this sparse option is ignored.
//
// （如果同时设置了 Partial 和 Sparse，Partial 会覆盖 Sparse 的设置，此 sparse 选项将被忽略。）
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
