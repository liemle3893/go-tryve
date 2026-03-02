/**
 * E2E Test Runner - Install Command
 *
 * Installs Claude Code skills to the current project.
 */

import * as fs from 'fs'
import * as path from 'path'
import type { CLIArgs } from '../types'
import { EXIT_CODES } from '../utils/exit-codes'

/**
 * Recursively copy a directory.
 */
function copyDirSync(src: string, dest: string): void {
    fs.mkdirSync(dest, { recursive: true })
    for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
        const srcPath = path.join(src, entry.name)
        const destPath = path.join(dest, entry.name)
        if (entry.isDirectory()) {
            copyDirSync(srcPath, destPath)
        } else {
            fs.copyFileSync(srcPath, destPath)
        }
    }
}

/**
 * Handle the install command — installs skill bundle to project.
 */
export async function installCommand(args: CLIArgs): Promise<{ exitCode: number }> {
    const wantsSkills = (args.options as unknown as Record<string, unknown>).skills === true

    if (!wantsSkills) {
        console.log('Usage: e2e install --skills')
        console.log('')
        console.log('Options:')
        console.log('  --skills    Install Claude Code skills to .claude/skills/e2e-runner/')
        return { exitCode: EXIT_CODES.SUCCESS }
    }

    const skillSrc = path.resolve(__dirname, '../../skills/e2e-runner')
    const docsSrc = path.resolve(__dirname, '../../docs/sections')
    const destDir = path.resolve(process.cwd(), '.claude/skills/e2e-runner')

    // Verify source files exist
    if (!fs.existsSync(path.join(skillSrc, 'SKILL.md'))) {
        console.error('Error: Skill bundle not found. Package may be corrupted.')
        return { exitCode: EXIT_CODES.FATAL }
    }
    if (!fs.existsSync(docsSrc)) {
        console.error('Error: Documentation sections not found. Package may be corrupted.')
        return { exitCode: EXIT_CODES.FATAL }
    }

    // Create destination and copy
    fs.mkdirSync(destDir, { recursive: true })

    // Copy SKILL.md
    fs.copyFileSync(path.join(skillSrc, 'SKILL.md'), path.join(destDir, 'SKILL.md'))

    // Copy docs/sections/ into references/
    const refsDir = path.join(destDir, 'references')
    copyDirSync(docsSrc, refsDir)

    // Remove index.json from references (not needed by the skill)
    const refsIndex = path.join(refsDir, 'index.json')
    if (fs.existsSync(refsIndex)) {
        fs.unlinkSync(refsIndex)
    }

    console.log('✓ Skills installed to .claude/skills/e2e-runner')
    return { exitCode: EXIT_CODES.SUCCESS }
}
