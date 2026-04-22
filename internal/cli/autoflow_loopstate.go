package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func newAutoflowLoopStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "loop-state",
		Short: "Generic agentic-loop state manager (replaces loop-state.sh)",
	}
	cmd.AddCommand(newLoopStateInitCmd(), newLoopStateAppendCmd(), newLoopStateReadCmd(), newLoopStateRoundCountCmd())
	return cmd
}

func newLoopStateInitCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init <state-file>",
		Short: "Create a new loop state file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loop, _ := cmd.Flags().GetString("loop")
			ticket, _ := cmd.Flags().GetString("ticket")
			maxRounds, _ := cmd.Flags().GetInt("max-rounds")
			force, _ := cmd.Flags().GetBool("force")
			return state.InitLoop(args[0], loop, ticket, maxRounds, force)
		},
	}
	c.Flags().String("loop", "", "loop name (required)")
	c.Flags().String("ticket", "", "ticket key (required)")
	c.Flags().Int("max-rounds", 0, "hard cap on rounds (required)")
	c.Flags().Bool("force", false, "overwrite an existing file")
	_ = c.MarkFlagRequired("loop")
	_ = c.MarkFlagRequired("ticket")
	_ = c.MarkFlagRequired("max-rounds")
	return c
}

func newLoopStateAppendCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "append <state-file>",
		Short: "Append a round to an existing loop state file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			round, _ := cmd.Flags().GetString("round-json")
			if round == "" {
				return fmt.Errorf("--round-json is required")
			}
			return state.AppendRound(args[0], json.RawMessage(round))
		},
	}
	c.Flags().String("round-json", "", "round body as JSON (required; must contain .status)")
	_ = c.MarkFlagRequired("round-json")
	return c
}

func newLoopStateReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <state-file>",
		Short: "Print the current state file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

func newLoopStateRoundCountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "round-count <state-file>",
		Short: "Print the current round count (0 when file is missing)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := state.RoundCount(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), n)
			return nil
		},
	}
}
