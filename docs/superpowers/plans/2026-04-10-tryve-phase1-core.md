# Tryve Phase 1: Core + HTTP + Shell + Console + CLI

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a working `tryve` binary that can load config, parse YAML tests, execute HTTP and shell steps with variable interpolation and assertions, and report results to the console.

**Architecture:** Go-native design with interfaces for adapters and reporters, `context.Context` for lifecycle management, and `errgroup` for bounded parallel execution. Shared types in `internal/tryve/`, all other packages import from there to avoid circular deps.

**Tech Stack:** Go 1.23+, cobra (CLI), yaml.v3, pgx (future), ohler55/ojg (JSONPath), google/uuid, pquerna/otp, fsnotify (future), golang.org/x/sync/errgroup

**Spec:** `docs/superpowers/specs/2026-04-10-go-port-design.md`

---

## Future Plans

This is Plan 1 of 4:

- **Plan 1 (this):** Core infrastructure + HTTP adapter + Shell adapter + Console reporter + CLI
- **Plan 2:** Database & queue adapters (PostgreSQL, MongoDB, Redis, Kafka, EventHub)
- **Plan 3:** Additional reporters (JUnit, HTML, JSON) + watch mode + embedded docs
- **Plan 4:** Public Go API (`pkg/runner`), GoReleaser, GitHub Actions, cross-compilation

---

## File Structure

All files created by this plan:

```
cmd/tryve/main.go                          # CLI entrypoint
internal/tryve/errors.go                   # TryveError + constructors
internal/tryve/errors_test.go
internal/tryve/types.go                    # Shared types (TestDefinition, StepResult, etc.)
internal/config/types.go                   # Config struct definitions
internal/config/config.go                  # Load + validate + env resolution
internal/config/config_test.go
internal/interpolate/builtins.go           # 19 built-in functions
internal/interpolate/builtins_test.go
internal/interpolate/interpolate.go        # Parser + resolver + topo sort
internal/interpolate/interpolate_test.go
internal/assertion/jsonpath.go             # JSONPath tokenizer + evaluator
internal/assertion/jsonpath_test.go
internal/assertion/matchers.go             # All assertion operators
internal/assertion/matchers_test.go
internal/assertion/assertion.go            # runAssertion() entrypoint
internal/assertion/assertion_test.go
internal/loader/discovery.go               # Glob-based file discovery
internal/loader/parser.go                  # YAML -> TestDefinition
internal/loader/validator.go               # Adapter-specific validation
internal/loader/loader_test.go
internal/adapter/adapter.go                # Interface + helpers
internal/adapter/registry.go               # Lazy initialization registry
internal/adapter/registry_test.go
internal/adapter/http.go                   # HTTP adapter
internal/adapter/http_test.go
internal/adapter/shell.go                  # Shell adapter
internal/adapter/shell_test.go
internal/reporter/reporter.go              # Interface + Multi
internal/reporter/console.go               # Console reporter
internal/reporter/console_test.go
internal/executor/step.go                  # Step execution
internal/executor/step_test.go
internal/executor/hooks.go                 # Shell hook execution
internal/executor/runner.go                # Test-level: phases + retries
internal/executor/runner_test.go
internal/executor/orchestrator.go          # Suite-level: parallel + deps + bail
internal/executor/orchestrator_test.go
internal/cli/root.go                       # Root cobra command
internal/cli/run.go                        # run command
internal/cli/validate.go                   # validate command
internal/cli/list.go                       # list command
internal/cli/health.go                     # health command
internal/cli/init_cmd.go                   # init command
internal/cli/version.go                    # version command
internal/cli/test_cmd.go                   # test create / test list-templates
Makefile
go.mod
go.sum
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`, `Makefile`, `cmd/tryve/main.go`, `.gitignore` (update)
- Move: `src/` → `ts/src/`, `package.json` → `ts/package.json`, and all other TS files

- [ ] **Step 1: Move TypeScript source to `ts/` directory**

```bash
mkdir -p ts
git mv src/ ts/src/
git mv package.json ts/package.json
git mv package-lock.json ts/package-lock.json
git mv tsconfig.json ts/tsconfig.json
git mv vitest.config.ts ts/vitest.config.ts
git mv bin/ ts/bin/
git mv dist/ ts/dist/ 2>/dev/null || true
```

Note: `docs/`, `tests/`, `skills/`, `CLAUDE.md`, `AGENTS.md`, `README.md`, `docker-compose.yaml`, `e2e.config.yaml` stay at root — they're shared.

- [ ] **Step 2: Create `go.mod`**

```go
// go.mod
module github.com/liemle3893/go-tryve

go 1.23

require (
	github.com/google/uuid v1.6.0
	github.com/ohler55/ojg v1.25.0
	github.com/pquerna/otp v1.4.0
	github.com/spf13/cobra v1.8.1
	golang.org/x/sync v0.10.0
	gopkg.in/yaml.v3 v3.0.1
)
```

Run: `go mod tidy` (will resolve exact versions)

- [ ] **Step 3: Create `Makefile`**

```makefile
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/tryve ./cmd/tryve

test:
	go test ./...

test-v:
	go test -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

run:
	go run ./cmd/tryve $(ARGS)
```

- [ ] **Step 4: Create stub `cmd/tryve/main.go`**

```go
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "tryve %s\n", version)
	os.Exit(0)
}
```

- [ ] **Step 5: Update `.gitignore`**

Append to existing `.gitignore`:
```
# Go
bin/
*.exe
*.test
*.out
```

- [ ] **Step 6: Verify build**

Run: `go build -o bin/tryve ./cmd/tryve && ./bin/tryve`
Expected: `tryve dev`

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: scaffold Go project, move TypeScript to ts/"
```

---

### Task 2: Shared Types & Errors

**Files:**
- Create: `internal/tryve/errors.go`, `internal/tryve/errors_test.go`, `internal/tryve/types.go`

- [ ] **Step 1: Write error type tests**

```go
// internal/tryve/errors_test.go
package tryve_test

import (
	"errors"
	"testing"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestTryveError_Error(t *testing.T) {
	err := tryve.ConfigError("invalid config", "check e2e.config.yaml", nil)
	if err.Error() != "invalid config" {
		t.Errorf("got %q, want %q", err.Error(), "invalid config")
	}
	if err.Code != "CONFIG_ERROR" {
		t.Errorf("got code %q, want %q", err.Code, "CONFIG_ERROR")
	}
	if err.Hint != "check e2e.config.yaml" {
		t.Errorf("got hint %q, want %q", err.Hint, "check e2e.config.yaml")
	}
}

func TestTryveError_Unwrap(t *testing.T) {
	cause := errors.New("file not found")
	err := tryve.ConfigError("load failed", "", cause)
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}

func TestTryveError_ErrorWithCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := tryve.ConnectionError("postgresql", "connect failed", cause)
	want := "connect failed: connection refused"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestTryveError_TypeCheck(t *testing.T) {
	err := tryve.AssertionError("$.status", "equals", 200, 404)
	var te *tryve.TryveError
	if !errors.As(err, &te) {
		t.Error("errors.As should match TryveError")
	}
	if te.Code != "ASSERTION_ERROR" {
		t.Errorf("got code %q, want %q", te.Code, "ASSERTION_ERROR")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tryve/...`
Expected: compilation error (package doesn't exist yet)

- [ ] **Step 3: Implement error types**

```go
// internal/tryve/errors.go
package tryve

import (
	"fmt"
	"time"
)

// TryveError is the structured error type for all tryve errors.
type TryveError struct {
	Code    string
	Message string
	Hint    string
	Cause   error
}

func (e *TryveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *TryveError) Unwrap() error { return e.Cause }

func ConfigError(msg, hint string, cause error) *TryveError {
	return &TryveError{Code: "CONFIG_ERROR", Message: msg, Hint: hint, Cause: cause}
}

func ValidationError(msg, hint string, cause error) *TryveError {
	return &TryveError{Code: "VALIDATION_ERROR", Message: msg, Hint: hint, Cause: cause}
}

func ConnectionError(adapter, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "CONNECTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check %s connection settings in e2e.config.yaml", adapter),
		Cause:   cause,
	}
}

func ExecutionError(step, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "EXECUTION_ERROR",
		Message: msg,
		Hint:    fmt.Sprintf("check step %q configuration", step),
		Cause:   cause,
	}
}

func AssertionError(path, operator string, expected, actual any) *TryveError {
	return &TryveError{
		Code:    "ASSERTION_ERROR",
		Message: fmt.Sprintf("assertion failed: %s %s %v, got %v", path, operator, expected, actual),
	}
}

func TimeoutError(operation string, duration time.Duration) *TryveError {
	return &TryveError{
		Code:    "TIMEOUT_ERROR",
		Message: fmt.Sprintf("%s timed out after %s", operation, duration),
		Hint:    "increase timeout in config or step definition",
	}
}

func InterpolationError(expr, msg string) *TryveError {
	return &TryveError{
		Code:    "INTERPOLATION_ERROR",
		Message: fmt.Sprintf("interpolation error in %q: %s", expr, msg),
	}
}

func AdapterError(adapter, action, msg string, cause error) *TryveError {
	return &TryveError{
		Code:    "ADAPTER_ERROR",
		Message: fmt.Sprintf("%s.%s: %s", adapter, action, msg),
		Cause:   cause,
	}
}
```

- [ ] **Step 4: Implement shared types**

```go
// internal/tryve/types.go
package tryve

import "time"

// TestPriority represents P0-P3 priority levels.
type TestPriority string

const (
	PriorityP0 TestPriority = "P0"
	PriorityP1 TestPriority = "P1"
	PriorityP2 TestPriority = "P2"
	PriorityP3 TestPriority = "P3"
)

// TestStatus represents the outcome of a test or step.
type TestStatus string

const (
	StatusPassed  TestStatus = "passed"
	StatusFailed  TestStatus = "failed"
	StatusSkipped TestStatus = "skipped"
	StatusWarned  TestStatus = "warned"
)

// TestPhase represents the four phases of test execution.
type TestPhase string

const (
	PhaseSetup    TestPhase = "setup"
	PhaseExecute  TestPhase = "execute"
	PhaseVerify   TestPhase = "verify"
	PhaseTeardown TestPhase = "teardown"
)

// TestDefinition represents a parsed YAML test file.
type TestDefinition struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Priority    TestPriority      `yaml:"priority"`
	Tags        []string          `yaml:"tags"`
	Skip        bool              `yaml:"skip"`
	SkipReason  string            `yaml:"skipReason"`
	Timeout     int               `yaml:"timeout"`
	Retries     int               `yaml:"retries"`
	Depends     []string          `yaml:"depends"`
	Variables   map[string]any    `yaml:"variables"`
	Setup       []StepDefinition  `yaml:"setup"`
	Execute     []StepDefinition  `yaml:"execute"`
	Verify      []StepDefinition  `yaml:"verify"`
	Teardown    []StepDefinition  `yaml:"teardown"`
	SourceFile  string            `yaml:"-"`
}

// StepDefinition represents a single step within a test phase.
// In YAML, adapter-specific fields are at the top level and collected into Params by the parser.
type StepDefinition struct {
	ID              string            `yaml:"-"`
	Adapter         string            `yaml:"adapter"`
	Action          string            `yaml:"action"`
	Description     string            `yaml:"description"`
	Params          map[string]any    `yaml:"-"`
	Capture         map[string]string `yaml:"capture"`
	Assert          any               `yaml:"assert"`
	ContinueOnError bool             `yaml:"continueOnError"`
	Retry           int               `yaml:"retry"`
	Delay           int               `yaml:"delay"`
}

// StepResult is the unified return from adapter execution.
type StepResult struct {
	Data     map[string]any
	Duration time.Duration
	Metadata map[string]any
}

// TestResult holds the outcome of a single test.
type TestResult struct {
	Test       *TestDefinition
	Status     TestStatus
	Duration   time.Duration
	Steps      []StepOutcome
	Error      error
	RetryCount int
}

// StepOutcome records the result of executing one step.
type StepOutcome struct {
	Step       *StepDefinition
	Phase      TestPhase
	Status     TestStatus
	Result     *StepResult
	Assertions []AssertionOutcome
	Error      error
	Duration   time.Duration
}

// AssertionOutcome records one assertion check.
type AssertionOutcome struct {
	Path     string
	Operator string
	Expected any
	Actual   any
	Passed   bool
	Message  string
}

// SuiteResult holds the outcome of the full test suite run.
type SuiteResult struct {
	Tests    []TestResult
	Duration time.Duration
	Passed   int
	Failed   int
	Skipped  int
	Total    int
}

// InterpolationContext carries all variable sources for interpolation.
type InterpolationContext struct {
	Variables map[string]any
	Captured  map[string]any
	BaseURL   string
	Env       map[string]string
}

// NewInterpolationContext creates a context with initialized maps.
func NewInterpolationContext() *InterpolationContext {
	return &InterpolationContext{
		Variables: make(map[string]any),
		Captured:  make(map[string]any),
		Env:       make(map[string]string),
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/tryve/...`
Expected: all 4 tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/tryve/
git commit -m "feat(tryve): add shared types and error types"
```

---

### Task 3: Config Loader

**Files:**
- Create: `internal/config/types.go`, `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Step 1: Write config loader tests**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/go-tryve/internal/config"
)

func TestLoad_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:3000"
`), 0644)

	cfg, err := config.Load(cfgFile, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Environment.BaseURL != "http://localhost:3000" {
		t.Errorf("baseUrl = %q, want %q", cfg.Environment.BaseURL, "http://localhost:3000")
	}
	// Defaults
	if cfg.Defaults.Timeout != 30000 {
		t.Errorf("timeout = %d, want 30000", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.Parallel != 1 {
		t.Errorf("parallel = %d, want 1", cfg.Defaults.Parallel)
	}
}

func TestLoad_EnvVarResolution(t *testing.T) {
	t.Setenv("TEST_BASE_URL", "http://test:8080")
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  local:
    baseUrl: "${TEST_BASE_URL}"
`), 0644)

	cfg, err := config.Load(cfgFile, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Environment.BaseURL != "http://test:8080" {
		t.Errorf("baseUrl = %q, want %q", cfg.Environment.BaseURL, "http://test:8080")
	}
}

func TestLoad_MissingEnvVarInBaseURL_Errors(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  local:
    baseUrl: "${NONEXISTENT_VAR}"
`), 0644)

	_, err := config.Load(cfgFile, "local")
	if err == nil {
		t.Fatal("expected error for missing env var in baseUrl")
	}
}

