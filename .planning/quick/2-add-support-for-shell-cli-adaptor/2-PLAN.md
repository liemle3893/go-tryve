---
phase: quick-2
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - src/types.ts
  - src/adapters/shell.adapter.ts
  - src/adapters/adapter-registry.ts
  - src/adapters/index.ts
  - src/core/yaml-loader.ts
  - src/index.ts
  - docs/sections/adapters/shell.md
  - docs/sections/adapters/index.md
  - .claude/skills/e2e-runner/SKILL.md
  - .claude/skills/e2e-runner/references/adapters/shell.md
  - tests/unit/shell-adapter.test.ts
autonomous: true
requirements: [QUICK-2]

must_haves:
  truths:
    - "User can write YAML tests with adapter: shell and execute shell commands"
    - "Shell adapter captures stdout, stderr, and exit code from command execution"
    - "Shell adapter supports timeout to prevent runaway commands"
    - "Shell adapter supports working directory (cwd) override"
    - "Shell adapter supports environment variable injection"
    - "User can assert on exit code, stdout content, and stderr content"
    - "Shell adapter captures values from stdout/stderr for use in later steps"
  artifacts:
    - path: "src/adapters/shell.adapter.ts"
      provides: "Shell/CLI adapter implementation"
      exports: ["ShellAdapter", "ShellRequestParams", "ShellResponse", "ShellAssertion"]
    - path: "src/types.ts"
      provides: "AdapterType union includes 'shell'"
      contains: "'shell'"
    - path: "src/adapters/adapter-registry.ts"
      provides: "Shell adapter registration in initializeAdapters"
      contains: "ShellAdapter"
    - path: "src/core/yaml-loader.ts"
      provides: "Shell adapter validation rules"
      contains: "'shell'"
    - path: "docs/sections/adapters/shell.md"
      provides: "Shell adapter documentation"
    - path: "tests/unit/shell-adapter.test.ts"
      provides: "Unit tests for shell adapter"
      min_lines: 100
  key_links:
    - from: "src/adapters/adapter-registry.ts"
      to: "src/adapters/shell.adapter.ts"
      via: "import and instantiation in initializeAdapters"
      pattern: "new ShellAdapter"
    - from: "src/core/yaml-loader.ts"
      to: "src/types.ts"
      via: "VALID_ADAPTERS array includes 'shell'"
      pattern: "'shell'"
    - from: "src/adapters/index.ts"
      to: "src/adapters/shell.adapter.ts"
      via: "re-export"
      pattern: "ShellAdapter"
---

<objective>
Add a shell/CLI adapter to the e2e-runner so users can execute shell commands as test steps alongside HTTP, database, and messaging adapters.

Purpose: Enables testing CLI tools, running setup/teardown scripts, verifying system commands, and orchestrating shell-based workflows within E2E test suites. This fills a gap where users need to invoke system commands (e.g., database migrations, file generation, process health checks) as part of their test flows.

Output: A fully functional `shell` adapter with `exec` action, YAML validation, unit tests, and documentation.
</objective>

<execution_context>
@./.claude/get-shit-done/workflows/execute-plan.md
@./.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@./CLAUDE.md
@.planning/STATE.md

<interfaces>
<!-- Key types and contracts the executor needs. Extracted from codebase. -->

From src/types.ts:
```typescript
export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http';

export interface EnvironmentConfig {
  baseUrl: string;
  adapters: {
    postgresql?: PostgreSQLAdapterConfig;
    redis?: RedisAdapterConfig;
    mongodb?: MongoDBAdapterConfig;
    eventhub?: EventHubAdapterConfig;
  };
}

export interface AdapterConfig {
  connectionString?: string;
  baseUrl?: string;
  [key: string]: unknown;
}

export interface AdapterContext {
  variables: Record<string, unknown>;
  captured: Record<string, unknown>;
  capture: (name: string, value: unknown) => void;
  logger: Logger;
  baseUrl: string;
  cookieJar: Map<string, string>;
}

export interface AdapterStepResult {
  success: boolean;
  data?: unknown;
  error?: Error;
  duration: number;
}
```

