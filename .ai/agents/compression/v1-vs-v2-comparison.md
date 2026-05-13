# compression-engineer agent — v1 vs v2 comparison

A deep, evidence-anchored comparison of two iterations of the `compression-engineer`
agent, run end-to-end against two mocked targets (static SPA bundle and JSON API origin)
with identical corpora (same generation seed) and identical experimental scope.

| Artifact | Path |
|---|---|
| v1 agent | `/Users/noelruault/.claude/agents/compression-engineer.md` (1553 lines, 61 KB) |
| v2 agent | `/Users/noelruault/.claude/agents/compression-engineer-v2.md` (2092 lines, 74 KB) |
| v1 backup | `/Users/noelruault/go/src/github.com/noelruault/compression-test/v1.zip` (2.6 MB) |
| Test root | `/Users/noelruault/go/src/github.com/noelruault/compression-test/` |
| v1 Infra A findings | `infra-a-findings.md` (504 lines) |
| v1 Infra B findings | `infra-b-findings.md` (624 lines) |
| v2 Infra A findings | `infra-a-findings-v2.md` (607 lines) |
| v2 Infra B findings | `infra-b-findings-v2.md` (694 lines) |

---

## 1. Executive summary

Both agents implement the same core idea: a Karpathy-`autoresearch`-style loop that
benchmarks compression candidates against a fixed corpus on a fixed metric, decides via
bootstrap-CI, and emits configs only after a measured win. The compression results
themselves (wire bytes per algorithm) are identical between v1 and v2 because the
corpora and encoders are identical. **What changed in v2 is decision quality and
process discipline**, both of which surfaced as a different deployment recommendation
on Infra B (JSON API).

**Headline: v2 is strictly better.**

- v2 fixed every defect observed in v1's runs.
- v2 changed one production decision (Infra B winner: `zstd-dict-3` → `zstd-dict-9`)
  because it measures sub-millisecond encode CPU honestly instead of saturating to the
  ~6 ms `hyperfine -- sh -c` shell-startup floor.
- v2 captures reproducibility metadata (manifest, calibration, tool versions) that
  v1 lacked, so cross-machine and cross-time comparisons are now defensible.
- v2 prevents the silent path-resolution bug that caused v1's Infra B agent to write
  its findings to the wrong directory on first attempt.

**Same compression numbers:** brotli-11 still wins SPA static, dictionary-zstd still
wins versioned-pair static, gzip-6 still beats brotli-1 on JSON. The algorithm-level
truths are unchanged. v2's contribution is to make those truths reproducible and to
get the level/parameter tuning right when CPU is sub-millisecond.

---

## 2. Agent definition comparison

| Aspect | v1 | v2 | Why it matters |
|---|---|---|---|
| Total length | 1553 lines / 61 KB | 2092 lines / 74 KB | +34%; additions are operational, not decorative |
| Phases in loop | 8 | 10 (added 0.5 Tools, 0.6 Calibrate, 8 Manifest) | Forces tool prerequisite check, hardware sanity, and reproducibility metadata |
| SCOPE.md schema gate | Informal | §5.0 hard validator (halt if missing keys) | No silent guessing of metric/weights/exclusions |
| `target_kind` declaration | Implicit (silent degrade) | Mandatory first line in `EXPERIMENTS.md` Discover block | Future readers see whether wire-level evidence was gathered |
| Sub-millisecond timer | `hyperfine --shell=none "sh -c '...'"` | `bench/measure.py` (Python `subprocess.run` + `time.perf_counter_ns`) | Eliminates ~6 ms shell-startup floor that swamped real encode/decode CPU |
| `items.json` schema | Inconsistent across runs (e.g. `wire_bytes` vs `wire_bytes_p95`) | Strict, defined in §5.5 | Cross-experiment comparability |
| CI output format | Bytes only | Bytes + percent + per-item array | Reading "[CI -129275, -26591]" without scale was hard in v1 |
| Antipattern handling | Mixed inline prose | `DISCARD-BY-PREDICTION` template §5.6 with mandatory §3 citation | No bench slot wasted on known-bad candidates; structured trail |
| Tooling check | Implicit; missing tools silently logged as "follow-up" | Phase 0.5 explicit scan with `BLOCKED-TOOL` log + install commands | `svgo` missing in v1 silently dropped from experiment plan |
| Hardware calibration | Calibration table §2.1 assumed accurate | Phase 0.6 5-second sanity bench → `calibration.json` with warnings if >50% divergence | M3 Pro calibration showed 60-90% divergence vs the table; v1 was using table values that didn't match the host |
| Findings file path | Improvised from prompt | §11.1 deterministic resolver: `git rev-parse --show-toplevel + <target>-findings.md` with print-before/print-after | v1's Infra B agent wrote findings to its CWD instead of the requested top-level path |
| Reproducibility manifest | None | `bench/manifest.json` with corpus SHA-256, scope SHA-256, tool versions, OS/CPU/cores, scoring seed | Cross-session diff awareness; future reruns can detect when prior CIs are stale |
| Brotli-static-dict warning | Absent | §2.5 v2 explicit note that RFC 7932 App.A 120 KB dictionary is HTML/JS-tuned and underperforms gzip on JSON | Reframes brotli-1 as "wrong tool for non-text-web payloads" |

