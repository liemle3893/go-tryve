package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/liemle3893/autoflow/internal/autoflow/jira"
	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func newAutoflowJiraCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jira",
		Short: "Jira config and attachment helpers",
	}
	cmd.AddCommand(newAutoflowJiraConfigCmd(), newAutoflowJiraUploadCmd(), newAutoflowJiraDownloadCmd())
	return cmd
}

func newAutoflowJiraConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read or write .autoflow/jira-config.json",
	}

	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "Write the Jira cache",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			cloudID, _ := cmd.Flags().GetString("cloud-id")
			siteURL, _ := cmd.Flags().GetString("site-url")
			projectKey, _ := cmd.Flags().GetString("project-key")
			email, _ := cmd.Flags().GetString("email")
			if _, err := jira.Set(root, cloudID, siteURL, projectKey, email); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved Jira config to "+jira.ConfigPath(root))
			return nil
		},
	}
	setCmd.Flags().String("cloud-id", "", "Atlassian cloud id (required)")
	setCmd.Flags().String("site-url", "", "Jira site URL, e.g. your-org.atlassian.net (required)")
	setCmd.Flags().String("project-key", "", "Default Jira project key (required)")
	setCmd.Flags().String("email", "", "Jira account email (strongly recommended)")
	_ = setCmd.MarkFlagRequired("cloud-id")
	_ = setCmd.MarkFlagRequired("site-url")
	_ = setCmd.MarkFlagRequired("project-key")

	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Print one field from the Jira cache",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			field, _ := cmd.Flags().GetString("field")
			val, err := jira.Get(root, field)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
	getCmd.Flags().String("field", "", "one of cloudId|siteUrl|projectKey|email (required)")
	_ = getCmd.MarkFlagRequired("field")

	var delCmd = &cobra.Command{
		Use:   "del",
		Short: "Delete the whole cache, or one field with --field",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			field, _ := cmd.Flags().GetString("field")
			return jira.Del(root, field)
		},
	}
	delCmd.Flags().String("field", "", "optional field to clear; omit to delete entire config")

	var showCmd = &cobra.Command{
		Use:   "show",
		Short: "Print the cached Jira config as JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			out, err := jira.Show(root)
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write(out)
			return nil
		},
	}

	cmd.AddCommand(setCmd, getCmd, delCmd, showCmd)
	return cmd
}

func newAutoflowJiraUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload <issue-key> <file>...",
		Short: "Upload one or more files as Jira attachments",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			creds, err := jira.ResolveCredentials(root)
			if err != nil {
				return err
			}
			client := jira.NewClient(creds)
			key := args[0]
			files := args[1:]
			results, err := client.Upload(context.Background(), key, files)
			if err != nil {
				return err
			}
			var failed int
			for _, r := range results {
				if r.Error != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  FAIL  %s (%v)\n", r.Filename, r.Error)
					failed++
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  OK    %s (id: %s)\n", r.Filename, r.ID)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d file(s) failed to upload", failed)
			}
			return nil
		},
	}
}

func newAutoflowJiraDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download <issue-key> <output-dir>",
		Short: "Download every attachment from a Jira issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := state.RepoRoot()
			if err != nil {
				return err
			}
			creds, err := jira.ResolveCredentials(root)
			if err != nil {
				return err
			}
			client := jira.NewClient(creds)
			results, err := client.Download(context.Background(), args[0], args[1])
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No attachments found on "+args[0])
				return nil
			}
			var failed int
			for _, r := range results {
				if r.Error != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  FAIL  %s (%v)\n", r.Attachment.Filename, r.Error)
					failed++
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  OK    %s (%s, %d bytes)\n",
						r.Attachment.Filename, r.Attachment.MimeType, r.Attachment.Size)
				}
			}
			if failed > 0 {
				return errors.New("one or more downloads failed")
			}
			return nil
		},
	}
}
