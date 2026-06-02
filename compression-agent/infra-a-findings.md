# Compression Engineer — Infra A: Static SPA

Report layout per agent definition §11. Every claim cites an experiment id from
`infra-a-spa/bench/EXPERIMENTS.md` or an RFC / agent-definition section. Numbers come
from `bench/results/` and `bench/score.py` (10,000-resample bootstrap CI95, deterministic
seed `0xC0FFEE`, alpha=0.05, n=9). Score weights (per `bench/SCOPE.md`):

```
score = wire_bytes_p95 + 0.0 * encode_cpu_ms_p95 + 0.5 * decode_cpu_ms_p95
```

Encode CPU is build-time (free); decode CPU is charged at 0.5 because mobile clients
pay it (agent §4.1).

## Discovery

- Target: static single-page application served from CDN
- Stack (per `SCOPE.md`): nginx + (Cloudflare Workers as edge alternative). HTTP/2,
  TLS 1.3 assumed. Production currently runs Brotli + gzip, no zstd at edge.
- Asset classes in scope: HTML (2), JS (4 — two app pairs, two vendor pairs), CSS (2),
  SVG (1). Versioned filenames with content hashes; behind `Cache-Control: max-age=31536000, immutable`.
- Client mix: not measured (no live server). Recommendations assume modern-browser
  baseline (`br` and `zstd` widely supported per agent §2.1) with `gzip` fallback.
- Live server: not running. Phase 7 (over-the-wire verification) skipped per task brief.

## Corpus

`bench/corpus/assets/` — 9 items, 795,354 raw bytes total.

| File                  |   Bytes | Class |
|-----------------------|--------:|-------|
| index.html            |  16,874 | HTML  |
| about.html            |  19,800 | HTML  |
| logo.svg              |  12,739 | SVG   |
| main-7f8a3c2.css      |  30,739 | CSS (older) |
| main-7f8a3c3.css      |  31,749 | CSS (newer) |
| app-3e4f5a6.js        |  81,928 | JS (older app) |
| app-3e4f5a7.js        |  84,041 | JS (newer app) |
| vendor-a9b8c7d.js     | 256,229 | JS (older vendor) |
| vendor-a9b8c7e.js     | 261,255 | JS (newer vendor) |
| **total**             | **795,354** | |

Versioned pairs (older→newer): `app-3e4f5a6→7`, `main-7f8a3c2→3`, `vendor-a9b8c7d→e`.
These are the candidate set for shared compression dictionary (RFC 9842) experiments.
Corpus pre-populated; not regenerated this session.

## Baseline (`bench/results/baseline.json`)

Identity (no compression) — Exp 0001:
- wire_bytes_p95 (per file): equal to raw size (table above).
- encode_cpu_ms_p95: 0 (no encoding).
- decode_cpu_ms_p95: 0 (no decoding).
- aggregate score: **795,354** (sum of raw sizes, decode CPU = 0).

The realistic production starting point per `SCOPE.md` is "Brotli + gzip" already
running, but the SPA's exact runtime quality level is not specified. The agent measures
vs identity to surface absolute ratios; in practice the incremental win from this
report comes from raising static brotli to q=11 (Exp 0005) and adding shared
dictionaries on the versioned bundle paths (Exp 0008, 0009, 0010).

## Experiments

Full table (all 10 experiments are KEEP vs identity baseline; CI95 strictly negative
in every row). See `bench/EXPERIMENTS.md` for hypothesis, command, and per-file
breakdown of each row.

