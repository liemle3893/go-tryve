---
name: autoflow-deep-review
description: This skill should be used when performing an ultra-comprehensive code review covering security, performance, maintainability, error handling, test coverage, API design, documentation, and architecture. Triggers when the user asks for a deep review, thorough code audit, or comprehensive security/quality analysis of a codebase or directory.
argument-hint: "[<path>]"
---

# Autoflow Deep Review

Perform an ultra-comprehensive code review across 8 dimensions: security, performance, maintainability, error handling, test coverage, API design, documentation, and architecture.

## Phase 1: Reconnaissance

Read the detailed checklists from `references/review-checklists.md` in this skill directory.

Then identify the target path. Use the path provided by the user, or default to the current working directory.

Read the main source files, dependency manifests (go.mod, package.json, requirements.txt, etc.), and any CI/CD configuration to understand the project structure.

## Phase 2: Launch Review Agents in Parallel

Use the Agent tool to launch four review agents concurrently in a single message. Pass each agent the target path and the relevant checklist sections.

### Agent 1: Security Review

Review the codebase against the **Security** checklist (Section 1 from `references/review-checklists.md`). Scan all source files, configuration, and dependency manifests.

### Agent 2: Performance & Efficiency Review

Review the codebase against the **Performance** checklist (Section 2). Focus on hot paths, database queries, async patterns, and resource management.

### Agent 3: Code Quality & Maintainability Review

Review the codebase against the **Maintainability & Code Quality**, **Error Handling**, and **API Design** checklists (Sections 3, 4, and 6). Focus on code structure, naming, error patterns, and interface design.

### Agent 4: Test Coverage, Documentation & Architecture Review

Review the codebase against the **Test Coverage**, **Documentation**, and **Architectural Concerns** checklists (Sections 5, 7, and 8). Focus on test gaps, doc quality, and structural issues.

## Phase 3: Aggregate and Report

Wait for all four agents to complete. Aggregate all findings into a single report.

### Output Format

For every finding, provide:
- **Category** (from the 8 dimensions)
- **Severity**: Critical / High / Medium / Low / Informational
- **File and line number** (if applicable)
- **Description** of the issue
- **Impact**: what can go wrong
- **Recommended fix** with a code snippet where helpful

Group findings by severity (Critical first). Conclude with a prioritised action plan listing the top issues to address.
