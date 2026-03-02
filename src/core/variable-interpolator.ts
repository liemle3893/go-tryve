/**
 * E2E Test Runner - Variable Interpolator
 *
 * Handles {{variable}} interpolation and built-in functions
 */

import { createHash, createHmac, randomUUID } from 'node:crypto';
import * as fs from 'node:fs';

import { InterpolationError } from '../errors';
import type { InterpolationContext, BuiltInFunction } from '../types';

// ============================================================================
// TOTP Helpers
// ============================================================================

/**
 * Decode a base32-encoded string (RFC 4648) to a Buffer
 */
function base32Decode(encoded: string): Buffer {
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    const stripped = encoded.replace(/=+$/, '').toUpperCase();

    let bits = 0;
    let value = 0;
    const output: number[] = [];

    for (const char of stripped) {
        const idx = alphabet.indexOf(char);
        if (idx === -1) {
            throw new InterpolationError(
                `Invalid base32 character in TOTP secret: ${char}`,
                '$totp()'
            );
        }
        value = (value << 5) | idx;
        bits += 5;
        if (bits >= 8) {
            bits -= 8;
            output.push((value >>> bits) & 0xff);
        }
    }

    return Buffer.from(output);
}

/**
 * Generate a TOTP code per RFC 6238 (6 digits, 30s period, HMAC-SHA1)
 */
function generateTOTP(secret: string): string {
    const key = base32Decode(secret);
    const epoch = Math.floor(Date.now() / 1000);
    const counter = Math.floor(epoch / 30);

    // Encode counter as 8-byte big-endian buffer
    const counterBuf = Buffer.alloc(8);
    counterBuf.writeUInt32BE(Math.floor(counter / 0x100000000), 0);
    counterBuf.writeUInt32BE(counter & 0xffffffff, 4);

    // HMAC-SHA1
    const hmac = createHmac('sha1', key).update(counterBuf).digest();

    // Dynamic truncation
    const offset = hmac[hmac.length - 1] & 0x0f;
    const code =
        ((hmac[offset] & 0x7f) << 24) |
        ((hmac[offset + 1] & 0xff) << 16) |
        ((hmac[offset + 2] & 0xff) << 8) |
        (hmac[offset + 3] & 0xff);

    // 6-digit zero-padded string
    return (code % 1000000).toString().padStart(6, '0');
}

// ============================================================================
// Built-in Functions
// ============================================================================

/**
 * Registry of built-in functions available in interpolation
 */
