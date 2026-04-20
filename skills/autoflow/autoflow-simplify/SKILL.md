---
name: autoflow-simplify
description: This skill should be used when reviewing changed code for reuse opportunities, quality issues, and efficiency problems. Triggers when the user asks to simplify, clean up, or review recent code changes. Launches three parallel review agents (reuse, quality, efficiency) and fixes all findings.
argument-hint: "[--base <ref>] [--files file1 ...]"
---

# Autoflow Simplify

Review all changed files for reuse, quality, and efficiency. Fix any issues found.

## Phase 1: Identify Changes

Run `git diff` (or `git diff HEAD` if there are staged changes) to capture the full diff.
If there are no git changes, review the most recently modified files that were
mentioned or edited earlier in the conversation.

## Phase 2: Launch Three Review Agents in Parallel

Use the Agent tool to launch all three agents concurrently in a single message.
Pass each agent the full diff so it has complete context.

### Agent 1: Code Reuse Review

For each change:
1. **Search for existing utilities and helpers** that could replace newly written code.
2. **Flag any new function that duplicates existing functionality.**
3. **Flag any inline logic that could use an existing utility** — hand-rolled string
   manipulation, manual path handling, custom environment checks, etc.

### Agent 2: Code Quality Review

Review the same changes for hacky patterns:
1. **Redundant state** that duplicates existing state.
2. **Parameter sprawl** — new parameters instead of restructuring.
3. **Copy-paste with slight variation** that should be unified.
4. **Leaky abstractions** — exposing internal details.
5. **Stringly-typed code** where constants or enums already exist.
6. **Unnecessary comments** narrating what code does (not why).

### Agent 3: Efficiency Review

Review the same changes for efficiency:
1. **Unnecessary work** — redundant computations, duplicate reads.
2. **Missed concurrency** — independent operations run sequentially.
3. **Hot-path bloat** — blocking work added to startup or per-request paths.
4. **Recurring no-op updates** — unconditional updates in polling loops.
5. **Memory** — unbounded data structures, missing cleanup.

## Phase 3: Fix Issues

Wait for all three agents to complete. Aggregate findings and fix each issue.
If a finding is a false positive, note it and move on.

When done, briefly summarize what was fixed (or confirm the code was already clean).
