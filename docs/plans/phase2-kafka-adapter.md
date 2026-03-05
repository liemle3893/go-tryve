# Phase 2: Kafka Adapter — Implementation Plan

**Goal:** Users can test Kafka-based event-driven systems by producing messages to topics, consuming messages with assertions, and waiting for specific messages using the waitFor pattern.

**Architecture:** A new `KafkaAdapter` class extends `BaseAdapter` and uses the `kafkajs` library (pure JavaScript, no native dependencies). The adapter supports three actions: `produce` (send messages), `consume` (read messages), and `waitFor` (poll until matching message arrives). Configuration is added to `EnvironmentConfig` and the adapter is registered in `AdapterRegistry`.

**Tech Stack:** TypeScript (existing), vitest (existing), kafkajs (new peer dependency).

---

## Current State

- **Project root:** `/home/goose/e2e-runner`
- **Existing files that matter:**
  - `src/types.ts` — Contains `AdapterType` union and `EnvironmentConfig.adapters` interface; needs `kafka` type and `KafkaAdapterConfig` interface
  - `src/adapters/base.adapter.ts` — Abstract base class with `connect()`, `disconnect()`, `execute()`, `healthCheck()`, and helper methods
  - `src/adapters/adapter-registry.ts` — Registry that instantiates adapters by type; needs Kafka case added
  - `src/adapters/eventhub.adapter.ts` — Reference implementation for message broker adapter with publish/consume/waitFor
  - `src/errors.ts` — Custom error classes (`AdapterError`, `TimeoutError`)
  - `package.json` — Peer dependencies section for optional adapters
- **Assumptions:**
  - Phase 0 (Foundation Fixes) is COMPLETE
  - Phase 1 (Technical Health) is COMPLETE
  - 311 tests passing on main branch
  - TypeScript strict mode enabled
  - Existing adapters (http, postgresql, redis, mongodb, eventhub, shell) are working
- **Design doc:** `docs/plans/e2e-runner-roadmap.md` (Phase 2)

## Constraints

- **No breaking changes**: Existing adapters and tests continue to work
- **Peer dependency**: kafkajs is a peer dependency (optional, like other adapters)
- **Follow existing patterns**: KafkaAdapter follows EventHubAdapter structure for consistency
- **No native dependencies**: Use kafkajs (pure JS) to avoid platform-specific builds
- **Error handling**: Connection failures, timeouts, and consumption errors must fail the step

## Rollback

```bash
git revert <phase-2-commit-range>
# Remove kafkajs from package.json peerDependencies (manual edit)
# No runtime directories created
```

---

## Task 1: Add Kafka Types and Config [independent]

**Files:**
- Modify: `src/types.ts`
- Test: None (type-only change, verified by TypeScript compilation)

### Step 1: Write failing test

N/A — Type-only change. Verified by TypeScript compilation.

### Step 2: Run test to verify it fails

Run: `npm run build`
Expected: Build succeeds (adding new optional type doesn't break anything)

### Step 3: Implement

> In `src/types.ts`, find:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell' | 'typescript';
```
> Replace with:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell' | 'typescript' | 'kafka';
```

> In `src/types.ts`, find the `EnvironmentConfig` interface adapters section:
```typescript
  adapters: {
    postgresql?: PostgreSQLAdapterConfig;
    redis?: RedisAdapterConfig;
    mongodb?: MongoDBAdapterConfig;
    eventhub?: EventHubAdapterConfig;
    shell?: ShellAdapterConfig;
  };
```
> Replace with:
```typescript
  adapters: {
    postgresql?: PostgreSQLAdapterConfig;
    redis?: RedisAdapterConfig;
    mongodb?: MongoDBAdapterConfig;
    eventhub?: EventHubAdapterConfig;
    shell?: ShellAdapterConfig;
    kafka?: KafkaAdapterConfig;
  };
```

> In `src/types.ts`, after `ShellAdapterConfig` interface, add:
```typescript
export interface KafkaAdapterConfig {
  brokers: string | string[];
  clientId?: string;
  ssl?: boolean;
  sasl?: {
    mechanism: 'plain' | 'scram-sha-256' | 'scram-sha-512';
    username: string;
    password: string;
  };
  connectionTimeout?: number;
  requestTimeout?: number;
}
```

### Step 4: Run test to verify it passes

Run: `npm run build`
Expected: Build succeeds with no errors

### Step 5: Commit

```bash
git add src/types.ts
git commit -m "feat(kafka): add Kafka adapter type and config interface"
```

---

## Task 2: Add kafkajs Peer Dependency [independent]

**Files:**
- Modify: `package.json`
- Test: None (dependency declaration only)

### Step 1: Write failing test

N/A — Dependency declaration only.

### Step 2: Run test to verify it fails

Run: `npm install`
Expected: Install succeeds (peer dependency is optional)

### Step 3: Implement

> In `package.json`, find `peerDependencies`, add `"kafkajs": "^2.0.0"` alphabetically.

> In `package.json`, find `peerDependenciesMeta`, add:
```json
    "kafkajs": {
      "optional": true
    },
```

> In `package.json`, add to `devDependencies`:
```json
    "kafkajs": "^2.2.4",
```

### Step 4: Run test to verify it passes

Run: `npm install && npm run build`
Expected: Install and build succeed

### Step 5: Commit

```bash
git add package.json package-lock.json
git commit -m "feat(kafka): add kafkajs as peer dependency"
```

---

