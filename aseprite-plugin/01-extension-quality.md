# 01 — Aseprite extension quality

How the best public Aseprite extensions are structured, and the patterns to adopt
so ours reads like a polished tool rather than a hobby script. Claims are verified
against the official `aseprite/api` and `aseprite/docs` repos and the Aseprite
C++ source; items relying on a search summary rather than a direct read are
flagged.

## 1. The manifest (`package.json`)

An extension is a folder whose root holds `package.json`. Verified fields
([api/plugin.md](https://github.com/aseprite/api/blob/main/api/plugin.md)):

```json
{
  "name": "pixelize",
  "displayName": "Pixelize",
  "description": "...",
  "version": "0.1.0",
  "author": { "name": "...", "email": "...", "url": "..." },
  "publisher": "noelruault",
  "license": "MIT",
  "categories": ["Scripts"],
  "contributes": { "scripts": [ { "path": "./pixelize.lua" } ] }
}
```

- Load-bearing fields: `name`, `version`, `contributes`. The rest is metadata
  shown in *Edit → Preferences → Extensions*.
- `version` is a plain string; Aseprite parses it into a `Version` object so it
  can be compared against `app.version`.
- `contributes` can carry more than scripts:
  [extensions.md](https://github.com/aseprite/docs/blob/main/extensions.md) lists
  `scripts`, `keys`, `palettes`, `languages`, `themes`, `dithering-matrices`.
  Bundling our palettes as `contributes.palettes` is an option, though the
  binary already embeds them — keep them in one place (the binary) to avoid drift.
- `keys` points to an `.aseprite-keys` XML file so a shortcut ships with the
  extension. *(XML shape from a search summary of the official Keys doc, not a
  direct read — verify before relying on it.)*

## 2. Plugin lifecycle

A script under `contributes.scripts` defines top-level `init(plugin)` and optional
`exit(plugin)` ([api/plugin.md](https://github.com/aseprite/api/blob/main/api/plugin.md)):

- `init(plugin)` runs at startup — register commands/menus here.
- `exit(plugin)` runs at shutdown.
- `plugin.preferences` is a Lua table whose contents **persist across sessions
  automatically** — the idiomatic settings store, no manual file I/O. (Our
  scaffold already uses this.)
- `plugin.path` is the install dir — use it to locate bundled assets (e.g. a
  bundled binary under `bin/`).

Registration API:

- `plugin:newCommand{ id, title, group, onclick, onenabled, onchecked }`.
  `onenabled` returns a bool to grey-out the item; `onchecked` drives a menu
  checkmark, re-evaluated each time the menu opens.
- `plugin:newMenuGroup{ id, title, group }` for a submenu; `plugin:newMenuSeparator{ group }`.
- `group` references an existing menu group id from Aseprite's
  [`data/gui.xml`](https://github.com/aseprite/aseprite/blob/main/data/gui.xml).
  Our scaffold uses `group = "sprite_color"`; confirm the id against `gui.xml`
  for the target version.

Requires Aseprite **v1.2.18+** (plugins); the scripting API itself exists since
v1.2.10.

## 3. Dialog UI idioms

`Dialog{ title, parent, onclose, notitlebar, resizeable }`; returns `nil` when no
UI is available, so guard with `app.isUIAvailable` for `--batch` safety
([api/dialog.md](https://github.com/aseprite/api/blob/main/api/dialog.md)).

Widgets are method-chainable, each keyed by `id`: `:button :check :radio :entry
:number{decimals} :slider{min,max,value} :combobox{option,options} :color :shades
:file{open,save,filetypes} :label :separator :newrow :tab/:endtabs`.

The two idioms that make a dialog feel native:

- **`dlg.data`** — a table keyed by widget `id`; types map per widget
  (check→bool, slider→int, combobox→string, color→Color, …). Read it after
  `dlg:show()`.
- **Reactive UI** — inside an `onchange`/`onclick` callback, read `dlg.data` and
  call `dlg:modify{ id=..., visible=, enabled=, text= }` to show/hide/update other
  widgets live. This is how we should wire "show the palette-file picker only when
  Palette = (custom file…)" and the future live preview.
- `dlg:show{ wait=false }` runs non-modal (needed for live preview that re-renders
  as sliders move); `dlg:show{ bounds=dlg.bounds }` restores window position.

## 4. The script-security sandbox (read from source)

This is the most folklore-ridden area, so it was read from the implementation
([`security.cpp`](https://github.com/aseprite/aseprite/blob/main/src/app/script/security.cpp) /
`security.h`) and cross-checked against
[api/README.md](https://github.com/aseprite/api/blob/main/README.md):

- `os.exit` and `os.tmpname` are **removed**. `os.execute`, `io.open`, `io.popen`,
  `os.rename`, `package.loadlib`, WebSocket and clipboard access are **wrapped**
  and prompt the user on first call. (Verified verbatim: *"Some functions like
  `os.exit`, `os.tmpname` are not available yet"* and *"Other functions like
  `os.execute` and `io.open` will ask for permissions"*.)
- The grant is **per access-mode** (`FileAccessMode{ Execute, Write, Read,
  OpenSocket, LoadLib, Full }`) and **per resource** (`File, Command, WebSocket,
  Clipboard`). The dialog offers a specific grant *and* a "Give Full Access"
  toggle plus "Don't show again".
- Persistence: a remembered grant is stored in the config under `[script_access]`,
  **keyed by a hash of the script filename** — trust is per-script-file, not
  per-folder.
- **Correction to common folklore:** there is *no* "extensions directory is
  automatically trusted" exemption in the source. Installed extensions go through
  the same `ask_access` path. *(DeepWiki was unreachable to cross-check, but the
  C++ source is authoritative and contradicts that claim.)*

**Implication for our binary-backed extension.** Running the binary
(`os.execute`), reading the captured stderr (`io.open`/read), writing the temp
input + reading the result back (`Image:saveAs` write, `app.open` read) touch
**three** access modes (Execute, Write, Read). Unless the user clicks "Give Full
Access" once, they could see up to three prompts. Two refinements follow:
1. **Document the one-time Full-trust grant** prominently (with a screenshot), as
   the JRiggles template does.
2. **Minimize prompts**: prefer fewer `io`/`os` calls. E.g. capturing stderr to a
   file then reading it is a second permission surface — consider dropping the
   read-back of the error file (or making it best-effort and silent) so the common
   path is just Execute + the Aseprite-native file ops.

## 5. Packaging & distribution

- A `.aseprite-extension` is a **`.zip` renamed**, with `package.json` at the
  archive root (not nested). Install by double-click or *Preferences → Extensions
  → Add Extension* ([extensions.md](https://github.com/aseprite/docs/blob/main/extensions.md)).
  (Our `Makefile`'s `package` target already does this.)
- Cross-platform: use `app.fs.joinPath()` / `app.fs.pathSeparator`, never
  hardcoded separators. (Scaffold complies.)
- Distribution: the Aseprite Community forum and **itch.io** (often
  pay-what-you-want) are the de-facto channels.
- CI: mature extensions automate "zip → publish to itch.io" with `butler` in a
  GitHub Action
  ([butler-push action](https://github.com/marketplace/actions/butler-to-itch)).

## 6. Polish checklist (adopt these)

- [ ] Wrap every sprite mutation in `app.transaction("Pixelize", function() … end)`
      so it is a single undo entry and `error()` rolls back cleanly
      ([api/app.md](https://github.com/aseprite/api/blob/main/api/app.md)).
- [ ] Gate newer APIs on `app.version` / `app.apiVersion`.
- [ ] Guard UI with `app.isUIAvailable` / `Dialog() == nil` so the same code runs
      under `--batch`.
- [ ] Persist settings via `plugin.preferences`; load assets via `plugin.path`.
- [ ] Reactive dialog (`dlg:modify`) instead of a static form.
- [ ] A README "Permissions" section explaining the Full-trust grant.
- [ ] `app.fs.*` for all path work; test on Windows + macOS.
- [ ] Place commands in a sensible `gui.xml` group.

## 7. Reference extensions (all public)

- [JRiggles/Aseprite-Extension-Template](https://github.com/JRiggles/Aseprite-Extension-Template)
  — canonical structure (`extension/` dir, shared lib via submodule, screenshots,
  Permissions section).
- [Astropulse/K-Centroid-Aseprite](https://github.com/Astropulse/K-Centroid-Aseprite)
  — MIT; clean dialog + a real algorithm (see [02](02-quantization-survey.md)).
- [behreajj/AsepriteAddons](https://github.com/behreajj/AsepriteAddons) — large,
  idiomatic color/palette handling; drives the built-in quantizer via `app.command`.

## Sources

- https://github.com/aseprite/api/blob/main/api/plugin.md
- https://github.com/aseprite/api/blob/main/api/dialog.md
- https://github.com/aseprite/api/blob/main/api/app.md
- https://github.com/aseprite/api/blob/main/README.md
- https://github.com/aseprite/aseprite/blob/main/src/app/script/security.cpp (+ `security.h`)
- https://github.com/aseprite/docs/blob/main/extensions.md
- https://github.com/aseprite/docs/blob/main/cli.md
- https://github.com/aseprite/aseprite/blob/main/data/gui.xml
- https://github.com/JRiggles/Aseprite-Extension-Template
- https://github.com/marketplace/actions/butler-to-itch
