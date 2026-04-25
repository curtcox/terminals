package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
)

// Operation kind constants identify host capabilities the executor can commit.
const (
	OperationUISet           = "ui.set"
	OperationUIPatch         = "ui.patch"
	OperationUITransition    = "ui.transition"
	OperationSchedulerAfter  = "scheduler.after"
	OperationSchedulerCancel = "scheduler.cancel"
	OperationAITTS           = "ai.tts"
	OperationBusEmit         = "bus.emit"
	OperationBroadcastNotify = "broadcast.notify"
	OperationFlowApply       = "flow.apply"
	OperationFlowStop        = "flow.stop"
)

// Operation is a typed side effect returned by result-oriented scenarios.
type Operation struct {
	Kind   string
	Target string
	Args   map[string]string
}

// ScenarioResult is the all-or-nothing result model shared by Go scenarios and TAL apps.
//
//revive:disable-next-line:exported
type ScenarioResult struct {
	State any
	Ops   []Operation
	Emit  []Trigger
	Done  bool
}

// ResultScenario can return operations instead of performing side effects directly.
type ResultScenario interface {
	Scenario
	StartResult(ctx context.Context, env *Environment) (ScenarioResult, error)
}

// ExecuteOperations validates all operations before committing any side effects.
func ExecuteOperations(ctx context.Context, env *Environment, ops []Operation, now time.Time) error {
	if err := ValidateOperations(env, ops); err != nil {
		eventlog.Emit(ctx, "scenario.ops.failed", slog.LevelError, "scenario operations validation failed",
			slog.String("component", "scenario.operations"),
			slog.Any("error", err),
		)
		return err
	}
	eventlog.Emit(ctx, "scenario.ops.validated", slog.LevelInfo, "scenario operations validated",
		slog.String("component", "scenario.operations"),
		slog.Int("operation_count", len(ops)),
	)

	for _, op := range ops {
		if err := executeOperation(ctx, env, op, now); err != nil {
			eventlog.Emit(ctx, "scenario.ops.failed", slog.LevelError, "scenario operation commit failed",
				slog.String("component", "scenario.operations"),
				slog.String("operation_kind", op.Kind),
				slog.Any("error", err),
			)
			return err
		}
	}
	eventlog.Emit(ctx, "scenario.ops.committed", slog.LevelInfo, "scenario operations committed",
		slog.String("component", "scenario.operations"),
		slog.Int("operation_count", len(ops)),
	)
	return nil
}

// ValidateOperations checks operation shape and required host dependencies.
func ValidateOperations(env *Environment, ops []Operation) error {
	for i, op := range ops {
		if op.Args == nil {
			op.Args = map[string]string{}
		}
		switch op.Kind {
		case OperationSchedulerAfter:
			if env == nil || env.Scheduler == nil {
				return fmt.Errorf("op %d %s requires scheduler", i, op.Kind)
			}
			if strings.TrimSpace(op.Target) == "" {
				return fmt.Errorf("op %d %s requires target key", i, op.Kind)
			}
			if _, err := strconv.ParseInt(strings.TrimSpace(op.Args["unix_ms"]), 10, 64); err != nil {
				return fmt.Errorf("op %d %s requires unix_ms: %w", i, op.Kind, err)
			}
		case OperationSchedulerCancel:
			if env == nil || env.Scheduler == nil {
				return fmt.Errorf("op %d %s requires scheduler", i, op.Kind)
			}
			if strings.TrimSpace(op.Target) == "" {
				return fmt.Errorf("op %d %s requires target key", i, op.Kind)
			}
		case OperationBroadcastNotify:
			if env == nil || env.Broadcast == nil {
				return fmt.Errorf("op %d %s requires broadcaster", i, op.Kind)
			}
			if strings.TrimSpace(op.Args["message"]) == "" {
				return fmt.Errorf("op %d %s requires message", i, op.Kind)
			}
		case OperationAITTS:
			if env == nil || env.TTS == nil {
				return fmt.Errorf("op %d %s requires tts", i, op.Kind)
			}
			if strings.TrimSpace(op.Args["text"]) == "" {
				return fmt.Errorf("op %d %s requires text", i, op.Kind)
			}
		case OperationBusEmit:
			if env == nil || env.TriggerBus == nil {
				return fmt.Errorf("op %d %s requires trigger bus", i, op.Kind)
			}
			if strings.TrimSpace(op.Target) == "" {
				return fmt.Errorf("op %d %s requires event kind target", i, op.Kind)
			}
		case OperationUISet, OperationUIPatch, OperationUITransition, OperationFlowApply, OperationFlowStop:
			return fmt.Errorf("op %d %s is not executable yet", i, op.Kind)
		default:
			return fmt.Errorf("op %d has unsupported kind %q", i, op.Kind)
		}
	}
	return nil
}

func executeOperation(ctx context.Context, env *Environment, op Operation, now time.Time) error {
	switch op.Kind {
	case OperationSchedulerAfter:
		unixMS, _ := strconv.ParseInt(strings.TrimSpace(op.Args["unix_ms"]), 10, 64)
		if structured, ok := env.Scheduler.(interface {
			ScheduleRecord(context.Context, storage.ScheduleRecord) error
		}); ok {
			return structured.ScheduleRecord(ctx, storage.ScheduleRecord{
				Key:      strings.TrimSpace(op.Target),
				Kind:     strings.TrimSpace(op.Args["kind"]),
				Subject:  strings.TrimSpace(op.Args["subject"]),
				DeviceID: strings.TrimSpace(op.Args["device_id"]),
				UnixMS:   unixMS,
				Payload:  operationPayload(op.Args, "duration_seconds"),
			})
		}
		return env.Scheduler.Schedule(ctx, strings.TrimSpace(op.Target), unixMS)
	case OperationSchedulerCancel:
		return env.Scheduler.Remove(ctx, strings.TrimSpace(op.Target))
	case OperationBroadcastNotify:
		return env.Broadcast.Notify(ctx, splitTargetList(op.Target), strings.TrimSpace(op.Args["message"]))
	case OperationAITTS:
		_, err := env.TTS.Synthesize(ctx, strings.TrimSpace(op.Args["text"]), TTSOptions{
			Voice:  stringDefault(op.Args["voice"], "default"),
			Format: stringDefault(op.Args["format"], "pcm16"),
		})
		return err
	case OperationBusEmit:
		env.TriggerBus.Publish(Trigger{
			Kind: TriggerEvent,
			EventV2: &EventRecord{
				Kind:       strings.TrimSpace(op.Target),
				Subject:    strings.TrimSpace(op.Args["subject"]),
				Attributes: operationPayload(op.Args, "duration_seconds"),
				Source:     SourceCascade,
				OccurredAt: now.UTC(),
			},
		})
		return nil
	default:
		return fmt.Errorf("unsupported operation kind %q", op.Kind)
	}
}

func operationPayload(args map[string]string, keys ...string) map[string]string {
	out := map[string]string{}
	for _, key := range keys {
		if value := strings.TrimSpace(args[key]); value != "" {
			out[key] = value
		}
	}
	return out
}

func splitTargetList(target string) []string {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	parts := strings.Split(target, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func stringDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
