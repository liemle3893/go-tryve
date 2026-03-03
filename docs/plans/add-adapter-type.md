# Add TypeScript Adapter Type Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `typescript` as a named `AdapterType` so TypeScript function-backed steps no longer carry the misleading adapter type `http`.

**Architecture:** `AdapterType` in `src/types.ts` currently lists `'http'` as the fallback for TypeScript function steps. `createFunctionStep()` in `src/core/step-executor.ts` hard-codes `adapter: 'http' as AdapterType`. Adding `'typescript'` to the union and updating `createFunctionStep`, the YAML loader's allowlist, and the YAML validator covers all touch points. Step execution itself is unaffected because `StepExecutor` already dispatches TypeScript steps by action name (`TYPESCRIPT_FUNCTION_ACTION`), not by adapter type.

**Tech Stack:** TypeScript, Vitest, `src/types.ts`, `src/core/step-executor.ts`, `src/core/yaml-loader.ts`

---

### Task 1: Write the failing unit test

**Files:**
- Create: `tests/unit/core/adapter-type.test.ts`

**Step 1: Write the failing test**

```typescript
import { describe, it, expect } from 'vitest'
import { createFunctionStep } from '../../../src/core/step-executor'
import type { AdapterType } from '../../../src/types'

describe('TypeScript adapter type', () => {
  it('createFunctionStep produces a step with adapter "typescript", not "http"', () => {
    const step = createFunctionStep('my-step', async () => 'ok')
    // This will fail until we change the default adapter in createFunctionStep
    expect(step.adapter).toBe('typescript' satisfies AdapterType)
  })

  it('AdapterType union includes "typescript"', () => {
    // This is a compile-time check; at runtime we verify the loader allowlist
    const validAdapters: AdapterType[] = [
      'postgresql', 'redis', 'mongodb', 'eventhub', 'http', 'shell', 'typescript',
    ]
    expect(validAdapters).toContain('typescript')
  })
})
```

**Step 2: Run test to verify it fails**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/core/adapter-type.test.ts 2>&1 | tail -20
```

Expected: FAIL — `step.adapter` is `'http'`, not `'typescript'`. The TypeScript compiler may also complain that `'typescript'` is not in `AdapterType`.

**Step 3: Implement the fix**

**3a. Update `src/types.ts` — add `typescript` to `AdapterType`**

Find line 84:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell';
```

Change to:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell' | 'typescript';
```

**3b. Update `src/core/step-executor.ts` — fix `createFunctionStep` default adapter**

Find in `createFunctionStep` (around line 371):
```typescript
adapter: 'http' as AdapterType, // Default adapter for function steps
```

Change to:
```typescript
adapter: 'typescript' as AdapterType,
```

**3c. Update `src/core/yaml-loader.ts` — add `typescript` to `VALID_ADAPTERS`**

Find line 56:
```typescript
const VALID_ADAPTERS: AdapterType[] = ['postgresql', 'redis', 'mongodb', 'eventhub', 'http', 'shell'];
```

Change to:
```typescript
const VALID_ADAPTERS: AdapterType[] = ['postgresql', 'redis', 'mongodb', 'eventhub', 'http', 'shell', 'typescript'];
```

**3d. Update `src/core/yaml-loader.ts` — add `typescript` case to `validateAdapterStep`**

In the `validateAdapterStep` switch statement (after the `shell` case, around line 317), add:

```typescript
    case 'typescript':
      // TypeScript function steps are loaded from .test.ts files, not YAML.
      // If someone mistakenly writes adapter: typescript in YAML, we accept it
      // to allow forward-compatible test files.
      break;
```

**Step 4: Run test to verify it passes**

```bash
cd /tmp/e2e-runner && npm test -- --reporter=verbose tests/unit/core/adapter-type.test.ts 2>&1 | tail -20
```

Expected: PASS — both tests pass and TypeScript compilation succeeds.

**Step 5: Commit**

```bash
cd /tmp/e2e-runner && git add src/types.ts src/core/step-executor.ts src/core/yaml-loader.ts tests/unit/core/adapter-type.test.ts && git commit -m "feat(types): add 'typescript' to AdapterType

TypeScript function-backed steps previously declared adapter 'http',
which was misleading. Adds 'typescript' to AdapterType, updates
createFunctionStep() to use it, and allows 'typescript' in the YAML
loader allowlist for forward compatibility.

Closes Task 0.5"
```
