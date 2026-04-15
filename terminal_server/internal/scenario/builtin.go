// Package scenario contains server-side scenario matching and runtime flows.
package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// AlertScenario broadcasts a critical alert across targeted devices.
type AlertScenario struct{}

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

// TimerReminderScenario schedules a timer and confirms it via broadcast.
type TimerReminderScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *TimerReminderScenario) Name() string { return "timer_reminder" }

// Match records trigger arguments when this scenario should run.
func (s *TimerReminderScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "set timer", "timer_reminder") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules the timer and confirms to the origin device.
func (s *TimerReminderScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}

	fireUnixMS := time.Now().Add(10 * time.Minute).UnixMilli()
	if raw := s.trigger.Arguments["fire_unix_ms"]; raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			fireUnixMS = parsed
		}
	}

	timerKey := "timer:" + s.trigger.SourceID + ":" + strconv.FormatInt(fireUnixMS, 10)
	if env.Scheduler != nil {
		if err := env.Scheduler.Schedule(ctx, timerKey, fireUnixMS); err != nil {
			return err
		}
	}
	if env.Broadcast != nil {
		deviceIDs := []string{}
		if s.trigger.SourceID != "" {
			deviceIDs = []string{s.trigger.SourceID}
		}
		return env.Broadcast.Notify(ctx, deviceIDs, "Timer set")
	}
	return nil
}

// Stop ends the scenario and currently has no side effects.
func (s *TimerReminderScenario) Stop() error { return nil }

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

// BluetoothPassthroughScenario dispatches server-directed BLE passthrough commands.
type BluetoothPassthroughScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *BluetoothPassthroughScenario) Name() string { return "bluetooth_passthrough" }

// Match records trigger metadata when Bluetooth passthrough is requested.
func (s *BluetoothPassthroughScenario) Match(trigger Trigger) bool {
	if !intentMatches(
		trigger.Intent,
		"bluetooth passthrough",
		"bluetooth_passthrough",
		"bluetooth scan",
		"bluetooth_scan",
		"ble scan",
		"bluetooth connect",
		"bluetooth_connect",
	) {
		return false
	}
	s.trigger = trigger
	return true
}

