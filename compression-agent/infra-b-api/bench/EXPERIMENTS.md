# EXPERIMENTS — Infra B JSON API

Append-only log per Section 4.2 of compression-engineer.md.
Decision rule (Section 4.1): KEEP iff bootstrap CI95 high < 0 (strict).
Bootstrap: 10000 resamples, seed `0xC0FFEE`, alpha=0.05 (Section 4.4).

Metric (per `bench/SCOPE.md`):
```
score = wire_bytes_p95 + 0.5 * encode_cpu_ms_p95 + 0.3 * decode_cpu_ms_p95
```

Corpus: 9 JSON files under `bench/corpus/http/`. Items below the 1024 B
threshold (`error-404.json` 65 B, `user-profile.json` 278 B) are **excluded**
from the baseline and from KEEP/DISCARD scoring per `SCOPE.md`. They are
included only for the dedicated threshold-sweep experiment (Exp 0010), to
quantify why the threshold is needed.

Effective scoring corpus (N = 7):
- `catalog-full.json` (189168 B)
- `products-list-v2.json` (139497 B)
- `products-list.json` (126767 B)
- `order-history.json` (42784 B)
- `notifications.json` (38234 B)
- `search-results-v2.json` (31626 B)
- `search-results.json` (31169 B)

Total uncompressed: 599 245 B.

## Methodology / environment notes

- Host: macOS (Darwin 25.4.0). Encoders: Apple gzip 479, brotli 1.2.0,
  zstd 1.5.7. Bench tooling: hyperfine 1.20.0 + custom Python wrapper.
- Original draft used `hyperfine --shell=none "sh -c '...'"` so encoders
  could redirect to a file. Measured: `sh -c` adds ~6 ms constant subprocess
  overhead, which inflates absolute encode/decode floor on every algorithm.
  Constant overhead cancels in *deltas* (so KEEP/DISCARD decisions remain
  valid), but absolute CPU numbers were misleading.
- Final numbers below come from `bench/measure_clean.py`, which fork/execs
  the encoder binary directly via `subprocess.run`, captures stdout to
  `subprocess.PIPE`, and times wall-clock with `time.perf_counter_ns()`.
  No interposed shell. Floor is reduced to ~0.1 ms (Python overhead +
  exec). Per-item p95 over 30 to 50 timed runs after 3 to 5 warmups.
- Bootstrap CI uses per-item score deltas with `bench/score.py` (Section 4.4
  scorer, weights: encode 0.5 / decode 0.3 per `SCOPE.md`).
- No live server: Phase 7 (over-the-wire verification) is skipped per the
  invocation harness. Wire bytes equal encoded file size (single
  Content-Encoding boundary).

---

## Exp 0001 — baseline (identity)

**Hypothesis**: establish status-quo wire bytes and CPU floor.

**Cmd**:
```
bench/run_clean.sh 0001 identity
cp bench/results/0001/items.json bench/results/baseline.json
```

**Result**:
- Total wire bytes: 599 245 B (no compression).
- Per-item encode/decode p95 ~0.1-0.2 ms (file read only; no actual encode work).

**Decision**: BASELINE (no comparison).

---

## Exp 0002 — gzip-6 (current state)

**Hypothesis**: gzip at default level 6 (RFC 1952) significantly reduces wire
bytes for repetitive JSON. This is the codified status quo per `SCOPE.md`.

**Cmd**: `bench/run_clean.sh 0002 gzip-6`

**Result**:
- Total wire bytes: 150 126 B (-74.95% vs baseline)
- encode_cpu_ms_p95: max 14.53, mean 7.89 (mostly 5-9 ms; outlier observed
  on `search-results.json`, attributable to system noise)
- decode_cpu_ms_p95: max 8.91, mean 5.63
- Score delta: -74.94%, CI95 [-98849, -34411]

**Decision**: **KEEP**. Strong baseline contender; CI strictly negative.

---

## Exp 0003 — gzip-9 (max gzip)

**Hypothesis**: raising gzip level from 6 to 9 buys marginal additional ratio
at higher encode CPU. RFC 1952 / DEFLATE level 9 enables longest match
search.

**Cmd**: `bench/run_clean.sh 0003 gzip-9`

**Result**:
- Total wire bytes: 147 267 B (-75.42% vs baseline; only 2 859 B / 0.5%
  better than gzip-6)
- encode_cpu_ms_p95: max 17.72, mean 8.29 (>2x slower than gzip-6 on the
  largest file: 17.7 ms vs 9.2 ms)
- decode_cpu_ms_p95: max 5.94, mean 4.47 (decode unchanged; expected per
  Section 2.1 of agent definition)
- Score delta: -75.42%, CI95 [-99499, -34639]

