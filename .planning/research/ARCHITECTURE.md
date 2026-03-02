# Architecture Research

**Domain:** E2E Testing Framework (API + Database + Message Queue)
**Researched:** 2026-03-02
**Confidence:** HIGH — Based on direct codebase analysis of all source files; external patterns verified against KafkaJS official docs and established parallel execution research.

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                         CLI Layer                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │  run.cmd │  │valid.cmd │  │list.cmd  │  │health.cmd│  ...    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘         │
└───────┼─────────────┼─────────────┼──────────────┼───────────────┘
        │             │             │              │
┌───────▼─────────────▼─────────────▼──────────────▼───────────────┐
│                         Core Layer                                │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                   TestOrchestrator                        │    │
│  │  (p-limit concurrency, event emission, phase lifecycle)   │    │
│  └──────────────┬─────────────────────────────┬─────────────┘    │
│                 │                             │                   │
│  ┌──────────────▼──────┐     ┌────────────────▼────────────────┐  │
│  │   ContextFactory    │     │         StepExecutor            │  │
│  │ (per-test isolated  │     │  (interpolate → dispatch →      │  │
│  │  captured/variables)│     │   assert → capture)             │  │
│  └─────────────────────┘     └────────────────┬────────────────┘  │
│                                               │                   │
│  ┌────────────────────────────────────────────▼────────────────┐  │
│  │              Assertion Engine (assertions/)                  │  │
│  │   assertion-runner.ts  ·  jsonpath.ts  ·  matchers.ts       │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │                  test-discovery.ts                          │   │
│  │  (sortTestsByDependencies · filterByTags · filterByGrep)    │   │
│  └─────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
        │
┌───────▼──────────────────────────────────────────────────────────┐
│                      Adapters Layer                               │
│  ┌──────┐  ┌────────┐  ┌────────┐  ┌───────┐  ┌──────────┐      │
│  │ HTTP │  │Postgres│  │MongoDB │  │ Redis │  │EventHub  │      │
│  └──────┘  └────────┘  └────────┘  └───────┘  └──────────┘      │
│                                                                   │
│  [NEW]  ┌────────────────────────────────────┐                    │
│         │   KafkaAdapter (to be added)        │                   │
│         │   produce · consume · waitFor       │                   │
│         └────────────────────────────────────┘                    │
└──────────────────────────────────────────────────────────────────┘
        │
┌───────▼──────────────────────────────────────────────────────────┐
│                     Reporters Layer                               │
│  ┌─────────┐  ┌───────┐  ┌──────┐  ┌──────┐                     │
│  │ Console │  │ JUnit │  │ HTML │  │ JSON │                     │
│  └─────────┘  └───────┘  └──────┘  └──────┘                     │
└──────────────────────────────────────────────────────────────────┘

Foundation: src/types.ts · src/errors.ts (no internal deps)
Utils:      src/utils/logger.ts · src/utils/retry.ts · src/utils/exit-codes.ts
```

### Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|----------------|-------------------|
| `TestOrchestrator` | Test lifecycle, p-limit concurrency, event emission, bail logic | StepExecutor, ContextFactory, Reporters (via events) |
| `ContextFactory` | Creates per-test isolated `TestContext` — fresh `captured{}` and merged `variables` per test | AdapterRegistry, LoadedConfig |
| `StepExecutor` | Interpolate params → dispatch to adapter → call assertion engine → return StepResult | AdapterRegistry, assertion-runner, variable-interpolator |
| `assertion-runner.ts` | Centralized assertion logic: equals, contains, matches, exists, type, length, greaterThan, lessThan, isNull | Used by StepExecutor (pending wiring) and HTTP adapter (already wired for JSON assertions) |
| `AdapterRegistry` | Lifecycle manager: only init adapters required by discovered tests, connect/disconnect all | All adapter implementations |
| `BaseAdapter` | Contract for execute(action, params, ctx), connect, disconnect, healthCheck | AdapterContext, AdapterStepResult |
| `test-discovery.ts` | Find test files, filter by tags/priority/grep, `sortTestsByDependencies` (DFS topological sort — exists but never called) | yaml-loader, ts-loader |
| `ReporterManager` | Bridge from orchestrator events to reporter implementations | All reporters, run.command.ts |

## Current Architecture Gaps (The Bugs to Fix)

These are precise, localized disconnects — the architectural skeleton is sound.

### Gap 1: Assertion Engine Not Wired in StepExecutor

**Current state:** `StepExecutor.validateAssertions()` (line 240) is a dead stub. It logs "Assertions pending validation" and does nothing. The `assertion-runner.ts` module exists and is complete with all operators.

**Current call path (broken):**
```
StepExecutor.executeStep()
  → step.assert exists?
  → this.validateAssertions(step.assert, adapterResult.data, step.id)  ← stub, no-op
