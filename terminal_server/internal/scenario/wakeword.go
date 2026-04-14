package scenario

import (
	"context"
	"strings"
)

// PrefixWakeWordDetector activates commands that start with configured wake-word prefixes.
type PrefixWakeWordDetector struct {
	Prefixes []string
}

// Detect reports whether spoken text starts with any configured wake-word prefix.
func (d PrefixWakeWordDetector) Detect(_ context.Context, spoken string) (WakeWordDetection, error) {
	normalized := strings.TrimSpace(strings.ToLower(spoken))
	if normalized == "" {
		return WakeWordDetection{}, nil
	}

	for _, prefix := range d.Prefixes {
		prefix = strings.TrimSpace(strings.ToLower(prefix))
		if prefix == "" {
			continue
		}
		if normalized == prefix {
			return WakeWordDetection{Detected: true}, nil
		}
		if strings.HasPrefix(normalized, prefix+" ") {
			return WakeWordDetection{
				Detected: true,
				Command:  strings.TrimSpace(strings.TrimPrefix(normalized, prefix+" ")),
			}, nil
		}
	}

	return WakeWordDetection{}, nil
}
