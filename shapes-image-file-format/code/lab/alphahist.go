package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
)

// alphaHistCmd answers the question that decides whether a per-pixel alpha mode is needed at all:
// how much of a real game sprite's alpha is genuinely soft, rather than the anti-aliased rim of a hard silhouette?
//
// The split matters because the two cost completely different things in a shape format.
// A soft pixel touching a fully-transparent one is a silhouette rim, which a stored boundary reproduces exactly and for free.
// A soft pixel with no transparent neighbour is interior translucency — glass, smoke, glow — and that is the only kind a single flat alpha per region cannot represent.
//
// The interior column is an UPPER BOUND on translucency, not a measurement of it:
// the 8-neighbour test only catches soft pixels directly touching alpha 0, so the inner row of a 2-pixel-wide anti-aliased gradient is counted as interior when it is still rim.
// Read it as "at most this much", and see DESIGN-ALPHA.md item A3 for the test that would settle it.
func alphaHistCmd(paths []string) {
	fmt.Printf("%-44s %9s %8s %8s %8s %8s %7s %8s %9s\n",
		"file", "pixels", "a=0", "a=255", "soft", "soft%", "rim", "interior", "interior%")
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, path, err)
			continue
		}
		b := img.Bounds()
		alphaAt := func(x, y int) uint32 {
			if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
				return 0 // outside the image reads as transparent
			}
			_, _, _, a := img.At(x, y).RGBA() // RGBA is 16-bit
			return a >> 8
		}
		var zero, full, rim, interior int
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				a := alphaAt(x, y)
				switch a {
				case 0:
					zero++
					continue
				case 255:
					full++
					continue
				}
				touchesVoid := false
				for dy := -1; dy <= 1 && !touchesVoid; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if (dx != 0 || dy != 0) && alphaAt(x+dx, y+dy) == 0 {
							touchesVoid = true
							break
						}
					}
				}
				if touchesVoid {
					rim++
				} else {
					interior++
				}
			}
		}
		soft := rim + interior
		n := zero + full + soft
		if n == 0 {
			continue
		}
		fmt.Printf("%-44s %9d %8d %8d %8d %7.2f%% %7d %8d %8.2f%%\n",
			path, n, zero, full, soft, 100*float64(soft)/float64(n),
			rim, interior, 100*float64(interior)/float64(n))
	}
}
