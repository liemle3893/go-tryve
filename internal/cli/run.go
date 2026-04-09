package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/liemle3893/e2e-runner/internal/adapter"
	"github.com/liemle3893/e2e-runner/internal/config"
	"github.com/liemle3893/e2e-runner/internal/executor"
	"github.com/liemle3893/e2e-runner/internal/loader"
	"github.com/liemle3893/e2e-runner/internal/reporter"
	"github.com/liemle3893/e2e-runner/internal/tryve"
)

// newRunCmd constructs the `run` sub-command which discovers, filters, and
// executes YAML test files with the configured adapters.
func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Discover and run YAML test files",
		RunE:  runCmdHandler,
	}

	cmd.Flags().StringP("test-dir", "d", "tests", "directory to search for test files")
	cmd.Flags().IntP("parallel", "p", 0, "number of tests to run in parallel (0 = use config default)")
	cmd.Flags().IntP("timeout", "t", 0, "per-test timeout in milliseconds (0 = use config default)")
	cmd.Flags().IntP("retries", "r", -1, "number of retries on failure (-1 = use config default)")
	cmd.Flags().Bool("bail", false, "stop after the first test failure")
	cmd.Flags().StringP("grep", "g", "", "filter tests by name (regexp or substring)")
	cmd.Flags().StringSlice("tag", nil, "filter tests by tag (can be repeated)")
	cmd.Flags().String("priority", "", "filter tests by priority (P0, P1, P2, P3)")
	cmd.Flags().Bool("dry-run", false, "print matching tests without executing them")
	cmd.Flags().Bool("skip-setup", false, "skip the setup phase for every test")
	cmd.Flags().Bool("skip-teardown", false, "skip the teardown phase for every test")
	cmd.Flags().StringSlice("reporter", nil, "additional reporter names (can be repeated)")
	cmd.Flags().StringP("output", "o", "", "output file path for file-based reporters")
	cmd.Flags().Bool("verbose", false, "enable verbose per-step output")

	return cmd
}

// runCmdHandler implements the `run` command execution logic.
func runCmdHandler(cmd *cobra.Command, _ []string) error {
	// Set up cancellable context tied to OS interrupts.
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Read persistent flags (defined on root, inherited by all sub-commands).
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	envName, _ := cmd.Root().PersistentFlags().GetString("env")

	// Read run-specific flags.
	testDir, _ := cmd.Flags().GetString("test-dir")
	parallel, _ := cmd.Flags().GetInt("parallel")
	timeout, _ := cmd.Flags().GetInt("timeout")
	retries, _ := cmd.Flags().GetInt("retries")
	bail, _ := cmd.Flags().GetBool("bail")
	grep, _ := cmd.Flags().GetString("grep")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	priority, _ := cmd.Flags().GetString("priority")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	skipSetup, _ := cmd.Flags().GetBool("skip-setup")
	skipTeardown, _ := cmd.Flags().GetBool("skip-teardown")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Load configuration.
	cfg, err := config.Load(cfgPath, envName)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI overrides on top of config defaults.
	if parallel > 0 {
		cfg.Defaults.Parallel = parallel
	}
	if timeout > 0 {
		cfg.Defaults.Timeout = timeout
	}
	if retries >= 0 {
		cfg.Defaults.Retries = retries
	}

	// Discover test files.
	paths, err := loader.Discover(testDir)
	if err != nil {
		return fmt.Errorf("discovering tests in %q: %w", testDir, err)
	}

	// Parse and validate each discovered file.
	var tests []*tryve.TestDefinition
	for _, p := range paths {
		td, parseErr := loader.ParseFile(p)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "WARN  parse error %s: %v\n", p, parseErr)
			continue
		}
		if errs := loader.Validate(td); len(errs) > 0 {
			for _, ve := range errs {
				fmt.Fprintf(os.Stderr, "WARN  validation error %s: %v\n", p, ve)
			}
			continue
		}
		tests = append(tests, td)
	}

	// Apply filters.
	filtered := executor.FilterTests(tests, executor.FilterOptions{
		Tags:     tags,
		Grep:     grep,
		Priority: priority,
	})

	// Dry-run: print matching tests and exit.
	if dryRun {
		for _, td := range filtered {
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (%s)\n", td.Priority, td.Name, td.SourceFile)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\n%d test(s) matched\n", len(filtered))
		return nil
	}

	// Honour skip-setup / skip-teardown by clearing those phases.
	if skipSetup {
		for _, td := range filtered {
			td.Setup = nil
		}
	}
	if skipTeardown {
		for _, td := range filtered {
			td.Teardown = nil
		}
	}

	// Build reporter.
	consoleRep := reporter.NewConsoleFromEnv(verbose)

	// Build adapter registry.
	reg := adapter.NewRegistry()
	baseURL := cfg.Environment.BaseURL
	reg.Register("http", adapter.NewHTTPAdapter(baseURL))
	reg.Register("shell", adapter.NewShellAdapter(&adapter.ShellConfig{}))
	defer reg.CloseAll(ctx)

	// Create and configure orchestrator.
	orch := executor.NewOrchestrator(reg, consoleRep, cfg)
	orch.SetBail(bail)

	// Run all filtered tests.
	result := orch.Run(ctx, filtered)

	if result.Failed > 0 {
		os.Exit(1)
	}
	return nil
}