From src/adapters/base.adapter.ts:
```typescript
export abstract class BaseAdapter {
  protected config: AdapterConfig;
  protected logger: Logger;
  protected connected: boolean = false;
  constructor(config: AdapterConfig, logger: Logger);
  abstract get name(): string;
  abstract connect(): Promise<void>;
  abstract disconnect(): Promise<void>;
  abstract execute(action: string, params: Record<string, unknown>, ctx: AdapterContext): Promise<AdapterStepResult>;
  abstract healthCheck(): Promise<boolean>;
  protected successResult(data: unknown, duration: number): AdapterStepResult;
  protected failResult(error: Error, duration: number): AdapterStepResult;
  protected logAction(action: string, params?: Record<string, unknown>): void;
  protected logResult(action: string, success: boolean, duration: number): void;
}
```

From src/adapters/adapter-registry.ts:
```typescript
export class AdapterRegistry {
  private initializeAdapters(): void;  // Where new adapters are registered
  get(type: AdapterType): BaseAdapter;
  has(type: AdapterType): boolean;
}
export function parseAdapterType(type: string): AdapterType;  // validTypes array
export function getRequiredAdapters(tests: UnifiedTestDefinition[]): Set<AdapterType>;
```

From src/core/yaml-loader.ts:
```typescript
const VALID_ADAPTERS: AdapterType[] = ['postgresql', 'redis', 'mongodb', 'eventhub', 'http'];
function validateAdapterStep(step: RawYAMLStep, location: string): string[];  // adapter-specific validation switch
```

From src/adapters/index.ts (re-exports pattern):
```typescript
export { HTTPAdapter } from './http.adapter';
export type { HTTPRequestParams, HTTPResponse, HTTPAssertion } from './http.adapter';
export { AdapterRegistry, createAdapterRegistry, parseAdapterType, getRequiredAdapters } from './adapter-registry';
```

