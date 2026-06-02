# EXPERIMENTS — Infra A v3 (Static SPA, smoke test)

Append-only log. One section per experiment.

Metric: `wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95` (per SCOPE.md).
Bootstrap CI: 10,000 resamples, seed 0xC0FFEE, alpha=0.05.
Corpus: `bench/corpus/assets/` (9 files, ~795 KB total).
Timing: `bench/measure.py` (v3 §4.5), 30 runs + 3 warmup, time.perf_counter_ns.

---

## Exp 0001 — identity (baseline)
Hypothesis: establish status quo; no compression on the wire.
Corpus: bench/corpus/assets (9 files, 795,354 raw bytes total).
Metric: wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95
Cmd: `python3 bench/measure.py identity bench/corpus/assets bench/results/baseline.json`
Result:
  - Total raw bytes: 795,354
  - Total wire bytes: 795,354 (no encoding)
  - encode/decode timings reflect `cat` subprocess overhead (~5–14 ms per file).
Decision: KEEP-AS-BASELINE. All later candidates are scored against this file.
Notes: Largest file is `vendor-a9b8c7e.js` at 261,255 B; smallest `logo.svg` at 12,739 B.

## Exp 0002 — gzip-6
Hypothesis: legacy fallback (RFC 1952), broadly supported. Establishes the floor any
brotli/zstd candidate must beat.
Cmd: `python3 bench/measure.py gzip-6 bench/corpus/assets bench/results/gzip-6/items.json`
       `python3 bench/score.py bench/results/baseline.json bench/results/gzip-6/items.json`
Result:
  - Total wire bytes: 163,804 (-79.4% vs identity)
  - mean Δ score: -70,172.9 bytes (-78.52%)
  - CI95 bytes: [-124,268.2, -25,721.5]
  - CI95 pct:   [-80.67%, -76.34%]
  - Decision: KEEP (CI95_high < 0)
Notes: gzip is the per-asset minimum-supported encoding; surprisingly beats brotli-5
on this synthetic JS corpus, see Exp 0003.

## Exp 0003 — brotli-5
Hypothesis: dynamic-friendly brotli (RFC 7932 quality 5, encoder default-ish), should
beat gzip-6 on HTML/CSS/JS by ratio.
Cmd: `python3 bench/measure.py brotli-5 bench/corpus/assets bench/results/brotli-5/items.json`
Result:
  - Total wire bytes: 182,161 (-77.1% vs identity)
  - mean Δ score: -68,132.4 bytes (-77.96%)
  - CI95 bytes: [-120,110.3, -25,505.1]
  - CI95 pct:   [-80.48%, -75.95%]
  - Decision: KEEP (CI95_high < 0) vs identity, but DEFEATED by gzip-6 on JS files in
    this corpus.
Notes: per-file table — brotli-5 produced *more* bytes than gzip-6 on every JS file
(app-3e4f5a6.js: 19109 vs 17035; vendor-a9b8c7d.js: 58958 vs 51395). It still beats
gzip-6 on HTML, CSS, SVG. Hypothesis: synthetic JS in this corpus has lower overlap
with brotli's RFC 7932 Appendix A static dictionary (HTML/JS-tuned but for *real*
web text); at quality 5 the encoder cannot recover from the mismatch. This echoes
v3 §2.5 (the original observation was for JSON; here we see it on synthetic JS too).
Brotli does win at quality 11 (Exp 0004).

## Exp 0004 — brotli-11
Hypothesis: static-asset champion (RFC 7932). Build-time encode cost is irrelevant
(scope weight α=0.0); ratio should win.
Cmd: `python3 bench/measure.py brotli-11 bench/corpus/assets bench/results/brotli-11/items.json`
Result:
  - Total wire bytes: 142,836 (-82.0% vs identity, smallest of all candidates)
  - mean Δ score: -72,501.9 bytes (-82.07%)
  - CI95 bytes: [-128,215.6, -26,775.3]
  - CI95 pct:   [-83.88%, -80.57%]
  - Decision: KEEP (CI95_high < 0). Winner among non-dict candidates.