export const BUILT_IN_FUNCTIONS: Record<string, BuiltInFunction> = {
  // UUID and identifiers
  $uuid: () => randomUUID(),

  // Timestamps
  $timestamp: () => Date.now(),
  $isoDate: () => new Date().toISOString(),

  // Random values
  $random: (min: string, max: string) => {
    const minVal = parseInt(min, 10) || 0;
    const maxVal = parseInt(max, 10) || 100;
    return Math.floor(Math.random() * (maxVal - minVal + 1)) + minVal;
  },

  $randomString: (length: string) => {
    const len = parseInt(length, 10) || 8;
    const chars =
      'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let result = '';
    for (let i = 0; i < len; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  },

  // Environment
  $env: (varName: string) => {
    const value = process.env[varName];
    if (value === undefined) {
      throw new InterpolationError(
        `Environment variable not found: ${varName}`,
        `$env(${varName})`
      );
    }
    return value;
  },

  // File operations
  $file: (filePath: string) => {
    try {
      return fs.readFileSync(filePath, 'utf8');
    } catch (error) {
      throw new InterpolationError(
        `Failed to read file: ${filePath}`,
        `$file(${filePath})`
      );
    }
  },

  // Encoding
  $base64: (value: string) => Buffer.from(value).toString('base64'),
  $base64Decode: (value: string) =>
    Buffer.from(value, 'base64').toString('utf8'),

  // Hashing
  $md5: (value: string) => createHash('md5').update(value).digest('hex'),
  $sha256: (value: string) => createHash('sha256').update(value).digest('hex'),

  // Date operations
  $now: (format: string) => formatDate(new Date(), format || 'iso'),

  $dateAdd: (amount: string, unit: string) => {
    const date = new Date();
    addToDate(date, parseInt(amount, 10) || 0, unit || 'days');
    return date.toISOString();
  },

  $dateSub: (amount: string, unit: string) => {
    const date = new Date();
    addToDate(date, -(parseInt(amount, 10) || 0), unit || 'days');
    return date.toISOString();
  },

  // JSON operations
  $jsonStringify: (value: string) => {
    try {
      return JSON.stringify(JSON.parse(value));
    } catch {
      return value;
    }
  },

  // String operations
  $lower: (value: string) => value.toLowerCase(),
  $upper: (value: string) => value.toUpperCase(),
  $trim: (value: string) => value.trim(),

  // TOTP
  $totp: (secret: string) => {
    if (!secret) {
      throw new InterpolationError('TOTP secret is required', '$totp()');
    }
    return generateTOTP(secret);
  },
};

// ============================================================================
// Date Helpers
// ============================================================================

/**
 * Format a date according to a format string
 */
function formatDate(date: Date, format: string): string {
  const pad = (n: number) => n.toString().padStart(2, '0');

  switch (format) {
    case 'iso':
      return date.toISOString();
    case 'date':
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
    case 'time':
      return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
    case 'datetime':
      return `${formatDate(date, 'date')} ${formatDate(date, 'time')}`;
    case 'unix':
      return Math.floor(date.getTime() / 1000).toString();
    case 'YYYY-MM-DD':
      return formatDate(date, 'date');
    case 'HH:mm:ss':
      return formatDate(date, 'time');
    default:
      return date.toISOString();
  }
}

/**
 * Add or subtract time from a date
 */
function addToDate(date: Date, amount: number, unit: string): void {
  switch (unit.toLowerCase()) {
    case 'second':
    case 'seconds':
    case 's':
      date.setSeconds(date.getSeconds() + amount);
      break;
    case 'minute':
    case 'minutes':
    case 'm':
      date.setMinutes(date.getMinutes() + amount);
      break;
    case 'hour':
    case 'hours':
    case 'h':
      date.setHours(date.getHours() + amount);
      break;
    case 'day':
    case 'days':
    case 'd':
      date.setDate(date.getDate() + amount);
      break;
    case 'week':
    case 'weeks':
    case 'w':
      date.setDate(date.getDate() + amount * 7);
      break;
    case 'month':
    case 'months':
      date.setMonth(date.getMonth() + amount);
      break;
    case 'year':
    case 'years':
    case 'y':
      date.setFullYear(date.getFullYear() + amount);
      break;
    default:
      date.setDate(date.getDate() + amount);
  }
}

// ============================================================================
// Interpolation Functions
// ============================================================================

/** Maximum number of interpolation passes before aborting */
export const MAX_INTERPOLATION_DEPTH = 10;

/**
 * Interpolate variables and functions in a template string
 *
 * Performs multi-pass resolution: after each replacement pass, checks whether
 * the result still contains {{...}} patterns. Loops until stable (no more
 * patterns) or until MAX_INTERPOLATION_DEPTH is reached. Detects cycles by
 * comparing each pass result to the previous one.
 *
 * Supports:
 * - Variable references: {{varName}}
 * - Nested variables: {{captured.fieldName}}
 * - Built-in functions: {{$uuid}}, {{$random(1, 100)}}
 * - Base URL: {{baseUrl}}
 */
export function interpolate(
  template: string,
  context: InterpolationContext
): string {
  if (!template || typeof template !== 'string') {
    return template;
  }

  let result = template;
  let prev: string | undefined;

  for (let depth = 0; depth < MAX_INTERPOLATION_DEPTH; depth++) {
    if (!hasInterpolation(result)) {
      return result;
    }

    if (result === prev) {
      throw new InterpolationError(
        `Circular variable reference detected during interpolation: "${result}"`,
        result
      );
    }
    prev = result;

    result = singlePassInterpolate(result, context);
  }

  if (hasInterpolation(result)) {
    throw new InterpolationError(
      `Max interpolation depth (${MAX_INTERPOLATION_DEPTH}) exceeded. Possible circular reference in: "${template}"`,
      template
    );
  }

  return result;
}

/**
 * Perform a single replacement pass over a template string
 */
function singlePassInterpolate(
  template: string,
  context: InterpolationContext
): string {
  const pattern = /\{\{([^}]+)\}\}/g;

  return template.replace(pattern, (match, expression) => {
    const trimmed = expression.trim();

    try {
      // Check for built-in function
      if (trimmed.startsWith('$')) {
        return String(evaluateFunction(trimmed));
      }

      // Check for baseUrl
      if (trimmed === 'baseUrl') {
        return context.baseUrl || '';
      }

      // Check for captured values (captured.fieldName or just the variable if in captured)
      if (trimmed.startsWith('captured.')) {
        const path = trimmed.slice(9); // Remove 'captured.'
        const value = getNestedValue(context.captured, path);
        if (value === undefined) {
          throw new InterpolationError(
            `Captured value not found: ${path}`,
            trimmed
          );
        }
        return String(value);
      }

      // Check in captured values first
      if (context.captured && trimmed in context.captured) {
        return String(context.captured[trimmed]);
      }

      // Check in variables
      const value = getNestedValue(context.variables, trimmed);
      if (value === undefined) {
        // Check environment variables as fallback
        if (context.env && trimmed in context.env) {
          return context.env[trimmed];
        }
        throw new InterpolationError(`Variable not found: ${trimmed}`, trimmed);
      }

      return String(value);
    } catch (error) {
      if (error instanceof InterpolationError) {
        throw error;
      }
      throw new InterpolationError(
        `Failed to interpolate: ${error instanceof Error ? error.message : String(error)}`,
        trimmed
      );
    }
  });
}

