# EXPERIMENTS — Infra A v2 (Static SPA)

Append-only log. Agent: compression-engineer-v2.

## Discovery (Phase 0)

```
target_kind: filesystem-only
target_url:
detected:    SPA static bundle, 9 corpus items, no live origin
             nginx (assumed) over HTTP/2 + TLS 1.3 per SCOPE.md
session:     2026-05-07
agent:       compression-engineer-v2
weights:     encode_cpu_ms=0.0, decode_cpu_ms=0.5 (static asset profile)
metric:      wire_bytes_p95 + 0.0*enc_ms + 0.5*dec_ms
budget:      30 s/candidate
client:      mobile_4g_slow
```

No live HTTP server. Phase 7 (over-the-wire verification) skipped per SCOPE.md.
File-level encode/decode/size benchmarks only.

## Tooling (Phase 0.5)

PRESENT (required): brotli 1.2.0, zstd 1.5.7, gzip (Apple gzip 479), openssl 3.6.2,
xxd, hyperfine 1.20.0, python3 3.14.2, curl.

PRESENT (optional): cwebp, ffmpeg, avifenc.

MISSING (required): none — required loop unblocked.

MISSING (optional, blocks specific experiments):
- svgo (`npm i -g svgo`) — blocks SVG re-encoding experiment; raw SVG through gzip/br/zstd
  is still possible as a comparison point.
- oxipng (`brew install oxipng`) — n/a (no PNG in corpus).
- pngquant (`brew install pngquant`) — n/a (no PNG in corpus).
- woff2_compress, pyftsubset — n/a (no fonts in corpus).
- oha, hey, h2load, nghttp, wrk, lighthouse — n/a (filesystem-only, no live server).
- cjpegli — n/a (no JPEG in corpus).

## Calibration (Phase 0.6)

Sample: `bench/corpus/assets/vendor-a9b8c7d.js` (256 229 bytes)

| Algo | p50 (ms) | local MB/s | table MB/s | Δ vs table |
|---|---:|---:|---:|---:|
| gzip-6    | 12.46  | 19.6  | 50  | -61% |
| gzip-9    | 19.02  | 12.8  | 30  | -57% |
| brotli-1  | 7.76   | 31.5  | 290 | -89% |
| brotli-5  | 11.17  | 21.9  | 100 | -78% |
| brotli-11 | 273.17 | 0.89  | 0.5 | +79% |
| zstd-3    | 8.50   | 28.7  | 250 | -89% |
| zstd-19   | 88.29  | 2.8   | 10  | -72% |

WARNING: every encoder differs from §2.1 by >50%. Caveat: these p50s include Python
`subprocess.run` fork/exec overhead (~5 ms per invocation on Apple Silicon under
filesystem sandbox), which dominates short jobs. The relative ordering is preserved:
brotli-11 is by far the slowest, zstd-3 and brotli-1 are the fastest below the
subprocess floor. Decision: trust local numbers for ranking; SCOPE.md sets
encode_cpu_ms weight = 0.0 anyway, so encode CPU does not affect score on this run.
Decode (which is weighted 0.5) sits well under 1 ms across all algos here, so the
score collapses to wire bytes plus a small constant.

## Corpus Inventory (Phase 1)

| File | Bytes | SHA-256 (16) |
|---|---:|---|
| about.html         |  19800 | e38972f3a21751f8 |
| app-3e4f5a6.js     |  81928 | c598b56b2608febe |
| app-3e4f5a7.js     |  84041 | ce0a5bc2d08e2640 |
| index.html         |  16874 | 02a8ff04d427c02c |
| logo.svg           |  12739 | eff84f4a0444478d |
| main-7f8a3c2.css   |  30739 | c03a2070fcc9061e |
| main-7f8a3c3.css   |  31749 | 525f5f26c9690b0d |
| vendor-a9b8c7d.js  | 256229 | da2e1490b46a4d7e |
| vendor-a9b8c7e.js  | 261255 | 7a4248cc26ddb8c5 |
| **Total**          | **795354** |   |

