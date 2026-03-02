# E2E Runner — Feature Complete

## What This Is

A flexible end-to-end testing framework for API and database testing. Tests are written in YAML (declarative) or TypeScript (programmatic), with adapters for HTTP, PostgreSQL, MongoDB, Redis, and EventHub. Distributed as an npm package (`@liemle3893/e2e-runner`) with a CLI and programmatic API. The framework exists but has significant incomplete features, bugs, and dead code that prevent real-world use with confidence.

## Core Value

Every test that passes actually passed, and every feature that exists actually works — no silent failures, no stubs, no dead code paths.

## Requirements

### Validated

- ✓ CLI with run/validate/list/health/init/test commands — existing
- ✓ YAML test file loading and schema validation — existing
- ✓ TypeScript test file loading — existing
- ✓ HTTP adapter with native fetch — existing
- ✓ PostgreSQL adapter via pg — existing
- ✓ MongoDB adapter via mongodb driver — existing
- ✓ Redis adapter via ioredis — existing
- ✓ EventHub adapter via @azure/event-hubs — existing
- ✓ Variable interpolation with built-ins (uuid, timestamp, random, md5, base64, file) — existing
- ✓ JSONPath-based assertions and value capture — existing
- ✓ 4 reporters (console, JUnit, HTML, JSON) — existing
- ✓ Test filtering by tags, priority, grep — existing
- ✓ Step-level retry with exponential backoff — existing
- ✓ Parallel test execution via p-limit — existing
- ✓ Phase-based test lifecycle (setup/execute/verify/teardown) — existing
- ✓ Programmatic API (runTests, validateTests, listTests, checkHealth) — existing
- ✓ Typed error hierarchy with structured exit codes — existing

### Active

- [ ] Complete assertion engine — replace StepExecutor.validateAssertions stub with actual assertion-runner calls
- [ ] Fix continueOnError status — introduce distinct status for forgiven failures instead of marking as 'passed'
- [ ] Fix retryCount aggregation — derive test-level retryCount from step results instead of hardcoded zero
- [ ] Implement test dependency ordering — call sortTestsByDependencies before execution
- [ ] Add dedicated TypeScript adapter type — replace 'http' placeholder for function-backed steps
- [ ] Consolidate capture logic — remove dead captureValues from StepExecutor, unify with base adapter
- [ ] Fix EventHub error handling — processError should reject promise, not resolve with failResult
- [ ] Add Kafka adapter — produce messages to topics and consume/verify messages with content assertions
- [ ] Fix mixed require/import — replace require('minimatch') with dynamic import in test-discovery
- [ ] Add unit test suite — set up Vitest, write tests for core, adapters, assertions, and CLI
- [ ] Fix Redis KEYS command — replace with SCAN-based iteration for flushPattern
- [ ] Address security concerns — validate regex patterns, restrict $file() paths, consider env var allowlist
- [ ] Validate parallel config — enforce upper bound and warn when exceeding connection pool limits
- [ ] Fix hook path resolution — resolve relative to config directory, not cwd
- [ ] Fix parallel test state — pass test/phase context explicitly instead of shared mutable instance state
- [ ] Fix MongoDB ObjectId import — import once at connect time instead of per-operation
- [ ] Fix sequential metadata loading — cache parsed metadata to avoid double-parsing test files
- [ ] Clean up ts-node peer dependency — add to peerDependenciesMeta with clear documentation
- [ ] Improve minimatch handling — either make required dependency or document fallback limitations

### Out of Scope

- New reporter formats — console/JUnit/HTML/JSON covers all standard use cases
- GraphQL adapter — not needed for current use cases
- gRPC adapter — not needed for current use cases
- GUI or dashboard — CLI-first tool
- Cloud-hosted execution — runs locally or in CI

## Context

This is a brownfield project at version 1.2.1, published to npm as `@liemle3893/e2e-runner`. The framework has a solid architectural foundation (layered plugin architecture with event-driven reporting) but accumulated significant technical debt during initial development. Multiple features were started but never completed — the assertion engine in StepExecutor is a stub referencing "Phase 5" that never happened, test dependency ordering is parsed but never enforced, and the TypeScript test adapter uses a misleading 'http' placeholder type.

The codebase has zero unit tests (`npm test` prints "No tests yet"). All validation happens through E2E test YAML files in `tests/e2e/adapters/`. Several bugs cause silent failures: `continueOnError` marks failed steps as passed, retryCount is always zero at the test level, and EventHub errors resolve instead of rejecting.

The immediate goal is to make every existing feature actually work correctly, add Kafka support for message queue testing, and build enough test coverage to refactor with confidence. The end state is a framework battle-tested enough to use on real projects.

## Constraints

- **Backward compatibility**: Existing YAML test files must continue to work — no breaking changes to test syntax
- **Peer dependency model**: Adapter drivers remain optional peer deps — install only what you need
- **Node.js >=18**: Minimum runtime requirement, uses native fetch
- **CommonJS output**: Published as CommonJS (tsconfig module: commonjs), must not break consumers
- **npm package**: Published as `@liemle3893/e2e-runner` — changes must be publishable

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Complete features before adding tests | Features need to work correctly first — testing broken stubs wastes effort | — Pending |
| Kafka adapter via kafkajs | Most popular Node.js Kafka client, good TypeScript support | — Pending |
| Vitest for unit tests | Fast, TypeScript-native, good ESM/CJS interop | — Pending |
| Fix bugs in-place (no rewrites) | Existing architecture is sound — issues are localized bugs, not structural | — Pending |

---
*Last updated: 2026-03-02 after initialization*