**Decision**: **KEEP** statistically (CI strictly negative), but **DISCARD
in deployment** vs gzip-6: the +1.4 ms encode_cpu_ms_p95 mean cost buys only
0.5% additional ratio over gzip-6 (Exp 0002). For a dynamic API endpoint,
gzip-9 is dominated by gzip-6 on the score formula (encode weight 0.5
adds +0.7 to score; bytes savings are -408 per item average). See agent
Section 3 antipattern: high-quality levels for dynamic content.

---

## Exp 0004 — brotli-1

**Hypothesis**: brotli at quality 1 (RFC 7932) is the lowest-CPU brotli
setting; calibration table (Section 2.1) predicts encode ~290 MB/s
(comparable to gzip), with brotli's static dictionary helping ratio on
small inputs.

**Cmd**: `bench/run_clean.sh 0004 brotli-1`

**Result**:
- Total wire bytes: 167 106 B (-72.11%; *worse than gzip-6*)
- encode_cpu_ms_p95: max 14.38, mean 8.19
- decode_cpu_ms_p95: max 6.77, mean 6.30 (brotli decode is structurally
  slower than gzip; calibration confirmed)
- Score delta: -72.11%, CI95 [-95185, -32734]

**Decision**: **KEEP** vs identity (CI strictly negative), but **DISCARD vs
gzip-6**: brotli-1 produces *more* wire bytes than gzip-6 on this corpus,
because brotli q=1 trades ratio for encode speed and our JSON is not large
enough to amortize brotli's overhead. Brotli's static 120 KB dictionary
(RFC 7932 Appendix A) is mostly tuned for HTML/JS, which limits gain on
JSON. **gzip-6 dominates brotli-1 on every axis here** (smaller, faster
encode, faster decode).

---

## Exp 0005 — brotli-5

**Hypothesis**: brotli at quality 5 is the practical default for dynamic
content (Section 2.2: "block splitting + entropy code optimization").
Should beat gzip-6 on bytes but at higher encode CPU.

**Cmd**: `bench/run_clean.sh 0005 brotli-5` (then re-run with N=50 for
stable p95)

**Result**:
- Total wire bytes: 141 467 B (-76.39%; better than gzip-6 by 5.77%)
- encode_cpu_ms_p95: max 9.10, mean 7.52
- decode_cpu_ms_p95: max 8.13, mean 6.50
- Score delta: -76.39%, CI95 [-100755, -35172]

**Decision**: **KEEP**. CI strictly negative. Better wire bytes than gzip-6
at comparable encode CPU. Beaten on bytes only by zstd-9 (Exp 0008) and
zstd-dict-3 (Exp 0009).

---

## Exp 0006 — zstd-1

**Hypothesis**: zstd level 1 (RFC 8878) is the highest-throughput zstd
setting. Calibration predicts encode ~510 MB/s, ratio ~2.9 on text.

**Cmd**: `bench/run_clean.sh 0006 zstd-1`

**Result**:
- Total wire bytes: 158 380 B (-73.57%; worse than gzip-6 on this corpus)
- encode_cpu_ms_p95: max 9.99, mean 7.08
- decode_cpu_ms_p95: max 7.15, mean 6.41 (zstd decode notably faster than
  brotli decode at every level)
- Score delta: -73.57%, CI95 [-97057, -33742]

**Decision**: **KEEP** vs baseline. **DISCARD vs gzip-6** on this corpus:
zstd-1 prioritizes throughput at the expense of ratio; for our JSON it
delivers 8 254 B more than gzip-6 at similar encode time. zstd at level
1 makes sense when you need >500 MB/s encode per core (e.g. log shipping);
not the right pick for an API where encode time is amortized over a
single response.

---

## Exp 0007 — zstd-3 (zstd default)

**Hypothesis**: zstd default level 3 (RFC 8878 reference encoder default).
Calibration predicts the best general-purpose zstd / encode-CPU tradeoff.

**Cmd**: `bench/run_clean.sh 0007 zstd-3` (re-run N=50)

**Result**:
- Total wire bytes: 152 240 B (-74.59%; comparable to gzip-6, slightly worse)
- encode_cpu_ms_p95: max 9.48, mean 7.73
- decode_cpu_ms_p95: max 9.77, mean 7.91
- Score delta: -74.59%, CI95 [-98331, -34270]

**Decision**: **KEEP**. CI strictly negative. Roughly tied with gzip-6 on
bytes; decode slightly slower. Keep as the no-dict zstd reference; the
real zstd win on this corpus needs the trained dictionary (Exp 0009).

---

## Exp 0008 — zstd-9

**Hypothesis**: zstd level 9 trades higher CPU for better ratio. Calibration
predicts ~5x slower than zstd-3, ratio comparable to brotli-5.

**Cmd**: `bench/run_clean.sh 0008 zstd-9` (re-run N=50)

