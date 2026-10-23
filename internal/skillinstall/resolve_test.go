package skillinstall

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Cris-17-21/Orbit/skills"
)

func TestResolveAgentDestinationsAndSharedPathDeduplication(t *testing.T) {
	tests := []struct {
		name   string
		agents []Agent
		paths  []string
		ids    []string
	}{
		{"codex", []Agent{Codex}, []string{".agents/skills/orbit/SKILL.md"}, []string{"project-orbit-skill-agents"}},
		{"antigravity", []Agent{Antigravity}, []string{".agents/skills/orbit/SKILL.md"}, []string{"project-orbit-skill-agents"}},
		{"shared path", []Agent{Antigravity, Codex}, []string{".agents/skills/orbit/SKILL.md"}, []string{"project-orbit-skill-agents"}},
		{"all", []Agent{ClaudeCode, Codex, Antigravity}, []string{".agents/skills/orbit/SKILL.md", ".claude/skills/orbit/SKILL.md"}, []string{"project-orbit-skill-agents", "project-orbit-skill-claude"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.agents...)
			if err != nil {
				t.Fatal(err)
			}
			var paths, ids []string
			for _, item := range got {
				paths = append(paths, item.Path)
				ids = append(ids, item.ResourceID)
			}
			if !reflect.DeepEqual(paths, tt.paths) || !reflect.DeepEqual(ids, tt.ids) {
				t.Fatalf("got %v / %v", paths, ids)
			}
			for _, item := range got {
				if item.Source != "orbit-skill-install" || !bytes.Equal(item.Content, skills.Orbit()) {
					t.Errorf("unexpected update metadata for %s", item.Path)
				}
			}
		})
	}
}

func TestResolveRejectsMissingAndUnknownAgents(t *testing.T) {
	for _, agents := range [][]Agent{nil, {"unknown"}} {
		if _, err := Resolve(agents...); err == nil {
			t.Errorf("Resolve(%v) succeeded", agents)
		}
	}
}

func TestOrbitSkillAssetMatchesCanonicalFileAndReturnsCopy(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "skills", "orbit", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	first := skills.Orbit()
	if !bytes.Equal(first, canonical) {
		t.Fatal("embedded skill differs from canonical file")
	}
	first[0] ^= 0xff
	if bytes.Equal(first, skills.Orbit()) {
		t.Fatal("Orbit returned shared mutable asset bytes")
	}
}
