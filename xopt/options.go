package xopt

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Opts struct {
	BsonOpts      *options.BSONOptions
	Registry      *bson.Registry
	PreserveField bool
	IgnoreTagErr  bool
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

// IgnoreTagErr 忽略 minsize & truncate & omitempty tag的报错
func IgnoreTagErr() Option {
	return func(option *Opts) {
		option.IgnoreTagErr = true
	}
}

func GetDefaultOpts() *Opts {
	return &Opts{
		BsonOpts:      nil,
		Registry:      nil,
		PreserveField: false,
		IgnoreTagErr:  false,
	}
}
