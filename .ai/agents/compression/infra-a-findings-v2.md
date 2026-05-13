# Compression Engineer — Infra A v2 (Static SPA)

Agent: `compression-engineer-v2` (10-phase loop, sub-ms timing via `bench/measure.py`,
local CPU calibration, manifest.json reproducibility).
Session: 2026-05-07.
Bench dir: `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-a-spa-v2/bench/`
Findings file: `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-a-findings-v2.md`

All claims below cite an experiment id in
`bench/EXPERIMENTS.md` whose CI95 of the score delta is strictly negative on the
SCOPE.md metric. Where citations are to RFCs or to the agent definition, the section
number is given.

---

## Discovery (Phase 0)

```
target_kind: filesystem-only
target_url:  (none — no live origin)
detected:    SPA static bundle, 9 corpus items, 795 354 raw bytes total
             stack per SCOPE.md: nginx + HTTP/2 + TLS 1.3 (assumed; mocked)
             encoding posture per SCOPE.md: production has Brotli + gzip; no zstd
             encode_cpu_ms weight = 0.0 (build-time encode is free)
             decode_cpu_ms weight = 0.5 (mobile decode penalised)
             client_profile: mobile_4g_slow
             budget_seconds_per_candidate: 30
```

No live HTTP server. Phase 7 (over-the-wire verification) is **skipped** per
SCOPE.md. All numbers below are filesystem-level encode/decode/size on the local
machine, not over-the-wire.

---

## Tooling check (Phase 0.5)

PRESENT (required, all required-loop tools available):
brotli 1.2.0, zstd 1.5.7, gzip (Apple gzip 479), openssl 3.6.2,
xxd, hyperfine 1.20.0, python3 3.14.2, curl.

PRESENT (optional): cwebp, ffmpeg, avifenc.

MISSING (optional) — install commands and blocked experiments:
- `svgo` — `npm i -g svgo` — blocks SVG-specific re-encoding (svg-minified-then-brotli).
  Mitigation taken: ran raw SVG through gzip/br/zstd as comparison; brotli-11 reduces
  `logo.svg` from 12 739 B to 2 607 B (-79.5%) before any svgo step.
- `oxipng`, `pngquant` — `brew install oxipng pngquant` — n/a (no PNG in corpus).
- `woff2_compress`, `pyftsubset` — n/a (no fonts in corpus).
- `oha`, `hey`, `wrk`, `h2load`, `nghttp`, `lighthouse` — n/a (filesystem-only;
  no live server to load-test or page-render).
- `cjpegli` — n/a (no JPEG in corpus).

No required tool is missing. Loop proceeded without halts.

---

## Calibration (Phase 0.6)

Sample: `bench/corpus/assets/vendor-a9b8c7d.js` (256 229 bytes).

| Algo      | Local p50 (ms) | Local MB/s | §2.1 table MB/s | Δ vs table |
|-----------|---------------:|-----------:|----------------:|-----------:|
| gzip-6    |          12.46 |       19.6 |              50 | -61% |
| gzip-9    |          19.02 |       12.8 |              30 | -57% |
| brotli-1  |           7.76 |       31.5 |             290 | -89% |
| brotli-5  |          11.17 |       21.9 |             100 | -78% |
| brotli-11 |         273.17 |       0.89 |             0.5 | +79% |
| zstd-3    |           8.50 |       28.7 |             250 | -89% |
| zstd-19   |          88.29 |        2.8 |              10 | -72% |

WARNING: every measured algorithm differs from §2.1 by >50%. Caveat: each
measurement includes Python `subprocess.run` fork/exec floor (~5 ms per invocation
on this Apple Silicon under filesystem sandbox), which dominates short jobs and
distorts per-call MB/s. The relative ordering is preserved (q=11 by far slowest;
zstd-3 and brotli-1 are the fastest beneath the floor).

Decision: trust the local numbers for ranking. SCOPE.md sets `encode_cpu_ms`
weight = 0.0, so encode CPU does not enter the score on this run; the calibration
divergence does not affect any KEEP decision below. (For future runs that weight
encode CPU > 0, replace the §2.1 table values with these local p50s.)

Full calibration JSON: `bench/results/calibration.json`.

---

## Inventory of corpus (Phase 1)

