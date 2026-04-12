# Tryve

A YAML-driven, multi-protocol end-to-end test runner. Write tests in YAML, run them against HTTP APIs and databases, and get results in console, JUnit, HTML, or JSON.

Single binary. Zero runtime dependencies. Cross-platform.

## Installation

### Shell (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/liemle3893/go-tryve/main/install.sh | sh
```

Options: `--dir /custom/path` to change install location, `--version v2.0.0` to pin a version.

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/liemle3893/go-tryve/main/install.ps1 | iex
```

Options: `-Dir C:\custom\path` to change install location, `-Version v2.0.0` to pin a version.

### Go Install

```bash
go install github.com/liemle3893/go-tryve/cmd/tryve@latest
```

### From Source

```bash
git clone https://github.com/liemle3893/go-tryve.git
cd go-tryve
make build    # binary at ./bin/tryve
```

## Quick Start

```bash
# Create a config file
tryve init

# Create a test
tryve test create my-api-test

# Run all tests
tryve run

# Run with filters
tryve run --tag smoke --bail
```

## Writing Tests

Tests are YAML files with four optional phases: `setup`, `execute`, `verify`, and `teardown`.

```yaml
name: Create and verify user
description: Full user lifecycle test
tags: [smoke, users]
priority: P1

variables:
  email: "test-${uuid()}@example.com"

setup:
  - id: clean-slate
    adapter: postgresql
    action: exec
    params:
      query: "DELETE FROM users WHERE email LIKE 'test-%@example.com'"

execute:
  - id: create-user
    adapter: http
    action: POST
    params:
      url: /api/users
      body:
        email: "${email}"
        name: "Test User"
    capture:
      userId: $.body.id
    assertions:
      - path: $.status
        operator: equals
        expected: 201

  - id: get-user
    adapter: http
    action: GET
    params:
      url: "/api/users/${captured.userId}"
    assertions:
      - path: $.body.email
        operator: equals
        expected: "${email}"

verify:
  - id: check-db
    adapter: postgresql
    action: query
    params:
      query: "SELECT * FROM users WHERE id = $1"
      params: ["${captured.userId}"]
    assertions:
      - path: $.rows[0].email
        operator: equals
        expected: "${email}"

teardown:
  - id: cleanup
    adapter: postgresql
    action: exec
    params:
      query: "DELETE FROM users WHERE id = $1"
      params: ["${captured.userId}"]
```

## Configuration

`e2e.config.yaml` defines environments, adapters, defaults, and variables.

```yaml
version: "1.0"

environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters:
      postgresql:
        connectionString: "${POSTGRESQL_CONNECTION_STRING}"
      mongodb:
        connectionString: "${MONGODB_CONNECTION_STRING}"
      redis:
        connectionString: "${REDIS_CONNECTION_STRING}"

defaults:
  timeout: 30000
  retries: 1
  retryDelay: 1000
  parallel: 4

variables:
  testPrefix: "e2e_test_"

reporters:
  - type: console
    verbose: true
  - type: junit
    output: "./reports/junit.xml"
```

## Adapters

