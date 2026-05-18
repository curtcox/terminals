package transport

import (
	"strconv"
	"strings"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
)

func capabilitiesToDataMap(caps *capabilitiesv1.DeviceCapabilities) map[string]string {
	if caps == nil {
		return map[string]string{}
	}

	out := map[string]string{
		"device_id": caps.GetDeviceId(),
	}
	addIdentityCapabilities(out, caps.GetIdentity())
	addScreenCapabilities(out, "screen", caps.GetScreen(), true)
	addDisplayCapabilities(out, caps.GetDisplays())
	addInputCapabilities(out, caps)
	addAudioOutputCapabilities(out, caps.GetSpeakers())
	addAudioInputCapabilities(out, caps.GetMicrophone())
	addCameraCapabilities(out, caps.GetCamera())
	addSensorCapabilities(out, caps.GetSensors())
	addConnectivityCapabilities(out, caps.GetConnectivity())
	addBatteryCapabilities(out, caps.GetBattery())
	addHapticCapabilities(out, caps.GetHaptics())
	addEdgeCapabilities(out, caps.GetEdge())
	return out
}

func addIdentityCapabilities(out map[string]string, identity *capabilitiesv1.DeviceIdentity) {
	if identity == nil {
		return
	}
	out["device_name"] = identity.GetDeviceName()
	out["device_type"] = identity.GetDeviceType()
	out["platform"] = identity.GetPlatform()
}

func addScreenCapabilities(out map[string]string, prefix string, screen *capabilitiesv1.ScreenCapability, includeExtras bool) {
	if screen == nil {
		return
	}
	out[prefix+".width"] = strconv.FormatInt(int64(screen.GetWidth()), 10)
	out[prefix+".height"] = strconv.FormatInt(int64(screen.GetHeight()), 10)
	out[prefix+".density"] = strconv.FormatFloat(screen.GetDensity(), 'f', -1, 64)
	if orientation := strings.TrimSpace(screen.GetOrientation()); orientation != "" {
		out[prefix+".orientation"] = orientation
	}
	if !includeExtras {
		return
	}
	if screen.GetTouch() {
		out[prefix+".touch"] = "true"
	}
	if screen.GetFullscreenSupported() {
		out[prefix+".fullscreen_supported"] = "true"
	}
	if screen.GetMultiWindowSupported() {
		out[prefix+".multi_window_supported"] = "true"
	}
	if safeArea := screen.GetSafeArea(); safeArea != nil {
		out[prefix+".safe_area.left"] = strconv.FormatInt(int64(safeArea.GetLeft()), 10)
		out[prefix+".safe_area.top"] = strconv.FormatInt(int64(safeArea.GetTop()), 10)
		out[prefix+".safe_area.right"] = strconv.FormatInt(int64(safeArea.GetRight()), 10)
		out[prefix+".safe_area.bottom"] = strconv.FormatInt(int64(safeArea.GetBottom()), 10)
	}
}

func addDisplayCapabilities(out map[string]string, displays []*capabilitiesv1.DisplayCapability) {
	if len(displays) == 0 {
		return
	}
	out["display.count"] = strconv.FormatInt(int64(len(displays)), 10)
	for idx, display := range displays {
		prefix := "display." + strconv.Itoa(idx)
		if displayID := strings.TrimSpace(display.GetDisplayId()); displayID != "" {
			out[prefix+".id"] = displayID
		}
		if name := strings.TrimSpace(display.GetDisplayName()); name != "" {
			out[prefix+".name"] = name
		}
		if display.GetPrimary() {
			out[prefix+".primary"] = "true"
		}
		addScreenCapabilities(out, prefix, display.GetScreen(), false)
	}
}

func addInputCapabilities(out map[string]string, caps *capabilitiesv1.DeviceCapabilities) {
	if keyboard := caps.GetKeyboard(); keyboard != nil {
		out["keyboard.physical"] = strconv.FormatBool(keyboard.GetPhysical())
		out["keyboard.layout"] = keyboard.GetLayout()
	}
	if pointer := caps.GetPointer(); pointer != nil {
		out["pointer.type"] = pointer.GetType()
		out["pointer.hover"] = strconv.FormatBool(pointer.GetHover())
	}
	if touch := caps.GetTouch(); touch != nil {
		out["touch.supported"] = strconv.FormatBool(touch.GetSupported())
		out["touch.max_points"] = strconv.FormatInt(int64(touch.GetMaxPoints()), 10)
	}
}

func addAudioOutputCapabilities(out map[string]string, speakers *capabilitiesv1.AudioOutputCapability) {
	if speakers == nil {
		return
	}
	out["speakers.present"] = "true"
	addAudioStats(out, "speakers", speakers.GetChannels(), speakers.GetSampleRates())
	addAudioEndpoints(out, "speakers.endpoint", speakers.GetEndpoints())
}

