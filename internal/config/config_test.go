package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/liemle3893/e2e-runner/internal/config"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// writeYAML writes content to a temporary file and returns its path.
// The file is automatically removed when the test completes.
func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "e2e.config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
	return path
}

// TestLoad_MinimalConfig verifies that a minimal valid config is loaded, baseUrl is set,
// and all builtin defaults are applied when not specified in the config.
func TestLoad_MinimalConfig(t *testing.T) {
	yaml := `
version: "1.0"
environments:
  production:
    baseUrl: "https://api.example.com"
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "production")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.EnvironmentName != "production" {
		t.Errorf("EnvironmentName = %q, want %q", cfg.EnvironmentName, "production")
	}
	if cfg.Environment.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.Environment.BaseURL, "https://api.example.com")
	}

	// Builtin defaults must be applied.
	if cfg.Defaults.Timeout != 30000 {
		t.Errorf("Defaults.Timeout = %d, want 30000", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.RetryDelay != 1000 {
		t.Errorf("Defaults.RetryDelay = %d, want 1000", cfg.Defaults.RetryDelay)
	}
	if cfg.Defaults.Parallel != 1 {
		t.Errorf("Defaults.Parallel = %d, want 1", cfg.Defaults.Parallel)
	}
	if cfg.Defaults.Retries != 0 {
		t.Errorf("Defaults.Retries = %d, want 0", cfg.Defaults.Retries)
	}

	// Default reporter must be injected when reporters are absent.
	if len(cfg.Reporters) != 1 {
		t.Fatalf("len(Reporters) = %d, want 1", len(cfg.Reporters))
	}
	if cfg.Reporters[0].Type != "console" {
		t.Errorf("Reporters[0].Type = %q, want %q", cfg.Reporters[0].Type, "console")
	}
}

// TestLoad_EnvVarResolution verifies that ${VAR_NAME} placeholders in baseUrl are
// resolved to their current OS environment variable values.
func TestLoad_EnvVarResolution(t *testing.T) {
	t.Setenv("TEST_BASE_URL", "https://resolved.example.com")

	yaml := `
version: "1.0"
environments:
  staging:
    baseUrl: "${TEST_BASE_URL}"
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "staging")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	want := "https://resolved.example.com"
	if cfg.Environment.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", cfg.Environment.BaseURL, want)
	}
}

