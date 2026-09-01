#!/usr/bin/env bash
# Both properties were once false: grep -q's early exit raced printf into SIGPIPE under pipefail, so a MATCH could report "not built" and 15 runs returned 6 distinct groups of already-built work.
# Usage: bash scripts/loop-next-selftest.sh [<docs-subdir>] [<runs>]
set -euo pipefail

DOCS="${1:-1brc}"
RUNS="${2:-40}"
ROOT="$(git rev-parse --show-toplevel)"
BUILT="$ROOT/docs/$DOCS/built.md"
fail=0

out="$(for _ in $(seq 1 "$RUNS"); do bash "$ROOT/scripts/loop-next.sh" "$DOCS" | tr '\n' ' '; echo; done)"
distinct="$(printf '%s\n' "$out" | sort -u)"
n="$(printf '%s\n' "$distinct" | wc -l | tr -d ' ')"
if [ "$n" -ne 1 ]; then
  echo "FAIL: $RUNS runs returned $n distinct groups, expected 1:"
  printf '%s\n' "$distinct" | sed 's/^/  /'
  fail=1
else
  echo "ok: $RUNS runs agree on: $(printf '%s' "$distinct")"
fi

# The selector greps built.md by literal id token, so a dispatched built id is a silent redispatch.
built_ids="$(
  grep -oE '`[a-z0-9-]+`' "$BUILT" | tr -d '`'
  grep -oE '^- ?[a-z0-9-]+' "$BUILT" | sed 's/^- *//'
)"
for id in $(printf '%s\n' "$out" | tr ' ' '\n' | sort -u); do
  [ -z "$id" ] && continue
  [ "$id" = DONE ] && continue
  if grep -qxF "$id" <<<"$built_ids"; then
    echo "FAIL: selector dispatched '$id', which is already in docs/$DOCS/built.md"
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "ok: no dispatched id appears in built.md"

exit "$fail"
