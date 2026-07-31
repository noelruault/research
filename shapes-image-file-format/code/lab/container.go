package main

// container.go — P4: an actual file, and a decoder that reads nothing else.
//
// Every byte figure in twenty reports is a cross-entropy: `binModel.cost` accumulates -log2(p) and returns a float.
// Nothing in this codebase had ever emitted a byte, so the two headline claims — +0.91% against WebP at 28.51 dB and
// -1.2% at the capability point — compared an idealised number against a real WebP file with a real container.
// This file closes that gap. It writes a self-contained `.shpc` and reconstructs the image from that file alone.
//
// What goes in:
//
//  1. WALLS — the two crack-edge planes under the interleaved V/Hz schedule (`interAsym` in crossplane.go), which is
//     the coder both headlines quote and which `lab wallcheck` confirms is decodable. Driven through a binary range
//     coder by exactly the `binModel` statistics `priceVariant` prices, so the emitted size is the cross-entropy plus
//     coder inefficiency and nothing else. NOT `caeBytes`' two-pass schedule.
//  2. COLOUR — recolour.go's `a` arm: the RCT residual with report 12's two chroma coefficients, planar, brotli -q11.
//     That stream is already real; the container carries it as a chunk and pays for the 8 bytes of coefficients.
//  3. HEADER — magic, version, width, height, region count, both chunk lengths. Paid for in the total.
//
// The partition is never re-derived. A published render's exact 4-connected partition IS that mark's partition — the recovery trick reports 09 through 20 all use — so `p4enc <render.png>` encodes the mark the reports quote.
//
// The deliverable is the round trip, not the file size: `p4dec` rebuilds the partition and the region colours from the file and asserts the result is bit-identical to the published render. If it is not, the file size means nothing.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------- range coder
//
// Carry-propagating binary range coder in the LZMA shape: 32-bit range, 32-bit low plus a one-byte cache and a run count of pending 0xFF bytes. The leading output byte is the carry catcher and is always 0, because the encoder's interval is always a subinterval of the initial one, so no carry can propagate out of the first real digit.
// Both sides assert that.
//
// The interval split is a pure function of the model counts and is computed identically by encoder and decoder, at full precision rather than through a quantised probability table, so the emitted length tracks
// `binModel.cost`'s -log2((n+1)/t) to within the 32-bit division.

const p4Top = 1 << 24 // renormalise below this, so range is always >= 2^24 when a bit is coded

// p4Split is the interval boundary for bit 0. Reads the model, never writes it.
func p4Split(rng uint32, m *binModel) uint32 {
	t := uint64(m.n0) + uint64(m.n1) + 2
	b := uint32(uint64(rng) * (uint64(m.n0) + 1) / t)
	// rng >= 2^24 and t <= 2*w*h+2, so b is comfortably inside (0, rng) on any image this study uses; the clamps are here so the coder degrades into a legal (merely inefficient) split rather than a corrupt one if it is not.
	if b < 1 {
		b = 1
	}
	if b > rng-1 {
		b = rng - 1
	}
	return b
}

type p4Enc struct {
	low     uint64
	rng     uint32
	cache   byte
	cacheSz int64
	out     []byte
}

func p4NewEnc() *p4Enc { return &p4Enc{rng: 0xFFFFFFFF, cacheSz: 1} }

func (e *p4Enc) shiftLow() {
	if e.low < 0xFF000000 || e.low > 0xFFFFFFFF {
		carry := byte(e.low >> 32)
		for ; e.cacheSz > 0; e.cacheSz-- {
			e.out = append(e.out, e.cache+carry)
			e.cache = 0xFF
		}
		e.cache = byte(e.low >> 24)
	}
	e.cacheSz++
	e.low = (e.low << 8) & 0xFFFFFFFF
}

