package apppackage

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

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
compatibility = "compatible"
drain_policy = "none"

[[migrate.step]]
from = "2"
to = "3"
compatibility = "compatible"
drain_policy = "none"

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

func TestVerifyTapRejectsMigrateStepMissingPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		stepPolicy  string
		wantMessage string
	}{
		{
			name:        "missing compatibility",
			stepPolicy:  `drain_policy = "none"`,
			wantMessage: "migrate.step 0001 must declare compatibility",
		},
		{
			name:        "missing drain policy",
			stepPolicy:  `compatibility = "compatible"`,
			wantMessage: "migrate.step 0001 must declare drain_policy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := strings.TrimSpace(fmt.Sprintf(`
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
%s

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`, tc.stepPolicy))

			tap := makeTapForTest(t, []tapEntry{
				{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
				{name: "kitchen_timer/manifest.toml", body: manifest},
				{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
				{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
				{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
				{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
				{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
			})

			_, err := VerifyTap(tap)
			if !errors.Is(err, ErrInvalidManifest) || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("VerifyTap() error = %v, want ErrInvalidManifest containing %q", err, tc.wantMessage)
			}
		})
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
compatibility = "compatible"
drain_policy = "none"

[[migrate.step]]
from = "2"
to = "3"
compatibility = "compatible"
drain_policy = "none"
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
compatibility = "compatible"
drain_policy = "none"
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

func TestVerifyTapRejectsMigrateNonPositiveMaxRuntimeSeconds(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1
max_runtime_seconds = 0

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for non-positive max_runtime_seconds, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "migrate.max_runtime_seconds must be a positive integer") {
		t.Fatalf("expected max_runtime_seconds diagnostic, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateNonPositiveCheckpointEvery(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1
checkpoint_every = -1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for non-positive checkpoint_every, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "migrate.checkpoint_every must be a positive integer") {
		t.Fatalf("expected checkpoint_every diagnostic, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateNonPositiveDrainTimeoutSeconds(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "2"
record_schema = "tests/schemas/history_v2.json"

[migrate]
declared_steps = 1
drain_timeout_seconds = 0

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for non-positive drain_timeout_seconds, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "migrate.drain_timeout_seconds must be a positive integer") {
		t.Fatalf("expected drain_timeout_seconds diagnostic, got %v", err)
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

func TestVerifyTapRejectsIncompatibleMigrationWithoutTargetSchema(t *testing.T) {
	manifest := strings.TrimSpace(`
name = "kitchen_timer"
version = "2"

[[storage.store_schema]]
store = "history"
version = "1"
record_schema = "tests/schemas/history_v1.json"

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
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for incompatible migration without target schema, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "expected schema is required for incompatible target version") {
		t.Fatalf("expected incompatible target schema error, got %v", err)
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
compatibility = "compatible"
drain_policy = "none"
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

func TestVerifyTapRejectsMigrateFixturePriorVersionMismatch(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "9"
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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for fixture prior_version mismatch, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "prior_version") {
		t.Fatalf("expected prior_version mismatch detail, got %v", err)
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
compatibility = "compatible"
drain_policy = "none"

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
compatibility = "compatible"
drain_policy = "none"

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

func TestVerifyTapRequiresReadAdapterForMultiVersionMigration(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "multi_version"

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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for missing read_adapter, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "must declare read_adapter") {
		t.Fatalf("expected read_adapter diagnostic, got %v", err)
	}
}

func TestVerifyTapAcceptsMultiVersionReadAdapter(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "multi_version"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
read_adapter = "tests/migrate_fixtures/read_v2_as_v1.tal"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/read_v2_as_v1.tal", body: "def read(record):\n    return record\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected multi_version read_adapter to verify, got %v", err)
	}
}

func TestVerifyTapRejectsUnsupportedReadAdapterReturn(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "multi_version"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
read_adapter = "tests/migrate_fixtures/read_v2_as_v1.tal"
`)

	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/read_v2_as_v1.tal", body: "def read(record):\n    return {}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err == nil {
		t.Fatalf("expected unsupported read_adapter return to fail verification")
	} else if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest, got %v", err)
	} else if !strings.Contains(err.Error(), "unsupported read_adapter return expression") {
		t.Fatalf("expected read_adapter return diagnostic, got %v", err)
	}
}

func TestVerifyTapIgnoresCommentedDisallowedLoadStatements(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

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
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: `# load("bus", emit = "emit")\nload("store", get = "get")\n\ndef migrate():\n    pass`},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected commented disallowed module load to be ignored, got %v", err)
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
compatibility = "compatible"
drain_policy = "none"

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
compatibility = "compatible"
drain_policy = "none"

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

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for nested downgrade path, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "must be a single-level file under migrate/downgrade/") {
		t.Fatalf("expected specific nested downgrade path error, got %v", err)
	}
}