// Start dispatches a Bluetooth passthrough command through the server bridge.
func (s *BluetoothPassthroughScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	action := strings.TrimSpace(s.trigger.Arguments["action"])
	if action == "" {
		action = "scan"
		if strings.Contains(strings.ToLower(strings.TrimSpace(s.trigger.Intent)), "connect") {
			action = "connect"
		}
	}
	targetID := strings.TrimSpace(s.trigger.Arguments["target_id"])
	if targetID == "" {
		targetID = strings.TrimSpace(s.trigger.Arguments["target"])
	}

	if env.Passthrough != nil {
		if err := env.Passthrough.DispatchBluetoothCommand(ctx, BluetoothCommand{
			DeviceID:   strings.TrimSpace(s.trigger.SourceID),
			Action:     action,
			TargetID:   targetID,
			Parameters: passthroughParameters(s.trigger.Arguments, "action", "target_id", "target"),
		}); err != nil {
			return err
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Bluetooth passthrough requested: "+action)
}

// Stop ends passthrough mode and currently has no side effects.
func (s *BluetoothPassthroughScenario) Stop() error { return nil }

// HandleBluetoothEvent reports passthrough updates to the source device.
func (s *BluetoothPassthroughScenario) HandleBluetoothEvent(ctx context.Context, env *Environment, event BluetoothEvent) error {
	message := "Bluetooth event"
	if evt := strings.TrimSpace(event.Event); evt != "" {
		message = "Bluetooth event: " + evt
	}
	return notifySource(ctx, env, s.trigger.SourceID, message)
}

// USBPassthroughScenario dispatches server-directed USB passthrough commands.
type USBPassthroughScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *USBPassthroughScenario) Name() string { return "usb_passthrough" }

// Match records trigger metadata when USB passthrough is requested.
func (s *USBPassthroughScenario) Match(trigger Trigger) bool {
	if !intentMatches(
		trigger.Intent,
		"usb passthrough",
		"usb_passthrough",
		"usb enumerate",
		"usb_enumerate",
		"usb claim",
		"usb_claim",
	) {
		return false
	}
	s.trigger = trigger
	return true
}

// Start dispatches a USB passthrough command through the server bridge.
func (s *USBPassthroughScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	action := strings.TrimSpace(s.trigger.Arguments["action"])
	if action == "" {
		action = "enumerate"
		if strings.Contains(strings.ToLower(strings.TrimSpace(s.trigger.Intent)), "claim") {
			action = "claim"
		}
	}
	vendorID := strings.TrimSpace(s.trigger.Arguments["vendor_id"])
	productID := strings.TrimSpace(s.trigger.Arguments["product_id"])

	if env.Passthrough != nil {
		if err := env.Passthrough.DispatchUSBCommand(ctx, USBCommand{
			DeviceID:   strings.TrimSpace(s.trigger.SourceID),
			Action:     action,
			VendorID:   vendorID,
			ProductID:  productID,
			Parameters: passthroughParameters(s.trigger.Arguments, "action", "vendor_id", "product_id"),
		}); err != nil {
			return err
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, "USB passthrough requested: "+action)
}

// Stop ends passthrough mode and currently has no side effects.
func (s *USBPassthroughScenario) Stop() error { return nil }

// HandleUSBEvent reports passthrough updates to the source device.
func (s *USBPassthroughScenario) HandleUSBEvent(ctx context.Context, env *Environment, event USBEvent) error {
	message := "USB event"
	if evt := strings.TrimSpace(event.Event); evt != "" {
		message = "USB event: " + evt
	}
	return notifySource(ctx, env, s.trigger.SourceID, message)
}

func passthroughParameters(args map[string]string, skip ...string) map[string]string {
	if len(args) == 0 {
		return map[string]string{}
	}
	skipSet := map[string]struct{}{}
	for _, key := range skip {
		skipSet[key] = struct{}{}
	}
	out := map[string]string{}
	for key, value := range args {
		if _, skipKey := skipSet[key]; skipKey {
			continue
		}
		out[key] = value
	}
	return out
}

// IntercomScenario connects source device audio to all peer devices.
type IntercomScenario struct {
	trigger Trigger

	mu          sync.Mutex
	env         *Environment
	ownedRoutes []ioRoute
}

// Name returns the stable scenario identifier.
func (s *IntercomScenario) Name() string { return "intercom" }

// Match records trigger metadata when intercom mode is requested.
func (s *IntercomScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "intercom", "start intercom") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source microphone audio to other devices and announces activation.
func (s *IntercomScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	targets := peerTargetDeviceIDs(env, s.trigger.SourceID, s.trigger.Arguments)
	ownedRoutes, err := connectBidirectionalSourceTargetsOwned(ctx, env, s.trigger.SourceID, targets, "audio")
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.env = env
	s.ownedRoutes = ownedRoutes
	s.mu.Unlock()
	if err := notifySource(ctx, env, s.trigger.SourceID, "Intercom active"); err != nil {
		return err
	}
	return nil
}

// Stop ends intercom mode and releases any owned routes.
func (s *IntercomScenario) Stop() error {
	s.mu.Lock()
	env := s.env
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.env = nil
	s.ownedRoutes = nil
	s.mu.Unlock()
	return disconnectOwnedRoutes(env, routes)
}

// Suspend releases owned routes while preempted.
func (s *IntercomScenario) Suspend() error {
	s.mu.Lock()
	env := s.env
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.mu.Unlock()
	return disconnectOwnedRoutes(env, routes)
}

// Resume reacquires owned routes after preemption.
func (s *IntercomScenario) Resume(_ context.Context, env *Environment) error {
	s.mu.Lock()
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.env = env
	s.mu.Unlock()
	return reconnectOwnedRoutes(env, routes)
}

// InternalVideoCallScenario connects source and target with bidirectional audio/video.
type InternalVideoCallScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *InternalVideoCallScenario) Name() string { return "internal_video_call" }

// Match records trigger metadata when an internal video call is requested.
func (s *InternalVideoCallScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "internal video call", "internal_video_call", "video call", "start video call") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes bidirectional audio/video between source and target devices.
func (s *InternalVideoCallScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}

	sourceID := strings.TrimSpace(s.trigger.SourceID)
	targetID := strings.TrimSpace(s.trigger.Arguments["target_device_id"])
	if targetID == "" {
		peers := nonSourceDeviceIDs(env, sourceID)
		if len(peers) > 0 {
			targetID = peers[0]
		}
	}
	if sourceID == "" || targetID == "" {
		return notifySource(ctx, env, sourceID, "Video call target unavailable")
	}

	if env.IO != nil {
		for _, streamKind := range []string{"audio", "video"} {
			if err := env.IO.Connect(sourceID, targetID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
				return err
			}
			if err := env.IO.Connect(targetID, sourceID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
				return err
			}
		}
	}

	if err := notifySource(ctx, env, sourceID, "Video call active: "+targetID); err != nil {
		return err
	}
	if env.Broadcast != nil {
		if err := env.Broadcast.Notify(ctx, []string{targetID}, "Incoming video call: "+sourceID); err != nil {
			return err
		}
	}
	return nil
}