func TestLoad_MissingEnvironment_Errors(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  staging:
    baseUrl: "http://staging"
`), 0644)

	_, err := config.Load(cfgFile, "local")
	if err == nil {
		t.Fatal("expected error for missing environment")
	}
}

func TestLoad_WithAdapters(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters:
      postgresql:
        connectionString: "postgres://localhost/test"
        poolSize: 5
`), 0644)

	cfg, err := config.Load(cfgFile, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pg, ok := cfg.Environment.Adapters["postgresql"]
	if !ok {
		t.Fatal("postgresql adapter not found")
	}
	if pg["connectionString"] != "postgres://localhost/test" {
		t.Errorf("connectionString = %v", pg["connectionString"])
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "e2e.config.yaml")
	os.WriteFile(cfgFile, []byte(`
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:3000"
defaults:
  timeout: 60000
  retries: 3
  retryDelay: 2000
  parallel: 4
`), 0644)

	cfg, err := config.Load(cfgFile, "local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Defaults.Timeout != 60000 {
		t.Errorf("timeout = %d, want 60000", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.Retries != 3 {
		t.Errorf("retries = %d, want 3", cfg.Defaults.Retries)
	}
	if cfg.Defaults.Parallel != 4 {
		t.Errorf("parallel = %d, want 4", cfg.Defaults.Parallel)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/...`
Expected: compilation error

- [ ] **Step 3: Implement config types**

```go
// internal/config/types.go
package config

// RawConfig mirrors the YAML structure of e2e.config.yaml.
type RawConfig struct {
	Version      string                       `yaml:"version"`
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	Defaults     DefaultsConfig               `yaml:"defaults"`
	Variables    map[string]any               `yaml:"variables"`
	Hooks        HooksConfig                  `yaml:"hooks"`
	Reporters    []ReporterConfig             `yaml:"reporters"`
}

// EnvironmentConfig holds per-environment settings.
type EnvironmentConfig struct {
	BaseURL  string                    `yaml:"baseUrl"`
	Adapters map[string]map[string]any `yaml:"adapters"`
}

// DefaultsConfig holds default execution settings.
type DefaultsConfig struct {
	Timeout    int `yaml:"timeout"`
	Retries    int `yaml:"retries"`
	RetryDelay int `yaml:"retryDelay"`
	Parallel   int `yaml:"parallel"`
}

// HooksConfig holds lifecycle hook commands.
type HooksConfig struct {
	BeforeAll  string `yaml:"beforeAll"`
	AfterAll   string `yaml:"afterAll"`
	BeforeEach string `yaml:"beforeEach"`
	AfterEach  string `yaml:"afterEach"`
}

// ReporterConfig holds reporter settings.
type ReporterConfig struct {
	Type    string `yaml:"type"`
	Output  string `yaml:"output"`
	Verbose bool   `yaml:"verbose"`
}

// LoadedConfig is the resolved configuration ready for use.
type LoadedConfig struct {
	Raw             RawConfig
	Environment     EnvironmentConfig
	EnvironmentName string
	Defaults        DefaultsConfig
	Variables       map[string]any
	Hooks           HooksConfig
	Reporters       []ReporterConfig
}
```

- [ ] **Step 4: Implement config loader**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/liemle3893/go-tryve/internal/tryve"
	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// Load reads and resolves the configuration file for the given environment.
func Load(path, envName string) (*LoadedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, tryve.ConfigError(
			fmt.Sprintf("cannot read config file: %s", path),
			"ensure e2e.config.yaml exists; run 'tryve init' to create one",
			err,
		)
	}

	var raw RawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, tryve.ConfigError("invalid YAML in config file", "check YAML syntax", err)
	}

	if raw.Version != "1.0" {
		return nil, tryve.ConfigError(
			fmt.Sprintf("unsupported config version %q", raw.Version),
			"version must be \"1.0\"",
			nil,
		)
	}

	env, ok := raw.Environments[envName]
	if !ok {
		return nil, tryve.ConfigError(
			fmt.Sprintf("environment %q not found in config", envName),
			fmt.Sprintf("available environments: %v", envKeys(raw.Environments)),
			nil,
		)
	}

	// Resolve env vars in baseUrl (strict — error if missing)
	resolved, err := resolveEnvVars(env.BaseURL, true)
	if err != nil {
		return nil, tryve.ConfigError("cannot resolve baseUrl", "set the environment variable", err)
	}
	env.BaseURL = resolved

	// Resolve env vars in adapter configs (non-strict — leave unresolved)
	for name, adapterCfg := range env.Adapters {
		env.Adapters[name] = resolveMapEnvVars(adapterCfg)
	}

	// Resolve env vars in variables (non-strict)
	vars := raw.Variables
	if vars == nil {
		vars = make(map[string]any)
	}
	for k, v := range vars {
		if s, ok := v.(string); ok {
			vars[k], _ = resolveEnvVars(s, false)
		}
	}

	// Apply defaults
	defaults := applyDefaults(raw.Defaults)

	// Default reporters
	reporters := raw.Reporters
	if len(reporters) == 0 {
		reporters = []ReporterConfig{{Type: "console"}}
	}

	return &LoadedConfig{
		Raw:             raw,
		Environment:     env,
		EnvironmentName: envName,
		Defaults:        defaults,
		Variables:       vars,
		Hooks:           raw.Hooks,
		Reporters:       reporters,
	}, nil
}

// resolveEnvVars replaces ${VAR} patterns with environment variable values.
// If strict is true, missing env vars cause an error.
func resolveEnvVars(s string, strict bool) (string, error) {
	var resolveErr error
	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		val, ok := os.LookupEnv(varName)
		if !ok {
			if strict {
				resolveErr = fmt.Errorf("environment variable %q not set", varName)
			}
			return match // leave unresolved
		}
		return val
	})
	return result, resolveErr
}

// resolveMapEnvVars resolves env vars in string values of a map (non-strict).
func resolveMapEnvVars(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k], _ = resolveEnvVars(s, false)
		} else {
			result[k] = v
		}
	}
	return result
}

func applyDefaults(d DefaultsConfig) DefaultsConfig {
	if d.Timeout <= 0 {
		d.Timeout = 30000
	}
	if d.RetryDelay <= 0 {
		d.RetryDelay = 1000
	}
	if d.Parallel <= 0 {
		d.Parallel = 1
	}
	// d.Retries defaults to 0, which is the zero value
	return d
}

func envKeys(m map[string]EnvironmentConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/config/...`
Expected: all 6 tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add config loader with env var resolution"
```

---

### Task 4: Built-in Functions

**Files:**
- Create: `internal/interpolate/builtins.go`, `internal/interpolate/builtins_test.go`

- [ ] **Step 1: Write built-in function tests**

```go
// internal/interpolate/builtins_test.go
package interpolate_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/liemle3893/go-tryve/internal/interpolate"
)

func TestBuiltin_UUID(t *testing.T) {
	result, err := interpolate.CallBuiltin("uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(result) {
		t.Errorf("invalid UUID: %q", result)
	}
}

func TestBuiltin_Timestamp(t *testing.T) {
	result, err := interpolate.CallBuiltin("timestamp")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		t.Fatalf("not a number: %q", result)
	}
	if ts < 1700000000000 {
		t.Errorf("timestamp too small: %d", ts)
	}
}

func TestBuiltin_Random(t *testing.T) {
	result, err := interpolate.CallBuiltin("random", "1", "10")
	if err != nil {
		t.Fatal(err)
	}
	n, _ := strconv.Atoi(result)
	if n < 1 || n > 10 {
		t.Errorf("random(%d) out of range [1,10]", n)
	}
}

func TestBuiltin_RandomString(t *testing.T) {
	result, err := interpolate.CallBuiltin("randomString", "16")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 16 {
		t.Errorf("length = %d, want 16", len(result))
	}
}

func TestBuiltin_Base64(t *testing.T) {
	encoded, _ := interpolate.CallBuiltin("base64", "hello")
	if encoded != base64.StdEncoding.EncodeToString([]byte("hello")) {
		t.Errorf("base64 = %q", encoded)
	}
	decoded, _ := interpolate.CallBuiltin("base64Decode", encoded)
	if decoded != "hello" {
		t.Errorf("base64Decode = %q", decoded)
	}
}

func TestBuiltin_MD5(t *testing.T) {
	result, _ := interpolate.CallBuiltin("md5", "hello")
	if result != "5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("md5 = %q", result)
	}
}

func TestBuiltin_SHA256(t *testing.T) {
	result, _ := interpolate.CallBuiltin("sha256", "hello")
	if result != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("sha256 = %q", result)
	}
}

func TestBuiltin_Env(t *testing.T) {
	t.Setenv("TEST_BUILTIN_VAR", "test_value")
	result, _ := interpolate.CallBuiltin("env", "TEST_BUILTIN_VAR")
	if result != "test_value" {
		t.Errorf("env = %q", result)
	}
}

func TestBuiltin_Env_Missing(t *testing.T) {
	_, err := interpolate.CallBuiltin("env", "NONEXISTENT_BUILTIN_VAR_XYZ")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestBuiltin_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("file content"), 0644)
	result, _ := interpolate.CallBuiltin("file", f)
	if result != "file content" {
		t.Errorf("file = %q", result)
	}
}

func TestBuiltin_Lower(t *testing.T) {
	result, _ := interpolate.CallBuiltin("lower", "HELLO")
	if result != "hello" {
		t.Errorf("lower = %q", result)
	}
}

func TestBuiltin_Upper(t *testing.T) {
	result, _ := interpolate.CallBuiltin("upper", "hello")
	if result != "HELLO" {
		t.Errorf("upper = %q", result)
	}
}

func TestBuiltin_Trim(t *testing.T) {
	result, _ := interpolate.CallBuiltin("trim", "  hello  ")
	if result != "hello" {
		t.Errorf("trim = %q", result)
	}
}

func TestBuiltin_ISODate(t *testing.T) {
	result, _ := interpolate.CallBuiltin("isoDate")
	if !strings.Contains(result, "T") || !strings.Contains(result, "Z") {
		t.Errorf("isoDate = %q, expected ISO 8601 format", result)
	}
}

func TestBuiltin_Unknown(t *testing.T) {
	_, err := interpolate.CallBuiltin("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown builtin")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/interpolate/...`
Expected: compilation error

- [ ] **Step 3: Implement built-in functions**

```go
// internal/interpolate/builtins.go
package interpolate

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

// BuiltinFunc takes variadic string args and returns a string result.
type BuiltinFunc func(args ...string) (string, error)

var builtins = map[string]BuiltinFunc{
	"uuid":         builtinUUID,
	"timestamp":    builtinTimestamp,
	"isoDate":      builtinISODate,
	"random":       builtinRandom,
	"randomString": builtinRandomString,
	"env":          builtinEnv,
	"file":         builtinFile,
	"base64":       builtinBase64,
	"base64Decode": builtinBase64Decode,
	"md5":          builtinMD5,
	"sha256":       builtinSHA256,
	"now":          builtinNow,
	"dateAdd":      builtinDateAdd,
	"dateSub":      builtinDateSub,
	"totp":         builtinTOTP,
	"jsonStringify": builtinJSONStringify,
	"lower":        builtinLower,
	"upper":        builtinUpper,
	"trim":         builtinTrim,
}

// CallBuiltin calls a named built-in function with the given arguments.
func CallBuiltin(name string, args ...string) (string, error) {
	fn, ok := builtins[name]
	if !ok {
		return "", fmt.Errorf("unknown built-in function: $%s", name)
	}
	return fn(args...)
}

func builtinUUID(args ...string) (string, error) {
	return uuid.New().String(), nil
}

func builtinTimestamp(args ...string) (string, error) {
	return strconv.FormatInt(time.Now().UnixMilli(), 10), nil
}

func builtinISODate(args ...string) (string, error) {
	return time.Now().UTC().Format(time.RFC3339), nil
}

func builtinRandom(args ...string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("$random requires min and max arguments")
	}
	min, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("$random: invalid min %q", args[0])
	}
	max, err := strconv.Atoi(args[1])
	if err != nil {
		return "", fmt.Errorf("$random: invalid max %q", args[1])
	}
	return strconv.Itoa(min + rand.Intn(max-min+1)), nil
}

func builtinRandomString(args ...string) (string, error) {
	length := 8
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return "", fmt.Errorf("$randomString: invalid length %q", args[0])
		}
		length = n
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b), nil
}

func builtinEnv(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$env requires variable name")
	}
	val, ok := os.LookupEnv(args[0])
	if !ok {
		return "", fmt.Errorf("environment variable %q not set", args[0])
	}
	return val, nil
}

func builtinFile(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$file requires path argument")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return "", fmt.Errorf("$file: %w", err)
	}
	return string(data), nil
}

func builtinBase64(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$base64 requires value argument")
	}
	return base64.StdEncoding.EncodeToString([]byte(args[0])), nil
}

func builtinBase64Decode(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$base64Decode requires value argument")
	}
	data, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		return "", fmt.Errorf("$base64Decode: %w", err)
	}
	return string(data), nil
}

func builtinMD5(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$md5 requires value argument")
	}
	hash := md5.Sum([]byte(args[0]))
	return fmt.Sprintf("%x", hash), nil
}

func builtinSHA256(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$sha256 requires value argument")
	}
	hash := sha256.Sum256([]byte(args[0]))
	return fmt.Sprintf("%x", hash), nil
}

func builtinNow(args ...string) (string, error) {
	if len(args) > 0 {
		return formatTime(time.Now(), args[0]), nil
	}
	return time.Now().UTC().Format(time.RFC3339), nil
}

func builtinDateAdd(args ...string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("$dateAdd requires amount and unit")
	}
	return dateShift(args[0], args[1], 1)
}

func builtinDateSub(args ...string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("$dateSub requires amount and unit")
	}
	return dateShift(args[0], args[1], -1)
}

func builtinTOTP(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$totp requires secret argument")
	}
	code, err := totp.GenerateCode(args[0], time.Now())
	if err != nil {
		return "", fmt.Errorf("$totp: %w", err)
	}
	return code, nil
}

func builtinJSONStringify(args ...string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("$jsonStringify requires value argument")
	}
	// Escape for safe JSON embedding
	s := args[0]
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s, nil
}

func builtinLower(args ...string) (string, error) {
	if len(args) < 1 {
		return "", nil
	}
	return strings.ToLower(args[0]), nil
}

func builtinUpper(args ...string) (string, error) {
	if len(args) < 1 {
		return "", nil
	}
	return strings.ToUpper(args[0]), nil
}

func builtinTrim(args ...string) (string, error) {
	if len(args) < 1 {
		return "", nil
	}
	return strings.TrimSpace(args[0]), nil
}

// dateShift adds or subtracts time based on amount, unit, and direction (+1 or -1).
func dateShift(amountStr, unit string, direction int) (string, error) {
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return "", fmt.Errorf("invalid amount %q", amountStr)
	}
	amount *= direction
	now := time.Now().UTC()

	switch strings.ToLower(unit) {
	case "s", "second", "seconds":
		return now.Add(time.Duration(amount) * time.Second).Format(time.RFC3339), nil
	case "m", "minute", "minutes":
		return now.Add(time.Duration(amount) * time.Minute).Format(time.RFC3339), nil
	case "h", "hour", "hours":
		return now.Add(time.Duration(amount) * time.Hour).Format(time.RFC3339), nil
	case "d", "day", "days":
		return now.AddDate(0, 0, amount).Format(time.RFC3339), nil
	case "w", "week", "weeks":
		return now.AddDate(0, 0, amount*7).Format(time.RFC3339), nil
	case "month", "months":
		return now.AddDate(0, amount, 0).Format(time.RFC3339), nil
	case "y", "year", "years":
		return now.AddDate(amount, 0, 0).Format(time.RFC3339), nil
	default:
		return "", fmt.Errorf("unknown time unit %q", unit)
	}
}

// formatTime applies a Go-style or common format string to a time.
func formatTime(t time.Time, format string) string {
	switch format {
	case "iso", "ISO":
		return t.UTC().Format(time.RFC3339)
	case "date":
		return t.UTC().Format("2006-01-02")
	case "time":
		return t.UTC().Format("15:04:05")
	case "datetime":
		return t.UTC().Format("2006-01-02 15:04:05")
	case "unix":
		return strconv.FormatInt(t.Unix(), 10)
	case "unixMs":
		return strconv.FormatInt(t.UnixMilli(), 10)
	default:
		return t.UTC().Format(format)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/interpolate/...`
Expected: all 15 tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/interpolate/builtins.go internal/interpolate/builtins_test.go
git commit -m "feat(interpolate): add 19 built-in functions"
```

---

### Task 5: Variable Interpolation

**Files:**
- Create: `internal/interpolate/interpolate.go`, `internal/interpolate/interpolate_test.go`

- [ ] **Step 1: Write interpolation tests**

```go
// internal/interpolate/interpolate_test.go
package interpolate_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/interpolate"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestResolve_SimpleVariable(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.Variables["name"] = "alice"
	result, err := interpolate.ResolveString("hello {{name}}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello alice" {
		t.Errorf("got %q, want %q", result, "hello alice")
	}
}

func TestResolve_DollarBraceSyntax(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.Variables["name"] = "bob"
	result, _ := interpolate.ResolveString("hello ${name}", ctx)
	if result != "hello bob" {
		t.Errorf("got %q, want %q", result, "hello bob")
	}
}

func TestResolve_CapturedValue(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.Captured["userId"] = "123"
	result, _ := interpolate.ResolveString("user: {{captured.userId}}", ctx)
	if result != "user: 123" {
		t.Errorf("got %q, want %q", result, "user: 123")
	}
}

func TestResolve_BuiltinFunction(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	result, _ := interpolate.ResolveString("{{$upper(hello)}}", ctx)
	if result != "HELLO" {
		t.Errorf("got %q, want %q", result, "HELLO")
	}
}

func TestResolve_BaseURL(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.BaseURL = "http://localhost:3000"
	result, _ := interpolate.ResolveString("{{baseUrl}}/api", ctx)
	if result != "http://localhost:3000/api" {
		t.Errorf("got %q, want %q", result, "http://localhost:3000/api")
	}
}

func TestResolve_NestedVariable(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.Variables["greeting"] = "hello {{name}}"
	ctx.Variables["name"] = "world"
	result, _ := interpolate.ResolveString("{{greeting}}", ctx)
	if result != "hello world" {
		t.Errorf("got %q, want %q", result, "hello world")
	}
}

func TestResolve_EnvVariable(t *testing.T) {
	t.Setenv("TEST_INTERP_VAR", "from_env")
	ctx := tryve.NewInterpolationContext()
	result, _ := interpolate.ResolveString("{{$env(TEST_INTERP_VAR)}}", ctx)
	if result != "from_env" {
		t.Errorf("got %q, want %q", result, "from_env")
	}
}

func TestResolve_MapValues(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	ctx.Variables["token"] = "abc123"
	input := map[string]any{
		"url":     "/api/users",
		"headers": map[string]any{"Authorization": "Bearer {{token}}"},
	}
	result, err := interpolate.ResolveMap(input, ctx)
	if err != nil {
		t.Fatal(err)
	}
	headers := result["headers"].(map[string]any)
	if headers["Authorization"] != "Bearer abc123" {
		t.Errorf("got %v", headers["Authorization"])
	}
}

func TestResolve_UnknownVariable_LeftAsIs(t *testing.T) {
	ctx := tryve.NewInterpolationContext()
	result, _ := interpolate.ResolveString("{{unknown}}", ctx)
	if result != "{{unknown}}" {
		t.Errorf("got %q, want unresolved", result)
	}
}

func TestResolveVariables_TopologicalOrder(t *testing.T) {
	vars := map[string]any{
		"greeting": "hello {{name}}",
		"name":     "world",
		"message":  "{{greeting}}!",
	}
	ctx := tryve.NewInterpolationContext()
	resolved, err := interpolate.ResolveVariables(vars, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resolved["message"] != "hello world!" {
		t.Errorf("message = %q, want %q", resolved["message"], "hello world!")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/interpolate/...`
Expected: compilation error (ResolveString not defined)

- [ ] **Step 3: Implement interpolation engine**

```go
// internal/interpolate/interpolate.go
package interpolate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

const maxDepth = 10

var (
	// Matches {{expression}} and ${expression}
	doubleBraceRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)
	dollarBraceRe = regexp.MustCompile(`\$\{([^}]+)\}`)
	// Matches $funcName(args) or $funcName
	builtinCallRe = regexp.MustCompile(`^\$(\w+)(?:\(([^)]*)\))?$`)
)

// ResolveString interpolates a single string against the given context.
func ResolveString(s string, ctx *tryve.InterpolationContext) (string, error) {
	prev := ""
	result := s
	for i := 0; i < maxDepth; i++ {
		resolved := singlePass(result, ctx)
		if resolved == prev {
			// No further changes — either fully resolved or has unresolvable refs
			return resolved, nil
		}
		prev = result
		result = resolved
	}
	return result, nil
}

// singlePass replaces all interpolation patterns once.
func singlePass(s string, ctx *tryve.InterpolationContext) string {
	// Handle {{...}} syntax
	s = doubleBraceRe.ReplaceAllStringFunc(s, func(match string) string {
		expr := doubleBraceRe.FindStringSubmatch(match)[1]
		expr = strings.TrimSpace(expr)
		val, found := resolveExpression(expr, ctx)
		if found {
			return fmt.Sprintf("%v", val)
		}
		return match // leave unresolved
	})
	// Handle ${...} syntax
	s = dollarBraceRe.ReplaceAllStringFunc(s, func(match string) string {
		expr := dollarBraceRe.FindStringSubmatch(match)[1]
		expr = strings.TrimSpace(expr)
		val, found := resolveExpression(expr, ctx)
		if found {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
	return s
}

// resolveExpression resolves a single expression (the content inside {{ }} or ${ }).
func resolveExpression(expr string, ctx *tryve.InterpolationContext) (any, bool) {
	// 1. Built-in functions: $funcName or $funcName(args)
	if strings.HasPrefix(expr, "$") {
		m := builtinCallRe.FindStringSubmatch(expr)
		if m != nil {
			funcName := m[1]
			var args []string
			if m[2] != "" {
				for _, a := range strings.Split(m[2], ",") {
					a = strings.TrimSpace(a)
					a = strings.Trim(a, `"'`)
					args = append(args, a)
				}
			}
			result, err := CallBuiltin(funcName, args...)
			if err != nil {
				return nil, false
			}
			return result, true
		}
	}

	// 2. baseUrl
	if expr == "baseUrl" {
		return ctx.BaseURL, true
	}

	// 3. Captured values: captured.fieldName
	if strings.HasPrefix(expr, "captured.") {
		path := strings.TrimPrefix(expr, "captured.")
		val := getNestedValue(ctx.Captured, path)
		if val != nil {
			return val, true
		}
		return nil, false
	}

	// 4. Variables (with nested path support)
	val := getNestedValue(ctx.Variables, expr)
	if val != nil {
		return val, true
	}

	// 5. Environment variables
	if v, ok := ctx.Env[expr]; ok {
		return v, true
	}

	return nil, false
}

