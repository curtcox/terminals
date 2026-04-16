package bugreport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestServiceFileAndListAndGet(t *testing.T) {
	logDir := t.TempDir()
	devices := device.NewManager()
	_, err := devices.Register(device.Manifest{DeviceID: "rep-1", DeviceName: "Reporter"})
	if err != nil {
		t.Fatalf("register reporter: %v", err)
	}
	_, err = devices.Register(device.Manifest{DeviceID: "sub-1", DeviceName: "Subject"})
	if err != nil {
		t.Fatalf("register subject: %v", err)
	}

	svc := NewService(logDir, devices, nil)
	ack, err := svc.File(context.Background(), &diagnosticsv1.BugReport{
		ReporterDeviceId: "rep-1",
		SubjectDeviceId:  "sub-1",
		Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_ADMIN,
		Description:      "screen is frozen",
		Tags:             []string{"unresponsive", "UI_Glitch", "unresponsive"},
		ScreenshotPng:    []byte{0x89, 0x50, 0x4e, 0x47},
	})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if strings.TrimSpace(ack.GetReportId()) == "" {
		t.Fatalf("ack report_id should be populated")
	}
	if ack.GetStatus() != diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED {
		t.Fatalf("ack status = %v, want FILED", ack.GetStatus())
	}
	if !strings.Contains(ack.GetReportPath(), ack.GetReportId()+".json") {
		t.Fatalf("ack report_path = %q, want id suffix", ack.GetReportPath())
	}
	if _, statErr := os.Stat(ack.GetReportPath()); statErr != nil {
		t.Fatalf("report file should exist at %q: %v", ack.GetReportPath(), statErr)
	}
	if _, statErr := os.Stat(filepath.Join(logDir, "bug_reports")); statErr != nil {
		t.Fatalf("bug report root should exist: %v", statErr)
	}

	items, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List() len = %d, want 1", len(items))
	}
	if items[0].ReportID != ack.GetReportId() {
		t.Fatalf("list report id = %q, want %q", items[0].ReportID, ack.GetReportId())
	}
	if got := strings.Join(items[0].Tags, ","); got != "ui_glitch,unresponsive" {
		t.Fatalf("normalized tags = %q, want ui_glitch,unresponsive", got)
	}

	rec, ok, err := svc.Get(ack.GetReportId())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatalf("Get(%q) should exist", ack.GetReportId())
	}
	if rec.Summary.SubjectOffline {
		t.Fatalf("subject_offline = true, want false for connected subject")
	}
}

func TestServiceListFiltered(t *testing.T) {
	logDir := t.TempDir()
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "rep-1"})
	_, _ = devices.Register(device.Manifest{DeviceID: "sub-1"})
	_, _ = devices.Register(device.Manifest{DeviceID: "sub-2"})

	svc := NewService(logDir, devices, nil)
	base := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }
	_, _ = svc.File(context.Background(), &diagnosticsv1.BugReport{
		ReporterDeviceId: "rep-1",
		SubjectDeviceId:  "sub-1",
		Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_ADMIN,
		Tags:             []string{"unresponsive"},
		TimestampUnixMs:  base.UnixMilli(),
	})

	svc.now = func() time.Time { return base.Add(2 * time.Minute) }
	_, _ = svc.File(context.Background(), &diagnosticsv1.BugReport{
		ReporterDeviceId: "rep-1",
		SubjectDeviceId:  "sub-2",
		Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_WEBHOOK,
		Tags:             []string{"lost_connection"},
		TimestampUnixMs:  base.Add(2 * time.Minute).UnixMilli(),
	})

	got, err := svc.ListFiltered(ListFilter{
		SubjectDeviceID: "sub-2",
		Source:          "webhook",
		Tag:             "lost_connection",
		FromUnixMS:      base.Add(time.Minute).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("ListFiltered() error = %v", err)
	}
	if len(got) != 1 || got[0].SubjectDeviceID != "sub-2" {
		t.Fatalf("ListFiltered() = %+v, want only sub-2 report", got)
	}
}

func TestServiceAutodetectMerge(t *testing.T) {
	logDir := t.TempDir()
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "rep-1"})
	_, _ = devices.Register(device.Manifest{DeviceID: "sub-1"})

	svc := NewService(logDir, devices, nil)
	base := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }
	autoAck, err := svc.FileAutodetect(context.Background(), "sub-1", "heartbeat timeout", nil)
	if err != nil {
		t.Fatalf("FileAutodetect() error = %v", err)
	}
	if autoAck.GetStatus() != diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED {
		t.Fatalf("autodetect status = %v, want FILED", autoAck.GetStatus())
	}

	svc.now = func() time.Time { return base.Add(2 * time.Minute) }
	userAck, err := svc.File(context.Background(), &diagnosticsv1.BugReport{
		ReporterDeviceId: "rep-1",
		SubjectDeviceId:  "sub-1",
		Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_ADMIN,
		Description:      "screen frozen",
	})
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if userAck.GetStatus() != diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT {
		t.Fatalf("user status = %v, want MERGED_WITH_AUTODETECT", userAck.GetStatus())
	}
	if userAck.GetMergedAutodetectReportId() != autoAck.GetReportId() {
		t.Fatalf("merged_autodetect_report_id = %q, want %q", userAck.GetMergedAutodetectReportId(), autoAck.GetReportId())
	}
}