// Stop ends internal video call mode and currently has no side effects.
func (s *InternalVideoCallScenario) Stop() error { return nil }

// PhoneCallScenario starts an external call through telephony bridge.
type PhoneCallScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *PhoneCallScenario) Name() string { return "phone_call" }

// Match records trigger metadata when a phone call is requested.
func (s *PhoneCallScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "phone call", "phone_call", "call") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start places a call and notifies the source device.
func (s *PhoneCallScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "unknown"
	}
	if env.Telephony != nil {
		if err := env.Telephony.Call(ctx, target); err != nil {
			return err
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Calling "+target)
}

// Stop ends phone call mode and currently has no side effects.
func (s *PhoneCallScenario) Stop() error { return nil }

// VoiceAssistantScenario proxies a prompt to AI backend and reports response.
type VoiceAssistantScenario struct {
	trigger Trigger

	mu           sync.Mutex
	env          *Environment
	activationID string
	planHandle   iorouter.PlanHandle
}

func (s *VoiceAssistantScenario) ensureActivationID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activationID != "" {
		return s.activationID
	}
	source := strings.TrimSpace(s.trigger.SourceID)
	if source == "" {
		source = "unknown"
	}
	s.activationID = "voice_assistant:" + source
	return s.activationID
}

// Name returns the stable scenario identifier.
func (s *VoiceAssistantScenario) Name() string { return "voice_assistant" }

// Match records trigger metadata when assistant mode is requested.
func (s *VoiceAssistantScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "voice assistant", "voice_assistant", "assistant") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start queries the configured LLM (preferred) or legacy AIBackend and
// notifies the source device with the response.
func (s *VoiceAssistantScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	s.mu.Lock()
	s.env = env
	s.mu.Unlock()

	activationID := s.ensureActivationID()
	if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
		if claims := routeIO.Claims(); claims != nil {
			_, err := claims.Request(ctx, []iorouter.Claim{
				{
					ActivationID: activationID,
					DeviceID:     strings.TrimSpace(s.trigger.SourceID),
					Resource:     "mic.analyze",
					Mode:         iorouter.ClaimShared,
					Priority:     int(PriorityNormal),
				},
				{
					ActivationID: activationID,
					DeviceID:     strings.TrimSpace(s.trigger.SourceID),
					Resource:     "speaker.main",
					Mode:         iorouter.ClaimExclusive,
					Priority:     int(PriorityNormal),
				},
				{
					ActivationID: activationID,
					DeviceID:     strings.TrimSpace(s.trigger.SourceID),
					Resource:     "screen.overlay",
					Mode:         iorouter.ClaimShared,
					Priority:     int(PriorityNormal),
				},
			})
			if err != nil && !errors.Is(err, iorouter.ErrClaimConflict) {
				return err
			}
		}
	}
	if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok {
		planner := routeIO.MediaPlanner()
		if planner != nil {
			handle, err := planner.Apply(ctx, iorouter.MediaPlan{
				Nodes: []iorouter.MediaNode{
					{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": strings.TrimSpace(s.trigger.SourceID)}},
					{ID: "fork", Kind: iorouter.NodeFork},
					{ID: "stt", Kind: iorouter.NodeSinkSTT, Args: map[string]string{"device_id": "server"}},
					{ID: "rec", Kind: iorouter.NodeRecorder, Args: map[string]string{"device_id": "server"}},
					{ID: "tts", Kind: iorouter.NodeSourceTTS, Args: map[string]string{"device_id": "server"}},
					{ID: "speaker", Kind: iorouter.NodeSinkSpeaker, Args: map[string]string{"device_id": strings.TrimSpace(s.trigger.SourceID)}},
				},
				Edges: []iorouter.MediaEdge{
					{From: "mic", To: "fork"},
					{From: "fork", To: "stt"},
					{From: "fork", To: "rec"},
					{From: "tts", To: "speaker"},
				},
			})
			if err != nil {
				return err
			}
			s.mu.Lock()
			s.planHandle = handle
			s.mu.Unlock()
		}
	}

	query := strings.TrimSpace(s.trigger.Arguments["query"])
	if query == "" {
		query = "hello"
	}
	response := "Voice assistant active"
	switch {
	case env.LLM != nil:
		out, err := env.LLM.Query(ctx, []LLMMessage{{Role: "user", Content: query}}, LLMOptions{})
		if err != nil {
			return err
		}
		if out != nil && strings.TrimSpace(out.Text) != "" {
			response = out.Text
		}
	case env.AI != nil:
		out, err := env.AI.Query(ctx, query)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			response = out
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, response)
}

