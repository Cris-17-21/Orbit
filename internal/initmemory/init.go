package initmemory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ManifestSchemaVersion = 1
	manifestRelativePath  = ".orbit/manifest.json"
	manifestLockName      = "init.lock"
	managedBy             = "orbit"
	orbitVersion          = "dev"
	contractURL           = "https://github.com/Cris-17-21/Orbit/blob/main/docs/memory-contract.md"
)

type EntryStatus string

const (
	StatusCreate   EntryStatus = "create"
	StatusPresent  EntryStatus = "already-present"
	StatusConflict EntryStatus = "conflict"
	StatusModified EntryStatus = "modified"
)

type Entry struct {
	Path     string
	Status   EntryStatus
	Reason   string
	content  []byte
	fileInfo os.FileInfo
}

type Plan struct {
	Root        string
	Entries     []Entry
	manifest    Manifest
	manifestRaw string
	manifestSet bool
}

type Result struct {
	Created        []Entry
	AlreadyPresent []Entry
	Conflicts      []Entry
}

type Manifest struct {
	ManagedBy     string     `json:"managed_by"`
	SchemaVersion int        `json:"schema_version"`
	OrbitVersion  string     `json:"orbit_version"`
	Scope         string     `json:"scope"`
	ProjectRoot   string     `json:"project_root"`
	InstalledAt   string     `json:"installed_at"`
	Resources     []Resource `json:"resources"`
}

type Resource struct {
	ResourceID      string `json:"resource_id"`
	Kind            string `json:"kind"`
	Target          string `json:"target"`
	Source          string `json:"source"`
	InstalledDigest string `json:"installed_digest"`
	State           string `json:"state"`
}

type resourceDefinition struct {
	id       string
	path     string
	category string
	text     string
}

func (p Plan) CreateCount() int {
	count := 0
	for _, entry := range p.Entries {
		if entry.Status == StatusCreate {
			count++
		}
	}
	return count
}

var memoryResources = []resourceDefinition{
	{
		id:       "project-memory-business-rules",
		path:     "project-context/business-rules.md",
		category: "Business rules",
		text:     "Durable product behavior, constraints, and exceptions for this project.",
	},
	{
		id:       "project-memory-technical-decisions",
		path:     "project-context/technical-decisions.md",
		category: "Technical decisions",
		text:     "Durable technical choices and the reasons behind them for this project.",
	},
	{
		id:       "project-memory-context",
		path:     "project-context/project-context.md",
		category: "Project context",
		text:     "Stable background that helps collaborators understand this project.",
	},
}

func FindGitRoot(start string) (string, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve starting directory: %w", err)
	}

	cmd := exec.Command("git", "-C", absolute, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%q is not inside a Git repository (run `git init` first)", absolute)
	}

	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errors.New("Git returned an empty repository root")
	}
	return filepath.Clean(root), nil
}

func BuildPlan(root string) (Plan, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project root: %w", err)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	plan := Plan{Root: absoluteRoot}
	manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(manifestRelativePath))
	manifestDir := filepath.Dir(manifestPath)
	if info, statErr := os.Lstat(manifestDir); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return Plan{}, fmt.Errorf("Orbit metadata path %q exists and is not a regular directory", manifestDir)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return Plan{}, fmt.Errorf("inspect Orbit metadata directory: %w", statErr)
	}

	manifestInfo, statErr := os.Lstat(manifestPath)
	switch {
	case statErr == nil:
		if !manifestInfo.Mode().IsRegular() || manifestInfo.Mode()&os.ModeSymlink != 0 {
			return Plan{}, fmt.Errorf("Orbit manifest %q is not a regular file", manifestPath)
		}
		data, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Plan{}, fmt.Errorf("read Orbit manifest: %w", readErr)
		}
		manifest, parseErr := parseManifest(data, absoluteRoot)
		if parseErr != nil {
			return Plan{}, parseErr
		}
		plan.manifest = manifest
		plan.manifestRaw = digest(data)
		plan.manifestSet = true
	case errors.Is(statErr, os.ErrNotExist):
		plan.manifest = newManifest(absoluteRoot)
	default:
		return Plan{}, fmt.Errorf("inspect Orbit manifest: %w", statErr)
	}

	resources := make(map[string]Resource, len(plan.manifest.Resources))
	for _, resource := range plan.manifest.Resources {
		if !safeRelativePath(resource.Target) {
			return Plan{}, fmt.Errorf("Orbit manifest contains unsafe target %q", resource.Target)
		}
		if _, exists := resources[resource.Target]; exists {
			return Plan{}, fmt.Errorf("Orbit manifest contains duplicate target %q", resource.Target)
		}
		resources[resource.Target] = resource
	}

	for _, resource := range memoryResources {
		entry := Entry{Path: resource.path, content: starterContent(resource.category, resource.text)}
		fullPath := filepath.Join(absoluteRoot, filepath.FromSlash(resource.path))
		if !isWithinRoot(absoluteRoot, fullPath) {
			return Plan{}, fmt.Errorf("target %q escapes the project root", resource.path)
		}

		info, fileErr := os.Lstat(fullPath)
		if errors.Is(fileErr, os.ErrNotExist) {
			entry.Status = StatusCreate
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if fileErr != nil {
			return Plan{}, fmt.Errorf("inspect target %q: %w", resource.path, fileErr)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			entry.Status = StatusConflict
			entry.Reason = "target exists and is not a regular file"
			plan.Entries = append(plan.Entries, entry)
			continue
		}

		current, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			return Plan{}, fmt.Errorf("read target %q: %w", resource.path, readErr)
		}
		owned, wasOwned := resources[resource.path]
		if !wasOwned {
			entry.Status = StatusConflict
			entry.Reason = "file already exists and is not recorded as Orbit-managed"
		} else if digest(current) != owned.InstalledDigest {
			entry.Status = StatusModified
			entry.Reason = "Orbit-managed file has changed since initialization"
		} else {
			entry.Status = StatusPresent
		}
		plan.Entries = append(plan.Entries, entry)
	}

	return plan, nil
}

