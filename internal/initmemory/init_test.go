package initmemory

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func makeGitRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	cmd := exec.Command("git", "init", "--quiet", root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initialize temporary Git repository: %v\n%s", err, output)
	}
	return root
}

func TestFindGitRootFromNestedDirectory(t *testing.T) {
	root := makeGitRepo(t)
	nested := filepath.Join(root, "nested", "directory")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindGitRoot(nested)
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}
	gotInfo, gotErr := os.Stat(got)
	wantInfo, wantErr := os.Stat(root)
	if gotErr != nil || wantErr != nil || !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("FindGitRoot() = %q, want %q", got, root)
	}
}

func TestFindGitRootRejectsDirectoryOutsideGit(t *testing.T) {
	_, err := FindGitRoot(t.TempDir())
	if err == nil {
		t.Fatal("FindGitRoot() succeeded outside a Git repository")
	}
}

func TestApplyCreatesMemoryFilesAndOwnershipManifest(t *testing.T) {
	root := makeGitRepo(t)
	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Entries) != 3 {
		t.Fatalf("BuildPlan() entries = %d, want 3", len(plan.Entries))
	}
	for _, entry := range plan.Entries {
		if entry.Status != StatusCreate {
			t.Errorf("entry %q status = %q, want %q", entry.Path, entry.Status, StatusCreate)
		}
	}

	result, err := Apply(plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.Created) != 3 {
		t.Fatalf("Apply() created = %d, want 3", len(result.Created))
	}

	wantPaths := []string{
		"project-context/business-rules.md",
		"project-context/technical-decisions.md",
		"project-context/project-context.md",
	}
	for _, name := range wantPaths {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read generated file %q: %v", name, err)
		}
		if strings.Contains(string(contents), "status: current") {
			t.Errorf("generated file %q contains an unapproved current rule", name)
		}
		if !strings.Contains(string(contents), "memory-contract.md") {
			t.Errorf("generated file %q does not link to the memory contract", name)
		}
	}

	manifestPath := filepath.Join(root, ".orbit", "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read ownership manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse ownership manifest: %v", err)
	}
	if manifest.SchemaVersion != ManifestSchemaVersion {
		t.Errorf("manifest schema_version = %d, want %d", manifest.SchemaVersion, ManifestSchemaVersion)
	}
	if len(manifest.Resources) != 3 {
		t.Errorf("manifest resources = %d, want 3", len(manifest.Resources))
	}
	for _, resource := range manifest.Resources {
		if resource.InstalledDigest == "" {
			t.Errorf("manifest resource %q has no installed digest", resource.Target)
		}
	}
}

func TestApplyPreservesPreexistingFilesAndCreatesOtherFiles(t *testing.T) {
	root := makeGitRepo(t)
	path := filepath.Join(root, "project-context", "business-rules.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("# Team-owned business rules\nkeep this content\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := Apply(plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("preexisting file changed: got %q, want %q", got, original)
	}
	if len(result.Created) != 2 {
		t.Errorf("Apply() created = %d, want 2", len(result.Created))
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Path != "project-context/business-rules.md" {
		t.Errorf("Apply() conflicts = %#v, want existing business-rules file", result.Conflicts)
	}
	if _, err := os.Stat(filepath.Join(root, ".orbit", "manifest.json")); err != nil {
		t.Fatalf("manifest was not created for newly created resources: %v", err)
	}
}

func TestApplyPreservesUserEditsOnRepeat(t *testing.T) {
	root := makeGitRepo(t)
	firstPlan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(firstPlan); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "project-context", "business-rules.md")
	userContent := []byte("# Team-owned rules after editing\n")
	if err := os.WriteFile(path, userContent, 0o644); err != nil {
		t.Fatal(err)
	}

	secondPlan, err := BuildPlan(root)
	if err != nil {
		t.Fatalf("BuildPlan() on second run: %v", err)
	}
	if secondPlan.Entries[0].Status != StatusModified {
		t.Fatalf("second plan status = %q, want %q", secondPlan.Entries[0].Status, StatusModified)
	}
	result, err := Apply(secondPlan)
	if err != nil {
		t.Fatalf("Apply() on second run: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(userContent) {
		t.Fatalf("user edit changed: got %q, want %q", got, userContent)
	}
	if len(result.AlreadyPresent) != 2 || len(result.Conflicts) != 1 {
		t.Errorf("repeat result = %#v, want 2 existing and 1 conflict", result)
	}
}

func TestApplyAddsMissingFileToExistingManifest(t *testing.T) {
	root := makeGitRepo(t)
	firstPlan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(firstPlan); err != nil {
		t.Fatal(err)
	}

	missingPath := filepath.Join(root, "project-context", "project-context.md")
	if err := os.Remove(missingPath); err != nil {
		t.Fatal(err)
	}
	secondPlan, err := BuildPlan(root)
	if err != nil {
		t.Fatalf("BuildPlan() after file removal: %v", err)
	}
	result, err := Apply(secondPlan)
	if err != nil {
		t.Fatalf("Apply() with existing manifest: %v", err)
	}
	if len(result.Created) != 1 || result.Created[0].Path != "project-context/project-context.md" {
		t.Fatalf("created files = %#v, want only the missing project context file", result.Created)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(root, ".orbit", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Resources) != 3 {
		t.Errorf("manifest resources = %d, want 3", len(manifest.Resources))
	}
}

func TestBuildPlanRejectsInvalidManifestBeforeWrites(t *testing.T) {
	root := makeGitRepo(t)
	manifestDir := filepath.Join(root, ".orbit")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "manifest.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := BuildPlan(root); err == nil {
		t.Fatal("BuildPlan() succeeded with an invalid manifest")
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("project-context directory exists after rejected plan: err = %v", err)
	}
}

func TestBuildPlanRejectsUnsafeManifestTarget(t *testing.T) {
	root := makeGitRepo(t)
	manifestDir := filepath.Join(root, ".orbit")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := newManifest(root)
	manifest.Resources = []Resource{{
		ResourceID:      "unexpected-resource",
		Kind:            "file",
		Target:          "../outside.md",
		Source:          "orbit-init",
		InstalledDigest: "placeholder",
		State:           "managed",
	}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := BuildPlan(root); err == nil {
		t.Fatal("BuildPlan() accepted a manifest target outside the project")
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("project-context directory exists after rejecting an unsafe target: err = %v", err)
	}
}

func TestApplyRejectsStalePlanWithoutOverwriting(t *testing.T) {
	root := makeGitRepo(t)
	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "project-context", "business-rules.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("created after preview\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("file created after preview changed: got %q, want %q", got, want)
	}
	if len(result.Conflicts) != 1 {
		t.Errorf("Apply() conflicts = %#v, want one stale target conflict", result.Conflicts)
	}
}

func TestApplyRollsBackFilesWhenInitializationLockExists(t *testing.T) {
	root := makeGitRepo(t)
	metadataDir := filepath.Join(root, ".orbit")
	if err := os.Mkdir(metadataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(metadataDir, manifestLockName)
	if err := os.WriteFile(lockPath, []byte("another initialization is running"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan); err == nil {
		t.Fatal("Apply() succeeded while another initialization lock existed")
	}
	if _, err := os.Stat(filepath.Join(root, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("project memory files were not rolled back: err = %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("existing lock was removed: %v", err)
	}
}