```

**Missing:** The stub needs to call `assertion-runner.ts` functions. The HTTP adapter already does this correctly for its own JSON path assertions — the fix is making StepExecutor do the same for the general case.

**What assertion-runner.ts already supports:**
- `runAssertion(value, assertion, path)` — evaluates a single `BaseAssertion` object against a value
- `BaseAssertion` covers: equals, contains, matches, exists, type, length, greaterThan, lessThan, notEmpty, isEmpty, isNull, isNotNull
- JSONPath extraction lives in `assertions/jsonpath.ts` — `evaluateJSONPath(data, path)`

**Fix target:** `StepExecutor.validateAssertions()` — replace the stub with a real implementation that:
1. Accepts assertions as `Record<string, BaseAssertion>` where the key is a JSONPath
2. For each `{path: assertion}` pair, calls `evaluateJSONPath(data, path)` then `runAssertion(value, assertion, path)`
3. Throws `AssertionError` (already typed) on first failure

### Gap 2: continueOnError Returns 'passed' Instead of Distinct Status

**Current state:** `StepExecutor.executeStep()` (line 141-148) returns `'passed'` status when a step fails with `continueOnError: true`.

**Fix target:** Add `'skipped'` or `'warned'` to `StepStatus` type in `types.ts`, then use it in the `continueOnError` branch. The `allStepsPassed()` utility at the bottom of `step-executor.ts` must also be updated.

### Gap 3: Test Dependency Ordering Never Applied

**Current state:** `sortTestsByDependencies()` exists in `test-discovery.ts` (line 277-312) — a complete DFS topological sort that detects cycles. It is never called in `run.command.ts`.

**`UnifiedTestDefinition.depends?: string[]`** — the field exists in `types.ts` (line 90). The YAML schema accepts it. But after tests are loaded in `run.command.ts` (`loadTestDefinitions()` call), nothing sorts them.

**Fix target:** In `run.command.ts`, after `loadTestDefinitions()`, call:
```typescript
const sortedDefinitions = sortTestsByDependencies(
  definitionsAsList,
  (def) => def.depends || []
)
```
Note: `sortTestsByDependencies` currently takes `DiscoveredTest[]` not `UnifiedTestDefinition[]`. The function needs to be generalized, or a parallel function needs to be added to `test-discovery.ts` that operates on `UnifiedTestDefinition[]` directly (since `depends` is only available after full loading).

### Gap 4: Shared Mutable Instance State in TestOrchestrator

**Current state:** `TestOrchestrator` has three mutable instance fields used for event context during parallel execution (lines 90-93):
```typescript
private currentTestIndex: number = 0
private totalTests: number = 0
private currentTest: UnifiedTestDefinition | null = null
private currentPhase: TestPhase | null = null
```

When `parallel > 1`, multiple tests run concurrently via `p-limit`. Each concurrent invocation of `runTest()` overwrites `this.currentTest` and `this.currentPhase`. Events emitted by one test's phase execution can carry stale context from another test.

**Fix target:** Pass test/phase context explicitly as parameters through the call chain instead of storing on `this`. The `runPhase()`, `emit()`, and helper methods should accept `currentTest` and current phase as parameters. No `this.currentTest` mutation needed.

### Gap 5: Kafka Adapter Missing

**Current state:** `AdapterType` in `types.ts` is `'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http'`. No Kafka support exists anywhere.

**Fix target:** New `kafka.adapter.ts` + extend `AdapterType`, `EnvironmentConfig`, and `AdapterRegistry`.

## Recommended Project Structure (No Changes Needed)

The existing structure is the right shape. New work slots into existing folders:

```
src/
├── adapters/
│   ├── base.adapter.ts          # Unchanged
│   ├── adapter-registry.ts      # Add 'kafka' to switch + getRequiredAdapters
│   ├── http.adapter.ts          # Unchanged
│   ├── postgresql.adapter.ts    # Unchanged
│   ├── mongodb.adapter.ts       # Fix ObjectId import
│   ├── redis.adapter.ts         # Fix KEYS → SCAN
│   ├── eventhub.adapter.ts      # Fix processError resolve→reject
│   └── kafka.adapter.ts         # [NEW] KafkaJS-backed adapter
├── assertions/
│   ├── assertion-runner.ts      # Exists and complete — no changes needed
│   ├── jsonpath.ts              # Exists and complete — no changes needed
│   ├── matchers.ts              # Exists
│   └── expect.ts                # Exists
├── core/
│   ├── step-executor.ts         # Fix validateAssertions stub + continueOnError status
│   ├── test-orchestrator.ts     # Fix shared mutable state (currentTest, currentPhase)
│   ├── test-discovery.ts        # Add sortUnifiedTestsByDependencies variant
│   ├── context-factory.ts       # Unchanged — already correct per-test isolation
│   └── variable-interpolator.ts # Unchanged
├── cli/
│   └── run.command.ts           # Wire dependency sorting after loadTestDefinitions
└── types.ts                     # Add 'kafka' to AdapterType, add KafkaAdapterConfig
```

## Architectural Patterns

### Pattern 1: Centralized Assertion Engine (Fix for Gap 1)

**What:** StepExecutor calls `assertion-runner.ts` for all non-HTTP adapters. HTTP adapter keeps its own extended assertion logic (status codes, headers, duration) and delegates JSON body assertions to the shared runner.

**When to use:** Any time a step has an `assert` field.

**The correct integration point:**

```typescript
// In StepExecutor.validateAssertions() — replacing the stub
private validateAssertions(
  assertions: Record<string, BaseAssertion>,
  data: unknown,
  stepId: string
): void {
  for (const [path, assertion] of Object.entries(assertions)) {
    const value = path === '.' || path === '$'
      ? data
      : evaluateJSONPath(data, path)
    runAssertion(value, assertion, path)  // throws AssertionError on failure
  }
}
```

**Why this works without breaking HTTP:** The HTTP adapter runs its own `runAssertions()` before returning `successResult()`. The `AdapterStepResult` it returns has `success: true` and `data: { request, response }`. StepExecutor only calls `validateAssertions` on `adapterResult.data` after the adapter returns — for HTTP, the adapter already threw if assertions failed, so StepExecutor's assertion call is on the rich `{request, response}` object with any user-defined path assertions.

**Confidence: HIGH** — direct code analysis confirms assertion-runner.ts is complete and `runAssertion` + `evaluateJSONPath` are both exported.

### Pattern 2: Kafka Adapter (Produce/Consume/WaitFor)

**What:** A `KafkaAdapter` extending `BaseAdapter` that uses `kafkajs` as its backing library. The adapter manages one `Producer` and one `Consumer` instance per connection lifecycle.

**KafkaJS API surface used** (confirmed against official docs at kafka.js.org):

```typescript
// Connection: kafka constructor accepts { clientId, brokers[] }
const kafka = new Kafka({ clientId: 'e2e-runner', brokers: config.brokers })

