package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// testYAMLTemplate is the starter YAML test file written by `tryve test create`.
const testYAMLTemplate = `name: "%s"
description: "Add a description for this test"
priority: P1
tags:
  - smoke

execute:
  - adapter: http
    action: request
    description: "GET request example"
    method: GET
    url: /health
    assert:
      - path: "$.status"
        equals: 200
`

// availableTemplates is the list of named templates for `tryve test list-templates`.
var availableTemplates = []string{"http", "shell"}

// httpTemplate is the starter template for an HTTP-focused test.
const httpTemplate = `name: "my-http-test"
description: "HTTP API test"
priority: P1
tags:
  - smoke

execute:
  - adapter: http
    action: request
    description: "Send GET request"
    method: GET
    url: /api/resource
    assert:
      - path: "$.status"
        equals: 200
`

// shellTemplate is the starter template for a shell command test.
const shellTemplate = `name: "my-shell-test"
description: "Shell command test"
priority: P2
tags:
  - shell

execute:
  - adapter: shell
    action: exec
    description: "Run a shell command"
    command: "echo hello world"
    assert:
      - path: "$.exitCode"
        equals: 0
`

// newTestCmd constructs the `test` sub-command group which provides helpers
// for creating new test files from templates.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Helpers for creating and managing test files",
	}
	cmd.AddCommand(newTestCreateCmd(), newTestListTemplatesCmd())
	return cmd
}

// newTestCreateCmd constructs the `test create <name>` sub-command.
func newTestCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new test file from the default template",
		Args:  cobra.ExactArgs(1),
		RunE:  testCreateCmdHandler,
	}
	cmd.Flags().StringP("template", "t", "http", "template to use (http, shell)")
	cmd.Flags().StringP("output", "o", "", "output file path (default: <name>.test.yaml)")
	return cmd
}

// testCreateCmdHandler implements the `test create` command.
func testCreateCmdHandler(cmd *cobra.Command, args []string) error {
	name := args[0]
	tmplName, _ := cmd.Flags().GetString("template")
	outputPath, _ := cmd.Flags().GetString("output")

	if outputPath == "" {
		// Derive file name from test name: replace spaces/slashes with dashes.
		safeName := strings.NewReplacer(" ", "-", "/", "-").Replace(name)
		outputPath = safeName + ".test.yaml"
	}

	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file %s already exists; choose a different name or remove it first", outputPath)
	}

	var content string
	switch tmplName {
	case "shell":
		content = shellTemplate
	default:
		content = fmt.Sprintf(testYAMLTemplate, name)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", outputPath)
	return nil
}

// newTestListTemplatesCmd constructs the `test list-templates` sub-command.
func newTestListTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-templates",
		Short: "Print the names of available test templates",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			for _, t := range availableTemplates {
				fmt.Fprintln(cmd.OutOrStdout(), t)
			}
		},
	}
}
