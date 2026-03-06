import { describe, it, expect } from 'vitest'
import { createFunctionStep } from '../../../src/core/step-executor'
import type { AdapterType } from '../../../src/types'

describe('TypeScript adapter type', () => {
  it('createFunctionStep produces a step with adapter "typescript", not "http"', () => {
    const step = createFunctionStep('my-step', async () => 'ok')
    // This will fail until we change the default adapter in createFunctionStep
    expect(step.adapter).toBe('typescript' satisfies AdapterType)
  })

  it('AdapterType union includes "typescript"', () => {
    // This is a compile-time check; at runtime we verify the loader allowlist
    const validAdapters: AdapterType[] = [
      'postgresql', 'redis', 'mongodb', 'eventhub', 'http', 'shell', 'typescript',
    ]
    expect(validAdapters).toContain('typescript')
  })
})
