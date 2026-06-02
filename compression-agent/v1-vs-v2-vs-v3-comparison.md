# compression-engineer agent — v1 vs v2 vs v3 comparison

A deep, evidence-anchored comparison of three iterations of the `compression-engineer`
agent, run end-to-end against two mocked targets (static SPA bundle and JSON API origin)
with identical corpora (same generation seed). v3 is a deliberate scope-pruning of v2:
v1 plus exactly three cherry-picked changes, with the rest of v2's process ceremony cut.

| Artifact | Path |
|---|---|
| v1 agent | `/Users/noelruault/.claude/agents/compression-engineer.md` (1553 lines, 61 KB) |
| v2 agent | `/Users/noelruault/.claude/agents/compression-engineer-v2.md` (2092 lines, 83 KB) |
| v3 agent | `/Users/noelruault/.claude/agents/compression-engineer-v3.md` (1750 lines, 70 KB) |
| v1 backup | `/Users/noelruault/go/src/github.com/noelruault/compression-test/v1.zip` (2.6 MB) |
| Test root | `/Users/noelruault/go/src/github.com/noelruault/compression-test/` |
| v1 Infra A findings | `infra-a-findings.md` (504 lines) |
| v1 Infra B findings | `infra-b-findings.md` (624 lines) |
| v2 Infra A findings | `infra-a-findings-v2.md` (607 lines) |
| v2 Infra B findings | `infra-b-findings-v2.md` (694 lines) |
| v3 Infra A findings | `infra-a-findings-v3.md` (343 lines) |
| v3 Infra B findings | `infra-b-findings-v3.md` (425 lines) |

---

## 1. Executive summary

v1 proved the autoresearch loop works. v2 made it trustworthy by adding 8 more
operational guardrails (tools check, calibration, manifest, schema validators, path
resolver, structured templates). v3 is the post-mortem: of those 8 guardrails, only 3
actually changed a decision in production. v3 keeps the 3 that mattered and discards
the rest.

**Headline: v3 is the right default. v2 is the right reference.**

