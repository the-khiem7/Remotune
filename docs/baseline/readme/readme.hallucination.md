---
baseline_schema: "2.0"
pack: "readme"
document: "hallucination"
status: "active"
updated: "2026-10-11"
code_ref: "uncommitted"
---

# Remotune README Visual Upgrade - Decision Ledger

## Entry index

| ID | About | Status | Related |
|---|---|---|---|
| RM-D1 | Where the README lives and whether it is promoted | closed | - |
| RM-D2 | README language | closed | - |
| RM-D3 | Which content becomes a figure and which stays text | closed | RM-D5 |
| RM-D4 | Navigation row form | closed | RM-D8 |
| RM-D5 | What the popup figure is | closed | RM-D3, RM-D6 |
| RM-D6 | Visual language of the images | closed | RM-D5 |
| RM-D7 | Render pipeline and where its procedure is written | closed | RM-D11 |
| RM-D8 | Navigation button size | closed | RM-D4, RM-Q2 |
| RM-D9 | Bound on README product claims | closed | RM-D10 |
| RM-D10 | README build section content | closed | RM-D9, RM-Q1 |
| RM-D11 | Three WebP files above the size budget | closed | RM-D7 |
| RM-D12 | CSS reset specificity overrode component margins | lesson | - |
| RM-D13 | SVG node stroke clipped at the viewBox edge | lesson | - |
| RM-Q1 | Docker runbook against the host-native workflow | open | RM-D10 |
| RM-Q2 | GitHub rendering of the navigation row and dark images | open | RM-D8 |

## Closed decisions

### RM-D1: README location and promotion

- Decision: the README is authored at the root of this repository and is not promoted.
- Why: this repository is already the production repository.
- What breaks if ignored: an agent starts a promote-release flow or cuts a semver branch for a file that is already where readers see it.
- Rejected: a README only in a separate production repository - leaves this repository without a source of truth for its own front page.

### RM-D2: README language

- Decision: English.
- Why: the README is read by other teams and by people evaluating the tool.
- What breaks if ignored: a mixed-language front page; image text and alt text no longer match the page.
- Rejected: Vietnamese - the audience is wider than the maintainer.

### RM-D3: Figure or text

- Decision: only sequences and structures become figures: the CRD session workflow, the three status dimensions with the tuning state flow, and the popup. Lookups stay text: the CRD ON/OFF choice table, requirements, build commands, the status lists.
- Why: a lookup read as an image cannot be searched or copied; commands must stay copyable.
- What breaks if ignored: build commands or the choice table end up in images, with alt text as the only text form.
- Rejected: a banner image - the repository has no banner, so the existing application mark `assets/app/remotune-256.png` leads the page; a figure per README section - decoration lowers trust and costs reading time.

### RM-D4: Navigation row form

- Decision: the navigation row under the positioning sentence is seven graphic buttons, each an image link to a README heading, with a dark `<picture>` source.
- Why: required as graphic, not text.
- What breaks if ignored: the page ships a text link row where image buttons were required.
- Rejected: plain text anchors - the first draft; replaced because graphic buttons were required.

### RM-D5: The popup figure is an illustration

- Decision: the popup figure is an HTML illustration built from the strings in `frontend/src/App.vue` and the tokens in `frontend/src/style.css`, scaled 1.5x inside the item, labeled "illustration" in the figure's eyebrow and in its alt text. The CRD detector line is omitted because the stylesheet hides it in the real UI.
- Why: the app's own text is 11 to 13 px, which shrinks to about 8 px at the README column width; a screenshot cannot be taken from the authoring environment.
- What breaks if ignored: a UI label or state changes and the figure keeps showing the old one; nothing compares them.
- Rejected: a screenshot of the running window - not capturable here, and the frameless popup of the latest build is unobserved on the target machine; a 1x full-height mock - unreadable text.

### RM-D6: Visual language

- Decision: accents use the logo gradient from `#102bff` to `#02ceff`; surfaces, text and status colors come from `frontend/src/style.css` for the dark theme and from its light `prefers-color-scheme` rules for the light theme; the font stack is `"Segoe UI Variable", "Segoe UI"`.
- Why: the logo and the application agree, so the images match what a visitor sees after install.
- What breaks if ignored: figures look like another product, or render with a fallback font.
- Rejected: the starter palette of the illustration skill - placeholder values; a web font - the app itself uses the system fonts, and a web font would diverge from it.

### RM-D7: Render pipeline

