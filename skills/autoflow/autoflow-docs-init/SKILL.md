---
name: autoflow-docs-init
description: "Bootstrap design documentation for an existing (gray-field) project by scanning the codebase with parallel mapper agents. Produces docs/design/ with 7 structured markdown files (stack, integrations, architecture, structure, conventions, testing, concerns). Triggers on: '/autoflow-docs-init', 'bootstrap docs', 'init design docs', 'map codebase for docs'."
argument-hint: "[--refresh | --update <doc>]"
metadata:
  context: fork
  model: opus
---

# Autoflow Docs Init

$ARGUMENTS

Orchestrate parallel codebase mapper agents to analyze a project and produce structured design documents in `docs/design/`.

Each agent has fresh context, explores a specific focus area, and **writes documents directly**. The orchestrator only receives confirmation + line counts, then writes the index.

**Output:** `docs/design/` directory with 7 structured documents + 1 index about the codebase.

## When to Use

- Project has code but no design documentation
- User wants to create a documentation baseline for an existing codebase
- Before starting regular `autoflow-deliver` workflows (so implementation summaries in `docs/changes/` have design docs to reference)
- After a major architectural change that invalidates existing docs

## Philosophy

**Why dedicated mapper agents:**
- Fresh context per domain (no token contamination)
- Agents write documents directly (no context transfer back to orchestrator)
- Orchestrator only summarizes what was created (minimal context usage)
- Faster execution (agents run simultaneously)

**Document quality over brevity:**
Include enough detail to be useful as reference. Prioritize practical examples (especially code patterns) over arbitrary brevity.

**Always include file paths:**
Documents are reference material for AI agents when implementing tickets. Always include actual file paths formatted with backticks: `src/services/user.ts`.

**Be prescriptive, not descriptive:**
"Use camelCase for functions" helps an AI write correct code. "Some functions use camelCase" doesn't.

## Output Structure

```
docs/design/
├── index.md              # Architecture overview + links to all sections
├── stack.md              # Languages, runtime, frameworks, dependencies
├── integrations.md       # External APIs, databases, auth, webhooks
├── architecture.md       # Patterns, layers, data flow, abstractions
├── structure.md          # Directory layout, naming conventions, where to add code
├── conventions.md        # Code style, naming, error handling, logging
├── testing.md            # Test framework, patterns, mocking, coverage
└── concerns.md           # Tech debt, bugs, security, fragile areas
```

Each file has YAML frontmatter:
```yaml
---
project: <project-name>
generated_by: autoflow-docs-init
generated_at: <YYYY-MM-DD>
last_synced: <YYYY-MM-DD>
source_hash: <short git SHA>
focus: <tech|arch|quality|concerns>
---
```

---

## Process

<step name="check_existing">
Check if `docs/design/` already exists.

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel)
ls -la "${PROJECT_ROOT}/docs/design/" 2>/dev/null
```

**If exists:**

```
docs/design/ already exists with these documents:
[List files found]

What's next?
1. Refresh — Delete existing and remap codebase
2. Update — Keep existing, only update specific documents
3. Skip — Use existing design docs as-is
```

Wait for user response via AskUserQuestion.

- If "Refresh": Delete `docs/design/`, continue to detect
- If "Update": Ask which documents to update, continue to spawn_agents (filtered)
- If "Skip": Exit workflow

**If doesn't exist:** Continue to detect.
</step>

<step name="detect">
Detect project signals to configure mapper dispatch.

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel)
PROJECT_NAME=$(basename "$PROJECT_ROOT")
SOURCE_HASH=$(git rev-parse --short HEAD)
DATE=$(date +%Y-%m-%d)
```

**Detect language:**
```bash
# Check for language indicators
ls "$PROJECT_ROOT"/go.mod "$PROJECT_ROOT"/package.json "$PROJECT_ROOT"/requirements.txt "$PROJECT_ROOT"/Cargo.toml "$PROJECT_ROOT"/pyproject.toml 2>/dev/null
```

