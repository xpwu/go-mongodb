package xopt

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Opts struct {
	BsonOpts      *options.BSONOptions
	Registry      *bson.Registry
	PreserveField bool
}

type Option func(opt *Opts)

func WithBsonOptions(bsonOpts *options.BSONOptions) Option {
	return func(opt *Opts) {
		opt.BsonOpts = bsonOpts
	}
}

func WithRegistry(registry *bson.Registry) Option {
	return func(opt *Opts) {
		opt.Registry = registry
	}
}

func WithPreserveField() Option {
	return func(opt *Opts) {
		opt.PreserveField = true
	}
}

func GetDefaultOpts() *Opts {
	return &Opts{
		BsonOpts:      nil,
		Registry:      nil,
		PreserveField: false,
	}
}
