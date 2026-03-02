# Stack Research

**Domain:** E2E testing framework (Node.js/TypeScript) — adding Kafka adapter + unit test suite
**Researched:** 2026-03-02
**Confidence:** HIGH (all key decisions verified via npm registry, official docs, WebSearch)

---

## Summary Verdict

Three additions are needed to make this framework feature-complete:

1. **Kafka client**: Use `kafkajs@2.2.4` — not `@confluentinc/kafka-javascript`. Despite KafkaJS being unmaintained since 2023, it is the only pure-JavaScript option that fits this project's peer dependency model (no native compilation). The Confluent client requires `node-pre-gyp install --fallback-to-build` (C++ native addon), which breaks the framework's promise of "install only what you need."

2. **Unit test runner**: Use `vitest@3.2.4` — not `vitest@4.x`. Vitest 4 requires Node >=20 which conflicts with this project's `node >=18` engine requirement. Vitest 3.2.4 supports `^18.0.0 || ^20.0.0 || >=22.0.0`, has built-in TypeScript support, and includes Chai-powered assertions — no separate assertion library needed.

3. **No additional assertion library**: Vitest includes `expect` (Jest-compatible) + Chai built-in. Adding chai separately would be redundant and add confusion.

---

## Recommended Stack

### Core Technologies (new additions)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `kafkajs` | `2.2.4` | Kafka adapter peer dependency | Only pure-JS Kafka client — no native compilation. Bundled TypeScript types. 1.9M weekly downloads. Works on Alpine/any platform. Peer dep model compatible. |
| `vitest` | `^3.2.4` | Unit test runner (devDependency) | Node >=18 compatible (v4 requires Node >=20). TypeScript-native via Vite transform. Built-in Chai assertions. No ts-jest or babel config needed. 7.7M weekly downloads. |
| `@vitest/coverage-v8` | `^3.2.4` | Test coverage (devDependency) | V8 native coverage — zero config, no Istanbul babel transform. Same version as vitest, must match. |

### Existing Stack (retained — no changes)

| Technology | Version | Purpose | Notes |
|------------|---------|---------|-------|
| `typescript` | `^5.7.0` | Compilation | No change needed |
| `yaml` | `^2.7.0` | Config/test file parsing | No change needed |
| `p-limit` | `^3.1.0` | Parallel concurrency | No change needed |
| `pg` | `^8.16.3` | PostgreSQL peer dep | No change needed |
| `mongodb` | `^7.0.0` | MongoDB peer dep | No change needed |
| `ioredis` | `^5.0.0` | Redis peer dep | No change needed |
| `@azure/event-hubs` | `^6.0.0` | EventHub peer dep | No change needed |

### Supporting Libraries (new devDependencies)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@types/node` | `^20.0.0` | Node type defs | Already present as devDependency |

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `kafkajs@2.2.4` | `@confluentinc/kafka-javascript@1.8.0` | Only if consumer needs maximum throughput at scale AND can tolerate native C++ compilation in all CI/CD environments. Not suitable for an npm-distributed testing framework with peer dep model. |
| `kafkajs@2.2.4` | `node-rdkafka` | Never for this project. Older C++ binding approach, more fragile, worse DX. |
| `vitest@3.2.4` | `vitest@4.0.18` | Only when project bumps minimum Node.js requirement to >=20. Vitest 4 requires `node ^20.0.0 || ^22.0.0 || >=24.0.0` — incompatible with current `node >=18` engine constraint. |
| `vitest@3.2.4` | `jest@29` | Jest requires manual TypeScript setup (ts-jest or babel). Slower test execution (no HMR). Jest has no built-in ESM support (still experimental). More config overhead. |
| `@vitest/coverage-v8` | `@vitest/coverage-istanbul` | Use Istanbul only if you need detailed branch coverage on transpiled/instrumented code. V8 is simpler and faster for pure Node.js testing. |
| Built-in `expect` (Vitest/Chai) | `chai` (separate) | Never install chai separately. Vitest already bundles Chai-powered assertions. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `@confluentinc/kafka-javascript` as peer dep | Runs `node-pre-gyp install --fallback-to-build` — compiles native C++ on every install. Breaks Alpine Docker, some CI environments. Incompatible with this project's zero-native-deps peer dependency model. | `kafkajs@2.2.4` |
| `vitest@4.x` | Requires Node >=20. Breaks project's `node >=18` engine guarantee. Would force consumers to upgrade Node. | `vitest@3.2.4` |
| `jest` | Heavier config overhead for TypeScript. Requires ts-jest or babel-jest. ESM support still experimental in Jest 29/30. Slower startup. No clear advantage over Vitest 3 for this project. | `vitest@3.2.4` |
| `kafka-node` | Officially deprecated. Repository archived. No support for modern Kafka protocol features (headers, incremental cooperative rebalancing, newer auth). | `kafkajs@2.2.4` |
| `@types/kafkajs` | KafkaJS bundles its own TypeScript types at `types/index.d.ts`. The `@types/kafkajs` package is a stub that may be stale. | Use types from `kafkajs` directly |
| `mocha` + `chai` + `sinon` | Three-package setup to replicate what Vitest includes in one package. More configuration, more version pinning, slower execution. | `vitest@3.2.4` |

---

## KafkaJS Maintenance Risk Assessment

KafkaJS (2.2.4) has not had a stable release since February 2023. This is a real concern but manageable for this project's use case:

