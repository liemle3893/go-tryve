# Technology Stack

**Analysis Date:** 2026-03-02

## Languages

**Primary:**
- TypeScript 5.7.x - All source code in `src/`; compiled to CommonJS via `tsc`

**Secondary:**
- JavaScript - CLI entry point `bin/e2e.js` (requires compiled `dist/index.js`)
- YAML - Test definitions (`*.test.yaml`) and configuration (`e2e.config.yaml`)

## Runtime

**Environment:**
- Node.js >=18.0.0 (required; development environment uses v22.21.1)
- CommonJS module system (`"module": "commonjs"` in `tsconfig.json`)
- ES2022 target and lib

**Package Manager:**
- npm (primary; `package-lock.json` lockfile version 3 present)
- yarn (secondary; `yarn.lock` also present — both lockfiles exist in repo)

## Frameworks

**Core:**
- None — pure Node.js library and CLI tool, no application framework

**Build/Dev:**
- TypeScript `^5.7.0` — compilation (`tsc`)
- ts-node `^10.9.2` (devDependency) — runtime TS execution for development/test loading

**Testing (framework itself has no test suite):**
- `package.json` scripts: `"test": "echo \"No tests yet\""`
- No test runner configured

## Key Dependencies

**Critical (runtime dependencies bundled):**
- `yaml` `^2.7.0` — YAML config and test file parsing (`src/core/config-loader.ts`, `src/core/yaml-loader.ts`)
- `p-limit` `^3.1.0` — Parallel test execution concurrency control (`src/core/test-orchestrator.ts`)

**Optional (validated at runtime, not bundled):**
- `ajv` `^8.17.1` — JSON schema validation for config files (`src/core/config-loader.ts`); skipped if absent
- `minimatch` `^10.0.1` — Glob pattern matching for test discovery

**Peer (adapter drivers — install only what is needed):**
- `pg` `^8.16.3` — PostgreSQL driver via `pg.Pool` (`src/adapters/postgresql.adapter.ts`)
- `mongodb` `^7.0.0` — MongoDB native driver (`src/adapters/mongodb.adapter.ts`)
- `ioredis` `^5.0.0` — Redis client (`src/adapters/redis.adapter.ts`)
- `@azure/event-hubs` `^6.0.0` — Azure Event Hubs SDK (`src/adapters/eventhub.adapter.ts`)
- `typescript` `^5.0.0` — Required peer for TypeScript test files (`.test.ts`)

**Dev dependencies (not shipped):**
- `@types/node` `^20.0.0` — Node.js type definitions
- `@types/pg` `^8.16.0` — PostgreSQL type definitions

## Node.js Built-ins Used

The codebase relies on several built-in Node.js APIs without third-party libraries:
- `node:crypto` — UUID generation (`randomUUID`), MD5/SHA256 hashing (`src/core/variable-interpolator.ts`)
- `node:fs` — Configuration and test file I/O
- `node:path` — File path resolution
- `fetch` (global, Node.js >=18) — HTTP adapter uses native `fetch` with `AbortSignal.timeout` (`src/adapters/http.adapter.ts`)

## Configuration

**TypeScript (`tsconfig.json`):**
- `target`: ES2022
- `module`: commonjs
- `outDir`: `./dist`
- `rootDir`: `./src`
- `strict`: false (type safety is relaxed)
- `noImplicitAny`: false
- `resolveJsonModule`: true
- `esModuleInterop`: true
- `declaration`: true (`.d.ts` files generated for consumers)

**Environment:**
- All secrets and connection strings passed via environment variables in `${VAR_NAME}` syntax within `e2e.config.yaml`
- No `.env` file detected; vars must be set in shell or CI environment
- Config file: `e2e.config.yaml` (default path, overridable with `-c` flag)
- Config schema version is fixed at `"1.0"`

**Key environment variables (from `e2e.config.yaml`):**
- `POSTGRESQL_CONNECTION_STRING` — PostgreSQL connection URI
- `REDIS_CONNECTION_STRING` — Redis connection URI
- `MONGODB_CONNECTION_STRING` — MongoDB connection URI
- `EVENTHUB_CONNECTION_STRING` — Azure Event Hubs connection string
- `JWT` — Access token injected as global `access_token` variable

**Build:**
- `npm run build` → `tsc` compiles `src/` → `dist/`
- `npm run clean` → removes `dist/`
- `npm run prepublishOnly` → clean + build (used for npm publish)
- Output: `dist/index.js` (CommonJS) + `dist/index.d.ts` (types)

## Platform Requirements

**Development:**
- Node.js >=18.0.0
- npm or yarn
- TypeScript 5.x (peer dependency)
- Optional: Docker + Docker Compose for local adapter services

**Production/Deployment:**
- Distributed as an npm package: `@liemle3893/go-tryve`
- Published files: `dist/`, `bin/`, `README.md`
- Consumers install peer dependencies selectively based on adapters needed
- No bundler — shipped as CommonJS source

---

*Stack analysis: 2026-03-02*
