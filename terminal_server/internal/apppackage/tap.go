// Package apppackage builds and validates canonical .tap application archives.
package apppackage

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
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
	"github.com/klauspost/compress/zstd"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/unicode/norm"
)

// CONTENTS:
//   line  31  const ( // canonicalFileMode, zstd constants, signature bundle constants
//   line  46  var ( // ErrInvalidTapFormat, ErrMissingManifest, and other sentinel errors
//   line  79  var allowedTopLevelDirs
//   line  88  var ( // migrateStepFilePattern, migrateLoadPattern, migrateReadPattern, migrateReadAdapterIdentityReturnPattern
//   line  95  var allowedMigrationModules
//   line 102  type VerifiedTap struct
//   line 109  type VerifiedStatement struct
//   line 120  type VerifiedPackage struct
//   line 128  func BuildTapFromDir(root string) ([]byte, string, error)
//   line 157  func VerifyTap(tapBytes []byte) (VerifiedTap, error)
//   line 174  func VerifyPackage(tapBytes []byte, sigBytes []byte) (VerifiedPackage, error)
//   line 208  func collectSourceFiles(root string) ([]string, error)
//   line 264  func buildCanonicalTar(root string, packageName string, relPaths []string) ([]byte, error)
//   line 304  func compressCanonicalTar(canonicalTar []byte) ([]byte, error)
//   line 321  func decompressTap(tapBytes []byte) ([]byte, error)
//   line 342  func validateCanonicalZstdFrame(tapBytes []byte) error
//   line 436  func zstdContentSizeFieldLen(fcsFlag byte, singleSegment bool) int
//   line 452  func validateCanonicalTarWithManifest(canonicalTar []byte) (VerifiedTap, []byte, error)
//   line 553  type manifestIdentity struct
//   line 558  type manifestMigration struct
//   line 563  type manifestStorageConfig struct
//   line 567  type manifestStoreSchema struct
//   line 573  type manifestMigrationConfig struct
//   line 582  type manifestMigrationStep struct
//   line 589  type manifestMigrationFixture struct
//   line 598  type parsedMigrationStep struct
//   line 605  type migrationFixtureRecord struct
//   line 611  type signatureBundle struct
//   line 617  type signatureStatement struct
//   line 628  func parseManifestIdentity(manifestBytes []byte) (string, string, error)
//   line 639  func validateManifestMigrations(manifestBytes []byte, files []string, migrationSources map[string][]byte) error
//   line 862  func validateMigrationFixtureNDJSON(path string, payload []byte) ([]migrationFixtureRecord, error)
//   line 911  func validateMigrationFixtureSeedSchema(fixture manifestMigrationFixture, records []migrationFixtureRecord, schemaPayload []byte) error
//   line 918  func validateMigrationFixtureValueSchema(fixturePath string, schemaPath string, records []migrationFixtureRecord, schemaPayload []byte) error
//   line 943  func resolveFixtureExpectedSchema(fixture manifestMigrationFixture, manifestStep manifestMigrationStep, stepByName map[string]parsedMigrationStep, storeSchemaByVersion map[string][]manifestStoreSchema, migrationSources map[string][]byte) (schemaPath string, schemaPayload []byte, shouldValidate bool, err error)
//   line 976  func validateMigrationReadAdapterSource(path string, payload []byte) error
//   line 1004 func stripTALLineComment(line string) string
//   line 1032 func parseCanonicalFixtureRecord(line []byte) ([]byte, string, map[string]any, error)
//   line 1080 func validateArchivePath(name string) error
//   line 1099 func validateRelativeArchivePath(rel string) error
//   line 1106 func validatePathComponent(component string) error
//   line 1116 func validateTopLevelPath(rel string) error

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
	migrateStepFilePattern                  = regexp.MustCompile(`^(\d+)_([^/]+)_to_([^/]+)\.tal$`)
	migrateLoadPattern                      = regexp.MustCompile(`(?m)^\s*load\(\s*["']([^"']+)["']`)
	migrateReadPattern                      = regexp.MustCompile(`(?m)^\s*def\s+read\s*\(\s*record\s*\)`)
	migrateReadAdapterIdentityReturnPattern = regexp.MustCompile(`^\s*return\s+record\s*$`)
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

