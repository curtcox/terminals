package apppackage

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
)

func TestBuildTapFromDirDeterministicPackageID(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "kitchen_timer")
	if err := os.MkdirAll(filepath.Join(appRoot, "lib"), 0o755); err != nil {
		t.Fatalf("mkdir lib: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte("name = \"kitchen_timer\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("write main.tal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "lib", "helpers.tal"), []byte("def helper(): pass\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	tapA, packageIDA, err := BuildTapFromDir(appRoot)
	if err != nil {
		t.Fatalf("build tap A: %v", err)
	}
	tapB, packageIDB, err := BuildTapFromDir(appRoot)
	if err != nil {
		t.Fatalf("build tap B: %v", err)
	}

	if packageIDA != packageIDB {
		t.Fatalf("package IDs differ: %q vs %q", packageIDA, packageIDB)
	}
	if !bytes.Equal(tapA, tapB) {
		t.Fatalf("tap output should be deterministic")
	}

	verified, err := VerifyTap(tapA)
	if err != nil {
		t.Fatalf("verify tap: %v", err)
	}
	if verified.PackageID != packageIDA {
		t.Fatalf("verified package id mismatch: %q vs %q", verified.PackageID, packageIDA)
	}
	if verified.PackageName != "kitchen_timer" {
		t.Fatalf("package name mismatch: %q", verified.PackageName)
	}
}

func TestVerifyTapRejectsUnknownTopLevel(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/secrets/key.txt", body: "shh"},
	})

	if _, err := VerifyTap(tap); err == nil {
		t.Fatalf("expected unknown top-level rejection")
	}
}

func TestVerifyTapRejectsPathTraversal(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/../escape.txt", body: "oops"},
	})

	if _, err := VerifyTap(tap); err == nil {
		t.Fatalf("expected unsafe path rejection")
	}
}

func TestVerifyTapRejectsMissingMain(t *testing.T) {
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/manifest.toml", body: "name='kitchen_timer'"},
	})

	if _, err := VerifyTap(tap); err != ErrMissingMainTAL {
		t.Fatalf("expected missing main.tal, got %v", err)
	}
}

func TestVerifyPackageAcceptsValidAuthorSignatureBundle(t *testing.T) {
	root := t.TempDir()
	appRoot := filepath.Join(root, "kitchen_timer")
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		t.Fatalf("mkdir app root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte("name = \"kitchen_timer\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}

	tapBytes, packageID, err := BuildTapFromDir(appRoot)
	if err != nil {
		t.Fatalf("build tap: %v", err)
	}

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	nonceRaw := []byte("nonce-nonce-0001")
	if len(nonceRaw) != statementNonceLen {
		t.Fatalf("nonce length mismatch")
	}

	stmt := signatureStatement{
		Role:            "author",
		KeyID:           "ed25519:" + base64.RawURLEncoding.EncodeToString(pub),
		CreatedUnix:     1714000000,
		ManifestName:    "kitchen_timer",
		ManifestVersion: "0.1.0",
		Nonce:           "base64url:" + base64.RawURLEncoding.EncodeToString(nonceRaw),
		Scope:           map[string]any{},
	}

	pkgHash := packageHashFromID(t, packageID)
	payload, err := encodeStatementCBOR(stmt, map[string]any{}, pkgHash, nonceRaw)
	if err != nil {
		t.Fatalf("encode cbor: %v", err)
	}
	stmt.Sig = "base64:" + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))

	bundle := signedBundleTOML(t, packageID, []signatureStatement{stmt})
	verified, err := VerifyPackage(tapBytes, []byte(bundle))
	if err != nil {
		t.Fatalf("verify package: %v", err)
	}

	if verified.ManifestName != "kitchen_timer" || verified.ManifestVersion != "0.1.0" {
		t.Fatalf("unexpected manifest identity: %q %q", verified.ManifestName, verified.ManifestVersion)
	}
	if len(verified.Statements) != 1 || verified.Statements[0].Role != "author" {
		t.Fatalf("unexpected verified statements: %+v", verified.Statements)
	}
}