/**
 * Recursively interpolate all string values in an object
 */
export function interpolateObject<T>(
  obj: T,
  context: InterpolationContext
): T {
  if (typeof obj === 'string') {
    return interpolate(obj, context) as T;
  }

  if (Array.isArray(obj)) {
    return obj.map((item) => interpolateObject(item, context)) as T;
  }

  if (obj && typeof obj === 'object') {
    const result: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(obj)) {
      result[key] = interpolateObject(value, context);
    }
    return result as T;
  }

  return obj;
}

// ============================================================================
// Variable Cross-Reference Resolution
// ============================================================================

/** Patterns that should be deferred (only available at step execution time) */
const DEFERRED_PATTERNS = ['baseUrl', 'captured.'];

/**
 * Check whether a variable value references only deferred runtime patterns
 *
 * Returns true if every {{...}} token in the value references baseUrl or
 * captured.*, meaning it cannot be resolved at definition time.
 */
function isDeferredOnly(value: string): boolean {
  const pattern = /\{\{([^}]+)\}\}/g;
  let match;
  while ((match = pattern.exec(value)) !== null) {
    const expr = match[1].trim();
    const isDeferred = DEFERRED_PATTERNS.some(
      (p) => expr === p || expr.startsWith(p)
    );
    if (!isDeferred) {
      return false;
    }
  }
  return true;
}

/**
 * Resolve cross-references between variables using topological sort
 *
 * Scans variable values for {{...}} references to other variables, builds a
 * dependency graph, and resolves them in dependency order using Kahn's
 * algorithm. Variables that only reference runtime values (baseUrl,
 * captured.*) are skipped.
 *
 * @param variables - Mutable variables object; values are resolved in-place
 * @param baseUrl - Base URL for interpolation context
 * @returns The same variables object with cross-references resolved
 * @throws InterpolationError on circular references
 */
