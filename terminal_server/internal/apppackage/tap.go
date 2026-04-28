// Package apppackage builds and validates canonical .tap application archives.
package apppackage

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/unicode/norm"
)

const (
	canonicalFileMode = 0o644
	zstdWindowSize    = 8 << 20
	zstdFrameMagic    = 0xFD2FB528

	signatureBundleSchema   = "tap-sig/1"
	signatureBundleMaxBytes = 1 << 20
	statementMaxCount       = 64
	stringFieldMaxBytes     = 8 << 10
	statementNonceLen       = 16
	voucherNotesMaxBytes    = 2 << 10
	publisherViaMaxBytes    = 256
	migrationFixtureMaxRows = 4096
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
	// ErrInvalidManifest indicates the manifest is missing required fields.
	ErrInvalidManifest = errors.New("tap manifest is invalid")
	// ErrInvalidSignatureBundle indicates the signature bundle is malformed.
	ErrInvalidSignatureBundle = errors.New("tap signature bundle is invalid")
	// ErrInvalidSignatureStatement indicates one signature statement is malformed.
	ErrInvalidSignatureStatement = errors.New("tap signature statement is invalid")
	// ErrSignaturePackageIDMismatch indicates the bundle package_id mismatches the tap bytes.
	ErrSignaturePackageIDMismatch = errors.New("tap signature bundle package_id mismatch")
	// ErrSignatureVerificationFailed indicates cryptographic signature verification failed.
	ErrSignatureVerificationFailed = errors.New("tap signature verification failed")
	// ErrMissingAuthorSignature indicates no author statement exists in the bundle.
	ErrMissingAuthorSignature = errors.New("tap signature bundle missing author statement")
)

var allowedTopLevelDirs = map[string]struct{}{
	"lib":     {},
	"tests":   {},
	"kernels": {},
	"models":  {},
	"assets":  {},
	"migrate": {},
}

var (
	migrateStepFilePattern = regexp.MustCompile(`^(\d+)_([^/]+)_to_([^/]+)\.tal$`)
	migrateLoadPattern     = regexp.MustCompile(`(?m)load\(\s*["']([^"']+)["']`)
)

var allowedMigrationModules = map[string]struct{}{
	"store":         {},
	"artifact.self": {},
	"log":           {},
	"migrate.env":   {},
}

// VerifiedTap is the pre-trust parsed output for a .tap archive.
type VerifiedTap struct {
	PackageID   string
	PackageName string
	Files       []string
}

// VerifiedStatement is one parsed and verified statement from a .tap.sig bundle.
type VerifiedStatement struct {
	Role            string
	KeyID           string
	CreatedUnix     uint64
	ManifestName    string
	ManifestVersion string
	Nonce           string
	Scope           map[string]any
}

// VerifiedPackage is the full pre-trust output for a .tap + .tap.sig pair.
type VerifiedPackage struct {
	Tap             VerifiedTap
	ManifestName    string
	ManifestVersion string
	Statements      []VerifiedStatement
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

	result, _, err := validateCanonicalTarWithManifest(canonicalTar)
	if err != nil {
		return VerifiedTap{}, err
	}

	hash := sha256.Sum256(canonicalTar)
	result.PackageID = "sha256:" + hex.EncodeToString(hash[:])
	return result, nil
}