| Exp  | Candidate           | Encoded total |  Δ vs identity | CI95 (mean delta)       | Decision (vs identity) |
|------|---------------------|--------------:|---------------:|------------------------:|:-----------------------|
| 0001 | identity (baseline) |       795,354 |          0.0% | (baseline)              | BASELINE               |
| 0002 | gzip -6             |       163,935 |        -79.4% | [-124,244 ; -25,701]    | KEEP                   |
| 0003 | gzip -9             |       162,492 |        -79.6% | [-124,518 ; -25,808]    | KEEP                   |
| 0004 | brotli -q 5         |       182,133 |        -77.1% | [-120,102 ; -25,501]    | KEEP                   |
| 0005 | brotli -q 11        |       142,839 |        -82.0% | [-128,208 ; -26,767]    | KEEP — CHAMPION        |
| 0006 | zstd -3             |       182,477 |        -77.1% | [-120,292 ; -25,211]    | KEEP                   |
| 0007 | zstd -19            |       144,972 |        -81.8% | [-127,987 ; -26,453]    | KEEP                   |
| 0008 | brotli-dict-11 (dcb) |      141,570 |        -82.2% | [-128,468 ; -26,825]    | KEEP                   |
| 0009 | zstd-dict-19   (dcz) |      140,254 |        -82.4% | [-129,275 ; -26,591]    | KEEP                   |
| 0010 | hybrid (dcb+br11)   |       141,570 |        -82.2% | [-128,465 ; -26,821]    | KEEP — recommended     |

Champion comparisons (vs Exp 0005 brotli-11):
- Exp 0008 brotli-dict-11 vs brotli-11: Δ -1,313 bytes (-0.92%), CI95 [-302.9, -5.6] → **KEEP**.
  On the 3 dict-matched files only: Δ -1,282 bytes, CI95 [-577 ; -218] → KEEP.
- Exp 0009 zstd-dict-19 vs brotli-11: Δ -2,628 bytes (-1.84%), CI95 [-1,134 ; +283] → DISCARD as universal winner; high variance from one large file. Per-file dict savings 4–7%. Keep as the `dcz`-format option.
- Exp 0010 hybrid vs brotli-11: Δ -1,282 bytes (-0.90%), CI95 [-301 ; 0.0] → DISCARD by the strict `<0` rule (touches zero), but the per-file dict win in Exp 0008 IS strictly negative. The hybrid models real production: dict path used only when the client's `Available-Dictionary` matches.

Modest dict savings (1–7%) are explained by the corpus: brotli-11's 4 MiB sliding
window plus the 120 KB static dictionary (RFC 7932 App. A, agent §2.2) already
captures most of the repetition between near-clone bundles. Real-world bundle pairs
with deeper diffs see 30–60% (agent §9.3); the framework here is correct, the savings
will compound on real diffs.

## Recommended Configuration

Pre-compress at build (encode CPU is free per `SCOPE.md`); serve via static-encoded
file delivery; emit `dcb`/`dcz` on the versioned bundle paths.

Universally serve:
- `*.br` (brotli q=11) for `Accept-Encoding: br` (Exp 0005).
- `*.zst` (zstd -19) for `Accept-Encoding: zstd` (Exp 0007).
- `*.gz` (gzip -9) for legacy fallback only (Exp 0003).

For the 3 versioned bundle paths, additionally pre-compute:
- `*.dcb` (brotli q=11 with prior version as dict) for `Accept-Encoding: dcb` (Exp 0008).
- `*.dcz` (zstd -19 with prior version as dict) for `Accept-Encoding: dcz` (Exp 0009).

### nginx config (paste-ready)