From src/errors.ts:
```typescript
export class AdapterError extends E2ERunnerError {
  constructor(adapter: string, action: string, message: string);
}
export class AssertionError extends E2ERunnerError {
  constructor(message: string, options: { expected?: unknown; actual?: unknown; path?: string; operator?: string });
}
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Implement shell adapter with types, registration, and validation</name>
  <files>
    src/types.ts,
    src/adapters/shell.adapter.ts,
    src/adapters/adapter-registry.ts,
    src/adapters/index.ts,
    src/core/yaml-loader.ts,
    src/index.ts,
    tests/unit/shell-adapter.test.ts
  </files>
  <behavior>
    - Test: ShellAdapter executes a simple command (`echo hello`) and returns stdout="hello\n", exitCode=0, success=true
    - Test: ShellAdapter captures exit code, stdout, and stderr in response data
    - Test: ShellAdapter returns exitCode!=0 for failing command (`exit 1`) but still returns success=true with data
    - Test: ShellAdapter respects timeout and rejects for long-running commands (timeout: 100, command: `sleep 10`)
    - Test: ShellAdapter passes env vars to child process (env: {FOO: "bar"}, command: `echo $FOO`)
    - Test: ShellAdapter supports cwd option to change working directory
    - Test: ShellAdapter exec action asserts on exitCode (assert: {exitCode: 0})
    - Test: ShellAdapter exec action asserts on stdout content (assert: {stdout: {contains: "hello"}})
    - Test: ShellAdapter exec action asserts on stderr content (assert: {stderr: {contains: "warning"}})
    - Test: ShellAdapter captures values via capture paths (capture: {version: "stdout"} captures full stdout)
    - Test: ShellAdapter throws AdapterError for unknown actions (not "exec")
    - Test: ShellAdapter throws AdapterError when command is missing from params
    - Test: YAML loader accepts adapter: shell with action: exec and command field
    - Test: YAML loader rejects shell step without command field
    - Test: YAML loader rejects shell step with invalid action (not "exec")
  </behavior>
  <action>
    **Security note:** The shell adapter intentionally uses `child_process.exec()` (not `execFile`) because users need full shell features (pipes, redirects, globbing, subshells). This is safe because:
    - The `command` field comes from YAML test files written by the test author (not untrusted user input)
    - This is the same trust model as any CI/CD system or test runner that executes commands
    - The adapter runs in a testing context, not a production web server

    1. **Update `src/types.ts`:**
       - Add `'shell'` to the `AdapterType` union type: `export type AdapterType = 'postgresql' | 'redis' | 'mongodb' | 'eventhub' | 'http' | 'shell';`
       - Add `ShellAdapterConfig` interface: `{ defaultTimeout?: number; defaultCwd?: string; defaultEnv?: Record<string, string>; }`
       - Add `shell?: ShellAdapterConfig` to the `EnvironmentConfig.adapters` type

    2. **Create `src/adapters/shell.adapter.ts`:**
       - Import `exec` from `node:child_process` and `promisify` from `node:util`
       - Define exported types:
         - `ShellRequestParams`: `{ command: string; cwd?: string; timeout?: number; env?: Record<string, string>; capture?: Record<string, string>; assert?: ShellAssertion }`
         - `ShellResponse`: `{ exitCode: number; stdout: string; stderr: string; duration: number }`
         - `ShellAssertion`: `{ exitCode?: number; stdout?: { contains?: string; matches?: string; equals?: string }; stderr?: { contains?: string; matches?: string; equals?: string } }`
       - Extend `BaseAdapter`:
         - `get name()` returns `'shell'`
         - `connect()` sets `this.connected = true` (no persistent connection needed)
         - `disconnect()` sets `this.connected = false`
         - `healthCheck()` runs a simple command (`echo ok`) and returns true if it exits cleanly, false otherwise
         - `execute(action, params, ctx)`:
           - Only accept action `'exec'` -- throw `AdapterError('shell', action, 'Unknown action: ...')` otherwise
           - Cast params to `ShellRequestParams`
           - Validate `params.command` exists and is a string -- throw `AdapterError('shell', 'exec', 'Missing required "command" parameter')` if not
           - Use `child_process.exec()` wrapped in a Promise. The `exec` callback provides `(error, stdout, stderr)`. On error, `error.code` has the exit code.
           - Options: `{ timeout: params.timeout ?? config.defaultTimeout ?? 30000, cwd: params.cwd ?? config.defaultCwd, env: { ...process.env, ...config.defaultEnv, ...params.env }, maxBuffer: 10 * 1024 * 1024 }`
           - Build `ShellResponse` with `{ exitCode, stdout: stdout.toString(), stderr: stderr.toString(), duration }`
           - Handle the exec error case: when command exits non-zero, `exec` calls back with an error that has `code` property (the exit code). Extract exitCode from `error.code`. Still build ShellResponse with the exit code, stdout, stderr. Do NOT throw -- return the data so assertions can check exitCode.
           - Handle true system errors (ENOENT = command not found): throw `AdapterError('shell', 'exec', message)`
           - Handle timeout: when exec times out it sets `error.killed = true`. Throw `AdapterError('shell', 'exec', 'Command timed out after Xms')`
           - Run assertions if `params.assert` is provided:
             - `exitCode`: exact number match, throw `AssertionError` if mismatch
             - `stdout.contains/matches/equals`: same pattern as HTTP body assertions
             - `stderr.contains/matches/equals`: same pattern as HTTP body assertions
           - Handle captures: for each `capture` entry, if path is `"stdout"` capture stdout, if `"stderr"` capture stderr, if `"exitCode"` capture exitCode. Call `ctx.capture(varName, value)`.
           - Return `this.successResult(shellResponse, duration)`

    3. **Update `src/adapters/adapter-registry.ts`:**
       - Import `ShellAdapter` from `./shell.adapter`
       - In `initializeAdapters()`, add shell adapter registration after eventhub block:
         ```
         if (this.isRequired('shell')) {
           const shellConfig = this.config.adapters?.shell;
           this.adapters.set('shell', new ShellAdapter({
             defaultTimeout: shellConfig?.defaultTimeout,
             defaultCwd: shellConfig?.defaultCwd,
             defaultEnv: shellConfig?.defaultEnv,
           }, this.logger));
         }
         ```
       - Add `getShell(): ShellAdapter` convenience method
       - Add `'shell'` to the `validTypes` array in `parseAdapterType()`

    4. **Update `src/adapters/index.ts`:**
       - Add: `export { ShellAdapter } from './shell.adapter';`
       - Add: `export type { ShellRequestParams, ShellResponse, ShellAssertion } from './shell.adapter';`

    5. **Update `src/core/yaml-loader.ts`:**
       - Add `'shell'` to `VALID_ADAPTERS` array
       - Add `case 'shell':` to `validateAdapterStep()`:
         - Valid actions: only `'exec'`
         - Required: `step.command` must exist and be a string

    6. **Update `src/index.ts`:**
       - Check existing programmatic API exports (around line 377 where `createAdapterRegistry` is exported). Add `ShellAdapter` export if other adapter classes are exported there.

    7. **Write unit tests in `tests/unit/shell-adapter.test.ts`:**
       - Follow exact pattern from `tests/unit/http-multipart.test.ts`
       - Use vitest: `import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'`
       - For shell adapter tests: use `vi.mock('node:child_process')` to mock `exec`
       - Mock exec to simulate: successful command (stdout, exit 0), failing command (exit 1 with stderr), timeout (killed=true), ENOENT error
       - Test assertions (exitCode mismatch throws AssertionError, stdout contains check)
       - Test captures (stdout, stderr, exitCode capture paths)
       - For YAML loader tests: write temp YAML files, import `loadYAMLTest`, assert validation passes/fails
  </action>
  <verify>
    <automated>cd /Users/liemlhd/Documents/git/Personal/e2e-runner && npm run build && npm test</automated>
  </verify>
  <done>
    - ShellAdapter class exists extending BaseAdapter with exec action
    - AdapterType union includes 'shell'
    - EnvironmentConfig.adapters includes shell?: ShellAdapterConfig
    - AdapterRegistry initializes and provides ShellAdapter
    - YAML loader validates shell steps (requires command field, only exec action)
    - parseAdapterType accepts 'shell'
    - All unit tests pass: command execution, exit code handling, timeout, cwd, env, assertions, captures, error cases
    - TypeScript compiles without errors (npm run build)
  </done>
