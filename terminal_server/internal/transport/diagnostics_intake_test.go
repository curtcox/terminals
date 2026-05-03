package transport

import (
	"context"
	"errors"
	"sync"
	"testing"

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

type statelessIntakeStub struct{}

func (statelessIntakeStub) File(_ context.Context, _ *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	return &diagnosticsv1.BugReportAck{ReportId: "r"}, nil
}

func TestDiagnosticsIntakeSetIntakeRaceWithHandle(t *testing.T) {
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