// bit codes one bit against a model and updates it exactly as binModel.cost does, so the statistics the reports priced and the statistics the file is built from are the same statistics.
func (e *p4Enc) bit(m *binModel, v int) {
	b := p4Split(e.rng, m)
	if v == 0 {
		e.rng = b
		m.n0++
	} else {
		e.low += uint64(b)
		e.rng -= b
		m.n1++
	}
	for e.rng < p4Top {
		e.shiftLow()
		e.rng <<= 8
	}
}

func (e *p4Enc) flush() []byte {
	for i := 0; i < 5; i++ {
		e.shiftLow()
	}
	if len(e.out) == 0 || e.out[0] != 0 {
		fmt.Fprintln(os.Stderr, "fatal: range coder carry escaped into the leading byte")
		os.Exit(1)
	}
	return e.out
}

type p4Dec struct {
	rng  uint32
	code uint32
	in   []byte
	pos  int
}

func p4NewDec(in []byte) *p4Dec {
	d := &p4Dec{rng: 0xFFFFFFFF, in: in}
	if len(in) < 5 || in[0] != 0 {
		fmt.Fprintln(os.Stderr, "fatal: wall chunk is not a range-coded stream")
		os.Exit(1)
	}
	d.pos = 1
	for i := 0; i < 4; i++ {
		d.code = d.code<<8 | uint32(d.next())
	}
	return d
}

// next reads past the end as zeros: the encoder's flush writes only what the decoder needs, and the last few renormalisations may reach beyond it.
func (d *p4Dec) next() byte {
	if d.pos >= len(d.in) {
		return 0
	}
	b := d.in[d.pos]
	d.pos++
	return b
}

func (d *p4Dec) bit(m *binModel) int {
	b := p4Split(d.rng, m)
	v := 0
	if d.code < b {
		d.rng = b
		m.n0++
	} else {
		d.code -= b
		d.rng -= b
		m.n1++
		v = 1
	}
	for d.rng < p4Top {
		d.rng <<= 8
		d.code = d.code<<8 | uint32(d.next())
	}
	return v
}

// ------------------------------------------------------------- the wall stream

// p4Variant is the coder both headlines use. Fetched from crossplane.go's own table rather than copied, so the container cannot silently drift away from the variant that was priced.
func p4Variant() wallVariant {
	for _, v := range variants() {
		if v.name == "interAsym" {
			if v.mode != interVH {
				break
			}
			if err := v.checkCausal(); err != nil {
				fmt.Fprintln(os.Stderr, "fatal:", err)
				os.Exit(1)
			}
			return v
		}
	}
	fmt.Fprintln(os.Stderr, "fatal: interAsym is missing or no longer interleaved")
	os.Exit(1)
	return wallVariant{}
}

// p4Scan walks the interleaved schedule: one raster pass, V(x,y) then Hz(x,y) at each pixel.
// Encoder and decoder share it, which is what makes the two orders identical by construction.
func p4Scan(w, h int, step func(pl, x, y int)) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w-1 {
				step(0, x, y)
			}
			if y < h-1 {
				step(1, x, y)
			}
		}
	}
}

func p4Ctx(taps []tap, V, Hz []byte, w, h, x, y int) int {
	ctx := 0
	for i, t := range taps {
		xx, yy := x+t.dx, y+t.dy
		if xx < 0 || yy < 0 || xx >= w || yy >= h {
			continue
		}
		if t.pl == 0 {
			ctx |= int(V[yy*w+xx]) << uint(i)
		} else {
			ctx |= int(Hz[yy*w+xx]) << uint(i)
		}
	}
	return ctx
}

func p4EncodeWalls(V, Hz []byte, w, h int) []byte {
	v := p4Variant()
	mv := make([]binModel, 1<<uint(len(v.tapsV)))
	mh := make([]binModel, 1<<uint(len(v.tapsH)))
	e := p4NewEnc()
	p4Scan(w, h, func(pl, x, y int) {
		if pl == 0 {
			e.bit(&mv[p4Ctx(v.tapsV, V, Hz, w, h, x, y)], int(V[y*w+x]))
		} else {
			e.bit(&mh[p4Ctx(v.tapsH, V, Hz, w, h, x, y)], int(Hz[y*w+x]))
		}
	})
	return e.flush()
}

