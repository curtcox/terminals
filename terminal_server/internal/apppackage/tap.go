// Package apppackage builds and validates canonical .tap application archives.
package apppackage

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"
)

const (
	canonicalFileMode = 0o644
	zstdWindowSize    = 8 << 20
)

var (
	// ErrInvalidTapFormat indicates the .tap payload cannot be decompressed or parsed.
	ErrInvalidTapFormat = errors.New("invalid tap format")
	// ErrMissingManifest indicates required manifest.toml is absent.
	ErrMissingManifest = errors.New("tap missing manifest.toml")
	// ErrMissingMainTAL indicates required main.tal is absent.
	ErrMissingMainTAL = errors.New("tap missing main.tal")
	// ErrUnknownTopLevelEntry indicates an entry is outside the allowed v1 path set.
	ErrUnknownTopLevelEntry = errors.New("tap contains unknown top-level entry")
	// ErrInvalidTarEntry indicates a tar entry violates canonical form constraints.
	ErrInvalidTarEntry = errors.New("tap contains invalid tar entry")
	// ErrDuplicateArchivePath indicates one archive path appears more than once.
	ErrDuplicateArchivePath = errors.New("tap contains duplicate archive path")
	// ErrCaseCollidingPath indicates two archive paths differ only by case.
	ErrCaseCollidingPath = errors.New("tap contains case-colliding path")
	// ErrNonCanonicalOrder indicates archive members are not lexicographically sorted.
	ErrNonCanonicalOrder = errors.New("tap archive entries are not sorted")
	// ErrPathTraversalDetected indicates an archive or source path is unsafe.
	ErrPathTraversalDetected = errors.New("tap contains unsafe path")
)

var allowedTopLevelDirs = map[string]struct{}{
	"lib":     {},
	"tests":   {},
	"kernels": {},
	"models":  {},
	"assets":  {},
	"migrate": {},
}

// VerifiedTap is the pre-trust parsed output for a .tap archive.
type VerifiedTap struct {
	PackageID   string
	PackageName string
	Files       []string
}

// BuildTapFromDir builds a canonical .tap archive from one app root directory.
func BuildTapFromDir(root string) ([]byte, string, error) {
	cleanRoot := filepath.Clean(root)
	packageName := filepath.Base(cleanRoot)
	if err := validatePathComponent(packageName); err != nil {
		return nil, "", fmt.Errorf("invalid package name %q: %w", packageName, err)
	}

	relPaths, err := collectSourceFiles(cleanRoot)
	if err != nil {
		return nil, "", err
	}

	canonicalTar, err := buildCanonicalTar(cleanRoot, packageName, relPaths)
	if err != nil {
		return nil, "", err
	}

	hash := sha256.Sum256(canonicalTar)
	packageID := "sha256:" + hex.EncodeToString(hash[:])

	tapBytes, err := compressCanonicalTar(canonicalTar)
	if err != nil {
		return nil, "", err
	}

	return tapBytes, packageID, nil
}

// VerifyTap validates a .tap archive and returns its canonical package metadata.
func VerifyTap(tapBytes []byte) (VerifiedTap, error) {
	canonicalTar, err := decompressTap(tapBytes)
	if err != nil {
		return VerifiedTap{}, err
	}

	result, err := validateCanonicalTar(canonicalTar)
	if err != nil {
		return VerifiedTap{}, err
	}

	hash := sha256.Sum256(canonicalTar)
	result.PackageID = "sha256:" + hex.EncodeToString(hash[:])
	return result, nil
}

func collectSourceFiles(root string) ([]string, error) {
	relPaths := make([]string, 0, 16)
	err := filepath.WalkDir(root, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}

		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if err := validateRelativeArchivePath(rel); err != nil {
			return fmt.Errorf("%w: %s", ErrPathTraversalDetected, rel)
		}

		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%w: %s", ErrInvalidTarEntry, rel)
		}
		if err := validateTopLevelPath(rel); err != nil {
			return err
		}
		relPaths = append(relPaths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	hasManifest := false
	hasMain := false
	for _, rel := range relPaths {
		switch rel {
		case "manifest.toml":
			hasManifest = true
		case "main.tal":
			hasMain = true
		}
	}
	if !hasManifest {
		return nil, ErrMissingManifest
	}
	if !hasMain {
		return nil, ErrMissingMainTAL
	}

	sort.Strings(relPaths)
	return relPaths, nil
}

