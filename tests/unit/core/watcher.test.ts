import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { createWatcher, type WatcherOptions } from '../../../src/core/watcher'

describe('createWatcher', () => {
  let tempDir: string
  let watcher: ReturnType<typeof createWatcher> | null = null

  beforeEach(() => {
    // Use unique temp directory with random suffix to prevent collisions
    const uniqueId = `${Date.now()}-${Math.random().toString(36).substring(7)}`
    tempDir = fs.mkdtempSync(path.join(process.cwd(), `watcher-test-${uniqueId}-`))
  })

  afterEach(async () => {
    // Close watcher first to release file handles
    if (watcher) {
      watcher.close()
      watcher = null
    }

    // Give time for file handles to be released
    await new Promise(resolve => setTimeout(resolve, 100))

    // Clean up temp directory
    try {
      fs.rmSync(tempDir, { recursive: true, force: true })
    } catch (error) {
      // Ignore cleanup errors - temp dirs are in node_modules anyway
    }
  })

  it('calls onChange callback when a .test.yaml file is modified', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'example.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))

    // Modify file
    fs.writeFileSync(testFile, 'name: test-modified')

    // Wait for debounce + file event propagation
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).toHaveBeenCalled()
  })

  it('debounces multiple rapid changes into a single callback', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'debounce.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))

    // Make multiple rapid changes
    fs.writeFileSync(testFile, 'name: change1')
    fs.writeFileSync(testFile, 'name: change2')
    fs.writeFileSync(testFile, 'name: change3')

    // Wait for debounce to complete (debounceMs + buffer)
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).toHaveBeenCalledTimes(1)
  })

  it('ignores files that do not match test patterns', async () => {
    const onChange = vi.fn()
    const configFile = path.join(tempDir, 'config.yaml')
    fs.writeFileSync(configFile, 'setting: value')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))

    // Modify non-test file
    fs.writeFileSync(configFile, 'setting: new-value')

    // Wait to ensure no callback fires
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).not.toHaveBeenCalled()
  })

  it('close() stops the watcher and prevents further callbacks', async () => {
    const onChange = vi.fn()
    const testFile = path.join(tempDir, 'close.test.yaml')
    fs.writeFileSync(testFile, 'name: test')

    watcher = createWatcher({
      paths: [tempDir],
      patterns: ['**/*.test.yaml', '**/*.test.ts'],
      debounceMs: 100,
      onChange,
    })

    // Wait for watcher to fully initialize
    await new Promise(resolve => setTimeout(resolve, 300))

    // Close watcher
    watcher.close()
    watcher = null

    // Small delay to ensure close is processed
    await new Promise(resolve => setTimeout(resolve, 100))

    // Modify file after close
    fs.writeFileSync(testFile, 'name: after-close')

    // Wait to ensure no callback fires
    await new Promise(resolve => setTimeout(resolve, 500))

    expect(onChange).not.toHaveBeenCalled()
  })
})
