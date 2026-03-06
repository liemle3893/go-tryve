import { describe, it, expect } from 'vitest'

describe('DSL Package Exports', () => {
  it('exports test builder function', async () => {
    const { test } = await import('../../../dist/dsl/index.js')
    expect(typeof test).toBe('function')
  })

  it('exports http step builders', async () => {
    const { http } = await import('../../../dist/dsl/index.js')
    expect(typeof http.get).toBe('function')
    expect(typeof http.post).toBe('function')
    expect(typeof http.put).toBe('function')
    expect(typeof http.patch).toBe('function')
    expect(typeof http.delete).toBe('function')
  })

  it('exports shell step builder', async () => {
    const { shell } = await import('../../../dist/dsl/index.js')
    expect(typeof shell.run).toBe('function')
  })

  it('exports step alias', async () => {
    const { step } = await import('../../../dist/dsl/index.js')
    expect(step.http).toBeDefined()
    expect(step.shell).toBeDefined()
  })
})
