/**
 * Tests for Shell/CLI adapter.
 *
 * Verifies that the ShellAdapter correctly executes shell commands,
 * captures output, handles assertions, timeouts, and error cases.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

// ---------------------------------------------------------------------------
// Shell Adapter Tests
// ---------------------------------------------------------------------------

describe('Shell Adapter', () => {
  let ShellAdapter: typeof import('../../src/adapters/shell.adapter').ShellAdapter;
  let childProcess: typeof import('node:child_process');

  const mockLogger = {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  };

  const mockCtx = {
    variables: {},
    captured: {},
    capture: vi.fn(),
    logger: mockLogger,
    baseUrl: 'http://localhost:3000',
    cookieJar: new Map<string, string>(),
  };

  beforeEach(async () => {
    vi.restoreAllMocks();
    mockCtx.capture.mockClear();
    mockCtx.captured = {};

    const mod = await import('../../src/adapters/shell.adapter');
    ShellAdapter = mod.ShellAdapter;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // -----------------------------------------------------------------------
  // Basic command execution
  // -----------------------------------------------------------------------
  it('should execute a simple command and return stdout, exitCode=0, success=true', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo hello',
    }, mockCtx);

    expect(result.success).toBe(true);
    expect(result.data).toBeDefined();
    const data = result.data as { exitCode: number; stdout: string; stderr: string };
    expect(data.exitCode).toBe(0);
    expect(data.stdout).toContain('hello');
  });

  it('should capture exit code, stdout, and stderr in response data', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo out && echo err >&2',
    }, mockCtx);

    expect(result.success).toBe(true);
    const data = result.data as { exitCode: number; stdout: string; stderr: string; duration: number };
    expect(data.exitCode).toBe(0);
    expect(data.stdout).toContain('out');
    expect(data.stderr).toContain('err');
    expect(typeof data.duration).toBe('number');
  });

  it('should return exitCode!=0 for failing command but still return success=true with data', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'exit 42',
    }, mockCtx);

    expect(result.success).toBe(true);
    const data = result.data as { exitCode: number; stdout: string; stderr: string };
    expect(data.exitCode).toBe(42);
  });

  // -----------------------------------------------------------------------
  // Timeout handling
  // -----------------------------------------------------------------------
  it('should respect timeout and throw for long-running commands', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('exec', {
        command: 'sleep 10',
        timeout: 100,
      }, mockCtx)
    ).rejects.toThrow(/timed out/i);
  });

  // -----------------------------------------------------------------------
  // Environment variable injection
  // -----------------------------------------------------------------------
  it('should pass env vars to child process', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo $SHELL_ADAPTER_TEST_VAR',
      env: { SHELL_ADAPTER_TEST_VAR: 'bar' },
    }, mockCtx);

    expect(result.success).toBe(true);
    const data = result.data as { stdout: string };
    expect(data.stdout).toContain('bar');
  });

  // -----------------------------------------------------------------------
  // Working directory (cwd)
  // -----------------------------------------------------------------------
  it('should support cwd option to change working directory', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'pwd',
      cwd: '/tmp',
    }, mockCtx);

    expect(result.success).toBe(true);
    const data = result.data as { stdout: string };
    // macOS resolves /tmp to /private/tmp
    expect(data.stdout.trim()).toMatch(/\/(tmp|private\/tmp)$/);
  });

  // -----------------------------------------------------------------------
  // Assertions
  // -----------------------------------------------------------------------
  it('should assert on exitCode and pass', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo ok',
      assert: { exitCode: 0 },
    }, mockCtx);

    expect(result.success).toBe(true);
  });

  it('should assert on exitCode and fail with AssertionError', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('exec', {
        command: 'exit 1',
        assert: { exitCode: 0 },
      }, mockCtx)
    ).rejects.toThrow(/exit.*code/i);
  });

  it('should assert on stdout content with contains', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo "hello world"',
      assert: { stdout: { contains: 'hello' } },
    }, mockCtx);

    expect(result.success).toBe(true);
  });

  it('should assert on stderr content with contains', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    const result = await adapter.execute('exec', {
      command: 'echo "warning message" >&2',
      assert: { stderr: { contains: 'warning' } },
    }, mockCtx);

    expect(result.success).toBe(true);
  });

  it('should pass stdout.equals trimming shell trailing newline', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    // echo adds a trailing newline; equals should pass without requiring it
    const result = await adapter.execute('exec', {
      command: 'echo "hello"',
      assert: { stdout: { equals: 'hello' } },
    }, mockCtx);

    expect(result.success).toBe(true);
  });

  it('should fail stdout.equals when content differs after trim', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('exec', {
        command: 'echo "hello"',
        assert: { stdout: { equals: 'world' } },
      }, mockCtx)
    ).rejects.toThrow(/does not equal/i);
  });

  // -----------------------------------------------------------------------
  // Captures
  // -----------------------------------------------------------------------
  it('should capture values via capture paths (stdout, stderr, exitCode)', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await adapter.execute('exec', {
      command: 'echo captured_output',
      capture: { version: 'stdout' },
    }, mockCtx);

    expect(mockCtx.capture).toHaveBeenCalledWith('version', expect.stringContaining('captured_output'));
  });

  it('should capture exitCode as a number', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await adapter.execute('exec', {
      command: 'exit 5',
      capture: { code: 'exitCode' },
    }, mockCtx);

    expect(mockCtx.capture).toHaveBeenCalledWith('code', 5);
  });

  it('should throw AdapterError for unknown capture source', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('exec', {
        command: 'echo hello',
        capture: { myVar: 'invalidSource' },
      }, mockCtx)
    ).rejects.toThrow(/unknown capture source/i);
  });

  // -----------------------------------------------------------------------
  // Error cases
  // -----------------------------------------------------------------------
  it('should throw AdapterError for unknown actions (not "exec")', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('run', {
        command: 'echo test',
      }, mockCtx)
    ).rejects.toThrow(/unknown action/i);
  });

  it('should throw AdapterError when command is missing from params', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();

    await expect(
      adapter.execute('exec', {}, mockCtx)
    ).rejects.toThrow(/command/i);
  });

  // -----------------------------------------------------------------------
  // Health check
  // -----------------------------------------------------------------------
  it('should return true for healthCheck', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();
    expect(await adapter.healthCheck()).toBe(true);
  });

  // -----------------------------------------------------------------------
  // Connect/disconnect
  // -----------------------------------------------------------------------
  it('should connect and disconnect without error', async () => {
    const adapter = new ShellAdapter({}, mockLogger);
    await adapter.connect();
    expect(adapter.isConnected()).toBe(true);
    await adapter.disconnect();
    expect(adapter.isConnected()).toBe(false);
  });

  it('should return "shell" for adapter name', () => {
    const adapter = new ShellAdapter({}, mockLogger);
    expect(adapter.name).toBe('shell');
  });
});

// ---------------------------------------------------------------------------
// YAML Loader validation tests for shell adapter
// ---------------------------------------------------------------------------
describe('YAML Loader - Shell Adapter Validation', () => {
  const tmpDir = path.join(__dirname, '..', 'fixtures');

  beforeEach(() => {
    fs.mkdirSync(tmpDir, { recursive: true });
  });

  it('should accept adapter: shell with action: exec and command field', async () => {
    const tmpFile = path.join(tmpDir, 'valid-shell.test.yaml');
    fs.writeFileSync(tmpFile, `
name: TC-SHELL-001
description: Valid shell test
execute:
  - adapter: shell
    action: exec
    command: "echo hello"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      const test = await loadYAMLTest(tmpFile);
      expect(test.execute[0].adapter).toBe('shell');
      expect(test.execute[0].action).toBe('exec');
      expect(test.execute[0].params.command).toBe('echo hello');
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  it('should reject shell step without command field', async () => {
    const tmpFile = path.join(tmpDir, 'no-command-shell.test.yaml');
    fs.writeFileSync(tmpFile, `
name: TC-SHELL-002
description: Invalid shell test - missing command
execute:
  - adapter: shell
    action: exec
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      await expect(loadYAMLTest(tmpFile)).rejects.toThrow(/command/i);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });

  it('should reject shell step with invalid action (not "exec")', async () => {
    const tmpFile = path.join(tmpDir, 'invalid-action-shell.test.yaml');
    fs.writeFileSync(tmpFile, `
name: TC-SHELL-003
description: Invalid shell test - bad action
execute:
  - adapter: shell
    action: run
    command: "echo hello"
`);

    try {
      const { loadYAMLTest } = await import('../../src/core/yaml-loader');
      await expect(loadYAMLTest(tmpFile)).rejects.toThrow(/action/i);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  });
});
