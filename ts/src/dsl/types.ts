/**
 * DSL Type Definitions
 */

import type { TestPriority, UnifiedStep } from '../types'

export interface TestBuilder {
  description(desc: string): TestBuilder
  tags(...tags: string[]): TestBuilder
  priority(p: TestPriority): TestBuilder
  setup(steps: StepBuilder[]): TestBuilder
  execute(steps: StepBuilder[]): TestBuilder
  verify(steps: StepBuilder[]): TestBuilder
  teardown(steps: StepBuilder[]): TestBuilder
  build(): import('../types').UnifiedTestDefinition
}

export interface StepBuilder {
  build(): UnifiedStep
}

export interface HttpStepBuilder extends StepBuilder {
  header(name: string, value: string): HttpStepBuilder
  body(data: unknown): HttpStepBuilder
  expectStatus(code: number): HttpStepBuilder
  capture(name: string, jsonPath: string): HttpStepBuilder
}

export interface ShellStepBuilder extends StepBuilder {
  timeout(ms: number): ShellStepBuilder
  capture(name: string, jsonPath: string): ShellStepBuilder
}
