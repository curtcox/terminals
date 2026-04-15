import 'package:fixnum/fixnum.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart';

/// Builds sensor telemetry envelopes for the control stream.
class SensorStreamer {
  ConnectRequest buildSensorRequest({
    required String deviceId,
    required int unixMs,
    required Map<String, double> values,
  }) {
    return ConnectRequest()
      ..sensor = (SensorData()
        ..deviceId = deviceId
        ..unixMs = Int64(unixMs)
        ..values.addAll(values));
  }
}
