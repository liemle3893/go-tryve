# Technical Priorities: e2e-runner

**Project:** e2e-runner  
**Created:** 2026-03-02  
**Author:** Tech PO  
**Status:** Proposed

---

## Executive Summary

This document outlines the **top 5 technical priorities** for the e2e-runner project, based on the technical assessment conducted on 2026-03-02. These priorities address critical improvements needed to ensure long-term maintainability, reliability, and developer experience.

**Priority Ranking Criteria:**
- **Impact**: How much this improves the codebase
- **Risk**: What happens if we don't do it
- **Effort**: Time and resources required
- **Dependencies**: What must be done first

---

## Priority 1: Enable TypeScript Strict Mode

**Category:** Code Quality  
**Severity:** Medium  
**Effort:** S (2-3 days)  
**Priority:** P0 (Critical)

### Rationale

**Evidence:**
```json
// tsconfig.json
{
  "strict": false,
  "noImplicitAny": false,
  "noUnusedLocals": false,
  "noUnusedParameters": false
}
```

Disabled strict mode allows type safety issues to slip through:
- Implicit `any` types in 19 locations
- Unused variables/parameters not detected
- Potential runtime errors from type mismatches

**Impact:**
- **If done:** Catches bugs at compile time, improves IDE support, better refactoring
- **If not done:** Type bugs may reach production, harder to maintain

### Scope

**Files Affected:**
- `tsconfig.json`
- All 45 TypeScript source files (minor fixes)

**Estimated Effort:** 16-24 hours

**Work Breakdown:**
1. Enable strict flags in `tsconfig.json` (1 hour)
2. Fix compilation errors (12-20 hours)
3. Update documentation (1 hour)
4. Test and verify (2 hours)

### Dependencies

- None (can start immediately)

### Risks

- **Risk:** May reveal existing bugs hidden by loose typing
- **Mitigation:** Incremental enablement (one flag at a time)
- **Risk:** Breaking changes to public API
- **Mitigation:** Major version bump if needed

### Success Criteria

- [ ] All strict flags enabled in `tsconfig.json`
- [ ] Zero TypeScript compilation errors
- [ ] All existing tests pass
- [ ] Documentation updated

---

## Priority 2: Implement Test Coverage Reporting

**Category:** Testing  
**Severity:** Medium  
**Effort:** S (1-2 days)  
**Priority:** P0 (Critical)

### Rationale

**Evidence:**
```bash
$ find tests -name "*.test.ts" | wc -l
2  # Only 2 unit test files

$ cat vitest.config.ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    include: ['tests/**/*.test.ts'],
  },
});
# No coverage configuration
```

**Current State:**
- Only 2 unit test files for 45 source files
- No coverage metrics or reporting
- Unknown code coverage percentage
- No visibility into testing gaps

**Impact:**
- **If done:** Visibility into testing gaps, prevent regressions, confidence in refactoring
- **If not done:** Risk of shipping untested code, harder to maintain

### Scope

**Files Affected:**
- `vitest.config.ts`
- `package.json` (add coverage scripts)
- `README.md` (add coverage badge)
- New test files (future work)

**Estimated Effort:** 8-16 hours

**Work Breakdown:**
1. Configure coverage in `vitest.config.ts` (2 hours)
2. Add coverage scripts to `package.json` (1 hour)
3. Generate baseline coverage report (1 hour)
4. Document coverage targets (2 hours)
5. Add coverage badge to README (1 hour)
6. CI integration (future work)

### Dependencies

- None (can start immediately)

### Risks

- **Risk:** Low initial coverage may look bad
- **Mitigation:** Set incremental improvement targets (e.g., +5% per sprint)
- **Risk:** Coverage doesn't guarantee test quality
- **Mitigation:** Combine with code review standards

### Success Criteria

- [ ] Coverage configured in `vitest.config.ts`
- [ ] Coverage report generated successfully
- [ ] Baseline coverage percentage documented
- [ ] Coverage badge added to README
- [ ] Improvement targets defined

### Recommended Coverage Targets

| Module | Current (Est.) | Target (3 months) | Target (6 months) |
|--------|----------------|-------------------|-------------------|
| Core | ~10% | 60% | 80% |
| Adapters | ~20% | 70% | 85% |
| Assertions | ~0% | 70% | 85% |
| CLI | ~0% | 50% | 70% |
| **Overall** | **~15%** | **65%** | **80%** |

---

## Priority 3: Implement Lifecycle Hooks

**Category:** Core Feature  
**Severity:** High  
**Effort:** M (4-6 hours, already estimated in TODO.md)  
**Priority:** P0 (Critical)

### Rationale

**Evidence:**
```yaml
# e2e.config.yaml - NOT SUPPORTED YET
hooks:
  beforeAll: "./hooks/global-setup.ts"
  afterAll: "./hooks/global-teardown.ts"
  beforeEach: "./hooks/test-setup.ts"
  afterEach: "./hooks/test-teardown.ts"
```

