package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// Tools for the 1:1 viewer.
// Every image the viewer shows is a native-resolution window of a real decoded file, never a resample, because the whole point of a 1:1 mirror is that one screen pixel is one image pixel.

// cropCmd writes a native-resolution window of an image: lab crop in.png out.png x y w h
func cropCmd(args []string) {
	in, out := args[0], args[1]
	x, _ := strconv.Atoi(args[2])
	y, _ := strconv.Atoi(args[3])
	w, _ := strconv.Atoi(args[4])
	h, _ := strconv.Atoi(args[5])
	im := load(in)
	if x < 0 || y < 0 || x+w > im.W || y+h > im.H {
		fmt.Fprintf(os.Stderr, "crop %d,%d %dx%d outside %dx%d\n", x, y, w, h, im.W, im.H)
		os.Exit(1)
	}
	o := &Img{W: w, H: h, P: make([]float64, w*h*3)}
	for j := 0; j < h; j++ {
		copy(o.P[j*w*3:(j+1)*w*3], im.P[((y+j)*im.W+x)*3:((y+j)*im.W+x+w)*3])
	}
	o.writePNG(out)
}

// diffCmd writes an amplified absolute difference, so error that is invisible at 1:1 becomes legible.
// The gain is printed into the caption rather than chosen per image, because a per-image auto-gain would make two panels with different error magnitudes look equally bad.
func diffCmd(args []string) {
	a, b, out := load(args[0]), load(args[1]), args[2]
	gain, _ := strconv.ParseFloat(args[3], 64)
	if a.W != b.W || a.H != b.H {
		fmt.Fprintln(os.Stderr, "size mismatch")
		os.Exit(1)
	}
	o := &Img{W: a.W, H: a.H, P: make([]float64, len(a.P))}
	var maxAbs, sumAbs float64
	for i := range a.P {
		d := math.Abs(a.P[i] - b.P[i])
		if d > maxAbs {
			maxAbs = d
		}
		sumAbs += d
		o.P[i] = math.Min(255, d*gain)
	}
	o.writePNG(out)
	fmt.Printf("%s: gain %.0fx, max |err| %.0f, mean |err| %.3f (per channel, 0-255)\n",
		out, gain, maxAbs, sumAbs/float64(len(a.P)))
}

// statCmd reports PSNR plus the error statistics a single PSNR figure hides.
func statCmd(args []string) {
	a, b := load(args[0]), load(args[1])
	var sse, sumAbs, maxAbs float64
	over := [4]int{}
	for i := range a.P {
		d := a.P[i] - b.P[i]
		sse += d * d
		ad := math.Abs(d)
		sumAbs += ad
		if ad > maxAbs {
			maxAbs = ad
		}
		for k, t := range []float64{1, 2, 4, 8} {
			if ad > t {
				over[k]++
			}
		}
	}
	n := float64(len(a.P))
	fmt.Printf("psnr %.2f  mae %.4f  max %.0f  pct>1 %.2f  pct>2 %.2f  pct>4 %.2f  pct>8 %.2f\n",
		psnrSSE(sse, a.W*a.H), sumAbs/n, maxAbs,
		100*float64(over[0])/n, 100*float64(over[1])/n, 100*float64(over[2])/n, 100*float64(over[3])/n)
}
