# 1brc-perf — measure the open performance questions while the laptop is idle

## Goal

The 1brc study is closed at **1.233 s** against a 1.000 s target (`1brc/09-result.md`). This loop does not reopen it. It runs the specific measurements that were left owed, on a machine that is **not being used by its operator**, and ledgers what they say. Every ticket here is one experiment or one repair, and the deliverable is a ledger row, not an opinion.

The one thing this loop may change about the shipped binary is a **default**, and only on a disjoint win.

## The rule that gates everything: never measure a machine in use

`E-37` was measured while the operator was working. Its CPU channel was clean (−5.28% against a 0.034% control bracket) and its wall clock was not (−2.28% with ranges overlapping the incumbent's), because a browser at 97% CPU steals cores from a wall clock and cannot touch a process's own user CPU.

So: **every timed run in this loop goes through `bash 1brc/scripts/measure-when-idle.sh`**, never `experiment.sh` directly. It waits for 10 minutes since the last keystroke, a one-minute load under 4.2, and up to 8 hours for both, and it never forces. A cycle that waits all night and measures nothing is a **success**; a cycle that measures a busy machine has produced a row nobody may quote.

Corollary, learned the same way: **do not build, test, or run anything else while a measurement is running.** Rebuilding the binary hyperfine is timing voids the invocation. Do the build and the correctness gate FIRST, then measure, then touch nothing until it returns.

## Green gate — exact commands, trust exit codes

From the repo root, all must exit 0 before any commit:

- `cd 1brc/code/go && test -z "$(gofmt -l .)" && go vet ./... && go build -o bin/1brc . && go test ./...`
- `bash 1brc/scripts/check-correctness.sh` — case zero is upstream's twelve sample pairs, then both 10m cases.
- For a ticket that adds or changes an arm: `ARM="<its flags>" bash 1brc/scripts/check-correctness.sh` too, because an arm is byte-compared before it is ever timed.
- `bash 1brc/scripts/experiment.sh --self-test` when anything under `scripts/` changes.

Docs-only tickets have no build gate.

## Method rules (binding, inherited from the study)

- **A delta is only taken inside one bracketed invocation**, incumbent named FIRST and LAST. Bracket spread over **3%** means no arm may be quoted: re-run, never subtract.
- **Report both channels.** Wall clock and user CPU, each with its own bracket. When they disagree, say so and say which one the confound could touch. A CPU-channel win with no wall win is a real result and is written as one.
- **A win is disjoint or it is a candidate**, not a verdict. A default changes only on a disjoint wall-clock win.
- **Prediction before the run**, containing a number, or labelled `sweep:`.
- **New code needs a differential test against inputs the corpus cannot express** (the `;`-in-name class), and every new test gets mutated to prove it fails.
- Measured / derived / hypothesis on every claim. Numbers from a busy machine are not published.
- Commit subjects are plain-English sentences; the guard rejects `feat:`-style prefixes.
- Append rows to `1brc/07-experiment-ledger.md` with the next free `E-` number. Never renumber.

## Every generalizable finding is fed back to the global skill

A ledger row is this study's memory. The `performance-golang` skill is every *future* project's, and a finding that stays in the ledger is one nobody outside this repo will ever benefit from.

So: when a cycle produces a result that would change how someone writes or reviews Go **outside this study**, append it to `/Users/noelruault/.claude/skills/performance-golang/SKILL.md` in the same cycle, under the section it belongs to. That includes negative results and harness defects, which are the most transferable things here.

Three hard rules, because this file outlives the study:

- **Employer- and project-agnostic.** No repo paths, no ticket ids, no `1brc`, no `E-NN` citations. Describe the shape of the workload ("a 13.8 GB single-file aggregation, ~14-byte records") and keep the mechanism and the number. A reader must never need this repo to use the entry.
- **Carry the number and its conditions.** "Two cursors are faster" is worthless; "−5.28% user CPU against a 0.034% control bracket, wall clock did not follow" is the entry. State the machine class and Go version.
- **Do not restate what the skill already says.** Read the neighbouring sections first and extend or correct them rather than appending a duplicate; a correction to an existing entry is worth more than a new one.

What does NOT go: anything true only of this corpus, this machine's exact clock, or this study's file layout. When unsure, write it in the ledger only and say in the handoff why it did not generalize.

## Definition of Done

`final-perf` verifies: every ticket below is either ledgered with a verdict or explicitly parked in `1brc/PARKED.md` with all seven fields; the gate is green; `1brc/09-result.md` and `1brc/README.md` agree with the ledger at every site that publishes a number; and any default that moved is justified by a disjoint, quiet-machine measurement.
