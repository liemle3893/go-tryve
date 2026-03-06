/**
 * Adapter Registry Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('Kafka adapter', () => {
  it('should create Kafka adapter when configured', async () => {
    vi.mock('kafkajs', () => ({
      Kafka: vi.fn().mockImplementation(() => ({
        producer: vi.fn().mockReturnValue({
          connect: vi.fn().mockResolvedValue(undefined),
          disconnect: vi.fn().mockResolvedValue(undefined),
          send: vi.fn().mockResolvedValue(undefined),
        }),
        consumer: vi.fn().mockReturnValue({
          connect: vi.fn().mockResolvedValue(undefined),
          disconnect: vi.fn().mockResolvedValue(undefined),
          subscribe: vi.fn().mockResolvedValue(undefined),
          run: vi.fn().mockResolvedValue(undefined),
          stop: vi.fn().mockResolvedValue(undefined),
        }),
      })),
    }));

    const mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };

    const config = {
      baseUrl: 'http://test',
      adapters: {
        kafka: {
          brokers: ['localhost:9092'],
        },
      },
    };

    const { AdapterRegistry } = await import('../../../src/adapters/adapter-registry');
    const registry = new AdapterRegistry(config, mockLogger, {
      requiredAdapters: new Set(['kafka']),
    });

    expect(registry.has('kafka')).toBe(true);
  });

  it('should not create Kafka adapter when not configured', async () => {
    const mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    };

    const config = {
      baseUrl: 'http://test',
      adapters: {},
    };

    const { AdapterRegistry } = await import('../../../src/adapters/adapter-registry');
    const registry = new AdapterRegistry(config, mockLogger);

    expect(registry.has('kafka')).toBe(false);
  });
});
