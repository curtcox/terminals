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

func signedAuthorStatement(t *testing.T, packageID string, manifestName string, manifestVersion string) (signatureStatement, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	nonceRaw := []byte("nonce-nonce-0001")
	stmt := signatureStatement{
		Role:            "author",
		KeyID:           "ed25519:" + base64.RawURLEncoding.EncodeToString(pub),
		CreatedUnix:     1714000000,
		ManifestName:    manifestName,
		ManifestVersion: manifestVersion,
		Nonce:           "base64url:" + base64.RawURLEncoding.EncodeToString(nonceRaw),
		Scope:           map[string]any{},
	}
	packageHash := packageHashFromID(t, packageID)
	payload, err := encodeStatementCBOR(stmt, map[string]any{}, packageHash, nonceRaw)
	if err != nil {
		t.Fatalf("encode cbor: %v", err)
	}
	stmt.Sig = "base64:" + base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	return stmt, priv
}

func buildFixtureRecords(count int) string {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		_, _ = fmt.Fprintf(&builder, "{\"key\":\"k%04d\",\"value\":{}}\n", i)
	}
	return builder.String()
}
