import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { createWatcher, type WatcherOptions } from '../../../src/core/watcher'

describe('createWatcher', () => {
  let tempDir: string
  let watcher: ReturnType<typeof createWatcher> | null = null

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(process.cwd(), 'watcher-test-'))
  })

  afterEach(() => {
    if (watcher) {
      watcher.close()
      watcher = null
    }
    fs.rmSync(tempDir, { recursive: true, force: true })
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

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(testFile, 'name: test-modified')
    await new Promise(resolve => setTimeout(resolve, 300))

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

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(testFile, 'name: change1')
    fs.writeFileSync(testFile, 'name: change2')
    fs.writeFileSync(testFile, 'name: change3')
    await new Promise(resolve => setTimeout(resolve, 300))

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

    await new Promise(resolve => setTimeout(resolve, 200))
    fs.writeFileSync(configFile, 'setting: new-value')
    await new Promise(resolve => setTimeout(resolve, 300))

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

    await new Promise(resolve => setTimeout(resolve, 200))
    watcher.close()
    watcher = null

    fs.writeFileSync(testFile, 'name: after-close')
    await new Promise(resolve => setTimeout(resolve, 300))

    expect(onChange).not.toHaveBeenCalled()
  })
})
