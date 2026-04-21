---
name: autoflow-design-sync
description: "Synthesize accumulated implementation summaries (docs/changes/) into updated design documentation (docs/design/). Dispatches parallel syncer agents per focus area. Run on a cadence — sprint end, release, or on-demand. Triggers on: '/autoflow-design-sync', 'sync design docs', 'update design docs from changes'."
argument-hint: "[--since <date>] [--area <area>] [--dry-run] [--focus <focus>]"
metadata:
  context: fork
  model: opus
---

# Autoflow Design Sync

$ARGUMENTS

Synthesize accumulated per-ticket implementation summaries (`docs/changes/*.md`) into updated design documentation (`docs/design/*.md`). This is the **Tier 1 update** mechanism — it reads the change history and evolves the "what the system IS" view.

## When to Use

- Sprint end / release boundary — batch update design docs
- On-demand when design docs feel stale
- After a burst of tickets that changed a specific area
- Prerequisite: `docs/design/` must exist (run `/autoflow-docs-init` first)
- Prerequisite: `docs/changes/` must have unsynced files (produced by `autoflow-deliver` Step 10)

## The Two-Tier Model

```
Tier 1: docs/design/     — "what the system IS" (updated by this skill)
                ▲
                │ synthesized from
                │
Tier 2: docs/changes/    — "what was DONE" per ticket (written by autoflow-deliver Step 10)
```

## Parameters

| Flag | Optional | Description |
|------|----------|-------------|
| `--since <date>` | yes | Only sync changes after this date (default: `last_synced` from design doc frontmatter) |
| `--area <area>` | yes | Only sync changes tagged with a specific area |
| `--dry-run` | yes | Show what would change without writing |
| `--focus <focus>` | yes | Only update one focus area (tech/arch/quality/concerns) |

## Output Structure

Design docs are updated in-place. No new files are created (except the sync log).

```
docs/
├── design/                     # Updated by this skill
│   ├── index.md                # last_synced bumped
│   ├── stack.md                # tech focus
│   ├── integrations.md         # tech focus
│   ├── architecture.md         # arch focus
│   ├── structure.md            # arch focus
│   ├── conventions.md          # quality focus
│   ├── testing.md              # quality focus
│   └── concerns.md             # concerns focus
└── changes/                    # Read by this skill (not modified)
    ├── 2026-04-12-proj-42-rewards-pagination.md
    ├── 2026-04-13-proj-43-voucher-expiry-fix.md
    └── ...
```

---

## Process

<step name="preflight">
Validate prerequisites.

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel)
DESIGN_DIR="${PROJECT_ROOT}/docs/design"
CHANGES_DIR="${PROJECT_ROOT}/docs/changes"
SOURCE_HASH=$(git rev-parse --short HEAD)
DATE=$(date +%Y-%m-%d)
```

**Check docs/design/ exists:**
```bash
ls "${DESIGN_DIR}/index.md" 2>/dev/null
```
If missing → abort with: "Run `/autoflow-docs-init` first to bootstrap design docs."

**Check docs/changes/ exists and has files:**
```bash
ls "${CHANGES_DIR}/"*.md 2>/dev/null | wc -l
```
If zero → abort with: "No implementation summaries found. Deliver some tickets first via `/autoflow-deliver`."
</step>

<step name="find_unsynced">
Determine which change files haven't been synced yet.

**Read `last_synced` from design doc frontmatter:**
```bash
LAST_SYNCED=$(grep -m1 'last_synced:' "${DESIGN_DIR}/index.md" | sed 's/last_synced: *//' | tr -d '"' | xargs)
```

If `--since` flag was provided, use that date instead.

**Find unsynced change files:**
```bash
# List all change files with their date frontmatter
for f in "${CHANGES_DIR}"/*.md; do
    FILE_DATE=$(grep -m1 'date:' "$f" | sed 's/date: *//' | tr -d '"' | xargs)
    echo "${FILE_DATE}|${f}"
done | sort
```

Filter to files where `FILE_DATE > LAST_SYNCED`. If `--area` flag was provided, also filter by area frontmatter.

**If zero unsynced files:** Report "Design docs are up to date" and exit.

**Present summary to user:**
```
Found N unsynced implementation summaries since ${LAST_SYNCED}:

| Date | Ticket | Title | Area | Type |
|------|--------|-------|------|------|
| 2026-04-12 | PROJ-42 | Add pagination to rewards | rewards | feature |
| 2026-04-13 | PROJ-43 | Fix voucher expiry | vouchers | bugfix |
| ... | ... | ... | ... | ... |

Proceeding to sync into docs/design/.
```

If `--dry-run`: show the summary and exit without dispatching agents.
</step>

<step name="dispatch_syncers">
Dispatch up to 4 parallel `autoflow-design-syncer` agents in a **single message**.

Each agent receives ALL unsynced change file paths and determines relevance for its own focus area.

**CRITICAL:** Use the dedicated `autoflow-design-syncer` agent. The syncer reads existing docs and merges — it does NOT explore the codebase.

If `--focus` was specified, only dispatch that one focus area.

**Agent 1: Tech Focus**

```
Agent(
  subagent_type="autoflow-design-syncer",
  description="Sync tech docs: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: tech

PROJECT_ROOT: ${PROJECT_ROOT}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}
DESIGN_DIR: ${DESIGN_DIR}
CHANGE_FILES:
${UNSYNCED_FILE_PATHS}