// Stop ends assistant mode and releases claims/media resources.
func (s *VoiceAssistantScenario) Stop() error {
	s.mu.Lock()
	env := s.env
	activationID := s.activationID
	planHandle := s.planHandle
	s.planHandle = ""
	s.env = nil
	s.mu.Unlock()

	if env == nil {
		return nil
	}
	if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok {
		if planner := routeIO.MediaPlanner(); planner != nil && planHandle != "" {
			if err := planner.Tear(context.Background(), planHandle); err != nil {
				return err
			}
		}
	}
	if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
		if claims := routeIO.Claims(); claims != nil && activationID != "" {
			if err := claims.Release(context.Background(), activationID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Suspend releases assistant-owned media resources while preempted.
func (s *VoiceAssistantScenario) Suspend() error {
	return s.Stop()
}

// Resume restores assistant media resources and claims after preemption.
func (s *VoiceAssistantScenario) Resume(ctx context.Context, env *Environment) error {
	return s.Start(ctx, env)
}

// AudioMonitorScenario stores monitor intent and confirms arming.
type AudioMonitorScenario struct {
	trigger Trigger

	mu           sync.Mutex
	stopFn       func()
	env          *Environment
	activationID string
	planHandle   iorouter.PlanHandle
}

// Name returns the stable scenario identifier.
func (s *AudioMonitorScenario) Name() string { return "audio_monitor" }

// Match records trigger metadata when audio monitor is requested.
func (s *AudioMonitorScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "audio monitor", "audio_monitor", "monitor audio") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start persists the monitor target and notifies the source device.
func (s *AudioMonitorScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	s.mu.Lock()
	s.env = env
	if s.activationID == "" {
		sourceID := strings.TrimSpace(s.trigger.SourceID)
		if sourceID == "" {
			sourceID = "unknown"
		}
		s.activationID = "audio_monitor:" + sourceID
	}
	s.mu.Unlock()
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "sound"
	}
	if env.Storage != nil && s.trigger.SourceID != "" {
		if err := env.Storage.Put(ctx, "audio_monitor:"+s.trigger.SourceID, target); err != nil {
			return err
		}
	}
	if err := notifySource(ctx, env, s.trigger.SourceID, "Audio monitor armed: "+target); err != nil {
		return err
	}
	return s.startMonitorLoop(ctx, env, target, strings.TrimSpace(s.trigger.SourceID))
}

// openAudioMonitorSource returns a live audio source for the monitored
// device, falling back to an immediate-EOF silence source when the runtime
// is not configured with a DeviceAudioSubscriber or no source device is set.
func openAudioMonitorSource(ctx context.Context, env *Environment, sourceID string) (AudioSource, func(), error) {
	if env != nil && env.DeviceAudio != nil && sourceID != "" {
		sub, err := env.DeviceAudio.SubscribeAudio(ctx, sourceID)
		if err != nil {
			return nil, nil, err
		}
		return sub, func() { _ = sub.Close() }, nil
	}
	return audioMonitorSilenceSource{}, func() {}, nil
}

// Stop ends monitor mode by canceling the active classifier goroutine and
// releasing any live audio subscription opened in Start. Safe to call when
// the scenario was never started or has already stopped.
func (s *AudioMonitorScenario) Stop() error {
	s.clearMonitorLoop()
	s.mu.Lock()
	env := s.env
	activationID := s.activationID
	planHandle := s.planHandle
	s.planHandle = ""
	s.env = nil
	s.mu.Unlock()
	if env != nil {
		if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok {
			if planner := routeIO.MediaPlanner(); planner != nil && planHandle != "" {
				_ = planner.Tear(context.Background(), planHandle)
			}
		}
		if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
			if claims := routeIO.Claims(); claims != nil && activationID != "" {
				_ = claims.Release(context.Background(), activationID)
			}
		}
	}
	return nil
}

// Suspend releases live monitor resources while the scenario is preempted.
func (s *AudioMonitorScenario) Suspend() error {
	s.clearMonitorLoop()
	return nil
}

// Resume reacquires live monitor resources after preemption using the
// original trigger target and source device.
func (s *AudioMonitorScenario) Resume(ctx context.Context, env *Environment) error {
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "sound"
	}
	return s.startMonitorLoop(ctx, env, target, strings.TrimSpace(s.trigger.SourceID))
}

