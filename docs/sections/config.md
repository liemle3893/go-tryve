# Configuration Reference

The E2E Test Runner is configured via `e2e.config.yaml` in your project root.

## Complete Configuration Schema

```yaml
version: "1.0"                    # Required: config version

testDir: "tests/e2e"              # Optional: test directory (default: ".")

environments:                     # Required: at least one environment
  local:                          # Environment name
    baseUrl: "http://localhost:3000"  # Required: base URL for HTTP adapter
    adapters:                     # Optional: adapter configurations
      postgresql:
        connectionString: "postgresql://user:pass@localhost:5432/db"
        schema: "public"          # Default schema
        poolSize: 10              # Connection pool size

      redis:
        connectionString: "redis://localhost:6379"
        db: 0                     # Redis database number
        keyPrefix: ""             # Key prefix for all operations

      mongodb:
        connectionString: "mongodb://user:pass@localhost:27017"
        database: "mydb"          # Database name

      eventhub:
        connectionString: "Endpoint=sb://...;EntityPath=events"
        consumerGroup: "$Default" # Consumer group name
        checkpointStore: ""       # Optional: checkpoint store connection

  staging:
    baseUrl: "https://staging.example.com"
    adapters:
      # Staging adapter configs...

  production:
    baseUrl: "https://api.example.com"
    adapters:
      # Production adapter configs...

defaults:                         # Optional: default settings
  timeout: 30000                  # Default test timeout (ms)
  retries: 0                      # Default retry count
  retryDelay: 1000                # Delay between retries (ms)
  parallel: 1                     # Parallel test count

variables:                        # Optional: global variables
  testPrefix: "e2e_"
  defaultUserId: "test-user"
  apiVersion: "v1"

hooks:                            # Optional: lifecycle hooks
  beforeAll: "npm run seed"       # Shell command before all tests
  afterAll: "npm run cleanup"     # Shell command after all tests
  beforeEach: ""                  # Shell command before each test
  afterEach: ""                   # Shell command after each test

reporters:                        # Optional: report configuration
  - type: console
    verbose: true

  - type: junit
    output: "./reports/junit.xml"

  - type: html
    output: "./reports/report.html"

  - type: json
    output: "./reports/results.json"
```

## Top-Level Options

| Option        | Type                          | Default | Description                              |
|---------------|-------------------------------|---------|------------------------------------------|
| `version`     | `"1.0"`                       | —       | Required. Config schema version.         |
| `testDir`     | `string`                      | `"."`   | Directory to discover test files in.     |
| `environments`| `Record<string, Environment>` | —       | Required. At least one environment.      |
| `defaults`    | `DefaultsConfig`              | —       | Default timeout, retries, parallelism.   |
| `variables`   | `Record<string, string\|number\|boolean>` | — | Global variables for all tests. |
| `hooks`       | `HooksConfig`                 | —       | Lifecycle shell commands.                |
| `reporters`   | `ReporterConfig[]`            | `[{type:"console"}]` | Output format configuration. |

## Environment Configuration

### Base URL

The `baseUrl` is required and used as the base for all HTTP requests:

```yaml
environments:
  local:
    baseUrl: "http://localhost:3000"
```

Access in tests via `{{baseUrl}}`:

```yaml
- adapter: http
  action: request
  url: "{{baseUrl}}/api/users"
```

### Adapter Configuration

Each adapter has specific configuration options:

#### PostgreSQL

| Option             | Type     | Default    | Description                   |
|--------------------|----------|------------|-------------------------------|
| `connectionString` | `string` | —          | Required. PostgreSQL URI.     |
| `schema`           | `string` | `"public"` | Default schema for queries.   |
| `poolSize`         | `number` | —          | Connection pool size.         |

```yaml
postgresql:
  connectionString: "postgresql://user:password@host:port/database"
  schema: "public"      # Default schema for queries
  poolSize: 10          # Connection pool size
```

#### Redis

| Option             | Type     | Default | Description                   |
|--------------------|----------|---------|-------------------------------|
| `connectionString` | `string` | —       | Required. Redis URI.          |
| `db`               | `number` | `0`     | Database number (0-15).       |
| `keyPrefix`        | `string` | `""`    | Prefix added to all keys.     |

```yaml
redis:
  connectionString: "redis://user:password@host:port"
  db: 0                 # Database number (0-15)
  keyPrefix: "test:"    # Prefix added to all keys
```

#### MongoDB

| Option             | Type     | Default | Description                   |
|--------------------|----------|---------|-------------------------------|
| `connectionString` | `string` | —       | Required. MongoDB URI.        |
| `database`         | `string` | —       | Database name.                |

```yaml
mongodb:
  connectionString: "mongodb://user:password@host:port"
  database: "mydb"      # Database name
```

#### EventHub

