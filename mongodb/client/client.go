package client

import (
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/index"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
	"github.com/xpwu/go-db-mongo/mongodb/x"
	"github.com/xpwu/go-x/exe"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"reflect"
	"sync"
	"time"
)

func GetPascalFieldRegistry(bsonOpts *options.BSONOptions) (r *bson.Registry, err error) {
	r = GetLowerFieldRegistry()

	structCodec, err := NewPascalStructCodec(r)
	if err != nil {
		return nil, err
	}

	if bsonOpts != nil {
		structCodec.ZeroStructs = bsonOpts.ZeroStructs
		structCodec.OmitZeroStruct = bsonOpts.OmitZeroStruct
		structCodec.OmitEmpty = bsonOpts.OmitEmpty
		structCodec.ErrorOnInlineDuplicates = bsonOpts.ErrorOnInlineDuplicates
		structCodec.UseJSONStructTags = bsonOpts.UseJSONStructTags
		structCodec.ZeroMaps = bsonOpts.ZeroMaps
		structCodec.UseLocalTimeZone = bsonOpts.UseLocalTimeZone
	}

	r.RegisterKindEncoder(reflect.Struct, structCodec)
	r.RegisterKindDecoder(reflect.Struct, structCodec)

	return r, nil
}

func GetLowerFieldRegistry() *bson.Registry {
	r := bson.NewRegistry()

	updaterType := x.TypeFor[updater.Updater]()
	updaterEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != updaterType {
			return bson.ValueEncoderError{
				Name:     "updaterEncoder",
				Types:    []reflect.Type{updaterType},
				Received: val,
			}
		}

		v := val.Interface().(updater.Updater).ToBsonM()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(updaterType, bson.ValueEncoderFunc(updaterEncoder))

	filterType := x.TypeFor[filter.Filter]()
	filterEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != filterType {
			return bson.ValueEncoderError{
				Name:     "filterEncoder",
				Types:    []reflect.Type{filterType},
				Received: val,
			}
		}

		v := val.Interface().(filter.Filter).ToBsonD()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(filterType, bson.ValueEncoderFunc(filterEncoder))

	keyType := x.TypeFor[index.Key]()
	keyEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != keyType {
			return bson.ValueEncoderError{
				Name:     "keyEncoder",
				Types:    []reflect.Type{keyType},
				Received: val,
			}
		}

		v := val.Interface().(index.Key).ToBsonD()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(keyType, bson.ValueEncoderFunc(keyEncoder))

	return r
}

type option struct {
	bsonOpts    *options.BSONOptions
	registry    *bson.Registry
	pascalField bool
}

type Option func(opt *option)

func WithBsonOptions(bsonOpts *options.BSONOptions) Option {
	return func(opt *option) {
		opt.bsonOpts = bsonOpts
	}
}

func WithRegistry(registry *bson.Registry) Option {
	return func(opt *option) {
		opt.registry = registry
	}
}

func WithPascalField() Option {
	return func(opt *option) {
		opt.pascalField = true
	}
}

func getDefaultOpt() *option {
	return &option{
		bsonOpts:    nil,
		registry:    nil,
		pascalField: false,
	}
}

func NewClient(config *Config, opts ...Option) (client *mongo.Client, err error) {
	opt := getDefaultOpt()
	for _, o := range opts {
		o(opt)
	}
	if opt.registry == nil {
		if opt.pascalField {
			opt.registry, err = GetPascalFieldRegistry(opt.bsonOpts)
			if err != nil {
				return nil, err
			}
		} else {
			opt.registry = GetLowerFieldRegistry()
		}
	}

	mongoOpt := options.Client()
	mongoOpt.SetAppName(exe.Name).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second).
		SetTimeout(5 * time.Second).
		SetRetryWrites(true).
		SetMaxConnIdleTime(3 * time.Minute).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetMinPoolSize(1)

	if opt.bsonOpts != nil {
		mongoOpt.SetBSONOptions(opt.bsonOpts)
	}

	// 然后是URI 携带的参数，最后是配置中明确指明的设置

	mongoOpt.ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxConn).
		SetAuth(options.Credential{
			Username:    config.User,
			Password:    config.Password,
			PasswordSet: true,
		})

	mongoOpt.SetRegistry(opt.registry)

	client, err = mongo.Connect(mongoOpt)

	return
}

var clients = sync.Map{}

// GetFromCache gets or creates a client using the config from the cache.
// The config is the ID of the client.
func GetFromCache(config *Config, opts ...Option) (client *mongo.Client, err error) {
	c, ok := clients.Load(*config)
	if ok {
		return c.(*mongo.Client), nil
	}

	nc, err := NewClient(config, opts...)
	if err != nil {
		return
	}

	c, ok = clients.LoadOrStore(*config, nc)
	return c.(*mongo.Client), nil
}

func MustGet(config *Config, opts ...Option) *mongo.Client {
	r, err := GetFromCache(config, opts...)
	if err != nil {
		panic(err)
	}

	return r
}
