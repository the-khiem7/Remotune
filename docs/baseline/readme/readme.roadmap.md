---
baseline_schema: "2.0"
pack: "readme"
document: "roadmap"
status: "active"
updated: "2026-10-11"
code_ref: "uncommitted"
---

# Remotune README Visual Upgrade - Roadmap

## Current checkpoint

**[IMPLEMENTED]** Phases 1 to 3 are done in the working tree: three figures, seven navigation buttons, the README and the runbook. Phase 4 (this pack, committed with the work) closes the repository side. **[UNVERIFIED]** Phase 5, observation on the hosting site, has not happened.

## Dependency chain

```text
Phase 1: Figures and sources
  └─ Phase 2: Navigation buttons
       └─ Phase 3: README and runbook
            └─ Phase 4: Baseline record and delivery commit
                 └─ Phase 5: Hosting-site verification
```

## Phase 1 - Figures and sources

**Status:** **[IMPLEMENTED]**.

| Item | Output | Checked against |
|---|---|---|
| `workflow` | `workflow.webp`, `workflow-dark.webp` | `internal/application/coordinator.go`: `applyLocked`, `releaseLocked`, `restoreLocked` |
| `states` | `states.webp`, `states-dark.webp` | `internal/application/state.go`, `coordinator.go` |
| `popup` | `popup.webp`, `popup-dark.webp` | `frontend/src/App.vue`, `frontend/src/style.css` |

- Sources: `assets/source/readme.html`, `shared.css`, `shared.js`, `finish.py`, no comments in any of them.
- Decisions: figure set RM-D3, popup as illustration RM-D5, visual language RM-D6, pipeline RM-D7.
- Corrections found by inspecting the first render: RM-D12, RM-D13.

### Evidence

**[IMPLEMENTED]** Each figure was rendered in both themes and viewed at full size after the corrections. The branches drawn in `workflow` and `states` were read against `coordinator.go`: connect applies after persisting the snapshot; a disconnect with the snapshot action restores; any other CRD OFF action applies that profile and restores the taskbar; Pause, Restore Now and Quit restore an owned snapshot; a failed verification is Partial/Error with the snapshot kept; a failed retire is Recovery Required.

| File | Bytes |
|---|---|
| `popup-dark.webp` | 358,748 |
| `popup.webp` | 307,224 |
| `states-dark.webp` | 305,014 |
| `states.webp` | 267,140 |
| `workflow-dark.webp` | 273,346 |
| `workflow.webp` | 238,914 |

The three largest exceed the 300 KB soft budget, accepted in RM-D11.

### Acceptance gate

- Every statement in an image traces to the README or the code - checked by reading, no automated comparison exists.
- Minimum text 17 CSS px outside the popup mock - holds by construction in `shared.css`.
- Light and dark both legible - viewed.

## Phase 2 - Navigation buttons

**Status:** **[IMPLEMENTED]**.

- Seven items `nav-why`, `nav-how`, `nav-using`, `nav-safety`, `nav-requirements`, `nav-build`, `nav-status`, 56 CSS px high, rendered at 2x, 5 to 8 KB per file, 92,750 bytes in total.
- Decisions: graphic form RM-D4, size and displayed height RM-D8.
- **[IMPLEMENTED]** Every button was viewed in at least one theme. The Why icon was redrawn after the first render because the question mark was too small to read.

### Acceptance gate

- Each button links to an existing README heading - checked against the heading text.
- The row fits one line at the README column width - **[UNVERIFIED]**, see RM-Q2.

## Phase 3 - README and runbook

**Status:** **[IMPLEMENTED]**.

- `README.md` follows the section order in [introduction](readme.introduction.md#current-truth); long reference material sits below the visual sections, the layout table in `<details>`.
- Install and build commands stay as text. Every figure has alt text that narrates its nodes and edges; every button's alt text is its label.
- Decisions: location RM-D1, language RM-D2, claim bound RM-D9, build section RM-D10.
- `docs/runbook/readme-illustrations.md` records files, the source contract, the render steps and the size budget. It does not reference this pack.

### Acceptance gate

- Every image path in the README exists - **[IMPLEMENTED]**, all twenty files and the application mark are present.
- No non-ASCII character in the README - **[IMPLEMENTED]**, searched.
- Product claims trace to the `remotune` pack - **[IMPLEMENTED]**, by reading; the Status and limits section mirrors its unverified list.

## Phase 4 - Baseline record and delivery commit

**Status:** **[IMPLEMENTED]** for the pack; the delivery commit carries the README, assets, runbook, `.gitignore` and this pack together, as `AGENTS.md` requires for an implementation and its docpack.

- The staged diff is scanned for key material before the commit.
- `code_ref` in this pack's documents reads `uncommitted` because it was written against the working tree; a later sync records the commit.

## Phase 5 - Hosting-site verification

**Status:** **[PLANNED]**; depends on the delivery commit being on a branch GitHub renders.

### Acceptance gate

- The row of seven buttons fits one line in light, dark and at phone width.
- `<picture>` selects the dark files in dark mode.
- All seven anchors resolve.
- Alt text reads as the figure's text form with images off.

Outcome and any change to the button height or markup are recorded here and close RM-Q2.

## Risks and controls

| Risk | Status | Control |
|---|---|---|
| Figures drift from the code | **[UNVERIFIED]** real | Re-edit `readme.html` and re-render with any UI label or state change; nothing compares them |
| README Status drifts from the product pack | **[PLANNED]** | RM-D9 obligation: change both in one edit |
| Anchor breaks on a heading rename | **[PLANNED]** | Rename the heading and its button together |
| Font differs on another render machine | **[DECIDED]** accepted | Render on Windows with Segoe UI Variable, RM-D6 |
| Navigation row wraps | **[UNVERIFIED]** | RM-D8, RM-Q2 |
| Runbook contradicts the build section | **[UNVERIFIED]** | RM-Q1 |

## Exact next action

1. Commit the delivery set after the staged-diff secret scan.
2. Open the README on the branch GitHub renders, in light, dark and phone width, and run the Phase 5 gate.
3. Answer RM-Q1 (Docker runbook against the host-native workflow) and mark the runbook superseded if so decided.
