package appruntime

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LoadPackage parses and registers one app package.
func (r *Runtime) LoadPackage(ctx context.Context, root string) (Package, error) {
	_ = ctx
	pkg, err := r.packageFromDisk(root)
	if err != nil {
		return Package{}, err
	}
	if err := r.runMigrationDryRunGate(pkg, pkg.Manifest.Name); err != nil {
		return Package{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	pkg.Revision = r.nextRevision
	r.nextRevision++
	r.packages[pkg.Manifest.Name] = pkg
	r.history[pkg.Manifest.Name] = append(r.history[pkg.Manifest.Name], pkg)
	r.migrations[pkg.Manifest.Name] = newMigrationState(pkg, "")
	return pkg, nil
}

func (r *Runtime) runMigrationDryRunGate(pkg Package, name string) error {
	if !r.migrationDryRunGateEnabled || r.skipDryRunGate {
		return nil
	}

	results, err := r.dryRunMigrationJournalReplayForPackage(pkg, name)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMigrationDryRunFailed, err)
	}
	if len(results) == 0 {
		return nil
	}
	return nil
}

// ReloadPackage reloads one known package if sources changed.
func (r *Runtime) ReloadPackage(_ context.Context, name string) (Package, bool, error) {
	r.mu.RLock()
	current, ok := r.packages[name]
	r.mu.RUnlock()
	if !ok {
		return Package{}, false, ErrPackageNotFound
	}

	nextDigest, err := collectDigest(current.RootPath)
	if err != nil {
		return Package{}, false, err
	}
	changed := hasDigestDiff(current.FileDigest, nextDigest)
	if !changed {
		return current, false, nil
	}

	next, err := r.packageFromDisk(current.RootPath)
	if err != nil {
		// Preserve current package; failed reload must not replace last-good.
		return current, true, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	next.Revision = r.nextRevision
	r.nextRevision++
	r.packages[name] = next
	r.history[name] = append(r.history[name], next)
	r.migrations[name] = newMigrationState(next, current.Manifest.Version)
	return next, true, nil
}

// RollbackPackage restores the previous successfully loaded package version.
func (r *Runtime) RollbackPackage(name string, options ...RollbackOptions) (Package, error) {
	rollbackOpts, err := normalizeRollbackOptions(options...)
	if err != nil {
		return Package{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	history := r.history[name]
	if len(history) == 0 {
		return Package{}, ErrPackageNotFound
	}
	if len(history) < 2 {
		return Package{}, ErrNoPriorVersion
	}
	if state, ok := r.migrations[name]; ok {
		if state.Verdict == "reconcile_pending" || len(state.PendingRecords) > 0 {
			return Package{}, ErrMigrationReconcilePending
		}
	}
	current := history[len(history)-1]
	previous := history[len(history)-2]
	hasDowngradeSteps := countDowngradeMigrationSteps(current.RootPath) > 0 || countDowngradeMigrationSteps(previous.RootPath) > 0
	if rollbackOpts.DataMode == RollbackDataModeKeepData && !hasDowngradeSteps {
		return Package{}, ErrRollbackKeepDataRequiresDowngrade
	}
	history = history[:len(history)-1]
	r.history[name] = history
	previous = history[len(history)-1]
	r.packages[name] = previous
	r.migrations[name] = newMigrationState(previous, "")
	return previous, nil
}

func normalizeRollbackOptions(options ...RollbackOptions) (RollbackOptions, error) {
	if len(options) > 1 {
		return RollbackOptions{}, ErrRollbackModeInvalid
	}
	opts := RollbackOptions{DataMode: RollbackDataModeArchiveData}
	if len(options) == 1 {
		opts = options[0]
	}
	mode := strings.ToLower(strings.TrimSpace(opts.DataMode))
	mode = strings.ReplaceAll(mode, "-", "_")
	if mode == "" {
		mode = RollbackDataModeArchiveData
	}
	switch mode {
	case RollbackDataModeArchiveData, RollbackDataModeKeepData, RollbackDataModePurge:
		opts.DataMode = mode
		return opts, nil
	default:
		return RollbackOptions{}, fmt.Errorf("%w: %s", ErrRollbackModeInvalid, opts.DataMode)
	}
}

func copyDirTree(src, dst string) error {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	return filepath.WalkDir(cleanSrc, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(cleanSrc, path)
		if err != nil {
			return err
		}
		target := filepath.Join(cleanDst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}

		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		in, err := os.Open(filepath.Clean(path))
		if err != nil {
			return err
		}
		defer func() {
			_ = in.Close()
		}()

		out, err := os.OpenFile(filepath.Clean(target), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		defer func() {
			_ = out.Close()
		}()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}