| Option             | Type     | Default      | Description                        |
|--------------------|----------|--------------|------------------------------------|
| `connectionString` | `string` | —            | Required. EventHub connection.     |
| `consumerGroup`    | `string` | `"$Default"` | Consumer group name.               |
| `checkpointStore`  | `string` | —            | Checkpoint store connection string.|

```yaml
eventhub:
  connectionString: "Endpoint=sb://namespace.servicebus.windows.net/;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=hub-name"
  consumerGroup: "$Default"
```

For local development with EventHub emulator:

```yaml
eventhub:
  connectionString: "Endpoint=sb://localhost;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SAS_KEY_VALUE;UseDevelopmentEmulator=true;EntityPath=events"
  consumerGroup: "$Default"
```

## Environment Variables

Use `${env.VAR_NAME}` or `${VAR_NAME}` in configuration to reference environment variables:

```yaml
environments:
  staging:
    baseUrl: "${STAGING_URL}"
    adapters:
      postgresql:
        connectionString: "${STAGING_PG_CONNECTION_STRING}"
      redis:
        connectionString: "${STAGING_REDIS_URL}"
```

Set variables before running:

```bash
export STAGING_URL="https://staging.example.com"
export STAGING_PG_CONNECTION_STRING="postgresql://..."
npx e2e run --env staging
```

## Global Variables

Define variables accessible in all tests:

```yaml
variables:
  testPrefix: "e2e_test_"
  defaultUserId: "test-user-001"
  apiKey: "${API_KEY}"
```

Access in tests:

```yaml
execute:
  - adapter: http
    action: request
    url: "{{baseUrl}}/users/{{defaultUserId}}"
    headers:
      X-API-Key: "{{apiKey}}"
```

## Default Settings

Configure defaults applied to all tests:

| Option       | Type     | Default | Description                     |
|--------------|----------|---------|---------------------------------|
| `timeout`    | `number` | `30000` | Test timeout in ms.             |
| `retries`    | `number` | `0`     | Retry count for failed tests.   |
| `retryDelay` | `number` | `1000`  | Delay between retries in ms.    |
| `parallel`   | `number` | `1`     | Number of tests to run concurrently. |

```yaml
defaults:
  timeout: 30000    # 30 second timeout
  retries: 0        # No retries by default
  retryDelay: 1000  # 1 second between retries
  parallel: 1       # Sequential execution by default
```

Override in individual tests:

```yaml
name: TC-SLOW-001
timeout: 120000     # Override: 2 minute timeout
retries: 5          # Override: 5 retries

execute:
  - adapter: http
    action: request
    url: "{{baseUrl}}/slow-endpoint"
```

## Hooks

Run shell commands at specific points in the test lifecycle:

| Hook         | Type     | Description                          |
|--------------|----------|--------------------------------------|
| `beforeAll`  | `string` | Runs once before all tests start.    |
| `afterAll`   | `string` | Runs once after all tests complete.  |
| `beforeEach` | `string` | Runs before each individual test.    |
| `afterEach`  | `string` | Runs after each individual test.     |

```yaml
hooks:
  beforeAll: "npm run db:seed"
  afterAll: "npm run db:cleanup"
  beforeEach: "echo 'Starting test...'"
  afterEach: "echo 'Test complete.'"
```

## Reporters

Configure multiple reporters for different output formats.

Each reporter has these fields:

| Field     | Type                                    | Default | Description                          |
|-----------|-----------------------------------------|---------|--------------------------------------|
| `type`    | `"console" \| "junit" \| "html" \| "json"` | —   | Required. Reporter type.             |
| `output`  | `string`                                | —       | Output file path (required for non-console). |
| `verbose` | `boolean`                               | —       | Show detailed step output.           |

### Console Reporter

Real-time terminal output:

```yaml
reporters:
  - type: console
    verbose: true     # Show detailed step output
```

### JUnit Reporter

XML format for CI/CD integration:

```yaml
reporters:
  - type: junit
    output: "./reports/junit.xml"
```

### HTML Reporter

Interactive HTML report:

```yaml
reporters:
  - type: html
    output: "./reports/report.html"
```

### JSON Reporter

Machine-readable JSON:

```yaml
reporters:
  - type: json
    output: "./reports/results.json"
```

## Multiple Environments

Define multiple environments for different stages:

```yaml
environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters:
      postgresql:
        connectionString: "postgresql://postgres:postgres@localhost:5432/dev"

  staging:
    baseUrl: "https://staging-api.example.com"
    adapters:
      postgresql:
        connectionString: "${STAGING_DB_URL}"

  production:
    baseUrl: "https://api.example.com"
    adapters:
      postgresql:
        connectionString: "${PROD_DB_URL}"
```

Run against specific environment:

```bash
npx e2e run --env staging
npx e2e run --env production
```
