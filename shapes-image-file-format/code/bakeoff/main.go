// Command research runs the encoder bake-off: it quantizes one image, then encodes that identical quantized grid several ways and reports bytes. Every encoder here is lossless of the grid, so they all reconstruct the same pixels (same fidelity) and differ only in size — a clean bytes race in the autoresearch sense (fixed eval, one variable: the encoding).
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"strconv"

	"github.com/noelruault/images"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: research <quantized-source.png> [n]")
		os.Exit(2)
	}
	// psnr mode: compare two decoded images so external codec outputs get a fidelity number.
	if os.Args[1] == "psnr" && len(os.Args) >= 4 {
		a := mustLoad(os.Args[2])
		b := mustLoad(os.Args[3])
		fmt.Printf("%.2f\n", psnrImg(a, b))
		return
	}

	// dump mode: write the raw n-color index map (one byte per pixel) so a shared dictionary can be trained and measured over a corpus of these substrates.
	if os.Args[1] == "dump" && len(os.Args) >= 4 {
		nn := 16
		if len(os.Args) >= 5 {
			nn, _ = strconv.Atoi(os.Args[4])
		}
		d, err := images.Convert(context.TODO(), mustLoad(os.Args[2]), images.Options{N: nn})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		g := rebuildGrid(d)
		out := make([]byte, d.W*d.H)
		i := 0
		for y := 0; y < d.H; y++ {
			for x := 0; x < d.W; x++ {
				out[i] = byte(g[x][y])
				i++
			}
		}
		_ = os.WriteFile(os.Args[3], out, 0o644)
		return
	}

	n := 16
	if len(os.Args) >= 3 {
		n, _ = strconv.Atoi(os.Args[2])
	}

	orig := mustLoad(os.Args[1])
	doc, err := images.Convert(context.TODO(), orig, images.Options{N: n})
	if err != nil {
		fmt.Fprintln(os.Stderr, "convert:", err)
		os.Exit(1)
	}

	grid := rebuildGrid(doc)       // [x][y] palette index, from the rect cover
	quant := paintQuant(doc, grid) // the reconstructed quantized image
	psnr := psnrRGBA(orig, quant)  // fidelity shared by every encoder below

	svg := doc.SVG()
	bin := binaryEncode(doc)
	quad := quadtreeEncode(grid, doc.W, doc.H)
	ipng := indexedPNG(doc, grid)

	fmt.Printf("eval: %dx%d, n=%d, %d rects, PSNR(quant vs original)=%.2f dB\n\n",
		doc.W, doc.H, n, len(doc.Rects), psnr)
	fmt.Printf("%-26s %12s %12s\n", "encoder (all lossless-of-grid)", "raw", "gzipped")
	row := func(name string, b []byte) {
		fmt.Printf("%-26s %12s %12s\n", name, human(len(b)), human(images.GzipLen(b)))
	}
	row("A. SVG rect-cover", svg)
	row("B. binary varint rects", bin)
	row("C. quadtree bitstream", quad)
	row("D. indexed PNG (stdlib)", ipng) // PNG is already entropy-coded; gz barely helps
	row("E. row-RLE (implicit x)", rleBytes(grid, doc.W, doc.H))

	// F..H: geometry dropped entirely, WebP's "predict then entropy-code" idea.
	// These print the arithmetic-coder floor directly (no gzip applies).
	fmt.Printf("%-26s %12s %12s\n", "F. entropy order-0", human(int(adaptiveEntropyBytes(grid, doc.W, doc.H, 0, len(doc.Pal)))), "-")
	fmt.Printf("%-26s %12s %12s\n", "G. entropy order-1 (left)", human(int(adaptiveEntropyBytes(grid, doc.W, doc.H, 1, len(doc.Pal)))), "-")
	fmt.Printf("%-26s %12s %12s\n", "H. entropy order-2 (left,up)", human(int(adaptiveEntropyBytes(grid, doc.W, doc.H, 2, len(doc.Pal)))), "-")
	fmt.Printf("%-26s %12s %12s\n", "I. entropy order-3 (+upleft)", human(int(adaptiveEntropyBytes(grid, doc.W, doc.H, 3, len(doc.Pal)))), "-")
	fmt.Println("\nwall: WebP-lossless 25.2 KB, PNG 27.2 KB (same pixels)")

	// Emit the quantized grid as a plain PNG so external codecs benchmark the same pixels (see the bash step).
	_ = os.WriteFile("quant.png", ipng, 0o644)
}

// --- encoders -------------------------------------------------------------

// binaryEncode packs the palette and the rect list as little varints: raw numbers, no text, and gzip mops up the residual redundancy.
// This is the v2 ".shapes" candidate from shapes.go.
func binaryEncode(d *images.Doc) []byte {
	var b bytes.Buffer
	putUvarint(&b, uint64(d.W))
	putUvarint(&b, uint64(d.H))
	putUvarint(&b, uint64(len(d.Pal)))
	for _, hexc := range d.Pal {
		r, g, bl := parseHex(hexc)
		b.WriteByte(r)
		b.WriteByte(g)
		b.WriteByte(bl)
	}
	putUvarint(&b, uint64(len(d.Rects)))
	for _, r := range d.Rects {
		for _, v := range r {
			putUvarint(&b, uint64(v))
		}
	}
	return b.Bytes()
}

