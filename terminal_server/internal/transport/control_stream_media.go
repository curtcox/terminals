package transport

import "strings"

func (h *StreamHandler) serverManagedSignalEngine(streamID string) (WebRTCSignalEngine, bool) {
	if h.mediaControl == nil {
		return nil, false
	}
	return h.mediaControl.ServerManagedSignalEngine(streamID)
}

func (h *StreamHandler) peerDeviceForStream(streamID, sourceDeviceID string) string {
	const prefix = "route:"
	if strings.HasPrefix(streamID, prefix) {
		parts := strings.SplitN(strings.TrimPrefix(streamID, prefix), "|", 3)
		if len(parts) == 3 {
			if sourceDeviceID == parts[0] {
				return parts[1]
			}
			if sourceDeviceID == parts[1] {
				return parts[0]
			}
		}
	}

	if h.mediaControl == nil {
		return ""
	}
	return h.mediaControl.PeerDeviceForStream(streamID, sourceDeviceID)
}

func (h *StreamHandler) registerMediaStream(start StartStreamResponse) {
	if h.mediaControl != nil {
		h.mediaControl.RegisterStream(start)
	}
}

func (h *StreamHandler) unregisterMediaStream(streamID string) {
	if h.mediaControl != nil {
		h.mediaControl.UnregisterStream(streamID)
	}
}

func (h *StreamHandler) markStreamReady(streamID string) {
	h.mediaControl.MarkStreamReady(streamID)
}

func (h *StreamHandler) mediaStreamStatusData() map[string]string {
	if h.mediaControl == nil {
		return map[string]string{
			"media_streams_active":  "0",
			"media_streams_ready":   "0",
			"media_streams_pending": "0",
			"media_streams":         "",
		}
	}
	return h.mediaControl.MediaStreamStatusData()
}

func (h *StreamHandler) recordingStatusData() map[string]string {
	if h.mediaControl == nil {
		return map[string]string{
			"recording_active_streams": "0",
			"recording_stream_ids":     "",
		}
	}
	return h.mediaControl.RecordingStatusData()
}