// p4DecodeWalls rebuilds both crack planes from the chunk alone. The context is read out of the planes it is filling in, so any tap that is not causal under this schedule would produce a different context here than at encode time and the round trip would fail — the same property `lab wallcheck` asserts, now enforced by the file itself.
func p4DecodeWalls(chunk []byte, w, h int) (V, Hz []byte) {
	v := p4Variant()
	mv := make([]binModel, 1<<uint(len(v.tapsV)))
	mh := make([]binModel, 1<<uint(len(v.tapsH)))
	V, Hz = make([]byte, w*h), make([]byte, w*h)
	d := p4NewDec(chunk)
	p4Scan(w, h, func(pl, x, y int) {
		if pl == 0 {
			V[y*w+x] = byte(d.bit(&mv[p4Ctx(v.tapsV, V, Hz, w, h, x, y)]))
		} else {
			Hz[y*w+x] = byte(d.bit(&mh[p4Ctx(v.tapsH, V, Hz, w, h, x, y)]))
		}
	})
	return V, Hz
}

// p4Label rebuilds the partition from the crack planes. Region ids follow raster first appearance, which is exactly what relabelComponents produces and therefore exactly the order the colour stream is written in.
func p4Label(V, Hz []byte, w, h int) ([]int32, int) {
	out := make([]int32, w*h)
	for i := range out {
		out[i] = -1
	}
	var next int32
	stack := make([]int32, 0, 1024)
	for i := range out {
		if out[i] >= 0 {
			continue
		}
		id := next
		next++
		out[i] = id
		stack = append(stack[:0], int32(i))
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := int(p)%w, int(p)/w
			try := func(q int32) {
				if out[q] < 0 {
					out[q] = id
					stack = append(stack, q)
				}
			}
			if x > 0 && V[p-1] == 0 {
				try(p - 1)
			}
			if x < w-1 && V[p] == 0 {
				try(p + 1)
			}
			if y > 0 && Hz[int(p)-w] == 0 {
				try(p - int32(w))
			}
			if y < h-1 && Hz[p] == 0 {
				try(p + int32(w))
			}
		}
	}
	return out, int(next)
}

// ----------------------------------------------------------- the colour stream

// p4ColourStream builds recolour.go's `a` arm on this partition: the planar RCT residual byte stream and the two transmitted chroma coefficients. Byte-identical to what `RCDUMP=... lab recolour` writes, which is the stream report 15's brotli column and report 17's 8,699 B were measured on.
func p4ColourStream(lab []int32, cols [][3]float64, w, h int) ([]byte, [2]float32) {
	n := len(cols)
	a := flBuildAdj(lab, n, w, h)
	fa := rcFitA(a, cols)
	dec := make([][3]float64, n)
	s := make([]byte, 0, 3*n)
	for r := 0; r < n; r++ {
		wm := rcWmean(a, dec, r)
		tG := int(cols[r][1] - wm[1])
		pR, pB := rcPredA(wm, tG, fa)
		s = append(s, byte(tG), byte(int(cols[r][0])-pR), byte(int(cols[r][2])-pB))
		dec[r] = cols[r] // lossless on region colours, so the decoded neighbour is the true colour
	}
	return flPlanar(s), fa
}