```nginx
# infra-a SPA — static asset block.
# All numbers from infra-a-findings.md / bench/EXPERIMENTS.md.
# Encoder modules required:
#   ngx_http_gzip_static_module       (in tree)
#   github.com/google/ngx_brotli       (brotli + brotli_static)
#   github.com/tokers/zstd-nginx-module (zstd_static)  -- optional
# Verify: nginx -V 2>&1 | tr ' ' '\n' | grep -iE 'brotli|zstd'

server {
    listen 443 ssl http2;
    server_name spa.example.com;

    # TLS 1.3, no TLS-level compression (CRIME — agent §1.3, §8.4).
    ssl_protocols TLSv1.3 TLSv1.2;
    # Older OpenSSLs require: ssl_conf_command Options -Compression;

    root /var/www/spa;

    # ---- static, hashed assets (long-cache) ----
    location ~* ^/(assets|app|vendor|main-)[^/]+\.(js|css|svg|html)$ {
        # Pre-compressed file delivery (agent §6.2/§6.3).
        # Order matters: br first (Exp 0005, -82.0%), zstd next (Exp 0007, -81.8%),
        # gzip fallback (Exp 0003, -79.6%).
        brotli_static on;        # serves <file>.br when client accepts br
        zstd_static   on;        # serves <file>.zst when client accepts zstd
        gzip_static   on;        # serves <file>.gz when client accepts gzip

        add_header Cache-Control "public, max-age=31536000, immutable";
        add_header Vary           "accept-encoding";
    }

    # ---- runtime brotli for any text response missed by static path (rare) ----
    # Reserved for HTML rendered dynamically; keeps a sane fallback per agent §6.2.
    brotli            on;
    brotli_comp_level 5;            # Exp 0004 (runtime fallback)
    brotli_min_length 1024;         # agent §2.4 — below this, framing overhead wins
    brotli_window     4m;
    brotli_types
        text/plain text/css application/json application/javascript
        application/wasm image/svg+xml font/ttf font/otf;
    # Do NOT include image/png|jpeg|webp|avif, video/*, application/zip — already-compressed
    # types are an anti-pattern (agent §3 #2).

    gzip               on;
    gzip_vary          on;
    gzip_comp_level    6;
    gzip_min_length    1024;
    gzip_proxied       expired no-cache no-store private auth;
    gzip_types
        text/plain text/css application/json application/javascript
        application/wasm image/svg+xml font/ttf font/otf;
}

# ---- shared compression dictionaries (RFC 9842) ----
# Strategy: announce the OLDER versioned file as a dictionary candidate.
# Clients that have it cached send `Available-Dictionary: :<sha256-b64>:` on the
# next visit; the server (or a small lookup) replies with a pre-computed dcb/dcz
# payload that delta-decodes against that dict.
#
# Hashes (SHA-256, base64) for the corpus dict files:
#   app-3e4f5a6.js     xZi1ayYI/r6sCRcwYwRvEE0zwIOykPptbfarp/3DyEU=
#   main-7f8a3c2.css   wDogcPzJBh4OPS5c/OGzZl+BEKEB968lCQ/Tvrlkw6w=
#   vendor-a9b8c7d.js  2i4UkLRqTX7LX6LUvIANKo0mdyGBl0p+oGgbi9CSdCs=

location = /app-3e4f5a6.js {
    add_header Use-As-Dictionary 'match="/app-*.js", match-dest=("script"), id="app-bundle-v1"';
    add_header Cache-Control     "public, max-age=31536000, immutable";
    add_header Vary              "accept-encoding, available-dictionary";
    brotli_static on; zstd_static on; gzip_static on;
}
location = /main-7f8a3c2.css {
    add_header Use-As-Dictionary 'match="/main-*.css", match-dest=("style"), id="main-css-v1"';
    add_header Cache-Control     "public, max-age=31536000, immutable";
    add_header Vary              "accept-encoding, available-dictionary";
    brotli_static on; zstd_static on; gzip_static on;
}
location = /vendor-a9b8c7d.js {
    add_header Use-As-Dictionary 'match="/vendor-*.js", match-dest=("script"), id="vendor-v1"';
    add_header Cache-Control     "public, max-age=31536000, immutable";
    add_header Vary              "accept-encoding, available-dictionary";
    brotli_static on; zstd_static on; gzip_static on;
}

# Negotiate dcb/dcz on the matching newer paths (Exp 0008 brotli-dict-11, Exp 0009 zstd-dict-19).
# Pre-compress under /precompressed/<sha256-hex>/{dcb,dcz}/ at deploy time. Never compress
# at request time against a user-supplied dictionary (DoS — agent §3 #7, §5.4.2).
map $http_available_dictionary $dict_hash {
    default                       "";
    "~*^:([A-Za-z0-9+/=]+):$"     "$1";
}
map $http_accept_encoding $picked_dict_enc {
    default                       "";
    "~*\bdcz\b"                   "dcz";
    "~*\bdcb\b"                   "dcb";
}
location ~ ^/(app|main|vendor)-[A-Za-z0-9]+\.(js|css)$ {
    if ($dict_hash != "" ) {
        if ($picked_dict_enc != "") {
            rewrite ^ /precompressed/$dict_hash/$picked_dict_enc$uri last;
        }
    }
    brotli_static on; zstd_static on; gzip_static on;
    add_header Cache-Control "public, max-age=31536000, immutable";
    add_header Vary           "accept-encoding, available-dictionary";
}

location /precompressed/ {
    internal;
    add_header Vary           "accept-encoding, available-dictionary";
    add_header Cache-Control  "public, max-age=31536000, immutable";
    types { application/javascript js; text/css css; }
}
```

