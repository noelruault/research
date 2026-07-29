// Context mixing over the palette-index grid, to settle a claim I got wrong.
//
// The earlier record read order-3 (18.3 KB) coming in above order-2 (17.4 KB) as "context dilution, we are at the entropy floor".
// That reading was wrong: order-3 has 4096 contexts over 147,456 samples, about 36 samples each, so it is sample-starved rather than saturated.
// A starved high-order model is not evidence of a floor; it is evidence the model needs help.
// Context mixing is the standard fix: run several orders at once and blend their predictions with weights learned online, so a high-order context contributes only where it has seen enough to be trusted.
//
// This measures where the raster baseline actually sits, which sets the real wall any shape representation has to clear.
package main

import (
	"fmt"
	"math"
	"os"
)

const nsym = 16

func stretch(p float64) float64 {
	if p < 1e-6 {
		p = 1e-6
	}
	if p > 1-1e-6 {
		p = 1 - 1e-6
	}
	return math.Log(p / (1 - p))
}

func squash(x float64) float64 { return 1 / (1 + math.Exp(-x)) }

// model is one context order.
// Each entry is an adaptive probability that the next bit of the symbol is 1, held per (context, node in the binary decomposition of the symbol).
type model struct {
	p    []float64
	n    []float64
	bits uint // context width in symbols
	mask uint32
}

func newModel(order int) *model {
	ctxs := 1
	for i := 0; i < order; i++ {
		ctxs *= nsym
	}
	m := &model{p: make([]float64, ctxs*nsym), n: make([]float64, ctxs*nsym)}
	for i := range m.p {
		m.p[i] = 0.5
	}
	return m
}

func (m *model) idx(ctx, node int) int { return ctx*nsym + node }

// predict returns this model's probability that the next bit is 1, plus a confidence that grows with how often the context has been visited, so a starved context is damped toward 0.5.
func (m *model) predict(ctx, node int) (float64, float64) {
	i := m.idx(ctx, node)
	conf := m.n[i] / (m.n[i] + 4)
	return m.p[i], conf
}

func (m *model) update(ctx, node int, bit float64) {
	i := m.idx(ctx, node)
	rate := 1 / (m.n[i] + 1.5)
	if rate < 0.02 {
		rate = 0.02
	}
	m.p[i] += (bit - m.p[i]) * rate
	m.n[i]++
}

// codeStream walks every symbol as 4 binary decisions down a fixed tree, mixes the models' predictions in the logistic domain with online-learned weights, and accumulates the exact cost an arithmetic coder would emit. The decoder reproduces every step, so nothing here is side information.
func codeStream(grid []byte, w, h int, ctxOf []func(x, y int) int, models []*model, lr float64) float64 {
	nm := len(models)
	// one weight vector per node of the binary tree, so the mixer can trust different models at different bit depths
	wt := make([][]float64, nsym)
	for i := range wt {
		wt[i] = make([]float64, nm)
		for j := range wt[i] {
			wt[i][j] = 0.3
		}
	}
	st := make([]float64, nm)
	var total float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sym := int(grid[y*w+x])
			ctxs := make([]int, nm)
			for i := range models {
				ctxs[i] = ctxOf[i](x, y)
			}
			node := 1
			for b := 3; b >= 0; b-- {
				bit := float64((sym >> uint(b)) & 1)
				var dot float64
				for i, m := range models {
					p, conf := m.predict(ctxs[i], node)
					st[i] = stretch(p) * conf
					dot += wt[node][i] * st[i]
				}
				pm := squash(dot)
				if pm < 1e-6 {
					pm = 1e-6
				}
				if pm > 1-1e-6 {
					pm = 1 - 1e-6
				}
				if bit == 1 {
					total += -math.Log2(pm)
				} else {
					total += -math.Log2(1 - pm)
				}
				err := bit - pm
				for i := range models {
					wt[node][i] += lr * err * st[i]
					models[i].update(ctxs[i], node, bit)
				}
				node = node*2 + int(bit)
			}
		}
	}
	return total
}

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	w, h := 512, 288
	if len(raw) != w*h {
		fmt.Fprintf(os.Stderr, "expected %d bytes, got %d\n", w*h, len(raw))
		os.Exit(1)
	}
	at := func(x, y int) int {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		return int(raw[y*w+x])
	}
	// causal contexts, all reconstructible by the decoder from pixels it has already emitted
	c0 := func(x, y int) int { return 0 }
	c1 := func(x, y int) int { return at(x-1, y) }
	c2 := func(x, y int) int { return at(x-1, y)*nsym + at(x, y-1) }
	c3 := func(x, y int) int { return (at(x-1, y)*nsym+at(x, y-1))*nsym + at(x-1, y-1) }
	c4 := func(x, y int) int {
		return ((at(x-1, y)*nsym+at(x, y-1))*nsym+at(x-1, y-1))*nsym + at(x+1, y-1)
	}
	// a deliberately different view: the two-pixel run to the left, which captures flat runs that the neighbourhood contexts spend capacity re-learning
	cRun := func(x, y int) int { return at(x-1, y)*nsym + at(x-2, y) }

	fmt.Printf("grid %dx%d, %d symbols, %d-colour alphabet\n\n", w, h, w*h, nsym)
	fmt.Printf("%-38s %10s %12s\n", "model", "KB", "bits/px")

	run := func(name string, orders []int, fns []func(int, int) int) float64 {
		ms := make([]*model, len(orders))
		for i, o := range orders {
			ms[i] = newModel(o)
		}
		bits := codeStream(raw, w, h, fns, ms, 0.02)
		fmt.Printf("%-38s %10.1f %12.3f\n", name, bits/8/1024, bits/float64(w*h))
		return bits
	}

	run("order-0 alone", []int{0}, []func(int, int) int{c0})
	run("order-1 (left)", []int{1}, []func(int, int) int{c1})
	run("order-2 (left,up)", []int{2}, []func(int, int) int{c2})
	run("order-3 (+upleft)", []int{3}, []func(int, int) int{c3})
	run("order-4 (+upright)", []int{4}, []func(int, int) int{c4})
	fmt.Println()
	run("mix: orders 0+1+2", []int{0, 1, 2}, []func(int, int) int{c0, c1, c2})
	run("mix: orders 0+1+2+3", []int{0, 1, 2, 3}, []func(int, int) int{c0, c1, c2, c3})
	run("mix: orders 0+1+2+3+4", []int{0, 1, 2, 3, 4}, []func(int, int) int{c0, c1, c2, c3, c4})
	run("mix: 0+1+2+3+4 + run-context", []int{0, 1, 2, 3, 4, 2}, []func(int, int) int{c0, c1, c2, c3, c4, cRun})
}
