package config

// RawConfig is the direct YAML-unmarshalled representation of e2e.config.yaml.
type RawConfig struct {
	Version      string                       `yaml:"version"`
	TestDir      string                       `yaml:"testDir"`
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	Defaults     DefaultsConfig               `yaml:"defaults"`
	Variables    map[string]any               `yaml:"variables"`
	Hooks        HooksConfig                  `yaml:"hooks"`
	Reporters    []ReporterConfig             `yaml:"reporters"`
}

// EnvironmentConfig holds the settings for a single named environment,
// including its base URL and per-adapter configuration maps.
type EnvironmentConfig struct {
	BaseURL  string                    `yaml:"baseUrl"`
	Adapters map[string]map[string]any `yaml:"adapters"`
}

// DefaultsConfig specifies the fallback values applied to every test when
// not overridden at the test level.
type DefaultsConfig struct {
	Timeout    int `yaml:"timeout"`
	Retries    int `yaml:"retries"`
	RetryDelay int `yaml:"retryDelay"`
	Parallel   int `yaml:"parallel"`
}

// HooksConfig holds the optional lifecycle script paths executed around each
// test and around the whole suite.
type HooksConfig struct {
	BeforeAll  string `yaml:"beforeAll"`
	AfterAll   string `yaml:"afterAll"`
	BeforeEach string `yaml:"beforeEach"`
	AfterEach  string `yaml:"afterEach"`
}

// ReporterConfig describes a single output reporter including its type,
// output destination, and verbosity flag.
type ReporterConfig struct {
	Type    string `yaml:"type"`
	Output  string `yaml:"output"`
	Verbose bool   `yaml:"verbose"`
}

// LoadedConfig is the fully-resolved configuration ready for use at runtime.
// It merges the raw YAML values with OS env var substitutions and builtin defaults.
type LoadedConfig struct {
	Raw             RawConfig
	Environment     EnvironmentConfig
	EnvironmentName string
	TestDir         string
	Defaults        DefaultsConfig
	Variables       map[string]any
	Hooks           HooksConfig
	Reporters       []ReporterConfig
}
