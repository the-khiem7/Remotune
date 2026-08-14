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

**[DECIDED]** `StuckRects3` `Settings` byte 8 mirrors the auto-hide bit but requires an Explorer restart to take effect. Remotune uses `ABM_SETSTATE` and does not write `StuckRects3`.

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

**[DECIDED]** Byte 2 (`0x07`) and byte 4 mask `0x10` were set but could not be attributed to any documented effect. Because a partially understood bitfield cannot guarantee an exact round-trip, `UserPreferencesMask` is treated as an **opaque 8-byte blob** that is snapshotted and restored verbatim, while `SystemParametersInfo` is the authoritative per-effect accessor. This preserves unattributed bits by construction.

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

**[VERIFIED]** The preset clears **byte 2 mask `0x04`, which maps to no documented effect**. This is direct proof that a snapshot limited to known effects would silently lose user state, and it converts the opaque-blob decision from caution into a requirement.

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

### Deliverables

- `VisualEffectsManager`: `Snapshot()`, `ApplyBestPerformance()`, `Restore(snapshot)`, `GetCurrentState()`;
- `TaskbarManager`: `GetAutoHide()`, `SetAutoHide(bool)` while preserving unrelated state;
- versioned recovery snapshot schema;
- exact capture/apply/verify/restore operations;
- adapters isolate Win32 and any justified registry details from product logic.

### Acceptance gate

- Best Performance matches the Windows Performance Options behavior.
- A pre-existing arbitrary `Custom` configuration round-trips exactly.
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

- explicitly pinned Wails v3 project;
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
| Silent state loss via unattributed mask bits | **[VERIFIED]** real, mitigated | The preset clears byte 2 mask `0x04`, which maps to no documented effect; the mask is written verbatim and last so the bit survives |
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
1. install the wails3 CLI and scaffold the project pinned to v3.0.0-beta.8
2. port TaskbarManager first: SHAppBarMessage read, single-bit write, work-area verification
3. port VisualEffectsManager using the verified three-layer snapshot and the four-step write order
4. reproduce the Custom round-trip as an automated test against the Go implementation
5. in parallel, build CRDDetector on the verified channel, provider, event IDs, session key,
   PID-scoped reconstruction, and bookmark handover
```

Treat `tools/phase0/Get-VisualState.ps1` and `tools/phase0/Restore-VisualState.ps1` as the behavioural specification: the Go implementation must reproduce their results, and `tools/phase0/Test-CustomRoundTrip.ps1` defines the gate it must pass.

Do not claim Windows 10, multi-monitor, or secondary-taskbar support until those environments are actually observed.