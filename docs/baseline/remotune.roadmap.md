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

**[VERIFIED]** A Phase 0 evidence spike was executed on 2026-08-14 on the actual Controlled machine. The CRD detector, taskbar, and Wails/prerequisite areas are closed with reproducible live evidence recorded in [Phase 0 recorded evidence](#phase-0-recorded-evidence). **[UNVERIFIED]** The Visual Effects area is partially closed: the value model and accessor API are established, but the `Adjust for best performance` ground-truth diff and the arbitrary `Custom` round-trip proof are still outstanding.

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

**Status:** **[VERIFIED]** for CRD detection, taskbar, and Wails/prerequisites. **[UNVERIFIED]** for the two Visual Effects items listed in [Outstanding Phase 0 items](#outstanding-phase-0-items). Windows 10 coverage and multi-monitor coverage remain uncollected because no such environment was available.

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

Supporting values captured for the snapshot schema: `DwmIsCompositionEnabled` = true; `HKCU\...\DWM` `EnableAeroPeek` = 1, `AlwaysHibernateThumbnails` = 1, `Composition` = 1; `Explorer\Advanced` `ListviewAlphaSelect` = 1, `ListviewShadow` = 1, `TaskbarAnimations` = 1, `IconsOnly` = 0; `Control Panel\Desktop` `DragFullWindows` = 1, `FontSmoothing` = 2, `MenuShowDelay` = 400; `WindowMetrics\MinAnimate` = 1.

### Wails and distribution

**[DECIDED]** Pin Wails **v3.0.0-beta.8**, published 2026-08-12, commit `81a149919f91f2149d3fe9be5a27472ae7617b8e`, confirmed as `@latest` from the Go module proxy. The published sequence runs `alpha.0`–`alpha.102`, `alpha2.103`–`alpha2.122`, then `beta.0`–`beta.8`.

**[VERIFIED]** WebView2 presence is detectable without elevation by reading `pv` under the `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}` EdgeUpdate client key; the observed system-wide install reported 151.0.4129.78. Probe the HKLM 64-bit, HKLM `WOW6432Node`, and HKCU variants to cover per-machine and per-user installs.

### Outstanding Phase 0 items

| Item | Status | Why it is still open |
|---|---|---|
| `Adjust for best performance` ground-truth diff | **[UNVERIFIED]** | Windows exposes no documented API for the preset itself, so the authoritative value set must be captured by diffing snapshots around the real Performance Options action. The dialog was opened and the pre-state snapshot captured; the operator action had not been performed when this checkpoint was written. |
| Arbitrary `Custom` exact round-trip | **[UNVERIFIED]** | Depends on the diff above to know the complete affected value set. |
| Windows 10 behavior | **[UNVERIFIED]** | No Windows 10 environment available. |
| Multi-monitor and secondary taskbar | **[UNVERIFIED]** | Only one display present on the evidence machine. |
| Explorer restart reconciliation | **[UNVERIFIED]** | Not exercised during this spike; deferred to the Phase 6 matrix. |
| Autostart Manager API and moved-portable-path behavior | **[UNVERIFIED]** | Requires the pinned Wails project to exist; belongs to Phase 4. |

## Phase 1 — Windows tuning engine

**Status:** **[PLANNED]**. The taskbar half is unblocked: `SHAppBarMessage` is proven for read, apply, restore, and bit preservation. The Visual Effects half remains blocked on the preset diff and the `Custom` round-trip proof.

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
| Incomplete Visual Effects snapshot | **[UNVERIFIED]** | Capture the preset diff, then prove the Custom round-trip; treat `UserPreferencesMask` as an opaque blob so unattributed bits survive |
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

Close the Visual Effects ground-truth gap on the Controlled machine, because it is the last blocker for Phase 1 and the only Phase 0 item that requires a manual operator action.

```text
1. tools/phase0/Get-VisualState.ps1 -Label 02-preset-bestperformance   (pre-state already saved as 01-baseline-bestappearance)
2. In Performance Options → Visual Effects, select "Adjust for best performance" → Apply → OK
3. Get-VisualState.ps1 -Label 02-preset-bestperformance
4. Get-VisualState.ps1 -Diff 01-baseline-bestappearance,02-preset-bestperformance
5. Restore the operator's original selection, re-capture, and confirm the diff is symmetric
```

The resulting diff is the authoritative set of values that `Adjust for best performance` changes. Record it in [Phase 0 recorded evidence](#phase-0-recorded-evidence), then prove an arbitrary `Custom` state round-trips exactly before writing `VisualEffectsManager`.

Detector constants are now evidence-backed, so **Phase 2 is unblocked and may start in parallel** without waiting for the Visual Effects diff.