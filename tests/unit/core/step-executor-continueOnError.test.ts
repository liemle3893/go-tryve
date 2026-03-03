import { describe, it, expect, beforeEach, vi } from 'vitest';
import { StepExecutor, createStepExecutor } from '../../../src/core/step-executor';
import { AdapterRegistry } from '../../../src/adapters';
import type { AdapterContext, Logger, UnifiedStep, InterpolationContext } from '../../../src/types';

describe('StepExecutor - continueOnError', () => {
  let executor: StepExecutor;
  let mockLogger: Logger;
  let context: AdapterContext;
  let interpolationContext: InterpolationContext;

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };

    const captured: Record<string, unknown> = {};

    context = {
      variables: {},
      captured,
      capture: (name, value) => { captured[name] = value },
      logger: mockLogger,
      baseUrl: 'http://localhost',
      cookieJar: new Map(),
    };

    interpolationContext = {
      variables: {},
      captured: {},
      baseUrl: 'http://localhost',
      env: {},
    };

    const registry = new AdapterRegistry({ baseUrl: 'http://localhost' }, mockLogger, { requiredAdapters: new Set() });
    executor = createStepExecutor(registry, { logger: mockLogger, defaultRetries: 0, retryDelay: 0 });
  });

  it('should return "warned" status when step fails with continueOnError=true', async () => {
    const mockAdapter = {
      execute: vi.fn().mockRejectedValue(new Error('Adapter error')),
      connect: vi.fn(),
      disconnect: vi.fn(),
      healthCheck: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      name: 'http',
    };

    const getAdapterSpy = vi.spyOn(executor['adapters'], 'get' as any).mockReturnValue(mockAdapter as any);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: true,
      params: { url: 'http://example.com' },
    };

    const result = await executor.executeStep(step, context, interpolationContext);

    expect(result.status).toBe('warned');
    expect(result.error).toBeDefined();
    expect(result.error?.message).toBe('Adapter error');
    expect(mockLogger.warn).toHaveBeenCalledWith(
      expect.stringContaining('Step step-1 failed but continueOnError=true'),
    );

    getAdapterSpy.mockRestore();
  });

  it('should return "failed" status when step fails without continueOnError', async () => {
    const mockAdapter = {
      execute: vi.fn().mockRejectedValue(new Error('Adapter error')),
      connect: vi.fn(),
      disconnect: vi.fn(),
      healthCheck: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      name: 'http',
    };

    const getAdapterSpy = vi.spyOn(executor['adapters'], 'get' as any).mockReturnValue(mockAdapter as any);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: false,
      params: { url: 'http://example.com' },
    };

    const result = await executor.executeStep(step, context, interpolationContext);

    expect(result.status).toBe('failed');
    expect(result.error).toBeDefined();
    expect(result.error?.message).toBe('Adapter error');
    expect(mockLogger.error).toHaveBeenCalledWith(
      expect.stringContaining('Step step-1 failed'),
    );

    getAdapterSpy.mockRestore();
  });

  it('should return "passed" status when step succeeds with continueOnError=true', async () => {
    const mockAdapter = {
      execute: vi.fn().mockResolvedValue({ data: 'success' }),
      connect: vi.fn(),
      disconnect: vi.fn(),
      healthCheck: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      name: 'http',
    };

    const getAdapterSpy = vi.spyOn(executor['adapters'], 'get' as any).mockReturnValue(mockAdapter as any);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: true,
      params: { url: 'http://example.com' },
    };

    const result = await executor.executeStep(step, context, interpolationContext);

    expect(result.status).toBe('passed');
    expect(result.error).toBeUndefined();
    expect(result.data).toBe('success');

    getAdapterSpy.mockRestore();
  });
});
