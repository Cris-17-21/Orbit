package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cris-17-21/Orbit/internal/atomicmemory"
	"github.com/Cris-17-21/Orbit/internal/initmemory"
	tea "github.com/charmbracelet/bubbletea"
)

func createRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "--quiet", root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initialize temporary Git repository: %v\n%s", err, output)
	}
	return root
}

func TestInitYesCreatesFilesAndReportsSummary(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Creados: 4") {
		t.Fatalf("summary does not report created files: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project-context", "business-rules.md")); err != nil {
		t.Fatalf("memory file was not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "atomic-memory.json")); err != nil {
		t.Fatalf("Atomic Memory project config was not created: %v", err)
	}
}

func TestInitRequiresGitRepository(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil)
	if code == 0 {
		t.Fatal("execute() succeeded outside a Git repository")
	}
	if !strings.Contains(stderr.String(), "git init") {
		t.Fatalf("error does not explain the Git requirement: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("memory files were written outside Git: err = %v", err)
	}
}

func TestInitRequiresExplicitConfirmationWithoutInteractiveTerminal(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"init"}, root, &stdout, &stderr, false, nil)
	if code == 0 {
		t.Fatal("execute() succeeded without interactive input or --yes")
	}
	if !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("error does not explain non-interactive confirmation: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("memory files were written without confirmation: err = %v", err)
	}
}

func TestInitCancellationDoesNotWriteFiles(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	prompt := func(_ initmemory.Plan) (bool, error) {
		return false, nil
	}
	code := execute([]string{"init"}, root, &stdout, &stderr, true, prompt)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Cancelado") {
		t.Fatalf("cancellation was not reported: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("memory files were written after cancellation: err = %v", err)
	}
}

func TestConfirmationModelAcceptsSpanishYesKey(t *testing.T) {
	model := confirmationModel{}
	updated, _ := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'s'}}))
	model = updated.(confirmationModel)
	if !model.done || !model.confirmed {
		t.Fatal("pressing s did not confirm the plan")
	}
}

func TestInitConfirmationAcceptsPlan(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	confirmed := false
	prompt := func(plan initmemory.Plan) (bool, error) {
		confirmed = len(plan.Entries) == 4
		return true, nil
	}
	code := execute([]string{"init"}, root, &stdout, &stderr, true, prompt)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !confirmed {
		t.Fatal("confirmation callback did not receive the full plan")
	}
	if _, err := os.Stat(filepath.Join(root, "project-context", "business-rules.md")); err != nil {
		t.Fatalf("accepted plan was not applied: %v", err)
	}
}

func TestInitPreviewIncludesAtomicMemoryConfigWithoutWritingIt(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"init"}, root, &stdout, &stderr, false, nil)
	if code == 0 {
		t.Fatal("execute() succeeded without confirmation")
	}
	if !strings.Contains(stdout.String(), ".orbit/atomic-memory.json") {
		t.Fatalf("preview omits Atomic Memory config: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "atomic-memory.json")); !os.IsNotExist(err) {
		t.Fatalf("preview wrote Atomic Memory config before confirmation: err = %v", err)
	}
}

func TestConfirmationModelDefaultsToNoAndRequiresExplicitChoice(t *testing.T) {
	model := confirmationModel{}
	updated, _ := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	model = updated.(confirmationModel)
	if !model.done {
		t.Fatal("Enter did not finish the default confirmation choice")
	}
	if model.confirmed {
		t.Fatal("confirmation defaulted to yes")
	}

	model = confirmationModel{}
	updated, _ = model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	model = updated.(confirmationModel)
	updated, _ = model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	model = updated.(confirmationModel)
	if !model.done || !model.confirmed {
		t.Fatal("selecting Yes and pressing Enter did not confirm the plan")
	}
}

func initializeMemory(t *testing.T, root string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("orbit init failed: %s", stderr.String())
	}
}

