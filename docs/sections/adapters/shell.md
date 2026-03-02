# Shell Adapter

Execute shell commands and scripts as test steps. Useful for CLI tool testing, database migrations, setup/teardown scripts, file operations, and system health checks.

## Configuration

```yaml
environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters:
      shell:
        defaultTimeout: 30000       # Default command timeout in ms (default: 30000)
        defaultCwd: "/app"           # Default working directory
        defaultEnv:                  # Default environment variables
          NODE_ENV: "test"
```

The Shell adapter uses the built-in Node.js `child_process` module and has **no peer dependencies**.

Commands originate from static YAML test files written by the test author (the same trust model as CI/CD systems). No untrusted user input is passed to the shell.

## Action: `exec`

Execute a shell command with full shell feature support (pipes, redirects, globbing, subshells).

```yaml
- adapter: shell
  action: exec
  command: string                    # Required: shell command to execute
  cwd?: string                      # Working directory override
  timeout?: number                  # Command timeout in ms (default: 30000)
  env?: Record<string, string>      # Environment variables (merged with process.env)
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | Yes | - | The shell command to execute |
| `cwd` | string | No | process.cwd() | Working directory for the command |
| `timeout` | number | No | 30000 | Kill command after this many milliseconds |
| `env` | object | No | process.env | Environment variables merged with process.env |

### Response Structure

The `exec` action returns the following data structure:

```typescript
{
  exitCode: number;    // Process exit code (0 = success)
  stdout: string;      // Standard output content
  stderr: string;      // Standard error content
  duration: number;    // Execution time in milliseconds
}
```

Non-zero exit codes are returned as data, not as errors. This allows you to assert on specific exit codes.

## Assertions

Assert on exit code, stdout, and stderr content:

```yaml
assert:
  # Exit code (exact match)
  exitCode: 0

  # Standard output assertions
  stdout:
    contains: "expected substring"
    matches: "^pattern\\d+$"         # Regex pattern
    equals: "exact output"

  # Standard error assertions
  stderr:
    contains: "warning"
    matches: "ERROR.*timeout"
    equals: "exact error message"
```

### Assertion Operators

| Field | Operator | Description |
|-------|----------|-------------|
| `exitCode` | exact number | Exit code must match exactly |
| `stdout.contains` | substring | stdout must include the string |
| `stdout.matches` | regex | stdout must match the pattern |
| `stdout.equals` | exact (trimmed) | stdout must equal the value (whitespace-trimmed) |
| `stderr.contains` | substring | stderr must include the string |
| `stderr.matches` | regex | stderr must match the pattern |
| `stderr.equals` | exact (trimmed) | stderr must equal the value (whitespace-trimmed) |

## Value Capture

Capture stdout, stderr, or exit code for use in later steps:

```yaml
capture:
  output: "stdout"      # Capture full stdout as variable
  errors: "stderr"      # Capture full stderr as variable
  code: "exitCode"      # Capture exit code as number
```

Use captured values in subsequent steps with `{{captured.output}}`.

## Examples

**Basic: Echo command with exit code assertion**
```yaml
- adapter: shell
  action: exec
  command: "echo 'hello world'"
  assert:
    exitCode: 0
    stdout:
      contains: "hello"
```

**Script: Run a script file and capture output**
```yaml
- adapter: shell
  action: exec
  command: "./scripts/get-version.sh"
  capture:
    version: "stdout"
  assert:
    exitCode: 0
```

**Setup: Database migration in setup phase**
```yaml
setup:
  - adapter: shell
    action: exec
    command: "npx prisma migrate deploy"
    timeout: 60000
    env:
      DATABASE_URL: "postgresql://user:pass@localhost:5432/testdb"
    assert:
      exitCode: 0
```

**Environment: Pass env vars and set cwd**
```yaml
- adapter: shell
  action: exec
  command: "npm run seed"
  cwd: "/app/backend"
  timeout: 30000
  env:
    NODE_ENV: "test"
    DB_HOST: "localhost"
  assert:
    exitCode: 0
```

**Capture: Use command output in later steps**
```yaml
execute:
  - adapter: shell
    action: exec
    command: "curl -s http://localhost:3000/version"
    capture:
      api_version: "stdout"
    assert:
      exitCode: 0

  - adapter: http
    action: request
    method: GET
    url: "{{baseUrl}}/health"
    assert:
      status: 200
      json:
        - path: "$.version"
          equals: "{{captured.api_version}}"
```

## Timeout Behavior

When a command exceeds its timeout:
- The process is killed (SIGTERM)
- An `AdapterError` is thrown with a "timed out" message
- The step fails immediately
