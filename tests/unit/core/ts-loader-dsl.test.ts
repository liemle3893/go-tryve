import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { loadTSTest } from '../../../src/core/ts-loader'
import type { UnifiedTestDefinition } from '../../../src/types'

describe('ts-loader DSL Integration', () => {
  const tempDir = path.join(__dirname, 'temp-dsl-tests')

  beforeEach(() => {
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true })
    }
  })

  afterEach(() => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true })
    }
  })

  it('loads DSL test with declarative execute steps', async () => {
    const testFile = path.join(tempDir, 'dsl-basic.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../../src/dsl'

export default test('Health Check')
  .description('Verify health endpoint')
  .execute([
    http.get('/health').expectStatus(200)
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)

    expect(definition.name).toBe('Health Check')
    expect(definition.description).toBe('Verify health endpoint')
    expect(definition.execute).toBeDefined()
    expect(Array.isArray(definition.execute)).toBe(true)
    expect(definition.execute.length).toBeGreaterThan(0)
    expect(definition.execute[0].adapter).toBe('http')
    expect(definition.execute[0].action).toBe('request')
  })

  it('preserves DSL test tags and priority', async () => {
    const testFile = path.join(tempDir, 'dsl-metadata.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../../src/dsl'

export default test('API Smoke Test')
  .tags('smoke', 'api', 'critical')
  .priority('P0')
  .execute([
    http.get('/api/ping').expectStatus(200)
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)

    expect(definition.tags).toEqual(['smoke', 'api', 'critical'])
    expect(definition.priority).toBe('P0')
  })

  it('loads DSL test with multiple phases', async () => {
    const testFile = path.join(tempDir, 'dsl-phases.test.ts')
    fs.writeFileSync(testFile, `
import { test, http, shell } from '../../../../src/dsl'

export default test('Full E2E Test')
  .setup([
    shell.run('npm run seed')
  ])
  .execute([
    http.post('/api/users').body({ name: 'Test' }).expectStatus(201)
  ])
  .verify([
    http.get('/api/users/1').expectStatus(200)
  ])
  .teardown([
    shell.run('npm run cleanup')
  ])
  .build()
`)

    const definition = await loadTSTest(testFile)

    expect(definition.setup).toBeDefined()
    expect(definition.setup!.length).toBe(1)
    expect(definition.setup![0].adapter).toBe('shell')

    expect(definition.execute).toBeDefined()
    expect(definition.execute[0].adapter).toBe('http')
    expect(definition.execute[0].action).toBe('request')

    expect(definition.verify).toBeDefined()
    expect(definition.verify!.length).toBe(1)

    expect(definition.teardown).toBeDefined()
    expect(definition.teardown!.length).toBe(1)
  })

  it('sets sourceFile to actual file path for DSL tests', async () => {
    const testFile = path.join(tempDir, 'dsl-source.test.ts')
    fs.writeFileSync(testFile, `
import { test, http } from '../../../../src/dsl'

export default test('Source Test')
  .execute([http.get('/test').expectStatus(200)])
  .build()
`)

    const definition = await loadTSTest(testFile)

    expect(definition.sourceFile).toBe(path.resolve(testFile))
    expect(definition.sourceType).toBe('typescript')
  })

  it('still loads function-based tests correctly', async () => {
    const testFile = path.join(tempDir, 'function-test.test.ts')
    fs.writeFileSync(testFile, `
export default {
  execute: async (ctx: unknown) => {
    console.log('Function-based test')
  }
}
`)

    const definition = await loadTSTest(testFile)

    expect(definition.name).toBe('function-test')
    expect(definition.execute).toBeDefined()
    expect(Array.isArray(definition.execute)).toBe(true)
    // Function-based tests get wrapped in special steps
    expect(definition.execute[0].action).toBe('__typescript_function__')
  })
})