type canonicalTarValidationState struct {
	seen           map[string]struct{}
	seenCaseFolded map[string]struct{}
	files          []string
	archiveSources map[string][]byte
	packageName    string
	lastName       string
	manifestCount  int
	mainCount      int
	manifestBytes  []byte
}

type canonicalTarEntry struct {
	rel     string
	payload []byte
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
	collector := sourceFileCollector{root: root, relPaths: make([]string, 0, 16)}
	err := filepath.WalkDir(root, collector.visit)
	if err != nil {
		return nil, err
	}

	hasManifest := false
	hasMain := false
	for _, rel := range collector.relPaths {
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

	sort.Strings(collector.relPaths)
	return collector.relPaths, nil
}

type sourceFileCollector struct {
	root     string
	relPaths []string
}

func (c *sourceFileCollector) visit(current string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil || current == c.root {
		return walkErr
	}
	rel, err := filepath.Rel(c.root, current)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if d.IsDir() {
		return validateSourceArchivePath(rel)
	}
	if err := validateSourceFileEntry(rel, d); err != nil {
		return err
	}
	c.relPaths = append(c.relPaths, rel)
	return nil
}

func validateSourceFileEntry(rel string, d fs.DirEntry) error {
	if err := validateSourceArchivePath(rel); err != nil {
		return err
	}
	if !d.Type().IsRegular() {
		return fmt.Errorf("%w: %s", ErrInvalidTarEntry, rel)
	}
	return validateTopLevelPath(rel)
}

func validateSourceArchivePath(rel string) error {
	if err := validateRelativeArchivePath(rel); err != nil {
		return fmt.Errorf("%w: %s", ErrPathTraversalDetected, rel)
	}
	return nil
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
	if err := validateZstdFrameDescriptor(descriptor, fcsFlag); err != nil {
		return err
	}

	if !singleSegment {
		nextOffset, err := validateZstdWindowDescriptor(tapBytes, offset)
		if err != nil {
			return err
		}
		offset = nextOffset
	}

	contentSizeFieldLen := zstdContentSizeFieldLen(fcsFlag, singleSegment)
	if len(tapBytes) < offset+contentSizeFieldLen {
		return fmt.Errorf("%w: truncated zstd content size field", ErrInvalidTapFormat)
	}
	offset += contentSizeFieldLen

	for {
		nextOffset, lastBlock, err := validateZstdBlock(tapBytes, offset)
		if err != nil {
			return err
		}
		offset = nextOffset
		if lastBlock {
			break
		}
	}

	if offset != len(tapBytes) {
		return fmt.Errorf("%w: trailing bytes or extra frame detected", ErrInvalidTapFormat)
	}
	return nil
}

func validateZstdFrameDescriptor(descriptor byte, fcsFlag byte) error {
	if descriptor&0x04 != 0 {
		return fmt.Errorf("%w: zstd content checksum flag must be unset", ErrInvalidTapFormat)
	}
	if descriptor&0x03 != 0 {
		return fmt.Errorf("%w: zstd dictionary id flag must be unset", ErrInvalidTapFormat)
	}
	if fcsFlag == 0 {
		return fmt.Errorf("%w: zstd content size flag must be set", ErrInvalidTapFormat)
	}
	return nil
}

func validateZstdWindowDescriptor(tapBytes []byte, offset int) (int, error) {
	if len(tapBytes) < offset+1 {
		return 0, fmt.Errorf("%w: truncated zstd window descriptor", ErrInvalidTapFormat)
	}
	windowDescriptor := tapBytes[offset]
	windowLog := uint(windowDescriptor>>3) + 10
	if windowLog > 23 {
		return 0, fmt.Errorf("%w: zstd window log too large", ErrInvalidTapFormat)
	}
	windowBase := uint64(1) << windowLog
	windowAdd := (windowBase / 8) * uint64(windowDescriptor&0x07)
	if windowBase+windowAdd > uint64(zstdWindowSize) {
		return 0, fmt.Errorf("%w: zstd window size too large", ErrInvalidTapFormat)
	}
	return offset + 1, nil
}

func validateZstdBlock(tapBytes []byte, offset int) (int, bool, error) {
	if len(tapBytes) < offset+3 {
		return 0, false, fmt.Errorf("%w: truncated zstd block header", ErrInvalidTapFormat)
	}
	blockHeader := uint32(tapBytes[offset]) | uint32(tapBytes[offset+1])<<8 | uint32(tapBytes[offset+2])<<16
	offset += 3

	lastBlock := blockHeader&0x1 != 0
	blockType := (blockHeader >> 1) & 0x3
	blockSize := int(blockHeader >> 3)

	switch blockType {
	case 0, 2:
		return validateZstdSizedBlock(tapBytes, offset, blockSize, lastBlock)
	case 1:
		return validateZstdRLEBlock(tapBytes, offset, lastBlock)
	default:
		return 0, false, fmt.Errorf("%w: reserved zstd block type", ErrInvalidTapFormat)
	}
}

func validateZstdSizedBlock(tapBytes []byte, offset int, blockSize int, lastBlock bool) (int, bool, error) {
	if len(tapBytes) < offset+blockSize {
		return 0, false, fmt.Errorf("%w: truncated zstd block payload", ErrInvalidTapFormat)
	}
	return offset + blockSize, lastBlock, nil
}

func validateZstdRLEBlock(tapBytes []byte, offset int, lastBlock bool) (int, bool, error) {
	if len(tapBytes) < offset+1 {
		return 0, false, fmt.Errorf("%w: truncated zstd RLE payload", ErrInvalidTapFormat)
	}
	return offset + 1, lastBlock, nil
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
	state := canonicalTarValidationState{
		seen:           make(map[string]struct{}),
		seenCaseFolded: make(map[string]struct{}),
		files:          make([]string, 0, 16),
		archiveSources: make(map[string][]byte),
	}

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return VerifiedTap{}, nil, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}
		entry, err := state.validateEntryHeader(hdr)
		if err != nil {
			return VerifiedTap{}, nil, err
		}
		payload, err := io.ReadAll(tr)
		if err != nil {
			return VerifiedTap{}, nil, fmt.Errorf("%w: %v", ErrInvalidTarEntry, err)
		}
		if int64(len(payload)) != hdr.Size {
			return VerifiedTap{}, nil, ErrInvalidTarEntry
		}
		entry.payload = payload
		state.addEntry(entry)
	}

	if state.packageName == "" {
		return VerifiedTap{}, nil, fmt.Errorf("%w: no entries", ErrInvalidTapFormat)
	}
	if state.manifestCount != 1 {
		return VerifiedTap{}, nil, ErrMissingManifest
	}
	if state.mainCount != 1 {
		return VerifiedTap{}, nil, ErrMissingMainTAL
	}
	if err := validateManifestMigrations(state.manifestBytes, state.files, state.archiveSources); err != nil {
		return VerifiedTap{}, nil, err
	}

	return VerifiedTap{PackageName: state.packageName, Files: state.files}, state.manifestBytes, nil
}

