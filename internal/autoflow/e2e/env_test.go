package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotenv_Basic(t *testing.T) {
	dir := t.TempDir()
	body := `# a comment
FOO=bar
EMPTY_OK=
QUOTED="with spaces"
SINGLE='single'
export EXPORTED=value
BAD NAME=should-skip
# trailing comment
BAZ=last
`
	p := filepath.Join(dir, ".env")
	_ = os.WriteFile(p, []byte(body), 0o644)

	clean := []string{"FOO", "EMPTY_OK", "QUOTED", "SINGLE", "EXPORTED", "BAZ"}
	for _, k := range clean {
		_ = os.Unsetenv(k)
	}

	if err := LoadDotenv(p); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"FOO":      "bar",
		"EMPTY_OK": "",
		"QUOTED":   "with spaces",
		"SINGLE":   "single",
		"EXPORTED": "value",
		"BAZ":      "last",
	}
	for k, want := range cases {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q; want %q", k, got, want)
		}
	}
}

func TestLoadDotenv_MissingIsOK(t *testing.T) {
	if err := LoadDotenv(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Errorf("missing file should return nil, got %v", err)
	}
}

func TestLoadDotenv_OverridesExisting(t *testing.T) {
	t.Setenv("OVERRIDE_KEY", "OLD")
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	_ = os.WriteFile(p, []byte("OVERRIDE_KEY=NEW\n"), 0o644)
	if err := LoadDotenv(p); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("OVERRIDE_KEY"); got != "NEW" {
		t.Errorf(".env should override existing env, got %q", got)
	}
}

func TestLoadLocalSettingsJSON_FillsUnsetOnly(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "local.settings.json")
	body := `{
		"Values": {
			"SHOULD_APPLY": "first",
			"ALREADY_SET":  "from-json",
			"NULL_SKIPPED": null,
			"BAD NAME":     "dropped"
		}
	}`
	_ = os.WriteFile(p, []byte(body), 0o644)

	t.Setenv("ALREADY_SET", "keep")
	_ = os.Unsetenv("SHOULD_APPLY")

	if err := LoadLocalSettingsJSON(p); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SHOULD_APPLY"); got != "first" {
		t.Errorf("SHOULD_APPLY = %q; want 'first'", got)
	}
	if got := os.Getenv("ALREADY_SET"); got != "keep" {
		t.Errorf("ALREADY_SET must not be overwritten; got %q", got)
	}
	if got := os.Getenv("NULL_SKIPPED"); got != "" {
		t.Errorf("null value should not be exported, got %q", got)
	}
}

func TestLoadLocalSettingsJSON_StringifiesScalars(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "local.settings.json")
	body := `{"Values": {"TIMEOUT": 42, "ENABLED": true}}`
	_ = os.WriteFile(p, []byte(body), 0o644)

	_ = os.Unsetenv("TIMEOUT")
	_ = os.Unsetenv("ENABLED")
	if err := LoadLocalSettingsJSON(p); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TIMEOUT") != "42" {
		t.Errorf("TIMEOUT = %q; want 42", os.Getenv("TIMEOUT"))
	}
	if os.Getenv("ENABLED") != "true" {
		t.Errorf("ENABLED = %q; want true", os.Getenv("ENABLED"))
	}
}