func (s *AudioMonitorScenario) clearMonitorLoop() {
	s.mu.Lock()
	stop := s.stopFn
	s.stopFn = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (s *AudioMonitorScenario) startMonitorLoop(ctx context.Context, env *Environment, target, sourceID string) error {
	if env == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.clearMonitorLoop()

	activationID := ""
	s.mu.Lock()
	activationID = s.activationID
	s.mu.Unlock()
	if activationID == "" {
		activationID = "audio_monitor:" + sourceID
	}

	analyzerPlanActive := false
	if routeIO, ok := env.IO.(interface {
		MediaPlanner() *iorouter.MediaPlanner
		Claims() *iorouter.ClaimManager
	}); ok {
		if claims := routeIO.Claims(); claims != nil {
			_, err := claims.Request(ctx, []iorouter.Claim{{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     "mic.analyze",
				Mode:         iorouter.ClaimShared,
				Priority:     int(PriorityNormal),
			}})
			if err != nil && !errors.Is(err, iorouter.ErrClaimConflict) {
				return err
			}
		}
		if planner := routeIO.MediaPlanner(); planner != nil {
			if planner.AnalyzerEnabled() {
				handle, err := planner.Apply(ctx, iorouter.MediaPlan{
					Nodes: []iorouter.MediaNode{
						{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID}},
						{ID: "analyzer", Kind: iorouter.NodeAnalyzer, Args: map[string]string{"name": "sound"}},
					},
					Edges: []iorouter.MediaEdge{
						{From: "mic", To: "analyzer"},
					},
				})
				if err == nil {
					s.mu.Lock()
					s.planHandle = handle
					s.mu.Unlock()
					analyzerPlanActive = true
				}
			}
		}
	}

	// Prefer event-bus subscriptions emitted by analyzer nodes.
	if analyzerPlanActive && env.TriggerBus != nil {
		sub, cancel := env.TriggerBus.Subscribe(16)
		audioCtx, cancelAudio := context.WithCancel(ctx)
		stopFn := func() {
			cancelAudio()
			cancel()
		}
		s.mu.Lock()
		s.stopFn = stopFn
		s.mu.Unlock()

		go func() {
			defer stopFn()
			for {
				select {
				case <-audioCtx.Done():
					return
				case trigger, ok := <-sub:
					if !ok || trigger.EventV2 == nil {
						continue
					}
					if strings.TrimSpace(trigger.EventV2.Kind) != "sound.detected" {
						continue
					}
					if src := strings.TrimSpace(trigger.SourceID); src != "" && src != sourceID {
						continue
					}
					label := strings.TrimSpace(trigger.EventV2.Subject)
					if label == "" {
						label = strings.TrimSpace(trigger.EventV2.Attributes["label"])
					}
					if !audioMonitorEventMatchesTarget(target, label) {
						continue
					}
					if label == "" {
						label = target
					}
					_ = notifySource(ctx, env, sourceID, "Audio monitor detected: "+label)
					return
				}
			}
		}()
		return nil
	}

	// Fallback path for tests/contexts without an event bus analyzer runner.
	if env.Sound == nil {
		return nil
	}
	audioCtx, cancelAudio := context.WithCancel(ctx)
	audio, closeAudio, err := openAudioMonitorSource(audioCtx, env, sourceID)
	if err != nil {
		cancelAudio()
		return err
	}
	stream, err := env.Sound.Classify(audioCtx, audio)
	if err != nil {
		closeAudio()
		cancelAudio()
		return err
	}
	var stopOnce sync.Once
	stopFn := func() {
		stopOnce.Do(func() {
			cancelAudio()
			closeAudio()
		})
	}
	s.mu.Lock()
	s.stopFn = stopFn
	s.mu.Unlock()
	go func() {
		defer stopFn()
		for event := range stream {
			if !audioMonitorEventMatchesTarget(target, event.Label) {
				continue
			}
			messageLabel := strings.TrimSpace(event.Label)
			if messageLabel == "" {
				messageLabel = target
			}
			_ = notifySource(ctx, env, sourceID, "Audio monitor detected: "+messageLabel)
			return
		}
	}()
	return nil
}

type audioMonitorSilenceSource struct{}

func (audioMonitorSilenceSource) Read([]byte) (int, error) {
	return 0, io.EOF
}

func audioMonitorEventMatchesTarget(target, label string) bool {
	normalizedTarget := strings.TrimSpace(strings.ToLower(target))
	normalizedLabel := strings.TrimSpace(strings.ToLower(label))
	if normalizedLabel == "" {
		return false
	}
	if normalizedTarget == "" || normalizedTarget == "sound" {
		return true
	}
	return strings.Contains(normalizedLabel, normalizedTarget) || strings.Contains(normalizedTarget, normalizedLabel)
}

// ScheduleMonitorScenario schedules a check and confirms activation.
type ScheduleMonitorScenario struct {
	trigger Trigger

	mu              sync.Mutex
	lastAlertUnixMS int64
}

