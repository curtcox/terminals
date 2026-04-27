package apppackage

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
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
