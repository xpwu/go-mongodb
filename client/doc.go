// Package client provides a shared MongoDB client pool.
//
// Clients are identified by CacheId, which is composed of a Config
// (deployment-level connection settings) and an optional suffix for
// distinguishing multiple clients that share the same Config.
//
// Developer-facing options (xopt.Option) are applied only once at
// creation time. Once cached, a client's behavior cannot be altered
// by subsequent option changes.
//
// Callers should not disconnect clients returned from the cache.
//
// This design prioritizes runtime stability over reconfiguration
// flexibility.
package client