func Apply(plan Plan) (Result, error) {
	result := Result{}
	manifestPath := filepath.Join(plan.Root, filepath.FromSlash(manifestRelativePath))
	if err := verifyManifestUnchanged(manifestPath, plan); err != nil {
		return result, err
	}

	var toCreate []Entry
	for _, entry := range plan.Entries {
		switch entry.Status {
		case StatusCreate:
			toCreate = append(toCreate, entry)
		case StatusPresent:
			result.AlreadyPresent = append(result.AlreadyPresent, entry)
		case StatusConflict, StatusModified:
			result.Conflicts = append(result.Conflicts, entry)
		default:
			return result, fmt.Errorf("unknown plan status %q for %q", entry.Status, entry.Path)
		}
	}
	if len(toCreate) == 0 {
		return result, nil
	}

	createdDirs := make([]string, 0, 2)
	createdFiles := make([]Entry, 0, len(toCreate))
	rollback := func() {
		for _, entry := range createdFiles {
			path := filepath.Join(plan.Root, filepath.FromSlash(entry.Path))
			removeCreatedFile(path, entry.fileInfo, digest(entry.content))
		}
		for i := len(createdDirs) - 1; i >= 0; i-- {
			_ = os.Remove(createdDirs[i])
		}
	}

	for _, entry := range toCreate {
		fullPath := filepath.Join(plan.Root, filepath.FromSlash(entry.Path))
		parent := filepath.Dir(fullPath)
		made, err := makeDirectory(parent)
		if err != nil {
			rollback()
			return Result{}, fmt.Errorf("create parent for %q: %w", entry.Path, err)
		}
		createdDirs = append(createdDirs, made...)

		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			entry.Status = StatusConflict
			entry.Reason = "target appeared after the preview; it was left unchanged"
			result.Conflicts = append(result.Conflicts, entry)
			continue
		}
		if err != nil {
			rollback()
			return Result{}, fmt.Errorf("create %q: %w", entry.Path, err)
		}
		fileInfo, err := file.Stat()
		if err != nil {
			_ = file.Close()
			rollback()
			return Result{}, fmt.Errorf("inspect created file %q: %w", entry.Path, err)
		}
		if _, err := file.Write(entry.content); err != nil {
			removeCreatedFile(fullPath, fileInfo, "")
			_ = file.Close()
			rollback()
			return Result{}, fmt.Errorf("write %q: %w", entry.Path, err)
		}
		if err := file.Sync(); err != nil {
			removeCreatedFile(fullPath, fileInfo, "")
			_ = file.Close()
			rollback()
			return Result{}, fmt.Errorf("sync %q: %w", entry.Path, err)
		}
		entry.fileInfo = fileInfo
		if err := file.Close(); err != nil {
			removeCreatedFile(fullPath, fileInfo, "")
			rollback()
			return Result{}, fmt.Errorf("close %q: %w", entry.Path, err)
		}
		createdFiles = append(createdFiles, entry)
		result.Created = append(result.Created, entry)
	}

	if len(createdFiles) == 0 {
		for i := len(createdDirs) - 1; i >= 0; i-- {
			_ = os.Remove(createdDirs[i])
		}
		return result, nil
	}

	metadataDir := filepath.Dir(manifestPath)
	made, err := makeDirectory(metadataDir)
	if err != nil {
		rollback()
		return Result{}, fmt.Errorf("create Orbit metadata directory: %w", err)
	}
	createdDirs = append(createdDirs, made...)

	lockPath := filepath.Join(metadataDir, manifestLockName)
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		rollback()
		return Result{}, fmt.Errorf("acquire Orbit initialization lock: %w", err)
	}
	if _, err := io.WriteString(lock, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		_ = lock.Close()
		_ = os.Remove(lockPath)
		rollback()
		return Result{}, fmt.Errorf("write Orbit initialization lock: %w", err)
	}
	if err := lock.Close(); err != nil {
		_ = os.Remove(lockPath)
		rollback()
		return Result{}, fmt.Errorf("close Orbit initialization lock: %w", err)
	}
	defer os.Remove(lockPath)

	if err := verifyManifestUnchanged(manifestPath, plan); err != nil {
		rollback()
		return Result{}, err
	}

	manifest := plan.manifest
	for _, entry := range createdFiles {
		resource := Resource{
			ResourceID:      resourceIDForPath(entry.Path),
			Kind:            "file",
			Target:          entry.Path,
			Source:          "orbit-init",
			InstalledDigest: digest(entry.content),
			State:           "managed",
		}
		replaced := false
		for index := range manifest.Resources {
			if manifest.Resources[index].Target == entry.Path {
				manifest.Resources[index] = resource
				replaced = true
				break
			}
		}
		if !replaced {
			manifest.Resources = append(manifest.Resources, resource)
		}
	}
	if err := writeManifest(manifestPath, manifest); err != nil {
		rollback()
		return Result{}, fmt.Errorf("write Orbit manifest: %w", err)
	}

	return result, nil
}

