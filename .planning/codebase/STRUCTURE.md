# Codebase Structure

**Analysis Date:** 2026-03-02

## Directory Layout

```
e2e-runner/
├── bin/
│   └── e2e.js                    # CLI shell entry point (requires dist/index.js)
├── src/
│   ├── index.ts                  # Main entry: CLI main() + programmatic API exports
│   ├── types.ts                  # All TypeScript interfaces and type definitions
│   ├── errors.ts                 # Custom error class hierarchy
│   ├── cli/
│   │   ├── index.ts              # Arg parser, help text, printError
│   │   ├── run.command.ts        # `e2e run` — full test execution pipeline
│   │   ├── validate.command.ts   # `e2e validate` — syntax/schema check
│   │   ├── list.command.ts       # `e2e list` — enumerate discovered tests
│   │   ├── health.command.ts     # `e2e health` — adapter connectivity check
│   │   ├── init.command.ts       # `e2e init` — create e2e.config.yaml
│   │   ├── test.command.ts       # `e2e test create` — generate test from template
│   │   └── init-templates.ts     # Config JSON schema + YAML template strings
│   ├── core/
│   │   ├── index.ts              # Re-exports all core public API
│   │   ├── config-loader.ts      # Load/validate/resolve e2e.config.yaml
│   │   ├── test-discovery.ts     # Walk filesystem, find *.test.yaml / *.test.ts
│   │   ├── yaml-loader.ts        # Parse YAML test files → UnifiedTestDefinition
│   │   ├── ts-loader.ts          # Load TypeScript test files → UnifiedTestDefinition
│   │   ├── context-factory.ts    # Create per-test TestContext with scoped variables
│   │   ├── variable-interpolator.ts  # {{...}} template engine + built-in functions
│   │   ├── step-executor.ts      # Execute single step: interpolate → dispatch → assert
│   │   └── test-orchestrator.ts  # Run suite/test lifecycle, parallelism, event emission
│   ├── adapters/
│   │   ├── index.ts              # Re-exports AdapterRegistry + factory functions
│   │   ├── base.adapter.ts       # Abstract BaseAdapter class
│   │   ├── adapter-registry.ts   # Lifecycle manager; maps AdapterType → BaseAdapter
│   │   ├── http.adapter.ts       # REST HTTP via native fetch
│   │   ├── postgresql.adapter.ts # PostgreSQL via 'pg'
│   │   ├── mongodb.adapter.ts    # MongoDB via 'mongodb'
│   │   ├── redis.adapter.ts      # Redis via 'ioredis'
│   │   └── eventhub.adapter.ts   # Azure EventHub via '@azure/event-hubs'
│   ├── assertions/
│   │   ├── index.ts              # Re-exports assertion public API
│   │   ├── assertion-runner.ts   # Core runAssertion() with all operators
│   │   ├── matchers.ts           # toBe, toEqual, toContain, etc. matcher implementations
│   │   ├── jsonpath.ts           # JSONPath ($.body.data[0].id) evaluation
│   │   └── expect.ts             # Fluent expect() API
│   ├── reporters/
│   │   ├── index.ts              # ReporterManager + createReporter factory
│   │   ├── base.reporter.ts      # Abstract BaseReporter with event hooks
│   │   ├── console.reporter.ts   # Colored terminal output
│   │   ├── junit.reporter.ts     # JUnit XML for CI/CD
│   │   ├── html.reporter.ts      # Self-contained interactive HTML report
│   │   └── json.reporter.ts      # Machine-readable JSON output
│   └── utils/
│       ├── logger.ts             # createLogger() — leveled, colored, injectable logger
│       ├── retry.ts              # withRetry(), withTimeout(), pollUntil(), sleep()
│       └── exit-codes.ts         # EXIT_CODES constants + errorCodeToExitCode()
├── tests/
│   └── e2e/
│       └── adapters/             # Example/integration test YAML files
│           ├── TC-HTTP-ASSERTIONS-001.test.yaml
│           ├── TC-POSTGRES-001.test.yaml
│           ├── TC-MONGODB-001.test.yaml
│           ├── TC-REDIS-001.test.yaml
│           ├── TC-EVENTHUB-001.test.yaml
│           ├── TC-INTEGRATION-001.test.yaml
│           └── TC-MONGODB-FINDONE-FILTER.test.yaml
├── dist/                         # Compiled output (generated, not committed)
├── reports/                      # Test report output directory (generated)
├── config/                       # Additional config examples
├── docs/                         # Documentation markdown files
├── demo-server/                  # Standalone Express demo server for testing
│   └── src/
│       ├── routes/               # Express route handlers
│       └── services/             # Business logic services
├── e2e.config.yaml               # Project-level E2E config (example/default)
├── tsconfig.json                 # TypeScript compiler config
├── package.json                  # NPM manifest, scripts, peer deps
└── docker-compose.yaml           # Local infra: PostgreSQL, MongoDB, Redis, EventHub
```