// p4ColourDecode is rcDec's `a` branch: partition plus stream plus the 8 bytes, nothing else.
func p4ColourDecode(lab []int32, n int, raw []byte, fa [2]float32, w, h int) [][3]float64 {
	if len(raw) != 3*n {
		fmt.Fprintf(os.Stderr, "fatal: colour stream is %d B, expected %d for %d regions\n", len(raw), 3*n, n)
		os.Exit(1)
	}
	a := flBuildAdj(lab, n, w, h)
	dec := make([][3]float64, n)
	for r := 0; r < n; r++ {
		pred := rcWmean(a, dec, r)
		g := int(raw[r])
		gr := (int(pred[1]) + g) & 255
		dec[r][1] = float64(gr)
		pR, pB := rcPredA(pred, gr-int(pred[1]), fa)
		dec[r][0] = float64((pR + int(raw[n+r])) & 255)
		dec[r][2] = float64((pB + int(raw[2*n+r])) & 255)
	}
	return dec
}

// p4Brotli shells out rather than vendoring a compressor: the whole point of report 15's colour column is that it is a real off-the-shelf stream, and `brotli -q 11` on stdin is what produced every published colour figure.
func p4Brotli(args []string, in []byte) []byte {
	cmd := exec.Command("brotli", args...)
	cmd.Stdin = bytes.NewReader(in)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	must(cmd.Run())
	return out.Bytes()
}

// ------------------------------------------------------------------ the format
//
//	 offset  bytes  field
//	 0       4      magic "SHPC"
//	 4       1      version — 1 has no alpha fields at all, 2 adds the two marked below
//	 5       v      uvarint width
//	         v      uvarint height
//	         v      uvarint region count
//	         v      uvarint wall chunk length
//	         v      uvarint colour chunk length
//	         1      alpha mode           (v2 only)  0 none · 1 per-region flat · 2 per-pixel plane
//	         v      uvarint alpha length (v2 only, and only when the mode is not 0)
//	         8      chroma coefficients aR, aB as little-endian float32
//	         ...    wall chunk   — range-coded crack planes, interleaved V/Hz schedule
//	         ...    colour chunk — brotli -q11 of the planar RCT residual
//	         ...    alpha chunk  — brotli -q11 of one byte per region, in region order
//
// The mode field is DESIGN-ALPHA.md approach C: one reserved byte now, so adding the per-pixel
// plane later is a mode value rather than another version bump. Mode 2 is deliberately NOT
// implemented — research item A3 has not yet shown that real game art needs it.
//
// v2 costs one byte more than v1 on an opaque image (the mode field), and report 21's ~20 B
// overhead figures were measured on v1. Mode 1 additionally pays the alpha chunk, which on a
// mostly-binary sprite alpha is nearly free after brotli.
//
// The region count is derivable from the decoded partition; it is transmitted anyway, as a 2-3 byte consistency check that catches a truncated or mismatched wall chunk before the colour stream is misread. It is paid for.

const p4Magic = "SHPC"
const p4Version = 3

// Selection modes. A selection is an instance id per region: an index set where 0 is background
// and each region belongs to exactly one instance, which is the shape of data every
// instance-segmentation model emits.
//
// The point is WHERE this is computed. A learned model supplies the semantics once, at encode
// time; the file then carries the answer as region ids, so every client gets the identical
// selection with no model and no per-device variation. Report 28 measured the alternative: a
// client re-deriving a mask keeps only 24-40% of its boundaries across two deliveries of the
// same image.
const (
	p4SelNone     = 0 // no selection stored
	p4SelInstance = 1 // one instance id per region, 0 = background
	// Mode 2 additionally records how much to trust the selection and which model produced it.
	// Mode 1 asserts a selection with nothing attached, which a reader cannot falsify; the claim
	// "region #4,211 means the same thing on every device forever" is only checkable if the file
	// says what drew it. Every segmentation model worth using reports a confidence, and storing
	// it costs one byte.
	//
	// Chunk payload, brotli'd as a whole:
	//   1 byte    confidence, quantised 0..255 from the model's 0..1
	//   uvarint   producer length
	//   ...       producer, e.g. "u2net/1"
	//   n bytes   instance id per region, 0 = background
	p4SelInstanceMeta = 2
)

