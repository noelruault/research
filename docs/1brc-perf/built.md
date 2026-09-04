# Built ledger — 1brcPerf

One line per shipped ticket, appended by the builder: `- <id> <sha> — summary`.
- idle-gate-sigpipe 9e2fbcc — hid_idle SIGPIPE'd ioreg (141 in 296/300 calls) and killed the unattended wait under set -euo pipefail; awk now drains, pinned by a mutated 40-call self-test case