// ResolveMap interpolates all string values in a map recursively.
func ResolveMap(m map[string]any, ctx *tryve.InterpolationContext) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for k, v := range m {
		resolved, err := resolveValue(v, ctx)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}
	return result, nil
}

// ResolveSlice interpolates all string values in a slice recursively.
func ResolveSlice(s []any, ctx *tryve.InterpolationContext) ([]any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		resolved, err := resolveValue(v, ctx)
		if err != nil {
			return nil, err
		}
		result[i] = resolved
	}
	return result, nil
}

func resolveValue(v any, ctx *tryve.InterpolationContext) (any, error) {
	switch val := v.(type) {
	case string:
		return ResolveString(val, ctx)
	case map[string]any:
		return ResolveMap(val, ctx)
	case []any:
		return ResolveSlice(val, ctx)
	default:
		return v, nil
	}
}

// ResolveVariables resolves a map of variables in topological order,
// so that variables can reference each other without ordering issues.
func ResolveVariables(vars map[string]any, ctx *tryve.InterpolationContext) (map[string]any, error) {
	// Build dependency graph
	deps := make(map[string][]string) // var -> vars it depends on
	for name, val := range vars {
		if s, ok := val.(string); ok {
			deps[name] = extractDependencies(s, vars)
		}
	}

	// Topological sort (Kahn's algorithm)
	order, err := topoSort(deps, vars)
	if err != nil {
		return nil, err
	}

	// Resolve in order
	resolved := make(map[string]any, len(vars))
	for k, v := range vars {
		resolved[k] = v
	}
	resolveCtx := &tryve.InterpolationContext{
		Variables: resolved,
		Captured:  ctx.Captured,
		BaseURL:   ctx.BaseURL,
		Env:       ctx.Env,
	}

	for _, name := range order {
		if s, ok := resolved[name].(string); ok {
			val, err := ResolveString(s, resolveCtx)
			if err != nil {
				return nil, err
			}
			resolved[name] = val
			resolveCtx.Variables[name] = val
		}
	}
	return resolved, nil
}

// extractDependencies finds variable names referenced in a string that exist in vars.
func extractDependencies(s string, vars map[string]any) []string {
	var deps []string
	seen := make(map[string]bool)
	for _, re := range []*regexp.Regexp{doubleBraceRe, dollarBraceRe} {
		matches := re.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			expr := strings.TrimSpace(m[1])
			// Skip builtins, baseUrl, captured values
			if strings.HasPrefix(expr, "$") || expr == "baseUrl" || strings.HasPrefix(expr, "captured.") {
				continue
			}
			if _, exists := vars[expr]; exists && !seen[expr] {
				deps = append(deps, expr)
				seen[expr] = true
			}
		}
	}
	return deps
}

// topoSort performs Kahn's algorithm for topological ordering.
func topoSort(deps map[string][]string, vars map[string]any) ([]string, error) {
	inDegree := make(map[string]int)
	for name := range vars {
		inDegree[name] = 0
	}
	for name, dd := range deps {
		_ = name
		for _, dep := range dd {
			inDegree[dep] = inDegree[dep] // ensure it exists
		}
		inDegree[name] = len(dd)
	}

	// Recalculate in-degree properly
	for name := range vars {
		inDegree[name] = 0
	}
	for _, dd := range deps {
		for _, dep := range dd {
			_ = dep
		}
	}
	// Count: for each var, how many other vars depend on it being resolved first?
	// Actually: inDegree[X] = number of vars X depends on
	for name := range vars {
		if dd, ok := deps[name]; ok {
			inDegree[name] = len(dd)
		} else {
			inDegree[name] = 0
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		order = append(order, name)

		// For all vars that depend on `name`, decrement their in-degree
		for other, dd := range deps {
			for _, dep := range dd {
				if dep == name {
					inDegree[other]--
					if inDegree[other] == 0 {
						queue = append(queue, other)
					}
				}
			}
		}
	}

	if len(order) != len(vars) {
		return nil, fmt.Errorf("circular dependency detected in variables")
	}
	return order, nil
}

// getNestedValue traverses a map using dot notation (e.g., "user.name").
func getNestedValue(m map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil
			}
			current = val
		default:
			return nil
		}
	}
	return current
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/interpolate/...`
Expected: all 25+ tests pass (builtins + interpolation)

- [ ] **Step 5: Commit**

```bash
git add internal/interpolate/interpolate.go internal/interpolate/interpolate_test.go
git commit -m "feat(interpolate): add variable interpolation engine with topo sort"
```

---

### Task 6: JSONPath Engine

**Files:**
- Create: `internal/assertion/jsonpath.go`, `internal/assertion/jsonpath_test.go`

- [ ] **Step 1: Write JSONPath tests**

```go
// internal/assertion/jsonpath_test.go
package assertion_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/assertion"
)

func TestJSONPath_SimpleProperty(t *testing.T) {
	data := map[string]any{"status": 200, "name": "alice"}
	val, found := assertion.EvalJSONPath(data, "$.status")
	if !found || val != 200 {
		t.Errorf("got %v (found=%v), want 200", val, found)
	}
}

func TestJSONPath_NestedProperty(t *testing.T) {
	data := map[string]any{"body": map[string]any{"user": map[string]any{"id": 42}}}
	val, found := assertion.EvalJSONPath(data, "$.body.user.id")
	if !found || val != 42 {
		t.Errorf("got %v, want 42", val)
	}
}

func TestJSONPath_ArrayIndex(t *testing.T) {
	data := map[string]any{"items": []any{"a", "b", "c"}}
	val, found := assertion.EvalJSONPath(data, "$.items[0]")
	if !found || val != "a" {
		t.Errorf("got %v, want 'a'", val)
	}
}