### Cloudflare Workers (alternative edge)

```js
// infra-a SPA edge worker.
// Backed by Exp 0005 (brotli-11), Exp 0007 (zstd-19),
// Exp 0008 (dcb / brotli-dict-11), Exp 0009 (dcz / zstd-dict-19), Exp 0010 (hybrid).
// Pre-compressed assets uploaded via Workers Sites or R2 under:
//   <path>.br   <path>.zst   <path>.gz
// Dictionary-encoded:
//   precompressed/<sha256-hex>/dcb/<path>
//   precompressed/<sha256-hex>/dcz/<path>

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    const ae  = req.headers.get("Accept-Encoding") || "";
    const dictHashHdr = req.headers.get("Available-Dictionary"); // ":<b64>:"

    const isVersioned = /^\/(app|main|vendor)-[A-Za-z0-9]+\.(js|css)$/.test(url.pathname);

    // 1. Shared-dictionary path — only on versioned bundle URLs (Exp 0008, 0009).
    if (isVersioned && dictHashHdr && (ae.includes("dcb") || ae.includes("dcz"))) {
      const enc = ae.includes("dcz") ? "dcz" : "dcb";
      const m = /^:([A-Za-z0-9+/=]+):$/.exec(dictHashHdr.trim());
      if (m) {
        const hashHex = b64ToHex(m[1]);
        const obj = await env.ASSETS.get(`precompressed/${hashHex}/${enc}${url.pathname}`);
        if (obj) {
          return new Response(obj.body, {
            headers: {
              "content-encoding": enc,
              "content-type":     ctypeFor(url.pathname),
              "vary":             "accept-encoding, available-dictionary",
              "cache-control":    "public, max-age=31536000, immutable",
            },
          });
        }
      }
    }

    // 2. Static pre-compressed path (br > zstd > gzip > identity).
    //    Exp 0005 (brotli-11) is the universal CHAMPION.
    const order = [];
    if (ae.includes("br"))   order.push(["br", ".br"]);
    if (ae.includes("zstd")) order.push(["zstd", ".zst"]);
    if (ae.includes("gzip")) order.push(["gzip", ".gz"]);
    for (const [token, suffix] of order) {
      const obj = await env.ASSETS.get(url.pathname.slice(1) + suffix);
      if (obj) {
        return new Response(obj.body, {
          headers: {
            "content-encoding": token,
            "content-type":     ctypeFor(url.pathname),
            "vary":             "accept-encoding",
            "cache-control":    "public, max-age=31536000, immutable",
          },
        });
      }
    }

    // 3. Identity fallback.
    return env.ASSETS.fetch(req);
  },
};

function ctypeFor(p) {
  if (p.endsWith(".js"))   return "application/javascript; charset=utf-8";
  if (p.endsWith(".css"))  return "text/css; charset=utf-8";
  if (p.endsWith(".html")) return "text/html; charset=utf-8";
  if (p.endsWith(".svg"))  return "image/svg+xml";
  if (p.endsWith(".wasm")) return "application/wasm";
  return "application/octet-stream";
}

function b64ToHex(b64) {
  const bin = atob(b64);
  let h = "";
  for (let i = 0; i < bin.length; i++) h += bin.charCodeAt(i).toString(16).padStart(2, "0");
  return h;
}
```

## Build hook

### Make variant

