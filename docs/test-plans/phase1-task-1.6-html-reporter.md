# Task 1.6: Refactor html.reporter.ts - Test Plan

**Priority:** P1 (High)
**Dependency:** Phase 0 complete
**Effort Estimate:** 16-24 hours

---

## Test Scenarios

### Unit Tests

1. **Module Extraction**
   - Verify `src/reporters/html/styles.ts` exists and exports CSS
   - Verify `src/reporters/html/scripts.ts` exists and exports JS
   - Verify `src/reporters/html/templates.ts` exists and exports templates
   - Verify `src/reporters/html/utils.ts` exists and exports utilities
   - Verify `src/reporters/html/reporter.ts` exists as main orchestrator

2. **File Size Verification**
   - Verify no file exceeds 300 lines
   - Count lines in each module
   - Verify total lines distributed across 5 modules

3. **Import/Export Verification**
   - Verify main reporter imports from modules correctly
   - Verify modules export required functions
   - Verify `src/reporters/index.ts` imports from new path

### Integration Tests

1. **HTML Output Parity**
   - Run E2E test before refactoring → Save HTML output
   - Run E2E test after refactoring → Compare HTML output
   - Verify HTML output is identical (diff shows no changes)

2. **Functional Parity**
   - Generate HTML report with test results
   - Verify dashboard displays correctly
   - Verify test list displays correctly
   - Verify test details expand/collapse work
   - Verify filtering works
   - Verify sorting works

3. **Report Generation**
   - Run test suite with HTML reporter
   - Verify HTML file is generated
   - Verify HTML is valid (can open in browser)
   - Verify CSS is embedded correctly
   - Verify JavaScript is embedded correctly

### Regression Tests

1. **All Reporters Still Work**
   - Verify console reporter works
   - Verify JSON reporter works
   - Verify JUnit XML reporter works
   - Verify HTML reporter works (refactored)

2. **Existing Tests Pass**
   - Run full test suite
   - Verify all tests pass (89/89)
   - Verify no reporter-related failures

3. **Backward Compatibility**
   - Verify HTML report format unchanged
   - Verify report file location unchanged
   - Verify report content structure unchanged

### Edge Cases

1. **Empty Test Results**
   - Generate report with 0 tests
   - Verify no crashes
   - Verify report shows "no tests" message

2. **Large Test Suites**
   - Generate report with 100+ tests
   - Verify rendering performance acceptable
   - Verify no memory issues

3. **All Status Combinations**
   - Tests with: passed, failed, skipped, warned, error
   - Verify all status icons display
   - Verify all status colors correct
   - Verify all status badges correct

---

## Acceptance Criteria

- [ ] No file exceeds 300 lines
- [ ] HTML output unchanged (behavioral parity)
- [ ] All existing tests pass
- [ ] HTML report functionality intact (expand/collapse, filter, sort)
- [ ] All 5 modules created with proper exports

---

## Test Commands

```bash
# Run all tests (verify no regressions)
npm test

# Check file sizes
wc -l src/reporters/html/*.ts

# Verify no file exceeds 300 lines
for file in src/reporters/html/*.ts; do
  lines=$(wc -l < "$file")
  if [ $lines -gt 300 ]; then
    echo "ERROR: $file has $lines lines (max 300)"
    exit 1
  fi
done
echo "All files under 300 lines"

# Generate HTML report and verify
npx e2e run --reporter html --env local
ls -la reports/

# Verify HTML is valid (if tidy available)
tidy -q reports/index.html

# Diff HTML output (before/after refactoring)
# 1. Generate before-refactor output
# 2. Commit refactored code
# 3. Generate after-refactor output
# 4. diff before.html after.html
# 5. Verify no differences (excluding timestamps)

# Verify imports work
node -e "const { HTMLReporter } = require('./dist/reporters/html/reporter.js'); console.log('Import works')"

# Verify all reporters work
npx e2e run --reporter console --env local
npx e2e run --reporter json --env local
npx e2e run --reporter junit --env local
npx e2e run --reporter html --env local
```

---

## Files Changed (Expected)

- Create: `src/reporters/html/styles.ts` (~280 lines)
- Create: `src/reporters/html/scripts.ts` (~150 lines)
- Create: `src/reporters/html/templates.ts` (~250 lines)
- Create: `src/reporters/html/utils.ts` (~100 lines)
- Rename: `src/reporters/html.reporter.ts` → `src/reporters/html/reporter.ts` (<200 lines)
- Modify: `src/reporters/index.ts` (update import path)

---

## Expected Test Results

**Pre-implementation:**
- html.reporter.ts: 1054 lines (monolithic)
- HTML modules directory: DOES NOT EXIST

**Post-implementation:**
- html/reporter.ts: <200 lines
- html/styles.ts: ~280 lines (<300)
- html/scripts.ts: ~150 lines (<300)
- html/templates.ts: ~250 lines (<300)
- html/utils.ts: ~100 lines (<300)
- HTML output: IDENTICAL to before refactoring
- All tests pass: 89/89
- All reporters work: YES

---

## Known Limitations / Cannot Test

**Environment limitation:** Node.js/npm not available in current environment. Cannot execute test commands or verify HTML output parity.

Test plan is complete and ready for execution once test environment is available.
