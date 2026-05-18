package usecasevalidation

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
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
	if frame.Descriptor != nil && frame.Descriptor.Type != "" {
		renderDescriptor(img, *frame.Descriptor, img.Bounds())
	}
	return img
}

// extractLatestDescriptor finds the most recent SetUI message in msgs and converts
// its root Node to a ui.Descriptor for rendering. Returns nil if no SetUI is found.
func extractLatestDescriptor(msgs []transport.ProtoServerEnvelope) *ui.Descriptor {
	for i := len(msgs) - 1; i >= 0; i-- {
		resp, ok := msgs[i].(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if setUI := resp.GetSetUi(); setUI != nil {
			d := nodeToDescriptor(setUI.GetRoot())
			return &d
		}
	}
	return nil
}

// nodeToDescriptor converts a proto uiv1.Node tree to a ui.Descriptor tree.
func nodeToDescriptor(n *uiv1.Node) ui.Descriptor {
	if n == nil {
		return ui.Descriptor{}
	}
	name := nodeWidgetName(n)
	props := make(map[string]string, len(n.GetProps())+4)
	for k, v := range n.GetProps() {
		props[k] = v
	}
	switch name {
	case "text":
		if t := n.GetText(); t != nil {
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
	case "button":
		if b := n.GetButton(); b != nil {
			if l := b.GetLabel(); l != "" {
				props["label"] = l
			}
			if a := b.GetAction(); a != "" {
				props["action"] = a
			}
		}
	case "image":
		if img := n.GetImage(); img != nil {
			if u := img.GetUrl(); u != "" {
				props["url"] = u
			}
		}
	case "video_surface":
		if v := n.GetVideoSurface(); v != nil {
			if id := v.GetTrackId(); id != "" {
				props["track_id"] = id
			}
		}
	case "audio_visualizer":
		if a := n.GetAudioVisualizer(); a != nil {
			if id := a.GetStreamId(); id != "" {
				props["stream_id"] = id
			}
		}
	case "grid":
		if g := n.GetGrid(); g != nil {
			if c := g.GetColumns(); c > 0 {
				props["columns"] = strconv.Itoa(int(c))
			}
		}
	}
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

const descPad = 8

// renderDescriptor paints a ui.Descriptor tree into img within the given bounds,
// following the same layout rules as the web and Flutter clients.
func renderDescriptor(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	if d.Type == "" || bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return
	}
	switch d.Type {
	case "stack", "center", "padding", "fullscreen", "keep_awake", "brightness":
		renderStack(img, d, bounds)
	case "row":
		renderRow(img, d, bounds)
	case "grid":
		renderGrid(img, d, bounds)
	case "overlay":
		renderOverlay(img, d, bounds)
	case "expand", "scroll":
		renderStack(img, d, bounds)
	case "text":
		renderText(img, d, bounds)
	case "button":
		renderButton(img, d, bounds)
	case "image":
		renderImagePlaceholder(img, d, bounds)
	case "video_surface", "audio_visualizer":
		renderMediaPlaceholder(img, d, bounds)
	default:
		if bg := d.Props["background"]; bg != "" {
			fillRect(img, bounds, parseHexColorRGBA(bg))
		}
		for _, child := range d.Children {
			renderDescriptor(img, child, bounds)
		}
	}
}

func renderStack(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	if bg := d.Props["background"]; bg != "" {
		fillRect(img, bounds, parseHexColorRGBA(bg))
	}
	children := d.Children
	if len(children) == 0 {
		return
	}
	inner := image.Rect(
		bounds.Min.X+descPad,
		bounds.Min.Y+descPad,
		bounds.Max.X-descPad,
		bounds.Max.Y-descPad,
	)
	if inner.Dx() <= 0 || inner.Dy() <= 0 {
		return
	}
	heights := make([]int, len(children))
	totalFixed, expandCount := 0, 0
	for i, child := range children {
		if child.Type == "expand" {
			expandCount++
			heights[i] = -1
		} else {
			h := estimateDescHeight(child, inner.Dx())
			heights[i] = h
			totalFixed += h
		}
	}
	const gap = 8
	gapTotal := gap * (len(children) - 1)
	expandH := 20
	if expandCount > 0 {
		if remaining := inner.Dy() - totalFixed - gapTotal; remaining > 0 {
			expandH = remaining / expandCount
		}
	}
	y := inner.Min.Y
	for i, child := range children {
		h := heights[i]
		if h < 0 {
			h = expandH
		}
		childBounds := image.Rect(inner.Min.X, y, inner.Max.X, min(y+h, bounds.Max.Y-descPad))
		if childBounds.Dy() > 0 {
			renderDescriptor(img, child, childBounds)
		}
		y += h + gap
		if y >= bounds.Max.Y-descPad {
			break
		}
	}
}

func renderRow(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	if bg := d.Props["background"]; bg != "" {
		fillRect(img, bounds, parseHexColorRGBA(bg))
	}
	children := d.Children
	if len(children) == 0 {
		return
	}
	childW := bounds.Dx() / len(children)
	x := bounds.Min.X
	for _, child := range children {
		renderDescriptor(img, child, image.Rect(x, bounds.Min.Y, x+childW, bounds.Max.Y))
		x += childW
	}
}

func renderGrid(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	if bg := d.Props["background"]; bg != "" {
		fillRect(img, bounds, parseHexColorRGBA(bg))
	}
	cols, _ := strconv.Atoi(d.Props["columns"])
	if cols < 1 {
		cols = 1
	}
	children := d.Children
	if len(children) == 0 {
		return
	}
	rows := (len(children) + cols - 1) / cols
	if rows == 0 {
		return
	}
	cellW := bounds.Dx() / cols
	cellH := bounds.Dy() / rows
	for i, child := range children {
		col := i % cols
		row := i / cols
		x := bounds.Min.X + col*cellW
		y := bounds.Min.Y + row*cellH
		renderDescriptor(img, child, image.Rect(x, y, x+cellW, y+cellH))
	}
}

func renderOverlay(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	// All children occupy the same bounds, rendered in order (later = on top).
	for _, child := range d.Children {
		renderDescriptor(img, child, bounds)
	}
}

func renderText(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	text := d.Props["value"]
	if text == "" || bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return
	}
	c := parseHexColorRGBA(d.Props["color"])
	if c == (color.RGBA{}) {
		c = color.RGBA{R: 238, G: 246, B: 247, A: 255}
	}
	style := d.Props["style"]
	lineH := descTextLineHeight(style)
	maxChars := bounds.Dx() / 12
	if maxChars < 1 {
		maxChars = 1
	}
	y := bounds.Min.Y
	for _, line := range wrapText(text, maxChars) {
		if y+lineH > bounds.Max.Y {
			break
		}
		drawText(img, bounds.Min.X, y, line, c)
		y += lineH + 2
	}
}

func renderButton(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	btnBg := color.RGBA{R: 37, G: 90, B: 99, A: 255}
	btnBorder := color.RGBA{R: 80, G: 150, B: 160, A: 255}
	fillRect(img, bounds, btnBg)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		img.SetRGBA(x, bounds.Min.Y, btnBorder)
		img.SetRGBA(x, bounds.Max.Y-1, btnBorder)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		img.SetRGBA(bounds.Min.X, y, btnBorder)
		img.SetRGBA(bounds.Max.X-1, y, btnBorder)
	}
	label := d.Props["label"]
	if label == "" {
		return
	}
	labelW := len(label) * 12
	textX := bounds.Min.X + (bounds.Dx()-labelW)/2
	textY := bounds.Min.Y + (bounds.Dy()-14)/2
	if textX < bounds.Min.X+2 {
		textX = bounds.Min.X + 2
	}
	if textY < bounds.Min.Y+2 {
		textY = bounds.Min.Y + 2
	}
	drawText(img, textX, textY, label, color.RGBA{R: 238, G: 246, B: 247, A: 255})
}

