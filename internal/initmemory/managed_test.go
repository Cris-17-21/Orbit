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

func TestUpdateManagedConfigEntriesMergesAndRegistersOnlyTheEntry(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, ".mcp.json")
	before := []byte(`{"$schema":"https://example.test/schema.json","mcpServers":{"other":{"command":"keep-me"}}}`)
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}

	entry := []byte(`{"command":"npx","args":["atomicmemory-mcp"]}`)
	update := ManagedJSONEntryUpdate{
		Path: ".mcp.json", Section: "mcpServers", Key: "atomicmemory",
		ResourceID: "project-claude-atomic-memory-config", Source: "orbit-memory-configure",
		Content: entry,
	}
	if err := UpdateManagedFilesWithJSONEntries(root, nil, []ManagedJSONEntryUpdate{update}); err != nil {
		t.Fatalf("UpdateManagedFilesWithJSONEntries() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(got, &config); err != nil {
		t.Fatalf("updated .mcp.json is invalid: %v", err)
	}
	if string(config["$schema"]) != `"https://example.test/schema.json"` {
		t.Errorf("$schema was not preserved: %s", config["$schema"])
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(config["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	if string(canonicalJSON(t, servers["other"])) != `{"command":"keep-me"}` {
		t.Errorf("unrelated MCP server changed: %s", servers["other"])
	}
	var installed map[string]json.RawMessage
	if err := json.Unmarshal(servers["atomicmemory"], &installed); err != nil {
		t.Fatal(err)
	}
	if string(installed["command"]) != `"npx"` {
		t.Errorf("Atomic Memory command = %s, want npx", installed["command"])
	}

	manifestBytes, err := os.ReadFile(filepath.Join(root, ".orbit", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, resource := range manifest.Resources {
		if resource.Target == ".mcp.json#/mcpServers/atomicmemory" {
			found = true
			if resource.Kind != "config-entry" || resource.ResourceID != update.ResourceID {
				t.Errorf("registered resource = %#v", resource)
			}
			if resource.InstalledDigest != digest(canonicalJSON(t, servers["atomicmemory"])) {
				t.Errorf("registered digest does not match the installed entry")
			}
		}
	}
	if !found {
		t.Fatal("manifest does not register the Atomic Memory config entry")
	}
	if _, err := BuildPlan(root); err != nil {
		t.Fatalf("BuildPlan() rejected manifest after adding a config-entry resource: %v", err)
	}
}

func TestUpdateManagedConfigEntriesRejectsUnownedNameCollision(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, ".mcp.json")
	before := []byte(`{"mcpServers":{"atomicmemory":{"command":"user-server"},"other":{"command":"keep-me"}}}`)
	if err := os.WriteFile(path, before, 0o644); err != nil {
		t.Fatal(err)
	}
	err = UpdateManagedFilesWithJSONEntries(root, nil, []ManagedJSONEntryUpdate{{
		Path: ".mcp.json", Section: "mcpServers", Key: "atomicmemory",
		ResourceID: "project-claude-atomic-memory-config", Source: "orbit-memory-configure",
		Content: []byte(`{"command":"npx"}`),
	}})
	if err == nil {
		t.Fatal("UpdateManagedFilesWithJSONEntries() replaced an unowned server")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(before) {
		t.Fatalf("conflicting .mcp.json changed: got %s, want %s", got, before)
	}
}

func TestUpdateManagedConfigEntriesUpdatesOwnedKeyAndPreservesConcurrentUnrelatedValues(t *testing.T) {
	root := makeGitRepo(t)
	initial, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(initial); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".mcp.json")
	update := ManagedJSONEntryUpdate{
		Path: ".mcp.json", Section: "mcpServers", Key: "atomicmemory",
		ResourceID: "project-claude-atomic-memory-config", Source: "orbit-memory-configure",
		Content: []byte(`{"command":"npx","args":["v1"]}`),
	}
	if err := UpdateManagedFilesWithJSONEntries(root, nil, []ManagedJSONEntryUpdate{update}); err != nil {
		t.Fatal(err)
	}

	// A user may add another server after Orbit's install. Updating Orbit's
	// entry must retain that unrelated change.
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"atomicmemory":{"command":"npx","args":["v1"]},"new-server":{"command":"preserve"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	update.Content = []byte(`{"command":"npx","args":["v2"]}`)
	if err := UpdateManagedFilesWithJSONEntries(root, nil, []ManagedJSONEntryUpdate{update}); err != nil {
		t.Fatalf("updating Orbit entry after unrelated edit: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(got, &config); err != nil {
		t.Fatal(err)
	}
	if string(config.MCPServers["new-server"]["command"]) != `"preserve"` {
		t.Errorf("unrelated user server was lost: %s", got)
	}
	if string(canonicalJSON(t, config.MCPServers["atomicmemory"]["args"])) != `["v2"]` {
		t.Errorf("Orbit entry was not updated: %s", got)
	}
}

func canonicalJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
