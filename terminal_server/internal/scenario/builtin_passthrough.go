package scenario

import (
	"context"
	"strings"
)

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
