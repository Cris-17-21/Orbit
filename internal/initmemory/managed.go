package initmemory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ManagedFileUpdate describes a project file Orbit owns and wants to update.
type ManagedFileUpdate struct {
	Path       string
	ResourceID string
	Source     string
	Content    []byte
}

// ManagedJSONEntryUpdate describes one namespaced key inside a shared JSON
// configuration file. The manifest tracks the key value, not ownership of the
// whole file.
type ManagedJSONEntryUpdate struct {
	Path           string
	Section        string
	Key            string
	ResourceID     string
	Source         string
	Content        []byte
	ExpectedDigest string
	ExpectedAbsent bool
}

type ManagedJSONEntryPlan struct {
	Entry  Entry
	Update ManagedJSONEntryUpdate
}

// PlanManagedJSONEntry validates the current config entry without writing and
// captures the file digest so a later confirmed apply can reject stale plans.
func PlanManagedJSONEntry(root string, update ManagedJSONEntryUpdate) (ManagedJSONEntryPlan, error) {
	if err := validateManagedJSONEntryUpdate(update); err != nil {
		return ManagedJSONEntryPlan{}, err
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return ManagedJSONEntryPlan{}, fmt.Errorf("resolve project root: %w", err)
	}
	manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(manifestRelativePath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return ManagedJSONEntryPlan{}, fmt.Errorf("read Orbit manifest: %w", err)
	}
	manifest, err := parseManifest(manifestBytes, filepath.Clean(absoluteRoot))
	if err != nil {
		return ManagedJSONEntryPlan{}, err
	}
	target := managedJSONEntryTarget(update)
	var resource *Resource
	for index := range manifest.Resources {
		if manifest.Resources[index].Target == target {
			resource = &manifest.Resources[index]
			break
		}
	}
	if resource != nil && (resource.Kind != "config-entry" || resource.ResourceID != update.ResourceID || resource.State != "managed") {
		return ManagedJSONEntryPlan{}, fmt.Errorf("JSON entry %q has incompatible Orbit ownership", target)
	}
	path := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
	if err := validateManagedParentDirectories(absoluteRoot, update.Path); err != nil {
		return ManagedJSONEntryPlan{}, err
	}
	info, statErr := os.Lstat(path)
	var current []byte
	if errors.Is(statErr, os.ErrNotExist) {
		if resource != nil {
			return ManagedJSONEntryPlan{}, fmt.Errorf("managed JSON entry %q disappeared", target)
		}
		update.ExpectedAbsent = true
	} else if statErr != nil {
		return ManagedJSONEntryPlan{}, fmt.Errorf("inspect shared JSON config %q: %w", update.Path, statErr)
	} else {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return ManagedJSONEntryPlan{}, fmt.Errorf("%q is not a regular file", update.Path)
		}
		current, err = os.ReadFile(path)
		if err != nil {
			return ManagedJSONEntryPlan{}, fmt.Errorf("read shared JSON config %q: %w", update.Path, err)
		}
		update.ExpectedDigest = digest(current)
	}
	_, previous, exists, err := mergeManagedJSONEntry(current, update)
	if err != nil {
		return ManagedJSONEntryPlan{}, fmt.Errorf("inspect JSON entry %q: %w", target, err)
	}
	entry := Entry{Path: update.Path + " (" + update.Section + "." + update.Key + ")"}
	if exists {
		if resource == nil {
			entry.Status = StatusConflict
			entry.Reason = "la entrada ya existe y no está administrada por Orbit"
			return ManagedJSONEntryPlan{Entry: entry, Update: update}, nil
		}
		previousDigest, err := canonicalJSONDigest(previous)
		if err != nil || previousDigest != resource.InstalledDigest {
			entry.Status = StatusConflict
			entry.Reason = "la entrada cambió desde que Orbit la instaló"
			return ManagedJSONEntryPlan{Entry: entry, Update: update}, nil
		}
		candidateDigest, err := canonicalJSONDigest(update.Content)
		if err != nil {
			return ManagedJSONEntryPlan{}, fmt.Errorf("digest proposed JSON entry: %w", err)
		}
		if candidateDigest == resource.InstalledDigest {
			entry.Status = StatusPresent
		} else {
			entry.Status = StatusModified
			entry.Reason = "se actualizará la entrada administrada por Orbit"
		}
	} else if resource != nil {
		entry.Status = StatusConflict
		entry.Reason = "la entrada administrada por Orbit ya no existe"
	} else {
		entry.Status = StatusCreate
	}
	return ManagedJSONEntryPlan{Entry: entry, Update: update}, nil
}