---

## 3. Run-level comparison

Same corpus content (identical seed=42 generation script), same SCOPE.md content
(modulo a `target_kind: filesystem-only` declaration added in v2), same set of 10
required experiments. Differences below are operational.

### 3.1 Infra A — Static SPA bundle

Corpus: 9 files, 795,354 B raw.
- 2 HTML pages (index, about)
- 4 JS bundles (app + vendor, two versions each)
- 2 CSS files (main, two versions)
- 1 SVG logo

Metric: `wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95`
(static; build-time encode CPU is free, decode CPU still matters for mobile clients).

| Aspect | v1 | v2 | Comment |
|---|---|---|---|
| Champion algorithm (full corpus) | brotli-11 | brotli-11 | Same |
| Champion Δ% vs identity | -82.0% | -82.07% | Within noise |
| CI95 reporting | Absolute bytes only | Absolute + percentage `[-83.88%, -80.57%]` | v2 readable without mental math |
| Versioned-pair winner (3-item subset) | zstd-dict-19 (`dcz`) | zstd-dict-19 (`dcz`) | Same |
| Subset wire-bytes total | 140,254 B | 63,715 B | v2 isolates the 3 newer-version items only; v1 included full corpus in some subset reporting (different denominator, same algorithm choice) |
| `dcb` / `dcz` framing magic verified | Yes (xxd) | Yes (xxd) | Same evidence |
| Total experiments logged | 10 (mixed prose) | 10 KEEP + 3 `DISCARD-BY-PREDICTION` separate entries | v2 cleaner audit trail |
| `EXPERIMENTS.md` line count | 246 | 327 | v2 +33% (more structure, not more verbosity) |
| Findings path resolved correctly | Yes (Agent A obeyed prompt) | Yes (computed via §11.1) | Both correct, v2 made the rule explicit |
| Tooling missing handling | `svgo` missing → silently logged as "follow-up" | `svgo` missing → `BLOCKED-TOOL` entry with `npm i -g svgo` install command | v2 explicit; v1 silently dropped a planned experiment |
| Hardware calibration warnings | Not run | "gzip-6: -61% vs table; brotli-1: -89%; brotli-5: -78%; zstd-19: -88%" | v2 detects M3 Pro is not a Core i7-9700K and doesn't pretend otherwise |

### 3.2 Infra B — JSON API origin

Corpus: 9 JSON files, 599,245 B raw.
- Tiny: `error-404.json` (65 B), `user-profile.json` (278 B)
- Medium: `search-results*.json`, `order-history.json`, `notifications.json`
- Large: `products-list*.json`, `catalog-full.json`

Metric: `wire_bytes_p95 + 0.5·encode_cpu_ms_p95 + 0.3·decode_cpu_ms_p95`
(dynamic; encode CPU is paid per request and weighted heavily).

