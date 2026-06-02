#!/usr/bin/env python3
"""Bootstrap-CI scorer for compression experiments.

Reads two JSON files (baseline and candidate), each keyed by item path with
fields: wire_bytes_p95, encode_cpu_ms_p95, decode_cpu_ms_p95.
Computes per-item score deltas and a 10,000-resample bootstrap CI95.

Decision rule (per agent definition Section 4.1, Section 5.6, SCOPE.md weights):
  KEEP iff CI95_high < 0 (delta strictly improves vs baseline).
"""
import json
import sys
import random
import statistics
from pathlib import Path


def bootstrap_ci(deltas, n_resamples=10000, alpha=0.05):
    n = len(deltas)
    if n == 0:
        return 0.0, 0.0
    means = []
    rng = random.Random(0xC0FFEE)
    for _ in range(n_resamples):
        sample = [deltas[rng.randrange(n)] for _ in range(n)]
        means.append(statistics.mean(sample))
    means.sort()
    lo = means[int(n_resamples * alpha / 2)]
    hi = means[int(n_resamples * (1 - alpha / 2))]
    return lo, hi


def score(item, weights):
    return (item["wire_bytes_p95"]
            + weights.get("encode_cpu_ms", 0.0) * item["encode_cpu_ms_p95"]
            + weights.get("decode_cpu_ms", 0.5) * item["decode_cpu_ms_p95"])


def main(baseline_path, candidate_path, weights):
    base = json.loads(Path(baseline_path).read_text())
    cand = json.loads(Path(candidate_path).read_text())
    keys = [k for k in base if k in cand]
    pairs = [(base[k], cand[k]) for k in keys]
    deltas = [score(c, weights) - score(b, weights) for b, c in pairs]
    base_scores = [score(b, weights) for b, _ in pairs]
    cand_scores = [score(c, weights) for _, c in pairs]
    lo, hi = bootstrap_ci(deltas)
    base_total = sum(base_scores)
    cand_total = sum(cand_scores)
    pct = ((cand_total - base_total) / base_total * 100.0) if base_total else 0.0
    decision = "KEEP" if hi < 0 else "DISCARD"
    print(json.dumps({
        "n": len(deltas),
        "base_total_score": base_total,
        "cand_total_score": cand_total,
        "delta_total_pct": pct,
        "mean_delta": statistics.mean(deltas) if deltas else 0.0,
        "ci_low": lo,
        "ci_high": hi,
        "decision": decision,
        "items": {k: {"base_score": score(b, weights),
                      "cand_score": score(c, weights),
                      "delta": score(c, weights) - score(b, weights)}
                  for k, (b, c) in zip(keys, pairs)},
    }, indent=2))


if __name__ == "__main__":
    # SCOPE.md weights for infra-a-spa: encode_cpu_ms=0.0 (build-time free),
    # decode_cpu_ms=0.5 (mobile decode matters).
    weights = {"encode_cpu_ms": 0.0, "decode_cpu_ms": 0.5}
    main(sys.argv[1], sys.argv[2], weights)