// TestLoad_MissingEnvVarInBaseURL_Errors verifies that a missing OS environment variable
// referenced in baseUrl causes a ConfigError (strict resolution).
func TestLoad_MissingEnvVarInBaseURL_Errors(t *testing.T) {
	// Ensure the variable is definitely not set.
	t.Setenv("DEFINITELY_NOT_SET_XYZ", "")
	os.Unsetenv("DEFINITELY_NOT_SET_XYZ") //nolint:errcheck

	yaml := `
version: "1.0"
environments:
  test:
    baseUrl: "${DEFINITELY_NOT_SET_XYZ}"
`
	path := writeYAML(t, yaml)

	_, err := config.Load(path, "test")
	if err == nil {
		t.Fatal("Load() expected error for missing env var in baseUrl, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "CONFIG_ERROR" {
		t.Errorf("error Code = %q, want %q", tryveErr.Code, "CONFIG_ERROR")
	}
}

// TestLoad_MissingEnvironment_Errors verifies that requesting a non-existent environment
// returns a ConfigError whose message lists the available environment names.
func TestLoad_MissingEnvironment_Errors(t *testing.T) {
	yaml := `
version: "1.0"
environments:
  production:
    baseUrl: "https://api.example.com"
  staging:
    baseUrl: "https://staging.example.com"
`
	path := writeYAML(t, yaml)

	_, err := config.Load(path, "nonexistent")
	if err == nil {
		t.Fatal("Load() expected error for missing environment, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "CONFIG_ERROR" {
		t.Errorf("error Code = %q, want %q", tryveErr.Code, "CONFIG_ERROR")
	}

	// Hint should mention available environments.
	if tryveErr.Hint == "" {
		t.Error("Hint should list available environments, got empty string")
	}
}

// TestLoad_WithAdapters verifies that adapter configurations nested under an environment
// are correctly parsed into the loaded config.
func TestLoad_WithAdapters(t *testing.T) {
	yaml := `
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:8080"
    adapters:
      http:
        timeout: 5000
        retries: 3
      postgresql:
        connectionString: "postgres://user:pass@localhost/db"
        poolSize: 5
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "local")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	httpAdapter, ok := cfg.Environment.Adapters["http"]
	if !ok {
		t.Fatal("http adapter not found in loaded config")
	}
	if timeout, _ := httpAdapter["timeout"].(int); timeout != 5000 {
		t.Errorf("http.timeout = %v, want 5000", httpAdapter["timeout"])
	}

	pgAdapter, ok := cfg.Environment.Adapters["postgresql"]
	if !ok {
		t.Fatal("postgresql adapter not found in loaded config")
	}
	if cs, _ := pgAdapter["connectionString"].(string); cs != "postgres://user:pass@localhost/db" {
		t.Errorf("postgresql.connectionString = %q, want %q", pgAdapter["connectionString"], "postgres://user:pass@localhost/db")
	}
	if poolSize, _ := pgAdapter["poolSize"].(int); poolSize != 5 {
		t.Errorf("postgresql.poolSize = %v, want 5", pgAdapter["poolSize"])
	}
}

// TestLoad_WithDefaults verifies that explicit defaults in the config override the builtin defaults.
func TestLoad_WithDefaults(t *testing.T) {
	yaml := `
version: "1.0"
environments:
  production:
    baseUrl: "https://api.example.com"
defaults:
  timeout: 60000
  retries: 3
  retryDelay: 2000
  parallel: 8
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "production")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Defaults.Timeout != 60000 {
		t.Errorf("Defaults.Timeout = %d, want 60000", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.Retries != 3 {
		t.Errorf("Defaults.Retries = %d, want 3", cfg.Defaults.Retries)
	}
	if cfg.Defaults.RetryDelay != 2000 {
		t.Errorf("Defaults.RetryDelay = %d, want 2000", cfg.Defaults.RetryDelay)
	}
	if cfg.Defaults.Parallel != 8 {
		t.Errorf("Defaults.Parallel = %d, want 8", cfg.Defaults.Parallel)
	}
}

// TestLoad_InvalidVersion_Errors verifies that a config with an unsupported version field
// returns a ConfigError.
func TestLoad_InvalidVersion_Errors(t *testing.T) {
	yaml := `
version: "2.0"
environments:
  production:
    baseUrl: "https://api.example.com"
`
	path := writeYAML(t, yaml)

	_, err := config.Load(path, "production")
	if err == nil {
		t.Fatal("Load() expected error for invalid version, got nil")
	}

	var tryveErr *tryve.TryveError
	if !errors.As(err, &tryveErr) {
		t.Fatalf("expected *tryve.TryveError, got %T: %v", err, err)
	}
	if tryveErr.Code != "CONFIG_ERROR" {
		t.Errorf("error Code = %q, want %q", tryveErr.Code, "CONFIG_ERROR")
	}
}

// TestLoad_NonStrictAdapterEnvVars verifies that missing env vars in adapter config
// are left as-is (non-strict) and do not cause an error.
func TestLoad_NonStrictAdapterEnvVars(t *testing.T) {
	// Ensure the variable is absent.
	os.Unsetenv("UNSET_ADAPTER_VAR") //nolint:errcheck

	yaml := `
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:8080"
    adapters:
      postgresql:
        connectionString: "${UNSET_ADAPTER_VAR}"
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "local")
	if err != nil {
		t.Fatalf("Load() unexpected error for unresolved adapter var: %v", err)
	}

	pgAdapter := cfg.Environment.Adapters["postgresql"]
	cs, _ := pgAdapter["connectionString"].(string)
	// Non-strict: original placeholder must be preserved.
	if cs != "${UNSET_ADAPTER_VAR}" {
		t.Errorf("connectionString = %q, want %q (placeholder preserved)", cs, "${UNSET_ADAPTER_VAR}")
	}
}

// TestLoad_NonStrictVariablesEnvVars verifies that missing env vars in variables section
// are left as-is (non-strict) and do not cause an error.
func TestLoad_NonStrictVariablesEnvVars(t *testing.T) {
	os.Unsetenv("UNSET_VAR") //nolint:errcheck

	yaml := `
version: "1.0"
environments:
  local:
    baseUrl: "http://localhost:8080"
variables:
  token: "${UNSET_VAR}"
  plain: "static"
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "local")
	if err != nil {
		t.Fatalf("Load() unexpected error for unresolved variable var: %v", err)
	}

	token, _ := cfg.Variables["token"].(string)
	if token != "${UNSET_VAR}" {
		t.Errorf("Variables[token] = %q, want %q (placeholder preserved)", token, "${UNSET_VAR}")
	}
	plain, _ := cfg.Variables["plain"].(string)
	if plain != "static" {
		t.Errorf("Variables[plain] = %q, want %q", plain, "static")
	}
}

// TestLoad_WithReporters verifies that explicit reporters in the config are loaded correctly
// and do NOT get the default console reporter appended.
func TestLoad_WithReporters(t *testing.T) {
	yaml := `
version: "1.0"
environments:
  production:
    baseUrl: "https://api.example.com"
reporters:
  - type: junit
    output: "./reports/junit.xml"
  - type: html
    output: "./reports/report.html"
    verbose: true
`
	path := writeYAML(t, yaml)

	cfg, err := config.Load(path, "production")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if len(cfg.Reporters) != 2 {
		t.Fatalf("len(Reporters) = %d, want 2", len(cfg.Reporters))
	}
	if cfg.Reporters[0].Type != "junit" {
		t.Errorf("Reporters[0].Type = %q, want %q", cfg.Reporters[0].Type, "junit")
	}
	if cfg.Reporters[1].Type != "html" {
		t.Errorf("Reporters[1].Type = %q, want %q", cfg.Reporters[1].Type, "html")
	}
	if !cfg.Reporters[1].Verbose {
		t.Error("Reporters[1].Verbose should be true")
	}
}
