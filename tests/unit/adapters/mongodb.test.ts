import { describe, it, expect, vi } from 'vitest'

describe('MongoDBAdapter ObjectId import', () => {
  it('imports mongodb module once (at connect time), not once per normalizeFilter call', async () => {
    // Track how many times the mongodb module is dynamically imported
    let importCount = 0
    const RealObjectId = (await import('mongodb')).ObjectId

    vi.doMock('mongodb', () => {
      importCount++
      return {
        MongoClient: vi.fn(() => ({
          connect: vi.fn().mockResolvedValue(undefined),
          close: vi.fn().mockResolvedValue(undefined),
          db: vi.fn(() => ({
            command: vi.fn().mockResolvedValue({ ok: 1 }),
            collection: vi.fn(() => ({
              findOne: vi.fn().mockResolvedValue({ _id: 'abc', name: 'test' }),
            })),
          })),
        })),
        ObjectId: RealObjectId,
      }
    })

    const { MongoDBAdapter } = await import('../../../src/adapters/mongodb.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new MongoDBAdapter(
      { connectionString: 'mongodb://localhost:27017', database: 'test' },
      logger
    )
    await adapter.connect()
    const importCountAfterConnect = importCount

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    // Execute findOne three times — each currently re-imports mongodb
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Alice' } }, ctx)
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Bob' } }, ctx)
    await adapter.execute('findOne', { collection: 'users', filter: { name: 'Carol' } }, ctx)

    // After the fix, importCount should not increase beyond connect time
    expect(importCount).toBe(importCountAfterConnect)

    vi.doUnmock('mongodb')
  })
})
