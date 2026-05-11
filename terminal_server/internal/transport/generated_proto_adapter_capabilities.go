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
	identity := caps.GetIdentity()
	if identity != nil {
		out["device_name"] = identity.GetDeviceName()
		out["device_type"] = identity.GetDeviceType()
		out["platform"] = identity.GetPlatform()
	}
	if screen := caps.GetScreen(); screen != nil {
		out["screen.width"] = strconv.FormatInt(int64(screen.GetWidth()), 10)
		out["screen.height"] = strconv.FormatInt(int64(screen.GetHeight()), 10)
		out["screen.density"] = strconv.FormatFloat(screen.GetDensity(), 'f', -1, 64)
		if screen.GetTouch() {
			out["screen.touch"] = "true"
		}
		if orientation := strings.TrimSpace(screen.GetOrientation()); orientation != "" {
			out["screen.orientation"] = orientation
		}
		if screen.GetFullscreenSupported() {
			out["screen.fullscreen_supported"] = "true"
		}
		if screen.GetMultiWindowSupported() {
			out["screen.multi_window_supported"] = "true"
		}
		if safeArea := screen.GetSafeArea(); safeArea != nil {
			out["screen.safe_area.left"] = strconv.FormatInt(int64(safeArea.GetLeft()), 10)
			out["screen.safe_area.top"] = strconv.FormatInt(int64(safeArea.GetTop()), 10)
			out["screen.safe_area.right"] = strconv.FormatInt(int64(safeArea.GetRight()), 10)
			out["screen.safe_area.bottom"] = strconv.FormatInt(int64(safeArea.GetBottom()), 10)
		}
	}
	if displays := caps.GetDisplays(); len(displays) > 0 {
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
			if screen := display.GetScreen(); screen != nil {
				out[prefix+".width"] = strconv.FormatInt(int64(screen.GetWidth()), 10)
				out[prefix+".height"] = strconv.FormatInt(int64(screen.GetHeight()), 10)
				out[prefix+".density"] = strconv.FormatFloat(screen.GetDensity(), 'f', -1, 64)
				if orientation := strings.TrimSpace(screen.GetOrientation()); orientation != "" {
					out[prefix+".orientation"] = orientation
				}
			}
		}
	}
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
	if speakers := caps.GetSpeakers(); speakers != nil {
		out["speakers.present"] = "true"
		if speakers.GetChannels() > 0 {
			out["speakers.channels"] = strconv.FormatInt(int64(speakers.GetChannels()), 10)
		}
		if rates := joinInts(speakers.GetSampleRates()); rates != "" {
			out["speakers.sample_rates"] = rates
		}
		if endpoints := speakers.GetEndpoints(); len(endpoints) > 0 {
			out["speakers.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "speakers.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				if endpoint.GetChannels() > 0 {
					out[prefix+".channels"] = strconv.FormatInt(int64(endpoint.GetChannels()), 10)
				}
				if rates := joinInts(endpoint.GetSampleRates()); rates != "" {
					out[prefix+".sample_rates"] = rates
				}
				if endpoint.GetAvailable() {
					out[prefix+".available"] = "true"
				}
			}
		}
	}
	if mic := caps.GetMicrophone(); mic != nil {
		out["microphone.present"] = "true"
		if mic.GetChannels() > 0 {
			out["microphone.channels"] = strconv.FormatInt(int64(mic.GetChannels()), 10)
		}
		if rates := joinInts(mic.GetSampleRates()); rates != "" {
			out["microphone.sample_rates"] = rates
		}
		if endpoints := mic.GetEndpoints(); len(endpoints) > 0 {
			out["microphone.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "microphone.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				if endpoint.GetChannels() > 0 {
					out[prefix+".channels"] = strconv.FormatInt(int64(endpoint.GetChannels()), 10)
				}
				if rates := joinInts(endpoint.GetSampleRates()); rates != "" {
					out[prefix+".sample_rates"] = rates
				}
				if endpoint.GetAvailable() {
					out[prefix+".available"] = "true"
				}
			}
		}
	}
	if camera := caps.GetCamera(); camera != nil {
		out["camera.present"] = "true"
		if front := camera.GetFront(); front != nil {
			if front.GetWidth() > 0 {
				out["camera.front.width"] = strconv.FormatInt(int64(front.GetWidth()), 10)
			}
			if front.GetHeight() > 0 {
				out["camera.front.height"] = strconv.FormatInt(int64(front.GetHeight()), 10)
			}
			if front.GetFps() > 0 {
				out["camera.front.fps"] = strconv.FormatInt(int64(front.GetFps()), 10)
			}
		}
		if back := camera.GetBack(); back != nil {
			if back.GetWidth() > 0 {
				out["camera.back.width"] = strconv.FormatInt(int64(back.GetWidth()), 10)
			}
			if back.GetHeight() > 0 {
				out["camera.back.height"] = strconv.FormatInt(int64(back.GetHeight()), 10)
			}
			if back.GetFps() > 0 {
				out["camera.back.fps"] = strconv.FormatInt(int64(back.GetFps()), 10)
			}
		}
		if endpoints := camera.GetEndpoints(); len(endpoints) > 0 {
			out["camera.endpoint_count"] = strconv.FormatInt(int64(len(endpoints)), 10)
			for idx, endpoint := range endpoints {
				prefix := "camera.endpoint." + strconv.Itoa(idx)
				if endpointID := strings.TrimSpace(endpoint.GetEndpointId()); endpointID != "" {
					out[prefix+".id"] = endpointID
				}
				if endpointName := strings.TrimSpace(endpoint.GetEndpointName()); endpointName != "" {
					out[prefix+".name"] = endpointName
				}
				if connectionType := strings.TrimSpace(endpoint.GetConnectionType()); connectionType != "" {
					out[prefix+".connection_type"] = connectionType
				}
				if facing := strings.TrimSpace(endpoint.GetFacing()); facing != "" {
					out[prefix+".facing"] = facing
				}
				if endpoint.GetAvailable() {
					out[prefix+".available"] = "true"
				}
				if modes := endpoint.GetModes(); len(modes) > 0 {
					for modeIndex, mode := range modes {
						modePrefix := prefix + ".mode." + strconv.Itoa(modeIndex)
						if mode.GetWidth() > 0 {
							out[modePrefix+".width"] = strconv.FormatInt(int64(mode.GetWidth()), 10)
						}
						if mode.GetHeight() > 0 {
							out[modePrefix+".height"] = strconv.FormatInt(int64(mode.GetHeight()), 10)
						}
						if mode.GetFps() > 0 {
							out[modePrefix+".fps"] = strconv.FormatInt(int64(mode.GetFps()), 10)
						}
					}
				}
			}
		}
	}
	if sensors := caps.GetSensors(); sensors != nil {
		if sensors.GetAccelerometer() {
			out["sensors.accelerometer"] = "true"
		}
		if sensors.GetGyroscope() {
			out["sensors.gyroscope"] = "true"
		}
		if sensors.GetCompass() {
			out["sensors.compass"] = "true"
		}
		if sensors.GetAmbientLight() {
			out["sensors.ambient_light"] = "true"
		}
		if sensors.GetProximity() {
			out["sensors.proximity"] = "true"
		}
		if sensors.GetGps() {
			out["sensors.gps"] = "true"
		}
	}
	if connectivity := caps.GetConnectivity(); connectivity != nil {
		if bluetoothVersion := strings.TrimSpace(connectivity.GetBluetoothVersion()); bluetoothVersion != "" {
			out["connectivity.bluetooth_version"] = bluetoothVersion
		}
		if connectivity.GetWifiSignalStrength() {
			out["connectivity.wifi_signal_strength"] = "true"
		}
		if connectivity.GetUsbHost() {
			out["connectivity.usb_host"] = "true"
		}
		if connectivity.GetUsbPorts() > 0 {
			out["connectivity.usb_ports"] = strconv.FormatInt(int64(connectivity.GetUsbPorts()), 10)
		}
		if connectivity.GetNfc() {
			out["connectivity.nfc"] = "true"
		}
	}
	if battery := caps.GetBattery(); battery != nil {
		out["battery.level"] = strconv.FormatFloat(float64(battery.GetLevel()), 'f', -1, 32)
		out["battery.charging"] = strconv.FormatBool(battery.GetCharging())
	}
	if haptics := caps.GetHaptics(); haptics != nil {
		if haptics.GetSupported() {
			out["haptics.supported"] = "true"
		}
		if haptics.GetVibration() {
			out["haptics.vibration"] = "true"
		}
		if haptics.GetHapticsEngine() {
			out["haptics.engine"] = "true"
		}
	}
	if edge := caps.GetEdge(); edge != nil {
		if runtimes := edge.GetRuntimes(); len(runtimes) > 0 {
			out["edge.runtimes"] = strings.Join(runtimes, ",")
		}
		operators := edge.GetOperators()
		if len(operators) > 0 {
			out["edge.operators"] = strings.Join(operators, ",")
		}
		foregroundOnly := false
		backgroundCapable := false
		for _, operator := range operators {
			normalized := strings.TrimSpace(strings.ToLower(operator))
			switch normalized {
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
		if compute := edge.GetCompute(); compute != nil {
			out["edge.compute.cpu_realtime"] = strconv.FormatInt(int64(compute.GetCpuRealtime()), 10)
			out["edge.compute.gpu_realtime"] = strconv.FormatInt(int64(compute.GetGpuRealtime()), 10)
			out["edge.compute.npu_realtime"] = strconv.FormatInt(int64(compute.GetNpuRealtime()), 10)
			out["edge.compute.mem_mb"] = strconv.FormatInt(int64(compute.GetMemMb()), 10)
		}
		if retention := edge.GetRetention(); retention != nil {
			out["edge.retention.audio_sec"] = strconv.FormatInt(int64(retention.GetAudioSec()), 10)
			out["edge.retention.video_sec"] = strconv.FormatInt(int64(retention.GetVideoSec()), 10)
			out["edge.retention.sensor_sec"] = strconv.FormatInt(int64(retention.GetSensorSec()), 10)
			out["edge.retention.radio_sec"] = strconv.FormatInt(int64(retention.GetRadioSec()), 10)
		}
		if timing := edge.GetTiming(); timing != nil {
			out["edge.timing.sync_error_ms"] = strconv.FormatFloat(timing.GetSyncErrorMs(), 'f', -1, 64)
		}
		if geometry := edge.GetGeometry(); geometry != nil {
			out["edge.geometry.mic_array"] = strconv.FormatBool(geometry.GetMicArray())
			out["edge.geometry.camera_intrinsics"] = strconv.FormatBool(geometry.GetCameraIntrinsics())
			out["edge.geometry.compass"] = strconv.FormatBool(geometry.GetCompass())
		}
	}
	return out
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
