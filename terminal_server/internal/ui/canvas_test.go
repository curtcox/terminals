package ui

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCanvasBuildersPopulateTaggedUnion(t *testing.T) {
	line := Line(1, 2, 3, 4, "#fff", 1.5)
	if line.Kind != CanvasOpLineKind || line.Line == nil {
		t.Fatalf("Line builder did not populate Kind/Line: %+v", line)
	}
	if line.Line.X1 != 1 || line.Line.Y1 != 2 || line.Line.X2 != 3 || line.Line.Y2 != 4 {
		t.Fatalf("Line coords mismatch: %+v", line.Line)
	}
	if line.Line.Stroke != "#fff" || line.Line.StrokeWidth != 1.5 {
		t.Fatalf("Line stroke mismatch: %+v", line.Line)
	}

	rect := Rect(5, 6, 7, 8, "#aaa", "#bbb", 2)
	if rect.Kind != CanvasOpRectKind || rect.Rect == nil {
		t.Fatalf("Rect builder did not populate Kind/Rect: %+v", rect)
	}

	circle := Circle(9, 10, 11, "#ccc", "#ddd", 3)
	if circle.Kind != CanvasOpCircleKind || circle.Circle == nil {
		t.Fatalf("Circle builder did not populate Kind/Circle: %+v", circle)
	}

	text := Text(12, 13, "hi", "#eee", 14, "monospace")
	if text.Kind != CanvasOpTextKind || text.Text == nil {
		t.Fatalf("Text builder did not populate Kind/Text: %+v", text)
	}
	if text.Text.Text != "hi" || text.Text.FontFamily != "monospace" {
		t.Fatalf("Text content mismatch: %+v", text.Text)
	}

	path := Path("M 0 0 L 10 10", "#fff", "#000", 0.5)
	if path.Kind != CanvasOpPathKind || path.Path == nil {
		t.Fatalf("Path builder did not populate Kind/Path: %+v", path)
	}
}

func TestCanvasNodeProducesDescriptorWithTypedAndLegacyMirror(t *testing.T) {
	d := CanvasNode("pulse",
		Rect(0, 0, 10, 10, "#100", "", 0),
		Circle(5, 5, 2, "#0f0", "", 0),
	)
	if d.Type != "canvas" {
		t.Fatalf("Type = %q, want canvas", d.Type)
	}
	if d.Props["id"] != "pulse" {
		t.Fatalf("id = %q, want pulse", d.Props["id"])
	}
	if len(d.CanvasOps) != 2 {
		t.Fatalf("CanvasOps len = %d, want 2", len(d.CanvasOps))
	}
	if d.CanvasOps[0].Kind != CanvasOpRectKind {
		t.Fatalf("CanvasOps[0].Kind = %v, want Rect", d.CanvasOps[0].Kind)
	}
	if d.CanvasOps[1].Kind != CanvasOpCircleKind {
		t.Fatalf("CanvasOps[1].Kind = %v, want Circle", d.CanvasOps[1].Kind)
	}

	legacy := d.Props["draw_ops_json"]
	if legacy == "" {
		t.Fatalf("expected legacy draw_ops_json mirror to be populated")
	}
	if !strings.Contains(legacy, `"rect"`) || !strings.Contains(legacy, `"circle"`) {
		t.Fatalf("legacy JSON missing rect/circle entries: %s", legacy)
	}

	var parsed struct {
		Ops []map[string]any `json:"ops"`
	}
	if err := json.Unmarshal([]byte(legacy), &parsed); err != nil {
		t.Fatalf("legacy JSON not valid: %v (%s)", err, legacy)
	}
	if len(parsed.Ops) != 2 {
		t.Fatalf("legacy ops count = %d, want 2", len(parsed.Ops))
	}
}

func TestCanvasNodeIsolatesOpsSlice(t *testing.T) {
	ops := []CanvasOp{Line(0, 0, 1, 1, "#000", 1)}
	d := CanvasNode("c1", ops...)
	ops[0] = Rect(0, 0, 1, 1, "#000", "", 0)
	if d.CanvasOps[0].Kind != CanvasOpLineKind {
		t.Fatalf("CanvasNode shared backing slice with caller; got kind %v", d.CanvasOps[0].Kind)
	}
}