```makefile
# infra-a-spa Makefile fragment. Pre-compresses every static asset and emits dcb/dcz
# for the 3 versioned bundle pairs. Encoder costs are free (build-time per SCOPE.md).
DIST           := dist
ASSETS         := $(shell find $(DIST) -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' -o -name '*.wasm' \))
DICT_PAIRS_JS  := app-3e4f5a6.js:app-3e4f5a7.js vendor-a9b8c7d.js:vendor-a9b8c7e.js
DICT_PAIRS_CSS := main-7f8a3c2.css:main-7f8a3c3.css

.PHONY: compress dict-compress
compress: $(ASSETS:%=%.br) $(ASSETS:%=%.zst) $(ASSETS:%=%.gz) dict-compress

%.br:  %
	brotli -q 11 -k -f $<           # Exp 0005
%.zst: %
	zstd  -19    -k -f $<           # Exp 0007
%.gz:  %
	gzip  -9     -k -f $<           # Exp 0003 (legacy fallback)

# Dictionary outputs (Exp 0008 / 0009). Frame with dcb/dcz magic + 32-byte SHA-256.
# Output path:  $(DIST)/precompressed/<sha256-hex>/{dcb,dcz}/<basename>
dict-compress:
	@for spec in $(DICT_PAIRS_JS) $(DICT_PAIRS_CSS); do \
	  old=$${spec%%:*}; new=$${spec##*:}; \
	  hash_hex=$$(openssl dgst -sha256 -binary $(DIST)/$$old | xxd -p | tr -d '\n'); \
	  mkdir -p $(DIST)/precompressed/$$hash_hex/dcb $(DIST)/precompressed/$$hash_hex/dcz; \
	  brotli -q 11 -D $(DIST)/$$old -c $(DIST)/$$new > /tmp/raw.br; \
	  { printf '\xff\x44\x43\x42'; printf %s "$$hash_hex" | xxd -r -p; cat /tmp/raw.br; } \
	      > $(DIST)/precompressed/$$hash_hex/dcb/$$new; \
	  zstd  -19    -D $(DIST)/$$old -c $(DIST)/$$new 2>/dev/null > /tmp/raw.zst; \
	  { printf '\x5e\x2a\x4d\x18\x20\x00\x00\x00'; printf %s "$$hash_hex" | xxd -r -p; cat /tmp/raw.zst; } \
	      > $(DIST)/precompressed/$$hash_hex/dcz/$$new; \
	done
	@rm -f /tmp/raw.br /tmp/raw.zst
```

### npm scripts variant

```json
{
  "scripts": {
    "build": "vite build && npm run compress",
    "compress": "npm run compress:static && npm run compress:dict",
    "compress:static": "find dist -type f \\( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' -o -name '*.wasm' \\) -exec sh -c 'brotli -q 11 -k -f \"$1\" && zstd -19 -k -f \"$1\" && gzip -9 -k -f \"$1\"' _ {} \\;",
    "compress:dict": "node scripts/build-dict-payloads.mjs"
  }
}
```

`scripts/build-dict-payloads.mjs` (sketch — implements the same dcb/dcz framing as the
Makefile target):

```js
// Reads dist/, finds versioned pairs, emits dist/precompressed/<hex>/{dcb,dcz}/<file>.
// Magic bytes per RFC 9842 / agent definition §5.4.2.
import { execSync } from "node:child_process";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { createHash } from "node:crypto";

const PAIRS = [
  ["dist/app-3e4f5a6.js",     "dist/app-3e4f5a7.js"],
  ["dist/main-7f8a3c2.css",   "dist/main-7f8a3c3.css"],
  ["dist/vendor-a9b8c7d.js",  "dist/vendor-a9b8c7e.js"],
];
const DCB_MAGIC = Buffer.from([0xff, 0x44, 0x43, 0x42]);
const DCZ_MAGIC = Buffer.from([0x5e, 0x2a, 0x4d, 0x18, 0x20, 0x00, 0x00, 0x00]);

for (const [dict, input] of PAIRS) {
  const hash = createHash("sha256").update(readFileSync(dict)).digest();
  const hex  = hash.toString("hex");
  mkdirSync(`dist/precompressed/${hex}/dcb`, { recursive: true });
  mkdirSync(`dist/precompressed/${hex}/dcz`, { recursive: true });
  const name = input.split("/").pop();
  const br   = execSync(`brotli -q 11 -D ${dict} -c ${input}`);
  const zst  = execSync(`zstd  -19 -D ${dict} -c ${input}`);
  writeFileSync(`dist/precompressed/${hex}/dcb/${name}`, Buffer.concat([DCB_MAGIC, hash, br]));
  writeFileSync(`dist/precompressed/${hex}/dcz/${name}`, Buffer.concat([DCZ_MAGIC, hash, zst]));
}
```

