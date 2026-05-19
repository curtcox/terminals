package transport

import (
	"sort"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func capabilityDisplayResources(caps map[string]string, resources map[string]struct{}) {
	if (caps["screen.width"] != "" && caps["screen.height"] != "") || truthyCapability(caps["display.count"]) {
		resources["screen.main"] = struct{}{}
		resources["screen.overlay"] = struct{}{}
	}
	for _, displayID := range endpointResourceIDs(caps, "display.") {
		resources["display."+displayID+".main"] = struct{}{}
		resources["display."+displayID+".overlay"] = struct{}{}
	}
}

func capabilityInputResources(caps map[string]string, resources map[string]struct{}) {
	if truthyCapability(caps["keyboard.physical"]) || strings.TrimSpace(caps["keyboard.layout"]) != "" {
		resources["keyboard.primary"] = struct{}{}
	}
	if strings.TrimSpace(caps["pointer.type"]) != "" {
		resources["pointer.primary"] = struct{}{}
	}
	if truthyCapability(caps["touch.supported"]) {
		resources["touch.primary"] = struct{}{}
	}
}

func capabilityAudioResources(caps map[string]string, resources map[string]struct{}) {
	if truthyCapability(caps["speakers.present"]) || truthyCapability(caps["speakers.endpoint_count"]) {
		resources["speaker.main"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "speakers.endpoint.") {
		resources["audio_out."+endpointID] = struct{}{}
	}
	if truthyCapability(caps["microphone.present"]) || truthyCapability(caps["microphone.endpoint_count"]) {
		resources["mic.capture"] = struct{}{}
		resources["mic.analyze"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "microphone.endpoint.") {
		resources["audio_in."+endpointID+".capture"] = struct{}{}
		resources["audio_in."+endpointID+".analyze"] = struct{}{}
	}
}

func capabilityCameraResources(caps map[string]string, resources map[string]struct{}) {
	if truthyCapability(caps["camera.present"]) || truthyCapability(caps["camera.endpoint_count"]) {
		resources["camera.capture"] = struct{}{}
		resources["camera.analyze"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "camera.endpoint.") {
		resources["camera."+endpointID+".capture"] = struct{}{}
		resources["camera."+endpointID+".analyze"] = struct{}{}
	}
}

func capabilityEdgeResources(caps map[string]string, resources map[string]struct{}) {
	if truthyCapability(caps["haptics.supported"]) {
		resources["haptic.primary"] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.cpu_realtime"]) {
		resources[iorouter.ResourceComputeCPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.gpu_realtime"]) {
		resources[iorouter.ResourceComputeGPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.npu_realtime"]) {
		resources[iorouter.ResourceComputeNPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.audio_sec"]) {
		resources[iorouter.ResourceBufferAudio] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.video_sec"]) {
		resources[iorouter.ResourceBufferVideo] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.sensor_sec"]) {
		resources[iorouter.ResourceBufferSensor] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.radio_sec"]) {
		resources[iorouter.ResourceBufferRadio] = struct{}{}
	}
}

func capabilityConnectivityResources(caps map[string]string, resources map[string]struct{}) {
	if truthyCapability(caps["connectivity.bluetooth_version"]) {
		resources[iorouter.ResourceRadioBLEScan] = struct{}{}
	}
	if truthyCapability(caps["connectivity.wifi_signal_strength"]) {
		resources[iorouter.ResourceRadioWiFiScan] = struct{}{}
	}
}

type endpointIndexState struct {
	indexToID            map[string]string
	indexes              map[string]struct{}
	indexHasAvailability map[string]bool
	indexAvailable       map[string]bool
}

func parseEndpointIndexState(caps map[string]string, prefix string) endpointIndexState {
	state := endpointIndexState{
		indexToID:            map[string]string{},
		indexes:              map[string]struct{}{},
		indexHasAvailability: map[string]bool{},
		indexAvailable:       map[string]bool{},
	}
	for key, value := range caps {
		rest, ok := strings.CutPrefix(key, prefix)
		if !ok {
			continue
		}
		parts := strings.Split(rest, ".")
		if len(parts) < 2 {
			continue
		}
		index := strings.TrimSpace(parts[0])
		if index == "" {
			continue
		}
		state.indexes[index] = struct{}{}
		switch parts[1] {
		case "id":
			if id := sanitizeResourceID(value); id != "" {
				state.indexToID[index] = id
			}
		case "available":
			state.indexHasAvailability[index] = true
			state.indexAvailable[index] = truthyCapability(value)
		}
	}
	return state
}

func endpointIDsFromIndexState(state endpointIndexState) []string {
	if len(state.indexes) == 0 {
		return nil
	}
	sortedIndexes := make([]string, 0, len(state.indexes))
	for index := range state.indexes {
		sortedIndexes = append(sortedIndexes, index)
	}
	sort.Strings(sortedIndexes)

	ids := make([]string, 0, len(sortedIndexes))
	for _, index := range sortedIndexes {
		if state.indexHasAvailability[index] && !state.indexAvailable[index] {
			continue
		}
		if id := state.indexToID[index]; id != "" {
			ids = append(ids, id)
			continue
		}
		ids = append(ids, "endpoint-"+sanitizeResourceID(index))
	}
	return ids
}
