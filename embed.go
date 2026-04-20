// Package assets embeds the skills bundle and documentation sections so they
// can be installed into a user's project by the `tryve install --skills`
// command without requiring access to the source tree at runtime.
package assets

import "embed"

// SkillsFS contains the skills/e2e-runner/ directory, including SKILL.md and
// any future sub-directories.
//
//go:embed skills/e2e-runner
var SkillsFS embed.FS

// DocsSectionsFS contains the docs/sections/ directory (markdown references
// plus index.json) that the install command copies into the user's project as
// the skill's reference material.
//
//go:embed docs/sections
var DocsSectionsFS embed.FS

// AutoflowSkillsFS contains the vendored autoflow skills that `tryve
// install --autoflow` drops into .claude/skills/.
//
//go:embed skills/autoflow
var AutoflowSkillsFS embed.FS

// AutoflowAgentsFS contains the vendored autoflow agents for
// .claude/agents/autoflow-*.md.
//
//go:embed agents/autoflow
var AutoflowAgentsFS embed.FS