func TestVerifyPackageRejectsUnknownVoucherScopeKey(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	authorNonce := []byte("nonce-nonce-0001")
	authorStmt := signatureStatement{
		Role:            "author",
		KeyID:           "ed25519:" + base64.RawURLEncoding.EncodeToString(pub),
		CreatedUnix:     1714000000,
		ManifestName:    "kitchen_timer",
		ManifestVersion: "0.1.0",
		Nonce:           "base64url:" + base64.RawURLEncoding.EncodeToString(authorNonce),
		Scope:           map[string]any{},
	}
	pkgHash := packageHashFromID(t, packageID)
	authorPayload, err := encodeStatementCBOR(authorStmt, map[string]any{}, pkgHash, authorNonce)
	if err != nil {
		t.Fatalf("encode author cbor: %v", err)
	}
	authorSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, authorPayload))

	bundle := strings.TrimSpace(fmt.Sprintf(`
schema = "tap-sig/1"
package_id = "%s"

[[statement]]
role = "author"
key_id = "ed25519:%s"
created_unix = 1714000000
manifest_name = "kitchen_timer"
manifest_version = "0.1.0"
nonce = "base64url:%s"
scope = {}
sig = "base64:%s"

[[statement]]
role = "voucher"
key_id = "ed25519:%s"
created_unix = 1714100000
manifest_name = "kitchen_timer"
manifest_version = "0.1.0"
nonce = "base64url:%s"
scope = { tier = "quarantine", reviewed = ["manifest"], tested_under = "sim-only", new_field = "not-allowed" }
sig = "base64:%s"
`,
		packageID,
		base64.RawURLEncoding.EncodeToString(pub),
		base64.RawURLEncoding.EncodeToString(authorNonce),
		authorSig,
		base64.RawURLEncoding.EncodeToString(pub),
		base64.RawURLEncoding.EncodeToString([]byte("nonce-nonce-0002")),
		base64.StdEncoding.EncodeToString([]byte("not-a-real-signature")),
	))

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrInvalidSignatureStatement {
		t.Fatalf("expected invalid signature statement, got %v", err)
	}
}

func TestVerifyPackageRejectsMissingAuthorStatement(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	nonceRaw := []byte("nonce-nonce-0001")
	stmt := signatureStatement{
		Role:            "voucher",
		KeyID:           "ed25519:" + base64.RawURLEncoding.EncodeToString(pub),
		CreatedUnix:     1714100000,
		ManifestName:    "kitchen_timer",
		ManifestVersion: "0.1.0",
		Nonce:           "base64url:" + base64.RawURLEncoding.EncodeToString(nonceRaw),
		Scope: map[string]any{
			"tier":         "quarantine",
			"reviewed":     []any{"manifest", "tal"},
			"tested_under": "sim-only",
		},
	}
	pkgHash := packageHashFromID(t, packageID)
	scope, err := normalizeScope(stmt.Role, stmt.Scope)
	if err != nil {
		t.Fatalf("normalize scope: %v", err)
	}
	payload, err := encodeStatementCBOR(stmt, scope, pkgHash, nonceRaw)
	if err != nil {
		t.Fatalf("encode cbor: %v", err)
	}
	stmt.Sig = "base64:" + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))

	bundle := signedBundleTOML(t, packageID, []signatureStatement{stmt})
	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrMissingAuthorSignature {
		t.Fatalf("expected missing author signature error, got %v", err)
	}
}

func TestVerifyPackageRejectsDuplicateNonceTriple(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pkgHash := packageHashFromID(t, packageID)
	nonceRaw := []byte("nonce-nonce-0001")

	stmtA := signatureStatement{
		Role:            "author",
		KeyID:           "ed25519:" + base64.RawURLEncoding.EncodeToString(pub),
		CreatedUnix:     1714000000,
		ManifestName:    "kitchen_timer",
		ManifestVersion: "0.1.0",
		Nonce:           "base64url:" + base64.RawURLEncoding.EncodeToString(nonceRaw),
		Scope:           map[string]any{},
	}
	payloadA, err := encodeStatementCBOR(stmtA, map[string]any{}, pkgHash, nonceRaw)
	if err != nil {
		t.Fatalf("encode cbor A: %v", err)
	}
	stmtA.Sig = "base64:" + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payloadA))

	stmtB := stmtA
	stmtB.CreatedUnix = 1714001111
	payloadB, err := encodeStatementCBOR(stmtB, map[string]any{}, pkgHash, nonceRaw)
	if err != nil {
		t.Fatalf("encode cbor B: %v", err)
	}
	stmtB.Sig = "base64:" + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payloadB))

	bundle := signedBundleTOML(t, packageID, []signatureStatement{stmtA, stmtB})
	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrInvalidSignatureStatement {
		t.Fatalf("expected duplicate nonce triple rejection, got %v", err)
	}
}

