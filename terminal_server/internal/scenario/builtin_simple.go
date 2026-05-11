package scenario

import (
	"context"
	"errors"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// AlertScenario broadcasts a critical alert across targeted devices.
type AlertScenario struct{}

// Name returns the stable scenario identifier.
func (AlertScenario) Name() string { return "red_alert" }

// Match checks whether the trigger intent activates this scenario.
func (AlertScenario) Match(trigger Trigger) bool {
	return intentMatches(trigger.Intent, "red alert", "red_alert")
}

// Start broadcasts an alert notification.
func (AlertScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "RED ALERT")
}

// Stop ends the scenario and currently has no side effects.
func (AlertScenario) Stop() error { return nil }

// PhotoFrameScenario marks a low-priority ambient mode.
type PhotoFrameScenario struct{}

// Name returns the stable scenario identifier.
func (PhotoFrameScenario) Name() string { return "photo_frame" }

// Match checks whether the trigger intent activates this scenario.
func (PhotoFrameScenario) Match(trigger Trigger) bool {
	return intentMatches(trigger.Intent, "photo frame", "photo_frame")
}

// Start broadcasts a mode activation notification.
func (PhotoFrameScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Devices == nil || env.Broadcast == nil {
		return nil
	}
	return env.Broadcast.Notify(ctx, env.Devices.ListDeviceIDs(), "Photo frame active")
}

// Stop ends the scenario and currently has no side effects.
func (PhotoFrameScenario) Stop() error { return nil }

// TerminalScenario activates interactive terminal mode on the requesting device.
type TerminalScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *TerminalScenario) Name() string { return "terminal" }

// Match records trigger metadata when terminal mode is requested.
func (s *TerminalScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "terminal", "open terminal") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start announces terminal activation for the requesting device.
func (s *TerminalScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil || env.Broadcast == nil {
		return nil
	}
	deviceIDs := []string{}
	if s.trigger.SourceID != "" {
		deviceIDs = []string{s.trigger.SourceID}
	}
	return env.Broadcast.Notify(ctx, deviceIDs, "Terminal active")
}

// Stop ends terminal mode and currently has no side effects.
func (s *TerminalScenario) Stop() error { return nil }

// MultiWindowScenario routes all peer cameras to source display.
type MultiWindowScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *MultiWindowScenario) Name() string { return "multi_window" }

// Match records trigger metadata when multi-window mode is requested.
func (s *MultiWindowScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "multi window", "multi_window", "show all cameras", "all cameras") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes peer video feeds into source device and confirms activation.
func (s *MultiWindowScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	if env.IO != nil && env.Devices != nil && strings.TrimSpace(s.trigger.SourceID) != "" {
		source := strings.TrimSpace(s.trigger.SourceID)
		peers := peerTargetDeviceIDs(env, source, s.trigger.Arguments)
		for _, peer := range peers {
			if err := env.IO.Connect(peer, source, "video"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
				return err
			}
		}
		if err := clearMultiWindowAudioRoutes(env, source); err != nil {
			return err
		}
		focusedPeer := strings.TrimSpace(s.trigger.Arguments["audio_focus_device_id"])
		if focusedPeer != "" {
			focusMatched := false
			for _, peer := range peers {
				if peer != focusedPeer {
					continue
				}
				if err := env.IO.Connect(peer, source, "audio"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
					return err
				}
				focusMatched = true
				break
			}
			if !focusMatched {
				focusedPeer = ""
			}
		}
		if focusedPeer == "" {
			for _, peer := range peers {
				if err := env.IO.Connect(peer, source, "audio_mix"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
					return err
				}
			}
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Multi-window active")
}

// Stop ends multi-window mode and currently has no side effects.
func (s *MultiWindowScenario) Stop() error { return nil }

func clearMultiWindowAudioRoutes(env *Environment, targetID string) error {
	if env == nil || env.IO == nil {
		return nil
	}
	routeIO, ok := env.IO.(interface {
		RoutesForDevice(string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return nil
	}
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return nil
	}
	for _, route := range routeIO.RoutesForDevice(targetID) {
		if route.TargetID != targetID {
			continue
		}
		if route.StreamKind != "audio_mix" && route.StreamKind != "audio" {
			continue
		}
		if err := routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind); err != nil && !errors.Is(err, iorouter.ErrRouteNotFound) {
			return err
		}
	}
	return nil
}

// TerminalVerificationScenario updates world-model verification state.
type TerminalVerificationScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *TerminalVerificationScenario) Name() string { return "terminal_verification" }

// Match records trigger metadata when terminal verification is requested.
func (s *TerminalVerificationScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "terminal_verification", "verify terminal") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start verifies one terminal placement method in the world model.
func (s *TerminalVerificationScenario) Start(ctx context.Context, env *Environment) error {
	deviceID := strings.TrimSpace(s.trigger.Arguments["device_id"])
	method := strings.TrimSpace(s.trigger.Arguments["method"])
	if method == "" {
		method = "manual"
	}
	if env == nil || env.World == nil || deviceID == "" {
		return notifySource(ctx, env, s.trigger.SourceID, "Terminal verification could not run.")
	}
	if err := env.World.VerifyDevice(ctx, deviceID, method); err != nil {
		return notifySource(ctx, env, s.trigger.SourceID, "Terminal verification failed for "+deviceID+".")
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Terminal "+deviceID+" verified with method "+method+".")
}

// Stop ends this one-shot scenario.
func (s *TerminalVerificationScenario) Stop() error { return nil }