| Adapter | Actions | Config Key |
|---------|---------|------------|
| **HTTP** | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` | `baseUrl` |
| **PostgreSQL** | `query`, `exec` | `connectionString` |
| **MongoDB** | `find`, `findOne`, `insertOne`, `updateOne`, `deleteOne`, `deleteMany`, `countDocuments`, `aggregate` | `connectionString` |
| **Redis** | `get`, `set`, `del`, `exists`, `hget`, `hset`, `hgetall`, `lpush`, `rpush`, `lrange`, `sadd`, `smembers`, `publish` | `connectionString` |
| **Kafka** | `produce`, `consume` | `brokers` |
| **Azure EventHub** | `send`, `receive` | `connectionString`, `eventHubName` |
| **Shell** | `exec` | _(none)_ |

## Assertions

Use JSONPath expressions in `path` to extract values, then assert with an operator.

| Operator | Description |
|----------|-------------|
| `equals` | Strict equality |
| `notEquals` | Not equal |
| `contains` | String/array contains |
| `notContains` | Does not contain |
| `matches` | Regex match |
| `type` | Type check (`string`, `number`, `boolean`, `array`, `object`, `null`) |
| `exists` | Value is present |
| `notExists` | Value is absent |
| `isNull` | Value is null |
| `isNotNull` | Value is not null |
| `greaterThan` | Numeric `>` |
| `lessThan` | Numeric `<` |
| `greaterThanOrEqual` | Numeric `>=` |
| `lessThanOrEqual` | Numeric `<=` |
| `length` | Array/string length equals |
| `isEmpty` | Array/string is empty |
| `notEmpty` | Array/string is not empty |
| `hasProperty` | Object has key |
| `notHasProperty` | Object lacks key |

## Built-in Functions

Use `${functionName(args)}` in any string value.

| Function | Description | Example Output |
|----------|-------------|----------------|
| `uuid()` | UUID v4 | `550e8400-e29b-41d4-a716-446655440000` |
| `now()` | Current time (ISO 8601) | `2026-04-12T10:30:00Z` |
| `timestamp()` | Unix timestamp (seconds) | `1744454400` |
| `isoDate()` | Current date (ISO) | `2026-04-12` |
| `random()` | Random float 0-1 | `0.7234` |
| `randomString(len)` | Random alphanumeric string | `aB3kZ9mQ` |
| `env(name)` | Environment variable | _(value of $name)_ |
| `file(path)` | Read file contents | _(file contents)_ |
| `base64(value)` | Base64 encode | `aGVsbG8=` |
| `base64Decode(value)` | Base64 decode | `hello` |
| `md5(value)` | MD5 hash | `5d41402abc4b2a76...` |
| `sha256(value)` | SHA-256 hash | `2cf24dba5fb0a30e...` |
| `dateAdd(duration)` | Current time + duration | `2026-04-13T10:30:00Z` |
| `dateSub(duration)` | Current time - duration | `2026-04-11T10:30:00Z` |
| `totp(secret)` | TOTP code | `483920` |
| `jsonStringify(value)` | JSON encode | `"{\"key\":\"val\"}"` |
| `lower(value)` | Lowercase | `hello` |
| `upper(value)` | Uppercase | `HELLO` |
| `trim(value)` | Trim whitespace | `hello` |

## CLI Reference

### Commands

```
tryve run          Run tests
tryve validate     Validate test file syntax
tryve list         List discovered tests
tryve health       Check adapter connectivity
tryve init         Create e2e.config.yaml
tryve test create  Create a test from template
tryve doc          Browse built-in documentation
tryve install      Install Claude Code skills
tryve version      Print version
```

### `tryve run` Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Config file path | `e2e.config.yaml` |
| `--env` | `-e` | Environment name | `local` |
| `--test-dir` | `-d` | Test directory | `tests` |
| `--parallel` | `-p` | Parallel test count | config default |
| `--timeout` | `-t` | Per-test timeout (ms) | config default |
| `--retries` | `-r` | Retry count on failure | config default |
| `--bail` | | Stop on first failure | `false` |
| `--grep` | `-g` | Filter by name (regex) | |
| `--tag` | | Filter by tag (repeatable) | |
| `--priority` | | Filter by priority (P0-P3) | |
| `--dry-run` | | Show matching tests without running | `false` |
| `--skip-setup` | | Skip setup phase | `false` |
| `--skip-teardown` | | Skip teardown phase | `false` |
| `--reporter` | | Reporter type (repeatable) | `console` |
| `--output` | `-o` | Output path for file reporters | |
| `--verbose` | | Show per-step detail | `false` |
| `--debug` | | Show full request/response data | `false` |
| `--watch` | | Re-run on file changes | `false` |

### Reporters

| Type | Output | Use Case |
|------|--------|----------|
| `console` | Terminal | Local development |
| `junit` | XML file | CI/CD integration |
| `html` | HTML file | Shareable reports |
| `json` | JSON file | Programmatic consumption |

```bash
# Multiple reporters
tryve run --reporter console --reporter junit -o reports/results.xml
```

## Programmatic Usage (Go)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liemle3893/go-tryve/pkg/runner"
)

func main() {
    r, err := runner.New(
        runner.WithConfig("e2e.config.yaml"),
        runner.WithEnv("local"),
        runner.WithTags("smoke"),
        runner.WithParallel(4),
    )
    if err != nil {
        log.Fatal(err)
    }

    result, err := r.Run(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Passed: %d, Failed: %d\n", result.Passed, result.Failed)
}
```

## Development

```bash
make build       # Build binary to bin/tryve
make test        # Run all tests
make test-v      # Run tests with verbose output
make lint        # Run golangci-lint
make clean       # Remove build artifacts
```

## License

MIT
