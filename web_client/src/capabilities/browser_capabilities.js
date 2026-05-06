import { readScreenMetrics } from "./screen_metrics.js";
import { mapBrowserProbeToCapabilities } from "../protocol/capability_mapper.js";

export function probeBrowserCapabilities({ navigator = globalThis.navigator, window = globalThis.window, document = globalThis.document } = {}) {
  const metrics = readScreenMetrics({ window, document });
  return {
    ...metrics,
    deviceId: "web-client",
    deviceName: navigator?.userAgentData?.platform || "Browser",
    keyboard: true,
    pointerType: window?.matchMedia?.("(pointer: fine)")?.matches ? "mouse" : "coarse",
    hover: window?.matchMedia?.("(hover: hover)")?.matches ?? false,
    touch: (navigator?.maxTouchPoints ?? 0) > 0,
    maxTouchPoints: navigator?.maxTouchPoints ?? 0,
    fullscreen: Boolean(document?.documentElement?.requestFullscreen),
    audioOutput: typeof Audio !== "undefined",
    audioInput: Boolean(navigator?.mediaDevices?.getUserMedia),
    camera: Boolean(navigator?.mediaDevices?.getUserMedia),
    webrtc: typeof RTCPeerConnection !== "undefined",
    notifications: typeof Notification !== "undefined",
    wakeLock: Boolean(navigator?.wakeLock)
  };
}

export function createBrowserCapabilitySnapshot(deps = {}) {
  return mapBrowserProbeToCapabilities(probeBrowserCapabilities(deps));
}