type tapEntry struct {
	name string
	body string
}

func makeTapForTest(t *testing.T, entries []tapEntry) []byte {
	t.Helper()
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)

	for _, entry := range entries {
		payload := []byte(entry.body)
		hdr := &tar.Header{
			Name:     entry.name,
			Mode:     canonicalFileMode,
			Uid:      0,
			Gid:      0,
			Size:     int64(len(payload)),
			ModTime:  time.Unix(0, 0).UTC(),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write(payload); err != nil {
			t.Fatalf("write tar payload: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	enc, err := zstd.NewWriter(
		nil,
		zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(19)),
		zstd.WithEncoderCRC(false),
		zstd.WithWindowSize(zstdWindowSize),
	)
	if err != nil {
		t.Fatalf("new zstd encoder: %v", err)
	}
	defer func() {
		_ = enc.Close()
	}()
	return enc.EncodeAll(tarBuf.Bytes(), nil)
}

func minimalTapAndID(t *testing.T) ([]byte, string) {
	t.Helper()
	root := t.TempDir()
	appRoot := filepath.Join(root, "kitchen_timer")
	if err := os.MkdirAll(appRoot, 0o755); err != nil {
		t.Fatalf("mkdir app root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte("name = \"kitchen_timer\"\nversion = \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	tapBytes, packageID, err := BuildTapFromDir(appRoot)
	if err != nil {
		t.Fatalf("build tap: %v", err)
	}
	return tapBytes, packageID
}

func packageHashFromID(t *testing.T, packageID string) []byte {
	t.Helper()
	const prefix = "sha256:"
	if !strings.HasPrefix(packageID, prefix) {
		t.Fatalf("unexpected package id prefix: %q", packageID)
	}
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(packageID, prefix))
	if err != nil {
		t.Fatalf("decode package id: %v", err)
	}
	if len(hashBytes) != sha256.Size {
		t.Fatalf("unexpected package hash size: %d", len(hashBytes))
	}
	return hashBytes
}

func signedBundleTOML(t *testing.T, packageID string, statements []signatureStatement) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("schema = \"tap-sig/1\"\n")
	_, _ = fmt.Fprintf(&sb, "package_id = \"%s\"\n\n", packageID)

	for _, stmt := range statements {
		sb.WriteString("[[statement]]\n")
		_, _ = fmt.Fprintf(&sb, "role = \"%s\"\n", stmt.Role)
		_, _ = fmt.Fprintf(&sb, "key_id = \"%s\"\n", stmt.KeyID)
		_, _ = fmt.Fprintf(&sb, "created_unix = %d\n", stmt.CreatedUnix)
		_, _ = fmt.Fprintf(&sb, "manifest_name = \"%s\"\n", stmt.ManifestName)
		_, _ = fmt.Fprintf(&sb, "manifest_version = \"%s\"\n", stmt.ManifestVersion)
		_, _ = fmt.Fprintf(&sb, "nonce = \"%s\"\n", stmt.Nonce)
		_, _ = fmt.Fprintf(&sb, "scope = %s\n", tomlInlineMap(t, stmt.Scope))
		_, _ = fmt.Fprintf(&sb, "sig = \"%s\"\n\n", stmt.Sig)
	}
	return sb.String()
}

func tomlInlineMap(t *testing.T, values map[string]any) string {
	t.Helper()
	if len(values) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(values))
	for key, value := range values {
		switch v := value.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s = \"%s\"", key, v))
		case []any:
			items := make([]string, 0, len(v))
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					t.Fatalf("unsupported array item type %T", item)
				}
				items = append(items, fmt.Sprintf("\"%s\"", s))
			}
			parts = append(parts, fmt.Sprintf("%s = [%s]", key, strings.Join(items, ", ")))
		case uint64:
			parts = append(parts, fmt.Sprintf("%s = %d", key, v))
		default:
			t.Fatalf("unsupported scope value type %T", value)
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
