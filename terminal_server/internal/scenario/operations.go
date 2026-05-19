package scenario

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// Operation kind constants identify host capabilities the executor can commit.
const (
	OperationUISet           = "ui.set"
	OperationUIPatch         = "ui.patch"
	OperationUIClear         = "ui.clear"
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
	Node   *ui.Descriptor
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
		if err := validateOperation(env, op); err != nil {
			if strings.HasPrefix(err.Error(), "has unsupported kind ") {
				return fmt.Errorf("op %d %w", i, err)
			}
			return fmt.Errorf("op %d %s %w", i, op.Kind, err)
		}
	}
	return nil
}

func validateOperation(env *Environment, op Operation) error {
	switch op.Kind {
	case OperationSchedulerAfter:
		return validateSchedulerAfterOperation(env, op)
	case OperationSchedulerCancel:
		return validateSchedulerCancelOperation(env, op)
	case OperationBroadcastNotify:
		return validateBroadcastNotifyOperation(env, op)
	case OperationAITTS:
		return validateTTSOperation(env, op)
	case OperationBusEmit:
		return validateBusEmitOperation(env, op)
	case OperationUISet:
		return validateUISetOperation(env, op)
	case OperationUIPatch:
		return validateUIPatchOperation(env, op)
	case OperationUIClear:
		return validateUIClearOperation(env, op)
	case OperationUITransition, OperationFlowApply, OperationFlowStop:
		return errors.New("is not executable yet")
	default:
		return fmt.Errorf("has unsupported kind %q", op.Kind)
	}
}

func validateSchedulerAfterOperation(env *Environment, op Operation) error {
	if env == nil || env.Scheduler == nil {
		return errors.New("requires scheduler")
	}
	if strings.TrimSpace(op.Target) == "" {
		return errors.New("requires target key")
	}
	if _, err := strconv.ParseInt(strings.TrimSpace(op.Args["unix_ms"]), 10, 64); err != nil {
		return fmt.Errorf("requires unix_ms: %w", err)
	}
	return nil
}

func validateSchedulerCancelOperation(env *Environment, op Operation) error {
	if env == nil || env.Scheduler == nil {
		return errors.New("requires scheduler")
	}
	if strings.TrimSpace(op.Target) == "" {
		return errors.New("requires target key")
	}
	return nil
}

func validateBroadcastNotifyOperation(env *Environment, op Operation) error {
	if env == nil || env.Broadcast == nil {
		return errors.New("requires broadcaster")
	}
	if strings.TrimSpace(op.Args["message"]) == "" {
		return errors.New("requires message")
	}
	return nil
}

func validateTTSOperation(env *Environment, op Operation) error {
	if env == nil || env.TTS == nil {
		return errors.New("requires tts")
	}
	if strings.TrimSpace(op.Args["text"]) == "" {
		return errors.New("requires text")
	}
	return nil
}

func validateBusEmitOperation(env *Environment, op Operation) error {
	if env == nil || env.TriggerBus == nil {
		return errors.New("requires trigger bus")
	}
	if strings.TrimSpace(op.Target) == "" {
		return errors.New("requires event kind target")
	}
	return nil
}

func validateUISetOperation(env *Environment, op Operation) error {
	if err := validateUIOperationTarget(env, op); err != nil {
		return err
	}
	if op.Node == nil {
		return errors.New("requires node")
	}
	return nil
}

func validateUIPatchOperation(env *Environment, op Operation) error {
	if err := validateUIOperationTarget(env, op); err != nil {
		return err
	}
	if strings.TrimSpace(op.Args["component_id"]) == "" {
		return errors.New("requires component_id")
	}
	if op.Node == nil {
		return errors.New("requires node")
	}
	return nil
}

func validateUIClearOperation(env *Environment, op Operation) error {
	return validateUIOperationTarget(env, op)
}

func validateUIOperationTarget(env *Environment, op Operation) error {
	if env == nil || env.UI == nil {
		return errors.New("requires ui host")
	}
	if strings.TrimSpace(op.Target) == "" {
		return errors.New("requires target device")
	}
	return nil
}

func executeOperation(ctx context.Context, env *Environment, op Operation, now time.Time) error {
	switch op.Kind {
	case OperationUISet:
		return env.UI.Set(ctx, strings.TrimSpace(op.Target), *op.Node)
	case OperationUIPatch:
		return env.UI.Patch(ctx, strings.TrimSpace(op.Target), strings.TrimSpace(op.Args["component_id"]), *op.Node)
	case OperationUIClear:
		return env.UI.Clear(ctx, strings.TrimSpace(op.Target), strings.TrimSpace(op.Args["root"]))
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
				Payload:  operationPayload(op.Args, "duration_seconds", "target_device_id", "expiry_unix_ms"),
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
