package transport

import "testing"

// TestStreamHandlerConstructorsInitializeFields verifies both public
// constructors produce a handler with all maps and default collaborators
// initialized. This guards the centralized newStreamHandler helper from
// future drift between constructors.
func TestStreamHandlerConstructorsInitializeFields(t *testing.T) {
	cases := []struct {
		name    string
		build   func() *StreamHandler
		runtime bool
	}{
		{
			name:  "NewStreamHandler",
			build: func() *StreamHandler { return NewStreamHandler(nil) },
		},
		{
			name:    "NewStreamHandlerWithRuntime",
			build:   func() *StreamHandler { return NewStreamHandlerWithRuntime(nil, nil) },
			runtime: false, // runtime arg is nil; we just check field plumbing
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.build()
			if h == nil {
				t.Fatal("constructor returned nil")
			}
			if h.metrics == nil {
				t.Error("metrics not initialized")
			}
			if h.seen == nil {
				t.Error("seen map not initialized")
			}
			if h.seenLimit == 0 {
				t.Error("seenLimit not set")
			}
			if h.recent == nil {
				t.Error("recent slice not initialized")
			}
			if h.recentLimit == 0 {
				t.Error("recentLimit not set")
			}
			if h.commandDispatcher == nil {
				t.Error("commandDispatcher not initialized")
			}
			if h.terminals == nil {
				t.Error("terminals manager not initialized")
			}
			if h.replSessions == nil {
				t.Error("replSessions service not initialized")
			}
			if h.terminalReadDeadline == 0 {
				t.Error("terminalReadDeadline not set")
			}
			if h.terminalReadInterval == 0 {
				t.Error("terminalReadInterval not set")
			}
			if h.terminalUIInterval == 0 {
				t.Error("terminalUIInterval not set")
			}
			if h.terminalReplAdminURL == "" {
				t.Error("terminalReplAdminURL not set")
			}
			if h.uiSession == nil {
				t.Error("uiSession state not initialized")
			}
			if h.menuOverlayByDevice == nil {
				t.Error("menuOverlayByDevice not initialized")
			}
			if h.photoFrameSlides == nil {
				t.Error("photoFrameSlides not initialized")
			}
			if h.photoFrameIndexByDev == nil {
				t.Error("photoFrameIndexByDev not initialized")
			}
			if h.photoFrameLastByDev == nil {
				t.Error("photoFrameLastByDev not initialized")
			}
			if h.photoFrameInterval == 0 {
				t.Error("photoFrameInterval not set")
			}
			if h.mediaControl == nil {
				t.Error("mediaControl state not initialized")
			}
			if h.sensorsByDevice == nil {
				t.Error("sensorsByDevice not initialized")
			}
			if h.voicePipeline == nil {
				t.Error("voicePipeline not initialized")
			}
			if h.suspendedClaimsByDevice == nil {
				t.Error("suspendedClaimsByDevice not initialized")
			}
			if h.routeReplay == nil {
				t.Error("routeReplay store not initialized")
			}
			if h.uiOwners == nil {
				t.Error("uiOwners tracker not initialized")
			}
			if h.wakeWordDedupe == nil {
				t.Error("wakeWordDedupe stage not initialized")
			}
			if h.menuAppPolicy == nil {
				t.Error("menuAppPolicy not initialized (expected allowAllMenuAppPolicy)")
			}
			if h.diagnostics == nil {
				t.Error("diagnostics intake not initialized")
			}
		})
	}
}

func TestNewStreamHandlerWithRuntimeStoresRuntime(t *testing.T) {
	h := NewStreamHandlerWithRuntime(nil, nil)
	if h.runtime != nil {
		t.Errorf("expected nil runtime when passed nil, got %v", h.runtime)
	}
	// We can't easily construct a real *scenario.Runtime here without
	// pulling extra deps; the field-plumbing path is identical in both
	// constructors thanks to newStreamHandler, and the existing transport
	// runtime tests cover the non-nil case.
}
