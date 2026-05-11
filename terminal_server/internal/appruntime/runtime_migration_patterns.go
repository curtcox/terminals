package appruntime

import "regexp"

var migrateStepFilePattern = regexp.MustCompile(`^(\d+)_([^/]+)_to_([^/]+)\.tal$`)

var migrateLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']([^"']+)["']`)

var migrateEntryPointPattern = regexp.MustCompile(`(?m)^\s*def\s+migrate\s*\(`)

var migrateReadAdapterEntryPointPattern = regexp.MustCompile(`(?m)^\s*def\s+read\s*\(\s*record\s*\)`)

var migrateReadAdapterIdentityReturnPattern = regexp.MustCompile(`^\s*return\s+record\s*$`)

var migrateRecordAssignmentPattern = regexp.MustCompile(`^\s*record\["([^"]+)"\]\s*=\s*(.+?)\s*$`)

var migrateRecordDeletePattern = regexp.MustCompile(`^\s*del\s+record\["([^"]+)"\]\s*$`)

var migrateRecordSkipIfPresentPattern = regexp.MustCompile(`^\s*if\s+["']([^"']+)["']\s+in\s+record\s*:\s*continue\s*$`)

var migrateRecordSkipIfPresentBlockPattern = regexp.MustCompile(`^\s*if\s+["']([^"']+)["']\s+in\s+record\s*:\s*$`)

var migrateRecordValuePattern = regexp.MustCompile(`^record\["([^"]+)"\]$`)

var migrateRecordGetValuePattern = regexp.MustCompile(`^record\.get\(\s*"([^"]+)"\s*,\s*(.+)\s*\)$`)

var migrateRecordLowerPattern = regexp.MustCompile(`^lower\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateRecordLowerGetPattern = regexp.MustCompile(`^lower\(\s*record\.get\(\s*"([^"]+)"\s*,\s*(.+)\s*\)\s*\)$`)

var migrateRecordTrimPattern = regexp.MustCompile(`^trim\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateRecordTrimGetPattern = regexp.MustCompile(`^trim\(\s*record\.get\(\s*"([^"]+)"\s*,\s*(.+)\s*\)\s*\)$`)

var migrateRecordLowerTrimPattern = regexp.MustCompile(`^lower\(\s*trim\(\s*record\["([^"]+)"\]\s*\)\s*\)$`)

var migrateRecordLowerTrimGetPattern = regexp.MustCompile(`^lower\(\s*trim\(\s*record\.get\(\s*"([^"]+)"\s*,\s*(.+)\s*\)\s*\)\s*\)$`)

var migrateRecordNormalizeGetPattern = regexp.MustCompile(`^_normalize\(\s*record\.get\(\s*"([^"]+)"\s*,\s*(.+)\s*\)\s*\)$`)

var migrateRecordNormalizePattern = regexp.MustCompile(`^_normalize\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateAbortPattern = regexp.MustCompile(`^abort\(\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)$`)

var migrateArtifactSelfLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']artifact\.self["']\s*,(?P<args>[^)]*)\)`)

var migrateLoadAliasPattern = regexp.MustCompile(`\bpatch\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateLogLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']log["']\s*,(?P<args>[^)]*)\)`)

var migrateLogAliasPattern = regexp.MustCompile(`\b(debug|info|warn|error)\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']store["']\s*,(?P<args>[^)]*)\)`)

var migrateStoreListKeysAliasPattern = regexp.MustCompile(`\blist_keys\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreGetAliasPattern = regexp.MustCompile(`\bget\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStorePutAliasPattern = regexp.MustCompile(`\bput\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreDeleteAliasPattern = regexp.MustCompile(`\bdelete\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateEnvLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']migrate\.env["']\s*,(?P<args>[^)]*)\)`)

var migrateEnvCheckpointAliasPattern = regexp.MustCompile(`\bcheckpoint\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateEnvAbortAliasPattern = regexp.MustCompile(`\babort\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateCallPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)\s*$`)

var migrateStringArgPattern = regexp.MustCompile(`^\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')`)

var migrateOwnerAppIDPattern = regexp.MustCompile(`\bowner_app_id\s*=\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')`)

var allowedMigrationModules = map[string]struct{}{
	"store":         {},
	"artifact.self": {},
	"log":           {},
	"migrate.env":   {},
}
