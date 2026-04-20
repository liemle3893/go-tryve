package worktree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_Missing(t *testing.T) {
	c, err := ReadConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if c.InstallCmd != "" || c.BaseBranch != "" {
		t.Errorf("missing file should return zero config, got %+v", c)
	}
}

func TestReadConfig_Good(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".autoflow"), 0o755)
	body := map[string]any{
		"install_cmd":  "echo hi",
		"verify_cmd":   "echo ok",
		"base_branch":  "develop",
		"config_files": []string{".env"},
	}
	raw, _ := json.Marshal(body)
	_ = os.WriteFile(ConfigPath(dir), raw, 0o644)

	c, err := ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.InstallCmd != "echo hi" || c.BaseBranch != "develop" {
		t.Errorf("parse failed: %+v", c)
	}
	if len(c.ConfigFiles) != 1 || c.ConfigFiles[0] != ".env" {
		t.Errorf("config_files: %+v", c.ConfigFiles)
	}
}

func TestAutoDetect_Go(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	c := &Config{}
	AutoDetect(c, dir)
	if c.InstallCmd != "go mod download" {
		t.Errorf("want go mod download, got %q", c.InstallCmd)
	}
	if c.BaseBranch != "main" {
		t.Errorf("want base_branch=main default, got %q", c.BaseBranch)
	}
}

func TestAutoDetect_PnpmPrefersYarn(t *testing.T) {
	// yarn.lock wins over later ladder rungs; confirms order.
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644)
	c := &Config{}
	AutoDetect(c, dir)
	if c.InstallCmd != "yarn install --frozen-lockfile" {
		t.Errorf("yarn should win, got %q", c.InstallCmd)
	}
}

func TestAutoDetect_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644)
	c := &Config{InstallCmd: "custom-installer", BaseBranch: "trunk"}
	AutoDetect(c, dir)
	if c.InstallCmd != "custom-installer" || c.BaseBranch != "trunk" {
		t.Errorf("auto-detect must not overwrite user values, got %+v", c)
	}
}

func TestAutoDetect_ConfigFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "e2e.config.yaml"), []byte(""), 0o644)
	c := &Config{}
	AutoDetect(c, dir)
	found := map[string]bool{}
	for _, f := range c.ConfigFiles {
		found[f] = true
	}
	if !found[".env"] || !found["e2e.config.yaml"] {
		t.Errorf("config_files auto-detect missing entries, got %v", c.ConfigFiles)
	}
}
