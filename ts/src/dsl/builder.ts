/**
 * DSL Builders - Fluent API for defining tests
 */

import type { TestPriority, UnifiedStep, UnifiedTestDefinition } from '../types'
import type { TestBuilder, StepBuilder, HttpStepBuilder, ShellStepBuilder } from './types'

// ============================================================================
// Test Builder
// ============================================================================

class TestBuilderImpl implements TestBuilder {
  private _name: string
  private _description?: string
  private _tags?: string[]
  private _priority?: TestPriority
  private _setup?: UnifiedStep[]
  private _execute?: UnifiedStep[]
  private _verify?: UnifiedStep[]
  private _teardown?: UnifiedStep[]

  constructor(name: string) {
    this._name = name
  }

  description(desc: string): TestBuilder {
    this._description = desc
    return this
  }

  tags(...tags: string[]): TestBuilder {
    this._tags = tags
    return this
  }

  priority(p: TestPriority): TestBuilder {
    this._priority = p
    return this
  }

  setup(steps: StepBuilder[]): TestBuilder {
    this._setup = steps.map(s => s.build())
    return this
  }

  execute(steps: StepBuilder[]): TestBuilder {
    this._execute = steps.map(s => s.build())
    return this
  }

  verify(steps: StepBuilder[]): TestBuilder {
    this._verify = steps.map(s => s.build())
    return this
  }

  teardown(steps: StepBuilder[]): TestBuilder {
    this._teardown = steps.map(s => s.build())
    return this
  }

  build(): UnifiedTestDefinition {
    if (!this._execute || this._execute.length === 0) {
      throw new Error('Test must have at least one execute step')
    }

    return {
      name: this._name,
      description: this._description,
      tags: this._tags,
      priority: this._priority,
      setup: this._setup,
      execute: this._execute,
      verify: this._verify,
      teardown: this._teardown,
      sourceFile: 'dsl',
      sourceType: 'typescript',
    }
  }
}

// ============================================================================
// HTTP Step Builder
// ============================================================================

class HttpStepBuilderImpl implements HttpStepBuilder {
  private _method: string
  private _url: string
  private _headers?: Record<string, string>
  private _body?: unknown
  private _assert?: { status?: number }
  private _capture?: Record<string, string>

  constructor(method: string, url: string) {
    this._method = method
    this._url = url
  }

  header(name: string, value: string): HttpStepBuilder {
    this._headers = this._headers || {}
    this._headers[name] = value
    return this
  }

  body(data: unknown): HttpStepBuilder {
    this._body = data
    return this
  }

  expectStatus(code: number): HttpStepBuilder {
    this._assert = this._assert || {}
    this._assert.status = code
    return this
  }

  capture(name: string, jsonPath: string): HttpStepBuilder {
    this._capture = this._capture || {}
    this._capture[name] = jsonPath
    return this
  }

  build(): UnifiedStep {
    return {
      id: `http-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      adapter: 'http',
      action: 'request',
      params: {
        method: this._method.toUpperCase(),
        url: this._url,
        headers: this._headers,
        body: this._body,
      },
      assert: this._assert,
      capture: this._capture,
    }
  }
}

// ============================================================================
// Shell Step Builder
// ============================================================================

class ShellStepBuilderImpl implements ShellStepBuilder {
  private _command: string
  private _timeout?: number
  private _capture?: Record<string, string>

  constructor(command: string) {
    this._command = command
  }

  timeout(ms: number): ShellStepBuilder {
    this._timeout = ms
    return this
  }

  capture(name: string, jsonPath: string): ShellStepBuilder {
    this._capture = this._capture || {}
    this._capture[name] = jsonPath
    return this
  }

  build(): UnifiedStep {
    return {
      id: `shell-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      adapter: 'shell',
      action: 'execute',
      params: {
        command: this._command,
        timeout: this._timeout,
      },
      capture: this._capture,
    }
  }
}

// ============================================================================
// Exports
// ============================================================================

export const test = (name: string) => new TestBuilderImpl(name)

export const http = {
  get: (url: string) => new HttpStepBuilderImpl('GET', url),
  post: (url: string) => new HttpStepBuilderImpl('POST', url),
  put: (url: string) => new HttpStepBuilderImpl('PUT', url),
  patch: (url: string) => new HttpStepBuilderImpl('PATCH', url),
  delete: (url: string) => new HttpStepBuilderImpl('DELETE', url),
}

export const shell = {
  run: (command: string) => new ShellStepBuilderImpl(command),
}

// Alias for step builders
export const step = { http, shell }