Read `.autoflow/bootstrap.json` if it exists for `language` field. Otherwise infer:
- `go.mod` → go
- `package.json` → typescript/javascript
- `requirements.txt` or `pyproject.toml` → python
- `Cargo.toml` → rust
- None → unknown

**Detect key paths (existence only — do NOT read contents):**
```bash
# API specs
ls "$PROJECT_ROOT"/docs/openapi/openapi.yaml "$PROJECT_ROOT"/docs/swagger.yaml "$PROJECT_ROOT"/docs/swagger.json 2>/dev/null

# Source directories
ls -d "$PROJECT_ROOT"/src/functions/ "$PROJECT_ROOT"/internal/handlers/ "$PROJECT_ROOT"/src/routes/ "$PROJECT_ROOT"/cmd/ 2>/dev/null
ls -d "$PROJECT_ROOT"/src/services/ "$PROJECT_ROOT"/internal/services/ 2>/dev/null
ls -d "$PROJECT_ROOT"/src/db/ "$PROJECT_ROOT"/internal/repositories/ 2>/dev/null

# Migrations
find "$PROJECT_ROOT" -type d -name "migrations" -not -path "*/node_modules/*" 2>/dev/null | head -5

# Config
ls "$PROJECT_ROOT"/CLAUDE.md "$PROJECT_ROOT"/CLAUDE.local.md "$PROJECT_ROOT"/docker-compose.yml "$PROJECT_ROOT"/Makefile 2>/dev/null
```

Store findings for mapper prompts.

**Discover existing documentation:**
```bash
# Existing docs structure (domain knowledge the mappers should reference)
ls -d "$PROJECT_ROOT"/docs/*/ 2>/dev/null
ls "$PROJECT_ROOT"/docs/*.md "$PROJECT_ROOT"/docs/*.txt "$PROJECT_ROOT"/docs/*.json 2>/dev/null

# Project-level context files
ls "$PROJECT_ROOT"/CLAUDE.md "$PROJECT_ROOT"/CLAUDE.local.md "$PROJECT_ROOT"/README.md "$PROJECT_ROOT"/TECH_DEBT.md "$PROJECT_ROOT"/AGENT_MANDATORY_RULES.md 2>/dev/null
```

Build `EXISTING_DOCS` — a newline-separated list of existing doc paths to pass to mappers. These are NOT replaced — mappers reference them in the "Related Documentation" sections of design docs.

Common patterns:
- `docs/openapi/` — hand-maintained OpenAPI specs
- `docs/swagger.*` — auto-generated Swagger specs
- `docs/<domain>/` — domain-specific reference docs (any subfolder with markdown)
- `docs/system-architect/` — C4 diagrams, architecture docs
- `docs/plans/` or `docs/superpowers/plans/` — prior implementation plans
- `docs/superpowers/specs/` — design specifications
</step>

<step name="create_structure">
```bash
mkdir -p "${PROJECT_ROOT}/docs/design"
```
</step>

<step name="spawn_agents">
Spawn 4 parallel `autoflow-codebase-mapper` agents in a **single message**.

Use the Agent tool with `subagent_type="autoflow-codebase-mapper"` and `run_in_background=true` for parallel execution.

**CRITICAL:** Use the dedicated `autoflow-codebase-mapper` agent, NOT `Explore` or general-purpose. The mapper agent writes documents directly.

**Agent 1: Tech Focus**

```
Agent(
  subagent_type="autoflow-codebase-mapper",
  description="Map tech stack: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: tech

PROJECT_ROOT: ${PROJECT_ROOT}
PROJECT_NAME: ${PROJECT_NAME}
LANGUAGE: ${LANGUAGE}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}

Existing documentation to reference (do NOT rewrite these — link to them):
${EXISTING_DOCS}

Analyze this codebase for technology stack and external integrations.

Write these documents to ${PROJECT_ROOT}/docs/design/:
- stack.md — Languages, runtime, frameworks, dependencies, configuration
- integrations.md — External APIs, databases, auth providers, webhooks

Explore thoroughly. Read existing docs for context. Write documents directly using templates. Include a 'Related Documentation' section linking to existing detailed docs. Return confirmation only.
"
)
```

