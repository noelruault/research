# Review ledger — 1brcPerf

One line per group, written by the cycle that built it after it audited its own diff. This file is what this codebase remembers about what it gets wrong; read it before you build.

- reviewed idle-gate-sigpipe 9e2fbcc, busiest-stamp 32e3e60: REPAIRED — both instrument fixes are correct and both self-test cases were mutate-verified this cycle (`awk … exit` → `FAIL (status 141 on a call within 40)`; deleted `busiest:` echo → `FAIL: named no busiest process on a quiet machine`), but `busiest-stamp` had landed with NO ledger line, NO handoff block and NO review line, and this file did not exist at all. The defect was bookkeeping, not code: a member whose code is committed but unledgered is invisible to `loop-next.sh`, which greps `built.md` by literal id, so the ticket would have been re-dispatched and re-implemented. Added the `built.md` line, the handoff block and this file.