func TestCanvasOpsToJSONReturnsEmptyForOnlyUnspecified(t *testing.T) {
	if got := CanvasOpsToJSON(nil); got != "" {
		t.Fatalf("nil input -> %q, want empty", got)
	}
	if got := CanvasOpsToJSON([]CanvasOp{}); got != "" {
		t.Fatalf("empty input -> %q, want empty", got)
	}
	bogus := []CanvasOp{{Kind: CanvasOpUnspecified}, {Kind: CanvasOpLineKind, Line: nil}}
	if got := CanvasOpsToJSON(bogus); got != "" {
		t.Fatalf("only-bogus input -> %q, want empty", got)
	}
}

func TestCanvasOpsToJSONSkipsBogusKeepsValid(t *testing.T) {
	mixed := []CanvasOp{
		{Kind: CanvasOpLineKind, Line: nil},
		Rect(1, 2, 3, 4, "#aaa", "#bbb", 1),
		{Kind: CanvasOpUnspecified},
	}
	got := CanvasOpsToJSON(mixed)
	if got == "" {
		t.Fatalf("expected non-empty output")
	}
	var parsed struct {
		Ops []map[string]any `json:"ops"`
	}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("output not valid JSON: %v (%s)", err, got)
	}
	if len(parsed.Ops) != 1 {
		t.Fatalf("ops count = %d, want 1", len(parsed.Ops))
	}
	if _, ok := parsed.Ops[0]["rect"]; !ok {
		t.Fatalf("expected rect entry, got %+v", parsed.Ops[0])
	}
}

func TestDiagnosticsConnectionPulseOverlayHealthy(t *testing.T) {
	d := DiagnosticsConnectionPulseOverlay("device-1", true, 42)
	if d.Type != "overlay" {
		t.Fatalf("Type = %q, want overlay", d.Type)
	}
	if d.Props["id"] != GlobalOverlayComponentID {
		t.Fatalf("id = %q, want %q", d.Props["id"], GlobalOverlayComponentID)
	}
	if len(d.Children) != 1 || d.Children[0].Type != "stack" {
		t.Fatalf("expected single stack child, got %+v", d.Children)
	}
	stack := d.Children[0]
	if stack.Props["id"] != DiagnosticsConnectionPulseComponentID {
		t.Fatalf("stack id = %q, want %q", stack.Props["id"], DiagnosticsConnectionPulseComponentID)
	}
	if len(stack.Children) != 2 {
		t.Fatalf("stack child count = %d, want 2 (label text + canvas)", len(stack.Children))
	}
	label := stack.Children[0]
	if label.Type != "text" {
		t.Fatalf("first stack child = %q, want text", label.Type)
	}
	if !strings.Contains(label.Props["value"], "OK") || !strings.Contains(label.Props["value"], "42ms") {
		t.Fatalf("label value missing OK/42ms: %q", label.Props["value"])
	}
	canvas := stack.Children[1]
	if canvas.Type != "canvas" {
		t.Fatalf("second stack child = %q, want canvas", canvas.Type)
	}
	if len(canvas.CanvasOps) != 4 {
		t.Fatalf("canvas typed op count = %d, want 4 (rect, circle, line, text)", len(canvas.CanvasOps))
	}
	if canvas.CanvasOps[1].Kind != CanvasOpCircleKind || canvas.CanvasOps[1].Circle == nil {
		t.Fatalf("expected circle indicator at index 1, got %+v", canvas.CanvasOps[1])
	}
	if canvas.CanvasOps[1].Circle.Fill != "#33CC66" {
		t.Fatalf("healthy circle fill = %q, want #33CC66", canvas.CanvasOps[1].Circle.Fill)
	}
	if canvas.Props["draw_ops_json"] == "" {
		t.Fatalf("expected legacy draw_ops_json mirror to be populated for compatibility window")
	}
}

func TestDiagnosticsConnectionPulseOverlayUnhealthy(t *testing.T) {
	d := DiagnosticsConnectionPulseOverlay("device-2", false, 999)
	stack := d.Children[0]
	canvas := stack.Children[1]
	if canvas.CanvasOps[1].Circle.Fill != "#CC3333" {
		t.Fatalf("unhealthy circle fill = %q, want #CC3333", canvas.CanvasOps[1].Circle.Fill)
	}
	if !strings.Contains(stack.Children[0].Props["value"], "DOWN") {
		t.Fatalf("unhealthy label missing DOWN: %q", stack.Children[0].Props["value"])
	}
}
