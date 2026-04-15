package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

const (
	postgresqlAdapterName  = "postgresql"
	defaultPostgresPoolSize = 5
)

// PostgreSQLAdapter executes SQL statements against a PostgreSQL database using
// a pgxpool connection pool. It supports the "execute", "query", "queryOne",
// and "count" actions.
type PostgreSQLAdapter struct {
	connectionString string
	schema           string
	poolSize         int
	pool             *pgxpool.Pool
}

// NewPostgreSQLAdapter constructs a PostgreSQLAdapter from a configuration map.
//
// Recognised keys:
//   - "connectionString" (string, required) — libpq-compatible DSN or URL.
//   - "schema"           (string, optional) — default search_path schema.
//   - "poolSize"         (int or float64, optional, default 5) — maximum pool connections.
func NewPostgreSQLAdapter(cfg map[string]any) *PostgreSQLAdapter {
	connStr, _ := cfg["connectionString"].(string)

	schema, _ := cfg["schema"].(string)

	poolSize := defaultPostgresPoolSize
	switch v := cfg["poolSize"].(type) {
	case int:
		if v > 0 {
			poolSize = v
		}
	case float64:
		if int(v) > 0 {
			poolSize = int(v)
		}
	}

	return &PostgreSQLAdapter{
		connectionString: connStr,
		schema:           schema,
		poolSize:         poolSize,
	}
}

// Name returns the adapter's registered identifier.
func (a *PostgreSQLAdapter) Name() string { return postgresqlAdapterName }

// Connect creates the pgxpool connection pool. It applies the configured
// poolSize as the maximum number of connections. If schema is non-empty,
// it is set as the default search_path for all connections in the pool.
func (a *PostgreSQLAdapter) Connect(ctx context.Context) error {
	if a.connectionString == "" {
		return tryve.ConnectionError(postgresqlAdapterName, "connectionString must not be empty", nil)
	}
	if err := CheckUnresolvedEnvVars(postgresqlAdapterName, "connectionString", a.connectionString); err != nil {
		return err
	}

	cfg, err := pgxpool.ParseConfig(a.connectionString)
	if err != nil {
		return tryve.ConnectionError(postgresqlAdapterName, "failed to parse connection string", err)
	}

	cfg.MaxConns = int32(a.poolSize) //nolint:gosec

	if a.schema != "" {
		// Prepend the configured schema to the search_path for every acquired connection.
		origAfterConnect := cfg.AfterConnect
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			if origAfterConnect != nil {
				if err := origAfterConnect(ctx, conn); err != nil {
					return err
				}
			}
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", a.schema))
			return err
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return tryve.ConnectionError(postgresqlAdapterName, "failed to create connection pool", err)
	}

	a.pool = pool
	return nil
}

// Close shuts down the connection pool, releasing all held connections.
func (a *PostgreSQLAdapter) Close(_ context.Context) error {
	if a.pool != nil {
		a.pool.Close()
		a.pool = nil
	}
	return nil
}

// Health performs a lightweight ping against the database to verify connectivity.
func (a *PostgreSQLAdapter) Health(ctx context.Context) error {
	if a.pool == nil {
		return tryve.ConnectionError(postgresqlAdapterName, "pool is not initialised; call Connect first", nil)
	}
	if err := a.pool.Ping(ctx); err != nil {
		return tryve.ConnectionError(postgresqlAdapterName, "ping failed", err)
	}
	return nil
}

// Execute dispatches the named action against the database.
//
// Supported actions:
//   - "execute"  — run a non-SELECT statement; returns {"rowsAffected": float64}.
//   - "query"    — run a SELECT; returns {"rows": []map[string]any, "rowCount": float64}.
//   - "queryOne" — run a SELECT, return the first row; returns {"row": map[string]any}.
//   - "count"    — run a SELECT, return only the row count; returns {"count": float64}.
func (a *PostgreSQLAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	switch action {
	case "execute":
		return a.executeAction(ctx, params)
	case "query":
		return a.queryAction(ctx, params)
	case "queryOne":
		return a.queryOneAction(ctx, params)
	case "count":
		return a.countAction(ctx, params)
	default:
		return nil, tryve.AdapterError(postgresqlAdapterName, action,
			fmt.Sprintf("unsupported action %q; valid actions are: execute, query, queryOne, count", action), nil)
	}
}

