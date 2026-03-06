# Task 1.4: Test Dependency Ordering - Test Plan

**Priority:** P0 (Critical)
**Dependency:** Phase 0 complete
**Effort Estimate:** 3-4 hours

---

## Test Scenarios

### Unit Tests

1. **Topological Sort Algorithm**
   - Tests with no dependencies → Discovery order maintained
   - Test B depends on A → A runs before B
   - Multiple dependencies (D depends on B, C) → B and C before D
   - Complex dependency graph → Correct ordering

2. **Circular Dependency Detection**
   - A depends on B, B depends on A → Error thrown
   - A depends on B, B depends on C, C depends on A → Error thrown
   - Self-dependency (A depends on A) → Error thrown
   - Empty test set → No error

3. **Failed Dependency Handling**
   - Test A fails, Test B depends on A → Test B skipped with reason
   - Test A passes, Test B depends on A → Test B runs
   - Multiple failed dependencies → Test skipped with all reasons

### Integration Tests

1. **End-to-End Dependency Execution**
   - Create 3 test YAML files with dependencies
   - Run test suite → Verify execution order
   - Verify dependent tests run after prerequisites

2. **Real-World Dependency Scenarios**
   - Setup test → Depends on nothing → Runs first
   - CRUD test → Depends on setup test → Runs second
   - Cleanup test → Depends on CRUD test → Runs third

### Regression Tests

1. **No Dependencies (Backward Compatibility)**
   - Run existing tests without `depends` field
   - Verify tests run in discovery order
   - Verify all tests pass as before

2. **Mixed Tests (Some with deps, some without)**
   - Test suite with 5 tests, 2 have dependencies
   - Verify ordering respects dependencies
   - Verify independent tests maintain order

### Edge Cases

1. **Empty Dependency List**
   - Test with `depends: []` → Treated as no dependencies
   - Test with `depends: null` → Treated as no dependencies

2. **Non-Existent Dependencies**
   - Test depends on non-existent test name → Error reported
   - Graceful handling of typos in dependency names

3. **Performance with Large Test Suites**
   - 100+ tests with complex dependencies → O(N + E) performance
   - Verify no exponential time complexity

---

## Acceptance Criteria

- [ ] Tests with `depends` execute in correct order
- [ ] Circular dependencies detected and reported
- [ ] Failed dependencies cause dependent tests to skip with reasons
- [ ] Unit tests pass (tests/unit/core/topological-sort.test.ts)
- [ ] No changes to tests without dependencies

---

## Test Commands

```bash
# Run topological sort unit tests
npm test -- tests/unit/core/topological-sort.test.ts

# Create dependency test files
mkdir -p tests/e2e/dependencies

# Test files creation (see implementation plan for YAML examples)
# TC-DEP-001: Setup test (no dependencies)
# TC-DEP-002: Depends on TC-DEP-001
# TC-CIRC-001, TC-CIRC-002: Circular dependency (should fail)

# Verify execution order
npx e2e run --env local | grep "TC-DEP"

# Test circular dependency (should fail with error)
npx e2e run --env local 2>&1 | grep "Circular"
```

---

## Files Changed (Expected)

- Create: `src/core/topological-sort.ts`
- Create: `tests/unit/core/topological-sort.test.ts`
- Modify: `src/core/test-orchestrator.ts`
- Modify: `src/core/test-discovery.ts`

---

## Expected Test Results

**Pre-implementation:**
- Topological sort module: DOES NOT EXIST
- Dependencies field: PARSED BUT IGNORED
- Tests run in: DISCOVERY ORDER (always)

**Post-implementation:**
- Topological sort module: EXISTS
- Dependencies field: ENFORCED
- Tests without dependencies: RUN IN DISCOVERY ORDER
- Tests with dependencies: RUN IN TOPOLOGICAL ORDER
- Circular dependencies: DETECTED AND REPORTED
- Failed dependencies: CAUSE SKIP WITH REASON

---

## Known Limitations / Cannot Test

**Environment limitation:** Node.js/npm not available in current environment. Cannot execute test commands.

Test plan is complete and ready for execution once test environment is available.