**Agent 2: Architecture Focus**

```
Agent(
  subagent_type="autoflow-codebase-mapper",
  description="Map architecture: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: arch

PROJECT_ROOT: ${PROJECT_ROOT}
PROJECT_NAME: ${PROJECT_NAME}
LANGUAGE: ${LANGUAGE}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}

Existing documentation to reference (do NOT rewrite these — link to them):
${EXISTING_DOCS}

Analyze this codebase architecture and directory structure.

Write these documents to ${PROJECT_ROOT}/docs/design/:
- architecture.md — Pattern, layers, data flow, abstractions, entry points
- structure.md — Directory layout, key locations, naming conventions, where to add new code

Explore thoroughly. Read existing docs for context (especially system-architect/, domain docs). Write documents directly using templates. Include a 'Related Documentation' section. Return confirmation only.
"
)
```

**Agent 3: Quality Focus**

```
Agent(
  subagent_type="autoflow-codebase-mapper",
  description="Map conventions: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: quality

PROJECT_ROOT: ${PROJECT_ROOT}
PROJECT_NAME: ${PROJECT_NAME}
LANGUAGE: ${LANGUAGE}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}

Existing documentation to reference (do NOT rewrite these — link to them):
${EXISTING_DOCS}

Analyze this codebase for coding conventions and testing patterns.

Write these documents to ${PROJECT_ROOT}/docs/design/:
- conventions.md — Code style, naming, patterns, error handling, logging
- testing.md — Framework, structure, mocking, coverage

Explore thoroughly. Read existing docs for context (especially CLAUDE.md conventions). Write documents directly using templates. Include a 'Related Documentation' section. Return confirmation only.
"
)
```

**Agent 4: Concerns Focus**

```
Agent(
  subagent_type="autoflow-codebase-mapper",
  description="Map concerns: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: concerns

PROJECT_ROOT: ${PROJECT_ROOT}
PROJECT_NAME: ${PROJECT_NAME}
LANGUAGE: ${LANGUAGE}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}

Existing documentation to reference (do NOT rewrite these — link to them):
${EXISTING_DOCS}

Analyze this codebase for technical debt, known issues, and areas of concern.

Write this document to ${PROJECT_ROOT}/docs/design/:
- concerns.md — Tech debt, bugs, security, performance, fragile areas, deprecated modules

Explore thoroughly. Read existing docs for context (especially TECH_DEBT.md, CLAUDE.md deprecated systems). Write document directly using template. Include a 'Related Documentation' section. Return confirmation only.
"
)
```

Wait for all 4 agents to complete.
</step>

<step name="verify_output">
Verify all documents were created:

```bash
ls -la "${PROJECT_ROOT}/docs/design/"
wc -l "${PROJECT_ROOT}/docs/design/"*.md
```

**Verification checklist:**
- All 7 documents exist
- No empty documents (each should have >20 lines)
- Frontmatter present in each file

If any documents missing or empty, note which agents may have failed.
</step>

<step name="scan_for_secrets">
**CRITICAL SECURITY CHECK:** Scan output files for accidentally leaked secrets before committing.

```bash
grep -E '(sk-[a-zA-Z0-9]{20,}|sk_live_[a-zA-Z0-9]+|sk_test_[a-zA-Z0-9]+|ghp_[a-zA-Z0-9]{36}|gho_[a-zA-Z0-9]{36}|glpat-[a-zA-Z0-9_-]+|AKIA[A-Z0-9]{16}|xox[baprs]-[a-zA-Z0-9-]+|-----BEGIN.*PRIVATE KEY|eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.)' "${PROJECT_ROOT}/docs/design/"*.md 2>/dev/null && SECRETS_FOUND=true || SECRETS_FOUND=false
```