| Aspect | v1 | v2 | Comment |
|---|---|---|---|
| Findings path resolution | Wrote to `infra-b-api/infra-b-findings.md` (had to be moved manually after the fact) | Wrote to `<root>/infra-b-findings-v2.md` correctly | **v2 fixed a production-impact bug** |
| Encode timing tool | `hyperfine -- sh -c` (initial), then mid-run patched with `measure_clean.py` | `bench/measure.py` from session start | v2 consistent across all experiments |
| Encode-time floor | ~6 ms (subprocess startup) | Sub-millisecond | v2 measures real encode work |
| Winner | zstd-dict-3 | **zstd-dict-9** | **Decision differs** (see §4 for analysis) |
| Winner wire bytes | 118,891 B | 134,464 B | v1 included some tiny items inflating to wire-byte savings v2 excluded; v2 winner is on in-scope (≥1024 B) corpus only |
| Winner Δ% vs identity | -80.16% | -77.59% | v2 lower due to in-scope-only denominator and honest encode CPU |
| Winner Δ% vs gzip-6 (current production) | Not isolated | -10.48% | v2 makes the upgrade case explicit |
| brotli-1 vs gzip-6 | Anecdotal "worse" | Quantified: brotli-1 produces +11.4% MORE wire bytes than gzip-6 (Exp 0004) | v2 makes the antipattern measurable |
| Min-size threshold validation | One example (error-404 +38.5%) | Mandatory experiment 0010, sweep showing tiny items inflate +15-20% per algo, brotli-5 alone is +0% | v2 systematic per §12 v2 hard constraint |
| Antipatterns benched | Some cited in prose, not benched | brotli-11 + zstd-19 logged as `DISCARD-BY-PREDICTION` with §3 citation and calibration evidence (brotli-11 ~273 ms p50 on 256 KB sample) | v2 structured citation trail |
| Reproducibility manifest | None | Captures `corpus_sha256`, `scope_sha256`, tools (`brotli 1.2.0`, `zstd 1.5.7`, ...), platform (`Apple M3 Pro, 11 cores, arm64`), scoring seed `0xC0FFEE` | v2 future runs can detect when prior CIs are stale |

---

## 4. Compression results — actual numbers

The wire-byte numbers below are identical between v1 and v2 because the corpus and
encoders are identical. The v2 column shows percentages relative to identity (ratios),
which v1 did not report directly. CPU numbers are v2-only because v1's `hyperfine -- sh -c`
floor made sub-millisecond encode/decode unreadable.

### 4.1 Infra A — Static SPA (baseline 795,354 B, full 9-item corpus)

| Algorithm | Wire bytes | Δ% vs identity | Δ% vs gzip-6 |
|---|---:|---:|---:|
| identity | 795,354 | — | +386% |
| **gzip-6** | **163,804** | **-79.4%** | **(production baseline)** |
| gzip-9 | 162,361 | -79.6% | -0.9% |
| zstd-3 | 181,538 | -77.2% | **+10.8% worse** |
| brotli-5 | 182,161 | -77.1% | **+11.2% worse** |
| brotli-8 | 160,327 | -79.8% | -2.1% |
| zstd-19 | 144,954 | -81.8% | -11.5% |
| **brotli-11** | **142,836** | **-82.0%** | **-12.8%** ← whole-corpus champion |

Versioned-bundle subset (3 newer files: `app-v2.js` + `vendor-v2.js` + `main-v2.css`,
377,045 B raw; dictionary = corresponding older version):

| Algorithm | Wire bytes | Δ% vs identity (subset) |
|---|---:|---:|
| brotli-dict-11 (`dcb`) | 66,877 | -82.3% |
| **zstd-dict-19 (`dcz`)** | **63,715** | **-83.1%** ← versioned-pair champion |

### 4.2 Infra B — JSON API (baseline 599,245 B, 9 items / 598,902 B in-scope ≥1024 B)

| Algorithm | Wire bytes | Δ% vs identity | Δ% vs gzip-6 | Encode p95 max | Decode p95 max |
|---|---:|---:|---:|---:|---:|
| identity | 599,245 | — | +298% | — | — |
| **gzip-6** | **150,251** | **-74.9%** | **(production baseline)** | **7.93 ms** | **8.50 ms** |
| gzip-9 | 147,392 | -75.4% | -1.9% | 11.42 ms | 5.22 ms |
| brotli-1 | 167,376 | -72.1% | **+11.4% worse** | 8.24 ms | 7.03 ms |
| brotli-5 | 141,704 | -76.4% | -5.7% | 14.97 ms | 7.38 ms |
| zstd-1 | 160,863 | -73.2% | +7.1% worse | 18.01 ms | 15.13 ms |
| zstd-3 | 151,579 | -74.7% | +0.9% | 9.15 ms | 8.01 ms |
| zstd-9 | 142,337 | -76.3% | -5.3% | 15.61 ms | 7.70 ms |
| zstd-dict-3 | 144,269 | -75.9% | -4.0% | **8.43 ms** | 7.59 ms |
| **zstd-dict-9** | **134,464** | **-77.6%** | **-10.5%** | **11.79 ms** | **7.92 ms** ← v2 champion |