const (
	p4AlphaNone     = 0 // no alpha channel in the source
	p4AlphaPerRegin = 1 // one flat alpha per region — approach A
	// 2 is reserved for the per-pixel plane (approach B) and is not implemented.
)

func p4PutUvarint(b *bytes.Buffer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	b.Write(tmp[:binary.PutUvarint(tmp[:], v)])
}

// p4enc <render.png> <out.shpc>
func p4EncCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lab p4enc <render.png> <out.shpc>")
		os.Exit(2)
	}
	im := load(args[0])
	w, h := im.W, im.H
	lab, cols, _ := exactPartition(im)
	n := len(cols)
	// Optional third argument: a model's foreground mask (greyscale PNG, bright = subject),
	// snapped onto the partition by per-region majority vote and stored as instance ids.
	selMode := byte(p4SelNone)
	var selChunk []byte
	if len(args) > 2 {
		ids := p4SnapSelection(args[2], lab, n, w, h)
		selMode = p4SelInstance
		payload := ids
		// Optional 4th arg: a file holding the model's confidence (0..1). 5th: the producer name.
		if len(args) > 3 {
			selMode = p4SelInstanceMeta
			conf := 0.0
			if b, err := os.ReadFile(args[3]); err == nil {
				conf, _ = strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
			}
			producer := "unknown"
			if len(args) > 4 {
				producer = args[4]
			}
			var buf bytes.Buffer
			buf.WriteByte(byte(math.Round(math.Max(0, math.Min(1, conf)) * 255)))
			p4PutUvarint(&buf, uint64(len(producer)))
			buf.WriteString(producer)
			buf.Write(ids)
			payload = buf.Bytes()
		}
		selChunk = p4Brotli([]string{"-q", "11", "-c", "-"}, payload)
	}
	alphas := regionAlphas(im, lab, n)
	mode := byte(p4AlphaNone)
	var alphaChunk []byte
	if alphas != nil {
		mode = p4AlphaPerRegin
		raw := make([]byte, n)
		for i, a := range alphas {
			raw[i] = clamp8(a)
		}
		alphaChunk = p4Brotli([]string{"-q", "11", "-c", "-"}, raw)
	}

	V, Hz := crackPlanes(lab, w, h)
	xeV, xeH := priceVariant(p4Variant(), V, Hz, w, h) // the number the reports quote
	wall := p4EncodeWalls(V, Hz, w, h)

	planar, fa := p4ColourStream(lab, cols, w, h)
	colour := p4Brotli([]string{"-q", "11", "-c", "-"}, planar)

	var hdr bytes.Buffer
	hdr.WriteString(p4Magic)
	hdr.WriteByte(p4Version)
	p4PutUvarint(&hdr, uint64(w))
	p4PutUvarint(&hdr, uint64(h))
	p4PutUvarint(&hdr, uint64(n))
	p4PutUvarint(&hdr, uint64(len(wall)))
	p4PutUvarint(&hdr, uint64(len(colour)))
	hdr.WriteByte(mode)
	if mode != p4AlphaNone {
		p4PutUvarint(&hdr, uint64(len(alphaChunk)))
	}
	hdr.WriteByte(selMode)
	if selMode != p4SelNone {
		p4PutUvarint(&hdr, uint64(len(selChunk)))
	}
	var coef [8]byte
	binary.LittleEndian.PutUint32(coef[0:], math.Float32bits(fa[0]))
	binary.LittleEndian.PutUint32(coef[4:], math.Float32bits(fa[1]))
	hdr.Write(coef[:])

	file := append(append(append(append(hdr.Bytes(), wall...), colour...), alphaChunk...), selChunk...)
	must(os.WriteFile(args[1], file, 0o644))

	xe := xeV + xeH
	if selMode != p4SelNone {
		fmt.Printf("  select  brotli -q11   %10d B   raw %8d B   mode %d (instance id per region)\n",
			len(selChunk), n, selMode)
	}
	est := xe + float64(len(colour)) + float64(len(alphaChunk)) + float64(len(selChunk)) + 8
	fmt.Printf("p4enc %s %dx%d regions=%d -> %s\n", args[0], w, h, n, args[1])
	fmt.Printf("  walls   cross-entropy %10.2f B   coded %8d B   coder overhead %+6d B (%+.3f%%)\n",
		xe, len(wall), len(wall)-int(math.Round(xe)), 100*(float64(len(wall))-xe)/xe)
	fmt.Printf("  colour  brotli -q11   %10d B   raw %8d B   coefficients 8 B (inside the header)\n",
		len(colour), len(planar))
	if mode != p4AlphaNone {
		fmt.Printf("  alpha   brotli -q11   %10d B   raw %8d B   mode %d (per-region flat)\n",
			len(alphaChunk), n, mode)
	}
	fmt.Printf("  header  %d B (magic 4, version %d, %d varint, mode 1, coefficients 8)\n",
		hdr.Len(), p4Version, hdr.Len()-14)
	fmt.Printf("  estimate (walls xentropy + colour brotli + 8) = %.2f B\n", est)
	fmt.Printf("  FILE                                          = %d B\n", len(file))
	fmt.Printf("  overhead                                      = %+.2f B (%+.4f%%)\n",
		float64(len(file))-est, 100*(float64(len(file))-est)/est)
}

