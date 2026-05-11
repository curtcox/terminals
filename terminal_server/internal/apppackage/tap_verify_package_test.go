package apppackage

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestVerifyPackageRejectsMalformedSignatureBundleTOML(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	if _, err := VerifyPackage(tapBytes, []byte("schema = \"tap-sig/1\"\npackage_id = \"sha256:abc\"\n[[statement]\n")); err == nil {
		t.Fatalf("expected malformed bundle rejection")
	}
}

func TestVerifyPackageRejectsSignatureBundleSchemaMismatch(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	bundle := strings.TrimSpace(fmt.Sprintf(`
schema = "tap-sig/2"
package_id = "%s"
`, packageID))

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrInvalidSignatureBundle {
		t.Fatalf("expected invalid signature bundle error, got %v", err)
	}
}

func TestVerifyPackageRejectsSignatureBundlePackageIDMismatch(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	bundle := strings.TrimSpace(`
schema = "tap-sig/1"
package_id = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
`)

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrSignaturePackageIDMismatch {
		t.Fatalf("expected package id mismatch, got %v", err)
	}
}

func TestVerifyPackageRejectsMissingStatementNonceField(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	bundle := strings.TrimSpace(fmt.Sprintf(`
schema = "tap-sig/1"
package_id = "%s"

[[statement]]
role = "author"
key_id = "ed25519:%s"
created_unix = 1714000000
manifest_name = "kitchen_timer"
manifest_version = "0.1.0"
scope = {}
sig = "base64:AAAA"
`, packageID, base64.RawURLEncoding.EncodeToString(pub)))

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrInvalidSignatureStatement {
		t.Fatalf("expected invalid statement for missing nonce, got %v", err)
	}
}

func TestVerifyPackageRejectsStatementManifestMismatch(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	stmt, _ := signedAuthorStatement(t, packageID, "other_app", "0.1.0")
	bundle := signedBundleTOML(t, packageID, []signatureStatement{stmt})

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrInvalidSignatureStatement {
		t.Fatalf("expected invalid signature statement for manifest mismatch, got %v", err)
	}
}

func TestVerifyPackageRejectsInvalidStatementSignature(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	stmt, _ := signedAuthorStatement(t, packageID, "kitchen_timer", "0.1.0")
	sigRaw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stmt.Sig, "base64:"))
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	sigRaw[0] ^= 0xFF
	stmt.Sig = "base64:" + base64.StdEncoding.EncodeToString(sigRaw)
	bundle := signedBundleTOML(t, packageID, []signatureStatement{stmt})

	if _, err := VerifyPackage(tapBytes, []byte(bundle)); err != ErrSignatureVerificationFailed {
		t.Fatalf("expected signature verification failure, got %v", err)
	}
}

func TestVerifyPackageRejectsOversizedSignatureBundle(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	bundle := bytes.Repeat([]byte("a"), signatureBundleMaxBytes+1)

	if _, err := VerifyPackage(tapBytes, bundle); err != ErrInvalidSignatureBundle {
		t.Fatalf("expected oversized bundle rejection, got %v", err)
	}
}

func TestVerifyPackageRejectsTooManyStatements(t *testing.T) {
	tapBytes, packageID := minimalTapAndID(t)
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("schema = \"tap-sig/1\"\n")
	_, _ = fmt.Fprintf(&sb, "package_id = \"%s\"\n\n", packageID)
	for i := 0; i < statementMaxCount+1; i++ {
		sb.WriteString("[[statement]]\n")
		sb.WriteString("role = \"author\"\n")
		_, _ = fmt.Fprintf(&sb, "key_id = \"ed25519:%s\"\n", base64.RawURLEncoding.EncodeToString(pub))
		_, _ = fmt.Fprintf(&sb, "created_unix = %d\n", 1714000000+i)
		sb.WriteString("manifest_name = \"kitchen_timer\"\n")
		sb.WriteString("manifest_version = \"0.1.0\"\n")
		_, _ = fmt.Fprintf(&sb, "nonce = \"base64url:%s\"\n", base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("nonce-%010d", i))))
		sb.WriteString("scope = {}\n")
		sb.WriteString("sig = \"base64:AAAA\"\n\n")
	}

	if _, err := VerifyPackage(tapBytes, []byte(sb.String())); err != ErrInvalidSignatureBundle {
		t.Fatalf("expected too-many-statements rejection, got %v", err)
	}
}
