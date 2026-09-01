# Lessons — 1brc

Durable, cycle-to-cycle. Read first, obey. One tight bullet each; no one-off noise.

- Gate for `1brc/code/gen`: `cd 1brc/code/gen && test -z "$(gofmt -l .)" && go vet ./... && go build ./... && go test ./...`. The `test -z` matters: `gofmt -l` exits 0 even when it lists unformatted files, so `gofmt -l . && ...` silently passes.
- Tracked files cannot be written from the shell here (a `bash-write-guard` PreToolUse hook blocks heredocs and `sed -i` on anything git-tracked). Use Edit/Write for tracked files; shell heredocs only work for files not yet committed. A `comment-verbosity` hook also rejects comment blocks over ~3 lines, and a formatter hook reflows comments to one sentence per line after every edit.
- Two of the 413 official station names contain a comma (`Washington, D.C.`, `Flores,  Petén`) and the output separator is `", "`. Never split the result line on `", "` to compare or count entries; byte-compare against `1brc/testdata/expected-*.out`, and count entries with `tr -cd '=' | wc -c`.
- A decimal that looks like a rounding tie usually is not. `-0.05` and `-0.1/2` both sit just below the tie in float64, so they round to `-0.1`, not `-0.0`. Two test expectations were wrong on this before the code was. When a mean is off by one tenth, suspect the float64 representation of the tie before suspecting the rounding rule.
- Label performance and numeric claims before writing them into a comment, and measure the cheap ones. A "float64 drifts ~1e-2" justification in `tenths.go` was wrong by four orders of magnitude; a 15-line test settled it in one run. The worst-case bound `n·eps·sum` is not the behaviour — summation error cancels like `sqrt(n)·eps·sum`.
- `openssl dgst -sha256` hashes the 13.8 GB file in ~6 s (hardware SHA); `shasum -a 256` is roughly 20x slower. Use openssl for anything over a gigabyte.
- Generation is ~725 MB/s single-threaded (1b rows in 18 s), so do not reach for a parallel generator. Regenerating any file is cheap; the seed and command in `02-baseline-data.txt` reproduce every one byte-for-byte.
- No JDK on this machine (Java 8 JRE, no `javac`), so upstream's Java cannot be run and output semantics are derived from its source. If a correctness question turns on what the JVM actually does, install a JDK 21+ first rather than reasoning about it.
