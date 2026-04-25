package scenario

import "github.com/curtcox/terminals/terminal_server/internal/appruntime"

// ResultFromAppRuntime adapts TAL runtime results to scenario operations.
func ResultFromAppRuntime(result appruntime.Result) ScenarioResult {
	ops := make([]Operation, 0, len(result.Ops))
	for _, op := range result.Ops {
		ops = append(ops, Operation{
			Kind:   op.Kind,
			Target: op.Target,
			Args:   copyStringMap(op.Args),
		})
	}

	emits := make([]Trigger, 0, len(result.Emit))
	for _, trigger := range result.Emit {
		emits = append(emits, Trigger{
			Kind: TriggerEvent,
			EventV2: &EventRecord{
				Kind:       trigger.Kind,
				Subject:    trigger.Subject,
				Attributes: copyStringMap(trigger.Attributes),
				Source:     SourceCascade,
				OccurredAt: trigger.OccurredAt,
			},
		})
	}

	return ScenarioResult{
		State: result.State,
		Ops:   ops,
		Emit:  emits,
		Done:  result.Done,
	}
}
