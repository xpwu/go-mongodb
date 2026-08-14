package client

import (
	"context"
	"github.com/xpwu/go-mongodb/fields"
	"github.com/xpwu/go-mongodb/filter"
	"github.com/xpwu/go-mongodb/index"
	"github.com/xpwu/go-mongodb/projection"
	"github.com/xpwu/go-mongodb/updater"
	"github.com/xpwu/go-mongodb/x"
	"github.com/xpwu/go-mongodb/xopt"
	"github.com/xpwu/go-x/exe"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"reflect"
	"sync"
	"time"
)

func GetPreserveFieldRegistry(bsonOpts *options.BSONOptions) (r *bson.Registry, err error) {
	r = GetLowerFieldRegistry()

	structCodec, err := NewPreserveStructCodec(r)
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

	arrayFilterType := x.TypeFor[fields.ArrayFilter]()
	arrayFilterEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != arrayFilterType {
			return bson.ValueEncoderError{
				Name:     "arrayFilterEncoder",
				Types:    []reflect.Type{arrayFilterType},
				Received: val,
			}
		}

		v := val.Interface().(fields.ArrayFilter).ToBsonD()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(arrayFilterType, bson.ValueEncoderFunc(arrayFilterEncoder))

	virValueType := x.TypeFor[fields.VirValue]()
	virValueEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != virValueType {
			return bson.ValueEncoderError{
				Name:     "virValueEncoder",
				Types:    []reflect.Type{virValueType},
				Received: val,
			}
		}

		v := val.Interface().(fields.VirValue).ToBsonD()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(virValueType, bson.ValueEncoderFunc(virValueEncoder))

	virPosType := x.TypeFor[fields.VirPos]()
	virPosEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != virPosType {
			return bson.ValueEncoderError{
				Name:     "virPosEncoder",
				Types:    []reflect.Type{virPosType},
				Received: val,
			}
		}

		v := val.Interface().(fields.VirPos).ToBsonD()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(virPosType, bson.ValueEncoderFunc(virPosEncoder))

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

	incType := x.TypeFor[projection.IncludeBuilder]()
	incEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != incType {
			return bson.ValueEncoderError{
				Name:     "IncludeBuilderEncoder",
				Types:    []reflect.Type{incType},
				Received: val,
			}
		}

		v := val.Interface().(*projection.IncludeBuilder).Build()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(incType, bson.ValueEncoderFunc(incEncoder))

	excType := x.TypeFor[projection.ExcludeBuilder]()
	excEncoder := func(
		ec bson.EncodeContext,
		vw bson.ValueWriter,
		val reflect.Value,
	) error {
		// All encoder implementations should check that val is valid and is of
		// the correct type before proceeding.
		if !val.IsValid() || val.Type() != excType {
			return bson.ValueEncoderError{
				Name:     "ExcludeBuilderEncoder",
				Types:    []reflect.Type{excType},
				Received: val,
			}
		}

		v := val.Interface().(*projection.ExcludeBuilder).Build()
		enc, err := ec.LookupEncoder(reflect.TypeOf(v))
		if err != nil {
			return err
		}

		return enc.EncodeValue(ec, vw, reflect.ValueOf(v))
	}
	r.RegisterTypeEncoder(excType, bson.ValueEncoderFunc(excEncoder))

	return r
}

// NewClient creates a new MongoDB client.
// It is primarily intended for internal use by GetFromCache.
// Most callers should use GetFromCache instead.
//
// Options are applied once and must not be relied upon to reconfigure
// an existing client.
func NewClient(config *Config, opts ...xopt.Option) (client *mongo.Client, err error) {
	opt := xopt.GetDefaultOpts()
	for _, o := range opts {
		o(opt)
	}
	if opt.Registry == nil {
		if opt.PreserveField {
			opt.Registry, err = GetPreserveFieldRegistry(opt.BsonOpts)
			if err != nil {
				return nil, err
			}
		} else {
			opt.Registry = GetLowerFieldRegistry()
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

	if opt.BsonOpts != nil {
		mongoOpt.SetBSONOptions(opt.BsonOpts)
	}

	// 然后是URI 携带的参数，最后是配置中明确指明的设置

	mongoOpt.ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxConn).
		SetAuth(options.Credential{
			Username:    config.User,
			Password:    config.Password,
			PasswordSet: true,
		})

	mongoOpt.SetRegistry(opt.Registry)

	client, err = mongo.Connect(mongoOpt)

	return
}

var clients = sync.Map{}

// GetFromCache returns a cached MongoDB client for the given Config.
// The Config uniquely identifies a client and is expected to originate
// from infrastructure/deployment configuration (e.g. environment variables,
// config files).
//
// Options (xopt.Option) are applied only during the initial creation of
// the client. If a client already exists for the given Config, subsequent
// calls ignore any provided options and return the existing instance.
// This ensures stable runtime behavior and avoids connection pool churn.
//
// Callers should treat Config as immutable and provide consistent options
// across calls. Inconsistent options after the first call are silently ignored.
func GetFromCache(config Config, opts ...xopt.Option) (client *mongo.Client, err error) {
	c, ok := clients.Load(config)
	if ok {
		return c.(*mongo.Client), nil
	}

	nc, err := NewClient(&config, opts...)
	if err != nil {
		return
	}

	if c, loaded := clients.LoadOrStore(config, nc); loaded {
		_ = nc.Disconnect(context.Background())
		return c.(*mongo.Client), nil
	}

	return nc, nil
}

// MustGet returns a cached MongoDB client for the given Config,
// panicking if the client cannot be created.
//
// It delegates to GetFromCache and is intended for use in initialization
// code where a failure to obtain a database client is considered fatal.
//
// Like GetFromCache, Config uniquely identifies the client, and any
// xopt.Options are applied only during the initial creation. Subsequent
// calls with the same Config ignore new options and return the existing
// client.
//
// This function should never be called with varying options for the same
// Config. Such misuse will not change the cached client and may indicate
// a configuration error in the application.
func MustGet(config Config, opts ...xopt.Option) *mongo.Client {
	r, err := GetFromCache(config, opts...)
	if err != nil {
		panic(err)
	}

	return r
}
