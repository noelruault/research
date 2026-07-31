# 44 — A portable detector, and the selection chunk it feeds

**Question.** The format's pitch is addressable regions with stable identity. Getting a *useful* selection into a file means something must decide what the subject is. Can that something be portable, redistributable, and good enough — and what does carrying its answer cost?

**Answer. Yes on all three.** `u2net` (Apache-2.0, ONNX) matches Apple's Vision at **mean IoU 0.9505** across six photographs, at Vision's own speed, and runs anywhere. Snapping its mask onto our partition costs **1.7–5.2% IoU** and stores as **96–168 bytes** in the file.

Data: [`DOD-portable-detector-data.txt`](DOD-portable-detector-data.txt). Target registered before the run: [`DOD-portable-detector.md`](DOD-portable-detector.md).

## Why a portable detector at all

The goal is a **web editor**. That rules out platform-locked detectors — not on licensing, but because they cannot run in a browser. So the detector had to be ONNX, redistributable, and match a strong reference.

Apple's Vision was used as the **reference to measure against**, via its public API, exactly as any benchmark uses a strong incumbent. It is not shipped and cannot be: its weights are licensed to run as part of macOS, not to be redistributed. On a Mac it remains a legitimate optional fast path.

## The result

| model | mean IoU vs Vision | ≥0.90 | warm ms | weights | licence |
|---|---|---|---|---|---|
| **u2net** | **0.9541** | **6/6** | **163–246** | 176 MB | Apache-2.0 |
| isnet-general-use | 0.9525 | 6/6 | 431–502 | 179 MB | not verified |
| birefnet-general | **0.9643** | 6/6 | 6455–8385 | 973 MB | not verified |
| *Apple Vision* | *reference* | — | *~160* | *15.2 MB* | *not redistributable* |

**u2net is the pick.** It matches Vision on output *and* speed while being redistributable. birefnet scores a point higher and is **50× slower at 973 MB** — not worth it on this corpus.

**Speed was deliberately excluded from the pre-registered target**, on the expectation that a portable model would lose to dedicated silicon. It did not. Recorded because the expectation was wrong in our favour, which is the kind of thing that otherwise gets quietly enjoyed instead of written down.

## What snapping costs, and what it buys

The model emits pixels. Our partition turns that into region ids:

| image | IoU(snapped, model) | selection stored |
|---|---|---|
| images | 0.9740 | +96 B |
| images-2 | 0.9480 | +151 B |
| images-3 | 0.9752 | +134 B |
| images-4 | 0.9785 | +136 B |
| images-5 | 0.9703 | +111 B |
| images-6 | 0.9833 | +168 B |

**Costs 1.7–5.2% IoU. Buys a selection that is 222 region ids instead of 27,918 pixels** — addressable, O(regions) to edit, and identical on every client forever.

And **edge fidelity improves**: 11.25 → 12.97 (dog), 12.98 → 15.40 (bobcat), judged against the source as a neutral referee. A model works on a fixed low-resolution analysis grid and upscales; our boundaries are exact on the native lattice. Snapping replaces a resampled boundary with a real one.

## SHPC v3 — the selection chunk

```
… mode(1) · [alphaLen] · selMode(1) · [selLen] · coef(8) · wall · colour · alpha · selection
```

| mode | payload |
|---|---|
| 0 | none |
| 1 | one instance id per region, 0 = background |
| **2** | **confidence byte + producer string + instance ids** |

`lab p4enc <render.png> <out.shpc> [mask.png] [confidence] [producer]`

**Measured:** 11,726 B → 11,822 B (mode 1, **+96 B**) → 11,885 B (mode 2, **+159 B**). Round-trips EXACT. **v1 and v2 files still decode.**

**Mode 2 exists because mode 1 is unfalsifiable.** "Region #4,211 means the same thing everywhere" is only checkable if the file records *what drew it and how sure it was*. Confidence values are real and vary — 0.899, 0.896, 0.952 across three photographs.

## Caveats

- **Vision is the reference, not ground truth.** No human-drawn masks exist for these six, so a model scoring lower could be better. Disagreements were not adjudicated.
- **Six images, one subject type**: a centred subject on a natural background.
- **Licence verified only for u2net** (Apache-2.0, at source) **and rembg** (MIT). isnet and birefnet were measured but never licence-checked, and whether rembg's redistributed `u2net.onnx` is byte-identical to the upstream artefact is unconfirmed. See [`LICENSES.md`](LICENSES.md).
- **`onnxruntime-web` in an actual browser is unmeasured.** These timings are native with a CoreML provider available.
- **One instance only.** The chunk allows 0–255; multi-instance is untested.
