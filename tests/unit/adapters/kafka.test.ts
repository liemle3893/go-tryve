/**
 * KafkaAdapter Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock kafkajs
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

describe('KafkaAdapter', () => {
  let adapter: any;
  const mockLogger: any = {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
  };
  const mockContext: any = {
    variables: {},
    captured: {},
    baseUrl: 'http://test',
    logger: mockLogger,
    capture: vi.fn(),
    cookieJar: new Map(),
  };

  const config: any = {
    brokers: ['localhost:9092'],
    clientId: 'test-client',
  };

  beforeEach(async () => {
    vi.clearAllMocks();
    // Dynamic import to load module with mocked dependencies
    const { KafkaAdapter } = await import('../../../src/adapters/kafka.adapter');
    adapter = new KafkaAdapter(config, mockLogger);
  });

  afterEach(async () => {
    if (adapter) {
      await adapter.disconnect().catch(() => {});
    }
  });

  describe('constructor', () => {
    it('should create adapter with name kafka', () => {
      expect(adapter.name).toBe('kafka');
    });
  });

  describe('connect', () => {
    it('should connect successfully with valid config', async () => {
      await adapter.connect();
      expect(adapter['connected']).toBe(true);
    });

    it('should not reconnect if already connected', async () => {
      await adapter.connect();
      await adapter.connect();
      expect(adapter['connected']).toBe(true);
    });
  });

  describe('disconnect', () => {
    it('should disconnect producer and consumer', async () => {
      await adapter.connect();
      await adapter.disconnect();
      expect(adapter['connected']).toBe(false);
    });
  });

  describe('execute - produce action', () => {
    it('should produce message to topic', async () => {
      await adapter.connect();
      const result = await adapter.execute(
        'produce',
        {
          topic: 'test-topic',
          message: { key: 'value' },
        },
        mockContext
      );

      expect(result.success).toBe(true);
      expect(result.data).toEqual({ count: 1 });
    });

    it('should produce multiple messages', async () => {
      await adapter.connect();
      const result = await adapter.execute(
        'produce',
        {
          topic: 'test-topic',
          messages: [{ key: 'value1' }, { key: 'value2' }],
        },
        mockContext
      );

      expect(result.success).toBe(true);
      expect(result.data).toEqual({ count: 2 });
    });
  });

  describe('execute - unknown action', () => {
    it('should throw AdapterError for unknown action', async () => {
      await adapter.connect();
      await expect(adapter.execute('unknown', {}, mockContext)).rejects.toThrow(
        'Unknown action: unknown'
      );
    });
  });

  describe('healthCheck', () => {
    it('should return true when connected', async () => {
      await adapter.connect();
      const healthy = await adapter.healthCheck();
      expect(healthy).toBe(true);
    });

    it('should return false when not connected', async () => {
      const healthy = await adapter.healthCheck();
      expect(healthy).toBe(false);
    });
  });
});
