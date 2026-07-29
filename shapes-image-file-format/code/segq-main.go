// P0: are the shape coder's regions semantically meaningful, or do they follow illumination?
//
// Twelve reports measured whether regions are cheap. None measured whether they are right.
// A published render is the partition painted with each region's own mean colour, so 4-connected runs of one colour ARE the regions. This recovers them and asks three questions:
//   1. how many regions cover a window whose content we can name (sky, ridge, snow)?
//   2. how are areas distributed -- a few big object-shaped regions, or many similar bands?
//   3. what do they actually look like? (false-colour dump, read by eye)
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
)

func load(p string) (w, h int, px []uint32) {
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
	w, h = b.Dx(), b.Dy()
	px = make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			px[y*w+x] = uint32(r>>8)<<16 | uint32(g>>8)<<8 | uint32(bb>>8)
		}
	}
	return
}

// 4-connected components of equal colour, iterative flood fill.
func label(w, h int, px []uint32) (lab []int32, n int) {
	lab = make([]int32, w*h)
	for i := range lab {
		lab[i] = -1
	}
	stack := make([]int32, 0, 1<<16)
	for s := 0; s < w*h; s++ {
		if lab[s] >= 0 {
			continue
		}
		id := int32(n)
		n++
		c := px[s]
		lab[s] = id
		stack = append(stack[:0], int32(s))
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := int(p)%w, int(p)/w
			try := func(nx, ny int) {
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					return
				}
				q := int32(ny*w + nx)
				if lab[q] < 0 && px[q] == c {
					lab[q] = id
					stack = append(stack, q)
				}
			}
			try(x-1, y)
			try(x+1, y)
			try(x, y-1)
			try(x, y+1)
		}
	}
	return
}

type win struct {
	name string
	x, y int
}

func main() {
	src := os.Args[1]
	size := 448
	w, h, px := load(src)
	lab, n := label(w, h, px)
	area := make([]int, n)
	for _, l := range lab {
		area[l]++
	}
	fmt.Printf("# %s  %dx%d  regions recovered: %d\n", src, w, h, n)

	// Global area distribution: object-shaped partitions are heavy-tailed.
	sorted := append([]int(nil), area...)
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))
	tot := w * h
	cum := 0
	for i, a := range sorted {
		cum += a
		if i == 0 || i == 9 || i == 99 || i == 999 {
			fmt.Printf("#   top %-5d regions cover %5.1f%% of the image (largest = %.2f%%)\n",
				i+1, 100*float64(cum)/float64(tot), 100*float64(sorted[0])/float64(tot))
		}
	}
	med := sorted[len(sorted)/2]
	fmt.Printf("#   median region %d px, largest %d px, ratio %.0fx\n", med, sorted[0], float64(sorted[0])/float64(med))

	for _, wd := range []win{{"sky", 288, 230}, {"ridge", 1344, 1000}, {"snow", 2400, 1350}} {
		seen := map[int32]int{}
		for dy := 0; dy < size; dy++ {
			for dx := 0; dx < size; dx++ {
				seen[lab[(wd.y+dy)*w+wd.x+dx]]++
			}
		}
		// how much of the window does its single biggest region cover?
		best := 0
		for _, c := range seen {
			if c > best {
				best = c
			}
		}
		fmt.Printf("%-6s @%5d,%-5d  %5d regions in %dx%d   biggest covers %5.1f%% of window\n",
			wd.name, wd.x, wd.y, len(seen), size, size, 100*float64(best)/float64(size*size))

		// false-colour dump so the partition can be looked at rather than argued about
		out := image.NewRGBA(image.Rect(0, 0, size*2, size))
		for dy := 0; dy < size; dy++ {
			for dx := 0; dx < size; dx++ {
				p := (wd.y+dy)*w + wd.x + dx
				c := px[p]
				out.Set(dx, dy, color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255})
				id := uint32(lab[p]) * 2654435761
				out.Set(size+dx, dy, color.RGBA{uint8(id >> 16), uint8(id >> 8), uint8(id), 255})
			}
		}
		f, _ := os.Create(fmt.Sprintf("%s_%s.png", os.Args[2], wd.name))
		png.Encode(f, out)
		f.Close()
	}
}
