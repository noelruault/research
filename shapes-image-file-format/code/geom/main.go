// Geometry-cost experiment: how much does the SHAPE of an image cost, with colour free?
//
// The question this settles: the shape pipeline produced 32,924 rects for a 512x288 photo.
// Is that number an artefact of palette quantization manufacturing false contours on smooth gradients, or is it what a photo actually costs in regions?
//
// Method: segment the ORIGINAL image directly with Felzenszwalb-Huttenlocher graph merging, no palette step, paint each region its mean colour, and sweep the scale parameter to find the segmentation that lands on the fixed eval's fidelity of PSNR 28.61 dB.
// At that point, measure what conveying the partition costs, using the best known method for coding a region map: adaptive binary arithmetic coding of the crack edges under a causal context.
// That is the same mechanism MPEG-4 chose over explicit vertex geometry, so it is a fair best-case for the shape idea.
// Colour is given away for free, and so is all serialization overhead.
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"sort"
)

type img struct {
	w, h int
	r    []float64
	g    []float64
	b    []float64
}

func load(path string) (*img, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	bd := src.Bounds()
	w, h := bd.Dx(), bd.Dy()
	im := &img{w: w, h: h, r: make([]float64, w*h), g: make([]float64, w*h), b: make([]float64, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cr, cg, cb, _ := src.At(bd.Min.X+x, bd.Min.Y+y).RGBA()
			i := y*w + x
			im.r[i] = float64(cr >> 8)
			im.g[i] = float64(cg >> 8)
			im.b[i] = float64(cb >> 8)
		}
	}
	return im, nil
}

// union-find with the per-component internal difference the FH criterion needs
type forest struct {
	parent []int
	rank   []int
	size   []int
	intDif []float64
}

func newForest(n int) *forest {
	f := &forest{parent: make([]int, n), rank: make([]int, n), size: make([]int, n), intDif: make([]float64, n)}
	for i := range f.parent {
		f.parent[i] = i
		f.size[i] = 1
	}
	return f
}

func (f *forest) find(x int) int {
	for f.parent[x] != x {
		f.parent[x] = f.parent[f.parent[x]]
		x = f.parent[x]
	}
	return x
}

func (f *forest) join(a, b int, w float64) {
	if f.rank[a] > f.rank[b] {
		a, b = b, a
	}
	f.parent[a] = b
	f.size[b] += f.size[a]
	if f.rank[a] == f.rank[b] {
		f.rank[b]++
	}
	f.intDif[b] = w
}

type edge struct {
	a, b int
	w    float64
}

// segment runs Felzenszwalb-Huttenlocher graph segmentation and returns a label per pixel plus the region count. k is the scale parameter: larger k means larger regions.
func segment(im *img, k float64, minSize int) ([]int, int) {
	n := im.w * im.h
	edges := make([]edge, 0, 2*n)
	dist := func(i, j int) float64 {
		dr, dg, db := im.r[i]-im.r[j], im.g[i]-im.g[j], im.b[i]-im.b[j]
		return math.Sqrt(dr*dr + dg*dg + db*db)
	}
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			i := y*im.w + x
			if x+1 < im.w {
				edges = append(edges, edge{i, i + 1, dist(i, i+1)})
			}
			if y+1 < im.h {
				edges = append(edges, edge{i, i + im.w, dist(i, i+im.w)})
			}
		}
	}
	sort.Slice(edges, func(a, b int) bool { return edges[a].w < edges[b].w })

	f := newForest(n)
	thr := make([]float64, n)
	for i := range thr {
		thr[i] = k
	}
	for _, e := range edges {
		a, b := f.find(e.a), f.find(e.b)
		if a == b {
			continue
		}
		if e.w <= thr[a] && e.w <= thr[b] {
			f.join(a, b, e.w)
			r := f.find(a)
			thr[r] = e.w + k/float64(f.size[r])
		}
	}
	// absorb runt regions so the count is not inflated by single-pixel noise
	for _, e := range edges {
		a, b := f.find(e.a), f.find(e.b)
		if a != b && (f.size[a] < minSize || f.size[b] < minSize) {
			f.join(a, b, e.w)
		}
	}

	labels := make([]int, n)
	remap := map[int]int{}
	for i := 0; i < n; i++ {
		root := f.find(i)
		id, ok := remap[root]
		if !ok {
			id = len(remap)
			remap[root] = id
		}
		labels[i] = id
	}
	return labels, len(remap)
}

