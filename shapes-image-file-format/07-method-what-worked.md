# 07 — Method: what actually worked

Notes for anyone (human or agent) running a similar investigation. These are the practices that caught real errors in this project, not general advice. Each one is here because it changed an outcome; report 06 is the casualty list they were built from.

## Declare one fixed eval before the first experiment

One image, one metric, one definition of "bytes", written down before anything is measured, and never renegotiated mid-investigation. Everything downstream becomes comparable for free, and rounds run weeks apart still sit in one table.

The failure this prevents is subtle: without a fixed eval, each experiment quietly picks the framing that flatters it. Four of the six killed claims in report 06 are baseline errors, and all four happened in the places where the eval had been silently redefined — lossless-of-our-own-output, original-instead-of-same-pixels, my-segmenter-instead-of-the-best-one.

## Make subagents reproduce the eval before believing them

The single highest-value rule in the whole investigation. Four independent agents investigated in parallel; each had to reproduce the shared baseline number before any of its findings were accepted.

It immediately caught an agent whose headline finding contradicted the established wall — the agent had been unable to locate the eval image and had silently substituted a different one. Its numbers were internally consistent and completely inapplicable. Nothing else in the process would have caught that; the finding was plausible, well-argued, and wrong.

## Require a mechanism, a runnable experiment, and a predicted number

Every investigation was required to produce all three, in that order, before running anything. It forces a falsifiable commitment and makes "this direction is promising" impossible to submit as a result.

The predicted-number requirement matters most. An investigation that predicts 5 KB and measures 21 KB has learned something; one that only measures has produced a number nobody can interpret.

## Frame the task as "kill this", not "explore this"

All four investigations were instructed to find the mechanism that *defeats* the idea. All four returned `VERDICT: NONE` — which is the useful outcome, because it is a bound rather than an opinion. An exploratory framing on the same material would have returned four promising directions and no bound.

## Regenerate data files; never transcribe them

When writing up, re-run the experiment and capture its output to the data file rather than pasting numbers from earlier in the session.

This caught the worst error in the investigation. Regenerating a frontier sweep produced numbers that did not match the ones already written into the report — which is how the region merge was discovered to be nondeterministic, and how a headline taken from a single lucky run was caught before publication.

## Run it twice before publishing it once

A result you have not seen reproduce is a sample, not a measurement. Cheap and near-universally skipped.

In Go specifically: `range` over a map is a silent randomizer. Any algorithm that builds a priority queue, a candidate list, or a tie-broken ordering by iterating a map is nondeterministic and will not tell you. The fix is a **total order** in the comparator — break ties on a stable key — not a seed.

## Compute comparisons in a script, not by eye

Interpolating a rate-distortion curve by hand across a dozen rows is exactly as error-prone as it sounds, and the errors are not random: they drift toward the answer you want. Report 05's first draft overstated its headline by more than 3× this way. `code/compare.py` replaced it: read both curves from their data files, interpolate at every measured point, refuse to extrapolate outside the measured range.

## Define "bytes" like-for-like, and say what you discounted

Container overhead is invisible until it is decisive. AVIF's container floor is 297 B; at the bottom of a low-rate sweep that is 10–14% of the file, large enough to manufacture a win out of nothing. WebP's is ~44 B.

The rule: state the byte definition per column, discount only what you can measure (an 8×8 flat image gives the floor directly), and say in the table caption what was discounted. An idealised cross-entropy estimate compared against whole files is not a like-for-like comparison and must be labelled.

## Steelman into the unmeasured regime before closing

Report 04 declared the idea dead after sweeping down to 256 regions — the bottom of the range the fixed eval happened to sit in. Extending the same sweep into the regime nobody had measured found the one place the idea wins.

Before parking a hypothesis, ask which part of the space the eval never visited, and go look. "We measured and it lost" is only a bound over the region you measured.

## Prefer the cheap decisive experiment to the long argument

The last live idea in the investigation — that shared dictionaries should favour structured shape data over entropy-coded rasters — could have been argued either way indefinitely. Building a best-case corpus and running `brotli -D` took about fifteen minutes and closed it permanently (1.02× versus 1.01×; no asymmetry).

When a hypothesis is testable in under an hour, testing beats reasoning about it, and beats delegating it.

## Keep a killed-mechanism table with numbers

Sixteen rows, each with what was tried, the discipline it came from, and the number that killed it. Its purpose is to stop the same idea being re-derived six months later by someone who reasons their way to it again — including its original author. A list of rejected ideas without numbers does not do this; people re-litigate it. A list with numbers ends the conversation.

## Write corrections in place, not only as appendices

When a claim is falsified, edit the original claim where it lives — README, plan, report — and leave a marked correction block at the point of the error. An appendix at the bottom of a long document does not reach the reader who quotes paragraph three.

Both consuming projects also carry an explicit risk note naming the killed claim, because the failure mode is specifically that someone re-derives an attractive wrong result and puts it back.

## What did not work

- **Reasoning about which of two explanations applies.** Starvation versus dilution (report 06 #2) was settled in ten minutes by an experiment after being wrong in prose for a full round.
- **Trusting a negative result that depends on your own quick implementation.** A wall measured with a component written in an afternoon is a measurement of that component (report 06 #3).
- **Literature-absence claims from agent sweeps.** "Nobody has benchmarked X against Y" is the hardest class of claim to establish and the easiest to assert. It is flagged as unverified in report 04 rather than load-bearing.