// Producer lifecycle
const producer = kafka.producer()
await producer.connect()          // in connect()
await producer.send({ topic, messages: [{ key, value }] })  // in execute('produce')
await producer.disconnect()       // in disconnect()

// Consumer lifecycle
const consumer = kafka.consumer({ groupId: config.consumerGroup })
await consumer.connect()          // in connect()
await consumer.subscribe({ topic, fromBeginning })  // in execute('consume')
// consumer.run() is the long-running handler — for testing, use Promise + timeout
await consumer.disconnect()       // in disconnect()
```

**Three actions the adapter must support:**

| Action | Params | Behavior |
|--------|--------|----------|
| `produce` | `topic`, `messages[]` (with `key?`, `value`, `headers?`) | Send one or more messages; return `{ topic, partition, offset }` per message |
| `consume` | `topic`, `fromBeginning?`, `count?`, `timeout?` | Subscribe + collect N messages within timeout; return `messages[]` |
| `waitFor` | `topic`, `predicate` (JSONPath + assertion), `timeout?`, `maxMessages?` | Poll until a message matching the predicate arrives or timeout; return matching message |

**Consume/WaitFor implementation pattern:**

```typescript
// For 'consume' and 'waitFor' — cannot use consumer.run() indefinitely in tests
// Pattern: Promise that resolves when condition is met, consumer.disconnect() cancels