**Current State:**
- Field exists in types but not enforced
- No way to run global setup/teardown
- Users must duplicate setup in each test

**Impact:**
- **If done:** Proper test isolation, cleaner test code, better resource management
- **If not done:** Test code duplication, potential resource leaks

### Scope

**Files Affected:**
- `src/types.ts` (types already exist)
- `src/core/config-loader.ts` (load hooks)
- `src/core/test-orchestrator.ts` (execute hooks)
- New: `src/core/hook-loader.ts`

**Estimated Effort:** 4-6 hours (from TODO.md)

**Work Breakdown:**
1. Create hook loader (2 hours)
2. Integrate with orchestrator (2 hours)
3. Add unit tests (1 hour)
4. Update documentation (1 hour)

### Dependencies

- None (can start immediately)

### Risks

- **Risk:** Hook failures may leave system in inconsistent state
- **Mitigation:** Implement proper error handling and cleanup
- **Risk:** Performance impact from running hooks
- **Mitigation:** Allow skipping hooks via CLI flag

### Success Criteria

- [ ] `beforeAll` hook runs before all tests
- [ ] `afterAll` hook runs after all tests
- [ ] `beforeEach` hook runs before each test
- [ ] `afterEach` hook runs after each test
- [ ] Hook failures properly reported
- [ ] Unit tests for hook execution
- [ ] Documentation updated

---

## Priority 4: Refactor html.reporter.ts

**Category:** Code Quality  
**Severity:** Medium  
**Effort:** M (2-3 days)  
**Priority:** P1 (High)

### Rationale

**Evidence:**
```bash
$ wc -l src/reporters/html.reporter.ts
1044 src/reporters/html.reporter.ts
```

**Current State:**
- Single file with 1044 lines
- 25 functions, 1 class
- High cyclomatic complexity
- Difficult to maintain and test

**Impact:**
- **If done:** Improved maintainability, easier to test, better code organization
- **If not done:** Technical debt accumulates, harder to add features

### Scope

**Files Affected:**
- `src/reporters/html.reporter.ts` (refactor)
- New: `src/reporters/html/` directory with modules

**Estimated Effort:** 16-24 hours

**Proposed Structure:**
```
src/reporters/html/
├── index.ts              (main reporter)
├── templates.ts          (HTML templates)
├── styles.ts             (CSS styles)
├── utils.ts              (helper functions)
├── test-summary.ts       (test summary generation)
├── step-details.ts       (step details generation)
└── types.ts              (HTML-specific types)
```

**Work Breakdown:**
1. Analyze current structure (2 hours)
2. Design module boundaries (2 hours)
3. Extract templates (4 hours)
4. Extract styles (2 hours)
5. Extract utilities (4 hours)
6. Update imports and tests (4 hours)
7. Documentation (2 hours)

### Dependencies

- None (can start immediately)

### Risks

- **Risk:** Breaking existing HTML output
- **Mitigation:** Visual regression testing, comprehensive tests
- **Risk:** Performance regression
- **Mitigation:** Benchmark before/after

### Success Criteria

- [ ] No file exceeds 300 lines
- [ ] All modules have single responsibility
- [ ] All existing tests pass
- [ ] HTML output unchanged (visual verification)
- [ ] Code coverage maintained or improved

---

## Priority 5: Add Unit Tests for Core Modules

**Category:** Testing  
**Severity:** High  
**Effort:** L (1-2 weeks)  
**Priority:** P1 (High)

### Rationale

**Evidence:**
```bash
$ find tests/unit -name "*.test.ts"
tests/unit/http-multipart.test.ts
tests/unit/shell-adapter.test.ts
# Only 2 unit test files!

$ find src/core -name "*.ts" | wc -l
8  # No unit tests for these
```

**Current State:**
- No unit tests for:
  - `test-orchestrator.ts` (632 lines)
  - `test-discovery.ts` (356 lines)
  - `variable-interpolator.ts` (468 lines)
  - `yaml-loader.ts` (446 lines)
  - `step-executor.ts` (408 lines)
  - `config-loader.ts` (318 lines)

**Impact:**
- **If done:** Confidence in refactoring, catch bugs early, documentation via tests
- **If not done:** High risk of regressions, fear of refactoring

### Scope

**Files to Create:**
- `tests/unit/core/test-orchestrator.test.ts`
- `tests/unit/core/test-discovery.test.ts`
- `tests/unit/core/variable-interpolator.test.ts`
- `tests/unit/core/yaml-loader.test.ts`
- `tests/unit/core/step-executor.test.ts`
- `tests/unit/core/config-loader.test.ts`

**Estimated Effort:** 40-80 hours

**Work Breakdown:**
1. `variable-interpolator.test.ts` (8 hours) - Test all built-in functions
2. `yaml-loader.test.ts` (6 hours) - Test YAML parsing
3. `config-loader.test.ts` (6 hours) - Test config loading
4. `test-discovery.test.ts` (8 hours) - Test file discovery
5. `step-executor.test.ts` (8 hours) - Test step execution
6. `test-orchestrator.test.ts` (12 hours) - Test orchestration logic

