# README illustrations

## Purpose

The README images are rendered from one HTML page, so every change is a text edit plus a re-render. Ten items, each in a light and a dark theme, give twenty lossless WebP files: three figures and seven navigation buttons.

| Item id | Files | Shows | Checked against |
|---|---|---|---|
| `workflow` | `workflow.webp`, `workflow-dark.webp` | One CRD session: snapshot, apply, restore | `internal/application/coordinator.go` (`applyLocked`, `releaseLocked`, `restoreLocked`) |
| `states` | `states.webp`, `states-dark.webp` | CRD, Automation and Tuning states, failure states | `internal/application/state.go`, `coordinator.go` |
| `popup` | `popup.webp`, `popup-dark.webp` | Illustration of the main popup, not a screenshot | `frontend/src/App.vue`, `frontend/src/style.css` |
| `nav-why`, `nav-how`, `nav-using`, `nav-safety`, `nav-requirements`, `nav-build`, `nav-status` | `nav-<name>.webp`, `nav-<name>-dark.webp` | The README navigation row, one image link per section | The README headings: an anchor changes when its heading text changes |

Nothing compares an image with the code it describes. When a UI label, a tuning state or a transition changes, edit `assets/source/readme.html` and re-render in the same change, or the README shows a state the product no longer has.

## Files

| Path | Role |
|---|---|
| `assets/source/readme.html` | One `.item` per image |
| `assets/source/shared.css` | Theme tokens and components; light tokens mirror the app's light theme, dark tokens its dark theme |
| `assets/source/shared.js` | Sets the theme from the `?theme=dark` query |
| `assets/source/finish.py` | Rounds the corners, draws the border, writes lossless WebP |
| `assets/readme/` | Rendered output referenced by `README.md` |

## Source contract

- Each `.item` has a unique `id`; the id is the output file name.
- `.item` is 1200 CSS px wide with `zoom: 2`, so an image is 2400 px wide. The README column is about 830 px, so the image keeps 2x density.
- Theming goes only through `html[data-theme="dark"]`. A page opened without `?theme=` renders the light theme.
- Set each item's height explicitly and leave at least 30 px of air under the last element. No element may cross the item's edge.
- Smallest text is 17 CSS px outside the `.ui` mock, where the app's own sizes are multiplied by 1.5.
- Navigation buttons (`.item.btn`) are the exception to the 1200 px width: 56 CSS px high, sized to their label, shown in the README at `height="32"` so the row of seven fits one line in the README column.
- The font stack is `"Segoe UI Variable", "Segoe UI"`, the app's own. These are system fonts: render on a Windows machine that has them, or the fallback font ends up in every image.

## Render

1. Serve the `assets` folder over HTTP so `../app/remotune-256.png` resolves. A file URL does not work.

```powershell
uv run --no-project python -I -m http.server 8765 --bind 127.0.0.1 --directory assets
```

2. In a browser at a 2600 px wide viewport, open `http://127.0.0.1:8765/source/readme.html?theme=light`, then screenshot each `.item` by its id at CSS scale into `.render-tmp/<id>-light.png`. Repeat with `?theme=dark` into `.render-tmp/<id>-dark.png`. The Playwright MCP tools `browser_resize`, `browser_navigate` and `browser_take_screenshot` (`target: #<id>`, `scale: css`) do this.

3. Finish the PNGs into WebP.

```powershell
uv run --no-project --with pillow python -I assets/source/finish.py .render-tmp assets/readme
```

4. Open every WebP, light and dark, and check legibility and contrast. The script cannot judge either.
5. Delete `.render-tmp/` and `.playwright-mcp/`. Both are ignored by git; only the WebP files and the sources are committed.

## Size

`finish.py` warns above 300 KB per file. The `popup` pair is the largest because of its card shadows and background gradient. Simplify gradients before reaching for a lossy re-encode, and say so when a lossy file is used. Each re-render adds its size to git history again, so settle the design before the final render.
