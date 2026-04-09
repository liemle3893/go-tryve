package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// configTemplate is the default e2e.config.yaml written by `tryve init`.
const configTemplate = `version: "1.0"

environments:
  local:
    baseUrl: "http://localhost:3000"
    adapters:
      http: {}
      shell: {}

defaults:
  timeout: 30000
  retries: 0
  retryDelay: 1000
  parallel: 1

variables: {}

hooks:
  beforeAll: ""
  afterAll: ""
  beforeEach: ""
  afterEach: ""

reporters:
  - type: console
`

// newInitCmd constructs the `init` sub-command which creates a starter
// e2e.config.yaml in the current working directory.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a starter e2e.config.yaml in the current directory",
		Args:  cobra.NoArgs,
		RunE:  initCmdHandler,
	}
}

// initCmdHandler implements the `init` command execution logic.
func initCmdHandler(cmd *cobra.Command, _ []string) error {
	const outputFile = "e2e.config.yaml"

	if _, err := os.Stat(outputFile); err == nil {
		return fmt.Errorf("%s already exists; remove it first or edit it directly", outputFile)
	}

	if err := os.WriteFile(outputFile, []byte(configTemplate), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputFile, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s — edit it to match your environment.\n", outputFile)
	return nil
}