**If SECRETS_FOUND=true:** Alert user, pause before commit. Wait for confirmation.
**If SECRETS_FOUND=false:** Continue to write_index.
</step>

<step name="write_index">
Write `docs/design/index.md` — the orchestrator writes this one directly (it's a summary, not an analysis):

```markdown
---
project: ${PROJECT_NAME}
generated_by: autoflow-docs-init
generated_at: ${DATE}
last_synced: ${DATE}
source_hash: ${SOURCE_HASH}
---

# ${PROJECT_NAME} — Design Documentation

> Generated by `autoflow-docs-init` from source at `${SOURCE_HASH}`.
> Fill in TODO sections to complete the documentation.

## Documents

| Document | Focus | Description |
|----------|-------|-------------|
| [stack.md](stack.md) | tech | Languages, runtime, frameworks, dependencies |
| [integrations.md](integrations.md) | tech | External APIs, databases, auth, webhooks |
| [architecture.md](architecture.md) | arch | Patterns, layers, data flow, abstractions |
| [structure.md](structure.md) | arch | Directory layout, naming, where to add code |
| [conventions.md](conventions.md) | quality | Code style, naming, error handling, logging |
| [testing.md](testing.md) | quality | Test framework, patterns, mocking, coverage |
| [concerns.md](concerns.md) | concerns | Tech debt, bugs, security, fragile areas |

## Documentation Map

| Path | Purpose | Managed By |
|------|---------|-----------|
| `docs/design/` | System design baseline (this folder) | `/autoflow-docs-init`, `/autoflow-design-sync` |
| `docs/changes/` | Per-ticket implementation summaries | `autoflow-deliver` Step 10 |
${EXISTING_DOCS_TABLE}

## Change History

Implementation summaries accumulate in [`docs/changes/`](../changes/) via `autoflow-deliver` Step 10.
To synthesize changes into these design docs, run `/autoflow-design-sync`.
```

Build `EXISTING_DOCS_TABLE` from the discovered existing docs. Example rows:
```
| `docs/openapi/` | API specification | Manual |
| `docs/<domain>/` | Domain-specific references | Manual |
| `docs/plans/` | Implementation plans | Manual |
| `CLAUDE.md` | Project conventions | Manual |
```

Only include rows for docs that actually exist. This gives developers a single place to find all documentation.

Adjust the table based on which documents were actually created (skip rows for missing docs).
</step>

<step name="commit">
Stage and commit:

```bash
cd "$PROJECT_ROOT"
git add docs/design/
git commit -m "docs: bootstrap design documentation via autoflow-docs-init

Generated from source at ${SOURCE_HASH}.
Contains structural inventory — TODO markers indicate sections needing human input."
```
</step>

<step name="report">
Present completion summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 AUTOFLOW ► DESIGN DOCS INITIALIZED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Created docs/design/:
- index.md ([N] lines) — Architecture overview
- stack.md ([N] lines) — Technologies and dependencies
- integrations.md ([N] lines) — External services and APIs
- architecture.md ([N] lines) — System design and patterns
- structure.md ([N] lines) — Directory layout and organization
- conventions.md ([N] lines) — Code style and patterns
- testing.md ([N] lines) — Test structure and practices
- concerns.md ([N] lines) — Technical debt and issues


---

## Next Steps

1. Review generated docs and fill in TODO sections
2. Run /autoflow-deliver to start accumulating docs/changes/
3. Run /autoflow-design-sync periodically to fold changes back
```
</step>

---

## Failure Handling

| Condition | Action |
|-----------|--------|
| No `.autoflow/bootstrap.json` | Detect language inline, warn user to run `/autoflow-settings` |
| Agent returns empty doc | Write file with "Not detected — fill manually" |
| Agent fails entirely | Log error, note missing doc in index, proceed |
| Secrets detected in output | Alert user, pause before commit |
| `docs/design/` already exists | Ask: Refresh / Update / Skip |
