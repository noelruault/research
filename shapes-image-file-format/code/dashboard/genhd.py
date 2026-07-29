#!/usr/bin/env python3
"""Builds hd-mirror.html: the 1:1 comparison at 3840x2160, across the whole rate axis.

Every image embedded here is a 3840x2160-space window of a real decoded file, encoded for display as
lossless WebP, so what the page shows is each encoder's own artefacts and never the display pipeline's.
The display path adds no resampling of its own. On steps 1 and 2 the WebP file is itself encoded below
native and upscaled -- that resampling is the thing being measured, and it happens before the window is
cut, exactly as it would in a browser.

Numbers come from, and only from:
  hd/rate/steps.tsv          the byte-matched rate ladder               (rate-build.sh)
  hd/rate/window-stats.tsv   per-window fidelity, keyed by step         (rate-build.sh)
  hd/ladder_<W>.txt          shape-coder scale-space per resolution     (lab hd)
  hd/ladder-codecs.txt       WebP + AVIF rate-distortion per resolution (ladder-sweep.sh)

Those four are read at build time and never transcribed. The constants below -- LOSSLESS, EXACT, LADDER,
PER_WINDOW -- are transcribed, from reports 08 and 05; they are the page's remaining hand-copied numbers
and the reason the header block can drift when a report is corrected. Check them against the report when
either changes.
"""
import base64
import json
import pathlib

HERE = pathlib.Path(__file__).resolve().parent
HD = HERE.parent / "hd"
PIX4K = 3840 * 2160

# snow is measured and tabulated, but not embedded: it behaves like ridge and costs another 1.2 MB of page.
WINDOWS = [
    dict(key="sky", title="Sky", where="288, 230",
         note="A smooth gradient with film grain. Nothing to segment: no edges, no regions, just a ramp."),
    dict(key="ridge", title="Sunlit ridge", where="1344, 1000",
         note="Dense texture at every scale, plus the hard terminator between lit rock and shadow."),
]


def uri(p):
    return "data:image/webp;base64," + base64.b64encode(pathlib.Path(p).read_bytes()).decode()


def kb(n):
    return f"{n / 1024:,.0f} KB" if n < 1024 * 1024 else f"{n / 1048576:.2f} MB"


LOSSLESS = [
    ("JPEG XL", "cjxl -d 0 -e 7", 6_468_598, "jxl"),
    ("WebP", "cwebp -lossless -z 9", 7_718_506, "webp"),
    ("AVIF", "avifenc --lossless", 11_969_137, "avif"),
    ("shape coder", "exact region partition", 12_159_385, "shapes"),
    ("PNG", "as delivered", 12_278_280, "png"),
]
EXACT = dict(regions=6_356_392, px_per_region=1.305, cracks=14_508_938,
             walls=822_369, colours=11_337_015, bits_per_edge=0.4534)

LADDER = [
    (28.7, [(512, 9_399, 9_382), (960, 23_172, 22_660), (1920, 64_112, 60_052), (3840, 163_471, 137_033)]),
    (30.0, [(512, 13_168, 12_363), (960, 33_193, 30_493), (1920, 93_104, 82_084), (3840, 245_180, 184_715)]),
    (31.5, [(512, 18_633, 17_060), (960, 47_951, 41_486), (1920, 139_056, 111_804), (3840, 381_339, 256_699)]),
    (34.0, [(512, 29_952, 30_500), (960, 80_350, 67_648), (1920, 247_112, 177_174), (3840, 726_980, 425_685)]),
]

# 448px windows from the earlier matched-size/matched-quality round, kept because they are the sharpest statement of *where* the deficit lives even though those panels are no longer embedded.
PER_WINDOW = [
    ("Sunlit ridge", "dense texture, hard shadow terminator", 26.01, 26.44),
    ("Snow and rock", "flat fields split by sharp dark edges", 26.82, 27.39),
    ("Sky", "smooth gradient, film grain", 36.90, 42.29),
]


def load_steps():
    steps = []
    for line in (HD / "rate" / "steps.tsv").read_text().splitlines():
        i, regions, sb, sp, wq, wb, wp, resampled = line.split("\t")
        steps.append(dict(i=int(i), regions=int(regions), sbytes=int(sb), spsnr=float(sp),
                          wq=wq, wbytes=int(wb), wpsnr=float(wp), resampled=resampled == "1"))
    return steps


def load_window_stats(nsteps):
    """{window: [{shapes, webp}, ...]} in step order, keyed explicitly by the build's own step column."""
    out = {w["key"]: [{} for _ in range(nsteps)] for w in WINDOWS}
    for ln in (HD / "rate" / "window-stats.tsv").read_text().splitlines():
        if not ln.strip():
            continue
        step, window, who, psnr = ln.split("\t")
        if window in out:  # the build measures more windows than this page embeds
            out[window][int(step) - 1][who] = float(psnr)
    missing = [(w, i + 1) for w, rows in out.items() for i, r in enumerate(rows) if len(r) != 2]
    if missing:
        raise SystemExit(f"window-stats.tsv is missing shapes/webp rows for {missing} — rebuild it")
    return out


STEPS = load_steps()
WSTATS = load_window_stats(len(STEPS))