// UpdateManagedFiles applies updates only when every target is new or still
// matches Orbit's recorded digest. It validates the whole set before writing.
func UpdateManagedFiles(root string, updates []ManagedFileUpdate) error {
	return UpdateManagedFilesWithJSONEntries(root, updates, nil)
}

// UpdateManagedFilesWithJSONEntries applies owned file updates and narrowly
// scoped JSON entry updates as one transaction. Existing unrelated JSON keys
// are retained, and an existing target key is changed only when its installed
// value matches the digest recorded by Orbit.
func UpdateManagedFilesWithJSONEntries(root string, updates []ManagedFileUpdate, entryUpdates []ManagedJSONEntryUpdate) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}
	metadataDir := filepath.Join(absoluteRoot, filepath.Dir(filepath.FromSlash(manifestRelativePath)))
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
	manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(manifestRelativePath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read Orbit manifest: %w", err)
	}
	manifest, err := parseManifest(manifestBytes, absoluteRoot)
	if err != nil {
		return err
	}
	if len(updates) == 0 && len(entryUpdates) == 0 {
		return nil
	}

	resources := make(map[string]int, len(manifest.Resources))
	for index, resource := range manifest.Resources {
		if _, exists := resources[resource.Target]; exists {
			return fmt.Errorf("Orbit manifest contains duplicate target %q", resource.Target)
		}
		resources[resource.Target] = index
	}
	seen := make(map[string]bool, len(updates))
	for _, update := range updates {
		if err := validateManagedUpdate(update); err != nil {
			return err
		}
		if seen[update.Path] {
			return fmt.Errorf("duplicate managed update target %q", update.Path)
		}
		seen[update.Path] = true
		if err := validateManagedParentDirectories(absoluteRoot, update.Path); err != nil {
			return err
		}
		path := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
		info, statErr := os.Lstat(path)
		resourceIndex, owned := resources[update.Path]
		if errors.Is(statErr, os.ErrNotExist) {
			if owned {
				return fmt.Errorf("managed file %q disappeared; no files were changed", update.Path)
			}
			continue
		}
		if statErr != nil {
			return fmt.Errorf("inspect %q: %w", update.Path, statErr)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q is not a regular file; no files were changed", update.Path)
		}
		if !owned {
			return fmt.Errorf("%q already exists and is not managed by Orbit; no files were changed", update.Path)
		}
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %q: %w", update.Path, readErr)
		}
		resource := manifest.Resources[resourceIndex]
		if resource.State != "managed" || digest(current) != resource.InstalledDigest {
			return fmt.Errorf("%q changed since Orbit installed it; no files were changed", update.Path)
		}
	}

	configFiles := make(map[string]*managedJSONFile)
	configFileOrder := make([]string, 0, len(entryUpdates))
	seenEntryTargets := make(map[string]bool, len(entryUpdates))
	for _, update := range entryUpdates {
		if err := validateManagedJSONEntryUpdate(update); err != nil {
			return err
		}
		if seen[update.Path] {
			return fmt.Errorf("target %q cannot be updated as both a file and a JSON entry", update.Path)
		}
		target := managedJSONEntryTarget(update)
		if seenEntryTargets[target] {
			return fmt.Errorf("duplicate managed JSON entry target %q", target)
		}
		seenEntryTargets[target] = true
		resourceIndex, owned := resources[target]

		file, exists := configFiles[update.Path]
		if !exists {
			if err := validateManagedParentDirectories(absoluteRoot, update.Path); err != nil {
				return err
			}
			path := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
			info, statErr := os.Lstat(path)
			var original []byte
			if errors.Is(statErr, os.ErrNotExist) {
				file = &managedJSONFile{}
			} else if statErr != nil {
				return fmt.Errorf("inspect shared JSON config %q: %w", update.Path, statErr)
			} else {
				if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
					return fmt.Errorf("%q is not a regular file; no files were changed", update.Path)
				}
				original, err = os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("read shared JSON config %q: %w", update.Path, err)
				}
				file = &managedJSONFile{existed: true, original: original, content: original}
			}
			if update.ExpectedAbsent && file.existed {
				return fmt.Errorf("shared JSON config %q appeared after the preview; no files were changed", update.Path)
			}
			if update.ExpectedDigest != "" && (!file.existed || digest(file.original) != update.ExpectedDigest) {
				return fmt.Errorf("shared JSON config %q changed after the preview; no files were changed", update.Path)
			}
			configFiles[update.Path] = file
			configFileOrder = append(configFileOrder, update.Path)
		} else {
			if update.ExpectedAbsent && file.existed {
				return fmt.Errorf("shared JSON config %q appeared after the preview; no files were changed", update.Path)
			}
			if update.ExpectedDigest != "" && (!file.existed || digest(file.original) != update.ExpectedDigest) {
				return fmt.Errorf("shared JSON config %q changed after the preview; no files were changed", update.Path)
			}
		}

		resource, hasResource := Resource{}, false
		if owned {
			resource = manifest.Resources[resourceIndex]
			hasResource = true
			if resource.Kind != "config-entry" || resource.ResourceID != update.ResourceID || resource.State != "managed" {
				return fmt.Errorf("JSON entry %q is registered with incompatible ownership; no files were changed", target)
			}
		}
		merged, previousEntry, exists, err := mergeManagedJSONEntry(file.content, update)
		if err != nil {
			return fmt.Errorf("prepare JSON entry %q: %w", target, err)
		}
		if exists {
			if !hasResource {
				return fmt.Errorf("JSON entry %q already exists and is not managed by Orbit; no files were changed", target)
			}
			previousDigest, err := canonicalJSONDigest(previousEntry)
			if err != nil || previousDigest != resource.InstalledDigest {
				return fmt.Errorf("JSON entry %q changed since Orbit installed it; no files were changed", target)
			}
		} else if hasResource {
			return fmt.Errorf("managed JSON entry %q disappeared; no files were changed", target)
		}
		file.content = merged
		file.updates = append(file.updates, update)
	}

	// Recheck manifest ownership immediately before any file operation.
	currentManifest, err := os.ReadFile(manifestPath)
	if err != nil || digest(currentManifest) != digest(manifestBytes) {
		return errors.New("Orbit manifest changed during configuration; no files were changed")
	}

	updatedManifest := manifest
	createdPaths := make([]string, 0, len(updates)+len(configFiles))
	originals := make(map[string][]byte)
	allWrites := make([]ManagedFileUpdate, 0, len(updates)+len(configFiles))
	allWrites = append(allWrites, updates...)
	for _, relativePath := range configFileOrder {
		file := configFiles[relativePath]
		allWrites = append(allWrites, ManagedFileUpdate{Path: relativePath, Content: file.content})
	}
	for _, update := range allWrites {
		path := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
		current, readErr := os.ReadFile(path)
		if file, isConfig := configFiles[update.Path]; isConfig {
			if file.existed && (readErr != nil || digest(current) != digest(file.original)) {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				return fmt.Errorf("shared JSON config %q changed during configuration; no files were changed", update.Path)
			}
			if !file.existed && !errors.Is(readErr, os.ErrNotExist) {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				return fmt.Errorf("shared JSON config %q appeared during configuration; no files were changed", update.Path)
			}
		}
		if readErr == nil {
			originals[update.Path] = current
		} else if !errors.Is(readErr, os.ErrNotExist) {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("read %q: %w", update.Path, readErr)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("create directory for %q: %w", update.Path, err)
		}
		if err := writeManagedFile(path, update.Content); err != nil {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("write %q: %w", update.Path, err)
		}
		if _, existed := originals[update.Path]; !existed {
			createdPaths = append(createdPaths, update.Path)
		}
	}
	for _, update := range updates {
		resource := Resource{ResourceID: update.ResourceID, Kind: "file", Target: update.Path, Source: update.Source, InstalledDigest: digest(update.Content), State: "managed"}
		if index, exists := resources[update.Path]; exists {
			updatedManifest.Resources[index] = resource
		} else {
			updatedManifest.Resources = append(updatedManifest.Resources, resource)
		}
	}
	for _, relativePath := range configFileOrder {
		for _, update := range configFiles[relativePath].updates {
			target := managedJSONEntryTarget(update)
			entry, exists, err := managedJSONEntryValue(configFiles[relativePath].content, update.Section, update.Key)
			if err != nil || !exists {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				if err != nil {
					return fmt.Errorf("verify JSON entry %q after merge: %w", target, err)
				}
				return fmt.Errorf("verify JSON entry %q after merge: entry is missing", target)
			}
			entryDigest, err := canonicalJSONDigest(entry)
			if err != nil {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				return fmt.Errorf("digest JSON entry %q: %w", target, err)
			}
			resource := Resource{ResourceID: update.ResourceID, Kind: "config-entry", Target: target, Source: update.Source, InstalledDigest: entryDigest, State: "managed"}
			if index, exists := resources[target]; exists {
				updatedManifest.Resources[index] = resource
			} else {
				updatedManifest.Resources = append(updatedManifest.Resources, resource)
			}
		}
	}
	if err := writeManifest(manifestPath, updatedManifest); err != nil {
		rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
		return fmt.Errorf("write Orbit manifest: %w", err)
	}
	return nil
}

