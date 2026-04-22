---
name: autoflow-deliver
description: "Deliver a Jira ticket end-to-end — from implementation through PR creation. Triggers on: 'deliver/ship/implement PROJ-XX'. Controller-driven: the LLM only calls `autoflow deliver next/complete` and executes what it returns."
argument-hint: "<TICKET-KEY>"
allowed-tools:
  - Bash
  - Agent
  - ToolSearch
  - TodoWrite
  - TodoRead
  - TaskCreate
  - TaskUpdate
---

$ARGUMENTS

## RULES

- You do NOT read code, write code, explore, or decide anything.
- You do NOT decide what to do — only call `autoflow deliver next/complete` and execute what it returns.
- You do NOT invoke any other skill during this workflow.
- You parse ONLY the `##` status line from subagent returns — ignore everything else.

---

## EXECUTE THESE STEPS IN ORDER

### 1. Parse the ticket key and seed todo list

Extract the Jira ticket key (e.g., `WINX-169`) from `$ARGUMENTS` above.

Then call TodoWrite with these exact 13 steps:

```
TodoWrite(todos=[
  { content: "Step 1 — Fetch Jira + build task brief",            status: "pending", activeForm: "Fetching Jira ticket" },
  { content: "Step 2 — Create worktree + bootstrap + transition",  status: "pending", activeForm: "Creating worktree" },
  { content: "Step 3 — Write E2E test definitions from ACs",       status: "pending", activeForm: "Writing E2E tests" },
  { content: "Step 4 — AC coverage review loop (max 3)",           status: "pending", activeForm: "Reviewing AC coverage" },
  { content: "Step 5 — Implement (direct or plan+execute)",        status: "pending", activeForm: "Implementing" },
  { content: "Step 6 — Build + test gate",                         status: "pending", activeForm: "Running build gate" },
  { content: "Step 7 — Run AC E2E (max 5)",                        status: "pending", activeForm: "Running E2E tests" },
  { content: "Step 8 — Add coverage tests from impl diff",         status: "pending", activeForm: "Adding coverage tests" },
  { content: "Step 9 — Review + fix",                              status: "pending", activeForm: "Reviewing and fixing" },
  { content: "Step 10 — Write implementation summary",             status: "pending", activeForm: "Writing summary" },
  { content: "Step 11 — Create PR",                                status: "pending", activeForm: "Creating PR" },
  { content: "Step 12 — Generate delivery reports",                status: "pending", activeForm: "Generating reports" },
  { content: "Step 13 — Jira update + upload artifacts",           status: "pending", activeForm: "Updating Jira" },
])
```

### 2. Run the step controller

**Run this EXACT command via the Bash tool (replace `<KEY>` with the ticket key):**

```bash
autoflow deliver next --ticket <KEY>
```

This prints a JSON instruction. Go to step 3.

**Do NOT read any files, check any state, or explore anything before running this command.**

### 3. Execute the returned instruction

Read the `"action"` field from the JSON and do EXACTLY what it says:

**If `"action": "bash"`** — join `instruction.commands` with ` && ` and run via Bash tool.

**If `"action": "dispatch"`** — call the Agent tool:
```
Agent(subagent_type=instruction.subagent_type, description=instruction.description, prompt=instruction.prompt)
```

**If `"action": "dispatch_parallel"`** — call the Agent tool for EVERY item in `instruction.dispatches` in a **single message** (multiple tool uses, so they run in parallel). Wait for all to complete. Then go to step 4.

**If `"action": "auto_complete"`** — nothing to execute. Go directly to step 4.

**If `"action": "escalate"`** — show `instruction.reason` to the user and wait for their input. If they say skip, go to step 4. If they say retry, go to step 2.

**If `"action": "done"`** — mark all todos completed. Print `instruction.summary`. Stop.

### 4. Complete and get next instruction

**Check:** does the instruction JSON contain `"loop": true`?

- **YES** → Skip complete. But first check two things:
  1. If the instruction has `"on_failure": "escalate"` AND the bash command crashed with an unexpected error (not a normal test failure), escalate to user instead of looping.
  2. If the subagent returned `## FIX FAILED` AND the instruction has `"on_fix_failed_marker"`, write the marker file first: `echo "failed" > <marker-path>`. Then run `next` — the controller will escalate.
  
  Otherwise, run `autoflow deliver next --ticket <KEY>` immediately. Go to step 3.
- **NO** → Run the complete command:

```bash
autoflow deliver complete --ticket <KEY>
```

If you extracted values (see step 5), add flags: `--title "..."` or `--pr-url "..."`.

Then update TodoWrite: mark current step `completed`, next step `in_progress`.

**Then ALWAYS run `next` again:**

```bash
autoflow deliver next --ticket <KEY>
```

Go to step 3.

**IMPORTANT: After every step, you MUST call `autoflow deliver next --ticket <KEY>`. Never decide what to do yourself.**

### 5. Extracting values and post-actions

**Extract:** If the instruction has an `"extract"` field like `{"title": "TITLE"}`, look for `TITLE=some value` in the bash stdout or the subagent's `##` status line. Pass it to the complete command: `--title "some value"`.

**Post-actions:** If the instruction has `"post_actions"`, execute each one after the main action succeeds:

- `"action": "jira_transition"` → Call `getTransitionsForJiraIssue(issueKey=instruction.ticket)`, find the transition matching `instruction.transition_name` (case-insensitive), then call `transitionJiraIssue(issueKey=instruction.ticket, transitionId=<found-id>)`. If not found, log warning and continue.

---

## RESUME

If the first `next` call returns a `"step"` value > 1, you are resuming. Rebuild TodoWrite with steps before `"step"` as `completed`, `"step"` as `in_progress`, rest as `pending`. Then continue from step 3.