9 files, 795 354 bytes total. Asset classes: 2 HTML, 4 JS, 2 CSS, 1 SVG. Versioned
hashed-filename pairs (RFC 9842 dictionary candidates):
- `app-3e4f5a6.js` (81 928) ↔ `app-3e4f5a7.js` (84 041)
- `main-7f8a3c2.css` (30 739) ↔ `main-7f8a3c3.css` (31 749)
- `vendor-a9b8c7d.js` (256 229) ↔ `vendor-a9b8c7e.js` (261 255)

SHA-256 of each item is recorded in `bench/EXPERIMENTS.md` ("Corpus Inventory");
corpus Merkle hash is in `bench/manifest.json`.

---

## Baseline (Phase 2, identity passthrough)

Total wire bytes (identity, no compression): **795 354**.
Per-item bytes are recorded under `bench/results/baseline.json` in the strict
`items.json` schema (Section 5.5): `raw_bytes`, `wire_bytes`, `wire_bytes_p95`,
`encode_cpu_ms_{p50,p95,p99}`, `decode_cpu_ms_{p50,p95,p99}`, `raw_sha256`, `n_runs`.

---

## Experiments — summary table (10 experiments, all bench-slot results)

Bootstrap CI: 10 000 resamples, deterministic seed `0xC0FFEE`, alpha = 0.05.
KEEP iff `CI95_high_bytes < 0` on `score = wire_bytes_p95 + 0.0·enc + 0.5·dec`.
Where the score is dominated by wire_bytes (decode contributes <1 ms on every
item), the percent delta of score ≈ percent delta of wire bytes.

| Exp  | Algorithm      | Wire total | Δ bytes  | Δ percent | CI95 bytes              | CI95 percent          | Decision |
|------|----------------|-----------:|---------:|----------:|-------------------------|-----------------------|----------|
| 0001 | identity       |    795 354 |        0 |     0.00% | n/a (baseline)          | n/a                   | KEEP (ref) |
| 0002 | gzip-6         |    163 804 |  -70 172 |   -78.52% | [-124 268, -25 720]     | [-80.66%, -76.34%]    | KEEP |
| 0003 | gzip-9         |    162 361 |  -70 332 |   -78.76% | [-124 543, -25 827]     | [-80.81%, -76.69%]    | KEEP |
| 0004 | brotli-5       |    182 161 |  -68 132 |   -77.96% | [-120 110, -25 504]     | [-80.49%, -75.95%]    | KEEP |
| 0005 | brotli-11      |    142 836 |  -72 501 |   -82.07% | [-128 215, -26 774]     | [-83.88%, -80.57%]    | **KEEP — single-algo winner** |
| 0006 | zstd-3         |    181 538 |  -68 201 |   -77.07% | [-120 558, -25 237]     | [-79.25%, -75.10%]    | KEEP |
| 0007 | zstd-19        |    144 954 |  -72 266 |   -81.06% | [-127 996, -26 461]     | [-82.37%, -79.76%]    | KEEP |
| 0008 | brotli-dict-11 |     66 877†|  -103 388†|  -81.65%†| [-215 718, -25 513]†    | [-82.57%, -80.35%]†   | KEEP (vs identity) / KEEP (-2.82% vs non-dict brotli-11) |
| 0009 | zstd-dict-19   |     63 715†|  -104 442†|  -82.01%†| [-218 509, -25 301]†    | [-83.64%, -79.68%]†   | **KEEP — versioned-pair winner** (-6.74% vs non-dict zstd-19) |
| 0010 | brotli-8       |    160 327 |  -70 558 |   -79.59% | [-124 705, -26 101]     | [-81.74%, -77.65%]    | KEEP (DISCARD vs brotli-11 for static use) |

† Exp 0008 and 0009 cover the 3 versioned newer-files only (`app-3e4f5a7.js`,
`main-7f8a3c3.css`, `vendor-a9b8c7e.js`); compared against an identity sub-baseline
of those same 3 files (sub-baseline total = 377 045 bytes). Cross-comparison vs
the matching non-dict baselines is logged in `score_vs_brotli11.json` (Exp 0008,
-2.82% vs Exp 0005 subset) and `score_vs_zstd19.json` (Exp 0009, -6.74% vs Exp 0007
subset).

Three additional candidates were logged as `DISCARD-BY-PREDICTION` (no bench slot
consumed) per agent §3 and §5.6 Template B:

- DBP-A: gzip on already-compressed types (§3 #2). N/a in this corpus, recorded as
  config invariant.
- DBP-B: brotli q=11 on dynamic responses (§3 #1). Calibration confirms 273 ms p50
  per 256 KB, vs zstd-3 at 8.5 ms.
- DBP-C: compression without `Vary: accept-encoding` (§3 #3). Cache poisoning;
  recorded as config invariant per RFC 9111 §4.1.

Full per-experiment text and citations: `bench/EXPERIMENTS.md`.

### Per-asset winner

brotli-11 (Exp 0005) wins **every** asset class on a per-item basis. No mixed
per-class config can beat the single-algorithm choice on this corpus.

| Item              | Best       | Wire bytes |   vs raw |
|-------------------|------------|-----------:|---------:|
| about.html        | brotli-11  |      3 026 |  -84.7%  |
| index.html        | brotli-11  |      2 051 |  -87.8%  |
| logo.svg          | brotli-11  |      2 607 |  -79.5%  |
| main-7f8a3c2.css  | brotli-11  |      6 234 |  -79.7%  |
| main-7f8a3c3.css  | brotli-11  |      6 484 |  -79.6%  |
| app-3e4f5a6.js    | brotli-11  |     15 381 |  -81.2%  |
| app-3e4f5a7.js    | brotli-11  |     15 626 |  -81.4%  |
| vendor-a9b8c7d.js | brotli-11  |     45 282 |  -82.3%  |
| vendor-a9b8c7e.js | brotli-11  |     46 145 |  -82.3%  |

For the 3 versioned newer files, applying the older sibling as an RFC 9842 shared
dictionary takes the wire down further:
- `vendor-a9b8c7e.js`: 46 145 → 42 745 (zstd-dict-19, -7.4%) / → 45 536 (brotli-dict-11, -1.3%)
- `main-7f8a3c3.css`:  6 484 →  6 447 (zstd-dict-19, -0.6%) /  →  6 235 (brotli-dict-11, -3.8%)
- `app-3e4f5a7.js`:   15 626 → 14 523 (zstd-dict-19, -7.1%) / → 15 106 (brotli-dict-11, -3.3%)

Aggregate over the 3 newer files: zstd-dict-19 yields the smallest wire total
(63 715 B) vs brotli-dict-11 (66 877 B) — zstd's dictionary mode does more work
because zstd has no built-in static dictionary, while brotli already includes its
RFC 7932 Appendix A 120 KB HTML/JS dictionary.

### Min-size cutoff (v2 mandatory)

SCOPE.md does not declare an explicit `min_compress_size`. Default per agent §2.4
is 1024 B. Smallest corpus item is `logo.svg` at 12 739 B — well above the
threshold. **No corpus item is gated by the cutoff in this session.** Even the
smallest item (logo.svg) compresses cleanly under brotli-11 (12 739 → 2 607 B,
-79.5%).

---

## Recommended configuration

Cite Exp 0005 + 0007 + 0009 inline. The recommended deployment serves three
encodings, picked by `Accept-Encoding` (per RFC 9110 §8.4.1):
1. `br` (brotli-11, build-time): primary text encoding for modern browsers.
2. `zstd` (zstd-19, build-time): served when the client advertises `zstd` in
   `Accept-Encoding` (RFC 9659).
3. `dcz` (zstd-dict-19, RFC 9842, framed payload): served when the client also
   advertises a usable `Available-Dictionary` for a hashed-bundle prior version.
4. Plus `gzip` (gzip-6) as a universal fallback.

### nginx configuration

```nginx
# === Infra A v2 — recommended static-asset compression (nginx) ===
# Citations:
#   Exp 0005 brotli-11:  -82.07% wire bytes vs identity, CI95 [-83.88%, -80.57%]
#   Exp 0007 zstd-19:    -81.06% wire bytes vs identity, CI95 [-82.37%, -79.76%]
#   Exp 0002 gzip-6:     -78.52% wire bytes vs identity, CI95 [-80.66%, -76.34%]
#   Exp 0009 zstd-dct19: -82.01% vs identity (subset), -6.74% vs non-dict zstd-19
#   DBP-A: never compress already-compressed types (image/png, jpg, webp, avif, woff2,
#          mp4, zip, gz, br, zst). compression-engineer §3 #2.
#   DBP-C: always Vary: Accept-Encoding when content-encoded. RFC 9111 §4.1.
# Module choice:
#   ngx_brotli (github.com/google/ngx_brotli) for brotli; tokers/zstd-nginx-module
#   or alternative third-party build for zstd. Verify with `nginx -V | tr ' ' '\n' | grep -E 'brotli|zstd'`.

# Pre-compressed serving (preferred; build emits .br and .zst alongside originals)
brotli_static      on;
zstd_static        on;
gzip_static        on;

# Runtime fallback for dynamically-generated bodies (none in this scope, but harmless)
brotli             on;
brotli_comp_level  5;          # not q=11 at runtime — DBP-B antipattern
brotli_min_length  1024;       # agent §2.4 default
brotli_window      4m;         # default lgwin=22; vendor.js 256 KB fits

zstd               on;
zstd_comp_level    3;          # default for runtime; -19 only at build (Exp 0007)
zstd_min_length    1024;

gzip               on;
gzip_vary          on;
gzip_comp_level    6;          # Exp 0002 vs Exp 0003: q=9 saves only 0.9% at 1.3x cost
gzip_min_length    1024;
gzip_proxied       expired no-cache no-store private auth;
gzip_http_version  1.1;
gzip_disable       "msie6";

# Types eligible for compression. Strict allowlist; never list image/* video/*
# font/woff2 application/zip etc. (DBP-A). Cite agent §3 #2.
brotli_types
    text/plain text/css text/xml text/html
    application/json application/javascript application/xml application/xml+rss
    application/wasm
    image/svg+xml
    font/ttf font/otf;
zstd_types
    text/plain text/css text/xml text/html
    application/json application/javascript application/xml application/xml+rss
    application/wasm
    image/svg+xml
    font/ttf font/otf;
gzip_types
    text/plain text/css text/xml text/html
    application/json application/javascript application/xml application/xml+rss
    application/wasm
    image/svg+xml
    font/ttf font/otf;

# Hashed static assets: long-cache + immutable. Vary mandatory when content-negotiated.
# Cite RFC 9111 §4.1 (Vary), agent §6.4 (dictionary block below).
location ~* "^/assets/.*\.[a-f0-9]{7,}\.(js|css)$" {
    add_header Cache-Control "public, max-age=31536000, immutable" always;
    add_header Vary          "Accept-Encoding"                     always;
    expires    1y;
    access_log off;
}

# === RFC 9842 shared compression dictionary (dcz) — Exp 0009 winner ===
# Pre-computed payloads at deploy time; never compress at request time (DoS).
# Cite RFC 9842 §2.2 (dcz framing), §2.2.1 (UA precedence), agent §5.4.2.
#
# Use-As-Dictionary on the older versioned bundle (declares it as a dict candidate):
location = /assets/vendor-a9b8c7d.js {
    add_header Use-As-Dictionary 'match="/assets/vendor-*.js", match-dest=("script"), id="vendor-bundle"' always;
    add_header Cache-Control     "public, max-age=31536000, immutable" always;
    add_header Vary              "Accept-Encoding, Available-Dictionary" always;
}

# Older->newer dictionary serving for the new bundle:
map $http_available_dictionary $dict_hash {
    default                   "";
    "~*^:([A-Za-z0-9+/=]+):$" "$1";
}
map $http_accept_encoding $picked_dict_enc {
    default      "";
    "~*\bdcz\b"  "dcz";
    "~*\bdcb\b"  "dcb";
}
location ~ ^/assets/(vendor|app|main)-[a-f0-9]{7,}\.(js|css)$ {
    if ($dict_hash != "") {
        if ($picked_dict_enc != "") {
            rewrite ^ /precompressed/$dict_hash/$picked_dict_enc$uri last;
        }
    }
    add_header Vary "Accept-Encoding, Available-Dictionary" always;
}
location /precompressed/ {
    internal;
    add_header Cache-Control "public, max-age=31536000, immutable" always;
    add_header Vary          "Accept-Encoding, Available-Dictionary" always;
    types {
        application/javascript js;
        text/css               css;
    }
    # The Content-Encoding header is set per-encoding by the layer that produces the
    # internal subrequest; ensure it reaches the client unmodified.
}
```

Key invariants enforced above:
- `Vary: Accept-Encoding` whenever any content-encoded response exists (RFC 9111 §4.1; DBP-C).
- `Vary: Accept-Encoding, Available-Dictionary` when serving `dcb`/`dcz` (RFC 9842 §3).
- `match` is same-origin and contains no regex (RFC 9842 §2.2.1).
- Pre-computed payloads only; no request-time encoding against attacker-influenced
  dictionary (RFC 9842 §6.1 DoS guidance).

### Cloudflare Workers configuration

```js
// === Infra A v2 — recommended static-asset compression (Cloudflare Workers) ===
// Citations:
//   Exp 0005 brotli-11:  -82.07% wire vs identity. Pre-computed at build, served via
//                        env.ASSETS as Content-Encoding: br.
//   Exp 0007 zstd-19:    -81.06% wire vs identity. Pre-computed at build,
//                        served as Content-Encoding: zstd.
//   Exp 0009 zstd-dct19: -82.01% vs identity (subset), -6.74% vs non-dict zstd-19.
//                        Served as Content-Encoding: dcz when client supplies
//                        Available-Dictionary for a known hash.
//   DBP-A/B/C: see EXPERIMENTS.md.
export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const ae  = (req.headers.get("Accept-Encoding") || "").toLowerCase();
    const dictHashHeader = req.headers.get("Available-Dictionary");

    // 1. RFC 9842 dictionary path (Exp 0009 winner). Hashed-bundle paths only.
    if (dictHashHeader && /\/assets\/(vendor|app|main)-[a-f0-9]{7,}\.(js|css)$/.test(url.pathname)) {
      const wantsDcz = ae.includes("dcz");
      const wantsDcb = ae.includes("dcb");
      const enc      = wantsDcz ? "dcz" : (wantsDcb ? "dcb" : "");
      if (enc) {
        const hash = dictHashHeader.replace(/^:|:$/g, "").replace(/\//g, "_");
        const key  = `precompressed/${hash}/${enc}${url.pathname}`;
        const obj  = await env.ASSETS_KV.get(key, { type: "arrayBuffer" });
        if (obj) {
          return new Response(obj, {
            headers: {
              "content-encoding": enc,
              "content-type":     contentTypeFor(url.pathname),
              "cache-control":    "public, max-age=31536000, immutable",
              "vary":             "Accept-Encoding, Available-Dictionary",
            },
          });
        }
      }
    }

    // 2. Standard pre-compressed serving by Accept-Encoding negotiation.
    //    Workers Sites picks .br / .zst variants; we annotate Vary correctly.
    //    (env.ASSETS handles content-negotiation when files are provisioned with .br/.zst.)
    const resp = await env.ASSETS.fetch(req);
    const out  = new Response(resp.body, resp);
    out.headers.set("Vary", "Accept-Encoding, Available-Dictionary");
    if (/\/assets\/.*\.[a-f0-9]{7,}\.(js|css|svg|html)$/.test(url.pathname)) {
      out.headers.set("Cache-Control", "public, max-age=31536000, immutable");
    }
    // Use-As-Dictionary on the explicit dictionary candidate (older sibling).
    if (url.pathname === "/assets/vendor-a9b8c7d.js") {
      out.headers.set("Use-As-Dictionary",
        'match="/assets/vendor-*.js", match-dest=("script"), id="vendor-bundle"');
    }
    return out;
  },
};

function contentTypeFor(p) {
  if (p.endsWith(".js"))   return "application/javascript; charset=utf-8";
  if (p.endsWith(".css"))  return "text/css; charset=utf-8";
  if (p.endsWith(".svg"))  return "image/svg+xml";
  if (p.endsWith(".html")) return "text/html; charset=utf-8";
  if (p.endsWith(".wasm")) return "application/wasm";
  return "application/octet-stream";
}
```

`wrangler.toml` should bind `ASSETS` (Workers Sites) and a separate `ASSETS_KV`
namespace populated at deploy time with the `precompressed/<hash>/<enc>/<path>`
keys produced by the build hook below.

---

## Build / Deploy hook

Two variants. Pick the one matching the project's existing toolchain.

### Make variant

```makefile
DIST     := dist
ASSETS   := $(shell find $(DIST) -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' -o -name '*.wasm' -o -name '*.json' \))
DICTSRC  := $(DIST)/assets/vendor-a9b8c7d.js   # the older versioned dict (example)
DICTHASH := $(shell openssl dgst -sha256 -binary $(DICTSRC) | base64)

compress: $(ASSETS:%=%.br) $(ASSETS:%=%.zst) $(ASSETS:%=%.gz) precompressed-dicts

%.br:  %
	brotli -q 11 -k -f $<        # Exp 0005: -82.07% wire vs identity
%.zst: %
	zstd  -19 -q -k -f $<        # Exp 0007: -81.06% wire vs identity
%.gz:  %
	gzip  -9 -k -f $<            # Exp 0003 fallback for ancient clients

precompressed-dicts:
	@for newer in dist/assets/vendor-*.js dist/assets/app-*.js dist/assets/main-*.css; do \
	    [ "$$newer" = "$(DICTSRC)" ] && continue; \
	    base=$$(basename $$newer); \
	    mkdir -p dist/precompressed/$(DICTHASH)/dcz/assets dist/precompressed/$(DICTHASH)/dcb/assets; \
	    zstd -19 -q -D $(DICTSRC) -c $$newer > dist/precompressed/$(DICTHASH)/dcz/assets/$$base; \
	    brotli -q 11 -D $(DICTSRC) -c $$newer > dist/precompressed/$(DICTHASH)/dcb/assets/$$base; \
	done

.PHONY: compress precompressed-dicts
```

### npm script variant

```json
{
  "scripts": {
    "build:compress": "node ./scripts/compress.mjs"
  }
}
```

```js
// scripts/compress.mjs — runs after `vite build`/`webpack`
// Exp 0005 brotli-11, Exp 0007 zstd-19, Exp 0003 gzip-9, Exp 0009 zstd-dict-19
import { execFileSync } from "node:child_process";
import { readdirSync, statSync, mkdirSync, existsSync } from "node:fs";
import { join, basename } from "node:path";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";

const DIST = "dist";
const TEXT = /\.(js|css|html|svg|json|wasm)$/;
const walk = (d) => readdirSync(d).flatMap(n => {
  const p = join(d, n);
  return statSync(p).isDirectory() ? walk(p) : [p];
});

const files = walk(DIST).filter(f => TEXT.test(f) && statSync(f).size >= 1024);
for (const f of files) {
  execFileSync("brotli", ["-q", "11", "-k", "-f", f]);   // Exp 0005
  execFileSync("zstd",   ["-19", "-q", "-k", "-f", f]);  // Exp 0007
  execFileSync("gzip",   ["-9", "-k", "-f", f]);         // Exp 0003 fallback
}

// RFC 9842 shared-dict pre-computation (Exp 0009 winner)
const DICT = "dist/assets/vendor-a9b8c7d.js";  // older sibling
if (existsSync(DICT)) {
  const dictBytes = readFileSync(DICT);
  const sha = createHash("sha256").update(dictBytes).digest("base64");
  for (const newer of walk("dist/assets").filter(f =>
       /(?:vendor|app|main)-[a-f0-9]{7,}\.(js|css)$/.test(f) && f !== DICT)) {
    const dczDir = `dist/precompressed/${sha}/dcz/${basename(newer)}`;
    const dcbDir = `dist/precompressed/${sha}/dcb/${basename(newer)}`;
    mkdirSync(`dist/precompressed/${sha}/dcz`, { recursive: true });
    mkdirSync(`dist/precompressed/${sha}/dcb`, { recursive: true });
    execFileSync("sh", ["-c", `zstd -19 -q -D ${DICT} -c ${newer} > ${dczDir}`]);
    execFileSync("sh", ["-c", `brotli -q 11 -D ${DICT} -c ${newer} > ${dcbDir}`]);
  }
  console.log(`Pre-computed dcb/dcz against dict ${DICT} (sha256=${sha.slice(0,8)}...)`);
}
```

In a CI step, add the `dcb`/`dcz` framing prefix per RFC 9842 (4-byte magic
`ff 44 43 42` for `dcb`, 8-byte magic `5e 2a 4d 18 20 00 00 00` for `dcz`,
followed by 32-byte raw SHA-256 of the dict). The agent has verified the framing
on the corpus output: `bench/results/0008/framed/vendor.dcb` and
`bench/results/0009/framed/vendor.dcz` carry the correct headers (see Exp 0008 /
Exp 0009 entries in `EXPERIMENTS.md`).

---

## Security review checklist

Static-asset corpus, no secrets, no reflected request input. Several attacks
trivially do not apply, but the discipline of stating that explicitly is part of
the v2 deliverable.

- **BREACH (Gluck/Harris/Prado, 2013):** **n/a.** The corpus contains no secret
  bearer token, session cookie, CSRF token, or PII; nothing the response body
  reflects from the request. SCOPE.md `exclusions: []` is correct. If
  user-personalized HTML is ever introduced (e.g. server-rendered profile pages),
  add those endpoints to `exclusions` and revisit this checklist before enabling
  body compression there.
- **CRIME (RFC 8446 §1.2 deprecation rationale):** TLS-level compression is
  forbidden by TLS 1.3 and disabled by `SSL_OP_NO_COMPRESSION` on older versions.
  Verify in production with:
  ```bash
  echo | openssl s_client -connect <host>:443 -tls1_2 2>&1 | grep -i compression
  # Expect: "Compression: NONE" (or absent under TLS 1.3 entirely)
  ```
  Cite agent §1.3 / §8.4.
- **HEIST (BlackHat 2016):** TCP/TLS timing side channel. Mitigated by SOP-respecting
  fetch boundaries in modern browsers. Static, non-credential bundles are not
  attacker-targeted via HEIST (no per-user secret to extract).
- **Shared dictionary safety (RFC 9842 §6):**
  - HTTPS only — confirmed by deployment (TLS 1.3 termination at edge per SCOPE.md).
  - Same-origin `match` — enforced in nginx config: `match="/assets/vendor-*.js"`
    is path-only, no scheme/host (RFC 9842 §2.2.1).
  - No secrets in the dict — these are public hashed bundles, the same JS the
    browser would parse anyway. No attacker advantage over a plain `<script>` fetch.
  - `Vary: Accept-Encoding, Available-Dictionary` set on every dict-candidate and
    dict-encoded response. Cache poisoning resistance per RFC 9111 §4.1 + RFC 9842 §3.
  - Brotli quality ≥ 5 / Zstandard level ≥ 3 — Exp 0008 uses q=11, Exp 0009 uses
    level 19. Within agent §5.4.2 invariant.
  - Pre-computed payloads at build, never request-time. DoS-safe (RFC 9842 §6.1).
- **HPACK / QPACK header oracles:** Static origin sets no `Set-Cookie` or
  `Authorization` for these assets, so the never-indexed bit is moot here. If a
  shared HTTP/2 connection is ever used for cross-tenant or auth-bearing requests,
  set `Cookie` and `Authorization` to literal-not-indexed (HPACK 0001 prefix per
  RFC 7541 §6.2.3, QPACK literal-not-indexed per RFC 9204 §4.5).
- **WebSocket permessage-deflate (RFC 7692):** **n/a.** No WebSocket in scope.
- **Range requests on compressed responses:** static assets are served with full
  bodies; no `Range`-against-encoded combinations issued in this scope. If the
  origin starts honoring `Range` on encoded responses, refuse range OR serve
  identity for ranges per agent §3 #10 (RFC 9110 undefined behavior).
- **Already-compressed types (DBP-A, agent §3 #2):** the explicit `*_types`
  allowlist in the nginx and Workers configs above excludes
  `image/{png,jpeg,webp,avif}`, `font/woff2`, `video/*`, `application/zip`. Verified
  by inspection.

---

## Honest follow-ups (what the run could not measure)

1. **No live server, no over-the-wire numbers.** Phase 7 is skipped per SCOPE.md
   `target_kind: filesystem-only`. All wire-byte numbers are filesystem
   `wc -c` of encoder output, not response bodies under TLS. **What would change:**
   on the wire, gzip and brotli also pay TLS record overhead (~30 B per record),
   HTTP/2 framing (9 B per HEADERS + ~9 B per DATA), and any front-of-line
   blocking observed at the client. None of these change the *relative* ordering,
   but they do compress the absolute ratio gap between algorithms by a few percent.
   To measure: run `oha -n 2000 -c 50 -H 'Accept-Encoding: br'` on the deployed
   nginx, compare `size_download` vs `Accept-Encoding: identity` baseline.
2. **Calibration divergence not isolated.** The §2.1 table comparison includes
   Python `subprocess.run` fork/exec overhead (~5 ms on this Apple Silicon under
   filesystem sandbox), inflating the reported per-call latency for fast encoders
   like zstd-3 and brotli-1. The relative ordering and brotli-11 / zstd-19 absolute
   numbers (where encode work dominates the floor) are reliable. **What would
   change:** isolating the floor would require a single-process bench (e.g. a Go
   binary that imports libzstd / libbrotli directly and calls in-process), bypassing
   subprocess overhead. Not pursued because SCOPE.md sets `encode_cpu_ms` weight = 0.0.
3. **`svgo` not available.** `logo.svg` was compressed as raw XML through
   brotli-11 (-79.5%). With `svgo --multipass`, the raw SVG typically shrinks
   30-60% before any compression layer; the post-brotli total would be smaller
   still. Install `npm i -g svgo` and re-run the SVG-only experiment to quantify.
4. **Synthetic versioned pairs.** The `app-3e4f5a6.js` / `app-3e4f5a7.js` etc.
   pairs differ more than a typical micro-deploy diff (the agent's calibration
   table cites 30-60% wins on hashed bundle minor revisions). Real production
   deploys with frequent minor releases would see proportionally larger
   shared-dictionary wins than the -2.82% / -6.74% measured here.
5. **No client-mix data.** SCOPE.md asserts `client_profile: mobile_4g_slow`, but
   without an access log we cannot quantify what fraction of traffic actually
   advertises `Accept-Encoding: zstd` vs `br` vs `gzip`. The recommended config
   serves all four (`br`, `zstd`, `gzip`, `dcz`) so any negotiation outcome is
   covered, but the deployment cost (storage, Workers KV puts) scales with the
   number of variants × number of assets. If access logs show ≥99% `br`-capable
   clients, dropping `gzip` saves a third of the storage at no measured cost.
6. **HTTP/2 vs HTTP/3 header compression.** No HPACK / QPACK measurement performed
   (no live server to introspect). Static assets have small, repeatable headers
   so the table-size choice barely moves the metric, but it's the kind of audit
   that happens in Phase 7. Re-bench once an origin is up.

---

## Run discipline notes

- target_kind = `filesystem-only` declared explicitly in `bench/EXPERIMENTS.md`
  Discovery block before any baseline run, per agent §5.1 v2 rule.
- Sub-millisecond timings produced by `bench/measure.py` (Section 4.5), not
  `hyperfine -- sh -c`. `hyperfine` was used only for the calibration micro-bench
  (Section 5.1.6) where per-iteration cost was already ≥5 ms.
- All antipatterns logged with `DISCARD-BY-PREDICTION` (Template B, §5.6) and a
  §3 citation; no bench slot consumed for known-bad candidates.
- `bench/manifest.json` written before exit with corpus SHA-256, scope SHA-256,
  scoring seed, tool versions, calibration summary.
- Findings file path computed via Section 11.1 (no git root → fallback to parent
  of `bench/`). Resolved path printed before and after writing:
  `/Users/noelruault/go/src/github.com/noelruault/compression-test/infra-a-findings-v2.md`.

---

## Files

| Path | Purpose |
|---|---|
| `bench/SCOPE.md` | Session metric, weights, exclusions, target_kind. (read-only this session) |
| `bench/EXPERIMENTS.md` | Append-only experiment log; 10 bench-slot entries + 3 DBP entries. |
| `bench/manifest.json` | v2 reproducibility manifest (agent, tool versions, OS/CPU, hashes, seed). |
| `bench/measure.py` | Canonical sub-ms encoder timer. |
| `bench/score.py` | Bootstrap-CI scorer (10 000 resamples, seed `0xC0FFEE`). |
| `bench/encode.sh`, `decode.sh`, `harness.sh` | Algorithm dispatch + bench wrapper. |
| `bench/results/baseline.json` | Identity baseline per strict items.json schema. |
| `bench/results/calibration.json` | Phase 0.6 local-CPU sanity bench. |
| `bench/results/{0001..0010}/items.json` | Per-experiment per-item measurements. |
| `bench/results/{0001..0010}/score.json` | Per-experiment bootstrap CI scores. |
| `bench/results/0008/framed/vendor.dcb`, `bench/results/0009/framed/vendor.dcz` | Magic-byte-verified dictionary payloads (RFC 9842). |
