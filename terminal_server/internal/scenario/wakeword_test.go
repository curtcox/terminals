package scenario

import (
	"context"
	"testing"
)

func TestPrefixWakeWordDetectorDetect(t *testing.T) {
	detector := PrefixWakeWordDetector{Prefixes: []string{"assistant", "hey terminal"}}

	tests := []struct {
		name       string
		spoken     string
		wantDetect bool
		wantCmd    string
	}{
		{name: "prefix with command", spoken: "assistant what is the weather", wantDetect: true, wantCmd: "what is the weather"},
		{name: "alternate prefix with command", spoken: "hey terminal open terminal", wantDetect: true, wantCmd: "open terminal"},
		{name: "exact prefix", spoken: "assistant", wantDetect: true, wantCmd: ""},
		{name: "no prefix", spoken: "what is the weather", wantDetect: false, wantCmd: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detector.Detect(context.Background(), tc.spoken)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}
			if got.Detected != tc.wantDetect {
				t.Fatalf("Detected = %v, want %v", got.Detected, tc.wantDetect)
			}
			if got.Command != tc.wantCmd {
				t.Fatalf("Command = %q, want %q", got.Command, tc.wantCmd)
			}
		})
	}
}
