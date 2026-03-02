# Testing Patterns

**Analysis Date:** 2026-03-02

## Test Framework

**Runner:**
- No unit test framework is installed or configured
- `package.json` test script: `"test": "echo \"No tests yet\""` — no runner (Jest, Vitest, etc.) present
- No `jest.config.*`, `vitest.config.*`, or equivalent found

**Run Commands:**
```bash
npm test                    # Prints "No tests yet" and exits
./bin/e2e.js run            # Runs the E2E tests defined in tests/
./bin/e2e.js run -c ./e2e.config.yaml   # With explicit config
```

**Test Coverage:**
- No coverage tooling configured — no thresholds enforced

## Test File Organization

**Location:**
- Integration/E2E test files live under `tests/e2e/adapters/`
- No unit or integration tests for source code itself exist

**Naming:**
- YAML tests use uppercase IDs: `TC-<ADAPTER>-<SCENARIO>-<NUMBER>.test.yaml`
- Examples: `TC-HTTP-ASSERTIONS-001.test.yaml`, `TC-POSTGRES-001.test.yaml`, `TC-MONGODB-001.test.yaml`

**Directory structure:**
```
tests/
└── e2e/
    └── adapters/
        ├── TC-HTTP-ASSERTIONS-001.test.yaml
        ├── TC-INTEGRATION-001.test.yaml
        ├── TC-MONGODB-001.test.yaml
        ├── TC-MONGODB-FINDONE-FILTER.test.yaml
        ├── TC-POSTGRES-001.test.yaml
        ├── TC-REDIS-001.test.yaml
        └── TC-EVENTHUB-001.test.yaml
```

## Test File Structure (YAML)

**Standard YAML test structure:**
```yaml
name: TC-POSTGRES-001
description: PostgreSQL adapter test - User CRUD operations
priority: P0
tags: [postgresql, adapter, e2e]
timeout: 30000

variables:
  test_name: "Test User Postgres"
  updated_name: "Updated User Postgres"

setup:
  - adapter: postgresql
    action: execute
    sql: "DELETE FROM users WHERE email LIKE 'test-postgres-%@example.com'"

execute:
  - adapter: http
    action: request
    method: POST
    url: "{{baseUrl}}/users"
    body:
      email: "test-postgres-{{$uuid()}}@example.com"
      name: "{{test_name}}"
    capture:
      user_id: "$.id"
    assert:
      status: 201
      json:
        - path: "$.name"
          equals: "{{test_name}}"

verify:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/users/{{captured.user_id}}"
    assert:
      status: 200
      json:
        - path: "$.name"
          equals: "{{updated_name}}"

teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/users/{{captured.user_id}}"
```

**Required fields:** `name`, `execute`

**Optional phases:** `setup`, `verify`, `teardown`

**Priority values:** `P0`, `P1`, `P2`, `P3`

## Test File Structure (TypeScript)

**TypeScript tests** export a default object conforming to `TSTestDefinition` (loaded by `src/core/ts-loader.ts`):

```typescript
// my-test.test.ts
export default {
    priority: 'P0',
    tags: ['integration'],
    timeout: 30000,

    async setup(ctx) {
        // Setup logic using ctx (AdapterContext)
    },

    async execute(ctx) {
        // Primary test logic
    },

    async verify(ctx) {
        // Post-execution assertions
    },

    async teardown(ctx) {
        // Cleanup logic
    },
}
```

The `ctx` parameter is an `AdapterContext` from `src/types.ts`, providing `variables`, `captured`, `capture()`, `logger`, and `baseUrl`.

## Assertions

**YAML assertion operators** (defined in `src/assertions/assertion-runner.ts`):
```yaml
assert:
  status: 200                    # HTTP status code
  json:
    - path: "$.data[0].id"       # JSONPath expression
      exists: true
      type: "string"
      equals: "some-value"
      contains: "partial"
      matches: "^[a-z]+$"        # Regex pattern
      notEmpty: true
      isEmpty: false
      length: 5
      greaterThan: 0
      lessThan: 100
      isNull: false
      isNotNull: true
```

**JSONPath capture:**
```yaml
capture:
  user_id: "$.id"               # Dot notation
  first_item: "$.data[0].id"    # Array indexing
  nested: "$.response.body.field"
```

**Captured values in subsequent steps:**
```yaml
url: "{{baseUrl}}/users/{{captured.user_id}}"
body:
  name: "{{test_name}}"
```

