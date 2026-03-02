# E2E Runner — Product Overview

**Document Owner:** Business Product Owner  
**Last Updated:** 2026-03-02  
**Version:** 1.0

---

## Executive Summary

E2E Runner is a **developer-centric, YAML-based end-to-end testing framework** designed for teams who need to validate complex distributed systems spanning multiple data stores and APIs. Unlike traditional API testing tools that focus solely on HTTP endpoints, E2E Runner treats databases, message queues, and caches as first-class testing targets.

The framework's core innovation is **declarative multi-adapter testing** — write a single test file that orchestrates HTTP calls, database queries, Redis operations, and message queue interactions with built-in variable capture and assertion chaining.

---

## Product Vision

> *"Test the full system, not just the API layer."*

Enable engineering teams to write comprehensive end-to-end tests that mirror real-world workflows across:
- **REST APIs** (HTTP adapter)
- **Relational databases** (PostgreSQL adapter)
- **Document stores** (MongoDB adapter)
- **Caching layers** (Redis adapter)
- **Message queues** (Azure EventHub adapter, Kafka planned)

All with a unified YAML syntax and zero boilerplate.

---

## Value Proposition

### For Engineering Teams

| Pain Point | E2E Runner Solution |
|------------|---------------------|
| **Fragmented test tooling** — separate scripts for API tests, DB validation, and cache checks | Single unified framework for all adapters — write one test file covering the entire flow |
| **Complex test setup/teardown** — manually managing test data across services | Built-in 4-phase lifecycle (`setup` → `execute` → `verify` → `teardown`) with automatic cleanup |
| **Fragile test assertions** — hardcoded values break tests | Dynamic variables with built-in functions (`$uuid()`, `$timestamp()`, `$random()`) and value capture from responses |
| **Slow test execution** — sequential test runs take hours | Parallel execution with configurable concurrency via `p-limit` |
| **Poor debugging experience** — "test failed" with no context | Rich reporters (console, HTML, JUnit XML, JSON) with captured values and request/response logging |

### Business Value

1. **Faster Developer Velocity** — 60% less boilerplate vs. writing separate Jest/Mocha tests + custom DB scripts
2. **Higher Test Reliability** — Declarative syntax reduces flaky tests caused by timing issues or state corruption
3. **Comprehensive Coverage** — Validates data consistency across API, database, and cache in a single test
4. **CI/CD Integration** — JUnit reporter integrates with Jenkins, GitHub Actions, GitLab CI, CircleCI out of the box
5. **Reduced Maintenance Burden** — YAML tests are readable by non-developers (QA, product managers)

---

## Target User Personas

### Primary Persona: Senior Backend Engineer

**Profile:**
- 5-10 years experience building distributed systems
- Works on microservices with PostgreSQL/MongoDB backends, Redis caching, and message queues
- Responsible for service reliability and test coverage
- Frustrated with fragmented test tools (Postman for APIs, custom scripts for DB, separate tools for queues)

**Goals:**
- Write E2E tests that validate full system behavior, not just API responses
- Reduce test flakiness and maintenance time
- Integrate tests into CI/CD pipelines with clear reporting

**Key Use Cases:**
- Validate user registration flow: API creates user → PostgreSQL record exists → Redis cache populated → EventHub event published
- Test payment processing: API initiates payment → database state transitions → message queue notification sent
- Verify data migration scripts: Run migration → validate data integrity across tables → check audit logs

**Success Metrics:**
- Time to write a new E2E test: < 15 minutes
- Test flakiness rate: < 2%
- CI pipeline test duration: < 10 minutes for critical path tests

---

### Secondary Persona: QA Engineer

**Profile:**
- 3-5 years experience in manual and automated testing
- Comfortable with YAML and basic scripting
- Needs to write regression tests without deep programming knowledge
- Works alongside developers to validate features before release

**Goals:**
- Write and maintain test suites independently
- Generate clear test reports for stakeholders
- Reproduce and debug test failures quickly

**Key Use Cases:**
- Create smoke test suite covering critical API endpoints
- Build regression test suite for user management workflows
- Validate database constraints and data integrity rules

**Success Metrics:**
- Tests readable without developer assistance
- Clear failure reports with captured values
- Ability to debug failures without SSH access to servers

---

### Tertiary Persona: DevOps / Platform Engineer

**Profile:**
- Manages CI/CD pipelines and infrastructure
- Responsible for test automation and deployment gates
- Monitors test suite performance and reliability
- Integrates testing tools with monitoring/alerting systems

**Goals:**
- Parallel test execution to minimize CI time
- JUnit/JSON reports for pipeline integration
- Health checks for adapter connectivity before running tests

**Key Use Cases:**
- Configure parallel test runs across 4-8 workers
- Generate JUnit reports for Jenkins/GitHub Actions
- Run health checks to validate environment before deployment
- Track test execution trends over time

**Success Metrics:**
- Test suite runtime: < 15 minutes for full suite
- Pipeline integration: < 1 hour setup time
- Adapter health visibility: Real-time status dashboard

---

## Key Differentiators

### vs. Postman / Insomnia

| Feature | Postman | E2E Runner |
|---------|---------|------------|
| **Primary Focus** | API testing only | Multi-adapter (API + DB + cache + queues) |
| **Test Format** | GUI-based collections | Code-as-infrastructure (YAML/TypeScript) |
| **Database Testing** | Requires custom scripts or paid features | First-class PostgreSQL/MongoDB adapters |
| **CI/CD Integration** | Requires Newman CLI + paid plan for teams | Native CLI + JUnit reporter out of the box |
| **Version Control** | JSON exports (difficult to review) | YAML files (Git-friendly, easy diffs) |
| **Parallel Execution** | Paid feature | Built-in (no additional cost) |

