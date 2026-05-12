package transport

import (
	"context"
	"sync"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
)

// bugReportCooldown is the minimum time between accepted reports from the
// same device. Duplicate presses within this window return the prior ack.
const bugReportCooldown = 10 * time.Second

type recentReport struct {
	ack  *diagnosticsv1.BugReportAck
	when time.Time
}

// DiagnosticsIntake owns persisted bug-report intake plumbing for the
// control stream: the swappable BugReportIntake, the nil-intake
// availability check, and the BugReportAck assembly. It owns its own
// mutex; StreamHandler does not share locks with it.
type DiagnosticsIntake struct {
	mu             sync.Mutex
	intake         BugReportIntake
	recentByDevice map[string]recentReport
	now            func() time.Time
}

// NewDiagnosticsIntake returns an intake collaborator. A nil intake is
// allowed; HandleBugReport will return ErrBugReportIntakeUnavailable
// until SetIntake wires one in.
func NewDiagnosticsIntake(intake BugReportIntake) *DiagnosticsIntake {
	return &DiagnosticsIntake{
		intake:         intake,
		recentByDevice: make(map[string]recentReport),
		now:            time.Now,
	}
}

// SetIntake swaps the underlying intake. Passing nil disables intake.
func (d *DiagnosticsIntake) SetIntake(intake BugReportIntake) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.intake = intake
}

// HandleBugReport files report through the configured intake and
// returns a ServerMessage containing the resulting ack. If no intake is
// configured, it returns ErrBugReportIntakeUnavailable. Reports from
// the same device within bugReportCooldown return the prior ack without
// re-filing. Errors from the underlying intake propagate unchanged.
func (d *DiagnosticsIntake) HandleBugReport(ctx context.Context, report *diagnosticsv1.BugReport) (ServerMessage, error) {
	d.mu.Lock()
	intake := d.intake
	deviceID := report.GetReporterDeviceId()
	now := d.now()
	if deviceID != "" {
		if prior, ok := d.recentByDevice[deviceID]; ok && now.Sub(prior.when) < bugReportCooldown {
			d.mu.Unlock()
			return ServerMessage{BugReportAck: prior.ack}, nil
		}
	}
	d.mu.Unlock()

	if intake == nil {
		return ServerMessage{}, ErrBugReportIntakeUnavailable
	}
	ack, err := intake.File(ctx, report)
	if err != nil {
		return ServerMessage{}, err
	}

	if deviceID != "" {
		d.mu.Lock()
		d.recentByDevice[deviceID] = recentReport{ack: ack, when: now}
		d.mu.Unlock()
	}
	return ServerMessage{BugReportAck: ack}, nil
}
