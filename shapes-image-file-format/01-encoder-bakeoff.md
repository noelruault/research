# 01 — Encoder bake-off: is a shape cover a good compression of an image?

**Question.** `noelruault/images` converts a raster into a lossless cover of coloured rectangles and ships it as SVG. Is that a good *compression* of the image, and if not, what actually gets a small file?

**Method.** One fixed eval, many single-variable encoders, honest measurement, keep or discard. Harness: `code/bakeoff/` (SVG, binary varint, quadtree, indexed PNG, PSNR). External codecs via `cwebp` / `avifenc` / `pngquant` / `sips`.

## Eval

- macOS Sierra wallpaper, resized to 512×288, quantized to n=16 colours via pixelize.
- Every *lossless-of-grid* encoder reproduces that exact quantized image, so fidelity is identical across them: **PSNR(quant vs original) = 28.61 dB**. They differ only in bytes — a clean bytes race.
- Lossy codecs are measured on the original and reported at their decoded PSNR, so the frontier is comparable at equal fidelity.

## Result 1 — lossless of the n=16 grid (same pixels, 28.61 dB)

| Encoder | Bytes | Discipline |
|---|---|---|
| WebP `-lossless` | **25.2 KB** | perceptual codec, lossless mode |
| indexed PNG (stdlib) | 27.2 KB | entropy-coded raster + 2D prediction |
| pngquant | 27.3 KB | palette PNG |
| quadtree bitstream (ours) | 44.8 KB | spatial tree |
| AVIF `--lossless` | 83.4 KB | bad fit for flat-index content |
| binary varint rects (ours) | 101.5 KB | serialization |
| **SVG rect cover (ours, shipped)** | **110.9 KB** | vector |

## Result 2 — lossy on the original, bytes at matched fidelity (~28.6 dB)

| Codec | Bytes | PSNR |
|---|---|---|
| **AVIF q30** | **8.7 KB** | 28.33 dB |
| WebP q35 | 12.3 KB | 28.71 dB |
| JPEG q30 | 15.8 KB | 27.92 dB |

AVIF q45 (16.5 KB @ 31.7 dB) and WebP q50 (16.2 KB @ 30.0 dB) buy *more* fidelity than this pipeline reaches at all, still far under its bytes.

## Verdicts

1. **The SVG rect cover is the worst encoder tested — discard it as a compression strategy.** A plain PNG of the same pixels is 4× smaller; AVIF at the same fidelity is ~13× smaller. Keep SVG as a render/delivery convenience, never as a size play.

2. **Binary `.shapes` barely helps after gzip — discard it as a "lever".** Raw drops 1.14 MB → 212 KB, but gzipped it is 101.5 KB against the SVG's 110.9 KB: an ~8% wire win, not the hoped-for step change. gzip already recovers most of the text overhead. This corrects an earlier note in the project that called binary "the efficiency lever".

3. **Quadtree is the best shape-domain encoder — keep as the internal winner (2.5× over SVG gz), but it still loses to PNG.** 2D decomposition beats a 1D rect list because it captures spatial correlation, which is a hint about where the real lever is.

4. **"Just pixels plus a colormap" beats every shape encoder.** PNG entropy-codes the index map with Paeth prediction; explicit per-region geometry cannot compete on a high-region-count photo — 32,924 rects here, for 147,456 pixels, which is 4.5 px per region and not meaningfully "shapes" at all. That observation is chased down in report 04, where it turns out to be the one real win available.

5. **For photos, use a real codec.** AVIF hits this fidelity in 8.7 KB.

## Where shapes were thought to win

Rects are O(regions), a raster is O(pixels), so a shape format should win when regions ≪ pixels *and* resolution independence matters: flat art, logos, icons, UI, diagrams, pixel art.

> **Correction (report 06).** The supporting claim — "who-king at a 96px grid is 9 KB gz and renders at 8K for the *same* 9 KB" — is fake vectorness for this pipeline. A rect cover of a *quantized grid* has no sub-grid geometry, so upscaling it is mathematically identical to nearest-neighbour upscaling of a tiny PNG, which is far smaller. Measured on 4×-upscaled pixel art the region coder does win (5,422 B vs order-2's 6,657 B vs WebP-lossless's 9,246 B), but at native resolution it loses (1,386 B vs 1,173 B). Resolution independence is only a genuine win when the geometry is authored *above* the pixel grid, which a cover of a raster never is.
