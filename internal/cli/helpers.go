package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/liemle3893/e2e-runner/internal/config"
)

// resolveTestDir determines the test directory by checking (in order):
// 1. The --test-dir CLI flag (if explicitly set by user)
// 2. The testDir field from e2e.config.yaml (resolved relative to the config file)
// 3. The default value of the --test-dir flag ("tests")
func resolveTestDir(cmd *cobra.Command) string {
	testDir, _ := cmd.Flags().GetString("test-dir")

	// If the user didn't explicitly set --test-dir, try the config file.
	if !cmd.Flags().Changed("test-dir") {
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		envName, _ := cmd.Root().PersistentFlags().GetString("env")
		if cfg, err := config.Load(cfgPath, envName); err == nil && cfg.TestDir != "" {
			// testDir in config is relative to the config file, not CWD.
			testDir = filepath.Join(filepath.Dir(cfgPath), cfg.TestDir)
		}
	}
	return testDir
}
