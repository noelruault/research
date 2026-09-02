# CLAUDE.md

Guidance for working in this repository.

## What this is

A record of finished and ongoing research, one directory per question. Each study keeps its reports numbered oldest-first with a `*-data.txt` companion holding raw output and the command that produced it, so every number can be re-derived rather than trusted.

## Parked ideas — the convention lives here

`PARKED.md` at the root defines it; each study keeps its own register. Every entry carries **what it depends on** and **a concrete revive trigger**, because an idea is usually shelved for a reason that is true at the time, and when that fact moves the idea can become good again with nobody noticing.

**Re-read a study's register whenever any baseline in it moves.** That is exactly when a parked entry becomes valuable and exactly when nobody looks. This is not hypothetical: a measured, verified −6.40% improvement became worth zero without being touched, because an unrelated commit improved a *different* coder and moved the threshold between them.

Distinguish **killed on an argument that does not expire** (safe to leave) from **killed on numbers** (numbers move — record the baseline it was measured against).

## shapes-image-file-format

**Start at `shapes-image-file-format/HANDOFF.md`.** It carries the state, what is settled, the method rules, and the queue.

The short version: **a poor image codec and a good structured-image format.** It loses to WebP alone on all 24 Kodak images (mean +27.8%, zero wins) and beats WebP **plus a region-map sidecar** on all 24 (mean +30.5%). Price it against raster+sidecar, never against a raster codec alone. The bytes question is settled — do not re-derive it. Thirteen claims were falsified in the making; the register is `06-corrections-and-falsifications.md`.

## 1brc

**Start at [`1brc/README.md`](1brc/README.md), and at [`1brc/09-result.md`](1brc/09-result.md) for the closing statement.** The question is settled and the study is closed: **1.233 s ± 0.010 s** for a billion rows against a **1.000 s** target, **+23.3%, NOT REACHED**. Do not re-derive the gap; it decomposes with no residual (80.5% user CPU, 8.0% system, 11.5% idle) and 09 names both bounds on closing it.

Three things in there bind any future work on this study, and each cost a measurement to learn.

**A 1b timing is only a number inside a bracketed invocation.** Deltas are taken between arms of ONE `hyperfine` run, with the incumbent named first AND last and a 20 s cooldown before every timed run; a bracket wider than 3% means no arm in that invocation may be quoted. Eight IDENTICAL arms without the cooldown rose monotonically by **21.08%**. Use `1brc/scripts/experiment.sh`, which enforces all of it and refuses to run when another measurement holds the lock or the machine is busy.

**A cheaper proxy does not rank arms.** Seven arms measured at 100m disagreed with the 1b file seven times, four of them inverting sign. The harness refuses any file but 1b unless `--mechanism-only`, and then stamps the output `NOT A VERDICT`.

**The headline does not reproduce across sessions, and the study says so.** The byte-identical binary reads 1.257 s a day later in a 0.000%-spread bracket (`CORRECTIONS.md` C13). Both figures are published, because the miss is the same size on either.

## Measuring wall clock in this repo

The sibling of the compression rules below, and the same kind of lesson: the number you get is a fact about the harness until you prove otherwise. Promoted from `1brc/08-method-what-worked.md`, which paid for each rule with a wrong verdict first; that report carries the evidence.

**A delta is only taken inside one bracketed invocation**, with the incumbent named FIRST and LAST — first is the most flattered slot there is, so an incumbent measured only there is the one arm the invocation cannot judge. **A bracket spread over 3% is a refusal to quote any arm, never a correction factor**: a modelled per-slot drift once "recovered" +19.18% for a change that re-measured at +0.06%. A contaminated invocation is re-run, not repaired.

**Ask the harness to rank N copies of one thing before believing it about N different things.** The null experiment is what you run before the first verdict, not after one embarrasses you — eight identical arms once rose monotonically by 21.08%, wider than every verdict then pending. Re-run it whenever the harness changes.

**A cheaper proxy has to be shown to rank arms before it is used to rank arms**, and so does a differently-shaped baseline — they fail independently. Seven arms on a 10x-smaller input disagreed with the real one seven times, four inverting sign; a kernel measured −40.4% against a baseline the real program did not contain scored +10.4% in it.

**Any provenance field describing a condition under which the number is invalid should be a precondition that refuses, not a field that records.** Recording load and power did not stop two measurements being voided by a housekeeping storm; a lock and a quiet gate did. This one costs a harness change per study — a study whose measurements are cheap and repeatable does not need it.

**For studies that ship code: mutate the decision the cycle just made**, not only the arithmetic, and classify each surviving mutant as a real gap, an equivalent mutant, or unkillable by construction before scoring it. A semantic choice with no test on it shipped three separate times in one study while every byte-compare passed, because no corpus input could express the divergence.

## Measuring compression in this repo

**A modelled cost is not a compressed stream.** Cross-entropy of an adaptive coder ranks *models*; it does not rank *streams*, and it can point the wrong way. Measured here: at four of twenty-one operating points a modelled figure said a transform was **+7.4% worse** while `brotli -q11` on the real residual stream said **−9.2% better** — the wrong sign, on the largest colour win in the study. The mechanism was that the stream was 37% zeros, and a general compressor lives on exact-hit rate and LZ matches rather than residual variance.

So: dump the stream, run `brotli -q11` or `xz -9e` on it, and report the compressed number as the headline. If only a modelled figure exists, label it as ranking models.

**Changing a partition invalidates every baseline priced on the old one.** Each arm must build its own and be compared at matched *fidelity*, never by pricing one arm's partition with the other's coder.