private async consumeMessages(
  topic: string,
  count: number,
  timeoutMs: number
): Promise<Message[]> {
  const collected: Message[] = []

  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new TimeoutError(`No messages received from ${topic} within ${timeoutMs}ms`))
    }, timeoutMs)

    this.consumer.run({
      eachMessage: async ({ message }) => {
        collected.push(message)
        if (collected.length >= count) {
          clearTimeout(timer)
          resolve()
        }
      }
    }).catch(reject)
  })

  return collected
}
```

**Config shape** (to add to `types.ts`):

```typescript
export interface KafkaAdapterConfig {
  brokers: string[]         // e.g. ['localhost:9092']
  clientId?: string         // defaults to 'e2e-runner'
  consumerGroup?: string    // defaults to 'e2e-runner-consumer'
  ssl?: boolean
  sasl?: { mechanism: 'plain'; username: string; password: string }
}
```

**Why KafkaJS:** Most popular Node.js Kafka client (7M+ weekly npm downloads), excellent TypeScript types, clean async API with Promise-based producer and callback-based consumer that fits the BaseAdapter lifecycle pattern. Confirmed active maintenance via npm registry.

**Confidence: MEDIUM** — KafkaJS API verified against official docs; test-specific waitFor pattern is a standard approach but the exact Promise+disconnect cancellation needs validation.

### Pattern 3: Test Dependency Graph with Parallel Execution

**What:** A topological sort that respects `depends` while still maximizing parallelism — tests with no inter-dependencies run concurrently, dependent tests are deferred until their prerequisites complete.

**Current state:** `sortTestsByDependencies` in `test-discovery.ts` produces a flat sorted `DiscoveredTest[]`. This is fine for sequential execution (runs A before B before C) but discards parallelism opportunities.

**Two-phase approach (build order implication):**

Phase A — fix the basic case: Wire `sortTestsByDependencies` to operate on `UnifiedTestDefinition[]`. Since `depends` names other tests, and the names are only available in the full definition, sorting must happen after loading.

```typescript
// Add to test-discovery.ts
export function sortUnifiedTestsByDependencies(
  tests: UnifiedTestDefinition[]
): UnifiedTestDefinition[] {
  const testMap = new Map(tests.map((t) => [t.name, t]))
  const sorted: UnifiedTestDefinition[] = []
  const visited = new Set<string>()
  const visiting = new Set<string>()

  function visit(test: UnifiedTestDefinition): void {
    if (visited.has(test.name)) return
    if (visiting.has(test.name)) {
      throw new ConfigurationError(
        `Circular dependency detected: ${test.name}`,
        { code: 'CIRCULAR_DEPENDENCY' }
      )
    }
    visiting.add(test.name)
    for (const depName of (test.depends || [])) {
      const dep = testMap.get(depName)
      if (dep) visit(dep)
      // Missing deps: log warning, don't crash (dep may be in a different file)
    }
    visiting.delete(test.name)
    visited.add(test.name)
    sorted.push(test)
  }

  for (const test of tests) visit(test)
  return sorted
}
```

Phase B (future): Level-based grouping for parallel execution:
```
Tests with no deps → Level 0 (run in parallel)
Tests whose only deps are in Level 0 → Level 1 (run in parallel after Level 0 completes)
```
This requires a second pass to compute in-degree levels (Kahn's algorithm). This is the right long-term pattern but is additional work beyond the current milestone scope.

**Wire location:** In `run.command.ts` after `loadTestDefinitions()`, before `getRequiredAdapters()`:

```typescript
const sortedDefinitions = sortUnifiedTestsByDependencies(definitions)
const requiredAdapters = getRequiredAdapters(sortedDefinitions)
```

**Confidence: HIGH** — topological sort algorithm is confirmed. The existing DFS implementation in `test-discovery.ts` is correct; the only gap is it operates on `DiscoveredTest` not `UnifiedTestDefinition`.

### Pattern 4: Safe Parallel State Management

**What:** Eliminate shared mutable instance fields on `TestOrchestrator` that cause data races at `parallel > 1`.

**Root cause:** `runTest()` sets `this.currentTest = test` at the start, then `this.emit()` reads `this.currentTest` during phase execution. With concurrent tests, another `runTest()` call overwrites `this.currentTest` mid-emission.

**Fix:** Pass context explicitly through the call chain. No instance mutation.

```typescript
// Before (broken at parallel > 1):
async runTest(test: UnifiedTestDefinition): Promise<TestExecutionResult> {
  this.currentTest = test           // shared mutable state
  const testIndex = this.currentTestIndex++  // shared mutable state
  this.emit('phase:start', {
    testName: this.currentTest?.name  // reads shared state, may be stale
  })
}

