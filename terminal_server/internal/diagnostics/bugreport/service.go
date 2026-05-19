// Package bugreport stores and retrieves diagnostics bug reports.
package bugreport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog/query"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"google.golang.org/protobuf/encoding/protojson"
)

// Service stores and retrieves bug reports plus server-side enrichment.
type Service struct {
	mu            sync.Mutex
	logDir        string
	rootDir       string
	devices       *device.Manager
	runtime       *scenario.Runtime
	now           func() time.Time
	readAllEvents func(string) ([]query.Record, error)
	// subjectEventsQueryBudget bounds best-effort enrichment so ack emission
	// does not block on large eventlog scans.
	subjectEventsQueryBudget time.Duration
	jsonMarshal              protojson.MarshalOptions
}

const (
	autodetectDedupWindow = 10 * time.Minute
	autodetectTagsReason  = "suspected_failure"
)

// ListFilter narrows bug report list output for admin queries.
type ListFilter struct {
	SubjectDeviceID  string
	ReporterDeviceID string
	Source           string
	Tag              string
	FromUnixMS       int64
	ToUnixMS         int64
	ConfirmedOnly    bool
	PendingOnly      bool
}

// Summary is a list-friendly view over one stored report.
type Summary struct {
	ReportID           string   `json:"report_id"`
	CorrelationID      string   `json:"correlation_id"`
	Status             string   `json:"status"`
	Source             string   `json:"source"`
	ReporterDeviceID   string   `json:"reporter_device_id"`
	SubjectDeviceID    string   `json:"subject_device_id"`
	SubjectOffline     bool     `json:"subject_offline"`
	Tags               []string `json:"tags"`
	Description        string   `json:"description"`
	TimestampUnixMS    int64    `json:"timestamp_unix_ms"`
	CreatedUnixMS      int64    `json:"created_unix_ms"`
	ReportPath         string   `json:"report_path"`
	ScreenshotPath     string   `json:"screenshot_path,omitempty"`
	AudioPath          string   `json:"audio_path,omitempty"`
	MergedAutodetectID string   `json:"merged_autodetect_report_id,omitempty"`
	Confirmed          bool     `json:"confirmed"`
}

