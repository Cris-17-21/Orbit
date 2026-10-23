package initmemory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateManagedFilesUpdatesOwnedFileAndRegistersNewFile(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}

	atomicMemoryConfig := []byte("{\"schema_version\":1,\"mode\":\"personal\",\"backend\":\"hosted-cloud\",\"api_url\":\"https://api.atomicstrata.ai\"}\n")
	codexConfig := []byte("[mcp_servers.atomicmemory]\ncommand = \"npx\"\n")
	if err := UpdateManagedFiles(root, []ManagedFileUpdate{
		{Path: ".orbit/atomic-memory.json", ResourceID: "project-atomic-memory-config", Source: "orbit-memory-configure", Content: atomicMemoryConfig},
		{Path: ".codex/config.toml", ResourceID: "project-codex-atomic-memory-config", Source: "orbit-memory-configure", Content: codexConfig},
	}); err != nil {
		t.Fatalf("UpdateManagedFiles() error = %v", err)
	}

	for path, want := range map[string][]byte{
		".orbit/atomic-memory.json": atomicMemoryConfig,
		".codex/config.toml":        codexConfig,
	} {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}

	manifestBytes, err := os.ReadFile(filepath.Join(root, ".orbit", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 5 {
		t.Fatalf("manifest resources = %d, want 5", len(manifest.Resources))
	}
	for _, resource := range manifest.Resources {
		if resource.Target == ".orbit/atomic-memory.json" && resource.InstalledDigest != digest(atomicMemoryConfig) {
			t.Errorf("Atomic Memory config digest = %q, want updated digest", resource.InstalledDigest)
		}
		if resource.Target == ".codex/config.toml" {
			if resource.ResourceID != "project-codex-atomic-memory-config" || resource.InstalledDigest != digest(codexConfig) {
				t.Errorf("Codex config manifest resource = %#v", resource)
			}
		}
	}
}

func TestUpdateManagedFilesValidatesAllTargetsBeforeWriting(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}

	atomicPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	if err := os.WriteFile(atomicPath, []byte("user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err = UpdateManagedFiles(root, []ManagedFileUpdate{
		{Path: ".orbit/atomic-memory.json", ResourceID: "project-atomic-memory-config", Source: "orbit-memory-configure", Content: []byte("new config\n")},
		{Path: ".codex/config.toml", ResourceID: "project-codex-atomic-memory-config", Source: "orbit-memory-configure", Content: []byte("new host config\n")},
	})
	if err == nil {
		t.Fatal("UpdateManagedFiles() succeeded after a managed file was edited")
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("new Codex config was written despite the conflict: err = %v", err)
	}
}

func TestUpdateManagedFilesPreservesUnownedExistingTarget(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("# personal Codex settings\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	err = UpdateManagedFiles(root, []ManagedFileUpdate{{
		Path: ".codex/config.toml", ResourceID: "project-codex-atomic-memory-config", Source: "orbit-memory-configure", Content: []byte("Orbit config\n"),
	}})
	if err == nil {
		t.Fatal("UpdateManagedFiles() overwrote a target not owned by Orbit")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("unowned Codex config changed: got %q, want %q", got, want)
	}
}