// VerifyPackage validates both .tap archive bytes and a .tap.sig signature bundle.
func VerifyPackage(tapBytes []byte, sigBytes []byte) (VerifiedPackage, error) {
	canonicalTar, err := decompressTap(tapBytes)
	if err != nil {
		return VerifiedPackage{}, err
	}

	verifiedTap, manifestBytes, err := validateCanonicalTarWithManifest(canonicalTar)
	if err != nil {
		return VerifiedPackage{}, err
	}

	manifestName, manifestVersion, err := parseManifestIdentity(manifestBytes)
	if err != nil {
		return VerifiedPackage{}, err
	}

	hash := sha256.Sum256(canonicalTar)
	packageID := "sha256:" + hex.EncodeToString(hash[:])
	verifiedTap.PackageID = packageID

	statements, err := verifySignatureBundle(sigBytes, packageID, hash[:], manifestName, manifestVersion)
	if err != nil {
		return VerifiedPackage{}, err
	}

	return VerifiedPackage{
		Tap:             verifiedTap,
		ManifestName:    manifestName,
		ManifestVersion: manifestVersion,
		Statements:      statements,
	}, nil
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
	if err := validateCanonicalZstdFrame(tapBytes); err != nil {
		return nil, err
	}

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

func validateCanonicalZstdFrame(tapBytes []byte) error {
	if len(tapBytes) < 4 {
		return fmt.Errorf("%w: truncated zstd frame", ErrInvalidTapFormat)
	}

	magic := binary.LittleEndian.Uint32(tapBytes[:4])
	if magic&0xFFFFFFF0 == 0x184D2A50 {
		return fmt.Errorf("%w: skippable zstd frame not allowed", ErrInvalidTapFormat)
	}
	if magic != zstdFrameMagic {
		return fmt.Errorf("%w: unsupported zstd magic", ErrInvalidTapFormat)
	}

	offset := 4
	if len(tapBytes) < offset+1 {
		return fmt.Errorf("%w: truncated zstd frame header", ErrInvalidTapFormat)
	}
	descriptor := tapBytes[offset]
	offset++

	fcsFlag := descriptor >> 6
	singleSegment := descriptor&0x20 != 0
	if descriptor&0x04 != 0 {
		return fmt.Errorf("%w: zstd content checksum flag must be unset", ErrInvalidTapFormat)
	}
	if descriptor&0x03 != 0 {
		return fmt.Errorf("%w: zstd dictionary id flag must be unset", ErrInvalidTapFormat)
	}
	if fcsFlag == 0 {
		return fmt.Errorf("%w: zstd content size flag must be set", ErrInvalidTapFormat)
	}

	if !singleSegment {
		if len(tapBytes) < offset+1 {
			return fmt.Errorf("%w: truncated zstd window descriptor", ErrInvalidTapFormat)
		}
		windowDescriptor := tapBytes[offset]
		offset++

		windowLog := uint(windowDescriptor>>3) + 10
		if windowLog > 23 {
			return fmt.Errorf("%w: zstd window log too large", ErrInvalidTapFormat)
		}
		windowBase := uint64(1) << windowLog
		windowAdd := (windowBase / 8) * uint64(windowDescriptor&0x07)
		windowSize := windowBase + windowAdd
		if windowSize > uint64(zstdWindowSize) {
			return fmt.Errorf("%w: zstd window size too large", ErrInvalidTapFormat)
		}
	}

	contentSizeFieldLen := zstdContentSizeFieldLen(fcsFlag, singleSegment)
	if len(tapBytes) < offset+contentSizeFieldLen {
		return fmt.Errorf("%w: truncated zstd content size field", ErrInvalidTapFormat)
	}
	offset += contentSizeFieldLen

	for {
		if len(tapBytes) < offset+3 {
			return fmt.Errorf("%w: truncated zstd block header", ErrInvalidTapFormat)
		}
		blockHeader := uint32(tapBytes[offset]) | uint32(tapBytes[offset+1])<<8 | uint32(tapBytes[offset+2])<<16
		offset += 3

		lastBlock := blockHeader&0x1 != 0
		blockType := (blockHeader >> 1) & 0x3
		blockSize := int(blockHeader >> 3)

		switch blockType {
		case 0, 2:
			if len(tapBytes) < offset+blockSize {
				return fmt.Errorf("%w: truncated zstd block payload", ErrInvalidTapFormat)
			}
			offset += blockSize
		case 1:
			if len(tapBytes) < offset+1 {
				return fmt.Errorf("%w: truncated zstd RLE payload", ErrInvalidTapFormat)
			}
			offset++
		default:
			return fmt.Errorf("%w: reserved zstd block type", ErrInvalidTapFormat)
		}

		if lastBlock {
			break
		}
	}

	if offset != len(tapBytes) {
		return fmt.Errorf("%w: trailing bytes or extra frame detected", ErrInvalidTapFormat)
	}
	return nil
}

func zstdContentSizeFieldLen(fcsFlag byte, singleSegment bool) int {
	switch fcsFlag {
	case 0:
		if singleSegment {
			return 1
		}
		return 0
	case 1:
		return 2
	case 2:
		return 4
	default:
		return 8
	}
}

func validateCanonicalTarWithManifest(canonicalTar []byte) (VerifiedTap, []byte, error) {
	tr := tar.NewReader(bytes.NewReader(canonicalTar))
	seen := make(map[string]struct{})
	seenCaseFolded := make(map[string]struct{})
	files := make([]string, 0, 16)
	archiveSources := make(map[string][]byte)
	packageName := ""
	lastName := ""
	manifestCount := 0
	mainCount := 0
	var manifestBytes []byte

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return VerifiedTap{}, nil, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}

		if hdr.Typeflag != tar.TypeReg {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}
		if hdr.Name <= lastName {
			return VerifiedTap{}, nil, ErrNonCanonicalOrder
		}
		lastName = hdr.Name

		if err := validateArchivePath(hdr.Name); err != nil {
			return VerifiedTap{}, nil, err
		}
		if _, ok := seen[hdr.Name]; ok {
			return VerifiedTap{}, nil, ErrDuplicateArchivePath
		}
		seen[hdr.Name] = struct{}{}

		folded := strings.ToLower(hdr.Name)
		if _, ok := seenCaseFolded[folded]; ok {
			return VerifiedTap{}, nil, ErrCaseCollidingPath
		}
		seenCaseFolded[folded] = struct{}{}

		parts := strings.Split(hdr.Name, "/")
		if len(parts) < 2 {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}
		entryPackage := parts[0]
		if packageName == "" {
			packageName = entryPackage
		}
		if entryPackage != packageName {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}
		rel := strings.Join(parts[1:], "/")
		if err := validateTopLevelPath(rel); err != nil {
			return VerifiedTap{}, nil, err
		}

		if hdr.Mode != canonicalFileMode || hdr.Uid != 0 || hdr.Gid != 0 || hdr.Uname != "" || hdr.Gname != "" {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}
		if !hdr.ModTime.Equal(time.Unix(0, 0).UTC()) {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}

		payload, err := io.ReadAll(tr)
		if err != nil {
			return VerifiedTap{}, nil, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}
		if int64(len(payload)) != hdr.Size {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}

		switch rel {
		case "manifest.toml":
			manifestCount++
			manifestBytes = append([]byte(nil), payload...)
		case "main.tal":
			mainCount++
		}
		archiveSources[rel] = append([]byte(nil), payload...)
		files = append(files, rel)
	}

	if packageName == "" {
		return VerifiedTap{}, nil, fmt.Errorf("%w: no entries", ErrInvalidTapFormat)
	}
	if manifestCount != 1 {
		return VerifiedTap{}, nil, ErrMissingManifest
	}
	if mainCount != 1 {
		return VerifiedTap{}, nil, ErrMissingMainTAL
	}
	if err := validateManifestMigrations(manifestBytes, files, archiveSources); err != nil {
		return VerifiedTap{}, nil, err
	}

	return VerifiedTap{PackageName: packageName, Files: files}, manifestBytes, nil
}