type managedJSONFile struct {
	existed  bool
	original []byte
	content  []byte
	updates  []ManagedJSONEntryUpdate
}

func validateManagedJSONEntryUpdate(update ManagedJSONEntryUpdate) error {
	path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(update.Path)))
	if path != update.Path || !safeRelativePath(update.Path) {
		return fmt.Errorf("unsafe managed JSON config path %q", update.Path)
	}
	if update.Section == "" || update.Key == "" || update.ResourceID == "" || update.Source == "" || len(update.Content) == 0 {
		return fmt.Errorf("managed JSON entry %q requires section, key, resource ID, source, and content", update.Path)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(update.Content, &object); err != nil || object == nil {
		return fmt.Errorf("managed JSON entry %q must contain a JSON object", update.Key)
	}
	return nil
}

func managedJSONEntryTarget(update ManagedJSONEntryUpdate) string {
	return update.Path + "#/" + escapeJSONPointerSegment(update.Section) + "/" + escapeJSONPointerSegment(update.Key)
}

func escapeJSONPointerSegment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func mergeManagedJSONEntry(current []byte, update ManagedJSONEntryUpdate) ([]byte, []byte, bool, error) {
	root := make(map[string]json.RawMessage)
	if len(bytes.TrimSpace(current)) > 0 {
		if err := json.Unmarshal(current, &root); err != nil || root == nil {
			return nil, nil, false, errors.New("existing file is not a JSON object")
		}
	}
	section := make(map[string]json.RawMessage)
	if raw, found := root[update.Section]; found {
		if err := json.Unmarshal(raw, &section); err != nil || section == nil {
			return nil, nil, false, fmt.Errorf("existing %q value is not a JSON object", update.Section)
		}
	}
	previous, exists := section[update.Key]
	var value any
	if err := json.Unmarshal(update.Content, &value); err != nil {
		return nil, nil, false, fmt.Errorf("invalid managed entry JSON: %w", err)
	}
	entryBytes, err := json.Marshal(value)
	if err != nil {
		return nil, nil, false, fmt.Errorf("encode managed entry: %w", err)
	}
	section[update.Key] = entryBytes
	sectionBytes, err := json.Marshal(section)
	if err != nil {
		return nil, nil, false, fmt.Errorf("encode %q section: %w", update.Section, err)
	}
	root[update.Section] = sectionBytes
	merged, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, nil, false, fmt.Errorf("encode shared JSON config: %w", err)
	}
	return append(merged, '\n'), previous, exists, nil
}

