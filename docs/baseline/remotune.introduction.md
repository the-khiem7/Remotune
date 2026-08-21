---
baseline_schema: "2.0"
pack: "remotune"
document: "introduction"
status: "active"
updated: "2026-08-21"
code_ref: "uncommitted"
---

# Remotune Baseline Introduction

**[VERIFIED]** In the target-machine v0.1.5 run, the operator confirmed animation was applied while the UI reported an active CRD session and available recovery snapshot. **[IMPLEMENTED]** `out/remotune-v0.1.6.exe` packages the Start with Windows quoted-path comparison correction. **[UNVERIFIED]** v0.1.6 still needs a target-machine run, startup-after-sign-in observation, and full Pause/Resume/Quit recovery acceptance.

**[IMPLEMENTED]** The development workflow is now host-native: `%USERPROFILE%\go\bin\wails3.exe` is installed at the pinned `v3.0.0-beta.8`; `wails3 dev` invokes the `build/config.yml` dev graph with Vite hot reload, and `wails3 task` runs the repository Taskfile for verification and portable Windows packaging. `build/config.yml` is the sole semantic-version source for both Wails metadata and versioned portable artifacts. Docker is no longer a supported development boundary.

**[IMPLEMENTED]** The Wails Go module, CLI, and frontend runtime are locked to the same exact `v3.0.0-beta.8` release through `go.mod`, `frontend/package.json`, and `frontend/bun.lock`. Wails lifecycle work is internal to `ServiceStartup`/`ServiceShutdown`; generated Vue bindings expose only operator commands and status reads.

## Authority and status