def lossless_rows():
    worst = max(r[2] for r in LOSSLESS)
    return "\n".join(
        f'<tr{" class=\"hl\"" if cls == "shapes" else ""}>'
        f'<td><span class="dot {cls}"></span>{name}</td>'
        f'<td class="mono dim">{how}</td><td class="mono num">{b:,}</td>'
        f'<td class="bar"><i style="width:{100 * b / worst:.1f}%" class="{cls}"></i></td>'
        f'<td class="mono num">{"&mdash;" if cls == "webp" else f"{b / 7_718_506:.2f}&times;"}</td></tr>'
        for name, how, b, cls in LOSSLESS)


def window_blocks():
    out = []
    for w in WINDOWS:
        imgs = [f'<img class="lyr" data-k="orig" src="{uri(HD / "rate" / f"{w["key"]}_orig.webp")}" '
                f'alt="{w["title"]}, original" draggable="false">']
        for s in STEPS:
            for who in ("webp", "shapes"):
                # An exact step's shape render IS the original, pixel for pixel. It still needs its own layer -- one element cannot be both sides of the wipe -- but embedding the same base64 twice would cost ~1 MB, so this layer borrows the orig layer's src at load.
                if who == "shapes" and s["spsnr"] >= 99:
                    imgs.append(
                        f'<img class="lyr" data-k="s{s["i"]}_{who}" data-src-from="orig" '
                        f'alt="{w["title"]}, step {s["i"]}, exact region partition" draggable="false">')
                    continue
                imgs.append(
                    f'<img class="lyr" data-k="s{s["i"]}_{who}" '
                    f'src="{uri(HD / "rate" / f"{w["key"]}_s{s["i"]}_{who}.webp")}" '
                    f'alt="{w["title"]}, step {s["i"]}, {who}" draggable="false">')
        out.append(f'''
<figure class="win" data-win="{w['key']}">
  <figcaption>
    <h3>{w['title']} <span class="coord mono">@ {w['where']}</span></h3>
    <p>{w['note']}</p>
    <dl class="wstat mono">
      <div><dt data-role="lname">left</dt><dd data-role="ldb">&mdash;</dd></div>
      <div><dt>shape coder</dt><dd data-role="rdb">&mdash;</dd></div>
      <div class="delta"><dt>difference here</dt><dd data-role="ddb">&mdash;</dd></div>
    </dl>
  </figcaption>
  <div class="stage">
    <div class="frame">
      {"".join(imgs)}
      <div class="handle" tabindex="0" role="slider" aria-label="wipe position"
           aria-valuemin="0" aria-valuemax="100" aria-valuenow="50"><span></span></div>
      <div class="tagL mono"></div><div class="tagR mono"></div>
    </div>
    <p class="framenote">384&times;384 native pixels from the 3840&times;2160 image. File sizes are the
      whole file; &ldquo;here&rdquo; is measured on this window alone.</p>
  </div>
</figure>''')
    return "\n".join(out)


def ladder_table():
    head = "".join(f"<th>{w}px</th>" for w, _, _ in LADDER[0][1])
    rows = "".join(
        f"<tr><th>{psnr:.1f} dB</th>" + "".join(
            f'<td class="mono num {"bad" if s > c else "good"}">{100 * (s - c) / c:+.1f}%</td>'
            for _w, s, c in pts) + "</tr>"
        for psnr, pts in LADDER)
    return f"<thead><tr><th></th>{head}</tr></thead><tbody>{rows}</tbody>"


def step_table():
    rows = []
    for s in STEPS:
        if s["wpsnr"] >= 99 and s["spsnr"] >= 99:
            # Both coders bit-exact: the only thing left to compare is the bill.
            verdict = (f'<span class="bad">{s["sbytes"] / s["wbytes"]:.2f}&times; the bytes '
                       f'for the same pixels</span>')
        elif s["wpsnr"] >= 99:
            verdict = '<span class="bad">WebP is exact, for fewer bytes</span>'
        else:
            d = s["spsnr"] - s["wpsnr"]
            verdict = f'<span class="{"good" if d > 0 else "bad"}">{d:+.2f} dB</span>'
        wdb = "exact" if s["wpsnr"] >= 99 else f'{s["wpsnr"]:.2f} dB'
        sdb = "exact" if s["spsnr"] >= 99 else f'{s["spsnr"]:.2f} dB'
        rows.append(
            f'<tr><td class="mono num">{s["i"]}</td>'
            f'<td class="mono num">{s["sbytes"]:,}</td><td class="mono num">{sdb}</td>'
            f'<td class="mono dim num">{s["regions"]:,}</td>'
            f'<td class="mono num">{s["wbytes"]:,}</td><td class="mono num">{wdb}</td>'
            f'<td class="mono dim">{s["wq"]}</td><td class="num">{verdict}</td></tr>')
    return "".join(rows)


