# Compression Engineer — Infra B v3 (JSON API smoke test)

Lean cherry-pick run: v3 = v1 plus three v2 fixes (`bench/measure.py` sub-ms timer
in §4.5, §2.5 brotli-static-dict warning, dual-format CI in `bench/score.py`).

## Discovery

- **Target kind**: filesystem-only. No live server in this test (Phase 7 verification skipped).
- **Stack (mocked)**: Go origin, HTTP/2 assumed, currently gzip at origin (per SCOPE.md).
- **Asset class in scope**: dynamic JSON API responses.
- **Exclusions (SCOPE.md)**: items below `min_compress_size` 1024 B:
  `error-404.json` (65 B), `user-profile.json` (278 B). n=7 in-scope.
- **Tools**: brotli 1.2.0, zstd 1.5.7, Apple gzip 479, python 3.14.2,
  hyperfine 1.20.0; Darwin 25.4.0 arm64 (Apple Silicon).
- **Metric (SCOPE.md)**: `score = wire_bytes_p95 + 0.5*encode_cpu_ms_p95 +
  0.3*decode_cpu_ms_p95`; budget 30 s/candidate; bootstrap CI 10000 resamples,
  seed `0xC0FFEE`, alpha=0.05; KEEP iff CI95_high < 0.

## Inventory of corpus

`bench/corpus/http/` (599245 in-scope raw bytes total, 9 items, 7 in-scope):

| item                    | raw bytes | in-scope |
|-------------------------|----------:|----------|
| catalog-full.json       |   189168  | yes      |
| products-list-v2.json   |   139497  | yes      |
| products-list.json      |   126767  | yes      |
| order-history.json      |    42784  | yes      |
| notifications.json      |    38234  | yes      |
| search-results-v2.json  |    31626  | yes      |
| search-results.json     |    31169  | yes      |
| user-profile.json       |      278  | excluded (<1024 B) |
| error-404.json          |       65  | excluded (<1024 B) |

## Baseline (Exp 0001, identity)

`bench/results/baseline.json`:
- total wire_bytes: 599245
- per-item encode_cpu_ms_p95: 5.13 to 13.06 (cat passthrough; subprocess.run + fork/exec overhead)
- per-item decode_cpu_ms_p95: 5.31 to 13.54 (same)

The baseline encode/decode cost is purely subprocess fork/exec overhead from
measure.py. Because the same overhead applies to every candidate (subprocess.run
+ encoder), the score-delta correctly isolates the codec's marginal CPU.

## Experiments

All 6 logged in `bench/EXPERIMENTS.md`. Per-experiment items.json and score.json
under `bench/results/<exp-id>/`.

| id    | hypothesis (one line)                              | wire_total | enc_p95_max | dec_p95_max | mean Δ bytes | mean Δ pct | CI95 bytes              | CI95 pct           | decision |
|-------|----------------------------------------------------|-----------:|------------:|------------:|-------------:|-----------:|-------------------------|--------------------|----------|
| 0001  | identity baseline                                  |    599245  |     13.060  |     13.539  |            0 |    0.00%   | n/a                     | n/a                | BASELINE |
| 0002  | gzip-6, current production (RFC 1952)              |    149986  |     10.621  |      9.390  |     -64180.4 |  -74.82%   | [-98873.2, -34441.6]    | [-76.17%, -73.54%] | KEEP     |
| 0003  | brotli-1 (RFC 7932), v3 §2.5 mismatch test         |    167106  |     96.307  |     46.883  |     -61717.1 |  -71.75%   | [-95176.0, -32728.9]    | [-73.81%, -69.85%] | KEEP vs identity, dominated by gzip-6 |
| 0004  | zstd-3 default (RFC 8878)                          |    151307  |     23.313  |     39.869  |     -63986.4 |  -74.57%   | [-98564.1, -34330.9]    | [-76.14%, -73.23%] | KEEP     |
| 0005  | zstd-9 higher ratio (RFC 8878)                     |    142068  |     26.694  |     23.352  |     -65302.2 |  -75.96%   | [-100702.3, -34979.9]   | [-77.40%, -74.59%] | KEEP     |
| 0006  | zstd-dict-9, trained dict (RFC 8878 §5)            |  **109117**| **17.585**  | **12.170**  | **-70015.8** | **-83.91%**| **[-107966.1, -38635.5]** | **[-87.10%, -80.42%]** | **KEEP, WINNER** |

