/**
 * E2E Test Runner - Doc Command
 *
 * Displays documentation sections from bundled markdown files.
 */

import * as fs from 'fs'
import * as path from 'path'
import type { CLIArgs } from '../types'
import { EXIT_CODES } from '../utils/exit-codes'

interface SectionEntry {
    file: string
    description: string
}

type SectionIndex = Record<string, SectionEntry>

/**
 * Resolve the docs/sections directory from the package install path.
 */
function getSectionsDir(): string {
    return path.resolve(__dirname, '../../docs/sections')
}

/**
 * Load the section index registry.
 */
function loadIndex(sectionsDir: string): SectionIndex {
    const indexPath = path.join(sectionsDir, 'index.json')
    if (!fs.existsSync(indexPath)) {
        throw new Error(`Section index not found: ${indexPath}`)
    }
    return JSON.parse(fs.readFileSync(indexPath, 'utf-8'))
}

/**
 * Print all available documentation sections.
 */
function listSections(index: SectionIndex): void {
    console.log('Available documentation sections:\n')
    const maxLen = Math.max(...Object.keys(index).map((k) => k.length))
    for (const [name, entry] of Object.entries(index)) {
        console.log(`  ${name.padEnd(maxLen + 2)} ${entry.description}`)
    }
    console.log('\nUsage: e2e doc <section>')
    console.log('Example: e2e doc assertions')
    console.log('Example: e2e doc adapters.http')
}

/**
 * Handle the doc command — prints documentation to stdout.
 */
export async function docCommand(args: CLIArgs): Promise<{ exitCode: number }> {
    const section = args.patterns[0]
    const sectionsDir = getSectionsDir()

    let index: SectionIndex
    try {
        index = loadIndex(sectionsDir)
    } catch {
        console.error('Error: Documentation files not found. Package may be corrupted.')
        return { exitCode: EXIT_CODES.FATAL }
    }

    // No section specified — list all
    if (!section) {
        listSections(index)
        return { exitCode: EXIT_CODES.SUCCESS }
    }

    // Look up section
    const entry = index[section]
    if (!entry) {
        console.error(`Error: Unknown section "${section}"`)
        console.error('')
        listSections(index)
        return { exitCode: EXIT_CODES.VALIDATION_ERROR }
    }

    // Read and print the doc file
    const filePath = path.join(sectionsDir, entry.file)
    if (!fs.existsSync(filePath)) {
        console.error(`Error: Documentation file missing: ${entry.file}`)
        return { exitCode: EXIT_CODES.FATAL }
    }

    const content = fs.readFileSync(filePath, 'utf-8')
    console.log(content)
    return { exitCode: EXIT_CODES.SUCCESS }
}
