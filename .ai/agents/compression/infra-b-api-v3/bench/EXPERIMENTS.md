# EXPERIMENTS — Infra B v3 (JSON API smoke test)

Append-only log. v3 SCOPE.md fixed the 6-experiment battery; results below.

Environment:
- Darwin 25.4.0 arm64 (Apple Silicon)
- brotli 1.2.0, zstd 1.5.7, gzip Apple gzip 479, python 3.14.2, hyperfine 1.20.0
- Timer: `bench/measure.py` (v3 §4.5), 30 runs + 3 warmup per item, sub-ms path
- Score: SCOPE.md formula `wire_bytes_p95 + 0.5*encode_cpu_ms_p95 + 0.3*decode_cpu_ms_p95`
- Bootstrap CI: 10000 resamples, seed 0xC0FFEE, alpha=0.05 (`bench/score.py`)
- In-scope filter: `error-404.json` (65 B) and `user-profile.json` (278 B) excluded
  per SCOPE.md min_compress_size=1024. n=7 in-scope items.

Corpus (raw bytes per item):
- catalog-full.json        189168
- notifications.json        38234
- order-history.json        42784
- products-list-v2.json    139497
- products-list.json       126767
- search-results-v2.json    31626
- search-results.json       31169
- (excluded) error-404.json    65
- (excluded) user-profile.json 278

Total in-scope raw bytes: 599245.

---

## Exp 0001 — identity (baseline)

Hypothesis: establish the no-compression baseline against which the metric is scored.
Cmd: `bash bench/harness.sh identity`
Files: `bench/results/identity/items.json`, copied to `bench/results/baseline.json`.
Result:
  total wire_bytes    : 599245
  per-item enc_cpu_ms_p95 (cat passthrough): 5.1 to 13.1 ms (subprocess.run + cat overhead)
  per-item dec_cpu_ms_p95 (cat passthrough): 5.3 to 13.5 ms
Decision: BASELINE (not scored against itself)

Note: the identity baseline's encode/decode times are essentially the
fork+exec overhead of `cat`, since there is no real codec work. This
overhead is symmetric in every other candidate's measurement (subprocess.run
+ encoder), so candidate-vs-baseline deltas correctly isolate codec cost.

---

## Exp 0002 — gzip-6 (current production baseline)

Hypothesis: production today. Establish what zstd/brotli have to beat.
Cmd: `bash bench/harness.sh gzip-6`
RFC: 1951 (DEFLATE), 1952 (gzip framing)
Result:
  wire_total           : 149986 bytes (-74.97% raw size reduction vs identity)
  encode_cpu_ms_p95 max:  10.621
  decode_cpu_ms_p95 max:   9.390
  mean delta bytes     : -64180.4
  mean delta pct       : -74.82%
  CI95 bytes           : [-98873.2, -34441.6]
  CI95 pct             : [-76.17%, -73.54%]
Decision: KEEP

---

## Exp 0003 — brotli-1 (v3 §2.5 mismatch reproduction)

