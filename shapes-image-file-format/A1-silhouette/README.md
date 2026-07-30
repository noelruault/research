# A1 / A1b — silhouette evidence

Research items A1 and A1b from [`../DESIGN-ALPHA.md`](../DESIGN-ALPHA.md). Numbers and commands: [`../DESIGN-ALPHA-A1-data.txt`](../DESIGN-ALPHA-A1-data.txt) and [`../DESIGN-ALPHA-A1b-data.txt`](../DESIGN-ALPHA-A1b-data.txt).

Every image here is four panels, left to right, at 8× nearest-neighbour:

| panel | what it shows |
|---|---|
| 1 | the sprite **as authored** — checkerboard is genuine transparency |
| 2 | **what the pre-A1b pipeline handed the merge** — alpha dropped, transparency premultiplied to black. Kept in the *after* images too, deliberately: it is the picture of the failure, and holding it beside a fixed render is what makes the fix legible. It is no longer what `load()` does |
| 3 | the **merged render** at that mark |
| 4 | panel 3 with **dissolved silhouette crossings in red** — adjacent pixel pairs that cross the true alpha silhouette and come out the same colour, meaning both landed in one region |

Regenerate any of them with:

```
lab silhouette <sprite.png> <render.png> <out.png> 8
```

## The files

| file | mark | dissolved |
|---|---|---|
| `ak74-328regions.png` | 328 regions, before | 50.43% |
| `ak74-330regions-AFTER.png` | 330 regions, after | **0.00%** |
| `bow-476regions.png` | 476 regions, before | 37.32% |
| `bow-473regions-AFTER.png` | 473 regions, after | **0.14%** |
| `bow-1368regions-AFTER.png` | 1,368 regions, after — the finest mark | **0.00%** |
| `pickaxe-160regions.png` | 160 regions, before | 16.30% |
| `pickaxe-160regions-AFTER.png` | 160 regions, after | **0.00%** |

Marks differ by a unit or two between arms because the partitions genuinely differ once alpha is carried; pairs are matched by nearest region count. `pickaxe` is an exact match at 160.

**The one to look at first is `ak74-328regions.png`.** The orange wooden stock keeps its silhouette while every edge of the black gun body is red — same image, same merge, outcome decided purely by whether the edge colour differs from black. That is the whole mechanism in one picture.
