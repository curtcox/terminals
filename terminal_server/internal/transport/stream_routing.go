package transport

import (
	"strings"

	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
)

// routeDeltaStreamRouting returns the typed StreamRouting produced for live
// route-delta application: origin=route_delta, webrtc_mode=server_managed.
func routeDeltaStreamRouting() *iov1.StreamRouting {
	return &iov1.StreamRouting{
		Origin:     iov1.StreamOrigin_STREAM_ORIGIN_ROUTE_DELTA,
		WebrtcMode: iov1.WebRTCMode_WEB_RTC_MODE_SERVER_MANAGED,
	}
}

// streamOriginFromString maps a legacy `origin` metadata value to the typed
// enum. Unknown or empty values map to STREAM_ORIGIN_UNSPECIFIED.
func streamOriginFromString(s string) iov1.StreamOrigin {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "route_delta":
		return iov1.StreamOrigin_STREAM_ORIGIN_ROUTE_DELTA
	case "restore":
		return iov1.StreamOrigin_STREAM_ORIGIN_RESTORE
	default:
		return iov1.StreamOrigin_STREAM_ORIGIN_UNSPECIFIED
	}
}

// webRTCModeFromString maps a legacy `webrtc_mode` metadata value to the
// typed enum. Unknown or empty values map to WEB_RTC_MODE_UNSPECIFIED.
func webRTCModeFromString(s string) iov1.WebRTCMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "server_managed":
		return iov1.WebRTCMode_WEB_RTC_MODE_SERVER_MANAGED
	case "peer_managed":
		return iov1.WebRTCMode_WEB_RTC_MODE_PEER_MANAGED
	default:
		return iov1.WebRTCMode_WEB_RTC_MODE_UNSPECIFIED
	}
}

// streamRoutingFromMetadata builds a typed StreamRouting from the legacy
// metadata map keys `origin` and `webrtc_mode`. Returns nil when neither key
// is present, so callers do not emit an empty typed routing message on the
// wire.
func streamRoutingFromMetadata(metadata map[string]string) *iov1.StreamRouting {
	if len(metadata) == 0 {
		return nil
	}
	originStr, hasOrigin := metadata["origin"]
	modeStr, hasMode := metadata["webrtc_mode"]
	if !hasOrigin && !hasMode {
		return nil
	}
	return &iov1.StreamRouting{
		Origin:     streamOriginFromString(originStr),
		WebrtcMode: webRTCModeFromString(modeStr),
	}
}

func mergeLegacyRoutingMetadata(metadata map[string]string, routing *iov1.StreamRouting) map[string]string {
	out := copyMediaStringMap(metadata)
	if out == nil {
		out = map[string]string{}
	}
	if routing == nil {
		return out
	}
	switch routing.GetOrigin() {
	case iov1.StreamOrigin_STREAM_ORIGIN_UNSPECIFIED:
		// no legacy key emitted
	case iov1.StreamOrigin_STREAM_ORIGIN_ROUTE_DELTA:
		out["origin"] = "route_delta"
	case iov1.StreamOrigin_STREAM_ORIGIN_RESTORE:
		out["origin"] = "restore"
	}
	if mode := webRTCModeStringFromEnum(routing.GetWebrtcMode()); mode != "" {
		out["webrtc_mode"] = mode
	}
	return out
}

// webRTCModeStringFromEnum returns the legacy `webrtc_mode` token for a
// typed enum. UNSPECIFIED returns the empty string so callers can fall back
// to a legacy metadata lookup.
func webRTCModeStringFromEnum(mode iov1.WebRTCMode) string {
	switch mode {
	case iov1.WebRTCMode_WEB_RTC_MODE_SERVER_MANAGED:
		return "server_managed"
	case iov1.WebRTCMode_WEB_RTC_MODE_PEER_MANAGED:
		return "peer_managed"
	case iov1.WebRTCMode_WEB_RTC_MODE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}
