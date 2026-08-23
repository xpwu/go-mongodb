package client

// Config holds the connection parameters for a MongoDB client.
// It is expected to be populated from infrastructure configuration
// and must be treated as immutable once used to create a client.
//
// Config alone does NOT determine client caching—CacheId wraps Config
// with an optional suffix to allow multiple independent clients for
// the same Config values.
type Config struct {
	URI      string `conf:"uri"`
	User     string `conf:"user"`
	Password string `conf:"passwd"`
	MaxConn  uint64 `conf:"maxconn"`
}
