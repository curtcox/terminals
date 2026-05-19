package appruntime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readMigrationJournalLines(pkg Package, journalPath string) ([]map[string]any, error) {
	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(journalPath))
	file, err := os.Open(filepath.Clean(absolutePath))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var entries []map[string]any
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
		entries = append(entries, entry)
	}
	return entries, nil
}

func applyMigrationJournalScalars(entry map[string]any, state *migrationState) {
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
	} else if migrationJournalVerdictClearsErrors(state.Verdict) {
		state.LastError = ""
	}
	if blockedSince, ok := migrationJournalTime(entry["blocked_since"]); ok {
		state.DrainBlockedAt = blockedSince
	} else if migrationJournalVerdictClearsErrors(state.Verdict) {
		state.DrainBlockedAt = time.Time{}
	}
}

func migrationJournalVerdictClearsErrors(verdict string) bool {
	return verdict == "ok" || verdict == "running" || verdict == "idle"
}

func applyMigrationJournalEvent(pkg Package, event string, entry map[string]any, state *migrationState) {
	switch event {
	case "drain_ready_changed":
		if ready, ok := migrationJournalBool(entry["ready"]); ok {
			state.DrainReady = ready
			if ready {
				state.DrainBlockedAt = time.Time{}
			}
		}
	case "artifact_inverse_failed":
		applyMigrationJournalArtifactInverseFailed(pkg, entry, state)
	case "reconcile_pending":
		applyMigrationJournalReconcilePending(pkg, entry, state)
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

func applyMigrationJournalArtifactInverseFailed(pkg Package, entry map[string]any, state *migrationState) {
	if state.PendingRecords == nil {
		state.PendingRecords = map[string]string{}
	}
	recordID := migrationJournalString(entry["record_id"])
	if recordID == "" {
		recordID = migrationJournalString(entry["artifact_id"])
	}
	if recordID == "" {
		return
	}
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

func applyMigrationJournalReconcilePending(pkg Package, entry map[string]any, state *migrationState) {
	pending := migrationPendingRecordsFromJournalValue(entry["pending_records"])
	if len(pending) == 0 {
		return
	}
	state.PendingRecords = pending
	state.Verdict = "reconcile_pending"
	state.LastError = ErrMigrationReconcilePending.Error()
	if state.ReconciliationPath == "" {
		state.ReconciliationPath = migrationReconciliationPath(pkg)
	}
}

func finalizeMigrationJournalReplay(state *migrationState, lastEvent string) {
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
	if state.Verdict != "running" {
		return
	}
	state.Verdict = "step_failed"
	if state.LastError != "" {
		return
	}
	if state.LastStep > 0 {
		state.LastError = fmt.Sprintf("step %d interrupted before commit", state.LastStep)
		return
	}
	state.LastError = ErrMigrationInterrupted.Error()
	if state.LastStep > 0 || !strings.HasPrefix(lastEvent, "step_") {
		return
	}
	state.LastStep = state.StepsCompleted + 1
	if state.LastStep > state.StepsPlanned {
		state.LastStep = state.StepsPlanned
	}
}

func applyMigrationJournalPendingEvent(pending map[string]string, event string, entry map[string]any) map[string]string {
	switch event {
	case "artifact_inverse_failed":
		recordID := migrationJournalString(entry["record_id"])
		if recordID == "" {
			recordID = migrationJournalString(entry["artifact_id"])
		}
		if recordID == "" {
			return pending
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
		return map[string]string{}
	}
	return pending
}
