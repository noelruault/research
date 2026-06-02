#!/usr/bin/env python3
"""Bootstrap-CI scorer for compression experiments. v2: dual-format CI."""
import json, sys, random, statistics
from pathlib import Path


def bootstrap_ci(deltas, n_resamples=10000, alpha=0.05):
    n = len(deltas)
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
            + weights.get("encode_cpu_ms", 0.05) * item["encode_cpu_ms_p95"]
            + weights.get("decode_cpu_ms", 0.5) * item["decode_cpu_ms_p95"])


def main(baseline_path, candidate_path, weights):
    base = json.loads(Path(baseline_path).read_text())
    cand = json.loads(Path(candidate_path).read_text())
    keys = [k for k in base if k in cand]
    base_scores = [score(base[k], weights) for k in keys]
    cand_scores = [score(cand[k], weights) for k in keys]
    deltas = [c - b for b, c in zip(base_scores, cand_scores)]
    pct_deltas = [(c - b) / b * 100.0 if b > 0 else 0.0
                  for b, c in zip(base_scores, cand_scores)]
    lo, hi = bootstrap_ci(deltas)
    plo, phi = bootstrap_ci(pct_deltas)
    decision = "KEEP" if hi < 0 else "DISCARD"
    out = {
        "n": len(keys),
        "mean_delta_bytes": statistics.mean(deltas),
        "mean_delta_pct": statistics.mean(pct_deltas),
        "ci_low_bytes": lo, "ci_high_bytes": hi,
        "ci_low_pct": plo, "ci_high_pct": phi,
        "decision": decision,
        "baseline_total_score": sum(base_scores),
        "candidate_total_score": sum(cand_scores),
        "per_item": [
            {"key": k, "baseline": b, "candidate": c, "delta": d, "pct": p}
            for k, b, c, d, p in zip(keys, base_scores, cand_scores, deltas, pct_deltas)
        ],
    }
    print(json.dumps(out, indent=2))


if __name__ == "__main__":
    weights = {"encode_cpu_ms": 0.0, "decode_cpu_ms": 0.5}
    main(sys.argv[1], sys.argv[2], weights)
