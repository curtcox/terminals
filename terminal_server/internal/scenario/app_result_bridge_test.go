package scenario

import (
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
)

func TestScenarioResultFromAppRuntimeResult(t *testing.T) {
	occurredAt := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	result := ResultFromAppRuntime(appruntime.Result{
		State: map[string]string{"status": "running"},
		Ops: []appruntime.Op{{
			Kind:   OperationBroadcastNotify,
			Target: "d1",
			Args:   map[string]string{"message": "hello"},
		}},
		Emit: []appruntime.Trigger{{
			Kind:       "timer.expired",
			Subject:    "pasta",
			Attributes: map[string]string{"duration_seconds": "600"},
			OccurredAt: occurredAt,
		}},
		Done: true,
	})

	if result.State == nil || !result.Done {
		t.Fatalf("result = %+v, want state and done", result)
	}
	if len(result.Ops) != 1 || result.Ops[0].Kind != OperationBroadcastNotify || result.Ops[0].Args["message"] != "hello" {
		t.Fatalf("ops = %+v", result.Ops)
	}
	if len(result.Emit) != 1 || result.Emit[0].EventV2 == nil || result.Emit[0].EventV2.Kind != "timer.expired" || result.Emit[0].EventV2.Subject != "pasta" {
		t.Fatalf("emit = %+v", result.Emit)
	}
	if result.Emit[0].EventV2.Attributes["duration_seconds"] != "600" || !result.Emit[0].EventV2.OccurredAt.Equal(occurredAt) {
		t.Fatalf("emit attributes/time = %+v", result.Emit[0].EventV2)
	}
}
