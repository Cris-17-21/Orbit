// Package skillinstall resolves Orbit's built-in skill to project-local agent
// destinations. It does not inspect or modify the project.
package skillinstall

import (
	"fmt"
	"sort"

	"github.com/Cris-17-21/Orbit/internal/initmemory"
	"github.com/Cris-17-21/Orbit/skills"
)

// Agent identifies an agent host that can discover project-local skills.
type Agent string

const (
	Codex       Agent = "codex"
	ClaudeCode  Agent = "claude"
	Antigravity Agent = "antigravity"
)

// Resolve returns deterministic managed-file updates for the selected agents.
// Codex and Antigravity share one path, so their selection is deduplicated.
func Resolve(agents ...Agent) ([]initmemory.ManagedFileUpdate, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("select at least one agent")
	}
	selected := make(map[Agent]bool, len(agents))
	for _, agent := range agents {
		switch agent {
		case Codex, ClaudeCode, Antigravity:
			selected[agent] = true
		default:
			return nil, fmt.Errorf("unsupported agent %q", agent)
		}
	}

	var updates []initmemory.ManagedFileUpdate
	if selected[Codex] || selected[Antigravity] {
		updates = append(updates, update(".agents/skills/orbit/SKILL.md", "project-orbit-skill-agents"))
	}
	if selected[ClaudeCode] {
		updates = append(updates, update(".claude/skills/orbit/SKILL.md", "project-orbit-skill-claude"))
	}
	sort.Slice(updates, func(i, j int) bool { return updates[i].Path < updates[j].Path })
	return updates, nil
}

func update(path, id string) initmemory.ManagedFileUpdate {
	return initmemory.ManagedFileUpdate{
		Path: path, ResourceID: id, Source: "orbit-skill-install", Content: skills.Orbit(),
	}
}
