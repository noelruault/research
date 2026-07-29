package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"

	"github.com/noelruault/images"
)

// ---------------------------------------------------------------- image I/O

// Img is a flat float RGB buffer; float so region means and energies stay exact.
type Img struct {
	W, H int
	P    []float64 // 3*W*H, channel-interleaved
}

func (im *Img) at(x, y int) (float64, float64, float64) {
	i := (y*im.W + x) * 3
	return im.P[i], im.P[i+1], im.P[i+2]
}

func load(path string) *Img {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	src, _, err := image.Decode(f)
	must(err)
	b := src.Bounds()
	im := &Img{W: b.Dx(), H: b.Dy(), P: make([]float64, b.Dx()*b.Dy()*3)}
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			i := (y*im.W + x) * 3
			im.P[i], im.P[i+1], im.P[i+2] = float64(r>>8), float64(g>>8), float64(bl>>8)
		}
	}
	return im
}

func (im *Img) writePNG(path string) {
	out := image.NewRGBA(image.Rect(0, 0, im.W, im.H))
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			r, g, b := im.at(x, y)
			out.SetRGBA(x, y, color.RGBA{clamp8(r), clamp8(g), clamp8(b), 255})
		}
	}
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must((&png.Encoder{CompressionLevel: png.BestCompression}).Encode(f, out))
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// psnrSSE converts a summed squared error over all channels into PSNR.
func psnrSSE(sse float64, npix int) float64 {
	if sse <= 0 {
		return 99
	}
	mse := sse / float64(npix*3)
	return 10 * math.Log10(255*255/mse)
}

func sseBetween(a, b *Img) float64 {
	var s float64
	for i := range a.P {
		d := a.P[i] - b.P[i]
		s += d * d
	}
	return s
}

// ------------------------------------------------------- quantize via repo

// Quant is the n-colour index grid produced by the project's own pipeline (pixelize palette + images rect cover), so every number below is anchored to the real thing under review rather than a re-implementation.
type Quant struct {
	W, H  int
	Idx   []uint8 // W*H, row-major
	Pal   [][3]float64
	Rects int
}

func quantize(path string, n int) *Quant {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	src, _, err := image.Decode(f)
	must(err)
	doc, err := images.Convert(context.TODO(), src, images.Options{N: n})
	must(err)

	q := &Quant{W: doc.W, H: doc.H, Idx: make([]uint8, doc.W*doc.H), Rects: len(doc.Rects)}
	for _, r := range doc.Rects {
		for yy := r[1]; yy < r[1]+r[3]; yy++ {
			for xx := r[0]; xx < r[0]+r[2]; xx++ {
				q.Idx[yy*doc.W+xx] = uint8(r[4])
			}
		}
	}
	for _, hexc := range doc.Pal {
		var r, g, b int
		fmt.Sscanf(hexc, "#%02x%02x%02x", &r, &g, &b)
		q.Pal = append(q.Pal, [3]float64{float64(r), float64(g), float64(b)})
	}
	return q
}

func (q *Quant) paint() *Img {
	im := &Img{W: q.W, H: q.H, P: make([]float64, q.W*q.H*3)}
	for i, ix := range q.Idx {
		c := q.Pal[ix]
		im.P[i*3], im.P[i*3+1], im.P[i*3+2] = c[0], c[1], c[2]
	}
	return im
}

// ------------------------------------------------------------- label stats

// components counts 4-connected regions of a label field: the honest "region count", as opposed to the rect count, which splits one region into many rects.
func components(lab []int32, w, h int) int {
	seen := make([]bool, len(lab))
	stack := make([]int32, 0, 1024)
	n := 0
	for i := range lab {
		if seen[i] {
			continue
		}
		n++
		seen[i] = true
		stack = append(stack[:0], int32(i))
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := int(p)%w, int(p)/w
			v := lab[p]
			if x > 0 && !seen[p-1] && lab[p-1] == v {
				seen[p-1] = true
				stack = append(stack, p-1)
			}
			if x < w-1 && !seen[p+1] && lab[p+1] == v {
				seen[p+1] = true
				stack = append(stack, p+1)
			}
			if y > 0 && !seen[p-int32(w)] && lab[p-int32(w)] == v {
				seen[p-int32(w)] = true
				stack = append(stack, p-int32(w))
			}
			if y < h-1 && !seen[p+int32(w)] && lab[p+int32(w)] == v {
				seen[p+int32(w)] = true
				stack = append(stack, p+int32(w))
			}
		}
	}
	return n
}

// crackLen is the total 4-neighbour boundary length (unit crack edges) of a label field.
func crackLen(lab []int32, w, h int) int {
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				n++
			}
			if y < h-1 && lab[p] != lab[p+w] {
				n++
			}
		}
	}
	return n
}

// ------------------------------------------------------------ entropy tools

// adaptiveBytes is the exact output size of an adaptive arithmetic coder over the index field with an order-k causal context (the repo's own measure).
func adaptiveBytes(idx []uint8, w, h, order, ncol int) float64 {
	nctx := 1
	for i := 0; i < order; i++ {
		nctx *= ncol
	}
	counts := make([]uint32, nctx*ncol)
	totals := make([]uint32, nctx)
	bits := 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			var left, up, ul int
			if x > 0 {
				left = int(idx[p-1])
			}
			if y > 0 {
				up = int(idx[p-w])
			}
			if x > 0 && y > 0 {
				ul = int(idx[p-w-1])
			}
			ctx := 0
			switch order {
			case 1:
				ctx = left
			case 2:
				ctx = left*ncol + up
			case 3:
				ctx = (left*ncol+up)*ncol + ul
			}
			s := int(idx[p])
			bits += -math.Log2(float64(counts[ctx*ncol+s]+1) / float64(totals[ctx]+uint32(ncol)))
			counts[ctx*ncol+s]++
			totals[ctx]++
		}
	}
	return bits / 8
}

// binModel is one adaptive binary arithmetic-coding context.
type binModel struct{ n0, n1 uint32 }

func (m *binModel) cost(bit int) float64 {
	t := float64(m.n0+m.n1) + 2
	var c float64
	if bit == 1 {
		c = -math.Log2((float64(m.n1) + 1) / t)
		m.n1++
	} else {
		c = -math.Log2((float64(m.n0) + 1) / t)
		m.n0++
	}
	return c
}

// pngIndexed is the "just pixels plus a colormap" raster reference.
func pngIndexed(q *Quant) []byte {
	pal := make(color.Palette, len(q.Pal))
	for i, c := range q.Pal {
		pal[i] = color.RGBA{clamp8(c[0]), clamp8(c[1]), clamp8(c[2]), 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, q.W, q.H), pal)
	for y := 0; y < q.H; y++ {
		for x := 0; x < q.W; x++ {
			img.SetColorIndex(x, y, q.Idx[y*q.W+x])
		}
	}
	var buf bytes.Buffer
	_ = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img)
	return buf.Bytes()
}

func gzLen(b []byte) int {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	zw.Write(b)
	zw.Close()
	return buf.Len()
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func kb(b float64) string { return fmt.Sprintf("%.1f KB", b/1024) }
