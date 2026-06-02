package main

import (
	"image"
	"image/color"
	"sort"
)

// quant.go — the Quantizer interface, the shared histogram, and the
// simplest baseline (popularity). Each piece we evaluate implements
// Quantizer; the harness scores its palette with the perceptual + RGB
// metrics in metrics.go.

// Quantizer selects a palette of at most n colors from an image.
type Quantizer interface {
	Name() string
	Quantize(img image.Image, n int) color.Palette
}

// ColorCount is one distinct opaque color and how many pixels use it.
type ColorCount struct {
	R, G, B uint8
	Count   int
}

// histogram returns the image's distinct opaque colors with pixel counts.
// This is the shared P2 input every selection piece builds on; precision
// variants (5-bit/6-bit/octree) will subclass this in report 03.
func histogram(img image.Image) []ColorCount {
	b := img.Bounds()
	counts := make(map[uint32]int, 1<<14)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, _ := img.At(x, y).RGBA()
			key := uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(bb>>8)
			counts[key]++
		}
	}
	out := make([]ColorCount, 0, len(counts))
	for k, c := range counts {
		out = append(out, ColorCount{
			R: uint8(k >> 16), G: uint8(k >> 8), B: uint8(k), Count: c,
		})
	}
	return out
}

// Popularity is the trivial baseline: keep the n most-used colors.
// It establishes the floor every real piece must beat.
type Popularity struct{}

func (Popularity) Name() string { return "popularity" }

func (Popularity) Quantize(img image.Image, n int) color.Palette {
	h := histogram(img)
	sort.Slice(h, func(i, j int) bool { return h[i].Count > h[j].Count })
	if n > len(h) {
		n = len(h)
	}
	pal := make(color.Palette, n)
	for i := 0; i < n; i++ {
		pal[i] = color.RGBA{R: h[i].R, G: h[i].G, B: h[i].B, A: 255}
	}
	return pal
}