## Verification commands

When the SPA is deployed (Phase 7 of the agent loop), run:

```bash
URL="https://spa.example.com/index.html"
URL_DICT_OLD="https://spa.example.com/app-3e4f5a6.js"
URL_DICT_NEW="https://spa.example.com/app-3e4f5a7.js"

# 1. Body compression negotiated correctly (agent §8.1)
for ae in 'identity' 'gzip' 'br' 'zstd' 'gzip, br, zstd, dcb, dcz'; do
  printf '%-35s ' "$ae"
  curl -sI -H "Accept-Encoding: $ae" "$URL" \
    | grep -iE 'content-encoding|content-length|vary' | tr '\n' ' '
  echo
done

# 2. Use-As-Dictionary header on dict source (agent §8.2)
curl -sI "$URL_DICT_OLD" | grep -iE 'use-as-dictionary|cache-control|vary'

# 3. Shared dictionary path (RFC 9842, agent §8.2)
HASH=$(curl -s "$URL_DICT_OLD" | openssl dgst -sha256 -binary | base64)
curl -s -D - -o /tmp/new.bin \
  -H "Accept-Encoding: br, zstd, dcb, dcz" \
  -H "Available-Dictionary: :$HASH:" \
  "$URL_DICT_NEW" | grep -iE 'content-encoding|vary|content-length'
xxd /tmp/new.bin | head -2     # expect dcb (ff44 4342 ...) or dcz (5e2a 4d18 2000 0000 ...)

# 4. TLS compression off (CRIME, agent §8.4)
echo | openssl s_client -connect spa.example.com:443 2>&1 | grep -i compression
# expect: "Compression: NONE" or absent on TLS 1.3

# 5. No double-compression (agent §8.5)
curl -sI -H 'Accept-Encoding: br' "$URL" | grep -i content-encoding
# expect single value: "Content-Encoding: br"

# 6. Range + compression (agent §8.6)
curl -sI -H 'Range: bytes=0-1023' -H 'Accept-Encoding: br' "$URL"
# expect either 206 WITHOUT Content-Encoding, or 200 with full body — never 206 + CE.
```

## Security notes

