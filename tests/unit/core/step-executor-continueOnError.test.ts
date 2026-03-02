import { describe, it, expect, beforeEach, vi } from 'vitest';
import { StepExecutor, createStepExecutor } from '../../../src/core/step-executor';
import { AdapterRegistry } from '../../../src/adapters';
import type { AdapterContext, Logger, UnifiedStep } from '../../../src/types';

describe('StepExecutor - continueOnError', () => {
  let executor: StepExecutor;
  let mockRegistry: AdapterRegistry;
  let mockLogger: Logger;
  let context: AdapterContext;

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };

    context = {
      logger: mockLogger,
      testResult: {
        name: 'test',
        status: 'passed',
        phases: [],
        duration: 0,
        retryCount: 0,
        capturedValues: {},
      },
    };

    mockRegistry = {
      getAdapter: vi.fn(),
    } as unknown as AdapterRegistry;

    executor = createStepExecutor(mockRegistry, mockLogger);
  });

  it('should return "warned" status when step fails with continueOnError=true', async () => {
    const mockAdapter = {
      execute: vi.fn().mockRejectedValue(new Error('Adapter error')),
    };

    vi.mocked(mockRegistry.getAdapter).mockReturnValue(mockAdapter);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: true,
      params: { url: 'http://example.com' },
    };

    const result = await executor.execute(step, context);

    expect(result.status).toBe('warned');
    expect(result.error).toBeDefined();
    expect(result.error?.message).toBe('Adapter error');
    expect(mockLogger.warn).toHaveBeenCalledWith(
      expect.stringContaining('Step step-1 failed but continueOnError=true'),
    );
  });

  it('should return "failed" status when step fails without continueOnError', async () => {
    const mockAdapter = {
      execute: vi.fn().mockRejectedValue(new Error('Adapter error')),
    };

    vi.mocked(mockRegistry.getAdapter).mockReturnValue(mockAdapter);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: false,
      params: { url: 'http://example.com' },
    };

    const result = await executor.execute(step, context);

    expect(result.status).toBe('failed');
    expect(result.error).toBeDefined();
    expect(result.error?.message).toBe('Adapter error');
    expect(mockLogger.error).toHaveBeenCalledWith(
      expect.stringContaining('Step step-1 failed'),
    );
  });

  it('should return "passed" status when step succeeds with continueOnError=true', async () => {
    const mockAdapter = {
      execute: vi.fn().mockResolvedValue({ data: 'success' }),
    };

    vi.mocked(mockRegistry.getAdapter).mockReturnValue(mockAdapter);

    const step: UnifiedStep = {
      id: 'step-1',
      adapter: 'http',
      action: 'get',
      continueOnError: true,
      params: { url: 'http://example.com' },
    };

    const result = await executor.execute(step, context);

    expect(result.status).toBe('passed');
    expect(result.error).toBeUndefined();
    expect(result.data).toBe('success');
  });
});
