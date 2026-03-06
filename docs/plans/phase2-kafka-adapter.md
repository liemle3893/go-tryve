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

## Task 3: Implement KafkaAdapter Class [depends on: Task 1, Task 2]

**Files:**
- Create: `src/adapters/kafka.adapter.ts`
- Create: `src/adapters/kafka.adapter.test.ts`

### Step 1: Write failing test

Create `src/adapters/kafka.adapter.test.ts`:

```typescript
/**
 * KafkaAdapter Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { KafkaAdapter } from './kafka.adapter';
import type { AdapterConfig, Logger, AdapterContext } from '../types';

// Mock kafkajs
vi.mock('kafkajs', () => ({
  Kafka: vi.fn().mockImplementation(() => ({
    producer: vi.fn().mockReturnValue({
      connect: vi.fn().mockResolvedValue(undefined),
      disconnect: vi.fn().mockResolvedValue(undefined),
      send: vi.fn().mockResolvedValue(undefined),
    }),
    consumer: vi.fn().mockReturnValue({
      connect: vi.fn().mockResolvedValue(undefined),
      disconnect: vi.fn().mockResolvedValue(undefined),
      subscribe: vi.fn().mockResolvedValue(undefined),
      run: vi.fn().mockResolvedValue(undefined),
      stop: vi.fn().mockResolvedValue(undefined),
    }),
  })),
}));

describe('KafkaAdapter', () => {
  let adapter: KafkaAdapter;
  const mockLogger: Logger = {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
  };
  const mockContext: AdapterContext = {
    variables: {},
    captured: {},
    baseUrl: 'http://test',
    logger: mockLogger,
    capture: vi.fn(),
    cookieJar: new Map(),
  };

  const config: AdapterConfig = {
    brokers: ['localhost:9092'],
    clientId: 'test-client',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    adapter = new KafkaAdapter(config, mockLogger);
  });

  afterEach(async () => {
    if (adapter) {
      await adapter.disconnect().catch(() => {});
    }
  });

  describe('constructor', () => {
    it('should create adapter with name kafka', () => {
      expect(adapter.name).toBe('kafka');
    });
  });

  describe('connect', () => {
    it('should connect successfully with valid config', async () => {
      await adapter.connect();
      expect(adapter['connected']).toBe(true);
    });

    it('should not reconnect if already connected', async () => {
      await adapter.connect();
      await adapter.connect();
      expect(adapter['connected']).toBe(true);
    });
  });

  describe('disconnect', () => {
    it('should disconnect producer and consumer', async () => {
      await adapter.connect();
      await adapter.disconnect();
      expect(adapter['connected']).toBe(false);
    });
  });

  describe('execute - produce action', () => {
    it('should produce message to topic', async () => {
      await adapter.connect();
      const result = await adapter.execute(
        'produce',
        {
          topic: 'test-topic',
          message: { key: 'value' },
        },
        mockContext
      );

      expect(result.status).toBe('pass');
      expect(result.output).toEqual({ count: 1 });
    });

    it('should produce multiple messages', async () => {
      await adapter.connect();
      const result = await adapter.execute(
        'produce',
        {
          topic: 'test-topic',
          messages: [{ key: 'value1' }, { key: 'value2' }],
        },
        mockContext
      );

      expect(result.status).toBe('pass');
      expect(result.output).toEqual({ count: 2 });
    });
  });

  describe('execute - unknown action', () => {
    it('should throw AdapterError for unknown action', async () => {
      await adapter.connect();
      await expect(adapter.execute('unknown', {}, mockContext)).rejects.toThrow(
        'Unknown action: unknown'
      );
    });
  });

  describe('healthCheck', () => {
    it('should return true when connected', async () => {
      await adapter.connect();
      const healthy = await adapter.healthCheck();
      expect(healthy).toBe(true);
    });

    it('should return false when not connected', async () => {
      const healthy = await adapter.healthCheck();
      expect(healthy).toBe(false);
    });
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- src/adapters/kafka.adapter.test.ts`
Expected: `FAIL — Cannot find module './kafka.adapter'`

