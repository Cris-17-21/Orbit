package initmemory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MemoryDisablePlan describes the Orbit-owned Atomic Memory resources that
// can be removed without discarding user changes.
type MemoryDisablePlan struct {
	Root    string
	Entries []Entry

	root           string
	manifestDigest string
	manifest       Manifest
	resources      []Resource
	targets        []managedTargetSnapshot
	files          map[string][]byte
	jsonEntries    map[string]memoryJSONRemoval
}

type memoryJSONRemoval struct {
	section string
	key     string
}

// PlanMemoryDisable only schedules resources whose manifest ownership and
// installed digest still match. Modified and unowned resources are reported
// and left untouched.
func PlanMemoryDisable(root string) (MemoryDisablePlan, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return MemoryDisablePlan{}, fmt.Errorf("resolve project root: %w", err)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(manifestRelativePath))
	if err := validateManagedParentDirectories(absoluteRoot, manifestRelativePath); err != nil {
		return MemoryDisablePlan{}, err
	}
	info, err := os.Lstat(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MemoryDisablePlan{}, errors.New("no se encontró el manifiesto de Orbit; no hay recursos cuya propiedad se pueda verificar")
		}
		return MemoryDisablePlan{}, fmt.Errorf("inspect Orbit manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return MemoryDisablePlan{}, errors.New("Orbit manifest is not a regular file")
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return MemoryDisablePlan{}, fmt.Errorf("read Orbit manifest: %w", err)
	}
	manifest, err := parseManifest(manifestBytes, absoluteRoot)
	if err != nil {
		return MemoryDisablePlan{}, err
	}
	plan := MemoryDisablePlan{Root: absoluteRoot, root: absoluteRoot, manifestDigest: digest(manifestBytes), manifest: manifest, files: map[string][]byte{}, jsonEntries: map[string]memoryJSONRemoval{}}
	known := []struct {
		id, kind, target string
	}{
		{"project-atomic-memory-config", "file", ".orbit/atomic-memory.json"},
		{"project-codex-atomic-memory-config", "file", ".codex/config.toml"},
		{"project-claude-atomic-memory-config", "config-entry", ".mcp.json#/mcpServers/atomicmemory"},
	}
	resources := make(map[string]Resource, len(manifest.Resources))
	for _, resource := range manifest.Resources {
		if _, exists := resources[resource.Target]; exists {
			return MemoryDisablePlan{}, fmt.Errorf("Orbit manifest contains duplicate target %q", resource.Target)
		}
		resources[resource.Target] = resource
	}
	for _, candidate := range known {
		resource, owned := resources[candidate.target]
		entry := Entry{Path: candidate.target}
		if !owned || resource.ResourceID != candidate.id || resource.Kind != candidate.kind || resource.State != "managed" {
			entry.Status = StatusConflict
			entry.Reason = "no hay propiedad de Orbit compatible en el manifiesto"
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if candidate.kind == "file" {
			if err := validateManagedParentDirectories(absoluteRoot, candidate.target); err != nil {
				return MemoryDisablePlan{}, err
			}
			path := filepath.Join(absoluteRoot, filepath.FromSlash(candidate.target))
			fileInfo, statErr := os.Lstat(path)
			if statErr != nil {
				entry.Status = StatusConflict
				entry.Reason = "el archivo ya no existe"
				if !errors.Is(statErr, os.ErrNotExist) {
					return MemoryDisablePlan{}, fmt.Errorf("inspect %q: %w", candidate.target, statErr)
				}
				plan.targets = append(plan.targets, managedTargetSnapshot{path: candidate.target})
				plan.Entries = append(plan.Entries, entry)
				continue
			}
			if !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 {
				entry.Status, entry.Reason = StatusConflict, "la ruta ya no es un archivo normal"
				plan.Entries = append(plan.Entries, entry)
				continue
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return MemoryDisablePlan{}, fmt.Errorf("read %q: %w", candidate.target, readErr)
			}
			plan.targets = append(plan.targets, managedTargetSnapshot{path: candidate.target, state: fileSnapshot{exists: true, digest: digest(content)}})
			if digest(content) != resource.InstalledDigest {
				entry.Status, entry.Reason = StatusModified, "el archivo cambió desde que Orbit lo instaló; se conservará"
			} else {
				entry.Status = StatusPresent
				plan.files[candidate.target] = content
				plan.resources = append(plan.resources, resource)
			}
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		path := ".mcp.json"
		if err := validateManagedParentDirectories(absoluteRoot, path); err != nil {
			return MemoryDisablePlan{}, err
		}
		configPath := filepath.Join(absoluteRoot, filepath.FromSlash(path))
		configInfo, statErr := os.Lstat(configPath)
		if statErr != nil {
			entry.Status, entry.Reason = StatusConflict, "la entrada administrada ya no existe"
			if !errors.Is(statErr, os.ErrNotExist) {
				return MemoryDisablePlan{}, fmt.Errorf("inspect %q: %w", path, statErr)
			}
			plan.targets = append(plan.targets, managedTargetSnapshot{path: path})
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		if !configInfo.Mode().IsRegular() || configInfo.Mode()&os.ModeSymlink != 0 {
			entry.Status, entry.Reason = StatusConflict, "la ruta compartida no es un archivo normal"
			plan.Entries = append(plan.Entries, entry)
			continue
		}
		configBytes, readErr := os.ReadFile(configPath)
		if readErr != nil {
			return MemoryDisablePlan{}, fmt.Errorf("read %q: %w", path, readErr)
		}
		plan.targets = append(plan.targets, managedTargetSnapshot{path: path, state: fileSnapshot{exists: true, digest: digest(configBytes)}})
		value, exists, valueErr := managedJSONEntryValue(configBytes, "mcpServers", "atomicmemory")
		if valueErr != nil {
			entry.Status, entry.Reason = StatusConflict, "no se pudo validar la entrada MCP; se conservará"
		} else if !exists {
			entry.Status, entry.Reason = StatusConflict, "la entrada administrada ya no existe"
		} else if valueDigest, digestErr := canonicalJSONDigest(value); digestErr != nil || valueDigest != resource.InstalledDigest {
			entry.Status, entry.Reason = StatusModified, "la entrada cambió desde que Orbit la instaló; se conservará"
		} else {
			entry.Status = StatusPresent
			plan.jsonEntries[path] = memoryJSONRemoval{section: "mcpServers", key: "atomicmemory"}
			plan.resources = append(plan.resources, resource)
		}
		plan.Entries = append(plan.Entries, entry)
	}
	return plan, nil
}

// ApplyMemoryDisable applies a previously reviewed removal plan. It rejects
// stale manifests or targets and restores changed files if the manifest write
// fails.
func ApplyMemoryDisable(plan MemoryDisablePlan) error {
	if plan.root == "" || filepath.Clean(plan.Root) != plan.root {
		return errors.New("memory disable plan was modified; create a new plan")
	}
	metadataDir := filepath.Join(plan.root, filepath.Dir(filepath.FromSlash(manifestRelativePath)))
	if err := validateManagedParentDirectories(plan.root, manifestRelativePath); err != nil {
		return err
	}
	lockPath := filepath.Join(metadataDir, manifestLockName)
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("acquire Orbit configuration lock: %w", err)
	}
	if err := lock.Close(); err != nil {
		_ = os.Remove(lockPath)
		return fmt.Errorf("close Orbit configuration lock: %w", err)
	}
	defer os.Remove(lockPath)
	manifestPath := filepath.Join(plan.root, filepath.FromSlash(manifestRelativePath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil || digest(manifestBytes) != plan.manifestDigest {
		return errors.New("Orbit manifest changed after the preview; no files were changed")
	}
	if err := verifyManagedTargetSnapshots(plan.root, plan.targets); err != nil {
		return err
	}
	latestManifest, err := os.ReadFile(manifestPath)
	if err != nil || digest(latestManifest) != plan.manifestDigest {
		return errors.New("Orbit manifest changed after the preview; no files were changed")
	}
	if len(plan.resources) == 0 {
		return nil
	}
	updated := plan.manifest
	removeTargets := make(map[string]bool, len(plan.resources))
	for _, resource := range plan.resources {
		removeTargets[resource.Target] = true
	}
	kept := updated.Resources[:0]
	for _, resource := range updated.Resources {
		if !removeTargets[resource.Target] {
			kept = append(kept, resource)
		}
	}
	updated.Resources = kept
	originals := map[string][]byte{}
	removedFiles := []string{}
	for target, original := range plan.files {
		path := filepath.Join(plan.root, filepath.FromSlash(target))
		current, readErr := os.ReadFile(path)
		if readErr != nil || digest(current) != digest(original) {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("target %q changed after the preview; no files were changed", target)
		}
		if err := os.Remove(path); err != nil {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("remove %q: %w", target, err)
		}
		originals[target] = original
		removedFiles = append(removedFiles, target)
	}
	for target, removal := range plan.jsonEntries {
		path := filepath.Join(plan.root, filepath.FromSlash(target))
		original, readErr := os.ReadFile(path)
		if readErr != nil {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("read %q before removal: %w", target, readErr)
		}
		snapshotDigest := ""
		for _, snapshot := range plan.targets {
			if snapshot.path == target {
				snapshotDigest = snapshot.state.digest
				break
			}
		}
		if snapshotDigest == "" || digest(original) != snapshotDigest {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("target %q changed after the preview; no files were changed", target)
		}
		updatedJSON, err := removeManagedJSONEntry(original, removal.section, removal.key)
		if err != nil {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("remove managed JSON entry from %q: %w", target, err)
		}
		originals[target] = original
		if err := writeManagedFile(path, updatedJSON); err != nil {
			rollbackMemoryRemoval(plan.root, originals, removedFiles)
			return fmt.Errorf("update %q: %w", target, err)
		}
		removedFiles = append(removedFiles, target)
	}
	if err := writeManifest(manifestPath, updated); err != nil {
		rollbackMemoryRemoval(plan.root, originals, removedFiles)
		return fmt.Errorf("write Orbit manifest: %w", err)
	}
	return nil
}

func removeManagedJSONEntry(data []byte, sectionKey, entryKey string) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil || root == nil {
		return nil, errors.New("shared config is not a JSON object")
	}
	var section map[string]json.RawMessage
	if err := json.Unmarshal(root[sectionKey], &section); err != nil || section == nil {
		return nil, fmt.Errorf("%q is not a JSON object", sectionKey)
	}
	if _, exists := section[entryKey]; !exists {
		return nil, fmt.Errorf("%q entry is missing", entryKey)
	}
	delete(section, entryKey)
	sectionBytes, err := json.Marshal(section)
	if err != nil {
		return nil, err
	}
	root[sectionKey] = sectionBytes
	merged, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(merged, '\n'), nil
}

func rollbackMemoryRemoval(root string, originals map[string][]byte, targets []string) {
	for i := len(targets) - 1; i >= 0; i-- {
		target := targets[i]
		path := filepath.Join(root, filepath.FromSlash(target))
		if original, exists := originals[target]; exists {
			_ = writeManagedFile(path, original)
		}
	}
}