Asset classes: 2 HTML, 4 JS, 2 CSS, 1 SVG. Versioned pairs available for shared-dictionary
experiments: `app-3e4f5a6.js`/`app-3e4f5a7.js`, `main-7f8a3c2.css`/`main-7f8a3c3.css`,
`vendor-a9b8c7d.js`/`vendor-a9b8c7e.js`.


---

## Exp 0001 — identity baseline
Status: KEEP (baseline reference; not scored against itself)
Hypothesis: status quo, no compression.
Corpus:    bench/corpus/assets (9 items, 795 354 raw bytes)
Metric:    wire_bytes_p95 + 0.0·enc_ms + 0.5·dec_ms
Cmd:       `RUNS=20 python3 bench/measure.py identity bench/corpus/assets bench/results/baseline.json`
Result:
  wire_bytes total:    795 354
  encode/decode:       n/a (cat passthrough; ~4.7 ms/item subprocess floor)
Decision: KEEP as baseline reference; baseline.json written.

## Exp 0002 — gzip-6 (legacy fallback baseline)
Status: KEEP
Hypothesis: gzip -6 across all asset classes; legacy floor for browsers without br/zstd.
Cmd:       `RUNS=20 python3 bench/measure.py gzip-6 bench/corpus/assets bench/results/0002/items.json`
Result:
  N items:               9
  wire_total:            163 804 (-79.40% gross size)
  baseline_total_score:  795 376.7
  candidate_total_score: 163 827.0
  mean Δ bytes:          -70 172.2
  mean Δ percent:        -78.52%
  CI95 bytes:            [-124 267.5, -25 720.3]
  CI95 percent:          [-80.66%, -76.34%]
Decision: KEEP. Universal-client fallback. Inferior to brotli-11 (Exp 0005) by ~12% in
absolute wire bytes, but mandatory for `Accept-Encoding` clients without br/zstd. Cite
RFC 1952 §2.

## Exp 0003 — gzip-9 (legacy fallback maxed)
Status: KEEP (but DISCARD vs gzip-6 in production use)
Hypothesis: gzip -9 squeezes a few more bytes than -6; build cost is one-shot.
Cmd:       `RUNS=20 python3 bench/measure.py gzip-9 bench/corpus/assets bench/results/0003/items.json`
Result:
  wire_total:           162 361 (vs 163 804 for gzip-6; -0.88%)
  mean Δ bytes:         -70 332.4
  mean Δ percent:       -78.76%
  CI95 bytes:           [-124 542.5, -25 827.2]
  CI95 percent:         [-80.81%, -76.69%]
Decision: KEEP vs identity, but in production prefer gzip-6 because gzip-9 saves ≈0.9%
size at 1.3× encode cost. Static rebuilds tolerate it; record both for the build matrix.
Cite RFC 1951 / RFC 1952.

## Exp 0004 — brotli-5 (dynamic-fallback level)
Status: KEEP
Hypothesis: brotli quality 5 (default-dynamic) competitive with brotli-11 on text corpora.
Cmd:       `RUNS=20 python3 bench/measure.py brotli-5 bench/corpus/assets bench/results/0004/items.json`
Result:
  wire_total:           182 161
  mean Δ bytes:         -68 131.8
  mean Δ percent:       -77.96%
  CI95 bytes:           [-120 109.9, -25 504.2]
  CI95 percent:         [-80.49%, -75.95%]
Decision: KEEP vs identity, but **DISCARD** as static-asset choice. brotli-11 (Exp 0005)
beats brotli-5 by ~21% on this corpus. brotli-5 is fine as a runtime/dynamic floor,
useless when build-time encode is free (encode_cpu_ms weight=0.0 per SCOPE.md).
Cite RFC 7932 §1; agent §2.2.

