package usecasevalidation

import (
	"image"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func renderStack(img *image.RGBA, d ui.Descriptor, bounds image.Rectangle) {
	if bg := d.Props["background"]; bg != "" {
		fillRect(img, bounds, parseHexColorRGBA(bg))
	}
	children := d.Children
	if len(children) == 0 {
		return
	}
	inner := stackInnerBounds(bounds)
	if inner.Dx() <= 0 || inner.Dy() <= 0 {
		return
	}
	heights, expandH := measureStackChildren(children, inner)
	y := inner.Min.Y
	const gap = 8
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

func stackInnerBounds(bounds image.Rectangle) image.Rectangle {
	return image.Rect(
		bounds.Min.X+descPad,
		bounds.Min.Y+descPad,
		bounds.Max.X-descPad,
		bounds.Max.Y-descPad,
	)
}

func measureStackChildren(children []ui.Descriptor, inner image.Rectangle) (heights []int, expandH int) {
	heights = make([]int, len(children))
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
	expandH = 20
	if expandCount > 0 {
		if remaining := inner.Dy() - totalFixed - gap*(len(children)-1); remaining > 0 {
			expandH = remaining / expandCount
		}
	}
	return heights, expandH
}
