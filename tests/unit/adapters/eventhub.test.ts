import { describe, it, expect, vi, beforeEach } from 'vitest'

// We test the processError path by constructing a minimal mock of the
// EventHubConsumerClient and injecting an error into processError.

describe('EventHubAdapter processError handling', () => {
  it('waitFor rejects when processError is called with an infrastructure error', async () => {
    // Build a fake consumerClient that immediately calls processError
    const infraError = new Error('AMQP link detached')

    const fakeConsumerClient = {
      subscribe: vi.fn().mockImplementation(({ processError }) => {
        // Simulate infrastructure error fired immediately
        setImmediate(() => processError(infraError))
        return { close: vi.fn().mockResolvedValue(undefined) }
      }),
      close: vi.fn().mockResolvedValue(undefined),
    }

    const fakeProducerClient = {
      getEventHubProperties: vi.fn().mockResolvedValue({}),
      createBatch: vi.fn(),
      sendBatch: vi.fn(),
      close: vi.fn().mockResolvedValue(undefined),
    }

    // Dynamically import the adapter after setting up mocks
    vi.doMock('@azure/event-hubs', () => ({
      EventHubProducerClient: vi.fn(() => fakeProducerClient),
      EventHubConsumerClient: vi.fn(() => fakeConsumerClient),
    }))

    const { EventHubAdapter } = await import('../../../src/adapters/eventhub.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new EventHubAdapter(
      { connectionString: 'fake-connection-string', consumerGroup: '$Default' },
      logger
    )

    // Manually set the internal clients (bypassing connect() which needs real Azure)
    ;(adapter as any).producerClient = fakeProducerClient
    ;(adapter as any).consumerClient = fakeConsumerClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    // waitFor should reject, not resolve, when processError fires
    await expect(
      adapter.execute('waitFor', { topic: 'test-topic', timeout: 5000 }, ctx)
    ).rejects.toThrow()

    vi.doUnmock('@azure/event-hubs')
  })

  it('consume rejects when processError is called with an infrastructure error', async () => {
    const infraError = new Error('Connection reset')

    const fakeConsumerClient = {
      subscribe: vi.fn().mockImplementation(({ processError }) => {
        setImmediate(() => processError(infraError))
        return { close: vi.fn().mockResolvedValue(undefined) }
      }),
      close: vi.fn().mockResolvedValue(undefined),
    }

    const fakeProducerClient = {
      getEventHubProperties: vi.fn().mockResolvedValue({}),
      close: vi.fn().mockResolvedValue(undefined),
    }

    vi.doMock('@azure/event-hubs', () => ({
      EventHubProducerClient: vi.fn(() => fakeProducerClient),
      EventHubConsumerClient: vi.fn(() => fakeConsumerClient),
    }))

    const { EventHubAdapter } = await import('../../../src/adapters/eventhub.adapter')
    const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
    const adapter = new EventHubAdapter(
      { connectionString: 'fake-connection-string', consumerGroup: '$Default' },
      logger
    )
    ;(adapter as any).producerClient = fakeProducerClient
    ;(adapter as any).consumerClient = fakeConsumerClient
    ;(adapter as any).connected = true

    const ctx = {
      variables: {},
      captured: {},
      capture: vi.fn(),
      logger,
      baseUrl: '',
      cookieJar: new Map(),
    }

    await expect(
      adapter.execute('consume', { topic: 'test-topic', count: 1, timeout: 5000 }, ctx)
    ).rejects.toThrow()

    vi.doUnmock('@azure/event-hubs')
  })
})