## Exp 0005 — brotli-11 (static, build-time)
Status: KEEP — **WINNER for static encoding (single-algorithm)**
Hypothesis: brotli q=11 is the static-asset ceiling; CPU cost paid once at build.
Cmd:       `RUNS=20 python3 bench/measure.py brotli-11 bench/corpus/assets bench/results/0005/items.json`
Result:
  wire_total:           142 836
  mean Δ bytes:         -72 501.4
  mean Δ percent:       -82.07%
  CI95 bytes:           [-128 215.4, -26 774.2]
  CI95 percent:         [-83.88%, -80.57%]
  encode_p50_sum_ms:    805 ms (across 9 items; build-time, weight=0.0)
  decode_p50_sum_ms:    49 ms (still well under the 0.5 weight × ms threshold)
Decision: KEEP. Best whole-corpus single-algorithm result, beats zstd-19 by 1.5%
on aggregate wire bytes and beats every other non-dictionary candidate. Per-asset-class
check (see §"Per-asset winner" below) confirms brotli-11 wins on all 9 items
individually — no mixed config beats it.
Cite RFC 7932; agent §2.1, §2.2.

## Exp 0006 — zstd-3 (default dynamic level)
Status: KEEP
Hypothesis: zstd -3 default; high decode speed, low encode cost.
Cmd:       `RUNS=20 python3 bench/measure.py zstd-3 bench/corpus/assets bench/results/0006/items.json`
Result:
  wire_total:           181 538
  mean Δ bytes:         -68 200.6
  mean Δ percent:       -77.07%
  CI95 bytes:           [-120 558.1, -25 236.8]
  CI95 percent:         [-79.25%, -75.10%]
Decision: KEEP vs identity. Inferior to zstd-19 for static use (-77.07% vs -81.06%).
Useful as runtime-dynamic baseline if origin produces zstd live, not relevant for
this filesystem-only static scope. Cite RFC 8878, RFC 9659.

## Exp 0007 — zstd-19 (static)
Status: KEEP
Hypothesis: zstd -19 (btopt strategy) approaches brotli-11 ratio with faster decode.
Cmd:       `RUNS=20 python3 bench/measure.py zstd-19 bench/corpus/assets bench/results/0007/items.json`
Result:
  wire_total:           144 954
  mean Δ bytes:         -72 265.5
  mean Δ percent:       -81.06%
  CI95 bytes:           [-127 995.8, -26 461.4]
  CI95 percent:         [-82.37%, -79.76%]
Decision: KEEP. Loses to brotli-11 by ~2 100 bytes aggregate (-1.5% relative on wire),
but decode is ~half the time on equivalent inputs. Adopt as **second wire encoding**
served to clients that advertise `Accept-Encoding: zstd` (browser support is rolling
out). Cite RFC 9659 (HTTP zstd token); agent §2.3.

## Exp 0008 — brotli q=11 with shared dictionary (RFC 9842)
Status: KEEP
Hypothesis: prior versioned bundle as dictionary, new bundle encoded as `dcb`. Versioned
hashed-filename pairs (`app-3e4f5a6.js` → `app-3e4f5a7.js`,
`main-7f8a3c2.css` → `main-7f8a3c3.css`,
`vendor-a9b8c7d.js` → `vendor-a9b8c7e.js`) match the RFC 9842 same-origin
`match="/app/*/main.js"` deployment pattern.
Cmd:       `DICT=<older> RUNS=8 python3 bench/measure.py brotli-dict-11 bench/corpus/assets <out>` per pair, merged.
Result (3-item subset; subset score uses identity sub-baseline):
  wire_total (3 items):  66 877 (vs 261 905 identity, vs 67 396 non-dict brotli-11)
  vs identity (subset):
    decision=KEEP, mean Δ bytes=-103 388, mean Δ percent=-81.65%
    CI95 bytes=[-215 718, -25 513], CI95 percent=[-82.57%, -80.35%]
  vs non-dict brotli-11 (subset, see results/0008/score_vs_brotli11.json):
    decision=KEEP, mean Δ bytes=-458.7, mean Δ percent=-2.82%
    CI95 bytes=[-608.2, -248.4], CI95 percent=[-3.83%, -1.32%]
