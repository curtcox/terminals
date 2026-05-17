package usecasevalidation

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func renderFrameImage(frame FrameRecord) image.Image {
	const width = 960
	const height = 540
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, img.Bounds(), color.RGBA{R: 16, G: 20, B: 24, A: 255})
	fillRect(img, image.Rect(0, 0, width, 72), color.RGBA{R: 37, G: 90, B: 99, A: 255})
	drawText(img, 32, 28, "Terminals validation frame", color.RGBA{R: 238, G: 246, B: 247, A: 255})
	drawText(img, 32, 94, "Step: "+frame.StepID, color.RGBA{R: 226, G: 232, B: 234, A: 255})
	if frame.Terminal != "" {
		drawText(img, 32, 122, "Terminal: "+frame.Terminal, color.RGBA{R: 190, G: 203, B: 207, A: 255})
	}
	y := 166
	for _, line := range wrapText(frame.Summary, 82) {
		drawText(img, 32, y, line, color.RGBA{R: 238, G: 238, B: 232, A: 255})
		y += 28
		if y > height-36 {
			break
		}
	}
	return img
}

func summarizeLatestUI(messages []transport.ProtoServerEnvelope) string {
	for i := len(messages) - 1; i >= 0; i-- {
		resp, ok := messages[i].(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if setUI := resp.GetSetUi(); setUI != nil {
			return "Set UI: " + summarizeNode(setUI.GetRoot())
		}
		if updateUI := resp.GetUpdateUi(); updateUI != nil {
			return "Update UI " + updateUI.GetComponentId() + ": " + summarizeNode(updateUI.GetNode())
		}
		if notification := resp.GetNotification(); notification != nil {
			return "Notification: " + notification.GetTitle() + " - " + notification.GetBody()
		}
	}
	return "No server-driven UI message was observed before this assertion."
}

func summarizeNode(node *uiv1.Node) string {
	if node == nil {
		return "(empty)"
	}
	parts := []string{nodeWidgetName(node)}
	if node.GetId() != "" {
		parts = append(parts, "#"+node.GetId())
	}
	if text := node.GetText(); text != nil && text.GetValue() != "" {
		parts = append(parts, quoteSummary(text.GetValue()))
	}
	if button := node.GetButton(); button != nil && button.GetLabel() != "" {
		parts = append(parts, "button "+quoteSummary(button.GetLabel()))
	}
	props := node.GetProps()
	if len(props) > 0 {
		keys := make([]string, 0, len(props))
		for key := range props {
			if key == "id" || key == "draw_ops_json" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if props[key] != "" {
				parts = append(parts, key+"="+quoteSummary(props[key]))
			}
		}
	}
	children := node.GetChildren()
	if len(children) > 0 {
		childSummaries := make([]string, 0, min(len(children), 4))
		for _, child := range children {
			if len(childSummaries) == 4 {
				break
			}
			childSummaries = append(childSummaries, summarizeNode(child))
		}
		if len(children) > 4 {
			childSummaries = append(childSummaries, fmt.Sprintf("%d more", len(children)-4))
		}
		parts = append(parts, "children=["+strings.Join(childSummaries, "; ")+"]")
	}
	return strings.Join(parts, " ")
}

func nodeWidgetName(node *uiv1.Node) string {
	switch {
	case node.GetStack() != nil:
		return "stack"
	case node.GetRow() != nil:
		return "row"
	case node.GetGrid() != nil:
		return "grid"
	case node.GetScroll() != nil:
		return "scroll"
	case node.GetPadding() != nil:
		return "padding"
	case node.GetCenter() != nil:
		return "center"
	case node.GetExpand() != nil:
		return "expand"
	case node.GetText() != nil:
		return "text"
	case node.GetImage() != nil:
		return "image"
	case node.GetVideoSurface() != nil:
		return "video_surface"
	case node.GetAudioVisualizer() != nil:
		return "audio_visualizer"
	case node.GetCanvas() != nil:
		return "canvas"
	case node.GetTextInput() != nil:
		return "text_input"
	case node.GetButton() != nil:
		return "button"
	case node.GetSlider() != nil:
		return "slider"
	case node.GetToggle() != nil:
		return "toggle"
	case node.GetDropdown() != nil:
		return "dropdown"
	case node.GetGestureArea() != nil:
		return "gesture_area"
	case node.GetOverlay() != nil:
		return "overlay"
	case node.GetProgress() != nil:
		return "progress"
	case node.GetFullscreen() != nil:
		return "fullscreen"
	case node.GetKeepAwake() != nil:
		return "keep_awake"
	case node.GetBrightness() != nil:
		return "brightness"
	default:
		return "node"
	}
}

// summarizeDescriptor produces a human-readable summary of a ui.Descriptor
// tree for use in host-side frame captures. Mirrors summarizeNode but works
// with the server-side Descriptor type rather than the proto Node type.
func summarizeDescriptor(d ui.Descriptor) string {
	parts := []string{d.Type}
	if d.ID != "" {
		parts = append(parts, "#"+d.ID)
	}
	props := d.Props
	if len(props) > 0 {
		keys := make([]string, 0, len(props))
		for key := range props {
			if key == "id" || key == "draw_ops_json" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if v := props[key]; v != "" {
				parts = append(parts, key+"="+quoteSummary(v))
			}
		}
	}
	if len(d.Children) > 0 {
		limit := min(len(d.Children), 4)
		childParts := make([]string, 0, limit)
		for _, child := range d.Children[:limit] {
			childParts = append(childParts, summarizeDescriptor(child))
		}
		if len(d.Children) > 4 {
			childParts = append(childParts, fmt.Sprintf("%d more", len(d.Children)-4))
		}
		parts = append(parts, "children=["+strings.Join(childParts, "; ")+"]")
	}
	return strings.Join(parts, " ")
}

func quoteSummary(value string) string {
	if len(value) > 80 {
		value = value[:77] + "..."
	}
	return fmt.Sprintf("%q", value)
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawText(img *image.RGBA, x, y int, text string, c color.RGBA) {
	for i, r := range text {
		if r < 32 || r > 126 {
			r = '?'
		}
		drawGlyph(img, x+i*8, y, byte(r), c)
	}
}

func drawGlyph(img *image.RGBA, x, y int, ch byte, c color.RGBA) {
	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if ((int(ch) >> uint((row+col)%7)) & 1) == 0 {
				continue
			}
			fillRect(img, image.Rect(x+col, y+row*2, x+col+1, y+row*2+2), c)
		}
	}
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{"(no UI summary)"}
	}
	var lines []string
	line := ""
	for _, word := range words {
		if len(line)+len(word)+1 > width && line != "" {
			lines = append(lines, line)
			line = word
			continue
		}
		if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