## Directory Purposes

**`src/cli/`:**
- Purpose: Command-line interface — argument parsing and command routing
- Contains: One file per CLI command; `index.ts` holds shared CLI utilities (arg parser, help, error printer)
- Key files: `run.command.ts` (most complex — full test execution pipeline), `init-templates.ts` (CONFIG_SCHEMA used by config-loader)

**`src/core/`:**
- Purpose: Test execution engine — everything between receiving CLI args and getting results
- Contains: Config loading, file discovery, test loading (YAML+TS), execution context, variable interpolation, step execution, orchestration
- Key files: `test-orchestrator.ts` (central coordinator), `step-executor.ts` (per-step logic), `config-loader.ts` (YAML + env var resolution)

**`src/adapters/`:**
- Purpose: Service connectors — uniform interface for HTTP, PostgreSQL, MongoDB, Redis, EventHub
- Contains: Abstract base + registry + one file per adapter type
- Key files: `adapter-registry.ts` (lazy initialization of only required adapters), `http.adapter.ts` (default adapter, always initialized)

**`src/assertions/`:**
- Purpose: Validation engine — evaluate assertion expressions against data
- Contains: Core runner, matchers, JSONPath evaluator, fluent expect API
- Key files: `assertion-runner.ts` (used directly by adapters), `jsonpath.ts` (path extraction from nested response data)

**`src/reporters/`:**
- Purpose: Output formatters — transform test results into human/machine-readable formats
- Contains: Abstract base + four reporter implementations + `ReporterManager` in `index.ts`
- Key files: `index.ts` (ReporterManager fan-outs events to all configured reporters)

**`src/utils/`:**
- Purpose: Shared utilities with no domain logic
- Contains: Logger, retry/timeout helpers, exit code mapping
- Key files: `logger.ts` (injected as `Logger` interface throughout; not a singleton), `retry.ts` (exponential backoff + jitter)

**`tests/e2e/adapters/`:**
- Purpose: Integration/example test files exercising each adapter
- Contains: YAML test files named with `TC-<ADAPTER>-<NNN>` convention
- Generated: No (hand-authored); Committed: Yes

**`dist/`:**
- Purpose: TypeScript compiled output
- Generated: Yes (via `npm run build`); Committed: No (`dist/` in `.gitignore`)

**`demo-server/`:**
- Purpose: Standalone Express.js server used as test target for HTTP adapter tests
- Generated: No; Committed: Yes

## Key File Locations

**Entry Points:**
- `bin/e2e.js`: Shell entry point — invoked by `./bin/e2e.js` or the `e2e` npm bin
- `src/index.ts`: TypeScript main — both CLI `main()` and programmatic exports (`runTests`, `validateTests`, `listTests`, `checkHealth`)

**Configuration:**
- `e2e.config.yaml`: Runtime configuration for tests (environments, adapters, defaults, reporters)
- `tsconfig.json`: TypeScript compiler settings
- `package.json`: Dependencies, scripts (`build`, `clean`, `prepublishOnly`), bin field

**Core Logic:**
- `src/core/test-orchestrator.ts`: Test lifecycle, parallelism via `p-limit`, event emission
- `src/core/step-executor.ts`: Per-step dispatch, variable interpolation, retry logic
- `src/core/config-loader.ts`: YAML config loading, `${ENV_VAR}` resolution, schema validation
- `src/types.ts`: All shared types — read this first when understanding the data model

**Adapter Contract:**
- `src/adapters/base.adapter.ts`: Abstract class all adapters must extend
- `src/adapters/adapter-registry.ts`: Registry that initializes, connects, and provides adapters

**Assertions:**
- `src/assertions/assertion-runner.ts`: `runAssertion(value, assertion, path?)` — called by adapters
- `src/assertions/jsonpath.ts`: JSONPath extraction for response data navigation

**Testing:**
- `tests/e2e/adapters/`: YAML-format integration tests for each adapter type

## Naming Conventions

