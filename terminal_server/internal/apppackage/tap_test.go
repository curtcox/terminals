package apppackage

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func TestVerifyTapAcceptsCanonicalMigrateStepLayout(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 2

[[migrate.step]]
from = "1"
to = "2"

[[migrate.step]]
from = "2"
to = "3"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"

[[migrate.fixture]]
step = "0002_2_to_3"
prior_version = "2"
prior_record_schema = "tests/schemas/history_v2.json"
seed = "tests/migrate_fixtures/history_v2_seed.ndjson"
expected = "tests/migrate_fixtures/history_v3_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/migrate/0002_2_to_3.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v3_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected canonical migration layout to verify, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateStepNumberingGap(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[migrate]
declared_steps = 2

[[migrate.step]]
from = "1"
to = "2"

[[migrate.step]]
from = "2"
to = "3"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/migrate/0003_2_to_3.tal", body: "def migrate(): pass"},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for migration numbering gap, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "migration step numbering gap") {
		t.Fatalf("expected specific numbering gap error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateDeclaredStepMismatch(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[migrate]
declared_steps = 2

[[migrate.step]]
from = "1"
to = "2"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
	})

	if _, err := VerifyTap(tap); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for declared-step mismatch, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateIncompatibleWithoutDrain(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "incompatible"
drain_policy = "none"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for incompatible migration without drain, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "compatibility=incompatible with drain_policy=none") {
		t.Fatalf("expected specific incompatible/drain policy error, got %v", err)
	}
}

func TestVerifyTapAcceptsMigrateIncompatibleWithDrain(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "incompatible"
drain_policy = "drain"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected incompatible migration with drain to verify, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateMissingFixtureMetadata(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
	})

	if _, err := VerifyTap(tap); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for missing migration fixtures, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateLoadBusModule(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: `load("bus", emit = "emit")\n\ndef migrate():\n    pass`},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for disallowed migration module, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), `loads disallowed module "bus"`) {
		t.Fatalf("expected specific disallowed module error, got %v", err)
	}
}

func TestVerifyTapAcceptsMigrateAllowedModules(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: `load("store", get = "get", put = "put")\nload("artifact.self", patch = "patch")\nload("log", info = "info")\nload("migrate.env", checkpoint = "checkpoint")\n\ndef migrate():\n    pass`},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected migration with allowed modules to verify, got %v", err)
	}
}

func TestVerifyTapAcceptsMigrateDowngradeScripts(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: `load("migrate.env", checkpoint = "checkpoint")\n\ndef migrate():\n    pass`},
		{name: "kitchen_timer/migrate/downgrade/0001_2_to_1.tal", body: `load("store", put = "put")\n\ndef migrate():\n    pass`},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected migration with downgrade script to verify, got %v", err)
	}
}

func TestVerifyTapRejectsNestedMigrateDowngradePath(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/migrate/downgrade/v1/0001_2_to_1.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for nested downgrade path, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureCRLFLineEndings(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\r\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for CRLF fixture line endings, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "must use LF line endings") {
		t.Fatalf("expected specific LF line ending error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureOutOfOrderKeys(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k2\",\"value\":{}}\n{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for out-of-order fixture keys, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "out of key order") {
		t.Fatalf("expected specific key order error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureDuplicateKeys(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for duplicate fixture keys, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("expected specific duplicate key error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureNonCanonicalJSONLine(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"value\":{},\"key\":\"k1\"}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for non-canonical fixture JSON, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "not canonical JSON") {
		t.Fatalf("expected specific canonical JSON error, got %v", err)
	}
}

func TestVerifyTapRejectsZstdChecksumFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] |= 0x04

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdDictionaryIDFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] = (mutated[4] &^ 0x03) | 0x01

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdMissingContentSizeFlag(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append([]byte(nil), tapBytes...)
	mutated[4] = (mutated[4] &^ 0xC0) | 0x20

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdTrailingBytes(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append(append([]byte(nil), tapBytes...), 0x00)

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdMultiframe(t *testing.T) {
	tapBytes, _ := minimalTapAndID(t)
	mutated := append(append([]byte(nil), tapBytes...), tapBytes...)

	if _, err := VerifyTap(mutated); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsSkippableFrameMagic(t *testing.T) {
	skippable := []byte{0x50, 0x2A, 0x4D, 0x18, 0x00, 0x00, 0x00, 0x00}

	if _, err := VerifyTap(skippable); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
	}
}

func TestVerifyTapRejectsZstdWindowTooLarge(t *testing.T) {
	tapBytes := []byte{
		0x28, 0xB5, 0x2F, 0xFD, // zstd frame magic
		0x40,       // FCS flag=1, single segment=0, checksum=0, dict ID=0
		0x70,       // window descriptor => window log 24 (>23)
		0x00, 0x01, // frame content size field (2 bytes for FCS flag=1)
		0x01, 0x00, 0x00, // last raw block, size 0
	}

	if _, err := VerifyTap(tapBytes); !errors.Is(err, ErrInvalidTapFormat) {
		t.Fatalf("expected invalid tap format, got %v", err)
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
