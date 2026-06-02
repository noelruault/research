#!/usr/bin/env bash
# bench/harness.sh — fixed harness, agent invokes per candidate.
# Uses bench/measure.py (Section 4.5) for sub-millisecond timing on JSON inputs <50 KB.
# Files in this corpus are mostly <200 KB and encode CPU is <30 ms; measure.py path used.
set -euo pipefail
CAND="$1"                              # e.g. zstd-3, brotli-5, zstd-dict-3
EXP_ID="${2:-$CAND}"                   # experiment slug for output dir
BUDGET="${BUDGET_SECONDS:-30}"
OUT="bench/results/$EXP_ID"
CORPUS="bench/corpus/http"
mkdir -p "$OUT"

# 1. Measure encode + decode CPU for every corpus item via measure.py
RUNS="${RUNS:-30}"
WARMUP="${WARMUP:-3}"
RUNS=$RUNS WARMUP=$WARMUP DICT="${DICT:-}" \
  python3 bench/measure.py "$CAND" "$CORPUS" "$OUT/items.json"

# 2. Score vs baseline
python3 bench/score.py bench/results/baseline.json "$OUT/items.json" > "$OUT/score.json"
echo "[$EXP_ID] score written to $OUT/score.json"
python3 -c "
import json
s = json.load(open('$OUT/score.json'))
print('  decision:', s['decision'])
print('  n_items:', s['n'])
print('  baseline_total_wire:', s['baseline_total_wire_bytes'])
print('  candidate_total_wire:', s['candidate_total_wire_bytes'])
print(f\"  mean delta bytes:  {s['mean_delta_bytes']:.1f} ({s['mean_delta_pct']:.2f}%)\")
print(f\"  CI95 bytes:        [{s['ci_low_bytes']:.1f}, {s['ci_high_bytes']:.1f}]\")
print(f\"  CI95 percent:      [{s['ci_low_pct']:.2f}%, {s['ci_high_pct']:.2f}%]\")
"