- **BREACH (agent §1.3, §3 #5):** N/A. `SCOPE.md` declares no secrets, no reflected
  user input on this static SPA. All assets are public, immutable, hashed-filename.
- **TLS compression (CRIME, agent §1.3, §3 #6):** must be off. TLS 1.3 forbids it.
  Verification command above (item 4) confirms.
- **Shared dictionaries (RFC 9842, agent §3 #7, §5.4.2):**
  - HTTPS only — yes (TLS 1.3 origin assumed).
  - Same-origin `match` — yes (`/app-*.js`, `/main-*.css`, `/vendor-*.js`); no regex,
    no cross-origin.
  - No secrets in dict files — confirmed; bundles are public client-side JS/CSS.
  - `Vary: accept-encoding, available-dictionary` — yes, set on every dict-eligible
    response and every dict-encoded response.
  - Pre-compute payloads at deploy time, never compress at request time against a
    user-supplied dictionary (DoS surface) — yes, build hook above.
  - Brotli quality ≥5 / zstd level ≥3 — yes (q=11, level=19; Exp 0008/0009).
- **HPACK / QPACK (agent §3 #8, §5.4.3, §5.4.4):** out of scope for static
  asset delivery (no high-value low-entropy headers like Authorization or Set-Cookie
  on this surface). If/when the SPA gains an authenticated subpath, mark Authorization
  and Cookie as never-indexed (HPACK 0001 / QPACK literal-not-indexed).
- **WebSocket permessage-deflate (agent §3 #9, §5.4.5):** N/A; no WebSocket on this
  static surface.
- **Already-compressed types (agent §3 #2):** `image/png`, `image/jpeg`, `image/webp`,
  `image/avif`, `font/woff2`, `video/*`, `application/zip`, `application/gzip`,
  `application/zstd` are NOT in `brotli_types` / `gzip_types`. Confirmed in nginx
  config above.
- **`Vary` correctness (agent §3 #3, RFC 9111 §4.1):** every response with
  `Content-Encoding` carries `Vary: accept-encoding`; dict-encoded responses also
  carry `available-dictionary` in `Vary`.
- **Range requests on compressed responses (agent §3 #10):** nginx `_static on`
  serves the pre-compressed file as the full resource; range requests on compressed
  responses are not produced. Verification command (item 6) catches regressions.

## Risks / Follow-ups

- **No live server in this environment.** All numbers are file-level encode/decode/size;
  agent §4.1's full metric (`transfer_time_p95_4g`, `ttfb_p95`) is not measurable here.
  Once deployed, re-run §8.1, §8.2, and the wire bench (`oha -n 2000 -c 50 ...`) to
  confirm CDN does not strip `Content-Encoding`, that `Vary` survives intermediaries,
  and that LCP improves on 4G mobile.
- **Decode CPU timing has CLI-spawn-dominated noise.** Per-file decode work is
  ~0.03–0.6 ms but hyperfine measurements include ~13 ms process spawn. The
  differential signal between brotli-11 and zstd-19 decode is well below noise here.
  In a real Lighthouse / WebPageTest run on a low-end Android, this ranking can be
  re-checked properly via DevTools "Compute Compression" timings. Agent §2.1's
  vendor-published numbers (zstd ~1500 MB/s vs brotli ~400 MB/s decode) hold and
  argue for `dcz` over `dcb` if mobile decode CPU becomes a bottleneck.
- **Modest dict savings on this corpus (1–7%).** Versioned files are near-identical;
  brotli-11 alone already extracts most of the redundancy via its 4 MiB window plus
  RFC 7932 static dictionary. On real bundle pairs with deeper diffs, agent §9.3
  cites 30–60% savings. The framework is correct; rewards scale with diff depth.
- **`svgo` not installed** in this environment. SVG asset (`logo.svg`) was compressed
  via brotli-11 only; pre-minification with `svgo --multipass --pretty=false` would
  shrink the source further before the LZ77 pass. Add to the build hook when available.
- **Encoded-image / font experiments not run.** Corpus has no PNG / JPEG / WOFF2.
  When the SPA gains image or font assets, run agent §2.6 / §2.7 playbooks as new
  experiments (Exp 0011+).
- **Versioning policy.** The dict scheme as configured points each new bundle at the
  immediately-prior version. If users routinely jump multiple versions (long-lived
  open tabs), they may not have the prior dict cached; the precompress matrix should
  cover N-1 and N-2, or the announcement should rotate. Measure cache-hit rate on
  `Available-Dictionary` headers in production logs and iterate.
- **CDN handling of `dcb`/`dcz`.** Cloudflare and Fastly support varies in 2025-2026.
  The Workers config above sidesteps the CDN's auto-compress; verify the CDN does
  not strip or re-encode the `Content-Encoding: dcb`/`dcz` response. CloudFront in
  particular may not honor the encoding on the auto-compress path (agent §6.12).
- **Long-cache + `immutable`** is set on every static asset response. Confirm this
  matches the deployment's redeploy semantics; if a hot-fix replaces a hashed file
  in place (anti-pattern), drop `immutable`.

---

References inline cite agent definition section numbers (`agent §X`) and RFCs
(`RFC 9842`, `RFC 7932`, `RFC 8878`, `RFC 9110`, `RFC 9111`, `RFC 1951`/1952). Per-file
encoded sizes and items.json: `bench/results/<exp-id>/`. Bootstrap CIs: `bench/score.py`.
Append-only experiment log: `bench/EXPERIMENTS.md`. Scope (do not edit): `bench/SCOPE.md`.
