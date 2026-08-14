---
baseline_schema: "2.0"
pack: "remotune"
document: "hallucination"
status: "active"
updated: "2026-08-14"
code_ref: "2b96ec90e9d73889291a45f9ce2508d344308aa0"
---

# Remotune Decision and Uncertainty Ledger

This ledger prevents candidate mechanisms and unresolved research from becoming accidental implementation facts. It is authoritative for Remotune decisions, uncertainties, evidence policy, and external technical references; no historical document is required to interpret it.

The Phase 0 spike of 2026-08-14 converted a large share of the original uncertainty into evidence. Observed values live in [Phase 0 recorded evidence](remotune.roadmap.md#phase-0-recorded-evidence); this ledger records only the resulting decisions and what remains genuinely unresolved.

## Closed decisions

All items in this section are **[DECIDED]**.

1. The official product brand is `Remotune`; `CRD Autotune` is a previous codename only.
2. Remotune runs on the Controlled machine. In the current setup, the work machine controls the home machine.
3. CRD is the first provider, but the product brand remains provider-neutral.
4. On CRD connection, enabled automation applies Windows Best Performance and disables Controlled-machine taskbar auto-hide.
5. On CRD disconnection, Remotune restores the exact previous affected Windows state; it does not force Best Appearance.
6. `Best Performance` means the existing Windows Performance Options preset, not a Remotune-defined profile.
7. Exact snapshot/restore and durable crash recovery are mandatory.
8. Windows owns the Visual Effects configuration model. Remotune owns automation only.
9. Remotune will not expose individual Visual Effects controls, recreate Performance Options, or offer Minimal/Recommended/Aggressive/Custom remote profiles.
10. The product is a compact, G-Helper-inspired tray utility, not a G-Helper clone, dashboard, generic optimizer, or tweak suite.
11. CRD Windows Event Log events are the chosen primary detector direction. Host process presence and network/system heuristics are not connection truth.
12. Startup must reconstruct current CRD state before relying on future subscriptions.
13. No snapshot means no guessed restore; Windows being in Best Performance does not prove Remotune owns that state.
14. The original baseline is captured only when no owned override exists; duplicate connect events cannot replace it.
15. Partial failures remain visible and recovery data remains durable until successful restore.
16. One coordinator serializes all system transitions. Event callbacks, Vue, tray handlers, and startup code do not mutate Windows directly.
17. Repeated apply/restore operations and duplicate events must be idempotent.
18. Pause Automation restores owned state before pausing automatic reactions.
19. Explicit Quit restores owned state before exit; closing the window normally hides it to the tray.
20. `Restore Now` acts only on a valid Remotune-owned snapshot.
21. Taskbar control should use Windows Shell APIs where practical and preserve unrelated appbar state.
22. The approved stack is Wails v3, Go, and Vue. Wails v3 Beta risk is accepted, but an explicit version must be pinned.
23. Wails owns application/tray/window/autostart and Go↔Vue integration; it does not own tuning logic.
24. Vue renders authoritative backend state and sends requests; it never assumes the OS changed.
25. Start with Windows is first-class functionality.
26. Primary distribution is intended to be a portable `Remotune.exe`.
27. WebView2 is a Windows runtime dependency and must be acknowledged and handled clearly.
28. Normal operation should preferably avoid whole-app Administrator elevation.
29. CRD identity data such as client email/account is not required for core state and must not be persisted by default.
30. Product claims are limited to switching Windows Visual Effects; Remotune cannot promise that every application stops custom animation.
31. Future provider expansion may add RDP, RustDesk, AnyDesk, or similar detectors, but must not expand Remotune into a Windows settings replacement.
32. When feature breadth conflicts with reliability, preserve user state and reliable restoration first.

## Decisions closed by Phase 0 evidence

Added 2026-08-14. Each item is **[DECIDED]** and rests on live observation recorded in [Phase 0 recorded evidence](remotune.roadmap.md#phase-0-recorded-evidence).

33. The detector reads the **Application** channel filtered to provider `chromoting`, a legacy event source. Event 1 is connect, event 2 is disconnect, event 4 is channel information and is diagnostic only.
34. Session identity is the JID resource component `chromoting_ftl_<uuid>` from `EventData/Data[0]`. It is unique per session and identical across a session's connect and disconnect, so it is the key for the active-client set.
35. Connect and disconnect events must not be assumed to alternate. A disconnect is genuinely lost when the CRD host process dies.
36. State reconstruction is scoped to the current CRD host process lifetime, using `System/Execution/@ProcessID`. A connect left dangling by an earlier host process is treated as disconnected, never as an active session.
37. The startup gap between historical query and live subscription is closed by subscribing from the bookmark of the last event consumed during the historical query. The read-existing-events behavior must stay enabled; disabling it silently degrades the subscription to future-only and reopens the gap.
38. Detector operation requires no elevation. Historical query, XPath filtering, bookmark capture, and real-time subscription all succeeded as a standard user.
39. Taskbar auto-hide is read and written through `SHAppBarMessage` (`ABM_GETSTATE`, `ABM_SETSTATE`). Writes flip only the `ABS_AUTOHIDE` bit and carry every other bit through unchanged. ~~`StuckRects3` is never written because it needs an Explorer restart.~~ **Partly superseded by decision 55:** `StuckRects3` byte 8 bit 0 must also be written, because `ABM_SETSTATE` alone is not durable.
40. Apply and restore outcomes are verified against observable effects rather than trusting the write. For the taskbar, comparing the primary screen work area against its bounds distinguishes auto-hide ON from OFF.
41. `UserPreferencesMask` is snapshotted and restored as an **opaque 8-byte blob**, with per-effect reads and writes going through `SystemParametersInfo`. Revised 2026-08-14: a full attribution pass resolved all but two bits (`byte[2]:0x01` and `byte[4]:0x10`), and the Best Performance preset touches neither. The decision therefore rests on those two unexplained bits being free to preserve verbatim, not on any observed state loss. See decision 54 for the retraction.
42. The 19 `HKCU\...\Explorer\VisualEffects` subkeys hold only `DefaultApplied` bookkeeping and are not a source of effect state.
43. Wails is pinned to **v3.0.0-beta.8** (2026-08-12, commit `81a1499`).
44. WebView2 presence is detected by reading `pv` under the `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}` EdgeUpdate client key, probing the HKLM 64-bit, HKLM `WOW6432Node`, and HKCU locations.
45. CRD events expose the account email in events 1, 2, and 4, and the client `ip:port` in event 4. Both are redacted at parse time and never persisted.
46. `Adjust for best performance` changes an exact, known set of 22 values on the supported configuration. Remotune reproduces that set rather than inventing its own definition of Best Performance. Seven probed effects are deliberately left untouched because the Windows preset does not touch them.
47. A Visual Effects snapshot must capture three layers together: per-effect `SystemParametersInfo` values, the discrete Explorer/DWM/Desktop registry values that have no SPI accessor, and the opaque `UserPreferencesMask`. Any single layer alone is provably incomplete.
48. `FontSmoothing` is not a boolean. Its registry value was `2` while its SPI accessor reported `1`, so restoring from the boolean would silently downgrade ClearType. The registry value is authoritative, and `FontSmoothingType` is captured alongside it.
49. `IconsOnly` is inverted relative to the other effects: the performance preset raises it from 0 to 1. A blanket "disable everything" apply is therefore incorrect.
50. Apply and restore share one write sequence, **settled 2026-08-14 after the Go port measured the original one to be racy**: per-effect `SystemParametersInfo` writes with `SPIF_SENDCHANGE` only, then the discrete registry values except the preset label, then `UserPreferencesMask` as a single whole-blob write, then a settle, then `VisualFXSetting` alone. No global broadcast occurs inside the sequence; one is issued only after the observed state already matches the target. Both operations run as a bounded convergence loop that re-asserts only diverging values and reports the residual difference on failure. See decisions 58 to 60 for the three measured mechanisms behind this shape.
51. Applying Best Performance is deterministic and independent of the starting state; the result from an arbitrary `Custom` start was identical to the operator-produced preset.
52. Phase 1 is built in a standalone, throwaway Go module containing only the Windows adapters and their tests. Phase 4 lets the `wails3` CLI generate its own canonical project, and the Phase 1 packages are then migrated into it. The adapters must therefore not import Wails and must be testable headlessly, so the migration is a file move plus an import-path rewrite rather than a redesign. This trades a known one-time migration for a canonical framework layout and for keeping the tuning engine independent of a Beta framework while it is being proven.
53. `ApplyBestPerformance()` is expressed as a **transformation**, not as a captured target state: change only the values the Windows preset changes, leave the seven untouched effects alone, and never replay a snapshot taken from a different machine. Replaying a stored Best Performance snapshot would propagate machine-specific values such as `FontSmoothingType` and any build-specific mask bits onto other systems. **[VERIFIED]** The transformation was validated to reproduce every Visual Effects value of the operator-produced preset exactly.

## Corrections and superseded decisions

54. **Retraction.** A claim that the Best Performance preset clears a `UserPreferencesMask` bit mapping to no documented effect, offered as proof that a known-effects-only snapshot would silently lose user state, was **wrong**. It came from cross-checking only 13 effects and omitting `DropShadow` and `FlatMenu`. A full toggle-and-observe attribution pass established `byte[2]:0x02` = `FlatMenu` and `byte[2]:0x04` = `DropShadow`. Only `byte[2]:0x01` and `byte[4]:0x10` remain unexplained, and the preset touches neither. Decision 41 stands on the narrower justification recorded there.
55. **Supersedes decision 39's prohibition on writing `StuckRects3`.** A taskbar override applied only via `ABM_SETSTATE` is not durable: the live appbar state and the persisted `StuckRects3` byte 8 were observed to diverge, and the override later reverted to the persisted value on its own. `TaskbarManager` therefore writes **both** layers on every change — `ABM_SETSTATE` for immediate live effect, the `StuckRects3` bit for persistence — snapshots both, and treats disagreement between them as a health signal. The original objection that `StuckRects3` requires an Explorer restart does not apply, because `ABM_SETSTATE` provides the live effect while the registry write only removes the divergence. A bisect over the individual phases of a Visual Effects apply failed to identify a single trigger, so this is treated as a durability defect rather than a specific interaction.
56. **[VERIFIED]** `VisualFXSetting` behaves as a label rather than a driver: writing `0` ("Let Windows choose") did not cause Windows to recompute or overwrite the individual effect values within a few seconds. Restoring the numeric value alongside the exact effect values is therefore sufficient. Whether Windows recomputes those defaults at next logon is **[UNVERIFIED]**.
57. **[VERIFIED]** Apply is idempotent. Applying Best Performance twice changed nothing on the second pass, and restoring a baseline that was already Best Performance introduced no changes.

## Open questions — CRD detector

Resolved on 2026-08-14, single machine and single CRD version:

- **Resolved.** Channel, provider, event IDs, and payload fields — see decisions 33 and 34.
- **Resolved.** Normal-user historical query and real-time subscription both work — decision 38.
- **Resolved.** Sufficient history and the late-start reconstruction rule — decision 36.
- **Resolved.** Stale-bookmark and gap handling — decision 37.
- **Resolved.** Stable identifier for an active-client set — decision 34.
- **Resolved.** Which identifying fields to discard — decision 45.

Still **[UNVERIFIED]**:

- Do event IDs, field order, or message text vary across CRD versions, locales, or installation modes? All evidence comes from host 152.0.7977.9 on one machine.
- Can the relevant CRD mode ever have multiple simultaneous controlling clients? No overlap appeared in 191 events, which narrows but does not close this. The active-client set is retained precisely so either answer is safe.
- What is the exact event sequence following a CRD host or service restart, and can delayed callbacks reorder observations? Host restarts were observed only indirectly, through the PID change that accompanied each lost disconnect.
- What is the safe fallback when the log has been cleared or has rotated past all relevant history? Retention was adequate in the sample but is not guaranteed.
- Does the emitting `ProcessID` remain a reliable host-lifetime discriminator across CRD upgrades, given that two `remoting_host` processes run concurrently?

## Open questions — Windows Best Performance

Substantially resolved on 2026-08-14, on one configuration.

Resolved:

- **Resolved.** The accessor surface: 17 `SPI_GET*` actions plus `SPI_GETANIMATION` cover the individual effects and all work non-elevated. Constants are tabulated in the Phase 0 evidence.
- **Resolved.** Where the preset selection lives (`VisualFXSetting`) and what its four values mean.
- **Resolved.** Which locations are *not* state: the `VisualEffects` subkeys — decision 42.
- **Resolved.** How to represent the mask safely without decoding every bit — decision 41, now proven necessary because the preset clears an unattributed bit.
- **Resolved.** Exactly which values `Adjust for best performance` changes: the 22-value set — decision 46.
- **Resolved.** Arbitrary `Custom` restores exactly; the round-trip gate passed with no differences.
- **Resolved.** How presets and `Custom` are represented without losing individual values: store all three layers and never rely on the radio selection — decision 47.
- **Resolved.** Which mechanism applies the changes and in what order — decision 50. Changes were immediate for the probed effects; no Explorer restart was needed to persist or verify them.
- **Resolved.** The safe boundary for registry-backed writes: they are required, not optional, because Explorer-only values have no SPI accessor. Safety comes from snapshot-verify-diff rather than from avoiding the registry.

Still **[UNVERIFIED]**:

- What differs on Windows 10? All evidence is Windows 11 23H2 build 22631. The affected value set, the mask layout, and the identity of unattributed bits must be re-derived there.
- What does the unattributed `UserPreferencesMask` byte 2 bit `0x04` actually control? Preserving it verbatim makes this safe to leave unknown, but it stays unexplained.
- Do the Explorer-backed values (`ListviewAlphaSelect`, `ListviewShadow`, `TaskbarAnimations`, `IconsOnly`) take visual effect without an Explorer restart? Their persisted values round-tripped correctly, which is what restore fidelity requires, but visual immediacy was not confirmed.
- Does the 22-value set shift across future Windows 11 feature updates?

Do not implement a restore that merely resets the radio selection; exact affected values must round-trip.

## Open questions — taskbar

Largely resolved on 2026-08-14.

Resolved:

- **Resolved.** `SHAppBarMessage` with `ABM_GETSTATE`/`ABM_SETSTATE` reliably reads and changes auto-hide on Windows 11 23H2, non-elevated, with an exact ON→OFF→ON round-trip.
- **Resolved.** Bit preservation: flip only `ABS_AUTOHIDE` and carry the rest through. On the evidence machine `ABS_ALWAYSONTOP` was clear and no other bits were set.
- **Resolved.** Timing and success verification: effect is immediate, settles in roughly 1.2 s, no Explorer restart, and the work-area-versus-bounds comparison confirms the real outcome — decision 40.
- **Resolved.** One OS-level auto-hide state was sufficient on this build.

Still **[UNVERIFIED]**:

- How do multiple monitors and secondary taskbars affect the state and its reconciliation? Only one display was available.
- How does an Explorer restart affect auto-hide state while Remotune owns an override?
- Does Windows 10 behave identically?
- Are there builds where `ABS_ALWAYSONTOP` or other bits are set, making bit preservation observable rather than trivially satisfied?

## Open questions — Wails, lifecycle, and distribution

Partially resolved on 2026-08-14.

Resolved:

- **Resolved.** The pinned release is v3.0.0-beta.8 — decision 43.
- **Resolved.** WebView2 detection mechanism — decision 44. The runtime is present on the evidence machine at 151.0.4129.78.
- **Resolved.** Permissions for the probed adapters: Event Log, Visual Effects reads, and taskbar read/write all work non-elevated.

Still **[UNVERIFIED]**:

- What tray, window, and shutdown-ordering APIs are stable in beta.8? Not exercised; the `wails3` CLI is not yet installed.
- What is the exact autostart Manager API behavior for a portable executable?
- How can a startup registration broken by moving or deleting the executable be detected rather than merely documented?
- How does the application behave when WebView2 is genuinely absent? Detection is proven, but the failure path is untested because the runtime is installed.
- Does any adapter need limited elevation on other systems? Only one machine was probed.

## Candidate choices, not requirements

The following design choices remain **[PLANNED candidates]**, not mandatory hard-coded designs:

- local storage under `%LOCALAPPDATA%\Remotune\` with possible `config.json`, `state.json`, and `logs\` layout;
- main window width around 380–450 px and the conceptual mock layout;
- optional independent Visual Effects and taskbar automation toggles;
- a development/diagnostic “Apply Best Performance Now” action;
- exact folder and type names in the suggested backend architecture;
- a provider-neutral `RemoteSessionDetector` interface while only `CRDDetector` is implemented;
- theme following where practical;
- exact diagnostics presentation and log retention policy.

These candidates may be refined without changing the approved product behavior, provided the boundaries and invariants remain intact.

Two former candidates were promoted by Phase 0 evidence and are no longer optional: event bookmarks are **required** for the gap-free startup handover (decision 37), and the Wails version is now a fixed pin rather than an open choice (decision 43).

## Evidence policy

- **[DECIDED]** Chromium source demonstrates CRD connect/disconnect Event Log semantics but does not establish the exact installed-machine channel, IDs, fields, permissions, or current-version behavior.
- **[DECIDED]** Microsoft API documentation establishes API contracts but does not prove Remotune's target integration works across target Windows versions, Explorer states, and monitor layouts.
- **[DECIDED]** Build success, formatting, mocks, or static inspection must not be reported as proof of live CRD/Windows behavior.
- **[DECIDED]** Phase evidence records environment, steps, observed values, verification result, and any unresolved failure; retain final material evidence rather than routine failed attempts.
- **[VERIFIED]** The Phase 0 spike of 2026-08-14 satisfied this policy for the CRD detector, taskbar, and Wails/prerequisite areas on Windows 11 Pro 23H2 build 22631.6494 with CRD host 152.0.7977.9. Its scope limit is explicit: one machine, one Windows version, one display, one CRD version, one user account. Findings are evidence for that configuration and are not yet generalized.
- **[DECIDED]** Evidence recorded from a single configuration does not close a question that is inherently about variation across versions, locales, or hardware layouts. Such questions stay **[UNVERIFIED]** until a second configuration is observed.

## Technical references carried forward

### Chromium / CRD

- [Chromoting Event Log resource definitions](https://chromium.googlesource.com/chromium/src/%2B/71810ed2d9be11b585b01f93fd1115bbfe52f7aa/remoting/resources/remoting_strings.grd)
- [CRD Windows host/service source](https://chromium.googlesource.com/chromium/src.git/%2B/96e2c5522647935d4be7179c28b3f2359cdf3880/remoting/host/host_service_win.cc)

### Microsoft Event Log

- [Subscribing to events (`EvtSubscribe`)](https://learn.microsoft.com/en-us/windows/win32/wes/subscribing-to-events)
- [Querying for events (`EvtQuery`)](https://learn.microsoft.com/en-us/windows/win32/wes/querying-for-events)
- [Windows Event Log functions](https://learn.microsoft.com/en-us/windows/win32/wes/windows-event-log-functions)

### Microsoft Windows integration

- [`SystemParametersInfo`](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfoa)
- [`SHAppBarMessage`](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shappbarmessage)
- [`ABM_GETSTATE`](https://learn.microsoft.com/en-us/windows/win32/shell/abm-getstate)
- [`ABM_SETSTATE`](https://learn.microsoft.com/en-us/windows/win32/shell/abm-setstate)

### Wails v3

- [Wails v3](https://v3.wails.io/)
- [Status](https://v3.wails.io/status/)
- [Architecture](https://v3.wails.io/concepts/architecture/)
- [System tray](https://v3.wails.io/features/menus/systray/)
- [Autostart / Manager API](https://v3.wails.io/concepts/manager-api/)
- [Installation / WebView2](https://v3.wails.io/quick-start/installation/)

These links are source references, not recorded Phase 0 verification.

58. **[VERIFIED]** `SPIF_UPDATEINIFILE` must not be used for per-effect writes. It persists a write by read-modify-writing the shared `UserPreferencesMask` byte, and consecutive writes to effects sharing one byte were measured losing each other's bits; some writes never reached the registry at all. A later write could also resurrect a byte from a stale copy and undo an explicit mask write, so the live and persisted layers never converged. `SystemParametersInfo` is therefore used with `SPIF_SENDCHANGE` only, changing the live session, while persistence is written explicitly as discrete registry values plus one whole-blob mask write. This is the same rule the taskbar adapter follows: write each layer deliberately rather than hoping one propagates.
59. **[VERIFIED]** Windows re-labels the configuration as `Custom` asynchronously when individual effects change, overwriting `VisualFXSetting` shortly after it is written. The label is therefore written last, on its own, after a short settle. Without this it never converged across four re-assertions. This does **not** contradict decision 56: writing the label does not push its preset onto the individual values, but changing the values does push a new label.
60. **[VERIFIED]** A global `WM_SETTINGCHANGE` broadcast issued in the middle of a write sequence makes the shell reload user settings from the registry, pulling the live session back to whatever the mask held at that instant and undoing writes that had already landed. No broadcast is issued inside the convergence loop; one is issued only once the observed state already matches the target, where a reload can only confirm it.
61. **[DECIDED]** Because the shell applies and reloads these settings asynchronously, a single write pass is never treated as sufficient. Apply and restore are implemented as a bounded convergence loop, currently four attempts, that re-reads the observed state, re-asserts only the values still diverging, and on exhaustion reports failure together with the residual difference. This follows the product's existing rule to verify practical outcomes instead of trusting a write.
62. **[VERIFIED]** `UserPreferencesMask` can be derived from the per-effect values using the attribution table, starting from a base mask so that the two unexplained bits survive. Deriving it keeps the live and persisted layers consistent by construction, and the derivation was cross-checked to produce exactly the recorded clear-mask for the Best Performance transformation.

## Retractions from the Go port

63. **Retraction.** An intermediate revision claimed the write order should put the per-effect SPI writes last, on the reasoning that an SPI write updates both the live session and the mask bit it owns. That ordering was implemented and **measured to be worse**, producing non-deterministic restores in which twenty or more values diverged. The reasoning was wrong because it assumed the mask persistence performed by `SPIF_UPDATEINIFILE` is reliable; decision 58 records that it is not. The settled sequence is in decision 50.
64. **Scope correction.** A probe concluded that writing `VisualFXSetting` does not re-apply its preset over the individual effect values. That probe was weak: it drove the effects to values they already held, so it could not have detected a re-apply. The claim in decision 56 is retained only in the narrow form actually observed — writing the label did not disturb the values — while decision 59 records the converse effect, which is real and was the cause of a non-converging value.

## Phase 2 operating decision

65. **[DECIDED]** Phase 2 CRD detector work is split into Group A (historical query, parsing, bookmark handling, PID-scoped reconstruction, active-client set — testable against the 191 events already in the Event Log, no live disconnect needed) and Group B (real-time `EvtSubscribe` verification, which needs one real connect/disconnect on the operator's active CRD session and is deferred to a time of the operator's choosing). Agreed with the operator on 2026-08-14 because the CRD session in scope for Phase 2 is the same session currently controlling this machine. See [Phase 2 — CRD detector](remotune.roadmap.md#phase-2--crd-detector) for the observation protocol used for Group B.
