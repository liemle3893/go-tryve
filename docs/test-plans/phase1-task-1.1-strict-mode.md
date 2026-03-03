# Task 1.1: TypeScript Strict Mode - Test Plan

**Priority:** P0 (Critical)  
**Dependency:** Phase 0 complete  
**Effort Estimate:** 16-24 hours

---

## Test Scenarios

### Unit Tests

1. **TypeScript Compilation**
   - Verify `npm run build` succeeds with zero errors
   - Verify strict mode flags are enabled in `tsconfig.json`
   - Test: All 45 TypeScript source files compile

2. **Type Safety Verification**
   - No implicit `any` types remain
   - Null checks are properly implemented
   - Unused variables/parameters removed
   - All return paths are explicit

### Integration Tests

1. **Behavioral Parity**
   - Run full test suite: `npm test`
   - All existing tests pass (89/89 from Phase 0)
   - No changes to test behavior or results

2. **Type Errors in Source Code**
   - Attempt to add `any` type → TypeScript error
   - Attempt to remove null check → TypeScript error
   - Attempt to unused variable → TypeScript error

### Regression Tests

1. **Phase 0 Fixes Remain Functional**
   - Task 0.1: Assertion engine runs assertions (not stub)
   - Task 0.2: `continueOnError: true` shows `warned` status
   - Task 0.3: EventHub errors fail the step
   - Task 0.4: Retry count correctly reported
   - Task 0.5: TypeScript adapter type works
   - Task 0.6: Redis uses SCAN not KEYS
   - Task 0.7: MongoDB ObjectId imported once

### Edge Cases

1. **Complex Type Inference**
   - Generic functions with strict mode
   - Union types with null checks
   - Async/await return types
   - Promise type handling

---

## Acceptance Criteria

- [ ] All strict flags enabled in `tsconfig.json`
- [ ] Zero TypeScript compilation errors (`npm run build` exits with code 0)
- [ ] All existing tests pass (89/89)
- [ ] No behavioral changes to existing functionality
- [ ] No `any` types, implicit returns, or null check bypasses

---

## Test Commands

```bash
# Verify strict mode enabled
grep '"strict": true' tsconfig.json

# Verify all strict flags
grep '"noImplicitAny": true' tsconfig.json
grep '"noUnusedLocals": true' tsconfig.json
grep '"noUnusedParameters": true' tsconfig.json
grep '"noImplicitReturns": true' tsconfig.json
grep '"strictNullChecks": true' tsconfig.json
grep '"strictFunctionTypes": true' tsconfig.json
grep '"strictBindCallApply": true' tsconfig.json
grep '"strictPropertyInitialization": true' tsconfig.json

# Verify build succeeds
npm run build
exit_code=$?
if [ $exit_code -ne 0 ]; then
  echo "FAILED: TypeScript compilation errors"
  exit 1
fi

# Verify all tests pass
npm test

# Verify no type errors
npx tsc --noEmit
```

---

## Files Changed (Expected)

- Modify: `tsconfig.json`
- Modify: Multiple files in `src/` (~45 files estimated)

---

## Expected Test Results

**Pre-implementation:**
- TypeScript compilation: SUCCEEDS (strict mode disabled)
- Test suite: 89/89 PASS

**Post-implementation:**
- TypeScript compilation: SUCCEEDS (strict mode enabled, all errors fixed)
- Test suite: 89/89 PASS (no behavioral changes)
- Strict mode flags: All enabled
- Type errors: 0

---

## Known Limitations / Cannot Test

**Environment limitation:** Node.js/npm not available in current environment. Cannot execute test commands.

Test plan is complete and ready for execution once test environment is available.