func TestJSONPath_ArrayWildcard(t *testing.T) {
	data := map[string]any{"items": []any{
		map[string]any{"name": "a"},
		map[string]any{"name": "b"},
	}}
	val, found := assertion.EvalJSONPath(data, "$.items[*].name")
	if !found {
		t.Fatal("not found")
	}
	arr, ok := val.([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("got %v, want [a, b]", val)
	}
}

func TestJSONPath_RecursiveDescent(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{"id": 1},
		"b": map[string]any{"nested": map[string]any{"id": 2}},
	}
	val, found := assertion.EvalJSONPath(data, "$..id")
	if !found {
		t.Fatal("not found")
	}
	arr, ok := val.([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("got %v, want [1, 2]", val)
	}
}

func TestJSONPath_NotFound(t *testing.T) {
	data := map[string]any{"a": 1}
	_, found := assertion.EvalJSONPath(data, "$.b")
	if found {
		t.Error("expected not found")
	}
}

func TestJSONPath_WithoutDollarPrefix(t *testing.T) {
	data := map[string]any{"status": 200}
	val, found := assertion.EvalJSONPath(data, "status")
	if !found || val != 200 {
		t.Errorf("got %v, want 200", val)
	}
}

func TestJSONPath_BracketNotation(t *testing.T) {
	data := map[string]any{"a-b": "value"}
	val, found := assertion.EvalJSONPath(data, "$['a-b']")
	if !found || val != "value" {
		t.Errorf("got %v, want 'value'", val)
	}
}

func TestHasJSONPath(t *testing.T) {
	data := map[string]any{"a": map[string]any{"b": nil}}
	if !assertion.HasJSONPath(data, "$.a.b") {
		t.Error("expected path to exist (even if nil)")
	}
	if assertion.HasJSONPath(data, "$.a.c") {
		t.Error("expected path to not exist")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/assertion/...`
Expected: compilation error

- [ ] **Step 3: Implement JSONPath using ojg**

```go
// internal/assertion/jsonpath.go
package assertion

import (
	"strings"

	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
)

// EvalJSONPath evaluates a JSONPath expression against data.
// Returns the value and whether it was found.
func EvalJSONPath(data any, path string) (any, bool) {
	path = normalizePath(path)

	expr, err := jp.ParseString(path)
	if err != nil {
		// Fall back to simple dot-notation for non-standard paths
		return evalSimplePath(data, path)
	}

	// Convert to ojg-compatible format (map[string]any is fine)
	results := expr.Get(data)
	if len(results) == 0 {
		return nil, false
	}
	if len(results) == 1 {
		return results[0], true
	}
	// Multiple results: return as slice
	return []any(results), true
}

// HasJSONPath checks whether a path exists in the data (value can be nil).
func HasJSONPath(data any, path string) bool {
	path = normalizePath(path)

	expr, err := jp.ParseString(path)
	if err != nil {
		_, found := evalSimplePath(data, path)
		return found
	}

	results := expr.Get(data)
	return len(results) > 0
}

// QueryJSONPath returns all matches for a path expression.
func QueryJSONPath(data any, path string) []any {
	path = normalizePath(path)

	expr, err := jp.ParseString(path)
	if err != nil {
		val, found := evalSimplePath(data, path)
		if found {
			return []any{val}
		}
		return nil
	}
	return expr.Get(data)
}

// normalizePath ensures the path starts with $.
func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" {
		return "$"
	}
	if !strings.HasPrefix(path, "$") {
		return "$." + path
	}
	return path
}

// evalSimplePath handles simple dot notation without JSONPath features.
func evalSimplePath(data any, path string) (any, bool) {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return data, true
	}

	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

// DataToGeneric converts data to a format compatible with ojg if needed.
// For most cases, map[string]any is already compatible.
func DataToGeneric(data any) any {
	switch v := data.(type) {
	case string:
		parsed, err := oj.ParseString(v)
		if err != nil {
			return v
		}
		return parsed
	default:
		return v
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/assertion/...`
Expected: all 9 JSONPath tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/assertion/jsonpath.go internal/assertion/jsonpath_test.go
git commit -m "feat(assertion): add JSONPath evaluation engine using ojg"
```

---

### Task 7: Assertion Engine

**Files:**
- Create: `internal/assertion/matchers.go`, `internal/assertion/matchers_test.go`, `internal/assertion/assertion.go`, `internal/assertion/assertion_test.go`

- [ ] **Step 1: Write matcher tests**

```go
// internal/assertion/matchers_test.go
package assertion_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/assertion"
)

func TestMatch_Equals(t *testing.T) {
	tests := []struct{ actual, expected any; pass bool }{
		{200, 200, true},
		{200, 201, false},
		{"hello", "hello", true},
		{true, true, true},
		{nil, nil, true},
	}
	for _, tt := range tests {
		r := assertion.Match("equals", tt.actual, tt.expected)
		if r.Pass != tt.pass {
			t.Errorf("equals(%v, %v) = %v, want %v", tt.actual, tt.expected, r.Pass, tt.pass)
		}
	}
}

func TestMatch_Contains(t *testing.T) {
	if r := assertion.Match("contains", "hello world", "world"); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("contains", "hello", "xyz"); r.Pass {
		t.Error("expected fail")
	}
	// Array contains
	if r := assertion.Match("contains", []any{1, 2, 3}, 2); !r.Pass {
		t.Error("expected pass for array contains")
	}
}

func TestMatch_Matches(t *testing.T) {
	if r := assertion.Match("matches", "user-123", `user-\d+`); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("matches", "user-abc", `user-\d+`); r.Pass {
		t.Error("expected fail")
	}
}

func TestMatch_Type(t *testing.T) {
	tests := []struct{ actual any; expected string; pass bool }{
		{"hello", "string", true},
		{42, "number", true},
		{42.5, "number", true},
		{true, "boolean", true},
		{nil, "null", true},
		{map[string]any{}, "object", true},
		{[]any{}, "array", true},
	}
	for _, tt := range tests {
		r := assertion.Match("type", tt.actual, tt.expected)
		if r.Pass != tt.pass {
			t.Errorf("type(%v, %q) = %v, want %v", tt.actual, tt.expected, r.Pass, tt.pass)
		}
	}
}

func TestMatch_GreaterThan(t *testing.T) {
	if r := assertion.Match("greaterThan", 10, 5); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("greaterThan", 5, 10); r.Pass {
		t.Error("expected fail")
	}
}

func TestMatch_Length(t *testing.T) {
	if r := assertion.Match("length", []any{1, 2, 3}, 3); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("length", "hello", 5); !r.Pass {
		t.Error("expected pass for string length")
	}
}

func TestMatch_Exists(t *testing.T) {
	if r := assertion.Match("exists", "anything", true); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("exists", nil, true); r.Pass {
		t.Error("expected fail for nil")
	}
}

func TestMatch_IsNull(t *testing.T) {
	if r := assertion.Match("isNull", nil, true); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("isNull", "value", true); r.Pass {
		t.Error("expected fail")
	}
}

func TestMatch_IsEmpty(t *testing.T) {
	if r := assertion.Match("isEmpty", []any{}, true); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("isEmpty", "", true); !r.Pass {
		t.Error("expected pass for empty string")
	}
}

func TestMatch_HasProperty(t *testing.T) {
	obj := map[string]any{"name": "alice", "age": 30}
	if r := assertion.Match("hasProperty", obj, "name"); !r.Pass {
		t.Error("expected pass")
	}
	if r := assertion.Match("hasProperty", obj, "email"); r.Pass {
		t.Error("expected fail")
	}
}
```

- [ ] **Step 2: Write assertion runner tests**

```go
// internal/assertion/assertion_test.go
package assertion_test

import (
	"testing"

	"github.com/liemle3893/go-tryve/internal/assertion"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestRunAssertions_HTTPStatus(t *testing.T) {
	data := map[string]any{"status": 200, "body": map[string]any{"id": 1}}
	assertDef := map[string]any{"status": 200}

	results, err := assertion.RunAssertions(data, assertDef)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("assertion failed: %s", r.Message)
		}
	}
}

func TestRunAssertions_HTTPStatusArray(t *testing.T) {
	data := map[string]any{"status": 201}
	assertDef := map[string]any{"status": []any{200, 201, 202}}

	results, _ := assertion.RunAssertions(data, assertDef)
	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
		}
	}
	if !allPassed {
		t.Error("expected 201 to be in [200, 201, 202]")
	}
}

func TestRunAssertions_JSONPathAssertions(t *testing.T) {
	data := map[string]any{
		"status": 200,
		"body":   map[string]any{"user": map[string]any{"name": "alice", "age": 30}},
	}
	assertDef := map[string]any{
		"json": []any{
			map[string]any{"path": "$.body.user.name", "equals": "alice"},
			map[string]any{"path": "$.body.user.age", "greaterThan": 18},
		},
	}

	results, _ := assertion.RunAssertions(data, assertDef)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("assertion failed: %s %s", r.Path, r.Message)
		}
	}
}

func TestRunAssertions_Duration(t *testing.T) {
	data := map[string]any{"duration": 150}
	assertDef := map[string]any{
		"duration": map[string]any{"lessThan": 200, "greaterThan": 100},
	}

	results, _ := assertion.RunAssertions(data, assertDef)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("assertion failed: %s", r.Message)
		}
	}
}

func TestRunAssertions_SliceFormat(t *testing.T) {
	data := map[string]any{"rows": []any{map[string]any{"id": 1}}}
	assertDef := []any{
		map[string]any{"path": "$.rows[0].id", "equals": 1},
	}

	results, _ := assertion.RunAssertions(data, assertDef)
	for _, r := range results {
		if !r.Passed {
			t.Errorf("assertion failed: %s", r.Message)
		}
	}
}

func TestRunAssertions_FailingAssertion(t *testing.T) {
	data := map[string]any{"status": 404}
	assertDef := map[string]any{"status": 200}

	results, _ := assertion.RunAssertions(data, assertDef)
	hasFail := false
	for _, r := range results {
		if !r.Passed {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected at least one failing assertion")
	}
}
```

- [ ] **Step 3: Implement matchers**

```go
// internal/assertion/matchers.go
package assertion

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// MatchResult holds the outcome of a single matcher check.
type MatchResult struct {
	Pass    bool
	Message string
}

// Match dispatches to the appropriate matcher by operator name.
func Match(operator string, actual, expected any) MatchResult {
	switch operator {
	case "equals":
		return matchEquals(actual, expected)
	case "notEquals":
		return matchNotEquals(actual, expected)
	case "contains":
		return matchContains(actual, expected)
	case "notContains":
		return matchNotContains(actual, expected)
	case "matches":
		return matchMatches(actual, expected)
	case "type":
		return matchType(actual, expected)
	case "exists":
		return matchExists(actual, expected)
	case "notExists":
		return matchNotExists(actual, expected)
	case "isNull":
		return matchIsNull(actual, expected)
	case "isNotNull":
		return matchIsNotNull(actual, expected)
	case "greaterThan":
		return matchGreaterThan(actual, expected)
	case "lessThan":
		return matchLessThan(actual, expected)
	case "greaterThanOrEqual":
		return matchGreaterThanOrEqual(actual, expected)
	case "lessThanOrEqual":
		return matchLessThanOrEqual(actual, expected)
	case "length":
		return matchLength(actual, expected)
	case "isEmpty":
		return matchIsEmpty(actual, expected)
	case "notEmpty":
		return matchNotEmpty(actual, expected)
	case "hasProperty":
		return matchHasProperty(actual, expected)
	case "notHasProperty":
		return matchNotHasProperty(actual, expected)
	default:
		return MatchResult{Pass: false, Message: fmt.Sprintf("unknown operator: %s", operator)}
	}
}

func matchEquals(actual, expected any) MatchResult {
	pass := reflect.DeepEqual(normalizeNumeric(actual), normalizeNumeric(expected))
	return MatchResult{
		Pass:    pass,
		Message: fmt.Sprintf("expected %v to equal %v", actual, expected),
	}
}

func matchNotEquals(actual, expected any) MatchResult {
	r := matchEquals(actual, expected)
	return MatchResult{Pass: !r.Pass, Message: fmt.Sprintf("expected %v to not equal %v", actual, expected)}
}

func matchContains(actual, expected any) MatchResult {
	switch v := actual.(type) {
	case string:
		pass := strings.Contains(v, fmt.Sprintf("%v", expected))
		return MatchResult{Pass: pass, Message: fmt.Sprintf("expected %q to contain %q", v, expected)}
	case []any:
		for _, item := range v {
			if reflect.DeepEqual(normalizeNumeric(item), normalizeNumeric(expected)) {
				return MatchResult{Pass: true}
			}
		}
		return MatchResult{Pass: false, Message: fmt.Sprintf("expected array to contain %v", expected)}
	default:
		return MatchResult{Pass: false, Message: fmt.Sprintf("contains not supported for type %T", actual)}
	}
}

func matchNotContains(actual, expected any) MatchResult {
	r := matchContains(actual, expected)
	return MatchResult{Pass: !r.Pass, Message: fmt.Sprintf("expected %v to not contain %v", actual, expected)}
}

func matchMatches(actual, expected any) MatchResult {
	pattern := fmt.Sprintf("%v", expected)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return MatchResult{Pass: false, Message: fmt.Sprintf("invalid regex: %s", pattern)}
	}
	pass := re.MatchString(fmt.Sprintf("%v", actual))
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected %v to match /%s/", actual, pattern)}
}

func matchType(actual, expected any) MatchResult {
	expectedType := fmt.Sprintf("%v", expected)
	actualType := typeOf(actual)
	pass := actualType == expectedType
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected type %q, got %q", expectedType, actualType)}
}

func matchExists(actual, expected any) MatchResult {
	expectExists := toBool(expected)
	exists := actual != nil
	pass := exists == expectExists
	msg := fmt.Sprintf("expected value to exist=%v, got %v", expectExists, actual)
	return MatchResult{Pass: pass, Message: msg}
}

func matchNotExists(actual, expected any) MatchResult {
	pass := actual == nil
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected value to not exist, got %v", actual)}
}

func matchIsNull(actual, expected any) MatchResult {
	pass := actual == nil
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected null, got %v", actual)}
}

func matchIsNotNull(actual, expected any) MatchResult {
	pass := actual != nil
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected non-null, got nil")}
}

func matchGreaterThan(actual, expected any) MatchResult {
	a, b := toFloat64(actual), toFloat64(expected)
	return MatchResult{Pass: a > b, Message: fmt.Sprintf("expected %v > %v", actual, expected)}
}

func matchLessThan(actual, expected any) MatchResult {
	a, b := toFloat64(actual), toFloat64(expected)
	return MatchResult{Pass: a < b, Message: fmt.Sprintf("expected %v < %v", actual, expected)}
}

func matchGreaterThanOrEqual(actual, expected any) MatchResult {
	a, b := toFloat64(actual), toFloat64(expected)
	return MatchResult{Pass: a >= b, Message: fmt.Sprintf("expected %v >= %v", actual, expected)}
}

func matchLessThanOrEqual(actual, expected any) MatchResult {
	a, b := toFloat64(actual), toFloat64(expected)
	return MatchResult{Pass: a <= b, Message: fmt.Sprintf("expected %v <= %v", actual, expected)}
}

func matchLength(actual, expected any) MatchResult {
	expectedLen := int(toFloat64(expected))
	actualLen := lengthOf(actual)
	return MatchResult{
		Pass:    actualLen == expectedLen,
		Message: fmt.Sprintf("expected length %d, got %d", expectedLen, actualLen),
	}
}

func matchIsEmpty(actual, expected any) MatchResult {
	pass := lengthOf(actual) == 0
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected empty, got length %d", lengthOf(actual))}
}

func matchNotEmpty(actual, expected any) MatchResult {
	pass := lengthOf(actual) > 0
	return MatchResult{Pass: pass, Message: fmt.Sprintf("expected non-empty, got length %d", lengthOf(actual))}
}

func matchHasProperty(actual, expected any) MatchResult {
	prop := fmt.Sprintf("%v", expected)
	m, ok := actual.(map[string]any)
	if !ok {
		return MatchResult{Pass: false, Message: fmt.Sprintf("expected object, got %T", actual)}
	}
	_, exists := m[prop]
	return MatchResult{Pass: exists, Message: fmt.Sprintf("expected object to have property %q", prop)}
}

func matchNotHasProperty(actual, expected any) MatchResult {
	r := matchHasProperty(actual, expected)
	return MatchResult{Pass: !r.Pass, Message: fmt.Sprintf("expected object to not have property %q", expected)}
}

// Helper: normalize YAML numeric types (int vs float64).
func normalizeNumeric(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	default:
		return v
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	case string:
		f := 0.0
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	default:
		return v != nil
	}
}

func typeOf(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case int, int64, float64, float32:
		return "number"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func lengthOf(v any) int {
	switch a := v.(type) {
	case string:
		return len(a)
	case []any:
		return len(a)
	case map[string]any:
		return len(a)
	default:
		return 0
	}
}
```

- [ ] **Step 4: Implement assertion runner**

```go
// internal/assertion/assertion.go
package assertion

import (
	"fmt"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// Operators recognized in assertion definitions.
var operators = []string{
	"equals", "notEquals", "contains", "notContains", "matches",
	"type", "exists", "notExists", "isNull", "isNotNull",
	"greaterThan", "lessThan", "greaterThanOrEqual", "lessThanOrEqual",
	"length", "isEmpty", "notEmpty", "hasProperty", "notHasProperty",
}

// RunAssertions evaluates assertions against step result data.
// assertDef can be a map (HTTP-style with status/headers/json/duration)
// or a slice of path-based assertions (generic adapter style).
func RunAssertions(data map[string]any, assertDef any) ([]tryve.AssertionOutcome, error) {
	switch def := assertDef.(type) {
	case map[string]any:
		return runMapAssertions(data, def)
	case []any:
		return runSliceAssertions(data, def)
	default:
		return nil, fmt.Errorf("unsupported assertion format: %T", assertDef)
	}
}

// runMapAssertions handles HTTP-style assertions: status, statusRange, headers, json, body, duration.
func runMapAssertions(data, def map[string]any) ([]tryve.AssertionOutcome, error) {
	var results []tryve.AssertionOutcome

	// status assertion
	if expected, ok := def["status"]; ok {
		actual := data["status"]
		outcome := assertStatus(actual, expected)
		results = append(results, outcome...)
	}

	// statusRange assertion
	if sr, ok := def["statusRange"]; ok {
		actual := data["status"]
		outcome := assertStatusRange(actual, sr)
		results = append(results, outcome)
	}

	// headers assertion
	if headers, ok := def["headers"]; ok {
		if hm, ok := headers.(map[string]any); ok {
			actualHeaders, _ := data["headers"].(map[string]any)
			for name, expected := range hm {
				actual := findHeader(actualHeaders, name)
				r := Match("equals", actual, expected)
				results = append(results, tryve.AssertionOutcome{
					Path: fmt.Sprintf("headers.%s", name), Operator: "equals",
					Expected: expected, Actual: actual, Passed: r.Pass, Message: r.Message,
				})
			}
		}
	}

	// json (JSONPath) assertions
	if jsonAsserts, ok := def["json"]; ok {
		if arr, ok := jsonAsserts.([]any); ok {
			outcomes, _ := runSliceAssertions(data, arr)
			results = append(results, outcomes...)
		}
	}

	// body assertions
	if bodyAssert, ok := def["body"]; ok {
		if ba, ok := bodyAssert.(map[string]any); ok {
			body := fmt.Sprintf("%v", data["body"])
			for op, expected := range ba {
				r := Match(op, body, expected)
				results = append(results, tryve.AssertionOutcome{
					Path: "body", Operator: op, Expected: expected, Actual: body,
					Passed: r.Pass, Message: r.Message,
				})
			}
		}
	}

	// duration assertion
	if durAssert, ok := def["duration"]; ok {
		if da, ok := durAssert.(map[string]any); ok {
			actual := data["duration"]
			for op, expected := range da {
				r := Match(op, actual, expected)
				results = append(results, tryve.AssertionOutcome{
					Path: "duration", Operator: op, Expected: expected, Actual: actual,
					Passed: r.Pass, Message: r.Message,
				})
			}
		}
	}

	// Generic operator assertions at top level (for non-HTTP adapters)
	for _, op := range operators {
		if expected, ok := def[op]; ok {
			// These apply to the whole data object or a specific path
			if path, ok := def["path"]; ok {
				actual, _ := EvalJSONPath(data, fmt.Sprintf("%v", path))
				r := Match(op, actual, expected)
				results = append(results, tryve.AssertionOutcome{
					Path: fmt.Sprintf("%v", path), Operator: op,
					Expected: expected, Actual: actual, Passed: r.Pass, Message: r.Message,
				})
			}
		}
	}

	return results, nil
}

// runSliceAssertions handles a list of {path, operator, expected} assertions.
func runSliceAssertions(data map[string]any, assertions []any) ([]tryve.AssertionOutcome, error) {
	var results []tryve.AssertionOutcome
	for _, item := range assertions {
		a, ok := item.(map[string]any)
		if !ok {
			continue
		}
		path, _ := a["path"].(string)
		if path == "" {
			continue
		}
		actual, found := EvalJSONPath(data, path)

		for _, op := range operators {
			expected, has := a[op]
			if !has {
				continue
			}
			// Special handling for exists/notExists: pass found status
			var r MatchResult
			if op == "exists" {
				if toBool(expected) {
					r = MatchResult{Pass: found, Message: fmt.Sprintf("expected path %s to exist", path)}
				} else {
					r = MatchResult{Pass: !found, Message: fmt.Sprintf("expected path %s to not exist", path)}
				}
			} else if op == "notExists" {
				r = MatchResult{Pass: !found, Message: fmt.Sprintf("expected path %s to not exist", path)}
			} else {
				r = Match(op, actual, expected)
			}
			results = append(results, tryve.AssertionOutcome{
				Path: path, Operator: op, Expected: expected, Actual: actual,
				Passed: r.Pass, Message: r.Message,
			})
		}
	}
	return results, nil
}

func assertStatus(actual, expected any) []tryve.AssertionOutcome {
	switch exp := expected.(type) {
	case []any:
		// status: [200, 201, 202]
		r := Match("contains", exp, actual)
		return []tryve.AssertionOutcome{{
			Path: "status", Operator: "oneOf", Expected: exp, Actual: actual,
			Passed: r.Pass, Message: r.Message,
		}}
	default:
		r := Match("equals", actual, expected)
		return []tryve.AssertionOutcome{{
			Path: "status", Operator: "equals", Expected: expected, Actual: actual,
			Passed: r.Pass, Message: r.Message,
		}}
	}
}

func assertStatusRange(actual, sr any) tryve.AssertionOutcome {
	arr, ok := sr.([]any)
	if !ok || len(arr) != 2 {
		return tryve.AssertionOutcome{
			Path: "status", Operator: "statusRange", Passed: false,
			Message: "statusRange must be [min, max]",
		}
	}
	min, max := toFloat64(arr[0]), toFloat64(arr[1])
	status := toFloat64(actual)
	pass := status >= min && status <= max
	return tryve.AssertionOutcome{
		Path: "status", Operator: "statusRange", Expected: sr, Actual: actual,
		Passed: pass, Message: fmt.Sprintf("expected status %v in range [%v, %v]", actual, arr[0], arr[1]),
	}
}

// findHeader does case-insensitive header lookup.
func findHeader(headers map[string]any, name string) any {
	if headers == nil {
		return nil
	}
	// Try exact match first
	if v, ok := headers[name]; ok {
		return v
	}
	// Case-insensitive fallback
	lower := toLower(name)
	for k, v := range headers {
		if toLower(k) == lower {
			return v
		}
	}
	return nil
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
```

- [ ] **Step 5: Run all assertion tests**

Run: `go test ./internal/assertion/... -v`
Expected: all matcher, JSONPath, and assertion runner tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/assertion/
git commit -m "feat(assertion): add matchers and assertion runner"
```

---

### Task 8: YAML Test Loader

**Files:**
- Create: `internal/loader/discovery.go`, `internal/loader/parser.go`, `internal/loader/validator.go`, `internal/loader/loader_test.go`

- [ ] **Step 1: Write loader tests**

```go
// internal/loader/loader_test.go
package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/go-tryve/internal/loader"
)

func TestDiscover_FindsTestFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "api.test.yaml"), []byte("name: test1"), 0644)
	os.WriteFile(filepath.Join(dir, "not-a-test.yaml"), []byte("data: 1"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "db.test.yaml"), []byte("name: test2"), 0644)

	files, err := loader.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("found %d files, want 2", len(files))
	}
}

func TestParse_MinimalTest(t *testing.T) {
	dir := t.TempDir()
	content := `
name: "Simple API Test"
description: "Tests the health endpoint"
tags: [smoke, api]
priority: P0
execute:
  - adapter: http
    action: request
    url: /health
    method: GET
    assert:
      status: 200
`
	f := filepath.Join(dir, "health.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, err := loader.ParseFile(f)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if td.Name != "Simple API Test" {
		t.Errorf("name = %q", td.Name)
	}
	if len(td.Tags) != 2 {
		t.Errorf("tags = %v", td.Tags)
	}
	if string(td.Priority) != "P0" {
		t.Errorf("priority = %q", td.Priority)
	}
	if len(td.Execute) != 1 {
		t.Fatalf("execute steps = %d, want 1", len(td.Execute))
	}

	step := td.Execute[0]
	if step.ID != "execute-0" {
		t.Errorf("step ID = %q, want %q", step.ID, "execute-0")
	}
	if step.Adapter != "http" {
		t.Errorf("adapter = %q", step.Adapter)
	}
	if step.Params["url"] != "/health" {
		t.Errorf("url = %v", step.Params["url"])
	}
	if step.Params["method"] != "GET" {
		t.Errorf("method = %v", step.Params["method"])
	}
}

func TestParse_WithAllPhases(t *testing.T) {
	content := `
name: "Full Lifecycle"
setup:
  - adapter: shell
    action: exec
    command: "echo setup"
execute:
  - adapter: http
    action: request
    url: /api/test
verify:
  - adapter: http
    action: request
    url: /api/verify
teardown:
  - adapter: shell
    action: exec
    command: "echo cleanup"
`
	dir := t.TempDir()
	f := filepath.Join(dir, "full.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, err := loader.ParseFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(td.Setup) != 1 {
		t.Errorf("setup steps = %d", len(td.Setup))
	}
	if len(td.Execute) != 1 {
		t.Errorf("execute steps = %d", len(td.Execute))
	}
	if len(td.Verify) != 1 {
		t.Errorf("verify steps = %d", len(td.Verify))
	}
	if len(td.Teardown) != 1 {
		t.Errorf("teardown steps = %d", len(td.Teardown))
	}
	if td.Setup[0].ID != "setup-0" {
		t.Errorf("setup step ID = %q", td.Setup[0].ID)
	}
}