func (s *canonicalTarValidationState) validateEntryHeader(hdr *tar.Header) (canonicalTarEntry, error) {
	if hdr.Typeflag != tar.TypeReg {
		return canonicalTarEntry{}, ErrInvalidTarEntry
	}
	if hdr.Name <= s.lastName {
		return canonicalTarEntry{}, ErrNonCanonicalOrder
	}
	s.lastName = hdr.Name
	if err := s.validateEntryPath(hdr.Name); err != nil {
		return canonicalTarEntry{}, err
	}
	if hdr.Mode != canonicalFileMode || hdr.Uid != 0 || hdr.Gid != 0 || hdr.Uname != "" || hdr.Gname != "" {
		return canonicalTarEntry{}, ErrInvalidTarEntry
	}
	if !hdr.ModTime.Equal(time.Unix(0, 0).UTC()) {
		return canonicalTarEntry{}, ErrInvalidTarEntry
	}
	rel, err := s.entryRelativePath(hdr.Name)
	if err != nil {
		return canonicalTarEntry{}, err
	}
	return canonicalTarEntry{rel: rel}, nil
}

func (s *canonicalTarValidationState) validateEntryPath(name string) error {
	if err := validateArchivePath(name); err != nil {
		return err
	}
	if _, ok := s.seen[name]; ok {
		return ErrDuplicateArchivePath
	}
	s.seen[name] = struct{}{}

	folded := strings.ToLower(name)
	if _, ok := s.seenCaseFolded[folded]; ok {
		return ErrCaseCollidingPath
	}
	s.seenCaseFolded[folded] = struct{}{}
	return nil
}

