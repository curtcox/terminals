package appruntime

// Journal replay and NDJSON field helpers for runtime migrations.

import (
	"strconv"
	"strings"
	"time"
)

func replayMigrationStateFromJournal(pkg Package, state migrationState) migrationState {
	if strings.TrimSpace(state.JournalPath) == "" {
		return state
	}

	entries, err := readMigrationJournalLines(pkg, state.JournalPath)
	if err != nil {
		return state
	}

	lastEvent := ""
	for _, entry := range entries {
		if event := migrationJournalString(entry["event"]); event != "" {
			lastEvent = event
		}
		applyMigrationJournalScalars(entry, &state)
		applyMigrationJournalEvent(pkg, lastEvent, entry, &state)
	}
	finalizeMigrationJournalReplay(&state, lastEvent)
	return state
}

func migrationArtifactInverseFailuresFromJournal(pkg Package, state migrationState) map[string]string {
	if strings.TrimSpace(state.JournalPath) == "" {
		return nil
	}
	entries, err := readMigrationJournalLines(pkg, state.JournalPath)
	if err != nil {
		return nil
	}

	pending := map[string]string{}
	for _, entry := range entries {
		event := migrationJournalString(entry["event"])
		pending = applyMigrationJournalPendingEvent(pending, event, entry)
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func migrationPendingRecordsFromJournalValue(raw any) map[string]string {
	values, ok := raw.(map[string]any)
	if !ok || len(values) == 0 {
		return nil
	}
	pending := make(map[string]string, len(values))
	for recordID, resolutionRaw := range values {
		recordID = strings.TrimSpace(recordID)
		if recordID == "" {
			continue
		}
		resolution := migrationJournalString(resolutionRaw)
		if resolution == "" {
			resolution = "manual"
		}
		pending[recordID] = resolution
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func migrationJournalInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func migrationJournalString(raw any) string {
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func migrationJournalBool(raw any) (bool, bool) {
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func migrationJournalTime(raw any) (time.Time, bool) {
	value := migrationJournalString(raw)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