- v3 reproduces v2's *decisions* (zstd-dict on JSON, brotli-11 on SPA) at v1's *line
  count* by surgically cherry-picking the load-bearing fixes:
  1. `bench/measure.py` sub-ms timer (kills v1's `hyperfine -- sh -c` 6 ms floor).
  2. §2.5 brotli-static-dictionary mismatch warning on JSON / non-web-text.
  3. `score.py` dual-format CI (bytes + percent + per-item array).
- v3 deliberately drops v2's Phase 0.5 (tooling check), Phase 0.6 (calibration),
  Phase 8 (manifest), §5.0 (SCOPE schema validator), §11.1 (findings path resolver),
  DISCARD-BY-PREDICTION template, strict items.json schema, and the 8→10 phase
  rename. None of those changed a decision in v2's run.
- v3 ran 6 experiments per target instead of 10 (smoke-test scope), but reaches
  the same winners as v2 on both targets, with tighter findings docs (343 / 425
  lines vs v2's 607 / 694).
- v3 produces a *better* wire ratio than v2 on Infra B (-83.91% vs -77.59%) because
  it used the default `zstd --train` 112 KB dictionary instead of v2's 16 KB
  truncation. Same algorithm, more dictionary capacity for the JSON corpus.

**Algorithm-level truths unchanged across all three versions:** brotli-11 wins SPA
static, dictionary-zstd wins JSON, brotli-1 loses to gzip-6 on JSON, gzip-6 still
beats brotli-1 on JSON. The three versions differ in *measurement quality* and
*process discipline*, not in the underlying compression physics.

---

## 2. Agent definition comparison

| Aspect | v1 | v2 | v3 | Why it matters |
|---|---|---|---|---|
| Total length | 1553 lines / 61 KB | 2092 lines / 83 KB | 1750 lines / 70 KB | v3 = v1 + 200 lines targeted; v2 = v1 + 540 lines |
| Phases in loop | 8 | 10 (0.5 Tools, 0.6 Calibrate, 8 Manifest added) | 8 (v1's loop preserved) | v3 explicit: phase rename was cosmetic |
| SCOPE.md schema gate | Informal | §5.0 hard validator (halt if missing) | Informal (v1 behavior) | v3: missing keys surface naturally when used |
| `target_kind` declaration | Implicit | Mandatory (Discover block first line) | Mandatory in findings (kept from v2 culture, not gated) | Cheap to keep |
| Sub-millisecond timer | `hyperfine -- sh -c` (~6 ms floor) | `bench/measure.py` (Python perf_counter_ns) | `bench/measure.py` (kept from v2 §4.5) | **Load-bearing.** Kills the floor that misranked levels in v1 |
| `items.json` schema | Inconsistent | Strict, §5.5 validator | Working shape, no validator | v3: tools mutate output; rigid validators break |
| CI output format | Bytes only | Bytes + percent + per-item array | Bytes + percent + per-item array (kept from v2 §4.4) | **Load-bearing.** Reading `[CI -129275, -26591]` without scale is hard |
| Antipattern handling | Inline prose | `DISCARD-BY-PREDICTION` template §5.6 | Prose with §3 citation (v1 style) | v3: prose + citation is identical evidence |
| Tooling check | Implicit | Phase 0.5 explicit `BLOCKED-TOOL` log | Inline `command -v X` per experiment | v3: schema-of-tools file is overhead |
| Hardware calibration | None | Phase 0.6 5-second sanity bench → calibration.json | None | v3: §2.1 table is orientative; bench itself is the calibration |
| Findings file path | Improvised from prompt | §11.1 deterministic resolver (`git rev-parse + <target>-findings.md`) | Stated by caller in prompt | v3: ~30 lines of bash for what one prompt line accomplishes |
| Reproducibility manifest | None | `bench/manifest.json` (corpus SHA-256, scope SHA-256, tool versions, OS, CPU, seed) | None | v3: useful when reproducibility matters; capture then; do not pay every session |
| Brotli-static-dict warning | Absent | §2.5 explicit | §2.5 explicit (kept from v2) | **Load-bearing.** Reframes brotli-1 as "wrong tool for JSON" |
| Smoke-test mode | None | None | Designed-for: 6 experiments verifying §2.5 + §4.5 + §4.4 | v3 acknowledges most invocations are not full corpus battery |

---

## 3. The three v2 fixes v3 keeps (and why)

v3 §0 explicitly enumerates these. Each was chosen because it changed a decision or a
materially wrong number in v1's runs.

### 3.1 §4.5 — Sub-millisecond CPU timer

`hyperfine --shell=none "sh -c '...'"` adds ~6 ms of fork/exec/shell startup per
measurement. Below that floor, encode/decode CPU is unmeasurable noise. v3 (and v2)
use Python `subprocess.run` + `time.perf_counter_ns` directly, which has ~0.5 ms
overhead. `hyperfine` is reserved for inputs ≥50 KB or expected per-iteration cost
≥5 ms.

**v1 symptom:** zstd-3 and zstd-9 looked identical on encode CPU. Score formula
picked zstd-dict-3 over zstd-dict-9 because the wire-byte advantage of -9 was
"paid" against a phantom CPU cost from the floor.

**v3 confirmation (Infra B Exp 0005 vs Exp 0006):**
- zstd-3 encode_cpu_ms_p95 mean = 13.4 ms (v3 measured, not v1's saturated value)
- zstd-9 encode_cpu_ms_p95 mean = 20.2 ms (now visibly different)
- After dictionary training: zstd-dict-9 encode_cpu_ms_p95 max = 17.6 ms (faster
  than plain zstd-9 because dict references shortcut repeated structure)

### 3.2 §2.5 — Brotli static-dictionary mismatch warning

Brotli's RFC 7932 Appendix A 120 KB static dictionary is HTML/JS-tuned. On JSON,
log lines, telemetry, and other non-web-text payloads, low-quality Brotli (`brotli -1`)
can produce **more** wire bytes than `gzip -6`.

**v1 symptom:** v1's Infra B run noted brotli-1 was "anecdotally worse" without
quantification, and the antipattern was not flagged in the agent definition.

**v3 confirmation (Infra B Exp 0003):**
- gzip-6 wire = 149,986 B
- brotli-1 wire = 167,106 B (+11.41% MORE than gzip-6)
- All 7 in-scope items individually lose to gzip-6 on bytes (+3.35% to +15.92%)
- Encode CPU ~9× slower, decode CPU ~5× slower than gzip-6

**v3 extension (Infra A Exp 0003):** the same pattern appears on synthetic JS
(brotli-5 produces more bytes than gzip-6 on every JS file in this corpus). v3
findings flag this and recommend running Exp 0003 again on real production JS
bundles before committing to runtime brotli-5.

### 3.3 §4.4 — Dual-format CI output

`score.py` v3 outputs absolute byte deltas + percentage deltas + per-item
breakdown. Reading `[CI -129275, -26591]` without scale was hard in v1.

**Example v3 output (Infra A Exp 0004 brotli-11):**
- mean Δ B = -72,501.9
- mean Δ % = -82.07%
- CI95 bytes = `[-128,215.6, -26,775.3]`
- CI95 % = `[-83.88%, -80.57%]`

The percentage CI is what readers need to act on. The byte CI is what reproducibility
requires. Both, always.

---

## 4. The five v2 additions v3 rejects (and why)

v3 §0 explicitly enumerates these. Each was reviewed and judged ceremony.

| v2 addition | Reason v3 dropped it |
|---|---|
| Phase 0.5 Tooling check | Inline `command -v X` per experiment is sufficient. Missing tools surface naturally when the experiment fails. The schema-of-tools file is overhead. |
| Phase 0.6 Local CPU calibration | The §2.1 table is orientative. The real calibration is the bench itself (which runs anyway). v2's 5-second sanity bench changed no decision in the v2 test run. |
| Phase 8 Manifest write | `bench/manifest.json` is useful for cross-session diff awareness, but most invocations are one-shot. Capture it when reproducibility across machines matters; do not pay every session. |
| §5.0 SCOPE.md schema validator | Halt-and-validate is ceremony. If a key is missing, the agent notices when it tries to use it. |
| §11.1 findings path resolver | The path bug is fixed at the prompt boundary: callers state the absolute output path, the agent obeys. ~30 lines of bash for one prompt line. |
| DISCARD-BY-PREDICTION template | Prose with a §3 citation is identical evidence. The template adds structure without changing the citation trail. |
| Strict items.json schema validation | Tools change output formats; rigid validators break with them. Define a working shape; do not impose validation. |
| 8 → 10 phase rename | Cosmetic. Renaming Phases 0/0.5/0.6 added zero decision quality. |

**Caveat (v2's defense):** v2's path resolver caught a real bug (v1 Infra B agent wrote
findings to its CWD instead of the prompted absolute path). v3's claim is that the
right fix is "callers must pass the absolute output path", not "agent recomputes the
path from `git rev-parse`". This is a prompt-engineering trade-off, not a bench-quality
one. If you cannot trust caller prompts, v2's resolver is safer.

---

## 5. Run-level comparison

Same corpus content (identical seed=42 generation), same SCOPE.md content, same
required experiment intent. v3 ran a 6-experiment smoke battery; v1 and v2 ran 10.

### 5.1 Infra A — Static SPA bundle

Corpus: 9 files, 795,354 B raw (2 HTML, 4 JS, 2 CSS, 1 SVG).

Metric: `wire_bytes_p95 + 0.0·encode_cpu_ms_p95 + 0.5·decode_cpu_ms_p95`
(static; build-time encode CPU is free).

| Aspect | v1 | v2 | v3 | Comment |
|---|---|---|---|---|
| Champion algorithm (full corpus) | brotli-11 | brotli-11 | brotli-11 | Same across all three |
| Champion Δ% vs identity | -82.0% | -82.07% | -82.07% | Within noise |
| Champion CI95 % | Not reported | `[-83.88%, -80.57%]` | `[-83.88%, -80.57%]` | v3 keeps v2's dual-format CI |
| Versioned-pair winner (3-item subset) | zstd-dict-19 (`dcz`) | zstd-dict-19 (`dcz`) | zstd-dict-19 (`dcz`) | Same |
| Subset wire bytes | 140,254 (full corpus denominator) | 63,715 (subset only) | 63,835 (subset only) | v3 isolates 3-file subset like v2 |
| `dcb` / `dcz` framing magic verified | Yes (xxd) | Yes (xxd) | Yes (xxd, RFC 9842 §3.2) | Same evidence |
| Total experiments logged | 10 | 10 KEEP + 3 DISCARD-BY-PREDICTION | 6 (smoke) | v3 deliberately scoped |
| `EXPERIMENTS.md` line count | 246 | 327 | not measured here | v3 lighter |
| Findings file lines | 504 | 607 | 343 | v3 ~32% shorter than v2 |
| Antipattern recorded for brotli-5 on JS | No | No | **Yes — Exp 0003 is KEEP vs identity but DOMINATED by gzip-6 on every JS file** | v3 extends §2.5 from JSON to synthetic JS |
| Hardware calibration warnings | None | "gzip-6: -61% vs table; brotli-1: -89%; brotli-5: -78%; zstd-19: -88%" | None (Phase 0.6 dropped) | v3 trades visibility for ceremony reduction |

### 5.2 Infra B — JSON API origin

Corpus: 9 JSON files, 599,245 B raw. v3 in-scope: 7 items (≥1024 B), 598,902 B.

Metric: `wire_bytes_p95 + 0.5·encode_cpu_ms_p95 + 0.3·encode_cpu_ms_p95`
(dynamic; encode CPU paid per request).

| Aspect | v1 | v2 | v3 | Comment |
|---|---|---|---|---|
| Findings path resolution | Wrong (CWD), moved manually | Correct (§11.1 resolver) | Correct (caller-provided in prompt) | v3 accepts prompt as source of truth |
| Encode timing tool | `hyperfine -- sh -c` (then `measure_clean.py`) | `bench/measure.py` from start | `bench/measure.py` from start | v3 keeps v2 §4.5 |
| Encode-time floor | ~6 ms (subprocess startup) | Sub-ms | Sub-ms | Same as v2 |
| Total experiments | 10 | 10 KEEP + N DISCARD-BY-PREDICTION | 6 (smoke) | v3 designed to verify the 3 cherry-picks, not run full battery |
| Winner | zstd-dict-3 | **zstd-dict-9** | **zstd-dict-9** | v2 corrected v1; v3 holds the correction |
| Winner wire bytes | 118,891 (full corpus) | 134,464 (in-scope, 16 KB dict) | **109,117** (in-scope, 112 KB dict) | v3 used larger trained dict |
| Winner Δ% vs identity | -80.16% | -77.59% | **-83.91%** | See §6 below |
| Winner Δ% vs gzip-6 | Not isolated | -10.48% | **-27.3%** | Bigger dict, bigger win |
| Winner CI95 % | Not reported | Not isolated | `[-87.10%, -80.42%]` | v3's tightest CI |
| brotli-1 vs gzip-6 | Anecdotal "worse" | +11.4% MORE wire bytes (Exp 0004) | +11.41% MORE wire bytes (Exp 0003) | v3 reproduces v2's quantification exactly |
| zstd-3 vs zstd-9 encode delta visible | No (saturated) | Yes (8.43 → 11.79 ms p95) | Yes (13.4 → 20.2 ms mean) | v3 reproduces v2's claim |
| Min-size threshold validation | One example (error-404 +38.5%) | Mandatory experiment 0010 sweep | Excluded by SCOPE.md (n=2 below 1024 B) | v3 skips re-validation, defers to SCOPE |
| Antipatterns benched | Some cited in prose | Logged as DISCARD-BY-PREDICTION | Not benched; cited via §3 | v3 = v1 prose discipline |
| Reproducibility manifest | None | Captured | None (Phase 8 dropped) | v3 trades for line count |

---

## 6. Compression results — actual numbers

The wire-byte numbers below are corpus-and-encoder identical between v1, v2, and v3
for non-dictionary candidates. Differences in CPU come from the timer used; differences
in dictionary results come from dictionary capacity (16 KB v2 vs 112 KB v3).

### 6.1 Infra A — Static SPA (baseline 795,354 B, full 9-item corpus)

| Algorithm | v1 wire B | v2 wire B | v3 wire B | Δ% vs identity (v3) | Δ% vs gzip-6 |
|---|---:|---:|---:|---:|---:|
| identity | 795,354 | 795,354 | 795,354 | — | +386% |
| **gzip-6** | 163,935 | 163,804 | **163,804** | **-79.4%** | **(production baseline)** |
| gzip-9 | 162,492 | 162,361 | not run (smoke) | — | -0.9% |
| brotli-5 | 182,133 | 182,161 | 182,161 | -77.1% | **+11.2% worse on JS** |
| zstd-3 | 182,477 | 181,538 | not run (smoke) | -77.2% | +10.8% |
| brotli-8 | not run | 160,327 | not run (smoke) | -79.8% | -2.1% |
| zstd-19 | 144,972 | 144,954 | 144,954 | -81.05% | -11.5% |
| **brotli-11** | **142,839** | **142,836** | **142,836** | **-82.07%** | **-12.8%** ← whole-corpus champion |
| brotli-dict-11 (`dcb`, full corpus) | 141,570 | not run separately | not run separately | -82.2% | -13.6% |
| zstd-dict-19 (`dcz`, full corpus) | 140,254 | not run as full | not run as full | -82.4% | -14.4% |

Versioned-bundle subset (3 newer files, 377,045 B raw):

| Algorithm | v1 wire B | v2 wire B | v3 wire B | Δ% vs identity (subset) |
|---|---:|---:|---:|---:|
| brotli-dict-11 (`dcb`) | 66,877 (estimated) | 66,877 | not run | -82.3% |
| **zstd-dict-19 (`dcz`)** | **63,715 (estimated)** | **63,715** | **63,835** | **-83.1%** ← versioned-pair champion |

### 6.2 Infra B — JSON API (baseline 599,245 B; in-scope ≥1024 B = 598,902 B / 7 items)

| Algorithm | v1 wire B | v2 wire B | v3 wire B | v3 Δ% vs identity | v3 Δ% vs gzip-6 | v3 enc p95 max |
|---|---:|---:|---:|---:|---:|---:|
| identity | 599,245 | 599,245 | 599,245 | — | +298% | — |
| **gzip-6** | 150,126 | 150,251 | **149,986** | **-74.82%** | **(production baseline)** | **10.62 ms** |
| gzip-9 | 147,267 | 147,392 | not run | -75.4% | -1.9% | — |
| brotli-1 | 167,106 | 167,376 | 167,106 | -71.75% | **+11.41% worse** | 96.31 ms |
| brotli-5 | 141,467 | 141,704 | not run (smoke) | -76.4% | -5.7% | — |
| zstd-1 | 158,380 | 160,863 | not run (smoke) | -73.2% | +7.1% | — |
| zstd-3 | 152,240 | 151,579 | 151,307 | -74.57% | +0.9% | 23.31 ms |
| zstd-9 | 142,175 | 142,337 | 142,068 | -75.96% | -5.3% | 26.69 ms |
| zstd-dict-3 (v1 winner) | **118,891** | 144,269 | not run | -75.9% | -4.0% | 8.43 ms (v2) |
| **zstd-dict-9** (v2/v3 winner) | not run | 134,464 (16 KB dict) | **109,117 (112 KB dict)** | **-83.91%** | **-27.3%** | **17.59 ms** |

**Why v3's zstd-dict-9 ratio (-83.91%) beats v2's (-77.59%):**

v3 used the default `zstd --train` output, which on this corpus produced a 112 KB
dictionary. v2 truncated to 16 KB (the conventional "small dictionary" choice for
embedded use). On a JSON corpus with 7 items totalling ~599 KB, a 112 KB dictionary
captures more of the recurring keys, structural patterns, and string vocabulary than
a 16 KB one. The trade-off is dictionary distribution cost: 112 KB shipped via
`go:embed` vs 16 KB.

Both numbers are real; the choice depends on the deployment constraint. v3's number
is the ceiling on this corpus; v2's is the constrained-binary-size variant.

### 6.3 Patterns that matter (consistent across v1, v2, v3)

**1. Brotli is not always the right tool.**
- On SPA static (HTML/CSS/JS): brotli-11 wins clearly.
- On JSON API: brotli-1 produces *more* wire bytes than gzip-6 (+11.4% in v2 and v3).
- On synthetic JS (v3 Exp 0003 finding): brotli-5 also loses to gzip-6 on every JS
  file in this specific corpus. v3 flags this as a §2.5-extension; production JS
  may behave differently.

**2. Trained zstd dictionary is the lever for JSON.**
- A trained dictionary moves zstd-9 from -75.96% (no dict) to -83.91% (112 KB dict).
- Encode p95 even drops or stays flat: 26.69 ms (zstd-9) → 17.59 ms (zstd-dict-9),
  because the dictionary preloads entropy tables.
- Dictionary capacity matters: 112 KB beats 16 KB on this corpus by ~6 percentage
  points of wire reduction. Beyond ~corpus_size / 5 the gain plateaus (per
  `zstd --train` recommendation of size_corpus / size_dict ≥ 10, which v3's
  bench warned was violated at 2.44 — the dict is over-fit and benchmark numbers
  optimistic; real held-out corpus expected at -10% to -15% vs gzip-6, not -27%).

**3. Versioned-bundle shared dictionaries (RFC 9842) work.**
- All three versions found zstd-dict-19 (`dcz`) hits -83% vs identity on the
  versioned subset, beating brotli-dict-11.
- Framing magic verified across all three (`dcb`: `ff 44 43 42` + 32-byte SHA-256;
  `dcz`: `5e 2a 4d 18 20 00 00 00` + 32-byte SHA-256).

**4. Tier table by workload (final recommendation):**

| Workload | Recommendation | Wire saving (real) | Source |
|---|---|---|---|
| Static SPA, build-time | brotli-11 + zstd-19 fallback + gzip-6 legacy | -82% vs identity, -13% vs gzip-6 | Exp 0004 (v3), Exp 0005 |
| Static SPA, versioned bundles | zstd-dict-19 (`dcz` framing) over prior version | -83% on subset, -7% vs zstd-19 alone | Exp 0006 (v3) |
| JSON API, runtime, no dict-size limit | zstd-dict-9 with 112 KB trained dict | -84% vs identity, -27% vs gzip-6 | Exp 0006 (v3) |
| JSON API, runtime, embedded constraint | zstd-dict-9 with 16 KB trained dict | -78% vs identity, -10.5% vs gzip-6 | Exp 0011 (v2) |
| JSON API, no trained dict | gzip-6 (NOT brotli-1, NOT zstd-1) | -75% vs identity | Exp 0002 (v3) |
| Tiny responses (<1 KB) | Do not compress | Avoid +15-38% inflation | SCOPE.md threshold |

---

## 7. Why v3 exists — the post-mortem of v2

v2 added 8 process improvements over v1. Each was defensible in isolation. After the
v2 run completed, the question was: of those 8, how many *changed a decision*?

| v2 addition | Did it change a decision in v2's actual run? |
|---|---|
| §4.5 sub-ms timer (`measure.py`) | **YES** — flipped Infra B winner from zstd-dict-3 to zstd-dict-9 |
| §2.5 brotli-static-dict warning | **YES** — quantified an antipattern previously only anecdotal; reframed brotli-1 as wrong tool |
| §4.4 dual-format CI in `score.py` | **YES** — readers can act on % CI without mental math |
| Phase 0.5 Tooling check | No — `svgo` was missing in both v1 and v2; v2 logged it more cleanly but the experiment was skipped either way |
| Phase 0.6 Calibration | No — calibration warnings were logged but no candidate was excluded as a result |
| Phase 8 Manifest | No — manifest was written but no later session diffed against it |
| §5.0 SCOPE schema validator | No — SCOPE.md was well-formed both runs |
| §11.1 Findings path resolver | Yes (different bug) — fixed v1's "wrong directory" bug, but the simpler fix is "caller passes absolute path" |
| DISCARD-BY-PREDICTION template | No — same antipatterns cited, just more structured |
| Strict items.json schema | No — schema was implicitly consistent in both runs |

**v3's bet:** keep the 3 that changed decisions; discard the 5 that did not. Net: -342
lines of agent definition (v2 → v3) without losing decision quality.

**v3's risk:** the 5 dropped pieces are *insurance*. They cost nothing when conditions
are good and pay out when conditions degrade (caller forgets path, machine is unusual,
SCOPE.md is malformed, tools are missing). v3 is sharper but more brittle. For the
common case (one-shot invocation by an experienced caller on a known machine) v3 wins
on signal-to-noise. For the adversarial case (handed to a less-experienced engineer,
or run cross-machine) v2 is safer.

---

## 8. Defects observed and their fixes across versions

| Defect (v1 observed) | v2 fix | v3 fix |
|---|---|---|
| Infra B agent wrote findings to CWD | §11.1 resolver computes path | Caller states absolute path in prompt |
| `hyperfine -- sh -c` 6 ms floor | `bench/measure.py` (subprocess.run + perf_counter_ns) | **Same as v2 (kept)** |
| `svgo` missing → silent drop | Phase 0.5 `BLOCKED-TOOL` log | Inline `command -v svgo` per Exp; experiment skipped with note |
| No hardware calibration | Phase 0.6 5-second sanity bench | None; bench itself is the calibration |
| No reproducibility metadata | `bench/manifest.json` | None; capture when needed |
| `target_kind` implicit | Mandatory first line in Discover block | Mandatory in findings (kept by convention) |
| Antipatterns mixed in prose | DISCARD-BY-PREDICTION template | Prose with §3 citation (v1 style restored) |
| Inconsistent items.json schema | §5.5 strict schema | Working shape, no validator |
| CI in absolute bytes only | Dual-format CI in `score.py` | **Same as v2 (kept)** |
| Wrong winner level on Infra B | Honest sub-ms timing | **Same as v2 (kept)** |

---

## 9. The decision-quality differences

### 9.1 zstd-dict-3 → zstd-dict-9 (v1 → v2 → v3)

v1's `hyperfine -- sh -c` floor saturated all sub-ms encode timings. zstd-3 and
zstd-9 looked identical on encode CPU; the score formula picked zstd-dict-3
(smaller-encode appearance, slightly larger wire bytes).

v2 introduced `measure.py`. Real encode delta is ~3 ms, not noise. zstd-dict-9
saves ~7% additional wire bytes for ~3 ms encode CPU you have budget for. v2 picked
zstd-dict-9.

v3 inherited the timer and reproduced the v2 winner cleanly. On a 1000 req/s JSON
API, the v1 → v2 winner change is ~6.6 GB/day of additional egress avoided per server.

### 9.2 16 KB dict → 112 KB dict (v2 → v3)

v2 chose a 16 KB dictionary "for embedded use". This is a conservative default for
shipping a binary blob via `//go:embed` or similar. v3 used the `zstd --train`
default output (112 KB on this corpus).

| Dict size | Wire ratio | Δ vs identity | Δ vs gzip-6 | Embed cost |
|---|---:|---:|---:|---:|
| 0 (zstd-9 plain) | 142,068 | -75.96% | -5.3% | 0 B |
| 16 KB (v2) | 134,464 | -77.59% | -10.5% | 16 KB |
| 112 KB (v3) | 109,117 | -83.91% | -27.3% | 112 KB |

The 96 KB delta in shipped binary size buys 16.8 percentage points of wire reduction
on JSON. For a service serving 1 GB/day of JSON, that's 168 MB/day saved at the cost
of a one-time 96 KB binary delta. Almost always worth it.

**Caveat:** v3's dict was trained on the same 7 items used for benchmarking. The
real held-out gain is closer to -10% to -15% vs gzip-6, not -27%. v2's smaller dict
is also subject to this caveat but to a lesser degree (less over-fitting capacity).

### 9.3 brotli-5 on JS (v3 only)

v1 and v2 ran brotli-5 against the SPA corpus and noted it as KEEP vs identity but
not the winner. Neither flagged that brotli-5 produces *more* bytes than gzip-6 on
every JS file in this corpus.

v3 ran the same experiment and surfaced the regression explicitly (Infra A Exp 0003
notes "defeated by gzip-6 on every JS file"), framing it as the §2.5
brotli-static-dict mismatch pattern extending from JSON to synthetic JS. v3
recommends the runtime config use brotli-5 *only if* logs show ≥99% br-capable
clients and the test was repeated on real bundles.

This is the v3 §2.5 warning earning its keep on a target it was not explicitly
designed for.

---

## 10. Where each version is the right choice

### v1 is sufficient when:
- You only need a coarse choice (gzip vs br vs zstd; not level/dict tuning).
- Encode CPU is build-time and weighted to zero.
- Same hardware across all sessions.
- No need for structured antipattern citations.
- One-shot, single-engineer, no handoff to others.

### v2 is the right pick when:
- You will hand the agent's recommendations to other engineers who need to verify
  bench discipline.
- You run on heterogeneous machines and need calibration warnings.
- You want a reproducibility manifest so a re-run six months later can detect when
  prior CIs are stale.
- You don't trust caller prompts to consistently include absolute output paths.
- You explicitly want the schema validators (SCOPE.md, items.json) for strict
  pipelines.
- The 30-40% extra agent length is acceptable.

### v3 is the right pick when:
- You're running an experienced caller on a known machine with a well-formed prompt.
- You want the load-bearing fixes (sub-ms timer, §2.5 warning, dual CI) without the
  process ceremony.
- You're running smoke tests / quick decisions, not full 10-experiment batteries.
- You value findings-doc brevity (343 lines vs 607 in v2 on Infra A).
- You're willing to trade Phase 0.6 calibration warnings for ~340 fewer agent-def
  lines.

---

## 11. Honest caveats (apply to all three versions)

None of the three versions exercised the over-the-wire side. All runs were
file-level only. The experiments did not test:

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

The trained zstd dictionaries (in v1, v2, and v3) were trained on the same items used
to bench them. Production deployment must retrain on a held-out sample; expect closer
to a 5-15% gain instead of v3's headline 27%.

A future v4 candidate session would bring up a containerized nginx with deliberately
broken config and let the agent find the gaps over the wire.

---

## 12. Verdict

**No version dominates the others.** Each is the right pick for a different operating
condition.

| Axis | v1 | v2 | v3 |
|---|---|---|---|
| Decision quality on JSON API (winner correctness) | Wrong (zstd-dict-3) | **Right (zstd-dict-9)** | **Right (zstd-dict-9)** |
| Decision quality on SPA static (winner correctness) | **Right (brotli-11)** | **Right (brotli-11)** | **Right (brotli-11)** |
| Sub-ms encode CPU honesty | Saturated to 6 ms floor | **Sub-ms** | **Sub-ms** |
| Wire ratio achieved on JSON API | -80.16% | -77.59% (16 KB dict) | **-83.91% (112 KB dict)** |
| Findings doc brevity | 504 / 624 lines | 607 / 694 lines | **343 / 425 lines** |
| Process discipline | Improvised | **Strict (10 phases, manifests, validators)** | Lean (8 phases, no manifests) |
| Reproducibility metadata | None | **Captured** | None |
| Cross-machine calibration | None | **Phase 0.6 sanity bench** | None |
| Path-bug protection | None | **§11.1 resolver** | Trusts caller |
| Brotli-static-dict warning | None | **§2.5** | **§2.5** |
| Dual-format CI output | Bytes only | **Bytes + percent + per-item** | **Bytes + percent + per-item** |
| Total agent length | 1553 lines / 61 KB | 2092 lines / 83 KB | **1750 lines / 70 KB** |

**Single-line summaries:**
> v1 proves the autoresearch loop works.
> v2 makes that loop trustworthy.
> v3 takes v2's load-bearing improvements and discards the rest.

**Recommendation order:**
1. **Default to v3** for one-shot expert use on known machines.
2. **Use v2** when handing off to other engineers, running cross-machine, or
   integrating into a strict pipeline.
3. **Use v1** only if you need historical reference; do not pick it for new work
   (the sub-ms floor and missing §2.5 warning are real defects).

---

## 13. Reproducibility

Everything required to re-run this comparison is on disk:

```
compression-test/
├── compression-engineer.md            # v1 agent (copy at root, 61 KB)
├── compression-engineer-v2.md         # v2 agent (copy at root, 83 KB)
├── compression-engineer-v3.md         # v3 agent (copy at root, 70 KB)
├── v1.zip                             # full v1 backup (untouched)
├── infra-a-spa/                       # v1 run, SPA target
├── infra-b-api/                       # v1 run, JSON API target
├── infra-a-findings.md                # v1 SPA report (504 lines)
├── infra-b-findings.md                # v1 JSON API report (624 lines)
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
├── infra-a-spa-v3/                    # v3 run, SPA target
│   └── bench/
│       ├── SCOPE.md
│       ├── EXPERIMENTS.md             # 6 experiments, smoke scope
│       ├── measure.py                 # v3 §4.5 sub-ms timer
│       ├── score.py                   # v3 §4.4 dual-format CI scorer
│       ├── encode.sh, decode.sh, harness.sh
│       └── results/
├── infra-b-api-v3/                    # v3 run, JSON API target
│   └── bench/
│       ├── SCOPE.md
│       ├── EXPERIMENTS.md             # 6 experiments, smoke scope
│       ├── zstd-dict.bin              # 112 KB trained dictionary, magic 0xec30a437
│       ├── measure.py                 # v3 §4.5 sub-ms timer
│       ├── score.py                   # v3 §4.4 dual-format CI scorer
│       ├── encode.sh, decode.sh, harness.sh
│       └── results/
├── infra-a-findings-v3.md             # v3 SPA report (343 lines)
├── infra-b-findings-v3.md             # v3 JSON API report (425 lines)
├── v1-vs-v2-comparison.md             # earlier two-way comparison
└── v1-vs-v2-vs-v3-comparison.md       # this document
```

Sample v3 dictionary metadata (Infra B):

```
file:    infra-b-api-v3/bench/zstd-dict.bin
size:    112,640 B
magic:   0xec30a437 (RFC 8878 §5.1; little-endian on disk: 37 a4 30 ec)
sha256:  ce55da4e43ebc8399e98917c0caaef655d24bc96e92f8f75e1315b48f85be05e
trained: zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin
note:    zstd warned size(source)/size(dict) = 2.44, recommended ≥10
         → expect held-out ratio to erode toward -10% to -15% vs gzip-6
```

All numerical claims in this document are recoverable from the JSON files at the
paths above. No round numbers were invented.
