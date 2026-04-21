---
name: autoflow-deliver
description: "Deliver a task end-to-end — from implementation through PR creation. Supports two modes: (1) Jira mode with a ticket key (e.g., 'PROJ-36'), (2) Local mode with --local flag and inline task description."
argument-hint: "<TICKET-KEY> | --local <plan-file | description> [--base <branch>]"
---

# Task to PR (v5 — Code Orchestrator)

Run the deterministic Python orchestrator:

```bash
autoflow-deliver $ARGUMENTS
```

The orchestrator detects the target repo via `git rev-parse --show-toplevel` and finds all config (`.autoflow/bootstrap.json`), scripts (`.claude/scripts/`), and state (`.planning/`) from there.

## Prerequisites

Install once from the winx-autoflow repo:
```bash
pip install -e /path/to/winx-autoflow/harness/autoflow-deliver
```

## Exit Codes

- **0** — Workflow completed successfully. PR created.
- **1** — Escalation needed. Present the error message to the user and ask how to proceed.
- **2** — Step failure. Present the error and ask the user to fix manually or retry.
