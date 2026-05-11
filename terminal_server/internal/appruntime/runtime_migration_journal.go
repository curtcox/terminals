// Journal replay and NDJSON field helpers for runtime migrations.
package appruntime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func replayMigrationStateFromJournal(pkg Package, state migrationState) migrationState {
	if strings.TrimSpace(state.JournalPath) == "" {
		return state
	}

	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(state.JournalPath))
	file, err := os.Open(filepath.Clean(absolutePath))
	if err != nil {
		return state
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lastEvent := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if event := migrationJournalString(entry["event"]); event != "" {
			lastEvent = event
		}
		if stepsCompleted, ok := migrationJournalInt(entry["steps_completed"]); ok {
			state.StepsCompleted = stepsCompleted
		}
		if step, ok := migrationJournalInt(entry["step"]); ok {
			state.LastStep = step
		}
		if verdict := migrationJournalString(entry["verdict"]); verdict != "" {
			state.Verdict = verdict
		}
		if lastError := migrationJournalString(entry["last_error"]); lastError != "" {
			state.LastError = lastError
		} else if state.Verdict == "ok" || state.Verdict == "running" || state.Verdict == "idle" {
			state.LastError = ""
		}
		if blockedSince, ok := migrationJournalTime(entry["blocked_since"]); ok {
			state.DrainBlockedAt = blockedSince
		} else if state.Verdict == "ok" || state.Verdict == "running" || state.Verdict == "idle" {
			state.DrainBlockedAt = time.Time{}
		}
		switch lastEvent {
		case "drain_ready_changed":
			if ready, ok := migrationJournalBool(entry["ready"]); ok {
				state.DrainReady = ready
				if ready {
					state.DrainBlockedAt = time.Time{}
				}
			}
		case "artifact_inverse_failed":
			if state.PendingRecords == nil {
				state.PendingRecords = map[string]string{}
			}
			recordID := migrationJournalString(entry["record_id"])
			if recordID == "" {
				recordID = migrationJournalString(entry["artifact_id"])
			}
			if recordID != "" {
				resolution := migrationJournalString(entry["recommended_resolution"])
				if resolution == "" {
					resolution = "manual"
				}
				state.PendingRecords[recordID] = resolution
				state.Verdict = "reconcile_pending"
				state.LastError = ErrMigrationReconcilePending.Error()
				if state.ReconciliationPath == "" {
					state.ReconciliationPath = migrationReconciliationPath(pkg)
				}
			}
		case "reconcile_pending":
			pending := migrationPendingRecordsFromJournalValue(entry["pending_records"])
			if len(pending) > 0 {
				state.PendingRecords = pending
				state.Verdict = "reconcile_pending"
				state.LastError = ErrMigrationReconcilePending.Error()
				if state.ReconciliationPath == "" {
					state.ReconciliationPath = migrationReconciliationPath(pkg)
				}
			}
		case "reconcile_record":
			recordID := migrationJournalString(entry["record_id"])
			if recordID != "" && len(state.PendingRecords) > 0 {
				delete(state.PendingRecords, recordID)
				if len(state.PendingRecords) == 0 {
					state.ReconciliationPath = ""
				}
			}
		case "retry_committed", "aborted":
			state.PendingRecords = nil
			state.ReconciliationPath = ""
		}
	}

	if state.StepsCompleted < 0 {
		state.StepsCompleted = 0
	}
	if state.StepsCompleted > state.StepsPlanned {
		state.StepsCompleted = state.StepsPlanned
	}
	if state.LastStep < 0 {
		state.LastStep = 0
	}
	if state.LastStep > state.StepsPlanned {
		state.LastStep = state.StepsPlanned
	}
	if state.Verdict == "running" {
		state.Verdict = "step_failed"
		if state.LastError == "" {
			if state.LastStep > 0 {
				state.LastError = fmt.Sprintf("step %d interrupted before commit", state.LastStep)
			} else {
				state.LastError = ErrMigrationInterrupted.Error()
			}
		}
		if state.LastStep <= 0 && strings.HasPrefix(lastEvent, "step_") {
			state.LastStep = state.StepsCompleted + 1
			if state.LastStep > state.StepsPlanned {
				state.LastStep = state.StepsPlanned
			}
		}
	}

	return state
}

func migrationArtifactInverseFailuresFromJournal(pkg Package, state migrationState) map[string]string {
	if strings.TrimSpace(state.JournalPath) == "" {
		return nil
	}
	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(state.JournalPath))
	file, err := os.Open(filepath.Clean(absolutePath))
	if err != nil {
		return nil
	}
	defer func() {
		_ = file.Close()
	}()

	pending := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		switch migrationJournalString(entry["event"]) {
		case "artifact_inverse_failed":
			recordID := migrationJournalString(entry["record_id"])
			if recordID == "" {
				recordID = migrationJournalString(entry["artifact_id"])
			}
			if recordID == "" {
				continue
			}
			resolution := migrationJournalString(entry["recommended_resolution"])
			if resolution == "" {
				resolution = "manual"
			}
			pending[recordID] = resolution
		case "reconcile_record":
			if recordID := migrationJournalString(entry["record_id"]); recordID != "" {
				delete(pending, recordID)
			}
		case "retry_committed", "aborted":
			pending = map[string]string{}
		}
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