func addAudioInputCapabilities(out map[string]string, mic *capabilitiesv1.AudioInputCapability) {
	if mic == nil {
		return
	}
	out["microphone.present"] = "true"
	addAudioStats(out, "microphone", mic.GetChannels(), mic.GetSampleRates())
	addAudioEndpoints(out, "microphone.endpoint", mic.GetEndpoints())
}

func addAudioStats(out map[string]string, prefix string, channels int32, rates []int32) {
	if channels > 0 {
		out[prefix+".channels"] = strconv.FormatInt(int64(channels), 10)
	}
	if formatted := joinInts(rates); formatted != "" {
		out[prefix+".sample_rates"] = formatted
	}
}

func addAudioEndpoints(out map[string]string, prefix string, endpoints []*capabilitiesv1.AudioEndpoint) {
	if len(endpoints) == 0 {
		return
	}
	out[prefix+"_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
	for idx, endpoint := range endpoints {
		endpointPrefix := prefix + "." + strconv.Itoa(idx)
		addTrimmed(out, endpointPrefix+".id", endpoint.GetEndpointId())
		addTrimmed(out, endpointPrefix+".name", endpoint.GetEndpointName())
		addTrimmed(out, endpointPrefix+".connection_type", endpoint.GetConnectionType())
		addAudioStats(out, endpointPrefix, endpoint.GetChannels(), endpoint.GetSampleRates())
		if endpoint.GetAvailable() {
			out[endpointPrefix+".available"] = "true"
		}
	}
}

func addCameraCapabilities(out map[string]string, camera *capabilitiesv1.CameraCapability) {
	if camera == nil {
		return
	}
	out["camera.present"] = "true"
	addCameraLens(out, "camera.front", camera.GetFront())
	addCameraLens(out, "camera.back", camera.GetBack())
	addCameraEndpoints(out, camera.GetEndpoints())
}

func addCameraLens(out map[string]string, prefix string, lens *capabilitiesv1.CameraLens) {
	if lens == nil {
		return
	}
	addPositiveInt(out, prefix+".width", lens.GetWidth())
	addPositiveInt(out, prefix+".height", lens.GetHeight())
	addPositiveInt(out, prefix+".fps", lens.GetFps())
}

func addCameraEndpoints(out map[string]string, endpoints []*capabilitiesv1.CameraEndpoint) {
	if len(endpoints) == 0 {
		return
	}
	out["camera.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
	for idx, endpoint := range endpoints {
		prefix := "camera.endpoint." + strconv.Itoa(idx)
		addTrimmed(out, prefix+".id", endpoint.GetEndpointId())
		addTrimmed(out, prefix+".name", endpoint.GetEndpointName())
		addTrimmed(out, prefix+".connection_type", endpoint.GetConnectionType())
		addTrimmed(out, prefix+".facing", endpoint.GetFacing())
		if endpoint.GetAvailable() {
			out[prefix+".available"] = "true"
		}
		addCameraModes(out, prefix, endpoint.GetModes())
	}
}

func addCameraModes(out map[string]string, prefix string, modes []*capabilitiesv1.CameraLens) {
	for modeIndex, mode := range modes {
		modePrefix := prefix + ".mode." + strconv.Itoa(modeIndex)
		addPositiveInt(out, modePrefix+".width", mode.GetWidth())
		addPositiveInt(out, modePrefix+".height", mode.GetHeight())
		addPositiveInt(out, modePrefix+".fps", mode.GetFps())
	}
}

func addSensorCapabilities(out map[string]string, sensors *capabilitiesv1.SensorCapability) {
	if sensors == nil {
		return
	}
	addTrue(out, "sensors.accelerometer", sensors.GetAccelerometer())
	addTrue(out, "sensors.gyroscope", sensors.GetGyroscope())
	addTrue(out, "sensors.compass", sensors.GetCompass())
	addTrue(out, "sensors.ambient_light", sensors.GetAmbientLight())
	addTrue(out, "sensors.proximity", sensors.GetProximity())
	addTrue(out, "sensors.gps", sensors.GetGps())
}

func addConnectivityCapabilities(out map[string]string, connectivity *capabilitiesv1.ConnectivityCapability) {
	if connectivity == nil {
		return
	}
	addTrimmed(out, "connectivity.bluetooth_version", connectivity.GetBluetoothVersion())
	addTrue(out, "connectivity.wifi_signal_strength", connectivity.GetWifiSignalStrength())
	addTrue(out, "connectivity.usb_host", connectivity.GetUsbHost())
	addPositiveInt(out, "connectivity.usb_ports", connectivity.GetUsbPorts())
	addTrue(out, "connectivity.nfc", connectivity.GetNfc())
}

func addBatteryCapabilities(out map[string]string, battery *capabilitiesv1.BatteryCapability) {
	if battery == nil {
		return
	}
	out["battery.level"] = strconv.FormatFloat(float64(battery.GetLevel()), 'f', -1, 32)
	out["battery.charging"] = strconv.FormatBool(battery.GetCharging())
}

func addHapticCapabilities(out map[string]string, haptics *capabilitiesv1.HapticCapability) {
	if haptics == nil {
		return
	}
	addTrue(out, "haptics.supported", haptics.GetSupported())
	addTrue(out, "haptics.vibration", haptics.GetVibration())
	addTrue(out, "haptics.engine", haptics.GetHapticsEngine())
}

func addEdgeCapabilities(out map[string]string, edge *capabilitiesv1.EdgeCapability) {
	if edge == nil {
		return
	}
	if runtimes := edge.GetRuntimes(); len(runtimes) > 0 {
		out["edge.runtimes"] = strings.Join(runtimes, ",")
	}
	addEdgeOperators(out, edge.GetOperators())
	addEdgeCompute(out, edge.GetCompute())
	addEdgeRetention(out, edge.GetRetention())
	if timing := edge.GetTiming(); timing != nil {
		out["edge.timing.sync_error_ms"] = strconv.FormatFloat(timing.GetSyncErrorMs(), 'f', -1, 64)
	}
	if geometry := edge.GetGeometry(); geometry != nil {
		out["edge.geometry.mic_array"] = strconv.FormatBool(geometry.GetMicArray())
		out["edge.geometry.camera_intrinsics"] = strconv.FormatBool(geometry.GetCameraIntrinsics())
		out["edge.geometry.compass"] = strconv.FormatBool(geometry.GetCompass())
	}
}

func addEdgeOperators(out map[string]string, operators []string) {
	if len(operators) == 0 {
		return
	}
	out["edge.operators"] = strings.Join(operators, ",")
	foregroundOnly := false
	backgroundCapable := false
	for _, operator := range operators {
		switch strings.TrimSpace(strings.ToLower(operator)) {
		case "monitor.foreground_only", "monitor.tier.foreground_only":
			foregroundOnly = true
			out["monitor.foreground_only"] = "true"
			out["monitor.support_tier"] = "foreground_only"
		case "monitor.background_capable", "monitor.tier.background_capable":
			backgroundCapable = true
			out["monitor.background_capable"] = "true"
			out["monitor.support_tier"] = "background_capable"
		case "monitor.lifecycle.foreground":
			out["monitor.runtime_state"] = "foreground"
		case "monitor.lifecycle.background":
			out["monitor.runtime_state"] = "background"
		}
	}
	if foregroundOnly && !backgroundCapable {
		out["monitor.background_capable"] = "false"
	}
}

func addEdgeCompute(out map[string]string, compute *capabilitiesv1.EdgeComputeCapability) {
	if compute == nil {
		return
	}
	out["edge.compute.cpu_realtime"] = strconv.FormatInt(int64(compute.GetCpuRealtime()), 10)
	out["edge.compute.gpu_realtime"] = strconv.FormatInt(int64(compute.GetGpuRealtime()), 10)
	out["edge.compute.npu_realtime"] = strconv.FormatInt(int64(compute.GetNpuRealtime()), 10)
	out["edge.compute.mem_mb"] = strconv.FormatInt(int64(compute.GetMemMb()), 10)
}

func addEdgeRetention(out map[string]string, retention *capabilitiesv1.EdgeRetentionCapability) {
	if retention == nil {
		return
	}
	out["edge.retention.audio_sec"] = strconv.FormatInt(int64(retention.GetAudioSec()), 10)
	out["edge.retention.video_sec"] = strconv.FormatInt(int64(retention.GetVideoSec()), 10)
	out["edge.retention.sensor_sec"] = strconv.FormatInt(int64(retention.GetSensorSec()), 10)
	out["edge.retention.radio_sec"] = strconv.FormatInt(int64(retention.GetRadioSec()), 10)
}

func addTrimmed(out map[string]string, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		out[key] = trimmed
	}
}

func addTrue(out map[string]string, key string, value bool) {
	if value {
		out[key] = "true"
	}
}

func addPositiveInt(out map[string]string, key string, value int32) {
	if value > 0 {
		out[key] = strconv.FormatInt(int64(value), 10)
	}
}

func joinInts(values []int32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatInt(int64(v), 10))
	}
	return strings.Join(parts, ",")
}