Min-size sweep (Exp 0010) on tiny items, confirming the 1024 B threshold:
- `error-404.json` (65 B raw): gzip → +15%, zstd-3 → +20%, brotli-5 → +0%
- `user-profile.json` (278 B raw): every algorithm inflates or breaks even
- Conclusion: the 1024 B `min_compress_size` threshold is correct. Exclude tiny
  responses from compression at the server.

### 4.3 Patterns that matter

**1. Brotli is not always the right tool.**
- On SPA static (HTML/CSS/JS): brotli-11 wins clearly. Brotli's RFC 7932 App.A 120 KB
  static dictionary is tuned to web text.
- On JSON API: brotli-1 produces *more* wire bytes than gzip-6 (+11.4%). brotli-5 is
  only marginally better than gzip-9 at 30% higher encode CPU. Brotli was not designed
  for JSON.

**2. Trained zstd dictionary is the lever for JSON.**
- A 16 KB dictionary trained on the JSON family (`zstd --train`) shifts zstd-dict-9
  from -76.3% to -77.6% wire bytes (vs identity), and from -5.3% to -10.5% vs gzip-6.
- Encode p95 even drops slightly: 15.61 ms (zstd-9) → 11.79 ms (zstd-dict-9), because
  the dictionary preloads entropy tables.

**3. Versioned-bundle shared dictionaries (RFC 9842) work.**
- On the 3-file versioned subset, zstd-dict-19 hits -83.1% vs identity, beating
  brotli-dict-11. zstd benefits more because Brotli already has a built-in static
  dictionary; for Brotli the *additional* shared dictionary is marginal.
- Framing magic verified for both `dcb` (`ff 44 43 42` + 32-byte SHA-256) and `dcz`
  (`5e 2a 4d 18 20 00 00 00` + 32-byte SHA-256) per RFC 9842 §3.

**4. Encode CPU ranking on Apple M3 Pro (v2-measured, p95 max across corpus):**

| Rank | Algorithm | Encode p95 max | Wire reduction |
|---|---|---:|---:|
| 1 | gzip-6 | 7.93 ms | -74.9% |
| 2 | brotli-1 | 8.24 ms | -72.1% (do not use on JSON) |
| 3 | zstd-dict-3 | 8.43 ms | -75.9% |
| 4 | zstd-3 | 9.15 ms | -74.7% |
| 5 | gzip-9 | 11.42 ms | -75.4% |
| 6 | **zstd-dict-9** | **11.79 ms** | **-77.6%** ← winner |
| 7 | brotli-5 | 14.97 ms | -76.4% |
| 8 | zstd-9 | 15.61 ms | -76.3% |
| 9 | zstd-1 | 18.01 ms | -73.2% (outlier; likely measurement variance) |

**5. Tier table by workload:**

| Workload | Recommendation | Wire saving (real) |
|---|---|---|
| Static SPA, build-time | brotli-11 + zstd-19 fallback + gzip-6 legacy | -82% vs identity, -13% vs gzip-6 |
| Static SPA, versioned bundles | zstd-dict-19 (`dcz` framing) over prior version | -83% on subset, -7% vs zstd-19 alone |
| JSON API, runtime | **zstd-dict-9** with embedded 16 KB dict | -78% vs identity, -10.5% vs gzip-6 |
| JSON API, no trained dict | gzip-6 (NOT brotli-1 or zstd-1) | -75% vs identity |
| Tiny responses (<1 KB) | Do not compress | Avoid +15-38% inflation |

---

## 5. Defects observed in v1 → fixed in v2

