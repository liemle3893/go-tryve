---
name: autoflow-local-merge
description: "Merge, undo, or redo autoflow-deliver branches on your local branch for testing. Supports two sources: (1) PR mode — merge a GitHub PR locally, (2) Worktree mode — merge from a local autoflow worktree branch. Triggers on: 'local-merge PR 42', 'merge worktree locally', 'undo merge', 'redo', 'list deliveries', or any variation of testing delivered changes on the local branch."
argument-hint: "<PR-number | worktree-path | branch> | list | undo | redo"
---

# Autoflow Apply

$ARGUMENTS

Apply, undo, and redo autoflow-deliver changes on your local working branch — without pushing.

## When to Use

- User asks to **apply/merge** a delivered PR onto their local branch (e.g., "apply PR 42", "merge PR 100 locally")
- User asks to **apply** changes from a delivery worktree (e.g., "apply worktree ../repo-proj-42", "apply feat/add-pagination")
- User asks to **list** available deliveries (worktrees + PRs)
- User asks to **undo/revert** a previously applied delivery (e.g., "undo that", "revert last apply")
- User asks to **redo** a delivery that was undone (e.g., "redo", "redo PR 42")
- User asks to **reapply** a fresh version of a currently-applied delivery (e.g., "reapply PR 42" — the branch may have new commits)

## Important Rules

1. **Never push** — all operations are local only.
2. **Never checkout other branches** — the delivery worktree is a separate git worktree; checking out its branch would detach the worktree from it. Use `git merge <branch>` (for local worktree branches) or `git fetch` + `git merge FETCH_HEAD` (for remote PR branches) instead.
3. **Always use `gh` CLI for PR metadata** — pipe output through `cat` to avoid pager issues.
4. **Track applied deliveries** in session memory as a stack so undo/redo works without re-specifying the source.
5. **Project-agnostic** — never hardcode org names, repo names, or domain-specific conventions. Discover repo info at runtime via `gh repo view --json nameWithOwner -q .nameWithOwner | cat`.

## Operation: `list`

Show available autoflow deliveries from both worktrees and PRs.

### List Worktrees

```bash
# List all git worktrees — filter for autoflow delivery patterns
git worktree list --porcelain
```

**Filter criteria:** Worktrees created by autoflow-deliver follow the naming convention:
- Path: `../<repo>-<ticket-key>` (e.g., `../my-api-proj-42`)
- Branch: `jira-iss/<key>-<slug>` or `feat/<key>-<slug>`

For each matching worktree, check if a `workflow-progress.json` exists to show delivery status:

```bash
# For each worktree path, check for progress file
# The ticket key is derived from the branch name
TICKET_KEY=$(echo "<branch>" | sed -E 's#^(jira-iss|feat)/##' | cut -d'-' -f1-2 | tr '[:lower:]' '[:upper:]')
PROGRESS_FILE=".planning/ticket/${TICKET_KEY}/workflow-progress.json"

if [ -f "$PROGRESS_FILE" ]; then
    STEP=$(jq -r '.current_step // "?"' "$PROGRESS_FILE")
    PR_URL=$(jq -r '.pr_url // "none"' "$PROGRESS_FILE")
    echo "  Step: $STEP | PR: $PR_URL"
fi
```

### List PRs

```bash
# List open PRs from autoflow branches
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner | cat)
gh pr list --json number,title,headRefName,state \
    --jq '.[] | select(.headRefName | test("^(jira-iss|feat)/"))' \
    --repo "$REPO" | cat
```

### Display Format

```
Available Deliveries:

Worktrees:
  1. ../my-api-proj-42  [branch: feat/proj-42-add-pagination]  Step: 11/13  PR: #85
  2. ../my-api-fix-login [branch: jira-iss/fix-login-bug]      Step: 7/13   PR: none

PRs (no local worktree):
  3. PR #90  feat/add-caching  "feat(cache): add Redis caching layer"
```

---

## Operation: `apply`

### Parse Arguments

Parse `$ARGUMENTS` to determine the source:

| Input | Source Type | Example |
|-------|-----------|---------|
| `apply <number>` | PR mode | `apply 42`, `apply PR 85` |
| `apply <path>` (directory exists) | Worktree mode (by path) | `apply ../repo-proj-42` |
| `apply <branch-name>` | Worktree mode (by branch) | `apply feat/proj-42-pagination` |
| `apply` (bare) | Auto-detect | Show `list`, let user pick |

### PR Mode (remote branch)

When the source is a PR number:

```bash
# 1. Get PR metadata
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner | cat)
gh pr view <PR_NUMBER> --json headRefName,title,state,baseRefName --repo "$REPO" | cat

# 2. Fetch the branch
git fetch origin <headRefName>

# 3. Merge into current local branch (no push)
git merge FETCH_HEAD --no-edit
```

### Worktree Mode (local branch)

When the source is a worktree path or branch name:

**By path:**
```bash
# Get branch name from worktree
BRANCH=$(git -C <worktree-path> branch --show-current)

# Merge directly — worktree branches are local refs, no fetch needed
git merge "$BRANCH" --no-edit
```