func TestParse_WithCapture(t *testing.T) {
	content := `
name: "Capture Test"
execute:
  - adapter: http
    action: request
    url: /api/users
    method: POST
    body:
      name: "test"
    capture:
      userId: "$.body.id"
`
	dir := t.TempDir()
	f := filepath.Join(dir, "capture.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, _ := loader.ParseFile(f)
	step := td.Execute[0]
	if step.Capture["userId"] != "$.body.id" {
		t.Errorf("capture = %v", step.Capture)
	}
}

func TestValidate_MissingName(t *testing.T) {
	content := `execute:
  - adapter: http
    action: request
    url: /test
`
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, _ := loader.ParseFile(f)
	errs := loader.Validate(td)
	if len(errs) == 0 {
		t.Error("expected validation errors for missing name")
	}
}

func TestValidate_InvalidAdapter(t *testing.T) {
	content := `
name: "Bad Adapter"
execute:
  - adapter: nonexistent
    action: test
`
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, _ := loader.ParseFile(f)
	errs := loader.Validate(td)
	hasAdapterErr := false
	for _, e := range errs {
		if contains(e.Error(), "adapter") {
			hasAdapterErr = true
		}
	}
	if !hasAdapterErr {
		t.Error("expected validation error for invalid adapter")
	}
}

func TestValidate_HTTPMissingURL(t *testing.T) {
	content := `
name: "No URL"
execute:
  - adapter: http
    action: request
    method: GET
`
	dir := t.TempDir()
	f := filepath.Join(dir, "nourl.test.yaml")
	os.WriteFile(f, []byte(content), 0644)

	td, _ := loader.ParseFile(f)
	errs := loader.Validate(td)
	if len(errs) == 0 {
		t.Error("expected validation error for missing url")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && containsStr(s, sub)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Implement discovery**

```go
// internal/loader/discovery.go
package loader

import (
	"os"
	"path/filepath"
	"strings"
)

// Discover finds all *.test.yaml and *.test.yml files recursively under dir.
func Discover(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden directories and node_modules
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".test.yaml") || strings.HasSuffix(path, ".test.yml") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
```

- [ ] **Step 3: Implement parser**

```go
// internal/loader/parser.go
package loader

import (
	"fmt"
	"os"

	"github.com/liemle3893/go-tryve/internal/tryve"
	"gopkg.in/yaml.v3"
)

// knownStepFields are fields handled directly by StepDefinition, not collected into Params.
var knownStepFields = map[string]bool{
	"adapter": true, "action": true, "description": true,
	"capture": true, "assert": true, "continueOnError": true,
	"retry": true, "delay": true, "id": true,
}

// ParseFile reads a YAML test file and returns a TestDefinition.
func ParseFile(path string) (*tryve.TestDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Parse into raw map first to extract params from top-level fields
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Also parse into struct for typed fields
	var td tryve.TestDefinition
	if err := yaml.Unmarshal(data, &td); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	td.SourceFile = path

	// Parse steps for each phase, extracting params from extra fields
	td.Setup = parseSteps(raw, "setup", tryve.PhaseSetup)
	td.Execute = parseSteps(raw, "execute", tryve.PhaseExecute)
	td.Verify = parseSteps(raw, "verify", tryve.PhaseVerify)
	td.Teardown = parseSteps(raw, "teardown", tryve.PhaseTeardown)

	return &td, nil
}

// parseSteps extracts step definitions from a phase in the raw YAML map.
func parseSteps(raw map[string]any, key string, phase tryve.TestPhase) []tryve.StepDefinition {
	stepsRaw, ok := raw[key]
	if !ok {
		return nil
	}
	arr, ok := stepsRaw.([]any)
	if !ok {
		return nil
	}

	steps := make([]tryve.StepDefinition, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		step := parseStep(m, phase, i)
		steps = append(steps, step)
	}
	return steps
}

// parseStep converts a raw YAML map into a StepDefinition.
// All fields not in knownStepFields are collected into Params.
func parseStep(m map[string]any, phase tryve.TestPhase, index int) tryve.StepDefinition {
	step := tryve.StepDefinition{
		ID:     fmt.Sprintf("%s-%d", phase, index),
		Params: make(map[string]any),
	}

	for k, v := range m {
		switch k {
		case "adapter":
			step.Adapter = fmt.Sprintf("%v", v)
		case "action":
			step.Action = fmt.Sprintf("%v", v)
		case "description":
			step.Description = fmt.Sprintf("%v", v)
		case "continueOnError":
			if b, ok := v.(bool); ok {
				step.ContinueOnError = b
			}
		case "retry":
			if n, ok := toInt(v); ok {
				step.Retry = n
			}
		case "delay":
			if n, ok := toInt(v); ok {
				step.Delay = n
			}
		case "capture":
			step.Capture = toStringMap(v)
		case "assert":
			step.Assert = v
		default:
			// Collect into params
			step.Params[k] = v
		}
	}
	return step
}

func toStringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, val := range m {
		result[k] = fmt.Sprintf("%v", val)
	}
	return result
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}
```

- [ ] **Step 4: Implement validator**

```go
// internal/loader/validator.go
package loader

import (
	"fmt"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

var validAdapters = map[string]bool{
	"http": true, "postgresql": true, "mongodb": true,
	"redis": true, "kafka": true, "eventhub": true, "shell": true,
}

var validPriorities = map[string]bool{
	"P0": true, "P1": true, "P2": true, "P3": true, "": true,
}

// Validate checks a TestDefinition for structural and semantic errors.
func Validate(td *tryve.TestDefinition) []error {
	var errs []error

	if td.Name == "" {
		errs = append(errs, fmt.Errorf("test name is required"))
	}
	if len(td.Execute) == 0 {
		errs = append(errs, fmt.Errorf("at least one execute step is required"))
	}
	if !validPriorities[string(td.Priority)] {
		errs = append(errs, fmt.Errorf("invalid priority %q (must be P0-P3)", td.Priority))
	}
	if td.Timeout < 0 || td.Timeout > 300000 {
		if td.Timeout != 0 {
			errs = append(errs, fmt.Errorf("timeout must be between 0 and 300000, got %d", td.Timeout))
		}
	}
	if td.Retries < 0 || td.Retries > 5 {
		errs = append(errs, fmt.Errorf("retries must be between 0 and 5, got %d", td.Retries))
	}

	// Validate all steps in all phases
	for _, s := range td.Setup {
		errs = append(errs, validateStep(&s, "setup")...)
	}
	for _, s := range td.Execute {
		errs = append(errs, validateStep(&s, "execute")...)
	}
	for _, s := range td.Verify {
		errs = append(errs, validateStep(&s, "verify")...)
	}
	for _, s := range td.Teardown {
		errs = append(errs, validateStep(&s, "teardown")...)
	}
	return errs
}

func validateStep(step *tryve.StepDefinition, phase string) []error {
	var errs []error
	prefix := fmt.Sprintf("%s step %q", phase, step.ID)

	if !validAdapters[step.Adapter] {
		errs = append(errs, fmt.Errorf("%s: unknown adapter %q", prefix, step.Adapter))
		return errs // skip adapter-specific validation
	}
	if step.Action == "" {
		errs = append(errs, fmt.Errorf("%s: action is required", prefix))
	}

	// Adapter-specific validation
	switch step.Adapter {
	case "http":
		errs = append(errs, validateHTTPStep(step, prefix)...)
	case "shell":
		errs = append(errs, validateShellStep(step, prefix)...)
	case "postgresql":
		errs = append(errs, validatePostgresStep(step, prefix)...)
	case "mongodb":
		errs = append(errs, validateMongoStep(step, prefix)...)
	case "redis":
		errs = append(errs, validateRedisStep(step, prefix)...)
	case "kafka":
		errs = append(errs, validateKafkaStep(step, prefix)...)
	case "eventhub":
		errs = append(errs, validateEventHubStep(step, prefix)...)
	}
	return errs
}

func validateHTTPStep(step *tryve.StepDefinition, prefix string) []error {
	var errs []error
	if step.Action != "request" {
		errs = append(errs, fmt.Errorf("%s: http adapter only supports action 'request', got %q", prefix, step.Action))
	}
	if _, ok := step.Params["url"]; !ok {
		errs = append(errs, fmt.Errorf("%s: http step requires 'url' field", prefix))
	}
	return errs
}

func validateShellStep(step *tryve.StepDefinition, prefix string) []error {
	var errs []error
	if step.Action != "exec" {
		errs = append(errs, fmt.Errorf("%s: shell adapter only supports action 'exec', got %q", prefix, step.Action))
	}
	if _, ok := step.Params["command"]; !ok {
		errs = append(errs, fmt.Errorf("%s: shell step requires 'command' field", prefix))
	}
	return errs
}

func validatePostgresStep(step *tryve.StepDefinition, prefix string) []error {
	validActions := map[string]bool{"execute": true, "query": true, "queryOne": true, "count": true}
	var errs []error
	if !validActions[step.Action] {
		errs = append(errs, fmt.Errorf("%s: invalid postgresql action %q", prefix, step.Action))
	}
	if _, ok := step.Params["sql"]; !ok {
		errs = append(errs, fmt.Errorf("%s: postgresql step requires 'sql' field", prefix))
	}
	return errs
}

func validateMongoStep(step *tryve.StepDefinition, prefix string) []error {
	validActions := map[string]bool{
		"insertOne": true, "insertMany": true, "findOne": true, "find": true,
		"updateOne": true, "updateMany": true, "deleteOne": true, "deleteMany": true,
		"count": true, "aggregate": true,
	}
	var errs []error
	if !validActions[step.Action] {
		errs = append(errs, fmt.Errorf("%s: invalid mongodb action %q", prefix, step.Action))
	}
	if _, ok := step.Params["collection"]; !ok {
		errs = append(errs, fmt.Errorf("%s: mongodb step requires 'collection' field", prefix))
	}
	return errs
}

func validateRedisStep(step *tryve.StepDefinition, prefix string) []error {
	validActions := map[string]bool{
		"get": true, "set": true, "del": true, "exists": true, "incr": true,
		"hget": true, "hset": true, "hgetall": true, "keys": true, "flushPattern": true,
	}
	var errs []error
	if !validActions[step.Action] {
		errs = append(errs, fmt.Errorf("%s: invalid redis action %q", prefix, step.Action))
	}
	return errs
}

func validateKafkaStep(step *tryve.StepDefinition, prefix string) []error {
	validActions := map[string]bool{"produce": true, "consume": true, "waitFor": true, "clear": true}
	var errs []error
	if !validActions[step.Action] {
		errs = append(errs, fmt.Errorf("%s: invalid kafka action %q", prefix, step.Action))
	}
	if step.Action != "clear" {
		if _, ok := step.Params["topic"]; !ok {
			errs = append(errs, fmt.Errorf("%s: kafka step requires 'topic' field", prefix))
		}
	}
	return errs
}

func validateEventHubStep(step *tryve.StepDefinition, prefix string) []error {
	validActions := map[string]bool{"publish": true, "waitFor": true, "consume": true, "clear": true}
	var errs []error
	if !validActions[step.Action] {
		errs = append(errs, fmt.Errorf("%s: invalid eventhub action %q", prefix, step.Action))
	}
	if step.Action != "clear" {
		if _, ok := step.Params["topic"]; !ok {
			errs = append(errs, fmt.Errorf("%s: eventhub step requires 'topic' field", prefix))
		}
	}
	return errs
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/loader/... -v`
Expected: all loader tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/loader/
git commit -m "feat(loader): add YAML test discovery, parser, and validator"
```

---

### Task 9: Adapter Interface & Registry

**Files:**
- Create: `internal/adapter/adapter.go`, `internal/adapter/registry.go`, `internal/adapter/registry_test.go`

- [ ] **Step 1: Write registry tests**

```go
// internal/adapter/registry_test.go
package adapter_test

import (
	"context"
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

type mockAdapter struct {
	name      string
	connected bool
}

func (m *mockAdapter) Name() string                     { return m.name }
func (m *mockAdapter) Connect(ctx context.Context) error { m.connected = true; return nil }
func (m *mockAdapter) Close(ctx context.Context) error   { m.connected = false; return nil }
func (m *mockAdapter) Health(ctx context.Context) error   { return nil }
func (m *mockAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	return &tryve.StepResult{Data: map[string]any{"mock": true}}, nil
}

func TestRegistry_GetInitializesOnce(t *testing.T) {
	r := adapter.NewRegistry()
	mock := &mockAdapter{name: "test"}
	r.Register("test", mock)

	a, err := r.Get(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name() != "test" {
		t.Errorf("name = %q", a.Name())
	}
	if !mock.connected {
		t.Error("expected Connect to be called")
	}

	// Second call should not re-connect
	mock.connected = false
	a2, _ := r.Get(context.Background(), "test")
	if a2 != a {
		t.Error("expected same instance")
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	r := adapter.NewRegistry()
	_, err := r.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for unknown adapter")
	}
}

func TestRegistry_CloseAll(t *testing.T) {
	r := adapter.NewRegistry()
	mock := &mockAdapter{name: "test"}
	r.Register("test", mock)
	r.Get(context.Background(), "test") // trigger connect

	r.CloseAll(context.Background())
	if mock.connected {
		t.Error("expected Close to be called")
	}
}
```

- [ ] **Step 2: Implement adapter interface and registry**

```go
// internal/adapter/adapter.go
package adapter

import (
	"context"
	"time"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// Adapter is the core interface all protocol adapters implement.
type Adapter interface {
	Name() string
	Connect(ctx context.Context) error
	Close(ctx context.Context) error
	Health(ctx context.Context) error
	Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error)
}

// MeasureDuration times a function and returns its duration alongside the result.
func MeasureDuration(fn func() error) (time.Duration, error) {
	start := time.Now()
	err := fn()
	return time.Since(start), err
}

// SuccessResult constructs a successful StepResult.
func SuccessResult(data map[string]any, duration time.Duration, metadata map[string]any) *tryve.StepResult {
	return &tryve.StepResult{Data: data, Duration: duration, Metadata: metadata}
}
```

```go
// internal/adapter/registry.go
package adapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// Registry manages adapter instances with lazy initialization.
type Registry struct {
	mu        sync.Mutex
	adapters  map[string]Adapter
	connected map[string]bool
}

// NewRegistry creates an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters:  make(map[string]Adapter),
		connected: make(map[string]bool),
	}
}

// Register adds an adapter to the registry without connecting it.
func (r *Registry) Register(name string, a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = a
}

// Get returns an adapter by name, connecting it on first access.
func (r *Registry) Get(ctx context.Context, name string) (Adapter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.adapters[name]
	if !ok {
		return nil, tryve.ConfigError(
			fmt.Sprintf("adapter %q not configured", name),
			fmt.Sprintf("add %s configuration to e2e.config.yaml", name),
			nil,
		)
	}

	if !r.connected[name] {
		if err := a.Connect(ctx); err != nil {
			return nil, tryve.ConnectionError(name, "connection failed", err)
		}
		r.connected[name] = true
	}
	return a, nil
}

// CloseAll disconnects all connected adapters.
func (r *Registry) CloseAll(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, a := range r.adapters {
		if r.connected[name] {
			a.Close(ctx)
			r.connected[name] = false
		}
	}
}

// Has checks if an adapter is registered (regardless of connection state).
func (r *Registry) Has(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.adapters[name]
	return ok
}

// Names returns the list of registered adapter names.
func (r *Registry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/adapter/... -v`
Expected: all registry tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/adapter.go internal/adapter/registry.go internal/adapter/registry_test.go
git commit -m "feat(adapter): add adapter interface and lazy registry"
```

---

### Task 10: HTTP Adapter

**Files:**
- Create: `internal/adapter/http.go`, `internal/adapter/http_test.go`

- [ ] **Step 1: Write HTTP adapter tests**

```go
// internal/adapter/http_test.go
package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
)

func TestHTTPAdapter_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())
	defer a.Close(context.Background())

	result, err := a.Execute(context.Background(), "request", map[string]any{
		"url":    "/test",
		"method": "GET",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["status"] == nil {
		t.Error("expected status in response data")
	}
	status, _ := result.Data["status"].(float64)
	if status != 200 {
		t.Errorf("status = %v, want 200", result.Data["status"])
	}
}

func TestHTTPAdapter_POST_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": body["name"]})
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())

	result, err := a.Execute(context.Background(), "request", map[string]any{
		"url":    "/users",
		"method": "POST",
		"body":   map[string]any{"name": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := result.Data["body"].(map[string]any)
	if body["name"] != "alice" {
		t.Errorf("body.name = %v", body["name"])
	}
}

func TestHTTPAdapter_Headers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())

	_, err := a.Execute(context.Background(), "request", map[string]any{
		"url":     "/secure",
		"method":  "GET",
		"headers": map[string]any{"Authorization": "Bearer token123"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPAdapter_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())

	_, err := a.Execute(context.Background(), "request", map[string]any{
		"url":   "/items",
		"query": map[string]any{"page": "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPAdapter_NonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())

	result, err := a.Execute(context.Background(), "request", map[string]any{
		"url": "/text",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := result.Data["body"]
	if body != "hello world" {
		t.Errorf("body = %v", body)
	}
}

func TestHTTPAdapter_Health(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	a := adapter.NewHTTPAdapter(srv.URL)
	a.Connect(context.Background())

	if err := a.Health(context.Background()); err != nil {
		t.Errorf("health check failed: %v", err)
	}
}

func TestHTTPAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewHTTPAdapter("http://localhost")
	a.Connect(context.Background())

	_, err := a.Execute(context.Background(), "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid action")
	}
}
```

- [ ] **Step 2: Implement HTTP adapter**

```go
// internal/adapter/http.go
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// HTTPAdapter handles HTTP/REST API requests.
type HTTPAdapter struct {
	baseURL string
	client  *http.Client
}

// NewHTTPAdapter creates a new HTTP adapter with the given base URL.
func NewHTTPAdapter(baseURL string) *HTTPAdapter {
	return &HTTPAdapter{baseURL: strings.TrimRight(baseURL, "/")}
}

func (a *HTTPAdapter) Name() string { return "http" }

func (a *HTTPAdapter) Connect(ctx context.Context) error {
	jar, _ := cookiejar.New(nil)
	a.client = &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	return nil
}

func (a *HTTPAdapter) Close(ctx context.Context) error {
	a.client.CloseIdleConnections()
	return nil
}

func (a *HTTPAdapter) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "HEAD", a.baseURL, nil)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	resp.Body.Close()
	return nil
}

func (a *HTTPAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	if action != "request" {
		return nil, tryve.AdapterError("http", action, "only 'request' action is supported", nil)
	}

	method := strings.ToUpper(getStr(params, "method", "GET"))
	rawURL := getStr(params, "url", "/")
	headers := getMap(params, "headers")
	query := getMap(params, "query")
	bodyParam := params["body"]

	// Build URL
	fullURL, err := a.buildURL(rawURL, query)
	if err != nil {
		return nil, tryve.AdapterError("http", action, "invalid URL", err)
	}

	// Build body
	var bodyReader io.Reader
	if bodyParam != nil && method != "GET" && method != "HEAD" {
		bodyBytes, err := json.Marshal(bodyParam)
		if err != nil {
			return nil, tryve.AdapterError("http", action, "cannot serialize body", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
		if headers == nil {
			headers = make(map[string]any)
		}
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, tryve.AdapterError("http", action, "cannot create request", err)
	}
	for k, v := range headers {
		req.Header.Set(k, fmt.Sprintf("%v", v))
	}

	// Execute with timing
	start := time.Now()
	resp, err := a.client.Do(req)
	duration := time.Since(start)
	if err != nil {
		return nil, tryve.AdapterError("http", action, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, tryve.AdapterError("http", action, "cannot read response body", err)
	}

	// Parse response body
	var parsedBody any
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.Unmarshal(respBody, &parsedBody); err != nil {
			parsedBody = string(respBody) // fallback to string
		}
	} else {
		parsedBody = string(respBody)
	}

	// Build response headers map
	respHeaders := make(map[string]any)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	data := map[string]any{
		"status":     float64(resp.StatusCode),
		"statusText": resp.Status,
		"headers":    respHeaders,
		"body":       parsedBody,
		"duration":   float64(duration.Milliseconds()),
	}

	metadata := map[string]any{
		"method":      method,
		"url":         fullURL,
		"contentType": contentType,
	}

	return SuccessResult(data, duration, metadata), nil
}

func (a *HTTPAdapter) buildURL(rawURL string, query map[string]any) (string, error) {
	var fullURL string
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		fullURL = rawURL
	} else {
		fullURL = a.baseURL + rawURL
	}

	if len(query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", err
		}
		q := u.Query()
		for k, v := range query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}
	return fullURL, nil
}

// Helper functions for extracting typed values from params map.
func getStr(m map[string]any, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return defaultVal
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/adapter/... -v`
Expected: all HTTP adapter and registry tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/http.go internal/adapter/http_test.go
git commit -m "feat(adapter): add HTTP adapter with JSON/cookie support"
```

---

### Task 11: Shell Adapter

**Files:**
- Create: `internal/adapter/shell.go`, `internal/adapter/shell_test.go`

- [ ] **Step 1: Write shell adapter tests**

```go
// internal/adapter/shell_test.go
package adapter_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
)

func TestShellAdapter_ExecSimple(t *testing.T) {
	a := adapter.NewShellAdapter(nil)
	a.Connect(context.Background())

	result, err := a.Execute(context.Background(), "exec", map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ := result.Data["stdout"].(string)
	if stdout != "hello\n" && stdout != "hello\r\n" {
		t.Errorf("stdout = %q, want 'hello\\n'", stdout)
	}
	exitCode, _ := result.Data["exitCode"].(float64)
	if exitCode != 0 {
		t.Errorf("exitCode = %v, want 0", exitCode)
	}
}

func TestShellAdapter_ExecWithEnv(t *testing.T) {
	a := adapter.NewShellAdapter(nil)
	a.Connect(context.Background())

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo %TEST_VAR%"
	} else {
		cmd = "echo $TEST_VAR"
	}
	result, err := a.Execute(context.Background(), "exec", map[string]any{
		"command": cmd,
		"env":     map[string]any{"TEST_VAR": "test_value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ := result.Data["stdout"].(string)
	if stdout != "test_value\n" && stdout != "test_value\r\n" {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestShellAdapter_ExecFailure(t *testing.T) {
	a := adapter.NewShellAdapter(nil)
	a.Connect(context.Background())

	result, err := a.Execute(context.Background(), "exec", map[string]any{
		"command": "exit 1",
	})
	// Non-zero exit is returned in the result, not as an error
	if err != nil {
		t.Fatal(err)
	}
	exitCode, _ := result.Data["exitCode"].(float64)
	if exitCode != 1 {
		t.Errorf("exitCode = %v, want 1", exitCode)
	}
}

func TestShellAdapter_InvalidAction(t *testing.T) {
	a := adapter.NewShellAdapter(nil)
	a.Connect(context.Background())

	_, err := a.Execute(context.Background(), "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid action")
	}
}
```

- [ ] **Step 2: Implement shell adapter**

```go
// internal/adapter/shell.go
package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// ShellConfig holds optional default settings for the shell adapter.
type ShellConfig struct {
	DefaultTimeout int    // milliseconds
	DefaultCwd     string // working directory
}

// ShellAdapter executes shell commands.
type ShellAdapter struct {
	config *ShellConfig
}

// NewShellAdapter creates a new shell adapter.
func NewShellAdapter(config *ShellConfig) *ShellAdapter {
	if config == nil {
		config = &ShellConfig{DefaultTimeout: 30000}
	}
	return &ShellAdapter{config: config}
}

func (a *ShellAdapter) Name() string                       { return "shell" }
func (a *ShellAdapter) Connect(ctx context.Context) error  { return nil }
func (a *ShellAdapter) Close(ctx context.Context) error    { return nil }
func (a *ShellAdapter) Health(ctx context.Context) error   { return nil }

func (a *ShellAdapter) Execute(ctx context.Context, action string, params map[string]any) (*tryve.StepResult, error) {
	if action != "exec" {
		return nil, tryve.AdapterError("shell", action, "only 'exec' action is supported", nil)
	}

	command := getStr(params, "command", "")
	if command == "" {
		return nil, tryve.AdapterError("shell", action, "command is required", nil)
	}

	// Build command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// Working directory
	if cwd := getStr(params, "cwd", a.config.DefaultCwd); cwd != "" {
		cmd.Dir = cwd
	}

	// Environment variables
	if envMap := getMap(params, "env"); envMap != nil {
		env := cmd.Environ()
		for k, v := range envMap {
			env = append(env, fmt.Sprintf("%s=%v", k, v))
		}
		cmd.Env = env
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, tryve.AdapterError("shell", action, "command execution failed", err)
		}
	}

	data := map[string]any{
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
		"exitCode": float64(exitCode),
	}
	metadata := map[string]any{
		"command": command,
	}

	return SuccessResult(data, duration, metadata), nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/adapter/... -v`
Expected: all adapter tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/shell.go internal/adapter/shell_test.go
git commit -m "feat(adapter): add shell adapter"
```

---

### Task 12: Console Reporter

**Files:**
- Create: `internal/reporter/reporter.go`, `internal/reporter/console.go`, `internal/reporter/console_test.go`

- [ ] **Step 1: Write reporter tests**

```go
// internal/reporter/console_test.go
package reporter_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestConsoleReporter_SuiteComplete(t *testing.T) {
	var buf bytes.Buffer
	r := reporter.NewConsole(&buf, false, false) // no verbose, no color

	ctx := context.Background()
	r.OnSuiteStart(ctx, nil)
	r.OnSuiteComplete(ctx, nil, &tryve.SuiteResult{
		Total: 3, Passed: 2, Failed: 1, Duration: 1500 * time.Millisecond,
	})
	r.Flush()

	output := buf.String()
	if !strings.Contains(output, "2 passed") {
		t.Errorf("missing pass count in output: %s", output)
	}
	if !strings.Contains(output, "1 failed") {
		t.Errorf("missing fail count in output: %s", output)
	}
}

func TestConsoleReporter_TestComplete(t *testing.T) {
	var buf bytes.Buffer
	r := reporter.NewConsole(&buf, false, false)

	ctx := context.Background()
	r.OnTestComplete(ctx, &tryve.TestDefinition{Name: "My Test"}, &tryve.TestResult{
		Status:   tryve.StatusPassed,
		Duration: 250 * time.Millisecond,
	})
	r.Flush()

	output := buf.String()
	if !strings.Contains(output, "My Test") {
		t.Errorf("missing test name: %s", output)
	}
	if !strings.Contains(output, "PASS") {
		t.Errorf("missing PASS: %s", output)
	}
}

func TestMultiReporter(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	r1 := reporter.NewConsole(&buf1, false, false)
	r2 := reporter.NewConsole(&buf2, false, false)
	multi := reporter.NewMulti(r1, r2)

	ctx := context.Background()
	multi.OnTestComplete(ctx, &tryve.TestDefinition{Name: "Test"}, &tryve.TestResult{
		Status: tryve.StatusPassed, Duration: 100 * time.Millisecond,
	})
	multi.Flush()

	if buf1.Len() == 0 || buf2.Len() == 0 {
		t.Error("expected both reporters to receive events")
	}
}
```

- [ ] **Step 2: Implement reporter interface and console reporter**

```go
// internal/reporter/reporter.go
package reporter

import (
	"context"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// Reporter receives lifecycle events during test execution.
type Reporter interface {
	OnSuiteStart(ctx context.Context, suite *tryve.SuiteResult) error
	OnTestStart(ctx context.Context, test *tryve.TestDefinition) error
	OnStepComplete(ctx context.Context, step *tryve.StepDefinition, outcome *tryve.StepOutcome) error
	OnTestComplete(ctx context.Context, test *tryve.TestDefinition, result *tryve.TestResult) error
	OnSuiteComplete(ctx context.Context, suite *tryve.SuiteResult, result *tryve.SuiteResult) error
	Flush() error
}

// Multi dispatches events to multiple reporters.
type Multi struct {
	reporters []Reporter
}

// NewMulti creates a multi-reporter.
func NewMulti(reporters ...Reporter) *Multi {
	return &Multi{reporters: reporters}
}

func (m *Multi) OnSuiteStart(ctx context.Context, s *tryve.SuiteResult) error {
	for _, r := range m.reporters {
		r.OnSuiteStart(ctx, s)
	}
	return nil
}

func (m *Multi) OnTestStart(ctx context.Context, td *tryve.TestDefinition) error {
	for _, r := range m.reporters {
		r.OnTestStart(ctx, td)
	}
	return nil
}

func (m *Multi) OnStepComplete(ctx context.Context, step *tryve.StepDefinition, outcome *tryve.StepOutcome) error {
	for _, r := range m.reporters {
		r.OnStepComplete(ctx, step, outcome)
	}
	return nil
}

func (m *Multi) OnTestComplete(ctx context.Context, td *tryve.TestDefinition, result *tryve.TestResult) error {
	for _, r := range m.reporters {
		r.OnTestComplete(ctx, td, result)
	}
	return nil
}

func (m *Multi) OnSuiteComplete(ctx context.Context, s *tryve.SuiteResult, result *tryve.SuiteResult) error {
	for _, r := range m.reporters {
		r.OnSuiteComplete(ctx, s, result)
	}
	return nil
}

func (m *Multi) Flush() error {
	for _, r := range m.reporters {
		r.Flush()
	}
	return nil
}
```

```go
// internal/reporter/console.go
package reporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/liemle3893/go-tryve/internal/tryve"
)

// Console writes human-readable test results to an io.Writer.
type Console struct {
	w       io.Writer
	verbose bool
	color   bool
}

// NewConsole creates a console reporter.
func NewConsole(w io.Writer, verbose, color bool) *Console {
	return &Console{w: w, verbose: verbose, color: color}
}

// NewConsoleFromEnv creates a console reporter with color auto-detected from NO_COLOR env.
func NewConsoleFromEnv(verbose bool) *Console {
	_, noColor := os.LookupEnv("NO_COLOR")
	return &Console{w: os.Stdout, verbose: verbose, color: !noColor}
}

func (c *Console) OnSuiteStart(ctx context.Context, s *tryve.SuiteResult) error {
	fmt.Fprintf(c.w, "\n%s\n\n", c.styled("Tryve Test Runner", bold))
	return nil
}

func (c *Console) OnTestStart(ctx context.Context, td *tryve.TestDefinition) error {
	if c.verbose {
		fmt.Fprintf(c.w, "  %s %s\n", c.styled("RUN", cyan), td.Name)
	}
	return nil
}

func (c *Console) OnStepComplete(ctx context.Context, step *tryve.StepDefinition, outcome *tryve.StepOutcome) error {
	if !c.verbose {
		return nil
	}
	icon := c.styled("  +", green)
	if outcome.Status == tryve.StatusFailed {
		icon = c.styled("  x", red)
	}
	desc := step.Description
	if desc == "" {
		desc = fmt.Sprintf("%s.%s", step.Adapter, step.Action)
	}
	fmt.Fprintf(c.w, "    %s %s (%s)\n", icon, desc, outcome.Duration.Round(time.Millisecond))

	// Show failed assertions in verbose mode
	if outcome.Status == tryve.StatusFailed {
		for _, a := range outcome.Assertions {
			if !a.Passed {
				fmt.Fprintf(c.w, "      %s %s: %s\n", c.styled("!", red), a.Path, a.Message)
			}
		}
	}
	return nil
}

func (c *Console) OnTestComplete(ctx context.Context, td *tryve.TestDefinition, result *tryve.TestResult) error {
	var icon, status string
	switch result.Status {
	case tryve.StatusPassed:
		icon = c.styled("PASS", green)
		status = ""
	case tryve.StatusFailed:
		icon = c.styled("FAIL", red)
		status = ""
	case tryve.StatusSkipped:
		icon = c.styled("SKIP", yellow)
		status = ""
	default:
		icon = "????"
	}
	_ = status
	fmt.Fprintf(c.w, "  %s %s (%s)\n", icon, td.Name, result.Duration.Round(time.Millisecond))
	return nil
}

func (c *Console) OnSuiteComplete(ctx context.Context, s *tryve.SuiteResult, result *tryve.SuiteResult) error {
	fmt.Fprintln(c.w)
	fmt.Fprintf(c.w, "  %s\n", c.styled("Results:", bold))
	fmt.Fprintf(c.w, "    %s %d passed\n", c.styled("+", green), result.Passed)
	if result.Failed > 0 {
		fmt.Fprintf(c.w, "    %s %d failed\n", c.styled("x", red), result.Failed)
	}
	if result.Skipped > 0 {
		fmt.Fprintf(c.w, "    %s %d skipped\n", c.styled("-", yellow), result.Skipped)
	}
	fmt.Fprintf(c.w, "    %d total in %s\n", result.Total, result.Duration.Round(time.Millisecond))
	fmt.Fprintln(c.w)
	return nil
}

func (c *Console) Flush() error { return nil }

// ANSI color codes
const (
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	reset  = "\033[0m"
)

func (c *Console) styled(text, style string) string {
	if !c.color {
		return text
	}
	return style + text + reset
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/reporter/... -v`
Expected: all reporter tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/reporter/
git commit -m "feat(reporter): add console reporter with ANSI color support"
```

---

### Task 13: Step Executor

**Files:**
- Create: `internal/executor/step.go`, `internal/executor/step_test.go`

- [ ] **Step 1: Write step executor tests**

```go
// internal/executor/step_test.go
package executor_test

import (
	"context"
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func newTestRegistry(baseURL string) *adapter.Registry {
	r := adapter.NewRegistry()
	r.Register("http", adapter.NewHTTPAdapter(baseURL))
	r.Register("shell", adapter.NewShellAdapter(nil))
	return r
}

func TestExecuteStep_BasicHTTP(t *testing.T) {
	// Uses shell to avoid needing an HTTP server
	registry := newTestRegistry("http://localhost")
	ctx := tryve.NewInterpolationContext()

	step := &tryve.StepDefinition{
		ID:      "execute-0",
		Adapter: "shell",
		Action:  "exec",
		Params:  map[string]any{"command": "echo hello"},
	}

	outcome, err := executor.ExecuteStep(context.Background(), step, registry, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != tryve.StatusPassed {
		t.Errorf("status = %s", outcome.Status)
	}
}

func TestExecuteStep_WithCapture(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	ctx := tryve.NewInterpolationContext()

	step := &tryve.StepDefinition{
		ID:      "execute-0",
		Adapter: "shell",
		Action:  "exec",
		Params:  map[string]any{"command": "echo captured_value"},
		Capture: map[string]string{"output": "$.stdout"},
	}

	outcome, _ := executor.ExecuteStep(context.Background(), step, registry, ctx)
	if outcome.Status != tryve.StatusPassed {
		t.Errorf("status = %s, error = %v", outcome.Status, outcome.Error)
	}
	if ctx.Captured["output"] == nil {
		t.Error("expected captured value")
	}
}

func TestExecuteStep_WithDelay(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	ctx := tryve.NewInterpolationContext()

	step := &tryve.StepDefinition{
		ID:      "execute-0",
		Adapter: "shell",
		Action:  "exec",
		Params:  map[string]any{"command": "echo fast"},
		Delay:   100, // 100ms delay
	}

	outcome, _ := executor.ExecuteStep(context.Background(), step, registry, ctx)
	if outcome.Duration < 100*1e6 { // 100ms in nanoseconds
		t.Errorf("duration = %v, expected >= 100ms", outcome.Duration)
	}
}

func TestExecuteStep_ContinueOnError(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	ctx := tryve.NewInterpolationContext()

	step := &tryve.StepDefinition{
		ID:              "execute-0",
		Adapter:         "shell",
		Action:          "exec",
		Params:          map[string]any{"command": "exit 1"},
		Assert:          map[string]any{"exitCode": float64(0)},
		ContinueOnError: true,
	}

	outcome, _ := executor.ExecuteStep(context.Background(), step, registry, ctx)
	// With continueOnError, should be warned, not failed
	if outcome.Status != tryve.StatusWarned {
		t.Errorf("status = %s, expected warned", outcome.Status)
	}
}

func TestExecuteStep_InterpolatesParams(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	ctx := tryve.NewInterpolationContext()
	ctx.Variables["msg"] = "interpolated"

	step := &tryve.StepDefinition{
		ID:      "execute-0",
		Adapter: "shell",
		Action:  "exec",
		Params:  map[string]any{"command": "echo {{msg}}"},
	}

	outcome, _ := executor.ExecuteStep(context.Background(), step, registry, ctx)
	if outcome.Status != tryve.StatusPassed {
		t.Errorf("status = %s, error = %v", outcome.Status, outcome.Error)
	}
	stdout := outcome.Result.Data["stdout"]
	if stdout != "interpolated\n" && stdout != "interpolated\r\n" {
		t.Errorf("stdout = %q, expected 'interpolated'", stdout)
	}
}
```

- [ ] **Step 2: Implement step executor**

```go
// internal/executor/step.go
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/assertion"
	"github.com/liemle3893/go-tryve/internal/interpolate"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

// ExecuteStep runs a single step: interpolate params → delay → execute → capture → assert.
func ExecuteStep(
	ctx context.Context,
	step *tryve.StepDefinition,
	registry *adapter.Registry,
	interpCtx *tryve.InterpolationContext,
) (*tryve.StepOutcome, error) {
	start := time.Now()
	outcome := &tryve.StepOutcome{Step: step}

	// Pre-delay
	if step.Delay > 0 {
		select {
		case <-time.After(time.Duration(step.Delay) * time.Millisecond):
		case <-ctx.Done():
			outcome.Status = tryve.StatusFailed
			outcome.Error = ctx.Err()
			outcome.Duration = time.Since(start)
			return outcome, nil
		}
	}

	// Interpolate params
	resolvedParams, err := interpolate.ResolveMap(step.Params, interpCtx)
	if err != nil {
		outcome.Status = tryve.StatusFailed
		outcome.Error = tryve.InterpolationError(step.ID, err.Error())
		outcome.Duration = time.Since(start)
		return outcome, nil
	}

	// Get adapter
	a, err := registry.Get(ctx, step.Adapter)
	if err != nil {
		outcome.Status = tryve.StatusFailed
		outcome.Error = err
		outcome.Duration = time.Since(start)
		return outcome, nil
	}

	// Execute
	result, err := a.Execute(ctx, step.Action, resolvedParams)
	if err != nil {
		outcome.Status = tryve.StatusFailed
		outcome.Error = err
		outcome.Duration = time.Since(start)
		if step.ContinueOnError {
			outcome.Status = tryve.StatusWarned
		}
		return outcome, nil
	}
	outcome.Result = result

	// Capture values
	if step.Capture != nil && result.Data != nil {
		for varName, path := range step.Capture {
			val, found := assertion.EvalJSONPath(result.Data, path)
			if found {
				interpCtx.Captured[varName] = val
			}
		}
	}

	// Run assertions
	if step.Assert != nil && result.Data != nil {
		// Interpolate assertion definitions
		resolvedAssert, _ := interpolate.ResolveMap(toAssertMap(step.Assert), interpCtx)
		var assertDef any = resolvedAssert
		if resolvedAssert == nil {
			assertDef = step.Assert
		}

		outcomes, err := assertion.RunAssertions(result.Data, assertDef)
		if err != nil {
			outcome.Status = tryve.StatusFailed
			outcome.Error = err
			outcome.Duration = time.Since(start)
			return outcome, nil
		}
		outcome.Assertions = outcomes

		// Check if any assertion failed
		for _, a := range outcomes {
			if !a.Passed {
				outcome.Status = tryve.StatusFailed
				outcome.Error = tryve.AssertionError(a.Path, a.Operator, a.Expected, a.Actual)
				if step.ContinueOnError {
					outcome.Status = tryve.StatusWarned
				}
				outcome.Duration = time.Since(start)
				return outcome, nil
			}
		}
	}

	outcome.Status = tryve.StatusPassed
	outcome.Duration = time.Since(start)
	return outcome, nil
}

// toAssertMap converts assertion definitions to map format if possible.
func toAssertMap(assert any) map[string]any {
	switch v := assert.(type) {
	case map[string]any:
		return v
	default:
		return nil
	}
}

// ExecuteStepWithRetry wraps ExecuteStep with retry logic.
func ExecuteStepWithRetry(
	ctx context.Context,
	step *tryve.StepDefinition,
	registry *adapter.Registry,
	interpCtx *tryve.InterpolationContext,
	maxRetries int,
	baseDelay time.Duration,
) (*tryve.StepOutcome, int) {
	var outcome *tryve.StepOutcome
	retries := 0

	for attempt := 0; attempt <= maxRetries; attempt++ {
		outcome, _ = ExecuteStep(ctx, step, registry, interpCtx)
		if outcome.Status == tryve.StatusPassed || outcome.Status == tryve.StatusWarned {
			return outcome, retries
		}

		if attempt < maxRetries {
			retries++
			delay := backoffDelay(baseDelay, attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return outcome, retries
			}
		}
	}
	return outcome, retries
}

// backoffDelay calculates exponential backoff with jitter.
func backoffDelay(base time.Duration, attempt int) time.Duration {
	delay := base
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	// Add ~15% jitter
	jitter := time.Duration(float64(delay) * 0.15)
	return delay + jitter
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/executor/... -v`
Expected: all step executor tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/executor/step.go internal/executor/step_test.go
git commit -m "feat(executor): add step executor with interpolation, capture, and retry"
```

---

### Task 14: Test Runner & Hooks

**Files:**
- Create: `internal/executor/hooks.go`, `internal/executor/runner.go`, `internal/executor/runner_test.go`

- [ ] **Step 1: Write test runner tests**

```go
// internal/executor/runner_test.go
package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestRunTest_SimplePass(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	rep := reporter.NewMulti()
	td := &tryve.TestDefinition{
		Name: "Simple Test",
		Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo pass"},
		}},
	}

	result := executor.RunTest(context.Background(), td, registry, rep, 0, 1000)
	if result.Status != tryve.StatusPassed {
		t.Errorf("status = %s, error = %v", result.Status, result.Error)
	}
}

func TestRunTest_SkippedTest(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	rep := reporter.NewMulti()
	td := &tryve.TestDefinition{
		Name: "Skipped", Skip: true, SkipReason: "not ready",
		Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo skip"},
		}},
	}

	result := executor.RunTest(context.Background(), td, registry, rep, 0, 1000)
	if result.Status != tryve.StatusSkipped {
		t.Errorf("status = %s, want skipped", result.Status)
	}
}

func TestRunTest_AllPhases(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	rep := reporter.NewMulti()
	td := &tryve.TestDefinition{
		Name: "Full Lifecycle",
		Setup: []tryve.StepDefinition{{
			ID: "setup-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo setup"},
		}},
		Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo execute"},
		}},
		Verify: []tryve.StepDefinition{{
			ID: "verify-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo verify"},
		}},
		Teardown: []tryve.StepDefinition{{
			ID: "teardown-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo teardown"},
		}},
	}

	result := executor.RunTest(context.Background(), td, registry, rep, 0, 1000)
	if result.Status != tryve.StatusPassed {
		t.Errorf("status = %s", result.Status)
	}
	// Should have 4 step outcomes (one per phase)
	if len(result.Steps) != 4 {
		t.Errorf("steps = %d, want 4", len(result.Steps))
	}
}

func TestRunTest_WithTimeout(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	rep := reporter.NewMulti()
	td := &tryve.TestDefinition{
		Name:    "Timeout Test",
		Timeout: 100, // 100ms
		Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "sleep 5"},
		}},
	}

	result := executor.RunTest(context.Background(), td, registry, rep, 0, 1000)
	if result.Status != tryve.StatusFailed {
		t.Errorf("status = %s, expected failed (timeout)", result.Status)
	}
}

func TestRunHook(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	os.WriteFile(script, []byte("#!/bin/sh\necho hook_ran"), 0755)

	err := executor.RunHook(context.Background(), script, dir, nil)
	if err != nil {
		t.Errorf("hook failed: %v", err)
	}
}

func TestRunHook_Failure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hook.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 1"), 0755)

	err := executor.RunHook(context.Background(), script, dir, nil)
	if err == nil {
		t.Error("expected error for failing hook")
	}
}
```

- [ ] **Step 2: Implement hooks executor**

```go
// internal/executor/hooks.go
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// RunHook executes a hook command. Returns error if the command fails (non-zero exit).
func RunHook(ctx context.Context, command, workDir string, env map[string]string) error {
	if command == "" {
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	if env != nil {
		cmdEnv := cmd.Environ()
		for k, v := range env {
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = cmdEnv
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook %q failed: %w\nstdout: %s\nstderr: %s",
			command, err, stdout.String(), stderr.String())
	}
	return nil
}
```

- [ ] **Step 3: Implement test runner**

```go
// internal/executor/runner.go
package executor

import (
	"context"
	"time"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

// RunTest executes a single test through all its phases.
func RunTest(
	ctx context.Context,
	td *tryve.TestDefinition,
	registry *adapter.Registry,
	rep reporter.Reporter,
	defaultRetries int,
	defaultRetryDelay int,
) *tryve.TestResult {
	result := &tryve.TestResult{Test: td}
	start := time.Now()

	// Handle skip
	if td.Skip {
		result.Status = tryve.StatusSkipped
		result.Duration = time.Since(start)
		return result
	}

	// Apply timeout
	if td.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(td.Timeout)*time.Millisecond)
		defer cancel()
	}

	rep.OnTestStart(ctx, td)

	// Create interpolation context with test variables
	interpCtx := tryve.NewInterpolationContext()
	if td.Variables != nil {
		for k, v := range td.Variables {
			interpCtx.Variables[k] = v
		}
	}

	// Determine retry settings
	retries := defaultRetries
	if td.Retries > 0 {
		retries = td.Retries
	}
	baseDelay := time.Duration(defaultRetryDelay) * time.Millisecond

	// Execute phases in order
	phases := []struct {
		name  tryve.TestPhase
		steps []tryve.StepDefinition
	}{
		{tryve.PhaseSetup, td.Setup},
		{tryve.PhaseExecute, td.Execute},
		{tryve.PhaseVerify, td.Verify},
		{tryve.PhaseTeardown, td.Teardown},
	}

	failed := false
	for _, phase := range phases {
		if len(phase.steps) == 0 {
			continue
		}

		// Skip verify/execute if a previous phase failed (but always run teardown)
		if failed && phase.name != tryve.PhaseTeardown {
			continue
		}

		for i := range phase.steps {
			step := &phase.steps[i]
			stepRetries := retries
			if step.Retry > 0 {
				stepRetries = step.Retry
			}

			outcome, retryCount := ExecuteStepWithRetry(ctx, step, registry, interpCtx, stepRetries, baseDelay)
			outcome.Phase = phase.name
			result.Steps = append(result.Steps, *outcome)
			result.RetryCount += retryCount

			rep.OnStepComplete(ctx, step, outcome)

			if outcome.Status == tryve.StatusFailed {
				failed = true
				result.Error = outcome.Error
				if phase.name == tryve.PhaseTeardown {
					continue // always try remaining teardown steps
				}
				break
			}
		}
	}

	if failed {
		result.Status = tryve.StatusFailed
	} else {
		result.Status = tryve.StatusPassed
	}
	result.Duration = time.Since(start)

	rep.OnTestComplete(ctx, td, result)
	return result
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/executor/... -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/executor/hooks.go internal/executor/runner.go internal/executor/runner_test.go
git commit -m "feat(executor): add test runner with phases, retries, and hooks"
```

---

### Task 15: Orchestrator

**Files:**
- Create: `internal/executor/orchestrator.go`, `internal/executor/orchestrator_test.go`

- [ ] **Step 1: Write orchestrator tests**

```go
// internal/executor/orchestrator_test.go
package executor_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
)

func TestOrchestrator_RunAll(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := &config.LoadedConfig{Defaults: config.DefaultsConfig{Parallel: 1, Timeout: 30000}}

	tests := []*tryve.TestDefinition{
		{Name: "Test 1", Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo test1"},
		}}},
		{Name: "Test 2", Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo test2"},
		}}},
	}

	orch := executor.NewOrchestrator(registry, rep, cfg)
	result := orch.Run(context.Background(), tests)

	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if result.Passed != 2 {
		t.Errorf("passed = %d, want 2", result.Passed)
	}
}

func TestOrchestrator_BailOnFailure(t *testing.T) {
	registry := newTestRegistry("http://localhost")
	var buf bytes.Buffer
	rep := reporter.NewConsole(&buf, false, false)
	cfg := &config.LoadedConfig{Defaults: config.DefaultsConfig{Parallel: 1, Timeout: 30000}}

	tests := []*tryve.TestDefinition{
		{Name: "Failing", Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "exit 1"},
			Assert: map[string]any{"exitCode": float64(0)},
		}}},
		{Name: "Should Skip", Execute: []tryve.StepDefinition{{
			ID: "execute-0", Adapter: "shell", Action: "exec",
			Params: map[string]any{"command": "echo skip"},
		}}},
	}

	orch := executor.NewOrchestrator(registry, rep, cfg)
	orch.SetBail(true)
	result := orch.Run(context.Background(), tests)

	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	// With bail, second test should not run
	if result.Total != 1 {
		t.Errorf("total = %d, want 1 (bail should stop)", result.Total)
	}
}

func TestOrchestrator_FilterByTag(t *testing.T) {
	tests := []*tryve.TestDefinition{
		{Name: "Tagged", Tags: []string{"smoke"}},
		{Name: "Untagged", Tags: []string{"integration"}},
	}

	filtered := executor.FilterTests(tests, executor.FilterOptions{Tags: []string{"smoke"}})
	if len(filtered) != 1 || filtered[0].Name != "Tagged" {
		t.Errorf("filtered = %v", filtered)
	}
}

func TestOrchestrator_FilterByGrep(t *testing.T) {
	tests := []*tryve.TestDefinition{
		{Name: "User API Test"},
		{Name: "Health Check"},
	}

	filtered := executor.FilterTests(tests, executor.FilterOptions{Grep: "API"})
	if len(filtered) != 1 || filtered[0].Name != "User API Test" {
		t.Errorf("filtered = %v", filtered)
	}
}
```

- [ ] **Step 2: Implement orchestrator**

```go
// internal/executor/orchestrator.go
package executor

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/liemle3893/go-tryve/internal/tryve"
	"golang.org/x/sync/errgroup"
)

// Orchestrator coordinates the execution of all tests.
type Orchestrator struct {
	registry *adapter.Registry
	reporter reporter.Reporter
	config   *config.LoadedConfig
	bail     bool
}

// NewOrchestrator creates a new test orchestrator.
func NewOrchestrator(registry *adapter.Registry, rep reporter.Reporter, cfg *config.LoadedConfig) *Orchestrator {
	return &Orchestrator{registry: registry, reporter: rep, config: cfg}
}

// SetBail enables bail-on-first-failure mode.
func (o *Orchestrator) SetBail(bail bool) { o.bail = bail }

// Run executes all tests and returns the suite result.
func (o *Orchestrator) Run(ctx context.Context, tests []*tryve.TestDefinition) *tryve.SuiteResult {
	start := time.Now()
	result := &tryve.SuiteResult{Total: len(tests)}
	o.reporter.OnSuiteStart(ctx, nil)

	// Run hooks
	if o.config.Hooks.BeforeAll != "" {
		if err := RunHook(ctx, o.config.Hooks.BeforeAll, "", nil); err != nil {
			result.Duration = time.Since(start)
			o.reporter.OnSuiteComplete(ctx, nil, result)
			return result
		}
	}

	// Sort by dependencies (simple: tests with depends go after their deps)
	sorted := topoSortTests(tests)

	// Execute
	parallel := o.config.Defaults.Parallel
	if parallel <= 0 {
		parallel = 1
	}

	var mu sync.Mutex
	var bailed bool
	completed := make(map[string]*tryve.TestResult)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(parallel)

	for _, td := range sorted {
		td := td
		g.Go(func() error {
			// Check bail
			mu.Lock()
			if bailed {
				mu.Unlock()
				return nil
			}
			mu.Unlock()

			// Wait for dependencies
			if len(td.Depends) > 0 {
				for _, dep := range td.Depends {
					// Simple spin-wait for dependency (could use channels for better perf)
					for {
						mu.Lock()
						depResult, ok := completed[dep]
						mu.Unlock()
						if ok {
							if depResult.Status == tryve.StatusFailed {
								tr := &tryve.TestResult{Test: td, Status: tryve.StatusSkipped}
								mu.Lock()
								result.Skipped++
								completed[td.Name] = tr
								mu.Unlock()
								o.reporter.OnTestComplete(gctx, td, tr)
								return nil
							}
							break
						}
						select {
						case <-time.After(50 * time.Millisecond):
						case <-gctx.Done():
							return gctx.Err()
						}
					}
				}
			}

			// Run hooks
			if o.config.Hooks.BeforeEach != "" {
				RunHook(gctx, o.config.Hooks.BeforeEach, "", nil)
			}

			tr := RunTest(gctx, td, o.registry, o.reporter,
				o.config.Defaults.Retries, o.config.Defaults.RetryDelay)

			if o.config.Hooks.AfterEach != "" {
				RunHook(gctx, o.config.Hooks.AfterEach, "", nil)
			}

			mu.Lock()
			completed[td.Name] = tr
			switch tr.Status {
			case tryve.StatusPassed:
				result.Passed++
			case tryve.StatusFailed:
				result.Failed++
				if o.bail {
					bailed = true
				}
			case tryve.StatusSkipped:
				result.Skipped++
			}
			result.Tests = append(result.Tests, *tr)
			mu.Unlock()

			return nil
		})
	}

	g.Wait()

	if o.config.Hooks.AfterAll != "" {
		RunHook(ctx, o.config.Hooks.AfterAll, "", nil)
	}

	result.Total = result.Passed + result.Failed + result.Skipped
	result.Duration = time.Since(start)
	o.reporter.OnSuiteComplete(ctx, nil, result)
	o.reporter.Flush()
	return result
}

// FilterOptions controls which tests are selected for execution.
type FilterOptions struct {
	Tags     []string
	Grep     string
	Priority string
}

// FilterTests returns tests matching the given filter criteria.
func FilterTests(tests []*tryve.TestDefinition, opts FilterOptions) []*tryve.TestDefinition {
	var result []*tryve.TestDefinition
	for _, td := range tests {
		if !matchesFilter(td, opts) {
			continue
		}
		result = append(result, td)
	}
	return result
}

func matchesFilter(td *tryve.TestDefinition, opts FilterOptions) bool {
	// Tag filter
	if len(opts.Tags) > 0 {
		found := false
		for _, filterTag := range opts.Tags {
			for _, testTag := range td.Tags {
				if testTag == filterTag {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Grep filter
	if opts.Grep != "" {
		re, err := regexp.Compile(opts.Grep)
		if err != nil {
			// Fall back to simple contains
			if !strings.Contains(td.Name, opts.Grep) {
				return false
			}
		} else if !re.MatchString(td.Name) {
			return false
		}
	}

	// Priority filter
	if opts.Priority != "" && string(td.Priority) != opts.Priority {
		return false
	}

	return true
}

// topoSortTests orders tests so that dependencies come first.
// Tests without depends are left in original order.
func topoSortTests(tests []*tryve.TestDefinition) []*tryve.TestDefinition {
	byName := make(map[string]*tryve.TestDefinition)
	for _, td := range tests {
		byName[td.Name] = td
	}

	visited := make(map[string]bool)
	var sorted []*tryve.TestDefinition

	var visit func(td *tryve.TestDefinition)
	visit = func(td *tryve.TestDefinition) {
		if visited[td.Name] {
			return
		}
		visited[td.Name] = true
		for _, dep := range td.Depends {
			if depTd, ok := byName[dep]; ok {
				visit(depTd)
			}
		}
		sorted = append(sorted, td)
	}

	for _, td := range tests {
		visit(td)
	}
	return sorted
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/executor/... -v`
Expected: all orchestrator tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/executor/orchestrator.go internal/executor/orchestrator_test.go
git commit -m "feat(executor): add orchestrator with parallel execution and bail"
```

---

### Task 16: CLI Commands

**Files:**
- Create: `internal/cli/root.go`, `internal/cli/run.go`, `internal/cli/validate.go`, `internal/cli/list.go`, `internal/cli/health.go`, `internal/cli/init_cmd.go`, `internal/cli/version.go`, `internal/cli/test_cmd.go`
- Modify: `cmd/tryve/main.go`

- [ ] **Step 1: Implement CLI root and commands**

```go
// internal/cli/root.go
package cli

import (
	"github.com/spf13/cobra"
)

// NewRoot creates the root cobra command with all subcommands.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "tryve",
		Short: "tryve — YAML-driven multi-protocol test runner",
		Long:  "tryve runs YAML-defined tests against HTTP APIs, databases, message queues, and shell commands.",
		SilenceUsage: true,
	}

	// Global flags
	root.PersistentFlags().StringP("config", "c", "e2e.config.yaml", "config file path")
	root.PersistentFlags().StringP("env", "e", "local", "environment name")

	// Register commands
	root.AddCommand(
		newRunCmd(),
		newValidateCmd(),
		newListCmd(),
		newHealthCmd(),
		newInitCmd(),
		newVersionCmd(version),
		newTestCmd(),
	)

	return root
}
```

```go
// internal/cli/run.go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/loader"
	"github.com/liemle3893/go-tryve/internal/reporter"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute tests",
		RunE:  runTests,
	}

	cmd.Flags().StringP("test-dir", "d", ".", "test directory")
	cmd.Flags().IntP("parallel", "p", 0, "parallel execution (0 = use config default)")
	cmd.Flags().IntP("timeout", "t", 0, "test timeout ms (0 = use config default)")
	cmd.Flags().IntP("retries", "r", 0, "retry count (0 = use config default)")
	cmd.Flags().Bool("bail", false, "stop on first failure")
	cmd.Flags().StringP("grep", "g", "", "filter tests by name pattern")
	cmd.Flags().StringSlice("tag", nil, "filter by tags")
	cmd.Flags().String("priority", "", "filter by priority (P0-P3)")
	cmd.Flags().Bool("dry-run", false, "show what would run without executing")
	cmd.Flags().Bool("skip-setup", false, "skip setup phase")
	cmd.Flags().Bool("skip-teardown", false, "skip teardown phase")
	cmd.Flags().StringSlice("reporter", nil, "reporter types (console,junit,html,json)")
	cmd.Flags().StringP("output", "o", "", "report output path")
	cmd.Flags().Bool("verbose", false, "verbose output")

	return cmd
}

func runTests(cmd *cobra.Command, args []string) error {
	// Set up signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load config
	cfgPath, _ := cmd.Flags().GetString("config")
	envName, _ := cmd.Flags().GetString("env")
	cfg, err := config.Load(cfgPath, envName)
	if err != nil {
		return err
	}

	// Apply CLI overrides
	if v, _ := cmd.Flags().GetInt("parallel"); v > 0 {
		cfg.Defaults.Parallel = v
	}
	if v, _ := cmd.Flags().GetInt("timeout"); v > 0 {
		cfg.Defaults.Timeout = v
	}
	if v, _ := cmd.Flags().GetInt("retries"); v > 0 {
		cfg.Defaults.Retries = v
	}

	// Discover tests
	testDir, _ := cmd.Flags().GetString("test-dir")
	files, err := loader.Discover(testDir)
	if err != nil {
		return fmt.Errorf("discover tests: %w", err)
	}

	var tests []*tryve.TestDefinition
	for _, f := range files {
		td, err := loader.ParseFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", f, err)
			continue
		}
		if errs := loader.Validate(td); len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "warning: %s: %v\n", f, e)
			}
			continue
		}
		tests = append(tests, td)
	}

	// Apply filters
	grep, _ := cmd.Flags().GetString("grep")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	priority, _ := cmd.Flags().GetString("priority")
	tests = executor.FilterTests(tests, executor.FilterOptions{
		Grep: grep, Tags: tags, Priority: priority,
	})

	if len(tests) == 0 {
		fmt.Println("No tests found.")
		return nil
	}

	// Dry run
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Printf("Would run %d tests:\n", len(tests))
		for _, td := range tests {
			fmt.Printf("  - %s [%s] %v\n", td.Name, td.Priority, td.Tags)
		}
		return nil
	}

	// Skip phases
	skipSetup, _ := cmd.Flags().GetBool("skip-setup")
	skipTeardown, _ := cmd.Flags().GetBool("skip-teardown")
	if skipSetup || skipTeardown {
		for _, td := range tests {
			if skipSetup {
				td.Setup = nil
			}
			if skipTeardown {
				td.Teardown = nil
			}
		}
	}

	// Create reporter
	verbose, _ := cmd.Flags().GetBool("verbose")
	rep := reporter.NewConsoleFromEnv(verbose)

	// Create adapter registry
	registry := adapter.NewRegistry()
	if cfg.Environment.BaseURL != "" {
		registry.Register("http", adapter.NewHTTPAdapter(cfg.Environment.BaseURL))
	}
	registry.Register("shell", adapter.NewShellAdapter(nil))
	defer registry.CloseAll(ctx)

	// Run
	bail, _ := cmd.Flags().GetBool("bail")
	orch := executor.NewOrchestrator(registry, rep, cfg)
	orch.SetBail(bail)
	result := orch.Run(ctx, tests)

	if result.Failed > 0 {
		os.Exit(1)
	}
	return nil
}
```

```go
// internal/cli/validate.go
package cli

import (
	"fmt"
	"os"

	"github.com/liemle3893/go-tryve/internal/loader"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate test file syntax",
		RunE: func(cmd *cobra.Command, args []string) error {
			testDir, _ := cmd.Flags().GetString("test-dir")
			if testDir == "" {
				testDir = "."
			}
			files, err := loader.Discover(testDir)
			if err != nil {
				return err
			}

			hasErrors := false
			for _, f := range files {
				td, err := loader.ParseFile(f)
				if err != nil {
					fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", f, err)
					hasErrors = true
					continue
				}
				errs := loader.Validate(td)
				if len(errs) > 0 {
					for _, e := range errs {
						fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", f, e)
					}
					hasErrors = true
				} else {
					fmt.Printf("OK   %s\n", f)
				}
			}
			if hasErrors {
				os.Exit(1)
			}
			fmt.Printf("\nAll %d test files valid.\n", len(files))
			return nil
		},
	}
	cmd.Flags().StringP("test-dir", "d", ".", "test directory")
	return cmd
}
```

```go
// internal/cli/list.go
package cli