// After (safe):
async runTest(test: UnifiedTestDefinition): Promise<TestExecutionResult> {
  const testIndex = this.currentTestIndex++  // atomic read+increment is fine
  // Pass test identity explicitly — no this.currentTest
  await this.executeTestPhases(test, context, phases, testIndex)
}

private async runPhase(
  phaseName: TestPhase,
  steps: UnifiedStep[],
  context: TestContext,
  testName: string,   // explicit parameter, not this.currentTest
  testIndex: number,
): Promise<PhaseResult> {
  this.emit('phase:start', { testName, phase: phaseName, timestamp: new Date() })
  // ...
}
```

**Additional fix:** `this.currentTestIndex++` is not atomic in single-threaded JS but is still safe because JS is single-threaded (no true parallelism). The race is only between the async continuation points, and `currentTestIndex` is only read/written in the synchronous portion before any `await`. This is safe to leave as-is.

**What is unsafe:** `this.currentTest` and `this.currentPhase` — these are set before an `await`, then read after the `await` resumes, by which point another parallel test may have overwritten them. This is the exact pattern that breaks.

**Confidence: HIGH** — JavaScript event loop behavior is well-understood. The specific mutation pattern is textbook async race condition.

## Data Flow

### Assertion Data Flow (Current vs Fixed)

```
Current (broken):
  Step YAML { assert: { "$.id": { exists: true } } }
    → yaml-loader separates assert from params
    → StepExecutor.executeStep(): if step.assert → this.validateAssertions() [STUB - no-op]
    → assertions never run

Fixed:
  Step YAML { assert: { "$.id": { exists: true } } }
    → yaml-loader separates assert from params (unchanged)
    → StepExecutor.executeStep(): if step.assert → this.validateAssertions() [REAL]
      → evaluateJSONPath(adapterResult.data, "$.id") → value
      → runAssertion(value, { exists: true }, "$.id")
      → AssertionError thrown if fails → propagates up → StepResult.status = 'failed'
```

### Kafka Adapter Data Flow

```
Test YAML step:
  adapter: kafka
  action: produce
  params: { topic: "orders", messages: [{ value: "{{order_id}}" }] }
    ↓
  StepExecutor interpolates params → { topic: "orders", messages: [{ value: "abc-123" }] }
    ↓
  AdapterRegistry.get('kafka') → KafkaAdapter.execute('produce', params, ctx)
    ↓
  producer.send({ topic: "orders", messages: [{value: "abc-123"}] })
    ↓
  Returns AdapterStepResult { success: true, data: { offsets: [...] } }

