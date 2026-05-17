package usecasevalidation

import (
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
)

// SensorDataRequest builds a ConnectRequest carrying SensorData for harness use.
func SensorDataRequest(deviceID string, unixMS int64, values map[string]float64) *controlv1.ConnectRequest {
	return &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Sensor{
			Sensor: &iov1.SensorData{
				DeviceId: deviceID,
				UnixMs:   unixMS,
				Values:   values,
			},
		},
	}
}
