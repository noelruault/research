#!/usr/bin/env python3
"""Bootstrap-CI scorer for compression experiments.

Per Section 4.4 of compression-engineer.md.
SCOPE.md weights: encode_cpu_ms=0.5, decode_cpu_ms=0.3.

Score formula:
    score = wire_bytes_p95 + 0.5 * encode_cpu_ms_p95 + 0.3 * decode_cpu_ms_p95

Decision rule (Section 4.1): KEEP iff CI95_high < 0 (strict).
Bootstrap: 10000 resamples, deterministic seed 0xC0FFEE, alpha=0.05.
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
            + weights.get("encode_cpu_ms", 0.5) * item["encode_cpu_ms_p95"]
            + weights.get("decode_cpu_ms", 0.3) * item["decode_cpu_ms_p95"])


def main(baseline_path, candidate_path, weights):
    base = json.loads(Path(baseline_path).read_text())
    cand = json.loads(Path(candidate_path).read_text())
    keys = sorted(set(base) & set(cand))
    pairs = [(base[k], cand[k]) for k in keys]
    deltas = [score(c, weights) - score(b, weights) for b, c in pairs]
    lo, hi = bootstrap_ci(deltas)
    decision = "KEEP" if hi < 0 else "DISCARD"
    base_total = sum(score(b, weights) for b, _ in pairs)
    cand_total = sum(score(c, weights) for _, c in pairs)
    pct = 100.0 * (cand_total - base_total) / base_total if base_total else 0.0
    out = {
        "baseline": baseline_path,
        "candidate": candidate_path,
        "n_items": len(deltas),
        "items": keys,
        "baseline_total_score": base_total,
        "candidate_total_score": cand_total,
        "pct_delta": pct,
        "mean_delta": statistics.mean(deltas) if deltas else 0.0,
        "median_delta": statistics.median(deltas) if deltas else 0.0,
        "ci_low": lo,
        "ci_high": hi,
        "decision": decision,
        "weights": weights,
    }
    print(json.dumps(out, indent=2))
    return out


if __name__ == "__main__":
    weights = {"encode_cpu_ms": 0.5, "decode_cpu_ms": 0.3}
    main(sys.argv[1], sys.argv[2], weights)