func (s *canonicalTarValidationState) entryRelativePath(name string) (string, error) {
	parts := strings.Split(name, "/")
	if len(parts) < 2 {
		return "", ErrInvalidTarEntry
	}
	entryPackage := parts[0]
	if s.packageName == "" {
		s.packageName = entryPackage
	}
	if entryPackage != s.packageName {
		return "", ErrInvalidTarEntry
	}
	rel := strings.Join(parts[1:], "/")
	if err := validateTopLevelPath(rel); err != nil {
		return "", err
	}
	return rel, nil
}

func (s *canonicalTarValidationState) addEntry(entry canonicalTarEntry) {
	payload := append([]byte(nil), entry.payload...)
	switch entry.rel {
	case "manifest.toml":
		s.manifestCount++
		s.manifestBytes = payload
	case "main.tal":
		s.mainCount++
	}
	s.archiveSources[entry.rel] = payload
	s.files = append(s.files, entry.rel)
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
	DeclaredSteps       int                        `toml:"declared_steps"`
	DrainTimeoutSeconds *int                       `toml:"drain_timeout_seconds"`
	MaxRuntimeSeconds   *int                       `toml:"max_runtime_seconds"`
	CheckpointEvery     *int                       `toml:"checkpoint_every"`
	Step                []manifestMigrationStep    `toml:"step"`
	Fixture             []manifestMigrationFixture `toml:"fixture"`
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
	ReadAdapter       string `toml:"read_adapter"`
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

	migrationFiles, err := parseManifestMigrationFiles(files)
	if err != nil {
		return err
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
	if err := validateManifestMigrationLimits(manifest.Migrate); err != nil {
		return err
	}

	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].stepNumber < migrationFiles[j].stepNumber
	})

	if err := validateManifestMigrationSteps(manifest.Migrate.Step, migrationFiles); err != nil {
		return err
	}
	if err := validateManifestStoreSchemas(manifest.Storage.StoreSchema, availableFiles); err != nil {
		return err
	}

	if len(manifest.Migrate.Fixture) != len(migrationFiles) {
		return ErrInvalidManifest
	}

	stepNames, stepByName := indexManifestMigrationSteps(migrationFiles)
	storeSchemaByVersion := indexManifestStoreSchemasByVersion(manifest.Storage.StoreSchema)

	if err := validateManifestMigrationFixtures(manifest.Migrate.Fixture, manifest.Migrate.Step, stepNames, stepByName, storeSchemaByVersion, availableFiles, migrationSources); err != nil {
		return err
	}

	return validateManifestMigrationModules(files, migrationSources)
}