func managedJSONEntryValue(current []byte, sectionKey, entryKey string) ([]byte, bool, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(current, &root); err != nil || root == nil {
		return nil, false, errors.New("shared config is not a JSON object")
	}
	var section map[string]json.RawMessage
	if err := json.Unmarshal(root[sectionKey], &section); err != nil || section == nil {
		return nil, false, fmt.Errorf("%q is not a JSON object", sectionKey)
	}
	entry, exists := section[entryKey]
	return entry, exists, nil
}

func canonicalJSONDigest(data []byte) (string, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return digest(canonical), nil
}

func validateManagedParentDirectories(root, relativePath string) error {
	parent := filepath.Dir(filepath.FromSlash(relativePath))
	if parent == "." {
		return nil
	}
	current := root
	for _, segment := range strings.Split(parent, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect managed parent %q: %w", current, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed parent %q is not a regular directory; no files were changed", current)
		}
	}
	return nil
}

func validateManagedUpdate(update ManagedFileUpdate) error {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(update.Path)))
	if clean != update.Path || strings.HasPrefix(clean, "../") || filepath.IsAbs(update.Path) || update.Path == "." {
		return fmt.Errorf("unsafe managed update path %q", update.Path)
	}
	if update.ResourceID == "" || update.Source == "" || len(update.Content) == 0 {
		return fmt.Errorf("managed update %q requires a resource ID, source, and content", update.Path)
	}
	return nil
}

func writeManagedFile(path string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".orbit-managed-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(content); err != nil {
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

func rollbackManagedFiles(root string, updates []ManagedFileUpdate, created []string, originals map[string][]byte) {
	for _, update := range updates {
		path := filepath.Join(root, filepath.FromSlash(update.Path))
		if original, ok := originals[update.Path]; ok {
			_ = writeManagedFile(path, original)
		}
	}
	for _, target := range created {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(target)))
	}
}
