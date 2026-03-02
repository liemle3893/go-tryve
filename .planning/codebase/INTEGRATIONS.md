# External Integrations

**Analysis Date:** 2026-03-02

## APIs & External Services

**REST APIs (target services under test):**
- Any HTTP/HTTPS endpoint — the HTTP adapter sends arbitrary REST requests
  - SDK/Client: Native Node.js `fetch` API (no third-party HTTP library)
  - Auth: Injected via `headers` in test step params; `access_token` global variable available for Bearer tokens
  - Config: `baseUrl` set per environment in `e2e.config.yaml`
  - Implementation: `src/adapters/http.adapter.ts`

**Azure Event Hubs:**
- Used for publish/consume/waitFor operations against event streaming topics
  - SDK/Client: `@azure/event-hubs` `^6.0.0` (peer dependency, optional)
  - Auth: Connection string via `EVENTHUB_CONNECTION_STRING` environment variable
  - Consumer group: configurable via `consumerGroup` in adapter config (default: `$Default`)
  - Implementation: `src/adapters/eventhub.adapter.ts`
  - Local emulator: `mcr.microsoft.com/azure-messaging/eventhubs-emulator:2.1.0` via `docker-compose.yaml`
  - Emulator depends on: Azurite (Azure Storage emulator) for checkpoint storage

## Data Storage

**Databases:**

- **PostgreSQL**
  - Driver: `pg` `^8.16.3` via connection pool (`pg.Pool`)
  - Connection: `POSTGRESQL_CONNECTION_STRING` environment variable
  - Pool config: min 2, max 5 connections; 30s idle timeout; 10s connect timeout
  - Schema: configurable (`public` default)
  - Supported actions: `execute`, `query`, `queryOne`, `count`
  - Implementation: `src/adapters/postgresql.adapter.ts`
  - Local dev image: `citusdata/citus:13.0` (Citus PostgreSQL extension) via `docker-compose.yaml`
  - Schema init: `demo-server/init/schema.sql` mounted as init script

- **MongoDB**
  - Driver: `mongodb` `^7.0.0` native Node.js driver
  - Connection: `MONGODB_CONNECTION_STRING` environment variable
  - Database: configurable per environment (`database` field in adapter config)
  - Supported actions: `insertOne`, `insertMany`, `findOne`, `find`, `updateOne`, `updateMany`, `deleteOne`, `deleteMany`, `count`, `aggregate`
  - ObjectId handling: auto-converts `_id` string values and `$oid` notation from YAML filters
  - Implementation: `src/adapters/mongodb.adapter.ts`
  - Local dev image: `mongo:7.0.14` via `docker-compose.yaml`

- **Redis**
  - Client: `ioredis` `^5.0.0`
  - Connection: `REDIS_CONNECTION_STRING` environment variable
  - DB index: configurable (`db` field, default 0)
  - Key prefix: configurable (`keyPrefix` field)
  - Retry strategy: up to 3 retries with 200ms–2000ms exponential backoff
  - Supported actions: `get`, `set` (with optional `EX` TTL), `del`, `exists`, `incr`, `hget`, `hset`, `hgetall`, `keys`, `flushPattern`
  - Implementation: `src/adapters/redis.adapter.ts`
  - Local dev image: `redis:7.4-alpine` via `docker-compose.yaml`

**File Storage:**
- Local filesystem only — no cloud file storage integration
- Azurite (Azure Storage emulator) is included in `docker-compose.yaml` but only used as a checkpoint backend for the Event Hubs emulator (ports 10000–10002 exposed but not directly tested)

**Caching:**
- Redis adapter serves as the caching integration under test (not used internally by the runner)

## Authentication & Identity

**Auth Provider:**
- No built-in auth provider — authentication is handled by passing tokens through test variables
- Access token pattern: `access_token: "${JWT}"` in `e2e.config.yaml` variables section
- The `JWT` environment variable is injected into tests and can be referenced as `{{access_token}}` in HTTP headers

## Monitoring & Observability

**Error Tracking:**
- None — no third-party error tracking (Sentry, Datadog, etc.) integrated

**Logs:**
- Custom built-in logger in `src/utils/logger.ts`
- Log levels: `debug`, `info`, `warn`, `error`, `silent`
- ANSI color output (auto-detects TTY)
- Optional timestamps
- Child loggers with prefix support via `createChildLogger`
- All output goes to `console.log` / `console.error` (stdout/stderr)

## CI/CD & Deployment

**Hosting:**
- Distributed via npm registry as `@liemle3893/e2e-runner`
- Repository: `https://github.com/liemle3893/e2e-runner.git`

**CI Pipeline:**
- None detected — no GitHub Actions, CircleCI, or other CI config files present

## Reporting Outputs

The runner generates files to local filesystem — no external reporting services:

- **Console reporter** — `src/reporters/console.reporter.ts` — ANSI terminal output
- **JUnit XML reporter** — `src/reporters/junit.reporter.ts` — `./reports/junit.xml`
  - Compatible with Jenkins, GitHub Actions, and other CI systems that consume JUnit XML
- **HTML reporter** — `src/reporters/html.reporter.ts` — `./reports/report.html`
- **JSON reporter** — `src/reporters/json.reporter.ts` — `./reports/results.json`

## Environment Configuration

**Required environment variables (for `local` environment):**
- `POSTGRESQL_CONNECTION_STRING` — PostgreSQL URI (e.g., `postgresql://user:pass@host:5432/db`)
- `REDIS_CONNECTION_STRING` — Redis URI (e.g., `redis://localhost:6379`)
- `MONGODB_CONNECTION_STRING` — MongoDB URI (e.g., `mongodb://user:pass@host:27017`)
- `EVENTHUB_CONNECTION_STRING` — Azure Event Hubs connection string
- `JWT` — Bearer token for API authentication (optional, only if tests use it)

**Secrets location:**
- All secrets in environment variables only — no `.env` file detected in repo
- Connection strings resolved from `${VAR_NAME}` syntax during config load in `src/core/config-loader.ts`
- Unresolved variables in adapter configs are tolerated at load time but fail at connection time

## Local Development Services (Docker Compose)

`docker-compose.yaml` defines the full local integration stack:

| Service | Image | Ports | Purpose |
|---|---|---|---|
| `postgres` | `citusdata/citus:13.0` | `5432` | PostgreSQL (Citus) |
| `mongodb` | `mongo:7.0.14` | `27017` | MongoDB |
| `redis` | `redis:7.4-alpine` | `6379` | Redis |
| `eventhubs-emulator` | `mcr.microsoft.com/azure-messaging/eventhubs-emulator:2.1.0` | `5672`, `9092`, `5300` | Azure EventHubs emulator |
| `azurite` | `mcr.microsoft.com/azure-storage/azurite:3.35.0` | `10000-10002` | Azure Storage emulator (EventHubs checkpoint backend) |

EventHubs emulator config file: `config/eventhubs-emulator.json`

## Webhooks & Callbacks

**Incoming:**
- None — this is a test runner tool, not a server

**Outgoing:**
- HTTP adapter sends arbitrary outgoing HTTP requests to services under test
- EventHub adapter publishes events to Azure Event Hubs topics

---

*Integration audit: 2026-03-02*
