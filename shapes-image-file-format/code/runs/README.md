# Run scripts

Every ad-hoc command behind reports 32–35 and the alpha work (A1, A1b), as scripts, so no number in this directory rests on a shell line that only ever existed in a transcript.

**All bash, never zsh.** `AUTORESEARCH.md`'s invariants say so, and the one time a loop here was written for zsh its `set -- $pair` split wrong and printed a table of zeros.

| script | produces |
|---|---|
| `a1-silhouette.sh` | A1 / A1b — does the merge dissolve a sprite's silhouette? (`DESIGN-ALPHA*.txt`) |
| `32-sprites-vs-codecs.sh` | report 32 — SHPC v2 vs PNG / WebP / AVIF on game sprites, lossless |
| `33-background-removal.sh` | report 33 — matched-fidelity bytes on photos, and the unsupervised flood |
| `34-why-webp-wins.sh` | report 34 — component split, the affine re-run, the wall-variant sweep |
| `35-bgclass.sh` | report 35 — supervised classification, region vs pixel, with the steelman |

## Usage

```sh
./32-sprites-vs-codecs.sh              # writes to $OUT, default $TMPDIR/shpc-runs
OUT=/tmp/mine ./35-bgclass.sh          # override the output directory
```

`common.sh` builds `lab` first and holds the shared paths. External tools are checked for up front and the script exits rather than silently skipping an arm — a missing `cwebp` used to mean an empty column, not an error.

## The corpora

- **Sprites** come from `noelruault/sprites` (`$SPRITES`, defaults to `~/go/src/github.com/noelruault/sprites`). Three prop PNGs — small, and not a corpus.
- **Photographs** for reports 33–35 were supplied for that session and are **not committed**. Point `PHOTOS` at a directory of images to re-run; the sample coordinates in `35-bgclass.sh` are specific to the two images used and will need re-picking for anything else.
- **Kodak-24** is not committed either; fetch from `r0k.us/graphics/kodak/`.