**Files:**
- Adapter files: `<name>.adapter.ts` (e.g., `http.adapter.ts`, `postgresql.adapter.ts`)
- Reporter files: `<name>.reporter.ts` (e.g., `console.reporter.ts`, `junit.reporter.ts`)
- Command files: `<name>.command.ts` (e.g., `run.command.ts`, `health.command.ts`)
- Core utility files: `<noun>-<noun>.ts` (e.g., `test-orchestrator.ts`, `variable-interpolator.ts`, `context-factory.ts`)
- Index files: `index.ts` for barrel re-exports at each layer boundary

**Test files:**
- YAML: `<name>.test.yaml` (discovery pattern `**/*.test.yaml`)
- TypeScript: `<name>.test.ts` (discovery pattern `**/*.test.ts`)
- Integration tests use `TC-<ADAPTER>-<NNN>` naming (e.g., `TC-HTTP-ASSERTIONS-001.test.yaml`)

**Directories:**
- Kebab-case: `src/core/`, `src/adapters/`, `src/reporters/`
- Tests mirror adapter categories: `tests/e2e/adapters/`

**TypeScript:**
- Interfaces: PascalCase (e.g., `UnifiedTestDefinition`, `AdapterContext`, `LoadedConfig`)
- Classes: PascalCase (e.g., `TestOrchestrator`, `AdapterRegistry`, `BaseReporter`)
- Functions: camelCase (e.g., `createOrchestrator()`, `loadConfig()`, `discoverTests()`)
- Constants: SCREAMING_SNAKE_CASE (e.g., `EXIT_CODES`, `BUILT_IN_FUNCTIONS`, `DEFAULT_CONFIG`)
- Types: PascalCase (e.g., `AdapterType`, `TestStatus`, `CLICommand`)
- Error codes: string literals in SCREAMING_SNAKE_CASE (e.g., `'CONFIG_ERROR'`, `'ADAPTER_ERROR'`)

## Where to Add New Code

**New Adapter (e.g., MySQL):**
- Implementation: `src/adapters/mysql.adapter.ts` — extend `BaseAdapter`
- Register: Add to `src/adapters/adapter-registry.ts` `initializeAdapters()` method
- Export: Add to `src/adapters/index.ts`
- Add type: Add `'mysql'` to `AdapterType` union in `src/types.ts`
- Add config interface: Add `MySQLAdapterConfig` to `src/types.ts` and `EnvironmentConfig.adapters`

**New CLI Command:**
- Implementation: `src/cli/<name>.command.ts` — export `<name>Command(args: CLIArgs)` returning `{ exitCode }`
- Register: Add case to `routeCommand()` in `src/index.ts` and add to `CLICommand` type in `src/types.ts`
- Add help: Update `printHelp()` in `src/cli/index.ts`

**New Reporter:**
- Implementation: `src/reporters/<name>.reporter.ts` — extend `BaseReporter`
- Register: Add to `createReporter()` factory in `src/reporters/index.ts`
- Add type: Add to `ReporterConfig.type` union in `src/types.ts`

**New Built-in Variable Function:**
- Location: `src/core/variable-interpolator.ts` in `BUILT_IN_FUNCTIONS` registry
- Pattern: `$funcName: (...args: string[]) => string | number`
- Usage in tests: `{{$funcName(arg1, arg2)}}`

**New Utility:**
- Location: `src/utils/<name>.ts` if broadly applicable
- Must: Export typed functions with JSDoc; no side effects at module load time

**New Test Files:**
- YAML tests: Place anywhere under `testDir` (default `.`) as `<name>.test.yaml`
- TypeScript tests: Place anywhere under `testDir` as `<name>.test.ts`
- Integration tests for this repo: `tests/e2e/adapters/TC-<ADAPTER>-<NNN>.test.yaml`

## Special Directories

**`dist/`:**
- Purpose: Compiled JavaScript output from TypeScript source
- Generated: Yes (via `npm run build` → `tsc`)
- Committed: No

**`reports/`:**
- Purpose: Default output directory for test reports (JUnit XML, HTML, JSON)
- Generated: Yes (at runtime)
- Committed: No (empty placeholder only)

**`.planning/`:**
- Purpose: GSD planning documents for Claude Code workflows
- Generated: No (hand-maintained)
- Committed: Yes

**`demo-server/`:**
- Purpose: Self-contained Express API used as the test target for HTTP adapter integration tests
- Has own `package.json` and `node_modules`
- Generated: No; Committed: Yes

---

*Structure analysis: 2026-03-02*