// SubjectSnapshot captures server-known state for the subject device.
type SubjectSnapshot struct {
	DeviceID          string   `json:"device_id"`
	DeviceName        string   `json:"device_name"`
	DeviceType        string   `json:"device_type"`
	Platform          string   `json:"platform"`
	State             string   `json:"state"`
	LastHeartbeatUnix int64    `json:"last_heartbeat_unix_ms"`
	Zone              string   `json:"zone,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	ActiveScenario    string   `json:"active_scenario,omitempty"`
}

// Record is the full persisted report shape returned to admin detail surfaces.
type Record struct {
	Summary       Summary          `json:"summary"`
	Report        map[string]any   `json:"report"`
	Subject       *SubjectSnapshot `json:"subject,omitempty"`
	SubjectEvents []query.Record   `json:"subject_event_tail"`
	Confirmed     bool             `json:"confirmed"`
	ConfirmedUnix int64            `json:"confirmed_unix_ms,omitempty"`
	ConfirmedBy   string           `json:"confirmed_by,omitempty"`
}

// NewService builds a diagnostics bug-report service rooted in logDir.
func NewService(logDir string, devices *device.Manager, runtime *scenario.Runtime) *Service {
	trimmed := strings.TrimSpace(logDir)
	if trimmed == "" {
		trimmed = "logs"
	}
	return &Service{
		logDir:                   trimmed,
		rootDir:                  filepath.Join(trimmed, "bug_reports"),
		devices:                  devices,
		runtime:                  runtime,
		now:                      time.Now,
		readAllEvents:            query.ReadAll,
		subjectEventsQueryBudget: 250 * time.Millisecond,
		jsonMarshal: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
	}
}

// File persists a bug report and returns a correlation-aware ack.
func (s *Service) File(ctx context.Context, in *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	return s.file(ctx, in, false)
}

// FileAutodetect files a suspected-failure report for a subject device.
func (s *Service) FileAutodetect(ctx context.Context, subjectDeviceID, description string, tags []string) (*diagnosticsv1.BugReportAck, error) {
	subjectDeviceID = strings.TrimSpace(subjectDeviceID)
	if subjectDeviceID == "" {
		return nil, fmt.Errorf("subject device id is required")
	}
	normalized := normalizeTags(append([]string{autodetectTagsReason}, tags...))
	return s.file(ctx, &diagnosticsv1.BugReport{
		SubjectDeviceId: subjectDeviceID,
		Source:          diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_AUTODETECT,
		Description:     strings.TrimSpace(description),
		Tags:            normalized,
	}, true)
}

func (s *Service) file(ctx context.Context, in *diagnosticsv1.BugReport, isAutodetect bool) (*diagnosticsv1.BugReportAck, error) {
	if in == nil {
		return nil, fmt.Errorf("bug report is required")
	}

	now := s.now().UTC()
	report := normalizeIncomingBugReport(in, now)
	if existing, ok, err := s.Get(report.GetReportId()); err != nil {
		return nil, fmt.Errorf("check existing bug report: %w", err)
	} else if ok {
		eventlog.Emit(eventlog.WithCorrelation(ctx, existing.Summary.CorrelationID), "bug.report.ack_replayed", slog.LevelInfo, "bug report ack replayed",
			slog.String("component", "diagnostics.bugreport"),
			slog.String("report_id", existing.Summary.ReportID),
			slog.String("correlation_id", existing.Summary.CorrelationID),
			slog.String("reporter_device_id", existing.Summary.ReporterDeviceID),
			slog.String("subject_device_id", existing.Summary.SubjectDeviceID),
			slog.String("report_path", existing.Summary.ReportPath),
		)
		return ackFromRecord(existing, "ack_replayed"), nil
	}

	plan := s.planBugReportFile(report, now, isAutodetect)
	if plan.earlyAck != nil {
		return plan.earlyAck, nil
	}

	subject, offline := s.subjectSnapshot(report.GetSubjectDeviceId())
	plan.summary.SubjectOffline = offline
	events := s.subjectEvents(ctx, report.GetSubjectDeviceId(), report.GetTimestampUnixMs())

	reportMap, err := s.bugReportRecordMap(report)
	if err != nil {
		return nil, err
	}

	rec := Record{
		Summary:       plan.summary,
		Report:        reportMap,
		Subject:       subject,
		SubjectEvents: events,
	}

	s.mu.Lock()
	summary, err := s.persistBugReportLocked(rec, report, now)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	emitBugReportFiled(ctx, report, summary)
	return &diagnosticsv1.BugReportAck{
		ReportId:                 summary.ReportID,
		CorrelationId:            summary.CorrelationID,
		Status:                   plan.status,
		ReportPath:               summary.ReportPath,
		MergedAutodetectReportId: plan.mergedAutodetectID,
		Message:                  bugReportFiledMessage(plan.status),
	}, nil
}

func ackFromRecord(rec Record, message string) *diagnosticsv1.BugReportAck {
	status := diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED
	if rec.Summary.Status == diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT.String() {
		status = diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT
	}
	return &diagnosticsv1.BugReportAck{
		ReportId:                 rec.Summary.ReportID,
		CorrelationId:            rec.Summary.CorrelationID,
		Status:                   status,
		ReportPath:               rec.Summary.ReportPath,
		MergedAutodetectReportId: rec.Summary.MergedAutodetectID,
		Message:                  message,
	}
}

// List returns summaries sorted newest-first.
func (s *Service) List() ([]Summary, error) {
	return s.ListFiltered(ListFilter{})
}

// ListFiltered returns summaries sorted newest-first with optional filters.
func (s *Service) ListFiltered(filter ListFilter) ([]Summary, error) {
	records, err := s.readAllRecords()
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(records))
	for _, rec := range records {
		rec.Summary.Confirmed = rec.Confirmed
		if !matchFilter(rec, filter) {
			continue
		}
		out = append(out, rec.Summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TimestampUnixMS == out[j].TimestampUnixMS {
			return out[i].CreatedUnixMS > out[j].CreatedUnixMS
		}
		return out[i].TimestampUnixMS > out[j].TimestampUnixMS
	})
	return out, nil
}

// Get returns one report by id.
func (s *Service) Get(reportID string) (Record, bool, error) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return Record{}, false, nil
	}
	records, err := s.readAllRecords()
	if err != nil {
		return Record{}, false, err
	}
	for _, rec := range records {
		if rec.Summary.ReportID == reportID {
			return rec, true, nil
		}
	}
	return Record{}, false, nil
}

// Confirm marks a stored report as confirmed.
func (s *Service) Confirm(ctx context.Context, reportID, confirmedBy string) (Record, bool, error) {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		return Record{}, false, nil
	}
	records, err := s.readAllRecords()
	if err != nil {
		return Record{}, false, err
	}
	for _, rec := range records {
		if rec.Summary.ReportID != reportID {
			continue
		}
		rec.Confirmed = true
		rec.Summary.Confirmed = true
		rec.ConfirmedUnix = s.now().UTC().UnixMilli()
		rec.ConfirmedBy = strings.TrimSpace(confirmedBy)

		encoded, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return Record{}, false, err
		}
		if strings.TrimSpace(rec.Summary.ReportPath) == "" {
			return Record{}, false, fmt.Errorf("report path is missing for %s", reportID)
		}
		if err := os.WriteFile(rec.Summary.ReportPath, encoded, 0o644); err != nil {
			return Record{}, false, err
		}
		eventlog.Emit(eventlog.WithCorrelation(ctx, rec.Summary.CorrelationID), "bug.report.confirmed", slog.LevelInfo, "bug report confirmed",
			slog.String("component", "diagnostics.bugreport"),
			slog.String("report_id", rec.Summary.ReportID),
			slog.String("correlation_id", rec.Summary.CorrelationID),
			slog.String("confirmed_by", rec.ConfirmedBy),
		)
		return rec, true, nil
	}
	return Record{}, false, nil
}

func matchFilter(rec Record, filter ListFilter) bool {
	return matchFilterConfirmation(rec, filter) &&
		matchFilterDevices(rec, filter) &&
		matchFilterSource(rec, filter) &&
		matchFilterTag(rec, filter) &&
		matchFilterTimeRange(rec, filter)
}

func matchFilterConfirmation(rec Record, filter ListFilter) bool {
	if filter.ConfirmedOnly && !rec.Confirmed {
		return false
	}
	if filter.PendingOnly && rec.Confirmed {
		return false
	}
	return true
}

func matchFilterDevices(rec Record, filter ListFilter) bool {
	subject := strings.TrimSpace(filter.SubjectDeviceID)
	if subject != "" && !strings.EqualFold(strings.TrimSpace(rec.Summary.SubjectDeviceID), subject) {
		return false
	}
	reporter := strings.TrimSpace(filter.ReporterDeviceID)
	return reporter == "" || strings.EqualFold(strings.TrimSpace(rec.Summary.ReporterDeviceID), reporter)
}

func matchFilterSource(rec Record, filter ListFilter) bool {
	source := normalizeSourceFilter(filter.Source)
	return source == "" || strings.TrimSpace(rec.Summary.Source) == source
}

func matchFilterTag(rec Record, filter ListFilter) bool {
	tag := strings.TrimSpace(strings.ToLower(filter.Tag))
	if tag == "" {
		return true
	}
	for _, item := range rec.Summary.Tags {
		if strings.TrimSpace(strings.ToLower(item)) == tag {
			return true
		}
	}
	return false
}

func matchFilterTimeRange(rec Record, filter ListFilter) bool {
	if filter.FromUnixMS > 0 && rec.Summary.TimestampUnixMS < filter.FromUnixMS {
		return false
	}
	if filter.ToUnixMS > 0 && rec.Summary.TimestampUnixMS > filter.ToUnixMS {
		return false
	}
	return true
}

func normalizeSourceFilter(raw string) string {
	trimmed := strings.TrimSpace(strings.ToUpper(raw))
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "BUG_REPORT_SOURCE_") {
		trimmed = "BUG_REPORT_SOURCE_" + trimmed
	}
	if _, ok := diagnosticsv1.BugReportSource_value[trimmed]; !ok {
		return ""
	}
	return trimmed
}

func (s *Service) findRecentAutodetectLocked(subjectID string, now time.Time, window time.Duration) *Record {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil
	}
	records, err := s.readAllRecords()
	if err != nil {
		return nil
	}
	cutoffUnix := now.Add(-window).UnixMilli()
	var best *Record
	for i := range records {
		rec := records[i]
		if strings.TrimSpace(rec.Summary.SubjectDeviceID) != subjectID {
			continue
		}
		if rec.Summary.Source != diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_AUTODETECT.String() {
			continue
		}
		if rec.Summary.TimestampUnixMS < cutoffUnix {
			continue
		}
		if best == nil || rec.Summary.TimestampUnixMS > best.Summary.TimestampUnixMS {
			copyRec := rec
			best = &copyRec
		}
	}
	return best
}

func (s *Service) readAllRecords() ([]Record, error) {
	entries := make([]string, 0, 128)
	err := filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if isBugReportRecordFile(d, walkErr) {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	records := make([]Record, 0, len(entries))
	for _, path := range entries {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var rec Record
		if unmarshalErr := json.Unmarshal(content, &rec); unmarshalErr != nil {
			continue
		}
		if strings.TrimSpace(rec.Summary.ReportPath) == "" {
			rec.Summary.ReportPath = path
		}
		records = append(records, rec)
	}
	return records, nil
}

func isBugReportRecordFile(d fs.DirEntry, walkErr error) bool {
	if walkErr != nil || d == nil || d.IsDir() {
		return false
	}
	return strings.HasSuffix(strings.ToLower(d.Name()), ".json")
}

func (s *Service) subjectSnapshot(subjectID string) (*SubjectSnapshot, bool) {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" || s.devices == nil {
		return nil, true
	}
	dev, ok := s.devices.Get(subjectID)
	if !ok {
		return nil, true
	}
	snapshot := &SubjectSnapshot{
		DeviceID:          dev.DeviceID,
		DeviceName:        dev.DeviceName,
		DeviceType:        dev.DeviceType,
		Platform:          dev.Platform,
		State:             string(dev.State),
		LastHeartbeatUnix: dev.LastHeartbeat.UTC().UnixMilli(),
		Zone:              dev.Placement.Zone,
		Roles:             append([]string(nil), dev.Placement.Roles...),
	}
	if s.runtime != nil && s.runtime.Engine != nil {
		if active, ok := s.runtime.Engine.Active(subjectID); ok {
			snapshot.ActiveScenario = active
		}
	}
	return snapshot, dev.State != device.StateConnected
}

func (s *Service) subjectEvents(ctx context.Context, subjectID string, reportUnixMS int64) []query.Record {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return nil
	}
	all := s.readAllSubjectEvents(ctx)
	if len(all) == 0 {
		return nil
	}
	return filterSubjectEvents(all, subjectID, reportUnixMS, s.now())
}

func readString(rec query.Record, key string) string {
	if rec == nil {
		return ""
	}
	if raw, ok := rec[key]; ok {
		if text, ok := raw.(string); ok {
			return text
		}
	}
	return ""
}

func readTime(rec query.Record) time.Time {
	raw := readString(rec, "ts")
	if raw == "" {
		return time.Time{}
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return ts.UTC()
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func makeReportID(now time.Time) string {
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	return fmt.Sprintf("bug-%s-%s", now.UTC().Format("20060102t150405.000"), hex.EncodeToString(randBytes))
}
