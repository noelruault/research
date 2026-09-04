# Lessons — 1brcPerf

Read every cycle. One tight bullet per durable lesson: a gate invocation that works here, a repo gotcha, a class of ticket that keeps failing and why.

- Gate that works, in this order, all from the repo root: `cd 1brc/code/go && test -z "$(gofmt -l .)" && go vet ./... && go build -o bin/1brc . && go test ./...`, then `bash 1brc/scripts/check-correctness.sh` (~40 s), then `ARM="<flags>" bash 1brc/scripts/check-correctness.sh` for any arm you are about to time, then `bash 1brc/scripts/experiment.sh --self-test`. Run ALL of them before launching a timed run — a build during hyperfine voids the invocation.
- A timed 1b run at `-r 10` is ~12 min (3 arms × 11 runs × (1.3 s + 20 s cooldown)) and exceeds the 600 s Bash ceiling, so it MUST be the background task's own command. Not `nohup … & sleep 5`: the task's process group dies when that command exits, which killed the first `lanes-requiet` attempt mid-bracket.
- `measure-when-idle.sh` has been operator-overridden to measure NOW (`IDLE_MIN=0 QUIET_LOAD=6.0 QUIET_FORCE=1`, 2026-09-04), but every knob is `${VAR:-default}`. A ticket whose question NEEDS a quiet machine passes its own `IDLE_MIN=600 QUIET_LOAD=4.2 QUIET_FORCE=0 QUIET_WAIT=28800` on that one invocation, rather than editing the operator's defaults for the whole loop.
- Ledger a member the moment its code is green, not at the end of the cycle. `loop-next.sh` greps `built.md` by literal id, so committed-but-unledgered code (this happened to `busiest-stamp`) is invisible to the selector and gets re-dispatched.
- `scripts/loop-next-selftest.sh` fails PRE-EXISTING on `skill-performance-assembly` and `x-transfer` — the OLD `docs/1brc` loop's selector fixture. It is not in spec.md's gate. Do not try to fix it.