Framing verified: results/0008/framed/vendor.dcb header `ff 44 43 42` + 32-byte SHA-256
of dict (`da2e1490b46a4d7e...`). Cite RFC 9842 §2.1 (dcb framing).
Decision: KEEP. Marginal but statistically significant -2.82% over non-dict brotli-11.
Magnitude limited because brotli's RFC 7932 Appendix A 120 KB built-in dictionary
already captures HTML/JS vocabulary, leaving less for a corpus-specific shared dict.
Real-world ratios for hashed-bundle minor revisions are 30-60% (compression-engineer §9.3
"typical wins"); the synthetic corpus pairs here have larger inter-version diffs than a
typical micro-deploy.

## Exp 0009 — zstd -19 with shared dictionary (RFC 9842 dcz)
Status: KEEP — **WINNER for versioned-pair shared-dict encoding**
Hypothesis: zstd has no built-in static dict; a corpus-specific shared dict will help
zstd more than brotli.
Cmd:       `DICT=<older> RUNS=20 python3 bench/measure.py zstd-dict-19 bench/corpus/assets <out>` per pair, merged.
Result (3-item subset):
  wire_total (3 items):  63 715 (vs 261 905 identity, vs 67 370 non-dict zstd-19)
  vs identity (subset):
    decision=KEEP, mean Δ bytes=-104 442, mean Δ percent=-82.01%
    CI95 bytes=[-218 509, -25 301], CI95 percent=[-83.64%, -79.68%]
  vs non-dict zstd-19 (subset, see results/0009/score_vs_zstd19.json):
    decision=KEEP, mean Δ bytes=-1 670.9, mean Δ percent=-6.74%
    CI95 bytes=[-3 449.5, -338.7], CI95 percent=[-7.77%, -4.99%]
Framing verified: results/0009/framed/vendor.dcz header `5e 2a 4d 18 20 00 00 00`
+ 32-byte SHA-256 of dict. Cite RFC 9842 §2.2 (dcz framing); RFC 8878 (zstd dictionary
mode); agent §5.4.2.
Decision: KEEP. Larger wins than brotli-dict (Exp 0008) because zstd has no built-in
static dictionary, so the shared dict is doing more work. **Beats brotli-dict-11
(Exp 0008) by 3 162 bytes aggregate on the 3-item subset (4.7% smaller).** This is
the recommended encoding when both client and CDN support `Accept-Encoding: dcz` and
hashed-bundle versioning is in place.

## Exp 0010 — brotli-8 (asset-class CI-budget intermediate)
Status: KEEP, but DISCARD vs brotli-11 for full static use
Hypothesis: q=8 (semi-dynamic) is a useful intermediate when CI cannot afford q=11 on
every asset class. Specifically: HTML/CSS encode fast at q=11; the slow asset class is
the 256 KB vendor.js. Asset-class-tunable: q=11 for small text/HTML/CSS/SVG, q=8 for
large JS bundles where build budget is tight.
Cmd:       `RUNS=15 python3 bench/measure.py brotli-8 bench/corpus/assets bench/results/0010/items.json`
Result:
  wire_total:           160 327 (vs 142 836 brotli-11; +12.2%)
  mean Δ bytes:         -70 557.9
  mean Δ percent:       -79.59%
  CI95 bytes:           [-124 705.1, -26 100.5]
  CI95 percent:         [-81.74%, -77.65%]
  encode_p50_sum_ms:    85 ms (vs 805 ms for q=11; 9.4× faster)
Decision: KEEP vs identity. **DISCARD as static-asset choice** — under SCOPE.md's
encode_cpu_ms weight=0.0, q=11 dominates. Recorded so future rebuild-budget-constrained
projects (where encode_cpu_ms weight > 0) can use this measurement. q=8 is the
"semi-dynamic" point where CPU and ratio meet. Cite agent §2.2.

---

## Per-asset winner (analytic, derived from Exp 0001-0007 + 0010)