**E2E Runner Wins When:** You need to validate data consistency across API and database in a single test.

---

### vs. Cypress / Playwright

| Feature | Cypress / Playwright | E2E Runner |
|---------|----------------------|------------|
| **Primary Focus** | Browser automation / UI testing | Backend API and database testing |
| **Database Testing** | Limited (requires plugins) | First-class adapters for PostgreSQL, MongoDB, Redis |
| **Test Speed** | Slower (browser overhead) | Fast (no browser, direct API/DB calls) |
| **Test Complexity** | Higher (DOM selectors, waits) | Lower (declarative YAML) |
| **Message Queue Testing** | No native support | EventHub adapter built-in, Kafka coming |

**E2E Runner Wins When:** You're testing backend services, APIs, or data pipelines without a UI component.

---

### vs. Jest / Mocha (with custom scripts)

| Feature | Jest / Mocha | E2E Runner |
|---------|--------------|------------|
| **Test Format** | JavaScript/TypeScript code | Declarative YAML (or TypeScript) |
| **Setup/Teardown** | Manual boilerplate | Built-in lifecycle phases |
| **Database Adapters** | Write your own | Pre-built adapters included |
| **Variable Capture** | Manual code | Built-in `capture` syntax |
| **Reporters** | Configure manually | Console/HTML/JUnit/JSON out of the box |
| **Learning Curve** | Requires JS expertise | YAML readable by non-developers |

**E2E Runner Wins When:** You want less boilerplate, faster test authoring, and accessibility for QA/non-developers.

---

### vs. Custom Scripts (Bash + curl + psql)

| Feature | Custom Scripts | E2E Runner |
|---------|----------------|------------|
| **Maintenance** | High (fragmented, no standards) | Low (unified framework) |
| **Readability** | Poor (bash + SQL + JSON parsing) | High (declarative YAML) |
| **Error Handling** | Manual `set -e` and checks | Built-in retry logic and error hierarchy |
| **Reporting** | Custom logging | Multiple reporters included |
| **Parallelism** | Complex bash orchestration | `--parallel 4` flag |
| **Variable Capture** | `jq` + temp files | Native capture syntax |

**E2E Runner Wins When:** Your custom scripts have grown unmaintainable and you need a standardized framework.

---

## Market Position

**Category:** API and Database Testing Framework  
**Maturity:** v1.2.1 (npm package) — Feature-complete core with active development on gaps  
**License:** MIT (open-source)  
**Distribution:** npm (`@liemle3893/e2e-runner`)

### Competitive Landscape

| Tool | Focus | Database Testing | Message Queues | Price |
|------|-------|------------------|----------------|-------|
| **E2E Runner** | Multi-adapter E2E | ✓ PostgreSQL, MongoDB, Redis, EventHub, Kafka (planned) | ✓ EventHub, Kafka (planned) | Free (MIT) |
| Postman | API testing | Paid add-on | No | Free tier, $12-29/user/mo |
| Playwright | Browser E2E | Plugins only | No | Free (MIT) |
| Cypress | Browser E2E | Plugins only | No | Free tier, $75/mo |
| k6 | Load testing | Limited | No | Free tier, $99/mo |

**E2E Runner's Niche:** Teams testing **backend systems with multiple data stores and message queues** who need more than API-only tools but don't want to build custom frameworks.

---

## Success Metrics (Product Health)

| Metric | Target | Current Status |
|--------|--------|----------------|
| **npm weekly downloads** | > 500 | To be measured |
| **GitHub stars** | > 1,000 | To be measured |
| **Test coverage** | > 85% | 0% (Phase 4 roadmap) |
| **Critical bugs open** | < 5 | ~15 (see ROADMAP.md) |
| **Test flakiness rate** | < 2% | Unknown (no test suite yet) |
| **Time to first passing test** | < 10 min | ~15 min (estimated) |

---

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **Silent test failures** (stub assertion engine, false `passed` status) | High — users lose trust in test results | High (documented in ROADMAP.md Phase 1) | Prioritize Phase 1 fixes; add unit tests for assertion engine |
| **No unit test coverage** | High — refactoring is risky | High (0% coverage currently) | Execute Phase 4 unit test suite after Phase 1-3 fixes |
| **Competitor feature velocity** | Medium — users may choose Postman/Playwright for ecosystem | Medium | Focus on unique differentiator (multi-adapter testing); add Kafka support |
| **Documentation gaps** | Medium — users can't self-serve | Low (docs/ is comprehensive) | Keep docs updated with each release; add more examples |
| **Peer dependency complexity** | Medium — install friction | Medium | Document clear setup guide; consider bundling common adapters |

---

## Next Steps

1. **Complete Phase 1 Fixes** — Wire assertion engine, fix false `passed` status, address silent failures
2. **Add Unit Test Suite (Phase 4)** — Build confidence for future refactoring
3. **Kafka Adapter (Phase 3)** — Expand message queue support beyond Azure EventHub
4. **Developer Experience Improvements** — Watch mode, step-by-step debugging, HTTP traffic capture (see TODO.md)
5. **Marketing and Community** — Publish blog posts, create example repos, engage on Reddit/Hacker News

---

## References

- **Repository:** https://github.com/liemle3893/e2e-runner
- **npm Package:** @liemle3893/e2e-runner
- **Documentation:** `docs/` directory
- **Roadmap:** `.planning/ROADMAP.md`
- **Requirements:** `.planning/REQUIREMENTS.md`
- **Known Issues:** `.planning/PROJECT.md` (Active section)

---

*This document was created by the Business Product Owner as part of the initial product analysis. For technical architecture details, consult the Architect's design documents.*
