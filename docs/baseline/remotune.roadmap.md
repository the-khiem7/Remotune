---
baseline_schema: "2.0"
pack: "remotune"
document: "roadmap"
status: "active"
updated: "2026-08-14"
code_ref: "uncommitted"
---

# Remotune Implementation Roadmap

## Current checkpoint

**[DECIDED]** Product direction is approved. **[PLANNED]** No application code exists yet; the repository contains documentation plus the Phase 0 evidence tooling under `tools/phase0/`.

**[VERIFIED]** A Phase 0 evidence spike was executed on 2026-08-14 on the actual Controlled machine and is now substantially complete. All four areas are closed with reproducible live evidence recorded in [Phase 0 recorded evidence](#phase-0-recorded-evidence), including the exact 22-value set that `Adjust for best performance` changes and a passing arbitrary-`Custom` round-trip. Remaining gaps are environment coverage only: Windows 10, multi-monitor and secondary taskbars, Explorer-restart reconciliation, and the Wails runtime APIs that require the pinned project to exist.

**Phase 1 and Phase 2 are both unblocked and may proceed in parallel.**

The roadmap builds toward the finished product; it does not prescribe a throwaway MVP. UI polish is intentionally blocked until the Windows/CRD fundamentals are proven.

## Dependency chain

```text
Phase 0: Windows/CRD evidence
  ├─ Phase 1: Windows tuning engine
  └─ Phase 2: CRD detector
       └──────────┬───────────┘
                  ▼
Phase 3: Coordinator and durable recovery
                  ▼
Phase 4: Wails tray shell
                  ▼
Phase 5: Compact Vue UI
                  ▼
Phase 6: Hardening and release evidence
```

## Phase 0 — Windows and CRD research spike

**Status:** **[VERIFIED]** and substantially complete. CRD detection, the Windows tuning value model, apply/restore with an exact arbitrary-`Custom` round-trip, taskbar control, and the Wails/prerequisite facts are all closed with reproducible live evidence. The residual gaps in [Outstanding Phase 0 items](#outstanding-phase-0-items) are environment-coverage items, not unknown mechanisms, and none of them block Phase 1 or Phase 2 on the supported configuration.

### Deliverables

1. **[UNVERIFIED] CRD Event Log evidence**
   - exact channel and provider/source;
   - exact connect/disconnect event IDs;
   - event XML/message fields and any client identifier fields;
   - normal-user query/subscription permissions;
   - observed sequence for connect, disconnect, host/service restart, and startup while connected;
   - multiple-client semantics and behavior across relevant CRD updates;
   - startup reconstruction strategy, including Event Log clear/rotation cases.
2. **[UNVERIFIED] Visual Effects evidence**
   - reliable method that produces Windows `Adjust for best performance` behavior;
   - complete set/representation of all values changed by that action;
   - complete snapshot representation and exact restoration of arbitrary `Custom` state;
   - immediate update/broadcast behavior and whether Explorer restart is required;
   - Windows 10 and Windows 11 results;
   - supported APIs preferred where clean; any registry-backed mechanism isolated and justified.
3. **[UNVERIFIED] Taskbar evidence**
   - `SHAppBarMessage` with `ABM_GETSTATE`/`ABM_SETSTATE` behavior;
   - preservation of unrelated appbar bits/state;
   - immediate apply and exact restore;
   - Windows 10/11, Explorer restart, multiple monitors, and secondary taskbar behavior.
4. **[UNVERIFIED] Wails evidence**
   - exact Wails v3 version to pin;
   - tray, window lifecycle, autostart Manager API, WebView2 failure behavior, and moved-portable-path behavior.

### Acceptance gate

Phase 0 is complete only when reproducible evidence identifies safe adapters and snapshot formats. Source review, compilation, static inspection, or documentation alone is not proof of live Windows/CRD integration behavior.

## Phase 0 recorded evidence

Collected 2026-08-14 by direct observation and live API probes. Every result below was obtained as a **normal, non-elevated user**; no step required Administrator.

### Environment

| Property | Observed value |
|---|---|
| OS | Windows 11 Pro 23H2, build 22631, UBR 6494, x64 |
| Machine role | Controlled machine |
| Process context | non-elevated standard user |
| CRD host | 152.0.7977.9 at `C:\Program Files (x86)\Google\Chrome Remote Desktop\152.0.7977.9\` |
| CRD service | name `chromoting`, `LocalSystem`, `AUTO_START`, state `RUNNING` |
| CRD host config | `C:\ProgramData\Google\Chrome Remote Desktop\host.json` |
| Displays | single `\\.\DISPLAY1`, 1920x1080 |
| WebView2 | 151.0.4129.78, system-wide |
| Toolchain | Go 1.26.1 windows/amd64, Node v24.13.1, npm 11.8.0, `wails3` CLI absent |

**[VERIFIED]** Tooling retained: `tools/phase0/Get-VisualState.ps1` (read-only full snapshot plus diff; doubles as the candidate `VisualEffectsManager.Snapshot()` schema) and `tools/phase0/Test-TaskbarRoundTrip.ps1` (taskbar apply/restore proof with guaranteed restore). Snapshot `tools/phase0/snapshots/01-baseline-bestappearance.json` records the pre-experiment user state.

### CRD Event Log

**[VERIFIED]** Provider `chromoting` writes to the **Application** channel as a legacy event source registered at `HKLM\SYSTEM\CurrentControlSet\Services\EventLog\Application\chromoting`, with `EventMessageFile` = `remoting_core.dll`, `TypesSupported` = 7, `CategoryCount` = 1. It is not a modern manifest provider.

| Event ID | Meaning | Level / Task / Opcode / Keywords |
|---|---|---|
| 1 | Client connected | 4 / 1 / 0 / `0x80000000000000` |
| 2 | Client disconnected | 4 / 1 / 0 / `0x80000000000000` |
| 4 | Channel IP for client | 4 / 1 / 0 / `0x80000000000000` |

`EventID` carries `Qualifiers='16384'`; the usable identifier is the low value (1, 2, 4).

Working selector, verified against live data:

```text
Channel: Application
XPath:   *[System[Provider[@Name='chromoting'] and (EventID=1 or EventID=2)]]
```

**[VERIFIED]** Payload shape. Events 1 and 2 carry a single `EventData/Data` entry holding the client JID `<email>/chromoting_ftl_<uuid>`. Event 4 carries five entries: `[0]` JID, `[1]` `unknown`, `[2]` client `ip:port`, `[3]` empty, `[4]` route type (observed `relay`). `System/Execution/@ProcessID` identifies the emitting host process.

**[VERIFIED]** The JID resource component (`chromoting_ftl_<uuid>`) is a **unique per-session identifier**, and connect and disconnect for one session carry the identical string. Across 191 transition events, 94 resource IDs appeared exactly twice and 3 appeared exactly once. This supplies the stable key required for an active-client set.

**[VERIFIED]** Transitions do **not** strictly alternate. The sample contained 97 connects against 94 disconnects, with three `CONNECT → CONNECT` sequences. In all three the host `ProcessID` changed across the boundary (22864→6544, 6544→6380, 5520→5628), so the missing disconnect coincides with host process death rather than with overlapping sessions. Lost disconnects are therefore detectable.

**[VERIFIED]** No same-`ProcessID` overlapping sessions occurred in 191 events spanning roughly three weeks. This is strong evidence that the current single-user setup yields one client at a time, but it does not prove CRD forbids concurrent clients, so the active-client set remains the required model.

**[VERIFIED]** Host presence is not connection truth. The service was `RUNNING` with two live `remoting_host` processes while no client was connected.

**[VERIFIED]** Retention. `Application` is `Circular` with a 20971520-byte maximum, observed at capacity with 33844 records. Available `chromoting` history covered about three weeks. Adequate in practice, so rotation and clear fallbacks remain mandatory.

**[VERIFIED]** Startup race resolved. Seeding the subscription with the bookmark of the last event consumed by the historical query replays exactly the events in the gap:

```text
EvtQuery historical replay      → bookmark of last consumed event
→ EvtSubscribe StartAfterBookmark
→ zero-gap handover
```

Two independent probes each replayed 5 of 5 expected records: a reader seeded with the bookmark, and a watcher seeded with the same bookmark. **Critical caveat:** the "read existing events" flag must be enabled. With it disabled the subscription silently degrades to future-only delivery and the gap returns.

**[VERIFIED]** Privacy. Connect, disconnect, and channel events expose the account email, and event 4 additionally exposes the client IP and port. Neither is required for state and must be redacted rather than persisted.

### Taskbar auto-hide

**[VERIFIED]** `SHAppBarMessage` is sufficient and needs no elevation. `ABM_GETSTATE` = `0x4`, `ABM_SETSTATE` = `0xA`, with the desired state passed in `APPBARDATA.lParam`.

Observed initial user state was `0x01`: `ABS_AUTOHIDE` set, `ABS_ALWAYSONTOP` clear, no other bits.

Round-trip result, flipping only bit `0x01` and preserving all other bits:

```text
0x01 (auto-hide ON)  → 0x00 (OFF)  → 0x01 (ON)
apply verified: yes   restore verified: yes   unrelated bits preserved: yes
```

Changes take effect immediately, settle within roughly 1.2 s, and require no Explorer restart.

**[VERIFIED]** A practical verification signal exists for the apply/restore gates: the primary screen work-area height was 1080 with auto-hide ON and 1026 with it OFF, against a 1080 bounds height and a 54 px taskbar rect (`edge=BOTTOM`, `rc=(0,1026)-(1920,1080)`). Comparing work area against bounds confirms the real outcome instead of trusting the write.

#### Durability of the taskbar override

**[VERIFIED] This supersedes the earlier decision to never write `StuckRects3`.**

A taskbar override applied only through `ABM_SETSTATE` is **not durable**. The live appbar state and the persisted `StuckRects3` `Settings` byte 8 were observed to diverge: `ABM_GETSTATE` reported auto-hide ON while the persisted bit read OFF. The override was later observed to revert to the persisted value on its own, turning auto-hide off without any Remotune call.

```text
ABM_SETSTATE          -> changes the LIVE state immediately
StuckRects3 byte 8    -> what Explorer persists and can reconcile back from
divergence between them = a window in which the override silently disappears
```

Scope of the claim, stated precisely: the divergence is **reproducible on demand**, and one spontaneous revert was observed. A bisect over the individual phases of a Visual Effects apply — the SPI batch, the `Explorer\Advanced` writes, `WindowMetrics` and `TraySettings` broadcasts, and the DWM plus mask writes — **failed to reproduce** the revert, so no single trigger is identified and the timing appears asynchronous. Treating it as a durability defect rather than hunting the trigger is the safer engineering response.

**[VERIFIED]** Writing both layers fixes it. A durable setter that issues `ABM_SETSTATE` for the live effect and writes the `StuckRects3` bit for persistence produced immediate agreement, and the override then survived a `WM_SETTINGCHANGE` broadcast with both layers still in agreement. The earlier objection that `StuckRects3` needs an Explorer restart does not apply here, because `ABM_SETSTATE` supplies the live effect while the registry write only removes the divergence.

**[DECIDED]** `TaskbarManager` writes both layers on every change, snapshots both, and treats disagreement between them as a health signal rather than as normal.

#### Taskbar baseline OFF

**[VERIFIED]** The Phase 1 gate case missed by Phase 0 now passes. Simulating a user whose baseline is auto-hide OFF, then applying the override and restoring, left the taskbar OFF with both layers in agreement. The baseline was never forced to ON.

### Visual Effects

**[VERIFIED]** `VisualFXSetting` lives at `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects` with `0` = Let Windows choose, `1` = Best appearance, `2` = Best performance, `3` = Custom. Observed value was `1`.

**[VERIFIED]** A planned assumption was wrong: the 19 `VisualEffects` subkeys contain only `DefaultApplied` bookkeeping and are **not** the storage for effect state. They must not be treated as the snapshot source.

**[VERIFIED]** Effect state is split across `UserPreferencesMask`, discrete registry values, and `SystemParametersInfo`. All 17 probed `SPI_GET*` actions plus `SPI_GETANIMATION` returned values with no failures and no elevation:

| Effect | SPI action | Effect | SPI action |
|---|---|---|---|
| DragFullWindows | `0x0026` | TooltipAnimation | `0x1016` |
| FontSmoothing | `0x004A` | TooltipFade | `0x1018` |
| MenuAnimation | `0x1002` | CursorShadow | `0x101A` |
| ComboBoxAnimation | `0x1004` | FlatMenu | `0x1022` |
| ListBoxSmoothScrolling | `0x1006` | DropShadow | `0x1024` |
| GradientCaptions | `0x1008` | UIEffects | `0x103E` |
| KeyboardCues | `0x100A` | ClientAreaAnimation | `0x1042` |
| HotTracking | `0x100E` | MinAnimate (`ANIMATIONINFO`) | `0x0048` |
| MenuFade | `0x1012` | | |
| SelectionFade | `0x1014` | | |

**[VERIFIED]** `UserPreferencesMask` was observed as `9E 3E 07 80 12 00 00 00`. A bit decode cross-checked against the SPI accessors reached full agreement on 13 of 13 comparable effects, after correcting two offsets: `UIEffects` is byte 3 mask `0x80` and `ClientAreaAnimation` is byte 4 mask `0x02`. Remaining agreeing offsets are byte 0 masks `0x02`/`0x04`/`0x08`/`0x10`/`0x20`/`0x80` and byte 1 masks `0x02`/`0x04`/`0x08`/`0x10`/`0x20`.

**[VERIFIED]** A later attribution pass toggled all 17 probed effects and resolved the mask almost completely. `FlatMenu` owns `byte[2]:0x02` and `DropShadow` owns `byte[2]:0x04`. Exactly two set bits remain unexplained: `byte[2]:0x01` and `byte[4]:0x10`. `DragFullWindows` and `FontSmoothing` own no mask bit and live only in the registry.

**[DECIDED]** `UserPreferencesMask` is still treated as an **opaque 8-byte blob** that is snapshotted and restored verbatim, while `SystemParametersInfo` remains the authoritative per-effect accessor. The justification is that two bits are unexplained and preserving them verbatim costs nothing, not that any loss was observed.

Supporting values captured for the snapshot schema: `DwmIsCompositionEnabled` = true; `HKCU\...\DWM` `EnableAeroPeek` = 1, `AlwaysHibernateThumbnails` = 1, `Composition` = 1; `Explorer\Advanced` `ListviewAlphaSelect` = 1, `ListviewShadow` = 1, `TaskbarAnimations` = 1, `IconsOnly` = 0; `Control Panel\Desktop` `DragFullWindows` = 1, `FontSmoothing` = 2, `FontSmoothingType` = 2, `MenuShowDelay` = 400; `WindowMetrics\MinAnimate` = 1.

#### Ground truth of `Adjust for best performance`

**[VERIFIED]** Captured by diffing a full snapshot before and after the real Performance Options action, moving from `Adjust for best appearance` to `Adjust for best performance`. The action changed exactly **22 values**.

Eleven effects went from 1 to 0 through `SystemParametersInfo`:

```text
DragFullWindows   FontSmoothing      MenuAnimation        ComboBoxAnimation
ListBoxSmoothScrolling               SelectionFade        TooltipAnimation
CursorShadow      DropShadow         ClientAreaAnimation  MinAnimate
```

Seven probed effects were **left untouched** by the preset and must therefore not be forced off by a naive "disable everything" apply: `GradientCaptions`, `KeyboardCues`, `HotTracking`, `MenuFade`, `TooltipFade`, `FlatMenu`, `UIEffects`.

Registry values changed: `VisualFXSetting` 1→2, `Desktop.DragFullWindows` 1→0, `Desktop.FontSmoothing` 2→0, `WindowMetrics.MinAnimate` 1→0, `Advanced.ListviewAlphaSelect` 1→0, `Advanced.ListviewShadow` 1→0, `Advanced.TaskbarAnimations` 1→0, `Advanced.IconsOnly` 0→**1**, `DWM.EnableAeroPeek` 1→0, `DWM.AlwaysHibernateThumbnails` 1→0. `DWM.Composition` and `Desktop.FontSmoothingType` were untouched.

`UserPreferencesMask` moved `9E 3E 07 80 12 00 00 00` → `90 12 03 80 10 00 00 00`. Every cleared bit reconciles with the SPI results, with one exception:

**[VERIFIED] Correction.** An earlier revision of this document claimed the preset clears a bit that maps to no documented effect, and used that as proof a known-effects-only snapshot would lose state. **That claim was wrong and is retracted.** It came from cross-checking only 13 effects and omitting `DropShadow` and `FlatMenu`. A later attribution pass toggled every probed effect and observed which bit moved:

```text
byte[2]:0x02 = FlatMenu
byte[2]:0x04 = DropShadow      <- the bit previously misreported as unknown
byte[2]:0x01 = still unknown
byte[4]:0x10 = still unknown
```

Corrected position: the preset touches **only attributable bits**, so no state loss was demonstrated during a preset apply. Two bits, `byte[2]:0x01` and `byte[4]:0x10`, remain genuinely unexplained, and the preset does not touch either. Also confirmed: `DragFullWindows` and `FontSmoothing` own **no** mask bit at all and live purely in the registry, which independently confirms the three-layer model.

The opaque-blob decision stands, but on the narrower and honest justification that two bits are unexplained and preserving them verbatim is free. It is not justified by a demonstrated loss.

**[VERIFIED]** Three traps that a per-effect boolean model would have introduced:

| Trap | Evidence | Consequence |
|---|---|---|
| `FontSmoothing` is not boolean | registry held `2`, while `SPI_GETFONTSMOOTHING` reports `1` | restoring from the SPI boolean would write `1` and silently downgrade ClearType; the registry value must be captured |
| `IconsOnly` inverts | 0 → 1, the only value the preset raises | a "set everything to 0" apply would be wrong |
| Explorer-only values have no SPI | `ListviewAlphaSelect`, `ListviewShadow`, `TaskbarAnimations`, `IconsOnly`, the DWM values | `SystemParametersInfo` alone cannot capture or restore a complete state |

A complete snapshot therefore requires all three layers together: the per-effect SPI values, the discrete registry values, and the opaque mask.

#### Apply and restore proof

**[VERIFIED]** A programmatic restore reproduced the operator's original state exactly. Applying snapshot `01` over the Best Performance state returned all 22 values to their originals, including the unattributed mask bit; the only reported difference was `FontSmoothingType` appearing as absent in the earlier snapshot, a schema artifact from adding that field after the capture rather than a state change.

**[VERIFIED]** The arbitrary `Custom` acceptance gate passed. An arbitrary combination matching neither preset was synthesised (`VisualFXSetting` = 3, mask `04 04 07 80 10 00 00 00`, mixed per-effect values, `IconsOnly` = 1, `ListviewShadow` = 0, `TaskbarAnimations` = 0, `EnableAeroPeek` = 0), applied, overwritten with Best Performance, then restored:

```text
synthesise Custom → apply → capture actual
→ apply Best Performance
→ restore captured Custom
→ diff: NO DIFFERENCES
```

**[VERIFIED]** Applying Best Performance from a `Custom` starting point produced a state identical to the operator-produced preset, so the apply operation is deterministic and independent of the starting state.

**[VERIFIED]** Verified write order for both apply and restore:

```text
1. SystemParametersInfo per-effect writes, flags SPIF_UPDATEINIFILE | SPIF_SENDCHANGE
2. discrete registry values (Explorer\Advanced, DWM, Desktop, WindowMetrics)
3. UserPreferencesMask written verbatim LAST, so no SPI write can clobber unattributed bits
4. WM_SETTINGCHANGE broadcast
```

The machine was returned to the operator's original state and independently confirmed: `VisualFXSetting` = 1, mask `9E 3E 07 80 12 00 00 00`, `FontSmoothing` 2 / type 2, `IconsOnly` = 0, `TaskbarAnimations` = 1, `EnableAeroPeek` = 1, `MinAnimate` = 1.

Tooling retained: `tools/phase0/Restore-VisualState.ps1` (generic "apply this exact state", the reference for both `ApplyBestPerformance()` and `Restore(snapshot)`) and `tools/phase0/Test-CustomRoundTrip.ps1` (the acceptance gate, which restores the operator snapshot in a `finally` block).

### Wails and distribution

**[DECIDED]** Pin Wails **v3.0.0-beta.8**, published 2026-08-12, commit `81a149919f91f2149d3fe9be5a27472ae7617b8e`, confirmed as `@latest` from the Go module proxy. The published sequence runs `alpha.0`–`alpha.102`, `alpha2.103`–`alpha2.122`, then `beta.0`–`beta.8`.

**[VERIFIED]** WebView2 presence is detectable without elevation by reading `pv` under the `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}` EdgeUpdate client key; the observed system-wide install reported 151.0.4129.78. Probe the HKLM 64-bit, HKLM `WOW6432Node`, and HKCU variants to cover per-machine and per-user installs.

### Outstanding Phase 0 items

| Item | Status | Why it is still open |
|---|---|---|
| Windows 10 behavior | **[UNVERIFIED]** | No Windows 10 environment available. The affected value set, mask layout, and unattributed bits must be re-derived there before claiming support. |
| Multi-monitor and secondary taskbar | **[UNVERIFIED]** | Only one display present on the evidence machine. |
| Explorer restart reconciliation | **[UNVERIFIED]** | Not exercised during this spike; deferred to the Phase 6 matrix. |
| Autostart Manager API and moved-portable-path behavior | **[UNVERIFIED]** | Requires the pinned Wails project to exist; belongs to Phase 4. |

## Phase 1 — Windows tuning engine

**Status:** **[PLANNED]**, and **unblocked**. Both halves now rest on evidence: `SHAppBarMessage` is proven for read, apply, restore, and bit preservation, and the Visual Effects value set, write order, and exact `Custom` round-trip are proven. `tools/phase0/Get-VisualState.ps1` and `tools/phase0/Restore-VisualState.ps1` are the working references to port to Go.

**[DECIDED]** Phase 1 lives in a standalone Go module holding only the adapters and their tests. It does not depend on Wails, and it is expected to be migrated into the CLI-generated project in Phase 4. See decision 52 in the [ledger](remotune.hallucination.md#decisions-closed-by-phase-0-evidence).

### Deliverables

- minimal standalone Go module, no Wails dependency, adapters testable headlessly;
- `VisualEffectsManager`: `Snapshot()`, `ApplyBestPerformance()`, `Restore(snapshot)`, `GetCurrentState()`;
- `ApplyBestPerformance()` implemented as a transformation over the known affected values, never as a replay of a captured snapshot (decision 53);
- `TaskbarManager`: `GetAutoHide()`, `SetAutoHide(bool)` while preserving unrelated state;
- versioned, validated recovery snapshot schema covering all three capture layers;
- exact capture/apply/verify/restore operations, with read-back verification;
- explicit failure model: per-category results, partial-failure reporting, and no discarding of recoverable state;
- adapters isolate Win32 and the justified registry details from product logic.

### Evidence gaps — closed

Both gate cases identified after Phase 0 were exercised on 2026-08-14 and now pass. A third finding surfaced while closing them.

| Case | Result |
|---|---|
| Taskbar baseline auto-hide **OFF** | **[VERIFIED] PASS.** Apply then restore left it OFF with both layers agreeing; never forced to ON. |
| Baseline **Let Windows choose** (`VisualFXSetting` = 0) | **[VERIFIED] PASS.** Round-tripped with no differences and the setting returned to 0. Writing 0 did **not** make Windows recompute or overwrite effect values, so `VisualFXSetting` behaves as a label while the effect values stand on their own. |
| Baseline already **Best Performance** | **[VERIFIED] PASS.** Apply is idempotent, a second apply changed nothing, and restore introduced no invented changes. |
| Transformation-based apply matches the real preset | **[VERIFIED] PASS.** Applying the transformation from the Best Appearance baseline reproduced every Visual Effects value of the operator-produced preset exactly. |
| Taskbar override durability | **[VERIFIED] defect found and fixed.** See [Durability of the taskbar override](#durability-of-the-taskbar-override). This was not on the original gate list and would have shipped as an intermittent bug. |

**[UNVERIFIED]** Residual: writing `VisualFXSetting` = 0 caused no recomputation within a few seconds, but whether Windows recomputes "Let Windows choose" defaults at next logon was not observed.

### Acceptance gate

- Best Performance matches the Windows Performance Options behavior.
- A pre-existing arbitrary `Custom` configuration round-trips exactly, reproduced as an automated test against the Go implementation.
- Taskbar auto-hide ON and OFF both round-trip exactly.
- Failures are reported without discarding recoverable state.

## Phase 2 — CRD detector

**Status:** **[PLANNED]**, and **unblocked**. The CRD portion of Phase 0 is closed, so this phase may begin immediately and in parallel with the remaining Visual Effects research.

### Deliverables

- historical bootstrap with native `EvtQuery` using the verified channel, provider, and XPath;
- real-time subscription with native `EvtSubscribe` seeded from the historical bookmark, with read-existing-events enabled;
- event parser for the verified event 1/2 payload, extracting the per-session JID resource and the emitting host `ProcessID`;
- current-state reconstructor scoped to the current host process lifetime, treating a dangling connect from a dead PID as disconnected;
- active-client set keyed by the per-session JID resource;
- transition deduplication and detector health/error reporting;
- immediate redaction of account email and client `ip:port`;
- handling for rotation, clear, stale bookmark, delayed callback, and host/service restart.

### Acceptance gate

- normal connect and disconnect are detected automatically;
- starting Remotune during an active CRD session reconstructs `Connected`;
- process presence is not used as the source of truth;
- duplicate transitions do not create false ownership cycles;
- detector failures are observable.

## Phase 3 — Coordinator and recovery

**Status:** **[PLANNED]**; depends on Phases 1 and 2.

### Deliverables

- one serialized `StateCoordinator` transition loop;
- desired-state derivation from observed CRD state, automation state, enabled categories, and persisted ownership;
- durable apply and restore transactions;
- idempotency, race handling, latest-desired-state reconciliation;
- crash/startup recovery and `Restore Now`;
- explicit `Unknown`, `Partial/Error`, and `Recovery Required` states.

### Apply verification gate

```text
read complete baselines
→ persist recovery snapshot
→ state Applying
→ apply enabled categories
→ verify practical outcomes
→ mark ownership/Active only as justified
```

A partial failure must remain `Partial/Error`, retain recovery data, and never be shown as complete success.

### Restore verification gate

```text
load valid owned snapshot
→ state Restoring
→ restore enabled/owned categories
→ verify practical outcomes
→ clear ownership and retire snapshot only after success
→ state Baseline
```

No valid owned snapshot means no guessed restoration.

## Phase 4 — Wails tray shell

**Status:** **[PLANNED]**; depends on stable coordinator contracts.

### Deliverables

- project generated by the `wails3` CLI and pinned to v3.0.0-beta.8, keeping the canonical framework layout;
- migration of the Phase 1 to Phase 3 packages into that project as a file move plus import-path rewrite, with the existing tests still passing afterwards and no adapter importing Wails;
- application lifecycle, system tray, show/hide, close-to-tray, and background operation;
- explicit Quit sequence that stops new transitions, serializes pending work, restores owned state, persists final state, closes subscription/handles, removes tray, and exits;
- Start with Windows plus clear failure handling;
- WebView2 prerequisite handling;
- portable executable path caveat handling/documentation.

### Acceptance gate

Closing the window leaves automation running; explicit Quit restores owned Windows state before process exit; autostart either works or reports failure clearly.

## Phase 5 — Compact Vue UI

**Status:** **[PLANNED]**; depends on authoritative backend state and tray lifecycle.

### Deliverables

- compact 380–450 px vertical utility window;
- separate CRD, automation, and tuning status;
- overall and optional per-category automation controls;
- Pause/Resume, Restore Now, Start with Windows, Settings, and diagnostics access;
- backend-driven state updates over Wails bindings/events;
- OS theme following where practical.

### Acceptance gate

- Vue renders verified backend state and never assumes a click changed Windows.
- No dashboard, Performance Options mimic, individual Visual Effects checklist, or Minimal/Recommended/Aggressive/Custom profile UI exists.

## Phase 6 — Hardening

**Status:** **[PLANNED]**; depends on all prior phases.

### Test matrix

- long-running and repeated CRD sessions;
- application crash/restart while connected and after later disconnect;
- Windows login and autostart;
- Explorer restart in Baseline, Active, and Restore states;
- Windows 10 and Windows 11;
- one and multiple monitors, including secondary taskbars;
- Event Log failures/rotation/clear and CRD host/service restart;
- moved/deleted portable executable with autostart enabled;
- unavailable WebView2;
- normal-user permissions and any compatibility exceptions;
- low idle CPU/memory and no aggressive polling.

### Critical scenario gates

| Scenario | Required outcome |
|---|---|
| Normal connect/disconnect | Apply both enabled overrides, then restore exact baseline |
| Start while CRD connected | Reconstruct connection and reconcile tuning |
| Crash while still connected | Restart and remain correctly tuned without replacing baseline |
| Crash, then CRD disconnects | Restart, detect disconnected, restore durable baseline |
| Duplicate connect | Do not overwrite baseline |
| Duplicate disconnect | Do not corrupt or invent state |
| Initial Visual Effects = Custom | Restore exact original values |
| Initial taskbar auto-hide = OFF | Remains OFF after restoration |
| Partial apply | Show partial/error and keep snapshot |
| Restore failure | Keep recovery data and allow retry |
| Close main window | Continue in tray |
| Explicit Quit | Restore owned state before exit |
| Connect followed by disconnect during apply | Serialize transitions; latest desired state eventually wins |

## Product acceptance checklist

### CRD detection

- [ ] Detect normal connection and disconnection automatically.
- [ ] Reconstruct an already-active session at startup.
- [ ] Do not rely solely on process presence.
- [ ] Expose detector failures in diagnostics.

### Visual Effects

- [ ] Apply Windows Best Performance behavior.
- [ ] Capture all prior affected state.
- [ ] Restore arbitrary Custom state exactly.
- [ ] Prevent duplicate connection from overwriting baseline.
- [ ] Expose no Visual Effects editor.

### Taskbar

- [ ] Disable auto-hide during active tuning.
- [ ] Restore original auto-hide state.
- [ ] Preserve an already-OFF baseline.

### Recovery

- [ ] Persist snapshots across restart.
- [ ] Recover from a crash during a CRD session.
- [ ] Retain snapshot until successful restore.
- [ ] Never guess restoration from unknown state.

### Lifecycle and UI

- [ ] Hide main window to tray.
- [ ] Restore owned state on explicit Quit.
- [ ] Start with Windows works or fails clearly.
- [ ] Show CRD and tuning state separately.
- [ ] Support Pause Automation and Restore Now.
- [ ] Remain compact and avoid a dashboard or Windows settings reimplementation.

## Risks and controls

| Risk | Status | Control |
|---|---|---|
| Wrong CRD event assumptions | **[VERIFIED]** resolved | Channel, provider, IDs, and payload captured live; constants may now be coded |
| Lost disconnect leaves stale ownership | **[VERIFIED]** real, mitigated | Observed three times; scope replay to the current host process lifetime and treat dangling connects from a dead PID as disconnected |
| Startup query/subscription gap | **[VERIFIED]** resolved | Subscribe from the bookmark of the last event consumed by the historical query; keep read-existing-events enabled |
| Incomplete Visual Effects snapshot | **[VERIFIED]** resolved | 22-value set captured from the real preset; snapshot spans SPI values, discrete registry values, and the opaque mask; arbitrary `Custom` round-trip passed with no differences |
| Silent state loss via unattributed mask bits | **[VERIFIED]** narrowed, mitigated | Earlier overclaim retracted: the preset touches only attributable bits. Two bits (`byte[2]:0x01`, `byte[4]:0x10`) remain unexplained and untouched by the preset; the mask is still written verbatim and last so they survive regardless |
| Taskbar override silently reverts | **[VERIFIED]** real, mitigated | `ABM_SETSTATE` alone diverges from persisted `StuckRects3` and was observed reverting on its own. Write both layers and treat disagreement as a health signal |
| Lossy per-effect boolean model | **[VERIFIED]** real, mitigated | `FontSmoothing` registry value is `2` while its SPI boolean reads `1`; `IconsOnly` inverts; Explorer-only values have no SPI accessor. Capture all three layers |
| Multiple CRD clients | **[UNVERIFIED]** narrowed | No overlap observed in 191 events; keep the active-client set keyed by the per-session JID resource so either behavior is handled |
| Partial system mutation | **[PLANNED]** | Durable pre-write snapshot, transaction states, verification, retry |
| Race between transitions | **[PLANNED]** | Single serialized coordinator; latest desired state wins |
| Wails v3 Beta change | **[DECIDED]** | Pin v3.0.0-beta.8 exactly and test deliberate upgrades |
| Elevated permissions | **[VERIFIED]** resolved for probed adapters | Event Log query/subscribe, all `SPI_GET*`, and `ABM_GETSTATE`/`ABM_SETSTATE` all succeeded non-elevated |
| Explorer/multi-monitor differences | **[UNVERIFIED]** | Single-display evidence only; include representative test matrix |
| Windows 10 differences | **[UNVERIFIED]** | All evidence is Windows 11 23H2; retest before claiming Windows 10 support |
| WebView2 absent | **[VERIFIED]** detectable | Read `pv` under the WebView2 EdgeUpdate client GUID across HKLM 64-bit, HKLM WOW6432Node, and HKCU |
| Portable path moved | **[PLANNED]** | Detect/document broken autostart registration |
| Sensitive CRD event identity | **[VERIFIED]** real | Account email appears in events 1/2/4 and client `ip:port` in event 4; redact and do not persist |

## Exact next action

Phase 0 is closed for the supported configuration. Begin implementation by scaffolding the pinned Wails project and porting the two proven adapters to Go, since Phase 1 and Phase 2 no longer depend on each other.

```text
1. close the two Phase 1 evidence gaps: taskbar OFF baseline, and VisualFXSetting=0 baseline
2. create the standalone Go module (no Wails)
3. port TaskbarManager first: SHAppBarMessage read, single-bit write, work-area verification
4. port VisualEffectsManager using the verified three-layer snapshot and the four-step write order
5. reproduce the Custom round-trip as an automated Go test
6. in parallel, build CRDDetector on the verified channel, provider, event IDs, session key,
   PID-scoped reconstruction, and bookmark handover
```

Treat `tools/phase0/Get-VisualState.ps1` and `tools/phase0/Restore-VisualState.ps1` as the behavioural specification: the Go implementation must reproduce their results, and `tools/phase0/Test-CustomRoundTrip.ps1` defines the gate it must pass.

Do not claim Windows 10, multi-monitor, or secondary-taskbar support until those environments are actually observed.