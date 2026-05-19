package bugreport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog/query"
	"google.golang.org/protobuf/proto"
)

type bugReportFilePlan struct {
	report             *diagnosticsv1.BugReport
	summary            Summary
	status             diagnosticsv1.BugReportStatus
	mergedAutodetectID string
	earlyAck           *diagnosticsv1.BugReportAck
}

func normalizeIncomingBugReport(in *diagnosticsv1.BugReport, now time.Time) *diagnosticsv1.BugReport {
	report := proto.Clone(in).(*diagnosticsv1.BugReport)
	if report.GetTimestampUnixMs() <= 0 {
		report.TimestampUnixMs = now.UnixMilli()
	}
	report.ReporterDeviceId = strings.TrimSpace(report.GetReporterDeviceId())
	report.SubjectDeviceId = strings.TrimSpace(report.GetSubjectDeviceId())
	report.Description = strings.TrimSpace(report.GetDescription())
	report.Tags = normalizeTags(report.GetTags())
	if report.GetReportId() == "" {
		report.ReportId = makeReportID(now)
	}
	if report.ReporterDeviceId == "" && report.GetClientContext() != nil && report.GetClientContext().GetIdentity() != nil {
		report.ReporterDeviceId = strings.TrimSpace(report.GetClientContext().GetIdentity().GetDeviceId())
	}
	if report.SubjectDeviceId == "" {
		report.SubjectDeviceId = report.ReporterDeviceId
	}
	return report
}

func newBugReportSummary(report *diagnosticsv1.BugReport, now time.Time) Summary {
	summary := Summary{
		ReportID:         report.GetReportId(),
		CorrelationID:    "bug:" + report.GetReportId(),
		Status:           diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED.String(),
		Source:           report.GetSource().String(),
		ReporterDeviceID: report.GetReporterDeviceId(),
		SubjectDeviceID:  report.GetSubjectDeviceId(),
		Tags:             append([]string(nil), report.GetTags()...),
		Description:      report.GetDescription(),
		TimestampUnixMS:  report.GetTimestampUnixMs(),
		CreatedUnixMS:    now.UnixMilli(),
	}
	if summary.Source == diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_UNSPECIFIED.String() {
		summary.Source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_OTHER.String()
	}
	return summary
}

func (s *Service) planBugReportFile(report *diagnosticsv1.BugReport, now time.Time, isAutodetect bool) bugReportFilePlan {
	plan := bugReportFilePlan{
		report:  report,
		summary: newBugReportSummary(report, now),
		status:  diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
	}
	if strings.TrimSpace(plan.summary.SubjectDeviceID) == "" {
		return plan
	}
	recent := s.findRecentAutodetectLocked(plan.summary.SubjectDeviceID, now, autodetectDedupWindow)
	if recent == nil {
		return plan
	}
	if isAutodetect {
		plan.earlyAck = &diagnosticsv1.BugReportAck{
			ReportId:                 recent.Summary.ReportID,
			CorrelationId:            recent.Summary.CorrelationID,
			Status:                   diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT,
			ReportPath:               recent.Summary.ReportPath,
			MergedAutodetectReportId: recent.Summary.ReportID,
			Message:                  "merged_with_autodetect",
		}
		return plan
	}
	if report.GetSource() == diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_AUTODETECT {
		return plan
	}
	plan.mergedAutodetectID = recent.Summary.ReportID
	plan.status = diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT
	plan.summary.MergedAutodetectID = recent.Summary.ReportID
	plan.summary.Status = diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT.String()
	return plan
}

func (s *Service) bugReportRecordMap(report *diagnosticsv1.BugReport) (map[string]any, error) {
	recordReport := proto.Clone(report).(*diagnosticsv1.BugReport)
	reportJSON, err := s.jsonMarshal.Marshal(recordReport)
	if err != nil {
		return nil, fmt.Errorf("marshal bug report: %w", err)
	}
	reportMap := map[string]any{}
	if err := json.Unmarshal(reportJSON, &reportMap); err != nil {
		return nil, fmt.Errorf("decode bug report json: %w", err)
	}
	return reportMap, nil
}

