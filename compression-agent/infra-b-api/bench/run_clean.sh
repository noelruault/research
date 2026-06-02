#!/usr/bin/env bash
# Clean per-experiment runner using bench/measure_clean.py.
# Bypasses hyperfine's shell-redirect overhead.
#
# Usage: ./bench/run_clean.sh <exp-id> <algo> [include-tiny=0|1]
set -euo pipefail
EXP_ID="${1:?exp id required}"
ALGO="${2:?algo required}"
INCLUDE_TINY="${3:-0}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CORPUS="$ROOT/bench/corpus/http"
OUT="$ROOT/bench/results/$EXP_ID"
mkdir -p "$OUT/encoded"

ITEMS=()
for f in "$CORPUS"/*.json; do
  size=$(wc -c < "$f")
  if [ "$INCLUDE_TINY" = "1" ] || [ "$size" -ge 1024 ]; then
    ITEMS+=("$(basename "$f")")
  fi
done

echo "[clean] exp=$EXP_ID algo=$ALGO items=${#ITEMS[@]} include_tiny=$INCLUDE_TINY"

python3 - "$ROOT" "$OUT" "$ALGO" "${ITEMS[@]}" <<'PYEOF'
import json, os, subprocess, sys
from pathlib import Path

ROOT = sys.argv[1]
OUT = Path(sys.argv[2])
ALGO = sys.argv[3]
ITEMS = sys.argv[4:]

items = {}
for name in ITEMS:
    in_path = f"{ROOT}/bench/corpus/http/{name}"
    p = subprocess.run(
        ["python3", f"{ROOT}/bench/measure_clean.py", ALGO, in_path, "30", "3"],
        check=True, capture_output=True, text=True, env=os.environ.copy(),
    )
    r = json.loads(p.stdout)
    items[name] = {
        "wire_bytes_p95": r["encoded_size"],
        "encode_cpu_ms_p95": r["encode_ms_p95"],
        "decode_cpu_ms_p95": r["decode_ms_p95"],
        "encode_cpu_ms_mean": r["encode_ms_mean"],
        "decode_cpu_ms_mean": r["decode_ms_mean"],
        "encode_cpu_ms_min": r["encode_ms_min"],
        "decode_cpu_ms_min": r["decode_ms_min"],
    }
    print(f"  {name}: bytes={r['encoded_size']} enc_p95={r['encode_ms_p95']:.3f}ms dec_p95={r['decode_ms_p95']:.3f}ms")

(OUT / "items.json").write_text(json.dumps(items, indent=2))
print(f"[clean] wrote {OUT}/items.json with {len(items)} items")
PYEOF

BASELINE="$ROOT/bench/results/baseline.json"
if [ -f "$BASELINE" ] && [ "$EXP_ID" != "0001" ] && [ "$INCLUDE_TINY" = "0" ]; then
  echo "[clean] scoring vs baseline..."
  python3 "$ROOT/bench/score.py" "$BASELINE" "$OUT/items.json" \
    | tee "$OUT/score.json"
fi
