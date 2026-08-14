---
baseline_schema: "2.0"
pack: "remotune"
document: "hallucination"
status: "active"
updated: "2026-08-14"
code_ref: "73f49b65063eacf9953d40d324c9c61e3b4e64eb"
---

# Remotune Decision and Uncertainty Ledger

This ledger prevents candidate mechanisms and unresolved research from becoming accidental implementation facts. It is authoritative for Remotune decisions, uncertainties, evidence policy, and external technical references; no historical document is required to interpret it.

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

## Open questions — CRD detector

Every item below is **[UNVERIFIED]** and must be resolved from representative live systems before hard-coding.

- What exact Windows Event Log channel and provider/source does the installed CRD host use?
- Which event IDs and XML/message fields represent connect and disconnect?
- Do useful fields or message text vary by CRD version, locale, installation mode, or update?
- Can a normal user perform historical queries and real-time subscriptions?
- What event history is sufficient to reconstruct current state after Remotune starts late?
- What is the safe fallback after log clear, rotation, missing history, or stale bookmark?
- Can the relevant CRD mode have multiple simultaneous controlling clients? If yes, what stable identifier supports an active-client set?
- What event sequence follows CRD host or service restart, and can delayed callbacks reorder observations?
- What identifying client information appears, and which fields can be discarded/redacted immediately?

## Open questions — Windows Best Performance

Every item below is **[UNVERIFIED]**.

- Which supported Win32 system parameter APIs map cleanly to the full Windows Best Performance behavior?
- Which user-level or registry-backed values does the current Windows UI change?
- What complete versioned snapshot representation can restore arbitrary `Custom` combinations exactly?
- How should “Let Windows choose,” Best Appearance, Best Performance, and Custom be represented without losing individual values?
- Which updates need `SystemParametersInfo`, settings broadcasts, Explorer notification, or another mechanism?
- Are changes immediate, and is Explorer restart ever required?
- What differs between supported Windows 10 and Windows 11 builds?
- If registry-backed writes are necessary, what API boundary and verification strategy make them safe?

Do not implement a restore that merely resets the radio selection; exact affected values must round-trip.

## Open questions — taskbar

Every item below is **[UNVERIFIED]**.

- Do `SHAppBarMessage`, `ABM_GETSTATE`, and `ABM_SETSTATE` reliably read/change auto-hide on target Windows 10/11 builds?
- Which unrelated appbar state bits must be preserved during writes?
- How quickly does the Shell reflect changes, and what verifies success?
- How do Explorer restart, multiple monitors, and secondary taskbars affect state and reconciliation?
- Is one OS-level auto-hide state sufficient, or are there target-build quirks requiring adapter logic?

## Open questions — Wails, lifecycle, and distribution

Every item below is **[UNVERIFIED]** until Phase 0 records evidence.

- Which exact Wails v3 release will be pinned?
- What tray/window APIs and shutdown ordering are stable in that release?
- What exact autostart Manager API behavior applies to a portable executable?
- How should Remotune detect and explain a missing WebView2 runtime?
- How can it detect or clearly document a startup registration broken by moving/deleting the executable?
- What permissions are required for each integration, and are there systems where one adapter needs limited elevation?

## Candidate choices, not requirements

The following design choices remain **[PLANNED candidates]**, not mandatory hard-coded designs:

- local storage under `%LOCALAPPDATA%\Remotune\` with possible `config.json`, `state.json`, and `logs\` layout;
- main window width around 380–450 px and the conceptual mock layout;
- optional independent Visual Effects and taskbar automation toggles;
- a development/diagnostic “Apply Best Performance Now” action;
- exact folder and type names in the suggested backend architecture;
- a provider-neutral `RemoteSessionDetector` interface while only `CRDDetector` is implemented;
- event bookmarks, if they prove useful;
- theme following where practical;
- exact diagnostics presentation and log retention policy.

These candidates may be refined without changing the approved product behavior, provided the boundaries and invariants remain intact.

## Evidence policy

- **[DECIDED]** Chromium source demonstrates CRD connect/disconnect Event Log semantics but does not establish the exact installed-machine channel, IDs, fields, permissions, or current-version behavior.
- **[DECIDED]** Microsoft API documentation establishes API contracts but does not prove Remotune's target integration works across target Windows versions, Explorer states, and monitor layouts.
- **[DECIDED]** Build success, formatting, mocks, or static inspection must not be reported as proof of live CRD/Windows behavior.
- **[PLANNED]** Phase evidence should record environment, steps, observed values, verification result, and any unresolved failure; retain final material evidence rather than routine failed attempts.

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