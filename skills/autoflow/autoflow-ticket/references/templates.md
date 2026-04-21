# Ticket Templates

Output format for enriched tickets. PO never sees or fills these templates directly.

## Length Rules (HARD)

1. **Delete any section that would say "None", "N/A", or restate a global rule from CLAUDE.md.** No empty headers.
2. **Word budgets:** Story ≤ 500 words. Subtask ≤ 300 words. Bug ≤ 400 words. Spike ≤ 200 words. If over budget, cut — don't compress.
3. **Conditional sections** (include ONLY when the condition holds):
   - `Failure Modes` — only if it adds HTTP status / error code info beyond what AC already says. If every row is "200 / null", kill the table.
   - `Rollback Strategy` — only if the ticket modifies schema, state, or API contracts in a non-trivial way. "Reverse the migration" is not a rollback strategy; skip the section.
   - `Cross-Cutting Concerns` — only if Moderate/Complex AND there is something non-obvious to say. "No new env vars, no new metrics" = delete.
   - `Test Strategy` — only if mocking boundaries or test data are non-obvious. Don't restate the CLAUDE.md coverage target.
   - `AI Complexity Estimate` — keep the "tricky areas" bullet if real; drop the level/file-count line when it's just "Moderate, ~5 files".
   - `Constraints` — list ONLY ticket-specific requirements. Global rules (CREATE INDEX style, no deprecation shims, logging format) belong in CLAUDE.md, not here.
   - `Open Items for Planning` — delete once the ticket leaves planning. Never appears on a Ready-For-QC or In-Progress ticket.
4. **AC and Failure Modes must not duplicate each other.** If a failure is already stated as an AC bullet, don't repeat it in the table.

## Feature Task (Parent)

```
Title: [Feature] <verb> <noun>

## Original Requirement
<PO's original description, verbatim>

---

## Context
2-3 sentences: why this matters NOW. Link to source (epic, incident, business request).

## Scope
- Bullet list of what IS included

## Out of Scope
- Bullet list of what is explicitly EXCLUDED

## Behavioral AC
- [ ] <observable outcome 1 — inputs and outputs with concrete values>
- [ ] <observable outcome 2>
- [ ] <unhappy path — "When X fails, system returns Y with status Z">
- [ ] <unhappy path — "When input is invalid, system returns Z">
- [ ] <backward compatibility statement if applicable>
(Max 7. More = split.)

## Failure Modes
| Trigger | Expected Behavior | HTTP Status | Error Code |
|---------|------------------|-------------|------------|
| ... | ... | ... | ... |
(Required for API endpoints — minimum 3 rows: input validation + auth + downstream.)

## Constraints
- [ ] <implementation requirements SPECIFIC to this ticket>
(Max 5.)

## Security
Classification: <Critical / Relevant / Neutral>
(If Critical/Relevant, add What's at Risk + Required Checks.)

## Test Strategy
Primary test layer: <E2E / Integration / Unit>
Test data requirements: <what needs to be seeded>
Mocking boundaries: <what to mock, what to keep real>

## AI Complexity Estimate
Level: <Routine / Moderate / Complex>
Files affected: <count>
Known tricky areas: <implementation-specific warnings>

## Cross-Cutting Concerns
(Required for Moderate/Complex.)
- Logging: <what to log, what NOT to log>
- Config: <new env vars or feature flags>
- Observability: <new metrics or alerts>

## Rollback Strategy
(Required if ticket modifies state/schema/API contracts.)
- Feature flag: <Yes/No, flag name>
- Data rollback: <reversible? procedure?>

## Key Files
- `path/to/file` — brief reason

## Plan Files
{code:title=<filename>|collapse=true}
<full file contents>
{code}

## Dependencies
Jira links (blocked by / blocks / related to).
```

## BE Subtask

```
Title: [BE] <verb> <noun>

## Context
1-2 sentences linking to parent. What this subtask delivers independently.

## Behavioral AC
- [ ] <API contract: method + route + request/response shape + status codes>
- [ ] <service logic: input → processing → output>
- [ ] <unhappy path: input validation failure — specific error code + message>
- [ ] <unhappy path: downstream failure — timeout/unavailable handling>
- [ ] <backward compatibility: existing callers unaffected>
(Max 7.)

## Failure Modes
| Trigger | Expected Behavior | HTTP Status | Error Code |
|---------|------------------|-------------|------------|
| ... | ... | ... | ... |

## Security
Classification: <Critical / Relevant / Neutral>

## Constraints
- [ ] <migration details, index strategy, scale notes>

## Key Files
- `path/to/file` — brief reason

## Rollback (if migration involved)
How to reverse safely.
```

## FE Subtask

```
Title: [FE] <verb> <noun>

## Context
1-2 sentences linking to parent. What this subtask delivers independently.

## Behavioral AC
- [ ] <UI element: what exists, where, what it does>
- [ ] <states: loading, empty, populated, error, disabled>
- [ ] <validation: inline errors, constraints — specific messages>
- [ ] <unhappy path: server error handling — what the user sees>
- [ ] <unhappy path: network failure — offline/timeout behavior>
(Max 7.)

## Security
Classification: <Critical / Relevant / Neutral>

## Constraints
- [ ] <API integration, accessibility>

## Key Files
- `path/to/component` — brief reason
```

## Bug Report

```
Title: [Bug] <what is broken> — <observable impact>

## Original Report
<reporter's original description, verbatim>

---

Severity: S1 / S2 / S3 / S4
Priority: P1 / P2 / P3

## Reproduction
1. <exact API call or UI action>
2. Observe: <exact error>
3. Expected: <what should happen>

## Root Cause
Status: Hypothesis | Confirmed
<analysis of codebase + affected files>

## Key Files
- `path/to/file:L##` — <why relevant>

## Regression Test
Test should cover: <specific scenario>

## Follow-ups (separate tickets)
<List any non-code work the fix does NOT address but ops/data needs to handle separately. ALWAYS include this section for bugs that involve any of the triggers below — leave it empty (not omitted) only after explicit consideration.>

- <e.g. "Data remediation: reverse inflated counters granted before the fix landed (mission X, env Y)" — file separately and link as `relates to`>
- <e.g. "Customer comms: notify users impacted by the off-by-one charge">
- <e.g. "Monitoring: add Grafana alert for the metric that would have caught this earlier">
```

**When to require a Follow-ups section (always, for these bug types):**
- Data corruption / inflated counters / duplicated rows / off-by-one accounting
- Anything where users were already credited / charged / notified incorrectly
- Security / auth bugs where exposure may have already occurred
- Schema or migration bugs that left orphaned rows in production
- Bugs whose fix prevents *new* occurrences but does not undo *historical* ones

If the bug only causes UI glitches or inert errors (no state changes leaked), the section may be left empty but must still appear so reviewers know it was considered.

## Spike / Research

```
Title: [Spike] <question to answer>

## Question
One clear question.

## Timebox
Maximum: <N hours/days>. Hard stop.

## Output
Deliverable: <decision doc, prototype, benchmark>

## Current Assumptions
- ...

## No AC — spikes produce knowledge, not features.
```

## Tech Debt

```
Title: [TechDebt] <what and why>

## Context
What pain exists TODAY.

## Risk of Inaction
What gets worse if ignored.

## Behavioral AC
- [ ] All existing tests pass without modification
- [ ] No behavior change from caller's perspective
- [ ] <specific measurable improvement>

## Key Files
- `path/to/file` — brief reason
```
