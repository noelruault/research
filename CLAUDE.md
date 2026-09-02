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

## Measuring compression in this repo

**A modelled cost is not a compressed stream.** Cross-entropy of an adaptive coder ranks *models*; it does not rank *streams*, and it can point the wrong way. Measured here: at four of twenty-one operating points a modelled figure said a transform was **+7.4% worse** while `brotli -q11` on the real residual stream said **−9.2% better** — the wrong sign, on the largest colour win in the study. The mechanism was that the stream was 37% zeros, and a general compressor lives on exact-hit rate and LZ matches rather than residual variance.

So: dump the stream, run `brotli -q11` or `xz -9e` on it, and report the compressed number as the headline. If only a modelled figure exists, label it as ranking models.

**Changing a partition invalidates every baseline priced on the old one.** Each arm must build its own and be compared at matched *fidelity*, never by pricing one arm's partition with the other's coder.
