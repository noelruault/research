# 21 — The format is a file now, and parity is proven

**Question (P4).** Every byte figure in twenty reports was a cross-entropy. `binModel` accumulates counts and returns `−log2(p)`; nothing in this codebase had ever emitted a byte. Reports 16 and 17 compared that idealised number against **real WebP files with real containers**, and both said in terms that parity was *plausible and unproven* until someone built the bitstream. The margins were 1,198 B and 301 B — a header and a coder flush could plausibly have eaten either.

**Answer.** The container costs **~20 bytes**, not ~1%. Both headlines survive, and the file round-trips bit-exactly.

## The two real files

| mark | cross-entropy estimate | **real file on disk** | overhead | round trip |
|---|---|---|---|---|
| 11,121 regions / 3840×2160 | 132,280 B | **132,301 B** | +21 B (**+0.016%**) | **0 wrong of 24,883,200** |
| 3,546 regions / 960×540 | 25,399 B | **25,418 B** | +19 B (**+0.079%**) | **0 wrong of 1,555,200** |

Against WebP, both sides real files:

| | WebP | PSNR | **shape file** | PSNR | delta | previously claimed |
|---|---|---|---|---|---|---|
| 28.5 dB | `-m 6 -q 3` — 131,082 B | 28.52 | **132,301 B** | 28.51 | **+0.930%** | +0.91% (estimate) |
| capability | `-m 6 -q 28 -resize 960 540` — 25,700 B | 24.99 | **25,418 B** | 24.97 | **−1.097%** | −1.2% (estimate) |

Report 16 budgeted "roughly the remaining 1%" for the container. **It costs 0.016%.** The estimates were honest to within tens of bytes.

## The round trip is the deliverable

`lab p4dec <file>.shpc <out>.png` reads the file and nothing else — no partition, no side table, no reference. Verified independently of the agent that built it: the decoded PNG is **byte-identical** to the published render at both marks (`md5` matches, not merely pixel-equality), and the decoded 4K file measures **28.51 dB** against the true source.

**Causality is now enforced by the format rather than asserted beside it.** The decoder builds its context out of the planes it is progressively filling in, so a non-causal tap would diverge and the round trip would fail loudly. Falsification #12 cannot hide in this path — it is structurally excluded rather than checked for.

## Where the twenty bytes go

| component | 11,121 / 4K | 3,546 / 960 |
|---|---|---|
| range coder vs its own cross-entropy | +4.46 B | +4.08 B |
| header + chunk framing | +17 B | +16 B |
| **total** | **+21.46 B** | **+20.08 B** |

The coder's ~4 B is its terminator — a carry-catcher byte plus a flush, less what full-precision interval splitting wins back. **It does not scale with the file**: the same ~4 B on a 16 KB stream and on a 105 KB one. That is a consequence of splitting from the raw `binModel` counts rather than a quantised probability table.

## SHPC v1

```
magic "SHPC"(4) · version(1) · uvarint W, H, nregions, wallLen, colourLen · coef(8) · wall chunk · colour chunk
```

Header is 25 B at 4K, 24 B at 960. The wall chunk is the **interleaved** `interAsym` schedule — the one `lab wallcheck` confirms decodable, not `caeBytes`' two-pass schedule. The colour chunk is report 15's `a` arm: planar RCT residuals, `brotli -q11`, with the 8 bytes of chroma coefficients in the header.

Encode 0.7 s, decode 1.9 s at 4K — on top of the 3m44s merge that produces the partition (report 18). Serialisation is not the encoder's cost.

## A regression this caught, which was mine

`lab wallxexact` and `lab wallx` were **exiting 1 before printing a row**. Both assert that one `crossplane.go` variant reprices `caeBytes` exactly, and that variant was `base` — until report 20 made `caeBytes` legal. The stale reference then compared the illegal coder against the legal one and fired on the legality cost itself: 1,500 B at 960×540, 5,536 B at 4K.

**Introduced by report 20 and invisible to me**, because I ran `wallxexact` *before* applying that fix and never re-ran it after. Now pinned by a test, and the `"vs base"` column label — wrong for the same reason — reads `vs caeBytes`. No published number changes; every quoted figure comes from the `interAsym` row.

## Not settled

- **One photograph**, two operating points. B7 is now the largest open item in the study by a distance.
- **SHPC v1 is the minimum that carries these two marks.** No error resilience, no truncation, no metadata, no alpha, no colour-space tag. The truncatable progressive stream that report 13 lists as a strength (and the merge hierarchy already supports) is *not* implemented.
- The colour chunk requires an external `brotli` at both ends.
- Encode still costs 3m44s at 4K (report 18, P8 open).

## What this changes

**"Parity is plausible" becomes "parity is measured."** The shape coder is +0.93% against WebP at 28.5 dB and −1.10% at the operating point where its segmentation is useful, as real files, round-tripping exactly — while carrying an addressable region graph WebP cannot carry at any size, and beating WebP-plus-a-sidecar by 40–44% (report 19).

Every remaining caveat in this study is now about **generality**, not about the numbers.