export function resolveVariableValues(
  variables: Record<string, unknown>,
  baseUrl: string = ''
): Record<string, unknown> {
  // Identify which variables contain interpolation references to other variables
  const varNamesSet = new Set(Object.keys(variables));
  const deps: Record<string, string[]> = {};
  const resolvable = new Set<string>();

  for (const name of varNamesSet) {
    const value = variables[name];
    if (typeof value !== 'string' || !hasInterpolation(value)) {
      continue; // No interpolation needed — skip
    }

    if (isDeferredOnly(value)) {
      continue; // Only references runtime values — defer
    }

    resolvable.add(name);

    // Extract variable references (non-function, non-deferred)
    const refs = extractVariableNames(value).filter(
      (ref) =>
        !DEFERRED_PATTERNS.some((p) => ref === p || ref.startsWith(p)) &&
        varNamesSet.has(ref)
    );
    deps[name] = refs;
  }

  // Kahn's algorithm for topological sort — O(V+E)
  const inDegree: Record<string, number> = {};
  const dependents: Record<string, string[]> = {};
  for (const name of resolvable) {
    inDegree[name] = 0;
    dependents[name] = [];
  }
  for (const name of resolvable) {
    for (const dep of deps[name] || []) {
      if (resolvable.has(dep)) {
        inDegree[name]++;
        dependents[dep].push(name);
      }
    }
  }

  const queue: string[] = [];
  for (const name of resolvable) {
    if (inDegree[name] === 0) {
      queue.push(name);
    }
  }

  const sorted: string[] = [];
  let head = 0;
  while (head < queue.length) {
    const current = queue[head++];
    sorted.push(current);

    for (const dependent of dependents[current]) {
      inDegree[dependent]--;
      if (inDegree[dependent] === 0) {
        queue.push(dependent);
      }
    }
  }

  if (sorted.length !== resolvable.size) {
    const sortedSet = new Set(sorted);
    const unresolved = [...resolvable].filter((n) => !sortedSet.has(n));
    throw new InterpolationError(
      `Circular variable reference detected among: ${unresolved.join(', ')}`,
      unresolved.join(', ')
    );
  }

  // Resolve in dependency order
  const context = createInterpolationContext(variables, {}, baseUrl);

  for (const name of sorted) {
    const raw = variables[name];
    if (typeof raw === 'string') {
      variables[name] = interpolate(raw, context);
    }
  }

  return variables;
}

/**
 * Evaluate a built-in function expression
 */
function evaluateFunction(expression: string): string | number {
  // Parse function call: $funcName(arg1, arg2)
  const funcMatch = expression.match(/^\$(\w+)(?:\(([^)]*)\))?$/);
  if (!funcMatch) {
    throw new InterpolationError(
      `Invalid function expression: ${expression}`,
      expression
    );
  }

  const [, funcName, argsStr] = funcMatch;
  const fullFuncName = `$${funcName}`;

  const func = BUILT_IN_FUNCTIONS[fullFuncName];
  if (!func) {
    throw new InterpolationError(
      `Unknown function: ${fullFuncName}`,
      expression
    );
  }

  // Parse arguments
  const args = argsStr
    ? argsStr.split(',').map((a) => a.trim().replace(/^['"]|['"]$/g, ''))
    : [];

  return func(...args);
}

/**
 * Get a nested value from an object using dot notation
 */
export function getNestedValue(
  obj: Record<string, unknown> | undefined,
  path: string
): unknown {
  if (!obj) return undefined;

  return path.split('.').reduce((current, key) => {
    if (current && typeof current === 'object') {
      return (current as Record<string, unknown>)[key];
    }
    return undefined;
  }, obj as unknown);
}

/**
 * Set a nested value in an object using dot notation
 */
export function setNestedValue(
  obj: Record<string, unknown>,
  path: string,
  value: unknown
): void {
  const keys = path.split('.');
  let current: Record<string, unknown> = obj;

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i];
    if (!(key in current) || typeof current[key] !== 'object') {
      current[key] = {};
    }
    current = current[key] as Record<string, unknown>;
  }

  current[keys[keys.length - 1]] = value;
}

/**
 * Create an interpolation context
 */
export function createInterpolationContext(
  variables: Record<string, unknown>,
  captured: Record<string, unknown>,
  baseUrl: string
): InterpolationContext {
  return {
    variables,
    captured,
    baseUrl,
    env: process.env as Record<string, string>,
  };
}

/**
 * Check if a string contains interpolation placeholders
 */
export function hasInterpolation(str: string): boolean {
  return /\{\{[^}]+\}\}/.test(str);
}

/**
 * Extract all variable names from a template
 */
export function extractVariableNames(template: string): string[] {
  const pattern = /\{\{([^}]+)\}\}/g;
  const names: string[] = [];
  let match;

  while ((match = pattern.exec(template)) !== null) {
    const expression = match[1].trim();
    if (!expression.startsWith('$')) {
      names.push(expression);
    }
  }

  return [...new Set(names)];
}
