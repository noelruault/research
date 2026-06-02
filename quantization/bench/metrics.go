package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// metrics.go — image-quality measures for palette evaluation.
//
// Two families:
//   - RGB MSE / PSNR: what the incumbent quantizers optimize.
//   - CIEDE2000 over CIELAB: perceptual color difference, the headline metric.
//
// sRGB->Lab and CIEDE2000 are implemented from the CIE formulation and
// self-tested against the Sharma et al. reference pairs (see metrics_test.go),
// so the perceptual numbers are trustworthy.

// Lab is a CIELAB color (D65 reference white).
type Lab struct{ L, A, B float64 }

// srgbToLinear inverts the sRGB transfer function for one 8-bit channel.
func srgbToLinear(c uint8) float64 {
	v := float64(c) / 255.0
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// D65 reference white in XYZ (Yn normalized to 1).
const (
	xn = 0.95047
	yn = 1.00000
	zn = 1.08883
)

func labF(t float64) float64 {
	const eps = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	if t > eps {
		return math.Cbrt(t)
	}
	return (kappa*t + 16.0) / 116.0
}

// RGBToLab converts an 8-bit sRGB triple to CIELAB.
func RGBToLab(r, g, b uint8) Lab {
	rl, gl, bl := srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)
	// Linear sRGB -> XYZ (D65).
	x := 0.4124564*rl + 0.3575761*gl + 0.1804375*bl
	y := 0.2126729*rl + 0.7151522*gl + 0.0721750*bl
	z := 0.0193339*rl + 0.1191920*gl + 0.9503041*bl
	fx, fy, fz := labF(x/xn), labF(y/yn), labF(z/zn)
	return Lab{
		L: 116.0*fy - 16.0,
		A: 500.0 * (fx - fy),
		B: 200.0 * (fy - fz),
	}
}

func deg2rad(d float64) float64 { return d * math.Pi / 180.0 }

// CIEDE2000 returns the perceptual color difference between two Lab colors,
// with the standard parametric weights kL=kC=kH=1.
func CIEDE2000(l1, l2 Lab) float64 {
	c1 := math.Hypot(l1.A, l1.B)
	c2 := math.Hypot(l2.A, l2.B)
	cBar := (c1 + c2) / 2.0
	cBar7 := math.Pow(cBar, 7)
	g := 0.5 * (1.0 - math.Sqrt(cBar7/(cBar7+math.Pow(25.0, 7))))

	a1p := (1.0 + g) * l1.A
	a2p := (1.0 + g) * l2.A
	c1p := math.Hypot(a1p, l1.B)
	c2p := math.Hypot(a2p, l2.B)

	h1p := hueAngle(l1.B, a1p)
	h2p := hueAngle(l2.B, a2p)

	dLp := l2.L - l1.L
	dCp := c2p - c1p

	var dhp float64
	switch {
	case c1p*c2p == 0:
		dhp = 0
	case math.Abs(h2p-h1p) <= 180:
		dhp = h2p - h1p
	case h2p-h1p > 180:
		dhp = h2p - h1p - 360
	default:
		dhp = h2p - h1p + 360
	}
	dHp := 2.0 * math.Sqrt(c1p*c2p) * math.Sin(deg2rad(dhp/2.0))

	lBarp := (l1.L + l2.L) / 2.0
	cBarp := (c1p + c2p) / 2.0

	var hBarp float64
	switch {
	case c1p*c2p == 0:
		hBarp = h1p + h2p
	case math.Abs(h1p-h2p) <= 180:
		hBarp = (h1p + h2p) / 2.0
	case h1p+h2p < 360:
		hBarp = (h1p + h2p + 360) / 2.0
	default:
		hBarp = (h1p + h2p - 360) / 2.0
	}

	t := 1.0 -
		0.17*math.Cos(deg2rad(hBarp-30)) +
		0.24*math.Cos(deg2rad(2*hBarp)) +
		0.32*math.Cos(deg2rad(3*hBarp+6)) -
		0.20*math.Cos(deg2rad(4*hBarp-63))

	dTheta := 30.0 * math.Exp(-math.Pow((hBarp-275)/25.0, 2))
	cBarp7 := math.Pow(cBarp, 7)
	rc := 2.0 * math.Sqrt(cBarp7/(cBarp7+math.Pow(25.0, 7)))
	sl := 1.0 + (0.015*math.Pow(lBarp-50, 2))/math.Sqrt(20.0+math.Pow(lBarp-50, 2))
	sc := 1.0 + 0.045*cBarp
	sh := 1.0 + 0.015*cBarp*t
	rt := -math.Sin(deg2rad(2*dTheta)) * rc

	dl := dLp / sl
	dc := dCp / sc
	dh := dHp / sh
	return math.Sqrt(dl*dl + dc*dc + dh*dh + rt*dc*dh)
}

// hueAngle returns atan2(b, a) in [0,360) degrees, with the CIEDE2000
// convention that a zero chroma yields 0.
func hueAngle(b, a float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	h := math.Atan2(b, a) * 180.0 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}

// Score holds the quality of a quantized image versus its original.
type Score struct {
	MSE       float64 // mean squared error in 8-bit RGB
	PSNR      float64 // dB
	MeanDE    float64 // mean CIEDE2000
	P95DE     float64 // 95th-percentile CIEDE2000
}

// scoreAgainst assigns every pixel of img to its nearest entry in pal (exact,
// via color.Palette.Index — bit-identical to pixelize's shipped kd-tree for
// opaque colors) and measures the error of that quantization.
func scoreAgainst(img image.Image, pal color.Palette) Score {
	b := img.Bounds()
	n := b.Dx() * b.Dy()
	if n == 0 {
		return Score{}
	}

	// Cache Lab for each palette entry.
	palLab := make([]Lab, len(pal))
	for i, c := range pal {
		r, g, bb, _ := c.RGBA()
		palLab[i] = RGBToLab(uint8(r>>8), uint8(g>>8), uint8(bb>>8))
	}

	var sumSq float64
	des := make([]float64, 0, n)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r0, g0, b0, _ := img.At(x, y).RGBA()
			sr, sg, sb := uint8(r0>>8), uint8(g0>>8), uint8(b0>>8)
			idx := pal.Index(color.RGBA{R: sr, G: sg, B: sb, A: 255})
			pr, pg, pb, _ := pal[idx].RGBA()
			dr := float64(sr) - float64(pr>>8)
			dg := float64(sg) - float64(pg>>8)
			db := float64(sb) - float64(pb>>8)
			sumSq += dr*dr + dg*dg + db*db
			de := CIEDE2000(RGBToLab(sr, sg, sb), palLab[idx])
			des = append(des, de)
		}
	}

	mse := sumSq / float64(n*3)
	psnr := math.Inf(1)
	if mse > 0 {
		psnr = 10 * math.Log10(255*255/mse)
	}

	sort.Float64s(des)
	var sumDE float64
	for _, d := range des {
		sumDE += d
	}
	p95 := des[int(math.Min(float64(len(des)-1), float64(len(des))*0.95))]
	return Score{MSE: mse, PSNR: psnr, MeanDE: sumDE / float64(len(des)), P95DE: p95}
}
