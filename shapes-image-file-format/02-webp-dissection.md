# 02 — Dissect WebP, port its levers, try to beat it

**Question.** Report 01 says WebP wins. Why, exactly — and can its mechanisms be ported into the shape domain?

## What WebP-lossless (VP8L) actually does

On a 16-colour image it stacks four transforms:

1. **Colour indexing + pixel bundling** — a palette, and for ≤16 colours several pixels packed per byte.
2. **A spatial predictor**, 14 modes selected per 2^k tile, storing only residuals.
3. **LZ77 backward references**, with distance mapped to 2D so a vertical neighbour gets a short code.
4. **Huffman with meta-Huffman** — an "entropy image" assigning a different Huffman code set per tile — plus a colour cache.

The crux: **WebP stores residuals of the index map; shapes store geometry of regions.** Geometry is the redundancy WebP never pays for.

Worth noting for later: transform 4 means WebP-lossless *already carries a spatial segmentation into regions that share statistics*. It simply never transmits boundaries, because the tile grid is free. The same is true in the lossy path — a VP8 key frame carries a segmentation map of up to 4 segments with per-segment quantizers, coded as a cheap tree — and in AV1, where wedge and geometric partition modes encode a straight-line boundary inside a block from a small codebook for a handful of bits. **The shape idea is already inside all three winners, in the only form that pays: implicit, block-local, and derivable without an explicit global boundary map.**

## Porting the levers one at a time

Same eval, same fidelity (28.61 dB). Entropy rows are the exact size an adaptive arithmetic coder outputs — its cross-entropy under an add-one model, within ~1 byte. The decoder rebuilds the same adaptive model, so there is no side-information cost.

| # | Method | Size | Lever |
|---|---|---|---|
| C | quadtree (shapes) | 44.8 KB | geometry, 2D |
| E | row-RLE (shapes, implicit x) | 29.1 KB | dropped x,y,w,h |
| D | indexed PNG | 27.2 KB | wall |
| — | **WebP-lossless** | **25.2 KB** | **wall** |
| G | entropy order-1 (left) | 22.1 KB | predict from left |
| **H** | **entropy order-2 (left, up)** | **17.4 KB** | **2D context** |
| I | entropy order-3 (+ upleft) | 18.3 KB | *see corrections* |

> **Correction (report 06), two claims struck.**
>
> **(1) "Beats WebP by 31%" was a framing error.** It compares a *lossy* pipeline (28.61 dB) against WebP in *lossless* mode. Lossless-of-our-grid is not a fidelity anyone asked for — it is lossless reproduction of our own quantization error. At matched fidelity on the original image, WebP q34 = 12.35 KB @ 28.63 dB and AVIF q30 = 8.9 KB @ 28.71 dB, so 17.4 KB is **1.41× and 1.95× behind**, not 31% ahead.
>
> **(2) "Order-3 overfits, so we are at the floor" was wrong.** Order-3 has 4,096 contexts over 147,456 samples — about 36 samples each, which is *starvation*, not saturation. If it were dilution, mixing could not recover it either. Measured with online logistic mixing of orders 0–4 (`code/mix/`): order-2 alone 16.9 KB, order-3 alone 17.5 KB, **mix of 0–4 = 16.2 KB**. Mixing recovers the loss, so the model was starved and the raster baseline is 16.2 KB, not a floor at 17.4 KB.

## Verdict

- **The lever is not shapes.** Geometry plateaus at row-RLE (29.1 KB); every gain past that comes from dropping shapes entirely and context-coding the palette index map.

- **Why order-2 beats WebP's *lossless mode* specifically:** WebP-lossless uses static Huffman plus LZ77; the order-2 coder uses adaptive arithmetic coding with a (left, up) context. Adaptive arithmetic beats static Huffman on this content — which is how FLIF and JPEG-XL modular beat WebP-lossless in the literature, so that part is credible. It says nothing about WebP's lossy mode, which is the mode anyone would actually use, and which wins.

- **Full circle.** The path that makes shapes small converts them into a modern lossless raster codec. The only surviving piece of "ours" is pixelize's palette — itself one of WebP's four transforms.

## What this means for the project

- **Bytes:** nothing here is shippable as a size win. The order-2 coder is a competent lossless coder *of our own quantization error*, which is not a product.
- **Shapes:** belong on the visualization axis — resolution-independent geometry, per-part animatable drawing, per-region addressability. Real properties, none of them measured in bytes.
