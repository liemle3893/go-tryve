# Skill Bundle, Install CLI, and Doc CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `e2e doc <section>` and `e2e install --skills` CLI commands, with a unified skill bundle shipped in the npm package.

**Architecture:** Documentation lives as markdown in `docs/sections/` (source of truth). The `doc` command reads them at runtime via `__dirname` path resolution. The `install` command copies `skills/e2e-runner/SKILL.md` + `docs/sections/*` into `.claude/skills/e2e-runner/`. No build step needed.

**Tech Stack:** Node.js, TypeScript, fs (built-in)

---

### Task 1: Create docs/sections/ from existing docs

Migrate existing `docs/0*.md` files into `docs/sections/` structure. Split `04-adapters.md` into per-adapter subsection files.

**Files:**
- Create: `docs/sections/index.json`
- Create: `docs/sections/yaml-test.md`
- Create: `docs/sections/assertions.md`
- Create: `docs/sections/built-in-functions.md`
- Create: `docs/sections/config.md`
- Create: `docs/sections/cli.md`
- Create: `docs/sections/examples.md`
- Create: `docs/sections/adapters/index.md`
- Create: `docs/sections/adapters/http.md`
- Create: `docs/sections/adapters/postgresql.md`
- Create: `docs/sections/adapters/mongodb.md`
- Create: `docs/sections/adapters/redis.md`
- Create: `docs/sections/adapters/eventhub.md`
- Reference: `docs/03-yaml-tests.md`, `docs/04-adapters.md`, `docs/05-assertions.md`, `docs/06-cli-reference.md`, `docs/07-built-in-functions.md`, `docs/02-configuration.md`
- Reference: `~/.claude/skills/e2e-runner/references/*.md`, `~/.claude/skills/e2e-assertions/references/assertions-full.md`

**Step 1: Create docs/sections/ directory and index.json**

```bash
mkdir -p docs/sections/adapters
```

Create `docs/sections/index.json`:
```json
{
  "yaml-test": { "file": "yaml-test.md", "description": "YAML test file syntax and structure" },
  "assertions": { "file": "assertions.md", "description": "Assertion operators and JSONPath syntax" },
  "built-in-functions": { "file": "built-in-functions.md", "description": "Built-in functions ($uuid, $timestamp, etc.)" },
  "config": { "file": "config.md", "description": "e2e.config.yaml configuration reference" },
  "cli": { "file": "cli.md", "description": "CLI commands and options" },
  "examples": { "file": "examples.md", "description": "Common test patterns and recipes" },
  "adapters": { "file": "adapters/index.md", "description": "Adapter overview and comparison" },
  "adapters.http": { "file": "adapters/http.md", "description": "HTTP adapter for REST API testing" },
  "adapters.postgresql": { "file": "adapters/postgresql.md", "description": "PostgreSQL adapter" },
  "adapters.mongodb": { "file": "adapters/mongodb.md", "description": "MongoDB adapter" },
  "adapters.redis": { "file": "adapters/redis.md", "description": "Redis adapter" },
  "adapters.eventhub": { "file": "adapters/eventhub.md", "description": "Azure EventHub adapter" }
}
```

**Step 2: Migrate doc content into sections**

For each section, take content from the corresponding `docs/0*.md` file. For adapters, split `docs/04-adapters.md` into per-adapter files under `docs/sections/adapters/`.

- `docs/sections/yaml-test.md` ← from `docs/03-yaml-tests.md`
- `docs/sections/assertions.md` ← merge `docs/05-assertions.md` + `~/.claude/skills/e2e-assertions/references/assertions-full.md`
- `docs/sections/built-in-functions.md` ← from `docs/07-built-in-functions.md`
- `docs/sections/config.md` ← from `docs/02-configuration.md`
- `docs/sections/cli.md` ← from `docs/06-cli-reference.md`
- `docs/sections/examples.md` ← new file with common patterns extracted from existing examples
- `docs/sections/adapters/index.md` ← overview section from `docs/04-adapters.md`
- `docs/sections/adapters/http.md` ← HTTP section from `docs/04-adapters.md`
- `docs/sections/adapters/postgresql.md` ← PostgreSQL section from `docs/04-adapters.md`
- `docs/sections/adapters/mongodb.md` ← MongoDB section from `docs/04-adapters.md`
- `docs/sections/adapters/redis.md` ← Redis section from `docs/04-adapters.md`
- `docs/sections/adapters/eventhub.md` ← EventHub section from `docs/04-adapters.md`

