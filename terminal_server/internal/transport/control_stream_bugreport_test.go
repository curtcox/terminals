package transport

import (
	"context"
	"testing"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestHandleMessageBugReportRequiresIntake(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		BugReport: &diagnosticsv1.BugReport{ReportId: "bug-1"},
	})
	if err == nil {
		t.Fatalf("expected error when bug report intake is missing")
	}
	if err != ErrBugReportIntakeUnavailable {
		t.Fatalf("err = %v, want %v", err, ErrBugReportIntakeUnavailable)
	}
}

func TestHandleMessageBugReportReturnsAck(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	handler.SetBugReportIntake(&bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-2",
			CorrelationId: "bug:bug-2",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		BugReport: &diagnosticsv1.BugReport{ReportId: "bug-2"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(bug_report) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].BugReportAck == nil || out[0].BugReportAck.GetReportId() != "bug-2" {
		t.Fatalf("bug_report_ack = %+v, want report_id bug-2", out[0].BugReportAck)
	}
}

func TestHandleMessageInputBugReportActionFilesReport(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	handler.SetBugReportIntake(&bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-from-ui-action",
			CorrelationId: "bug:bug-from-ui-action",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: bugReportButtonID,
			Action:      "bug_report:subject-1",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input bug_report) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].BugReportAck == nil || out[0].BugReportAck.GetReportId() != "bug-from-ui-action" {
		t.Fatalf("first response bug_report_ack = %+v", out[0].BugReportAck)
	}
	if out[1].Notification == "" {
		t.Fatalf("second response should include filing notification")
	}
}

func TestHandleMessageInputBugReportActionRespectsModalitySources(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		wantSource diagnosticsv1.BugReportSource
	}{
		{name: "screen button", action: "bug_report", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON},
		{name: "gesture", action: "bug_report.gesture", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_GESTURE},
		{name: "shake", action: "bug_report.shake", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SHAKE},
		{name: "keyboard", action: "bug_report.keyboard", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_KEYBOARD},
		{name: "voice", action: "bug_report.voice", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_VOICE},
		{name: "qr", action: "bug_report.qr:subject-2", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_QR},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := device.NewManager()
			service := NewControlService("srv-1", manager)
			handler := NewStreamHandler(service)
			intake := &bugReportIntakeStub{
				ack: &diagnosticsv1.BugReportAck{
					ReportId:      "bug-from-modality",
					CorrelationId: "bug:bug-from-modality",
					Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
				},
			}
			handler.SetBugReportIntake(intake)

			_, err := handler.HandleMessage(context.Background(), ClientMessage{
				Input: &InputRequest{
					DeviceID:    "device-1",
					ComponentID: bugReportButtonID,
					Action:      tc.action,
				},
			})
			if err != nil {
				t.Fatalf("HandleMessage(input bug_report) error = %v", err)
			}
			if intake.lastReport == nil {
				t.Fatalf("expected intake to receive bug report payload")
			}
			if got := intake.lastReport.GetSource(); got != tc.wantSource {
				t.Fatalf("source = %v, want %v", got, tc.wantSource)
			}
			if tc.wantSource == diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_QR {
				if got := intake.lastReport.GetSubjectDeviceId(); got != "subject-2" {
					t.Fatalf("subject_device_id = %q, want subject-2", got)
				}
				return
			}
			if got := intake.lastReport.GetSubjectDeviceId(); got != "device-1" {
				t.Fatalf("subject_device_id = %q, want device-1", got)
			}
		})
	}
}
