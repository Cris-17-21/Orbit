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
	Path           string
	ResourceID     string
	Source         string
	Content        []byte
	ExpectedDigest string
	ExpectedAbsent bool
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
	return updateManagedFilesWithSnapshot(root, updates, entryUpdates, nil)
}

type manifestSnapshot struct {
	exists  bool
	digest  string
	targets []managedTargetSnapshot
}

type managedTargetSnapshot struct {
	path  string
	state fileSnapshot
}

func updateManagedFilesWithSnapshot(root string, updates []ManagedFileUpdate, entryUpdates []ManagedJSONEntryUpdate, snapshot *manifestSnapshot) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}
	metadataDir := filepath.Join(absoluteRoot, filepath.Dir(filepath.FromSlash(manifestRelativePath)))
	if err := validateManagedParentDirectories(absoluteRoot, manifestRelativePath); err != nil {
		return err
	}
	createdDirectories, err := makeDirectory(metadataDir)
	if err != nil {
		return fmt.Errorf("prepare Orbit metadata directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			removeEmptyDirectories(createdDirectories)
		}
	}()
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
	manifestInfo, manifestStatErr := os.Lstat(manifestPath)
	manifestExists := manifestStatErr == nil
	var manifestBytes []byte
	var manifest Manifest
	if errors.Is(manifestStatErr, os.ErrNotExist) {
		if snapshot == nil {
			return errors.New("Orbit manifest is missing; run orbit init before updating managed resources")
		}
		manifest = newManifest(absoluteRoot)
	} else if manifestStatErr != nil {
		return fmt.Errorf("inspect Orbit manifest: %w", manifestStatErr)
	} else {
		if !manifestInfo.Mode().IsRegular() || manifestInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("Orbit manifest is not a regular file; no project files were changed")
		}
		manifestBytes, err = os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("read Orbit manifest: %w", err)
		}
		manifest, err = parseManifest(manifestBytes, absoluteRoot)
		if err != nil {
			return err
		}
	}
	if snapshot != nil && (snapshot.exists != manifestExists || (snapshot.exists && snapshot.digest != digest(manifestBytes))) {
		return errors.New("Orbit manifest changed after the preview; no project files were changed")
	}
	if snapshot != nil {
		if err := verifyManagedTargetSnapshots(absoluteRoot, snapshot.targets); err != nil {
			return err
		}
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
	fileSnapshots := make(map[string]fileSnapshot, len(updates))
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
			if update.ExpectedDigest != "" {
				return fmt.Errorf("managed file %q disappeared after the preview; no files were changed", update.Path)
			}
			fileSnapshots[update.Path] = fileSnapshot{}
			continue
		}
		if statErr != nil {
			return fmt.Errorf("inspect %q: %w", update.Path, statErr)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q is not a regular file; no files were changed", update.Path)
		}
		if update.ExpectedAbsent {
			return fmt.Errorf("target %q appeared after the preview; no files were changed", update.Path)
		}
		if !owned {
			return fmt.Errorf("%q already exists and is not managed by Orbit; no files were changed", update.Path)
		}
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %q: %w", update.Path, readErr)
		}
		if update.ExpectedDigest != "" && digest(current) != update.ExpectedDigest {
			return fmt.Errorf("target %q changed after the preview; no files were changed", update.Path)
		}
		resource := manifest.Resources[resourceIndex]
		if resource.State != "managed" || digest(current) != resource.InstalledDigest {
			return fmt.Errorf("%q changed since Orbit installed it; no files were changed", update.Path)
		}
		fileSnapshots[update.Path] = fileSnapshot{exists: true, digest: digest(current)}
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
		fileSnapshots[update.Path] = fileSnapshot{exists: file.existed}
		if file.existed {
			fileSnapshots[update.Path] = fileSnapshot{exists: true, digest: digest(file.original)}
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
	currentManifestInfo, currentManifestErr := os.Lstat(manifestPath)
	if manifestExists {
		if currentManifestErr != nil || !currentManifestInfo.Mode().IsRegular() || currentManifestInfo.Mode()&os.ModeSymlink != 0 {
			return errors.New("Orbit manifest changed during configuration; no files were changed")
		}
		currentManifest, readErr := os.ReadFile(manifestPath)
		if readErr != nil || digest(currentManifest) != digest(manifestBytes) {
			return errors.New("Orbit manifest changed during configuration; no files were changed")
		}
	} else if !errors.Is(currentManifestErr, os.ErrNotExist) {
		return errors.New("Orbit manifest appeared during configuration; no files were changed")
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
		fileInfo, statErr := os.Lstat(path)
		snapshot := fileSnapshots[update.Path]
		if snapshot.exists {
			if statErr != nil || !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				return fmt.Errorf("target %q changed during configuration; no files were changed", update.Path)
			}
			currentBytes, readErr := os.ReadFile(path)
			if readErr != nil || digest(currentBytes) != snapshot.digest {
				rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
				return fmt.Errorf("target %q changed during configuration; no files were changed", update.Path)
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("target %q appeared during configuration; no files were changed", update.Path)
		}
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
		created, dirErr := makeDirectory(filepath.Dir(path))
		if dirErr != nil {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("create directory for %q: %w", update.Path, dirErr)
		}
		createdDirectories = append(createdDirectories, created...)
		var writeErr error
		if update.ExpectedAbsent {
			writeErr = createManagedFile(path, update.Content)
		} else {
			writeErr = writeManagedFile(path, update.Content)
		}
		if writeErr != nil {
			rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
			return fmt.Errorf("write %q: %w", update.Path, writeErr)
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
	var writeErr error
	if !manifestExists {
		writeErr = writeManifestExclusive(manifestPath, updatedManifest)
	} else {
		writeErr = writeManifest(manifestPath, updatedManifest)
	}
	if writeErr != nil {
		rollbackManagedFiles(absoluteRoot, allWrites, createdPaths, originals)
		return fmt.Errorf("write Orbit manifest: %w", writeErr)
	}
	committed = true
	return nil
}

type fileSnapshot struct {
	exists     bool
	digest     string
	mode       os.FileMode
	linkTarget string
}

func verifyManagedTargetSnapshots(root string, targets []managedTargetSnapshot) error {
	for _, target := range targets {
		if !safeRelativePath(target.path) {
			return fmt.Errorf("unsafe managed target %q in plan", target.path)
		}
		path := filepath.Join(root, filepath.FromSlash(target.path))
		info, err := os.Lstat(path)
		if !target.state.exists {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("target %q appeared after the preview; no project files were changed", target.path)
		}
		if err != nil || info.Mode().Type() != target.state.mode {
			return fmt.Errorf("target %q changed after the preview; no project files were changed", target.path)
		}
		switch {
		case info.Mode().IsRegular():
			content, readErr := os.ReadFile(path)
			if readErr != nil || digest(content) != target.state.digest {
				return fmt.Errorf("target %q changed after the preview; no project files were changed", target.path)
			}
		case info.Mode()&os.ModeSymlink != 0:
			linkTarget, readErr := os.Readlink(path)
			if readErr != nil || linkTarget != target.state.linkTarget {
				return fmt.Errorf("target %q changed after the preview; no project files were changed", target.path)
			}
		}
	}
	return nil
}

func removeEmptyDirectories(directories []string) {
	for i := len(directories) - 1; i >= 0; i-- {
		_ = os.Remove(directories[i])
	}
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
	if !safeRelativePath(update.Path) {
		return fmt.Errorf("unsafe managed update path %q", update.Path)
	}
	if update.Path == manifestRelativePath || update.Path == filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(manifestRelativePath)), manifestLockName)) {
		return fmt.Errorf("managed update %q targets Orbit's own metadata", update.Path)
	}
	if update.ExpectedAbsent && update.ExpectedDigest != "" {
		return fmt.Errorf("managed update %q cannot expect both an absent and an existing target", update.Path)
	}
	if update.ResourceID == "" || update.Source == "" || len(update.Content) == 0 {
		return fmt.Errorf("managed update %q requires a resource ID, source, and content", update.Path)
	}
	return nil
}

func createManagedFile(path string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".orbit-managed-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return err
	}
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
	if err := os.Link(tempPath, path); err != nil {
		return err
	}
	return nil
}

func writeManifestExclusive(path string, manifest Manifest) error {
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
	return os.Link(tempPath, path)
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