**Step 3: Commit**

```bash
git add docs/sections/
git commit -m "docs: create docs/sections/ from existing documentation"
```

---

### Task 2: Create the unified SKILL.md

**Files:**
- Create: `skills/e2e-runner/SKILL.md`

**Step 1: Create skills directory**

```bash
mkdir -p skills/e2e-runner
```

**Step 2: Write SKILL.md**

Create `skills/e2e-runner/SKILL.md` with frontmatter + quick reference + links to references. Follow the playwright-cli pattern:

```markdown
---
name: e2e-runner
description: This skill should be used when writing E2E tests for APIs and databases using the @liemle3893/go-tryve framework. Use when creating YAML test files, configuring adapters (HTTP, PostgreSQL, MongoDB, Redis, EventHub), writing assertions, or running tests. Provides complete syntax reference for YAML tests, assertion operators, variable interpolation, and built-in functions.
---

# E2E Test Runner

## Quick Start

[Minimal YAML test example]

## CLI Commands

[Table of commands]

## Test File Structure

[Skeleton]

## Variable Interpolation

[Cheat sheet]

## Assertion Operators

[Quick reference table from e2e-assertions]

## Reference Files

- YAML Test Syntax: [references/yaml-test.md](references/yaml-test.md)
- Assertions: [references/assertions.md](references/assertions.md)
- Built-in Functions: [references/built-in-functions.md](references/built-in-functions.md)
- Configuration: [references/config.md](references/config.md)
- CLI Reference: [references/cli.md](references/cli.md)
- Examples: [references/examples.md](references/examples.md)
- Adapters Overview: [references/adapters/index.md](references/adapters/index.md)
- HTTP Adapter: [references/adapters/http.md](references/adapters/http.md)
- PostgreSQL Adapter: [references/adapters/postgresql.md](references/adapters/postgresql.md)
- MongoDB Adapter: [references/adapters/mongodb.md](references/adapters/mongodb.md)
- Redis Adapter: [references/adapters/redis.md](references/adapters/redis.md)
- EventHub Adapter: [references/adapters/eventhub.md](references/adapters/eventhub.md)
```

**Step 3: Commit**

```bash
git add skills/
git commit -m "feat: create unified skill bundle (SKILL.md)"
```

---

### Task 3: Add `doc` and `install` to CLICommand type

**Files:**
- Modify: `src/types.ts:202`

**Step 1: Update CLICommand type**

Change line 202 from:
```typescript
export type CLICommand = 'run' | 'validate' | 'list' | 'health' | 'init' | 'test';
```
to:
```typescript
export type CLICommand = 'run' | 'validate' | 'list' | 'health' | 'init' | 'test' | 'doc' | 'install';
```

**Step 2: Commit**

```bash
git add src/types.ts
git commit -m "feat: add doc and install to CLICommand type"
```

---

### Task 4: Implement doc command

**Files:**
- Create: `src/cli/doc.command.ts`

**Step 1: Write doc.command.ts**

```typescript
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
 * Print all available sections.
 */
function listSections(index: SectionIndex): void {
  console.log('Available documentation sections:\n')
  const maxLen = Math.max(...Object.keys(index).map(k => k.length))
  for (const [name, entry] of Object.entries(index)) {
    console.log(`  ${name.padEnd(maxLen + 2)} ${entry.description}`)
  }
  console.log('\nUsage: e2e doc <section>')
  console.log('Example: e2e doc assertions')
  console.log('Example: e2e doc adapters.http')
}

/**
 * Handle the doc command.
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
```

**Step 2: Commit**

```bash
git add src/cli/doc.command.ts
git commit -m "feat: implement e2e doc command"
```

---

### Task 5: Implement install command

**Files:**
- Create: `src/cli/install.command.ts`

**Step 1: Write install.command.ts**

```typescript
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
 * Handle the install command.
 */
export async function installCommand(args: CLIArgs): Promise<{ exitCode: number }> {
  const flags = args.patterns
  const wantsSkills = flags.includes('--skills')

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

  console.log(`✓ Skills installed to .claude/skills/e2e-runner`)
  return { exitCode: EXIT_CODES.SUCCESS }
}
```

**Step 2: Commit**

```bash
git add src/cli/install.command.ts
git commit -m "feat: implement e2e install --skills command"
```

---

### Task 6: Wire up commands in CLI router

