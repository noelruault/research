// P1: steelman the "raster + region-map sidecar" baseline.
//
// Report 13 claimed the shape coder is 41% smaller than WebP-plus-a-sidecar, but priced the sidecar with OUR wall coder — which falsification #12 showed is not even decodable. A consumer assembling the same capability from off-the-shelf parts would use the best lossless coder they could find for the label map. This emits the label map in several forms so the best one can be measured, rather than assuming ours wins.
//
// Labels fit in 16 bits at every operating point we care about (11,121 << 65536), so the natural generic encodings are a 16-bit grey PNG and the raw plane handed to a strong general compressor.
package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	out := os.Args[2]
	w := int(binary.LittleEndian.Uint32(raw[0:]))
	h := int(binary.LittleEndian.Uint32(raw[4:]))
	n := w * h
	if len(raw) < 8+4*n {
		panic(fmt.Sprintf("short file: %d B for %dx%d", len(raw), w, h))
	}
	lab := make([]int32, n)
	maxID := int32(0)
	for i := 0; i < n; i++ {
		lab[i] = int32(binary.LittleEndian.Uint32(raw[8+4*i:]))
		if lab[i] > maxID {
			maxID = lab[i]
		}
	}
	fmt.Printf("%dx%d, %d px, max label id %d (fits in 16 bits: %v)\n", w, h, n, maxID, maxID < 65536)

	// 1. 16-bit grey PNG — the obvious off-the-shelf lossless container for a label plane.
	g := image.NewGray16(image.Rect(0, 0, w, h))
	for i, l := range lab {
		g.SetGray16(i%w, i/w, color.Gray16{Y: uint16(l)})
	}
	f, _ := os.Create(out + "/labels16.png")
	png.Encode(f, g)
	f.Close()

	// 2. raw 16-bit little-endian plane, for a general compressor to chew on.
	buf := make([]byte, 2*n)
	for i, l := range lab {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(l))
	}
	os.WriteFile(out+"/labels16.raw", buf, 0o644)

	// 3. the same plane as a horizontal delta — label maps are piecewise constant along a row, so
	//    most deltas are zero and a general compressor sees runs instead of magnitudes.
	d := make([]byte, 2*n)
	for y := 0; y < h; y++ {
		prev := int32(0)
		for x := 0; x < w; x++ {
			v := lab[y*w+x]
			binary.LittleEndian.PutUint16(d[2*(y*w+x):], uint16(v-prev))
			prev = v
		}
	}
	os.WriteFile(out+"/labels16_dx.raw", d, 0o644)

	// 4. a bare "is there a boundary here" bit plane, one byte per pixel — the information a decoder
	//    actually needs to rebuild the partition, which is what our own wall coder codes.
	b := make([]byte, n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			var v byte
			if x+1 < w && lab[p] != lab[p+1] {
				v |= 1
			}
			if y+1 < h && lab[p] != lab[p+w] {
				v |= 2
			}
			b[p] = v
		}
	}
	os.WriteFile(out+"/boundary.raw", b, 0o644)
	fmt.Println("wrote labels16.png, labels16.raw, labels16_dx.raw, boundary.raw")
}
