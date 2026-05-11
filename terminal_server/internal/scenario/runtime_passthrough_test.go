package scenario

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeScheduleMonitorSensorHookNotifiesOnMotion(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &ScheduleMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "schedule_monitor",
	}); err != nil {
		t.Fatalf("HandleTrigger(schedule_monitor) error = %v", err)
	}

	err := runtime.ProcessSensorReading(context.Background(), SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000000000,
		Values: map[string]float64{
			"accelerometer.x": 0.9,
			"accelerometer.y": 0.9,
			"accelerometer.z": 0.9,
		},
	})
	if err != nil {
		t.Fatalf("ProcessSensorReading() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "Schedule monitor active" {
		t.Fatalf("arming message = %q, want Schedule monitor active", events[0].Message)
	}
	if events[1].Message != "Schedule monitor activity detected: magnitude=1.56" {
		t.Fatalf("activity message = %q, want motion detection", events[1].Message)
	}
	if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d1" {
		t.Fatalf("activity device IDs = %+v, want [d1]", events[1].DeviceIDs)
	}
}

func TestRuntimeScheduleMonitorSensorHookRespectsCooldown(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &ScheduleMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "schedule_monitor",
		Arguments: map[string]string{
			"cooldown_ms": "60000",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(schedule_monitor) error = %v", err)
	}

	first := SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000000000,
		Values: map[string]float64{
			"motion.magnitude": 2.0,
		},
	}
	second := SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000005000, // 5s later, still within cooldown.
		Values: map[string]float64{
			"motion.magnitude": 3.0,
		},
	}

	if err := runtime.ProcessSensorReading(context.Background(), first); err != nil {
		t.Fatalf("ProcessSensorReading(first) error = %v", err)
	}
	if err := runtime.ProcessSensorReading(context.Background(), second); err != nil {
		t.Fatalf("ProcessSensorReading(second) error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2 (armed + first detection)", len(events))
	}
	if events[1].Message != "Schedule monitor activity detected: magnitude=2.00" {
		t.Fatalf("detection message = %q, want first detection magnitude", events[1].Message)
	}
}

func TestRuntimeBluetoothPassthroughDispatchAndEventHook(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	passthrough := &testPassthroughBridge{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &BluetoothPassthroughScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Passthrough: passthrough,
	})

	started, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "bluetooth_passthrough",
		Arguments: map[string]string{
			"action":    "connect",
			"target_id": "AA:BB:CC:DD",
			"profile":   "a2dp",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(bluetooth_passthrough) error = %v", err)
	}
	if started != "bluetooth_passthrough" {
		t.Fatalf("started scenario = %q, want bluetooth_passthrough", started)
	}

	if len(passthrough.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(passthrough.bluetooth))
	}
	cmd := passthrough.bluetooth[0]
	if cmd.DeviceID != "d1" || cmd.Action != "connect" || cmd.TargetID != "AA:BB:CC:DD" {
		t.Fatalf("bluetooth command = %+v", cmd)
	}
	if cmd.Parameters["profile"] != "a2dp" {
		t.Fatalf("profile = %q, want a2dp", cmd.Parameters["profile"])
	}

	if err := runtime.ProcessBluetoothEvent(context.Background(), BluetoothEvent{
		DeviceID: "d1",
		Event:    "scan_result",
		Data:     map[string]string{"target_id": "AA:BB:CC:DD"},
	}); err != nil {
		t.Fatalf("ProcessBluetoothEvent() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "Bluetooth passthrough requested: connect" {
		t.Fatalf("start message = %q", events[0].Message)
	}
	if events[1].Message != "Bluetooth event: scan_result" {
		t.Fatalf("event message = %q", events[1].Message)
	}
}

func TestRuntimeUSBPassthroughDispatchAndEventHook(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	passthrough := &testPassthroughBridge{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &USBPassthroughScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Passthrough: passthrough,
	})

	started, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "usb_passthrough",
		Arguments: map[string]string{
			"action":     "claim",
			"vendor_id":  "1a2b",
			"product_id": "3c4d",
			"mode":       "raw",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(usb_passthrough) error = %v", err)
	}
	if started != "usb_passthrough" {
		t.Fatalf("started scenario = %q, want usb_passthrough", started)
	}

	if len(passthrough.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(passthrough.usb))
	}
	cmd := passthrough.usb[0]
	if cmd.DeviceID != "d1" || cmd.Action != "claim" || cmd.VendorID != "1a2b" || cmd.ProductID != "3c4d" {
		t.Fatalf("usb command = %+v", cmd)
	}
	if cmd.Parameters["mode"] != "raw" {
		t.Fatalf("mode = %q, want raw", cmd.Parameters["mode"])
	}

	if err := runtime.ProcessUSBEvent(context.Background(), USBEvent{
		DeviceID: "d1",
		Event:    "device_claimed",
		Data:     map[string]string{"vendor_id": "1a2b", "product_id": "3c4d"},
	}); err != nil {
		t.Fatalf("ProcessUSBEvent() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "USB passthrough requested: claim" {
		t.Fatalf("start message = %q", events[0].Message)
	}
	if events[1].Message != "USB event: device_claimed" {
		t.Fatalf("event message = %q", events[1].Message)
	}
}

func TestRuntimeDispatchPassthroughCommandsUsesBridge(t *testing.T) {
	passthrough := &testPassthroughBridge{}
	runtime := NewRuntime(NewEngine(), &Environment{Passthrough: passthrough})

	err := runtime.DispatchBluetoothCommand(context.Background(), BluetoothCommand{
		DeviceID: " d1 ",
		Action:   " scan ",
		Parameters: map[string]string{
			"profile": "le",
		},
	})
	if err != nil {
		t.Fatalf("DispatchBluetoothCommand() error = %v", err)
	}
	err = runtime.DispatchUSBCommand(context.Background(), USBCommand{
		DeviceID:  " d1 ",
		Action:    " enumerate ",
		VendorID:  " 1a2b ",
		ProductID: " 3c4d ",
	})
	if err != nil {
		t.Fatalf("DispatchUSBCommand() error = %v", err)
	}

	if len(passthrough.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(passthrough.bluetooth))
	}
	if passthrough.bluetooth[0].DeviceID != "d1" || passthrough.bluetooth[0].Action != "scan" {
		t.Fatalf("unexpected bluetooth command: %+v", passthrough.bluetooth[0])
	}
	if len(passthrough.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(passthrough.usb))
	}
	if passthrough.usb[0].DeviceID != "d1" || passthrough.usb[0].Action != "enumerate" {
		t.Fatalf("unexpected usb command: %+v", passthrough.usb[0])
	}
	if passthrough.usb[0].VendorID != "1a2b" || passthrough.usb[0].ProductID != "3c4d" {
		t.Fatalf("unexpected usb vid/pid: %+v", passthrough.usb[0])
	}
}