// paintMean gives every region its mean colour: the best possible flat-region reconstruction.
func paintMean(im *img, labels []int, nreg int) *img {
	sr := make([]float64, nreg)
	sg := make([]float64, nreg)
	sb := make([]float64, nreg)
	cnt := make([]float64, nreg)
	for i := range labels {
		l := labels[i]
		sr[l] += im.r[i]
		sg[l] += im.g[i]
		sb[l] += im.b[i]
		cnt[l]++
	}
	out := &img{w: im.w, h: im.h, r: make([]float64, len(labels)), g: make([]float64, len(labels)), b: make([]float64, len(labels))}
	for i := range labels {
		l := labels[i]
		out.r[i] = math.Round(sr[l] / cnt[l])
		out.g[i] = math.Round(sg[l] / cnt[l])
		out.b[i] = math.Round(sb[l] / cnt[l])
	}
	return out
}

func psnr(a, b *img) float64 {
	var se float64
	for i := range a.r {
		dr, dg, db := a.r[i]-b.r[i], a.g[i]-b.g[i], a.b[i]-b.b[i]
		se += dr*dr + dg*dg + db*db
	}
	mse := se / float64(len(a.r)*3)
	if mse == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(255*255/mse)
}

// binCtx is an adaptive binary model: the exact cost an arithmetic coder would emit, counted as cross-entropy so no actual bitstream is needed.
type binCtx struct{ c0, c1 []float64 }

func newBinCtx(n int) *binCtx {
	b := &binCtx{c0: make([]float64, n), c1: make([]float64, n)}
	for i := range b.c0 {
		b.c0[i], b.c1[i] = 0.5, 0.5
	}
	return b
}

func (b *binCtx) code(ctx int, bit bool) float64 {
	p1 := b.c1[ctx] / (b.c0[ctx] + b.c1[ctx])
	var cost float64
	if bit {
		cost = -math.Log2(p1)
		b.c1[ctx]++
	} else {
		cost = -math.Log2(1 - p1)
		b.c0[ctx]++
	}
	return cost
}

// crackEdgeBits is the cost of conveying the partition itself, colour excluded.
// The region map is exactly determined by its crack edges: for every pixel, does the neighbour to the right belong to a different region, and does the neighbour below.
// Each of those two binary planes is coded under a causal 5-neighbour context, which is the same family of model MPEG-4 binary shape coding selected over explicit vertex geometry.
func crackEdgeBits(labels []int, w, h int) float64 {
	at := func(x, y int) int { return labels[y*w+x] }
	V := make([]bool, w*h) // boundary between (x,y) and (x+1,y)
	H := make([]bool, w*h) // boundary between (x,y) and (x,y+1)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+1 < w {
				V[y*w+x] = at(x, y) != at(x+1, y)
			}
			if y+1 < h {
				H[y*w+x] = at(x, y) != at(x, y+1)
			}
		}
	}
	get := func(p []bool, x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		if p[y*w+x] {
			return 1
		}
		return 0
	}
	var total float64
	// each plane gets its own model, and sees the other plane in its context:
	// a vertical crack strongly predicts the horizontal cracks that meet it at a corner
	for plane := 0; plane < 2; plane++ {
		self, other := V, H
		if plane == 1 {
			self, other = H, V
		}
		m := newBinCtx(1 << 7)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if plane == 0 && x+1 >= w {
					continue
				}
				if plane == 1 && y+1 >= h {
					continue
				}
				ctx := get(self, x-1, y) |
					get(self, x, y-1)<<1 |
					get(self, x-1, y-1)<<2 |
					get(self, x+1, y-1)<<3 |
					get(self, x-2, y)<<4 |
					get(other, x, y-1)<<5 |
					get(other, x-1, y)<<6
				total += m.code(ctx, self[y*w+x])
			}
		}
	}
	return total
}