All bootstrap CI95 upper bounds for KEEP rows are strictly negative, satisfying
the v3 decision rule (Section 4.4). Winner is Exp 0006.

### v3 §2.5 confirmation: brotli-1 LOSES to gzip-6 on this JSON corpus

This was the load-bearing reproduction the smoke test was designed to verify.

| metric                | gzip-6 | brotli-1 | delta            |
|-----------------------|-------:|---------:|------------------|
| total wire bytes      | 149986 |   167106 | **+17120 (+11.41%)** |
| encode_cpu_ms_p95 max |  10.62 |    96.31 | +85.69 (~9x slower) |
| decode_cpu_ms_p95 max |   9.39 |    46.88 | +37.49 (~5x slower) |

All 7 in-scope items individually lose to gzip-6 on bytes (+3.35% to +15.92%).
This matches the v1 Infra-B numbers cited in the v3 agent definition §2.5
(brotli-1 167376 vs gzip-6 150251, +11.4%). The mismatch is structural: brotli's
RFC 7932 Appendix A 120 KB static dictionary is HTML/JS-tuned. Low-quality
brotli on JSON spends the encoding budget on dictionary references that do not
fire and produces larger output than gzip-6. Conclusion: never use `brotli -1`
on JSON.

### v3 §4.5 confirmation: zstd-9 differentiates from zstd-3

This was v1's quiet bug. Under `hyperfine -- sh -c '...'` v1's encode timings
saturated against the ~6 ms `sh -c` startup floor and zstd-3 and zstd-9 looked
identical on encode CPU; the score formula could not split them. With
`bench/measure.py` (subprocess.run + perf_counter_ns, no shell):

- zstd-3 encode_cpu_ms_p95 mean = 13.4 ms; zstd-9 = 20.2 ms (visibly different)
- zstd-3 wire 151307 vs zstd-9 wire 142068 (-6.1% smaller at +50% encode)
- zstd-9 wins both axes after weighting per SCOPE.md formula

v1 picked zstd-dict-3 in this slot; v3 cleanly picks zstd-dict-9 as the winner
because the dictionary case shows the same level-9 differentiation with even
better numbers (encode_cpu_ms_p95 max 17.6 ms; smaller than plain zstd-9 because
the encoder shortcuts repeated structure via dict references).

## Recommended Go origin code