Test YAML step:
  adapter: kafka
  action: waitFor
  params: { topic: "order-events", predicate: { "$.type": { equals: "OrderCreated" } }, timeout: 5000 }
    ↓
  KafkaAdapter.execute('waitFor', params, ctx)
    ↓
  consumer.subscribe({ topic: "order-events", fromBeginning: false })
    ↓
  Promise race: consumer.run(eachMessage) vs setTimeout(timeout)
    ↓
  On each message: evaluateJSONPath(message.value, "$.type") + runAssertion()
    ↓
  Matching message found → resolve with message → AdapterStepResult.data = message
  Timeout exceeded → TimeoutError thrown → AdapterStepResult.success = false
```

### Dependency-Ordered Parallel Execution Data Flow

```
Discovery: DiscoveredTest[]  (name only, no depends)
    ↓
loadTestDefinitions(): UnifiedTestDefinition[]  (has depends field)
    ↓
sortUnifiedTestsByDependencies(): UnifiedTestDefinition[]  (topologically ordered)
    ↓
p-limit(parallel): executes tests in order but concurrently up to parallelism limit
    Note: topological order guarantees A finishes before B starts IF parallel=1
    At parallel > 1: not guaranteed unless level-based grouping is implemented (future)
    For current milestone: sort + parallel=1 for dependent tests is acceptable
```

### Per-Test State Isolation Flow

```
For each test (concurrent via p-limit):
  ContextFactory.createTestContext(test)
    → captured = {}          (new object per test, no sharing)
    → variables = { ...globalVars, ...test.variables }  (merged, immutable source)
    → capture = (name, val) => captured[name] = val   (closure over local captured)

  AdapterContext = { variables, captured, capture, baseUrl, logger }
    → Passed to adapter.execute() and step assertions
    → captured mutations are local to this test's closure

  Result: TestExecutionResult.capturedValues = context.captured  (this test only)
```

This is already correct. The `TestOrchestrator` bugs are about its *own* state for event emission, not about the `TestContext` per-test state, which is already correctly isolated.

## Anti-Patterns

### Anti-Pattern 1: Adapter-Level Assertions as the Primary Path

**What people do:** Each adapter implements its own assertion logic (as EventHub and PostgreSQL partially do) — duplicating operators like `equals`, `contains`, `exists`.

**Why it's wrong:** Logic drift between adapters, assertion operators not available on all adapters, tests behave differently depending on which adapter they use.

**Do this instead:** All adapters return raw data from `execute()`. StepExecutor calls the shared `assertion-runner.ts` on the result. HTTP adapter is the accepted exception because HTTP assertions (status codes, headers, duration) are inherently HTTP-specific — but it delegates JSON body assertions to the shared runner.

### Anti-Pattern 2: Passing Assertions Through Adapter Params

**What people do (current workaround in StepExecutor):** StepExecutor copies `step.assert` into `interpolatedParams.assert` before calling `adapter.execute()`. This requires every adapter to unpack and re-run assertions.

**Why it's wrong:** Leaks test framework concerns (assertions) into the adapter layer. Adapters should not know about assertions — they should just execute operations and return data.

**Do this instead:** Fix `StepExecutor.validateAssertions()` to call `assertion-runner.ts` directly. Stop injecting `assert` into adapter params. This requires updating the HTTP adapter to not expect `params.assert` (or keep backward compat by reading it, but the canonical path is through StepExecutor).

### Anti-Pattern 3: Consumer.run() Without a Completion Signal for Tests

**What people do:** Call `consumer.run()` and wait for messages indefinitely — works in production services, hangs test steps.

**Why it's wrong:** Test steps need deterministic completion. A test step must return a `StepResult` within a bounded time.

**Do this instead:** Wrap `consumer.run()` in a `Promise` that resolves when the expected message count or predicate is satisfied, and reject on timeout. After resolution, call `consumer.disconnect()` to stop the consumer. This is the correct pattern for a test adapter context.

### Anti-Pattern 4: Mutating Orchestrator State for Event Context

**What people do (current TestOrchestrator):** Store `this.currentTest` and `this.currentPhase` as instance fields, mutate them in `runTest()`, read them in `runPhase()` and `emit()`.

**Why it's wrong:** When `parallel > 1`, `p-limit` runs multiple `runTest()` calls concurrently. Between the `await` calls within a single `runTest()`, the JS event loop can switch to another `runTest()` that overwrites `this.currentTest`. Emitted events carry the wrong test name.

**Do this instead:** Thread `testName`, `testIndex`, and `phaseName` as explicit parameters through the call chain. Each concurrent invocation carries its own context on the stack, never on the heap.

## Component Boundaries (Build Order for This Milestone)

The dependency graph between the fixes determines build order:

```
1. types.ts
   ├── Add 'kafka' to AdapterType
   ├── Add KafkaAdapterConfig to EnvironmentConfig.adapters
   └── Add 'warned' status to StepStatus (for continueOnError fix)
         ↓
