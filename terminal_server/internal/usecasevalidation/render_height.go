package usecasevalidation

import (
	"strconv"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func estimateDescHeight(d ui.Descriptor, width int) int {
	switch d.Type {
	case "text":
		return estimateTextDescHeight(d, width)
	case "button":
		return 32
	case "image":
		return 80
	case "video_surface", "audio_visualizer":
		return 120
	case "expand":
		return -1
	case "overlay":
		return estimateOverlayDescHeight(d, width)
	case "row":
		return estimateRowDescHeight(d, width)
	case "grid":
		return estimateGridDescHeight(d, width)
	default:
		return estimateDefaultDescHeight(d, width)
	}
}

func estimateTextDescHeight(d ui.Descriptor, width int) int {
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
}

func estimateOverlayDescHeight(d ui.Descriptor, width int) int {
	return maxChildDescHeight(d.Children, width)
}

func estimateRowDescHeight(d ui.Descriptor, width int) int {
	childW := width / max(1, len(d.Children))
	return maxChildDescHeight(d.Children, childW) + 2*descPad
}

func estimateGridDescHeight(d ui.Descriptor, width int) int {
	cols, _ := strconv.Atoi(d.Props["columns"])
	if cols < 1 {
		cols = 1
	}
	cellW := width / cols
	maxCellH := maxChildDescHeight(d.Children, cellW)
	rows := (len(d.Children) + cols - 1) / cols
	return rows*maxCellH + 2*descPad
}

func estimateDefaultDescHeight(d ui.Descriptor, width int) int {
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

func maxChildDescHeight(children []ui.Descriptor, width int) int {
	maxH := 0
	for _, child := range children {
		if h := estimateDescHeight(child, width); h > maxH {
			maxH = h
		}
	}
	return maxH
}