func indexManifestMigrationSteps(migrationFiles []parsedMigrationStep) (map[string]struct{}, map[string]parsedMigrationStep) {
	stepNames := make(map[string]struct{}, len(migrationFiles))
	stepByName := make(map[string]parsedMigrationStep, len(migrationFiles))
	for _, step := range migrationFiles {
		stepNames[step.stepName] = struct{}{}
		stepByName[step.stepName] = step
	}
	return stepNames, stepByName
}

func indexManifestStoreSchemasByVersion(schemas []manifestStoreSchema) map[string][]manifestStoreSchema {
	storeSchemaByVersion := make(map[string][]manifestStoreSchema, len(schemas))
	for _, schema := range schemas {
		version := strings.TrimSpace(schema.Version)
		storeSchemaByVersion[version] = append(storeSchemaByVersion[version], schema)
	}
	return storeSchemaByVersion
}

func parseManifestMigrationFiles(files []string) ([]parsedMigrationStep, error) {
	migrationFiles := make([]parsedMigrationStep, 0)
	for _, rel := range files {
		step, include, err := parseManifestMigrationFile(rel)
		if err != nil {
			return nil, err
		}
		if include {
			migrationFiles = append(migrationFiles, step)
		}
	}
	return migrationFiles, nil
}

func parseManifestMigrationFile(rel string) (parsedMigrationStep, bool, error) {
	if !strings.HasPrefix(rel, "migrate/") {
		return parsedMigrationStep{}, false, nil
	}
	name := strings.TrimPrefix(rel, "migrate/")
	if strings.HasPrefix(name, "downgrade/") {
		return parsedMigrationStep{}, false, validateManifestMigrationDowngradeFile(rel, strings.TrimPrefix(name, "downgrade/"))
	}
	step, err := parseManifestMigrationStepFile(rel, name, "migrate/")
	return step, true, err
}

func validateManifestMigrationDowngradeFile(rel string, name string) error {
	_, err := parseManifestMigrationStepFileWithMode(rel, name, "migrate/downgrade/", true)
	return err
}

func parseManifestMigrationStepFile(rel string, name string, dir string) (parsedMigrationStep, error) {
	return parseManifestMigrationStepFileWithMode(rel, name, dir, false)
}