// Name returns the stable scenario identifier.
func (s *ScheduleMonitorScenario) Name() string { return "schedule_monitor" }

// Match records trigger metadata when schedule monitoring is requested.
func (s *ScheduleMonitorScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "schedule monitor", "schedule_monitor", "watch schedule") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start schedules a follow-up check and notifies the source device.
func (s *ScheduleMonitorScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	checkUnixMS := time.Now().Add(5 * time.Minute).UnixMilli()
	if raw := strings.TrimSpace(s.trigger.Arguments["check_unix_ms"]); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			checkUnixMS = parsed
		}
	}
	if env.Scheduler != nil && s.trigger.SourceID != "" {
		if err := env.Scheduler.Schedule(ctx, "schedule_monitor:"+s.trigger.SourceID, checkUnixMS); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.lastAlertUnixMS = 0
	s.mu.Unlock()
	return notifySource(ctx, env, s.trigger.SourceID, "Schedule monitor active")
}

// Stop ends schedule monitor mode and currently has no side effects.
func (s *ScheduleMonitorScenario) Stop() error { return nil }

// HandleSensor consumes live sensor telemetry while schedule monitoring is
// active and raises an activity alert when movement exceeds threshold.
func (s *ScheduleMonitorScenario) HandleSensor(ctx context.Context, env *Environment, reading SensorReading) error {
	if env == nil || env.Broadcast == nil {
		return nil
	}
	if strings.TrimSpace(reading.DeviceID) == "" {
		return nil
	}
	monitorDeviceID := strings.TrimSpace(s.trigger.SourceID)
	if monitorDeviceID != "" && reading.DeviceID != monitorDeviceID {
		return nil
	}

	magnitude, ok := sensorMotionMagnitude(reading.Values)
	if !ok {
		return nil
	}
	threshold := parseFloatOrDefault(s.trigger.Arguments["motion_threshold"], 1.20)
	if magnitude < threshold {
		return nil
	}

	eventUnixMS := reading.UnixMS
	if eventUnixMS <= 0 {
		eventUnixMS = time.Now().UnixMilli()
	}
	cooldownMS := int64(parseFloatOrDefault(s.trigger.Arguments["cooldown_ms"], 60_000))
	if cooldownMS < 0 {
		cooldownMS = 0
	}

	s.mu.Lock()
	if s.lastAlertUnixMS > 0 && eventUnixMS-s.lastAlertUnixMS < cooldownMS {
		s.mu.Unlock()
		return nil
	}
	s.lastAlertUnixMS = eventUnixMS
	s.mu.Unlock()

	return notifySource(ctx, env, reading.DeviceID, fmt.Sprintf("Schedule monitor activity detected: magnitude=%.2f", magnitude))
}

func parseFloatOrDefault(raw string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func sensorMotionMagnitude(values map[string]float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	if scalar, ok := values["motion.magnitude"]; ok {
		return math.Abs(scalar), true
	}
	x, hasX := values["accelerometer.x"]
	y, hasY := values["accelerometer.y"]
	z, hasZ := values["accelerometer.z"]
	if hasX || hasY || hasZ {
		return math.Sqrt((x * x) + (y * y) + (z * z)), true
	}
	return 0, false
}

// PASystemScenario fans out source audio to peers with PA semantics.
type PASystemScenario struct {
	trigger Trigger

	recipe scenarioRecipeState
}

// Name returns the stable scenario identifier.
func (s *PASystemScenario) Name() string { return "pa_system" }

// Match records trigger metadata when PA mode is requested.
func (s *PASystemScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "pa system", "pa_system", "pa mode") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source audio to peers and announces PA mode.
func (s *PASystemScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	activationID := "pa_system:" + sourceID
	return s.recipe.start(ctx, env, ScenarioRecipe{
		ActivationID: activationID,
		Resolve: func(_ context.Context, env *Environment) []string {
			return peerTargetDeviceIDs(env, sourceID, s.trigger.Arguments)
		},
		Claims: func(targets []string) []iorouter.Claim {
			out := make([]iorouter.Claim, 0, len(targets)+1)
			out = append(out, iorouter.Claim{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     "mic.capture",
				Mode:         iorouter.ClaimExclusive,
				Priority:     int(PriorityHigh),
			})
			for _, targetID := range targets {
				out = append(out, iorouter.Claim{
					ActivationID: activationID,
					DeviceID:     targetID,
					Resource:     "speaker.main",
					Mode:         iorouter.ClaimExclusive,
					Priority:     int(PriorityHigh),
				})
			}
			return out
		},
		MediaPlan: func(targets []string) *iorouter.MediaPlan {
			nodes := []iorouter.MediaNode{
				{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID}},
				{ID: "fork", Kind: iorouter.NodeFork},
			}
			edges := []iorouter.MediaEdge{{From: "mic", To: "fork"}}
			for idx, targetID := range targets {
				nodeID := fmt.Sprintf("speaker_%d", idx)
				nodes = append(nodes, iorouter.MediaNode{
					ID:   nodeID,
					Kind: iorouter.NodeSinkSpeaker,
					Args: map[string]string{"device_id": targetID, "stream_kind": "pa_audio"},
				})
				edges = append(edges, iorouter.MediaEdge{From: "fork", To: nodeID})
			}
			return &iorouter.MediaPlan{Nodes: nodes, Edges: edges}
		},
		OnStart: func(ctx context.Context, env *Environment, _ []string) error {
			if err := notifySource(ctx, env, sourceID, "PA system active"); err != nil {
				return err
			}
			if env.Broadcast == nil {
				return nil
			}
			peerIDs := nonSourceDeviceIDs(env, sourceID)
			if len(peerIDs) == 0 {
				return nil
			}
			return env.Broadcast.Notify(ctx, peerIDs, "PA from "+sourceID)
		},
	})
}

