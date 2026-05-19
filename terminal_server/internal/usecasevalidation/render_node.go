package usecasevalidation

import (
	"strconv"

	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// nodeToDescriptor converts a proto uiv1.Node tree to a ui.Descriptor tree.
func nodeToDescriptor(n *uiv1.Node) ui.Descriptor {
	if n == nil {
		return ui.Descriptor{}
	}
	name := nodeWidgetName(n)
	props := nodeDescriptorProps(name, n)
	children := make([]ui.Descriptor, 0, len(n.GetChildren()))
	for _, child := range n.GetChildren() {
		children = append(children, nodeToDescriptor(child))
	}
	return ui.Descriptor{
		ID:       n.GetId(),
		Type:     name,
		Props:    props,
		Children: children,
	}
}

func nodeDescriptorProps(name string, n *uiv1.Node) map[string]string {
	props := make(map[string]string, len(n.GetProps())+4)
	for k, v := range n.GetProps() {
		props[k] = v
	}
	switch name {
	case "text":
		mergeTextNodeProps(n, props)
	case "button":
		mergeButtonNodeProps(n, props)
	case "image":
		mergeImageNodeProps(n, props)
	case "video_surface":
		mergeVideoSurfaceNodeProps(n, props)
	case "audio_visualizer":
		mergeAudioVisualizerNodeProps(n, props)
	case "grid":
		mergeGridNodeProps(n, props)
	}
	return props
}

func mergeTextNodeProps(n *uiv1.Node, props map[string]string) {
	t := n.GetText()
	if t == nil {
		return
	}
	if v := t.GetValue(); v != "" {
		props["value"] = v
	}
	if c := t.GetColor(); c != "" {
		props["color"] = c
	}
	if s := t.GetStyle(); s != "" {
		props["style"] = s
	}
}

func mergeButtonNodeProps(n *uiv1.Node, props map[string]string) {
	b := n.GetButton()
	if b == nil {
		return
	}
	if l := b.GetLabel(); l != "" {
		props["label"] = l
	}
	if a := b.GetAction(); a != "" {
		props["action"] = a
	}
}

func mergeImageNodeProps(n *uiv1.Node, props map[string]string) {
	img := n.GetImage()
	if img == nil {
		return
	}
	if u := img.GetUrl(); u != "" {
		props["url"] = u
	}
}

func mergeVideoSurfaceNodeProps(n *uiv1.Node, props map[string]string) {
	v := n.GetVideoSurface()
	if v == nil {
		return
	}
	if id := v.GetTrackId(); id != "" {
		props["track_id"] = id
	}
}

func mergeAudioVisualizerNodeProps(n *uiv1.Node, props map[string]string) {
	a := n.GetAudioVisualizer()
	if a == nil {
		return
	}
	if id := a.GetStreamId(); id != "" {
		props["stream_id"] = id
	}
}

func mergeGridNodeProps(n *uiv1.Node, props map[string]string) {
	g := n.GetGrid()
	if g == nil {
		return
	}
	if c := g.GetColumns(); c > 0 {
		props["columns"] = strconv.Itoa(int(c))
	}
}