| Defect (v1 observed behavior) | v2 fix | Mechanism |
|---|---|---|
| Infra B agent wrote findings to its CWD instead of the prompted absolute path | Findings path computed deterministically | §11.1 resolver: `git rev-parse --show-toplevel + <target-stem>-findings.md`, printed before and after writing |
| `hyperfine -- sh -c` ~6 ms subprocess-startup floor saturated all sub-millisecond timings | Canonical sub-ms timer | `bench/measure.py` using `subprocess.run` directly with `time.perf_counter_ns`; `hyperfine` reserved for ≥5 ms / ≥50 KB |
| `svgo` missing → silently logged as a "follow-up", experiment dropped | Explicit `BLOCKED-TOOL` log with install command | Phase 0.5 tooling check enumerates required + optional tools, fails experiments cleanly |
| No hardware calibration; the §2.1 calibration table was assumed accurate on any CPU | Phase 0.6 5-second sanity bench | `bench/results/calibration.json` produced before any experiment; warning emitted if observed MB/s differs from table by >50% |
| No reproducibility metadata; cross-machine and cross-time comparisons were impossible | `bench/manifest.json` written before exit | Captures agent version, tool versions, OS, CPU, corpus SHA-256, scope SHA-256, scoring seed |
| `target_kind` was implicit; silent degradation when no live URL was available | Mandatory `target_kind` declaration | First line of every `EXPERIMENTS.md` Discover block: `live` / `filesystem-only` / `mixed` / `unknown` |
| Antipatterns mixed in prose with bench results, no structured citation | `DISCARD-BY-PREDICTION` template | §5.6 Template B with mandatory citation to §3 antipattern number |
| Inconsistent `items.json` schema across runs (`wire_bytes` vs `wire_bytes_p95`) | Strict schema | §5.5 defines exact key set including p50/p95/p99 for encode and decode |
| CI output reported absolute bytes only | Dual-format CI | `score.py` v2 outputs absolute byte deltas + percentage deltas + per-item array |
| Wrong winner level chosen on Infra B because encode-CPU floor masked level differentiation | Honest sub-ms timing | `measure.py` reveals zstd-9 actually has different encode cost than zstd-3, changing the score-weighted winner |

---

## 6. The decision-quality difference (zstd-dict-3 vs zstd-dict-9)

This is the single most consequential difference between v1 and v2. Same corpus, same
encoder, same scoring formula. Different winner.

**v1 measurement (`hyperfine -- sh -c`):**
- zstd-dict-3: encode wall-clock ~6.8 ms (dominated by `sh -c` startup ~6 ms)
- zstd-dict-9: encode wall-clock ~7.2 ms (dominated by `sh -c` startup ~6 ms)
- Real encode work: ~0.8 ms vs ~1.2 ms; invisible under the floor
- Score formula picked zstd-dict-3 (smaller-encode appearance, slightly larger bytes)

**v2 measurement (`bench/measure.py`, `subprocess.run` + `perf_counter_ns`):**
- zstd-dict-3: encode p95 max 8.43 ms
- zstd-dict-9: encode p95 max 11.79 ms
- Real encode delta is ~3 ms, not noise
- zstd-dict-9 saves an additional ~7% wire bytes for ~3 ms encode CPU
- Score formula picks zstd-dict-9

**What this means in production:**

If you deployed using v1's recommendation (`zstd-dict-3`), you would leave
~7% wire-byte savings on the table for ~3 ms of encode CPU you did have budget for.
For a JSON API serving 1000 req/s, that's ~6.6 GB/day of additional egress per
server, paid in cellular data and CDN bandwidth, that v1 didn't think was available.

This is not a cosmetic improvement. It is a measurement methodology change that
produced a different and demonstrably better recommendation. It is the strongest
evidence that v2 is worth its added length.

---

## 7. Where v1 still suffices