type manifestIdentity struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type manifestMigration struct {
	Storage manifestStorageConfig   `toml:"storage"`
	Migrate manifestMigrationConfig `toml:"migrate"`
}

type manifestStorageConfig struct {
	StoreSchema []manifestStoreSchema `toml:"store_schema"`
}

type manifestStoreSchema struct {
	Store        string `toml:"store"`
	Version      string `toml:"version"`
	RecordSchema string `toml:"record_schema"`
}

type manifestMigrationConfig struct {
	DeclaredSteps int                        `toml:"declared_steps"`
	Step          []manifestMigrationStep    `toml:"step"`
	Fixture       []manifestMigrationFixture `toml:"fixture"`
}

type manifestMigrationStep struct {
	From          string `toml:"from"`
	To            string `toml:"to"`
	Compatibility string `toml:"compatibility"`
	DrainPolicy   string `toml:"drain_policy"`
}

type manifestMigrationFixture struct {
	Step              string `toml:"step"`
	PriorVersion      string `toml:"prior_version"`
	PriorRecordSchema string `toml:"prior_record_schema"`
	Seed              string `toml:"seed"`
	Expected          string `toml:"expected"`
}

type parsedMigrationStep struct {
	stepNumber int
	stepName   string
	from       string
	to         string
}

type migrationFixtureRecord struct {
	Key   string
	Value map[string]any
	Line  int
}

type signatureBundle struct {
	Schema    string               `toml:"schema"`
	PackageID string               `toml:"package_id"`
	Statement []signatureStatement `toml:"statement"`
}

type signatureStatement struct {
	Role            string         `toml:"role"`
	KeyID           string         `toml:"key_id"`
	CreatedUnix     uint64         `toml:"created_unix"`
	ManifestName    string         `toml:"manifest_name"`
	ManifestVersion string         `toml:"manifest_version"`
	Nonce           string         `toml:"nonce"`
	Scope           map[string]any `toml:"scope"`
	Sig             string         `toml:"sig"`
}

func parseManifestIdentity(manifestBytes []byte) (string, string, error) {
	var manifest manifestIdentity
	if _, err := toml.Decode(string(manifestBytes), &manifest); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return "", "", ErrInvalidManifest
	}
	return manifest.Name, manifest.Version, nil
}