func renderImagePlaceholder(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	fillRect(img, bounds, color.RGBA{R: 40, G: 40, B: 40, A: 255})
	label := "[image]"
	if u := d.Props["url"]; u != "" {
		if len(u) > 30 {
			u = u[:27] + "..."
		}
		label = "[image: " + u + "]"
	}
	drawText(img, bounds.Min.X+4, bounds.Min.Y+4, label, color.RGBA{R: 160, G: 160, B: 160, A: 255})
}

func renderMediaPlaceholder(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	fillRect(img, bounds, color.RGBA{R: 20, G: 20, B: 20, A: 255})
	id := d.Props["track_id"]
	if id == "" {
		id = d.Props["stream_id"]
	}
	label := "[" + d.Type + ": " + id + "]"
	if len(label) > 40 {
		label = label[:37] + "...]"
	}
	drawText(img, bounds.Min.X+4, bounds.Min.Y+4, label, color.RGBA{R: 100, G: 100, B: 100, A: 255})
}

func estimateDescHeight(d ui.Descriptor, width int) int {
	switch d.Type {
	case "text":
		if width <= 0 {
			return 20
		}
		maxChars := width / 12
		if maxChars < 1 {
			maxChars = 1
		}
		lines := wrapText(d.Props["value"], maxChars)
		if len(lines) == 0 {
			lines = []string{""}
		}
		return len(lines)*(descTextLineHeight(d.Props["style"])+2) + 4
	case "button":
		return 32
	case "image":
		return 80
	case "video_surface", "audio_visualizer":
		return 120
	case "expand":
		return -1
	case "overlay":
		maxH := 0
		for _, child := range d.Children {
			if h := estimateDescHeight(child, width); h > maxH {
				maxH = h
			}
		}
		return maxH
	case "row":
		childW := width / max(1, len(d.Children))
		maxH := 0
		for _, child := range d.Children {
			if h := estimateDescHeight(child, childW); h > maxH {
				maxH = h
			}
		}
		return maxH + 2*descPad
	case "grid":
		cols, _ := strconv.Atoi(d.Props["columns"])
		if cols < 1 {
			cols = 1
		}
		cellW := width / cols
		maxCellH := 0
		for _, child := range d.Children {
			if h := estimateDescHeight(child, cellW); h > maxCellH {
				maxCellH = h
			}
		}
		rows := (len(d.Children) + cols - 1) / cols
		return rows*maxCellH + 2*descPad
	default:
		total := 2 * descPad
		for _, child := range d.Children {
			h := estimateDescHeight(child, width-2*descPad)
			if h < 0 {
				h = 40
			}
			total += h + 8
		}
		if total == 2*descPad {
			return 20
		}
		return total
	}
}

