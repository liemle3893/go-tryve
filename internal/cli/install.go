package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	assets "github.com/liemle3893/go-tryve"
)

// newInstallCmd constructs the `install` sub-command which copies bundled
// skills and documentation references into the user's project under
// `.claude/skills/e2e-runner/`.
func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Claude Code skills into the current project",
		Long: `Install the bundled Claude Code skill into .claude/skills/e2e-runner/
in the current working directory.

Copies SKILL.md plus the documentation sections (as references/) from the
tryve binary's embedded bundle. Use --skills (default) to install skills.`,
		Args: cobra.NoArgs,
		RunE: installCmdHandler,
	}

	cmd.Flags().Bool("skills", false, "install Claude Code skills to .claude/skills/e2e-runner/")
	return cmd
}

// installCmdHandler implements the `install` command execution logic.
func installCmdHandler(cmd *cobra.Command, _ []string) error {
	skills, _ := cmd.Flags().GetBool("skills")
	if !skills {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Usage: tryve install --skills")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		fmt.Fprintln(out, "  --skills    Install Claude Code skills to .claude/skills/e2e-runner/")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("install: cannot determine working directory: %w", err)
	}

	destDir := filepath.Join(cwd, ".claude", "skills", "e2e-runner")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("install: creating %s: %w", destDir, err)
	}

	// Copy the skill bundle (SKILL.md and any sibling files) from
	// skills/e2e-runner into destDir.
	if err := copyEmbedDir(assets.SkillsFS, "skills/e2e-runner", destDir, nil); err != nil {
		return fmt.Errorf("install: copying skill bundle: %w", err)
	}

	// Copy the documentation sections into destDir/references, skipping the
	// internal index.json which is not used by the skill itself.
	refsDir := filepath.Join(destDir, "references")
	skip := map[string]struct{}{"docs/sections/index.json": {}}
	if err := copyEmbedDir(assets.DocsSectionsFS, "docs/sections", refsDir, skip); err != nil {
		return fmt.Errorf("install: copying documentation references: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Skills installed to %s\n", destDir)
	return nil
}

// copyEmbedDir recursively copies an embedded directory tree rooted at root
// into destDir on disk. Files matching any entry in skip (keyed by their
// embed path) are omitted.
func copyEmbedDir(src fs.FS, root, destDir string, skip map[string]struct{}) error {
	return fs.WalkDir(src, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if _, skipped := skip[path]; skipped {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))

		if d.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", target, err)
			}
			return nil
		}

		data, err := fs.ReadFile(src, path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("creating parent of %s: %w", target, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
		return nil
	})
}