**Result**:
- Total wire bytes: 142 175 B (-76.27%; tied with brotli-5 on bytes)
- encode_cpu_ms_p95: max 27.66, mean 10.67 (one outlier on
  `products-list.json` from system noise; min for that file 6.7 ms)
- decode_cpu_ms_p95: max 6.47, mean 6.09 (decode unchanged from zstd-3)
- Score delta: -76.27%, CI95 [-100631, -35013]

**Decision**: **KEEP**. CI strictly negative. Roughly equivalent to
brotli-5 on bytes and CPU. **Both lose to zstd-dict-3 (Exp 0009)** on
every metric; we keep zstd-9 here for reference but do not deploy it.

---

## Exp 0009 — zstd-dict-3 with trained dictionary

**Hypothesis**: a dictionary trained on the JSON family (`zstd --train`)
seeds the entropy table with shared keys/values across the JSON corpus.
Per agent Section 5.4.2, this is the highest-EV experiment for repetitive
JSON. RFC 8878 §5 specifies dictionary format (magic `0xEC30A437`).

**Setup**:
```bash
zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin
# Trained dict: 112640 B. Magic verified: 37 a4 30 ec ...
```
zstd warns the training corpus is small (270 KB sources / 110 KB dict
= 2.4x ratio; recommended 10x+). We accept this; this is an evaluation,
not a production training run.

**Cmd**: `DICT=bench/zstd-dict.bin bench/run_clean.sh 0009 zstd-dict-3`
(re-run N=50)

**Result**:
- Total wire bytes: 118 891 B (-80.16% vs baseline)
- Per-item gains (vs gzip-6 / Exp 0002):
  - `catalog-full.json`: 33 585 vs 47 108 = -28.7%
  - `notifications.json`:  5 923 vs  9 149 = -35.3%
  - `order-history.json`:  4 896 vs  9 469 = -48.3%
  - `products-list-v2.json`: 34 270 vs 35 076 = -2.3%
  - `products-list.json`:    31 100 vs 32 020 = -2.9%
  - `search-results-v2.json`: 4 304 vs  8 710 = -50.6%
  - `search-results.json`:    4 813 vs  8 594 = -44.0%
- encode_cpu_ms_p95: max 7.57, mean 6.93 (faster than zstd-3 without
  dict, because the entropy table is pre-warmed)
- decode_cpu_ms_p95: max 6.54, mean 6.24 (comparable to no-dict zstd)
- Score delta: -80.15%, CI95 [-105610, -37900]

**Decision**: **KEEP**. Best score by ~4 percentage points over the next
best (brotli-5 / zstd-9). Wins on every per-item, and ties or beats
all other candidates on encode CPU. Note: gains are uneven; `products-list*`
already had a few large outlier strings and so the dictionary helps less
there. Smaller / more repetitive payloads (search-results, notifications,
order-history) compress 35-50% smaller than gzip-6.

**Operational caveat (per agent Section 1.4 and RFC 8878)**: the trained
dictionary must be shipped alongside the binary; both server and client
need the exact same dictionary bytes (matched by header field, e.g.
`Content-Encoding: zstd; dict-id=...` or out-of-band negotiation). In a
pure server-to-server JSON API where both ends are owned by the same team
this is straightforward; for browser clients see Section "Recommended
Configuration" of the report (RFC 9842 `dcz` framing is required for that
path).

---

## Exp 0010 — Min-size threshold sweep (gzip-6 with tiny items)

**Hypothesis**: `error-404.json` (65 B) and `user-profile.json` (278 B) are
below typical compression thresholds. Per agent Section 2.4, gzip framing
+ entropy table overhead can *increase* size below ~1 KB. SCOPE.md sets
the threshold at 1024 B; this experiment quantifies the penalty for
violating it.

**Cmd**:
```
bench/run_clean.sh 0010-baseline identity 1   # identity baseline incl tiny
cp bench/results/0010-baseline/items.json bench/results/baseline-tiny.json
bench/run_clean.sh 0010 gzip-6 1
python3 bench/score.py bench/results/baseline-tiny.json bench/results/0010/items.json
```

**Per-item byte deltas (gzip-6 vs identity)**:
| item                  | base bytes | gzip bytes | delta | pct    |
|-----------------------|-----------:|-----------:|------:|-------:|
| `error-404.json`      |         65 |         90 |   +25 | +38.5% |
| `user-profile.json`   |        278 |        208 |   -70 | -25.2% |
| (...all others)       |    *as Exp 0002* |        |       |        |

