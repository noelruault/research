# Autoresearch ledger — shapes image format, bottleneck programme

**This file is the handover.** An agent with no memory of the conversation should be able to read this file alone, pick the top open item, and continue without asking anything. Update it in the same commit as the work it describes. Newest results go at the top of the log.

## Mission

Attack the shape coder's remaining gap to WebP **one bottleneck at a time**, biggest share of the bill first. Every claimed improvement is measured on the real eval, applied and committed if it holds, reverted if it does not. A clean negative is a successful iteration; do not tune until a number looks good.

The compression *verdict* is settled and is not what this programme is trying to overturn — see `README.md`. What is open is whether the **coder** is anywhere near its own floor. So far it demonstrably was not: report 09 found a 12.6% win and a decodability bug in a component eight reports had assumed was mature.

## Invariants — violate these and the result is worthless

- **Eval**: the macOS Sierra wallpaper. `src4k.png` (3840×2160), plus 1920/960/512 resamples, in `$SCRATCH/hd/`. PSNR is this study's RGB definition — use `labx psnr`, never your own.
- **Reproduce before believing.** Run the unmodified coder and match the published number *before* changing anything. If you cannot reproduce, that is the finding; stop and report it.
- **Price variants side by side on identical partitions.** Add a pricing function; never replace the baseline. A change that alters the partition invalidates the comparison unless the baseline is re-run on the new partition too (falsification #3).
- **Pay for everything.** Side information, learned parameters, mode flags, second passes — all counted in the same total. Most dead "wins" here died when the side channel was priced.
- **Steelman both sides.** Any knob given to WebP (`-m 6`, encode resolution, delivery pipeline) must be offered to the shape coder too. Falsification #11 is what happens otherwise.
- **Causality.** Every context tap must be supplied by a real scan order. Assert it with a decoder-side replay. Falsification #12 is what happens otherwise.
- **Run it twice.** Go map iteration has produced three separate nondeterminism bugs here (#6, #7).
- **The killed list is binding**: sixteen mechanisms in report 04, twelve claims in report 06. Re-deriving a dead idea is this study's most common failure.
- Shell loops under `bash`, not zsh.

## Where the bill sits at 3840×2160 — this drives priority

| regions | PSNR | total | walls | colour | wall coder actually used |
|---|---|---|---|---|---|
| 227 | 21.99 | 19,819 B | 96.3% | 3.7% | **contour** |
| 1,383 | 24.99 | 50,016 B | 91.6% | 8.4% | **contour** |
| 11,121 | 28.51 | 153,190 B | 79.0% | 21.0% | CAE |
| 96,359 | 32.74 | 533,107 B | 51.8% | 48.2% | CAE |
| 710,144 | 40.42 | 2,413,389 B | 28.7% | 71.3% | CAE |
| 6,356,392 | exact | 12,159,385 B | 6.8% | **93.2%** | CAE |

## Targets

| goal | requirement |
|---|---|
| match WebP at 28.7 dB at 4K | cut 26,438 B = 16.2% of total (**8.3% still to find** after report 09) |
| match WebP at high-rate steps | close −2.39 dB @ 533 KB, −3.20 dB @ 2.41 MB |
| match WebP lossless | cut 36.5% of total ≈ 39% of the colour bill |
| beat AVIF anywhere | 30–50% — treat as out of reach, do not aim here |

## Bottleneck queue — work the top open item

| # | bottleneck | why it is worth doing | status |
|---|---|---|---|
| B1 | **Colour coder** — mean predictor, order-0 residual, **no cross-channel transform** | 48–71% of the bill at high rate, 93% at lossless. The most primitive component in the pipeline against WebP's 14 predictors + cross-colour transform | **OPEN — next up**, awaiting fan-out results |
| B2 | **Contour coder, never examined** | It is the *chosen* coder below ~1,400 regions, where walls are 92–96% of the bill. Every wall result so far (report 09) improves CAE and is therefore worth **exactly zero** at the low-rate arm | **IN PROGRESS** — two agents: junction map (its context is the same 10-bit width report 09 proved under-conditioned at 4K) and turn coding. First deliverable is the `vertBits`/`turnBits` split, which nobody has measured |
| B3 | **`bitsPerEdge = 1.73`, `bitsPerReg = 25.0`** (`potts2.go:15`) measured at 512×288, drive the RD merge key and Ising λ at every resolution | Actual cost is 1.22–1.61 bits/edge at 4K rungs and 0.4534 at lossless. The 4K scale-space is therefore not the coder's own RD frontier — the shape coder is undersold at the resolution where the verdict was sharpened | OPEN — must re-run baseline on re-tuned partitions or reproduces #3 |
| B4 | **Re-price report 08 against a legal wall coder** | #12: published CAE numbers are optimistic by 3.4–12.7%. Report 08's tables are flagged but not corrected | OPEN — bookkeeping, no research risk |
| B5 | **Rung 2 of the rate ladder** | No mark within ±5% of 50,016 B; needs a merge run at ~7,800 regions to settle whether WebP's lead there is real | OPEN — one run |
| B6 | Inconsistent pricing: lossless rung uses `colorBytesLean`, rungs 1–5 use `colorBytes2` (`hd.go:140`) | Two colour coders inside one published table | OPEN — one line |
| B7 | Generalisation: every result is one photograph | The frozen 16-tap template and any colour win may not transfer. Cheapest real test: Kodak-24 at one small size through the existing frontier | OPEN — blocks any "ship it" claim |

## Log — newest first

### 2026-07-29 — report 09: wall coder, two findings (commit `a6d1c73`, pushed)

- **Cross-plane interleave wins: −12.6%** of the wall bill at 11,121 regions (121,047 → 105,752 B). `interVH` gets −11.1% at the *same* 10/10 context width and same 2,048 models; `base12` with 4× the models buys 1.1%. **Schedule, not capacity.** Coding Hz first instead is +10.4% worse. Win band ~25–31.5 dB; reverses at fine partitions (+12.6% at 3.4M regions, +1.7% lossless). At 28.7 dB the WebP deficit goes **+19.3% → +8.3%**.
- **Context width is real but half as good: −6.2%**, and mostly outside the band where CAE is chosen. Settles a review question: the same frozen 16-bit template is **+2–4% worse at 512×288 and −10.3% at 4K**, so the 10-bit context was right for the old eval and under-conditioned at native resolution. Static `H(X|ctx)` falls to 20 bits while adaptive cost turns at 16 → ceiling is model-learning cost.
- **#12: the published CAE coder is not decodable.** `potts.go:311` reads `Hz(x+1,y)` (uncoded) alongside `Hz(x-1,y)`/`Hz(x-2,y)`. Replay: 21,554 bad contexts at 512×288, 51,995 at 960×540. Legality costs +3.4% to +12.7% of the wall bill.
- **#11: the rate ladder was one-sided.** Only WebP got a resolution search below its floor. Given the same knob the shape coder ties rung 1 — **24.59 dB at 20,618 B** vs WebP 24.54 dB at 20,066 B. The published −2.55 dB is retracted.
- Not done: mixing arm still running; report 08 tables not re-priced; template tested on one image only.

## How to resume

1. Read this file, then `README.md`, then `06-corrections-and-falsifications.md` (what is already dead).
2. Pick the top **OPEN** item in the bottleneck queue.
3. Reproduce the relevant baseline number first. Then implement the smallest honest variant beside it.
4. Measure at a low-rate point, a high-rate point, and lossless — a lever can matter at one end and be irrelevant at the other.
5. If it holds: apply, commit with the numbers in the message, push, and add a log entry here. If it does not: add the log entry anyway with the number that killed it, and mark the bottleneck closed-negative. **Both outcomes are results.**

Paths: repo `research/shapes-image-file-format`; Go lab `code/lab` (copy it out, `go build -o labx .`); working assets under the session scratchpad `hd/` (`src4k.png`, `renders4k/`, `ladder_*.txt`).
