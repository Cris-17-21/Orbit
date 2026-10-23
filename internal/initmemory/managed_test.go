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

func TestManagedFilePlanCreatesManifestBeforeOrbitInit(t *testing.T) {
	root := makeGitRepo(t)
	update := ManagedFileUpdate{
		Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-orchestrator-skill",
		Source: "orbit-skill-install", Content: []byte("---\nname: orbit\n---\n"),
	}
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{update})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Status != StatusCreate {
		t.Fatalf("plan entries = %#v, want one create", plan.Entries)
	}
	result, err := ApplyManagedFiles(plan)
	if err != nil {
		t.Fatalf("ApplyManagedFiles() error = %v", err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("created entries = %d, want 1", len(result.Created))
	}

	manifestPath := filepath.Join(root, ".orbit", "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest created by skill install: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 1 || manifest.Resources[0].InstalledDigest != digest(update.Content) {
		t.Fatalf("manifest resources = %#v, want the installed skill", manifest.Resources)
	}

	initPlan, err := BuildPlan(root)
	if err != nil {
		t.Fatalf("BuildPlan() after skill install: %v", err)
	}
	if _, err := Apply(initPlan); err != nil {
		t.Fatalf("Apply() after skill install: %v", err)
	}
	manifestBytes, err = os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 5 {
		t.Fatalf("manifest resources after orbit init = %d, want 5 (skill plus four init resources)", len(manifest.Resources))
	}
}

func TestManagedFilePlanSkipsUnownedConflictAndAppliesIndependentTarget(t *testing.T) {
	root := makeGitRepo(t)
	conflictPath := filepath.Join(root, ".claude", "skills", "orbit", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
		t.Fatal(err)
	}
	wantConflict := []byte("personal skill\n")
	if err := os.WriteFile(conflictPath, wantConflict, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{
		{Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")},
		{Path: ".claude/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-claude", Source: "orbit-skill-install", Content: []byte("orbit skill\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Entries) != 2 || plan.Entries[0].Status != StatusCreate || plan.Entries[1].Status != StatusConflict {
		t.Fatalf("plan entries = %#v, want create then conflict", plan.Entries)
	}
	result, err := ApplyManagedFiles(plan)
	if err != nil {
		t.Fatalf("ApplyManagedFiles() error = %v", err)
	}
	if len(result.Created) != 1 || len(result.Conflicts) != 1 {
		t.Fatalf("result = %#v, want one installed file and one conflict", result)
	}
	gotConflict, err := os.ReadFile(conflictPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotConflict) != string(wantConflict) {
		t.Fatalf("conflicting skill changed: got %q, want %q", gotConflict, wantConflict)
	}
}

func TestManagedFilePlanRecognizesOwnedTargetAndPreservesEditedTarget(t *testing.T) {
	root := makeGitRepo(t)
	path := ".agents/skills/orbit/SKILL.md"
	update := ManagedFileUpdate{Path: path, ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{update})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyManagedFiles(plan); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanManagedFiles(root, []ManagedFileUpdate{update})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].Status != StatusPresent {
		t.Fatalf("unchanged owned status = %q, want %q", plan.Entries[0].Status, StatusPresent)
	}

	edited := []byte("collaborator edit\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), edited, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanManagedFiles(root, []ManagedFileUpdate{update})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].Status != StatusModified {
		t.Fatalf("edited owned status = %q, want %q", plan.Entries[0].Status, StatusModified)
	}
	result, err := ApplyManagedFiles(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(result.Conflicts))
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(edited) {
		t.Fatalf("edited target changed: got %q, want %q", got, edited)
	}
}

func TestApplyManagedFilesRejectsTargetCreatedAfterPlan(t *testing.T) {
	root := makeGitRepo(t)
	path := ".agents/skills/orbit/SKILL.md"
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{{Path: path, ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}})
	if err != nil {
		t.Fatal(err)
	}
	created := []byte("created after preview\n")
	absolutePath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolutePath, created, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyManagedFiles(plan); err == nil {
		t.Fatal("ApplyManagedFiles() succeeded after the planned target appeared")
	}
	got, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(created) {
		t.Fatalf("raced target changed: got %q, want %q", got, created)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("manifest was unexpectedly created after a stale plan: err = %v", err)
	}
}

func TestApplyManagedFilesRejectsManifestChangedAfterPlan(t *testing.T) {
	root := makeGitRepo(t)
	manifest := newManifest(root)
	manifestPath := filepath.Join(root, ".orbit", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{{Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}})
	if err != nil {
		t.Fatal(err)
	}
	manifest.OrbitVersion = "changed-after-preview"
	if err := writeManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyManagedFiles(plan); err == nil {
		t.Fatal("ApplyManagedFiles() succeeded after the manifest changed")
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("skill was written after the manifest changed: err = %v", err)
	}
}

func TestApplyManagedFilesRejectsPresentTargetChangedAfterPlan(t *testing.T) {
	root := makeGitRepo(t)
	present := ManagedFileUpdate{Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{present})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyManagedFiles(plan); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanManagedFiles(root, []ManagedFileUpdate{
		present,
		{Path: ".claude/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-claude", Source: "orbit-skill-install", Content: []byte("orbit skill\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Entries[0].Status != StatusPresent || plan.Entries[1].Status != StatusCreate {
		t.Fatalf("plan entries = %#v, want present then create", plan.Entries)
	}
	edited := []byte("edited after preview\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(present.Path)), edited, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyManagedFiles(plan); err == nil {
		t.Fatal("ApplyManagedFiles() succeeded after a present target changed")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "orbit", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("independent target was written after a stale plan: err = %v", err)
	}
}

func TestApplyManagedFilesRejectsMutatedPlanEntries(t *testing.T) {
	root := makeGitRepo(t)
	plan, err := PlanManagedFiles(root, []ManagedFileUpdate{{Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}})
	if err != nil {
		t.Fatal(err)
	}
	plan.Entries[0].Status = StatusPresent
	if _, err := ApplyManagedFiles(plan); err == nil {
		t.Fatal("ApplyManagedFiles() accepted an edited plan")
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("manifest was unexpectedly created for an edited plan: err = %v", err)
	}
}

func TestPlanManagedFilesRejectsInvalidManifestWithoutWriting(t *testing.T) {
	root := makeGitRepo(t)
	manifestPath := filepath.Join(root, ".orbit", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := PlanManagedFiles(root, []ManagedFileUpdate{{Path: ".agents/skills/orbit/SKILL.md", ResourceID: "project-orbit-skill-common", Source: "orbit-skill-install", Content: []byte("orbit skill\n")}})
	if err == nil {
		t.Fatal("PlanManagedFiles() accepted an invalid manifest")
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("planning wrote the skill: err = %v", err)
	}
}

func TestPlanManagedFilesRejectsUnsafeAndReservedTargets(t *testing.T) {
	root := makeGitRepo(t)
	for _, path := range []string{"../outside.txt", ".orbit/manifest.json", ".orbit/init.lock"} {
		t.Run(path, func(t *testing.T) {
			_, err := PlanManagedFiles(root, []ManagedFileUpdate{{Path: path, ResourceID: "resource", Source: "test", Content: []byte("content")}})
			if err == nil {
				t.Fatalf("PlanManagedFiles() accepted unsafe/reserved target %q", path)
			}
		})
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
