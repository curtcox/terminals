package transport

import (
	"context"
	"errors"
	"testing"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// TestHandleMessageBugReportIntakeErrorPropagates pins down that when the
// configured bug-report intake returns an error, HandleMessage forwards it
// as the (err) result and emits exactly one ServerMessage carrying the
// error text and the unknown_error code (no dedicated mapping today).
//
// This characterizes the path that will move into DiagnosticsIntake in
// Phase 6; the extracted collaborator must preserve identical shape.
func TestHandleMessageBugReportIntakeErrorPropagates(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	intakeErr := errors.New("intake offline")
	handler.SetBugReportIntake(&bugReportIntakeStub{err: intakeErr})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		BugReport: &diagnosticsv1.BugReport{ReportId: "bug-err-1"},
	})
	if !errors.Is(err, intakeErr) {
		t.Fatalf("err = %v, want %v", err, intakeErr)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].BugReportAck != nil {
		t.Fatalf("BugReportAck = %+v, want nil for error path", out[0].BugReportAck)
	}
	if out[0].Error != intakeErr.Error() {
		t.Fatalf("Error = %q, want %q", out[0].Error, intakeErr.Error())
	}
	if out[0].ErrorCode != ErrorCodeUnknown {
		t.Fatalf("ErrorCode = %q, want %q", out[0].ErrorCode, ErrorCodeUnknown)
	}
}

// TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse pins
// down the response shape for command validation failures. Each variant
// returns exactly one ServerMessage, populated with the stable error code
// and the sentinel's Error() text, and forwards the sentinel error.
//
// This characterizes the surface CommandDispatcher must replicate when
// extracted in Phase 5.
func TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse(t *testing.T) {
	cases := []struct {
		name    string
		req     *CommandRequest
		wantErr error
		wantCode string
	}{
		{
			name: "invalid action",
			req: &CommandRequest{
				RequestID: "bad-action",
				DeviceID:  "device-1",
				Kind:      "manual",
				Intent:    "photo frame",
				Action:    "pause",
			},
			wantErr:  ErrInvalidCommandAction,
			wantCode: ErrorCodeInvalidCommandAction,
		},
		{
			name: "invalid kind",
			req: &CommandRequest{
				RequestID: "bad-kind",
				DeviceID:  "device-1",
				Kind:      "remote",
				Intent:    "photo frame",
			},
			wantErr:  ErrInvalidCommandKind,
			wantCode: ErrorCodeInvalidCommandKind,
		},
		{
			name: "missing manual intent",
			req: &CommandRequest{
				RequestID: "bad-intent",
				DeviceID:  "device-1",
				Kind:      "manual",
				Intent:    "   ",
			},
			wantErr:  ErrMissingCommandIntent,
			wantCode: ErrorCodeMissingIntent,
		},
		{
			name: "missing voice text",
			req: &CommandRequest{
				RequestID: "bad-text",
				DeviceID:  "device-1",
				Kind:      "voice",
				Text:      "",
			},
			wantErr:  ErrMissingCommandText,
			wantCode: ErrorCodeMissingText,
		},
		{
			name: "missing device id",
			req: &CommandRequest{
				RequestID: "bad-device",
				Kind:      "manual",
				Intent:    "photo frame",
			},
			wantErr:  ErrMissingCommandDeviceID,
			wantCode: ErrorCodeMissingDeviceID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			devices := device.NewManager()
			control := NewControlService("srv-1", devices)
			engine := scenario.NewEngine()
			scenario.RegisterBuiltins(engine)
			runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
			handler := NewStreamHandlerWithRuntime(control, runtime)

			_, _ = handler.HandleMessage(context.Background(), ClientMessage{
				Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
			})

			out, err := handler.HandleMessage(context.Background(), ClientMessage{Command: tc.req})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if len(out) != 1 {
				t.Fatalf("len(out) = %d, want 1 (out=%+v)", len(out), out)
			}
			if out[0].ErrorCode != tc.wantCode {
				t.Fatalf("ErrorCode = %q, want %q", out[0].ErrorCode, tc.wantCode)
			}
			if out[0].Error != tc.wantErr.Error() {
				t.Fatalf("Error = %q, want %q", out[0].Error, tc.wantErr.Error())
			}
			if out[0].CommandAck != "" {
				t.Fatalf("CommandAck = %q, want empty for validation error", out[0].CommandAck)
			}
		})
	}
}

// TestHandleMessageCommandRecordsValidationErrorInRecentEvents pins down
// that command validation failures append a CommandEvent with an
// outcome prefix of "error:" and the original error string. This
// captures the recent-command audit behavior CommandDispatcher must
// preserve in Phase 5.
func TestHandleMessageCommandRecordsValidationErrorInRecentEvents(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "audit-bad-kind",
			DeviceID:  "device-1",
			Kind:      "remote",
			Intent:    "photo frame",
		},
	})
	if !errors.Is(err, ErrInvalidCommandKind) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidCommandKind)
	}

	handler.mu.Lock()
	events := append([]CommandEvent(nil), handler.recent...)
	handler.mu.Unlock()
	if len(events) == 0 {
		t.Fatalf("expected at least one recent command event")
	}
	last := events[len(events)-1]
	if last.RequestID != "audit-bad-kind" {
		t.Fatalf("RequestID = %q, want audit-bad-kind", last.RequestID)
	}
	if last.Outcome != "error:"+ErrInvalidCommandKind.Error() {
		t.Fatalf("Outcome = %q, want %q", last.Outcome, "error:"+ErrInvalidCommandKind.Error())
	}
}
