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
    first_user_id: "[0].id"
  assert:
    - row: 0
      column: status
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
    - column: email
      equals: "{{user_email}}"
```

## Action: `count`

Count rows matching a query. The SQL must return a `count` column (e.g., via `SELECT COUNT(*)`). Returns the parsed integer count.

```yaml
- adapter: postgresql
  action: count
  sql: "SELECT COUNT(*) FROM users WHERE status = $1"
  params: ["active"]
```

Note: The `count` action does not support `capture` or `assert`. Use the `query` action with assertions if you need to validate the count value.

## PostgreSQL Assertions

```yaml
assert:
  - row: 0                           # Row index (default: 0)
    column: "email"                  # Column name
    equals: "test@example.com"
  - column: "age"
    greaterThan: 18
    lessThan: 100
  - column: "name"
    contains: "John"
    matches: "^[A-Z]"
  - column: "deleted_at"
    isNull: true
  - column: "id"
    isNotNull: true
```

## Value Capture

Capture column values from query results:

```yaml
- adapter: postgresql
  action: queryOne
  sql: "SELECT id, email FROM users WHERE id = $1"
  capture:
    db_id: "id"                  # Column name
    db_email: "email"
```

Use captured values with `{{captured.db_id}}` in subsequent steps.
