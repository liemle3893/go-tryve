# Jira Transition Discovery

Reference loaded on-demand by `autoflow-deliver/SKILL.md`.

## Protocol

Transition IDs are instance-specific and must never be hardcoded. Always discover at runtime:

1. Call `mcp__atlassian__getTransitionsForJiraIssue` with the ticket key.
2. Match the desired transition by **name** (case-insensitive), not by ID.
3. If no match found, log all available transition names and escalate to user.

## Transition Table

| From | Transition Name | To Status |
|------|----------------|-----------|
| To Do | start dev | In Development |
| In Development | Dev Done | In Code Review |
| QC | QC Done, Start UAT | Ready for UAT |
| Any | Done | Done |

## Usage in autoflow-deliver

### Step 2 — Start Development (Jira mode only)

After worktree creation, transition from "To Do" → "In Development":

```
mcp__atlassian__getTransitionsForJiraIssue → issueIdOrKey: "<KEY>"
# Find transition matching "start dev" (case-insensitive)
mcp__atlassian__transitionJiraIssue → transition.id: "<discovered_id>"
```

**Rules:**
- Only transition if current status is "To Do" (from task-brief.md STATUS field)
- Skip if already "In Development" or later (resumed workflow)

### Step 13 — Dev Done (Jira mode only)

After all gates pass and PR is ready, transition from "In Development" → "In Code Review":

```
mcp__atlassian__getTransitionsForJiraIssue → issueIdOrKey: "<KEY>"
# Find transition matching "Dev Done" (case-insensitive)
mcp__atlassian__transitionJiraIssue → transition.id: "<discovered_id>"
```

**Rules:**
- Ticket should already be in "In Development" (set by Step 2)
- If enhance loop escalated (Step 9 failed), SKIP this transition — ticket stays in "In Development"
