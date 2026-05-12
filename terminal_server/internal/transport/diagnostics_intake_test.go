package transport

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
)

func TestDiagnosticsIntakeNilReturnsUnavailable(t *testing.T) {
	d := NewDiagnosticsIntake(nil)
	msg, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{ReportId: "x"})
	if !errors.Is(err, ErrBugReportIntakeUnavailable) {
		t.Fatalf("err = %v, want %v", err, ErrBugReportIntakeUnavailable)
	}
	if msg.BugReportAck != nil {
		t.Fatalf("ack = %+v, want nil on unavailable", msg.BugReportAck)
	}
}

func TestDiagnosticsIntakeReturnsAck(t *testing.T) {
	stub := &bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId: "ack-1",
			Status:   diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	}
	d := NewDiagnosticsIntake(stub)
	msg, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{ReportId: "in-1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if msg.BugReportAck == nil || msg.BugReportAck.GetReportId() != "ack-1" {
		t.Fatalf("ack = %+v, want report_id ack-1", msg.BugReportAck)
	}
	if stub.lastReport == nil || stub.lastReport.GetReportId() != "in-1" {
		t.Fatalf("intake did not see report: %+v", stub.lastReport)
	}
}

func TestDiagnosticsIntakeErrorPropagatesUnchanged(t *testing.T) {
	sentinel := errors.New("intake exploded")
	d := NewDiagnosticsIntake(&bugReportIntakeStub{err: sentinel})
	msg, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{ReportId: "x"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v via errors.Is", err, sentinel)
	}
	if msg.BugReportAck != nil {
		t.Fatalf("ack = %+v, want nil on error", msg.BugReportAck)
	}
}

type ctxAwareIntakeStub struct{}

func (ctxAwareIntakeStub) File(ctx context.Context, _ *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDiagnosticsIntakeContextCancellationPropagates(t *testing.T) {
	d := NewDiagnosticsIntake(ctxAwareIntakeStub{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := d.HandleBugReport(ctx, &diagnosticsv1.BugReport{ReportId: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDiagnosticsIntakeDedupWithinCooldown(t *testing.T) {
	stub := &bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId: "original",
			Status:   diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	}
	now := time.Now()
	d := NewDiagnosticsIntake(stub)
	d.now = func() time.Time { return now }

	report := &diagnosticsv1.BugReport{
		ReportId:         "r1",
		ReporterDeviceId: "device-1",
	}
	msg1, err := d.HandleBugReport(context.Background(), report)
	if err != nil {
		t.Fatalf("first report: unexpected err: %v", err)
	}
	if msg1.BugReportAck.GetReportId() != "original" {
		t.Fatalf("first ack report_id = %q, want original", msg1.BugReportAck.GetReportId())
	}
	if stub.callCount != 1 {
		t.Fatalf("intake call count = %d, want 1 after first report", stub.callCount)
	}

	// Second press 2 seconds later — within cooldown — must NOT call intake again.
	d.now = func() time.Time { return now.Add(2 * time.Second) }
	stub.ack = &diagnosticsv1.BugReportAck{ReportId: "should-not-appear"}
	msg2, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{
		ReportId:         "r2",
		ReporterDeviceId: "device-1",
	})
	if err != nil {
		t.Fatalf("second report: unexpected err: %v", err)
	}
	if msg2.BugReportAck.GetReportId() != "original" {
		t.Fatalf("second ack report_id = %q, want original (deduped)", msg2.BugReportAck.GetReportId())
	}
	if stub.callCount != 1 {
		t.Fatalf("intake call count = %d, want 1 (second should be deduped)", stub.callCount)
	}
}

func TestDiagnosticsIntakeAllowsReportAfterCooldown(t *testing.T) {
	stub := &bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{ReportId: "first"},
	}
	now := time.Now()
	d := NewDiagnosticsIntake(stub)
	d.now = func() time.Time { return now }

	_, _ = d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{
		ReportId: "r1", ReporterDeviceId: "device-1",
	})

	// After cooldown elapses a new report should be filed.
	d.now = func() time.Time { return now.Add(bugReportCooldown + time.Second) }
	stub.ack = &diagnosticsv1.BugReportAck{ReportId: "second"}
	msg, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{
		ReportId: "r2", ReporterDeviceId: "device-1",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if msg.BugReportAck.GetReportId() != "second" {
		t.Fatalf("ack = %q, want second", msg.BugReportAck.GetReportId())
	}
	if stub.callCount != 2 {
		t.Fatalf("intake call count = %d, want 2", stub.callCount)
	}
}

func TestDiagnosticsIntakeDifferentDevicesNotDeduped(t *testing.T) {
	calls := 0
	stub := &bugReportIntakeStub{}
	stub.ack = &diagnosticsv1.BugReportAck{ReportId: "r"}
	now := time.Now()
	d := NewDiagnosticsIntake(stub)
	d.now = func() time.Time { return now }
	_ = calls

	for _, id := range []string{"device-a", "device-b"} {
		stub.ack = &diagnosticsv1.BugReportAck{ReportId: id + "-ack"}
		msg, err := d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{
			ReportId: id, ReporterDeviceId: id,
		})
		if err != nil {
			t.Fatalf("device %s: unexpected err: %v", id, err)
		}
		if msg.BugReportAck.GetReportId() != id+"-ack" {
			t.Fatalf("device %s: ack = %q, want %s-ack", id, msg.BugReportAck.GetReportId(), id)
		}
	}
	if stub.callCount != 2 {
		t.Fatalf("intake call count = %d, want 2 (one per device)", stub.callCount)
	}
}

type statelessIntakeStub struct{}

func (statelessIntakeStub) File(_ context.Context, _ *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	return &diagnosticsv1.BugReportAck{ReportId: "r"}, nil
}

func TestDiagnosticsIntakeSetIntakeRaceWithHandle(_ *testing.T) {
	d := NewDiagnosticsIntake(statelessIntakeStub{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			d.SetIntake(statelessIntakeStub{})
		}()
		go func() {
			defer wg.Done()
			_, _ = d.HandleBugReport(context.Background(), &diagnosticsv1.BugReport{ReportId: "x"})
		}()
	}
	wg.Wait()
}