// p4dec <in.shpc> <out.png> [ref.png]
// Reads the file and nothing else. With a reference it asserts the reconstruction is bit-identical, which is the deliverable: without it the file size above is meaningless.
func p4DecCmd(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: lab p4dec <in.shpc> <out.png> [ref.png]")
		os.Exit(2)
	}
	file, err := os.ReadFile(args[0])
	must(err)
	if len(file) < 5 || string(file[:4]) != p4Magic || (file[4] < 1 || file[4] > 3) {
		fmt.Fprintln(os.Stderr, "fatal: not a SHPC v1, v2 or v3 file")
		os.Exit(1)
	}
	version := file[4]
	r := bytes.NewReader(file[5:])
	rd := func() int {
		v, err := binary.ReadUvarint(r)
		must(err)
		return int(v)
	}
	w, h, nHdr, wallLen, colLen := rd(), rd(), rd(), rd(), rd()
	// v1 predates alpha entirely, so it has neither field and decodes as mode 0.
	mode, alphaLen := byte(p4AlphaNone), 0
	if version >= 2 {
		b, err := r.ReadByte()
		must(err)
		mode = b
		if mode != p4AlphaNone {
			alphaLen = rd()
		}
	}
	selMode, selLen := byte(p4SelNone), 0
	if version >= 3 {
		b, err := r.ReadByte()
		must(err)
		selMode = b
		if selMode != p4SelNone {
			selLen = rd()
		}
	}
	if selMode > p4SelInstanceMeta {
		fmt.Fprintf(os.Stderr, "fatal: selection mode %d is not implemented\n", selMode)
		os.Exit(1)
	}
	if mode > p4AlphaPerRegin {
		fmt.Fprintf(os.Stderr, "fatal: alpha mode %d is not implemented (only 0 and 1)\n", mode)
		os.Exit(1)
	}
	var coef [8]byte
	if _, err := r.Read(coef[:]); err != nil {
		must(err)
	}
	fa := [2]float32{
		math.Float32frombits(binary.LittleEndian.Uint32(coef[0:])),
		math.Float32frombits(binary.LittleEndian.Uint32(coef[4:])),
	}
	off := len(file) - r.Len()
	if off+wallLen+colLen+alphaLen+selLen != len(file) {
		fmt.Fprintf(os.Stderr, "fatal: chunk lengths %d+%d+%d+%d do not fill %d B after a %d B header\n",
			wallLen, colLen, alphaLen, selLen, len(file), off)
		os.Exit(1)
	}

	V, Hz := p4DecodeWalls(file[off:off+wallLen], w, h)
	lab, n := p4Label(V, Hz, w, h)
	if n != nHdr {
		fmt.Fprintf(os.Stderr, "fatal: decoded %d regions, header says %d\n", n, nHdr)
		os.Exit(1)
	}
	planar := p4Brotli([]string{"-d", "-c", "-"}, file[off+wallLen:off+wallLen+colLen])
	dec := p4ColourDecode(lab, n, planar, fa, w, h)

	im := &Img{W: w, H: h, P: make([]float64, w*h*3)}
	for p, l := range lab {
		im.P[p*3], im.P[p*3+1], im.P[p*3+2] = dec[l][0], dec[l][1], dec[l][2]
	}
	if mode == p4AlphaPerRegin {
		raw := p4Brotli([]string{"-d", "-c", "-"}, file[off+wallLen+colLen:])
		if len(raw) != n {
			fmt.Fprintf(os.Stderr, "fatal: alpha chunk holds %d values, partition has %d regions\n", len(raw), n)
			os.Exit(1)
		}
		im.A = make([]float64, w*h)
		for p, l := range lab {
			im.A[p] = float64(raw[l])
		}
	}
	if selMode != p4SelNone {
		raw := p4Brotli([]string{"-d", "-c", "-"}, file[off+wallLen+colLen+alphaLen:])
		if selMode == p4SelInstanceMeta {
			if len(raw) < 2 {
				fmt.Fprintln(os.Stderr, "fatal: selection chunk too short for its metadata")
				os.Exit(1)
			}
			conf := float64(raw[0]) / 255
			plen, adv := binary.Uvarint(raw[1:])
			if adv <= 0 || 1+adv+int(plen) > len(raw) {
				fmt.Fprintln(os.Stderr, "fatal: selection producer field is malformed")
				os.Exit(1)
			}
			producer := string(raw[1+adv : 1+adv+int(plen)])
			fmt.Printf("  selection confidence %.3f, produced by %q\n", conf, producer)
			raw = raw[1+adv+int(plen):]
		}
		if len(raw) != n {
			fmt.Fprintf(os.Stderr, "fatal: selection holds %d ids, partition has %d regions\n", len(raw), n)
			os.Exit(1)
		}
		inst := map[byte]int{}
		for _, v := range raw {
			inst[v]++
		}
		fmt.Printf("  selection: %d instance(s) + background; regions per id %v\n", len(inst)-1, inst)
	}
	im.writePNG(args[1])
	fmt.Printf("p4dec %s -> %s: v%d, %dx%d, %d regions, alpha mode %d, selection mode %d, a=(%.5f,%.5f)\n",
		args[0], args[1], version, w, h, n, mode, selMode, fa[0], fa[1])

	if len(args) < 3 {
		return
	}
	ref := load(args[2])
	if ref.W != w || ref.H != h {
		fmt.Fprintln(os.Stderr, "fatal: reference is a different size")
		os.Exit(1)
	}
	bad, maxd := 0, 0.0
	for i := range ref.P {
		if d := math.Abs(im.P[i] - ref.P[i]); d > 0 {
			bad++
			if d > maxd {
				maxd = d
			}
		}
	}
	// Alpha is checked on the same footing as colour. Without this an alpha plane that decoded to
	// garbage would still print EXACT, since nothing else in this function reads it.
	for p := 0; p < w*h; p++ {
		if d := math.Abs(im.alphaAt(p) - ref.alphaAt(p)); d > 0 {
			bad++
			if d > maxd {
				maxd = d
			}
		}
	}
	fmt.Printf("  round trip against %s: %d wrong samples of %d, max |delta| %.0f\n", args[2], bad, len(ref.P), maxd)
	if bad != 0 {
		fmt.Fprintln(os.Stderr, "fatal: the file does not reconstruct the render")
		os.Exit(1)
	}
	fmt.Println("  EXACT: the file alone rebuilds the partition and every region colour, bit for bit")
}