// Stop ends PA mode and releases any owned routes.
func (s *PASystemScenario) Stop() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "pa_system:" + sourceID,
	})
}

// Suspend releases PA-owned routes while preempted.
func (s *PASystemScenario) Suspend() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "pa_system:" + sourceID,
	})
}

// Resume reacquires PA-owned routes after preemption.
func (s *PASystemScenario) Resume(_ context.Context, env *Environment) error {
	return s.Start(context.Background(), env)
}

// AnnouncementScenario fans out source audio one-way for whole-house announcements.
type AnnouncementScenario struct {
	trigger Trigger

	recipe scenarioRecipeState
}

// Name returns the stable scenario identifier.
func (s *AnnouncementScenario) Name() string { return "announcement" }

// Match records trigger metadata when announcement mode is requested.
func (s *AnnouncementScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "announcement", "announce", "whole house announcement") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source audio to peers and announces announcement mode.
func (s *AnnouncementScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	activationID := "announcement:" + sourceID
	return s.recipe.start(ctx, env, ScenarioRecipe{
		ActivationID: activationID,
		Resolve: func(_ context.Context, env *Environment) []string {
			return peerTargetDeviceIDs(env, sourceID, s.trigger.Arguments)
		},
		Claims: func(targets []string) []iorouter.Claim {
			out := make([]iorouter.Claim, 0, len(targets)+1)
			out = append(out, iorouter.Claim{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     "mic.capture",
				Mode:         iorouter.ClaimExclusive,
				Priority:     int(PriorityHigh),
			})
			for _, targetID := range targets {
				out = append(out, iorouter.Claim{
					ActivationID: activationID,
					DeviceID:     targetID,
					Resource:     "speaker.main",
					Mode:         iorouter.ClaimExclusive,
					Priority:     int(PriorityHigh),
				})
			}
			return out
		},
		MediaPlan: func(targets []string) *iorouter.MediaPlan {
			nodes := []iorouter.MediaNode{
				{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID}},
				{ID: "fork", Kind: iorouter.NodeFork},
			}
			edges := []iorouter.MediaEdge{{From: "mic", To: "fork"}}
			for idx, targetID := range targets {
				nodeID := fmt.Sprintf("speaker_%d", idx)
				nodes = append(nodes, iorouter.MediaNode{
					ID:   nodeID,
					Kind: iorouter.NodeSinkSpeaker,
					Args: map[string]string{"device_id": targetID, "stream_kind": "announcement_audio"},
				})
				edges = append(edges, iorouter.MediaEdge{From: "fork", To: nodeID})
			}
			return &iorouter.MediaPlan{Nodes: nodes, Edges: edges}
		},
		OnStart: func(ctx context.Context, env *Environment, _ []string) error {
			if err := notifySource(ctx, env, sourceID, "Announcement active"); err != nil {
				return err
			}
			if env.Broadcast == nil {
				return nil
			}
			peerIDs := nonSourceDeviceIDs(env, sourceID)
			if len(peerIDs) == 0 {
				return nil
			}
			return env.Broadcast.Notify(ctx, peerIDs, "Announcement from "+sourceID)
		},
	})
}

// Stop ends announcement mode and releases any owned routes.
func (s *AnnouncementScenario) Stop() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "announcement:" + sourceID,
	})
}

// Suspend releases announcement-owned routes while preempted.
func (s *AnnouncementScenario) Suspend() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "announcement:" + sourceID,
	})
}

// Resume reacquires announcement-owned routes after preemption.
func (s *AnnouncementScenario) Resume(_ context.Context, env *Environment) error {
	return s.Start(context.Background(), env)
}

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

