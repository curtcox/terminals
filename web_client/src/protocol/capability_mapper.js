import { create } from "@bufbuild/protobuf";
import { DeviceCapabilitiesSchema } from "./generated/terminals/capabilities/v1/capabilities_pb.js";
import { ConnectRequestSchema } from "./generated/terminals/control/v1/control_pb.js";

export function mapBrowserProbeToCapabilities(probe) {
  return create(DeviceCapabilitiesSchema, {
    deviceId: probe.deviceId ?? "web-client",
    identity: {
      deviceName: probe.deviceName ?? "Browser",
      deviceType: "browser",
      platform: "web"
    },
    screen: {
      width: probe.viewportWidth ?? 0,
      height: probe.viewportHeight ?? 0,
      density: probe.devicePixelRatio ?? 1,
      touch: Boolean(probe.touch),
      orientation: probe.orientation ?? "",
      fullscreenSupported: Boolean(probe.fullscreen)
    },
    keyboard: { physical: Boolean(probe.keyboard), layout: "" },
    pointer: { type: probe.pointerType ?? "unknown", hover: Boolean(probe.hover) },
    touch: { supported: Boolean(probe.touch), maxPoints: probe.maxTouchPoints ?? 0 },
    speakers: probe.audioOutput ? { channels: 2, sampleRates: [44100, 48000], endpoints: [] } : null,
    microphone: probe.audioInput ? { channels: 1, sampleRates: [44100, 48000], endpoints: [] } : null,
    camera: probe.camera ? { endpoints: [] } : null,
    connectivity: { bluetoothVersion: "", wifiSignalStrength: false, usbHost: false, usbPorts: 0, nfc: false },
    displays: []
  });
}

export function mapCapabilitiesToHelloRequest(capabilities, { clientVersion = "web-client/dev" } = {}) {
  const deviceId = capabilities?.deviceId || "web-client";
  return create(ConnectRequestSchema, {
    payload: {
      case: "hello",
      value: {
        deviceId,
        identity: capabilities?.identity ?? {
          deviceName: "Browser",
          deviceType: "browser",
          platform: "web"
        },
        clientVersion
      }
    }
  });
}

export function mapCapabilitiesToSnapshotRequest(capabilities, { generation = 1n } = {}) {
  const deviceId = capabilities?.deviceId || "web-client";
  return create(ConnectRequestSchema, {
    payload: {
      case: "capabilitySnapshot",
      value: {
        deviceId,
        generation,
        capabilities
      }
    }
  });
}