func starterContent(category, description string) []byte {
	return []byte(fmt.Sprintf("# %s\n\n%s\n\nUse the [Orbit memory contract](%s) for record fields and approval status.\n", category, description, contractURL))
}

func newManifest(root string) Manifest {
	return Manifest{
		ManagedBy:     managedBy,
		SchemaVersion: ManifestSchemaVersion,
		OrbitVersion:  orbitVersion,
		Scope:         "project",
		ProjectRoot:   root,
		InstalledAt:   time.Now().UTC().Format(time.RFC3339),
		Resources:     []Resource{},
	}
}

func parseManifest(data []byte, root string) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("read Orbit manifest: %w; no project files were changed", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("Orbit manifest contains trailing data; no project files were changed")
	}
	if manifest.ManagedBy != managedBy {
		return Manifest{}, errors.New("existing .orbit/manifest.json is not an Orbit manifest; no project files were changed")
	}
	if manifest.SchemaVersion != ManifestSchemaVersion {
		return Manifest{}, fmt.Errorf("unsupported Orbit manifest schema version %d; expected %d; no project files were changed", manifest.SchemaVersion, ManifestSchemaVersion)
	}
	if manifest.Scope != "project" || filepath.Clean(manifest.ProjectRoot) != root {
		return Manifest{}, errors.New("Orbit manifest belongs to a different project or scope; no project files were changed")
	}
	if manifest.Resources == nil {
		manifest.Resources = []Resource{}
	}
	return manifest, nil
}

func verifyManifestUnchanged(path string, plan Plan) error {
	data, err := os.ReadFile(path)
	if !plan.manifestSet {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err == nil {
			return errors.New("Orbit manifest appeared after the preview; no files were changed")
		}
		return fmt.Errorf("inspect Orbit manifest before applying the plan: %w", err)
	}
	if err != nil {
		return fmt.Errorf("Orbit manifest changed or disappeared after the preview; no files were changed: %w", err)
	}
	if digest(data) != plan.manifestRaw {
		return errors.New("Orbit manifest changed after the preview; no files were changed")
	}
	return nil
}

func writeManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func removeCreatedFile(path string, createdInfo os.FileInfo, expectedDigest string) {
	if createdInfo == nil {
		return
	}
	currentInfo, err := os.Lstat(path)
	if err != nil || currentInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(createdInfo, currentInfo) {
		return
	}
	if expectedDigest != "" {
		current, err := os.ReadFile(path)
		if err != nil || digest(current) != expectedDigest {
			return
		}
	}
	_ = os.Remove(path)
}

func makeDirectory(path string) ([]string, error) {
	var missing []string
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("%q is not a regular directory", current)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return nil, fmt.Errorf("could not find an existing parent for %q", path)
		}
	}

	var created []string
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0o755); err != nil {
			for j := len(created) - 1; j >= 0; j-- {
				_ = os.Remove(created[j])
			}
			return nil, err
		}
		created = append(created, missing[i])
	}
	return created, nil
}

func safeRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.ContainsAny(path, `\\:`) || strings.IndexByte(path, 0) >= 0 {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func isWithinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func resourceIDForPath(path string) string {
	for _, resource := range memoryResources {
		if resource.path == path {
			return resource.id
		}
	}
	return "project-memory-" + strings.NewReplacer("/", "-", ".", "-").Replace(path)
}
