# 03 — Shared priors: dictionaries and cached corpora

**Question.** WebP and PNG are per-image and carry a fixed generic prior. If a *corpus* ships with a shared dictionary, could each image become a tiny delta and beat them?

There are two versions of this idea and both were tested.

## Round 3 — zstd dictionaries on index maps

| Corpus | Per-image alone | With cached shared dict | Dict size |
|---|---|---|---|
| 6 dissimilar photos | 266.3 KB | 267.2 KB (no gain) | ref image |
| 32 self-similar tiles (one photo) | 23.0 KB | 19.2 KB (−17%/tile, *if cached*) | 21.7 KB once |

- **The win is conditional on (redundancy × reuse), not on shape cleverness.** Dissimilar images: 0%. Similar tiles: ~17% per image, but only once the dictionary is amortized across many downloads — for a single shipment the dictionary cost more than it saved.
- **It is format-agnostic.** Compression Dictionary Transport (shipping in Chrome) gives WebP the same benefit. A dictionary does not make a shape representation beat WebP; both gain equally where a redundant corpus exists.

## Round 5 — the best-case corpus, built to favour shapes

The stronger version of the hypothesis, and the last live idea in the whole investigation:

> Entropy-coded rasters have near-zero cross-file byte redundancy — their bytes are already incompressible. Structured shape data (a rect list, an SVG) has enormous cross-file redundancy. So a dictionary shared across an asset set should favour shapes *asymmetrically*, even if per-image shapes lose.

The corpus was constructed to favour it as hard as possible: 8 images, **one shared 16-colour palette** (pico8), **identical 96×96 dimensions**, so palette tokens, coordinate patterns and SVG scaffolding all repeat across files. Method: `brotli -q 11`, leave-one-out (dictionary = the other 7 files concatenated). Raw output in [`03-corpus-dictionary-data.txt`](03-corpus-dictionary-data.txt).

| format | solo | + shared dict | gain |
|---|---|---|---|
| WebP-lossless | 15,494 B | 15,383 B | **1.01×** |
| `.svg`.br | 43,765 B | 42,142 B | **1.04×** |
| `.shapes`.br | 54,937 B | 53,789 B | **1.02×** |

**The asymmetry does not exist.** Shapes gain 2–4%, WebP gains 1%. Shapes stay 2.8–3.5× behind before the dictionary and after it. The intuition — that structured text is dictionary-friendly in a way compressed rasters are not — is true in kind but far too small in degree to matter, because brotli at `-q11` has already found the intra-file redundancy and the cross-file residue is thin.

## The real lever behind all of this

Every byte-lever tried across five rounds — prediction, context, LZ, dictionary — is the same move: **shift information out of the file and into a prior the decoder already shares.** A predictor is a 1-pixel prior; a context model is a few-pixel prior; a dictionary is a cached-corpus prior; the limit is a *learned model* (neural or implicit-neural-representation codecs), where a large cached decoder reconstructs detail from a tiny latent.

"Better and lighter" for real content lives on that axis. WebP ships a small fixed prior and cannot specialize; a domain-specific shared prior is the only structural way past it — and at that point it is no longer shapes, nor even a generic codec, but a shared-prior system for a bounded domain. Note also that this is where the research frontier already sits: diffusion and neural codecs beat AVIF and BPG at ultra-low bitrates on perceptual metrics, which is the same band report 05 identifies as the only place shapes are competitive.
