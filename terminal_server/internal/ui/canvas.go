package ui

import (
	"encoding/json"
	"strconv"
)

// CanvasOpKind identifies which primitive a CanvasOp carries.
type CanvasOpKind int

const (
	// CanvasOpUnspecified is the zero value; CanvasOp instances of this
	// kind are skipped by serializers and the transport adapter.
	CanvasOpUnspecified CanvasOpKind = iota
	CanvasOpLineKind
	CanvasOpRectKind
	CanvasOpCircleKind
	CanvasOpTextKind
	CanvasOpPathKind
)

// CanvasOp is a tagged union of canvas primitives. Exactly one of the typed
// pointer fields is set; Kind records which.
type CanvasOp struct {
	Kind   CanvasOpKind
	Line   *CanvasLine
	Rect   *CanvasRect
	Circle *CanvasCircle
	Text   *CanvasText
	Path   *CanvasPath
}

// CanvasLine is a typed line primitive.
type CanvasLine struct {
	X1, Y1, X2, Y2 float64
	Stroke         string
	StrokeWidth    float64
}

// CanvasRect is a typed rectangle primitive.
type CanvasRect struct {
	X, Y, Width, Height float64
	Fill, Stroke        string
	StrokeWidth         float64
}

// CanvasCircle is a typed circle primitive.
type CanvasCircle struct {
	CX, CY, Radius float64
	Fill, Stroke   string
	StrokeWidth    float64
}

// CanvasText is a typed text primitive.
type CanvasText struct {
	X, Y       float64
	Text       string
	Fill       string
	FontSize   float64
	FontFamily string
}

// CanvasPath is a typed path primitive (SVG-style "d" attribute).
type CanvasPath struct {
	D            string
	Fill, Stroke string
	StrokeWidth  float64
}

// Line builds a typed line CanvasOp.
func Line(x1, y1, x2, y2 float64, stroke string, strokeWidth float64) CanvasOp {
	return CanvasOp{Kind: CanvasOpLineKind, Line: &CanvasLine{
		X1: x1, Y1: y1, X2: x2, Y2: y2, Stroke: stroke, StrokeWidth: strokeWidth,
	}}
}

// Rect builds a typed rectangle CanvasOp.
func Rect(x, y, w, h float64, fill, stroke string, strokeWidth float64) CanvasOp {
	return CanvasOp{Kind: CanvasOpRectKind, Rect: &CanvasRect{
		X: x, Y: y, Width: w, Height: h, Fill: fill, Stroke: stroke, StrokeWidth: strokeWidth,
	}}
}

// Circle builds a typed circle CanvasOp.
func Circle(cx, cy, radius float64, fill, stroke string, strokeWidth float64) CanvasOp {
	return CanvasOp{Kind: CanvasOpCircleKind, Circle: &CanvasCircle{
		CX: cx, CY: cy, Radius: radius, Fill: fill, Stroke: stroke, StrokeWidth: strokeWidth,
	}}
}

// Text builds a typed text CanvasOp.
func Text(x, y float64, text, fill string, fontSize float64, fontFamily string) CanvasOp {
	return CanvasOp{Kind: CanvasOpTextKind, Text: &CanvasText{
		X: x, Y: y, Text: text, Fill: fill, FontSize: fontSize, FontFamily: fontFamily,
	}}
}

// Path builds a typed path CanvasOp.
func Path(d, fill, stroke string, strokeWidth float64) CanvasOp {
	return CanvasOp{Kind: CanvasOpPathKind, Path: &CanvasPath{
		D: d, Fill: fill, Stroke: stroke, StrokeWidth: strokeWidth,
	}}
}

// CanvasNode constructs a canvas Descriptor populated with typed ops. The
// returned descriptor carries the typed slice on Descriptor.CanvasOps for
// the transport adapter and a serialized JSON mirror in Props["draw_ops_json"]
// so legacy consumers continue to render during the compatibility window.
func CanvasNode(id string, ops ...CanvasOp) Descriptor {
	props := map[string]string{}
	if id != "" {
		props["id"] = id
	}
	if legacy := CanvasOpsToJSON(ops); legacy != "" {
		props["draw_ops_json"] = legacy
	}
	typed := append([]CanvasOp(nil), ops...)
	return Descriptor{
		ID:        id,
		Type:      "canvas",
		Props:     props,
		CanvasOps: typed,
	}
}

