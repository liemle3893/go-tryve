package adapter

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// RedisAdapter executes Redis commands against a target Redis server.
// It maintains a persistent *goredis.Client across operations.
type RedisAdapter struct {
	connStr   string
	db        int
	keyPrefix string
	client    *goredis.Client
}

// NewRedisAdapter constructs a RedisAdapter from the provided configuration map.
// Recognised keys:
//   - "connectionString" (string, required): Redis URL (e.g. "redis://:password@localhost:6379/0").
//   - "db" (int, optional, default 0): database index to select.
//   - "keyPrefix" (string, optional): prefix prepended to every key operation.
//
// Connect must be called before Execute or Health.
func NewRedisAdapter(cfg map[string]any) *RedisAdapter {
	connStr, _ := cfg["connectionString"].(string)

	db := 0
	if raw, ok := cfg["db"]; ok {
		switch v := raw.(type) {
		case int:
			db = v
		case float64:
			db = int(v)
		}
	}

	keyPrefix, _ := cfg["keyPrefix"].(string)

	return &RedisAdapter{
		connStr:   connStr,
		db:        db,
		keyPrefix: keyPrefix,
	}
}

// Name returns the adapter's registered identifier.
func (a *RedisAdapter) Name() string { return "redis" }

// Connect parses the connection string, overrides the DB index when explicitly
// configured, and establishes the underlying Redis client.
func (a *RedisAdapter) Connect(_ context.Context) error {
	if a.connStr == "" {
		return tryve.ConnectionError("redis", "connect: connectionString is required", nil)
	}
	if err := CheckUnresolvedEnvVars("redis", "connectionString", a.connStr); err != nil {
		return err
	}

	opts, err := goredis.ParseURL(a.connStr)
	if err != nil {
		return tryve.ConnectionError("redis", fmt.Sprintf("connect: invalid connectionString: %v", err), err)
	}

	// Override the DB from explicit config when non-zero, or when the URL does
	// not encode a database (opts.DB == 0 and cfg.db > 0).
	if a.db != 0 {
		opts.DB = a.db
	}

	a.client = goredis.NewClient(opts)
	return nil
}

// Close releases the Redis client and its underlying connection pool.
func (a *RedisAdapter) Close(_ context.Context) error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// Health performs a lightweight PING to verify Redis connectivity.
func (a *RedisAdapter) Health(ctx context.Context) error {
	if a.client == nil {
		return tryve.ConnectionError("redis", "health: not connected", nil)
	}
	if err := a.client.Ping(ctx).Err(); err != nil {
		return tryve.ConnectionError("redis", fmt.Sprintf("health: ping failed: %v", err), err)
	}
	return nil
}

// Execute dispatches the named action to the corresponding Redis command.
// All key parameters have the configured keyPrefix prepended before use.
//
// Supported actions: get, set, del, exists, incr, hget, hset, hgetall, keys, flushPattern.
func (a *RedisAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	switch action {
	case "get":
		return a.actionGet(ctx, params)
	case "set":
		return a.actionSet(ctx, params)
	case "del":
		return a.actionDel(ctx, params)
	case "exists":
		return a.actionExists(ctx, params)
	case "incr":
		return a.actionIncr(ctx, params)
	case "hget":
		return a.actionHGet(ctx, params)
	case "hset":
		return a.actionHSet(ctx, params)
	case "hgetall":
		return a.actionHGetAll(ctx, params)
	case "keys":
		return a.actionKeys(ctx, params)
	case "flushPattern":
		return a.actionFlushPattern(ctx, params)
	default:
		return nil, tryve.AdapterError("redis", action,
			fmt.Sprintf("unsupported action %q", action), nil)
	}
}

// prefixedKey prepends the adapter's keyPrefix to key.
// When keyPrefix is empty the key is returned unchanged.
func (a *RedisAdapter) prefixedKey(key string) string {
	if a.keyPrefix == "" {
		return key
	}
	return a.keyPrefix + key
}

// actionGet executes a Redis GET and returns {"value": string}.
func (a *RedisAdapter) actionGet(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "get", err.Error(), err)
	}

	var val string
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		val, cmdErr = a.client.Get(ctx, a.prefixedKey(key)).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "get", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"value": val}, duration, nil), nil
}

// actionSet executes a Redis SET and returns {"ok": true}.
// The optional "ttl" param specifies expiry in seconds (0 = no expiry).
func (a *RedisAdapter) actionSet(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "set", err.Error(), err)
	}

	value, ok := params["value"]
	if !ok {
		return nil, tryve.AdapterError("redis", "set", "required parameter \"value\" is missing", nil)
	}

	ttl := parseTTL(params)

	duration, execErr := MeasureDuration(func() error {
		return a.client.Set(ctx, a.prefixedKey(key), value, ttl).Err()
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "set", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"ok": true}, duration, nil), nil
}