func descTextLineHeight(style string) int {
	switch style {
	case "headline", "title":
		return 28
	default:
		return 16
	}
}

func parseHexColorRGBA(hex string) color.RGBA {
	hex = strings.TrimSpace(hex)
	if !strings.HasPrefix(hex, "#") {
		return color.RGBA{}
	}
	hex = hex[1:]
	var r, g, b, a uint8 = 0, 0, 0, 255
	switch len(hex) {
	case 6:
		n, err := strconv.ParseUint(hex, 16, 32)
		if err == nil {
			r, g, b = uint8(n>>16), uint8(n>>8), uint8(n)
		}
	case 8:
		n, err := strconv.ParseUint(hex, 16, 32)
		if err == nil {
			r, g, b, a = uint8(n>>24), uint8(n>>16), uint8(n>>8), uint8(n)
		}
	case 3:
		n, err := strconv.ParseUint(hex, 16, 16)
		if err == nil {
			r = uint8((n>>8)&0xF) * 17
			g = uint8((n>>4)&0xF) * 17
			b = uint8(n&0xF) * 17
		}
	}
	return color.RGBA{R: r, G: g, B: b, A: a}
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
		drawGlyph(img, x+i*12, y, byte(r), c)
	}
}

// drawGlyph renders one character from the embedded 5×7 bitmap font.
// Each glyph is scaled 2× in both axes (10×14 px on screen).
// Bit layout per row: bit 4 = leftmost column, bit 0 = rightmost.
func drawGlyph(img *image.RGBA, x, y int, ch byte, c color.RGBA) {
	if ch < 32 || ch > 126 {
		ch = '?'
	}
	glyph := font5x7[ch]
	for row := 0; row < 7; row++ {
		bits := glyph[row]
		for col := 0; col < 5; col++ {
			if (bits>>uint(4-col))&1 == 0 {
				continue
			}
			fillRect(img, image.Rect(x+col*2, y+row*2, x+col*2+2, y+row*2+2), c)
		}
	}
}

