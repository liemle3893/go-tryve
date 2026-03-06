# Task 1.5: Unit Tests for Core Modules - Test Plan

**Priority:** P1 (High)
**Dependency:** Task 1.1 (strict mode), Task 1.2 (coverage reporting)
**Effort Estimate:** 40-80 hours

---

## Test Scenarios

### Test Coverage Analysis

1. **Pre-Implementation Coverage**
   - Run `npm run test:coverage`
   - Document baseline coverage for each core module
   - Identify modules with <70% coverage

2. **Post-Implementation Coverage**
   - Run `npm run test:coverage`
   - Verify each core module ≥70% coverage
   - Compare to baseline

### Module-Specific Tests

#### test-orchestrator.ts
- Test: Simple test execution (single phase)
- Test: Multiple phases (setup, execute, verify, teardown)
- Test: Test failure reporting
- Test: Retry logic
- Test: Error handling
- Test: Variable interpolation
- Test: Hook integration

#### test-discovery.ts
- Test: YAML file discovery
- Test: TypeScript test file discovery
- Test: Empty directory handling
- Test: Nested directory handling
- Test: File filtering (ignore non-test files)

#### yaml-loader.ts
- Test: Valid YAML file loading
- Test: Malformed YAML handling
- Test: Test structure validation
- Test: Relative path resolution
- Test: Error reporting

#### config-loader.ts
- Test: Config file loading
- Test: CLI argument merging
- Test: Config schema validation
- Test: Error handling
- Test: Hook loading (if not covered by Task 1.3)

#### step-executor.ts (enhance existing)
- Test: HTTP adapter execution
- Test: PostgreSQL adapter execution
- Test: Redis adapter execution
- Test: MongoDB adapter execution
- Test: EventHub adapter execution
- Test: Assertion validation
- Test: Retry with exponential backoff
- Test: Error handling

#### variable-interpolator.ts (verify existing)
- Test: Already has 42 tests → Verify all pass
- Test: Coverage check → Verify ≥70%
- Add tests if coverage <70%

### Integration Tests

1. **Full Test Suite**
   - Run `npm test` → All unit tests pass
   - Verify no test flakes (run 3x to confirm)

2. **Coverage Report**
   - Run `npm run test:coverage`
   - Open `coverage/index.html`
   - Verify all `src/core/` files ≥70%

### Regression Tests

1. **Phase 0 Fixes Still Work**
   - Run existing E2E tests
   - Verify 89/89 tests pass
   - No behavioral changes

2. **Existing Unit Tests**
   - Verify existing unit tests still pass
   - `tests/unit/core/step-executor.test.ts`
   - `tests/unit/core/step-executor-continueOnError.test.ts`

---

## Acceptance Criteria

- [ ] Core module coverage ≥70% (all files in `src/core/`)
- [ ] All tests pass (no flaky tests)
- [ ] Existing E2E tests still pass (89/89)
- [ ] Test files follow existing patterns (describe/it/expect)
- [ ] Tests use mocks for external dependencies

---

## Test Commands

```bash
# Generate coverage report
npm run test:coverage

# Check coverage per file
# Open coverage/index.html and navigate to src/core/

# Run all unit tests
npm test

# Run specific module tests
npm test -- tests/unit/core/test-orchestrator.test.ts
npm test -- tests/unit/core/test-discovery.test.ts
npm test -- tests/unit/core/yaml-loader.test.ts
npm test -- tests/unit/core/config-loader.test.ts
npm test -- tests/unit/core/step-executor.test.ts
npm test -- tests/unit/core/variable-interpolator.test.ts

# Verify E2E tests still pass
npm test

# Check for flaky tests (run 3 times)
npm test && npm test && npm test

# Verify no new test failures compared to baseline
# Compare test results to Phase 0 baseline
```

---

## Files Changed (Expected)

- Create: `tests/unit/core/test-orchestrator.test.ts`
- Create: `tests/unit/core/test-discovery.test.ts`
- Create: `tests/unit/core/yaml-loader.test.ts`
- Create: `tests/unit/core/config-loader.test.ts`
- Modify: `tests/unit/core/step-executor.test.ts` (enhance existing)

---

## Expected Test Results

**Pre-implementation:**
- Core module coverage: ~15% (baseline)
- Unit test files: 2 existing (step-executor)

**Post-implementation:**
- Core module coverage: ≥70% (all files)
- Unit test files: 6 total (4 new + 2 enhanced)
- All tests pass: YES
- No flaky tests: YES
- E2E tests still pass: 89/89

---

## Known Limitations / Cannot Test

**Environment limitation:** Node.js/npm not available in current environment. Cannot execute test commands or verify coverage.

Test plan is complete and ready for execution once test environment is available.
