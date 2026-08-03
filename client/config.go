package client

// Config identifies a MongoDB client instance.
// It is expected to be populated from infrastructure configuration
// and must be treated as immutable once used to create a client.
//
// Two Config values that compare equal MUST refer to the same
// logical MongoDB deployment.
type Config struct {
	URI      string `conf:"uri"`
	User     string `conf:"user"`
	Password string `conf:"passwd"`
	MaxConn  uint64 `conf:"maxconn"`
}