func (s *Service) persistBugReportLocked(rec Record, report *diagnosticsv1.BugReport, now time.Time) (Summary, error) {
	dateDir := time.UnixMilli(report.GetTimestampUnixMs()).UTC().Format("2006-01-02")
	if dateDir == "" || dateDir == "0001-01-01" {
		dateDir = now.Format("2006-01-02")
	}
	relDir := filepath.ToSlash(filepath.Join("bug_reports", dateDir))
	absDir := filepath.Join(s.logDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create bug report dir: %w", err)
	}
	if png := report.GetScreenshotPng(); len(png) > 0 {
		name := report.GetReportId() + ".screenshot.png"
		rel := filepath.ToSlash(filepath.Join(relDir, name))
		if err := os.WriteFile(filepath.Join(s.logDir, rel), png, 0o644); err != nil {
			return Summary{}, fmt.Errorf("write screenshot: %w", err)
		}
		rec.Summary.ScreenshotPath = filepath.ToSlash(filepath.Join(s.logDir, rel))
	}
	if wav := report.GetAudioWav(); len(wav) > 0 {
		name := report.GetReportId() + ".audio.wav"
		rel := filepath.ToSlash(filepath.Join(relDir, name))
		if err := os.WriteFile(filepath.Join(s.logDir, rel), wav, 0o644); err != nil {
			return Summary{}, fmt.Errorf("write audio: %w", err)
		}
		rec.Summary.AudioPath = filepath.ToSlash(filepath.Join(s.logDir, rel))
	}
	relJSON := filepath.ToSlash(filepath.Join(relDir, report.GetReportId()+".json"))
	rec.Summary.ReportPath = filepath.ToSlash(filepath.Join(s.logDir, relJSON))
	encoded, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return Summary{}, fmt.Errorf("encode bug report record: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.logDir, relJSON), encoded, 0o644); err != nil {
		return Summary{}, fmt.Errorf("write bug report record: %w", err)
	}
	return rec.Summary, nil
}

func emitBugReportFiled(ctx context.Context, report *diagnosticsv1.BugReport, summary Summary) {
	eventCtx := eventlog.WithCorrelation(ctx, summary.CorrelationID)
	eventName := "bug.report.filed"
	eventMsg := "bug report filed"
	if report.GetSource() == diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_AUTODETECT {
		eventName = "bug.report.autodetected"
		eventMsg = "bug report autodetected"
	}
	eventAttrs := []slog.Attr{
		slog.String("component", "diagnostics.bugreport"),
		slog.String("report_id", summary.ReportID),
		slog.String("correlation_id", summary.CorrelationID),
		slog.String("reporter_device_id", summary.ReporterDeviceID),
		slog.String("subject_device_id", summary.SubjectDeviceID),
		slog.String("source", summary.Source),
		slog.String("report_path", summary.ReportPath),
		slog.Int("tag_count", len(summary.Tags)),
		slog.Bool("subject_offline", summary.SubjectOffline),
	}
	if tokenWord := strings.TrimSpace(report.GetSourceHints()["bug_token_word"]); tokenWord != "" {
		eventAttrs = append(eventAttrs, slog.String("bug_token_word", tokenWord))
	}
	if tokenCode := strings.TrimSpace(report.GetSourceHints()["bug_token_code"]); tokenCode != "" {
		eventAttrs = append(eventAttrs, slog.String("bug_token_code", tokenCode))
	}
	eventlog.Emit(eventCtx, eventName, slog.LevelInfo, eventMsg, eventAttrs...)
}

func bugReportFiledMessage(status diagnosticsv1.BugReportStatus) string {
	if status == diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT {
		return "merged_with_autodetect"
	}
	return "filed"
}

func (s *Service) readAllSubjectEvents(ctx context.Context) []query.Record {
	readAll := s.readAllEvents
	if readAll == nil {
		readAll = query.ReadAll
	}
	queryCtx := ctx
	if queryCtx == nil {
		queryCtx = context.Background()
	}
	if budget := s.subjectEventsQueryBudget; budget > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(queryCtx, budget)
		defer cancel()
	}
	type eventReadResult struct {
		records []query.Record
		err     error
	}
	readDone := make(chan eventReadResult, 1)
	go func() {
		all, err := readAll(s.logDir)
		readDone <- eventReadResult{records: all, err: err}
	}()
	select {
	case <-queryCtx.Done():
		return nil
	case result := <-readDone:
		if result.err != nil || len(result.records) == 0 {
			return nil
		}
		return result.records
	}
}

func filterSubjectEvents(all []query.Record, subjectID string, reportUnixMS int64, now time.Time) []query.Record {
	reportTime := time.UnixMilli(reportUnixMS).UTC()
	if reportUnixMS <= 0 {
		reportTime = now.UTC()
	}
	windowStart := reportTime.Add(-5 * time.Minute)
	out := make([]query.Record, 0, 64)
	for _, rec := range all {
		devID := strings.TrimSpace(readString(rec, "device_id"))
		if devID == "" {
			devID = strings.TrimSpace(readString(rec, "subject_device_id"))
		}
		if devID != subjectID {
			continue
		}
		ts := readTime(rec)
		if ts.IsZero() || ts.Before(windowStart) || ts.After(reportTime) {
			continue
		}
		out = append(out, rec)
		if len(out) > 64 {
			out = out[len(out)-64:]
		}
	}
	return out
}
