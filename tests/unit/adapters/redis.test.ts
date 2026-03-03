import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('RedisAdapter.flushPattern uses SCAN not KEYS', () => {
  it('calls scan() instead of keys() when flushing by pattern', async () => {
    const mockScan = vi.fn()
      .mockResolvedValueOnce(['5', ['key:1', 'key:2']])  // cursor 5, 2 keys
      .mockResolvedValueOnce(['0', ['key:3']])             // cursor 0, 1 key (done)
    const mockDel = vi.fn().mockResolvedValue(1)
    const mockKeys = vi.fn()  // should NOT be called

    const fakeClient = {
      scan: mockScan,
      del: mockDel,
      keys: mockKeys,
      ping: vi.fn().mockResolvedValue('PONG'),
      quit: vi.fn().mockResolvedValue(undefined),
      connect: vi.fn().mockResolvedValue(undefined),
    }

    vi.doMock('ioredis', () => ({
      default: vi.fn(() => fakeClient),
    }))

    const { RedisAdapter } = await import('../../../src/adapters/redis.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new RedisAdapter({ connectionString: 'redis://localhost:6379' }, logger)

    // Bypass connect() — inject fake client directly
    ;(adapter as any).client = fakeClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    await adapter.execute('flushPattern', { pattern: 'key:*' }, ctx)

    // SCAN must have been called; KEYS must NOT have been called
    expect(mockScan).toHaveBeenCalled()
    expect(mockKeys).not.toHaveBeenCalled()

    // All 3 keys should have been deleted across two batches
    const deletedKeys = mockDel.mock.calls.flatMap((call) => call)
    expect(deletedKeys).toEqual(expect.arrayContaining(['key:1', 'key:2', 'key:3']))

    vi.doUnmock('ioredis')
  })

  it('returns 0 and makes no del calls when no keys match the pattern', async () => {
    const mockScan = vi.fn().mockResolvedValue(['0', []])  // empty, done immediately
    const mockDel = vi.fn()

    const fakeClient = {
      scan: mockScan,
      del: mockDel,
      keys: vi.fn(),
      ping: vi.fn().mockResolvedValue('PONG'),
    }

    vi.doMock('ioredis', () => ({
      default: vi.fn(() => fakeClient),
    }))

    const { RedisAdapter } = await import('../../../src/adapters/redis.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new RedisAdapter({ connectionString: 'redis://localhost:6379' }, logger)
    ;(adapter as any).client = fakeClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    const result = await adapter.execute('flushPattern', { pattern: 'no-match:*' }, ctx)

    expect(mockDel).not.toHaveBeenCalled()
    expect(result.success).toBe(true)

    vi.doUnmock('ioredis')
  })
})
