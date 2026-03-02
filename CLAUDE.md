# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
npm run build          # Compile TypeScript to dist/
npm run clean          # Remove dist/
npm run prepublishOnly # Clean + build (for publishing)
npm run local:install  # Local installation
```

## CLI Commands

```bash
# Run tests
./bin/e2e.js run                          # Run all tests
./bin/e2e.js run --tag smoke --bail       # Filter by tag, stop on failure

# Other commands
./bin/e2e.js validate                     # Validate test file syntax
./bin/e2e.js list                         # List discovered tests
./bin/e2e.js health                       # Check adapter connectivity
./bin/e2e.js init                         # Initialize e2e.config.yaml
./bin/e2e.js test create <name>           # Create test from template

# Documentation
./bin/e2e.js doc                          # List documentation sections
./bin/e2e.js doc assertions               # Show assertions reference
./bin/e2e.js doc adapters.http            # Show HTTP adapter docs

# Skills
./bin/e2e.js install --skills             # Install Claude Code skills to project
```

## Architecture

### Directory Structure

```
src/
├── index.ts              # CLI entry point + programmatic API exports
├── types.ts              # All TypeScript interfaces and types
├── errors.ts             # Custom error classes (E2ERunnerError, AdapterError, etc.)
├── cli/                  # Command handlers (run, validate, list, health, init, test)
├── core/                 # Test execution engine
│   ├── test-orchestrator.ts  # Manages test lifecycle: setup → execute → verify → teardown
│   ├── step-executor.ts      # Executes individual steps with retry logic
│   ├── context-factory.ts    # Creates test execution contexts
│   ├── variable-interpolator.ts  # Handles ${...} variable substitution
│   └── test-discovery.ts     # Finds *.test.yaml and *.test.ts files
├── adapters/             # Database/service connectors
│   ├── base.adapter.ts       # Abstract BaseAdapter class
│   ├── http.adapter.ts       # REST API testing (fetch-based)
│   ├── postgresql.adapter.ts # PostgreSQL via 'pg'
│   ├── mongodb.adapter.ts    # MongoDB via 'mongodb'
│   ├── redis.adapter.ts      # Redis via 'ioredis'
│   └── eventhub.adapter.ts   # Azure EventHub
├── assertions/           # Test assertion system
│   ├── matchers.ts           # Matcher implementations (toBe, toEqual, toContain, etc.)
│   ├── assertion-runner.ts   # Runs assertions with operator dispatch
│   └── jsonpath.ts           # JSONPath evaluation ($.body.data[0].id)
├── reporters/            # Output formatters
│   ├── console.reporter.ts   # Terminal output with colors
│   ├── junit.reporter.ts     # XML for CI/CD
│   ├── html.reporter.ts      # Interactive HTML reports
│   └── json.reporter.ts      # Machine-readable JSON
└── utils/                # Helpers (logger, retry, exit-codes)
```

## Documentation Sync Rule

Every change to CLI commands, adapters, configuration, assertions, built-in functions, or YAML test syntax **must** also be reflected in the corresponding files under `docs/sections/`. Keep documentation in sync with code at all times.

Relevant doc files:
- `docs/sections/cli.md` — CLI commands and flags
- `docs/sections/adapters/` — Adapter reference (HTTP, PostgreSQL, MongoDB, Redis, EventHub)
- `docs/sections/config.md` — Configuration (`e2e.config.yaml`) reference
- `docs/sections/assertions.md` — Assertion operators and JSONPath syntax
- `docs/sections/built-in-functions.md` — Built-in functions (`$uuid`, `$now`, `$totp`, etc.)
- `docs/sections/yaml-test.md` — YAML test file syntax and structure
- `docs/sections/examples.md` — Usage examples
