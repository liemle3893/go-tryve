# Task 1.3: Lifecycle Hooks - Test Plan

**Priority:** P0 (Critical)
**Dependency:** Phase 0 complete
**Effort Estimate:** 4-6 hours

---

## Test Scenarios

### Unit Tests

1. **Hook Loading**
   - Load config with `beforeAll` hook → Function instance
   - Load config with `afterAll` hook → Function instance
   - Load config with `beforeEach` hook → Function instance
   - Load config with `afterEach` hook → Function instance
   - Load config without hooks → No errors

2. **Hook Execution Order**
   - Verify `beforeAll` runs before any test
   - Verify `afterAll` runs after all tests
   - Verify `beforeEach` runs before each test
   - Verify `afterEach` runs after each test
   - Verify hooks execute in correct order

3. **Hook Error Handling**
   - `beforeAll` fails → Error reported, tests don't run
   - `afterAll` fails → Error reported, but tests already ran
   - `beforeEach` fails → Error reported, test doesn't run
   - `afterEach` fails → Error reported, but test already ran

### Integration Tests

1. **End-to-End Hook Flow**
   - Create test YAML with all 4 hooks
   - Run test → Verify all hooks execute in order
   - Verify test results include hook status

2. **Hook Failure Scenarios**
   - Test with failing `beforeAll` → Suite fails, tests skipped
   - Test with failing `afterEach` → Test completes, error reported
   - Test with all passing hooks → All tests pass

### Regression Tests

1. **No Hooks (Backward Compatibility)**
   - Run existing tests (Phase 0) without hooks
   - Verify all tests pass as before
   - Verify no hook-related errors

2. **Multiple Tests with Hooks**
   - Run test suite with multiple tests having hooks
   - Verify `beforeAll` runs once per suite
   - Verify `afterAll` runs once per suite
   - Verify `beforeEach`/`afterEach` run per test

### Edge Cases

1. **Empty Hook Functions**
   - Hook defined but returns nothing → No errors
   - Hook returns value → Value ignored, no errors

2. **Async Hooks**
   - Hook returns Promise → Awaits before proceeding
   - Hook with timeout → Handles timeout correctly

3. **Hook Errors with Tests**
   - Hook fails but `continueOnError` true → Documented behavior
   - Hook fails with test dependencies → Dependency handling

---

## Acceptance Criteria

- [ ] All hooks (beforeAll, afterAll, beforeEach, afterEach) work
- [ ] Hook failures properly reported
- [ ] Unit tests pass (tests/unit/core/config-loader.test.ts)
- [ ] No changes to tests without hooks (backward compatibility)
- [ ] Hook execution order is correct

---

## Test Commands

```bash
# Run hook unit tests
npm test -- tests/unit/core/config-loader.test.ts

# Run full test suite
npm test

# Create test directories
mkdir -p tests/e2e/hooks/scripts

# Create hook files and test YAML (see implementation plan for details)
# Then run: npx e2e run --env local

# Verify hooks execute
npx e2e run --env local 2>&1 | grep -E "(beforeAll|afterAll|beforeEach|afterEach)"
```

---

## Files Changed (Expected)

- Create: `src/core/hook-loader.ts`
- Modify: `src/core/config-loader.ts`
- Modify: `src/core/test-orchestrator.ts`
- Create: `tests/unit/core/config-loader.test.ts` (enhanced)

---

## Expected Test Results

**Pre-implementation:**
- Hook loader module: DOES NOT EXIST
- Hooks in YAML: IGNORED (not implemented)
- Hook execution: DOES NOT HAPPEN

**Post-implementation:**
- Hook loader module: EXISTS
- Hooks loaded from config: YES
- beforeAll executes: YES (before any test)
- afterAll executes: YES (after all tests)
- beforeEach executes: YES (before each test)
- afterEach executes: YES (after each test)
- Hook failures: REPORTED with error messages
- Tests without hooks: STILL WORK (backward compatibility)

---

## Known Limitations / Cannot Test

**Environment limitation:** Node.js/npm not available in current environment. Cannot execute test commands.

Test plan is complete and ready for execution once test environment is available.