### Step 3: Implement

Create `src/adapters/kafka.adapter.ts` (see full implementation in Part 3b).

### Step 4: Run test to verify it passes

Run: `npm test -- src/adapters/kafka.adapter.test.ts`
Expected: 10 tests PASS

### Step 5: Commit

```bash
git add src/adapters/kafka.adapter.ts src/adapters/kafka.adapter.test.ts
git commit -m "feat(kafka): implement KafkaAdapter with produce/consume/waitFor actions"
```

---

### Step 3 Implementation (continued) - Create `src/adapters/kafka.adapter.ts`:

```typescript
/**
 * E2E Test Runner - Kafka Adapter
 *
 * Apache Kafka operations using kafkajs
 */

import { AdapterError, AssertionError, TimeoutError } from '../errors';
import type { AdapterConfig, AdapterContext, AdapterStepResult, Logger } from '../types';
import { BaseAdapter } from './base.adapter';

export interface KafkaAssertion {
  path: string;
  equals?: unknown;
  contains?: string;
  matches?: string;
  exists?: boolean;
}

export class KafkaAdapter extends BaseAdapter {
  private kafka: import('kafkajs').Kafka | null = null;
  private producer: import('kafkajs').Producer | null = null;
  private consumer: import('kafkajs').Consumer | null = null;

  constructor(config: AdapterConfig, logger: Logger) {
    super(config, logger);
  }

  get name(): string {
    return 'kafka';
  }

  async connect(): Promise<void> {
    if (this.connected) return;

    try {
      const { Kafka } = await import('kafkajs');

      const brokers = Array.isArray(this.config.brokers)
        ? this.config.brokers
        : [this.config.brokers as string];

      this.kafka = new Kafka({
        clientId: (this.config.clientId as string) || 'e2e-runner',
        brokers,
        ssl: this.config.ssl as boolean | undefined,
        sasl: this.config.sasl as import('kafkajs').SASLOptions | undefined,
        connectionTimeout: (this.config.connectionTimeout as number) || 10000,
        requestTimeout: (this.config.requestTimeout as number) || 30000,
      });

      this.producer = this.kafka.producer();
      await this.producer.connect();

      this.connected = true;
      this.logger.info('Kafka connected');
    } catch (error) {
      throw new AdapterError(
        'kafka',
        'connect',
        `Failed to connect: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  async disconnect(): Promise<void> {
    if (this.producer) {
      await this.producer.disconnect();
      this.producer = null;
    }
    if (this.consumer) {
      await this.consumer.disconnect();
      this.consumer = null;
    }
    this.connected = false;
    this.logger.info('Kafka disconnected');
  }

  async execute(
    action: string,
    params: Record<string, unknown>,
    ctx: AdapterContext
  ): Promise<AdapterStepResult> {
    if (!this.producer) {
      throw new AdapterError('kafka', action, 'Not connected');
    }

    this.logAction(action, { topic: params.topic });
    const start = Date.now();

    try {
      let result: AdapterStepResult;

      switch (action) {
        case 'produce':
          result = await this.produceMessage(params);
          break;
        case 'consume':
          result = await this.consumeMessages(params);
          break;
        case 'waitFor':
          result = await this.waitForMessage(params, ctx);
          break;
        case 'clear':
          result = this.successResult(null, Date.now() - start);
          break;
        default:
          throw new AdapterError('kafka', action, `Unknown action: ${action}`);
      }

      this.logResult(action, true, result.duration);
      return result;
    } catch (error) {
      const duration = Date.now() - start;
      this.logResult(action, false, duration);

      if (
        error instanceof AssertionError ||
        error instanceof AdapterError ||
        error instanceof TimeoutError
      ) {
        throw error;
      }

      throw new AdapterError(
        'kafka',
        action,
        error instanceof Error ? error.message : String(error)
      );
    }
  }

  async healthCheck(): Promise<boolean> {
    return this.connected;
  }

  private async produceMessage(params: Record<string, unknown>): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const messages: import('kafkajs').Message[] = [];
    const inputMessages = params.messages || [params.message];

    for (const msg of inputMessages as Record<string, unknown>[]) {
      messages.push({
        key: msg.key as string | undefined,
        value: JSON.stringify(msg.value || msg),
        partition: msg.partition as number | undefined,
        headers: msg.headers as Record<string, string> | undefined,
      });
    }

    await this.producer!.send({ topic, messages });
    return this.successResult({ count: messages.length }, Date.now() - start);
  }

  private async consumeMessages(params: Record<string, unknown>): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const count = (params.count as number) || 1;
    const timeout = (params.timeout as number) || 10000;
    const messages: unknown[] = [];

    if (!this.consumer) {
      this.consumer = this.kafka!.consumer({ groupId: 'e2e-runner-consumer' });
      await this.consumer.connect();
    }

    await this.consumer.subscribe({ topic, fromBeginning: false });

    return new Promise<AdapterStepResult>((resolve, reject) => {
      const timeoutId = setTimeout(async () => {
        await this.consumer!.stop();
        resolve(this.successResult(messages, Date.now() - start));
      }, timeout);

      this.consumer!.run({
        eachMessage: async ({ message }) => {
          try {
            const value = JSON.parse(message.value?.toString() || '{}');
            messages.push(value);

            if (messages.length >= count) {
              clearTimeout(timeoutId);
              await this.consumer!.stop();
              resolve(this.successResult(messages, Date.now() - start));
            }
          } catch (error) {
            clearTimeout(timeoutId);
            reject(error);
          }
        },
      }).catch(reject);
    });
  }

  private async waitForMessage(
    params: Record<string, unknown>,
    ctx: AdapterContext
  ): Promise<AdapterStepResult> {
    const start = Date.now();
    const topic = params.topic as string;
    const timeout = (params.timeout as number) || 30000;
    const filter = params.filter as Record<string, unknown>;

    if (!this.consumer) {
      this.consumer = this.kafka!.consumer({ groupId: 'e2e-runner-consumer' });
      await this.consumer.connect();
    }

    await this.consumer.subscribe({ topic, fromBeginning: false });

    return new Promise<AdapterStepResult>((resolve, reject) => {
      const timeoutId = setTimeout(async () => {
        await this.consumer!.stop();
        resolve(
          this.failResult(
            new TimeoutError(
              `Waiting for message matching filter: ${JSON.stringify(filter)}`,
              timeout
            ),
            Date.now() - start
          )
        );
      }, timeout);

      this.consumer!.run({
        eachMessage: async ({ message }) => {
          try {
            const body = JSON.parse(message.value?.toString() || '{}');

            if (this.matchesFilter(body, filter)) {
              clearTimeout(timeoutId);
              await this.consumer!.stop();

              if (params.capture) {
                for (const [varName, path] of Object.entries(params.capture as Record<string, string>)) {
                  ctx.capture(varName, this.getNestedValue(body, path));
                }
              }

              if (params.assert) {
                try {
                  this.runAssertions(body, params.assert as KafkaAssertion[]);
                } catch (error) {
                  resolve(this.failResult(error as Error, Date.now() - start));
                  return;
                }
              }

              resolve(this.successResult(body, Date.now() - start));
            }
          } catch (error) {
            clearTimeout(timeoutId);
            reject(error);
          }
        },
      }).catch(reject);
    });
  }

  private matchesFilter(message: Record<string, unknown>, filter: Record<string, unknown>): boolean {
    if (!filter) return true;
    for (const [key, expected] of Object.entries(filter)) {
      const actual = this.getNestedValue(message, key);
      if (actual !== expected) return false;
    }
    return true;
  }

  private getNestedValue(obj: unknown, path: string): unknown {
    if (!path) return obj;
    return path.split('.').reduce((current, key) => {
      if (current && typeof current === 'object') {
        return (current as Record<string, unknown>)[key];
      }
      return undefined;
    }, obj);
  }

  private runAssertions(data: unknown, assertions: KafkaAssertion[]): void {
    for (const assertion of assertions) {
      const value = this.getNestedValue(data, assertion.path);

      if (assertion.exists === true && value === undefined) {
        throw new AssertionError(`${assertion.path} does not exist`, {
          path: assertion.path,
          operator: 'exists',
        });
      }

      if (assertion.equals !== undefined && value !== assertion.equals) {
        throw new AssertionError(
          `${assertion.path} = ${JSON.stringify(value)}, expected ${JSON.stringify(assertion.equals)}`,
          { path: assertion.path, expected: assertion.equals, actual: value, operator: 'equals' }
        );
      }

      if (assertion.contains && !String(value).includes(assertion.contains)) {
        throw new AssertionError(`${assertion.path} does not contain "${assertion.contains}"`, {
          path: assertion.path, expected: assertion.contains, actual: value, operator: 'contains',
        });
      }

      if (assertion.matches && !new RegExp(assertion.matches).test(String(value))) {
        throw new AssertionError(`${assertion.path} does not match /${assertion.matches}/`, {
          path: assertion.path, expected: assertion.matches, actual: value, operator: 'matches',
        });
      }
    }
  }
}
```

## Task 4: Register KafkaAdapter in AdapterRegistry [depends on: Task 3]

**Files:**
- Modify: `src/adapters/adapter-registry.ts` — Add Kafka import and instantiation
- Modify: `src/adapters/adapter-registry.test.ts` — Add Kafka adapter test

### Step 1: Write failing test

> In `src/adapters/adapter-registry.test.ts`, add to the describe block:

```typescript
describe('Kafka adapter', () => {
  it('should create Kafka adapter when configured', () => {
    const config: EnvironmentConfig = {
      baseUrl: 'http://test',
      adapters: {
        kafka: {
          brokers: ['localhost:9092'],
        },
      },
    };
    const registry = new AdapterRegistry(config, mockLogger, {
      requiredAdapters: new Set(['kafka']),
    });

    expect(registry.has('kafka')).toBe(true);
  });

  it('should not create Kafka adapter when not configured', () => {
    const config: EnvironmentConfig = {
      baseUrl: 'http://test',
      adapters: {},
    };
    const registry = new AdapterRegistry(config, mockLogger);

    expect(registry.has('kafka')).toBe(false);
  });
});
```

### Step 2: Run test to verify it fails

Run: `npm test -- src/adapters/adapter-registry.test.ts`
Expected: `FAIL — Expected registry.has('kafka') to be true`

### Step 3: Implement

> In `src/adapters/adapter-registry.ts`, find:
```typescript
import { EventHubAdapter } from './eventhub.adapter';
import { ShellAdapter } from './shell.adapter';
```
> Insert after:
```typescript
import { KafkaAdapter } from './kafka.adapter';
```

> In `src/adapters/adapter-registry.ts`, find the EventHub adapter initialization block:
```typescript
    // Create EventHub adapter if configured AND required
    if (this.isRequired('eventhub') && this.config.adapters?.eventhub) {
      this.adapters.set(
        'eventhub',
        new EventHubAdapter(
          {
            connectionString: this.config.adapters.eventhub.connectionString,
            consumerGroup: this.config.adapters.eventhub.consumerGroup,
          },
          this.logger
        )
      );
    }
