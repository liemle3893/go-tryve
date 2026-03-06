import { describe, it, expect, vi } from 'vitest'
import { StepExecutor } from '../../../src/core/step-executor'
import { AdapterRegistry } from '../../../src/adapters'
import type { AdapterContext, InterpolationContext, UnifiedStep } from '../../../src/types'

function makeLogger() {
  return { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() }
}

function makeContext(): AdapterContext {
  const captured: Record<string, unknown> = {}
  return {
    variables: {},
    captured,
    capture: (name, value) => { captured[name] = value },
    logger: makeLogger(),
    baseUrl: 'http://localhost',
    cookieJar: new Map(),
  }
}

function makeInterpolationContext(): InterpolationContext {
  return { variables: {}, captured: {}, baseUrl: 'http://localhost', env: {} }
}

describe('StepExecutor.validateAssertions', () => {
  it('returns failed status when assert.equals does not match adapter result', async () => {
    const logger = makeLogger()
    const registry = new AdapterRegistry({ baseUrl: 'http://localhost' }, logger)
    const httpAdapter = registry.get('http')

    // Spy on the HTTP adapter's execute method
    const executeSpy = vi.spyOn(httpAdapter, 'execute')
      .mockResolvedValue({ success: true, data: { value: 'actual' }, duration: 1 })

    const executor = new StepExecutor(registry, {
      defaultRetries: 0,
      retryDelay: 0,
      logger: makeLogger(),
    })

    const step: UnifiedStep = {
      id: 'test-step',
      adapter: 'http',
      action: 'request',
      params: { url: '/test', method: 'GET' },
      assert: { equals: 'expected' },  // value is 'actual', so this should fail
    }

    const result = await executor.executeStep(step, makeContext(), makeInterpolationContext())
    expect(result.status).toBe('failed')
    expect(result.error).toBeDefined()

    executeSpy.mockRestore()
  })

  it('passes when assert.equals matches adapter result data', async () => {
    const logger = makeLogger()
    const registry = new AdapterRegistry({ baseUrl: 'http://localhost' }, logger)
    const httpAdapter = registry.get('http')

    // Spy on the HTTP adapter's execute method
    const executeSpy = vi.spyOn(httpAdapter, 'execute')
      .mockResolvedValue({ success: true, data: 'expected', duration: 1 })

    const executor = new StepExecutor(registry, {
      defaultRetries: 0,
      retryDelay: 0,
      logger: makeLogger(),
    })

    const step: UnifiedStep = {
      id: 'test-step',
      adapter: 'http',
      action: 'request',
      params: { url: '/test', method: 'GET' },
      assert: { equals: 'expected' },
    }

    const result = await executor.executeStep(step, makeContext(), makeInterpolationContext())
    expect(result.status).toBe('passed')

    executeSpy.mockRestore()
  })
})
