package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// sbsCmd builds a side-by-side comparison strip so a fidelity claim can be looked at rather than only read. Four panels: the source, this coder's render, the rival's render, and this coder's region boundaries drawn over its own render.
//
// The fourth panel is the one that matters for the structured-image pitch. Boundaries are derived from the render itself — adjacent pixels of different colour — which needs no label dump and is exactly the region structure a consumer would address. If the subject is not separated there, no amount of byte parity makes the format useful for selection.
//
// usage: sbs <source.png> <ours.png> <rival.png> <out.png>
func sbsCmd(args []string) {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: lab sbs <source.png> <ours.png> <rival.png> <out.png>")
		os.Exit(2)
	}
	src, ours, rival := decodePNG(args[0]), decodePNG(args[1]), decodePNG(args[2])
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	for i, im := range []image.Image{ours, rival} {
		if im.Bounds().Dx() != w || im.Bounds().Dy() != h {
			fmt.Fprintf(os.Stderr, "fatal: arm %d is %dx%d, source is %dx%d\n",
				i+1, im.Bounds().Dx(), im.Bounds().Dy(), w, h)
			os.Exit(1)
		}
	}
	at := func(im image.Image, x, y int) color.NRGBA {
		r := im.Bounds()
		return color.NRGBAModel.Convert(im.At(r.Min.X+x, r.Min.Y+y)).(color.NRGBA)
	}
	same := func(a, c color.NRGBA) bool { return a.R == c.R && a.G == c.G && a.B == c.B && a.A == c.A }

	const gap = 4
	out := image.NewNRGBA(image.Rect(0, 0, w*4+gap*3, h))
	blit := func(px int, f func(x, y int) color.NRGBA) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.SetNRGBA(px+x, y, f(x, y))
			}
		}
	}
	blit(0, func(x, y int) color.NRGBA { return at(src, x, y) })
	blit(w+gap, func(x, y int) color.NRGBA { return at(ours, x, y) })
	blit((w+gap)*2, func(x, y int) color.NRGBA { return at(rival, x, y) })
	// Region boundaries: a pixel differing from its right or lower neighbour sits on a wall.
	// Drawn dark over a lightened render so the structure reads without hiding the picture.
	blit((w+gap)*3, func(x, y int) color.NRGBA {
		c := at(ours, x, y)
		edge := (x+1 < w && !same(c, at(ours, x+1, y))) || (y+1 < h && !same(c, at(ours, x, y+1)))
		if edge {
			return color.NRGBA{0x10, 0x10, 0x10, 255}
		}
		wash := func(v uint8) uint8 { return uint8(int(v)/3 + 170) }
		return color.NRGBA{wash(c.R), wash(c.G), wash(c.B), 255}
	})
	f, err := os.Create(args[3])
	must(err)
	must(png.Encode(f, out))
	must(f.Close())
	fmt.Printf("sbs %s -> %s (%dx%d, 4 panels: source | ours | rival | our regions)\n",
		args[0], args[3], w, h)
}
