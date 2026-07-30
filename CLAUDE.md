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

## Measuring compression in this repo

**A modelled cost is not a compressed stream.** Cross-entropy of an adaptive coder ranks *models*; it does not rank *streams*, and it can point the wrong way. Measured here: at four of twenty-one operating points a modelled figure said a transform was **+7.4% worse** while `brotli -q11` on the real residual stream said **−9.2% better** — the wrong sign, on the largest colour win in the study. The mechanism was that the stream was 37% zeros, and a general compressor lives on exact-hit rate and LZ matches rather than residual variance.

So: dump the stream, run `brotli -q11` or `xz -9e` on it, and report the compressed number as the headline. If only a modelled figure exists, label it as ranking models.

**Changing a partition invalidates every baseline priced on the old one.** Each arm must build its own and be compared at matched *fidelity*, never by pricing one arm's partition with the other's coder.
