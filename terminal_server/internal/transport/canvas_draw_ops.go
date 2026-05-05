package transport

import (
	"encoding/json"
	"strings"

	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type drawOpsEnvelope struct {
	Ops []drawOpJSON `json:"ops"`
}

type drawOpJSON struct {
	Line   *drawLineJSON   `json:"line,omitempty"`
	Rect   *drawRectJSON   `json:"rect,omitempty"`
	Circle *drawCircleJSON `json:"circle,omitempty"`
	Text   *drawTextJSON   `json:"text,omitempty"`
	Path   *drawPathJSON   `json:"path,omitempty"`
}

type drawLineJSON struct {
	X1          float64 `json:"x1"`
	Y1          float64 `json:"y1"`
	X2          float64 `json:"x2"`
	Y2          float64 `json:"y2"`
	Stroke      string  `json:"stroke"`
	StrokeWidth float64 `json:"stroke_width"`
}

type drawRectJSON struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Fill        string  `json:"fill"`
	Stroke      string  `json:"stroke"`
	StrokeWidth float64 `json:"stroke_width"`
}

type drawCircleJSON struct {
	CX          float64 `json:"cx"`
	CY          float64 `json:"cy"`
	Radius      float64 `json:"radius"`
	Fill        string  `json:"fill"`
	Stroke      string  `json:"stroke"`
	StrokeWidth float64 `json:"stroke_width"`
}

type drawTextJSON struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Text       string  `json:"text"`
	Fill       string  `json:"fill"`
	FontSize   float64 `json:"font_size"`
	FontFamily string  `json:"font_family"`
}

type drawPathJSON struct {
	D           string  `json:"d"`
	Fill        string  `json:"fill"`
	Stroke      string  `json:"stroke"`
	StrokeWidth float64 `json:"stroke_width"`
}

// canvasDrawOpsFromJSON parses a CanvasWidget.draw_ops_json string into typed
// DrawOp messages. It returns nil for empty input, malformed JSON, or when no
// op object had exactly one recognized variant. The legacy JSON string is
// always preserved verbatim by the caller; this helper only computes the
// additive typed mirror.
func canvasDrawOpsFromJSON(raw string) []*uiv1.DrawOp {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var env drawOpsEnvelope
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil
	}
	if len(env.Ops) == 0 {
		return nil
	}
	out := make([]*uiv1.DrawOp, 0, len(env.Ops))
	for _, op := range env.Ops {
		converted := drawOpToProto(op)
		if converted == nil {
			continue
		}
		out = append(out, converted)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func drawOpToProto(op drawOpJSON) *uiv1.DrawOp {
	variants := 0
	if op.Line != nil {
		variants++
	}
	if op.Rect != nil {
		variants++
	}
	if op.Circle != nil {
		variants++
	}
	if op.Text != nil {
		variants++
	}
	if op.Path != nil {
		variants++
	}
	if variants != 1 {
		return nil
	}
	switch {
	case op.Line != nil:
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Line{Line: &uiv1.DrawLine{
			X1: op.Line.X1, Y1: op.Line.Y1, X2: op.Line.X2, Y2: op.Line.Y2,
			Stroke: op.Line.Stroke, StrokeWidth: op.Line.StrokeWidth,
		}}}
	case op.Rect != nil:
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Rect{Rect: &uiv1.DrawRect{
			X: op.Rect.X, Y: op.Rect.Y, Width: op.Rect.Width, Height: op.Rect.Height,
			Fill: op.Rect.Fill, Stroke: op.Rect.Stroke, StrokeWidth: op.Rect.StrokeWidth,
		}}}
	case op.Circle != nil:
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Circle{Circle: &uiv1.DrawCircle{
			Cx: op.Circle.CX, Cy: op.Circle.CY, Radius: op.Circle.Radius,
			Fill: op.Circle.Fill, Stroke: op.Circle.Stroke, StrokeWidth: op.Circle.StrokeWidth,
		}}}
	case op.Text != nil:
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Text{Text: &uiv1.DrawText{
			X: op.Text.X, Y: op.Text.Y, Text: op.Text.Text, Fill: op.Text.Fill,
			FontSize: op.Text.FontSize, FontFamily: op.Text.FontFamily,
		}}}
	case op.Path != nil:
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Path{Path: &uiv1.DrawPath{
			D: op.Path.D, Fill: op.Path.Fill, Stroke: op.Path.Stroke, StrokeWidth: op.Path.StrokeWidth,
		}}}
	}
	return nil
}

// canvasDrawOpsFromUI converts native typed CanvasOp values from the
// internal/ui package into proto DrawOp messages without going through the
// legacy JSON envelope. Ops with CanvasOpUnspecified kind or a nil variant
// pointer are skipped. Returns nil if every op was skipped.
func canvasDrawOpsFromUI(ops []ui.CanvasOp) []*uiv1.DrawOp {
	if len(ops) == 0 {
		return nil
	}
	out := make([]*uiv1.DrawOp, 0, len(ops))
	for _, op := range ops {
		converted := uiCanvasOpToProto(op)
		if converted == nil {
			continue
		}
		out = append(out, converted)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func uiCanvasOpToProto(op ui.CanvasOp) *uiv1.DrawOp {
	switch op.Kind {
	case ui.CanvasOpLineKind:
		if op.Line == nil {
			return nil
		}
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Line{Line: &uiv1.DrawLine{
			X1: op.Line.X1, Y1: op.Line.Y1, X2: op.Line.X2, Y2: op.Line.Y2,
			Stroke: op.Line.Stroke, StrokeWidth: op.Line.StrokeWidth,
		}}}
	case ui.CanvasOpRectKind:
		if op.Rect == nil {
			return nil
		}
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Rect{Rect: &uiv1.DrawRect{
			X: op.Rect.X, Y: op.Rect.Y, Width: op.Rect.Width, Height: op.Rect.Height,
			Fill: op.Rect.Fill, Stroke: op.Rect.Stroke, StrokeWidth: op.Rect.StrokeWidth,
		}}}
	case ui.CanvasOpCircleKind:
		if op.Circle == nil {
			return nil
		}
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Circle{Circle: &uiv1.DrawCircle{
			Cx: op.Circle.CX, Cy: op.Circle.CY, Radius: op.Circle.Radius,
			Fill: op.Circle.Fill, Stroke: op.Circle.Stroke, StrokeWidth: op.Circle.StrokeWidth,
		}}}
	case ui.CanvasOpTextKind:
		if op.Text == nil {
			return nil
		}
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Text{Text: &uiv1.DrawText{
			X: op.Text.X, Y: op.Text.Y, Text: op.Text.Text, Fill: op.Text.Fill,
			FontSize: op.Text.FontSize, FontFamily: op.Text.FontFamily,
		}}}
	case ui.CanvasOpPathKind:
		if op.Path == nil {
			return nil
		}
		return &uiv1.DrawOp{Op: &uiv1.DrawOp_Path{Path: &uiv1.DrawPath{
			D: op.Path.D, Fill: op.Path.Fill, Stroke: op.Path.Stroke, StrokeWidth: op.Path.StrokeWidth,
		}}}
	default:
		return nil
	}
}