func TestVerifyTapRejectsMalformedMigrateDowngradeFilename(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

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
		{name: "kitchen_timer/migrate/downgrade/reverse.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for malformed downgrade filename, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "must match <step>_<from>_to_<to>.tal") {
		t.Fatalf("expected specific malformed downgrade filename error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateMalformedStepFilename(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

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
		{name: "kitchen_timer/migrate/not_a_step.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for malformed migration step filename, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "must match <step>_<from>_to_<to>.tal") {
		t.Fatalf("expected specific malformed migration step filename error, got %v", err)
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
compatibility = "compatible"
drain_policy = "none"

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
compatibility = "compatible"
drain_policy = "none"

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
compatibility = "compatible"
drain_policy = "none"

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
compatibility = "compatible"
drain_policy = "none"

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

func TestVerifyTapRejectsMigrateFixtureSeedSchemaMismatch(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

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
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{\"label\":\"alpha\"}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object","required":["legacy_id"],"properties":{"legacy_id":{"type":"string"}}}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for seed schema mismatch, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "violates schema tests/schemas/history_v1.json") {
		t.Fatalf("expected prior schema validation error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureExpectedSchemaMismatch(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

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
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: "{\"key\":\"k1\",\"value\":{\"legacy_id\":\"one\"}}\n"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object","required":["legacy_id"],"properties":{"legacy_id":{"type":"string"}}}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object","required":["label_normalized"],"properties":{"label_normalized":{"type":"string"}}}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for expected schema mismatch, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "violates schema tests/schemas/history_v2.json") {
		t.Fatalf("expected expected-schema validation error, got %v", err)
	}
}

func TestVerifyTapRejectsMigrateFixtureTooManyRecords(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	tooMany := buildFixtureRecords(migrationFixtureMaxRows + 1)
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: tooMany},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"k1\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	_, err := VerifyTap(tap)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest for oversized migration fixture, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "exceeds max records") {
		t.Fatalf("expected specific max records error, got %v", err)
	}
}

func TestVerifyTapAcceptsMigrateFixtureAtRecordLimit(t *testing.T) {
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
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "tests/schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`)

	atLimit := buildFixtureRecords(migrationFixtureMaxRows)
	tap := makeTapForTest(t, []tapEntry{
		{name: "kitchen_timer/main.tal", body: "def on_start(): pass"},
		{name: "kitchen_timer/manifest.toml", body: manifest},
		{name: "kitchen_timer/migrate/0001_1_to_2.tal", body: "def migrate(): pass"},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v1_seed.ndjson", body: atLimit},
		{name: "kitchen_timer/tests/migrate_fixtures/history_v2_expected.ndjson", body: "{\"key\":\"z\",\"value\":{}}\n"},
		{name: "kitchen_timer/tests/schemas/history_v1.json", body: `{"type":"object"}`},
		{name: "kitchen_timer/tests/schemas/history_v2.json", body: `{"type":"object"}`},
	})

	if _, err := VerifyTap(tap); err != nil {
		t.Fatalf("expected migration fixture at record limit to verify, got %v", err)
	}
}
