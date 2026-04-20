// Package jira implements the autoflow Jira client + config cache. It
// replaces scripts/autoflow/{jira-config,jira-env,jira-upload,jira-download}.sh
// from winx-autoflow.
//
// Config lives at .autoflow/jira-config.json under the repo root. Only the
// bearer credential (JIRA_API_TOKEN) is ever required to be in the user's
// shell — everything else is cached.
package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config is the on-disk cache. Matches the jq schema in jira-config.sh so
// callers of either the bash or Go tooling see the same JSON.
type Config struct {
	CloudID    string `json:"cloudId"`
	SiteURL    string `json:"siteUrl"`
	ProjectKey string `json:"projectKey"`
	Email      string `json:"email,omitempty"`
	CachedAt   string `json:"cached_at"`
}

// ConfigPath returns the location of the cache file under root.
func ConfigPath(root string) string {
	return filepath.Join(root, ".autoflow", "jira-config.json")
}

// ErrNoConfig is returned when the cache file does not exist.
var ErrNoConfig = errors.New("jira config not cached")

// ErrUnknownField is returned by Get/Del when the field name is not one
// of the recognised keys.
var ErrUnknownField = errors.New("unknown jira config field")

// allowedFields is the whitelist of field names accepted by Get and the
// per-field form of Del. camelCase chosen to match the JSON keys users
// already see in the file.
var allowedFields = []string{"cloudId", "siteUrl", "projectKey", "email"}

// Read returns the cached config, or (nil, ErrNoConfig) when missing.
func Read(root string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoConfig
	}
	if err != nil {
		return nil, fmt.Errorf("read jira config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse jira config: %w", err)
	}
	return &c, nil
}

// Set writes the config to disk, replacing any previous cache. Email is
// optional but strongly recommended — upload/download require it for
// Basic auth. CachedAt is set to the current UTC instant.
func Set(root, cloudID, siteURL, projectKey, email string) (*Config, error) {
	if cloudID == "" || siteURL == "" || projectKey == "" {
		return nil, fmt.Errorf("cloudId, siteUrl, and projectKey are required")
	}
	c := &Config{
		CloudID:    cloudID,
		SiteURL:    siteURL,
		ProjectKey: projectKey,
		Email:      email,
		CachedAt:   time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	path := ConfigPath(root)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("write jira config: %w", err)
	}
	return c, nil
}

// Get returns the string value of one field. When the cache is absent,
// returns ErrNoConfig so the CLI can exit with a clear message.
func Get(root, field string) (string, error) {
	c, err := Read(root)
	if err != nil {
		return "", err
	}
	switch field {
	case "cloudId":
		return c.CloudID, nil
	case "siteUrl":
		return c.SiteURL, nil
	case "projectKey":
		return c.ProjectKey, nil
	case "email":
		return c.Email, nil
	default:
		return "", fmt.Errorf("%w: %q (allowed: %v)", ErrUnknownField, field, allowedFields)
	}
}

// Del removes the whole cache file when field is "". Otherwise clears one
// field (leaves the rest intact) and rewrites the cache. Returns nil when
// there is nothing to delete.
func Del(root, field string) error {
	if field == "" {
		err := os.Remove(ConfigPath(root))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	c, err := Read(root)
	if err != nil {
		if errors.Is(err, ErrNoConfig) {
			return nil
		}
		return err
	}
	switch field {
	case "cloudId":
		c.CloudID = ""
	case "siteUrl":
		c.SiteURL = ""
	case "projectKey":
		c.ProjectKey = ""
	case "email":
		c.Email = ""
	default:
		return fmt.Errorf("%w: %q (allowed: %v)", ErrUnknownField, field, allowedFields)
	}
	c.CachedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(root), data, 0o644)
}

// Show returns the cache JSON formatted for display (indented, trailing newline).
func Show(root string) ([]byte, error) {
	c, err := Read(root)
	if err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// HostFromSiteURL strips the scheme from siteUrl if one is present,
// returning bare hostname form (your-org.atlassian.net).
func HostFromSiteURL(siteURL string) string {
	return strings.TrimPrefix(strings.TrimPrefix(siteURL, "https://"), "http://")
}
