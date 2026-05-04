package transport

import (
	"strconv"
	"strings"

	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
)

func streamAudioMetadataFromLegacy(metadata map[string]string) *iov1.StreamAudioMetadata {
	if len(metadata) == 0 {
		return nil
	}
	var typed iov1.StreamAudioMetadata
	if sampleRate := parseLegacyPositiveUint32(metadata["sample_rate"]); sampleRate > 0 {
		typed.SampleRate = sampleRate
	}
	if channels := parseLegacyPositiveUint32(metadata["channels"]); channels > 0 {
		typed.Channels = channels
	}
	if codec := strings.TrimSpace(metadata["codec"]); codec != "" {
		typed.Codec = codec
	}
	if typed.SampleRate == 0 && typed.Channels == 0 && typed.Codec == "" {
		return nil
	}
	return &typed
}

func mergeLegacyAudioMetadata(metadata map[string]string, audio *iov1.StreamAudioMetadata) map[string]string {
	out := copyMediaStringMap(metadata)
	if out == nil {
		out = map[string]string{}
	}
	if audio == nil {
		return out
	}
	if audio.GetSampleRate() > 0 {
		out["sample_rate"] = strconv.FormatUint(uint64(audio.GetSampleRate()), 10)
	}
	if audio.GetChannels() > 0 {
		out["channels"] = strconv.FormatUint(uint64(audio.GetChannels()), 10)
	}
	if codec := strings.TrimSpace(audio.GetCodec()); codec != "" {
		out["codec"] = codec
	}
	return out
}

func parseLegacyPositiveUint32(raw string) uint32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || v == 0 {
		return 0
	}
	return uint32(v)
}