</task>

<task type="auto">
  <name>Task 2: Add shell adapter documentation and update skills</name>
  <files>
    docs/sections/adapters/shell.md,
    docs/sections/adapters/index.md,
    .claude/skills/e2e-runner/SKILL.md,
    .claude/skills/e2e-runner/references/adapters/shell.md
  </files>
  <action>
    1. **Create `docs/sections/adapters/shell.md`** following the structure of existing adapter docs (e.g., `docs/sections/adapters/http.md`):
       - Title: "Shell Adapter"
       - Overview: Execute shell commands and scripts as test steps. No peer dependencies (uses Node.js built-in `child_process`).
       - Configuration section:
         ```yaml
         environments:
           local:
             baseUrl: "http://localhost:3000"
             adapters:
               shell:
                 defaultTimeout: 30000
                 defaultCwd: "/app"
                 defaultEnv:
                   NODE_ENV: "test"
         ```
       - Actions section: document the `exec` action with all parameters:
         - `command` (required, string) -- the shell command to execute
         - `cwd` (optional, string) -- working directory override
         - `timeout` (optional, number ms, default 30000) -- kill after timeout
         - `env` (optional, Record) -- environment variables merged with process.env
       - Response structure: `{ exitCode: number, stdout: string, stderr: string, duration: number }`
       - Assertions section:
         - `exitCode`: exact number match
         - `stdout`: `{ contains, matches, equals }` -- same operators as HTTP body
         - `stderr`: `{ contains, matches, equals }` -- same operators as HTTP body
       - Capture section:
         - `capture: { var_name: "stdout" }` -- captures full stdout
         - `capture: { var_name: "stderr" }` -- captures full stderr
         - `capture: { var_name: "exitCode" }` -- captures exit code as number
       - Examples section with 4-5 practical examples:
         - Basic: echo command with exit code assertion
         - Script: run a script file, capture output
         - Setup: database migration in setup phase
         - Environment: pass env vars and set cwd
         - Capture: capture command output for later steps

    2. **Update `docs/sections/adapters/index.md`:**
       - Add Shell row to the "Available Adapters" table:
         `| [Shell](shell.md) | Shell/CLI command execution | None (built-in) |`
       - Add to "Peer Dependencies" section a note: Shell adapter has no peer dependencies.

    3. **Update `.claude/skills/e2e-runner/SKILL.md`:**
       - In the step definition comment, update adapter list: `# http|postgresql|mongodb|redis|eventhub|shell`
       - Add a brief "Shell Commands" section (like the existing "Multipart/Form-Data Uploads" section) with a minimal example:
         ```yaml
         - adapter: shell
           action: exec
           command: "echo 'hello world'"
           assert:
             exitCode: 0
             stdout:
               contains: "hello"
         ```
       - Add reference link at bottom: `* **Shell Adapter** [references/adapters/shell.md](references/adapters/shell.md)`

    4. **Create `.claude/skills/e2e-runner/references/adapters/shell.md`:**
       - Concise reference following pattern of other adapter reference files
       - Include: configuration, exec action params, assertions, captures, 2 examples
  </action>
  <verify>
    <automated>cd /Users/liemlhd/Documents/git/Personal/e2e-runner && test -f docs/sections/adapters/shell.md && test -f .claude/skills/e2e-runner/references/adapters/shell.md && grep -q 'shell' docs/sections/adapters/index.md && grep -q 'shell' .claude/skills/e2e-runner/SKILL.md && echo "PASS: All docs exist and references updated" || echo "FAIL: Missing docs or references"</automated>
  </verify>
  <done>
    - docs/sections/adapters/shell.md exists with full adapter documentation (config, actions, assertions, captures, examples)
    - docs/sections/adapters/index.md lists Shell adapter in the table
    - SKILL.md updated with shell in adapter list, basic example, and reference link
    - references/adapters/shell.md created with concise reference
    - All documentation follows existing patterns and conventions
  </done>
</task>

</tasks>

<verification>
1. `npm run build` compiles without TypeScript errors
2. `npm test` passes all unit tests including new shell adapter tests
3. `./bin/e2e.js validate` still works for existing test files
4. Shell adapter is discoverable: `grep -r "shell" src/types.ts src/adapters/ src/core/yaml-loader.ts` shows all integration points
5. Documentation files exist and reference each other correctly
</verification>

<success_criteria>
- A user can write `adapter: shell` with `action: exec` in YAML test files
- Shell commands execute via Node.js child_process and return stdout, stderr, exitCode
- Assertions work on exitCode, stdout, and stderr
- Values can be captured from stdout/stderr for use in later steps
- Timeout kills long-running commands
- Environment variables and working directory can be overridden per-step
- All existing tests continue to pass
- TypeScript compiles cleanly
- Documentation is complete and consistent with other adapters
</success_criteria>

<output>
After completion, create `.planning/quick/2-add-support-for-shell-cli-adaptor/2-SUMMARY.md`
</output>
