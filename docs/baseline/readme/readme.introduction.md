---
baseline_schema: "2.0"
pack: "readme"
document: "introduction"
status: "active"
updated: "2026-10-11"
code_ref: "uncommitted"
---

# Remotune README Visual Upgrade - Introduction

## Authority and scope

This pack records the state, decisions and verification gaps of the Remotune root `README.md`, its light and dark illustrations, the sources they render from, and the runbook that regenerates them. It does not own product behavior: every product claim in the README traces to the `remotune` pack in `docs/baseline` (`remotune.*.md`) or to the code, and those stay authoritative.

Status vocabulary, same as the `remotune` pack:

| Token | Meaning |
|---|---|
| **[DECIDED]** | Approved direction |
| **[PLANNED]** | Intended, not done |
| **[UNVERIFIED]** | Needs observation before it is relied on |
| **[VERIFIED]** | Observed directly, with scope stated |
| **[IMPLEMENTED]** | Present in the working tree and inspected |

## Current truth

**[IMPLEMENTED]** The deliverable was written on top of base commit `8f0a6cf` and is delivered by one commit that also carries this pack.

| Artifact | Path | State |
|---|---|---|
| README | `README.md` | **[IMPLEMENTED]** English, 117 lines |
| Figures | `assets/readme/workflow*.webp`, `states*.webp`, `popup*.webp` | **[IMPLEMENTED]** 3 figures, light and dark |
| Navigation buttons | `assets/readme/nav-*.webp` | **[IMPLEMENTED]** 7 buttons, light and dark |
| Render sources | `assets/source/readme.html`, `shared.css`, `shared.js`, `finish.py` | **[IMPLEMENTED]** |
| Runbook | `docs/runbook/readme-illustrations.md` | **[IMPLEMENTED]** |
| Ignore rules | `.gitignore` | **[IMPLEMENTED]** `.render-tmp/`, `.playwright-mcp/` |

**[IMPLEMENTED]** README section order: lead (mark, positioning sentence, navigation buttons), Why Remotune, How it works (workflow figure, CRD ON/OFF table), Using Remotune (popup figure, controls), Safety model (states figure, guarantees), Requirements and install, Build from source, Status and limits, Repository layout in `<details>`, Documentation.

**[IMPLEMENTED]** Twenty WebP files total 1,843,136 bytes. The README column shows each figure at `width="100%"` and each button at `height="32"`, through `<picture>` with a dark `<source>`.

**[UNVERIFIED]** How the README renders on github.com, in light and dark, and on a phone. Nothing here was observed on the hosting site.

## Constraints

| Constraint | Source |
|---|---|
| Code files, including `assets/source/*`, carry no comments; rationale lives in docs | `AGENTS.md` |
| ASCII hyphen only, no en or em dash; one paragraph is one line | pack contract |
| README claims stop at the evidence in the `remotune` pack; unverified configurations are listed, not implied | decision RM-D9 (README claims bounded by pack evidence) in `readme.hallucination.md` |
| No pack filename or section is referenced from code or shipped files | pack contract |
| Windows packaging naming `out/remotune-v<version>.exe` is unchanged by this work | `AGENTS.md` |

## Pack navigation

- [Roadmap](readme.roadmap.md): phases, evidence, risks, exact next action.
- [Decision ledger](readme.hallucination.md): closed decisions, rejected alternatives, lessons, open questions.
