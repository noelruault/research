#!/usr/bin/env bash
# bench/harness.sh — v3 harness driver.
# Section 5.5: chooses measure.py for sub-ms work; hyperfine reserved for >=5 ms / >=50 KB.
# This smoke test uses measure.py for ALL experiments per SCOPE.md note.
set -euo pipefail

CAND="$1"                              # e.g. zstd-9
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd)"
CORPUS="$HERE/corpus/http"
OUT="$HERE/results/$CAND"
mkdir -p "$OUT"

# Sub-ms path: measure.py (Section 4.5).
python3 "$HERE/measure.py" "$CAND" "$CORPUS" "$OUT/items.json.full"

# Apply SCOPE.md in-scope filter: exclude items below 1024 B threshold
# (error-404.json @ 65 B, user-profile.json @ 278 B).
python3 -c "
import json, sys
data = json.load(open('$OUT/items.json.full'))
EXCLUDE = {'error-404.json', 'user-profile.json'}
filtered = {k: v for k, v in data.items() if k not in EXCLUDE}
json.dump(filtered, open('$OUT/items.json', 'w'), indent=2)
print('  in-scope items   :', len(filtered))
print('  excluded         :', sorted(EXCLUDE & data.keys()))
"

# Score vs baseline (skipped for the baseline run itself).
if [[ "$CAND" != "identity" ]]; then
  python3 "$HERE/score.py" "$HERE/results/baseline.json" "$OUT/items.json" \
    > "$OUT/score.json"
  python3 -c "
import json,sys
s = json.load(open('$OUT/score.json'))
print(f\"  decision           : {s['decision']}\")
print(f\"  n                  : {s['n']}\")
print(f\"  mean_delta_bytes   : {s['mean_delta_bytes']:+.1f}\")
print(f\"  mean_delta_pct     : {s['mean_delta_pct']:+.2f}%\")
print(f\"  ci95 bytes         : [{s['ci_low_bytes']:+.1f}, {s['ci_high_bytes']:+.1f}]\")
print(f\"  ci95 pct           : [{s['ci_low_pct']:+.2f}%, {s['ci_high_pct']:+.2f}%]\")"
fi