**Files:**
- Modify: `src/cli/index.ts:16` (VALID_COMMANDS)
- Modify: `src/cli/index.ts:18-64` (HELP_TEXT)
- Modify: `src/cli/index.ts:220-234` (booleanOptions — add `skills`)
- Modify: `src/index.ts:60-98` (routeCommand switch)

**Step 1: Update VALID_COMMANDS in src/cli/index.ts**

Change line 16:
```typescript
const VALID_COMMANDS: CLICommand[] = ['run', 'validate', 'list', 'health', 'init', 'test']
```
to:
```typescript
const VALID_COMMANDS: CLICommand[] = ['run', 'validate', 'list', 'health', 'init', 'test', 'doc', 'install']
```

**Step 2: Update HELP_TEXT in src/cli/index.ts**

Add to the COMMANDS section:
```
  doc         Show documentation        install     Install skills/plugins
```

Add new section after TEST SUBCOMMANDS:
```
DOC USAGE:
  doc                                 List available sections
  doc <section>                       Show section documentation
  doc adapters.http                   Show HTTP adapter docs
  Available: yaml-test, assertions, built-in-functions, config, cli, examples,
             adapters, adapters.http, adapters.postgresql, adapters.mongodb,
             adapters.redis, adapters.eventhub

INSTALL USAGE:
  install --skills                    Install Claude Code skills to project
```

**Step 3: Add `skills` to booleanOptions in parseLongOption**

Add `'skills'` to the booleanOptions array on line ~220-234.

Also add to keyMap in normalizeLongOption:
```typescript
skills: 'skills',
```

**Step 4: Add routes in src/index.ts routeCommand()**

Add imports at top:
```typescript
import { docCommand } from './cli/doc.command'
import { installCommand } from './cli/install.command'
```

Add cases to switch:
```typescript
case 'doc': {
    const result = await docCommand(args)
    return result.exitCode
}

case 'install': {
    const result = await installCommand(args)
    return result.exitCode
}
```

**Step 5: Commit**

```bash
git add src/cli/index.ts src/index.ts
git commit -m "feat: wire up doc and install commands in CLI router"
```

---

### Task 7: Update package.json files array

**Files:**
- Modify: `package.json:13-17`

**Step 1: Add docs/sections and skills to files**

Change:
```json
"files": [
    "dist",
    "bin",
    "README.md"
],
```
to:
```json
"files": [
    "dist",
    "bin",
    "docs/sections",
    "skills",
    "README.md"
],
```

**Step 2: Commit**

```bash
git add package.json
git commit -m "chore: include docs/sections and skills in npm package"
```

---

### Task 8: Build and verify

**Step 1: Build**

```bash
npm run build
```

Expected: Clean compilation, no errors.

**Step 2: Test doc command**

```bash
./bin/e2e.js doc
```

Expected: Lists all sections with descriptions.

```bash
./bin/e2e.js doc assertions
```

Expected: Prints assertions documentation.

```bash
./bin/e2e.js doc adapters.http
```

Expected: Prints HTTP adapter documentation.

```bash
./bin/e2e.js doc nonexistent
```

Expected: Error message + section list.

**Step 3: Test install command**

```bash
# Clean any existing install
rm -rf /tmp/test-install && mkdir /tmp/test-install && cd /tmp/test-install
```

```bash
# Run install from the project
node /Users/liemlhd/Documents/git/Personal/e2e-runner/bin/e2e.js install --skills
```

Expected: `✓ Skills installed to .claude/skills/e2e-runner`

Verify:
```bash
ls -la .claude/skills/e2e-runner/
# Should show: SKILL.md, references/
ls .claude/skills/e2e-runner/references/
# Should show: yaml-test.md, assertions.md, adapters/, etc.
# Should NOT show: index.json
```

**Step 4: Test help text**

```bash
./bin/e2e.js --help
```

Expected: Shows doc and install in command list.

**Step 5: Commit any fixes**

If any issues found, fix and commit.

---

### Task 9: Final commit — update CLAUDE.md CLI section

**Files:**
- Modify: `CLAUDE.md` (CLI Commands section)

**Step 1: Add doc and install to CLAUDE.md CLI section**

Add to CLI Commands:
```bash
./bin/e2e.js doc                          # List documentation sections
./bin/e2e.js doc assertions               # Show assertions reference
./bin/e2e.js doc adapters.http            # Show HTTP adapter docs
./bin/e2e.js install --skills             # Install Claude Code skills
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add doc and install commands to CLAUDE.md"
```
