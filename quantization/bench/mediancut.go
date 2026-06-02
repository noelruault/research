package main

import (
	"image"
	"image/color"
	"sort"
)

// mediancut.go — Heckbert (1982) median cut, the classic adaptive
// quantizer and the baseline every cross-disciplinary piece is measured
// against (report 04).
//
// Algorithm: treat the whole color histogram as one box; repeatedly take
// the box with the most pixels, split it along its longest RGB axis at the
// population median, until n boxes exist. Each box yields one palette color:
// its population-weighted mean.

type MedianCut struct{}

func (MedianCut) Name() string { return "median-cut" }

type vbox struct {
	colors []ColorCount
	pixels int // total pixel count in the box
}

func newVBox(colors []ColorCount) vbox {
	p := 0
	for _, c := range colors {
		p += c.Count
	}
	return vbox{colors: colors, pixels: p}
}

// longestAxis returns 0,1,2 for R,G,B — whichever has the widest extent.
func (v vbox) longestAxis() int {
	rmin, gmin, bmin := uint8(255), uint8(255), uint8(255)
	rmax, gmax, bmax := uint8(0), uint8(0), uint8(0)
	for _, c := range v.colors {
		rmin, rmax = min8(rmin, c.R), max8(rmax, c.R)
		gmin, gmax = min8(gmin, c.G), max8(gmax, c.G)
		bmin, bmax = min8(bmin, c.B), max8(bmax, c.B)
	}
	dr, dg, db := rmax-rmin, gmax-gmin, bmax-bmin
	switch {
	case dr >= dg && dr >= db:
		return 0
	case dg >= db:
		return 1
	default:
		return 2
	}
}

// split partitions the box along axis at the population median.
func (v vbox) split() (vbox, vbox) {
	axis := v.longestAxis()
	sort.Slice(v.colors, func(i, j int) bool {
		return axisVal(v.colors[i], axis) < axisVal(v.colors[j], axis)
	})
	half := v.pixels / 2
	acc, cut := 0, 1
	for i, c := range v.colors {
		acc += c.Count
		if acc >= half {
			cut = i + 1
			break
		}
	}
	if cut <= 0 {
		cut = 1
	}
	if cut >= len(v.colors) {
		cut = len(v.colors) - 1
	}
	return newVBox(v.colors[:cut]), newVBox(v.colors[cut:])
}

// mean is the box's population-weighted average color.
func (v vbox) mean() color.RGBA {
	var sr, sg, sb, sw int
	for _, c := range v.colors {
		sr += int(c.R) * c.Count
		sg += int(c.G) * c.Count
		sb += int(c.B) * c.Count
		sw += c.Count
	}
	if sw == 0 {
		sw = 1
	}
	return color.RGBA{R: uint8(sr / sw), G: uint8(sg / sw), B: uint8(sb / sw), A: 255}
}

func (MedianCut) Quantize(img image.Image, n int) color.Palette {
	h := histogram(img)
	if len(h) <= n {
		pal := make(color.Palette, len(h))
		for i, c := range h {
			pal[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
		}
		return pal
	}

	boxes := []vbox{newVBox(h)}
	for len(boxes) < n {
		// Pick the splittable box with the most pixels.
		best := -1
		for i := range boxes {
			if len(boxes[i].colors) < 2 {
				continue
			}
			if best < 0 || boxes[i].pixels > boxes[best].pixels {
				best = i
			}
		}
		if best < 0 {
			break // every box is a single color
		}
		a, b := boxes[best].split()
		boxes[best] = a
		boxes = append(boxes, b)
	}

	pal := make(color.Palette, len(boxes))
	for i, bx := range boxes {
		pal[i] = bx.mean()
	}
	return pal
}

func axisVal(c ColorCount, axis int) uint8 {
	switch axis {
	case 0:
		return c.R
	case 1:
		return c.G
	default:
		return c.B
	}
}

func min8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func max8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
