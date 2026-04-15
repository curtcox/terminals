package appruntime

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	// ErrInvalidManifest indicates app manifest parse or validation failure.
	ErrInvalidManifest = errors.New("invalid app manifest")
	// ErrPermissionDenied indicates an app requested undeclared capabilities.
	ErrPermissionDenied = errors.New("permission denied")
)

// Manifest is the minimal TAR/TAL app package descriptor.
type Manifest struct {
	Name              string
	Version           string
	Language          string
	RequiresKernelAPI string
	Description       string
	Permissions       []string
	Exports           []string
	Kernels           []string
	Models            []string
	DevMode           bool
}

// Package bundles one parsed app package from disk.
type Package struct {
	RootPath   string
	Manifest   Manifest
	MainPath   string
	LoadedAt   time.Time
	FileDigest map[string]time.Time
}

// Runtime loads and hot-reloads app packages from disk.
type Runtime struct {
	mu       sync.RWMutex
	packages map[string]Package
}

// NewRuntime returns an empty application runtime.
func NewRuntime() *Runtime {
	return &Runtime{
		packages: make(map[string]Package),
	}
}

// LoadPackage parses and registers one app package.
func (r *Runtime) LoadPackage(ctx context.Context, root string) (Package, error) {
	manifestPath := filepath.Join(root, "manifest.toml")
	manifest, err := parseManifest(manifestPath)
	if err != nil {
		return Package{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Package{}, err
	}

	mainPath := filepath.Join(root, "main.tal")
	if _, err := os.Stat(mainPath); err != nil {
		return Package{}, ErrInvalidManifest
	}

	digest, _ := collectDigest(root)
	pkg := Package{
		RootPath:   root,
		Manifest:   manifest,
		MainPath:   mainPath,
		LoadedAt:   time.Now().UTC(),
		FileDigest: digest,
	}
	r.mu.Lock()
	r.packages[manifest.Name] = pkg
	r.mu.Unlock()
	_ = ctx
	return pkg, nil
}

// ReloadPackage reloads one known package if sources changed.
func (r *Runtime) ReloadPackage(ctx context.Context, name string) (Package, bool, error) {
	r.mu.RLock()
	current, ok := r.packages[name]
	r.mu.RUnlock()
	if !ok {
		return Package{}, false, ErrInvalidManifest
	}

	nextDigest, _ := collectDigest(current.RootPath)
	changed := hasDigestDiff(current.FileDigest, nextDigest)
	if !changed {
		return current, false, nil
	}
	reloaded, err := r.LoadPackage(ctx, current.RootPath)
	return reloaded, true, err
}

// ListPackages returns loaded package names.
func (r *Runtime) ListPackages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.packages))
	for name := range r.packages {
		out = append(out, name)
	}
	return out
}

func validateManifest(manifest Manifest) error {
	if strings.TrimSpace(manifest.Name) == "" ||
		strings.TrimSpace(manifest.Version) == "" ||
		strings.TrimSpace(manifest.Language) == "" {
		return ErrInvalidManifest
	}
	if manifest.Language != "tal/1" {
		return ErrInvalidManifest
	}
	return nil
}

func parseManifest(path string) (Manifest, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return Manifest{}, ErrInvalidManifest
	}
	defer file.Close()

	manifest := Manifest{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(strings.TrimSpace(val), "\"")
		switch key {
		case "name":
			manifest.Name = val
		case "version":
			manifest.Version = val
		case "language":
			manifest.Language = val
		case "requires_kernel_api":
			manifest.RequiresKernelAPI = val
		case "description":
			manifest.Description = val
		case "dev_mode":
			manifest.DevMode = strings.EqualFold(val, "true")
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, ErrInvalidManifest
	}
	return manifest, nil
}

func collectDigest(root string) (map[string]time.Time, error) {
	digest := make(map[string]time.Time)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tal") && !strings.HasSuffix(path, ".toml") {
			return nil
		}
		digest[path] = info.ModTime().UTC()
		return nil
	})
	return digest, err
}

func hasDigestDiff(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return true
	}
	for k, v := range a {
		if ov, ok := b[k]; !ok || !ov.Equal(v) {
			return true
		}
	}
	return false
}
