# Built ledger — 1brcPerf

One line per shipped ticket, appended by the builder: `- <id> <sha> — summary`.
- idle-gate-sigpipe 9e2fbcc — hid_idle SIGPIPE'd ioreg (141 in 296/300 calls) and killed the unattended wait under set -euo pipefail; awk now drains, pinned by a mutated 40-call self-test case
- busiest-stamp 32e3e60 — provenance_header stamps busiest: on every header, not only over the load line, so a core-stealer under the threshold (3 Defender daemons at ~150% of a core at load 4.50) is recorded instead of voiding a bracket silently like E-37
