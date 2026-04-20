// Package worktree orchestrates git-worktree bootstrap: copying gitignored
// .claude/ infrastructure + config files from the main dir into a new
// worktree, then running install and verify commands. Replaces
// scripts/autoflow/worktree-bootstrap.sh from winx-autoflow.
package worktree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the on-disk shape of .autoflow/bootstrap.json. All fields are
// optional so legacy configs remain readable.
type Config struct {
	Language    string   `json:"language,omitempty"`
	BaseBranch  string   `json:"base_branch,omitempty"`
	ConfigFiles []string `json:"config_files,omitempty"`
	InstallCmd  string   `json:"install_cmd,omitempty"`
	VerifyCmd   string   `json:"verify_cmd,omitempty"`
	BuildCmd    string   `json:"build_cmd,omitempty"`
	TestCmd     string   `json:"test_cmd,omitempty"`
	ServicesCmd string   `json:"services_cmd,omitempty"`
}

// ConfigPath returns the location of the bootstrap config under mainDir.
func ConfigPath(mainDir string) string {
	return filepath.Join(mainDir, ".autoflow", "bootstrap.json")
}

// ReadConfig returns the config if present. Returns a zero-valued Config
// (not an error) when the file is absent — callers should then call
// AutoDetect to fill in sensible fallbacks.
func ReadConfig(mainDir string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(mainDir))
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse bootstrap config: %w", err)
	}
	return &c, nil
}

// AutoDetect fills in InstallCmd and ConfigFiles based on files present in
// mainDir. Does not overwrite values already set on c. Mirrors the fallback
// ladder in worktree-bootstrap.sh.
func AutoDetect(c *Config, mainDir string) {
	if c.InstallCmd == "" {
		switch {
		case fileExists(mainDir, "go.mod"):
			c.InstallCmd = "go mod download"
		case fileExists(mainDir, "yarn.lock"):
			c.InstallCmd = "yarn install --frozen-lockfile"
		case fileExists(mainDir, "pnpm-lock.yaml"):
			c.InstallCmd = "pnpm install --frozen-lockfile"
		case fileExists(mainDir, "package-lock.json"):
			c.InstallCmd = "npm ci"
		case fileExists(mainDir, "Cargo.toml"):
			c.InstallCmd = "cargo fetch"
		case fileExists(mainDir, "requirements.txt"):
			c.InstallCmd = "pip install -r requirements.txt"
		}
	}
	if len(c.ConfigFiles) == 0 {
		for _, candidate := range []string{
			".env", ".env.local", "local.settings.json",
			"e2e.config.yaml", "config/local.yaml",
		} {
			if fileExists(mainDir, candidate) {
				c.ConfigFiles = append(c.ConfigFiles, candidate)
			}
		}
	}
	if c.BaseBranch == "" {
		c.BaseBranch = "main"
	}
}

func fileExists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}
