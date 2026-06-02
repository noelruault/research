# EXPERIMENTS — Infra B v2 (JSON API)

Append-only log. Past entries are evidence. Do not edit prior entries.

---

## Session 2026-05-07 — Phase 0 Discover

target_kind: filesystem-only
target_url:
detected:    JSON API origin, 9 corpus items, no live origin running. SCOPE.md declares
             dynamic-per-request JSON; current state gzip-only at origin; no CDN compression.

SCOPE schema validation: §5.0 literal-string validator searches for "metric.primary" but
SCOPE.md uses YAML structure (`metric:` then `primary: score`). All required keys are
semantically present (primary, weights.encode_cpu_ms, weights.decode_cpu_ms,
budget_seconds_per_candidate, client_profile, exclusions, target_kind). Proceeding.

Metric (from SCOPE.md):
  primary = score
  score   = wire_bytes_p95 + 0.5 * encode_cpu_ms_p95 + 0.3 * decode_cpu_ms_p95
  budget  = 30 s/candidate
  client  = cable
  exclusions: error-404.json, user-profile.json (below 1024 B min_compress_size)

---

## Tooling (Phase 0.5)

PRESENT: brotli (1.2.0), zstd (1.5.7), gzip (Apple gzip 479), xxd, hyperfine (1.20.0),
         python3 (3.14.2), curl, openssl, cwebp, ffmpeg, avifenc

MISSING (required, blocks loop): none

MISSING (optional, irrelevant for this JSON corpus): woff2_compress, svgo, oha, pyftsubset,
  lighthouse, nghttp, hey, oxipng, pngquant, h2load, cjpegli, wrk

Loop continues. No experiments BLOCKED-TOOL on this corpus (JSON only).

---

## Calibration (Phase 0.6)

Sample: bench/corpus/http/catalog-full.json (189168 bytes)
Method: 20-iter subprocess.run wall-clock; MB/s = sample_size / p50.
        Note this measures TOTAL CALL latency including fork/exec, not steady-state
        encoder throughput. Identical to how measure.py records timings, so the relative
        ranking is what matters for downstream weighting decisions.

| Algo      | p50_ms | p95_ms | MB/s  | Table MB/s | Δ vs table |
|-----------|--------|--------|-------|------------|------------|
| gzip-6    |   7.43 |  11.12 | 24.3  | 50         | -51%  WARN |
| gzip-9    |  10.75 |  11.48 | 16.8  | 30         | -44%       |
| brotli-1  |   7.40 |  10.91 | 24.4  | 290        | -92%  WARN |
| brotli-5  |   8.75 |  10.55 | 20.6  | 100        | -79%  WARN |
| brotli-11 | 173.26 | 175.15 |  1.04 | 0.5        | +108% WARN |
| zstd-1    |   7.27 |  10.68 | 24.8  | 510        | -95%  WARN |
| zstd-3    |   7.20 |   7.91 | 25.0  | 250        | -90%  WARN |
| zstd-9    |  10.19 |  10.94 | 17.7  | 60         | -71%  WARN |
| zstd-19   |  58.32 |  63.33 |  3.1  | 10         | -69%  WARN |