Notes: Encode CPU is heavy as expected (`vendor-a9b8c7e.js` ≈ 423 ms p95 single-shot)
but α=0.0 in the metric per SCOPE.md, so this does not penalize. Decode p95 is
within noise of identity-subprocess overhead (~7–10 ms per file via `brotli -d -c`).
Per v3 §4.5: vendor JS encode at 11 is comfortably above the 5 ms threshold, so
either `measure.py` or `hyperfine` would work; we used `measure.py` for uniformity
with the smaller files.

## Exp 0005 — zstd-19
Hypothesis: static-strong zstd (RFC 8878). Decode throughput is roughly level-
independent; level 19 is the static sweet spot.
Cmd: `python3 bench/measure.py zstd-19 bench/corpus/assets bench/results/zstd-19/items.json`
Result:
  - Total wire bytes: 144,954 (-81.8% vs identity, ~1.5% larger than brotli-11)
  - mean Δ score: -72,264.5 bytes (-81.05%)
  - CI95 bytes: [-127,994.7, -26,460.7]
  - CI95 pct:   [-82.36%, -79.75%]
  - Decision: KEEP (CI95_high < 0).
Notes: Wire bytes very close to brotli-11 on this corpus (142,836 vs 144,954).
Encode is much faster than brotli-11. With α=0.0 in the metric brotli-11 wins by
score, but zstd-19 is the right choice for any scenario where ngx_brotli is not
available or decode-speed dominates.

## Exp 0006 — zstd-dict-19 (versioned subset, dcz framing per RFC 9842)
Hypothesis: prior-version bundle as shared dictionary, new-version bundle as input.
On hashed-filename versioned assets, the diff between adjacent versions is tiny;
dictionary should reduce wire bytes substantially over no-dict zstd-19.
Pairs (older = dict, newer = input):
  - app-3e4f5a6.js → app-3e4f5a7.js
  - main-7f8a3c2.css → main-7f8a3c3.css
  - vendor-a9b8c7d.js → vendor-a9b8c7e.js
Cmd:
  `DICT=bench/corpus/versioned-older/app.js \
     python3 bench/measure.py zstd-dict-19 bench/corpus/dict-pair-app /tmp/dict-app.json`
  (analogous for main and vendor; merged into bench/results/zstd-dict-19/items.json,
   then 40 B dcz framing added per-file)
dcz framing verification (RFC 9842 §3.2):
  - magic: `5e 2a 4d 18 20 00 00 00` (8 bytes)
  - hash:  32-byte SHA-256 of dictionary contents
  - body:  raw zstd frame (begins with `28 b5 2f fd`)
  - example for app pair, dict SHA-256:
    c598b56b2608febeac09173063046f104d33c083b290fa6d6df6aba7fdc3c845
  - verified via:
    `{ printf '\x5e\x2a\x4d\x18\x20\x00\x00\x00'; printf '%s' "$HASH" | xxd -r -p; \
       cat /tmp/payload.zst; } > /tmp/out.dcz && xxd /tmp/out.dcz | head -4`
Result (vs identity baseline on the 3 newer files, framing included):
  - Total wire bytes: 63,835 (vs 377,045 raw, -83.1%)
  - mean Δ score: -104,329.9 bytes (-81.94%)
  - CI95 bytes: [-218,389.2, -25,174.1]
  - CI95 pct:   [-83.62%, -79.53%]
  - Decision: KEEP (CI95_high < 0).
Result (vs zstd-19 no-dict on the same 3 files, framing included):
  - Total wire bytes: 63,835 dict vs 68,729 no-dict (-7.1%)
  - mean Δ pct: -6.73%
  - CI95 pct: [-7.77%, -4.96%]
  - Decision: KEEP — dictionary gives a modest but real win on top of zstd-19.
Notes: dcz framing is a fixed 40 B/file overhead and does not affect the decision.
Encode CPU (zstd-19) is much faster than brotli-11. Decoder must hold the prior
dictionary; per RFC 9842 the client signals possession via Available-Dictionary
header keyed by SHA-256 (base64), and server selects pre-encoded payload by hash.
Roundtrip verified byte-identical for all 3 pairs.
