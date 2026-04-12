package ui

// Descriptor is a generic server-driven UI node.
type Descriptor struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type"`
	Props    map[string]string `json:"props,omitempty"`
	Children []Descriptor      `json:"children,omitempty"`
}

// New builds a descriptor node with type and optional props.
func New(nodeType string, props map[string]string, children ...Descriptor) Descriptor {
	if props == nil {
		props = map[string]string{}
	}
	return Descriptor{
		Type:     nodeType,
		Props:    props,
		Children: children,
	}
}

// HelloWorld returns a minimal initial screen used at connect time.
func HelloWorld(deviceName string) Descriptor {
	return New("stack", map[string]string{
		"background": "#101418",
	}, New("text", map[string]string{
		"value": "Connected: " + deviceName,
		"style": "headline",
		"color": "#E7F0F7",
	}))
}