func TestConfigureCodexUpdatesProjectConfigWithoutDuplicatingExistingServer(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	lister := func() ([]string, error) { return []string{"atomicmemory", "other"}, nil }
	code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no agregará otro") {
		t.Fatalf("existing server detection was not reported: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("duplicate project MCP config was written: err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"mode": "personal"`) || !strings.Contains(string(data), `"backend": "hosted-cloud"`) {
		t.Fatalf("project config was not enabled: %s", data)
	}
}

func TestConfigureCodexPreservesExistingBackendWhenFlagIsOmitted(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	config := atomicmemory.Config{SchemaVersion: atomicmemory.ConfigSchemaVersion, Mode: atomicmemory.ModePersonal, Backend: atomicmemory.BackendSelfHosted, APIURL: "https://memory.example.com"}
	configBytes, err := config.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err := initmemory.UpdateManagedFiles(root, []initmemory.ManagedFileUpdate{{
		Path: ".orbit/atomic-memory.json", ResourceID: "project-atomic-memory-config", Source: "test", Content: configBytes,
	}}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	lister := func() ([]string, error) { return []string{"atomicmemory"}, nil }
	code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	updated, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := atomicmemory.ParseConfig(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got.Backend != config.Backend || got.APIURL != config.APIURL {
		t.Fatalf("backend changed without an explicit flag: %#v", got)
	}
}

func TestConfigureCodexAddsProjectConfigWithLocalEnvironmentReferences(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	lister := func() ([]string, error) { return []string{"other"}, nil }
	code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"[mcp_servers.atomicmemory]", "@atomicmemory/mcp-server@0.1.6", "ATOMICMEMORY_API_KEY", "ATOMICMEMORY_SCOPE_USER", "env_vars"} {
		if !strings.Contains(string(data), expected) {
			t.Errorf("Codex config omits %q: %s", expected, data)
		}
	}
	for _, forbidden := range []string{"private-key", "private-user", "user_"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("Codex config contains sensitive value %q", forbidden)
		}
	}
	if !strings.Contains(stdout.String(), "confianza") {
		t.Errorf("did not mention Codex project trust: %s", stdout.String())
	}
}

func TestConfigureCodexCancellationDoesNotWrite(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	prompt := func(plan initmemory.Plan) (bool, error) {
		if len(plan.Entries) != 2 {
			t.Fatalf("confirmation received %d plan entries, want 2", len(plan.Entries))
		}
		return false, nil
	}
	lister := func() ([]string, error) { return nil, nil }
	code := executeWithCodexLister([]string{"memory", "configure", "codex"}, root, &stdout, &stderr, true, prompt, lister)
	if code != 0 || !strings.Contains(stdout.String(), "Cancelado") {
		t.Fatalf("execute() = %d, stdout %s, stderr %s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("Codex config was written after cancellation: err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"mode": "personal"`) {
		t.Fatal("Atomic Memory project config changed after cancellation")
	}
}

func TestConfigureCodexFailsClosedWhenCodexCannotBeInspected(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	lister := func() ([]string, error) { return nil, errCodexList }
	code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister)
	if code == 0 || !strings.Contains(stderr.String(), "no se aplicaron cambios") {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("configuration changed after Codex inspection failed: err = %v", err)
	}
}