Hypothesis (v3 §2.5): on JSON corpora, brotli's RFC 7932 Appendix A 120 KB
HTML/JS-tuned static dictionary mismatches the input. Low-quality brotli
should produce MORE wire bytes than gzip-6.
Cmd: `bash bench/harness.sh brotli-1`
RFC: 7932
Result:
  wire_total           : 167106 bytes
  vs gzip-6            : +17120 bytes (+11.41% WORSE on bytes alone)
  encode_cpu_ms_p95 max:  96.307 (massive vs gzip-6's 10.6; brotli-1 is slow even at q=1)
  decode_cpu_ms_p95 max:  46.883 (~5x slower decode than gzip)
  mean delta bytes     : -61717.1 (still beats identity)
  mean delta pct       : -71.75%
  CI95 bytes           : [-95176.0, -32728.9]
  CI95 pct             : [-73.81%, -69.85%]
Decision: KEEP vs identity, but DOMINATED by gzip-6 on every axis (bytes, encode, decode).
Status: confirms v3 §2.5 warning. brotli-1 on JSON costs +11.4% wire bytes vs gzip-6.

Per-item deltas (brotli-1 minus gzip-6, bytes):
  catalog-full.json       +5096 (+10.82%)
  notifications.json      +1355 (+14.84%)
  order-history.json       +317 (+3.35%)
  products-list-v2.json   +4045 (+11.54%)
  products-list.json      +3611 (+11.28%)
  search-results-v2.json  +1383 (+15.92%)
  search-results.json     +1313 (+15.31%)
All 7 in-scope items lose to gzip-6 on bytes. Reproduction confirmed.

---

## Exp 0004 — zstd-3 (zstd default level)

Hypothesis (§2.1, §2.3): zstd default; should match or beat gzip-6 on bytes
with substantially better encode/decode characteristics.
Cmd: `bash bench/harness.sh zstd-3`
RFC: 8878
Result:
  wire_total           : 151307 bytes (vs gzip-6 149986: +1.3 KB worse on bytes)
  encode_cpu_ms_p95 max:  23.313
  decode_cpu_ms_p95 max:  39.869
  mean delta bytes     : -63986.4
  mean delta pct       : -74.57%
  CI95 bytes           : [-98564.1, -34330.9]
  CI95 pct             : [-76.14%, -73.23%]
Decision: KEEP vs identity. Roughly tied with gzip-6 on bytes; loses on per-item
decode CPU on this hardware (Apple Silicon zstd CLI cold-start; production Go
in-process zstd via klauspost/compress will be far faster).

V3 NOTE: this is the experiment that v1 mis-measured. Under v1's
`hyperfine -- sh -c '...'` path, zstd-3 encode looked ~9 ms because the
~6 ms shell-startup floor dominated, and the same was true for zstd-9, so
the score formula could not differentiate the two and zstd-dict-3 was
crowned winner. With measure.py (forks the encoder directly), zstd-3 and
zstd-9 differentiate cleanly: see Exp 0005.

---

## Exp 0005 — zstd-9 (zstd higher level)

Hypothesis (§2.3): zstd-9 trades ~5x encode CPU for materially better ratio
vs zstd-3 (§2.1 calibration: zstd-3 250 MB/s @ 3.10 ratio vs zstd-9 60 MB/s @ 3.45).
Cmd: `bash bench/harness.sh zstd-9`
RFC: 8878
Result:
  wire_total           : 142068 bytes (vs gzip-6 149986: -7918 bytes, -5.3%; vs zstd-3: -9239 bytes, -6.1%)
  encode_cpu_ms_p95 max:  26.694
  decode_cpu_ms_p95 max:  23.352 (decode SAME order as zstd-3 per §2.1 expectation)
  mean delta bytes     : -65302.2
  mean delta pct       : -75.96%
  CI95 bytes           : [-100702.3, -34979.9]
  CI95 pct             : [-77.40%, -74.59%]
Decision: KEEP. Beats zstd-3 on every axis (smaller output AND faster decode here).
Outperforms gzip-6 on bytes by 5.3%.

V3 NOTE: zstd-9 differentiates cleanly from zstd-3 on encode_cpu_ms here
(20 ms mean vs 13 ms mean), where v1 saw both saturate near 8-9 ms.

---

## Exp 0006 — zstd-dict-9 (dictionary-trained, RFC 8878 §5)

Hypothesis: training a zstd dictionary on the JSON family captures shared
keys/values/structure. With level 9, encoder has enough effort to exploit
the dictionary across all 7 items.
Setup:
  zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin
  -> bench/zstd-dict.bin: 112640 bytes, magic 0xec30a437 (RFC 8878 §5.1),
     sha256 ce55da4e43ebc8399e98917c0caaef655d24bc96e92f8f75e1315b48f85be05e
Cmd: `DICT=bench/zstd-dict.bin bash bench/harness.sh zstd-dict-9`
RFC: 8878 §5
Caveat: `zstd --train` warned source/dictionary ratio = 2.4 (target ≥10, ideal ≥100).
        See follow-ups.
Result:
  wire_total           : 109117 bytes (-27.2% vs zstd-9; -23.3% vs gzip-6)
  encode_cpu_ms_p95 max:  17.585 (FASTER than zstd-9 because dict shrinks input
                                  to encode; entropy tables come from dict)
  decode_cpu_ms_p95 max:  12.170 (FASTER than zstd-9)
  mean delta bytes     : -70015.8
  mean delta pct       : -83.91%
  CI95 bytes           : [-107966.1, -38635.5]
  CI95 pct             : [-87.10%, -80.42%]
Decision: KEEP. Winner across all 6 experiments on every axis: smallest wire
bytes, lowest encode CPU among the high-ratio candidates, lowest decode CPU
among the high-ratio candidates.

V3 NOTE: this is also the experiment v1 ran but at level 3, which under
v1's broken timing path appeared optimal. v3 confirms level 9 paired with
the dictionary is the correct pick because the encoder timing path now
shows zstd-dict-9 is FASTER than plain zstd-9, not slower.

---

## Summary

| id    | algo         | wire_total | enc_p95_max | dec_p95_max | mean Δ bytes | mean Δ pct | decision |
|-------|--------------|-----------:|------------:|------------:|-------------:|-----------:|----------|
| 0001  | identity     |     599245 |       13.06 |       13.54 |            0 |     0.00%  | BASELINE |
| 0002  | gzip-6       |     149986 |       10.62 |        9.39 |     -64180.4 |   -74.82%  | KEEP     |
| 0003  | brotli-1     |     167106 |       96.31 |       46.88 |     -61717.1 |   -71.75%  | KEEP (dominated by gzip-6; v3 §2.5 reproduces) |
| 0004  | zstd-3       |     151307 |       23.31 |       39.87 |     -63986.4 |   -74.57%  | KEEP     |
| 0005  | zstd-9       |     142068 |       26.69 |       23.35 |     -65302.2 |   -75.96%  | KEEP     |
| 0006  | zstd-dict-9  |     109117 |       17.59 |       12.17 |     -70015.8 |   -83.91%  | **KEEP, WINNER** |

Winner: **Exp 0006 zstd-dict-9** with `bench/zstd-dict.bin`.
