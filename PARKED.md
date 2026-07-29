# Parked ideas — the convention

**The problem this exists to solve.** An idea gets set aside for a reason that is true *at the time*. Later something else moves — a baseline improves, a constant gets re-measured, a competitor's assumption is dropped — and the idea silently becomes good again. Nobody notices, because what was written down was "tried it, didn't help".

That has already happened twice in this repo in a single day:

- **A measured −6.40% improvement became worth zero without being touched.** `shapes-image-file-format` report 09 improved the CAE wall coder, which moved the CAE/contour crossover from ~6,400 regions down to ~849. That retroactively made the contour turn-coder work worth ~0% instead of −2.76%. Nothing about the turn coder changed. If the crossover ever moves back, that work is instantly valuable again — and only a *dependency-aware* note would tell you.
- **Six research agents were cancelled on an unmeasured ceiling** and restored an hour later when the reasoning was challenged. The cancellation note said "~1% headroom remains", which was an inference from an oracle inside one model family, not a measurement.

So: **a parked idea is not documented until someone could tell, from the note alone, whether today's world has made it good again.**

## Required fields

Every parked entry carries all seven. An entry missing "revive when" or "depends on" is not parked, it is forgotten.

| field | why it is mandatory |
|---|---|
| **What it is** | Concrete enough to rebuild without re-deriving. Name the mechanism, not the vibe. |
| **Status** | `parked` (might return) / `killed` (mechanism is wrong) / `subsumed` (something else does it better) / `blocked` (waiting on a dependency). |
| **Why parked — with the number** | The measurement or argument that stopped it. "Didn't help" is not a reason; "−0.076% of the file at its best operating point" is. |
| **Depends on** | The facts that made it a bad idea. **This is the field that does the work.** If any of them changes, the entry needs re-reading. |
| **Revive when** | A concrete, checkable trigger. "If X exceeds Y" or "once Z lands". Not "if we have time". |
| **Cost to revive** | Rough. Distinguishes "an afternoon" from "a new subsystem". |
| **Where the work is** | Commits, code paths, data files. So reviving is resuming, not restarting. |

## Distinguishing killed from parked

**Killed** means the mechanism is wrong on an argument that does not expire — e.g. "anything the decoder can regrow from decoded pixels is a context model, bounded by `H(X | causal past)`". That does not become false when a baseline improves.

**Parked** means the mechanism works but lost on *numbers*, and numbers move.

**The dangerous middle: things killed on numbers that have since changed.** A mechanism rejected because it cost more than it saved was measured against a *specific* baseline. Improve the baseline and the comparison can flip in either direction. Those belong in the parked register with the baseline they were measured against recorded explicitly, so a future reader can see the comparison is stale.

## Where entries live

- Per study: `<study>/PARKED.md` holds the entries.
- This file holds the convention and the index below.

| study | parked register |
|---|---|
| `shapes-image-file-format` | [`shapes-image-file-format/PARKED.md`](shapes-image-file-format/PARKED.md) |
| `quantization` | not yet written |
| `nearest-color-scaling` | not yet written |
| `compression-agent` | not yet written |

## Review trigger

Re-read a study's parked register whenever **any baseline in that study moves** — a coder improves, a constant is re-measured, a comparison is re-run. That is precisely the moment a parked entry can silently become valuable, and precisely the moment nobody thinks to look.