2. assertion-runner.ts (no changes needed — already complete)
         ↓
3. step-executor.ts
   ├── Fix validateAssertions() stub → call runAssertion + evaluateJSONPath
   └── Fix continueOnError status → use 'warned' not 'passed'
         ↓
4. test-orchestrator.ts
   └── Fix shared mutable state → pass testName/phase explicitly
         ↓
5. test-discovery.ts
   └── Add sortUnifiedTestsByDependencies() for UnifiedTestDefinition[]
         ↓
6. kafka.adapter.ts  (new file — depends on types.ts changes)
   ├── BaseAdapter.connect/disconnect lifecycle
   ├── execute('produce') → producer.send()
   ├── execute('consume') → Promise-wrapped consumer.run()
   └── execute('waitFor') → predicate-filtered consume with timeout
         ↓
7. adapter-registry.ts
   └── Add 'kafka' instantiation path (after KafkaAdapterConfig in types.ts)
         ↓
8. run.command.ts
   └── Wire sortUnifiedTestsByDependencies after loadTestDefinitions
```

Steps 3 and 4 are independent (fix assertion engine vs fix orchestrator state). Steps 3, 4, 5 can be done in parallel with 6 (Kafka adapter). Step 7 requires both types.ts (step 1) and kafka.adapter.ts (step 6). Step 8 requires test-discovery.ts (step 5).

## Integration Points

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| StepExecutor → assertion-runner | Direct function call: `runAssertion(value, assertion, path)` | assertion-runner throws; StepExecutor catches as step failure |
| StepExecutor → adapters | `AdapterRegistry.get(step.adapter).execute(action, params, ctx)` | Adapter throws on failure; StepExecutor handles retry |
| Orchestrator → Reporters | Event emission via `this.emit()` → `OrchestratorEventListener[]` | Observer pattern, reporters are decoupled |
| run.command.ts → sortUnifiedTestsByDependencies | Direct import from test-discovery | Call site: after `loadTestDefinitions()`, before `getRequiredAdapters()` |
| KafkaAdapter → AdapterRegistry | `AdapterRegistry.initializeAdapters()` adds `new KafkaAdapter(config, logger)` | Follows exact same pattern as EventHub, PostgreSQL |

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Kafka broker | KafkaJS client: `new Kafka({ brokers })` | `kafkajs` must be added as optional peer dependency in package.json |
| KafkaJS | Producer: connect → send → disconnect | Single producer instance per adapter lifecycle |
| KafkaJS | Consumer: connect → subscribe → run (Promise-wrapped) → disconnect | Consumer must be disconnected after each consume/waitFor call to release group membership |

## Sources

- Direct codebase analysis: all source files in `src/` read 2026-03-02 — HIGH confidence
- KafkaJS official documentation: [Consuming Messages](https://kafka.js.org/docs/consuming), [Getting Started](https://kafka.js.org/docs/getting-started) — MEDIUM confidence for consumer lifecycle patterns
- JavaScript event loop and async concurrency — HIGH confidence (well-established)
- Topological sort algorithm (DFS) — HIGH confidence; existing implementation in `test-discovery.ts` is correct
- Parallel test state isolation patterns — MEDIUM confidence from multiple framework sources (JUnit5, pytest, xunit.net) all converge on "explicit parameters over shared instance state"

---
*Architecture research for: E2E Testing Framework — Bug Fixes and Kafka Adapter*
*Researched: 2026-03-02*
