package main

// floor.go — the entropy floor of the region colours, given the partition as known to the decoder.
//
// This measures, it does not propose. Every number here is one of three kinds and the kind is printed with it:
//
//	adaptive  — one pass, counts start uniform, the coder pays its own learning cost. ACHIEVABLE.
//	static    — the same final counts read back as an empirical conditional entropy. ORACLE: the table is not paid for.
//	oracle*   — a conditioning so wide that contexts are seen a handful of times. Not a floor, printed only to show the shape of the curve.
//
// Causality: every predictor and every context tap is computed from `dec`, the colours the decoder has already reconstructed, and from the partition, which the decoder holds in full. `dec` is filled in region-id order and region ids come from relabelComponents, which numbers in raster first-appearance order. The replay at the end of
// run() rebuilds every colour from its residual and asserts equality, so a tap that reads an undecoded colour
// cannot pass. That is falsification #12's check, applied here before any number is believed.

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// ---- adjacency -------------------------------------------------------------------------------

// flAdj is the region adjacency graph in CSR form: region r touches nb[off[r]:off[r+1]] along ln[...] crack edges.
// Built by counting sort rather than a map per region; at 6.4M regions the map form costs over a gigabyte.
type flAdj struct {
	off []int32
	nb  []int32
	ln  []int32
}

func flBuildAdj(lab []int32, n, w, h int) *flAdj {
	deg := make([]int32, n+1)
	inc := func(a, b int32) { deg[a]++; deg[b]++ }
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				inc(lab[p], lab[p+1])
			}
			if y < h-1 && lab[p] != lab[p+w] {
				inc(lab[p], lab[p+w])
			}
		}
	}
	off := make([]int32, n+1)
	var acc int32
	for r := 0; r < n; r++ {
		off[r] = acc
		acc += deg[r]
	}
	off[n] = acc
	cur := make([]int32, n)
	copy(cur, off[:n])
	raw := make([]int32, acc)
	put := func(a, b int32) { raw[cur[a]] = b; cur[a]++; raw[cur[b]] = a; cur[b]++ }
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := y*w + x
			if x < w-1 && lab[p] != lab[p+1] {
				put(lab[p], lab[p+1])
			}
			if y < h-1 && lab[p] != lab[p+w] {
				put(lab[p], lab[p+w])
			}
		}
	}
	// Compact each region's incidence list into (neighbour, shared length) pairs, in place.
	a := &flAdj{off: make([]int32, n+1), nb: make([]int32, 0, acc/2), ln: make([]int32, 0, acc/2)}
	for r := 0; r < n; r++ {
		s := raw[off[r]:off[r+1]]
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		a.off[r] = int32(len(a.nb))
		for i := 0; i < len(s); {
			j := i
			for j < len(s) && s[j] == s[i] {
				j++
			}
			a.nb = append(a.nb, s[i])
			a.ln = append(a.ln, int32(j-i))
			i = j
		}
	}
	a.off[n] = int32(len(a.nb))
	return a
}

// ---- models ----------------------------------------------------------------------------------

// flModel is an adaptive Laplace-smoothed symbol coder over nctx contexts, identical in form to colorBytes2's per-channel counter so the numbers stay comparable with every earlier round of this study.
type flModel struct {
	nsym   int
	counts []uint32
	totals []uint32
	bits   float64
}

func flNewModel(nctx, nsym int) *flModel {
	return &flModel{nsym: nsym, counts: make([]uint32, nctx*nsym), totals: make([]uint32, nctx)}
}

func (m *flModel) code(ctx, sym int) {
	b := ctx * m.nsym
	m.bits += -math.Log2(float64(m.counts[b+sym]+1) / float64(m.totals[ctx]+uint32(m.nsym)))
	m.counts[b+sym]++
	m.totals[ctx]++
}

// codeExcl codes sym knowing the decoder can rule out every symbol in excl.
// The exclusion is decoder-side knowledge (adjacent regions of the exact partition cannot share a colour), not a hint from the data.
func (m *flModel) codeExcl(ctx, sym int, excl []int) {
	b := ctx * m.nsym
	den := float64(m.totals[ctx] + uint32(m.nsym))
	for _, e := range excl {
		den -= float64(m.counts[b+e] + 1)
	}
	if den <= 1e-9 {
		den = 1e-9
	}
	m.bits += -math.Log2(float64(m.counts[b+sym]+1) / den)
	m.counts[b+sym]++
	m.totals[ctx]++
}

func (m *flModel) adaptiveBytes() float64 { return m.bits / 8 }

// staticBytes reads the final counts back as an empirical conditional entropy: the ideal two-pass cost with the table given away free. It is an oracle, and the gap to adaptiveBytes is what learning the flModel costs.
func (m *flModel) staticBytes() float64 {
	bits := 0.0
	for c := range m.totals {
		t := float64(m.totals[c])
		if t == 0 {
			continue
		}
		b := c * m.nsym
		for s := 0; s < m.nsym; s++ {
			if k := float64(m.counts[b+s]); k > 0 {
				bits += k * math.Log2(t/k)
			}
		}
	}
	return bits / 8
}