| Item                | Best algo  | Wire bytes | vs raw |
|---------------------|------------|-----------:|-------:|
| about.html          | brotli-11  | 3 026      | -84.7% |
| index.html          | brotli-11  | 2 051      | -87.8% |
| logo.svg            | brotli-11  | 2 607      | -79.5% |
| main-7f8a3c2.css    | brotli-11  | 6 234      | -79.7% |
| main-7f8a3c3.css    | brotli-11  | 6 484      | -79.6% |
| app-3e4f5a6.js      | brotli-11  | 15 381     | -81.2% |
| app-3e4f5a7.js      | brotli-11  | 15 626     | -81.4% |
| vendor-a9b8c7d.js   | brotli-11  | 45 282     | -82.3% |
| vendor-a9b8c7e.js   | brotli-11  | 46 145     | -82.3% |

brotli-11 wins all 9 items individually. No mixed per-class config can beat the
single-algorithm choice. With shared dictionaries (Exp 0009 zstd-dict-19), the
versioned subset drops a further 4.7% on the wire over plain brotli-11 on those
same items. Recommended deployment: `brotli-11` for everything, plus
`zstd-19` second encoding, plus `dcz` precomputed payloads keyed on `Available-Dictionary`
for hashed-bundle versions.

## Min-size cutoff (v2 mandatory)

SCOPE.md does not declare an explicit `min_compress_size`; agent §2.4 default is 1024 B.
Smallest corpus item is `logo.svg` at 12 739 B — well above the 1024 B threshold.
**No corpus item is below the threshold; the cutoff does not gate any item in this
session.** Recorded measurement: brotli-11 reduces logo.svg from 12 739 B to 2 607 B
(-79.5%), so even the smallest item benefits substantially from compression.
nginx production directives below set `*_min_length 1024` per §2.4 default.

---

## DISCARD-BY-PREDICTION entries (no bench slot consumed)

These are antipattern matches per agent §3; not benched. Logged for traceability so
future readers see the alternatives were considered.

### DBP-A — gzip on already-compressed types
Status: DISCARD-BY-PREDICTION
Hypothesis: applying gzip to image/png, image/webp, image/avif, font/woff2, audio/mp4,
application/zip etc. would shrink them further.
Reason:    Antipattern §3 #2. Already-compressed bytes have ~no remaining redundancy;
           gzip framing overhead increases size by ~10-30 bytes per file. Wasted CPU.
Citation:  compression-engineer §3 #2; nginx ngx_http_gzip_module: configure
           `gzip_types` to exclude these MIME types.
Note:      n/a in this corpus (no PNG, JPEG, font, video, archive). Recorded as
           configuration guidance for the emitted nginx config.
No bench slot consumed.

### DBP-B — brotli q=11 on dynamic responses
Status: DISCARD-BY-PREDICTION
Hypothesis: maximum brotli quality on dynamic responses.
Reason:    Antipattern §3 #1 (brotli q=11 ≈ 0.5 MB/s encode → P99 latency disaster on
           dynamic responses). Local calibration confirms: q=11 measured at ~0.9 MB/s
           on this CPU vs zstd-3 at ~28.7 MB/s on the same sample (vendor-a9b8c7d.js,
           256 KB).
Citation:  compression-engineer §3 #1; calibration.json q=11 p50 = 273 ms on 256 KB
           sample.
Note:      target_kind = filesystem-only; no dynamic responses in scope. Recorded so
           the emitted runtime-fallback config uses brotli q=5, not q=11.
No bench slot consumed.

### DBP-C — Compression without `Vary: accept-encoding`
Status: DISCARD-BY-PREDICTION
Hypothesis: skip Vary header to reduce response size.
Reason:    Antipattern §3 #3. Cache poisoning across capable and incapable clients.
           Saving ~30 bytes of header creates a correctness defect.
Citation:  compression-engineer §3 #3; RFC 9111 §4.1; RFC 9110 §12.5.5.
Note:      Recorded as configuration invariant for the emitted nginx and Workers configs.
No bench slot consumed.
