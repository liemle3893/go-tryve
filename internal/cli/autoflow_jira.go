package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/liemle3893/autoflow/internal/autoflow/jira"
	"github.com/liemle3893/autoflow/internal/autoflow/state"
)

func newAutoflowJiraCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jira",
		Short: "Jira config, attachments, and REST helpers",
	}
	cmd.AddCommand(
		newAutoflowJiraConfigCmd(),
		newAutoflowJiraUploadCmd(),
		newAutoflowJiraDownloadCmd(),
		newAutoflowJiraFetchCmd(),
		newAutoflowJiraSearchCmd(),
		newAutoflowJiraTransitionsCmd(),
		newAutoflowJiraTransitionCmd(),
	)
	return cmd
}

// jiraClient constructs a Jira REST client from the cached config + env.
func jiraClient() (*jira.Client, error) {
	root, err := state.RepoRoot()
	if err != nil {
		return nil, err
	}
	creds, err := jira.ResolveCredentials(root)
	if err != nil {
		return nil, err
	}
	return jira.NewClient(creds), nil
}

// writeJSON renders v as pretty JSON to --out file (when set) or to stdout.
func writeJSON(cmd *cobra.Command, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')
	out, _ := cmd.Flags().GetString("out")
	if out == "" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s (%d bytes)\n", out, len(data))
	return nil
}

// parseCSV splits a comma-separated flag into a trimmed slice, dropping empty entries.
func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func newAutoflowJiraFetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch <KEY>",
		Short: "Fetch a Jira issue as JSON (GET /rest/api/3/issue/{key})",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := jiraClient()
			if err != nil {
				return err
			}
			fields, _ := cmd.Flags().GetString("fields")
			expand, _ := cmd.Flags().GetString("expand")
			issue, err := c.GetIssue(context.Background(), args[0], parseCSV(fields), parseCSV(expand))
			if err != nil {
				return err
			}
			return writeJSON(cmd, issue)
		},
	}
	cmd.Flags().String("fields", "", "comma-separated list of fields to include")
	cmd.Flags().String("expand", "", "comma-separated expand selectors (e.g. renderedFields)")
	cmd.Flags().String("out", "", "write JSON to FILE instead of stdout")
	return cmd
}

func newAutoflowJiraSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search --jql '<JQL>'",
		Short: "Run a JQL search against /rest/api/3/search/jql",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jql, _ := cmd.Flags().GetString("jql")
			if strings.TrimSpace(jql) == "" {
				return errors.New("--jql is required")
			}
			c, err := jiraClient()
			if err != nil {
				return err
			}
			fields, _ := cmd.Flags().GetString("fields")
			token, _ := cmd.Flags().GetString("page-token")
			page, err := c.SearchJQL(context.Background(), jql, parseCSV(fields), token)
			if err != nil {
				return err
			}
			return writeJSON(cmd, page)
		},
	}
	cmd.Flags().String("jql", "", "JQL query (required)")
	cmd.Flags().String("fields", "", "comma-separated list of fields to include")
	cmd.Flags().String("page-token", "", "nextPageToken returned by a previous search")
	cmd.Flags().String("out", "", "write JSON to FILE instead of stdout")
	return cmd
}

func newAutoflowJiraTransitionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transitions <KEY>",
		Short: "List transitions available on the issue (JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := jiraClient()
			if err != nil {
				return err
			}
			ts, err := c.GetTransitions(context.Background(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd, ts)
		},
	}
	cmd.Flags().String("out", "", "write JSON to FILE instead of stdout")
	return cmd
}

func newAutoflowJiraTransitionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transition <KEY> (--name NAME | --id ID)",
		Short: "Transition an issue by transition name (case-insensitive) or id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			id, _ := cmd.Flags().GetString("id")
			if name == "" && id == "" {
				return errors.New("one of --name or --id is required")
			}
			if name != "" && id != "" {
				return errors.New("--name and --id are mutually exclusive")
			}
			c, err := jiraClient()
			if err != nil {
				return err
			}
			key := args[0]
			if name != "" {
				ts, err := c.GetTransitions(context.Background(), key)
				if err != nil {
					return err
				}
				tr, err := jira.FindTransitionByName(ts, name)
				if err != nil {
					// The error already includes available names; re-emit to stderr.
					fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
					return fmt.Errorf("transition %q not found for %s", name, key)
				}
				id = tr.ID
			}
			if err := c.DoTransition(context.Background(), key, id); err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if name != "" {
				fmt.Fprintf(w, "Transitioned %s via %q (id=%s)\n", key, name, id)
			} else {
				fmt.Fprintf(w, "Transitioned %s via id=%s\n", key, id)
			}
			return nil
		},
	}
	cmd.Flags().String("name", "", "transition name (case-insensitive, e.g. 'Start Dev')")
	cmd.Flags().String("id", "", "transition id (bypass name lookup)")
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
	setCmd.Flags().String("cloud-id", "", "Atlassian cloud id (optional; filled in by myself round-trip later)")
	setCmd.Flags().String("site-url", "", "Jira site URL, e.g. your-org.atlassian.net (required)")
	setCmd.Flags().String("project-key", "", "Default Jira project key (required)")
	setCmd.Flags().String("email", "", "Jira account email (strongly recommended)")
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