// quadtreeEncode splits the grid into uniform quadrants recursively: 1 bit per node (leaf vs split) plus the color byte at leaves. Great on big flat areas, degenerate on high-detail photos where it splits to single pixels.
func quadtreeEncode(grid [][]int, w, h int) []byte {
	bw := &bitWriter{}
	var rec func(x, y, ww, hh int)
	rec = func(x, y, ww, hh int) {
		if ww <= 0 || hh <= 0 { // an odd split can hand a child a zero-width or zero-height slice; it covers nothing
			return
		}
		c, uniform := regionColor(grid, x, y, ww, hh)
		if uniform || (ww == 1 && hh == 1) {
			bw.bit(0)
			bw.byte8(byte(c))
			return
		}
		bw.bit(1)
		mx, my := (ww+1)/2, (hh+1)/2
		rec(x, y, mx, my)
		rec(x+mx, y, ww-mx, my)
		rec(x, y+my, mx, hh-my)
		rec(x+mx, y+my, ww-mx, hh-my)
	}
	rec(0, 0, w, h)
	return bw.done()
}

// indexedPNG is the "just pixels + a color map" baseline: a paletted PNG, which is stdlib DEFLATE over the index map. It is the honest raster reference the shape encoders must beat to justify existing.
func indexedPNG(d *images.Doc, grid [][]int) []byte {
	pal := make(color.Palette, len(d.Pal))
	for i, hexc := range d.Pal {
		r, g, b := parseHex(hexc)
		pal[i] = color.RGBA{R: r, G: g, B: b, A: 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, d.W, d.H), pal)
	for x := 0; x < d.W; x++ {
		for y := 0; y < d.H; y++ {
			img.SetColorIndex(x, y, uint8(grid[x][y]))
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	_ = enc.Encode(&buf, img)
	return buf.Bytes()
}

// --- grid + fidelity ------------------------------------------------------

func rebuildGrid(d *images.Doc) [][]int {
	g := make([][]int, d.W)
	for x := range g {
		g[x] = make([]int, d.H)
	}
	for _, r := range d.Rects {
		for yy := r[1]; yy < r[1]+r[3]; yy++ {
			for xx := r[0]; xx < r[0]+r[2]; xx++ {
				g[xx][yy] = r[4]
			}
		}
	}
	return g
}

func paintQuant(d *images.Doc, grid [][]int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, d.W, d.H))
	for x := 0; x < d.W; x++ {
		for y := 0; y < d.H; y++ {
			r, g, b := parseHex(d.Pal[grid[x][y]])
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func psnrImg(a, b image.Image) float64 {
	ab, bb := a.Bounds(), b.Bounds()
	w := min(ab.Dx(), bb.Dx())
	h := min(ab.Dy(), bb.Dy())
	var se float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := a.At(ab.Min.X+x, ab.Min.Y+y).RGBA()
			r2, g2, b2, _ := b.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
			dr := float64(int(r1>>8) - int(r2>>8))
			dg := float64(int(g1>>8) - int(g2>>8))
			db := float64(int(b1>>8) - int(b2>>8))
			se += dr*dr + dg*dg + db*db
		}
	}
	mse := se / float64(w*h*3)
	if mse == 0 {
		return 99
	}
	return 10 * math.Log10(255*255/mse)
}

func psnrRGBA(a image.Image, b *image.RGBA) float64 {
	ab := a.Bounds()
	w := min(ab.Dx(), b.Bounds().Dx())
	h := min(ab.Dy(), b.Bounds().Dy())
	var se float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := a.At(ab.Min.X+x, ab.Min.Y+y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			dr := float64(int(r1>>8) - int(r2>>8))
			dg := float64(int(g1>>8) - int(g2>>8))
			db := float64(int(b1>>8) - int(b2>>8))
			se += dr*dr + dg*dg + db*db
		}
	}
	mse := se / float64(w*h*3)
	if mse == 0 {
		return 99
	}
	return 10 * math.Log10(255*255/mse)
}

// --- small helpers --------------------------------------------------------

func regionColor(grid [][]int, x, y, w, h int) (int, bool) {
	c := grid[x][y]
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if grid[xx][yy] != c {
				return 0, false
			}
		}
	}
	return c, true
}

type bitWriter struct {
	buf  []byte
	cur  byte
	nbit uint8
}

func (w *bitWriter) bit(b byte) {
	w.cur |= (b & 1) << (7 - w.nbit)
	w.nbit++
	if w.nbit == 8 {
		w.buf = append(w.buf, w.cur)
		w.cur, w.nbit = 0, 0
	}
}

func (w *bitWriter) byte8(v byte) {
	for i := 0; i < 8; i++ {
		w.bit((v >> (7 - i)) & 1)
	}
}

func (w *bitWriter) done() []byte {
	if w.nbit > 0 {
		w.buf = append(w.buf, w.cur)
	}
	return w.buf
}

func putUvarint(b *bytes.Buffer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	nb := binary.PutUvarint(tmp[:], v)
	b.Write(tmp[:nb])
}

func parseHex(s string) (r, g, b uint8) {
	if len(s) == 7 && s[0] == '#' {
		rv, _ := strconv.ParseUint(s[1:3], 16, 8)
		gv, _ := strconv.ParseUint(s[3:5], 16, 8)
		bv, _ := strconv.ParseUint(s[5:7], 16, 8)
		return uint8(rv), uint8(gv), uint8(bv)
	}
	return 0, 0, 0
}

func human(nbytes int) string {
	if nbytes < 1024 {
		return strconv.Itoa(nbytes) + " B"
	}
	if nbytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(nbytes)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(nbytes)/(1024*1024))
}

func mustLoad(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return img
}
