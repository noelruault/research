# 1brc — solve the One Billion Row Challenge as research

## Goal

Produce a Go implementation of the 1BRC (https://github.com/gunnarmorling/1brc) that processes 1,000,000,000 measurement rows in **under 1 second wall clock on this machine**, built the research way: definition first, assembly kernel exploration second, Go techniques recon third, then an autoresearch-style optimization loop. Every claim measured, every number re-derivable, every dead end recorded.

This is a study in the `research` repo: one directory per question, reports numbered oldest-first, each report with a `*-data.txt` companion holding the raw output AND the command that produced it. The study lives at `1brc/` (repo root, sibling of `shapes-image-file-format/`). Code lives under `1brc/code/`.

## The machine (record precisely in the baseline report; these are session-observed, re-verify)

- Apple M5 Pro, `hw.ncpu` 15, 24 GB RAM, macOS Darwin 25.5.0, arm64.
- go1.27.0 darwin/arm64.
- Free disk at setup: ~349 GB.

## Inputs and data — NEVER commit big files

- All generated measurement files live OUTSIDE the repo, in `/Users/noelruault/Downloads/1brc/1brc-assets/`. Never commit any file over 5 MB; never put measurement data under the repo tree.
- Starter material (read-only reference, SOURCE): `/Users/noelruault/Downloads/1brc/1brc/` — a Go module with `main.go` (imports a missing `internal/reader/v99`, treat as skeleton inspiration only), `cmd/synthetic_generator.go`, `cmd/generate.sh`, `cmd/profile.sh`, a Makefile; `/Users/noelruault/Downloads/1brc/1brc-assets/` has `cities/` (10k-cities.csv, world-cities.csv) and `lines.out` (a sample expected output).
- The official challenge generates data with ~413 weather stations (Gaussian around per-station means) plus a 10k-station variant. The `env-data` ticket decides the generator (port the official one or extend the starter's) and records for EVERY generated file: exact command, row count, byte size, sha256 — in the data companion. A number nobody can re-derive is worthless here.
- Standard files: `measurements-1b.txt` (1,000,000,000 rows, ~13-14 GB), `measurements-100m.txt` (dev iteration), `measurements-10m.txt` (correctness gate). Plus 10k-station variants if the definition report says the official eval uses them.

## Green gate — exact commands, trust exit codes

- Docs/report-only groups: nothing to build; commit directly.
- Groups touching `1brc/code/go`: from repo root, ALL must exit 0:
  - `cd 1brc/code/go && test -z "$(gofmt -l .)" && go vet ./... && go build -o bin/1brc . && go test ./...`
  - `bash 1brc/scripts/check-correctness.sh` — runs the binary on `measurements-10m.txt` and byte-compares against the reference output (`go-skeleton` creates both the script and the reference generator; the reference is a trivially-correct implementation, not a fast one).
- Groups touching `1brc/code/gen` (the generator and the reference implementation): `cd 1brc/code/gen && test -z "$(gofmt -l .)" && go vet ./... && go build ./... && go test ./...` must exit 0. Note the `test -z`: `gofmt -l` exits 0 even when it lists unformatted files, so `gofmt -l . && ...` is not a gate.
- Groups touching `1brc/code/asm`: `cd 1brc/code/asm && make test` must exit 0. Every kernel ships with a runnable check that fails if the kernel lies (compare against a scalar reference over random + adversarial inputs).
- Never commit red. Noisy warnings are not failures; exit codes are the truth.

## Method rules (this repo's conventions — binding)

- **Measured / derived / hypothesis.** Label every performance claim. A comparative claim is a hypothesis until benchmarked on THIS machine. Leaderboard timings from the official repo are facts about THEIR machine (32-core EPYC limited to 8 cores); never compare our wall clock against them as if same hardware — record both, compare techniques, not clocks.
- **Headline number = the best measured wall clock under a reproducible, RECORDED storage state.** `02-baseline.md` measured that the official warm-page-cache state is unreachable here (the file is 53.5% of RAM; the head is evicted while the tail is read) and that uncached parallel `read()` (F_NOCACHE, 15 readers) beat the page-cached path — so every published number names its storage state (page-cached / uncached-F_NOCACHE / cold) and its reader configuration. This rule does NOT pre-decide I/O strategy: mmap, plain `read()`, F_NOCACHE, reader counts are ledger hypotheses, each measured end-to-end before any is kept or discarded. Use `hyperfine` with warmup runs; record the full output in the data companion. (`01-definition.md`'s "warm cache" target phrasing predates this rule — correcting it at its site is the first CORRECTIONS.md entry, due with the next docs-touching group.)
- **Correctness before speed.** A variant that fails byte-compare is not a result; it is a bug. The correctness gate runs before any benchmark is recorded.
- **Microbenchmarks rank kernels; only end-to-end wall clock ranks solutions.** The repo's own lesson (a modelled figure once pointed the wrong way: +7.4% modelled vs −9.2% on the real stream) transposes here: a kernel winning an L1-resident GB/s microbench can lose end-to-end when 15 cores contend for memory bandwidth. Nothing is discarded on a microbench alone — a discard requires an end-to-end (or representative-scale) measurement recorded with its baseline.
- **Reports numbered oldest-first** in `1brc/`: `01-definition.md`, `02-baseline.md`, ... each with `NN-*-data.txt`. One paragraph = one source line; no hard-wrapping.
- **Clean-room reading of other solutions.** Read gunnarmorling/1brc top entries, marcelroed/gigatoken, automataIA/1brc-rs, Go blog posts for TECHNIQUES (mmap, SWAR, NEON, custom hash, branchless parse). Reimplement from understanding; never paste code verbatim; note the license of anything studied in `1brc/LICENSES.md` if an implementation is closely derived.
- **Parked ideas**: `1brc/PARKED.md`, format defined by root `/PARKED.md` — ALL seven fields (what / status / why-with-number / depends-on / revive-when / cost-to-revive / where-the-work-is), statuses parked|killed|subsumed|blocked. No possibility is discarded until proven wrong: killed-on-argument names an argument that does not expire; killed-on-numbers records the number AND the baseline it was measured against, because numbers move.
- **Corrections are first-class.** A published number or claim later found wrong is corrected at EVERY site that publishes it, and the falsified claim gets one entry in `1brc/CORRECTIONS.md` (claim, where published, what disproved it, correcting commit). The casualty list is part of the record — see `shapes-image-file-format/06-corrections-and-falsifications.md` for the shape.
- **Benchmark confounds.** Every data companion records power source (`pmset -g batt | head -1`) and load (`uptime`) at measurement time; benchmark runs keep the machine otherwise quiet (other loops paused, this cycle blocked on the benchmark command). Numbers taken on battery are labelled provisional; the final headline must be measured on AC power.
- **Commit subjects are plain English sentences** naming what changed and why. This machine's commit guard BLOCKS `feat:`/`fix:`/type(scope) prefixes — a conventional-commit subject will be rejected at commit time.
- **Cross-check the definition from the source.** `01-definition.md` is written from the official repo (README + rules + eval scripts), fetched and cited — never from memory. Once written, it is the authority for constraints (station name byte limits, temperature range, rounding semantics, output format).

## The staged plan (what the backlog encodes)

1. **Definition** — what 1BRC asks, exactly; official eval method; leaderboard top-10 timings and their techniques; our target restated for this machine.
2. **Data + baseline** — generate the files; measure the physical floor: raw read bandwidth, `wc -l`, a naive Go scan. The floor tells us what <1s demands (≥14 GB/s effective parse rate).
3. **Assembly kernel research** — the problem is tokenization (find `;` and `\n`, parse a fixed-point temp, hash a name). Explore N kernel approaches in parallel on arm64: SWAR on 8-byte words, NEON vector scans, gigatoken-style tokenization, branchless fixed-point parse. Microbenchmark each in GB/s; a measured winner emerges.
4. **Skill distillation** — write the global `performance-assembly` skill (macOS/arm64 assembly performance playbook) from what the kernel work taught. Employer-agnostic, no repo/PR references.
5. **Go recon** — how far raw/unsafe Go goes: mmap, unsafe pointer walks, bounds-check elimination, custom open-addressing hash, per-core sharding, GC off, Go asm (Plan 9 syntax) for the kernels that beat pure Go.
6. **Build + autoresearch** — skeleton with correctness gate, then v1 (parallel I/O + sharding; the I/O strategy starts from 02-baseline's measured winner, mmap kept as a measured alternative, not assumed away), v2 (winning kernels integrated), then optimization rounds run the autoresearch way: hypothesis queue → one experiment per idea → measure → keep/kill → ledger. The adaptation starts from the repo's own prior art — `shapes-image-file-format/AUTORESEARCH.md`, `METHODOLOGY.md`, `07-method-what-worked.md`, `quantization/00-methodology.md` — with https://github.com/karpathy/autoresearch as the secondary source. Before the rounds, a cross-disciplinary transfer pass (`x-transfer`) seeds the queue: the repo's signature move. The experiment ledger is `1brc/07-experiment-ledger.md` (append-only; every row: idea, prediction, measured result, verdict).

## Definition of Done (final-dod checks all of these)

- `01-definition.md` exists, written from cited sources, with the official rules and eval method.
- Correctness gate green: byte-identical output vs reference on 10m AND on the 1b file (run once, record).
- A best Go binary with a measured 1B-row wall clock on this machine, hyperfine-backed, storage state and reader configuration recorded, reported with mean ± stddev.
- Target: **< 1.0 s**. If not reached, the final report states the measured best, the gap, the bottleneck analysis (profile-backed), and PARKED.md carries the untried ideas with revive triggers. An honest miss with the ledger intact is a valid research outcome; an unmeasured claim is not.
- The `performance-assembly` skill exists at `/Users/noelruault/.claude-work-home/.claude/skills/performance-assembly/SKILL.md` (with frontmatter matching the sibling skills' format) and is employer-agnostic.
- Every report has its data companion; every number re-derivable from a recorded command.
- `1brc/README.md` summarizes the study: question, method, headline numbers, report index.
- `1brc/CORRECTIONS.md` exists (empty is a valid state — it means nothing published was later disproved).
- The headline is re-measured in a quiet window before publishing: other loops paused, machine on AC power — if on battery at final-dod time, end the cycle with `pause: needs AC power for the headline re-measure` — power + load recorded in the companion.
- Index rows staged in THIS branch for the merge: the 1brc entry in root `README.md` Records, the 1brc row in root `PARKED.md`'s register index, and a `## 1brc` pointer section in repo `CLAUDE.md`.
- The method retrospective exists (`method-retro`'s `NN-method-what-worked.md`): what the execution taught about METHOD, distilled from lessons, corrections, review repairs and the ledger — the results are not the only deliverable; the transferable method lessons are one too.

## Workflow authorization (explicit opt-in, scoped)

Tickets whose backlog line carries the token FANOUT are genuine design-space searches: N independent attempts judged against each other. For those tickets ONLY, the cycle MAY author and run ONE Workflow to build the attempts in parallel. Afterward the same green gate runs, still one group per cycle, still never commit red; adversarially verify each attempt's measurement before declaring a winner. Every other ticket stays single-threaded (serial-first: parallelism buys wall clock only, and this loop can wait).
