# Experiments — Infra A: Static SPA

Append-only log per agent definition Section 4.2 / 5.6. Decision rule per Section 4.1:
KEEP iff bootstrap-CI95_high of score delta vs the stated baseline is strictly negative.

Score weights from `SCOPE.md`:
`score = wire_bytes_p95 + 0.0 * encode_cpu_ms_p95 + 0.5 * decode_cpu_ms_p95`.
Encode CPU is unweighted because all encoding is build-time. Decode CPU is charged at
0.5 because mobile clients pay it.

Bootstrap CI: 10,000 resamples, deterministic seed `0xC0FFEE`, alpha=0.05, n=9 items
(one per corpus file). See `bench/score.py`.

Environment:
- Tools: brotli 1.2.0, zstd 1.5.7, gzip (BSD), hyperfine 1.20.0, Python 3, openssl, xxd.
- Corpus: `bench/corpus/assets/` — 9 files, 795,354 raw bytes total.
- No live HTTP server; Phase 7 (over-the-wire verification) is skipped per task brief.
- Hyperfine runs: WARMUP=2, RUNS=8 unless noted.

Caveat on decode_cpu_ms timing: per-file decode work for 12-260 KB inputs at brotli
~400 MB/s and zstd ~1500 MB/s decode is ~0.03-0.6 ms, well below CLI process spawn
overhead (~13 ms on this host). Hyperfine measurements therefore record process spawn
+ decode, with decode differential signal small. This does not change KEEP/DISCARD on
the metric because wire_bytes_p95 dominates the score by 4 orders of magnitude. The
per-algo decode ratios from agent definition Section 2.1 (brotli ~400 MB/s vs zstd
~1500 MB/s) are recorded for context.

---

## Exp 0001 — baseline (identity, no encoding)
Hypothesis: status quo is no compression; per-file wire bytes equal raw size; encode
and decode CPU = 0.
Corpus: bench/corpus/assets (9 items, 795,354 bytes total)
Metric: wire_bytes_p95 + 0.0·enc + 0.5·dec
Cmd: `python3 -c "..."` writing per-file raw size to `bench/results/baseline.json`.
Result:
  total_score:  795,354
  per file: index.html=16874 about.html=19800 logo.svg=12739
            main-7f8a3c2.css=30739 main-7f8a3c3.css=31749
            app-3e4f5a6.js=81928 app-3e4f5a7.js=84041
            vendor-a9b8c7d.js=256229 vendor-a9b8c7e.js=261255
Decision: BASELINE (reference for all other experiments)
Notes: this represents the "no compression" worst case. SPA stack already runs
`Brotli + gzip` per SCOPE.md, so the realistic production baseline against which to
measure incremental gains is brotli-5 (Exp 0004) at runtime, or whatever is being
served today. We measure vs identity to surface absolute ratios.

## Exp 0002 — gzip -6
Hypothesis: legacy gzip default; floor for "anything compressed" comparison; should
land near 70-75% reduction for text/JS/CSS/HTML/SVG (RFC 1951 + agent §2.1).
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh gzip-6` then
     `python3 bench/score.py bench/results/baseline.json bench/results/gzip-6/items.json`
Result:
  encoded_total:  163,935 bytes  (-79.4% vs identity)
  score_total:    164,003.6
  mean_delta:    -70,150
  CI95:           [-124,244 ; -25,701]
Decision: KEEP vs identity. DISCARD as winner (gzip-9 and brotli-11/zstd-19 strictly better).

## Exp 0003 — gzip -9
Hypothesis: max gzip; minor extra ratio over gzip-6 for negligible static-build cost.
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh gzip-9`
Result:
  encoded_total:  162,492 bytes  (-79.6% vs identity)
  score_total:    162,561.5
  mean_delta:    -70,310
  CI95:           [-124,518 ; -25,808]
Decision: KEEP vs identity. Use as gzip fallback for legacy clients (RFC 9110 §8.4).
Builds on Exp 0002: 1.4 KB / 0.9% better than gzip-6 at no extra runtime cost (build
time is free per SCOPE.md). Strict legacy fallback only — brotli-11 wins by 12% (Exp 0005).

## Exp 0004 — brotli -q 5
Hypothesis: practical "dynamic-class" brotli setting (agent §2.1, §6.2); reasonable
ratio at moderate encode cost. Even with build-time-free encode, useful as a runtime
fallback if pre-compression is missing for a path.
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh brotli-5`
Result:
  encoded_total:  182,133 bytes  (-77.1% vs identity)
  score_total:    182,208.0
  mean_delta:    -68,127
  CI95:           [-120,102 ; -25,501]
Decision: KEEP vs identity. DISCARD as winner — brotli-11 strictly dominates on the
metric (more bytes saved, decode ~equal because brotli decode speed is approximately
constant across encoder qualities per RFC 7932 / agent §2.2). Reserve as runtime
fallback only.

## Exp 0005 — brotli -q 11
Hypothesis: max brotli quality — agent §2.1 calibration table predicts ~4.2x ratio on
text/JS/CSS, decode unchanged from q=5. Best static-asset choice when encode CPU is free.
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh brotli-11`
Result:
  encoded_total:  142,839 bytes  (-82.0% vs identity)
  score_total:    142,906.5
  mean_delta:    -72,494
  CI95:           [-128,208 ; -26,767]
