#!/usr/bin/env bash
# bench/harness.sh — fixed harness, agent invokes per candidate. v3 §5.5.
# Usage: harness.sh <candidate> [corpus_dir]
set -euo pipefail
CAND="${1:?usage: harness.sh <candidate> [corpus_dir]}"
CORPUS="${2:-bench/corpus/assets}"
OUT="bench/results/$CAND"
mkdir -p "$OUT"

# Sub-ms timing path: use measure.py (v3 §4.5).
# Largest input here is ~256 KB vendor JS at brotli-11 (>5 ms expected on that one),
# but measure.py handles all 9 items uniformly with a single subprocess.run model.
# Per v3 §4.5: hyperfine only when per-iter cost ≥ 5 ms AND input ≥ 50 KB across
# the whole set; mixed corpus → measure.py is the right tool.

python3 bench/measure.py "$CAND" "$CORPUS" "$OUT/items.json"

# Score vs baseline if it exists
if [[ -f bench/results/baseline.json ]] && [[ "$CAND" != "identity" ]]; then
  python3 bench/score.py bench/results/baseline.json "$OUT/items.json" \
    > "$OUT/score.json"
  echo "Score for $CAND:"
  python3 -c "import json; d=json.load(open('$OUT/score.json')); \
    print(f\"  decision={d['decision']}\"); \
    print(f\"  mean_delta_bytes={d['mean_delta_bytes']:.1f}\"); \
    print(f\"  mean_delta_pct={d['mean_delta_pct']:.2f}%\"); \
    print(f\"  ci_bytes=[{d['ci_low_bytes']:.1f}, {d['ci_high_bytes']:.1f}]\"); \
    print(f\"  ci_pct=[{d['ci_low_pct']:.2f}%, {d['ci_high_pct']:.2f}%]\")"
fi
