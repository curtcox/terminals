package appruntime

import (
	"bufio"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

type migrationJournalEntry struct {
	Event           string `json:"event"`
	Step            int    `json:"step"`
	FromVersion     string `json:"from_version"`
	ToVersion       string `json:"to_version"`
	Script          string `json:"script"`
	Error           string `json:"error"`
	Level           string `json:"level"`
	Message         string `json:"message"`
	Arguments       string `json:"arguments"`
	ArtifactID      string `json:"artifact_id"`
	OwnerAppID      string `json:"owner_app_id"`
	EffectSequence  int    `json:"effect_sequence"`
	CheckpointEvery int    `json:"checkpoint_every"`
}

func parseMigrationJournalEntries(t *testing.T, data []byte) []migrationJournalEntry {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	entries := make([]migrationJournalEntry, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry migrationJournalEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Unmarshal(journal line %q) error = %v", line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan(journal) error = %v", err)
	}
	return entries
}

func buildRuntimeFixtureRows(count int) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 1; i <= count; i++ {
		b.WriteString("{\"key\":\"history/")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("\",\"value\":{\"count\":")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("}}\n")
	}
	return b.String()
}

func hasMigrationJournalEvent(entries []migrationJournalEntry, event string) bool {
	for _, entry := range entries {
		if entry.Event == event {
			return true
		}
	}
	return false
}

func hasMigrationJournalErrorContaining(entries []migrationJournalEntry, event string, want string) bool {
	for _, entry := range entries {
		if entry.Event == event && strings.Contains(entry.Error, want) {
			return true
		}
	}
	return false
}

func hasMigrationJournalArtifactPatch(entries []migrationJournalEntry, artifactID string, ownerAppID string, sequence int) bool {
	for _, entry := range entries {
		if entry.Event == "artifact_patch_planned" &&
			entry.ArtifactID == artifactID &&
			entry.OwnerAppID == ownerAppID &&
			entry.EffectSequence == sequence {
			return true
		}
	}
	return false
}

func hasMigrationJournalEventForStep(entries []migrationJournalEntry, event string, step int) bool {
	for _, entry := range entries {
		if entry.Event == event && entry.Step == step {
			return true
		}
	}
	return false
}

func hasMigrationJournalEventSequence(entries []migrationJournalEntry, sequence []string) bool {
	if len(sequence) == 0 {
		return true
	}
	index := 0
	for _, entry := range entries {
		if entry.Event != sequence[index] {
			continue
		}
		index++
		if index == len(sequence) {
			return true
		}
	}
	return false
}

func hasMigrationStepMetadata(entries []migrationJournalEntry, event string, step int, fromVersion string, toVersion string, script string) bool {
	for _, entry := range entries {
		if entry.Event != event || entry.Step != step {
			continue
		}
		if entry.FromVersion == fromVersion && entry.ToVersion == toVersion && entry.Script == script {
			return true
		}
	}
	return false
}

func hasMigrationCheckpointMetadata(entries []migrationJournalEntry, step int, effectSequence int, checkpointEvery int) bool {
	for _, entry := range entries {
		if entry.Event != "checkpoint_committed" || entry.Step != step {
			continue
		}
		if entry.EffectSequence == effectSequence && entry.CheckpointEvery == checkpointEvery {
			return true
		}
	}
	return false
}

func hasMigrationLogEntry(entries []migrationJournalEntry, step int, level string, message string, arguments string) bool {
	for _, entry := range entries {
		if entry.Event != "migration_log" || entry.Step != step {
			continue
		}
		if entry.Level == level && entry.Message == message && entry.Arguments == arguments {
			return true
		}
	}
	return false
}