// usedCtx reports how many contexts were ever visited and the mean samples each got, which is how starvation shows itself (falsification #2).
func (m *flModel) usedCtx() (int, float64) {
	used, tot := 0, 0.0
	for _, t := range m.totals {
		if t > 0 {
			used++
			tot += float64(t)
		}
	}
	if used == 0 {
		return 0, 0
	}
	return used, tot / float64(used)
}

// flTok is the same adaptive counter over a magnitude-token alphabet: a residual is sent as a signed
// magnitude class (21 tokens) plus the raw low bits of its magnitude. It carries exactly the same information as
// the 1024-bin flModel — the mapping is a bijection — but a context costs 21 counters to learn instead of 1024,
// which is the whole difference between a context that pays for itself and one that starves.
// The extra bits are charged at their raw width, i.e. not modelled at all, so every flTok number is pessimistic.
type flTok struct {
	m     *flModel
	extra float64
}

func flNewTok(nctx int) *flTok { return &flTok{m: flNewModel(nctx, 21)} }

func flTokenOf(v int) (tok, nextra, low int) {
	if v == 0 {
		return 0, 0, 0
	}
	mg, sign := v, 0
	if mg < 0 {
		mg, sign = -mg, 1
	}
	b := 0
	for t := mg; t > 0; t >>= 1 {
		b++
	}
	return 2*b - 1 + sign, b - 1, mg &^ (1 << (b - 1))
}

func (t *flTok) code(ctx, v int) {
	tok, nx, _ := flTokenOf(v)
	t.m.code(ctx, tok)
	t.extra += float64(nx)
}

func (t *flTok) adaptiveBytes() float64  { return (t.m.bits + t.extra) / 8 }
func (t *flTok) staticBytes() float64    { return t.m.staticBytes() + t.extra/8 }
func (t *flTok) usedCtx() (int, float64) { return t.m.usedCtx() }

// ---- quantizers for the context taps -----------------------------------------------------------

func flBucket(v int, edges []int) int {
	for i, e := range edges {
		if v <= e {
			return i
		}
	}
	return len(edges)
}

var (
	flSpreadEdges = []int{0, 1, 2, 4, 8, 16, 32} // 8 buckets
	flAreaEdges   = []int{1, 2, 4, 8, 16, 64}    // 7 buckets
	flMagEdges    = []int{0, 1, 2, 4, 8, 16, 32} // 8 buckets
	flSignEdges   = []int{-16, -8, -4, -2, -1, 0, 1, 2, 4, 8, 16}
)

func flQSigned(v int) int { // 12 buckets, sign preserved: cross-channel correlation is signed
	for i, e := range flSignEdges {
		if v <= e {
			return i
		}
	}
	return len(flSignEdges)
}

// ---- the ladder ------------------------------------------------------------------------------

