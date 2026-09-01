# Built ledger — 1brc

One line per shipped ticket, appended by the builder: `- <id> <sha> — summary`.
- def-understand 4145227 — 01-definition.md written from gunnarmorling/1brc@db06419 with cited rules, derived output arithmetic, and the integer-tenths core it requires
- env-data d22e3dc — seeded generator (413 + synthetic 10k key sets), 10m/100m/1b/10k-stations files generated outside the repo with recorded rows+bytes+sha256, trivially-correct reference and its committed expected outputs
- env-baseline d5d62fb — physical floor measured: 754.4 ms +/- 8.8 ms to read 13.8 GB (15 uncached preads), mmap 5-9x slower than read(), page cache slower than F_NOCACHE; also closed `asm-recon` 61dcbb6 (top-five technique inventory, six hypotheses, NEON movemask and merykitty's parse both verified in Go) and `go-skeleton` ddefd8f (1brc/code/go module, byte-compare gate green on 413 and 10k stations, hyperfine harness, 26.1 ns/row)