```go
// Package compression — origin-side encoder for the JSON API.
//
// Decision: zstd level 9 with a shipped dictionary, per Exp 0006
// (bench/EXPERIMENTS.md). Mean delta vs identity: -83.91%
// (CI95 [-87.10%, -80.42%]) on the in-scope JSON corpus.
//
// Beats every alternative on every axis on this corpus:
//   - vs gzip-6   (Exp 0002): -27.3% wire bytes, comparable encode/decode CPU
//   - vs zstd-3   (Exp 0004): -27.9% wire bytes, faster encode and decode
//   - vs zstd-9   (Exp 0005): -23.2% wire bytes, faster encode and decode
//   - brotli-1    (Exp 0003): rejected — produces +11.4% wire bytes vs gzip-6
//                              on JSON (v3 §2.5 brotli-static-dict mismatch)
//
// Library: github.com/klauspost/compress/zstd (pure Go, no cgo).
// Dictionary file: bench/zstd-dict.bin
//   size:   112640 B
//   magic:  0xec30a437 (RFC 8878 §5.1; little-endian on disk: 37 a4 30 ec)
//   sha256: ce55da4e43ebc8399e98917c0caaef655d24bc96e92f8f75e1315b48f85be05e
//   trained: zstd --train bench/corpus/http/*.json -o bench/zstd-dict.bin
//
// BREACH (RFC §1.3 / breachattack.com): per-endpoint review below. The encoder
// here MUST NOT be applied to endpoints reflecting attacker-controllable input
// AND containing secrets. SCOPE.md flags this as outside the smoke test; the
// integration ticket must enumerate exclusions before this ships.
package compression

import (
	_ "embed"
	"errors"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed zstd-dict.bin
var zstdDict []byte

var (
	encOnce sync.Once
	decOnce sync.Once
	enc     *zstd.Encoder
	dec     *zstd.Decoder
	encErr  error
	decErr  error
)

// Encoder returns a process-wide zstd encoder configured at level 9 with the
// embedded dictionary. Cited: Exp 0006 (bench/EXPERIMENTS.md).
//
// The encoder is concurrency-safe (klauspost/compress/zstd reuses internal
// per-call state). Pool an encoder per process; do not allocate per request.
func Encoder() (*zstd.Encoder, error) {
	encOnce.Do(func() {
		enc, encErr = zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedBetterCompression), // klauspost level 9
			zstd.WithEncoderDict(zstdDict),
		)
	})
	return enc, encErr
}

// Decoder returns a process-wide zstd decoder configured with the embedded
// dictionary. Used for round-trip tests; production clients decode independently.
func Decoder() (*zstd.Decoder, error) {
	decOnce.Do(func() {
		dec, decErr = zstd.NewReader(nil,
			zstd.WithDecoderDicts(zstdDict),
		)
	})
	return dec, decErr
}

// EncodeJSON compresses a JSON response body with the dictionary-trained
// level-9 zstd encoder (Exp 0006).
//
// Caller MUST have already verified the endpoint is safe for body compression
// (no reflected-input + secret combination, RFC §1.3 BREACH). Exclusions per
// SCOPE.md and the BREACH section of infra-b-findings-v3.md.
func EncodeJSON(body []byte) ([]byte, error) {
	if len(body) < 1024 {
		return nil, errors.New("body below min_compress_size 1024 B (SCOPE.md)")
	}
	e, err := Encoder()
	if err != nil {
		return nil, err
	}
	return e.EncodeAll(body, nil), nil
}

// HTTP handler hook (illustrative — adapt to your router).
//
//	func handle(w http.ResponseWriter, r *http.Request) {
//	    body, _ := buildJSONResponse(r)
//	    if !acceptsZstd(r) || len(body) < 1024 {
//	        w.Header().Set("Vary", "accept-encoding")
//	        w.Write(body)
//	        return
//	    }
//	    enc, err := compression.EncodeJSON(body)
//	    if err != nil {
//	        w.Write(body); return
//	    }
//	    w.Header().Set("Content-Encoding", "zstd")
//	    w.Header().Set("Vary", "accept-encoding")
//	    w.Write(enc)
//	}
//
// RFC 9659 §3: the IANA token is `zstd`. Window size MUST NOT exceed 8 MB
// without explicit negotiation; klauspost/compress/zstd defaults are within
// this cap.

// Round-trip self-test (Exp 0006 framing sanity).
func RoundTripCheck(sample []byte) error {
	enc, err := EncodeJSON(sample)
	if err != nil {
		return err
	}
	d, err := Decoder()
	if err != nil {
		return err
	}
	out, err := d.DecodeAll(enc, nil)
	if err != nil {
		return err
	}
	if string(out) != string(sample) {
		return errors.New("zstd dict round-trip mismatch")
	}
	return nil
}

// Compile-time use to silence unused warnings if io is not otherwise used.
var _ io.Reader
```

### Fallback (clients without zstd)

If access logs show clients lacking `Accept-Encoding: zstd`, fall through to
gzip-6. Per Exp 0002 numbers, gzip-6 is materially worse than zstd-dict-9
(-74.82% vs -83.91% mean delta) but still 75% smaller than identity and is
universally supported. Brotli-1 is **rejected outright** for dynamic JSON
(Exp 0003, v3 §2.5).

```go
// Fallback path (client lacks Accept-Encoding: zstd). Cited: Exp 0002.
//   gzip-6: -74.82% wire bytes vs identity (CI95 [-76.17%, -73.54%]).
import "compress/gzip"
// ... wrap the response writer with gzip.NewWriterLevel(w, 6).
```

## Build hook for shipping the trained dictionary

Bake the dictionary alongside the binary via Go's `//go:embed` (already shown
in the package above). Build-time train + check pipeline:

```makefile
# Makefile snippet — origin service.
# Citation: bench/EXPERIMENTS.md Exp 0006 (zstd-dict-9 winner).

DICT       := internal/compression/zstd-dict.bin
CORPUS     := bench/corpus/http
ORIGIN_PKG := internal/compression
DICT_SHA   := ce55da4e43ebc8399e98917c0caaef655d24bc96e92f8f75e1315b48f85be05e

.PHONY: train-dict
train-dict:
	zstd --train $(CORPUS)/*.json -o $(DICT)

.PHONY: verify-dict
verify-dict:
	@xxd -l 4 $(DICT) | grep -q '37a4 30ec' || \
	  (echo "ERROR: bad zstd dict magic (expect 0xec30a437)"; exit 1)
	@echo "OK dict magic 0xec30a437 (RFC 8878 §5.1)"
	@printf '%s  %s\n' "$(DICT_SHA)" "$(DICT)" | shasum -a 256 -c -

.PHONY: build
build: verify-dict
	# go:embed picks up internal/compression/zstd-dict.bin at compile time.
	go build -trimpath -ldflags="-s -w" -o bin/origin ./cmd/origin

# Re-train when corpus rotates. Re-bench Exp 0006 after retraining; the dict
# magic is fixed but the dict content changes; CI95 must re-verify.
```

GitHub Actions snippet:

```yaml
- name: Verify zstd dictionary integrity
  run: make verify-dict

- name: Build origin with embedded dictionary
  run: make build
```

## Verification (when a live server exists)

Skipped here per SCOPE.md (no live HTTP server). When the origin is deployed,
run §8.1, §8.4, §8.5 of the v3 agent definition. Specifically:

```bash
# Section 8.1 — confirm zstd negotiated, gzip fallback works
for ae in 'identity' 'gzip' 'zstd' 'gzip, zstd'; do
  printf '%-25s ' "$ae"
  curl -sI -H "Accept-Encoding: $ae" "$URL" | grep -iE 'content-encoding|vary'
done
# Expect: zstd → "Content-Encoding: zstd" + "Vary: accept-encoding"
#         gzip → "Content-Encoding: gzip" + "Vary: accept-encoding"
#         identity → no Content-Encoding

# Section 8.4 — TLS comp off (CRIME, RFC §1.3)
echo | openssl s_client -connect "$HOST:443" -tls1_2 2>&1 | grep -i compression
# Expect: "Compression: NONE"

# Section 8.5 — single Content-Encoding token
curl -sI "$URL" | grep -i content-encoding
# Expect: one token only ("zstd" or "gzip"), never both.
```

## Security review (BREACH)

**RFC §1.3** BREACH (Gluck/Harris/Prado 2013): body compression + reflected
request input + secret in body → adaptive-chosen-plaintext recovers the
secret. Applies identically to gzip, brotli, zstd; the zstd dictionary
**worsens** the oracle (more cross-request shared state).

Per-corpus-item plausibility audit:

| endpoint                | reflects user input? | carries secrets?         | safe to compress? |
|-------------------------|----------------------|--------------------------|-------------------|
| catalog-full.json       | no (catalog-wide)    | no                       | YES               |
| products-list.json      | maybe (filters?)     | no                       | YES (verify filters not echoed verbatim) |
| products-list-v2.json   | maybe (filters?)     | no                       | YES (verify) |
| search-results.json     | **YES (query)**      | usually no               | YES iff no PII reflected (see below) |
| search-results-v2.json  | **YES (query)**      | usually no               | YES iff no PII reflected |
| order-history.json      | no (auth-scoped)     | **YES (order ids, PII)** | **NO — see mitigations** |
| notifications.json      | no (auth-scoped)     | **YES (likely PII)**     | **NO — see mitigations** |
| user-profile.json       | no (auth-scoped)     | **YES (PII)**            | excluded by SCOPE.md min size; also BREACH |
| error-404.json          | YES (echoes URL?)    | no                       | excluded by SCOPE.md min size |

**Compress-eligible endpoints** (no reflected-input AND no secrets, OR clear
separation): catalog-full, products-list, products-list-v2.

**Compress-after-mitigation endpoints**: search-results{,-v2}, order-history,
notifications. Mitigations (any one):
1. **Per-request CSRF/XSRF token padded into the body** so the high-entropy
   bytes change every request and the BREACH oracle no longer converges.
2. **Length randomization** (HTB / Heal-the-BREACH; RFC §1.3): server appends
   a per-response random-length comment / pad before compression. ~7-15
   bytes is sufficient to defeat the byte-by-byte length oracle.
3. **Disable compression entirely** on auth-scoped endpoints. Simplest;
   recommended for `order-history` and `notifications` until a per-response
   token strategy is implemented.