func intentMatches(intent string, accepted ...string) bool {
	normalized := strings.TrimSpace(strings.ToLower(intent))
	for _, candidate := range accepted {
		if normalized == strings.TrimSpace(strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func notifySource(ctx context.Context, env *Environment, sourceID, message string) error {
	if env == nil || env.Broadcast == nil {
		return nil
	}
	deviceIDs := []string{}
	if strings.TrimSpace(sourceID) != "" {
		deviceIDs = []string{strings.TrimSpace(sourceID)}
	}
	return env.Broadcast.Notify(ctx, deviceIDs, message)
}

type ioRoute struct {
	sourceID   string
	targetID   string
	streamKind string
}

func connectSourceToTargetsOwned(
	_ context.Context,
	env *Environment,
	sourceID string,
	targetIDs []string,
	streamKind string,
) ([]ioRoute, error) {
	if env == nil || env.IO == nil || env.Devices == nil {
		return nil, nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil, nil
	}
	if targetIDs == nil {
		targetIDs = nonSourceDeviceIDs(env, sourceID)
	}
	routes := make([]ioRoute, 0)
	for _, targetID := range targetIDs {
		if targetID == "" || targetID == sourceID {
			continue
		}
		if err := env.IO.Connect(sourceID, targetID, streamKind); err != nil {
			if errors.Is(err, iorouter.ErrRouteExists) {
				continue
			}
			return nil, err
		}
		routes = append(routes, ioRoute{
			sourceID:   sourceID,
			targetID:   targetID,
			streamKind: streamKind,
		})
	}
	return routes, nil
}

func connectBidirectionalSourceTargetsOwned(
	_ context.Context,
	env *Environment,
	sourceID string,
	targetIDs []string,
	streamKind string,
) ([]ioRoute, error) {
	if env == nil || env.IO == nil || env.Devices == nil {
		return nil, nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil, nil
	}
	if targetIDs == nil {
		targetIDs = nonSourceDeviceIDs(env, sourceID)
	}
	routes := make([]ioRoute, 0)
	for _, peerID := range targetIDs {
		if peerID == "" || peerID == sourceID {
			continue
		}
		if err := env.IO.Connect(sourceID, peerID, streamKind); err != nil {
			if !errors.Is(err, iorouter.ErrRouteExists) {
				return nil, err
			}
		} else {
			routes = append(routes, ioRoute{
				sourceID:   sourceID,
				targetID:   peerID,
				streamKind: streamKind,
			})
		}
		if err := env.IO.Connect(peerID, sourceID, streamKind); err != nil {
			if !errors.Is(err, iorouter.ErrRouteExists) {
				return nil, err
			}
		} else {
			routes = append(routes, ioRoute{
				sourceID:   peerID,
				targetID:   sourceID,
				streamKind: streamKind,
			})
		}
	}
	return routes, nil
}

func disconnectOwnedRoutes(env *Environment, routes []ioRoute) error {
	if env == nil || env.IO == nil || len(routes) == 0 {
		return nil
	}
	for _, route := range routes {
		err := env.IO.Disconnect(route.sourceID, route.targetID, route.streamKind)
		if err != nil && !errors.Is(err, iorouter.ErrRouteNotFound) {
			return err
		}
	}
	return nil
}

func reconnectOwnedRoutes(env *Environment, routes []ioRoute) error {
	if env == nil || env.IO == nil || len(routes) == 0 {
		return nil
	}
	for _, route := range routes {
		err := env.IO.Connect(route.sourceID, route.targetID, route.streamKind)
		if err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}

func nonSourceDeviceIDs(env *Environment, sourceID string) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	peers := make([]string, 0)
	for _, deviceID := range env.Devices.ListDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" || deviceID == sourceID {
			continue
		}
		peers = append(peers, deviceID)
	}
	return peers
}

func peerTargetDeviceIDs(env *Environment, sourceID string, args map[string]string) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	raw := ""
	if args != nil {
		raw = strings.TrimSpace(args["device_ids"])
	}
	if raw == "" {
		return nonSourceDeviceIDs(env, sourceID)
	}

	validSet := map[string]struct{}{}
	for _, deviceID := range env.Devices.ListDeviceIDs() {
		trimmed := strings.TrimSpace(deviceID)
		if trimmed == "" || trimmed == sourceID {
			continue
		}
		validSet[trimmed] = struct{}{}
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		deviceID := strings.TrimSpace(part)
		if deviceID == "" || deviceID == sourceID {
			continue
		}
		if _, ok := validSet[deviceID]; !ok {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		out = append(out, deviceID)
	}
	return out
}