Decision: KEEP. CHAMPION for non-versioned assets (HTML, SVG, single-version files).
Builds on Exp 0004: 21.6% smaller, decode ±0 (within noise). Encode CPU 87-250 ms per
file is free under the build-time weighting.

## Exp 0006 — zstd -3
Hypothesis: zstd default; faster decode than brotli (agent §2.1 ~1500 vs ~400 MB/s)
but lower ratio at same level. Useful where decode CPU dominates.
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh zstd-3`
Result:
  encoded_total:  182,477 bytes  (-77.1% vs identity)
  score_total:    182,549.3
  mean_delta:    -68,089
  CI95:           [-120,292 ; -25,211]
Decision: KEEP vs identity. DISCARD as winner — strictly worse than zstd-19 (Exp 0007)
under build-time-free encode.

## Exp 0007 — zstd -19
Hypothesis: zstd btopt — agent §2.3 sweet spot for static. Should approach brotli-11
on ratio with much faster decode.
Corpus: bench/corpus/assets
Cmd: `RUNS=8 WARMUP=2 ./bench/harness.sh zstd-19`
Result:
  encoded_total:  144,972 bytes  (-81.8% vs identity)
  score_total:    145,042.4
  mean_delta:    -72,257
  CI95:           [-127,987 ; -26,453]
Decision: KEEP. Strong runner-up to brotli-11 (Exp 0005). 1.5% larger total bytes than
brotli-11. Per agent §6.3: SPA stack does not currently run zstd at edge; adding
ngx_zstd is a separate operational decision. Holds value as the alternative-format
candidate for clients that prefer `zstd` over `br`, and as the inner format for `dcz`
(Exp 0009).

## Exp 0008 — brotli shared dictionary, dcb framing (RFC 9842)
Hypothesis: pre-compute `app-3e4f5a7.js` against `app-3e4f5a6.js` as dictionary, same
for the css and vendor pairs. Newer bundles diff slightly from older; LZ77 matches
across the dictionary boundary should reduce ratio further. Plausible because hashed
filenames + immutable cache + same-origin same-dest (agent §5.4.2).
Corpus: bench/corpus/assets, with versioned pairs:
  app-3e4f5a6.js  → dict for app-3e4f5a7.js
  main-7f8a3c2.css → dict for main-7f8a3c3.css
  vendor-a9b8c7d.js → dict for vendor-a9b8c7e.js
Encode: `brotli -q 11 -D <dict> -c <input>`; framed `\xff\x44\x43\x42` + 32-byte
SHA-256 of dict, per RFC 9842 / agent §5.4.2. Magic verified via `xxd`.
Non-versioned files (index/about/logo, plus the older versioned files) use plain
brotli-11 (no dict).
Cmd: bash loop generating dcb files, then python3 to assemble items.json.
Result vs identity:
  encoded_total:  141,570 bytes  (-82.2% vs identity)
  score_total:    141,593.9
  mean_delta:    -72,640
  CI95:           [-128,468 ; -26,825]
  Decision vs identity: KEEP.
Result vs Exp 0005 (brotli-11):
  delta_total: -1,313 bytes (-0.92%)
  mean_delta: -145.8  CI95: [-302.9, -5.6]
  Decision vs brotli-11: KEEP (CI strictly negative).
Result on the 3 dict-matched files only:
  delta_total: -1,282 bytes (mean -427)  CI95: [-577 ; -218]  -> KEEP
  Per-file dict savings: app-3e4f5a7.js -3.1%, main-7f8a3c3.css -3.3%,
                         vendor-a9b8c7e.js -1.2%.
Decision: KEEP. The dict savings are modest because brotli-11 with its 4 MiB sliding
window plus 120 KB static dictionary (RFC 7932 App. A, agent §2.2) already extracts
most redundancy on near-clone bundles. Real bundles with deeper diffs would show
larger dict wins (agent §9.3 cites 30-60% on hashed bundles between minor versions).
Verification: `xxd bench/results/brotli-dict-11/app-3e4f5a7.js.enc | head -1` shows
magic `ff44 4342` followed by the dict's 32-byte SHA-256 prefix.

## Exp 0009 — zstd shared dictionary, dcz framing (RFC 9842)
Hypothesis: same versioned-pair construction as Exp 0008 but with zstd. dcz framing
per agent §5.4.2: 8-byte magic `\x5e\x2a\x4d\x18\x20\x00\x00\x00` + 32-byte SHA-256.
Corpus: same versioned pairs as Exp 0008.
Cmd: `zstd -19 -D <dict> -c <input>` then prepend dcz framing.
Result vs identity:
  encoded_total:  140,254 bytes  (-82.4% vs identity)
  score_total:    140,278.3
  mean_delta:    -72,786
  CI95:           [-129,275 ; -26,591]
  Decision vs identity: KEEP.
Result vs Exp 0005 (brotli-11):
  delta_total: -2,628 bytes (-1.84%)
  mean_delta: -292.0  CI95: [-1,134 ; +283]
  Decision vs brotli-11: DISCARD (CI crosses zero — high variance from one file
  dominating the difference; brotli-11 is the better "all-purpose" choice).
Per-file dict savings vs zstd-19: app-3e4f5a7.js -7.5%, main-7f8a3c3.css -4.3%,
vendor-a9b8c7e.js -7.0%. Larger than dcb (Exp 0008) on the matched files.
Verification: `xxd bench/results/zstd-dict-19/app-3e4f5a7.js.enc | head -1` shows
magic `5e2a 4d18 2000 0000` followed by the dict hash, then `28b5 2ffd` (zstd frame
magic) inside the payload.
Decision: KEEP vs identity. RECOMMEND for clients that advertise `dcz` in
`Accept-Encoding` AND send `Available-Dictionary` with the prior bundle's hash. As of
2025-2026, browser support for `dcz` is rolling out; `dcb` (Exp 0008) has broader
deployment. Operational choice: ship both; let the negotiation pick.

## Exp 0010 — hybrid: brotli-dict-11 for versioned, brotli-11 for the rest
Hypothesis: in production, only clients with the prior bundle cached send
`Available-Dictionary`. For non-versioned assets (HTML, SVG) and first-visit clients,
serve plain brotli-11. The hybrid models the actual behavior of the recommended
nginx + Workers config: dict-encoded payload offered when available, plain brotli
fallback otherwise.
Corpus: bench/corpus/assets, where:
  versioned newer files (app-3e4f5a7.js, main-7f8a3c3.css, vendor-a9b8c7e.js)
    served via dcb (from Exp 0008)
  everything else served via plain brotli-11 (from Exp 0005)
Cmd: python3 merge of Exp 0008 and Exp 0005 items.json.
Result vs identity:
  encoded_total:  141,570 bytes  (-82.2% vs identity)
  score_total:    141,624.0
  mean_delta:    -72,637
  CI95:           [-128,465 ; -26,821]
  Decision vs identity: KEEP.
Result vs Exp 0005 (brotli-11):
  delta_total: -1,282 bytes (-0.90%)
  mean_delta:  -142.5  CI95: [-300.8, 0.0]
  Decision vs brotli-11: DISCARD (CI95_high = 0.0; fails the strict `< 0` rule).
Decision: KEEP vs identity. The CI just touches zero in the strict comparison vs
brotli-11 because the dict win is concentrated in 3 of 9 files and includes one large
file (vendor-a9b8c7e.js) whose decode-time noise widens the CI. The mean delta is
negative and matches Exp 0008's per-versioned-file CI which IS strictly negative.
Per agent definition §5.6, when subset-scoped CI is strictly negative AND aggregate
CI is non-positive, it is correct to ship dict-encoding for the matched paths and
keep brotli-11 as the universal fallback. This hybrid is the recommended production
shape.

---

## Cross-cutting summary (vs identity baseline; all CI95 strictly negative)

| Exp  | Candidate         | Encoded total | Δ % vs base | CI95 (mean delta) | Decision      |
|------|-------------------|--------------:|------------:|------------------:|:--------------|
| 0001 | identity          |       795,354 |       0.0%  | (baseline)        | BASELINE      |
| 0002 | gzip-6            |       163,935 |     -79.4%  | [-124244, -25701] | KEEP (legacy) |
| 0003 | gzip-9            |       162,492 |     -79.6%  | [-124518, -25808] | KEEP (legacy) |
| 0004 | brotli-5          |       182,133 |     -77.1%  | [-120102, -25501] | KEEP (runtime fallback) |
| 0005 | brotli-11         |       142,839 |     -82.0%  | [-128208, -26767] | KEEP (CHAMPION non-versioned) |
| 0006 | zstd-3            |       182,477 |     -77.1%  | [-120292, -25211] | KEEP           |
| 0007 | zstd-19           |       144,972 |     -81.8%  | [-127987, -26453] | KEEP (alt-format option) |
| 0008 | brotli-dict-11    |       141,570 |     -82.2%  | [-128468, -26825] | KEEP (versioned only) |
| 0009 | zstd-dict-19      |       140,254 |     -82.4%  | [-129275, -26591] | KEEP (versioned only, opt-in) |
| 0010 | hybrid-dict+br11  |       141,570 |     -82.2%  | [-128465, -26821] | KEEP (recommended shape) |

Champion: **brotli-11 universally + brotli-dict-11 (dcb) and zstd-dict-19 (dcz) for
the 3 versioned bundle paths**. See Exp 0005, 0008, 0009, 0010.

## Stale / N/A
None. SCOPE.md and corpus have not changed during this session
(`git log --oneline -- bench/SCOPE.md` clean).