// executeAction runs a non-SELECT SQL statement and returns the number of rows affected.
func (a *PostgreSQLAdapter) executeAction(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	sql, queryParams, err := extractSQLParams(params)
	if err != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "execute", err.Error(), err)
	}

	var tag interface{ RowsAffected() int64 }
	duration, execErr := MeasureDuration(func() error {
		ct, e := a.pool.Exec(ctx, sql, queryParams...)
		if e == nil {
			tag = ct
		}
		return e
	})
	if execErr != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "execute", "statement execution failed", execErr)
	}

	data := map[string]any{
		"rowsAffected": float64(tag.RowsAffected()),
	}
	return SuccessResult(data, duration, nil), nil
}

// queryAction runs a SELECT statement and returns all rows as a slice of maps.
func (a *PostgreSQLAdapter) queryAction(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	sql, queryParams, err := extractSQLParams(params)
	if err != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "query", err.Error(), err)
	}

	var rows []map[string]any
	duration, execErr := MeasureDuration(func() error {
		var e error
		rows, e = a.fetchRows(ctx, sql, queryParams)
		return e
	})
	if execErr != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "query", "query execution failed", execErr)
	}

	data := map[string]any{
		"rows":     rows,
		"rowCount": float64(len(rows)),
	}
	return SuccessResult(data, duration, nil), nil
}

// queryOneAction runs a SELECT and returns the first row. Returns an error when
// the result set is empty.
func (a *PostgreSQLAdapter) queryOneAction(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	sql, queryParams, err := extractSQLParams(params)
	if err != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "queryOne", err.Error(), err)
	}

	var rows []map[string]any
	duration, execErr := MeasureDuration(func() error {
		var e error
		rows, e = a.fetchRows(ctx, sql, queryParams)
		return e
	})
	if execErr != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "queryOne", "query execution failed", execErr)
	}
	if len(rows) == 0 {
		return nil, tryve.AdapterError(postgresqlAdapterName, "queryOne", "query returned no rows", nil)
	}

	// Return the row columns at top level (matching TS behavior).
	// This allows capture paths like "id" to work as $.id.
	data := rows[0]
	return SuccessResult(data, duration, nil), nil
}

// countAction runs a SELECT and returns only the number of rows produced.
func (a *PostgreSQLAdapter) countAction(ctx context.Context, params map[string]any) (*tryve.StepResult, error) {
	sql, queryParams, err := extractSQLParams(params)
	if err != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "count", err.Error(), err)
	}

	var rows []map[string]any
	duration, execErr := MeasureDuration(func() error {
		var e error
		rows, e = a.fetchRows(ctx, sql, queryParams)
		return e
	})
	if execErr != nil {
		return nil, tryve.AdapterError(postgresqlAdapterName, "count", "query execution failed", execErr)
	}

	data := map[string]any{
		"count": float64(len(rows)),
	}
	return SuccessResult(data, duration, nil), nil
}

// fetchRows executes sql with args and collects all result rows into a slice of
// maps keyed by column name.
func (a *PostgreSQLAdapter) fetchRows(ctx context.Context, sql string, args []any) ([]map[string]any, error) {
	pgRows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	fieldDescs := pgRows.FieldDescriptions()
	colNames := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		colNames[i] = string(fd.Name)
	}

	var result []map[string]any
	for pgRows.Next() {
		values, err := pgRows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(colNames))
		for i, name := range colNames {
			row[name] = normalizeValue(values[i])
		}
		result = append(result, row)
	}
	if err := pgRows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// extractSQLParams reads the "sql" (required) and "params" (optional) keys from
// a params map. Returns an error when "sql" is absent or not a string.
func extractSQLParams(params map[string]any) (string, []any, error) {
	sqlVal, ok := params["sql"]
	if !ok {
		return "", nil, fmt.Errorf("required parameter \"sql\" is missing")
	}
	sql, ok := sqlVal.(string)
	if !ok {
		return "", nil, fmt.Errorf("parameter \"sql\" must be a string, got %T", sqlVal)
	}
	if sql == "" {
		return "", nil, fmt.Errorf("parameter \"sql\" must not be empty")
	}

	var queryParams []any
	if p, ok := params["params"]; ok && p != nil {
		if slice, ok := p.([]any); ok {
			queryParams = slice
		}
	}

	return sql, queryParams, nil
}

// normalizeValue converts pgx-specific Go types into JSON-friendly representations
// so that captured values and assertions work as expected.
func normalizeValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case [16]byte:
		// UUID: format as standard string "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			val[0:4], val[4:6], val[6:8], val[8:10], val[10:16])
	case []byte:
		// Try JSON first, then string
		var js any
		if err := json.Unmarshal(val, &js); err == nil {
			return js
		}
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	case net.IP:
		return val.String()
	case fmt.Stringer:
		return val.String()
	default:
		return v
	}
}