func floorRun(im *Img, lab []int32, cols [][3]float64, tag string, dumpDir string) {
	t0 := time.Now()
	n := len(cols)
	npix := im.W * im.H
	fmt.Printf("\n## %s — %d regions, %d px, %.3f px/region\n", tag, n, npix, float64(npix)/float64(n))

	// The anchor: the published coder, unmodified, on these exact labels.
	ref := colorBytes2(lab, cols, im.W, im.H)
	fmt.Printf("colorBytes2 (published coder, unmodified)   %14.2f B\n", ref)

	a := flBuildAdj(lab, n, im.W, im.H)
	fmt.Fprintf(os.Stderr, "[%s] adjacency built in %s\n", tag, time.Since(t0).Round(time.Millisecond))

	// seed[r] is region r's first pixel in raster order. Region ids come from first appearance, so every 4- and
	// 8-neighbour of that pixel that lies west, north-west, north or north-east belongs to a region with a LOWER id
	// and is therefore already decoded. That makes a JPEG-LS style raster predictor available to the region coder
	// at no cost in side information, which at 1.3 px/region is most of the picture.
	seed := make([]int32, n)
	for i := range seed {
		seed[i] = -1
	}
	for q, l := range lab {
		if seed[l] < 0 {
			seed[l] = int32(q)
		}
	}

	// dec holds only what the decoder has reconstructed. Everything below reads dec, never cols, except to form the residual being coded.
	dec := make([][3]float64, n)
	// residual of every region's G channel against the weighted mean, kept as a causal context tap.
	prevG := make([]int16, n)

	// Models. Channel order is G, R, B: G is coded first and then conditions R and B, which is information the decoder has (same region, earlier channel) and the published coder throws away.
	const (
		chG = 0
		chR = 1
		chB = 2
	)
	// One flModel per channel, 512 bins, Laplace +1 — colorBytes2's exact counter, so mWmean must reproduce it to the last bit.
	mBest := flNewModel(3, 512)  // residual against the single longest decoded neighbour
	mWmean := flNewModel(3, 512) // residual against the boundary-weighted mean == colorBytes2's predictor
	mRCT := flNewModel(3, 1024)  // + reversible residual transform, one context per channel
	mC1 := flNewModel(3*8*4, 1024)
	mC2 := flNewModel(3*8*4*12, 1024)
	mC3 := flNewModel(3*8*4*12*7, 1024)
	mC4 := flNewModel(3*8*4*12*7*8, 1024)
	mC4x := flNewModel(3*8*4*12*7*8, 1024) // identical contexts, plus the decoder-side colour exclusion
	// The raster arm: same information, a predictor that uses the decoded pixels around the region's seed instead of
	// the region-mean of its neighbours.
	mMED := flNewModel(3, 512)       // MED / LOCO-I residual, per channel, directly comparable with the anchor
	mMEDrct := flNewModel(3, 1024)   // + the reversible residual transform
	mAvg := flNewModel(3, 1024)      // predictor = mean of the weighted-region-mean and MED, + RCT
	mP2 := flNewModel(3*81*12, 1024) // + JPEG-LS gradient context, + cross-channel
	mP3 := flNewModel(3*81*12*8, 1024)
	mP3x := flNewModel(3*81*12*8, 1024) // same, plus the decoder-side colour exclusion

	// The same models over the token alphabet, which is where the context ladder stops being starved.
	tRCT := flNewTok(3)
	tC4 := flNewTok(3 * 8 * 4 * 12 * 7 * 8)
	tP2 := flNewTok(3 * 81 * 12)
	tP3 := flNewTok(3 * 81 * 12 * 8)
	tP4 := flNewTok(3 * 81 * 12 * 8 * 8)
	// A junk control matched context-for-context to tP3, over the same alphabet. Its *static* number is pure
	// overfitting: contexts that carry no information at all still split the sample and lower an empirical
	// conditional entropy. Subtracting it is what turns a static oracle into something worth quoting.
	tJunk := flNewTok(3 * 81 * 12 * 8)

	// A junk control with the same number of contexts as mC3 but no information in them: if the win were capacity rather than information this would win too. Falsification #2's distinguishing experiment.
	mJunk := flNewModel(3*8*4*12*7, 1024)

	// Streams for the general-purpose compressor bound.
	// The byte streams are a lossless re-encoding of the residuals: every value lies in [-511,511] and the decoder
	// knows the predictor, so the low byte determines the colour uniquely (c = (pred + d) mod 256, and c is a byte).
	var wmStream, rctStream, medStream []byte
	if dumpDir != "" {
		wmStream = make([]byte, 0, 3*n)
		rctStream = make([]byte, 0, 3*n)
		medStream = make([]byte, 0, 3*n)
	}

	// Oracle rows collected by sorting rather than by counting into a table.
	rawKeys := make([]uint64, 0, n) // joint colour, for H(C) order-0 joint
	nbKeys := make([]uint64, 0, n)  // (neighbour colour, colour), for the oracle H(C | best neighbour)
	resKeys := make([]uint64, 0, n) // joint weighted-mean residual triple

	mismatch := 0
	flExclViolations = 0
	junkState := uint32(12345)

	for r := 0; r < n; r++ {
		lo, hi := a.off[r], a.off[r+1]
		// ---- predictors, from decoded neighbours only ------------------------------------------
		var acc [3]float64
		wsum := 0.0
		ndec := 0
		var best int32 = -1
		var bestLen int32 = 0
		var second int32 = -1
		var secondLen int32 = 0
		for i := lo; i < hi; i++ {
			nb, ln := a.nb[i], a.ln[i]
			if int(nb) >= r {
				continue
			}
			ndec++
			for c := 0; c < 3; c++ {
				acc[c] += float64(ln) * dec[nb][c]
			}
			wsum += float64(ln)
			// Total order: longest wall, lowest id on a tie (falsification #7).
			if ln > bestLen || (ln == bestLen && best >= 0 && nb < best) {
				second, secondLen = best, bestLen
				best, bestLen = nb, ln
			} else if ln > secondLen || (ln == secondLen && second >= 0 && nb < second) {
				second, secondLen = nb, ln
			}
		}
		var wmean, pbest [3]float64
		if wsum > 0 {
			for c := 0; c < 3; c++ {
				wmean[c] = math.Round(acc[c] / wsum)
			}
		} else {
			wmean = [3]float64{128, 128, 128}
		}
		if best >= 0 {
			pbest = dec[best]
		} else {
			pbest = [3]float64{128, 128, 128}
		}

		// ---- context taps, all decoder-known --------------------------------------------------- spread: boundary-weighted mean absolute deviation of the decoded neighbours from the prediction.
		spread := 0
		if ndec > 1 {
			dev := 0.0
			for i := lo; i < hi; i++ {
				nb, ln := a.nb[i], a.ln[i]
				if int(nb) >= r {
					continue
				}
				for c := 0; c < 3; c++ {
					dev += float64(ln) * math.Abs(dec[nb][c]-wmean[c])
				}
			}
			spread = int(dev / wsum / 3)
		}
		qsp := flBucket(spread, flSpreadEdges)
		qnb := ndec
		if qnb > 3 {
			qnb = 3
		}
		// area: the decoder counts the region's pixels straight off the partition.
		qar := flBucket(int(flRegArea[r]), flAreaEdges)
		// previous residual: the G residual of the longest decoded neighbour, else of region r-1.
		pr := 0
		if best >= 0 {
			pr = int(prevG[best])
		} else if r > 0 {
			pr = int(prevG[r-1])
		}
		if pr < 0 {
			pr = -pr
		}
		qpr := flBucket(pr, flMagEdges)
		// local gradient: how far apart the two heaviest decoded neighbours are.
		grad := 0
		if second >= 0 {
			for c := 0; c < 3; c++ {
				grad += int(math.Abs(dec[best][c] - dec[second][c]))
			}
			grad /= 3
		}
		qgr := flBucket(grad, flSpreadEdges)

		// ---- residuals -------------------------------------------------------------------------
		var dW, dBst [3]int
		for c := 0; c < 3; c++ {
			dW[c] = int(cols[r][c] - wmean[c])
			dBst[c] = int(cols[r][c] - pbest[c])
		}
		// Reversible residual transform: send the G residual, then the other two as differences from it.
		// Reversible and decoder-side; costs nothing and needs no side information.
		tG := dW[1]
		tR := dW[0] - dW[1]
		tB := dW[2] - dW[1]

		for c := 0; c < 3; c++ {
			mWmean.code(c, dW[c]+256)
			mBest.code(c, dBst[c]+256)
		}

		mRCT.code(chG, tG+512)
		mRCT.code(chR, tR+512)
		mRCT.code(chB, tB+512)

		c1 := func(ch int) int { return (ch*8+qsp)*4 + qnb }
		mC1.code(c1(chG), tG+512)
		mC1.code(c1(chR), tR+512)
		mC1.code(c1(chB), tB+512)

		qg := flQSigned(tG)
		c2 := func(ch, x int) int { return c1(ch)*12 + x }
		mC2.code(c2(chG, 0), tG+512)
		mC2.code(c2(chR, qg), tR+512)
		mC2.code(c2(chB, qg), tB+512)

		c3 := func(ch, x int) int { return c2(ch, x)*7 + qar }
		mC3.code(c3(chG, 0), tG+512)
		mC3.code(c3(chR, qg), tR+512)
		mC3.code(c3(chB, qg), tB+512)

		c4 := func(ch, x, y int) int { return c3(ch, x)*8 + y }
		mC4.code(c4(chG, 0, qpr), tG+512)
		mC4.code(c4(chR, qg, qgr), tR+512)
		mC4.code(c4(chB, qg, qgr), tB+512)

		// Same contexts, plus the exclusion: on the exact partition adjacent regions cannot share a colour, so once two channels are known the third cannot take the value that would make this region equal to a neighbour.
		mC4x.code(c4(chG, 0, qpr), tG+512)
		mC4x.code(c4(chR, qg, qgr), tR+512)
		var excl []int
		for i := lo; i < hi; i++ {
			nb := a.nb[i]
			if int(nb) >= r {
				continue
			}
			if dec[nb][1] == cols[r][1] && dec[nb][0] == cols[r][0] {
				// B is the only free channel left; the value that would collide is ruled out.
				v := int(dec[nb][2]-wmean[2]) - tG + 512
				if v >= 0 && v < 1024 {
					excl = append(excl, v)
				}
			}
		}
		// The exclusion has to be true, not merely plausible: on the exact partition adjacent regions differ, so the coded symbol may never be one the decoder was told to rule out. On a merged partition it can be, and the counter below is what says so.
		for _, e := range excl {
			if e == tB+512 {
				flExclViolations++
			}
		}
		mC4x.codeExcl(c4(chB, qg, qgr), tB+512, excl)

		// Junk control: same context count, contents a deterministic hash of the region id.
		junkState = junkState*1664525 + 1013904223
		jc := int(junkState>>8) % (8 * 4 * 12 * 7)
		mJunk.code(chG*(8*4*12*7)+jc, tG+512)
		mJunk.code(chR*(8*4*12*7)+jc, tR+512)
		mJunk.code(chB*(8*4*12*7)+jc, tB+512)

		// ---- the raster arm ---------------------------------------------------------------------
		sp := int(seed[r])
		sx, sy := sp%im.W, sp/im.W
		var pw, pn, pnw, pne [3]float64
		hasW, hasN := sx > 0, sy > 0
		if hasW {
			pw = dec[lab[sp-1]]
		}
		if hasN {
			pn = dec[lab[sp-im.W]]
		}
		if hasW && hasN {
			pnw = dec[lab[sp-im.W-1]]
		}
		if hasN && sx < im.W-1 {
			pne = dec[lab[sp-im.W+1]]
		}
		var pmed [3]float64
		switch {
		case hasW && hasN:
			for c := 0; c < 3; c++ {
				lo, hi := math.Min(pw[c], pn[c]), math.Max(pw[c], pn[c])
				switch {
				case pnw[c] >= hi:
					pmed[c] = lo
				case pnw[c] <= lo:
					pmed[c] = hi
				default:
					pmed[c] = pw[c] + pn[c] - pnw[c]
				}
			}
		case hasW:
			pmed = pw
		case hasN:
			pmed = pn
		default:
			pmed = wmean
		}
		var dM, dA [3]int
		for c := 0; c < 3; c++ {
			dM[c] = int(cols[r][c] - pmed[c])
			dA[c] = int(cols[r][c] - math.Round((pmed[c]+wmean[c])/2))
		}
		mG, mR, mB := dM[1], dM[0]-dM[1], dM[2]-dM[1]
		aG, aR, aB := dA[1], dA[0]-dA[1], dA[2]-dA[1]
		for c := 0; c < 3; c++ {
			mMED.code(c, dM[c]+256)
		}
		mMEDrct.code(chG, mG+512)
		mMEDrct.code(chR, mR+512)
		mMEDrct.code(chB, mB+512)
		mAvg.code(chG, aG+512)
		mAvg.code(chR, aR+512)
		mAvg.code(chB, aB+512)

		// JPEG-LS local gradients, from the same already-decoded pixels.
		g1 := flBucket(int(math.Abs(pnw[1]-pw[1])), flSpreadEdges)
		g2 := flBucket(int(math.Abs(pn[1]-pnw[1])), flSpreadEdges)
		g3 := 0
		if hasN && sx < im.W-1 {
			g3 = flBucket(int(math.Abs(pne[1]-pn[1])), flSpreadEdges)
		}
		gctx := g1*9 + g2 // 8 buckets each, so 0..79
		qm := flQSigned(mG)
		p2 := func(ch, x int) int { return ((ch*81)+gctx)*12 + x }
		mP2.code(p2(chG, 0), mG+512)
		mP2.code(p2(chR, qm), mR+512)
		mP2.code(p2(chB, qm), mB+512)
		p3 := func(ch, x, y int) int { return p2(ch, x)*8 + y }
		mP3.code(p3(chG, 0, qpr), mG+512)
		mP3.code(p3(chR, qm, g3), mR+512)
		mP3.code(p3(chB, qm, g3), mB+512)
		mP3x.code(p3(chG, 0, qpr), mG+512)
		mP3x.code(p3(chR, qm, g3), mR+512)
		var exclM []int
		for i := lo; i < hi; i++ {
			nb := a.nb[i]
			if int(nb) >= r {
				continue
			}
			if dec[nb][1] == cols[r][1] && dec[nb][0] == cols[r][0] {
				v := int(dec[nb][2]-pmed[2]) - mG + 512
				if v >= 0 && v < 1024 {
					exclM = append(exclM, v)
				}
			}
		}
		mP3x.codeExcl(p3(chB, qm, g3), mB+512, exclM)

		tRCT.code(chG, mG)
		tRCT.code(chR, mR)
		tRCT.code(chB, mB)
		tC4.code(c4(chG, 0, qpr), tG)
		tC4.code(c4(chR, qg, qgr), tR)
		tC4.code(c4(chB, qg, qgr), tB)
		tP2.code(p2(chG, 0), mG)
		tP2.code(p2(chR, qm), mR)
		tP2.code(p2(chB, qm), mB)
		tP3.code(p3(chG, 0, qpr), mG)
		tP3.code(p3(chR, qm, g3), mR)
		tP3.code(p3(chB, qm, g3), mB)
		jc3 := int(junkState>>7) % (81 * 12 * 8)
		tJunk.code(chG*(81*12*8)+jc3, mG)
		tJunk.code(chR*(81*12*8)+jc3, mR)
		tJunk.code(chB*(81*12*8)+jc3, mB)
		p4 := func(ch, x, y, z int) int { return p3(ch, x, y)*8 + z }
		tP4.code(p4(chG, 0, qpr, qsp), mG)
		tP4.code(p4(chR, qm, g3, qsp), mR)
		tP4.code(p4(chB, qm, g3, qsp), mB)

		// ---- oracle rows -----------------------------------------------------------------------
		ck := uint64(cols[r][0])<<16 | uint64(cols[r][1])<<8 | uint64(cols[r][2])
		rawKeys = append(rawKeys, ck)
		pk := uint64(pbest[0])<<16 | uint64(pbest[1])<<8 | uint64(pbest[2])
		nbKeys = append(nbKeys, pk<<24|ck)
		resKeys = append(resKeys, uint64(dW[0]+255)<<20|uint64(dW[1]+255)<<10|uint64(dW[2]+255))

		if dumpDir != "" {
			wmStream = append(wmStream, byte(dW[0]), byte(dW[1]), byte(dW[2]))
			rctStream = append(rctStream, byte(tG), byte(tR), byte(tB))
			medStream = append(medStream, byte(mG), byte(mR), byte(mB))
		}

		// ---- decode replay: reconstruct from the residual, nothing else ------------------------
		for c := 0; c < 3; c++ {
			dec[r][c] = wmean[c] + float64(dW[c])
			if dec[r][c] != cols[r][c] {
				mismatch++
			}
		}
		prevG[r] = int16(dW[1])
	}

	fmt.Fprintf(os.Stderr, "[%s] models done in %s\n", tag, time.Since(t0).Round(time.Second))

	// ---- report ----------------------------------------------------------------------------------
	h0chan := flMarginalBytes(cols)
	h0joint, njoint := flSortedEntropy(rawKeys)
	hNb, nNb := flCondEntropy(nbKeys, 24)
	hResJoint, nResJoint := flSortedEntropy(resKeys)

	fmt.Printf("%-46s %14s %14s %10s %9s %s\n", "model", "adaptive B", "static B", "ctx used", "smp/ctx", "kind")
	row := func(name string, m *flModel, kind string) {
		u, per := m.usedCtx()
		fmt.Printf("%-46s %14.0f %14.0f %10d %9.0f %s\n", name, m.adaptiveBytes(), m.staticBytes(), u, per, kind)
	}
	fmt.Printf("%-46s %14.0f %14s %10d %9s %s\n", "H(C) order-0, per channel (raw values)", h0chan, "-", 3, "-", "static, needs no table")
	fmt.Printf("%-46s %14s %14.0f %10d %9.1f %s\n", "H(C) order-0, joint RGB symbol", "-", h0joint, njoint, float64(n)/float64(njoint), "static + a "+fmt.Sprintf("%.0f", float64(njoint)*24/8)+" B codebook")
	fmt.Printf("%-46s %14s %14.0f %10d %9.1f %s\n", "H(C | best decoded neighbour), exact", "-", hNb, nNb, float64(n)/float64(nNb), "ORACLE* — see note")
	row("H(C | best decoded neighbour), residual", mBest, "achievable")
	row("H(C | weighted mean) == the current coder", mWmean, "achievable — anchor")
	fmt.Printf("%-46s %14s %14.0f %10d %9.1f %s\n", "H(residual triple), joint, weighted mean", "-", hResJoint, nResJoint, float64(n)/float64(nResJoint), "ORACLE* — codebook unpaid")
	row("+ residual RCT (G, R-G, B-G)", mRCT, "achievable")
	row("+ ctx spread x n_decoded_nb", mC1, "achievable")
	row("+ ctx cross-channel (G residual)", mC2, "achievable")
	row("+ ctx region area", mC3, "achievable")
	row("+ ctx prev residual / local gradient", mC4, "achievable")
	row("+ decoder-side colour exclusion", mC4x, "achievable")
	row("MED/LOCO-I on decoded pixels, per channel", mMED, "achievable")
	row("MED + residual RCT", mMEDrct, "achievable")
	row("mean(wmean, MED) + residual RCT", mAvg, "achievable")
	row("MED + RCT + JPEG-LS gradient + cross-ch", mP2, "achievable")
	row("+ prev residual / NE gradient", mP3, "achievable")
	row("+ decoder-side colour exclusion", mP3x, "achievable")
	trow := func(name string, t *flTok, kind string) {
		u, per := t.usedCtx()
		fmt.Printf("%-46s %14.0f %14.0f %10d %9.0f %s\n", name, t.adaptiveBytes(), t.staticBytes(), u, per, kind)
	}
	trow("token alphabet: MED + RCT", tRCT, "achievable")
	trow("token: wmean + RCT + full ctx", tC4, "achievable")
	trow("token: MED + RCT + gradctx + cross-ch", tP2, "achievable")
	trow("token: + prev residual / NE gradient", tP3, "achievable")
	trow("token: + neighbour spread", tP4, "achievable")
	row("JUNK CONTROL: mC3's capacity, no info", mJunk, "control")
	trow("JUNK CONTROL: tP3's capacity, no info", tJunk, "control")
	fmt.Printf("%-46s %14s %14.0f %10s %9s %s\n", "  overfit in a static number at tP3's size", "-",
		tRCT.staticBytes()-tJunk.staticBytes(), "-", "-", "subtract this from any static row of that width")

	// Best of the achievable ladder. Choosing among a handful of fixed models is an RD choice the encoder can make
	// and signal in one byte, so it is priced; the choice is over models, never over data the decoder cannot see.
	cands := []struct {
		name string
		m    *flModel
	}{{"best-neighbour residual", mBest}, {"weighted mean (anchor)", mWmean}, {"residual RCT", mRCT},
		{"+ctx spread x nnb", mC1}, {"+ctx cross-channel", mC2}, {"+ctx area", mC3}, {"+ctx prev/grad", mC4},
		{"+exclusion", mC4x}, {"MED", mMED}, {"MED+RCT", mMEDrct}, {"mean(wmean,MED)+RCT", mAvg},
		{"MED+RCT+gradctx", mP2}, {"MED+RCT+gradctx+prev", mP3}, {"MED+RCT+gradctx+prev+excl", mP3x}}
	toks := []struct {
		name string
		t    *flTok
	}{{"token MED+RCT", tRCT}, {"token wmean+RCT+ctx", tC4}, {"token MED+RCT+gradctx", tP2},
		{"token MED+RCT+gradctx+prev", tP3}, {"token +spread", tP4}}
	bestName, best := cands[0].name, cands[0].m.adaptiveBytes()
	sBestName, sBest := cands[0].name, cands[0].m.staticBytes()
	for _, c := range cands[1:] {
		if v := c.m.adaptiveBytes(); v < best {
			bestName, best = c.name, v
		}
		if v := c.m.staticBytes(); v < sBest {
			sBestName, sBest = c.name, v
		}
	}
	for _, c := range toks {
		if v := c.t.adaptiveBytes(); v < best {
			bestName, best = c.name, v
		}
		if v := c.t.staticBytes(); v < sBest {
			sBestName, sBest = c.name, v
		}
	}
	fmt.Printf("\nanchor colorBytes2 %.0f B\n", ref)
	fmt.Printf("best ACHIEVABLE (one pass, model paid): %.0f B via %s  -> %+.2f%% of the colour bill\n", best, bestName, 100*(best-ref)/ref)
	fmt.Printf("best STATIC oracle (table free):        %.0f B via %s  -> %+.2f%% of the colour bill\n", sBest, sBestName, 100*(sBest-ref)/ref)
	fmt.Printf("gap between them (model-learning cost): %.0f B = %.1f%% of the colour bill\n", best-sBest, 100*(best-sBest)/ref)
	_ = sBestName
	fmt.Printf("replay mismatches %d (0 == every context and predictor is causal)\n", mismatch)
	fmt.Printf("exclusion violations %d (0 == the exclusion row is legal on this partition)\n", flExclViolations)
	fmt.Printf("# %s took %s\n", tag, time.Since(t0).Round(time.Second))

	if dumpDir != "" {
		flWriteStream(dumpDir+"/"+tag+"_wmean_interleaved.bin", wmStream)
		flWriteStream(dumpDir+"/"+tag+"_rct_interleaved.bin", rctStream)
		flWriteStream(dumpDir+"/"+tag+"_wmean_planar.bin", flPlanar(wmStream))
		flWriteStream(dumpDir+"/"+tag+"_rct_planar.bin", flPlanar(rctStream))
		flWriteStream(dumpDir+"/"+tag+"_medrct_interleaved.bin", medStream)
		flWriteStream(dumpDir+"/"+tag+"_medrct_planar.bin", flPlanar(medStream))
	}
}