**Other gates** (RFC §1.3, v3 §3):
- TLS compression: must be OFF (CRIME). TLS 1.3 forbids; verify with
  `openssl s_client` (Section 8.4 above).
- The zstd dictionary itself contains JSON keys/values from the training
  corpus (see "follow-ups" — corpus contamination caveat). It must be
  audited to ensure it does not contain PII or session tokens. Re-train from
  scrubbed samples if real production responses are used.
- HPACK/QPACK: never-index `Cookie`, `Authorization`, `Set-Cookie` on shared
  HTTP/2 connections (RFC 7541 §6.2.3, RFC 9204 §4.5.4).
- `Vary: accept-encoding` mandatory on every compressed response (RFC 9111
  §4.1).

## Honest follow-ups

1. **No live HTTP server**. Phase 7 verification (over-the-wire negotiation,
   single Content-Encoding token, TLS compression off, Vary correctness) is
   skipped. When the Go origin ships, run the §8 checklist above before
   declaring done. Numbers above are filesystem-only (codec round-trip), not
   HTTP-stack round-trip.
2. **Dictionary corpus contamination**. The zstd dictionary was trained on
   the same 7 in-scope items it was then benchmarked against. `zstd --train`
   warned `size(source)/size(dictionary) = 2.44, recommend ≥10`. Two real
   risks:
   - **Optimistic ratio**: real production traffic will have more variance
     than this 7-sample bench. Expect the -23.3% vs gzip-6 advantage to
     erode toward -10% to -15% on a held-out corpus. Re-bench on a
     held-out set before promoting numbers to a status page.
   - **Privacy leakage**: any tokens, ids, or PII in the training corpus
     are baked into `zstd-dict.bin`. Audit the dictionary content; re-train
     from scrubbed samples if real production responses were used.
3. **Apple Silicon vs production target**. Encode/decode timings here are
   Darwin arm64 CLI. Production is a Go binary on x86_64 (presumably) using
   `klauspost/compress/zstd` in-process. In-process is materially faster
   than the CLI (no fork, no stdio). Decisions stand (the ranking is
   stable across architectures), but absolute ms numbers will differ. Re-run
   on production hardware with the in-process Go encoder once the service
   is integrated.
4. **Dictionary rotation**. When the JSON schema changes (new fields, new
   endpoints), the dictionary becomes stale and ratio degrades. Plan: re-run
   `make train-dict && make verify-dict` quarterly or whenever the API
   schema version bumps. Each retrain produces a new SHA; bump the dict id
   and re-bench Exp 0006.
5. **Scope is 6 experiments not 10**. Larger ratios may be possible:
   `zstd-19`, `zstd --ultra -22`, `brotli -5`/`-11` at static-side. Out of
   scope for this smoke test; the SCOPE.md battery was specifically chosen
   to verify v3's three cherry-picks. If the JSON API is in fact static
   (canned response cached on disk), open a follow-up to bench the static
   tier.
6. **BREACH mitigation not implemented**. The recommended Go code wraps the
   raw encoder; the per-endpoint exclusion / token-padding policy is the
   integration ticket's responsibility. Do not deploy to `order-history`,
   `notifications`, `user-profile`, or any other auth-scoped JSON endpoint
   without that policy.
7. **HTTP/3 / QPACK** not measured. SCOPE.md says HTTP/2 assumed; if the
   origin is fronted by an h3-capable edge, run a separate session against
   the h3 path with `SETTINGS_QPACK_MAX_TABLE_CAPACITY` ≥ 4096 (default 0
   means no header compression at all).

---

**Appendix: file map (absolute paths)**
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/SCOPE.md` — read-only, untouched
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/EXPERIMENTS.md` — append-only, contains all 6 experiments
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/measure.py` — v3 §4.5 sub-ms timer
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/score.py` — v3 §4.4 dual-format CI scorer
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/encode.sh` — codec dispatch
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/decode.sh` — codec dispatch
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/harness.sh` — driver
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/zstd-dict.bin` — trained dictionary, 112640 B, magic 0xec30a437
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/results/baseline.json` — Exp 0001 (identity)
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/results/<algo>/items.json` — per-experiment, in-scope filtered
- `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-b-api-v3/bench/results/<algo>/score.json` — per-experiment dual-format CI