func validateManifestMigrations(manifestBytes []byte, files []string, migrationSources map[string][]byte) error {
	var manifest manifestMigration
	if _, err := toml.Decode(string(manifestBytes), &manifest); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}

	availableFiles := make(map[string]struct{}, len(files))
	for _, rel := range files {
		availableFiles[rel] = struct{}{}
	}

	migrationFiles := make([]parsedMigrationStep, 0)
	for _, rel := range files {
		if !strings.HasPrefix(rel, "migrate/") {
			continue
		}
		name := strings.TrimPrefix(rel, "migrate/")
		if strings.HasPrefix(name, "downgrade/") {
			downgradeName := strings.TrimPrefix(name, "downgrade/")
			if strings.TrimSpace(downgradeName) == "" {
				return fmt.Errorf("%w: migration downgrade script path is empty", ErrInvalidManifest)
			}
			if strings.Contains(downgradeName, "/") {
				return fmt.Errorf("%w: migration downgrade script %s must be a single-level file under migrate/downgrade/", ErrInvalidManifest, rel)
			}
			if !strings.HasSuffix(downgradeName, ".tal") {
				return fmt.Errorf("%w: migration downgrade script %s must end with .tal", ErrInvalidManifest, rel)
			}
			continue
		}
		if strings.Contains(name, "/") {
			return fmt.Errorf("%w: migration script %s must be a single-level file under migrate/", ErrInvalidManifest, rel)
		}
		match := migrateStepFilePattern.FindStringSubmatch(name)
		if match == nil {
			return fmt.Errorf("%w: migration script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, rel)
		}
		stepNumber, err := strconv.Atoi(match[1])
		if err != nil || stepNumber <= 0 {
			return fmt.Errorf("%w: migration script %s has invalid step number", ErrInvalidManifest, rel)
		}
		migrationFiles = append(migrationFiles, parsedMigrationStep{stepNumber: stepNumber, stepName: strings.TrimSuffix(name, ".tal"), from: match[2], to: match[3]})
	}

	declaredSteps := manifest.Migrate.DeclaredSteps
	declaredManifestSteps := len(manifest.Migrate.Step)
	if len(migrationFiles) == 0 {
		if declaredSteps != 0 || declaredManifestSteps != 0 {
			return ErrInvalidManifest
		}
		return nil
	}

	if declaredSteps <= 0 || declaredSteps != declaredManifestSteps || declaredSteps != len(migrationFiles) {
		return ErrInvalidManifest
	}

	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].stepNumber < migrationFiles[j].stepNumber
	})

	for i, fileStep := range migrationFiles {
		if fileStep.stepNumber != i+1 {
			return fmt.Errorf("%w: migration step numbering gap: expected step %04d, found %04d", ErrInvalidManifest, i+1, fileStep.stepNumber)
		}
		manifestStep := manifest.Migrate.Step[i]
		if strings.TrimSpace(manifestStep.From) == "" || strings.TrimSpace(manifestStep.To) == "" {
			return ErrInvalidManifest
		}
		if manifestStep.Compatibility != "" && manifestStep.Compatibility != "compatible" && manifestStep.Compatibility != "incompatible" {
			return ErrInvalidManifest
		}
		if manifestStep.DrainPolicy != "" && manifestStep.DrainPolicy != "none" && manifestStep.DrainPolicy != "drain" && manifestStep.DrainPolicy != "multi_version" {
			return ErrInvalidManifest
		}
		if manifestStep.Compatibility == "incompatible" && manifestStep.DrainPolicy == "none" {
			return fmt.Errorf("%w: migrate.step %04d declares compatibility=incompatible with drain_policy=none", ErrInvalidManifest, i+1)
		}
		if manifestStep.From != fileStep.from || manifestStep.To != fileStep.to {
			return ErrInvalidManifest
		}
	}

	if len(manifest.Storage.StoreSchema) == 0 {
		return ErrInvalidManifest
	}
	for _, schema := range manifest.Storage.StoreSchema {
		if strings.TrimSpace(schema.Store) == "" || strings.TrimSpace(schema.Version) == "" || strings.TrimSpace(schema.RecordSchema) == "" {
			return ErrInvalidManifest
		}
		if _, ok := availableFiles[schema.RecordSchema]; !ok {
			return ErrInvalidManifest
		}
	}

	if len(manifest.Migrate.Fixture) != len(migrationFiles) {
		return ErrInvalidManifest
	}

	stepNames := make(map[string]struct{}, len(migrationFiles))
	stepByName := make(map[string]parsedMigrationStep, len(migrationFiles))
	for _, step := range migrationFiles {
		stepNames[step.stepName] = struct{}{}
		stepByName[step.stepName] = step
	}

	storeSchemaByVersion := make(map[string][]manifestStoreSchema, len(manifest.Storage.StoreSchema))
	for _, schema := range manifest.Storage.StoreSchema {
		version := strings.TrimSpace(schema.Version)
		storeSchemaByVersion[version] = append(storeSchemaByVersion[version], schema)
	}

	fixtureByStep := make(map[string]struct{}, len(manifest.Migrate.Fixture))
	for _, fixture := range manifest.Migrate.Fixture {
		if strings.TrimSpace(fixture.Step) == "" ||
			strings.TrimSpace(fixture.PriorVersion) == "" ||
			strings.TrimSpace(fixture.PriorRecordSchema) == "" ||
			strings.TrimSpace(fixture.Seed) == "" ||
			strings.TrimSpace(fixture.Expected) == "" {
			return ErrInvalidManifest
		}
		if _, ok := stepNames[fixture.Step]; !ok {
			return ErrInvalidManifest
		}
		if step, ok := stepByName[fixture.Step]; ok {
			if fixture.PriorVersion != step.from {
				return fmt.Errorf("%w: migrate.fixture %s prior_version %q does not match step from-version %q", ErrInvalidManifest, fixture.Step, fixture.PriorVersion, step.from)
			}
		}
		if _, ok := fixtureByStep[fixture.Step]; ok {
			return ErrInvalidManifest
		}
		fixtureByStep[fixture.Step] = struct{}{}
		if _, ok := availableFiles[fixture.PriorRecordSchema]; !ok {
			return ErrInvalidManifest
		}
		if _, ok := availableFiles[fixture.Seed]; !ok {
			return ErrInvalidManifest
		}
		if _, ok := availableFiles[fixture.Expected]; !ok {
			return ErrInvalidManifest
		}
		seedRecords, err := validateMigrationFixtureNDJSON(fixture.Seed, migrationSources[fixture.Seed])
		if err != nil {
			return err
		}
		if err := validateMigrationFixtureSeedSchema(fixture, seedRecords, migrationSources[fixture.PriorRecordSchema]); err != nil {
			return err
		}
		expectedRecords, err := validateMigrationFixtureNDJSON(fixture.Expected, migrationSources[fixture.Expected])
		if err != nil {
			return err
		}
		targetSchemaPath, targetSchemaPayload, shouldValidateExpected, err := resolveFixtureExpectedSchema(fixture, stepByName, storeSchemaByVersion, migrationSources)
		if err != nil {
			return err
		}
		if shouldValidateExpected {
			if err := validateMigrationFixtureValueSchema(fixture.Expected, targetSchemaPath, expectedRecords, targetSchemaPayload); err != nil {
				return err
			}
		}
	}

	for _, rel := range files {
		if !strings.HasPrefix(rel, "migrate/") {
			continue
		}
		source, ok := migrationSources[rel]
		if !ok {
			return ErrInvalidManifest
		}
		for _, match := range migrateLoadPattern.FindAllSubmatch(source, -1) {
			if len(match) < 2 {
				continue
			}
			module := strings.TrimSpace(string(match[1]))
			if _, allowed := allowedMigrationModules[module]; !allowed {
				return fmt.Errorf("%w: migration %s loads disallowed module %q", ErrInvalidManifest, rel, module)
			}
		}
	}

	return nil
}

