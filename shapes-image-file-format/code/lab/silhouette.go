package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"
)

// silhouetteCmd is research item A1 (DESIGN-ALPHA.md): does the colour-driven merge dissolve a sprite's silhouette?
//
// It needs no label dump. The merge renders every region as one flat colour, so a region that straddles the silhouette shows the SAME colour on both sides of it. So for every adjacent pixel pair that crosses the true alpha silhouette — one transparent, one not — the test is simply whether the render's colour changes there. No change means both pixels landed in one region and the silhouette was dissolved at that crossing.
//
// Note what `load` does to alpha before the merge ever runs: it drops it, and Go's PNG decoder returns premultiplied values, so every transparent pixel arrives at the merge as pure black.
// The silhouette therefore survives only where the sprite's own edge colour differs from black.
func silhouetteCmd(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "silhouette <sprite.png> <render.png> <out.png> [scale]")
		os.Exit(2)
	}
	scale := 8
	if len(args) > 3 {
		scale, _ = strconv.Atoi(args[3])
	}
	sprite := decodePNG(args[0])
	render := decodePNG(args[1])
	sb, rb := sprite.Bounds(), render.Bounds()
	if sb.Dx() != rb.Dx() || sb.Dy() != rb.Dy() {
		fmt.Fprintf(os.Stderr, "size mismatch: sprite %dx%d, render %dx%d\n", sb.Dx(), sb.Dy(), rb.Dx(), rb.Dy())
		os.Exit(2)
	}
	w, h := sb.Dx(), sb.Dy()

	// Straight (un-premultiplied) sprite colour, plus alpha.
	nrgbaAt := func(x, y int) color.NRGBA {
		return color.NRGBAModel.Convert(sprite.At(sb.Min.X+x, sb.Min.Y+y)).(color.NRGBA)
	}
	renderAt := func(x, y int) color.NRGBA {
		return color.NRGBAModel.Convert(render.At(rb.Min.X+x, rb.Min.Y+y)).(color.NRGBA)
	}
	// A pixel is "void" when fully transparent. Partial alpha counts as part of the object, so the silhouette measured here is the outermost edge of anything drawn at all.
	void := func(x, y int) bool { return nrgbaAt(x, y).A == 0 }

	// What load() actually hands the merge on the opaque side of a crossing: alpha dropped,
	// and premultiplied, so a soft rim pixel arrives darkened toward black. Where that value is itself near-black it is indistinguishable from the void, and NO merge rule can separate them — the information was destroyed before the merge ran. This column separates "the merge joined two distinguishable things" from "the merge was handed two identical things".
	const blackish = 24
	nearBlack := func(x, y int) bool {
		c := nrgbaAt(x, y)
		pm := func(v uint8) int { return int(v) * int(c.A) / 255 }
		return pm(c.R) <= blackish && pm(c.G) <= blackish && pm(c.B) <= blackish
	}

	var crossings, dissolved, invisible int
	lost := make([]bool, w*h) // marks the opaque side of a dissolved crossing, for the overlay
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= w || ny >= h {
					continue
				}
				if void(x, y) == void(nx, ny) {
					continue // not a silhouette crossing
				}
				crossings++
				if void(x, y) && nearBlack(nx, ny) || void(nx, ny) && nearBlack(x, y) {
					invisible++
				}
				a, b := renderAt(x, y), renderAt(nx, ny)
				if a.R == b.R && a.G == b.G && a.B == b.B {
					dissolved++
					if void(x, y) {
						lost[ny*w+nx] = true
					} else {
						lost[y*w+x] = true
					}
				}
			}
		}
	}
	pct := 0.0
	if crossings > 0 {
		pct = 100 * float64(dissolved) / float64(crossings)
	}
	ipct := 0.0
	if crossings > 0 {
		ipct = 100 * float64(invisible) / float64(crossings)
	}
	verdict := "silhouette held"
	if pct > 0 {
		verdict = "DISSOLVED"
	}
	fmt.Printf("%-28s %6d %8d %9.2f%% %10d %9.2f%% %s\n",
		shortName(args[1]), crossings, dissolved, pct, invisible, ipct, verdict)

	// Four panels, magnified: the sprite as authored, what the merge actually receives, the merged render, and the render with dissolved crossings marked.
	const pad = 2
	pw, ph := w*scale, h*scale
	out := image.NewNRGBA(image.Rect(0, 0, pw*4+pad*3, ph))
	blit := func(px int, f func(x, y int) color.NRGBA) {
		for y := 0; y < ph; y++ {
			for x := 0; x < pw; x++ {
				out.SetNRGBA(px+x, y, f(x/scale, y/scale))
			}
		}
	}
	// Panel 1: authored sprite over a checkerboard, so transparency is visible as transparency.
	blit(0, func(x, y int) color.NRGBA {
		c := nrgbaAt(x, y)
		if c.A == 255 {
			return c
		}
		bg := uint8(0x99)
		if ((x/4)+(y/4))%2 == 0 {
			bg = 0xcc
		}
		mix := func(v uint8) uint8 { return uint8((int(v)*int(c.A) + int(bg)*(255-int(c.A))) / 255) }
		return color.NRGBA{mix(c.R), mix(c.G), mix(c.B), 255}
	})
	// Panel 2: what load() hands the merge — alpha gone, transparent premultiplied to black.
	blit(pw+pad, func(x, y int) color.NRGBA {
		c := nrgbaAt(x, y)
		if c.A == 0 {
			return color.NRGBA{0, 0, 0, 255}
		}
		return color.NRGBA{c.R, c.G, c.B, 255}
	})
	// Panel 3: the merged render.
	blit((pw+pad)*2, func(x, y int) color.NRGBA { return renderAt(x, y) })
	// Panel 4: dissolved crossings in red.
	blit((pw+pad)*3, func(x, y int) color.NRGBA {
		c := renderAt(x, y)
		if lost[y*w+x] {
			return color.NRGBA{0xff, 0x00, 0x00, 255}
		}
		return c
	})
	f, err := os.Create(args[2])
	must(err)
	must(png.Encode(f, out))
	must(f.Close())
}

// shortName keeps the table readable when renders are addressed by absolute path.
func shortName(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

func decodePNG(path string) image.Image {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	must(err)
	return img
}