// flRegArea is filled by floorCmd before floorRun; the decoder counts these off the partition itself, so it is not side information.
var flRegArea []int32

// flExclViolations counts times the decoder-side colour exclusion ruled out the symbol that was actually coded.
// It must be 0 on the exact partition. On a merged partition it need not be, and a non-zero count invalidates that row.
var flExclViolations int

func flPlanar(s []byte) []byte {
	out := make([]byte, 0, len(s))
	for c := 0; c < 3; c++ {
		for i := c; i < len(s); i += 3 {
			out = append(out, s[i])
		}
	}
	return out
}

func flWriteStream(path string, b []byte) {
	f, err := os.Create(path)
	must(err)
	w := bufio.NewWriter(f)
	_, err = w.Write(b)
	must(err)
	must(w.Flush())
	must(f.Close())
	fmt.Printf("wrote %s (%d B)\n", path, len(b))
}

// flMarginalBytes is the order-0 entropy of the raw channel values, no prediction at all.
func flMarginalBytes(cols [][3]float64) float64 {
	var cnt [3][256]float64
	for _, c := range cols {
		for k := 0; k < 3; k++ {
			cnt[k][int(c[k])]++
		}
	}
	nn := float64(len(cols))
	bits := 0.0
	for k := 0; k < 3; k++ {
		for _, v := range cnt[k] {
			if v > 0 {
				bits += v * math.Log2(nn/v)
			}
		}
	}
	return bits / 8
}

