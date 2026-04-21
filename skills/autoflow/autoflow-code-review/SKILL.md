---
name: autoflow-code-review
description: "Standalone code review with optional auto-fix. Reviews changed files for bugs, security, and quality using 3 depth modes. Optionally applies fixes atomically. Triggers on: '/autoflow-code-review', 'review my code', 'review changes', 'review and fix'."
argument-hint: "[--depth quick|standard|deep] [--fix] [--base <ref>] [--files file1 ...]"
metadata:
  context: fork
  model: opus
---

# Code Review

$ARGUMENTS

---

## Modes

| Trigger | Behavior |
|---------|----------|
| `/autoflow-code-review` | Review changed files (standard depth, no fix) |
| `/autoflow-code-review --fix` | Review + auto-fix Critical and Warning findings |
| `/autoflow-code-review --depth deep` | Deep cross-file analysis |
| `/autoflow-code-review --depth quick` | Fast pattern-matching scan |
| `/autoflow-code-review --fix --scope all` | Fix all findings including Info |
| `/autoflow-code-review --base origin/uat` | Diff against specific ref |
| `/autoflow-code-review --files src/a.ts src/b.ts` | Review specific files |

---

## Defaults

| Setting | Default | Override |
|---------|---------|----------|
| Depth | `standard` | `--depth quick\|deep` |
| Diff base | configured `base_branch` from `.autoflow/bootstrap.json`, else `origin/uat`, else `HEAD~5` | `--base <ref>` |
| Fix | off | `--fix` |
| Fix scope | `critical_warning` | `--scope all` |
| Output dir | `.autoflow/reviews/` | `--output <path>` |

---

## Workflow

```
1. Determine scope (files to review)         [DETERMINISTIC]
2. Spawn autoflow-code-reviewer              [AGENTIC]
3. Present REVIEW-code.md to user             [DETERMINISTIC]
4. If --fix: spawn autoflow-code-fixer       [AGENTIC]
5. Present REVIEW-FIX.md to user             [DETERMINISTIC]
```

---

## Step 1: Determine Scope

### Files from arguments

If `--files` provided, use those directly.

### Files from git diff

```bash
# Detect base branch from autoflow config, then fall back to well-known remotes
CONFIG_FILE=".autoflow/bootstrap.json"
if [ -f "$CONFIG_FILE" ]; then
    CONFIGURED_BRANCH=$(cat "$CONFIG_FILE" | grep -o '"base_branch"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"base_branch"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
fi

if [ -n "$CONFIGURED_BRANCH" ] && git rev-parse --verify "origin/${CONFIGURED_BRANCH}" &>/dev/null; then
    BASE="origin/${CONFIGURED_BRANCH}"
elif git rev-parse --verify origin/uat &>/dev/null; then
    BASE="origin/uat"
elif git rev-parse --verify origin/main &>/dev/null; then
    BASE="origin/main"
else
    BASE="HEAD~5"
fi

# Override with --base if provided
# Then get changed files
git diff --name-only ${BASE}..HEAD -- . ':!.planning/' ':!.autoflow/' ':!package-lock.json' ':!yarn.lock'
```

If no files changed, report "No changes to review" and exit.

### Output directory

```bash
mkdir -p .autoflow/reviews
TIMESTAMP=$(date -u +%Y%m%d-%H%M%S)
REVIEW_PATH=".autoflow/reviews/REVIEW-code-${TIMESTAMP}.md"
FIX_PATH=".autoflow/reviews/REVIEW-FIX-${TIMESTAMP}.md"
```

---

## Step 2: Review

Dispatch to `autoflow-code-reviewer`:

```
Agent(
  subagent_type="autoflow-code-reviewer",
  description="Code review: ${DEPTH}",
  prompt="
<config>
depth: ${DEPTH}
output_path: ${REVIEW_PATH}
mode: standalone
files:
${FILES_YAML}
</config>

Follow your role definition. Write REVIEW-code.md to output_path. Do not commit.
"
)
```

Parse return line `## REVIEW COMPLETE: <path>`. Read REVIEW-code.md.

### Present to user

Show the summary:
- File count, finding counts by severity
- If `status: clean` → "No issues found."
- If `status: issues_found` → show the findings, ask "Fix these? (Critical + Warning)"

If `--fix` was NOT specified and issues were found:
- Ask user: "Run `/autoflow-code-review --fix` to auto-fix Critical and Warning issues."
- Stop here.

---

## Step 3: Fix (only with --fix)

Dispatch to `autoflow-code-fixer`:

```
Agent(
  subagent_type="autoflow-code-fixer",
  description="Fix review findings",
  prompt="
<config>
review_paths:
  - ${REVIEW_PATH}
output_path: ${FIX_PATH}
fix_scope: ${FIX_SCOPE}
</config>

Follow your role definition. Fix findings from REVIEW-code.md. Commit each fix atomically. Write REVIEW-FIX.md to output_path.
"
)
```

Parse return line `## FIX COMPLETE: <path>`.

### Present to user

Show fix summary:
- N fixed, N skipped
- For skipped: show reasons
- If any skipped: "Review skipped items manually or re-run with adjusted scope."

---

## Examples

```bash
# Quick scan of current changes
/autoflow-code-review --depth quick

# Standard review against uat
/autoflow-code-review

# Deep review with auto-fix
/autoflow-code-review --depth deep --fix

# Review specific files
/autoflow-code-review --files src/services/auth.ts src/middleware/rate-limiter.ts

# Review and fix everything including style issues
/autoflow-code-review --fix --scope all
```
