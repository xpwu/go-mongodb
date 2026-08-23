package client

import (
	"reflect"
	"sync"
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
	_ = MustGet(cfg.CacheId())
}

func TestMustGet_PanicEmptyURI(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustGet: expected panic for empty URI, got none")
		}
	}()

	cfg := Config{URI: ""}
	_ = MustGet(cfg.CacheId())
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

// --- CacheId tests ---

func TestCacheId_WithSuffix_DifferentSuffixesDifferentClients(t *testing.T) {
	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}

	idA := cfg.CacheId().WithSuffix("moduleA")
	idB := cfg.CacheId().WithSuffix("moduleB")

	if idA == idB {
		t.Error("CacheIds with different suffixes should not be equal")
	}
}

func TestCacheId_WithSuffix_SameSuffixSameClient(t *testing.T) {
	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}

	idA := cfg.CacheId().WithSuffix("moduleA")
	idB := cfg.CacheId().WithSuffix("moduleA")

	if idA != idB {
		t.Error("CacheIds with same suffix should be equal")
	}
}

func TestCacheId_NoSuffix_EquivalentToConfig(t *testing.T) {
	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}

	id := cfg.CacheId()
	// CacheId 的 Config 部分应该跟 cfg 一样
	if id.Config != cfg {
		t.Error("CacheId.Config should equal the original Config")
	}
	if id.suffix != "" {
		t.Error("CacheId without suffix should have empty suffix")
	}
}

func TestCacheId_WithSuffix_Immutability(t *testing.T) {
	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}

	id1 := cfg.CacheId()
	id2 := id1.WithSuffix("test")

	// id1 不应该被修改（值接收者）
	if id1.suffix != "" {
		t.Error("original CacheId should not be modified by WithSuffix")
	}
	// id2 应该有 suffix
	if id2.suffix != "test" {
		t.Errorf("new CacheId should have suffix 'test', got %q", id2.suffix)
	}
}

func TestCacheId_SyncMapKey(t *testing.T) {
	// 确保 CacheId 可以作为 sync.Map 的 key
	var m sync.Map

	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}
	idA := cfg.CacheId().WithSuffix("a")
	idB := cfg.CacheId().WithSuffix("b")

	m.Store(idA, "client-a")
	m.Store(idB, "client-b")

	v, ok := m.Load(idA)
	if !ok || v.(string) != "client-a" {
		t.Error("CacheId should work as sync.Map key")
	}

	v, ok = m.Load(idB)
	if !ok || v.(string) != "client-b" {
		t.Error("CacheId with different suffix should be a different key")
	}
}

func TestCacheId_SameConfigDifferentSuffix_CacheIsolation(t *testing.T) {
	// 验证：同一个 Config + 不同 suffix → 缓存里是两个不同的 key
	cfg := Config{URI: "mongodb://localhost:27017", MaxConn: 10}

	id1 := cfg.CacheId().WithSuffix("service1")
	id2 := cfg.CacheId().WithSuffix("service2")

	// 模拟 cache 行为
	var m sync.Map
	m.Store(id1, "cli1")

	v, ok := m.Load(id2)
	if ok {
		t.Error("different suffix should NOT hit the same cache entry")
	}
	_ = v

	// 确认 id1 能命中
	v, ok = m.Load(id1)
	if !ok || v.(string) != "cli1" {
		t.Error("same suffix should hit the cache")
	}
}
