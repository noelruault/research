#!/usr/bin/env bash
# bench/harness.sh — fixed harness, runs measure.py for one candidate.
# Usage: harness.sh <algo> [<exp-id>] [<corpus_dir>]
# Env: DICT (for dict variants), RUNS (default 30), ITEMS (comma-separated filter)
set -euo pipefail

ALGO="$1"
EXP_ID="${2:-$ALGO}"
CORPUS_DIR="${3:-bench/corpus/assets}"

OUT_DIR="bench/results/$EXP_ID"
mkdir -p "$OUT_DIR"

python3 bench/measure.py "$ALGO" "$CORPUS_DIR" "$OUT_DIR/items.json"

# Compare against baseline; tolerate missing baseline (e.g. baseline run itself)
if [ -f bench/results/baseline.json ] && [ "$EXP_ID" != "baseline" ] && [ "$EXP_ID" != "0001" ]; then
  python3 bench/score.py bench/results/baseline.json "$OUT_DIR/items.json" > "$OUT_DIR/score.json"
  python3 -c "import json; s=json.load(open('$OUT_DIR/score.json')); print(f\"score: decision={s['decision']} mean_delta_bytes={s['mean_delta_bytes']:.1f} mean_delta_pct={s['mean_delta_pct']:.2f}% CI95_bytes=[{s['ci_low_bytes']:.1f},{s['ci_high_bytes']:.1f}] CI95_pct=[{s['ci_low_pct']:.2f}%,{s['ci_high_pct']:.2f}%]\")"
fi