### Dependencies

- None (can start immediately)
- Benefits from Priority 2 (coverage reporting)

### Risks

- **Risk:** Core modules may be hard to test (no DI)
- **Mitigation:** Add dependency injection where needed
- **Risk:** Time-consuming to achieve good coverage
- **Mitigation:** Prioritize critical paths first

### Success Criteria

- [ ] Unit tests exist for all core modules
- [ ] Core module coverage ≥ 70%
- [ ] All tests pass
- [ ] Edge cases covered
- [ ] Error paths tested

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
```
Priority 1: Enable TypeScript Strict Mode
Priority 2: Implement Test Coverage Reporting
```

**Goal:** Improve code quality baseline and visibility

**Deliverables:**
- Strict mode enabled
- Coverage reporting configured
- Baseline coverage established

### Phase 2: Core Features (Week 3-4)
```
Priority 3: Implement Lifecycle Hooks
```

**Goal:** Complete critical P0 feature from backlog

**Deliverables:**
- Lifecycle hooks working
- Documentation updated
- Unit tests added

### Phase 3: Code Quality (Week 5-6)
```
Priority 4: Refactor html.reporter.ts
```

**Goal:** Reduce technical debt and improve maintainability

**Deliverables:**
- Modular HTML reporter
- Improved testability
- Documentation updated

### Phase 4: Testing Excellence (Week 7-10)
```
Priority 5: Add Unit Tests for Core Modules
```

**Goal:** Achieve 70% coverage for core modules

**Deliverables:**
- Comprehensive unit tests
- 70% core module coverage
- Improved confidence in refactoring

---

## Additional Recommendations (Lower Priority)

### P2 - Medium Priority

6. **Implement Watch Mode** (3-4 hours)
   - Improves developer experience
   - Already planned in TODO.md

7. **Add Dependency Auditing** (2 hours)
   - Automate security checks
   - Add to CI pipeline

8. **Migrate console.log to Logger** (4 hours)
   - 148 console statements to migrate
   - Consistent logging

9. **Add Dependency Injection** (8-16 hours)
   - Improve testability
   - Easier mocking

### P3 - Low Priority

10. **GraphQL Adapter** (6-8 hours)
11. **gRPC Adapter** (6-8 hours)
12. **Plugin System** (8-10 hours)
13. **Performance Optimizations** (variable)

---

## Success Metrics

### Code Quality Metrics

| Metric | Current | Target (3 months) | Target (6 months) |
|--------|---------|-------------------|-------------------|
| TypeScript strict mode | ❌ Disabled | ✅ Enabled | ✅ Enabled |
| Test coverage | ~15% | 65% | 80% |
| Files >500 lines | 6 | 3 | 0 |
| Unit test files | 2 | 8+ | 12+ |
| Console.log statements | 148 | 50 | 0 |

### Feature Completeness

| Feature | Current | Target (3 months) |
|---------|---------|-------------------|
| Lifecycle hooks | ❌ Not implemented | ✅ Implemented |
| Watch mode | ❌ CLI flag only | ✅ Implemented |
| Test dependencies | ❌ Not enforced | ✅ Enforced |

---

## Risk Assessment

### High Risks

1. **Low test coverage** - May hide bugs
   - **Mitigation:** Priority 2 and 5 address this

2. **Type safety issues** - Runtime errors
   - **Mitigation:** Priority 1 addresses this

### Medium Risks

3. **Large files** - Maintenance burden
   - **Mitigation:** Priority 4 addresses this

4. **Missing lifecycle hooks** - Test isolation issues
   - **Mitigation:** Priority 3 addresses this

### Low Risks

5. **Performance** - Not an issue for current use case
   - **Mitigation:** Monitor and optimize if needed

---

## Conclusion

These 5 priorities address the most critical technical issues in the e2e-runner codebase:

1. **Type safety** (Priority 1) - Foundation for reliability
2. **Test coverage** (Priority 2) - Visibility and confidence
3. **Lifecycle hooks** (Priority 3) - Critical feature gap
4. **Code complexity** (Priority 4) - Maintainability
5. **Core testing** (Priority 5) - Long-term quality

By following this roadmap, the project will achieve:
- ✅ Better type safety
- ✅ Comprehensive test coverage
- ✅ Complete feature set
- ✅ Improved maintainability
- ✅ Higher confidence in refactoring

**Total Estimated Effort:** 88-144 hours (11-18 days)

**Recommended Timeline:** 10 weeks (with parallel work where possible)

---

**Approval Required:**
- [ ] Architect review
- [ ] Biz PO assessment
- [ ] Human approval

**Next Steps:**
1. Create individual tasks for each priority
2. Assign to appropriate team member via PM
3. Track progress in project board
