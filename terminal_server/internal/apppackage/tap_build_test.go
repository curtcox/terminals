package apppackage

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

func TestBuildTapFromDirMatchesPinnedZstdCLIProfile(t *testing.T) {
	if _, err := exec.LookPath("zstd"); err != nil {
		t.Skip("zstd CLI not found")
	}

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

	tapBytes, packageID, err := BuildTapFromDir(appRoot)
	if err != nil {
		t.Fatalf("build tap: %v", err)
	}

	canonicalTar, err := decompressTap(tapBytes)
	if err != nil {
		t.Fatalf("decompress tap: %v", err)
	}

	tarPath := filepath.Join(root, "canonical.tar")
	if err := os.WriteFile(tarPath, canonicalTar, 0o644); err != nil {
		t.Fatalf("write canonical tar: %v", err)
	}

	cmd := exec.Command("zstd", "-19", "--no-check", "--content-size", "--format=zstd", "--single-thread", "-q", "-c", tarPath)
	cliTapBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("zstd cli encode: %v", err)
	}

	verifiedGoTap, err := VerifyTap(tapBytes)
	if err != nil {
		t.Fatalf("verify go-built tap: %v", err)
	}
	verifiedCLITap, err := VerifyTap(cliTapBytes)
	if err != nil {
		t.Fatalf("verify cli-reencoded tap: %v", err)
	}

	if verifiedGoTap.PackageID != packageID {
		t.Fatalf("go-built package id mismatch: got %q want %q", verifiedGoTap.PackageID, packageID)
	}
	if verifiedCLITap.PackageID != packageID {
		t.Fatalf("cli-reencoded package id mismatch: got %q want %q", verifiedCLITap.PackageID, packageID)
	}
	if verifiedCLITap.PackageID != verifiedGoTap.PackageID {
		t.Fatalf("package identity changed across pinned CLI re-encoding")
	}
}
