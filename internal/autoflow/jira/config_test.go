package jira

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSetRead_RoundTrip(t *testing.T) {
	root := t.TempDir()
	_, err := Set(root, "abc-123", "https://x.atlassian.net", "PROJ", "me@x")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.CloudID != "abc-123" || c.Email != "me@x" {
		t.Errorf("round-trip lost data: %+v", c)
	}
	// CachedAt should be non-empty ISO-like.
	if c.CachedAt == "" {
		t.Errorf("cached_at not set")
	}
}

func TestSet_Required(t *testing.T) {
	root := t.TempDir()
	if _, err := Set(root, "", "url", "p", ""); err == nil {
		t.Errorf("empty cloudId should fail")
	}
	if _, err := Set(root, "c", "", "p", ""); err == nil {
		t.Errorf("empty siteUrl should fail")
	}
	if _, err := Set(root, "c", "url", "", ""); err == nil {
		t.Errorf("empty projectKey should fail")
	}
}

func TestRead_Missing(t *testing.T) {
	_, err := Read(t.TempDir())
	if !errors.Is(err, ErrNoConfig) {
		t.Errorf("want ErrNoConfig, got %v", err)
	}
}

func TestGet_AllFields(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c-1", "https://x.atlassian.net", "P", "me@x")
	cases := map[string]string{
		"cloudId":    "c-1",
		"siteUrl":    "https://x.atlassian.net",
		"projectKey": "P",
		"email":      "me@x",
	}
	for field, want := range cases {
		got, err := Get(root, field)
		if err != nil {
			t.Errorf("get %s: %v", field, err)
		}
		if got != want {
			t.Errorf("get %s: got %q want %q", field, got, want)
		}
	}
	if _, err := Get(root, "bogus"); !errors.Is(err, ErrUnknownField) {
		t.Errorf("want ErrUnknownField, got %v", err)
	}
}

func TestDel_WholeConfig(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c", "u", "p", "e")
	if err := Del(root, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ConfigPath(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("config file not removed, got %v", err)
	}
	// Idempotent.
	if err := Del(root, ""); err != nil {
		t.Errorf("del on missing file: %v", err)
	}
}

func TestDel_Field(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c", "u", "p", "me@x")
	if err := Del(root, "email"); err != nil {
		t.Fatal(err)
	}
	c, _ := Read(root)
	if c.Email != "" {
		t.Errorf("email not cleared, got %q", c.Email)
	}
	if c.CloudID != "c" {
		t.Errorf("del clobbered other fields, cloudId=%q", c.CloudID)
	}
}

func TestShow_IsValidJSON(t *testing.T) {
	root := t.TempDir()
	_, _ = Set(root, "c", "u", "p", "e")
	out, err := Show(root)
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := json.Unmarshal(out, &c); err != nil {
		t.Errorf("Show output not valid JSON: %v", err)
	}
}

func TestConfigPath(t *testing.T) {
	got := ConfigPath("/r")
	want := filepath.Join("/r", ".autoflow", "jira-config.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHostFromSiteURL(t *testing.T) {
	cases := map[string]string{
		"https://x.atlassian.net": "x.atlassian.net",
		"http://x.atlassian.net":  "x.atlassian.net",
		"x.atlassian.net":         "x.atlassian.net",
		"":                        "",
	}
	for in, want := range cases {
		if got := HostFromSiteURL(in); got != want {
			t.Errorf("HostFromSiteURL(%q)=%q, want %q", in, got, want)
		}
	}
}
