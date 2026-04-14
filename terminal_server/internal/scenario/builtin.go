// Package scenario contains server-side scenario matching and runtime flows.
package scenario

import (
	"context"
	"errors"
	"io"
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

// IntercomScenario connects source device audio to all peer devices.
type IntercomScenario struct {
	trigger Trigger
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
	if err := connectBidirectionalSourcePeers(ctx, env, s.trigger.SourceID, "audio"); err != nil {
		return err
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Intercom active")
}

// Stop ends intercom mode and currently has no side effects.
func (s *IntercomScenario) Stop() error { return nil }

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

// Stop ends assistant mode and currently has no side effects.
func (s *VoiceAssistantScenario) Stop() error { return nil }

// AudioMonitorScenario stores monitor intent and confirms arming.
type AudioMonitorScenario struct {
	trigger Trigger

	mu     sync.Mutex
	stopFn func()
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
	if env.Sound == nil {
		return nil
	}

	sourceID := strings.TrimSpace(s.trigger.SourceID)

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
	s.mu.Lock()
	stop := s.stopFn
	s.stopFn = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
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
	return notifySource(ctx, env, s.trigger.SourceID, "Schedule monitor active")
}

// Stop ends schedule monitor mode and currently has no side effects.
func (s *ScheduleMonitorScenario) Stop() error { return nil }

// PASystemScenario fans out source audio to peers with PA semantics.
type PASystemScenario struct {
	trigger Trigger
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
	if err := connectSourceToPeers(ctx, env, s.trigger.SourceID, "pa_audio"); err != nil {
		return err
	}
	if err := notifySource(ctx, env, s.trigger.SourceID, "PA system active"); err != nil {
		return err
	}
	if env.Broadcast != nil {
		sourceID := strings.TrimSpace(s.trigger.SourceID)
		peerIDs := nonSourceDeviceIDs(env, sourceID)
		if len(peerIDs) > 0 {
			return env.Broadcast.Notify(ctx, peerIDs, "PA from "+sourceID)
		}
	}
	return nil
}

// Stop ends PA mode and currently has no side effects.
func (s *PASystemScenario) Stop() error { return nil }

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
		peers := make([]string, 0, len(env.Devices.ListDeviceIDs()))
		for _, peer := range env.Devices.ListDeviceIDs() {
			if peer == "" || peer == source {
				continue
			}
			peers = append(peers, peer)
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

func connectSourceToPeers(_ context.Context, env *Environment, sourceID, streamKind string) error {
	if env == nil || env.IO == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil
	}
	for _, targetID := range env.Devices.ListDeviceIDs() {
		if targetID == "" || targetID == sourceID {
			continue
		}
		if err := env.IO.Connect(sourceID, targetID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}

func connectBidirectionalSourcePeers(_ context.Context, env *Environment, sourceID, streamKind string) error {
	if env == nil || env.IO == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil
	}
	for _, peerID := range env.Devices.ListDeviceIDs() {
		if peerID == "" || peerID == sourceID {
			continue
		}
		if err := env.IO.Connect(sourceID, peerID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
		if err := env.IO.Connect(peerID, sourceID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
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
