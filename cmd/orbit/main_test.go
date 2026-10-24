package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

func TestMemoryStatusReportsMarkdownOnlyWhenAtomicMemoryIsNotConfigured(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = execute([]string{"memory", "status"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("status: code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{"Modalidad: desactivada", "Codex: no configurado", "Claude Code: no configurado", "Conectividad: no verificada", "Memoria compartida: no verificada"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("status omitted %q: %s", expected, stdout.String())
		}
	}
}

func TestMemoryStatusReportsConfiguredHostsWithoutLeakingSecrets(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	lister := func() ([]string, error) { return []string{"other"}, nil }
	if code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister); code != 0 {
		t.Fatalf("configure Codex: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("configure Claude Code: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "status"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("status: code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{"Modalidad: personal", "Codex: configurado", "Claude Code: configurado", "Conectividad: no verificada", "Memoria compartida: no verificada"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("status omitted %q: %s", expected, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "ATOMICMEMORY_API_KEY") || strings.Contains(stdout.String(), "ATOMICMEMORY_SCOPE_USER") {
		t.Fatalf("status leaked credential configuration: %s", stdout.String())
	}
}

func TestMemoryDisableRemovesOnlyOrbitConfigurationAndKeepsMarkdownAndOtherClaudeServers(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	lister := func() ([]string, error) { return []string{"other"}, nil }
	if code := executeWithCodexLister([]string{"memory", "configure", "codex", "--yes"}, root, &stdout, &stderr, false, nil, lister); code != 0 {
		t.Fatalf("configure Codex: code=%d stderr=%s", code, stderr.String())
	}
	if code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("configure Claude Code: code=%d stderr=%s", code, stderr.String())
	}
	claudePath := filepath.Join(root, ".mcp.json")
	claudeBytes, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	var claudeConfig map[string]any
	if err := json.Unmarshal(claudeBytes, &claudeConfig); err != nil {
		t.Fatal(err)
	}
	claudeConfig["mcpServers"].(map[string]any)["team-tool"] = map[string]any{"command": "team-tool"}
	claudeBytes, _ = json.MarshalIndent(claudeConfig, "", "  ")
	claudeBytes = append(claudeBytes, '\n')
	if err := os.WriteFile(claudePath, claudeBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "disable", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("disable Atomic Memory: code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "atomic-memory.json")); !os.IsNotExist(err) {
		t.Fatalf("Orbit's Atomic Memory config was not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("Orbit's Codex config was not removed: %v", err)
	}
	updatedClaude, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(updatedClaude, []byte("atomicmemory")) || !bytes.Contains(updatedClaude, []byte("team-tool")) {
		t.Fatalf("disable did not remove only Orbit's Claude entry: %s", updatedClaude)
	}
	if _, err := os.Stat(filepath.Join(root, "project-context", "business-rules.md")); err != nil {
		t.Fatalf("Markdown memory was removed: %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, ".orbit", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest initmemory.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, resource := range manifest.Resources {
		if strings.Contains(resource.Target, "atomic-memory") || strings.Contains(resource.Target, "atomicmemory") {
			t.Fatalf("removed integration remains in manifest: %+v", resource)
		}
	}
}

func TestMemoryDisablePreservesModifiedOrbitFilesAndRequiresConfirmation(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	if code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("configure Claude Code: code=%d stderr=%s", code, stderr.String())
	}
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	if err := os.WriteFile(configPath, []byte("{\"mode\":\"personal\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "disable"}, root, &stdout, &stderr, false, nil); code != 2 {
		t.Fatalf("unconfirmed noninteractive disable returned %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "requiere confirmación") {
		t.Fatalf("missing confirmation explanation: %s", stderr.String())
	}
	accept := func(plan initmemory.Plan) (bool, error) {
		if len(plan.Entries) != 1 || !strings.HasPrefix(plan.Entries[0].Path, ".mcp.json") {
			t.Fatalf("confirmation did not show exactly the removal: %+v", plan.Entries)
		}
		return false, nil
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "disable"}, root, &stdout, &stderr, true, accept); code != 0 {
		t.Fatalf("cancelled disable: code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("cancelled disable changed modified config: %v", err)
	}
	claudeBytes, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil || !bytes.Contains(claudeBytes, []byte("atomicmemory")) {
		t.Fatalf("cancelled disable removed Claude configuration: err=%v config=%s", err, claudeBytes)
	}
}

func TestMemoryDisablePreservesModifiedClaudeEntry(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	if code := execute([]string{"memory", "configure", "claude", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("configure Claude Code: code=%d stderr=%s", code, stderr.String())
	}
	configPath := filepath.Join(root, ".mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	server := config["mcpServers"].(map[string]any)["atomicmemory"].(map[string]any)
	server["args"] = []any{"user-customized"}
	data, _ = json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := execute([]string{"memory", "disable", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("disable: code=%d stderr=%s", code, stderr.String())
	}
	updated, err := os.ReadFile(configPath)
	if err != nil || !bytes.Contains(updated, []byte("user-customized")) {
		t.Fatalf("modified Claude entry was not preserved: err=%v config=%s", err, updated)
	}
	if !strings.Contains(stdout.String(), "se conservará") {
		t.Fatalf("preview did not explain preserving the modified Claude entry: %s", stdout.String())
	}
}

func TestMemoryDisableRejectsStalePreview(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	plan, err := initmemory.PlanMemoryDisable(root)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	if err := os.WriteFile(configPath, []byte("user edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initmemory.ApplyMemoryDisable(plan); err == nil || !strings.Contains(err.Error(), "changed after the preview") {
		t.Fatalf("stale disable plan returned %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("stale plan removed a changed file: %v", err)
	}
}

func TestDoctorReportsHostEvidenceAndNeverPrintsCredentialValues(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	if code := execute([]string{"init", "--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("initialize project: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runMemoryConfigureClaude([]string{"--yes"}, root, &stdout, &stderr, false, nil); code != 0 {
		t.Fatalf("configure Atomic Memory: code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	deps := doctorDependencies{
		lookPath:   func(name string) (string, error) { return name, nil },
		listCodex:  func() ([]string, error) { return []string{"atomicmemory"}, nil },
		listClaude: func() (string, error) { return "atomicmemory: ✘ Failed to connect", nil },
		getenv: func(name string) string {
			if name == "ATOMICMEMORY_API_KEY" {
				return "do-not-print-this-key"
			}
			return "do-not-print-this-user"
		},
	}
	if code := runDoctor(root, &stdout, &stderr, deps); code != 0 {
		t.Fatalf("doctor: code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{"Atomic Memory: modalidad personal", "Codex: servidor registrado; conexión no verificada", "Claude Code: fallo al conectar (según `claude mcp list`)", "ATOMICMEMORY_API_KEY: definida", "ATOMICMEMORY_SCOPE_USER: definida", "Memoria compartida: no verificada"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("doctor omitted %q: %s", expected, stdout.String())
		}
	}
	for _, secret := range []string{"do-not-print-this-key", "do-not-print-this-user"} {
		if strings.Contains(stdout.String()+stderr.String(), secret) {
			t.Fatalf("doctor leaked a credential value: %s", stdout.String()+stderr.String())
		}
	}
}

func TestClassifyClaudeAtomicMemoryStatus(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "pending approval", output: "atomicmemory: Pending approval", want: "pendiente de aprobación"},
		{name: "authentication", output: "atomicmemory: Needs authentication", want: "requiere autenticación"},
		{name: "failed", output: "atomicmemory: Failed to connect", want: "fallo al conectar"},
		{name: "connected", output: "atomicmemory: Connected", want: "estado conectado reportado"},
		{name: "absent", output: "other: Connected", want: "no aparece"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyClaudeAtomicMemoryStatus(test.output); !strings.Contains(got, test.want) {
				t.Fatalf("classifyClaudeAtomicMemoryStatus() = %q, want substring %q", got, test.want)
			}
		})
	}
}

func TestSkillInstallRequiresSelectionAndExplicitYesWhenNonInteractive(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"skill", "install", "--codex"}, root, &stdout, &stderr, false, nil)
	if code == 0 || !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("expected non-interactive confirmation requirement; code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("skill was written without confirmation: %v", err)
	}
}

func TestSkillInstallDryRunShowsPlanWithoutWriting(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"skill", "install", "--claude", "--dry-run"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".claude/skills/orbit/SKILL.md") {
		t.Fatalf("plan omitted Claude destination: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote files: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote manifest: %v", err)
	}
}

func TestSkillInstallYesAppliesSelectedAgentsAndManifest(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	code := execute([]string{"skill", "install", "--codex", "--antigravity", "--yes"}, root, &stdout, &stderr, false, nil)
	if code != 0 {
		t.Fatalf("execute() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Codex y Antigravity") {
		t.Fatalf("shared destination explanation missing: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")); err != nil {
		t.Fatalf("shared skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("unselected Claude destination was written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "manifest.json")); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
}

func TestSkillInstallAndInitWorkInEitherOrder(t *testing.T) {
	for _, initFirst := range []bool{true, false} {
		name := "skill-first"
		if initFirst {
			name = "init-first"
		}
		t.Run(name, func(t *testing.T) {
			root := createRepo(t)
			var stdout, stderr bytes.Buffer
			run := func(args ...string) {
				t.Helper()
				stdout.Reset()
				stderr.Reset()
				if code := execute(args, root, &stdout, &stderr, false, nil); code != 0 {
					t.Fatalf("execute(%v) = %d; stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
				}
			}
			if initFirst {
				run("init", "--yes")
			}
			run("skill", "install", "--codex", "--yes")
			if !initFirst {
				run("init", "--yes")
			}
			skillPath := filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")
			originalSkill, err := os.ReadFile(skillPath)
			if err != nil {
				t.Fatal(err)
			}
			run("init", "--yes")
			finalSkill, err := os.ReadFile(skillPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(originalSkill, finalSkill) {
				t.Fatal("orbit init changed the installed skill")
			}
			manifestBytes, err := os.ReadFile(filepath.Join(root, ".orbit", "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var manifest initmemory.Manifest
			if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
				t.Fatal(err)
			}
			if len(manifest.Resources) != 5 {
				t.Fatalf("manifest has %d resources, want 5 (four init resources plus skill)", len(manifest.Resources))
			}
		})
	}
}

func TestSkillInstallConflictPreservesTargetAndInstallsIndependentDestination(t *testing.T) {
	root := createRepo(t)
	conflictPath := filepath.Join(root, ".agents", "skills", "orbit", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("user-owned skill\n")
	if err := os.WriteFile(conflictPath, want, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := execute([]string{"skill", "install", "--codex", "--claude", "--yes"}, root, &stdout, &stderr, false, nil)
	if code == 0 || !strings.Contains(stdout.String(), "conflictos: 1") {
		t.Fatalf("expected reported conflict; code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(conflictPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("user-owned skill changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "orbit", "SKILL.md")); err != nil {
		t.Fatalf("independent Claude destination was not installed: %v", err)
	}
}

func TestSkillInstallRejectsAmbiguousFlags(t *testing.T) {
	root := createRepo(t)
	for _, args := range [][]string{{"skill", "install", "--dry-run", "--yes", "--codex"}, {"skill", "install", "--codex", "--agent", "claude"}, {"skill", "install"}} {
		var stdout, stderr bytes.Buffer
		if code := execute(args, root, &stdout, &stderr, false, nil); code == 0 {
			t.Errorf("execute(%v) unexpectedly succeeded", args)
		}
	}
}

func TestSkillInstallCancellationDoesNotWrite(t *testing.T) {
	root := createRepo(t)
	var stdout, stderr bytes.Buffer
	confirm := func(_ initmemory.ManagedFilesPlan, _ io.Reader, _ io.Writer) (bool, error) { return false, nil }
	code := runSkillInstallWithConfirmation([]string{"--claude"}, root, &stdout, &stderr, true, confirm)
	if code != 0 || !strings.Contains(stdout.String(), "Cancelado") {
		t.Fatalf("expected clean cancellation; code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf("cancelled install wrote destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit")); !os.IsNotExist(err) {
		t.Fatalf("cancelled install wrote manifest: %v", err)
	}
}

func TestSkillAgentSelectorRequiresExplicitChoice(t *testing.T) {
	model := skillAgentSelector{selected: make(map[int]bool)}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	selection := updated.(skillAgentSelector)
	if !selection.done || len(selection.selected) != 0 {
		t.Fatalf("empty selector result should remain unselected: %#v", selection)
	}
	model = skillAgentSelector{selected: make(map[int]bool)}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = updated.(skillAgentSelector)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	selection = updated.(skillAgentSelector)
	if !selection.done || !selection.selected[0] || selection.selected[1] || selection.selected[2] {
		t.Fatalf("selector did not preserve explicit selection: %#v", selection)
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
	if strings.Count(stderr.String(), "Uso: orbit memory <status | disable") != 1 {
		t.Fatalf("usage was not printed exactly once to stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("help was duplicated to stdout: %q", stdout.String())
	}
}

var errCodexList = errors.New("Codex list unavailable")
