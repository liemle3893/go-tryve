# Task Brief Template

Fill this template and save to `.planning/ticket/<TICKET-KEY>/task-brief.md`:

```markdown
# Task Brief: <TICKET-KEY>

TICKET: <TICKET-KEY>
TITLE: <verbatim from ticket title>
PARENT: <parent-key> -- <parent title, or "none">
SIBLINGS: <KEY> (status), <KEY> (status), ...
STATUS: <current ticket status>

## Description
<verbatim from ticket description — no rephrasing>

## Acceptance Criteria
1. <verbatim from ticket>
2. <verbatim from ticket>

## Definition of Done
1. <verbatim from ticket>
2. <verbatim from ticket>

## Attachments
- <filename> (<type>) -- <one-line description of what it shows>

## Context from Parent
<1-3 sentences summarizing the parent goal — this is the only interpreted field>

## Sibling Context
<which siblings are done, what patterns they established — interpreted>

## Constraints (from CLAUDE.md)
- <relevant conventions from CLAUDE.md applicable to the changed area>
- <relevant scale concerns if applicable>
```

**Rules:**
- Every AC and DoD item is copied verbatim — no rephrasing, no interpretation
- "Context from Parent" and "Sibling Context" are the only interpreted fields
- Constraints are pulled from CLAUDE.md, not invented
