---
name: e2e-runner
description: This skill should be used when writing E2E tests for APIs and databases using the @liemle3893/e2e-runner framework. Use when creating YAML test files, configuring adapters (HTTP, Shell, PostgreSQL, MongoDB, Redis, EventHub), writing assertions, or running tests. Provides complete syntax reference for YAML tests, assertion operators, variable interpolation, and built-in functions.
---

# E2E Test Runner

A flexible end-to-end testing framework for API and database testing. Tests are written in YAML (declarative) or TypeScript (programmatic).

## Quick Start

```yaml
name: TC-HEALTH-001
description: Verify API health endpoint
tags: [smoke]

execute:
  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/health"
    assert:
      status: 200
      json:
        - path: "$.status"
          equals: "ok"
```

Run with: `e2e run --env local`

## CLI Commands

| Command | Description |
|---------|-------------|
| `e2e run` | Execute E2E tests |
| `e2e validate` | Validate test files without execution |
| `e2e list` | List discovered tests |
| `e2e health` | Check adapter connectivity |
| `e2e init` | Initialize `e2e.config.yaml` |
| `e2e test create <name>` | Create test from template |
| `e2e doc <section>` | Show documentation for a section |
| `e2e install <adapter>` | Install adapter peer dependencies |

Common options: `--env <name>`, `--tag <tag>`, `--priority <level>`, `--parallel <n>`, `--bail`, `--watch`, `--dry-run`

## Test File Structure

Files must have `.test.yaml` extension. Four phases run sequentially:

```yaml
name: TC-FEATURE-001               # Required: unique identifier
description: What the test verifies # Optional
priority: P0                        # P0|P1|P2|P3
tags: [smoke, crud]                 # For filtering
timeout: 30000                      # Test timeout (ms)

variables:                          # Test-scoped variables
  email: "test-{{$uuid()}}@example.com"

setup: []                           # Prepare prerequisites
execute: []                         # Required: main test actions
verify: []                          # Assert expected outcomes
teardown: []                        # Cleanup (always runs)
```

## Step Definition

Each phase contains an array of steps:

```yaml
- id: create_user                   # Step identifier
  adapter: http                     # http|postgresql|mongodb|redis|eventhub|kafka|shell
  action: request                   # Adapter-specific action
  description: "Create a user"      # Optional
  continueOnError: false            # Continue on failure
  retry: 3                          # Retry count
  delay: 1000                       # Delay before execution (ms)

  # Adapter-specific params (e.g. HTTP)
  method: POST
  url: "{{baseUrl}}/users"
  body:
    email: "{{email}}"

  capture:                          # Capture values for later steps
    user_id: "$.id"

  assert:                           # Inline assertions
    status: 201
```

## Variable Interpolation

```yaml
# Test variables
"{{my_variable}}"

# Captured values from prior steps
"{{captured.user_id}}"

# Config variables (from e2e.config.yaml)
"{{baseUrl}}"

# Environment variables
"{{$env(API_KEY)}}"

# Built-in functions
"{{$uuid()}}"                       # UUID v4
"{{$timestamp()}}"                  # Unix ms
"{{$isoDate()}}"                    # ISO 8601 date
"{{$random(1, 100)}}"              # Random integer
"{{$randomString(32)}}"            # Random alphanumeric
"{{$now(date)}}"                   # Formatted date (iso|date|time|datetime|unix)
"{{$dateAdd(1, day)}}"             # Future date
"{{$dateSub(1, hour)}}"            # Past date
"{{$totp(SECRET)}}"                # TOTP code (RFC 6238)
"{{$base64(value)}}"               # Base64 encode
"{{$sha256(value)}}"               # SHA256 hash
"{{$file(./fixtures/data.json)}}"  # File contents
"{{$lower(value)}}"                # Lowercase
"{{$upper(value)}}"                # Uppercase

# Variable cross-references (resolved in dependency order)
base_id: "TEST"
run_id: "{{base_id}}_RUN"          # → "TEST_RUN"
full_id: "{{run_id}}_{{$uuid()}}"  # → "TEST_RUN_<uuid>"
# Circular references are detected and throw errors
# Max nesting depth: 10 levels
# {{baseUrl}} and {{captured.*}} refs are deferred to step time
```