// flSortedEntropy is the empirical entropy of a symbol stream with an alphabet too large to tabulate.
func flSortedEntropy(keys []uint64) (float64, int) {
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	nn := float64(len(keys))
	bits, distinct := 0.0, 0
	for i := 0; i < len(keys); {
		j := i
		for j < len(keys) && keys[j] == keys[i] {
			j++
		}
		k := float64(j - i)
		bits += k * math.Log2(nn/k)
		distinct++
		i = j
	}
	return bits / 8, distinct
}

// flCondEntropy is H(low | high) for keys packed as high<<shift|low. Wide conditionings make this an oracle, not a floor.
func flCondEntropy(keys []uint64, shift uint) (float64, int) {
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	mask := uint64(1)<<shift - 1
	bits := 0.0
	nctx := 0
	for i := 0; i < len(keys); {
		j := i
		for j < len(keys) && keys[j]>>shift == keys[i]>>shift {
			j++
		}
		t := float64(j - i)
		nctx++
		for k := i; k < j; {
			l := k
			for l < j && keys[l]&mask == keys[k]&mask {
				l++
			}
			c := float64(l - k)
			bits += c * math.Log2(t/c)
			k = l
		}
		i = j
	}
	return bits / 8, nctx
}

// floorCmd: floor <image.png> [labels.bin ...]; with no labels file it runs the exact lossless partition.
func floorCmd(args []string) {
	im := load(args[0])
	dump := os.Getenv("FLOORDUMP")
	fmt.Printf("# %s %dx%d\n", args[0], im.W, im.H)
	if len(args) == 1 {
		lab, cols, ncol := exactPartition(im)
		fmt.Printf("# exact lossless partition, %d distinct pixel colours\n", ncol)
		flRegArea = flAreasOf(lab, len(cols))
		floorRun(im, lab, cols, "lossless", dump)
		return
	}
	for _, p := range args[1:] {
		lab, w, h := loadLabels(p)
		if w != im.W || h != im.H {
			fmt.Fprintf(os.Stderr, "partition %s is %dx%d, image is %dx%d\n", p, w, h, im.W, im.H)
			os.Exit(1)
		}
		n := 0
		for _, l := range lab {
			if int(l)+1 > n {
				n = int(l) + 1
			}
		}
		// Region colours exactly as priceSeg forms them: the rounded mean of the region's pixels.
		sum := make([][3]float64, n)
		cnt := make([]float64, n)
		for q, l := range lab {
			cnt[l]++
			for c := 0; c < 3; c++ {
				sum[l][c] += im.P[q*3+c]
			}
		}
		cols := make([][3]float64, n)
		for k := 0; k < n; k++ {
			if cnt[k] == 0 {
				continue
			}
			for c := 0; c < 3; c++ {
				cols[k][c] = math.Round(sum[k][c] / cnt[k])
			}
		}
		flRegArea = make([]int32, n)
		for k := range cnt {
			flRegArea[k] = int32(cnt[k])
		}
		floorRun(im, lab, cols, fmt.Sprintf("lossy_%d", n), dump)
	}
}