import (
	"fmt"
	"os"

	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/loader"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			testDir, _ := cmd.Flags().GetString("test-dir")
			if testDir == "" {
				testDir = "."
			}
			files, _ := loader.Discover(testDir)

			var tests []*tryve.TestDefinition
			for _, f := range files {
				td, err := loader.ParseFile(f)
				if err != nil {
					continue
				}
				tests = append(tests, td)
			}

			// Apply filters
			grep, _ := cmd.Flags().GetString("grep")
			tags, _ := cmd.Flags().GetStringSlice("tag")
			priority, _ := cmd.Flags().GetString("priority")
			tests = executor.FilterTests(tests, executor.FilterOptions{
				Grep: grep, Tags: tags, Priority: priority,
			})

			for _, td := range tests {
				priority := string(td.Priority)
				if priority == "" {
					priority = "-"
				}
				fmt.Printf("  [%s] %s  %v  (%s)\n", priority, td.Name, td.Tags, td.SourceFile)
			}
			fmt.Fprintf(os.Stdout, "\n%d tests found.\n", len(tests))
			return nil
		},
	}
	cmd.Flags().StringP("test-dir", "d", ".", "test directory")
	cmd.Flags().StringP("grep", "g", "", "filter by name")
	cmd.Flags().StringSlice("tag", nil, "filter by tags")
	cmd.Flags().String("priority", "", "filter by priority")
	return cmd
}
```

```go
// internal/cli/health.go
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check adapter connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			envName, _ := cmd.Flags().GetString("env")
			cfg, err := config.Load(cfgPath, envName)
			if err != nil {
				return err
			}

			registry := adapter.NewRegistry()
			if cfg.Environment.BaseURL != "" {
				registry.Register("http", adapter.NewHTTPAdapter(cfg.Environment.BaseURL))
			}
			registry.Register("shell", adapter.NewShellAdapter(nil))

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			for _, name := range registry.Names() {
				a, err := registry.Get(ctx, name)
				if err != nil {
					fmt.Printf("  FAIL %s: %v\n", name, err)
					continue
				}
				if err := a.Health(ctx); err != nil {
					fmt.Printf("  FAIL %s: %v\n", name, err)
				} else {
					fmt.Printf("  OK   %s\n", name)
				}
			}
			registry.CloseAll(ctx)
			return nil
		},
	}
}
```

```go
// internal/cli/init_cmd.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize e2e.config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat("e2e.config.yaml"); err == nil {
				return fmt.Errorf("e2e.config.yaml already exists")
			}
			template := `version: "1.0"

environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters: {}

defaults:
  timeout: 30000
  retries: 0
  retryDelay: 1000
  parallel: 1

variables: {}

hooks: {}

reporters:
  - type: console
`
			if err := os.WriteFile("e2e.config.yaml", []byte(template), 0644); err != nil {
				return err
			}
			fmt.Println("Created e2e.config.yaml")
			return nil
		},
	}
}
```

```go
// internal/cli/version.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tryve %s\n", version)
		},
	}
}
```

```go
// internal/cli/test_cmd.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test file management",
	}

	createCmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new test file from template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			filename := name + ".test.yaml"
			if _, err := os.Stat(filename); err == nil {
				return fmt.Errorf("%s already exists", filename)
			}
			template := fmt.Sprintf(`name: "%s"
description: ""
tags: []
priority: P2

execute:
  - adapter: http
    action: request
    url: /
    method: GET
    assert:
      status: 200
`, name)
			if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
				return err
			}
			fmt.Printf("Created %s\n", filename)
			return nil
		},
	}

	listTemplatesCmd := &cobra.Command{
		Use:   "list-templates",
		Short: "List available test templates",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available templates:")
			fmt.Println("  api             - Basic API test")
			fmt.Println("  crud            - CRUD lifecycle test")
			fmt.Println("  integration     - Multi-adapter integration test")
			fmt.Println("  event-driven    - Event/message queue test")
			fmt.Println("  db-verification - Database verification test")
		},
	}

	testCmd.AddCommand(createCmd, listTemplatesCmd)
	return testCmd
}
```

- [ ] **Step 2: Wire CLI into main.go**

```go
// cmd/tryve/main.go
package main

