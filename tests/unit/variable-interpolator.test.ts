/**
 * Tests for variable interpolation.
 *
 * Verifies single-pass and multi-pass interpolation, variable cross-references
 * with topological resolution, cycle detection, depth limits, and built-in
 * function evaluation within variable values.
 */
import { describe, it, expect } from 'vitest';

import {
  interpolate,
  interpolateObject,
  hasInterpolation,
  extractVariableNames,
  resolveVariableValues,
  MAX_INTERPOLATION_DEPTH,
  createInterpolationContext,
} from '../../src/core/variable-interpolator';
import { InterpolationError } from '../../src/errors';
import type { InterpolationContext } from '../../src/types';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Build a minimal InterpolationContext for testing */
function ctx(
  variables: Record<string, unknown> = {},
  captured: Record<string, unknown> = {},
  baseUrl = 'http://localhost',
): InterpolationContext {
  return {
    variables,
    captured,
    baseUrl,
    env: { TEST_VAR: 'env_value', PATH: '/usr/bin' },
  };
}

// ---------------------------------------------------------------------------
// interpolate() — basic behaviour
// ---------------------------------------------------------------------------

describe('interpolate()', () => {
  it('returns non-string input unchanged', () => {
    expect(interpolate(null as unknown as string, ctx())).toBeNull();
    expect(interpolate(undefined as unknown as string, ctx())).toBeUndefined();
    expect(interpolate('' as string, ctx())).toBe('');
  });

  it('returns string without placeholders unchanged', () => {
    expect(interpolate('hello world', ctx())).toBe('hello world');
  });

  it('resolves a simple variable reference', () => {
    const result = interpolate('Hello {{name}}', ctx({ name: 'World' }));
    expect(result).toBe('Hello World');
  });

  it('resolves multiple variables in a single string', () => {
    const result = interpolate(
      '{{greeting}}, {{name}}!',
      ctx({ greeting: 'Hi', name: 'Alice' }),
    );
    expect(result).toBe('Hi, Alice!');
  });

  it('resolves baseUrl', () => {
    expect(interpolate('{{baseUrl}}/api', ctx())).toBe(
      'http://localhost/api',
    );
  });

  it('resolves captured values with captured. prefix', () => {
    const result = interpolate(
      'id={{captured.user_id}}',
      ctx({}, { user_id: 42 }),
    );
    expect(result).toBe('id=42');
  });

  it('resolves captured values without prefix when key exists', () => {
    const result = interpolate('id={{user_id}}', ctx({}, { user_id: 42 }));
    expect(result).toBe('id=42');
  });

  it('falls back to env variables', () => {
    const context = ctx();
    context.env = { MY_KEY: 'secret' };
    expect(interpolate('key={{MY_KEY}}', context)).toBe('key=secret');
  });

  it('throws InterpolationError for unknown variable', () => {
    expect(() => interpolate('{{unknown}}', ctx())).toThrow(InterpolationError);
  });

  // ---------------------------------------------------------------------------
  // Multi-pass resolution
  // ---------------------------------------------------------------------------

  describe('multi-pass resolution', () => {
    it('resolves two-level nested references', () => {
      const variables = { inner: 'resolved', outer: '{{inner}}_suffix' };
      const result = interpolate('{{outer}}', ctx(variables));
      expect(result).toBe('resolved_suffix');
    });

    it('resolves three-level nested references', () => {
      const variables = {
        a: 'base',
        b: '{{a}}_mid',
        c: '{{b}}_end',
      };
      const result = interpolate('{{c}}', ctx(variables));
      expect(result).toBe('base_mid_end');
    });

    it('stops immediately when no placeholders remain', () => {
      // Single pass suffices — no extra work
      const result = interpolate('plain text', ctx());
      expect(result).toBe('plain text');
    });
  });

  // ---------------------------------------------------------------------------
  // Cycle detection
  // ---------------------------------------------------------------------------

  describe('cycle detection', () => {
    it('detects a direct two-variable cycle', () => {
      const variables = { a: '{{b}}', b: '{{a}}' };
      expect(() => interpolate('{{a}}', ctx(variables))).toThrow(
        InterpolationError,
      );
    });

    it('detects a self-referencing variable', () => {
      const variables = { loop: '{{loop}}' };
      expect(() => interpolate('{{loop}}', ctx(variables))).toThrow(
        InterpolationError,
      );
    });

    it('detects a three-way cycle', () => {
      const variables = { x: '{{y}}', y: '{{z}}', z: '{{x}}' };
      expect(() => interpolate('{{x}}', ctx(variables))).toThrow(
        InterpolationError,
      );
    });
  });

  // ---------------------------------------------------------------------------
  // Built-in functions
  // ---------------------------------------------------------------------------

  describe('built-in functions', () => {
    it('evaluates $uuid()', () => {
      const result = interpolate('{{$uuid()}}', ctx());
      expect(result).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/,
      );
    });

    it('evaluates $lower()', () => {
      expect(interpolate('{{$lower(HELLO)}}', ctx())).toBe('hello');
    });

    it('evaluates $upper()', () => {
      expect(interpolate('{{$upper(hello)}}', ctx())).toBe('HELLO');
    });

    it('evaluates $base64()', () => {
      expect(interpolate('{{$base64(hello)}}', ctx())).toBe(
        Buffer.from('hello').toString('base64'),
      );
    });

    it('throws on unknown function', () => {
      expect(() => interpolate('{{$nonexistent()}}', ctx())).toThrow(
        InterpolationError,
      );
    });
  });
});

