package ui

import "fmt"

var supportedNodeTypes = map[string]struct{}{
	"stack":            {},
	"row":              {},
	"grid":             {},
	"scroll":           {},
	"padding":          {},
	"center":           {},
	"expand":           {},
	"text":             {},
	"image":            {},
	"video_surface":    {},
	"audio_visualizer": {},
	"canvas":           {},
	"text_input":       {},
	"button":           {},
	"slider":           {},
	"toggle":           {},
	"dropdown":         {},
	"gesture_area":     {},
	"notification":     {},
	"overlay":          {},
	"progress":         {},
	"fullscreen":       {},
	"keep_awake":       {},
	"brightness":       {},
}

// Validate ensures the descriptor tree only uses supported node types.
func Validate(d Descriptor) error {
	return validateNode(d, "root")
}

func validateNode(d Descriptor, path string) error {
	if _, ok := supportedNodeTypes[d.Type]; !ok {
		return fmt.Errorf("unsupported node type %q at %s", d.Type, path)
	}
	for i := range d.Children {
		childPath := fmt.Sprintf("%s.children[%d]", path, i)
		if err := validateNode(d.Children[i], childPath); err != nil {
			return err
		}
	}
	return nil
}