Read the unsynced change summaries. Extract relevant tech changes (new dependencies, config changes, platform updates). Update stack.md and integrations.md in docs/design/. Preserve human content. Return confirmation only.
"
)
```

**Agent 2: Architecture Focus**

```
Agent(
  subagent_type="autoflow-design-syncer",
  description="Sync arch docs: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: arch

PROJECT_ROOT: ${PROJECT_ROOT}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}
DESIGN_DIR: ${DESIGN_DIR}
CHANGE_FILES:
${UNSYNCED_FILE_PATHS}

Read the unsynced change summaries. Extract relevant architecture changes (new files, new layers, new entry points, structural changes). Update architecture.md and structure.md in docs/design/. Preserve human content. Return confirmation only.
"
)
```

**Agent 3: Quality Focus**

```
Agent(
  subagent_type="autoflow-design-syncer",
  description="Sync quality docs: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: quality

PROJECT_ROOT: ${PROJECT_ROOT}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}
DESIGN_DIR: ${DESIGN_DIR}
CHANGE_FILES:
${UNSYNCED_FILE_PATHS}

Read the unsynced change summaries. Extract relevant quality changes (new patterns, convention changes, new test approaches). Update conventions.md and testing.md in docs/design/. Preserve human content. Return confirmation only.
"
)
```

**Agent 4: Concerns Focus**

```
Agent(
  subagent_type="autoflow-design-syncer",
  description="Sync concerns docs: ${PROJECT_NAME}",
  run_in_background=true,
  prompt="
Focus: concerns

PROJECT_ROOT: ${PROJECT_ROOT}
DATE: ${DATE}
SOURCE_HASH: ${SOURCE_HASH}
DESIGN_DIR: ${DESIGN_DIR}
CHANGE_FILES:
${UNSYNCED_FILE_PATHS}

Read the unsynced change summaries. Extract relevant concern changes (resolved debt, new workarounds, deprecated modules, new fragile areas). Update concerns.md in docs/design/. Preserve human content. Return confirmation only.
"
)
```

Wait for all agents to complete.
</step>

<step name="collect_results">
Parse agent return lines:
- `## SYNC COMPLETE` — changes were applied
- `## SYNC SKIP: ...` — no relevant changes for that focus

Build a summary of what was updated.
</step>

<step name="update_index">
Update `docs/design/index.md` frontmatter:

Read the current `index.md`, update only:
- `last_synced: ${DATE}`
- `source_hash: ${SOURCE_HASH}`

Preserve all other content.

If no agents applied any changes (all returned SYNC SKIP), still update `last_synced` to record that a sync was attempted and the docs are current.
</step>

<step name="commit">
Stage and commit:

```bash
cd "$PROJECT_ROOT"
git add docs/design/
git commit -m "docs: sync design docs from ${N_CHANGES} implementation summaries

Synced changes from ${FIRST_DATE} to ${LAST_DATE}.
Tickets: ${TICKET_LIST}
Previous sync: ${LAST_SYNCED} | Current sync: ${DATE}"
```
</step>

<step name="report">
Present completion summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 AUTOFLOW ► DESIGN DOCS SYNCED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

| Focus | Documents | Status | From Tickets |
|-------|-----------|--------|--------------|
| tech | stack.md, integrations.md | <updated/skipped> | <tickets> |
| arch | architecture.md, structure.md | <updated/skipped> | <tickets> |
| quality | conventions.md, testing.md | <updated/skipped> | <tickets> |
| concerns | concerns.md | <updated/skipped> | <tickets> |

Synced ${N_CHANGES} implementation summaries (${FIRST_DATE} to ${LAST_DATE}).
Previous sync: ${LAST_SYNCED} | Current sync: ${DATE}
```
</step>

---

## Update Rules

### What Gets Updated

| Design Doc | Trigger | Update Type |
|------------|---------|-------------|
| `stack.md` | New dependency in "Dependencies & Side Effects" | Add row to Key Dependencies table |
| `integrations.md` | New external service or config var | Add row to relevant section |
| `architecture.md` | New handler/service/layer pattern | Add to Layers or Entry Points |
| `structure.md` | New files created | Add to Key File Locations, update Directory Layout |
| `conventions.md` | New pattern in "Key Design Decisions" | Add to relevant convention section |
| `testing.md` | New test type or approach | Add to Test Types section |
| `concerns.md` | Bugfix resolves known issue | Mark as resolved. New workaround → add as new concern |

### What Never Gets Updated

- Architecture prose (Pattern Overview, Key Characteristics) — requires human judgment
- Design decision rationales — only humans know the "why"
- Existing code examples — unless explicitly superseded
- Section ordering — never reorder
- Removed items — syncer only adds, never removes

### Source Tracing

Every addition by the syncer includes the ticket key for traceability:

```markdown
## Key Dependencies

| Package | Version | Why it matters |
|---------|---------|---------------|
| pg | 8.x | Primary PostgreSQL client |
| cursor-pagination | 1.2.0 | Keyset pagination [PROJ-42] |
```

This lets humans know which changes came from automated sync vs. manual documentation.

---

## Failure Handling

| Condition | Action |
|-----------|--------|
| `docs/design/` missing | Abort: "Run `/autoflow-docs-init` first" |
| `docs/changes/` empty | Abort: "No implementation summaries found" |
| Zero unsynced changes | Report "Design docs are up to date" and exit |
| Agent fails | Log error, proceed with successful agents, note in report |
| Agent returns SYNC SKIP | Normal — not every change affects every focus area |
| Design doc missing for focus | Agent returns SYNC SKIP with guidance to run docs-init |
| `--dry-run` | Show unsynced summary only, no agents dispatched |
