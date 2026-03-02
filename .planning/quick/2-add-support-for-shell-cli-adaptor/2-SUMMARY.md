---
phase: quick-2
plan: 01
subsystem: adapters
tags: [shell, cli, adapter]
dependency-graph:
  requires: []
  provides: [shell-adapter, shell-yaml-validation]
  affects: [adapter-registry, types, yaml-loader, health-command]
tech-stack:
  added: []
  patterns: [adapter-pattern, inline-assertions, value-capture]
key-files:
  created:
    - src/adapters/shell.adapter.ts
    - docs/sections/adapters/shell.md
    - .claude/skills/e2e-runner/references/adapters/shell.md
    - tests/unit/shell-adapter.test.ts
  modified:
    - src/types.ts
    - src/adapters/adapter-registry.ts
    - src/adapters/index.ts
    - src/core/yaml-loader.ts
    - src/cli/health.command.ts
    - docs/sections/adapters/index.md
    - .claude/skills/e2e-runner/SKILL.md
decisions:
  - Used Node.js child_process for full shell features -- commands from static YAML test files (same trust model as CI/CD)
  - Non-zero exit codes returned as data, not errors -- enables exit code assertions
  - No peer dependencies needed -- uses built-in Node.js module
  - Followed existing adapter pattern (BaseAdapter extension, registry integration)
metrics:
  duration: 7m 23s
  completed: 2026-03-03
  tasks: 2/2
  tests-added: 20
  files-created: 4
  files-modified: 7
---

# Quick Task 2: Shell/CLI Adapter Summary

Shell/CLI adapter using Node.js built-in child_process with stdout/stderr/exitCode capture, inline assertions, timeout, cwd, and env override support.

## Task Completion

| Task | Name | Commit(s) | Key Files |
|------|------|-----------|-----------|
| 1 | Implement shell adapter (TDD) | ccfd36b (RED), 8b70bdd (GREEN) | src/adapters/shell.adapter.ts, tests/unit/shell-adapter.test.ts |
| 2 | Documentation and skills | 8b3c560 | docs/sections/adapters/shell.md, .claude/skills/e2e-runner/references/adapters/shell.md |

## What Was Built

### ShellAdapter (`src/adapters/shell.adapter.ts`)
- Extends `BaseAdapter` with single `exec` action
- Executes shell commands via Node.js built-in child_process with full shell features
- Returns `ShellResponse`: `{ exitCode, stdout, stderr, duration }`
- Non-zero exit codes returned as data (not errors) for assertion-based testing
- Inline assertions: `exitCode` (exact match), `stdout`/`stderr` (`contains`, `matches`, `equals`)
- Value capture: `stdout`, `stderr`, `exitCode` paths
- Configurable: `timeout` (default 30s), `cwd`, `env` (merged with process.env)
- Timeout kills the process and throws AdapterError

### Integration Points
- `src/types.ts`: `'shell'` added to `AdapterType` union, `ShellAdapterConfig` interface added
- `src/adapters/adapter-registry.ts`: Shell adapter registered (no connection cost, like HTTP), `getShell()` convenience method, `'shell'` in `parseAdapterType()`
- `src/adapters/index.ts`: Re-exports `ShellAdapter`, `ShellRequestParams`, `ShellResponse`, `ShellAssertion`
- `src/core/yaml-loader.ts`: `'shell'` in `VALID_ADAPTERS`, validation requires `command` field and only `exec` action
- `src/cli/health.command.ts`: `'Shell'` added to adapter name formatting map

### Unit Tests (20 tests)
- Command execution: stdout capture, stderr capture, exit code 0 and non-zero
- Timeout: kills long-running commands
- Environment: passes env vars to child process
- Working directory: cwd override
- Assertions: exitCode match/mismatch, stdout contains, stderr contains
- Captures: stdout, stderr, exitCode
- Error handling: unknown action, missing command
- Health check, connect/disconnect, adapter name

### Documentation
- `docs/sections/adapters/shell.md`: Full adapter reference (config, action, assertions, captures, 5 examples)
- `docs/sections/adapters/index.md`: Shell added to available adapters table
- `.claude/skills/e2e-runner/SKILL.md`: Shell in adapter list, examples, reference link
- `.claude/skills/e2e-runner/references/adapters/shell.md`: Concise reference

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed TypeScript compilation error in health.command.ts**
- **Found during:** Task 1 (build verification)
- **Issue:** Adding 'shell' to AdapterType union caused `Record<AdapterType, string>` in health.command.ts to require a 'shell' key
- **Fix:** Added `shell: 'Shell'` to the adapter name mapping object
- **Files modified:** src/cli/health.command.ts
- **Commit:** 8b70bdd

**2. [Rule 1 - Bug] Fixed TypeScript type narrowing for exec error handling**
- **Found during:** Task 1 (build verification)
- **Issue:** ExecException.code is number but ErrnoException.code is string, causing TS2352
- **Fix:** Used errno property and message check instead of direct ErrnoException cast
- **Files modified:** src/adapters/shell.adapter.ts
- **Commit:** 8b70bdd

## Verification Results

- `npm run build`: PASS (TypeScript compiles without errors)
- `npm test`: PASS (32 tests: 20 shell adapter + 12 existing multipart)
- Shell adapter integration points confirmed via grep across all source files
- Documentation files exist and cross-reference correctly

## Self-Check: PASSED

All 4 created files verified on disk. All 3 commits verified in git log.