func parseManifestMigrationStepFileWithMode(rel string, name string, dir string, downgrade bool) (parsedMigrationStep, error) {
	if strings.TrimSpace(name) == "" {
		if downgrade {
			return parsedMigrationStep{}, fmt.Errorf("%w: migration downgrade script path is empty", ErrInvalidManifest)
		}
		return parsedMigrationStep{}, fmt.Errorf("%w: migration script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, rel)
	}
	if strings.Contains(name, "/") {
		if downgrade {
			return parsedMigrationStep{}, fmt.Errorf("%w: migration downgrade script %s must be a single-level file under migrate/downgrade/", ErrInvalidManifest, rel)
		}
		return parsedMigrationStep{}, fmt.Errorf("%w: migration script %s must be a single-level file under %s", ErrInvalidManifest, rel, dir)
	}
	match := migrateStepFilePattern.FindStringSubmatch(name)
	if match == nil {
		if downgrade {
			return parsedMigrationStep{}, fmt.Errorf("%w: migration downgrade script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, rel)
		}
		return parsedMigrationStep{}, fmt.Errorf("%w: migration script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, rel)
	}
	stepNumber, err := strconv.Atoi(match[1])
	if err != nil || stepNumber <= 0 {
		if downgrade {
			return parsedMigrationStep{}, fmt.Errorf("%w: migration downgrade script %s has invalid step number", ErrInvalidManifest, rel)
		}
		return parsedMigrationStep{}, fmt.Errorf("%w: migration script %s has invalid step number", ErrInvalidManifest, rel)
	}
	return parsedMigrationStep{stepNumber: stepNumber, stepName: strings.TrimSuffix(name, ".tal"), from: match[2], to: match[3]}, nil
}

func validateManifestMigrationLimits(migrate manifestMigrationConfig) error {
	if migrate.DrainTimeoutSeconds != nil && *migrate.DrainTimeoutSeconds <= 0 {
		return fmt.Errorf("%w: migrate.drain_timeout_seconds must be a positive integer", ErrInvalidManifest)
	}
	if migrate.MaxRuntimeSeconds != nil && *migrate.MaxRuntimeSeconds <= 0 {
		return fmt.Errorf("%w: migrate.max_runtime_seconds must be a positive integer", ErrInvalidManifest)
	}
	if migrate.CheckpointEvery != nil && *migrate.CheckpointEvery <= 0 {
		return fmt.Errorf("%w: migrate.checkpoint_every must be a positive integer", ErrInvalidManifest)
	}
	return nil
}

func validateManifestMigrationSteps(manifestSteps []manifestMigrationStep, migrationFiles []parsedMigrationStep) error {
	for i, fileStep := range migrationFiles {
		if fileStep.stepNumber != i+1 {
			return fmt.Errorf("%w: migration step numbering gap: expected step %04d, found %04d", ErrInvalidManifest, i+1, fileStep.stepNumber)
		}
		if err := validateManifestMigrationStep(i+1, manifestSteps[i], fileStep); err != nil {
			return err
		}
	}
	return nil
}

func validateManifestMigrationStep(stepNumber int, manifestStep manifestMigrationStep, fileStep parsedMigrationStep) error {
	if strings.TrimSpace(manifestStep.From) == "" || strings.TrimSpace(manifestStep.To) == "" {
		return ErrInvalidManifest
	}
	if strings.TrimSpace(manifestStep.Compatibility) == "" {
		return fmt.Errorf("%w: migrate.step %04d must declare compatibility", ErrInvalidManifest, stepNumber)
	}
	if strings.TrimSpace(manifestStep.DrainPolicy) == "" {
		return fmt.Errorf("%w: migrate.step %04d must declare drain_policy", ErrInvalidManifest, stepNumber)
	}
	if manifestStep.Compatibility != "compatible" && manifestStep.Compatibility != "incompatible" {
		return ErrInvalidManifest
	}
	if manifestStep.DrainPolicy != "none" && manifestStep.DrainPolicy != "drain" && manifestStep.DrainPolicy != "multi_version" {
		return ErrInvalidManifest
	}
	if manifestStep.Compatibility == "incompatible" && manifestStep.DrainPolicy == "none" {
		return fmt.Errorf("%w: migrate.step %04d declares compatibility=incompatible with drain_policy=none", ErrInvalidManifest, stepNumber)
	}
	if manifestStep.From != fileStep.from || manifestStep.To != fileStep.to {
		return ErrInvalidManifest
	}
	return nil
}

func validateManifestStoreSchemas(schemas []manifestStoreSchema, availableFiles map[string]struct{}) error {
	if len(schemas) == 0 {
		return ErrInvalidManifest
	}
	for _, schema := range schemas {
		if strings.TrimSpace(schema.Store) == "" || strings.TrimSpace(schema.Version) == "" || strings.TrimSpace(schema.RecordSchema) == "" {
			return ErrInvalidManifest
		}
		if _, ok := availableFiles[schema.RecordSchema]; !ok {
			return ErrInvalidManifest
		}
	}
	return nil
}

func validateManifestMigrationFixtures(
	fixtures []manifestMigrationFixture,
	manifestSteps []manifestMigrationStep,
	stepNames map[string]struct{},
	stepByName map[string]parsedMigrationStep,
	storeSchemaByVersion map[string][]manifestStoreSchema,
	availableFiles map[string]struct{},
	migrationSources map[string][]byte,
) error {
	fixtureByStep := make(map[string]struct{}, len(fixtures))
	for _, fixture := range fixtures {
		if err := validateManifestMigrationFixture(fixture, manifestSteps, stepNames, stepByName, storeSchemaByVersion, availableFiles, migrationSources, fixtureByStep); err != nil {
			return err
		}
	}
	return nil
}

func validateManifestMigrationFixture(
	fixture manifestMigrationFixture,
	manifestSteps []manifestMigrationStep,
	stepNames map[string]struct{},
	stepByName map[string]parsedMigrationStep,
	storeSchemaByVersion map[string][]manifestStoreSchema,
	availableFiles map[string]struct{},
	migrationSources map[string][]byte,
	fixtureByStep map[string]struct{},
) error {
	if err := validateManifestMigrationFixtureMetadata(fixture, manifestSteps, stepNames, stepByName, availableFiles, fixtureByStep); err != nil {
		return err
	}
	readAdapter := strings.TrimSpace(fixture.ReadAdapter)
	if readAdapter != "" {
		source, ok := migrationSources[readAdapter]
		if !ok {
			return fmt.Errorf("%w: migrate.fixture %s read_adapter %q missing from archive", ErrInvalidManifest, fixture.Step, readAdapter)
		}
		if err := validateMigrationReadAdapterSource(readAdapter, source); err != nil {
			return err
		}
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
	manifestStep := manifestSteps[stepByName[fixture.Step].stepNumber-1]
	return validateManifestMigrationFixtureExpected(fixture, manifestStep, stepByName, storeSchemaByVersion, migrationSources, expectedRecords)
}

func validateManifestMigrationFixtureMetadata(
	fixture manifestMigrationFixture,
	manifestSteps []manifestMigrationStep,
	stepNames map[string]struct{},
	stepByName map[string]parsedMigrationStep,
	availableFiles map[string]struct{},
	fixtureByStep map[string]struct{},
) error {
	if strings.TrimSpace(fixture.Step) == "" || strings.TrimSpace(fixture.PriorVersion) == "" || strings.TrimSpace(fixture.PriorRecordSchema) == "" || strings.TrimSpace(fixture.Seed) == "" || strings.TrimSpace(fixture.Expected) == "" {
		return ErrInvalidManifest
	}
	if _, ok := stepNames[fixture.Step]; !ok {
		return ErrInvalidManifest
	}
	step := stepByName[fixture.Step]
	if fixture.PriorVersion != step.from {
		return fmt.Errorf("%w: migrate.fixture %s prior_version %q does not match step from-version %q", ErrInvalidManifest, fixture.Step, fixture.PriorVersion, step.from)
	}
	if _, ok := fixtureByStep[fixture.Step]; ok {
		return ErrInvalidManifest
	}
	fixtureByStep[fixture.Step] = struct{}{}
	if manifestSteps[step.stepNumber-1].DrainPolicy == "multi_version" && strings.TrimSpace(fixture.ReadAdapter) == "" {
		return fmt.Errorf("%w: migrate.fixture %s must declare read_adapter for multi_version migration", ErrInvalidManifest, fixture.Step)
	}
	for _, path := range []string{fixture.PriorRecordSchema, fixture.Seed, fixture.Expected} {
		if _, ok := availableFiles[path]; !ok {
			return ErrInvalidManifest
		}
	}
	return nil
}

func validateManifestMigrationFixtureExpected(
	fixture manifestMigrationFixture,
	manifestStep manifestMigrationStep,
	stepByName map[string]parsedMigrationStep,
	storeSchemaByVersion map[string][]manifestStoreSchema,
	migrationSources map[string][]byte,
	expectedRecords []migrationFixtureRecord,
) error {
	targetSchemaPath, targetSchemaPayload, shouldValidateExpected, err := resolveFixtureExpectedSchema(fixture, manifestStep, stepByName, storeSchemaByVersion, migrationSources)
	if err != nil {
		return err
	}
	if !shouldValidateExpected {
		return nil
	}
	return validateMigrationFixtureValueSchema(fixture.Expected, targetSchemaPath, expectedRecords, targetSchemaPayload)
}

func validateManifestMigrationModules(files []string, migrationSources map[string][]byte) error {
	for _, rel := range files {
		if !strings.HasPrefix(rel, "migrate/") {
			continue
		}
		if err := validateManifestMigrationModule(rel, migrationSources[rel]); err != nil {
			return err
		}
	}
	return nil
}

func validateManifestMigrationModule(rel string, source []byte) error {
	if source == nil {
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
		key, value, err := validateMigrationFixtureNDJSONLine(path, lineNumber, lines[i], previousKey, seenKeys)
		if err != nil {
			return nil, err
		}
		seenKeys[key] = struct{}{}
		previousKey = key
		records = append(records, migrationFixtureRecord{Key: key, Value: value, Line: lineNumber})
	}

	return records, nil
}

func validateMigrationFixtureNDJSONLine(path string, lineNumber int, line []byte, previousKey string, seenKeys map[string]struct{}) (string, map[string]any, error) {
	if len(line) == 0 {
		return "", nil, fmt.Errorf("%w: migration fixture %s line %d is blank", ErrInvalidManifest, path, lineNumber)
	}
	canonical, key, value, err := parseCanonicalFixtureRecord(line)
	if err != nil {
		return "", nil, fmt.Errorf("%w: migration fixture %s line %d: %v", ErrInvalidManifest, path, lineNumber, err)
	}
	if !bytes.Equal(line, canonical) {
		return "", nil, fmt.Errorf("%w: migration fixture %s line %d is not canonical JSON", ErrInvalidManifest, path, lineNumber)
	}
	if _, dup := seenKeys[key]; dup {
		return "", nil, fmt.Errorf("%w: migration fixture %s line %d has duplicate key %q", ErrInvalidManifest, path, lineNumber, key)
	}
	if previousKey != "" && bytes.Compare([]byte(previousKey), []byte(key)) >= 0 {
		return "", nil, fmt.Errorf("%w: migration fixture %s line %d is out of key order", ErrInvalidManifest, path, lineNumber)
	}
	return key, value, nil
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
	manifestStep manifestMigrationStep,
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
		if manifestStep.Compatibility == "incompatible" {
			return "", nil, false, fmt.Errorf("%w: migrate.fixture %s expected schema is required for incompatible target version %q", ErrInvalidManifest, fixture.Step, step.to)
		}
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

func validateMigrationReadAdapterSource(path string, payload []byte) error {
	if len(bytes.TrimSpace(payload)) == 0 {
		return fmt.Errorf("%w: migration read_adapter %s is empty", ErrInvalidManifest, path)
	}
	if !migrateReadPattern.Match(payload) {
		return fmt.Errorf("%w: migration read_adapter %s missing read(record) entrypoint", ErrInvalidManifest, path)
	}
	for _, match := range migrateLoadPattern.FindAllSubmatch(payload, -1) {
		if len(match) < 2 {
			continue
		}
		module := strings.TrimSpace(string(match[1]))
		if _, allowed := allowedMigrationModules[module]; !allowed {
			return fmt.Errorf("%w: migration read_adapter %s loads disallowed module %q", ErrInvalidManifest, path, module)
		}
	}
	for lineNumber, rawLine := range strings.Split(string(payload), "\n") {
		line := strings.TrimSpace(stripTALLineComment(rawLine))
		if !strings.HasPrefix(line, "return") {
			continue
		}
		if !migrateReadAdapterIdentityReturnPattern.MatchString(line) {
			return fmt.Errorf("%w: migration read_adapter %s line %d uses unsupported read_adapter return expression %q", ErrInvalidManifest, path, lineNumber+1, line)
		}
	}
	return nil
}

func stripTALLineComment(line string) string {
	var quote rune
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if r == '"' || r == '\'' {
			switch quote {
			case 0:
				quote = r
			case r:
				quote = 0
			}
			continue
		}
		if r == '#' && quote == 0 {
			return line[:i]
		}
	}
	return line
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
