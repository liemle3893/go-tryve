# Skill Bundle, Install CLI, and Doc CLI Design

**Date:** 2026-03-03
**Status:** Approved

## Summary

Add three capabilities to e2e-runner:

1. **Skill bundle** — A unified Claude Code skill shipped inside the npm package
2. **`e2e install --skills`** — CLI command to install the skill bundle into a project
3. **`e2e doc <section>[.<subsection>]`** — CLI command to print documentation to terminal

## Decisions

- **Install target:** Project-local `.claude/skills/e2e-runner/` (like playwright-cli)
- **Doc mechanism:** CLI command printing to stdout (works outside Claude)
- **Doc sections:** Expanded — yaml-test, adapters (with subsections), assertions, built-in-functions, config, examples, cli
- **Skill structure:** Unified single skill merging e2e-runner and e2e-assertions
- **Architecture:** Shared markdown (Approach A) — `docs/sections/` is source of truth, consumed by both CLI and skill

## Directory Structure

```
e2e-runner/
├── docs/
│   └── sections/                    # Source of truth for all docs
│       ├── index.json               # Section registry (name → file mapping)
│       ├── yaml-test.md             # YAML test syntax reference
│       ├── assertions.md            # All assertion operators
│       ├── built-in-functions.md    # $uuid(), $timestamp(), etc.
│       ├── config.md                # e2e.config.yaml reference
│       ├── cli.md                   # CLI command reference
│       ├── examples.md              # Common test patterns
│       └── adapters/                # Adapter subsections
│           ├── index.md             # Adapter overview
│           ├── http.md
│           ├── postgresql.md
│           ├── mongodb.md
│           ├── redis.md
│           └── eventhub.md
├── skills/                          # Skill bundle (shipped in npm package)
│   └── e2e-runner/
│       ├── SKILL.md                 # Main skill definition
│       └── (references/ created at install time from docs/sections/)
├── package.json                     # files: ["dist", "bin", "docs/sections", "skills"]
```

## CLI Commands

### `e2e doc [section]`

```bash
e2e doc                    # List all sections
e2e doc assertions         # Print assertions reference
e2e doc adapters.http      # Print HTTP adapter reference
e2e doc config             # Print config reference
```

- No args: prints section list from index.json with descriptions
- With section: reads markdown file, prints to stdout
- Unknown section: error with valid sections list
- Resolves files via `__dirname/../../docs/sections/`

### `e2e install --skills`

```bash
e2e install --skills
# ✓ Skills installed to .claude/skills/e2e-runner
```

- Copies `skills/e2e-runner/SKILL.md` to `.claude/skills/e2e-runner/SKILL.md`
- Copies `docs/sections/*` to `.claude/skills/e2e-runner/references/`
- Creates directories as needed
- Idempotent (overwrites existing)

## New/Modified Files

### New files
- `src/cli/doc.command.ts` — doc command handler
- `src/cli/install.command.ts` — install command handler
- `docs/sections/index.json` — section registry
- `docs/sections/*.md` — doc content (migrated from existing skill references)
- `docs/sections/adapters/*.md` — adapter subsection docs
- `skills/e2e-runner/SKILL.md` — unified skill definition

### Modified files
- `src/index.ts` — add doc and install to command router
- `src/cli/index.ts` — add doc and install to VALID_COMMANDS, help text, option parsing
- `src/types.ts` — add 'doc' | 'install' to CLICommand type
- `package.json` — add "docs/sections" and "skills" to files array

## Path Resolution

```typescript
// doc.command.ts
const sectionsDir = path.resolve(__dirname, '../../docs/sections')

// install.command.ts
const skillSrc = path.resolve(__dirname, '../../skills/e2e-runner')
const docsSrc = path.resolve(__dirname, '../../docs/sections')
const destDir = path.resolve(process.cwd(), '.claude/skills/e2e-runner')
```

## Error Handling

- `e2e doc unknownSection` → lists valid sections
- `e2e install --skills` when `.claude/` doesn't exist → creates it
- Missing doc files → clear error with package integrity hint
