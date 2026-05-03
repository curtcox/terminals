package transport

import (
	"context"
	"sync"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
)

// DiagnosticsIntake owns persisted bug-report intake plumbing for the
// control stream: the swappable BugReportIntake, the nil-intake
// availability check, and the BugReportAck assembly. It owns its own
// mutex; StreamHandler does not share locks with it.
type DiagnosticsIntake struct {
	mu     sync.Mutex
	intake BugReportIntake
}

// NewDiagnosticsIntake returns an intake collaborator. A nil intake is
// allowed; HandleBugReport will return ErrBugReportIntakeUnavailable
// until SetIntake wires one in.
func NewDiagnosticsIntake(intake BugReportIntake) *DiagnosticsIntake {
	return &DiagnosticsIntake{intake: intake}
}

// SetIntake swaps the underlying intake. Passing nil disables intake.
func (d *DiagnosticsIntake) SetIntake(intake BugReportIntake) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.intake = intake
}

// HandleBugReport files report through the configured intake and
// returns a ServerMessage containing the resulting ack. If no intake is
// configured, it returns ErrBugReportIntakeUnavailable. Errors from the
// underlying intake propagate unchanged.
func (d *DiagnosticsIntake) HandleBugReport(ctx context.Context, report *diagnosticsv1.BugReport) (ServerMessage, error) {
	d.mu.Lock()
	intake := d.intake
	d.mu.Unlock()
	if intake == nil {
		return ServerMessage{}, ErrBugReportIntakeUnavailable
	}
	ack, err := intake.File(ctx, report)
	if err != nil {
		return ServerMessage{}, err
	}
	return ServerMessage{BugReportAck: ack}, nil
}