- Decision: all images render from `assets/source/readme.html` by screenshotting each `.item` through the Playwright MCP tools at CSS scale, once per theme, then `assets/source/finish.py` rounds the corners, draws the border and writes lossless WebP into `assets/readme/`. The procedure is written in `docs/runbook/readme-illustrations.md`, not in a header comment of the source files.
- Why: Playwright MCP was the chosen route; `AGENTS.md` forbids comments in code files, so the render command cannot sit in a comment.
- What breaks if ignored: a comment-bearing source file violates `AGENTS.md`; a render by another browser or machine changes fonts and pixels silently.
- Rejected: the `uvx --from playwright ... render.py` route of the illustration skill - not chosen; copying that script into the repository - untested here, and `finish.py` covers the only step used.

### RM-D8: Navigation button size

- Decision: each button is a 56 CSS px high item sized to its label, rendered at 2x, and shown at `height="32"` so the row of seven fits one line.
- Why: the seven images total about 2775 px of width at 112 px of height; at a displayed height of 34 the row is about 866 px, wider than the README column of about 830 px; at 32 it is about 813 px.
- What breaks if ignored: the row wraps onto two lines.
- Rejected: `height="34"` - wraps at the stated column width; shorter labels - the labels match the README headings.

### RM-D9: Bound on README product claims

- Decision: a README statement about product behavior must trace to the `remotune` pack's verified or implemented material or to the code. The Status and limits section lists unverified configurations explicitly: Windows 10, multiple monitors and secondary taskbars, Explorer restart under an owned override, simultaneous CRD clients, the frameless popup of the latest build, long-running and crash-recovery hardening, the missing-WebView2 path.
- Why: the pack's single-machine evidence does not close questions that vary by Windows version or layout.
- What breaks if ignored: the front page promises support the product has not demonstrated.
- Rejected: a feature-forward README without a limits section - hides what is unobserved.
- Obligation: when the `remotune` pack's evidence changes, this README's Status and limits section changes in the same edit.

### RM-D10: README build section

- Decision: the build section follows the host-native workflow of `Taskfile.yml` and the `remotune` pack (`wails3 dev`, `wails3 task verify`, `wails3 task windows:portable`). It names the exact Wails CLI version and links the Wails installation guide, and writes no CLI install command. It states that the build stops when the destination executable is locked, and does not claim an existing executable is never overwritten.
- Why: `Taskfile.yml` only refuses a locked destination and overwrites an unlocked one; the install command was not verifiable because `go` was not on the PATH of the authoring shell, and `build/config.yml` was unreadable.
- What breaks if ignored: a README line that is wrong on first use, or a claim about overwrite behavior the Taskfile does not implement.
- Rejected: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8` - not run; a Docker-based build section - contradicts the `remotune` pack, see RM-Q1.

### RM-D11: Files above the size budget

- Decision: accept `popup-dark.webp`, `popup.webp` and `states-dark.webp` above the 300 KB per-file soft budget; the whole set stays well under the few-megabyte budget.
- Why: the card shadows and the background gradient dominate the lossless size.
- What breaks if ignored: repeated re-renders add each file to git history again, so the design is settled before further renders.
- Rejected: flattening the background gradient - declined; a lossy re-encode - lossless output is the stated format.

## Lessons

### RM-D12: A zero-specificity reset for component margins

- Mistake: the source stylesheet reset margins with `.item p { margin: 0 }`, then set margins on single-class components such as `.valid`, `.lead` and `.sub`.
- Why it looked reasonable: a reset is normally written first and expected to lose to later component rules.
- What disproved it: `.item p` has higher specificity than a single class, so the component margins never applied; the closing paragraph of the states figure rendered at the left edge with no gutter, found only by looking at the PNG.
- Fix: the reset is `:where(.item h2, .item h3, .item p, .item ol) { margin: 0; }`, which carries zero specificity.

### RM-D13: SVG node stroke clipped at the viewBox edge

- Mistake: the tuning flow SVG used a viewBox exactly as wide as the nodes, `0 0 1053 190`, with the last node ending at x = 1053.
- Why it looked reasonable: the nodes fit the box, and no overflow check flags a stroke.
- What disproved it: half of the 1.5 px stroke fell outside the box, so the Restoring node showed a cut right border in the first render.
- Fix: the viewBox is `-2 0 1057 190`, leaving a stroke-width margin on both sides.

## Open questions

### RM-Q1: Docker runbook against the host-native workflow

- Question: `docs/runbook/docker-build-environment.md` states that all dependency installation, generation, verification and builds must run in Docker, while the `remotune` pack and `Taskfile.yml` describe the host-native workflow and call the Docker files retired. Mark the runbook superseded, or change the README?
- Recommended: mark the runbook superseded in a separate edit; the README stays host-native.
- Blocks: the README build section is right only if host-native is the supported workflow.

### RM-Q2: GitHub rendering

- Question: on github.com, in light and dark and on a phone, does the row of seven buttons fit one line, does `<picture>` select the dark files, and do all seven anchors resolve?
- Recommended: open the README on the hosting branch in each mode and fix the button height or the markup if the row wraps.
- Blocks: the claim that the README renders as designed.
