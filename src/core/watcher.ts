/**
 * E2E Test Runner - File Watcher
 *
 * Monitors test directories for changes and triggers re-runs.
 */

import * as path from 'node:path'
import chokidar from 'chokidar'
import type { FSWatcher } from 'chokidar'
import { minimatch } from 'minimatch'

export interface WatcherOptions {
  paths: string[]
  patterns: string[]
  debounceMs: number
  onChange: (changedPath: string) => void
  onError?: (error: Error) => void
}

export interface Watcher {
  close(): void
}

export function createWatcher(options: WatcherOptions): Watcher {
  const { paths, patterns, debounceMs, onChange, onError } = options

  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let lastChangedPath: string | null = null

  function matchesPattern(filePath: string): boolean {
    return patterns.some(pattern => minimatch(filePath, pattern))
  }

  function handleChange(eventPath: string): void {
    if (!matchesPattern(eventPath)) return

    lastChangedPath = eventPath
    if (debounceTimer) clearTimeout(debounceTimer)

    debounceTimer = setTimeout(() => {
      if (lastChangedPath) onChange(lastChangedPath)
      debounceTimer = null
      lastChangedPath = null
    }, debounceMs)
  }

  const internalWatcher = chokidar.watch(paths, {
    ignored: /(node_modules|\.git)/,
    ignoreInitial: true,
    awaitWriteFinish: { stabilityThreshold: 100, pollInterval: 50 },
  })

  internalWatcher
    .on('add', handleChange)
    .on('change', handleChange)
    .on('unlink', handleChange)

  if (onError) internalWatcher.on('error', onError)

  return {
    close: () => {
      if (debounceTimer) clearTimeout(debounceTimer)
      internalWatcher.close()
    },
  }
}
