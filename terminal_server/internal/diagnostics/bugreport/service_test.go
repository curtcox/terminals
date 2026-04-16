package bugreport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
