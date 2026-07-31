#!/usr/bin/env python3
"""Build RESULTS.html: every experiment, every version, every image, self-contained.

Regenerate rather than hand-edit:  python3 code/dashboard/genall.py
Images are downscaled to WIDTH and embedded base64 so the page works offline from the folder.
"""
import base64, subprocess, os, sys, tempfile, html

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
ROOT = os.path.join(ROOT, "shapes-image-file-format") if not ROOT.endswith("shapes-image-file-format") else ROOT
WIDTH = 1600

def img(rel, cap):
    """Downscale to WIDTH and inline as base64. Missing files become a visible gap, never a silent one."""
    p = os.path.join(ROOT, rel)
    if not os.path.exists(p):
        return f'<div class="missing">missing: {html.escape(rel)}</div>'
    with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as t:
        tmp = t.name
    subprocess.run(["sips", "-Z", str(WIDTH), p, "--out", tmp],
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
    src = tmp if os.path.exists(tmp) and os.path.getsize(tmp) else p
    b64 = base64.b64encode(open(src, "rb").read()).decode()
    try: os.unlink(tmp)
    except OSError: pass
    return (f'<figure><img loading="lazy" src="data:image/png;base64,{b64}" alt="{html.escape(cap)}">'
            f'<figcaption>{cap}</figcaption></figure>')

def sec(n, title, verdict, body, tone="neutral"):
    return f'''<section id="r{n}"><h2><span class="num">{n}</span>{title}</h2>
<p class="verdict {tone}">{verdict}</p>{body}</section>'''

def table(head, rows, hi=()):
    h = "".join(f"<th>{c}</th>" for c in head)
    r = ""
    for row in rows:
        cls = ' class="hi"' if row[0] in hi else ""
        r += "<tr%s>%s</tr>" % (cls, "".join(f"<td>{c}</td>" for c in row))
    return f'<div class="tw"><table><thead><tr>{h}</tr></thead><tbody>{r}</tbody></table></div>'

P = []
P.append(sec("32", "SHPC v2 vs shipping codecs, on game sprites",
  "1 win, 2 losses vs WebP lossless. 3 of 3 vs PNG and AVIF. Not a general win.",
  table(["sprite","ours .shpc","WebP-ll","PNG (Go)","AVIF-ll"],
    [["ak74","<b>1,831</b>","1,938 <i>−5.5%</i>","2,617 <i>−30.0%</i>","3,605 <i>−49.2%</i>"],
     ["bow","1,310","1,054 <i>+24.3%</i>","2,397 <i>−45.4%</i>","3,114 <i>−57.9%</i>"],
     ["pickaxe","345","218 <i>+58.3%</i>","466 <i>−26.0%</i>","1,242 <i>−72.2%</i>"]])
  + "<p>Lossless — the finest mark has nothing to merge, so every arm stores identical pixels. Verified by pointing <code>p4dec</code> at the <b>original sprite</b>, not the render.</p>"))

P.append(sec("A1", "Alpha: does the merge dissolve a sprite's silhouette?",
  "Yes, 16–62% — and the merge was never the defect. load() dropped alpha and premultiplication turned transparency into black.",
  img("A1-silhouette/ak74-328regions.png","<b>Before.</b> Authored · what the merge received · render · red = dissolved. The orange stock keeps its outline; every black edge loses it. Same image, same merge — decided purely by whether the edge colour differs from black.")
  + img("A1-silhouette/ak74-330regions-AFTER.png","<b>After (A1b).</b> Alpha carried through load → merge → container → decode. 0.00% dissolved.")
  + table(["sprite","before","after"],
    [["ak74","36.75 – 62.39%","<b>0.00% every mark</b>"],
     ["bow","37.32 – 52.02%","<b>0.00%</b> except coarsest (10.66%)"],
     ["pickaxe","16.30%","<b>0.00%</b>"]])
  + "<p>Opaque baselines stayed <b>byte-identical across all 13 marks</b> — a constant 4th SSE channel contributes exactly zero. Verified against a pre-change binary in a worktree.</p>", "good"))

P.append(sec("33", "Background removal: unsupervised flood",
  "Failed on both substrates. No tolerance separates the subject. The mask mechanics won; the decision did not.",
  img("33-bgcut/dog-bgcut-tol55.png","At tolerance 55: background mostly gone, but the dog's belly and legs went with it. At 28 the background survives. There is no setting in between.")
  + "<p><b>Superseded in part by report 35</b> — the failure was the algorithm's, not the representation's.</p>", "bad"))

P.append(sec("34", "Why WebP wins, decomposed",
  "79.3% of our file is geometry. Our wall chunk alone is +29.6% against WebP's entire file.",
  table(["component","bytes","share"],
    [["<b>wall chunk</b>","<b>9,302</b>","<b>79.3%</b>"],["colour chunk","2,395","20.4%"],["header","24","0.2%"]])
  + "<p>WebP at matched fidelity: <b>7,176 B</b>. With colour entirely free we still lose by 30.0%. Parity needs cutting <b>48.9% of the wall bill</b> — nothing measured or parked is that size.</p>"
  + "<p><b>P-08 affine colour revived and closed negative:</b> loses at every operating point, up to 1.9×. The plane coefficients cost 9.2 KB against the boundary's 6.3 KB.</p>"
  + "<p>Against the baseline that matches the product — WebP + a region map — <b>ours is 28.8% smaller</b>, reproducing report 24 off-corpus.</p>"))

P.append(sec("35", "Supervised colour classification",
  "Chromatic separation works. The cleanliness headline was retracted (falsification #14); cost and edge fidelity survive.",
  img("35-bgclass/dog-7keep-11remove.png","Grass gone, dog intact — what no tolerance achieved in report 33.")
  + table(["metric","region","pixel"],
    [["decisions","<b>1,229</b>","305,532"],
     ["blobs, no filter","77","451"],
     ["blobs, 5×5 filter","61","111"],
     ["edge dE (neutral referee)","<b>4.12</b>","3.05"]])
  + '<p class="retract"><b>Retracted:</b> the 3.5–5.9× fragmentation claim. The pixel arm had no cleanup pass and nobody ships a raw per-pixel mask. Given a median filter on both arms it falls to 1.0–1.7×. A reader asked whether the arms were on equal conditions. They were not.</p>'))

P.append(sec("36–37", "Tone-curve preprocessing, and cross-segmentation",
  "Flattening helps — and helps the pixel arm most. Segmenting one image to colour another is free for a mask, catastrophic for a file.",
  img("36-preprocess/dogpop-cut.png","<b>Black point +100.</b> Grass and sky gone cleanly — and holes in the dog's face, because the boost clipped its nose and the sky both to rgb(0,0,0).")
  + img("37-crosseg/nobp-cut.png","<b>Black point 0.</b> Face survives, a block of dark trees comes with it. Same collision, relocated.")
  + table(["partition from","regions","PSNR","bytes"],
    [["flattened (cross)","1,296","<b>28.44</b>","12,409"],["original (baseline)","1,141","<b>39.27</b>","13,911"]])
  + "<p><b>−10.82 dB to save 10.8% bytes.</b> A partition must match the colours it will carry.</p>"
  + '<p class="retract"><b>Withdrawn:</b> report 36 compared edge dE <i>across</i> images. dE scales with the image&#39;s own contrast, so the rise was the tone curve, not mask quality. Valid between arms on one image only.</p>'))

P.append(sec("39", "Connectivity: fill holes, drop what is not attached",
  "The foreground collapses to exactly one piece — on both images and both substrates.",
  img("39-connectivity/bobcat-connected.png","Bobcat: dark markings filled, background gone. The branch survives because it touches the cat.")
  + img("39-connectivity/dog-connected.png","Dog: the detached tree blobs are gone. The mass behind the ear survives — it is connected.")
  + table(["arm","bg blobs","fg blobs"],
    [["bobcat region","133 → <b>10</b>","11 → <b>1</b>"],["bobcat pixel","344 → <b>8</b>","191 → <b>1</b>"],
     ["dog region","44 → <b>4</b>","33 → <b>1</b>"],["dog pixel","187 → <b>5</b>","264 → <b>1</b>"]])
  + "<p><b>Not a format advantage</b> — connectivity is a graph operation any substrate can run, and the arms end level. Rollback with <code>CONN=0</code>.</p>", "good"))

P.append(sec("40", "Apple Lift Subject as the selector, this format as the substrate",
  "Vision solves the semantic half in 150 ms with zero examples. Snapping its mask to our regions costs 2–3% IoU and improves edge fidelity.",
  img("40-vision/dog-vision-mask.png","Vision's mask. Zero examples, zero tuning, 150 ms.")
  + img("40-vision/dog-snap.png","Source · Vision's cut · snapped to our regions · disagreement. The disagreement is a one-pixel outline.")
  + table(["mask","selection is","edge dE"],
    [["dog — model","27,918 px","11.25"],["dog — snapped","<b>222 / 1,229 regions</b>","<b>12.97</b>"],
     ["bobcat — model","79,909 px","12.98"],["bobcat — snapped","<b>1,208 / 1,433 regions</b>","<b>15.40</b>"]])
  + "<p><b>IoU(snapped, model) = 0.9740 (dog), 0.9833 (bobcat).</b> The snap costs 1.7–2.6% and turns a 27,918-pixel selection into 222 addressable region ids — and the edge lands on <i>larger</i> real colour steps than the model's own output.</p>"
  + "<p><b>IoU is against the model, not ground truth.</b> It measures what snapping costs, not whether either is correct. No GT masks exist yet.</p>", "good"))

body = "\n".join(P)
open(os.path.join(ROOT, "RESULTS.html"), "w").write(f'''<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>shapes-image-file-format — all results</title>
<style>
:root{{--bg:#fff;--fg:#16181d;--mut:#5b6472;--line:#e3e6ea;--card:#f7f8fa;--good:#0a7d3f;--bad:#b3261e;--warn:#8a5a00;--acc:#1a4fd6}}
@media(prefers-color-scheme:dark){{:root{{--bg:#0f1216;--fg:#e6e9ee;--mut:#9aa4b2;--line:#232830;--card:#161a20;--good:#4ade80;--bad:#f87171;--warn:#fbbf24;--acc:#7aa2ff}}}}
:root[data-theme=dark]{{--bg:#0f1216;--fg:#e6e9ee;--mut:#9aa4b2;--line:#232830;--card:#161a20;--good:#4ade80;--bad:#f87171;--warn:#fbbf24;--acc:#7aa2ff}}
:root[data-theme=light]{{--bg:#fff;--fg:#16181d;--mut:#5b6472;--line:#e3e6ea;--card:#f7f8fa;--good:#0a7d3f;--bad:#b3261e;--warn:#8a5a00;--acc:#1a4fd6}}
*{{box-sizing:border-box}}body{{margin:0;background:var(--bg);color:var(--fg);font:16px/1.6 ui-sans-serif,-apple-system,system-ui,sans-serif}}
.wrap{{max-width:1180px;margin:0 auto;padding:0 20px 80px}}
header{{padding:56px 0 28px;border-bottom:1px solid var(--line);margin-bottom:8px}}
h1{{font-size:2rem;margin:0 0 10px;letter-spacing:-.02em}}
.sub{{color:var(--mut);max-width:70ch}}
nav{{position:sticky;top:0;background:var(--bg);border-bottom:1px solid var(--line);padding:10px 0;z-index:9;margin-bottom:28px}}
nav a{{color:var(--acc);text-decoration:none;margin-right:16px;font-size:.9rem;white-space:nowrap}}
nav a:hover{{text-decoration:underline}}
section{{padding:34px 0;border-bottom:1px solid var(--line)}}
h2{{font-size:1.3rem;margin:0 0 10px;display:flex;gap:12px;align-items:baseline}}
.num{{font:600 .75rem/1 ui-monospace,monospace;color:var(--bg);background:var(--fg);padding:5px 8px;border-radius:5px;flex:none}}
.verdict{{font-size:1.02rem;margin:0 0 18px;padding-left:14px;border-left:3px solid var(--mut);color:var(--fg)}}
.verdict.good{{border-color:var(--good)}}.verdict.bad{{border-color:var(--bad)}}
figure{{margin:20px 0;background:var(--card);border:1px solid var(--line);border-radius:10px;padding:10px}}
figure img{{width:100%;height:auto;display:block;border-radius:6px}}
figcaption{{color:var(--mut);font-size:.87rem;margin-top:9px;line-height:1.5}}
.tw{{overflow-x:auto;margin:16px 0}}table{{border-collapse:collapse;width:100%;font-size:.92rem;min-width:440px}}
th,td{{text-align:left;padding:8px 12px;border-bottom:1px solid var(--line);white-space:nowrap}}
th{{color:var(--mut);font-weight:600;font-size:.8rem;text-transform:uppercase;letter-spacing:.04em}}
td i{{font-style:normal;color:var(--mut)}}
.retract{{background:var(--card);border-left:3px solid var(--warn);padding:12px 14px;border-radius:0 8px 8px 0;font-size:.93rem}}
.missing{{color:var(--bad);font-family:ui-monospace,monospace;font-size:.85rem;padding:10px}}
code{{background:var(--card);padding:2px 6px;border-radius:4px;font-size:.88em}}
.tags{{display:flex;gap:8px;flex-wrap:wrap;margin:18px 0 0}}
.tag{{background:var(--card);border:1px solid var(--line);border-radius:20px;padding:5px 13px;font-size:.85rem}}
footer{{color:var(--mut);font-size:.87rem;padding:34px 0}}
</style></head><body><div class="wrap">
<header><h1>shapes-image-file-format — all results</h1>
<p class="sub">Every experiment in the background-removal and alpha lines, with the retractions kept in place. A poor image codec and a good structured-image format; this page shows both, including the claims that did not survive.</p>
<div class="tags"><span class="tag">v0.1.0 — research record complete</span><span class="tag">v0.2.0 — SHPC v2, alpha works</span><span class="tag">v0.3.0 — clean subject cutout</span><span class="tag">branch <code>vision-liftsubject</code></span></div>
</header>
<nav><a href="#r32">32 sprites</a><a href="#rA1">alpha</a><a href="#r33">33 flood</a><a href="#r34">34 why WebP</a><a href="#r35">35 supervised</a><a href="#r36–37">36–37 preprocess</a><a href="#r39">39 connectivity</a><a href="#r40">40 Vision</a></nav>
{body}
<footer><p><b>Every number here has a <code>*-data.txt</code> companion</b> holding raw output and the command that produced it. Scripts: <code>code/runs/</code>. Method and confounds: <code>METHODOLOGY.md</code>. Falsifications: <code>06-corrections-and-falsifications.md</code>.</p>
<p>Regenerate: <code>python3 code/dashboard/genall.py</code></p></footer>
</div></body></html>''')
print("wrote RESULTS.html", os.path.getsize(os.path.join(ROOT,"RESULTS.html"))//1024, "KB")