html = f'''<title>1:1 &mdash; shapes against WebP at 4K</title>
<style>
:root {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jxl:#2f8f6b; --png:#818b9e;
  --good:#1a7f47; --bad:#c2412f; --grid:#e6eaf1;
}}
@media (prefers-color-scheme:dark) {{
  :root {{
    --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
    --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
    --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jxl:#3fb98c; --png:#8b95a8;
    --good:#3fb950; --bad:#f85149; --grid:#222937;
  }}
}}
:root[data-theme="dark"] {{
  --bg:#0f1218; --surface:#161b23; --line:#252c38; --line2:#1c222c;
  --ink:#e7eaf1; --ink2:#98a2b6; --ink3:#6c7689;
  --shapes:#4cc9f0; --avif:#f5a524; --webp:#a78bfa; --jxl:#3fb98c; --png:#8b95a8;
  --good:#3fb950; --bad:#f85149; --grid:#222937;
}}
:root[data-theme="light"] {{
  --bg:#f4f6f9; --surface:#ffffff; --line:#dfe4ec; --line2:#eef1f6;
  --ink:#151922; --ink2:#5b6479; --ink3:#8b94a8;
  --shapes:#0e8fb8; --avif:#b8791a; --webp:#7c5cd0; --jxl:#2f8f6b; --png:#818b9e;
  --good:#1a7f47; --bad:#c2412f; --grid:#e6eaf1;
}}
*,*::before,*::after {{ box-sizing:border-box; }}
body {{ margin:0; background:var(--bg); color:var(--ink);
  font:400 16px/1.6 ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;
  -webkit-font-smoothing:antialiased; }}
.mono {{ font-family:ui-monospace,SFMono-Regular,"SF Mono",Menlo,Consolas,monospace;
  font-variant-numeric:tabular-nums; }}
.wrap {{ max-width:1180px; margin:0 auto; padding:0 24px 96px; }}
a {{ color:var(--shapes); }}

header {{ padding:72px 0 40px; border-bottom:1px solid var(--line); }}
.eyebrow {{ font-family:ui-monospace,Menlo,monospace; font-size:11px; letter-spacing:.16em;
  text-transform:uppercase; color:var(--ink3); margin:0 0 20px; display:flex; gap:14px; flex-wrap:wrap; }}
.eyebrow a {{ color:var(--ink3); }}
/* No measure caps on prose: text runs to the width of its container. A ch-based max-width leaves a
   dead gutter beside every paragraph on a wide screen, and the wrap point is the viewport's business. */
h1 {{ margin:0 0 20px; font-size:clamp(2rem,5vw,3.4rem); line-height:1.03; letter-spacing:-.035em;
  font-weight:660; text-wrap:balance; }}
h1 b {{ color:var(--shapes); font-weight:660; }}
.lede {{ margin:0; font-size:1.075rem; color:var(--ink2); }}
.lede strong {{ color:var(--ink); font-weight:600; }}

.stats {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(200px,1fr)); gap:1px;
  background:var(--line); border:1px solid var(--line); margin:40px 0 0; }}
.stat {{ background:var(--surface); padding:20px 22px; }}
.stat .k {{ font-family:ui-monospace,Menlo,monospace; font-size:10.5px; letter-spacing:.14em;
  text-transform:uppercase; color:var(--ink3); }}
.stat .v {{ font-family:ui-monospace,Menlo,monospace; font-size:1.5rem; font-weight:600;
  letter-spacing:-.02em; margin-top:6px; font-variant-numeric:tabular-nums; }}
.stat .n {{ font-size:13px; color:var(--ink2); margin-top:4px; }}

section {{ padding:64px 0 0; }}
h2 {{ font-size:1.6rem; letter-spacing:-.02em; margin:0 0 8px; font-weight:640; }}
.sub {{ color:var(--ink2); margin:0 0 28px; }}

.tw {{ overflow-x:auto; border:1px solid var(--line); background:var(--surface); }}
table {{ border-collapse:collapse; width:100%; font-size:14.5px; min-width:560px; }}
th,td {{ padding:11px 16px; text-align:left; border-bottom:1px solid var(--line2); }}
thead th {{ font-size:10.5px; letter-spacing:.13em; text-transform:uppercase; color:var(--ink3);
  font-weight:600; border-bottom:1px solid var(--line); }}
tbody tr:last-child td, tbody tr:last-child th {{ border-bottom:0; }}
.num {{ text-align:right; }}
.dim {{ color:var(--ink3); font-size:12.5px; }}
tr.hl td {{ background:color-mix(in srgb, var(--shapes) 8%, transparent); }}
.good {{ color:var(--good); }} .bad {{ color:var(--bad); }}
.dot {{ display:inline-block; width:9px; height:9px; border-radius:2px; margin-right:9px; }}
.dot.shapes{{background:var(--shapes)}} .dot.webp{{background:var(--webp)}}
.dot.avif{{background:var(--avif)}} .dot.jxl{{background:var(--jxl)}} .dot.png{{background:var(--png)}}
td.bar {{ width:38%; }}
td.bar i {{ display:block; height:8px; border-radius:2px; opacity:.75; }}
i.shapes{{background:var(--shapes)}} i.webp{{background:var(--webp)}}
i.avif{{background:var(--avif)}} i.jxl{{background:var(--jxl)}} i.png{{background:var(--png)}}

.rate {{ border:1px solid var(--line); background:var(--surface); padding:22px 24px; margin:0 0 8px; }}
.rate .top {{ display:flex; justify-content:space-between; align-items:flex-end; gap:20px;
  flex-wrap:wrap; margin:0 0 16px; }}
.rate .lab {{ font-family:ui-monospace,Menlo,monospace; font-size:10.5px; letter-spacing:.14em;
  text-transform:uppercase; color:var(--ink3); }}
.rate .now {{ font-size:1.35rem; font-weight:600; letter-spacing:-.02em; margin-top:4px; }}
input[type=range] {{ width:100%; accent-color:var(--shapes); height:26px; }}
.ticks {{ display:flex; justify-content:space-between; font-family:ui-monospace,Menlo,monospace;
  font-size:10.5px; color:var(--ink3); margin-top:2px; }}
.sides {{ display:grid; grid-template-columns:1fr 1fr; gap:1px; background:var(--line);
  border:1px solid var(--line); margin-top:18px; }}
.side {{ background:var(--surface); padding:14px 16px; }}
.side .who {{ font-size:11px; letter-spacing:.12em; text-transform:uppercase; font-weight:700; }}
.side.l .who {{ color:var(--webp); }} .side.r .who {{ color:var(--shapes); }}
.side .big {{ font-family:ui-monospace,Menlo,monospace; font-size:1.15rem; font-weight:600;
  margin-top:6px; font-variant-numeric:tabular-nums; }}
.side .sm {{ font-size:12.5px; color:var(--ink2); margin-top:2px; }}
.toggle {{ display:inline-flex; border:1px solid var(--line); }}
.toggle button {{ appearance:none; border:0; background:transparent; cursor:pointer; color:var(--ink2);
  font:inherit; font-size:13px; padding:8px 14px; border-right:1px solid var(--line); }}
.toggle button:last-child {{ border-right:0; }}
.toggle button[aria-pressed="true"] {{ background:color-mix(in srgb, var(--shapes) 12%, transparent);
  color:var(--ink); font-weight:600; }}
.toggle button:focus-visible {{ outline:2px solid var(--shapes); outline-offset:-2px; }}

.win {{ margin:0 0 28px; border:1px solid var(--line); background:var(--surface);
  display:grid; grid-template-columns:minmax(0,1fr) minmax(0,auto); }}
@media (max-width:900px) {{ .win {{ grid-template-columns:minmax(0,1fr); }} }}
.win figcaption {{ padding:22px 24px; min-width:0; }}
.win h3 {{ margin:0 0 6px; font-size:1.1rem; font-weight:620; letter-spacing:-.01em; }}
.coord {{ color:var(--ink3); font-size:12px; font-weight:400; }}
.win p {{ margin:0 0 18px; color:var(--ink2); font-size:14px; }}
.wstat {{ margin:0; display:flex; flex-direction:column; gap:1px; background:var(--line);
  border:1px solid var(--line); font-size:12.5px; }}
.wstat div {{ display:flex; justify-content:space-between; gap:14px; background:var(--surface);
  padding:8px 12px; }}
.wstat dt {{ color:var(--ink3); }}
.wstat dd {{ margin:0; font-weight:600; }}
.wstat .delta dd {{ font-weight:700; }}
.stage {{ padding:16px; border-left:1px solid var(--line); min-width:0; }}
@media (max-width:900px) {{ .stage {{ border-left:0; border-top:1px solid var(--line); }} }}
/* 384px is the source size, so at 384px and up one screen pixel is one image pixel. Narrower than that the frame scales: a comparison you cannot see on a phone is worse than one that is not literally 1:1. */
.frame {{ position:relative; width:min(384px,100%); aspect-ratio:1; overflow:hidden;
  user-select:none; touch-action:pan-y; cursor:ew-resize; background:var(--line2); }}
.frame img {{ position:absolute; inset:0; width:100%; height:100%; display:none; }}
.frame img.showR {{ display:block; z-index:1; }}
/* The left panel is revealed by clipping, not by nesting it in a resized container: one element, one coordinate system, nothing to fall out of alignment. */
.frame img.showL {{ display:block; z-index:2; clip-path:inset(0 calc(100% - var(--wipe,50%)) 0 0); }}
/* Above BOTH images, and load-bearing: without it the z-index:2 left image paints over its own caption and the wipe handle, which reads as "only one image is showing". */
.handle {{ position:absolute; top:0; bottom:0; left:50%; width:2px; background:#fff; z-index:4;
  box-shadow:0 0 0 1px rgba(0,0,0,.45); transform:translateX(-1px); }}
.handle span {{ position:absolute; top:50%; left:50%; width:34px; height:34px; margin:-17px 0 0 -17px;
  border-radius:50%; background:#fff; box-shadow:0 1px 6px rgba(0,0,0,.4);
  display:grid; place-items:center; }}
.handle span::before {{ content:"\\2194"; color:#111; font-size:15px; line-height:1; }}
.handle:focus-visible {{ outline:2px solid var(--shapes); outline-offset:3px; }}
.tagL,.tagR {{ position:absolute; bottom:10px; z-index:5; font-size:11px; line-height:1.35;
  background:rgba(10,12,16,.86); color:#fff; padding:6px 9px; border-radius:3px;
  pointer-events:none; max-width:calc(50% - 12px); }}
.tagL {{ left:10px; }} .tagR {{ right:10px; text-align:right; }}
.tagL b,.tagR b {{ display:block; font-size:11.5px; letter-spacing:.05em; text-transform:uppercase;
  font-weight:700; }}
.tagL i,.tagR i {{ font-style:normal; opacity:.85; }}
.framenote {{ margin:12px 0 0 !important; font-size:11.5px; color:var(--ink3); max-width:384px; }}

.chart {{ border:1px solid var(--line); background:var(--surface); padding:20px; }}
canvas {{ display:block; width:100%; height:340px; }}
.legend {{ display:flex; gap:22px; flex-wrap:wrap; margin:14px 0 0; font-size:12.5px; color:var(--ink2); }}
.legend i {{ display:inline-block; width:14px; height:3px; border-radius:2px; margin-right:7px;
  vertical-align:middle; }}
.why {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(280px,1fr)); gap:1px;
  background:var(--line); border:1px solid var(--line); }}
.why div {{ background:var(--surface); padding:24px; }}
.why h4 {{ margin:0 0 8px; font-size:14px; font-weight:640; }}
.why p {{ margin:0; color:var(--ink2); font-size:14px; }}
.why code {{ font-family:ui-monospace,Menlo,monospace; font-size:12.5px; color:var(--ink); }}
footer {{ margin-top:72px; padding-top:28px; border-top:1px solid var(--line);
  color:var(--ink3); font-size:13px; }}
@media (prefers-reduced-motion:reduce) {{ * {{ transition:none !important; }} }}
</style>

<div class="wrap">
<header>
  <p class="eyebrow"><span>shapes-image-file-format &middot; report 08</span>
    <a href="dashboard.html">&larr; results dashboard</a></p>
  <h1>At 4K, the shape idea runs out at <b>both ends</b> of the rate axis.</h1>
  <p class="lede">The earlier rounds ran on a 512&times;288 crop, where the region coder appeared to edge
    WebP below 29.2 dB &mdash; a result that did not survive taking WebP off its default encoder setting,
    and whose last refuge, the very bottom of the rate range, did not survive letting WebP encode small
    and upscale the way real delivery does. This is the same picture at <strong>3840&times;2160</strong>,
    its native size, priced by the same coder. Every panel below is a <strong>1:1 window</strong> &mdash; one screen pixel is one image pixel,
    no resampling, and the display copies are lossless so the artefacts are each encoder's own.</p>
  <div class="stats">
    <div class="stat"><div class="k">exact partition</div><div class="v">{EXACT['regions']:,}</div>
      <div class="n">regions for a bit-exact image &mdash; {EXACT['px_per_region']} px each</div></div>
    <div class="stat"><div class="k">shapes, lossless</div><div class="v">{kb(12_159_385)}</div>
      <div class="n">1.58&times; WebP lossless, 1.88&times; JPEG XL</div></div>
    <div class="stat"><div class="k">walls vs colours</div><div class="v">93%</div>
      <div class="n">of those bytes are colour; the geometry is nearly free</div></div>
    <div class="stat"><div class="k">deficit at 28.7 dB</div><div class="v">+19.3%</div>
      <div class="n">vs WebP at 4K &mdash; against +0.2% at 512&times;288</div></div>
  </div>
</header>

<section>
  <h2>Being exactly right</h2>
  <p class="sub">What each encoder costs to reproduce all 8,294,400 pixels with zero error. The shape coder
    is priced with this study's own crack-edge and colour coders, with no container and no header &mdash;
    an idealised lower bound, competing against real files.</p>
  <div class="tw"><table>
    <thead><tr><th>encoder</th><th>setting</th><th class="num">bytes</th><th></th>
      <th class="num">vs WebP</th></tr></thead>
    <tbody>{lossless_rows()}</tbody>
  </table></div>
  <div class="why" style="margin-top:20px">
    <div><h4>The partition degenerates</h4><p>To be exact, every run of identical pixels becomes its own
      region: <code>{EXACT['regions']:,}</code> of them across {PIX4K:,} pixels, averaging
      <code>{EXACT['px_per_region']}</code> px. There is no geometry left to exploit.</p></div>
    <div><h4>So the walls become free</h4><p>{EXACT['cracks']:,} crack edges cost
      <code>{EXACT['bits_per_edge']}</code> bits each &mdash; when almost every pixel boundary is a wall,
      the wall map is nearly constant and compresses to {kb(EXACT['walls'])}.</p></div>
    <div><h4>And colour is the whole bill</h4><p>{kb(EXACT['colours'])} of {kb(12_159_385)}. At this point
      the coder <em>is</em> a raster coder, with one predictor where WebP has fourteen, plus a colour
      transform, LZ77 with 2D distances, and a colour cache.</p></div>
  </div>
</section>

<section>
  <h2>1:1, across the whole rate axis</h2>
  <p class="sub">Drag the slider to change the <strong>file size</strong>; drag inside a window to wipe
    between the two coders. Every step is byte-matched to within 2.5%, so the only thing left to compare
    is the picture &mdash; and the axis ends where both coders are bit-exact and only the bill differs.</p>

  <div class="rate">
    <div class="top">
      <div><div class="lab">file size</div><div class="now" id="nowsize">&mdash;</div></div>
      <div class="toggle" role="group" aria-label="what to show on the left">
        <button data-left="webp" aria-pressed="true">vs WebP</button>
        <button data-left="orig" aria-pressed="false">vs the original</button>
      </div>
    </div>
    <input type="range" id="rate" min="1" max="{len(STEPS)}" value="3" step="1"
           aria-label="file size step">
    <div class="ticks">{"".join(f"<span>{kb(s['sbytes'])}</span>" for s in STEPS)}</div>
    <div class="sides">
      <div class="side l"><div class="who" id="lwho">WebP</div>
        <div class="big" id="lbig">&mdash;</div><div class="sm" id="lsm">&mdash;</div></div>
      <div class="side r"><div class="who">shape coder</div>
        <div class="big" id="rbig">&mdash;</div><div class="sm" id="rsm">&mdash;</div></div>
    </div>
  </div>
  <p class="sub" id="verdict" style="margin:14px 0 28px"></p>

  {window_blocks()}

  <div class="tw"><table>
    <thead><tr><th class="num">step</th><th class="num">shapes</th><th class="num">dB</th>
      <th class="num dim">regions</th><th class="num">WebP</th><th class="num">dB</th>
      <th class="dim">setting</th><th class="num">shapes vs WebP</th></tr></thead>
    <tbody>{step_table()}</tbody>
  </table></div>
  <p class="sub" style="margin-top:20px">Steps 1 and 2 buy their bytes a second way. At 3840&times;2160
    <span class="mono">cwebp</span> bottoms out at <span class="mono">q0</span> = 85,102&nbsp;B, so a
    20&nbsp;KB file cannot be reached by turning quality down &mdash; it is reached the way small images
    are actually delivered, by encoding at a lower resolution and letting the client scale it up, which
    still puts 3840&times;2160 pixels on screen. An earlier version of this page pinned those two steps to
    the native floor and called them the one place the region coder goes where WebP cannot. That was
    wrong, and it is the tenth claim this study has had to retract: at
    <span class="mono">-q 18 -resize 960 540</span> WebP lands on the same 20&nbsp;KB and wins by
    <b>2.55&nbsp;dB</b>. The deficit is <b>U-shaped</b> &mdash; worst at the very bottom of the axis and
    at the very top, least bad in the middle &mdash; so the low-rate band every earlier round called this
    idea's best hope is where it does worst.</p>
</section>

<section>
  <h2>The 512&times;288 win does not survive</h2>
  <p class="sub">The same photograph, resampled to four sizes, each encoded by both coders and read at a
    fixed PSNR so the same picture quality is bought at every rung. Positive means the shape coder needs
    more bytes than WebP for the same result.</p>
  <div class="chart">
    <canvas id="cv" width="1100" height="340"></canvas>
    <div class="legend">
      {"".join(f'<span><i style="background:hsl({200 - k * 34} 70% 50%)"></i>{p:.1f} dB</span>'
               for k, (p, _) in enumerate(LADDER))}
    </div>
  </div>
  <div class="tw" style="margin-top:20px"><table>{ladder_table()}</table></div>
  <p class="sub" style="margin-top:20px">Boundary length grows with the linear dimension and area with its
    square, so a 16&times; pixel count should only be a 4&times; boundary cost &mdash; the isoperimetric
    term that sinks the shape coder ought to <em>weaken</em> at 4K. It does. The deficit still grows,
    because WebP's own cost per pixel falls faster: at 4K it has more correlated neighbourhood to predict
    from, longer LZ77 matches, and more tiles to segment its entropy image over, while the shape coder
    gains only a slightly better area-to-perimeter ratio.</p>
</section>

<section>
  <h2>Where the bytes actually go</h2>
  <p class="sub">Whole-image PSNR at the 200&nbsp;KB operating point says WebP is 0.94 dB ahead. Measured
    on 448&times;448 windows instead, it says something more specific.</p>
  <div class="tw"><table>
    <thead><tr><th>window</th><th>content</th><th class="num">shapes</th><th class="num">WebP</th>
      <th class="num">gap</th></tr></thead>
    <tbody>{"".join(
      f'<tr><td>{"<b>" + n + "</b>" if n == "Sky" else n}</td><td class="dim">{c}</td>'
      f'<td class="mono num">{a:.2f} dB</td><td class="mono num">{b:.2f} dB</td>'
      f'<td class="mono num bad">{a - b:+.2f} dB</td></tr>' for n, c, a, b in PER_WINDOW)}</tbody>
  </table></div>
  <div class="why" style="margin-top:20px">
    <div><h4>Texture is a near-tie</h4><p>On the ridge and the snowfield the shape coder is within
      0.6 dB. Sharp edges and busy detail are what a region model is for, and it holds its own.</p></div>
    <div><h4>The gradient is the loss</h4><p>The sky costs it <b>5.39 dB</b>. A piecewise-constant model
      has one way to draw a smooth ramp &mdash; stack flat bands and pay for every boundary. A DCT spends
      one low-frequency coefficient.</p></div>
    <div><h4>Which is fixable, and still not enough</h4><p>Report 04 priced the affine variant: linear
      colour per region closes most of the gradient gap and costs more in coefficients than it saves.
      The bands are a symptom; the explicit boundary is the disease.</p></div>
  </div>
</section>

<footer>
  <p>Source: the macOS Sierra wallpaper at 3840&times;2160, decoded from a q95 JPEG &mdash; so the
  &ldquo;original&rdquo; carries JPEG artefacts and film grain that a downscale would have averaged away.
  Both coders see the identical PNG. Shape-coder bytes are contour + colour with no container; WebP, AVIF,
  JPEG XL and PNG bytes are whole files. PSNR is this study's RGB definition throughout. Codec settings:
  <span class="mono">cwebp -m 6</span>, <span class="mono">avifenc -s 6</span>,
  <span class="mono">cjxl -e 7</span>. A third window (snow and rock) is measured and tabulated above but
  not embedded, to keep this page's weight down.</p>
</footer>
</div>

<script>
const STEPS = {json.dumps(STEPS)};
const WSTATS = {json.dumps(WSTATS)};
let step = 3, leftKind = "webp";

const fmtB = n => n >= 1048576 ? (n / 1048576).toFixed(2) + " MB"
                               : Math.round(n / 1024).toLocaleString() + " KB";
const el = id => document.getElementById(id);
const NB = "\\u00a0";

function apply() {{
  const s = STEPS[step - 1];
  const orig = leftKind === "orig";
  const wExact = s.wpsnr >= 99;
  const sExact = s.spsnr >= 99;
  const sdb = sExact ? "exact" : s.spsnr.toFixed(2) + NB + "dB";

  el("nowsize").textContent = fmtB(s.sbytes);
  el("lwho").textContent = orig ? "the original" : "WebP";
  el("lbig").textContent = orig ? "lossless reference" : s.wbytes.toLocaleString() + " B";
  el("lsm").textContent = orig ? "the exact pixels both coders were given"
    : (wExact ? "exact \\u00b7 " + s.wq : s.wpsnr.toFixed(2) + " dB whole image \\u00b7 " + s.wq);
  el("rbig").textContent = s.sbytes.toLocaleString() + " B";
  el("rsm").textContent = (sExact ? "exact" : s.spsnr.toFixed(2) + " dB whole image") + " \\u00b7 "
    + s.regions.toLocaleString() + " regions";

  let v;
  if (orig) {{
    v = sExact
      ? "Left is the untouched source, and so is the right: this is the exact region partition, which "
        + "reconstructs every one of the 8,294,400 pixels. The wipe has nothing to show. The cost is "
        + fmtB(s.sbytes) + ", against " + fmtB(s.wbytes) + " for a WebP that is equally perfect."
      : "Left is the untouched source. The wipe shows how far the shape coder is from the truth at "
        + fmtB(s.sbytes) + " \\u2014 " + s.spsnr.toFixed(2) + " dB over the whole image.";
  }} else if (wExact && sExact) {{
    v = "<b>Both coders are bit-exact here \\u2014 only the bill differs.</b> The exact region partition "
      + "costs " + s.sbytes.toLocaleString() + NB + "B against WebP's " + s.wbytes.toLocaleString() + NB
      + "B, <b>" + (s.sbytes / s.wbytes).toFixed(2) + "\\u00d7</b>. At the exact end there is no geometry "
      + "left to exploit: the partition is down to 1.3 pixels per region, so the region coder is a raster "
      + "coder with one predictor where WebP has fourteen.";
  }} else if (s.resampled) {{
    v = "<b>Below cwebp's native floor</b> of q0 = 85,102" + NB + "B, so this WebP is encoded at "
      + s.wq.split("@")[1].replace("x", "\\u00d7") + " and scaled up \\u2014 which is how a file this "
      + "small is actually delivered, and still 3840\\u00d72160 pixels on screen. Byte-matched to within "
      + (100 * Math.abs(s.sbytes - s.wbytes) / s.wbytes).toFixed(1) + "%, WebP is <b>"
      + Math.abs(s.spsnr - s.wpsnr).toFixed(2) + NB + "dB ahead</b>. An earlier version of this page "
      + "claimed WebP could not reach this size at all; see report 06 #10.";
  }} else if (wExact) {{
    v = "<b>WebP is bit-exact here, in fewer bytes.</b> " + s.wbytes.toLocaleString() + NB
      + "B for a perfect reconstruction, against the shape coder's " + s.sbytes.toLocaleString() + NB
      + "B for " + sdb + ".";
  }} else {{
    const d = s.spsnr - s.wpsnr;
    v = "Byte-matched to within " + (100 * Math.abs(s.sbytes - s.wbytes) / s.wbytes).toFixed(1)
      + "%. WebP is <b>" + Math.abs(d).toFixed(2) + NB + "dB " + (d < 0 ? "ahead" : "behind")
      + "</b> on the whole image.";
  }}
  el("verdict").innerHTML = v;

  const lkey = orig ? "orig" : "s" + s.i + "_webp";
  const rkey = "s" + s.i + "_shapes";
  document.querySelectorAll(".win").forEach(win => {{
    const frame = win.querySelector(".frame");
    frame.querySelectorAll("img.lyr").forEach(img => {{
      img.classList.toggle("showL", img.dataset.k === lkey);
      img.classList.toggle("showR", img.dataset.k === rkey);
    }});
    const st = WSTATS[win.dataset.win][step - 1];
    const here = orig ? null : (wExact ? null : st.webp);

    frame.querySelector(".tagL").innerHTML = "<b>" + (orig ? "original" : "WebP") + "</b><i>"
      + (orig ? "lossless reference" : s.wbytes.toLocaleString() + " B")
      + (orig ? "" : "<br>" + (wExact ? "exact" : st.webp.toFixed(2) + " dB here")) + "</i>";
    frame.querySelector(".tagR").innerHTML = "<b>shape coder</b><i>"
      + s.sbytes.toLocaleString() + " B<br>"
      + (sExact ? "exact" : st.shapes.toFixed(2) + " dB here") + "</i>";

    win.querySelector('[data-role="lname"]').textContent = orig ? "original" : "WebP";
    win.querySelector('[data-role="ldb"]').textContent =
      orig ? "exact" : (wExact ? "exact" : st.webp.toFixed(2) + " dB");
    win.querySelector('[data-role="rdb"]').textContent =
      sExact ? "exact" : st.shapes.toFixed(2) + " dB";
    const dd = win.querySelector('[data-role="ddb"]');
    if (here === null) {{ dd.textContent = "\\u2014"; dd.className = ""; }}
    else {{
      const d = st.shapes - here;
      dd.textContent = (d >= 0 ? "+" : "") + d.toFixed(2) + " dB";
      dd.className = d >= 0 ? "good" : "bad";
    }}
  }});
}}

function wire(frame) {{
  const handle = frame.querySelector(".handle");
  const set = pct => {{
    pct = Math.max(0, Math.min(100, pct));
    frame.style.setProperty("--wipe", pct + "%");
    handle.style.left = pct + "%";
    handle.setAttribute("aria-valuenow", Math.round(pct));
  }};
  const at = e => {{
    const r = frame.getBoundingClientRect();
    set((e.clientX - r.left) / r.width * 100);
  }};
  frame.addEventListener("pointerdown", e => {{ frame.setPointerCapture(e.pointerId); at(e); }});
  frame.addEventListener("pointermove", e => {{ if (e.buttons) at(e); }});
  handle.addEventListener("keydown", e => {{
    const d = e.shiftKey ? 10 : 2, now = parseFloat(handle.style.left) || 50;
    if (e.key === "ArrowLeft") {{ set(now - d); e.preventDefault(); }}
    if (e.key === "ArrowRight") {{ set(now + d); e.preventDefault(); }}
  }});
  set(50);
}}

el("rate").addEventListener("input", e => {{ step = +e.target.value; apply(); }});
document.querySelectorAll(".toggle button").forEach(b =>
  b.addEventListener("click", () => {{
    leftKind = b.dataset.left;
    document.querySelectorAll(".toggle button").forEach(x =>
      x.setAttribute("aria-pressed", x === b ? "true" : "false"));
    apply();
  }}));
// Layers that show the same pixels as another layer borrow its src rather than carrying a second copy.
document.querySelectorAll("img.lyr[data-src-from]").forEach(img => {{
  const src = img.closest(".frame").querySelector('img.lyr[data-k="' + img.dataset.srcFrom + '"]');
  if (src) img.src = src.src;
}});
document.querySelectorAll(".frame").forEach(wire);
apply();

// ---- the ladder chart ----
const LADDER = {json.dumps([[p, [[w, s, c] for w, s, c in pts]] for p, pts in LADDER])};
const cv = el("cv"), cx = cv.getContext("2d");
const css = v => getComputedStyle(document.documentElement).getPropertyValue(v).trim();

function draw() {{
  const dpr = devicePixelRatio || 1, W = cv.clientWidth, H = 340;
  cv.width = W * dpr; cv.height = H * dpr; cx.setTransform(dpr, 0, 0, dpr, 0, 0);
  cx.clearRect(0, 0, W, H);
  const L = 62, R = W - 18, T = 18, B = H - 34;
  const xs = [512, 960, 1920, 3840];
  const X = i => L + (R - L) * i / (xs.length - 1);
  const maxY = 80, Y = v => B - (B - T) * v / maxY;
  cx.strokeStyle = css("--grid"); cx.lineWidth = 1;
  cx.fillStyle = css("--ink3"); cx.font = '11px ui-monospace,Menlo,monospace';
  for (let v = 0; v <= maxY; v += 20) {{
    cx.beginPath(); cx.moveTo(L, Y(v)); cx.lineTo(R, Y(v)); cx.stroke();
    cx.textAlign = "right"; cx.textBaseline = "middle"; cx.fillText(v + "%", L - 10, Y(v));
  }}
  cx.strokeStyle = css("--ink3"); cx.setLineDash([3, 3]);
  cx.beginPath(); cx.moveTo(L, Y(0)); cx.lineTo(R, Y(0)); cx.stroke(); cx.setLineDash([]);
  cx.textAlign = "center"; cx.textBaseline = "top"; cx.fillStyle = css("--ink3");
  xs.forEach((w, i) => cx.fillText(w + "\\u00d7" + (w * 9 / 16), X(i), B + 10));
  LADDER.forEach(([psnr, pts], k) => {{
    const col = `hsl(${{200 - k * 34}} 70% 50%)`;
    cx.strokeStyle = col; cx.lineWidth = 2.4; cx.lineJoin = "round"; cx.beginPath();
    pts.forEach(([w, s, c], i) => {{
      const v = 100 * (s - c) / c; i ? cx.lineTo(X(i), Y(v)) : cx.moveTo(X(i), Y(v));
    }});
    cx.stroke(); cx.fillStyle = col;
    pts.forEach(([w, s, c], i) => {{
      const v = 100 * (s - c) / c; cx.beginPath(); cx.arc(X(i), Y(v), 3.2, 0, 7); cx.fill();
    }});
  }});
  cx.save(); cx.translate(15, H / 2); cx.rotate(-Math.PI / 2);
  cx.textAlign = "center"; cx.fillStyle = css("--ink3");
  cx.fillText("extra bytes vs WebP, same PSNR", 0, 0); cx.restore();
}}
draw();
addEventListener("resize", draw);
new MutationObserver(draw).observe(document.documentElement,
  {{ attributes: true, attributeFilter: ["data-theme"] }});
matchMedia("(prefers-color-scheme:dark)").addEventListener("change", draw);
</script>
'''

out = HERE / "hd-mirror.html"
out.write_text(html)
print(f"wrote {out}  {out.stat().st_size / 1048576:.2f} MB")