// font5x7 is a public-domain 5×7 bitmap font covering printable ASCII (32–126).
// Each entry holds 7 rows; within a row bit 4 is the leftmost column and bit 0
// is the rightmost. Characters outside this range are rendered as '?'.
var font5x7 = [128][7]byte{
	' ':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	'!':  {0x04, 0x04, 0x04, 0x04, 0x00, 0x04, 0x00},
	'"':  {0x0A, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00},
	'#':  {0x0A, 0x0A, 0x1F, 0x0A, 0x1F, 0x0A, 0x0A},
	'$':  {0x04, 0x0E, 0x14, 0x0E, 0x05, 0x0E, 0x04},
	'%':  {0x18, 0x19, 0x02, 0x04, 0x08, 0x13, 0x03},
	'&':  {0x0C, 0x12, 0x14, 0x08, 0x15, 0x12, 0x0D},
	'\'': {0x04, 0x04, 0x08, 0x00, 0x00, 0x00, 0x00},
	'(':  {0x02, 0x04, 0x08, 0x08, 0x08, 0x04, 0x02},
	')':  {0x08, 0x04, 0x02, 0x02, 0x02, 0x04, 0x08},
	'*':  {0x00, 0x04, 0x15, 0x0E, 0x15, 0x04, 0x00},
	'+':  {0x00, 0x04, 0x04, 0x1F, 0x04, 0x04, 0x00},
	',':  {0x00, 0x00, 0x00, 0x00, 0x06, 0x04, 0x08},
	'-':  {0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00},
	'.':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00},
	'/':  {0x01, 0x02, 0x02, 0x04, 0x08, 0x08, 0x10},
	'0':  {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1':  {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2':  {0x0E, 0x11, 0x01, 0x02, 0x04, 0x08, 0x1F},
	'3':  {0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	'4':  {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5':  {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6':  {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7':  {0x1F, 0x01, 0x02, 0x02, 0x04, 0x04, 0x04},
	'8':  {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9':  {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	':':  {0x00, 0x04, 0x00, 0x00, 0x04, 0x00, 0x00},
	';':  {0x00, 0x04, 0x00, 0x00, 0x06, 0x04, 0x08},
	'<':  {0x02, 0x04, 0x08, 0x10, 0x08, 0x04, 0x02},
	'=':  {0x00, 0x00, 0x1F, 0x00, 0x1F, 0x00, 0x00},
	'>':  {0x08, 0x04, 0x02, 0x01, 0x02, 0x04, 0x08},
	'?':  {0x0E, 0x11, 0x01, 0x06, 0x04, 0x00, 0x04},
	'@':  {0x0E, 0x11, 0x17, 0x15, 0x17, 0x10, 0x0E},
	'A':  {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B':  {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C':  {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D':  {0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	'E':  {0x1F, 0x10, 0x10, 0x1C, 0x10, 0x10, 0x1F},
	'F':  {0x1F, 0x10, 0x10, 0x1C, 0x10, 0x10, 0x10},
	'G':  {0x0E, 0x11, 0x10, 0x10, 0x13, 0x11, 0x0E},
	'H':  {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I':  {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'J':  {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K':  {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L':  {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M':  {0x11, 0x1B, 0x15, 0x11, 0x11, 0x11, 0x11},
	'N':  {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'O':  {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P':  {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q':  {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R':  {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S':  {0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
	'T':  {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U':  {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V':  {0x11, 0x11, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'W':  {0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	'X':  {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y':  {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z':  {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	'[':  {0x0C, 0x08, 0x08, 0x08, 0x08, 0x08, 0x0C},
	'\\': {0x10, 0x10, 0x08, 0x04, 0x02, 0x01, 0x01},
	']':  {0x06, 0x02, 0x02, 0x02, 0x02, 0x02, 0x06},
	'^':  {0x04, 0x0A, 0x11, 0x00, 0x00, 0x00, 0x00},
	'_':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F},
	'`':  {0x08, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
	'a':  {0x00, 0x00, 0x0E, 0x01, 0x0F, 0x11, 0x0F},
	'b':  {0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x1E},
	'c':  {0x00, 0x00, 0x0E, 0x10, 0x10, 0x10, 0x0E},
	'd':  {0x01, 0x01, 0x0F, 0x11, 0x11, 0x11, 0x0F},
	'e':  {0x00, 0x00, 0x0E, 0x11, 0x1F, 0x10, 0x0E},
	'f':  {0x03, 0x04, 0x0E, 0x04, 0x04, 0x04, 0x04},
	'g':  {0x00, 0x0E, 0x11, 0x11, 0x0F, 0x01, 0x0E},
	'h':  {0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x11},
	'i':  {0x04, 0x00, 0x0C, 0x04, 0x04, 0x04, 0x0E},
	'j':  {0x02, 0x00, 0x06, 0x02, 0x02, 0x12, 0x0C},
	'k':  {0x10, 0x10, 0x12, 0x14, 0x18, 0x14, 0x12},
	'l':  {0x0C, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'm':  {0x00, 0x00, 0x1A, 0x15, 0x15, 0x11, 0x11},
	'n':  {0x00, 0x00, 0x1E, 0x11, 0x11, 0x11, 0x11},
	'o':  {0x00, 0x00, 0x0E, 0x11, 0x11, 0x11, 0x0E},
	'p':  {0x00, 0x00, 0x1E, 0x11, 0x11, 0x1E, 0x10},
	'q':  {0x00, 0x00, 0x0F, 0x11, 0x11, 0x0F, 0x01},
	'r':  {0x00, 0x00, 0x16, 0x18, 0x10, 0x10, 0x10},
	's':  {0x00, 0x00, 0x0E, 0x10, 0x0E, 0x01, 0x0E},
	't':  {0x04, 0x04, 0x1E, 0x04, 0x04, 0x04, 0x04},
	'u':  {0x00, 0x00, 0x11, 0x11, 0x11, 0x11, 0x0F},
	'v':  {0x00, 0x00, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'w':  {0x00, 0x00, 0x11, 0x11, 0x15, 0x15, 0x0A},
	'x':  {0x00, 0x00, 0x11, 0x0A, 0x04, 0x0A, 0x11},
	'y':  {0x00, 0x00, 0x11, 0x11, 0x0F, 0x01, 0x0E},
	'z':  {0x00, 0x00, 0x1F, 0x02, 0x04, 0x08, 0x1F},
	'{':  {0x06, 0x04, 0x04, 0x18, 0x04, 0x04, 0x06},
	'|':  {0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'}':  {0x0C, 0x04, 0x04, 0x03, 0x04, 0x04, 0x0C},
	'~':  {0x00, 0x00, 0x08, 0x15, 0x02, 0x00, 0x00},
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