func validateMigrationFixtureNDJSON(path string, payload []byte) ([]migrationFixtureRecord, error) {
	if payload == nil {
		return nil, fmt.Errorf("%w: migration fixture %s missing from archive payload", ErrInvalidManifest, path)
	}
	if bytes.Contains(payload, []byte{'\r'}) {
		return nil, fmt.Errorf("%w: migration fixture %s must use LF line endings", ErrInvalidManifest, path)
	}
	if len(payload) == 0 || payload[len(payload)-1] != '\n' {
		return nil, fmt.Errorf("%w: migration fixture %s must end with trailing LF", ErrInvalidManifest, path)
	}

	lines := bytes.Split(payload, []byte{'\n'})
	recordCount := len(lines) - 1
	if recordCount > migrationFixtureMaxRows {
		return nil, fmt.Errorf("%w: migration fixture %s exceeds max records (%d > %d)", ErrInvalidManifest, path, recordCount, migrationFixtureMaxRows)
	}
	records := make([]migrationFixtureRecord, 0, recordCount)
	seenKeys := make(map[string]struct{}, len(lines))
	var previousKey string

	for i := 0; i < len(lines)-1; i++ {
		lineNumber := i + 1
		line := lines[i]
		if len(line) == 0 {
			return nil, fmt.Errorf("%w: migration fixture %s line %d is blank", ErrInvalidManifest, path, lineNumber)
		}

		canonical, key, value, err := parseCanonicalFixtureRecord(line)
		if err != nil {
			return nil, fmt.Errorf("%w: migration fixture %s line %d: %v", ErrInvalidManifest, path, lineNumber, err)
		}
		if !bytes.Equal(line, canonical) {
			return nil, fmt.Errorf("%w: migration fixture %s line %d is not canonical JSON", ErrInvalidManifest, path, lineNumber)
		}
		if _, dup := seenKeys[key]; dup {
			return nil, fmt.Errorf("%w: migration fixture %s line %d has duplicate key %q", ErrInvalidManifest, path, lineNumber, key)
		}
		if previousKey != "" && bytes.Compare([]byte(previousKey), []byte(key)) >= 0 {
			return nil, fmt.Errorf("%w: migration fixture %s line %d is out of key order", ErrInvalidManifest, path, lineNumber)
		}

		seenKeys[key] = struct{}{}
		previousKey = key
		records = append(records, migrationFixtureRecord{Key: key, Value: value, Line: lineNumber})
	}

	return records, nil
}