This Baseline Docs pack is the complete active source of truth for Remotune. An implementer can understand, plan, build, verify, operate, and resume the project using only the five documents linked under [Pack navigation](#pack-navigation). Historical proposal material is non-authoritative and is not required to interpret any requirement.

Status vocabulary:

- **[DECIDED]** Approved product or engineering direction.
- **[PLANNED]** Required or intended behavior that has not been implemented.
- **[UNVERIFIED]** A proposed mechanism or compatibility claim that requires evidence before hard-coding.
- **[VERIFIED]** A platform or integration fact established by direct live observation on a real machine, recorded with its environment and scope limits. It describes how Windows or CRD actually behaves, not Remotune code.
- **[IMPLEMENTED]** Verified in inspected code, with automated tests passing.

A **[VERIFIED]** fact is only as broad as the configuration it was observed on. Where a question is inherently about variation across Windows versions, locales, or hardware layouts, single-machine evidence does not close it.

**[DECIDED]** When documents appear to overlap, responsibility is resolved as follows:

| Concern | Authoritative document |
|---|---|
| product identity, scope, invariants, constraints | this introduction |
| phases, dependencies, acceptance, risks, next action | [roadmap](remotune.roadmap.md) |
| closed decisions, candidates, open research, external references | [hallucination ledger](remotune.hallucination.md) |
| component boundaries, state, persistence, runtime flows | [source architecture](remotune.sourcecode.md) |
| user-visible and operator behavior | [use guide](remotune.useguide.md) |

## Current truth

**[DECIDED]** Remotune is approved for implementation handoff as a Windows tray-first background utility built with Wails v3, Go, and Vue.

**[IMPLEMENTED]** Phases 1 through 4 exist in the root Go module at the workspace root (module `github.com/khiemnguyen/remotune`, Go 1.26.1, dependencies: `golang.org/x/sys` and `github.com/wailsapp/wails/v3 v3.0.0-beta.8`):

- **Phase 1**, the Windows adapter layer (`VisualEffectsManager`, `TaskbarManager`): a three-layer Visual Effects snapshot, a transformation-based apply, and a bounded convergence loop that re-verifies the observed outcome instead of trusting a write. Verified on the real Controlled machine, five consecutive clean runs.
- **Phase 2**, the CRD detector (`Bootstrap`, `Reconstruct`, `Subscribe`): PID-scoped active-client reconstruction and a gap-free bookmark handover from historical query to live subscription. Verified against the real Event Log, including one live operator disconnect/reconnect cycle that found and fixed a real `EvtSubscribe` signaling defect.
- **Phase 3**, the `Coordinator` and durable `RecoveryStore`: one mutex-serialized transition loop implementing the exact apply/restore transaction flows, atomic snapshot persistence, and the full `Unknown`/`Baseline`/`Applying`/`Active`/`Restoring`/`Partial-Error`/`Recovery Required` state machine. Verified against fake adapters (29 tests, 8 consecutive clean runs), **and** verified end-to-end against the real Phase 1/2 adapters on 2026-08-14.
- **Phase 4**, the Wails tray shell (`internal/lifecycle`): system tray with status/Pause/Resume/Restore Now/Start with Windows/Quit; explicit Quit sequence that restores owned state before exit; WebView2 prerequisite check; autostart with stale-path detection; migration of Phases 1–3 from the standalone `engine/` module as a file move plus import-path rewrite with all tests passing and no adapter importing Wails.

Full detail is in the [roadmap](remotune.roadmap.md). The standalone `engine/` module is superseded but retained for reference. The remaining repository content is documentation plus the Phase 0 evidence tooling in `tools/phase0/`. **[IMPLEMENTED]** Phase 5 now provides the compact Vue/Vite control surface. It is embedded in the portable executable, opens from the tray, and renders separate CRD, automation, tuning, and recovery ownership state. The frontend invokes only coordinator-backed lifecycle commands through generated Wails bindings and always refreshes the backend state after a command.

**[VERIFIED]** A Phase 0 evidence spike ran on 2026-08-14 against the real Controlled machine (Windows 11 Pro 23H2 build 22631.6494, CRD host 152.0.7977.9, non-elevated user) and is substantially complete. CRD detection, the Windows tuning value model, apply and restore including an exact arbitrary-`Custom` round-trip, taskbar control, and the Wails/WebView2 prerequisites are all evidence-backed. Remaining gaps are environment coverage rather than unknown mechanisms: Windows 10, multi-monitor and secondary taskbars, and Explorer-restart reconciliation. Full observations and scope limits are in [Phase 0 recorded evidence](remotune.roadmap.md#phase-0-recorded-evidence).

**[VERIFIED]** The core invariant is demonstrated rather than merely intended: arbitrary `Custom`, `Let Windows choose`, `Best Appearance`, and already-`Best Performance` baselines each survived an apply/restore cycle with no differences, and a taskbar baseline of auto-hide OFF was preserved rather than forced ON.

**[VERIFIED]** Closing those cases also exposed a defect that would otherwise have shipped as an intermittent bug: a taskbar override applied only through `ABM_SETSTATE` is not durable, because the live state can diverge from what Explorer persists and later revert on its own. The fix is to write both the live and persisted layers on every change.

**[VERIFIED]** Porting the PowerShell reference to Go and running the resulting suite repeatedly showed that the reference sequence was **racy rather than correct**: it had appeared stable only because it was not exercised across enough runs. Three measured mechanisms forced a different write sequence, recorded as decisions 58 to 62 in the ledger, and two of the author's own intermediate conclusions were themselves wrong and are retracted there as decisions 63 and 64. The lesson generalizes past Visual Effects: a single passing run is not evidence of correctness for any Windows setting the shell reloads asynchronously, and the practice of verifying the observed outcome rather than trusting a write, already required by the product's reliability order, is what caught this.

**[DECIDED]** Remotune runs on the **Controlled machine** (currently the user's home machine), where Windows settings and the CRD host exist. The **Controlling machine** is currently the user's work machine. Chrome Remote Desktop (CRD) is the initial remote-session provider and trigger.

**[DECIDED]** The core invariant is: every Remotune-owned change must have a durable, reliable path back to the exact user state that existed before Remotune changed it.

**[DECIDED]** The product boundary is: Remotune automates remote-session transitions and will expose the Windows Performance Options **Visual Effects** choices needed for that automation. Windows remains the source of truth for the applied Visual Effects state; Remotune owns the selected CRD-on profile, the selected CRD-off action, and the durable snapshot required for recovery. The scope excludes the Advanced and Data Execution Prevention tabs and generic Windows tuning.

**[IMPLEMENTED]** The canonical app-mark source is `assets/branding/remotune.svg`. Derived PNGs serve Wails application and tray identity; `build/windows/icon.ico` and `build/windows/wails.exe.manifest` generate the Windows `.syso` resource before the versioned portable executable is compiled.

**[IMPLEMENTED]** Phase 5's Vue window maps backend numeric CRD and tuning enums to display labels before rendering, and reads `PortablePathStatus.PathMismatch` from the actual backend contract. The prior implementation treated a numeric CRD state as a string and crashed during initial render, leaving only the styled window background.

**[VERIFIED]** On 2026-08-20, the target machine opened the corrected `out/remotune-v0.1.4.exe` with its Vue controls rendered in the live Wails window, and Close-to-tray worked. Restore Now, Start with Windows, Pause, and Resume had previously been exercised in v0.1.3, but they are not yet accepted as an end-to-end recovery workflow. A later v0.1.4 observation reported CRD as `Disconnected` during an active connection; Pause from the resulting `Baseline`/no-snapshot state produced no visible Windows change; and a prior Explicit Quit did not restore animation. These safety-critical runtime incidents remain unresolved.

## Problem and product outcome

A Windows desktop optimized for direct physical use is not always ideal when its screen is captured, encoded, transferred, and displayed remotely.

### Visual motion over remote desktop

Windows Visual Effects can include window animation, minimize/maximize animation, taskbar animation, fade/slide effects, content redraw, and other effects managed by Performance Options. The product rationale is:

```text
More animation on the Controlled machine
→ more visual changes over time
→ more screen changes to capture, encode, and transmit
→ network limitations become more noticeable
→ remote interaction feels laggier or more stuttery
```

**[DECIDED]** Remotune does not control CRD's codec or transport. It reduces unnecessary source-side visual motion by automating the Windows configuration that already exists.

### Two-taskbar edge conflict

During CRD use, the effective view contains two taskbars:

```text
Controlling machine → real/local taskbar
Controlled machine  → remote taskbar rendered inside CRD
```

If both auto-hide at the bottom edge, pointer movement can reveal both and the local taskbar can cover the remote taskbar. Temporarily disabling auto-hide on the Controlled machine keeps its remote taskbar visible inside the stream so the edge interaction primarily belongs to the Controlling machine.

### Approved transition

**[PLANNED]** With the corresponding automation categories enabled:

```text
Before CRD
└─ Windows has the user's exact state

CRD Connected
├─ snapshot affected Visual Effects state
├─ snapshot taskbar auto-hide state
├─ Windows Visual Effects → selected CRD-on profile
└─ taskbar auto-hide      → OFF

CRD Disconnected
├─ Windows Visual Effects → selected CRD-off action
└─ taskbar auto-hide      → exact previous state
```

The expected relationship is:

```text
Configure once → leave in tray → use CRD normally → Remotune reacts automatically
```

## Brand and terminology

**[DECIDED]** The official project and product name is `Remotune`; `CRD Autotune` is only a previous codename and must not appear as the new user-facing name.

The name means:

```text
Remote + Tune → Remotune
```

It describes remote-session tuning without locking the brand to one provider. Use `Remotune` for the app title, tray menu, executable, project heading, package/build metadata where appropriate, UI copy, releases, logs, and local data names where migration compatibility is not required.

Recommended names:

- executable: `Remotune.exe`;
- local data directory: `%LOCALAPPDATA%\Remotune\`;
- product-facing language: “Remotune detected a CRD connection” and “Remotune applied Best Performance.”

`Chrome Remote Desktop (CRD)` names the current provider, not the product. CRD-specific technical names such as `CRDDetector` and `CrdEventParser` remain correct when they genuinely implement CRD behavior; provider-neutral names such as `RemoteSessionDetector` are appropriate only at real abstraction boundaries.

**[DECIDED]** Product positioning:

> Remotune automatically tunes Windows for remote desktop sessions and restores the user's original state when the session ends.

Current CRD-specific definition:

> Remotune is a lightweight Windows tray utility that detects Chrome Remote Desktop sessions, applies a selected Windows Visual Effects profile while remote control is active, disables taskbar auto-hide, and then restores a snapshot or applies the selected CRD-off Visual Effects choice after disconnect.

The provider-neutral brand does not broaden the product into a generic Windows tuning suite. The domain remains remote-session automation.

## Meaning of Best Performance

**[DECIDED]** The supported CRD-on choices are `Let Windows choose`, `Best Appearance`, `Best Performance`, and `Custom`. The supported CRD-off choices are `Revert to snapshot`, `Let Windows choose`, `Best Appearance`, and `Best Performance`. `Best Performance` means the behavior represented by:

```text
System Properties
→ Advanced
→ Performance
→ Settings
→ Performance Options
→ Visual Effects
→ Adjust for best performance
```

The same Windows surface contains “Let Windows choose what's best for my computer,” “Adjust for best appearance,” “Custom,” and the 17-item Visual Effects checklist. Remotune adopts those three Windows profiles as explicit targets. Its `Custom` profile is the persisted set of checkbox choices made in Remotune; editing any checkbox selects `Custom`, while a full match with a built-in target normalizes back to that target.

A user may begin in Best Appearance, Best Performance, Let Windows choose, or an arbitrary Custom combination. `Revert to snapshot` restores that exact affected state; selecting a CRD-off profile is an intentional replacement of it, not a recovery operation.

## Required behavior

### Detection

Reliable connection detection is the most important technical component: unreliable detection makes every automation transition unreliable.

- **[VERIFIED]** Use CRD connect/disconnect evidence in Windows Event Log as the primary source of truth. The provider is `chromoting` on the `Application` channel, with event 1 for connect and event 2 for disconnect.
- **[VERIFIED]** CRD host process presence does not prove an active client session. The service was observed `RUNNING` with two live host processes and no connected client.
- **[DECIDED]** Service state, sockets, network traffic, and CPU activity are weaker signals and may be diagnostics only, not primary connection truth.
- **[PLANNED]** At startup, load persisted state, reconstruct CRD state from historical events, subscribe to future events, and reconcile desired Windows state.
- **[VERIFIED]** The startup handover uses a bookmark: subscribe starting after the last event consumed by the historical query so no transition is lost between the two phases.
- **[VERIFIED]** Connect and disconnect events do not reliably alternate. A disconnect is genuinely lost when the CRD host process dies, so state reconstruction is scoped to the current host process lifetime and a dangling connect from a dead process is treated as disconnected.
- **[PLANNED]** Handle duplicate and delayed transitions, quick connect/disconnect, start while connected/disconnected, subscription failure, Event Log rotation/clear, stale bookmarks, CRD host/service restart, and delayed callbacks.
- **[UNVERIFIED]** Do not assume any disconnect means zero clients. No overlapping sessions appeared in the observed sample, which narrows but does not close the question, so an active-client set keyed by the per-session identifier remains the required model.

### Windows state preservation

- **[PLANNED]** Before the first owned override, snapshot every affected Visual Effects value and taskbar auto-hide, then durably persist the snapshot before mutation.
- **[DECIDED]** A duplicate connect cannot overwrite the original baseline with the already-tuned state.
- **[DECIDED]** No valid owned snapshot means no guessed restoration. Observing Best Performance does not prove Remotune caused it.
- **[PLANNED]** Preserve unrelated taskbar state and restore auto-hide to ON or OFF exactly as found.
- **[PLANNED]** Keep recovery data until restoration is successfully verified; partial apply or restore failures remain visible and retryable.
- **[PLANNED]** After crash/restart, reconcile durable ownership with reconstructed CRD state so Windows is not stranded in a temporary override.

### Lifecycle and controls

- **[PLANNED]** Closing the main window hides it to the tray; explicit tray `Quit` restores owned state before exit.
- **[PLANNED]** `Pause Automation` restores owned state, then stops automatic reactions until resumed.
- **[PLANNED]** `Restore Now` restores only a valid Remotune-owned snapshot and otherwise reports that no restorable snapshot exists.
- **[PLANNED]** Support `Start with Windows`; moving or deleting a portable executable may invalidate startup registration and must be handled or documented clearly.
- **[PLANNED]** The UI may enable/disable overall automation and optionally the Visual Effects and taskbar categories independently.
- **[PLANNED]** The CRD-on profile and CRD-off action are durable preferences. Changing a CRD-on profile while CRD is connected reconciles to the new target without replacing the original snapshot; changing a CRD-off action controls the next disconnected transition.

## Product and UX constraints

**[DECIDED]** Remotune is a compact, tray-centered utility inspired by G-Helper's quick status, minimal navigation, fast controls, tray lifecycle, and lightweight feel. It is not a G-Helper clone and must not inherit domain-specific controls, exact layout, or unnecessary density/settings.

**[PLANNED]** The main window is approximately 380–450 px wide, vertically arranged, Windows-native in feel, and follows OS dark/light theme where practical. It reports separate authoritative dimensions:

- CRD: `Unknown`, `Disconnected`, or `Connected`;
- automation: `Enabled` or `Paused`;
- tuning: `Baseline`, `Applying`, `Active`, `Restoring`, `Partial/Error`, or `Recovery Required`.

**[DECIDED]** `CRD: Connected` plus `Automation: Paused` is valid; one ambiguous red/green indicator is insufficient. The main window is a control surface, while background automation is the real product.

**[DECIDED]** Remotune is not:

- a CRD replacement or remote desktop client;
- a network or codec optimizer;
- a Windows debloater, generic system tweaker, or customization suite;
- a replacement for all Windows Performance Options tabs, Advanced settings, or Data Execution Prevention;
- a generic Visual Effects editor unrelated to CRD automation;
- a profile engine with Minimal, Recommended, Aggressive, or unbounded tuning presets;
- a dashboard or app expected to remain open on-screen.

The CRD profile surface is deliberately limited to the Visual Effects choices and checklist shown in Windows Performance Options. Internal knowledge of individual values remains subject to the existing snapshot, convergence, and verification requirements.

## Framework and distribution rationale

**[DECIDED]** Platform: Windows. Stack: Wails v3 + Go + Vue.

Wails is selected because Go fits Windows-native integration and compiled distribution, frontend assets can be bundled with the native app, Vue fits a compact control UI, expected resource use is lighter than an Electron-style distribution, and the backend can remain event-driven.

Wails v3 is selected over v2 because this product benefits directly from first-class system tray support, background utility lifecycle, autostart management, and the improved application/window model. Its Beta status is accepted for this new project, but an explicit version must be pinned and upgrades tested intentionally rather than floated uncontrolled.

**[DECIDED]** The pinned version is **v3.0.0-beta.8** (published 2026-08-12). **[VERIFIED]** WebView2 presence is detectable without elevation via its EdgeUpdate client key, and the evidence machine already carries the runtime, Go 1.26.1, and Node v24.13.1. The `wails3` CLI is not yet installed, and the framework's tray, lifecycle, and autostart APIs have not been exercised.

- **[PLANNED]** Primary distribution is a portable `Remotune.exe` without requiring a traditional installer for the basic use case.
- **[DECIDED]** Wails uses WebView2 on Windows. “Single executable” does not mean “zero OS runtime dependencies”; document the requirement and fail clearly when it is unavailable.
- **[PLANNED]** Prefer normal-user operation. Verify Event Log, Visual Effects, taskbar, and autostart permissions rather than elevating the whole app by default.
- **[PLANNED]** Remain event-driven with negligible idle CPU, low memory, no aggressive polling, no unnecessary hidden-window frontend activity, and quick apply/restore transitions.
- **[DECIDED]** Accurate product claim: Remotune applies the selected Windows Visual Effects profile during a CRD session. Applications may render custom animations independently, so Remotune cannot promise that every animation in every app is disabled.

## Reliability order

**[DECIDED]** Prioritize:

1. preserve user Windows state;
2. correctly detect CRD connection state;
3. apply the selected Visual Effects profile reliably;
4. restore reliably;
5. manage taskbar auto-hide reliably;
6. recover after crash/restart;
7. run quietly in the tray;
8. UI polish.

When choosing between more features and more reliability, choose reliability. When choosing between recreating a Windows setting and automating the existing setting, automate the existing setting.

## Pack navigation

- [Roadmap](remotune.roadmap.md): phases, dependencies, evidence gates, risks, scenarios, acceptance, and exact next action.
- [Hallucination ledger](remotune.hallucination.md): closed decisions, candidates, unresolved research, evidence policy, and technical references.
- [Source architecture](remotune.sourcecode.md): topology, component contracts, persistence, state machine, and runtime flows.
- [Use guide / behavior contract](remotune.useguide.md): planned installation, controls, status, recovery, diagnostics, and operator validation.

## Normalization provenance

This pack was initialized on 2026-08-14 from an approved project proposal last consolidated on 2026-08-13, then standalone-normalized on 2026-08-14. The historical input is archived and non-authoritative. All normative content needed for implementation and resume now lives in the domain documents above. The originally inspected code reference was `73f49b65063eacf9953d40d324c9c61e3b4e64eb`.

Updated on 2026-08-14 to record the Phase 0 evidence spike. Code state at that point was commit `1d8c9e2d6419125d1b8020249b4bd870394ecfd1` plus uncommitted additions under `tools/phase0/`, so `code_ref` is recorded as `uncommitted`. The spike introduced the **[VERIFIED]** status token and resolved a substantial part of the [uncertainty ledger](remotune.hallucination.md); no product decision or boundary was changed by it.

Updated again on 2026-08-14 to record the Phase 1 Go implementation of the Windows adapter layer, and to introduce the **[IMPLEMENTED]** status token as an active claim rather than a placeholder. Code state was commit `ec82796325797a5675c6cdd150e503f0d415d109` plus the uncommitted `engine/` module, so `code_ref` remains `uncommitted`. No product decision or boundary changed; the write-sequence mechanisms discovered during the port refined *how* the already-decided invariant is achieved, not *what* Remotune is required to do.
