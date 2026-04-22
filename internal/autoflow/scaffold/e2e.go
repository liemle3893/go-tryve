// Package scaffold generates stub E2E test YAML files for an autoflow
// ticket. Replaces skills/autoflow-deliver/scripts/scaffold-e2e.sh.
package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

// Options controls Generate. All three fields are required.
type Options struct {
	// Root is the repo root where tests/e2e/<area>/ lives.
	Root string
	// Ticket is the Jira ticket key. Validated against the state package
	// regex before use.
	Ticket string
	// Area is the test area sub-directory (e.g. "user-api"). Empty or
	// path-traversal values are rejected.
	Area string
	// Count is how many stub files to create (must be > 0).
	Count int
}

// Result describes one generated (or skipped) stub file.
type Result struct {
	Path    string
	Created bool // false when the target existed already
}

// ErrBadArea is returned when Area contains path separators.
var ErrBadArea = errors.New("area must not contain path separators")

// leadingDigits captures the NNN group at the start of the suffix once
// the TC-<TICKET>- prefix has been stripped. Lives at the start of the
// remaining filename, so anchoring to ^ is sufficient.
var leadingDigits = regexp.MustCompile(`^(\d+)`)

// Generate creates up to opts.Count stub files and returns one Result per
// slot, in generation order.
func Generate(opts Options) ([]Result, error) {
	if err := state.ValidateTicketKey(opts.Ticket); err != nil {
		return nil, err
	}
	if opts.Area == "" || strings.ContainsAny(opts.Area, `/\`) || opts.Area == "." || opts.Area == ".." {
		return nil, fmt.Errorf("%w: %q", ErrBadArea, opts.Area)
	}
	if opts.Count <= 0 {
		return nil, fmt.Errorf("count must be > 0")
	}

	testDir := filepath.Join(opts.Root, "tests", "e2e", opts.Area)
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", testDir, err)
	}

	startNum := nextNumber(testDir, opts.Ticket) + 1

	tmpl := template.Must(template.New("stub").Parse(stubTemplate))

	out := make([]Result, 0, opts.Count)
	for i := 0; i < opts.Count; i++ {
		num := startNum + i
		padded := fmt.Sprintf("%03d", num)
		name := fmt.Sprintf("TC-%s-%s-STUB.test.yaml", opts.Ticket, padded)
		path := filepath.Join(testDir, name)
		if _, err := os.Stat(path); err == nil {
			out = append(out, Result{Path: path, Created: false})
			continue
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return out, err
		}
		err = tmpl.Execute(f, map[string]string{
			"Ticket": opts.Ticket,
			"Area":   opts.Area,
			"Num":    padded,
		})
		_ = f.Close()
		if err != nil {
			return out, fmt.Errorf("render %s: %w", path, err)
		}
		out = append(out, Result{Path: path, Created: true})
	}
	return out, nil
}

// nextNumber scans testDir for files named TC-<TICKET>-NNN* and returns
// the highest NNN found, or 0 when no matching file exists.
func nextNumber(testDir, ticket string) int {
	entries, err := os.ReadDir(testDir)
	if err != nil {
		return 0
	}
	max := 0
	prefix := "TC-" + ticket + "-"
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		suffix := strings.TrimPrefix(e.Name(), prefix)
		m := leadingDigits.FindStringSubmatch(suffix)
		if len(m) != 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max
}

// stubTemplate is the fixed YAML body written for each new stub. Retained
// verbatim from scaffold-e2e.sh so agents already trained on this layout
// continue to produce the same test shape.
const stubTemplate = `# yaml-language-server: $schema=../schemas/e2e-test.schema.json

# TC-{{.Ticket}}-{{.Num}}: <AC description here>
#
# Acceptance Criteria: <paste the AC this test covers>
# Status: STUB — fill in setup, execute, verify sections

name: TC-{{.Ticket}}-{{.Num}}-STUB
description: "<fill: one-line description of what this test verifies>"
priority: P1
tags:
  - {{.Area}}
  - {{.Ticket}}
  - stub
timeout: 60000
retries: 0

variables: {}

setup:
  # ── Generate JWT token ──
  - id: generate_jwt
    adapter: shell
    action: exec
    description: "Generate test JWT"
    command: >-
      INTERNAL_JWT_KEYS=$(cat local.settings.json | jq '.Values.INTERNAL_JWT_KEYS' -r)
      ./scripts/generate-test-jwt.sh --expires 1h --phone 84987654321 --quiet
    timeout: 10000
    capture:
      access_token: "stdout"

  # ── Additional setup steps here ──

execute:
  # ── Main request ──
  - id: main_request
    adapter: http
    action: request
    method: GET
    url: "{{"{{"}}baseUrl{{"}}"}}/api/v1/<endpoint>"
    headers:
      Authorization: "Bearer {{"{{"}}captured.access_token{{"}}"}}"
      x-mobile-version: "1.0.3"
    capture:
      response_body: "body"
      response_status: "status"

verify:
  # ── Assertions ──
  - id: check_status
    assert:
      - actual: "{{"{{"}}captured.response_status{{"}}"}}"
        expected: 200
        operator: eq
        message: "Expected HTTP 200"
`
