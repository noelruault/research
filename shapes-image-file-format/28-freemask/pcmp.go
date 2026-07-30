// Does a client-side re-segmentation give you what a transmitted partition gives you?
//
// Report 24's margin says: pay ~30% more than WebP to get pixels plus a segmentation. The obvious objection is that a client can segment the decoded pixels itself for free, making the mask cost nothing. Report 13 answers that a transmitted partition is deterministic and identical on every client, but that has never been measured against the free alternative.
//
// This compares two partitions recovered from two flat renders, over every 4-neighbour pixel pair:
// agreement = fraction of pairs where both partitions agree on "same region or not". A pair that is a boundary in one and an interior in the other is a disagreement, which is exactly what breaks
// "select this region" when the mask and the pixels came from different places.
package main

import (
	"fmt"
	"image/png"
	"os"
)

func labelPNG(p string) (int, int, []int32) {
	f, err := os.Open(p)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	px := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			px[y*w+x] = uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(bb>>8)
		}
	}
	lab := make([]int32, w*h)
	for i := range lab {
		lab[i] = -1
	}
	st := make([]int32, 0, 1<<16)
	n := int32(0)
	for s := 0; s < w*h; s++ {
		if lab[s] >= 0 {
			continue
		}
		id := n
		n++
		c := px[s]
		lab[s] = id
		st = append(st[:0], int32(s))
		for len(st) > 0 {
			q := st[len(st)-1]
			st = st[:len(st)-1]
			x, y := int(q)%w, int(q)/w
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				k := int32(ny*w + nx)
				if lab[k] < 0 && px[k] == c {
					lab[k] = id
					st = append(st, k)
				}
			}
		}
	}
	return w, h, lab
}

func main() {
	w1, h1, a := labelPNG(os.Args[1])
	w2, h2, b := labelPNG(os.Args[2])
	if w1 != w2 || h1 != h2 {
		fmt.Println("size mismatch")
		os.Exit(1)
	}
	w, h := w1, h1
	na, nb := 0, 0
	for _, l := range a {
		if int(l)+1 > na {
			na = int(l) + 1
		}
	}
	for _, l := range b {
		if int(l)+1 > nb {
			nb = int(l) + 1
		}
	}
	// Agreement over 4-neighbour pairs, and the two error kinds separately: a boundary the client invents where the transmitted partition has none, and one it misses.
	var pairs, agree, extra, missed int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= w || ny >= h {
					continue
				}
				q := ny*w + nx
				ba := a[p] != a[q] // boundary in A (the reference)
				bb := b[p] != b[q] // boundary in B
				pairs++
				switch {
				case ba == bb:
					agree++
				case bb && !ba:
					extra++
				default:
					missed++
				}
			}
		}
	}
	fmt.Printf("%s regions=%d  |  %s regions=%d\n", os.Args[1], na, os.Args[2], nb)
	fmt.Printf("  pair agreement      %.4f%%  (%d of %d)\n", 100*float64(agree)/float64(pairs), agree, pairs)
	fmt.Printf("  boundaries invented %d\n", extra)
	fmt.Printf("  boundaries missed   %d\n", missed)
	// Boundary-only agreement: of the pairs that are a boundary in either, how many are in both.
	un := extra + missed
	both := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= w || ny >= h {
					continue
				}
				q := ny*w + nx
				if a[p] != a[q] && b[p] != b[q] {
					both++
				}
			}
		}
	}
	fmt.Printf("  boundary Jaccard    %.4f  (shared %d, union %d)\n", float64(both)/float64(both+un), both, both+un)
}