## Variable Interpolation in Tests

**Built-in functions available in YAML test params:**
```yaml
body:
  id: "{{$uuid()}}"              # UUID v4
  timestamp: "{{$timestamp}}"    # Unix timestamp
  date: "{{$isoDate}}"           # ISO 8601 datetime
  rand: "{{$random(1, 100)}}"    # Random integer
  token: "{{$randomString(16)}}" # Random alphanumeric string
  hashed: "{{$md5(value)}}"      # MD5 hash
  encoded: "{{$base64(value)}}"  # Base64 encode
```

**Variable resolution order:**
1. Captured values (`captured.fieldName`)
2. Test-level variables (`variables:` block)
3. Global config variables
4. Environment variables (via `process.env`)

## Mocking

**Framework:** None — no mocking library present

**Approach:** Tests run against real services started via `docker-compose.yaml` in the project root. There is no in-process service virtualization or HTTP interception.

**What is tested against real services:**
- PostgreSQL via the `postgresql` adapter
- MongoDB via the `mongodb` adapter
- Redis via the `redis` adapter
- Azure EventHub via the `eventhub` adapter
- HTTP APIs via the `http` adapter (requires running `demo-server/`)

**No mocking, no stubs:** Tests depend on live infrastructure. Use `docker-compose up` to start dependencies before running tests.

## Test Execution Configuration

**Config file:** `e2e.config.yaml` (project root)

Required fields:
```yaml
version: '1.0'
environments:
  local:
    baseUrl: http://localhost:3000
    adapters:
      postgresql:
        connectionString: postgresql://user:pass@localhost:5432/db
      mongodb:
        connectionString: mongodb://localhost:27017
        database: mydb
      redis:
        connectionString: redis://localhost:6379
defaults:
  timeout: 30000
  retries: 0
  parallel: 1
```

## Test Isolation Patterns

**Setup/Teardown for isolation:**
- The `setup` phase deletes or seeds data before test execution
- The `teardown` phase cleans up created resources
- `continueOnError: true` on teardown steps prevents cleanup failures from masking test results

**Unique data per test run:**
```yaml
body:
  email: "test-{{$uuid()}}@example.com"    # Unique email per run
  title: "Test Document {{$uuid()}}"        # Unique title per run
```

**Example teardown with fault tolerance:**
```yaml
teardown:
  - adapter: http
    action: request
    method: DELETE
    url: "{{baseUrl}}/documents/{{captured.doc_id}}"
    continueOnError: true     # Don't fail the test if cleanup fails
```

## Test Types

**Unit Tests:**
- Not present — no unit tests exist for source code

**Integration Tests:**
- All tests in `tests/e2e/adapters/` are integration tests
- They test the full stack: CLI → Orchestrator → Adapter → Real Service

**E2E Tests:**
- Synonymous with integration tests in this project
- All tests exercise real HTTP endpoints and database adapters end-to-end

## Retry Configuration

Tests can configure retry behavior per-step or globally:
```yaml
# Per-step retry
- adapter: http
  action: request
  retry: 3          # Retry up to 3 times
  delay: 500        # Wait 500ms before step

# Global via config defaults
defaults:
  retries: 2
  retryDelay: 1000
```

The retry implementation uses exponential backoff with jitter (see `src/utils/retry.ts`).

## Test Filtering

Tests can be run selectively using CLI flags:
```bash
./bin/e2e.js run --tag smoke          # Run only tests tagged "smoke"
./bin/e2e.js run --priority P0        # Run only P0 tests
./bin/e2e.js run --grep "POSTGRES"    # Run tests matching name pattern
./bin/e2e.js run --bail               # Stop on first failure
```

## Test Coverage Gaps

**No unit tests exist** for:
- `src/core/variable-interpolator.ts` — complex interpolation logic with many built-in functions
- `src/assertions/matchers.ts` — all matcher implementations
- `src/assertions/assertion-runner.ts` — assertion operator dispatch
- `src/utils/retry.ts` — retry and timeout logic
- `src/core/test-discovery.ts` — file discovery and filtering
- `src/core/yaml-loader.ts` — YAML parsing and validation
- `src/adapters/*.ts` — all adapter logic

**No test runner is configured.** Adding unit tests requires selecting and installing a test framework (Vitest recommended for TypeScript projects).

---

*Testing analysis: 2026-03-02*
