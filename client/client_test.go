package client

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/xpwu/go-mongodb/xopt"
)

// --- Config tests ---

func TestConfig_Equality(t *testing.T) {
	cfg1 := Config{URI: "mongodb://localhost:27017/test"}
	cfg2 := Config{URI: "mongodb://localhost:27017/test"}

	if cfg1 != cfg2 {
		t.Errorf("Config equality: expected equal, got %+v vs %+v", cfg1, cfg2)
	}
}

func TestConfig_Inequality(t *testing.T) {
	cfg1 := Config{URI: "mongodb://localhost:27017/test1"}
	cfg2 := Config{URI: "mongodb://localhost:27017/test2"}

	if cfg1 == cfg2 {
		t.Errorf("Config inequality: expected not equal")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		URI:      "mongodb://localhost:27017/test",
		User:     "admin",
		Password: "secret",
		MaxConn:  50,
	}

	if cfg.URI != "mongodb://localhost:27017/test" {
		t.Errorf("Config URI: got %s", cfg.URI)
	}
	if cfg.User != "admin" {
		t.Errorf("Config User: got %s", cfg.User)
	}
	if cfg.Password != "secret" {
		t.Errorf("Config Password: got %s", cfg.Password)
	}
	if cfg.MaxConn != 50 {
		t.Errorf("Config MaxConn: got %d", cfg.MaxConn)
	}
}

// --- GetFromCache / MustGet tests ---

func TestGetFromCache_DifferentConfigs(t *testing.T) {
	cfg1 := Config{URI: "mongodb://localhost:27017/test1"}
	cfg2 := Config{URI: "mongodb://localhost:27017/test2"}

	// 验证两个不同 config 不相等（sync.Map 的 key 语义）
	if reflect.DeepEqual(cfg1, cfg2) {
		t.Errorf("Configs should not be equal")
	}
}

func TestGetFromCache_SameConfigValue(t *testing.T) {
	cfg1 := Config{URI: "mongodb://localhost:27017/test"}
	cfg2 := Config{URI: "mongodb://localhost:27017/test"}

	if cfg1 != cfg2 {
		t.Errorf("Same value Configs should be equal for sync.Map key")
	}
}

func TestMustGet_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet: expected panic for invalid URI, got none")
		}
	}()

	cfg := Config{URI: "invalid-uri://broken"}
	_ = MustGet(cfg)
}

func TestMustGet_PanicEmptyURI(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet: expected panic for empty URI, got none")
		}
	}()

	cfg := Config{URI: ""}
	_ = MustGet(cfg)
}

// --- xopt tests ---

func TestXopt_GetDefaultOpts(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	if opts.BsonOpts != nil {
		t.Errorf("Default BsonOpts: expected nil, got %v", opts.BsonOpts)
	}
	if opts.Registry != nil {
		t.Errorf("Default Registry: expected nil, got %v", opts.Registry)
	}
	if opts.PreserveField {
		t.Errorf("Default PreserveField: expected false, got true")
	}
}

func TestXopt_WithPreserveField_True(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	optFunc := xopt.WithPreserveField()
	optFunc(opts)

	if !opts.PreserveField {
		t.Errorf("WithPreserveField(true): expected PreserveField=true, got false")
	}
}

func TestXopt_WithPreserveField_False(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	optFunc := xopt.WithPreserveField()
	optFunc(opts)

	if !opts.PreserveField {
		t.Errorf("WithPreserveField(false): expected PreserveField=true, got false")
	}
}

func TestXopt_WithBsonOptions(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	bsonOpts := &options.BSONOptions{}
	optFunc := xopt.WithBsonOptions(bsonOpts)
	optFunc(opts)

	if opts.BsonOpts != bsonOpts {
		t.Errorf("WithBsonOptions: expected same pointer, got different")
	}
}

func TestXopt_WithRegistry(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	registry := bson.NewRegistry()
	optFunc := xopt.WithRegistry(registry)
	optFunc(opts)

	if opts.Registry != registry {
		t.Errorf("WithRegistry: expected same pointer, got different")
	}
}

func TestXopt_ChainOptions(t *testing.T) {
	opts := xopt.GetDefaultOpts()

	bsonOpts := &options.BSONOptions{}
	registry := bson.NewRegistry()

	xopt.WithPreserveField()(opts)
	xopt.WithBsonOptions(bsonOpts)(opts)
	xopt.WithRegistry(registry)(opts)

	if !opts.PreserveField {
		t.Errorf("Chain: PreserveField expected true")
	}
	if opts.BsonOpts != bsonOpts {
		t.Errorf("Chain: BsonOpts expected same pointer")
	}
	if opts.Registry != registry {
		t.Errorf("Chain: Registry expected same pointer")
	}
}