```
> Insert after:
```typescript
    // Create Kafka adapter if configured AND required
    if (this.isRequired('kafka') && this.config.adapters?.kafka) {
      this.adapters.set(
        'kafka',
        new KafkaAdapter(
          {
            brokers: this.config.adapters.kafka.brokers,
            clientId: this.config.adapters.kafka.clientId,
            ssl: this.config.adapters.kafka.ssl,
            sasl: this.config.adapters.kafka.sasl,
            connectionTimeout: this.config.adapters.kafka.connectionTimeout,
            requestTimeout: this.config.adapters.kafka.requestTimeout,
          },
          this.logger
        )
      );
    }
```

> In `src/adapters/adapter-registry.ts`, find the `getEventHub()` method:
```typescript
  getEventHub(): EventHubAdapter {
    return this.get('eventhub') as EventHubAdapter;
  }
```
> Insert after:
```typescript
  getKafka(): KafkaAdapter {
    return this.get('kafka') as KafkaAdapter;
  }
```

> In `src/adapters/adapter-registry.ts`, find `parseAdapterType` valid types:
```typescript
  const validTypes: AdapterType[] = [
    'postgresql',
    'redis',
    'mongodb',
    'eventhub',
    'http',
    'shell',
  ];
```
> Replace with:
```typescript
  const validTypes: AdapterType[] = [
    'postgresql',
    'redis',
    'mongodb',
    'eventhub',
    'http',
    'shell',
    'kafka',
  ];