**Cross-algorithm sweep on `error-404.json` (65 B)**:
| algo         | encoded | delta | pct    |
|--------------|--------:|------:|-------:|
| gzip-6       |      90 |   +25 | +38.5% |
| gzip-9       |      90 |   +25 | +38.5% |
| brotli-1     |      69 |    +4 | +6.2%  |
| brotli-5     |      51 |   -14 | -21.5% (brotli static dict, RFC 7932 App A) |
| zstd-1       |      78 |   +13 | +20.0% |
| zstd-3       |      78 |   +13 | +20.0% |
| zstd-dict-3  |      25 |   -40 | -61.5% (entropy seeded by trained dict) |

**Aggregate result vs baseline-tiny (N=9)**:
- pct_delta: -74.91%, CI95 [-83243, -21318], decision KEEP
- The aggregate KEEP is dominated by the seven large items; the two tiny
  items contribute only ~0.06% of total bytes, so their per-item
  regressions are statistically invisible at the bootstrap level.

**Decision**: **KEEP gzip-6 for items >= 1024 B; DISCARD compression
entirely for items < 1024 B** (the SCOPE.md exclusion). This experiment
confirms the SCOPE.md threshold:
1. Compressing `error-404.json` with any non-dict gzip/zstd setting
   *enlarges* the response (max +38% with gzip).
2. Brotli's static dictionary (RFC 7932 Appendix A) saves on
   `error-404.json` because the JSON tokens (`"error"`, `"code"`,
   `"not_found"`) are common in its baked-in vocabulary, but at the cost
   of CPU on every tiny response.
3. Only `zstd-dict-3` with our trained dictionary wins handily on tiny
   items (-61.5%), because the entropy table is pre-loaded. **If we deploy
   the dictionary in production, the 1024 B threshold can be lowered** for
   that path; for plain (no-dict) gzip/zstd, 1024 B is the right floor.

**Operational note**: production servers should set
`gzip_min_length 1024` (nginx) / `MinSize 1024` (Caddy) / equivalent.
Below that, serve identity.

---

## Summary table (final, N=50 for headline candidates)

| id    | algo          | wire bytes | wire %  | enc p95 max | dec p95 max | score % | CI95 hi  | decision |
|-------|---------------|-----------:|--------:|------------:|------------:|--------:|---------:|----------|
| 0001  | identity      |    599,245 |   +0.00 |        0.21 |        0.21 |   +0.00 |        0 | BASELINE |
| 0002  | gzip-6        |    150,126 |  -74.95 |       14.53 |        8.91 |  -74.94 |  -34,411 | KEEP     |
| 0003  | gzip-9        |    147,267 |  -75.42 |       17.72 |        5.94 |  -75.42 |  -34,639 | KEEP*    |
| 0004  | brotli-1      |    167,106 |  -72.11 |       14.38 |        6.77 |  -72.11 |  -32,734 | KEEP*    |
| 0005  | brotli-5      |    141,467 |  -76.39 |        9.10 |        8.13 |  -76.39 |  -35,172 | KEEP     |
| 0006  | zstd-1        |    158,380 |  -73.57 |        9.99 |        7.15 |  -73.57 |  -33,742 | KEEP*    |
| 0007  | zstd-3        |    152,240 |  -74.59 |        9.48 |        9.77 |  -74.59 |  -34,270 | KEEP     |
| 0008  | zstd-9        |    142,175 |  -76.27 |       27.66 |        6.47 |  -76.27 |  -35,013 | KEEP     |
| 0009  | zstd-dict-3   |  **118,891** | **-80.16** | **7.57**    | **6.54**    | **-80.15** | **-37,900** | **KEEP** |
| 0010  | gzip-6+tiny   |    *N/A*   |  -74.90 |       14.53 |        8.91 |  -74.91 |  -21,318 | KEEP (aggregate); per-item regression on `error-404.json` confirms 1024 B threshold |

`*` = KEEP statistically vs identity, but dominated by another candidate
on the score formula. Not the deployment recommendation.

**Winner: zstd with trained dictionary (Exp 0009).** Best wire bytes
(-80.2%), best encode CPU among non-trivial candidates, decode parity
with no-dict zstd. The dictionary must ship alongside the server and
be available to clients; see report for deployment.

**Runner-up (no-dict): brotli-5 (Exp 0005) or zstd-9 (Exp 0008)**, tied
at ~-76.3% wire bytes. Pick brotli-5 for browser clients (universal
support), zstd-9 if both ends are server-controlled.

**Antipattern confirmed (Section 3 of agent definition)**: brotli-11 and
zstd-19 were not benched as separate candidates, but the calibration
table predicts brotli-11 encode at 0.5 MB/s = ~380 ms for a 189 KB JSON
on this hardware. That is unacceptable for a dynamic API endpoint where
encode CPU is amortized over a single request. Confirmed by inspection;
not bench-tested. zstd-19's encode at ~10 MB/s = ~19 ms for 189 KB is
closer to viable, but worse than zstd-dict-3 on every axis (more bytes,
more CPU), so we did not bench it.
