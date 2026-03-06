/**
 * TypeScript Test DSL
 *
 * Fluent API for defining E2E tests with full type inference.
 *
 * @example
 * ```typescript
 * import { test, http, shell } from 'e2e-runner/dsl'
 *
 * export default test('API Health Check')
 *   .description('Verify API is healthy')
 *   .tags('smoke', 'health')
 *   .priority('P0')
 *   .execute([
 *     http.get('/health').expectStatus(200)
 *   ])
 *   .build()
 * ```
 */

export { test, http, shell, step } from './builder'
export type { TestBuilder, StepBuilder, HttpStepBuilder, ShellStepBuilder } from './types'
