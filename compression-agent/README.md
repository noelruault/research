# compression-engineer

A Claude Code subagent that picks the right HTTP compression for your stack
by **measuring**, not by opinion. You point it at a corpus, it runs an
autoresearch loop (hypothesize, bench, bootstrap-CI, keep-or-discard, log),
and emits a server config plus build hooks only after a measured win.

This repo is the test harness, the three iterations of the agent definition
(`v1` to `v3`), and the full evidence trail from running each version
end-to-end against two parallel mocked infrastructures. If you want to use
the agent, skip to [Quick start](#quick-start). If you want to know why you
should trust it, read on.

## Headline

On a representative JSON API corpus, v3 picks **zstd level 9 with a 112 KB
trained dictionary**, which produces:

* **-83.91% wire bytes vs identity** (CI95 `[-87.10%, -80.42%]`)
* **-27.3% wire bytes vs gzip-6** (the typical production floor)
* encode p95 max **17.59 ms** on Apple M3 Pro (vs gzip-6 at 10.62 ms)

On a representative static SPA corpus, v3 picks **brotli quality 11**,
which produces:

* **-82.07% wire bytes vs identity** (CI95 `[-83.88%, -80.57%]`)
* **-12.8% wire bytes vs gzip-6**
* plus `dcz` shared-dictionary delta-encoding for adjacent versioned
  bundles (RFC 9842), at **-83.1%** on the versioned subset

Every number above is reproducible from the JSON files in
`infra-a-spa-v3/bench/results/` and `infra-b-api-v3/bench/results/`.

## The idea

The agent operates like Karpathy's
[autoresearch](https://github.com/karpathy/autoresearch) loop, applied to
compression instead of LLM training. One fixed metric per session, one
fixed bench budget per candidate (default 30 s wall clock), tight loop.
Two files own the state: `bench/SCOPE.md` (human-edited, agent reads only)
and `bench/EXPERIMENTS.md` (agent appends, never edits prior entries).

The metric is whatever the SCOPE.md says, but the canonical formula is:

```
score = wire_bytes_p95 + α·encode_cpu_ms_p95 + β·decode_cpu_ms_p95
```

A candidate is **KEPT** iff the bootstrap 95% CI of its score delta vs
baseline is strictly negative. Otherwise it is **DISCARDED**. No exceptions,
no eyeballing, no "looks good enough". The CI is computed by `bench/score.py`
with 10,000 resamples and a deterministic seed.

The agent is language-agnostic: it probes over the wire (`curl`, `oha`,
`hyperfine`, `h2load`) and runs CLI encoders (`brotli`, `zstd`, `gzip`,
`cjpegli`, `avifenc`, `cwebp`, `oxipng`, `pngquant`, `svgo`,
`woff2_compress`, `pyftsubset`). Build/deploy integration is emitted in
the project's existing language: Make, Justfile, npm scripts, Cargo, Maven,
Bazel, whatever you already use.

## Quick start

**Requirements:** [Claude Code](https://claude.com/claude-code), `bash`,
`brotli`, `zstd`, `gzip`, `python3`, `hyperfine`, `xxd`, `openssl`, `curl`.
On macOS:

```bash
# 1. Install Claude Code (if you don't already have it)
curl -fsSL https://claude.ai/install.sh | bash
# 2. Install the encoder/measurement toolchain (~30 s on warm Homebrew)
brew install brotli zstd hyperfine xxd
# 3. Drop the agent definition into your Claude Code agents directory
cp compression-engineer-v3.md ~/.claude/agents/
# 4. Open Claude Code in your repo
claude
```

In Claude Code, ask:

```
Use the compression-engineer-v3 agent to figure out the right body
compression for this service. Corpus is under bench/corpus/. Write findings
to /absolute/path/to/findings.md.
```

The agent will create `bench/SCOPE.md` and `bench/EXPERIMENTS.md`,
populate `bench/encode.sh` / `decode.sh` / `harness.sh` / `measure.py` /
`score.py`, run the experiments, append every result to `EXPERIMENTS.md`,
and emit a findings document at the path you gave it. Each experiment is
budgeted to 30 s; a typical 6-experiment smoke run completes in under
5 minutes on an M-class laptop.

If you want to repeat the bench documented in this repo without an LLM in
the loop:

```bash
# 5. Reproduce the JSON API smoke run (~3 min on M3 Pro)
cd infra-b-api-v3
bash bench/harness.sh identity
bash bench/harness.sh gzip-6
bash bench/harness.sh brotli-1
bash bench/harness.sh zstd-3
bash bench/harness.sh zstd-9
bash bench/harness.sh zstd-dict-9
python3 bench/score.py
```

The output JSON in `bench/results/<algo>/score.json` will match the numbers
in `infra-b-findings-v3.md` to byte-precision.

## How it works

The repo is deliberately kept small. Six files matter inside any
`infra-*/bench/` directory:

* **`SCOPE.md`**: human-edited, agent reads only. Declares the metric, the
  weights, the budget, the in-scope asset classes, the exclusions, and the
  required experiment slate.
* **`EXPERIMENTS.md`**: agent-appended, never edited. One entry per
  hypothesis. Includes hypothesis, command, raw bench output, bootstrap CI,
  and the KEEP-or-DISCARD decision.
* **`encode.sh`** / **`decode.sh`**: per-candidate dispatch. The agent adds
  recipes (`gzip-6`, `brotli-11`, `zstd-dict-9`, etc.) without editing
  existing ones.
* **`measure.py`**: the v3 sub-millisecond CPU timer. `subprocess.run` plus
  `time.perf_counter_ns`, no shell. Replaces v1's `hyperfine -- sh -c` path
  for any per-iteration cost below 5 ms.
* **`score.py`**: bootstrap-CI scorer. Reads the per-item `items.json`
  from a candidate run, compares to `bench/results/baseline.json`, emits
  bytes + percent + per-item array CI in `score.json`.
* **`harness.sh`**: the driver. Runs encode and decode under `measure.py`,
  aggregates to per-item p95 JSON, writes results.

The autoresearch loop runs in 8 phases:

```
Phase 0   Discover (target_kind, stack, tools)
Phase 1   Corpus (assemble or sample, freeze SHAs)
Phase 2   Baseline (identity wire bytes + CPU)
Phase 3   Hypothesize (one candidate, one rationale)
Phase 4   Bench (encode + decode + wire test, fixed budget)
Phase 5   Decide (bootstrap CI95; KEEP iff CI95_high < 0)
Phase 6   Emit (config + build hook, only after KEEP)
Phase 7   Verify (over-the-wire smoke; skipped in this repo, no live server)
```

## Project structure

```
compression-engineer.md       v1 agent definition (1553 lines, 61 KB)
compression-engineer-v2.md    v2 agent definition (2092 lines, 83 KB)
compression-engineer-v3.md    v3 agent definition (1750 lines, 70 KB) — recommended
v1.zip                        full v1 backup, untouched

infra-a-spa/                  v1 run, static SPA target
infra-a-spa-v2/               v2 run, static SPA target
infra-a-spa-v3/               v3 run, static SPA target (smoke, 6 experiments)
infra-a-findings.md           v1 SPA report (504 lines)
infra-a-findings-v2.md        v2 SPA report (607 lines)
infra-a-findings-v3.md        v3 SPA report (343 lines)

infra-b-api/                  v1 run, JSON API target
infra-b-api-v2/               v2 run, JSON API target
infra-b-api-v3/               v3 run, JSON API target (smoke, 6 experiments)
infra-b-findings.md           v1 JSON API report (624 lines)
infra-b-findings-v2.md        v2 JSON API report (694 lines)
infra-b-findings-v3.md        v3 JSON API report (425 lines)

v1-vs-v2-comparison.md        original two-way comparison
v1-vs-v2-vs-v3-comparison.md  full three-way comparison (recommended read)
```

Inside each `infra-*-v3/bench/`:

```
SCOPE.md            metric, weights, budget, in-scope, required experiments
EXPERIMENTS.md      append-only log of every hypothesis and result
encode.sh           per-candidate encode dispatch
decode.sh           per-candidate decode dispatch
harness.sh          driver: encode + decode + aggregate
measure.py          v3 §4.5 sub-millisecond timer
score.py            v3 §4.4 bootstrap-CI scorer (bytes + percent + per-item)
zstd-dict.bin       (Infra B only) trained dictionary, 112 KB, magic 0xec30a437
results/baseline.json
results/<algo>/items.json
results/<algo>/score.json
corpus/             frozen test corpus, deterministic seed
```

## How v3 was tested

The repo runs the agent end-to-end against two parallel mocked
infrastructures, each with its own metric weighting:

* **Infra A: static SPA.** 9 files, 795,354 B raw (HTML, CSS, JS, SVG).
  Hashed-filename, long-cache, build-time encode CPU is free, decode CPU
  matters for mobile clients. Metric: `wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95`.
* **Infra B: JSON API.** 9 files, 599,245 B raw (catalogs, search results,
  order history, notifications, etc.). Encode CPU is paid per request and
  weighted heavily. Metric: `wire_bytes_p95 + 0.5·encode_cpu_ms_p95 + 0.3·decode_cpu_ms_p95`.

Both ran v3's 6-experiment smoke battery (identity baseline, gzip-6,
brotli-1 or brotli-5, zstd-3, zstd-9, dictionary-zstd) plus dcz framing
verification on the SPA versioned-bundle subset. The harness produced
per-experiment `items.json` and bootstrap-CI `score.json` artifacts under
`bench/results/`. Findings docs are under `infra-{a,b}-findings-v3.md`.

The full 10-experiment battery (gzip-9, brotli-5, brotli-8, zstd-1,
zstd-19, brotli-dict-11, hybrid, etc.) was run by v1 and v2 and is on disk
under `infra-a-spa{,-v2}/` and `infra-b-api{,-v2}/`. v3 reproduces v2's
*decisions* with the smaller battery; the full sweep numbers are
documented for reference in `v1-vs-v2-vs-v3-comparison.md`.

## Results

### Static SPA (Infra A, baseline 795,354 B raw)

| algorithm | wire bytes | Δ% vs identity | Δ% vs gzip-6 | decision |
|---|---:|---:|---:|---|
| identity | 795,354 | | +386% | baseline |
| gzip-6 | 163,804 | -79.4% | (production floor) | KEEP |
| brotli-5 | 182,161 | -77.1% | **+11.2% worse on JS** | KEEP vs identity, dominated |
| zstd-19 | 144,954 | -81.05% | -11.5% | KEEP |
| **brotli-11** | **142,836** | **-82.07%** | **-12.8%** | **KEEP, winner** |
| zstd-dict-19 (`dcz`, versioned subset) | 63,835 / 377,045 | -83.1% on subset | | KEEP, subset winner |

Bootstrap CI95 for the brotli-11 row: bytes `[-128,215.6, -26,775.3]`,
percent `[-83.88%, -80.57%]`. Source:
[`infra-a-findings-v3.md`](infra-a-findings-v3.md).

Side observation worth flagging on this corpus: brotli-5 produces *more*
bytes than gzip-6 on every JS file. This is the v3 §2.5 brotli static-
dictionary mismatch pattern (RFC 7932 Appendix A is HTML/JS-tuned for real
web text, not synthetic JS) showing up on a target it was originally
documented for on JSON. Re-bench on real production bundles before
committing to runtime brotli-5 for JS.

### JSON API (Infra B, baseline 599,245 B raw, 7 in-scope items ≥1024 B)

| algorithm | wire bytes | Δ% vs identity | Δ% vs gzip-6 | encode p95 max | decision |
|---|---:|---:|---:|---:|---|
| identity | 599,245 | | +298% | | baseline |
| gzip-6 | 149,986 | -74.82% | (production floor) | 10.62 ms | KEEP |
| brotli-1 | 167,106 | -71.75% | **+11.41% worse** | 96.31 ms | KEEP vs identity, dominated |
| zstd-3 | 151,307 | -74.57% | +0.9% | 23.31 ms | KEEP |
| zstd-9 | 142,068 | -75.96% | -5.3% | 26.69 ms | KEEP |
| **zstd-dict-9** | **109,117** | **-83.91%** | **-27.3%** | **17.59 ms** | **KEEP, winner** |

Bootstrap CI95 for the zstd-dict-9 row: bytes `[-107,966.1, -38,635.5]`,
percent `[-87.10%, -80.42%]`. Source:
[`infra-b-findings-v3.md`](infra-b-findings-v3.md).

The trained dictionary is at `infra-b-api-v3/bench/zstd-dict.bin`,
112,640 B, magic `0xec30a437` (RFC 8878 §5.1, little-endian on disk
`37 a4 30 ec`), sha256
`ce55da4e43ebc8399e98917c0caaef655d24bc96e92f8f75e1315b48f85be05e`.
Trained with `zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin`.

The findings doc emits a Go origin package using `klauspost/compress/zstd`
with `//go:embed zstd-dict.bin` plus a verify-dict Make target and a
GitHub Actions step. See [`infra-b-findings-v3.md`](infra-b-findings-v3.md)
section "Recommended Go origin code".

### v3 §4.5 confirmation: encode-CPU floor lifted

v1 used `hyperfine --shell=none "sh -c '...'"`, which adds ~6 ms of
fork/exec/shell startup per measurement. Below that floor, encode/decode
CPU is unmeasurable noise. v1 picked **zstd-dict-3** as the JSON API
winner because zstd-3 and zstd-9 looked identical on encode CPU under the
floor.

v3 (and v2) use `bench/measure.py` with `subprocess.run` plus
`time.perf_counter_ns`, no shell, ~0.5 ms overhead. With the floor lifted:

* zstd-3 encode_cpu_ms_p95 mean = 13.4 ms
* zstd-9 encode_cpu_ms_p95 mean = 20.2 ms

Levels are now visibly different, the score formula correctly picks
**zstd-dict-9**, and the production winner moves from -80.16% (v1) to
-83.91% (v3) wire reduction. Same algorithm family, ~7 GB/day of egress
saved on a 1000 req/s JSON API. Full reproduction:
[`v1-vs-v2-vs-v3-comparison.md` §9.1](v1-vs-v2-vs-v3-comparison.md).

## Lineage: v1 to v2 to v3

| | v1 | v2 | v3 (recommended) |
|---|---|---|---|
| Lines | 1553 | 2092 | 1750 |
| Phases | 8 | 10 | 8 |
| Sub-ms timer | hyperfine + sh (floor ~6 ms) | `measure.py` | `measure.py` |
| §2.5 brotli mismatch warning | absent | yes | yes |
| Dual-format CI (bytes + percent) | bytes only | yes | yes |
| Tooling check phase | implicit | Phase 0.5 explicit | inline |
| Hardware calibration phase | none | Phase 0.6 5 s sanity bench | none |
| Reproducibility manifest | none | `bench/manifest.json` | none |
| SCOPE.md schema validator | informal | §5.0 hard validator | informal |
| Findings path resolver | improvised | §11.1 `git rev-parse` resolver | caller-provided |
| DISCARD-BY-PREDICTION template | prose | §5.6 template | prose |
| Strict items.json schema | implicit | §5.5 strict | working shape |

v3 is v1 plus exactly three changes from v2: the sub-ms timer (§4.5), the
brotli static-dict mismatch warning (§2.5), and the dual-format CI scorer
(§4.4). Those three were the only v2 additions that *changed a decision*
in v2's actual run. The rest of v2's process ceremony (Phase 0.5 / 0.6 / 8,
schema validators, path resolver, templates) was reviewed and judged
ceremony, and dropped. Full post-mortem in
[`v1-vs-v2-vs-v3-comparison.md` §7](v1-vs-v2-vs-v3-comparison.md).

If you are handing the agent off to other engineers who need to verify
bench discipline cross-machine, **use v2**. Its calibration phase, manifest,
and schema validators are insurance for the adversarial case. If you trust
the caller and the machine, **use v3**.

## Design choices

* **One fixed metric per session.** The metric is locked in `SCOPE.md` and
  the agent does not negotiate. The upside is that every candidate is
  scored on the same axis. The downside is that you have to know what you
  want to optimize before you start. Pick wire bytes for static
  build-time, balanced for dynamic runtime, latency for mobile.
* **One fixed bench budget per candidate.** Default 30 s wall clock. The
  upside is that the loop terminates predictably. The downside is that
  some candidates need more samples to produce a tight CI than the budget
  allows. Raise the budget in `SCOPE.md` if your CI bytes-range is too
  wide to act on.
* **KEEP iff CI95 high is strictly negative.** No "looks good enough", no
  "trends in the right direction". If the CI touches zero, the candidate
  is DISCARDED. The upside is that decisions survive corpus rotation and
  hardware changes. The downside is that you may discard a real win that
  your corpus is too small to prove. Bigger corpus, tighter CI.
* **Append-only EXPERIMENTS.md.** The agent never edits a prior entry.
  Wrong hypotheses stay in the log with their DISCARD reason. The upside
  is that future sessions can read the full provenance. The downside is
  that the file grows; rotate per major refactor.
* **Two files of state, one of input.** `SCOPE.md` is human-owned input.
  `EXPERIMENTS.md` is agent-owned output. The findings doc is the
  human-facing summary. No other state. The upside is that one git diff
  shows everything the agent did. The downside is that the agent cannot
  carry context across sessions without re-reading the log.
* **Cite every claim by RFC §, vendor doc, or measured experiment id.**
  No claims without one of those three. The upside is that you can audit
  any line. The downside is that the findings docs are dense; skim with
  `grep -n "Exp 0"` to walk the citations.

## Honest caveats

This repo runs the agent file-level only. No live HTTP server is brought
up in any of the six runs (v1, v2, v3 across two infrastructures). Phase 7
(over-the-wire verification) is skipped throughout. The experiments do
**not** test:

* Real server discovery via `curl -sI` against a live origin.
* Detection of compression already in place at the server or CDN.
* Detection of common misconfigurations: missing `Vary`, double-compression,
  gzip applied to JPEG/PNG, range requests on compressed responses.
* Wire-bytes-on-the-wire vs file size (HTTP framing, TLS overhead, chunked
  encoding).
* Real client `Accept-Encoding` mix from access logs.
* Validation that emitted nginx/Caddy/Workers configs actually parse on a
  live server.
* HPACK / QPACK header compression (no HTTP/2 traffic generated).
* WebSocket permessage-deflate.
* Image re-encoding. The corpus has only synthetic SVG; no real
  PNG/JPEG/AVIF/WebP work was done. The agent definition has Phase 5.4.6
  recipes for raster pipelines but they are not exercised here.

The trained zstd dictionary in Infra B was trained on the same 7 items
used to bench it. `zstd --train` warned `size(source)/size(dictionary) =
2.44, recommend ≥10`. The headline -27.3% vs gzip-6 will erode toward
-10% to -15% on a held-out corpus. Re-train on a held-out sample before
declaring numbers to a status page.

The numbers are from Apple M3 Pro arm64. Production targets are typically
x86_64 with the in-process Go encoder, which is materially faster than the
CLI (no fork, no stdio). The ranking of algorithms is stable across
architectures, but absolute milliseconds will differ. Re-run on production
hardware once the service is integrated.

A v4 candidate session would bring up a containerized nginx with
deliberately broken config and let the agent find the gaps over the wire.
I have not done that yet.

## Platform support

The agent runs anywhere Claude Code runs (macOS, Linux, Windows via WSL).
The encoder toolchain is the limiting factor: `brotli`, `zstd`, `gzip`,
`hyperfine`, `xxd`, `openssl`, `python3` need to be on PATH. The reference
configs target nginx, Caddy, Apache, Envoy, HAProxy, Varnish, Cloudflare
Workers, Fastly, CloudFront, and Akamai. The build hooks emit Make,
Justfile, npm scripts, Cargo, Maven, and Bazel snippets. The agent is
language-agnostic by design.

There is no CI or test runner in this repo, by intent: every artifact
under `infra-*/bench/results/` is the test suite. Re-run any
`bench/harness.sh <candidate>` and diff the resulting `score.json` against
the committed one to verify no regression.

## Further reading

* [`v1-vs-v2-vs-v3-comparison.md`](v1-vs-v2-vs-v3-comparison.md): full
  three-way comparison with decision-change post-mortem and dictionary-
  size analysis.
* [`v1-vs-v2-comparison.md`](v1-vs-v2-comparison.md): the earlier
  two-way comparison, kept for historical reference.
* [`infra-a-findings-v3.md`](infra-a-findings-v3.md): v3 SPA report,
  including nginx + ngx_brotli + zstd-nginx-module config and the
  Cloudflare Worker dictionary path.
* [`infra-b-findings-v3.md`](infra-b-findings-v3.md): v3 JSON API report,
  including Go origin code, BREACH per-endpoint audit, and dictionary
  build pipeline.
* [`compression-engineer-v3.md`](compression-engineer-v3.md): the agent
  definition itself. The §0 v3 lineage block enumerates the three
  cherry-picks from v2 and the eight rejections, with rationale.

## License

MIT
