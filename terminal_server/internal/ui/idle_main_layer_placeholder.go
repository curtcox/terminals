package ui

// IdleMainLayerPlaceholder returns the canonical main-layer UI tree shown by
// thin clients after control registration until the first server SetUI
// arrives (plans/features/terminal-ui/plan.md, Phase H). The shape is mirrored
// on the Flutter client for display-surface mode; keep wire bytes in sync
// with transport/testdata/idle_main_layer_placeholder_root.pb (see
// TestIdleMainLayerPlaceholderGoldenWire in internal/transport).
func IdleMainLayerPlaceholder() Descriptor {
	return Descriptor{
		Type: "stack",
		ID:   "__runtime.main_placeholder.root",
		Props: map[string]string{
			"client_chrome": "hidden",
			"background":    "#101418",
		},
		Children: []Descriptor{
			{
				Type: "text",
				ID:   "__runtime.main_placeholder.message",
				Props: map[string]string{
					"value": "Awaiting server UI",
					"style": "headline",
					"color": "#E7F0F7",
				},
			},
		},
	}
}