func validateMigrationFixtureSeedSchema(fixture manifestMigrationFixture, records []migrationFixtureRecord, schemaPayload []byte) error {
	if schemaPayload == nil {
		return fmt.Errorf("%w: migration fixture %s prior schema %s missing from archive payload", ErrInvalidManifest, fixture.Seed, fixture.PriorRecordSchema)
	}
	return validateMigrationFixtureValueSchema(fixture.Seed, fixture.PriorRecordSchema, records, schemaPayload)
}

func validateMigrationFixtureValueSchema(fixturePath string, schemaPath string, records []migrationFixtureRecord, schemaPayload []byte) error {
	if schemaPayload == nil {
		return fmt.Errorf("%w: migration fixture %s schema %s missing from archive payload", ErrInvalidManifest, fixturePath, schemaPath)
	}
	compiler := jsonschema.NewCompiler()
	schemaURL := "memory://" + schemaPath
	var schemaDoc any
	if err := json.Unmarshal(schemaPayload, &schemaDoc); err != nil {
		return fmt.Errorf("%w: migration schema %s is invalid JSON: %v", ErrInvalidManifest, schemaPath, err)
	}
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("%w: migration schema %s is invalid: %v", ErrInvalidManifest, schemaPath, err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("%w: migration schema %s is invalid: %v", ErrInvalidManifest, schemaPath, err)
	}
	for _, record := range records {
		if err := schema.Validate(record.Value); err != nil {
			return fmt.Errorf("%w: migration fixture %s line %d key %q violates schema %s: %v", ErrInvalidManifest, fixturePath, record.Line, record.Key, schemaPath, err)
		}
	}
	return nil
}

func resolveFixtureExpectedSchema(
	fixture manifestMigrationFixture,
	stepByName map[string]parsedMigrationStep,
	storeSchemaByVersion map[string][]manifestStoreSchema,
	migrationSources map[string][]byte,
) (schemaPath string, schemaPayload []byte, shouldValidate bool, err error) {
	step, ok := stepByName[fixture.Step]
	if !ok {
		return "", nil, false, ErrInvalidManifest
	}

	candidateSchemas := storeSchemaByVersion[strings.TrimSpace(step.to)]
	if len(candidateSchemas) == 0 {
		// Expected schema validation is optional until every package declares
		// per-target-version record schemas for migration fixtures.
		return "", nil, false, nil
	}
	if len(candidateSchemas) > 1 {
		return "", nil, false, fmt.Errorf("%w: migrate.fixture %s expected schema is ambiguous for target version %q", ErrInvalidManifest, fixture.Step, step.to)
	}

	schemaPath = strings.TrimSpace(candidateSchemas[0].RecordSchema)
	schemaPayload = migrationSources[schemaPath]
	if schemaPayload == nil {
		return "", nil, false, fmt.Errorf("%w: migrate.fixture %s expected schema %s missing from archive payload", ErrInvalidManifest, fixture.Step, schemaPath)
	}
	return schemaPath, schemaPayload, true, nil
}

