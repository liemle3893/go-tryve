/**
 * E2E Test Runner - Shell/CLI Adapter
 *
 * Executes shell commands as test steps. Supports command execution with
 * stdout/stderr capture, exit code checking, timeout, cwd, and env overrides.
 *
 * SECURITY NOTE: This adapter intentionally uses child_process.exec() (not execFile)
 * because users need full shell features (pipes, redirects, globbing, subshells).
 * This is safe because:
 * - The `command` field comes from YAML test files written by the test author
 * - This is the same trust model as any CI/CD system or test runner
 * - The adapter runs in a testing context, not a production web server
 * - No untrusted user input reaches exec() -- commands are static YAML values
 */

import { exec } from 'node:child_process';
import { AdapterError, AssertionError } from '../errors';
import type { AdapterConfig, AdapterContext, AdapterStepResult, Logger } from '../types';
import { runAssertion } from '../assertions/assertion-runner';
import { BaseAdapter, captureValues } from './base.adapter';

// ============================================================================
// Types
// ============================================================================

/**
 * Parameters for the shell exec action.
 */
export interface ShellRequestParams {
  /** Shell command to execute. */
  command: string;
  /** Working directory override. */
  cwd?: string;
  /** Command timeout in milliseconds. */
  timeout?: number;
  /** Environment variables merged with process.env. */
  env?: Record<string, string>;
  /** Capture map: variable name to source (stdout, stderr, exitCode). */
  capture?: Record<string, string>;
  /** Inline assertions on the command result. */
  assert?: ShellAssertion;
}

/**
 * Shell command execution response data.
 */
export interface ShellResponse {
  /** Process exit code (0 = success). */
  exitCode: number;
  /** Standard output content. */
  stdout: string;
  /** Standard error content. */
  stderr: string;
  /** Execution duration in milliseconds. */
  duration: number;
}

/**
 * Assertion schema for shell command results.
 *
 * Note: `equals` on stdout/stderr trims surrounding whitespace to handle
 * the trailing newline that shell commands (e.g. echo) add by default.
 */
export interface ShellAssertion {
  /** Expected exit code (exact match). */
  exitCode?: number;
  /** Assertions on stdout content. */
  stdout?: { contains?: string; matches?: string; equals?: string };
  /** Assertions on stderr content. */
  stderr?: { contains?: string; matches?: string; equals?: string };
}

// ============================================================================
// Shell Adapter
// ============================================================================

/**
 * Shell/CLI adapter for executing shell commands as test steps.
 *
 * Uses Node.js child_process.exec() for full shell feature support
 * (pipes, redirects, globbing). Commands come from YAML test files
 * authored by the test writer, following the same trust model as CI/CD.
 */
export class ShellAdapter extends BaseAdapter {
  private defaultTimeout: number;
  private defaultCwd?: string;
  private defaultEnv?: Record<string, string>;

  constructor(config: AdapterConfig, logger: Logger) {
    super(config, logger);
    this.defaultTimeout = (config.defaultTimeout as number) ?? 30000;
    this.defaultCwd = config.defaultCwd as string | undefined;
    this.defaultEnv = config.defaultEnv as Record<string, string> | undefined;
  }

  /**
   * Get the adapter name for logging.
   */
  get name(): string {
    return 'shell';
  }

  /**
   * Connect - no persistent connection needed for shell execution.
   */
  async connect(): Promise<void> {
    this.connected = true;
    this.logger.debug('[shell] Adapter connected (no persistent connection needed)');
  }

  /**
   * Disconnect - release connected state.
   */
  async disconnect(): Promise<void> {
    this.connected = false;
    this.logger.debug('[shell] Adapter disconnected');
  }