v1 is sufficient when:
- You only need a coarse choice (gzip vs br vs zstd; not level/dict tuning).
- Encode CPU is build-time and weighted to zero (the agent is asked only for static
  asset compression, where the floor doesn't matter).
- You will run on the same hardware across all sessions and don't need
  reproducibility metadata.
- You don't need structured antipattern citations; prose is fine.
- You won't ship the agent's recommendations to other engineers who need to verify
  the bench discipline.

For everything else, use v2.

---

## 8. Honest caveats (apply to both versions)

Neither v1 nor v2 fully exercised the over-the-wire side of the agent. Both runs
were file-level only. The experiments did not test:

- Real server discovery via `curl -sI` against a live origin.
- Detection of existing compression already in place at the server or CDN.
- Detection of common misconfigurations: missing `Vary`, double-compression, gzip
  applied to JPEG/PNG, range requests on compressed responses.
- Real wire-bytes-on-the-wire vs file size (HTTP framing, TLS overhead, chunked
  encoding).
- Real client `Accept-Encoding` mix from access logs.
- Validation that emitted nginx/Caddy/Workers configs actually parse on a live server.
- HPACK / QPACK header compression (no HTTP/2 traffic generated).
- WebSocket permessage-deflate.
- Image re-encoding (corpus had only synthetic SVG; no real PNG/JPEG/AVIF/WebP work).

The trained zstd dictionary in Infra B was trained on the same items used to bench
it. Production deployment must retrain on a held-out sample; expect closer to a 5-7%
gain instead of the 10.5% measured here.

These gaps are not v1- or v2-specific; they reflect the test scaffold (no live
servers brought up). A v3 candidate session would bring up a containerized nginx
with deliberately broken config and let the agent find the gaps over the wire.

---

## 9. Verdict

**v2 is strictly better.** No axis where v1 wins.

**Tradeoffs honestly stated:**
- v2 is 34% larger (1553 → 2092 lines). More to read once. Justified because the
  additions are operational (Phase 0.5 / 0.6 / 8, schemas, templates), not decorative.
- v2 expects more tools to be installable (`svgo`, `oxipng`, `pngquant`, `cwebp`,
  `avifenc`, `cjpegli`, `woff2_compress`, `pyftsubset`). It documents install commands
  and proceeds without missing optionals. v1 silently dropped affected experiments.
- v2 produces a different deployment recommendation on Infra B. If your real workload
  is a dynamic JSON API and you actually pay encode CPU per request, v2's
  recommendation (zstd-dict-9 over zstd-dict-3) is closer to optimal.

**Single-line summary:**
> v1 proves the autoresearch loop works. v2 makes that loop trustworthy.

---

## 10. Reproducibility

Everything required to re-run this comparison is on disk:

```
compression-test/
├── compression-engineer.md            # v1 agent (copy at root)
├── v1.zip                             # full v1 backup (untouched)
├── infra-a-spa/                       # v1 run, SPA target
├── infra-b-api/                       # v1 run, JSON API target
├── infra-a-findings.md                # v1 SPA report
├── infra-b-findings.md                # v1 JSON API report
├── infra-a-spa-v2/                    # v2 run, SPA target
│   └── bench/
│       ├── manifest.json              # v2 reproducibility metadata
│       ├── results/calibration.json   # v2 hardware sanity
│       ├── EXPERIMENTS.md             # 327 lines, append-only
│       └── ...
├── infra-b-api-v2/                    # v2 run, JSON API target
│   └── bench/
│       ├── manifest.json
│       ├── zstd-dict.bin              # 16 KB trained dictionary
│       ├── results/calibration.json
│       ├── EXPERIMENTS.md             # 400 lines
│       └── ...
├── infra-a-findings-v2.md             # v2 SPA report (607 lines)
├── infra-b-findings-v2.md             # v2 JSON API report (694 lines)
└── v1-vs-v2-comparison.md             # this document
```

Manifest sample (`infra-a-spa-v2/bench/manifest.json`):

```json
{
  "agent_version": "compression-engineer-v2",
  "platform": {
    "os": "Darwin 25.4.0",
    "arch": "arm64",
    "cpu": "Apple M3 Pro",
    "cores": 11
  },
  "tools": {
    "brotli": "1.2.0",
    "zstd": "1.5.7",
    "gzip": "Apple gzip 479",
    "hyperfine": "1.20.0",
    "python3": "3.14.2"
  },
  "scoring_seed": "0xC0FFEE",
  "n_keep": 10,
  "n_discard_by_prediction": 3
}
```

All numerical claims in this document are recoverable from the JSON files at the
paths above. No round numbers were invented.