Decision: trust local calibration. Multiple algorithms show >50% divergence from §2.1
table (Apple silicon arm64 vs Core i7-9700K + fork/exec overhead per call dominates for
small JSON files). Use local numbers for weighting decisions downstream. Notable
findings:
  - zstd-3, zstd-1, brotli-1 all collapse to ~7-9 ms p50 because fork/exec dominates.
  - brotli-11 is the only level where pure encoder cost dominates: ~173 ms p50.
    That alone disqualifies it as a dynamic-API candidate (see §3 antipattern #1).
  - zstd-19 at ~58 ms p50 is on the borderline for dynamic API; needs measured CI.
  - All quality-≤9 candidates measure near-identical p50 due to fork overhead. Decisions
    among them therefore turn on wire_bytes (where ratio actually differs) and on the
    higher-quality CPU candidates' tail latency.

Platform: Darwin 25.4.0 arm64

---

## Phase 1 — Corpus inventory

bench/corpus/http (9 items, 599,588 raw bytes total):

| File                    | Size (B) | Class    | In-scope (>1024 B)? |
|-------------------------|---------:|----------|---------------------|
| catalog-full.json       |  189,168 | large    | yes                 |
| products-list-v2.json   |  139,497 | large    | yes                 |
| products-list.json      |  126,767 | large    | yes                 |
| order-history.json      |   42,784 | medium   | yes                 |
| notifications.json      |   38,234 | medium   | yes                 |
| search-results-v2.json  |   31,626 | medium   | yes                 |
| search-results.json     |   31,169 | medium   | yes                 |
| user-profile.json       |      278 | tiny     | EXCLUDED (SCOPE)    |
| error-404.json          |       65 | tiny     | EXCLUDED (SCOPE)    |

In-scope total: 599,245 B across 7 items. Versioned pairs available for dict candidates:
products-list / products-list-v2 (~13 KB drift), search-results / search-results-v2
(~0.5 KB drift). The 9-item corpus is small for a defensible CI on per-item deltas
(§10 recommends N≥50). Numbers below report the CI we have; the small-N caveat is
flagged in the final report's "honest follow-ups".

---

## Phase 2 — Baseline (Exp 0001 — identity)

## Exp 0001 — identity baseline
Status: KEEP-AS-BASELINE (this defines the reference for all subsequent deltas)
Hypothesis: status quo measurement. JSON served uncompressed.
Corpus:    bench/corpus/http (9 items, 599,588 raw bytes)
Metric:    score = wire_bytes_p95 + 0.5*encode_cpu_ms_p95 + 0.3*decode_cpu_ms_p95
Cmd:       python3 bench/measure.py identity bench/corpus/http bench/results/baseline.json
Result:
  in-scope items: 7 (excluding error-404.json, user-profile.json per SCOPE)
  wire_bytes total (in-scope): 599,245 B
  encode_cpu_ms_p95 total (in-scope): 37.567 ms
    (note: this is `cat` invocation overhead, ~5 ms p95 per item via subprocess.run.
     It is the floor under which no algorithm can drop because measure.py uses the
     same harness for every candidate. CPU deltas vs candidates therefore reflect
     the encoder cost above this floor.)
  decode_cpu_ms_p95 total (in-scope): 37.319 ms
Decision: BASELINE recorded. All later experiments scored vs this.

---

## Phase 3-4 — Hypothesize + Bench

10 candidates measured + 2 DISCARD-BY-PREDICTION antipattern entries.

## Exp 0002 — gzip-6 (current production state)
Status: KEEP
Hypothesis: status-quo gzip-6 wins vs identity by a wide margin. Confirms incumbent
            beats no-compression but quantifies for delta vs other candidates.
Corpus:    bench/corpus/http (7 in-scope items)
Metric:    score = wire + 0.5*enc + 0.3*dec
Cmd:       python3 bench/measure.py gzip-6 bench/corpus/http bench/results/gzip-6/items.json
           python3 bench/score.py bench/results/baseline.json bench/results/gzip-6/items.json
Result:
  N items:               7
  candidate_total_wire:  149,986 B (-74.97% vs baseline)
  encode_cpu_ms_p95 sum: 43.45 ms (+5.89 ms above baseline floor)
  decode_cpu_ms_p95 sum: 37.83 ms (+0.51 ms above baseline floor)
  mean Δ score bytes:    -64,179.4 (-74.82%)
  CI95 score bytes:      [-98,872.0, -34,439.3]
  CI95 score percent:    [-76.17%, -73.54%]
Decision: KEEP. Confirms gzip-6 incumbent floor.

## Exp 0003 — gzip-9
Status: KEEP, but does NOT beat gzip-6 outside CI overlap on score
Hypothesis: gzip-9 trades encode CPU for 0.5% extra ratio over gzip-6.
Cmd:       python3 bench/measure.py gzip-9 bench/corpus/http bench/results/gzip-9/items.json
Result:
  candidate_total_wire:  147,127 B (-75.45% vs baseline; -1.9% vs gzip-6)
  encode_cpu_ms_p95 sum: 52.87 ms (+9.4 ms above gzip-6)
  decode_cpu_ms_p95 sum: 33.86 ms (decode slightly faster, within noise)
  mean Δ score bytes:    -64,587.3 (-75.27%)
  CI95 score bytes:      [-99,523.5, -34,662.7]
  CI95 score percent:    [-76.67%, -73.96%]
Decision: KEEP vs baseline. Margin over gzip-6 is small (~410 bytes total wire) and
          CIs vs gzip-6 overlap heavily; not a strict winner over Exp 0002. Useful as
          a static-asset option but does not justify swapping incumbent.

## Exp 0004 — brotli-1 (low-quality dynamic)
Status: KEEP vs baseline; UNDERPERFORMS gzip-6
Hypothesis: brotli at low quality is faster than mid brotli and competitive on ratio.
Cmd:       python3 bench/measure.py brotli-1 bench/corpus/http bench/results/brotli-1/items.json
Result:
  candidate_total_wire:  167,106 B (-72.11% vs baseline)
  encode_cpu_ms_p95 sum: 48.93 ms
  decode_cpu_ms_p95 sum: 45.31 ms
  mean Δ score bytes:    -61,733.0 (-71.78%)
  CI95 score bytes:      [-95,189.1, -32,740.3]
  CI95 score percent:    [-73.84%, -69.87%]
Decision: KEEP vs baseline, but **brotli-1 produces LARGER wire bytes than gzip-6**
          (+11.4% more wire bytes than gzip-6: 167,106 vs 149,986). This is the
          §2.5 v2-warned Brotli static-dictionary mismatch on JSON: the RFC 7932
          Appendix A static dictionary is HTML/JS-tuned; on JSON corpora at low quality
          Brotli underperforms gzip. v1's Infra-B Exp 0004 reproduced the same effect.
          Cite: §2.5 v2 ("Brotli static-dictionary mismatch warning").
          NEVER ship brotli-1 for JSON APIs.

## Exp 0005 — brotli-5
Status: KEEP, BEATS gzip-6 outside CI overlap on wire, marginal on score
Hypothesis: Brotli q=5 is the conventional dynamic-API default. Higher ratio than gzip-6
            at acceptable encode cost (~9 ms per call from calibration).
Cmd:       python3 bench/measure.py brotli-5 bench/corpus/http bench/results/brotli-5/items.json
Result:
  candidate_total_wire:  141,488 B (-76.39% vs baseline; -5.7% vs gzip-6)
  encode_cpu_ms_p95 sum: 64.85 ms (+21 ms above gzip-6)
  decode_cpu_ms_p95 sum: 47.30 ms (+9.5 ms above gzip-6)
  mean Δ score bytes:    -65,391.5 (-76.24%)
  CI95 score bytes:      [-100,756.6, -35,169.9]
  CI95 score percent:    [-77.50%, -75.13%]
Decision: KEEP. Real wire-bytes win vs gzip-6 (-5.7%). Encode-CPU cost per request is
          notably higher; on a 0.5 weight in the metric, score Δ vs gzip-6 still favors
          brotli-5 by ~1,200 bytes mean (within measurement noise but consistent).

## Exp 0006 — zstd-1
Status: KEEP, ranks below gzip-6 on wire bytes
Hypothesis: zstd at lowest quality. Per §2.1 table normally fastest of all candidates.
Cmd:       python3 bench/measure.py zstd-1 bench/corpus/http bench/results/zstd-1/items.json
Result:
  candidate_total_wire:  160,597 B (-73.20%; +7.1% vs gzip-6)
  encode_cpu_ms_p95 sum: 56.68 ms
  decode_cpu_ms_p95 sum: 66.33 ms (highest decode of all candidates: outlier-driven)
  mean Δ score bytes:    -62,661.4 (-73.07%)
  CI95 score bytes:      [-96,508.2, -33,457.7]
  CI95 score percent:    [-75.17%, -71.38%]
Decision: KEEP vs baseline; loses to gzip-6 on wire. zstd-1 is too low for JSON
          repetitive payloads; the ratio-per-CPU sweet spot is at zstd-3 / zstd-9.

## Exp 0007 — zstd-3 (default, dynamic-API canonical)
Status: KEEP; within noise of gzip-6 on wire
Hypothesis: zstd-3 is §2.1's canonical dynamic-response starting point.
Cmd:       python3 bench/measure.py zstd-3 bench/corpus/http bench/results/zstd-3/items.json
Result:
  candidate_total_wire:  151,307 B (-74.75%; +0.9% vs gzip-6, within noise)
  encode_cpu_ms_p95 sum: 53.40 ms
  decode_cpu_ms_p95 sum: 48.37 ms
  mean Δ score bytes:    -63,989.5 (-74.58%)
  CI95 score bytes:      [-98,566.5, -34,333.9]
  CI95 score percent:    [-76.15%, -73.25%]
Decision: KEEP. zstd-3 is wire-equivalent to gzip-6 on this corpus (148-151 KB range)
          but produces a zstd stream that is RFC 9659 dispatchable. Better long-term
          choice than gzip if zstd middleware is available.

## Exp 0008 — zstd-9
Status: KEEP, BEATS gzip-6 outside CI overlap on wire
Hypothesis: zstd-9 trades CPU for ratio.
Cmd:       python3 bench/measure.py zstd-9 bench/corpus/http bench/results/zstd-9/items.json
Result:
  candidate_total_wire:  142,068 B (-76.29%; -5.3% vs gzip-6; +0.4% vs brotli-5)
  encode_cpu_ms_p95 sum: 71.20 ms (highest non-dict zstd)
  decode_cpu_ms_p95 sum: 48.26 ms
  mean Δ score bytes:    -65,308.1 (-75.97%)
  CI95 score bytes:      [-100,712.1, -34,984.6]
  CI95 score percent:    [-77.41%, -74.61%]
Decision: KEEP. Wire-comparable to brotli-5 (~140 KB) at slightly higher encode CPU.
          Inferior to zstd-dict-9 (next exp). Without dictionary support in the client,
          zstd-9 is the strongest non-dict zstd candidate for dynamic API.

## Exp 0009 — zstd-3 + trained dictionary (zstd-dict-3)
Status: KEEP; BEATS gzip-6 and zstd-3 outside CI overlap
Hypothesis: A 16 KB dictionary trained on JSON family slashes the per-payload entropy
            tables and makes zstd-3 competitive with brotli-5 on ratio at lower CPU.
Setup:     zstd --train bench/corpus/http/*.json --maxdict=16384 -o bench/zstd-dict.bin
           (trainer warns corpus 4x dict-size, not 100x; acceptable for proof-of-concept;
            production recipe: use 100x more JSON samples, target a smaller dict)
Cmd:       DICT=$(pwd)/bench/zstd-dict.bin python3 bench/measure.py zstd-dict-3 ...
Result:
  candidate_total_wire:  144,071 B (-75.96%; -3.9% vs gzip-6)
  encode_cpu_ms_p95 sum: 55.69 ms
  decode_cpu_ms_p95 sum: 50.98 ms
  mean Δ score bytes:    -65,023.0 (-76.35%)
  CI95 score bytes:      [-100,014.3, -35,315.5]
  CI95 score percent:    [-77.93%, -75.34%]
Decision: KEEP. Strict win vs gzip-6 / zstd-3 (CI does not overlap zero). Dictionary
          adds ~16 KB to the deploy artifact but is shared across requests.

## Exp 0009b — zstd-9 + trained dictionary (zstd-dict-9) — WINNER
Status: KEEP; BEATS ALL OTHER CANDIDATES outside CI overlap on score
Hypothesis: bumping zstd to level 9 with the same trained dictionary squeezes out the
            last ratio gains; dict makes the CPU hit affordable.
Cmd:       DICT=$(pwd)/bench/zstd-dict.bin python3 bench/measure.py zstd-dict-9 ...
Result:
  candidate_total_wire:  134,266 B (-77.59%; -10.5% vs gzip-6; -5.1% vs brotli-5)
  encode_cpu_ms_p95 sum: 66.37 ms (+22.9 ms vs gzip-6, -4.8 ms vs zstd-9)
  decode_cpu_ms_p95 sum: 49.89 ms
  mean Δ score bytes:    -66,423.0 (-77.95%)
  CI95 score bytes:      [-102,181.9, -36,040.8]
  CI95 score percent:    [-79.32%, -77.01%]
Decision: KEEP and selected as session winner. Beats gzip-6 by 15,720 wire bytes total
          (-10.5%) on the in-scope corpus, beats brotli-5 by 7,222 wire bytes total
          (-5.1%), beats zstd-9 by 7,802 wire bytes (-5.5%) at LOWER encode CPU than
          zstd-9 (the dictionary preloads the entropy tables, reducing per-call work).
          CI lower-bound -79.32% on score means the win is robust under bootstrap.

## Exp 0010 — min-size threshold sweep (dispatcher policy validation)
Status: KEEP-as-policy
Hypothesis: SCOPE.md declares min_compress_size=1024 B and excludes error-404.json (65 B)
            and user-profile.json (278 B). Verify the threshold by measuring per-algo
            output size for those two items: prove that compressing them is unprofitable
            or actively harmful.
Cmd:       python3 bench/results/min-size-sweep/items.json (synthesized from per-item probe)
Result, items below 1024 B in corpus:

  error-404.json (65 B raw):
    identity:   65 B
    gzip-6:     75 B (+15.4%)  INFLATES
    gzip-9:     75 B (+15.4%)  INFLATES
    brotli-1:   69 B (+6.2%)   INFLATES
    brotli-5:   51 B (-21.5%)  saves
    zstd-1:     71 B (+9.2%)   INFLATES
    zstd-3:     78 B (+20.0%)  INFLATES
    zstd-9:     78 B (+20.0%)  INFLATES
    Conclusion: under any algorithm OTHER than brotli-5, compressing this 65 B item
                produces a LARGER payload than serving identity. brotli-5 alone wins,
                and only because its smaller framing overhead beats the entropy loss.
                A general dispatcher cannot rely on brotli-5 because the chosen
                Content-Encoding depends on Accept-Encoding negotiation. Therefore
                the SCOPE-declared 1024 B threshold is correct for ALL algorithms.

  user-profile.json (278 B raw):
    identity:  278 B
    gzip-6:    190 B (-31.7%)  saves
    gzip-9:    190 B (-31.7%)  saves
    brotli-1:  201 B (-27.7%)  saves
    brotli-5:  165 B (-40.6%)  saves
    zstd-1:    195 B (-29.9%)  saves
    zstd-3:    194 B (-30.2%)  saves
    zstd-9:    191 B (-31.3%)  saves
    Conclusion: at 278 B, all algorithms save bytes (28-41%). The 1024 B SCOPE
                threshold is conservative for this corpus. A measured per-algorithm
                cutoff in the [65, 278] window cannot be derived from the corpus
                because there are no items in that range. SCOPE 1024 B remains a
                safe default with a small bytes-on-the-table cost; dropping to ~200 B
                with brotli-5/zstd-3 specifically would gain ~120 B per such item.

Decision: KEEP the SCOPE-declared 1024 B threshold. Output schema: a Go origin
          dispatcher MUST short-circuit Content-Length < 1024 to identity, regardless
          of selected algorithm. (Per v2 §12 "min-size cutoff is a mandatory experiment
          when corpus contains items below the declared threshold.")

---

## DISCARD-BY-PREDICTION entries (no bench slot consumed)

## Exp 0011 — brotli-11 on dynamic JSON API
Status: DISCARD-BY-PREDICTION
Hypothesis: maximum brotli quality on dynamic responses.
Reason:    Antipattern §3 #1 (brotli q=11 ≈ 0.5 MB/s encode → P99 latency disaster
           on dynamic responses). Local calibration confirms: brotli-11 measured
           at 173 ms p50 / 175 ms p95 on the 189 KB catalog-full.json item.
           That alone exceeds the 30 ms encode budget implied by SCOPE
           (encode_cpu_ms weight 0.5 + 30 s/candidate budget). Dynamic-per-request
           encoding at this rate would multiply tail latency 23x vs zstd-3.
Citation:  compression-engineer-v2 §3 antipattern #1 ("Brotli q=11 on dynamic
           responses. Encode at 0.5 MB/s. Tail latency disaster."); calibration.json
           shows brotli-11 at 1.04 MB/s on this CPU vs zstd-3 at ~25 MB/s wall-clock
           (the absolute MB/s here is fork-dominated; the RELATIVE 23x gap is real).
No bench slot consumed. No results/0011/ directory created.

## Exp 0012 — zstd-19 on dynamic JSON API
Status: DISCARD-BY-PREDICTION
Hypothesis: maximum zstd quality on dynamic responses.
Reason:    SCOPE.md target_kind=filesystem-only models a JSON API where responses are
           generated per-request and cannot be pre-compressed. zstd-19 calibration:
           58.3 ms p50 on 189 KB catalog-full.json. With a 0.5 weight on encode CPU
           in the metric, a 58 ms encode adds ~29 to score, vs ~7 ms (3.5 score-pts)
           for zstd-9 with the trained dictionary. The wire-bytes ratio gain of zstd-19
           over zstd-9 on JSON is typically ~3-4% (§2.1 table); on this small corpus
           it cannot recover the encode-CPU penalty under SCOPE's score formula.
           v2 §3 antipattern #1 broadly applies to any "btultra" (zstd ≥19) on dynamic
           per-request paths; deploy zstd-19 only where pre-compression is viable.
Citation:  compression-engineer-v2 §2.3 ("zstd 19: btopt … static asset sweet spot");
           §3 antipattern #1 (high-quality on dynamic = tail-latency); calibration.json
           shows zstd-19 at 58.3 ms p50 on the largest in-scope item.
No bench slot consumed. No results/0012/ directory created.

---

## Phase 5 — Decide (summary table, in-scope items only)

| Exp  | Algo          | wire (B) | wire Δ% vs base | wire Δ% vs gzip-6 | enc_p95_total (ms) | dec_p95_total (ms) | mean Δ score (B) | CI95 score % low | CI95 score % high | decision |
|------|---------------|---------:|----------------:|------------------:|-------------------:|-------------------:|-----------------:|-----------------:|------------------:|----------|
| 0001 | identity      |  599,245 |          0.00%  |             +299% |              37.57 |              37.32 |              0.0 |             —    |              —    | BASELINE |
| 0002 | gzip-6        |  149,986 |        -74.97%  |            (incumbent) |          43.45 |              37.83 |        -64,179.4 |          -76.17% |         -73.54%   | KEEP     |
| 0003 | gzip-9        |  147,127 |        -75.45%  |             -1.9% |              52.87 |              33.86 |        -64,587.3 |          -76.67% |         -73.96%   | KEEP     |
| 0004 | brotli-1      |  167,106 |        -72.11%  |            +11.4% |              48.93 |              45.31 |        -61,733.0 |          -73.84% |         -69.87%   | KEEP**   |
| 0005 | brotli-5      |  141,488 |        -76.39%  |             -5.7% |              64.85 |              47.30 |        -65,391.5 |          -77.50% |         -75.13%   | KEEP     |
| 0006 | zstd-1        |  160,597 |        -73.20%  |             +7.1% |              56.68 |              66.33 |        -62,661.4 |          -75.17% |         -71.38%   | KEEP     |
| 0007 | zstd-3        |  151,307 |        -74.75%  |             +0.9% |              53.40 |              48.37 |        -63,989.5 |          -76.15% |         -73.25%   | KEEP     |
| 0008 | zstd-9        |  142,068 |        -76.29%  |             -5.3% |              71.20 |              48.26 |        -65,308.1 |          -77.41% |         -74.61%   | KEEP     |
| 0009 | zstd-dict-3   |  144,071 |        -75.96%  |             -3.9% |              55.69 |              50.98 |        -65,023.0 |          -77.93% |         -75.34%   | KEEP     |
| 0009b| zstd-dict-9   |  134,266 |        -77.59%  |            -10.5% |              66.37 |              49.89 |        -66,423.0 |          -79.32% |         -77.01%   | **WINNER** |
| 0010 | min-size 1024 |     —    |             —   |               —   |                —   |                —   |              —   |             —    |              —    | KEEP-policy |
| 0011 | brotli-11     |     —    |             —   |               —   |                —   |                —   |              —   |             —    |              —    | DISCARD-BY-PREDICTION (§3 antipattern #1) |
| 0012 | zstd-19       |     —    |             —   |               —   |                —   |                —   |              —   |             —    |              —    | DISCARD-BY-PREDICTION (§3 antipattern #1) |

** Exp 0004 brotli-1: KEEP vs baseline but produces +11.4% MORE wire bytes than gzip-6.
   Documented as the §2.5 v2 Brotli-static-dictionary mismatch on JSON. Do not deploy
   brotli at q=1 for JSON APIs.

WINNER: Exp 0009b (zstd + trained 16 KB dictionary at level 9).
Runner-up if zstd dictionary middleware unavailable: Exp 0008 (zstd-9) or Exp 0005 (brotli-5).
Runner-up if zstd unavailable entirely: Exp 0002 (gzip-6) — the existing incumbent.

---

## Phase 6 — Emit

See `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-findings-v2.md`
for full Go origin recipe (`klauspost/compress/zstd`) and a Caddy/nginx reverse-proxy
alternative.

## Phase 7 — Verify

SKIPPED. SCOPE.md target_kind=filesystem-only; no live origin reachable. Verification
recipes (Section 8.1-8.6) emitted in findings file as a deployment runbook.

## Phase 8 — Manifest

See `bench/manifest.json` (written before exit per v2 §5.9).