  /**
   * Health check by running a simple echo command.
   */
  async healthCheck(): Promise<boolean> {
    try {
      await this.runCommand('echo ok', {});
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Execute a shell action.
   *
   * Only the 'exec' action is supported. Runs the specified command,
   * captures stdout/stderr/exitCode, runs assertions, and captures values.
   */
  async execute(
    action: string,
    params: Record<string, unknown>,
    ctx: AdapterContext,
  ): Promise<AdapterStepResult> {
    if (action !== 'exec') {
      throw new AdapterError('shell', action, `Unknown action: "${action}". Only "exec" is supported.`);
    }

    const shellParams = params as unknown as ShellRequestParams;

    if (!shellParams.command || typeof shellParams.command !== 'string') {
      throw new AdapterError('shell', 'exec', 'Missing required "command" parameter');
    }

    this.logAction('exec', { command: shellParams.command });

    const { result: { exitCode, stdout, stderr }, duration } = await this.measureDuration(
      () => this.runCommand(shellParams.command, {
        timeout: shellParams.timeout,
        cwd: shellParams.cwd ?? this.defaultCwd,
        env: { ...process.env, ...this.defaultEnv, ...shellParams.env } as Record<string, string>,
      })
    );

    const response: ShellResponse = { exitCode, stdout, stderr, duration };

    // Run assertions if provided
    if (shellParams.assert) {
      this.runAssertions(shellParams.assert, response);
    }

    // Capture values for use in later steps
    captureValues(response, shellParams.capture, ctx, (_data, source) => {
      switch (source) {
        case 'stdout': return stdout;
        case 'stderr': return stderr;
        case 'exitCode': return exitCode;
        default:
          throw new AdapterError(
            'shell',
            'exec',
            `Unknown capture source "${source}". Valid sources: stdout, stderr, exitCode`,
          );
      }
    });

    this.logResult('exec', true, duration);
    return this.successResult(response, duration);
  }

  /**
   * Execute a shell command and return its output.
   *
   * Handles non-zero exit codes as data (not errors), timeouts as
   * AdapterError, and system-level failures (ENOENT) as AdapterError.
   *
   * Uses exec() intentionally for full shell feature support -- commands
   * originate from static YAML test files, not untrusted user input.
   */
  private runCommand(
    command: string,
    options: { timeout?: number; cwd?: string; env?: Record<string, string> },
  ): Promise<{ exitCode: number; stdout: string; stderr: string }> {
    const timeout = options.timeout ?? this.defaultTimeout;

    return new Promise((resolve, reject) => {
      exec(
        command,
        {
          timeout,
          cwd: options.cwd,
          env: options.env as NodeJS.ProcessEnv | undefined,
          maxBuffer: 10 * 1024 * 1024, // 10MB
        },
        (error, stdout, stderr) => {
          if (error) {
            // Timeout: exec kills the process and sets error.killed = true
            if (error.killed) {
              reject(
                new AdapterError('shell', 'exec', `Command timed out after ${timeout}ms`),
              );
              return;
            }

            // System-level error (command not found, permission denied)
            // Check errno string for system-level errors (ENOENT has string code on ErrnoException)
            const errnoCode = (error as unknown as { errno?: string }).errno;
            if (errnoCode === 'ENOENT' || (error.message && error.message.includes('ENOENT'))) {
              reject(new AdapterError('shell', 'exec', `Command not found: ${command}`));
              return;
            }

            // Non-zero exit code: extract exit code from error, return as data
            const exitCode = typeof error.code === 'number' ? error.code : 1;

            resolve({
              exitCode,
              stdout: stdout?.toString() ?? '',
              stderr: stderr?.toString() ?? '',
            });
            return;
          }

          // Successful execution (exit code 0)
          resolve({
            exitCode: 0,
            stdout: stdout?.toString() ?? '',
            stderr: stderr?.toString() ?? '',
          });
        },
      );
    });
  }

  /**
   * Run inline assertions on shell command output.
   *
   * Throws AssertionError if any assertion fails.
   */
  private runAssertions(assert: ShellAssertion, response: ShellResponse): void {
    // Exit code assertion
    if (assert.exitCode !== undefined) {
      runAssertion(response.exitCode, { equals: assert.exitCode }, 'exitCode');
    }

    // Stdout/stderr assertions
    for (const field of ['stdout', 'stderr'] as const) {
      const assertion = assert[field];
      if (!assertion) continue;

      const actual = response[field];

      // Delegate contains/matches to the shared assertion runner
      runAssertion(actual, { contains: assertion.contains, matches: assertion.matches }, field);

      // equals trims surrounding whitespace to handle shell's trailing newlines (e.g. from echo)
      if (assertion.equals !== undefined && actual.trim() !== assertion.equals.trim()) {
        throw new AssertionError(
          `${field} does not equal expected value`,
          {
            expected: assertion.equals,
            actual,
            path: field,
            operator: 'equals',
          },
        );
      }
    }
  }
}