func parseCanonicalFixtureRecord(line []byte) ([]byte, string, map[string]any, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(line, &envelope); err != nil {
		return nil, "", nil, fmt.Errorf("invalid JSON object")
	}
	if len(envelope) != 2 {
		return nil, "", nil, fmt.Errorf("fixture record must contain exactly key and value fields")
	}

	rawKey, ok := envelope["key"]
	if !ok {
		return nil, "", nil, fmt.Errorf("fixture record missing key field")
	}
	rawValue, ok := envelope["value"]
	if !ok {
		return nil, "", nil, fmt.Errorf("fixture record missing value field")
	}

	var key string
	if err := json.Unmarshal(rawKey, &key); err != nil {
		return nil, "", nil, fmt.Errorf("fixture key must be a string")
	}
	if !utf8.ValidString(key) {
		return nil, "", nil, fmt.Errorf("fixture key must be valid UTF-8")
	}
	if !norm.NFC.IsNormalString(key) {
		return nil, "", nil, fmt.Errorf("fixture key must be NFC normalized")
	}
	if len([]byte(key)) == 0 || len([]byte(key)) > 256 {
		return nil, "", nil, fmt.Errorf("fixture key byte length must be 1..256")
	}

	var value map[string]any
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return nil, "", nil, fmt.Errorf("fixture value must be an object")
	}

	canonical, err := json.Marshal(map[string]any{
		"key":   key,
		"value": value,
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to canonicalize fixture record")
	}

	return canonical, key, value, nil
}

func verifySignatureBundle(sigBytes []byte, expectedPackageID string, packageHash []byte, expectedManifestName string, expectedManifestVersion string) ([]VerifiedStatement, error) {
	if len(sigBytes) == 0 || len(sigBytes) > signatureBundleMaxBytes {
		return nil, ErrInvalidSignatureBundle
	}

	var bundle signatureBundle
	if _, err := toml.Decode(string(sigBytes), &bundle); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureBundle, err)
	}

	if bundle.Schema != signatureBundleSchema {
		return nil, ErrInvalidSignatureBundle
	}
	if err := requireMaxStringLen(bundle.PackageID); err != nil {
		return nil, err
	}
	if bundle.PackageID != expectedPackageID {
		return nil, ErrSignaturePackageIDMismatch
	}
	if len(bundle.Statement) == 0 || len(bundle.Statement) > statementMaxCount {
		return nil, ErrInvalidSignatureBundle
	}

	seen := make(map[string]struct{}, len(bundle.Statement))
	authorSeen := false
	verified := make([]VerifiedStatement, 0, len(bundle.Statement))

	for _, stmt := range bundle.Statement {
		if err := validateStatementFields(stmt); err != nil {
			return nil, err
		}
		nonceRaw, err := parsePrefixedBase64URL(stmt.Nonce, "base64url:", statementNonceLen)
		if err != nil {
			return nil, err
		}
		sigRaw, err := parsePrefixedBase64(stmt.Sig, "base64:")
		if err != nil {
			return nil, err
		}
		publicKey, err := parseKeyID(stmt.KeyID)
		if err != nil {
			return nil, err
		}
		scope, err := normalizeScope(stmt.Role, stmt.Scope)
		if err != nil {
			return nil, err
		}

		payload, err := encodeStatementCBOR(stmt, scope, packageHash, nonceRaw)
		if err != nil {
			return nil, err
		}
		if !ed25519.Verify(publicKey, payload, sigRaw) {
			return nil, ErrSignatureVerificationFailed
		}

		if stmt.ManifestName != expectedManifestName || stmt.ManifestVersion != expectedManifestVersion {
			return nil, ErrInvalidSignatureStatement
		}

		nonceKey := base64.RawURLEncoding.EncodeToString(nonceRaw)
		triple := stmt.KeyID + "\x00" + expectedPackageID + "\x00" + nonceKey
		if _, ok := seen[triple]; ok {
			return nil, ErrInvalidSignatureStatement
		}
		seen[triple] = struct{}{}

		if stmt.Role == "author" {
			authorSeen = true
		}

		verified = append(verified, VerifiedStatement{
			Role:            stmt.Role,
			KeyID:           stmt.KeyID,
			CreatedUnix:     stmt.CreatedUnix,
			ManifestName:    stmt.ManifestName,
			ManifestVersion: stmt.ManifestVersion,
			Nonce:           stmt.Nonce,
			Scope:           scope,
		})
	}

	if !authorSeen {
		return nil, ErrMissingAuthorSignature
	}

	return verified, nil
}

func validateStatementFields(stmt signatureStatement) error {
	if stmt.Role == "" || stmt.KeyID == "" || stmt.ManifestName == "" || stmt.ManifestVersion == "" || stmt.Nonce == "" || stmt.Sig == "" {
		return ErrInvalidSignatureStatement
	}
	if err := requireMaxStringLen(stmt.Role, stmt.KeyID, stmt.ManifestName, stmt.ManifestVersion, stmt.Nonce, stmt.Sig); err != nil {
		return err
	}
	return nil
}

func requireMaxStringLen(values ...string) error {
	for _, value := range values {
		if len(value) > stringFieldMaxBytes {
			return ErrInvalidSignatureStatement
		}
	}
	return nil
}

