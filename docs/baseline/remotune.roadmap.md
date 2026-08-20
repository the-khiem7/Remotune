---
baseline_schema: "2.0"
pack: "remotune"
document: "roadmap"
status: "active"
updated: "2026-08-14"
code_ref: "b36da05286ab9df4fce0a89edd5c18c3231e0054 + working tree Phase 5"
---

# Remotune Implementation Roadmap

## Current checkpoint

**[DECIDED]** Product direction is approved. **[IMPLEMENTED]** Phase 1 (Windows tuning adapters), Phase 2 (CRD detector), Phase 3 (StateCoordinator and durable recovery), and Phase 4 (Wails tray shell) all exist in the root Go module `github.com/khiemnguyen/remotune`, each verified against real Windows state and/or a real, live CRD session. **[PLANNED]** The Vue UI (Phase 5) does not exist yet. The repository also contains documentation, the Phase 0 evidence tooling under `tools/phase0/`, and the original standalone `engine/` module (superseded by the root module).

**[VERIFIED]** A Phase 0 evidence spike was executed on 2026-08-14 on the actual Controlled machine and is now substantially complete. All four areas are closed with reproducible live evidence recorded in [Phase 0 recorded evidence](#phase-0-recorded-evidence), including the exact 22-value set that `Adjust for best performance` changes and a passing arbitrary-`Custom` round-trip. Remaining gaps are environment coverage only: Windows 10, multi-monitor and secondary taskbars, Explorer-restart reconciliation, and the Wails runtime API exercising that requires a live app run.

**Phases 1 through 4 are all implemented.** Phase 2 was unblocked independently of Phase 3.

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

#### Write sequence, established during the Go port

The PowerShell reference used `SPIF_UPDATEINIFILE | SPIF_SENDCHANGE` on every per-effect write and then wrote the mask last. It appeared stable, but porting it to Go and running the suite repeatedly exposed it as **racy rather than correct**. Two intermediate orderings were tried and rejected before the cause was measured; the sequence below is the one that survives repeated runs.

```text
1. SystemParametersInfo per-effect writes, SPIF_SENDCHANGE only  -> live session
2. discrete registry values, EXCEPT the preset label            -> persistence
3. UserPreferencesMask, one whole-blob write                    -> persistence
4. settle, then VisualFXSetting alone                           -> preset label
5. verify by re-reading; re-assert only what diverges; broadcast once at the end
```

**[VERIFIED]** Three mechanisms forced this shape, each measured rather than assumed:

| Mechanism | Evidence | Consequence |
|---|---|---|
| `SPIF_UPDATEINIFILE` persists a per-effect write by read-modify-writing the shared mask byte | consecutive writes to effects sharing byte 1 lost each other's bits, and some writes never reached the registry at all | drop the flag; `SystemParametersInfo` changes only the live session and persistence is written explicitly |
| Windows re-labels the configuration as `Custom` asynchronously when individual effects change | `VisualFXSetting` never converged, being overwritten after each of four re-assertions | write the label last, on its own, after a short settle |
| A global `WM_SETTINGCHANGE` mid-sequence makes the shell reload settings from the registry | writes that had already landed were pulled back to whatever the mask said at that instant | do not broadcast inside the write loop; broadcast once after the observed state already matches |

**[DECIDED]** The unifying rule is the same one the taskbar defect produced: **write each layer deliberately instead of hoping one propagates to the other**, then verify the observable outcome. Apply and restore are consequently implemented as one bounded convergence loop that re-asserts only the values still diverging, and reports failure with the residual difference rather than declaring success.

**[VERIFIED]** With this sequence the full suite passed five consecutive clean runs, having previously failed most runs.

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

**Status:** **[IMPLEMENTED]** for the adapter layer, verified by an automated Go suite on 2026-08-14. Remaining Phase 1 work is the durable snapshot store and its persistence, which belongs with the coordinator's recovery data.

Implemented in the standalone module `engine/` (module path `github.com/khiemnguyen/remotune/engine`, Go 1.26.1, only `golang.org/x/sys`, **no Wails dependency**):

| Unit | State |
|---|---|
| `internal/wintune/win32.go` | SPI, appbar, broadcast and metrics bindings, with the uiParam/pvParam distinction encoded as `spiStyle` |
| `internal/wintune/taskbar.go` | `TaskbarManager` writing both the live and persisted layers, with work-area verification |
| `internal/wintune/visualfx.go` | `VisualEffectsManager` with three-layer snapshot, transformation-based apply, exact restore, and `DiffVisualEffects` |
| `internal/wintune/snapshot.go` | versioned schema, validation, and the per-category failure model (`Partial`, `FullyVerified`) |
| `cmd/tbset` | operator utility to inspect or correct taskbar state without running tests |

Test results, `go test ./... -count=1` with `REMOTUNE_SYSTEM_TESTS=1`, all 12 passing:

- arbitrary `Custom` round-trip is exact, with explicit assertions that both unexplained mask bits survive the apply;
- taskbar round-trips from **both** an ON and an OFF baseline, with the two layers agreeing afterwards;
- apply is idempotent;
- restore refuses a nil or incomplete snapshot instead of guessing;
- a partial failure never reports full success, and an unconfirmed write never counts as verified;
- guard tests pin the 10-effect change list, the `IconsOnly` inversion, and the mask clear-mask against the recorded evidence.

The suite earned its keep: running it repeatedly showed that the PowerShell reference behaviour was **racy rather than correct**, and it took several measured iterations to settle. What the tests found:

1. **`SPIF_UPDATEINIFILE` races on the shared mask byte.** The root cause of most flakiness. See the write-sequence table above.
2. **Windows re-labels `VisualFXSetting` asynchronously**, so that value never converged until it was written last and alone.
3. **A mid-sequence global broadcast undoes landed writes.**
4. **Unfair test fixture.** The first `Custom` test built a synthetic snapshot whose per-effect values and mask contradicted each other, a state Windows can never be in. Fixed by deriving the mask from the per-effect values so the layers agree by construction.

Two of my own intermediate conclusions were wrong and are retracted in the ledger as decisions 63 and 64: an ordering change that reasoned from `SPIF_UPDATEINIFILE` being reliable, and a probe too weak to detect what it claimed to rule out. Both were caught by repeated runs rather than by a single passing run, which is the reason the suite is executed several times rather than once.

System-mutating tests are opt-in behind `REMOTUNE_SYSTEM_TESTS=1` and restore the operator's state in `t.Cleanup`, so a casual `go test` cannot reconfigure a desktop.

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

**Status:** **[IMPLEMENTED]**. Both work groups are complete as of 2026-08-14: Group A (historical query, parsing, PID-scoped reconstruction, active-client set) and Group B (live `EvtSubscribe`), including a defect found and fixed during Group B. Implemented in `engine/internal/crd`, same standalone module as Phase 1, no Wails dependency.

Real events cannot be manufactured by Remotune; the detector only reads what CRD itself writes to the Event Log. This phase was split into two work groups for that reason before implementation started:

**[DECIDED]** Group split, agreed with the operator on 2026-08-14 before implementation started:

- **Group A — no live disconnect required.** Historical query, XML parsing, bookmark handling, PID-scoped reconstruction, and the active-client set are all testable against the 191 transition events already recorded in the Event Log (2026-07-23 onward, see [Phase 0 recorded evidence](#phase-0-recorded-evidence)). This is most of Phase 2 and does not touch the operator's active CRD session.
- **Group B — requires one real connect/disconnect cycle.** Verifying the real-time `EvtSubscribe` path needs a transition that happens while a subscription is live, which historical replay cannot substitute for. The operator is currently controlling this machine through the CRD session under test, so this step is deferred to a time of the operator's choosing rather than requested mid-task.

**[DECIDED]** Observation protocol for Group B, so the disconnect does not sever the ability to hand off a result:

```text
1. start a background watcher process BEFORE the operator disconnects, writing to a log file
2. operator sends a message confirming the disconnect is about to happen
3. operator disconnects and reconnects on their own schedule, outside the chat
4. operator's next message (any content) is the signal to read the watcher's log file
   and report whether both the disconnect and the reconnect were captured
```

No observation happened while the operator was disconnected; a background watcher process did the capturing and its log was read back afterward, once, per the protocol above.

### Deliverables

Implemented in `engine/internal/crd`:

| Unit | State |
|---|---|
| `win32.go` | `wevtapi.dll` bindings (`EvtQuery`, `EvtNext`, `EvtRender`, `EvtClose`, `EvtCreateBookmark`, `EvtUpdateBookmark`, `EvtSubscribe`) plus the `kernel32.dll` event primitives pull-mode subscription needs |
| `event.go` | `ParseTransition`, redacting the account email at the parse boundary; verified channel, provider, XPath |
| `history.go` | `QueryHistory`: full historical replay oldest-to-newest plus a bookmark positioned at the newest event |
| `reconstruct.go` | `Reconstruct`: pure, no build tag, PID-scoped active-client set |
| `subscribe.go` | `SubscribeAfterBookmark` / `Poll`: live subscription, gap-free handover from a bookmark |
| `detector.go` | `Bootstrap`: the mandatory startup sequence, query then reconstruct |
| `cmd/crdwatch` | operator observation tool for Group B: bootstraps, subscribes, and appends every transition to a log file |

- historical bootstrap with native `EvtQuery` using the verified channel, provider, and XPath;
- real-time subscription with native `EvtSubscribe` seeded from the historical bookmark, with read-existing-events enabled;
- event parser for the verified event 1/2 payload, extracting the per-session JID resource and the emitting host `ProcessID`;
- current-state reconstructor scoped to the current host process lifetime, treating a dangling connect from a dead PID as disconnected;
- active-client set keyed by the per-session JID resource;
- immediate redaction of account email and client `ip:port`.

**[UNVERIFIED]**, deferred to Phase 6: transition deduplication under adversarial reordering, Event Log rotation/clear fallback, and detector health/error reporting as a first-class signal (errors currently surface as Go `error` values, not yet as a diagnostics-facing health state).

### Phase 2 recorded evidence

**[VERIFIED]** Group A: `TestBootstrapAgainstRealEventLog` and `TestQueryHistoryAgainstRealEventLog` run against the real Event Log on every test run (no gating needed, since Group A performs no mutation). On 2026-08-14 they found 198 real transitions with 0 malformed, and correctly reconstructed the operator's actual live state (`Connected`, 1 active session, current host PID matching the running `remoting_host.exe`) from history alone. `Reconstruct` additionally has 11 unit tests reproducing every Phase 0 anomaly, including the exact host-restart `CONNECT -> CONNECT` sequence.

**[VERIFIED]** Group B closed a real defect, found during the observation protocol itself rather than beforehand:

```text
1. watcher started (bootstrap correctly found the operator's already-Connected session)
2. operator disconnected, then reconnected, twice
3. watcher's log showed zero transitions captured -> defect
```

**[VERIFIED]** Root cause, isolated with raw Win32 probes independent of this package's own wrapper: `EvtSubscribe`'s pull-mode `SignalEvent` must be created **already signaled** (`CreateEventW(NULL, TRUE, TRUE, NULL)`). An ordinarily-defaulted, initially-unsignaled event made `EvtSubscribe`'s own signal never fire, confirmed with a real backlog event already queued past the bookmark. `WaitForSingleObject` on the same handle worked correctly in isolation, which pinpointed the defect to the initial state argument specifically. See ledger decisions 66 to 69; decision 38 is corrected by decision 67.

**[VERIFIED]** After the fix, the same live protocol captured all 4 real transitions from the operator's two disconnect/reconnect cycles, in order, with correct kind, timing, and session ID:

```text
Disconnected (rec 47688) -> Connected (rec 47689, new session)
-> Disconnected (rec 47691) -> Connected (rec 47692, new session)
```

Two permanent regression tests (`TestSubscribeReplaysFromMidHistoryBookmark`, `TestWaitForSignalFiresAfterSubscribeWithExistingBacklog`) now cover this without requiring a future live disconnect: both seed a bookmark mid-history, where a real backlog already exists, and assert it is delivered through the subscription.

### Acceptance gate

- normal connect and disconnect are detected automatically — **[VERIFIED]**, live protocol above;
- starting Remotune during an active CRD session reconstructs `Connected` — **[VERIFIED]**, `TestBootstrapAgainstRealEventLog`;
- process presence is not used as the source of truth — **[VERIFIED]** by construction, `Bootstrap` only calls `QueryHistory` and `Reconstruct`, neither of which inspects running processes;
- duplicate transitions do not create false ownership cycles — **[VERIFIED]**, `TestReconstructDuplicateConnectIsIdempotent` / `TestReconstructDuplicateDisconnectDoesNotCorrupt`;
- detector failures are observable — **[PLANNED]**, currently a Go `error` return, not yet wired to a diagnostics-facing health state (belongs with Phase 3).

## Phase 3 — Coordinator and recovery

**Status:** **[IMPLEMENTED]**, 2026-08-14. Implemented in `engine/internal/application`, same standalone module as Phases 1 and 2, no Wails dependency.

### Deliverables

| Unit | State |
|---|---|
| `state.go` | `TuningState` enum, pure, no build tag: `Unknown / Baseline / Applying / Active / Restoring / Partial-Error / Recovery Required`, plus `CanApply` / `CanRestore` / `IsTransient` |
| `recovery.go` | `RecoveryStore`: atomic durable persistence (temp file same directory + `os.Rename`, atomic on NTFS), `Load` folds missing/corrupt/schema-mismatched files into one `ErrNoRecovery`, `Retire`, `Exists` |
| `coordinator.go` | `Coordinator`: one `sync.Mutex` serializing every method (`Bootstrap`, `Observe`, `Pause`, `Resume`, `RestoreNow`, `Quit`, `Status`); `desiredOwned()` desired-state formula; `applyLocked` / `restoreLocked` transaction flows |
| `startup.go` | `Run`: bootstrap → `Coordinator.Bootstrap` → subscribe from the bootstrap's own bookmark → poll loop feeding `crd.Reconstruct` incrementally |

- one serialized `StateCoordinator` transition loop — **[VERIFIED]**, a single mutex around every method; there is no separate queue, and none is needed;
- desired-state derivation from observed CRD state, automation state, enabled categories, and persisted ownership — **[VERIFIED]**, `desiredOwned()`;
- durable apply and restore transactions — **[VERIFIED]**, see gates below;
- idempotency, race handling, latest-desired-state reconciliation — **[VERIFIED]**, see [Phase 3 recorded evidence](#phase-3-recorded-evidence);
- crash/startup recovery and `Restore Now` — **[VERIFIED]**;
- explicit `Unknown`, `Partial/Error`, and `Recovery Required` states — **[VERIFIED]**, plus a distinction the baseline did not originally separate (see below).

`VisualEffectsAdapter`, `TaskbarAdapter`, and `Bootstrapper`/`Subscription` are declared as local interfaces satisfied by the real `wintune`/`crd` types, rather than depending on those concrete types directly. This exists so coordinator *decisions* can be tested with fakes, fast and repeatably, without mutating the operator's real desktop on every test run — extending the Phase 1 lesson that adapter correctness and coordinator-logic correctness need different kinds of tests.

**[DECIDED]** A restore that verifies successfully but whose durable cleanup (`RecoveryStore.Retire`) fails is `Recovery Required`, not `Partial/Error`. The two are kept distinct because they mean different things to an operator: `Partial/Error` means Windows itself is not fully in the desired state; this case means Windows **is** correctly restored and only the durable record of that could not be deleted. Reporting it as `Partial/Error` would incorrectly imply Windows is left inconsistent.

**[DECIDED]** `Pause` always sets `Paused = true`, even when the restore it triggers is itself partial. Pausing is a deliberate operator command to stop automation; it must be recorded as having happened, with the restore's own failure remaining separately visible through `Status().Tuning`.

**[DECIDED]** `Quit` is idempotent and, once begun, makes `Observe`, `Pause`, and `Resume` return `ErrShuttingDown` rather than accepting further *automatic* transitions — matching the baseline's "stop accepting new automatic transitions." `RestoreNow` is deliberately exempt from this guard: if `Quit`'s own restore attempt ends in `Partial/Error`, an operator or a subsequent retry path must still be able to invoke `RestoreNow` afterward.

### Apply verification gate

```text
read complete baselines
→ persist recovery snapshot
→ state Applying
→ apply enabled categories
→ verify practical outcomes
→ mark ownership/Active only as justified
```

A partial failure must remain `Partial/Error`, retain recovery data, and never be shown as complete success. **[VERIFIED]** by `TestApplyPersistsSnapshotBeforeMutating` and `TestPartialApplyRetainsSnapshotAndReportsPartialError`.

### Restore verification gate

```text
load valid owned snapshot
→ state Restoring
→ restore enabled/owned categories
→ verify practical outcomes
→ clear ownership and retire snapshot only after success
→ state Baseline
```

No valid owned snapshot means no guessed restoration. **[VERIFIED]** by `TestRestoreNowRefusesWithNoOwnedSnapshot`, `TestRestoreRetiresSnapshotOnlyOnFullSuccess`, and `TestPartialRestoreRetainsSnapshot`.

### Phase 3 recorded evidence

**[VERIFIED]** 29 tests in `engine/internal/application`, all passing across 8 consecutive clean runs (`go test ./internal/application -count=1`, repeated per the Phase 1 lesson that a single passing run is not evidence for logic with concurrency or asynchronous dependents). `go test -race` is unavailable on the evidence machine (no cgo/gcc toolchain), so concurrency claims below rest on repeated-run stability and explicit serialization tests, not on the race detector.

Gate coverage, by baseline requirement:

| Requirement | Test | Result |
|---|---|---|
| Snapshot persisted before mutation | `TestApplyPersistsSnapshotBeforeMutating` | PASS |
| Duplicate connect does not replace baseline | `TestRepeatedApplyDoesNotReplaceOriginalBaseline` | PASS |
| Partial apply retains snapshot, reports Partial/Error | `TestPartialApplyRetainsSnapshotAndReportsPartialError` | PASS |
| No categories enabled stays Baseline, nothing persisted | `TestApplyWithNoCategoriesEnabledStaysBaseline` | PASS |
| Restore Now refuses with no owned snapshot | `TestRestoreNowRefusesWithNoOwnedSnapshot` | PASS |
| Snapshot retired only on verified full restore | `TestRestoreRetiresSnapshotOnlyOnFullSuccess` | PASS |
| Partial restore retains snapshot for retry | `TestPartialRestoreRetainsSnapshot` | PASS |
| Repeated restore after Baseline is a no-op | `TestRepeatedRestoreAfterSuccessIsNoOp` | PASS |
| Pause restores owned state and sets Paused | `TestPauseRestoresOwnedStateAndSetsPaused` | PASS |
| Pause sets Paused even on a partial restore | `TestPauseSetsPausedEvenIfRestoreIsPartial` | PASS |
| Resume re-applies if still Connected | `TestResumeReappliesWhenStillConnected` | PASS |
| Quit restores owned state; idempotent | `TestQuitRestoresOwnedStateAndIsIdempotent` | PASS |
| Quit rejects new automatic observations | `TestQuitRejectsNewObservations` | PASS |
| No ownership record ⇒ Baseline regardless of observed Windows state (decision 13) | `TestBootstrapNoOwnershipIsBaselineRegardlessOfCurrentWindowsState` | PASS |
| Bootstrap with ownership + still Connected reconciles without replacing baseline | `TestBootstrapWithOwnershipStillConnectedReconciliesWithoutReplacingBaseline` | PASS |
| Bootstrap with ownership + Disconnected restores | `TestBootstrapWithOwnershipDisconnectedRestores` | PASS |
| 50 concurrent `Observe` calls serialize without corruption; deterministic settle | `TestConcurrentObservationsAreSerializedAndLatestWins` | PASS |
| Connect-then-disconnect-during-apply scenario (critical scenario gate) | `TestDisconnectDuringApplyReconcilesToRestored` | PASS |
| `Status()` remains responsive, does not deadlock | `TestStatusDoesNotBlockDuringLongRunningTransition` | PASS |
| `Run` applies at bootstrap and restores on a live-subscription disconnect | `TestRunAppliesOnBootstrapAndRestoresOnLiveDisconnect` | PASS |
| `RecoveryStore` save/load/retire, atomic overwrite, corrupt/schema-mismatch rejection | 7 tests in `recovery_test.go` | PASS |
| `TuningState` string/transient/CanApply/CanRestore | 3 tests in `state_test.go` | PASS |

**[VERIFIED]** Operator machine confirmed untouched after the full suite: `VisualFXSetting=1`, mask `9E 3E 07 80 12 00 00 00`, taskbar live/persisted both `true`/agreed, and `%LOCALAPPDATA%\Remotune\` does not exist — every test used an isolated `t.TempDir()`, never the real recovery path.

**[VERIFIED]** End-to-end with real adapters: on 2026-08-14, `cmd/e2e` ran the coordinator against the real `wintune.VisualEffectsManager` and `wintune.TaskbarManager` through a full simulated CRD Connected → apply → CRD Disconnected → restore cycle, 3 consecutive clean runs. The operator's exact original state was recovered with no differences, and the recovery store was correctly retired afterward. The `[UNVERIFIED]` gap from the initial Phase 3 documentation is now closed.

## Phase 4 — Wails tray shell

**Status:** **[IMPLEMENTED]**, 2026-08-14. The Wails tray shell is scaffolded and compiles. The `wails3` CLI was absent on the evidence machine, so the project was scaffolded manually following the canonical Wails v3 layout from official examples.

### Deliverables

- root Go module `github.com/khiemnguyen/remotune` (module path change from `engine` sub-module), pinned to Wails `v3.0.0-beta.8`;
- migration of Phase 1 to Phase 3 packages from `engine/internal/` to `internal/` as a file move plus import-path rewrite, with all existing tests still passing and no adapter importing Wails;
- `internal/lifecycle` package: `Service` bridging Wails to the coordinator, `CheckWebView2`, `GetAutostartStatus`/`SetAutostart`, `CheckPortablePath`/`RepairAutostartPath`;
- system tray with context menu: status line, Pause/Resume, Restore Now, Start with Windows (checked toggle), Quit;
- Wails `DisableQuitOnLastWindowClosed: true` so closing the main window leaves automation running;
- explicit Quit sequence: `app.Quit()` → `app.Run()` returns → `svc.Shutdown()` → cancel Run context → wait for poll loop exit → `coord.Quit()` (restore owned state) → process exit;
- Start with Windows via `HKCU\...\Run` with path comparison and toggle; stale-path detection surfaced by `CheckPortablePath`;
- WebView2 prerequisite: `CheckWebView2()` before `application.New`, fatal with actionable message if absent;
- portable executable path: `PortablePathStatus` detects mismatch between running exe and registered autostart path, `RepairAutostartPath` fixes it.

### Phase 4 recorded evidence

**[VERIFIED]** Build: `go build ./...` succeeds with no errors on the evidence machine (Go 1.26.1, Windows 11 23H2).

**[VERIFIED]** Unit tests: `go test ./internal/... -count=1 -short` passes all tests across the three migrated packages (`internal/application` 29 tests, `internal/crd`, `internal/wintune`). No test failures from the migration.

**[VERIFIED]** Adapter isolation: `Select-String -Path "internal\*\*.go" -Pattern "wailsapp"` returns zero matches. No internal package imports Wails.

**[UNVERIFIED]** Live app run (tray visible, WebView2 window creation, tray menu interaction, Quit restore) has not been exercised because `wails3 build` is unavailable and Wails application entry requires the framework runtime. This will be confirmed when the `wails3` CLI is installed or when the app is built with `go build` and executed manually.

**[UNVERIFIED]** Autostart round-trip (write, reboot, verify launch) not exercised. Registry read/write API paths are confirmed individually.

### Acceptance gate

- Closing the window leaves automation running — **[IMPLEMENTED]** via `DisableQuitOnLastWindowClosed: true`;
- explicit Quit restores owned Windows state before process exit — **[IMPLEMENTED]** via the Shutdown sequence;
- autostart either works or reports failure clearly — **[IMPLEMENTED]** via `SetAutostart`/`GetAutostartStatus`/`CheckPortablePath`;
- live verification deferred to first operational run.

## Phase 5 — Compact Vue UI

**Status:** **[IMPLEMENTED]** in the working tree; live Windows acceptance remains pending.

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

## Phase 5 implementation update (2026-08-18)

**[IMPLEMENTED]** The root module now embeds a compact Vue/Vite control surface into the Windows executable. The window starts hidden and is opened from the tray icon or its `Open` menu item. Closing the window hides it again, leaving the lifecycle service running in the tray.

**[IMPLEMENTED]** The UI keeps CRD, automation, tuning, and recovery ownership visibly separate. It exposes only the actual coordinator-backed commands: Pause/Resume, Restore Now, and Start with Windows. Each command is invoked through generated Wails TypeScript bindings and then re-reads authoritative backend state; the UI does not assume a Windows mutation succeeded.

**[IMPLEMENTED]** Docker now installs the pinned Wails CLI, generates bindings, and builds the Vue assets before Go verification or packaging. The UI follows the system light/dark preference and deliberately does not include a dashboard, Windows Performance Options clone, Visual Effects checklist, or tuning presets.

**[UNVERIFIED]** The live Windows acceptance remains: verify tray opening, close-to-tray, and exposed command behavior using the newly packaged executable.

## Exact next action

Phases 1 through 4 are implemented. The root module compiles and all migrated tests pass. The next step is Phase 5: the compact Vue UI.

```text
1. install the wails3 CLI (or confirm go build produces a runnable exe)
2. run the app once to verify tray appears, Quit restores state, autostart toggle writes correctly
3. scaffold the Vue frontend in frontend/ with Vite
4. implement the compact 380–450 px window per Phase 5 deliverables
5. wire Wails bindings/events from coordinator Status to Vue reactivity
```

Do not claim Windows 10, multi-monitor, secondary-taskbar, or multi-client-CRD support until those configurations are actually observed.
