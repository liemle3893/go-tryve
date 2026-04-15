# PostgreSQL Adapter

For testing PostgreSQL database operations.

## Configuration

```yaml
environments:
  local:
    adapters:
      postgresql:
        connectionString: "postgresql://user:password@host:port/database"
        poolMin: 2            # Minimum connection pool size (default: 2)
        poolMax: 5            # Maximum connection pool size (default: 5)
```

Connection pool settings:
- `idleTimeoutMillis`: 30000 (fixed, not configurable)
- `connectionTimeoutMillis`: 10000 (fixed, not configurable)

**Peer dependency:** `npm install pg`

## Action: `execute`

Execute SQL without returning results.

```yaml
- adapter: postgresql
  action: execute
  sql: "DELETE FROM users WHERE email LIKE $1"
  params: ["test-%@example.com"]
```

## Action: `query`

Execute SQL and return all rows.

```yaml
- adapter: postgresql
  action: query
  sql: "SELECT * FROM users WHERE status = $1"
  params: ["active"]
  capture:
    first_user_id: "$.rows[0].id"
  assert:
    - path: "$.rowCount"
      greaterThan: 0
    - path: "$.rows[0].status"
      equals: "active"
```

## Action: `queryOne`

Execute SQL and return exactly one row.

```yaml
- adapter: postgresql
  action: queryOne
  sql: "SELECT * FROM users WHERE id = $1"
  params: ["{{captured.user_id}}"]
  capture:
    db_email: "email"
    db_name: "name"
  assert:
    - path: "$.email"
      equals: "{{user_email}}"
```

## Action: `count`

Run a SELECT query and return the number of rows it produces as `{"count": N}`. Use a plain SELECT (not `SELECT COUNT(*)`); the adapter counts the returned rows itself.

```yaml
- adapter: postgresql
  action: count
  sql: "SELECT * FROM users WHERE status = $1"
  params: ["active"]
  assert:
    - path: "$.count"
      greaterThan: 0
```

Note: If you need the value from a SQL `COUNT(*)` aggregate, use the `queryOne` action instead and assert on the returned column directly.

## PostgreSQL Assertions

Assertions use JSONPath (`path:`) evaluated against the action's result data.

- **`query`** returns `{ rows: [...], rowCount: N }` — use `$.rows[0].col`, `$.rowCount`, etc.
- **`queryOne`** returns the first row's columns at the top level — use `$.col_name`.

```yaml
# query assertions
assert:
  - path: "$.rowCount"
    greaterThan: 0
  - path: "$.rows[0].email"
    equals: "test@example.com"
  - path: "$.rows[0].age"
    greaterThan: 18
  - path: "$.rows[0].deleted_at"
    isNull: true

# queryOne assertions (row fields at top level)
assert:
  - path: "$.email"
    equals: "test@example.com"
  - path: "$.id"
    isNotNull: true
```

## Value Capture

Capture values from query results using JSONPath:

```yaml
# From queryOne — row fields are at top level
- adapter: postgresql
  action: queryOne
  sql: "SELECT id, email FROM users WHERE id = $1"
  capture:
    db_id: "id"           # short form, equivalent to $.id
    db_email: "email"

# From query — access via $.rows[N].col
- adapter: postgresql
  action: query
  sql: "SELECT id, email FROM users LIMIT 5"
  capture:
    first_id: "$.rows[0].id"
    first_email: "$.rows[0].email"
```

Use captured values with `{{captured.db_id}}` in subsequent steps.
