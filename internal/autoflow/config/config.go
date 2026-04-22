// Package config manages .autoflow/config.json — the per-repo autoflow
// preferences cache covering coding-agent selection and sandbox settings.
// Patterned after internal/autoflow/jira/config.go.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config is the on-disk cache at .autoflow/config.json.
type Config struct {
	CodingAgent string        `json:"coding_agent,omitempty"`
	Sandbox     SandboxConfig `json:"sandbox"`
}

// SandboxConfig describes sandbox preferences. Name defaults to the repo
// directory basename when empty; policy is sbx's --policy flag.
type SandboxConfig struct {
	Enabled     bool     `json:"enabled"`
	Name        string   `json:"name,omitempty"`
	ExtraMounts []string `json:"extra_mounts,omitempty"`
	Policy      string   `json:"policy,omitempty"`
}

// ErrUnknownField is returned by Get/Set/Del for names outside the allow-list.
var ErrUnknownField = errors.New("unknown autoflow config field")

// ErrInvalidValue is returned when a value fails validation (e.g. coding_agent).
var ErrInvalidValue = errors.New("invalid autoflow config value")

// allowedAgents is the closed set of supported coding agents.
var allowedAgents = []string{"claude", "copilot"}

// allowedFields lists every field name accepted by Set/Get/Del.
var allowedFields = []string{
	"coding_agent",
	"sandbox.enabled",
	"sandbox.name",
	"sandbox.policy",
	"sandbox.extra_mounts",
}

// Path returns the canonical config location under root.
func Path(root string) string {
	return filepath.Join(root, ".autoflow", "config.json")
}

// Read loads the config. Missing file is not an error — it returns a
// zero-value Config, matching how other autoflow packages treat absent caches
// as "no customisation yet".
func Read(root string) (*Config, error) {
	data, err := os.ReadFile(Path(root))
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read autoflow config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse autoflow config: %w", err)
	}
	return &c, nil
}

// Write persists c, creating parent directories as needed.
func Write(root string, c *Config) error {
	if c == nil {
		return errors.New("nil config")
	}
	if err := validateAgent(c.CodingAgent); err != nil {
		return err
	}
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write autoflow config: %w", err)
	}
	return nil
}

// Set updates a single field and persists. Fields accepted:
//
//	coding_agent          (claude|copilot)
//	sandbox.enabled       (true|false|1|0)
//	sandbox.name          (string)
//	sandbox.policy        (string)
//	sandbox.extra_mounts  (comma-separated list; empty string clears)
func Set(root, field, value string) error {
	c, err := Read(root)
	if err != nil {
		return err
	}
	switch field {
	case "coding_agent":
		if err := validateAgent(value); err != nil {
			return err
		}
		c.CodingAgent = value
	case "sandbox.enabled":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		c.Sandbox.Enabled = b
	case "sandbox.name":
		c.Sandbox.Name = value
	case "sandbox.policy":
		c.Sandbox.Policy = value
	case "sandbox.extra_mounts":
		c.Sandbox.ExtraMounts = parseList(value)
	default:
		return fmt.Errorf("%w: %q (allowed: %v)", ErrUnknownField, field, allowedFields)
	}
	return Write(root, c)
}

// Get returns the string form of one field.
func Get(root, field string) (string, error) {
	c, err := Read(root)
	if err != nil {
		return "", err
	}
	switch field {
	case "coding_agent":
		return c.CodingAgent, nil
	case "sandbox.enabled":
		return strconv.FormatBool(c.Sandbox.Enabled), nil
	case "sandbox.name":
		return c.Sandbox.Name, nil
	case "sandbox.policy":
		return c.Sandbox.Policy, nil
	case "sandbox.extra_mounts":
		return strings.Join(c.Sandbox.ExtraMounts, ","), nil
	default:
		return "", fmt.Errorf("%w: %q (allowed: %v)", ErrUnknownField, field, allowedFields)
	}
}

// Del clears one field when field is non-empty; otherwise removes the file.
func Del(root, field string) error {
	if field == "" {
		err := os.Remove(Path(root))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	c, err := Read(root)
	if err != nil {
		return err
	}
	switch field {
	case "coding_agent":
		c.CodingAgent = ""
	case "sandbox.enabled":
		c.Sandbox.Enabled = false
	case "sandbox.name":
		c.Sandbox.Name = ""
	case "sandbox.policy":
		c.Sandbox.Policy = ""
	case "sandbox.extra_mounts":
		c.Sandbox.ExtraMounts = nil
	default:
		return fmt.Errorf("%w: %q (allowed: %v)", ErrUnknownField, field, allowedFields)
	}
	return Write(root, c)
}

// Show returns the config as pretty-printed JSON with a trailing newline.
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

// validateAgent allows empty (uninitialised) or one of the known agents.
func validateAgent(v string) error {
	if v == "" {
		return nil
	}
	for _, a := range allowedAgents {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("%w: coding_agent=%q (allowed: %v)", ErrInvalidValue, v, allowedAgents)
}

func parseBool(v string) (bool, error) {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false, fmt.Errorf("%w: expected true|false, got %q", ErrInvalidValue, v)
	}
	return b, nil
}

func parseList(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