// regionColourBits estimates the colour side of the payload, predicting each region's colour from an already-decoded adjacent region so the estimate is not a naive 24 bits per region.
func regionColourBits(im *img, labels []int, nreg int, w, h int) float64 {
	sr := make([]float64, nreg)
	sg := make([]float64, nreg)
	sb := make([]float64, nreg)
	cnt := make([]float64, nreg)
	first := make([]int, nreg)
	for i := range first {
		first[i] = -1
	}
	for i := range labels {
		l := labels[i]
		sr[l] += im.r[i]
		sg[l] += im.g[i]
		sb[l] += im.b[i]
		cnt[l]++
		if first[l] < 0 {
			first[l] = i
		}
	}
	// residual against the region above-left of the region's first pixel, in raster order
	hist := map[int]float64{}
	var n float64
	push := func(v float64) {
		hist[int(math.Round(v))]++
		n++
	}
	for l := 0; l < nreg; l++ {
		i := first[l]
		x, y := i%w, i/w
		var pr, pg, pb float64 = 128, 128, 128
		if x > 0 {
			j := i - 1
			m := labels[j]
			pr, pg, pb = sr[m]/cnt[m], sg[m]/cnt[m], sb[m]/cnt[m]
		} else if y > 0 {
			j := i - w
			m := labels[j]
			pr, pg, pb = sr[m]/cnt[m], sg[m]/cnt[m], sb[m]/cnt[m]
		}
		push(sr[l]/cnt[l] - pr)
		push(sg[l]/cnt[l] - pg)
		push(sb[l]/cnt[l] - pb)
	}
	var ent float64
	for _, c := range hist {
		p := c / n
		ent += -c * math.Log2(p)
	}
	return ent
}

// flatLabels treats every 4-connected run of identical pixels as one region.
// Run on an already-quantized image this recovers exactly the region set the shape pipeline sees, which makes the coder's cost on it directly comparable to the 17.4 KB the order-2 raster coder spends on the same picture.
// That comparison is the fairness check: if this geometry coder were simply weak, it would lose that one too.
func flatLabels(im *img) ([]int, int) {
	n := im.w * im.h
	labels := make([]int, n)
	for i := range labels {
		labels[i] = -1
	}
	next := 0
	stack := make([]int, 0, 1024)
	same := func(a, b int) bool {
		return im.r[a] == im.r[b] && im.g[a] == im.g[b] && im.b[a] == im.b[b]
	}
	for s := 0; s < n; s++ {
		if labels[s] >= 0 {
			continue
		}
		id := next
		next++
		labels[s] = id
		stack = append(stack[:0], s)
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := p%im.w, p/im.w
			if x > 0 && labels[p-1] < 0 && same(p, p-1) {
				labels[p-1] = id
				stack = append(stack, p-1)
			}
			if x+1 < im.w && labels[p+1] < 0 && same(p, p+1) {
				labels[p+1] = id
				stack = append(stack, p+1)
			}
			if y > 0 && labels[p-im.w] < 0 && same(p, p-im.w) {
				labels[p-im.w] = id
				stack = append(stack, p-im.w)
			}
			if y+1 < im.h && labels[p+im.w] < 0 && same(p, p+im.w) {
				labels[p+im.w] = id
				stack = append(stack, p+im.w)
			}
		}
	}
	return labels, next
}

func main() {
	src := os.Args[1]
	im, err := load(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	npx := im.w * im.h
	fmt.Printf("source %s  %dx%d  (%d px)\n\n", src, im.w, im.h, npx)

	if len(os.Args) > 2 && os.Args[2] == "flat" {
		labels, nreg := flatLabels(im)
		gb := crackEdgeBits(labels, im.w, im.h)
		cb := regionColourBits(im, labels, nreg, im.w, im.h)
		fmt.Printf("flat-region labelling (already-quantized input)\n")
		fmt.Printf("  regions      %d  (%.1f px/region)\n", nreg, float64(npx)/float64(nreg))
		fmt.Printf("  geometry     %.1f KB\n", gb/8/1024)
		fmt.Printf("  colour       %.1f KB\n", cb/8/1024)
		fmt.Printf("  total        %.1f KB\n", (gb+cb)/8/1024)
		return
	}

	fmt.Printf("%8s %8s %8s %8s %10s %10s %10s %9s\n",
		"k", "minSize", "regions", "px/reg", "PSNR dB", "geom KB", "colour KB", "total KB")

	for _, cfg := range []struct {
		k       float64
		minSize int
	}{
		{5, 1}, {10, 2}, {20, 2}, {30, 3}, {40, 4},
		{50, 4}, {70, 6}, {100, 8}, {200, 16}, {500, 32},
	} {
		labels, nreg := segment(im, cfg.k, cfg.minSize)
		rec := paintMean(im, labels, nreg)
		p := psnr(im, rec)
		gb := crackEdgeBits(labels, im.w, im.h)
		cb := regionColourBits(im, labels, nreg, im.w, im.h)
		fmt.Printf("%8.0f %8d %8d %8.1f %10.2f %10.1f %10.1f %9.1f\n",
			cfg.k, cfg.minSize, nreg, float64(npx)/float64(nreg), p,
			gb/8/1024, cb/8/1024, (gb+cb)/8/1024)
	}
}
