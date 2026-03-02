# Requirements: E2E Runner — Feature Complete

**Defined:** 2026-03-02
**Core Value:** Every test that passes actually passed, and every feature that exists actually works

## v1 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Core Correctness

- [ ] **CORE-01**: Assertion engine in StepExecutor calls assertion-runner instead of being a no-op stub
- [ ] **CORE-02**: Steps that fail with continueOnError report a 'warned' status instead of false 'passed'
- [ ] **CORE-03**: Test-level retryCount is derived from step retry counts instead of hardcoded zero
- [ ] **CORE-04**: EventHub adapter rejects promise on processError instead of resolving with failResult

### Execution Engine

- [ ] **EXEC-01**: Tests with `depends` field execute in dependency order via topological sort
- [ ] **EXEC-02**: Parallel test execution passes context explicitly instead of sharing mutable instance state
- [ ] **EXEC-03**: Hook paths resolve relative to config file directory instead of cwd

### Adapter Improvements

- [ ] **ADPT-01**: TypeScript function-backed steps use a dedicated 'typescript' adapter type instead of 'http' placeholder
- [ ] **ADPT-02**: Redis flushPattern uses SCAN-based iteration instead of blocking KEYS command
- [ ] **ADPT-03**: MongoDB normalizeFilter imports ObjectId once at connect time instead of per-operation
- [ ] **ADPT-04**: Kafka adapter supports producing messages to topics
- [ ] **ADPT-05**: Kafka adapter supports consuming and verifying messages with content assertions
- [ ] **ADPT-06**: Kafka adapter waitFor pattern resolves on matching message with configurable timeout

### Code Quality

- [ ] **QUAL-01**: test-discovery uses dynamic import instead of require() for minimatch
- [ ] **QUAL-02**: Dead captureValues method removed from StepExecutor; capture logic consolidated
- [ ] **QUAL-03**: ts-node and tsx listed in peerDependenciesMeta as optional peers
- [ ] **QUAL-04**: minimatch either made required dependency or fallback limitations documented
- [ ] **QUAL-05**: Vitest configured with vitest.config.ts for unit testing
- [ ] **QUAL-06**: Unit tests cover assertion engine (matchers, assertion-runner, jsonpath)
- [ ] **QUAL-07**: Unit tests cover core execution (variable-interpolator, step-executor, test-orchestrator)
- [ ] **QUAL-08**: Unit tests cover adapter logic (HTTP, PostgreSQL, MongoDB, Redis, EventHub, Kafka)
- [ ] **QUAL-09**: Unit tests achieve 85%+ coverage on core modules

## v2 Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Security Hardening

- **SEC-01**: Regex patterns from user YAML wrapped in try/catch with length limits
- **SEC-02**: $file() interpolation restricted to paths within project root
- **SEC-03**: Environment variable interpolation uses explicit allowlist instead of full process.env
- **SEC-04**: Parallel config validated against connection pool limits with warning on exceed

### Performance

- **PERF-01**: Test metadata cached after first parse to avoid double-loading during filtering
- **PERF-02**: HTML report generation uses streaming write instead of full in-memory string

## Out of Scope

| Feature | Reason |
|---------|--------|
| GraphQL adapter | Not needed for current use cases |
| gRPC adapter | Not needed for current use cases |
| GUI / dashboard | CLI-first tool — out of scope |
| Cloud-hosted execution | Runs locally or in CI |
| New reporter formats | Console/JUnit/HTML/JSON covers all standard use cases |
| Breaking YAML syntax changes | Backward compatibility with existing test files required |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CORE-01 | Phase 1 | Pending |
| CORE-02 | Phase 1 | Pending |
| CORE-03 | Phase 1 | Pending |
| CORE-04 | Phase 1 | Pending |
| EXEC-01 | Phase 2 | Pending |
| EXEC-02 | Phase 2 | Pending |
| EXEC-03 | Phase 2 | Pending |
| ADPT-01 | Phase 1 | Pending |
| ADPT-02 | Phase 5 | Pending |
| ADPT-03 | Phase 5 | Pending |
| ADPT-04 | Phase 3 | Pending |
| ADPT-05 | Phase 3 | Pending |
| ADPT-06 | Phase 3 | Pending |
| QUAL-01 | Phase 1 | Pending |
| QUAL-02 | Phase 1 | Pending |
| QUAL-03 | Phase 1 | Pending |
| QUAL-04 | Phase 1 | Pending |
| QUAL-05 | Phase 4 | Pending |
| QUAL-06 | Phase 4 | Pending |
| QUAL-07 | Phase 4 | Pending |
| QUAL-08 | Phase 4 | Pending |
| QUAL-09 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0

---
*Requirements defined: 2026-03-02*
*Last updated: 2026-03-02 after roadmap creation — all 22 requirements mapped*