func parsePrefixedBase64URL(raw string, prefix string, expectedLen int) ([]byte, error) {
	if !strings.HasPrefix(raw, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return nil, ErrInvalidSignatureStatement
	}
	if len(decoded) != expectedLen {
		return nil, ErrInvalidSignatureStatement
	}
	return decoded, nil
}

func parsePrefixedBase64(raw string, prefix string) ([]byte, error) {
	if !strings.HasPrefix(raw, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return nil, ErrInvalidSignatureStatement
	}
	if len(decoded) == 0 {
		return nil, ErrInvalidSignatureStatement
	}
	return decoded, nil
}

func parseKeyID(keyID string) (ed25519.PublicKey, error) {
	const prefix = "ed25519:"
	if !strings.HasPrefix(keyID, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	encoded := strings.TrimPrefix(keyID, prefix)
	publicKey, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		publicKey, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, ErrInvalidSignatureStatement
	}
	return ed25519.PublicKey(publicKey), nil
}

func normalizeScope(role string, input map[string]any) (map[string]any, error) {
	if input == nil {
		input = map[string]any{}
	}

	switch role {
	case "author":
		if len(input) != 0 {
			return nil, ErrInvalidSignatureStatement
		}
		return map[string]any{}, nil
	case "voucher":
		allowed := map[string]struct{}{"tier": {}, "reviewed": {}, "tested_under": {}, "notes": {}, "expires_unix": {}}
		for key := range input {
			if _, ok := allowed[key]; !ok {
				return nil, ErrInvalidSignatureStatement
			}
		}

		tier, ok := input["tier"].(string)
		if !ok || (tier != "full" && tier != "quarantine" && tier != "custom") {
			return nil, ErrInvalidSignatureStatement
		}

		reviewedRaw, ok := input["reviewed"].([]any)
		if !ok {
			return nil, ErrInvalidSignatureStatement
		}
		reviewed := make([]string, 0, len(reviewedRaw))
		for _, value := range reviewedRaw {
			v, ok := value.(string)
			if !ok {
				return nil, ErrInvalidSignatureStatement
			}
			switch v {
			case "manifest", "tal", "tests", "kernels", "models", "assets":
				reviewed = append(reviewed, v)
			default:
				return nil, ErrInvalidSignatureStatement
			}
		}

		testedUnder, ok := input["tested_under"].(string)
		if !ok || (testedUnder != "sim-only" && testedUnder != "hardware" && testedUnder != "production") {
			return nil, ErrInvalidSignatureStatement
		}

		normalized := map[string]any{
			"tier":         tier,
			"reviewed":     reviewed,
			"tested_under": testedUnder,
		}
		if notesRaw, ok := input["notes"]; ok {
			notes, ok := notesRaw.(string)
			if !ok || len(notes) > voucherNotesMaxBytes {
				return nil, ErrInvalidSignatureStatement
			}
			normalized["notes"] = notes
		}
		if expiresRaw, ok := input["expires_unix"]; ok {
			expiresUnix, ok := asUint64(expiresRaw)
			if !ok {
				return nil, ErrInvalidSignatureStatement
			}
			normalized["expires_unix"] = expiresUnix
		}
		return normalized, nil
	case "publisher":
		allowed := map[string]struct{}{"via": {}}
		for key := range input {
			if _, ok := allowed[key]; !ok {
				return nil, ErrInvalidSignatureStatement
			}
		}
		via, ok := input["via"].(string)
		if !ok || via == "" || len(via) > publisherViaMaxBytes {
			return nil, ErrInvalidSignatureStatement
		}
		return map[string]any{"via": via}, nil
	default:
		return nil, ErrInvalidSignatureStatement
	}
}

func asUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case uint64:
		return v, true
	case float64:
		if v < 0 || float64(uint64(v)) != v {
			return 0, false
		}
		return uint64(v), true
	default:
		return 0, false
	}
}

func encodeStatementCBOR(stmt signatureStatement, normalizedScope map[string]any, packageHash []byte, nonceRaw []byte) ([]byte, error) {
	encMode, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureStatement, err)
	}

	payload := map[uint64]any{
		0: uint64(1),
		1: append([]byte(nil), packageHash...),
		2: stmt.Role,
		3: stmt.KeyID,
		4: stmt.CreatedUnix,
		5: stmt.ManifestName,
		6: stmt.ManifestVersion,
		7: normalizedScope,
		8: append([]byte(nil), nonceRaw...),
	}

	b, err := encMode.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureStatement, err)
	}
	return b, nil
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