import (
	"fmt"
	"os"

	"github.com/liemle3893/go-tryve/internal/cli"
)

var version = "dev"

func main() {
	root := cli.NewRoot(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Fix the list.go import** (needs tryve import)

Add missing import to `internal/cli/list.go`:
```go
import (
	"github.com/liemle3893/go-tryve/internal/tryve"
	// ... other imports
)
```

- [ ] **Step 4: Build and verify**

Run: `go build -o bin/tryve ./cmd/tryve && ./bin/tryve version`
Expected: `tryve dev`

Run: `./bin/tryve --help`
Expected: Shows all commands (run, validate, list, health, init, test, version)

- [ ] **Step 5: Commit**

```bash
git add internal/cli/ cmd/tryve/main.go
git commit -m "feat(cli): add all CLI commands (run, validate, list, health, init, test, version)"
```

---

### Task 17: Integration Test & Build Verification

**Files:**
- Create: `tests/integration/tryve_test.go`

- [ ] **Step 1: Write integration test**

```go
// tests/integration/tryve_test.go
package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/go-tryve/internal/adapter"
	"github.com/liemle3893/go-tryve/internal/config"
	"github.com/liemle3893/go-tryve/internal/executor"
	"github.com/liemle3893/go-tryve/internal/loader"
	"github.com/liemle3893/go-tryve/internal/reporter"
)

func TestIntegration_FullHTTPFlow(t *testing.T) {
	// Start test HTTP server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case "/api/users":
			if r.Method == "POST" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "test-user"})
			}
		case "/api/users/42":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "test-user"})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	// Write config
	dir := t.TempDir()
	cfgContent := `
version: "1.0"
environments:
  test:
    baseUrl: "` + srv.URL + `"
defaults:
  timeout: 10000
  parallel: 1
`
	os.WriteFile(filepath.Join(dir, "e2e.config.yaml"), []byte(cfgContent), 0644)

	// Write test file
	testContent := `
name: "Integration: User CRUD"
tags: [integration, api]
priority: P0

variables:
  userName: "test-user"

execute:
  - adapter: http
    action: request
    url: /api/health
    method: GET
    assert:
      status: 200
      json:
        - path: "$.body.status"
          equals: "ok"

  - adapter: http
    action: request
    url: /api/users
    method: POST
    body:
      name: "{{userName}}"
    capture:
      userId: "$.body.id"
    assert:
      status: 201
      json:
        - path: "$.body.name"
          equals: "test-user"

  - adapter: http
    action: request
    url: "/api/users/{{captured.userId}}"
    method: GET
    assert:
      status: 200
      json:
        - path: "$.body.id"
          equals: 42
`
	os.WriteFile(filepath.Join(dir, "user-crud.test.yaml"), []byte(testContent), 0644)

	// Load config
	cfg, err := config.Load(filepath.Join(dir, "e2e.config.yaml"), "test")
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	// Discover and parse
	files, _ := loader.Discover(dir)
	if len(files) != 1 {
		t.Fatalf("found %d files, want 1", len(files))
	}

	td, err := loader.ParseFile(files[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if errs := loader.Validate(td); len(errs) > 0 {
		t.Fatalf("validate: %v", errs)
	}

	// Set up registry and reporter
	registry := adapter.NewRegistry()
	registry.Register("http", adapter.NewHTTPAdapter(cfg.Environment.BaseURL))
	registry.Register("shell", adapter.NewShellAdapter(nil))
	defer registry.CloseAll(context.Background())

	rep := reporter.NewMulti()

	// Run
	orch := executor.NewOrchestrator(registry, rep, cfg)
	result := orch.Run(context.Background(), []*tryve.TestDefinition{td})

	if result.Failed > 0 {
		for _, tr := range result.Tests {
			if tr.Status == tryve.StatusFailed {
				t.Errorf("test %q failed: %v", tr.Test.Name, tr.Error)
				for _, so := range tr.Steps {
					if so.Status == tryve.StatusFailed {
						t.Errorf("  step %q: %v", so.Step.ID, so.Error)
						for _, a := range so.Assertions {
							if !a.Passed {
								t.Errorf("    %s %s: %s", a.Path, a.Operator, a.Message)
							}
						}
					}
				}
			}
		}
	}
	if result.Passed != 1 {
		t.Errorf("passed = %d, want 1", result.Passed)
	}
}
```

- [ ] **Step 2: Fix missing import in integration test**

Add the tryve import:
```go
import (
	"github.com/liemle3893/go-tryve/internal/tryve"
	// ... other imports
)
```

- [ ] **Step 3: Run integration test**

Run: `go test ./tests/integration/... -v`
Expected: Integration test passes — full HTTP flow with variable interpolation, capture, and assertions working end-to-end.

- [ ] **Step 4: Run full test suite**

Run: `go test ./... -v`
Expected: All unit and integration tests pass.

- [ ] **Step 5: Build final binary**

Run: `make build && ./bin/tryve version`
Expected: `tryve dev`

Run: `./bin/tryve --help`
Expected: All commands listed.

- [ ] **Step 6: Commit**

```bash
git add tests/integration/ Makefile
git commit -m "test: add integration test for full HTTP flow"
```

- [ ] **Step 7: Run go mod tidy and final commit**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: tidy go modules"
```
