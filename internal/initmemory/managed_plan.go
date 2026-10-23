package initmemory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ManagedFilesPlan is a read-only preview for installing a set of new
// Orbit-managed files. Its snapshot prevents applying a stale preview.
type ManagedFilesPlan struct {
	Root    string
	Entries []Entry

	root           string
	updates        []ManagedFileUpdate
	manifestExists bool
	manifestDigest string
	targets        []managedTargetSnapshot
}

// PlanManagedFiles classifies file destinations without writing project
// files. Existing files are installable only when the manifest proves they
// are unchanged resources with the same identity and content.
func PlanManagedFiles(root string, updates []ManagedFileUpdate) (ManagedFilesPlan, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ManagedFilesPlan{}, fmt.Errorf("resolve project root: %w", err)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	rootInfo, err := os.Lstat(absoluteRoot)
	if err != nil {
		return ManagedFilesPlan{}, fmt.Errorf("inspect project root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return ManagedFilesPlan{}, fmt.Errorf("project root %q is not a regular directory", absoluteRoot)
	}
	if len(updates) == 0 {
		return ManagedFilesPlan{}, errors.New("managed file plan requires at least one update")
	}

	manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(manifestRelativePath))
	if err := validateManagedParentDirectories(absoluteRoot, manifestRelativePath); err != nil {
		return ManagedFilesPlan{}, err
	}
	manifestInfo, statErr := os.Lstat(manifestPath)
	manifestExists := statErr == nil
	var manifest Manifest
	var manifestHash string
	if errors.Is(statErr, os.ErrNotExist) {
		manifest = newManifest(absoluteRoot)
	} else if statErr != nil {
		return ManagedFilesPlan{}, fmt.Errorf("inspect Orbit manifest: %w", statErr)
	} else {
		if !manifestInfo.Mode().IsRegular() || manifestInfo.Mode()&os.ModeSymlink != 0 {
			return ManagedFilesPlan{}, errors.New("Orbit manifest is not a regular file")
		}
		manifestBytes, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return ManagedFilesPlan{}, fmt.Errorf("read Orbit manifest: %w", readErr)
		}
		manifest, err = parseManifest(manifestBytes, absoluteRoot)
		if err != nil {
			return ManagedFilesPlan{}, err
		}
		manifestHash = digest(manifestBytes)
	}

	resources := make(map[string]Resource, len(manifest.Resources))
	for _, resource := range manifest.Resources {
		if !safeRelativePath(resource.Target) {
			return ManagedFilesPlan{}, fmt.Errorf("Orbit manifest contains unsafe target %q", resource.Target)
		}
		if _, exists := resources[resource.Target]; exists {
			return ManagedFilesPlan{}, fmt.Errorf("Orbit manifest contains duplicate target %q", resource.Target)
		}
		resources[resource.Target] = resource
	}

	plan := ManagedFilesPlan{Root: absoluteRoot, root: absoluteRoot, manifestExists: manifestExists, manifestDigest: manifestHash}
	seen := make(map[string]bool, len(updates))
	for _, update := range updates {
		if err := validateManagedUpdate(update); err != nil {
			return ManagedFilesPlan{}, err
		}
		update.Content = append([]byte(nil), update.Content...)
		if seen[update.Path] {
			return ManagedFilesPlan{}, fmt.Errorf("duplicate managed update target %q", update.Path)
		}
		seen[update.Path] = true
		absolutePath := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
		if !isWithinRoot(absoluteRoot, absolutePath) {
			return ManagedFilesPlan{}, fmt.Errorf("target %q escapes the project root", update.Path)
		}
		if err := validateManagedParentDirectories(absoluteRoot, update.Path); err != nil {
			return ManagedFilesPlan{}, err
		}

		entry := Entry{Path: update.Path}
		resource, owned := resources[update.Path]
		info, pathErr := os.Lstat(absolutePath)
		if errors.Is(pathErr, os.ErrNotExist) {
			plan.targets = append(plan.targets, managedTargetSnapshot{path: update.Path})
			if owned {
				entry.Status = StatusConflict
				entry.Reason = "el archivo administrado por Orbit ya no existe"
			} else {
				entry.Status = StatusCreate
				update.ExpectedAbsent = true
				plan.updates = append(plan.updates, update)
			}
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if pathErr != nil {
			return ManagedFilesPlan{}, fmt.Errorf("inspect target %q: %w", update.Path, pathErr)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			observed := fileSnapshot{exists: true, mode: info.Mode().Type()}
			if info.Mode()&os.ModeSymlink != 0 {
				observed.linkTarget, err = os.Readlink(absolutePath)
				if err != nil {
					return ManagedFilesPlan{}, fmt.Errorf("read target link %q: %w", update.Path, err)
				}
			}
			plan.targets = append(plan.targets, managedTargetSnapshot{path: update.Path, state: observed})
			entry.Status = StatusConflict
			entry.Reason = "la ruta ya existe y no es un archivo normal"
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		current, readErr := os.ReadFile(absolutePath)
		if readErr != nil {
			return ManagedFilesPlan{}, fmt.Errorf("read target %q: %w", update.Path, readErr)
		}
		plan.targets = append(plan.targets, managedTargetSnapshot{path: update.Path, state: fileSnapshot{exists: true, digest: digest(current)}})
		if !owned {
			entry.Status = StatusConflict
			entry.Reason = "el archivo ya existe y no figura como administrado por Orbit"
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if resource.ResourceID != update.ResourceID || resource.Kind != "file" || resource.State != "managed" {
			entry.Status = StatusConflict
			entry.Reason = "la ruta tiene una propiedad de Orbit incompatible"
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if digest(current) != resource.InstalledDigest {
			entry.Status = StatusModified
			entry.Reason = "el archivo cambió desde que Orbit lo instaló"
		} else if digest(update.Content) != resource.InstalledDigest {
			entry.Status = StatusModified
			entry.Reason = "hay una versión distinta de la skill; la actualización no forma parte de esta instalación"
		} else {
			entry.Status = StatusPresent
		}
		plan.Entries = append(plan.Entries, entry)
	}
	return plan, nil
}

// ApplyManagedFiles applies only new, non-conflicting resources from a plan.
// It never updates or removes an already-present managed file.
func ApplyManagedFiles(plan ManagedFilesPlan) (Result, error) {
	if plan.root == "" || filepath.Clean(plan.Root) != plan.root {
		return Result{}, errors.New("managed file plan root was modified; create a new plan before applying")
	}
	result := Result{}
	plannedCreates := make(map[string]bool, len(plan.updates))
	for _, entry := range plan.Entries {
		switch entry.Status {
		case StatusCreate:
			if plannedCreates[entry.Path] {
				return Result{}, fmt.Errorf("managed file plan contains duplicate create entry %q", entry.Path)
			}
			plannedCreates[entry.Path] = true
			result.Created = append(result.Created, entry)
		case StatusPresent:
			result.AlreadyPresent = append(result.AlreadyPresent, entry)
		case StatusConflict, StatusModified:
			result.Conflicts = append(result.Conflicts, entry)
		default:
			return Result{}, fmt.Errorf("unknown managed file plan status %q for %q", entry.Status, entry.Path)
		}
	}
	if len(plannedCreates) != len(plan.updates) {
		return Result{}, errors.New("managed file plan entries were modified; create a new plan before applying")
	}
	for _, update := range plan.updates {
		if !plannedCreates[update.Path] {
			return Result{}, errors.New("managed file plan entries were modified; create a new plan before applying")
		}
	}
	if len(plan.updates) == 0 {
		manifestPath := filepath.Join(plan.root, filepath.FromSlash(manifestRelativePath))
		manifestInfo, err := os.Lstat(manifestPath)
		if plan.manifestExists {
			if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Mode()&os.ModeSymlink != 0 {
				return Result{}, errors.New("Orbit manifest changed after the preview; no project files were changed")
			}
			manifestBytes, readErr := os.ReadFile(manifestPath)
			if readErr != nil || digest(manifestBytes) != plan.manifestDigest {
				return Result{}, errors.New("Orbit manifest changed after the preview; no project files were changed")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return Result{}, errors.New("Orbit manifest appeared after the preview; no project files were changed")
		}
		if err := verifyManagedTargetSnapshots(plan.root, plan.targets); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	snapshot := manifestSnapshot{exists: plan.manifestExists, digest: plan.manifestDigest, targets: plan.targets}
	if err := updateManagedFilesWithSnapshot(plan.root, plan.updates, nil, &snapshot); err != nil {
		return Result{}, err
	}
	return result, nil
}