// ---------------------------------------------------------------------------
// interpolateObject()
// ---------------------------------------------------------------------------

describe('interpolateObject()', () => {
  it('interpolates string values in objects', () => {
    const obj = { url: '{{baseUrl}}/api', name: '{{user}}' };
    const result = interpolateObject(obj, ctx({ user: 'Alice' }));
    expect(result).toEqual({ url: 'http://localhost/api', name: 'Alice' });
  });

  it('interpolates strings in arrays', () => {
    const arr = ['{{baseUrl}}', '{{name}}'];
    const result = interpolateObject(arr, ctx({ name: 'Bob' }));
    expect(result).toEqual(['http://localhost', 'Bob']);
  });

  it('passes non-string values through', () => {
    const obj = { count: 42, active: true, data: null };
    const result = interpolateObject(obj, ctx());
    expect(result).toEqual({ count: 42, active: true, data: null });
  });

  it('handles nested objects', () => {
    const obj = { outer: { inner: '{{val}}' } };
    const result = interpolateObject(obj, ctx({ val: 'deep' }));
    expect(result).toEqual({ outer: { inner: 'deep' } });
  });
});

// ---------------------------------------------------------------------------
// hasInterpolation()
// ---------------------------------------------------------------------------

describe('hasInterpolation()', () => {
  it('returns true for strings with {{...}}', () => {
    expect(hasInterpolation('Hello {{name}}')).toBe(true);
  });

  it('returns false for plain strings', () => {
    expect(hasInterpolation('Hello world')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(hasInterpolation('')).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// extractVariableNames()
// ---------------------------------------------------------------------------

describe('extractVariableNames()', () => {
  it('extracts variable names from template', () => {
    const names = extractVariableNames('{{a}} and {{b}} and {{a}}');
    expect(names).toEqual(['a', 'b']); // deduplicated
  });

  it('ignores function calls', () => {
    const names = extractVariableNames('{{$uuid()}} {{name}}');
    expect(names).toEqual(['name']);
  });
});

// ---------------------------------------------------------------------------
// resolveVariableValues()
// ---------------------------------------------------------------------------

describe('resolveVariableValues()', () => {
  it('resolves simple cross-reference between two variables', () => {
    const vars: Record<string, unknown> = {
      base_id: 'TEST',
      run_id: '{{base_id}}_RUN',
    };
    resolveVariableValues(vars);
    expect(vars.run_id).toBe('TEST_RUN');
  });

  it('resolves a three-level dependency chain', () => {
    const vars: Record<string, unknown> = {
      a: 'root',
      b: '{{a}}_mid',
      c: '{{b}}_end',
    };
    resolveVariableValues(vars);
    expect(vars.a).toBe('root');
    expect(vars.b).toBe('root_mid');
    expect(vars.c).toBe('root_mid_end');
  });

  it('handles variables with no cross-references', () => {
    const vars: Record<string, unknown> = {
      x: 'plain',
      y: 'also plain',
    };
    resolveVariableValues(vars);
    expect(vars.x).toBe('plain');
    expect(vars.y).toBe('also plain');
  });

  it('passes non-string values through unchanged', () => {
    const vars: Record<string, unknown> = {
      count: 42,
      active: true,
      name: 'Alice',
    };
    resolveVariableValues(vars);
    expect(vars.count).toBe(42);
    expect(vars.active).toBe(true);
    expect(vars.name).toBe('Alice');
  });

  it('defers variables that only reference baseUrl', () => {
    const vars: Record<string, unknown> = {
      endpoint: '{{baseUrl}}/api',
    };
    resolveVariableValues(vars, 'http://example.com');
    // Should remain unresolved — deferred to step time
    expect(vars.endpoint).toBe('{{baseUrl}}/api');
  });

  it('defers variables that only reference captured.*', () => {
    const vars: Record<string, unknown> = {
      user_url: '{{captured.user_id}}/profile',
    };
    resolveVariableValues(vars);
    expect(vars.user_url).toBe('{{captured.user_id}}/profile');
  });

  it('resolves mixed references (some deferred, some resolvable)', () => {
    const vars: Record<string, unknown> = {
      prefix: 'api/v1',
      url: '{{baseUrl}}/{{prefix}}/users',
    };
    resolveVariableValues(vars, 'http://localhost');
    // prefix is resolvable, baseUrl is deferred — but because the variable
    // contains a non-deferred reference (prefix), it should still be resolved
    expect(vars.url).toBe('http://localhost/api/v1/users');
  });

  it('resolves built-in functions in variable values', () => {
    const vars: Record<string, unknown> = {
      email: 'test-{{$lower(HELLO)}}@example.com',
    };
    resolveVariableValues(vars);
    expect(vars.email).toBe('test-hello@example.com');
  });

  // ---------------------------------------------------------------------------
  // Cycle detection in resolveVariableValues
  // ---------------------------------------------------------------------------

  describe('cycle detection', () => {
    it('detects direct two-variable cycle', () => {
      const vars: Record<string, unknown> = {
        a: '{{b}}',
        b: '{{a}}',
      };
      expect(() => resolveVariableValues(vars)).toThrow(InterpolationError);
      expect(() => resolveVariableValues(vars)).toThrow(/Circular/);
    });

    it('detects self-referencing variable', () => {
      const vars: Record<string, unknown> = {
        loop: 'prefix_{{loop}}_suffix',
      };
      expect(() => resolveVariableValues(vars)).toThrow(InterpolationError);
    });

    it('detects three-way cycle', () => {
      const vars: Record<string, unknown> = {
        x: '{{y}}',
        y: '{{z}}',
        z: '{{x}}',
      };
      expect(() => resolveVariableValues(vars)).toThrow(InterpolationError);
      expect(() => resolveVariableValues(vars)).toThrow(/Circular/);
    });
  });
});

// ---------------------------------------------------------------------------
// MAX_INTERPOLATION_DEPTH
// ---------------------------------------------------------------------------

describe('MAX_INTERPOLATION_DEPTH', () => {
  it('is exported and equals 10', () => {
    expect(MAX_INTERPOLATION_DEPTH).toBe(10);
  });
});

// ---------------------------------------------------------------------------
// createInterpolationContext()
// ---------------------------------------------------------------------------

describe('createInterpolationContext()', () => {
  it('creates a valid context object', () => {
    const context = createInterpolationContext(
      { key: 'val' },
      { cap: 'data' },
      'http://localhost',
    );
    expect(context.variables).toEqual({ key: 'val' });
    expect(context.captured).toEqual({ cap: 'data' });
    expect(context.baseUrl).toBe('http://localhost');
    expect(context.env).toBeDefined();
  });
});