// actionDel executes a Redis DEL and returns {"deleted": float64(n)}.
func (a *RedisAdapter) actionDel(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "del", err.Error(), err)
	}

	var n int64
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		n, cmdErr = a.client.Del(ctx, a.prefixedKey(key)).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "del", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"deleted": float64(n)}, duration, nil), nil
}

// actionExists executes a Redis EXISTS and returns {"exists": bool}.
func (a *RedisAdapter) actionExists(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "exists", err.Error(), err)
	}

	var n int64
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		n, cmdErr = a.client.Exists(ctx, a.prefixedKey(key)).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "exists", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"exists": n > 0}, duration, nil), nil
}

// actionIncr executes a Redis INCR and returns {"value": float64(n)}.
func (a *RedisAdapter) actionIncr(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "incr", err.Error(), err)
	}

	var n int64
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		n, cmdErr = a.client.Incr(ctx, a.prefixedKey(key)).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "incr", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"value": float64(n)}, duration, nil), nil
}

// actionHGet executes a Redis HGET and returns {"value": string}.
func (a *RedisAdapter) actionHGet(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "hget", err.Error(), err)
	}
	field, err := getStr(params, "field")
	if err != nil {
		return nil, tryve.AdapterError("redis", "hget", err.Error(), err)
	}

	var val string
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		val, cmdErr = a.client.HGet(ctx, a.prefixedKey(key), field).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "hget", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"value": val}, duration, nil), nil
}

// actionHSet executes a Redis HSET and returns {"ok": true}.
func (a *RedisAdapter) actionHSet(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "hset", err.Error(), err)
	}
	field, err := getStr(params, "field")
	if err != nil {
		return nil, tryve.AdapterError("redis", "hset", err.Error(), err)
	}
	value, ok := params["value"]
	if !ok {
		return nil, tryve.AdapterError("redis", "hset", "required parameter \"value\" is missing", nil)
	}

	duration, execErr := MeasureDuration(func() error {
		return a.client.HSet(ctx, a.prefixedKey(key), field, value).Err()
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "hset", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"ok": true}, duration, nil), nil
}

// actionHGetAll executes a Redis HGETALL and returns {"value": map[string]any}.
func (a *RedisAdapter) actionHGetAll(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	key, err := getStr(params, "key")
	if err != nil {
		return nil, tryve.AdapterError("redis", "hgetall", err.Error(), err)
	}

	var m map[string]string
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		m, cmdErr = a.client.HGetAll(ctx, a.prefixedKey(key)).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "hgetall", execErr.Error(), execErr)
	}

	// Convert map[string]string → map[string]any for uniform result typing.
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}

	return SuccessResult(map[string]any{"value": out}, duration, nil), nil
}

// actionKeys executes a Redis KEYS and returns {"keys": []string}.
// The pattern param is NOT prefixed; callers must include the prefix if required.
func (a *RedisAdapter) actionKeys(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	pattern, err := getStr(params, "pattern")
	if err != nil {
		return nil, tryve.AdapterError("redis", "keys", err.Error(), err)
	}

	var keys []string
	duration, execErr := MeasureDuration(func() error {
		var cmdErr error
		keys, cmdErr = a.client.Keys(ctx, pattern).Result()
		return cmdErr
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "keys", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"keys": keys}, duration, nil), nil
}

// actionFlushPattern scans for all keys matching pattern, deletes them, and
// returns {"deleted": float64(n)}.
// Like actionKeys, the pattern is used verbatim and is NOT prefixed.
func (a *RedisAdapter) actionFlushPattern(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	pattern, err := getStr(params, "pattern")
	if err != nil {
		return nil, tryve.AdapterError("redis", "flushPattern", err.Error(), err)
	}

	var totalDeleted int64
	duration, execErr := MeasureDuration(func() error {
		var cursor uint64
		for {
			var batch []string
			var scanErr error
			batch, cursor, scanErr = a.client.Scan(ctx, cursor, pattern, 100).Result()
			if scanErr != nil {
				return scanErr
			}
			if len(batch) > 0 {
				n, delErr := a.client.Del(ctx, batch...).Result()
				if delErr != nil {
					return delErr
				}
				totalDeleted += n
			}
			if cursor == 0 {
				break
			}
		}
		return nil
	})
	if execErr != nil {
		return nil, tryve.AdapterError("redis", "flushPattern", execErr.Error(), execErr)
	}

	return SuccessResult(map[string]any{"deleted": float64(totalDeleted)}, duration, nil), nil
}

// ExportedPrefixedKey exposes the prefixedKey helper for white-box unit tests.
// Do not use in production code outside the adapter package.
func (a *RedisAdapter) ExportedPrefixedKey(key string) string {
	return a.prefixedKey(key)
}

// parseTTL extracts an optional "ttl" parameter (seconds) from params.
// Returns 0 (no expiry) when the key is absent or zero.
func parseTTL(params map[string]any) time.Duration {
	raw, ok := params["ttl"]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return time.Duration(v) * time.Second
	case float64:
		return time.Duration(int(v)) * time.Second
	}
	return 0
}
