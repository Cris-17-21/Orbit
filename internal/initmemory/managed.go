package initmemory

import (
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

// UpdateManagedFiles applies updates only when every target is new or still
// matches Orbit's recorded digest. It validates the whole set before writing.
func UpdateManagedFiles(root string, updates []ManagedFileUpdate) error {
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
	if len(updates) == 0 {
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

	// Recheck manifest ownership immediately before any file operation.
	currentManifest, err := os.ReadFile(manifestPath)
	if err != nil || digest(currentManifest) != digest(manifestBytes) {
		return errors.New("Orbit manifest changed during configuration; no files were changed")
	}

	updatedManifest := manifest
	createdPaths := make([]string, 0, len(updates))
	originals := make(map[string][]byte)
	for _, update := range updates {
		path := filepath.Join(absoluteRoot, filepath.FromSlash(update.Path))
		current, readErr := os.ReadFile(path)
		if readErr == nil {
			originals[update.Path] = current
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return fmt.Errorf("read %q: %w", update.Path, readErr)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			rollbackManagedFiles(absoluteRoot, updates, createdPaths, originals)
			return fmt.Errorf("create directory for %q: %w", update.Path, err)
		}
		if err := writeManagedFile(path, update.Content); err != nil {
			rollbackManagedFiles(absoluteRoot, updates, createdPaths, originals)
			return fmt.Errorf("write %q: %w", update.Path, err)
		}
		if _, existed := originals[update.Path]; !existed {
			createdPaths = append(createdPaths, update.Path)
		}
		resource := Resource{ResourceID: update.ResourceID, Kind: "file", Target: update.Path, Source: update.Source, InstalledDigest: digest(update.Content), State: "managed"}
		if index, exists := resources[update.Path]; exists {
			updatedManifest.Resources[index] = resource
		} else {
			updatedManifest.Resources = append(updatedManifest.Resources, resource)
		}
	}
	if err := writeManifest(manifestPath, updatedManifest); err != nil {
		rollbackManagedFiles(absoluteRoot, updates, createdPaths, originals)
		return fmt.Errorf("write Orbit manifest: %w", err)
	}
	return nil
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