## Assertion Operators

All 12 operators work across every adapter (HTTP, PostgreSQL, MongoDB, Redis):

| Operator | Type | Description |
|----------|------|-------------|
| `equals` | any | Exact value match |
| `contains` | string | Substring match |
| `matches` | string | Regex pattern match |
| `exists` | boolean | Path exists (true/false) |
| `type` | string | Type check (`string`, `number`, `boolean`, `array`, `object`, `null`) |
| `length` | number | Array/string length |
| `greaterThan` | number | Numeric > |
| `lessThan` | number | Numeric < |
| `notEmpty` | boolean | Has content |
| `isEmpty` | boolean | No content |
| `isNull` | boolean | Value is null |
| `isNotNull` | boolean | Value is not null |

## HTTP Assertions

```yaml
assert:
  status: 201                       # or [200, 201] or statusRange: [200, 299]
  headers:
    Content-Type: "application/json"
  json:
    - path: "$.id"
      exists: true
      type: "string"
    - path: "$.errors[0].code"
      equals: 8006
  body:
    contains: "success"             # Raw body assertions
  duration:
    lessThan: 1000                  # Response time (ms)
```

## Shell Commands

Execute shell commands and scripts as test steps:

```yaml
- adapter: shell
  action: exec
  command: "echo 'hello world'"
  assert:
    exitCode: 0
    stdout:
      contains: "hello"
```

Capture command output for later steps:

```yaml
- adapter: shell
  action: exec
  command: "./scripts/get-version.sh"
  cwd: "/app"
  timeout: 30000
  env:
    NODE_ENV: "test"
  capture:
    version: "stdout"
```

## Kafka Messaging

Produce and consume messages from Apache Kafka topics:

```yaml
# Produce a message
- adapter: kafka
  action: produce
  topic: "user-events"
  message:
    key: "user-123"
    value:
      type: "user.created"
      userId: "{{captured.user_id}}"

# Wait for a specific message
- adapter: kafka
  action: waitFor
  topic: "user-events"
  timeout: 30000
  filter:
    type: "user.created"
  capture:
    event_data: "data"
  assert:
    - path: "type"
      equals: "user.created"

# Consume multiple messages
- adapter: kafka
  action: consume
  topic: "events"
  count: 5
  timeout: 10000
```

## Database Assertions

**PostgreSQL** -- assert on columns and rows:

```yaml
assert:
  - column: "email"
    row: 0                          # Optional, default: 0
    equals: "test@example.com"
  - column: "deleted_at"
    isNull: true
```

**MongoDB** -- assert on document paths:

```yaml
assert:
  - path: "email"
    equals: "test@example.com"
  - path: "roles"
    type: "array"
    length: 2
```

**Redis** -- assert on value:

```yaml
assert:
  isNotNull: true
  contains: "expected"
```

## Reference Files

* **YAML Test Syntax** [references/yaml-test.md](references/yaml-test.md)
* **Assertions** [references/assertions.md](references/assertions.md)
* **Built-in Functions** [references/built-in-functions.md](references/built-in-functions.md)
* **Configuration** [references/config.md](references/config.md)
* **CLI Reference** [references/cli.md](references/cli.md)
* **Examples** [references/examples.md](references/examples.md)
* **Adapters Overview** [references/adapters/index.md](references/adapters/index.md)
* **HTTP Adapter** [references/adapters/http.md](references/adapters/http.md)
* **PostgreSQL Adapter** [references/adapters/postgresql.md](references/adapters/postgresql.md)
* **MongoDB Adapter** [references/adapters/mongodb.md](references/adapters/mongodb.md)
* **Redis Adapter** [references/adapters/redis.md](references/adapters/redis.md)
* **EventHub Adapter** [references/adapters/eventhub.md](references/adapters/eventhub.md)
* **Kafka Adapter** [references/adapters/kafka.md](references/adapters/kafka.md)
* **Shell Adapter** [references/adapters/shell.md](references/adapters/shell.md)