func TestConfigureCodexPreservesUnownedProjectConfig(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	path := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("# personal settings\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	lister := func() ([]string, error) { return nil, nil }
	code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister)
	if code == 0 {
		t.Fatal("configure succeeded despite an unowned config conflict")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("unowned Codex config changed: got %q, want %q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"mode": "personal"`) {
		t.Fatal("Atomic Memory config was partially updated despite Codex conflict")
	}
}

func TestConfigureClaudeMergesProjectMCPConfigAndKeepsOtherServers(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	mcpPath := filepath.Join(root, ".mcp.json")
	before := []byte(`{"$schema":"https://example.test/schema.json","mcpServers":{"other":{"command":"keep-me"}}}`)
	if err := os.WriteFile(mcpPath, before, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Schema     string `json:"$schema"`
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid .mcp.json: %v", err)
	}
	if config.Schema != "https://example.test/schema.json" {
		t.Errorf("$schema = %q, want preserved schema", config.Schema)
	}
	if got := config.MCPServers["other"].Command; got != "keep-me" {
		t.Errorf("unrelated server command = %q, want keep-me", got)
	}
	server, found := config.MCPServers["atomicmemory"]
	if !found {
		t.Fatal("Atomic Memory MCP entry was not installed")
	}
	if server.Command != "npx" || strings.Join(server.Args, " ") != "-y --package=@atomicmemory/mcp-server@0.1.6 atomicmemory-mcp" {
		t.Errorf("unexpected Atomic Memory command: %#v", server)
	}
	if server.Env["ATOMICMEMORY_API_KEY"] != "${ATOMICMEMORY_API_KEY}" || server.Env["ATOMICMEMORY_SCOPE_USER"] != "${ATOMICMEMORY_SCOPE_USER}" {
		t.Errorf("local credential/identity references missing: %#v", server.Env)
	}
	for _, forbidden := range []string{"amc_", "user_", "personal@example.com"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf(".mcp.json contains sensitive value %q", forbidden)
		}
	}
	if !strings.Contains(stdout.String(), "aprobar") {
		t.Errorf("approval requirement was not mentioned: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "no se copiaron") {
		t.Errorf("credential handling was not reported: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "protocolo de memoria de Orbit se agregará") {
		t.Errorf("MCP-only scope was not explained: %s", stdout.String())
	}
}

func TestConfigureClaudePreservesUnownedAtomicMemoryEntry(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	mcpPath := filepath.Join(root, ".mcp.json")
	want := []byte(`{"mcpServers":{"atomicmemory":{"command":"user-owned"},"other":{"command":"keep-me"}}}`)
	if err := os.WriteFile(mcpPath, want, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil)
	if code == 0 || !strings.Contains(stderr.String(), "no está administrada por Orbit") {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	got, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("unowned MCP entry changed: got %s, want %s", got, want)
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"mode": "personal"`) {
		t.Fatal("Atomic Memory config was partially updated despite the conflict")
	}
}

func TestConfigureClaudeCancellationDoesNotWrite(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	prompt := func(plan initmemory.Plan) (bool, error) {
		if len(plan.Entries) != 2 {
			t.Fatalf("confirmation received %d plan entries, want 2", len(plan.Entries))
		}
		return false, nil
	}
	code := execute([]string{"memory", "configure", "claude"}, root, &stdout, &stderr, true, prompt)
	if code != 0 || !strings.Contains(stdout.String(), "Cancelado") {
		t.Fatalf("execute() = %d, stdout %s, stderr %s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatalf(".mcp.json was written after cancellation: err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"mode": "personal"`) {
		t.Fatal("Atomic Memory config changed after cancellation")
	}
}

func TestConfigureClaudeRejectsStalePreviewWithoutPartialWrite(t *testing.T) {
	root := createRepo(t)
	initializeMemory(t, root)
	var stdout, stderr bytes.Buffer
	confirm := func(plan initmemory.Plan) (bool, error) {
		if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(`{"mcpServers":{"other":{"command":"new"}}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		return true, nil
	}
	code := execute([]string{"memory", "configure", "claude"}, root, &stdout, &stderr, true, confirm)
	if code == 0 || !strings.Contains(stderr.String(), "after the preview") {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, ".orbit", "atomic-memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"mode": "personal"`) {
		t.Fatal("Atomic Memory config changed after stale preview")
	}
}

func TestConfigureClaudeHelpShowsUsageOnce(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"memory", "configure", "claude", "--help"}, t.TempDir(), &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("execute() = %d", code)
	}
	if strings.Count(stderr.String(), "Uso: orbit memory configure") != 1 {
		t.Fatalf("usage was not printed exactly once to stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("help was duplicated to stdout: %q", stdout.String())
	}
}

var errCodexList = errors.New("Codex list unavailable")