```

### Step 4: Run test to verify it passes

Run: `npm test -- src/adapters/adapter-registry.test.ts`
Expected: All tests PASS (including 2 new Kafka tests)

### Step 5: Commit

```bash
git add src/adapters/adapter-registry.ts src/adapters/adapter-registry.test.ts
git commit -m "feat(kafka): register KafkaAdapter in AdapterRegistry"
```

---

## Final Task: Verification

**Files:** None — verification only.

### Step 1: Run full test suite

Run: `npm test`
Expected: All tests PASS (311 existing + 10 KafkaAdapter + 2 registry = ~323 tests).

### Step 2: Run build

Run: `npm run build`
Expected: Build succeeds with no errors.

### Step 3: Manual smoke test

| # | Criterion | Steps | Expected |
|---|-----------|-------|----------|
| 1 | Kafka adapter loads | Create test YAML with `adapters.kafka.brokers: ['localhost:9092']` | Config parses without error |
| 2 | KafkaAdapter type check | Import `KafkaAdapter` in a test file | TypeScript recognizes type |
| 3 | Registry has Kafka | `registry.has('kafka')` with configured Kafka | Returns true |
| 4 | Produce action | Run test with `adapter: kafka`, `action: produce` | Step passes (requires running Kafka) |

---

## Files Changed (Summary)

| Action | Path | Purpose |
|--------|------|---------|
| Modify | `src/types.ts` | Add `kafka` to AdapterType, add KafkaAdapterConfig interface (1 line + 15 lines) |
| Modify | `package.json` | Add kafkajs to peerDependencies, peerDependenciesMeta, devDependencies (3 sections) |
| Create | `src/adapters/kafka.adapter.ts` | KafkaAdapter class with produce/consume/waitFor actions (~280 lines) |
| Create | `src/adapters/kafka.adapter.test.ts` | Unit tests for KafkaAdapter (~120 lines) |
| Modify | `src/adapters/adapter-registry.ts` | Import KafkaAdapter, add initialization block, add getKafka() method (~25 lines) |
| Modify | `src/adapters/adapter-registry.test.ts` | Add 2 tests for Kafka adapter registration (~25 lines) |

---

## Dependency Graph

```
Task 1 (Types) ─────┐
                    ├──► Task 3 (KafkaAdapter) ──► Task 4 (Registry)
Task 2 (Deps) ──────┘

Task 1 and Task 2 can run in parallel.
Task 3 requires both Task 1 and Task 2.
Task 4 requires Task 3.
```

**Parallel execution possible for:** Task 1, Task 2
**Sequential execution required for:** Task 3 → Task 4