**By branch:**
```bash
# Verify the branch exists locally
git branch --list "<branch-name>"

# Merge directly
git merge <branch-name> --no-edit
```

### Pre-checks (both modes)

Before applying:

1. **Clean working tree** — Run `git status --short`. If dirty, warn the user and ask whether to proceed or stash first.
2. **Base branch match** — For PR mode, check PR's `baseRefName` matches the current branch. For worktree mode, check the worktree branch was created from the current branch (via `git merge-base`). If mismatched, warn the user but proceed if they confirm.
3. **Already applied** — Check the session stack to see if this delivery was already applied. Warn if so.

### After applying

- Show the merge result summary (files changed count, insertions/deletions).
- Push the source identifier onto the session stack:
  ```
  { type: "pr" | "worktree", ref: "<PR-number>" | "<branch>", title: "...", merge_commit: "<SHA>" }
  ```
- Display a short summary:
  ```
  Applied: feat/proj-42-add-pagination (PR #85)
  Files changed: 12 (+340, -45)
  Merge commit: abc1234

  Use "undo" to revert.
  ```

---

## Operation: `undo`

Revert the last applied delivery by resetting the merge commit.

```bash
# Reset the merge commit (last entry in the stack)
git reset --hard HEAD~1
```

### Pre-checks

1. **Stack not empty** — If no deliveries have been applied in this session, inform the user.
2. **HEAD matches expected merge** — Verify `HEAD` SHA matches the `merge_commit` recorded in the stack. If additional commits were made after the apply, warn the user that undo will only remove the last commit.
3. **Confirm with user** — `git reset --hard` is destructive. Show what will be undone and ask for confirmation before proceeding.

### After undoing

- Pop the entry from the applied stack; push it onto the undo stack (for redo).
- Show `git log --oneline -3` to confirm state.
- Display:
  ```
  Undone: feat/proj-42-add-pagination (PR #85)
  HEAD is now at: <prev-SHA> <prev-message>

  Use "redo" to reapply.
  ```

---

## Operation: `redo`

Reapply the last undone delivery.

### PR source

```bash
# Re-fetch (in case of force-push) and re-merge
git fetch origin <headRefName>
git merge FETCH_HEAD --no-edit
```

### Worktree source

```bash
# Merge directly (local branch, no fetch needed)
git merge <branch> --no-edit
```

### After redoing

- Pop from undo stack, push back onto applied stack with the new merge commit SHA.
- Show merge result summary.

---

## Operation: `reapply`

When the user says "reapply" — they want to undo the current merge and apply a fresh version (the branch may have been updated with new commits). This is NOT the same as `redo` — `reapply` requires the delivery to be **currently applied**.

### Pre-checks (mandatory)

1. **Delivery must be in the applied stack.** If not present, inform the user: "This delivery is not currently applied. Use `apply` instead."
2. **HEAD SHA must match the `merge_commit`** recorded for this delivery. If extra commits were made after the apply, warn the user that `reset --hard HEAD~1` will destroy those commits — suggest manual `git revert` instead.
3. **Confirm with user** before proceeding (destructive operation).

### PR source

```bash
git reset --hard HEAD~1
git fetch origin <headRefName>
git merge FETCH_HEAD --no-edit
```

### Worktree source

```bash
git reset --hard HEAD~1
git merge <branch> --no-edit
```

---

## Handling Multiple Deliveries

Track applied deliveries as a stack (LIFO):

- **`apply`** pushes onto the stack.
- **`undo`** pops the last entry (or a named entry if specified).
- **`undo <ref>`** — if the specified delivery is not the most recent, warn the user that intermediate deliveries will also be undone, or suggest using `git revert <merge-commit> -m 1` for surgical removal.
- **`redo`** re-applies the last undone delivery.

### Stack State (session memory)

```
applied_stack: [
  { type: "worktree", ref: "feat/proj-42-pagination", title: "Add pagination", merge_commit: "abc1234" },
  { type: "pr", ref: "85", title: "feat(cache): add caching", merge_commit: "def5678" }
]
undo_stack: []
```

---

## Merge Conflicts

If `git merge` fails with conflicts:

1. Show the conflicting files.
2. Ask the user if they want to:
   - **Resolve manually** — leave conflicts in place for the user to fix.
   - **Abort** — run `git merge --abort` to return to pre-merge state.
3. Do NOT auto-resolve conflicts.

---

## Example Session

```
User: "list"
→ Show worktrees + PRs from autoflow-deliver

User: "apply ../my-api-proj-42"
→ Get branch from worktree, merge locally, show summary
   Stack: [{ type: "worktree", ref: "feat/proj-42-pagination", ... }]

User: "apply PR 85"
→ Fetch PR branch, merge locally, show summary
   Stack: [{ worktree... }, { type: "pr", ref: "85", ... }]

User: "undo"
→ Reset HEAD~1, remove PR #85 from stack
   Stack: [{ worktree... }]
   Undo stack: [{ pr... }]

User: "redo"
→ Re-fetch + merge PR #85
   Stack: [{ worktree... }, { pr... }]

User: "reapply PR 85"
→ Verify PR #85 is at HEAD, reset HEAD~1, re-fetch + merge fresh
   (picks up new commits pushed to the PR branch)
```
