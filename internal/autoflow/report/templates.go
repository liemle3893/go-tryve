package report

// prBodyTmpl is the template for PR-BODY.md — concise, ready to paste
// into a GitHub pull-request description. Minimises loop noise; agents
// or humans will fill in anything specific in the PR UI later.
const prBodyTmpl = `## Summary
- {{.OneLiner}}

## Changes

{{renderChanges .Changes}}
## Test Plan
- [x] E2E tests: {{len .E2E.Tests}} tests — {{.E2E.Passed}} passed{{if .E2E.Duration}} ({{.E2E.Duration}}){{end}}
{{- if gt .E2E.Failed 0}}
- [ ] E2E failures: {{.E2E.Failed}} failed
{{- end}}

## E2E Results

{{renderE2E .E2E false}}
`

// jiraCommentTmpl is the template for JIRA-COMMENT.md — per-loop summary
// tables plus the E2E detail table. Structured enough for a PM/QA reader.
const jiraCommentTmpl = `# Delivery Report: {{.Ticket}}

**PR:** #{{.PRNumber}} — ` + "`" + `{{.PRTitle}}` + "`" + `
**Branch:** ` + "`" + `{{.Branch}}` + "`" + `
**Status:** {{.OverallStatus}}

## Changes

{{renderChanges .Changes}}
## E2E Test Results

{{renderE2E .E2E true}}
## Loop Detail

{{renderLoopTable .CoverageFile "AC Coverage"}}
{{renderLoopTable .E2EFixFile "E2E Fix"}}
{{renderLoopTable .ReviewFile "Enhance Loop"}}
## Usage

{{.UsageNotes}}
`

// executionReportTmpl is the full artefact — written to disk, uploaded
// to the Jira ticket as the final attachment. Round-by-round detail so
// historians can reconstruct what happened.
const executionReportTmpl = `# Execution Report: {{.Ticket}}

**PR:** #{{.PRNumber}} — ` + "`" + `{{.PRTitle}}` + "`" + `
**Branch:** ` + "`" + `{{.Branch}}` + "`" + `
**Status:** {{.OverallStatus}}
**Total iterations:** {{.TotalIterations}} (coverage: {{.CoverageRounds}}, e2e-fix: {{.E2ERounds}}, enhance-loop: {{.EnhanceRounds}})

## Changes

{{renderChanges .Changes}}
## E2E Test Results

{{renderE2E .E2E true}}
## Loop Detail (Full)

{{renderLoopDetail .CoverageFile "AC Coverage"}}
{{renderLoopDetail .E2EFixFile "E2E Fix"}}
{{renderLoopDetail .ReviewFile "Enhance Loop"}}

## Usage

{{.UsageNotes}}
`