func buildCanonicalTar(root string, packageName string, relPaths []string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, rel := range relPaths {
		fullPath := filepath.Join(root, filepath.FromSlash(rel))
		payload, err := os.ReadFile(fullPath)
		if err != nil {
			_ = tw.Close()
			return nil, err
		}

		hdr := &tar.Header{
			Name:     packageName + "/" + rel,
			Mode:     canonicalFileMode,
			Uid:      0,
			Gid:      0,
			Uname:    "",
			Gname:    "",
			Size:     int64(len(payload)),
			ModTime:  time.Unix(0, 0).UTC(),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = tw.Close()
			return nil, err
		}
		if _, err := tw.Write(payload); err != nil {
			_ = tw.Close()
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func compressCanonicalTar(canonicalTar []byte) ([]byte, error) {
	enc, err := zstd.NewWriter(
		nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(19)),
		zstd.WithEncoderCRC(false),
		zstd.WithWindowSize(zstdWindowSize),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = enc.Close()
	}()

	return enc.EncodeAll(canonicalTar, make([]byte, 0, len(canonicalTar))), nil
}

func decompressTap(tapBytes []byte) ([]byte, error) {
	zr, err := zstd.NewReader(bytes.NewReader(tapBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTapFormat, err)
	}
	defer zr.Close()

	canonicalTar, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTapFormat, err)
	}
	if len(canonicalTar) == 0 {
		return nil, fmt.Errorf("%w: empty payload", ErrInvalidTapFormat)
	}
	return canonicalTar, nil
}

func validateCanonicalTar(canonicalTar []byte) (VerifiedTap, error) {
	tr := tar.NewReader(bytes.NewReader(canonicalTar))
	seen := make(map[string]struct{})
	seenCaseFolded := make(map[string]struct{})
	files := make([]string, 0, 16)
	packageName := ""
	lastName := ""
	manifestCount := 0
	mainCount := 0

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return VerifiedTap{}, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}

		if hdr.Typeflag != tar.TypeReg {
			return VerifiedTap{}, ErrInvalidTarEntry
		}
		if hdr.Name <= lastName {
			return VerifiedTap{}, ErrNonCanonicalOrder
		}
		lastName = hdr.Name

		if err := validateArchivePath(hdr.Name); err != nil {
			return VerifiedTap{}, err
		}
		if _, ok := seen[hdr.Name]; ok {
			return VerifiedTap{}, ErrDuplicateArchivePath
		}
		seen[hdr.Name] = struct{}{}

		folded := strings.ToLower(hdr.Name)
		if _, ok := seenCaseFolded[folded]; ok {
			return VerifiedTap{}, ErrCaseCollidingPath
		}
		seenCaseFolded[folded] = struct{}{}

		parts := strings.Split(hdr.Name, "/")
		if len(parts) < 2 {
			return VerifiedTap{}, ErrInvalidTarEntry
		}
		entryPackage := parts[0]
		if packageName == "" {
			packageName = entryPackage
		}
		if entryPackage != packageName {
			return VerifiedTap{}, ErrInvalidTarEntry
		}
		rel := strings.Join(parts[1:], "/")
		if err := validateTopLevelPath(rel); err != nil {
			return VerifiedTap{}, err
		}

		if hdr.Mode != canonicalFileMode || hdr.Uid != 0 || hdr.Gid != 0 || hdr.Uname != "" || hdr.Gname != "" {
			return VerifiedTap{}, ErrInvalidTarEntry
		}
		if !hdr.ModTime.Equal(time.Unix(0, 0).UTC()) {
			return VerifiedTap{}, ErrInvalidTarEntry
		}

		switch rel {
		case "manifest.toml":
			manifestCount++
		case "main.tal":
			mainCount++
		}

		if _, err := io.Copy(io.Discard, tr); err != nil {
			return VerifiedTap{}, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}
		files = append(files, rel)
	}

	if packageName == "" {
		return VerifiedTap{}, fmt.Errorf("%w: no entries", ErrInvalidTapFormat)
	}
	if manifestCount != 1 {
		return VerifiedTap{}, ErrMissingManifest
	}
	if mainCount != 1 {
		return VerifiedTap{}, ErrMissingMainTAL
	}

	return VerifiedTap{PackageName: packageName, Files: files}, nil
}

func validateArchivePath(name string) error {
	if !utf8.ValidString(name) {
		return ErrPathTraversalDetected
	}
	if strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return ErrPathTraversalDetected
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ErrPathTraversalDetected
	}
	for _, component := range strings.Split(name, "/") {
		if err := validatePathComponent(component); err != nil {
			return ErrPathTraversalDetected
		}
	}
	return nil
}

func validateRelativeArchivePath(rel string) error {
	if strings.Contains(rel, "\\") {
		return ErrPathTraversalDetected
	}
	return validateArchivePath(rel)
}

func validatePathComponent(component string) error {
	if component == "" || component == "." || component == ".." {
		return ErrPathTraversalDetected
	}
	if len(component) > 255 {
		return ErrPathTraversalDetected
	}
	return nil
}

func validateTopLevelPath(rel string) error {
	if rel == "manifest.toml" || rel == "main.tal" {
		return nil
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return fmt.Errorf("%w: %s", ErrUnknownTopLevelEntry, rel)
	}
	if _, ok := allowedTopLevelDirs[parts[0]]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknownTopLevelEntry, rel)
	}
	return nil
}
