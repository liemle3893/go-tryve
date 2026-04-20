package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/liemle3893/go-tryve/internal/autoflow/doctor"
	"github.com/liemle3893/go-tryve/internal/autoflow/state"
)

func newAutoflowDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Preflight checklist for autoflow dependencies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				// Doctor should still run without a repo root — use cwd.
				root, _ = os.Getwd()
			}
			worst, results := doctor.Run(context.Background(), doctor.Opts{Root: root})
			doctor.Format(cmd.OutOrStdout(), results)
			os.Exit(doctor.ExitCode(worst))
			return nil
		},
	}
}
