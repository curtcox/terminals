package scenario

import "context"

// AlertScenario broadcasts a critical alert across targeted devices.
type AlertScenario struct{}

func (AlertScenario) Name() string { return "red_alert" }

func (AlertScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "red alert"
}

func (AlertScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "RED ALERT")
}

func (AlertScenario) Stop() error { return nil }

// PhotoFrameScenario marks a low-priority ambient mode.
type PhotoFrameScenario struct{}

func (PhotoFrameScenario) Name() string { return "photo_frame" }

func (PhotoFrameScenario) Match(trigger Trigger) bool {
	return trigger.Intent == "photo frame"
}

func (PhotoFrameScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "Photo frame active")
}

func (PhotoFrameScenario) Stop() error { return nil }