func flAreasOf(lab []int32, n int) []int32 {
	out := make([]int32, n)
	for _, l := range lab {
		out[l]++
	}
	return out
}

// ---- the check that makes the general-compressor bound real -------------------------------------

// floorDec is a decoder. It is given only the partition (which the wall bill pays for) and one of the residual
// byte streams, and it rebuilds the image. If it matches the source exactly, then whatever a general compressor
// squeezes that stream down to is a real, decodable colour bill and not a bookkeeping trick.
// The residuals are stored mod 256; the colour is recovered as (predictor + residual) mod 256, which is unique
// because the colour is itself a byte.
func floorDec(labPath, streamPath, mode, refPath string) {
	lab, w, h := loadLabels(labPath)
	raw, err := os.ReadFile(streamPath)
	must(err)
	n := 0
	for _, l := range lab {
		if int(l)+1 > n {
			n = int(l) + 1
		}
	}
	if len(raw) != 3*n {
		fmt.Fprintf(os.Stderr, "stream is %d B, expected %d\n", len(raw), 3*n)
		os.Exit(1)
	}
	planarIn := len(os.Args) > 6 && os.Args[6] == "planar"
	at := func(r, c int) int {
		if planarIn {
			return c*n + r
		}
		return r*3 + c
	}
	flRegArea = flAreasOf(lab, n)
	a := flBuildAdj(lab, n, w, h)
	seed := make([]int32, n)
	for i := range seed {
		seed[i] = -1
	}
	for q, l := range lab {
		if seed[l] < 0 {
			seed[l] = int32(q)
		}
	}
	dec := make([][3]float64, n)
	for r := 0; r < n; r++ {
		var acc [3]float64
		wsum := 0.0
		for i := a.off[r]; i < a.off[r+1]; i++ {
			nb, ln := a.nb[i], a.ln[i]
			if int(nb) >= r {
				continue
			}
			for c := 0; c < 3; c++ {
				acc[c] += float64(ln) * dec[nb][c]
			}
			wsum += float64(ln)
		}
		pred := [3]float64{128, 128, 128}
		if wsum > 0 {
			for c := 0; c < 3; c++ {
				pred[c] = math.Round(acc[c] / wsum)
			}
		}
		if mode == "medrct" {
			sp := int(seed[r])
			sx, sy := sp%w, sp/w
			hasW, hasN := sx > 0, sy > 0
			var pw, pn, pnw [3]float64
			if hasW {
				pw = dec[lab[sp-1]]
			}
			if hasN {
				pn = dec[lab[sp-w]]
			}
			if hasW && hasN {
				pnw = dec[lab[sp-w-1]]
			}
			switch {
			case hasW && hasN:
				for c := 0; c < 3; c++ {
					lo, hi := math.Min(pw[c], pn[c]), math.Max(pw[c], pn[c])
					switch {
					case pnw[c] >= hi:
						pred[c] = lo
					case pnw[c] <= lo:
						pred[c] = hi
					default:
						pred[c] = pw[c] + pn[c] - pnw[c]
					}
				}
			case hasW:
				pred = pw
			case hasN:
				pred = pn
			}
		}
		switch mode {
		case "wmean":
			for c := 0; c < 3; c++ {
				dec[r][c] = float64((int(pred[c]) + int(raw[at(r, c)])) & 255)
			}
		case "rct", "medrct":
			// stream order is G, R-G, B-G
			g := int(raw[at(r, 0)])
			dg := (int(pred[1]) + g) & 255
			dec[r][1] = float64(dg)
			dec[r][0] = float64((int(pred[0]) + g + int(raw[at(r, 1)])) & 255)
			dec[r][2] = float64((int(pred[2]) + g + int(raw[at(r, 2)])) & 255)
		default:
			fmt.Fprintln(os.Stderr, "unknown mode")
			os.Exit(2)
		}
	}
	ref := load(refPath)
	bad, maxd := 0, 0.0
	for p, l := range lab {
		for c := 0; c < 3; c++ {
			d := math.Abs(dec[l][c] - ref.P[p*3+c])
			if d > 0 {
				bad++
				if d > maxd {
					maxd = d
				}
			}
		}
	}
	fmt.Printf("decode %s (%s, %s): %d regions, %d wrong samples of %d, max |delta| %.0f\n",
		streamPath, mode, map[bool]string{true: "planar", false: "interleaved"}[planarIn], n, bad, 3*w*h, maxd)
	if bad != 0 {
		os.Exit(1)
	}
	fmt.Println("EXACT: the partition plus this byte stream reconstruct the source bit for bit")
}

// floorDump writes the exact lossless partition so the decoder above can be run against it.
func floorDump(imgPath, out string) {
	im := load(imgPath)
	lab, cols, _ := exactPartition(im)
	dumpLabels(out, len(cols), im.W, im.H, lab)
	fmt.Printf("wrote %s/lab_%08d.bin (%d regions)\n", out, len(cols), len(cols))
}
