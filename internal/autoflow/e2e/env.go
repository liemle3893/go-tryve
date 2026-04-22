// Package e2e runs autoflow E2E tests against a worktree branch applied
// on the main repository. Replaces skills/autoflow-deliver/scripts/
// {e2e-local,e2e-loop}.sh with a Go implementation that calls the
// autoflow runner API directly instead of shelling out.
package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// posixEnvName validates an env var key per POSIX: starts with letter or
// underscore, contains only letters/digits/underscore. Keys that don't
// match are silently dropped when sourcing .env and local.settings.json —
// this matches the bash e2e-local.sh behaviour (defence against shell
// metacharacter injection via a malformed key).
var posixEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// LoadDotenv parses path as a .env file and applies each `KEY=VALUE` pair
// to the process environment. Lines starting with `#` and blank lines are
// ignored. Single- and double-quoted values are unquoted. Values already
// set on os.Environ ARE overwritten, matching the script's `set -a; .
// .env; set +a` semantics (".env is the developer's local override").
//
// A missing file returns nil. Parse errors are ignored line-by-line so a
// stray malformed entry does not block a whole bootstrap.
func LoadDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Optional `export KEY=...` prefix.
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if !posixEnvName.MatchString(key) {
			continue
		}
		value = unquote(value)
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

// LoadLocalSettingsJSON imports the Values.* map from an Azure Functions
// local.settings.json into the env, only for keys that are NOT already
// set. Null values are skipped. Keeps parity with the bash form of the
// script, which treated local.settings.json as a baseline and .env as an
// override.
func LoadLocalSettingsJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var parsed struct {
		Values map[string]any `json:"Values"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	for key, raw := range parsed.Values {
		if !posixEnvName.MatchString(key) {
			continue
		}
		if _, already := os.LookupEnv(key); already {
			continue
		}
		if raw == nil {
			continue
		}
		_ = os.Setenv(key, stringify(raw))
	}
	return nil
}

// ImportWorkspaceEnv runs the two loaders in the precedence order used by
// e2e-local.sh: .env overrides caller env; local.settings.json fills
// remaining gaps. Missing files are fine.
func ImportWorkspaceEnv(workdir string) error {
	if err := LoadDotenv(workdir + "/.env"); err != nil {
		return err
	}
	return LoadLocalSettingsJSON(workdir + "/local.settings.json")
}

// unquote strips matching surrounding single or double quotes.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// stringify renders a JSON scalar as its shell-friendly string form.
// Matches the bash form which pipes through `jq | tostring`.
func stringify(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		// JSON numbers come back as float64. Strip trailing zero decimal.
		s := fmt.Sprintf("%v", x)
		return s
	case nil:
		return ""
	default:
		raw, _ := json.Marshal(x)
		return string(raw)
	}
}
