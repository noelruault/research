package main

import (
	"image"
	"image/color"
	"math"
)

// space.go — color-space abstraction (piece P1). Selection algorithms work on
// Vec3 points in a chosen Space; only the space changes where "close" is
// measured. RGB is the baseline; OKLab is the cheap perceptual space from
// research report 01 (cluster in OKLab, evaluate in CIEDE2000).

type Vec3 [3]float64

func (a Vec3) add(b Vec3) Vec3   { return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec3) scale(s float64) Vec3 { return Vec3{a[0] * s, a[1] * s, a[2] * s} }
func (a Vec3) dist2(b Vec3) float64 {
	d0, d1, d2 := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return d0*d0 + d1*d1 + d2*d2
}

// Space maps 8-bit sRGB to a working vector space and back.
type Space interface {
	Name() string
	FromRGB(r, g, b uint8) Vec3
	ToRGBA(v Vec3) color.RGBA
}

func clamp8(f float64) uint8 {
	i := int(math.Round(f))
	switch {
	case i < 0:
		return 0
	case i > 255:
		return 255
	default:
		return uint8(i)
	}
}

// RGBSpace: plain 8-bit RGB as floats in [0,255].
type RGBSpace struct{}

func (RGBSpace) Name() string { return "rgb" }
func (RGBSpace) FromRGB(r, g, b uint8) Vec3 {
	return Vec3{float64(r), float64(g), float64(b)}
}
func (RGBSpace) ToRGBA(v Vec3) color.RGBA {
	return color.RGBA{R: clamp8(v[0]), G: clamp8(v[1]), B: clamp8(v[2]), A: 255}
}

func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

// OKLabSpace: Björn Ottosson's OKLab (2020). Cheap, better hue linearity than
// CIELAB; used here for clustering geometry only.
type OKLabSpace struct{}

func (OKLabSpace) Name() string { return "oklab" }

func (OKLabSpace) FromRGB(r, g, b uint8) Vec3 {
	rl, gl, bl := srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)
	l := 0.4122214708*rl + 0.5363325363*gl + 0.0514459929*bl
	m := 0.2119034982*rl + 0.6806995451*gl + 0.1073969566*bl
	s := 0.0883024619*rl + 0.2817188376*gl + 0.6299787005*bl
	l_, m_, s_ := math.Cbrt(l), math.Cbrt(m), math.Cbrt(s)
	return Vec3{
		0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_,
		1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_,
		0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_,
	}
}

func (OKLabSpace) ToRGBA(v Vec3) color.RGBA {
	L, A, B := v[0], v[1], v[2]
	l_ := L + 0.3963377774*A + 0.2158037573*B
	m_ := L - 0.1055613458*A - 0.0638541728*B
	s_ := L - 0.0894841775*A - 1.2914855480*B
	l, m, s := l_*l_*l_, m_*m_*m_, s_*s_*s_
	rl := +4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	gl := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	bl := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	return color.RGBA{
		R: clamp8(255 * linearToSRGB(rl)),
		G: clamp8(255 * linearToSRGB(gl)),
		B: clamp8(255 * linearToSRGB(bl)),
		A: 255,
	}
}

// WPoint is a histogram entry projected into a working space, carrying its
// pixel frequency as weight.
type WPoint struct {
	P Vec3
	W float64
}

func weightedPoints(h []ColorCount, sp Space) []WPoint {
	out := make([]WPoint, len(h))
	for i, c := range h {
		out[i] = WPoint{P: sp.FromRGB(c.R, c.G, c.B), W: float64(c.Count)}
	}
	return out
}

// paletteFromImage is a convenience for quantizers that operate via a
// histogram: it returns the histogram once so callers don't re-walk pixels.
func paletteFromImage(img image.Image, sp Space) []WPoint {
	return weightedPoints(histogram(img), sp)
}