// CanvasOpsToJSON serializes typed ops into the {"ops":[...]} envelope that
// CanvasWidget.draw_ops_json carries during the compatibility window. Ops
// with CanvasOpUnspecified kind or a nil variant pointer are skipped. Returns
// the empty string if every op was skipped (so callers can omit the legacy
// prop entirely).
func CanvasOpsToJSON(ops []CanvasOp) string {
	envelope := struct {
		Ops []map[string]any `json:"ops"`
	}{Ops: make([]map[string]any, 0, len(ops))}
	for _, op := range ops {
		entry := opToJSONEntry(op)
		if entry == nil {
			continue
		}
		envelope.Ops = append(envelope.Ops, entry)
	}
	if len(envelope.Ops) == 0 {
		return ""
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func opToJSONEntry(op CanvasOp) map[string]any {
	switch op.Kind {
	case CanvasOpLineKind:
		if op.Line == nil {
			return nil
		}
		return map[string]any{"line": map[string]any{
			"x1": op.Line.X1, "y1": op.Line.Y1, "x2": op.Line.X2, "y2": op.Line.Y2,
			"stroke": op.Line.Stroke, "stroke_width": op.Line.StrokeWidth,
		}}
	case CanvasOpRectKind:
		if op.Rect == nil {
			return nil
		}
		return map[string]any{"rect": map[string]any{
			"x": op.Rect.X, "y": op.Rect.Y, "width": op.Rect.Width, "height": op.Rect.Height,
			"fill": op.Rect.Fill, "stroke": op.Rect.Stroke, "stroke_width": op.Rect.StrokeWidth,
		}}
	case CanvasOpCircleKind:
		if op.Circle == nil {
			return nil
		}
		return map[string]any{"circle": map[string]any{
			"cx": op.Circle.CX, "cy": op.Circle.CY, "radius": op.Circle.Radius,
			"fill": op.Circle.Fill, "stroke": op.Circle.Stroke, "stroke_width": op.Circle.StrokeWidth,
		}}
	case CanvasOpTextKind:
		if op.Text == nil {
			return nil
		}
		return map[string]any{"text": map[string]any{
			"x": op.Text.X, "y": op.Text.Y, "text": op.Text.Text, "fill": op.Text.Fill,
			"font_size": op.Text.FontSize, "font_family": op.Text.FontFamily,
		}}
	case CanvasOpPathKind:
		if op.Path == nil {
			return nil
		}
		return map[string]any{"path": map[string]any{
			"d": op.Path.D, "fill": op.Path.Fill, "stroke": op.Path.Stroke,
			"stroke_width": op.Path.StrokeWidth,
		}}
	default:
		return nil
	}
}

// DiagnosticsConnectionPulseComponentID is the stable component id used by
// DiagnosticsConnectionPulseOverlay so the global overlay slot can be
// patched by id.
const DiagnosticsConnectionPulseComponentID = "diagnostics_connection_pulse"

// DiagnosticsConnectionPulseOverlay renders a transient diagnostics overlay
// that visualizes connection health for a single device using a typed canvas
// (the first native typed-canvas producer in the server). When healthy is
// true the indicator dot is green; otherwise it's red. RTT is rendered as a
// short text label inside the canvas. The view is shaped as a global-overlay
// patch so it can be applied via UpdateUI on the GlobalOverlayComponentID
// slot without disturbing the underlying root.
func DiagnosticsConnectionPulseOverlay(deviceID string, healthy bool, rttMs int) Descriptor {
	dotFill := "#33CC66"
	if !healthy {
		dotFill = "#CC3333"
	}
	statusLabel := "OK"
	if !healthy {
		statusLabel = "DOWN"
	}
	rttLabel := strconv.Itoa(rttMs) + "ms"
	canvasID := DiagnosticsConnectionPulseComponentID + "_canvas"
	if deviceID != "" {
		canvasID = canvasID + "_" + deviceID
	}
	return New("overlay", map[string]string{
		"id": GlobalOverlayComponentID,
	}, New("stack", map[string]string{
		"id":         DiagnosticsConnectionPulseComponentID,
		"background": "#0B1622",
	}, New("text", map[string]string{
		"id":    DiagnosticsConnectionPulseComponentID + "_label",
		"value": "Connection: " + statusLabel + " (" + rttLabel + ")",
		"style": "body",
		"color": "#E7F0F7",
	}), CanvasNode(canvasID,
		Rect(0, 0, 96, 24, "#101820", "#26323F", 1),
		Circle(12, 12, 6, dotFill, dotFill, 0),
		Line(24, 12, 88, 12, "#26323F", 1),
		Text(28, 16, rttLabel, "#E7F0F7", 10, "monospace"),
	)))
}