**Why it is still the right choice here:**
- The Kafka wire protocol is stable. KafkaJS implements protocol versions compatible with Kafka 0.10+ through Kafka 3.x
- The project is an **E2E testing framework** — it needs a Kafka client for test harness use (produce/consume in tests), not for production message processing at scale
- 1.9M weekly downloads in 2025/2026 indicates broad community continued use
- Active beta track: versions 2.3.0-beta.0 through 2.3.0-beta.3 exist on npm
- Pure JavaScript — no security patches from outdated C++ dependencies to worry about
- The Confluent JavaScript client explicitly provides a KafkaJS-compatible migration path, acknowledging KafkaJS's position as the incumbent

**Acceptable risk because:**
- This is a devDependency/peer dependency for test infrastructure, not a production runtime
- If KafkaJS eventually breaks on a future Kafka version, migration to `@confluentinc/kafka-javascript` (which offers a KafkaJS-compatible API) is straightforward

---

## Stack Patterns by Variant

**For the Kafka adapter (peer dependency — production consumers install this):**
- Add `kafkajs` to `peerDependencies` with `optional: true` in `peerDependenciesMeta`
- Add `kafkajs` to `devDependencies` for the project's own tests
- Pattern matches existing adapters: pg, mongodb, ioredis, @azure/event-hubs

**For unit tests (devDependencies only — never shipped to consumers):**
- Use `vitest@^3.2.4` with `@vitest/coverage-v8@^3.2.4`
- Configure via `vitest.config.ts` at project root (no Vite needed — Vitest can run standalone)
- Set `test.environment: 'node'` and `test.coverage.provider: 'v8'`
- Do NOT add vitest to peerDependencies or dependencies — it is internal tooling only

**If minimum Node.js is bumped to >=20 in a future milestone:**
- Upgrade to `vitest@4.x` — stable Browser Mode, improved performance
- No other changes needed (API is compatible)

---

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| `vitest@3.2.4` | `node ^18.0.0 \|\| ^20.0.0 \|\| >=22.0.0` | Verified via `npm info vitest@3.2.4 engines` |
| `vitest@3.2.4` | `vite ^5.0.0 \|\| ^6.0.0 \|\| ^7.0.0-0` | Vitest brings its own Vite dep — no consumer install needed |
| `@vitest/coverage-v8@3.2.4` | `vitest@3.2.4` | Must match Vitest version exactly |
| `kafkajs@2.2.4` | `node >=14.0.0` | Works on Node 18, 20, 22 |
| `kafkajs@2.2.4` | `kafka 0.10 – 3.x` | Kafka 4.x compatibility: not confirmed (no releases since 2023) |
| `vitest@4.0.18` | `node ^20.0.0 \|\| ^22.0.0 \|\| >=24.0.0` | INCOMPATIBLE with project's `node >=18` constraint |

---

## Installation

```bash
# Kafka adapter — add to peerDependencies in package.json (optional)
# Consumers who need Kafka testing will install this themselves
# kafkajs is a peer dep like pg, mongodb, ioredis

# devDependencies — for the project's own unit tests and Kafka adapter dev
npm install -D vitest@^3.2.4 @vitest/coverage-v8@^3.2.4 kafkajs@^2.2.4

# package.json peerDependencies addition:
# "kafkajs": "^2.2.4"
# package.json peerDependenciesMeta addition:
# "kafkajs": { "optional": true }
```

---

## Vitest Configuration for this Project

Vitest can run standalone (no Vite build tool required for the project). Key configuration for a pure Node.js CommonJS testing project:

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
    globals: false,           // Use import { describe, it, expect } explicitly
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html'],
      exclude: ['dist/**', 'tests/**', 'bin/**']
    }
  }
})
```

This works with the existing CommonJS TypeScript output because Vitest uses its own transform pipeline for `src/**/*.ts` files during testing — it does not depend on the project's `tsc` build output.

---

## Sources

- `npm info kafkajs version` — confirmed 2.2.4, last published 2023-02-27
- `npm info kafkajs versions --json` — beta track 2.3.0-beta.0 through 2.3.0-beta.3 exists
- [KafkaJS GitHub issue #1603](https://github.com/tulios/kafkajs/issues/1603) — "Looking for maintainers" — MEDIUM confidence (WebSearch verified)
- [KafkaJS GitHub issue #1753](https://github.com/tulios/kafkajs/issues/1753) — "KafkaJS status" — MEDIUM confidence
- `npm info @confluentinc/kafka-javascript scripts.install` — confirmed `node-pre-gyp install --fallback-to-build` — HIGH confidence
- `npm info @confluentinc/kafka-javascript engines` — `{ node: '>=18.0.0' }` — HIGH confidence
- `npm info vitest@3.2.4 engines` — `{ node: '^18.0.0 || ^20.0.0 || >=22.0.0' }` — HIGH confidence
- `npm info vitest engines` (latest v4.0.18) — `{ node: '^20.0.0 || ^22.0.0 || >=24.0.0' }` — HIGH confidence
- [Vitest blog: vitest-3](https://vitest.dev/blog/vitest-3) — 7.7M weekly downloads, Node 18 support — MEDIUM confidence
- [Confluent Kafka JavaScript docs](https://docs.confluent.io/kafka-clients/javascript/current/overview.html) — GA status, librdkafka-based — HIGH confidence
- [GitHub issue: confluent-kafka-javascript#48](https://github.com/confluentinc/confluent-kafka-javascript/issues/48) — Alpine/musl segfault issues — MEDIUM confidence (WebSearch)
- [npmtrends: kafkajs](https://npmtrends.com/kafka-vs-kafka-node-vs-kafkajs-vs-node-rdkafka) — download comparisons — MEDIUM confidence

---

*Stack research for: E2E Runner — Kafka adapter + unit test suite additions*
*Researched: 2026-03-02*
