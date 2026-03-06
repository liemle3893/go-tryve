import { describe, it, expect } from 'vitest'
import { test, step, http, shell } from '../../../src/dsl'

describe('TypeScript DSL Builder', () => {
  describe('test() builder', () => {
    it('creates a test definition with name and execute phase', () => {
      const definition = test('my-test')
        .execute([
          http.get('/api/health').expectStatus(200)
        ])
        .build()

      expect(definition.name).toBe('my-test')
      expect(definition.execute).toHaveLength(1)
      expect(definition.execute[0].adapter).toBe('http')
      expect(definition.execute[0].action).toBe('request')
    })

    it('adds optional description', () => {
      const definition = test('my-test')
        .description('A sample test')
        .execute([http.get('/api/ping').expectStatus(200)])
        .build()

      expect(definition.description).toBe('A sample test')
    })

    it('adds optional tags', () => {
      const definition = test('my-test')
        .tags('smoke', 'api')
        .execute([http.get('/api/ping').expectStatus(200)])
        .build()

      expect(definition.tags).toEqual(['smoke', 'api'])
    })

    it('adds optional priority', () => {
      const definition = test('my-test')
        .priority('P0')
        .execute([http.get('/api/ping').expectStatus(200)])
        .build()

      expect(definition.priority).toBe('P0')
    })

    it('adds setup phase', () => {
      const definition = test('my-test')
        .setup([
          shell.run('npm run seed')
        ])
        .execute([http.get('/api/ping').expectStatus(200)])
        .build()

      expect(definition.setup).toHaveLength(1)
      expect(definition.setup![0].adapter).toBe('shell')
    })

    it('adds verify phase', () => {
      const definition = test('my-test')
        .execute([http.get('/api/ping').expectStatus(200)])
        .verify([
          http.get('/api/users').expectStatus(200)
        ])
        .build()

      expect(definition.verify).toHaveLength(1)
    })

    it('adds teardown phase', () => {
      const definition = test('my-test')
        .execute([http.get('/api/ping').expectStatus(200)])
        .teardown([
          shell.run('npm run cleanup')
        ])
        .build()

      expect(definition.teardown).toHaveLength(1)
    })
  })

  describe('http step builder', () => {
    it('creates GET request step', () => {
      const s = http.get('/api/users').expectStatus(200).build()

      expect(s.adapter).toBe('http')
      expect(s.action).toBe('request')
      expect(s.params.method).toBe('GET')
      expect(s.params.url).toBe('/api/users')
      expect(s.assert).toEqual({ status: 200 })
    })

    it('creates POST request step with body', () => {
      const s = http.post('/api/users')
        .body({ name: 'John' })
        .expectStatus(201)
        .build()

      expect(s.action).toBe('request')
      expect(s.params.method).toBe('POST')
      expect(s.params.body).toEqual({ name: 'John' })
    })

    it('creates request with headers', () => {
      const s = http.get('/api/private')
        .header('Authorization', 'Bearer token')
        .expectStatus(200)
        .build()

      expect(s.params.headers).toEqual({ Authorization: 'Bearer token' })
    })

    it('captures response value', () => {
      const s = http.get('/api/users/1')
        .capture('userId', '$.id')
        .expectStatus(200)
        .build()

      expect(s.capture).toEqual({ userId: '$.id' })
    })
  })

  describe('shell step builder', () => {
    it('creates shell command step', () => {
      const s = shell.run('echo hello').build()

      expect(s.adapter).toBe('shell')
      expect(s.action).toBe('execute')
      expect(s.params.command).toBe('echo hello')
    })

    it('adds timeout', () => {
      const s = shell.run('sleep 10').timeout(5000).build()

      expect(s.params.timeout).toBe(5000)
    })
  })
})
