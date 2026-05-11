package transport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func (h *StreamHandler) recordSensorData(sensor *SensorDataRequest) {
	if sensor == nil {
		return
	}
	deviceID := strings.TrimSpace(sensor.DeviceID)
	if deviceID == "" {
		return
	}
	values := map[string]float64{}
	for key, value := range sensor.Values {
		values[key] = value
	}
	h.mu.Lock()
	h.sensorsByDevice[deviceID] = sensorSnapshot{
		UnixMS: sensor.UnixMS,
		Values: values,
	}
	h.mu.Unlock()
}

func (h *StreamHandler) sensorDataForDevice(deviceID string) (sensorSnapshot, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return sensorSnapshot{}, false
	}
	h.mu.Lock()
	snapshot, ok := h.sensorsByDevice[deviceID]
	h.mu.Unlock()
	if !ok {
		return sensorSnapshot{}, false
	}
	values := map[string]float64{}
	for key, value := range snapshot.Values {
		values[key] = value
	}
	return sensorSnapshot{
		UnixMS: snapshot.UnixMS,
		Values: values,
	}, true
}

func (h *StreamHandler) sensorStatusData() map[string]string {
	h.mu.Lock()
	byDevice := make(map[string]sensorSnapshot, len(h.sensorsByDevice))
	for deviceID, snapshot := range h.sensorsByDevice {
		values := map[string]float64{}
		for key, value := range snapshot.Values {
			values[key] = value
		}
		byDevice[deviceID] = sensorSnapshot{
			UnixMS: snapshot.UnixMS,
			Values: values,
		}
	}
	h.mu.Unlock()

	deviceIDs := make([]string, 0, len(byDevice))
	latestUnixMS := int64(0)
	details := make([]string, 0, len(byDevice))
	for deviceID, snapshot := range byDevice {
		deviceIDs = append(deviceIDs, deviceID)
		if snapshot.UnixMS > latestUnixMS {
			latestUnixMS = snapshot.UnixMS
		}
		keys := make([]string, 0, len(snapshot.Values))
		for key := range snapshot.Values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		details = append(details, fmt.Sprintf(
			"%s|unix_ms=%d|keys=%s",
			deviceID,
			snapshot.UnixMS,
			strings.Join(keys, ","),
		))
	}
	sort.Strings(deviceIDs)
	sort.Strings(details)

	return map[string]string{
		"sensor_devices_reporting": strconv.Itoa(len(deviceIDs)),
		"sensor_latest_unix_ms":    strconv.FormatInt(latestUnixMS, 10),
		"sensor_device_ids":        strings.Join(deviceIDs, ","),
		"sensor_summaries":         strings.Join(details, ";"),
	}
}
