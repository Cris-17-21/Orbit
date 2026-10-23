package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
	if !strings.Contains(stdout.String(), "Creados: 3") {
		t.Fatalf("summary does not report created files: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project-context", "business-rules.md")); err != nil {
		t.Fatalf("memory file was not created: %v", err)
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
		confirmed = len(plan.Entries) == 3
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
