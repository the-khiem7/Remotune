---
baseline_schema: "2.0"
pack: "remotune"
document: "sourcecode"
status: "active"
updated: "2026-08-14"
code_ref: "uncommitted"
---

# Remotune Planned Source Architecture

## Implementation status

Everything in this document is **[PLANNED]** unless explicitly marked **[DECIDED]**, **[VERIFIED]**, or **[UNVERIFIED]**. There is still no application source code; the repository contains documentation plus the Phase 0 evidence tooling in `tools/phase0/`. Exact folders and type names may change; responsibility boundaries must remain.

**[VERIFIED]** items rest on the live observations recorded in [Phase 0 recorded evidence](remotune.roadmap.md#phase-0-recorded-evidence), collected on Windows 11 Pro 23H2 with CRD host 152.0.7977.9 as a non-elevated user. `tools/phase0/Get-VisualState.ps1` is the working reference for the snapshot shape described under [Persistence](#persistence).

## Architecture

```text
Chrome Remote Desktop
        │ connect/disconnect evidence
        ▼
Windows Event Log
        ├─ EvtQuery historical bootstrap
        └─ EvtSubscribe real-time events
        ▼
CRD Detector
        ▼
State Coordinator
        ├──────────────┬──────────────────────┐
        ▼              ▼                      ▼
Recovery Store   VisualEffectsManager    TaskbarManager
                       │                      │
                       ▼                      ▼
               Best Performance         Auto-hide OFF
                       └──────────┬───────────┘
                                  ▼
                       Controlled Windows machine
                                  ▼
                         Authoritative status
                           ┌──────┴──────┐
                           ▼             ▼
                        Tray UI      Vue window
```

## Responsibility topology

```text
Remotune
├─ application
│  └─ StateCoordinator
├─ crd
│  ├─ Detector
│  ├─ EventLogReader
│  ├─ EventSubscriber
│  ├─ EventParser
│  └─ StateReconstructor
├─ windows
│  ├─ VisualEffectsManager
│  ├─ TaskbarManager
│  └─ isolated Win32/platform adapters
├─ persistence
│  ├─ ConfigStore
│  ├─ RecoveryStore
│  └─ StateStore
├─ diagnostics
│  └─ Logger
└─ ui
   ├─ Wails lifecycle/tray/bindings
   └─ Vue frontend
```

### CRD detector

Reliable detection is a prerequisite for all automation. **[DECIDED]** The Windows Event Log is the primary source because it exposes CRD host connection transitions; process presence, service state, sockets, traffic, and CPU activity may be diagnostic only. The detector reports remote-session observations and health and never changes Windows settings.

**[VERIFIED]** Concrete detector inputs:

```text
channel  : Application
provider : chromoting            (legacy event source, remoting_core.dll)
xpath    : *[System[Provider[@Name='chromoting'] and (EventID=1 or EventID=2)]]
event 1  : client connected
event 2  : client disconnected
event 4  : channel information, diagnostic only
payload  : EventData/Data[0] = <email>/chromoting_ftl_<uuid>
host pid : System/Execution/@ProcessID
```

Native API roles:

- `EvtQuery`: inspect recent relevant CRD events, reconstruct startup state, support restart recovery, and provide diagnostics;
- `EvtSubscribe`: receive new CRD transitions in real time.

Startup detection order is mandatory:

```text
load application and recovery state
→ query/reconstruct current CRD state
→ establish real-time subscription without losing the query/subscription boundary
→ reconcile desired Windows state
```

A future-only subscription is insufficient because Remotune may start after CRD is already connected.

**[VERIFIED]** The query/subscription boundary is closed with a bookmark rather than a time window:

```text
EvtQuery replay → retain bookmark of last consumed event
→ EvtSubscribe starting after that bookmark, read-existing-events ENABLED
→ no transition can fall between the two phases
```

Disabling read-existing-events silently reduces the subscription to future-only delivery and reopens the gap. Both a bookmark-seeded reader and a bookmark-seeded watcher replayed exactly the expected missed records during Phase 0.

### Session and lifetime model

**[VERIFIED]** The JID resource component `chromoting_ftl_<uuid>` is unique per session and identical across that session's connect and disconnect. It is the key for the active-client set.

**[VERIFIED]** Connect and disconnect do not reliably alternate. A disconnect is genuinely lost when the CRD host process dies, which was observed three times, each accompanied by a change of the emitting `ProcessID`. Reconstruction must therefore be scoped to the current host process lifetime:

```text
determine the current CRD host process lifetime
→ replay connect/disconnect events within that lifetime, keyed by session id
→ active set non-empty → Connected
→ active set empty     → Disconnected
→ a connect from an earlier host lifetime is NOT an active session
```

Without this scoping, a lost disconnect would strand Remotune in a permanent `Connected` belief and leave Windows tuned indefinitely.

**[VERIFIED]** Host process presence is not connection truth: the `chromoting` service was `RUNNING` with two live `remoting_host` processes while no client was connected.

Detector responsibilities:

- query recent events and reconstruct current state;
- parse relevant connect/disconnect events;
- subscribe to future events;
- deduplicate transitions;
- expose `Unknown`, `Disconnected`, or `Connected` plus detector health/errors;
- handle duplicate events, quick connect/disconnect, start while connected or disconnected, subscription failure, Event Log rotation/clear, stale bookmarks if used, host/service restart, and delayed callbacks;
- track an active-client set if Phase 0 proves multiple clients are possible.

**[UNVERIFIED]** Do not treat any disconnect as “no clients remain” until actual CRD behavior is verified. A single-client simplification is allowed only after evidence establishes it for the supported use case.

A clean internal abstraction is allowed:

```go
type RemoteSessionDetector interface {
    CurrentState(ctx context.Context) (SessionState, error)
    Subscribe(ctx context.Context, events chan<- SessionEvent) error
}
```

Only `CRDDetector` is in current scope. Do not turn this abstraction into a multi-provider feature now.

### VisualEffectsManager

Product-level operations:

```text
Snapshot()
ApplyBestPerformance()
Restore(snapshot)
GetCurrentState()
```

**[DECIDED]** The manager may internally understand every affected Windows value required for correctness, but that detail is not a user-facing settings model.

**[VERIFIED]** The accessor surface is `SystemParametersInfo`. Seventeen `SPI_GET*` actions plus `SPI_GETANIMATION` cover the individual effects and all succeed non-elevated; the constants are tabulated in the Phase 0 evidence. The preset selection itself is `VisualFXSetting` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects` (`0` Let Windows choose, `1` Best appearance, `2` Best performance, `3` Custom).

**[VERIFIED]** Two traps are settled. The 19 `VisualEffects` subkeys hold only `DefaultApplied` bookkeeping and are not effect state. And `UserPreferencesMask` contains set bits that could not be attributed to any documented effect, so it is snapshotted and restored as an **opaque 8-byte blob**; unattributed bits are preserved by construction rather than by understanding them. Per-effect reads and writes go through `SystemParametersInfo`.

**[UNVERIFIED]** The exact value set that Windows' `Adjust for best performance` action changes is still outstanding, because Windows exposes no documented API for the preset. Until it is captured by diffing snapshots around the real operator action, `ApplyBestPerformance()` must not be written. The arbitrary `Custom` round-trip proof depends on the same evidence. Windows 10 behavior is also uncollected.

### TaskbarManager

Product-level operations:

```text
GetAutoHide()
SetAutoHide(bool)
```

**[DECIDED]** It must preserve unrelated taskbar/appbar state.

**[VERIFIED]** `SHAppBarMessage` with `ABM_GETSTATE` (`0x4`) and `ABM_SETSTATE` (`0xA`) is confirmed, non-elevated, with the desired state passed in `APPBARDATA.lParam`. An ON→OFF→ON cycle round-tripped exactly while carrying every non-`ABS_AUTOHIDE` bit through unchanged. Effects are immediate, settle in roughly 1.2 s, and need no Explorer restart.

Write rule:

```text
state = ABM_GETSTATE
target = state with ONLY the ABS_AUTOHIDE bit changed
ABM_SETSTATE target
```

**[VERIFIED]** Outcome verification does not trust the write. Comparing the primary screen work area against its bounds distinguishes the states: on the evidence machine the work-area height was 1080 with auto-hide ON and 1026 with it OFF, against a 54 px taskbar.

**[DECIDED]** `StuckRects3` mirrors the auto-hide bit but requires an Explorer restart, so Remotune never writes it.

**[UNVERIFIED]** Multi-monitor and secondary-taskbar behavior, Explorer-restart interaction, and Windows 10 parity remain uncollected. Bit preservation was also only trivially exercised, since no bits beyond `ABS_AUTOHIDE` were set on the evidence machine.

### Persistence

The active recovery snapshot must be durable before any system mutation. Candidate storage is `%LOCALAPPDATA%\Remotune\`, potentially containing `config.json`, `state.json`, and `logs\`.

Conceptual, not final, snapshot. The shape below reflects Phase 0 evidence: the mask is stored verbatim as an opaque blob, per-effect values are stored as read through `SystemParametersInfo`, and the taskbar stores the whole appbar state rather than just a boolean so unrelated bits can be restored.

```json
{
  "schemaVersion": 1,
  "overrideOwned": true,
  "createdAt": "...",
  "ownedCategories": ["visualEffects", "taskbar"],
  "visualEffects": {
    "visualFxSetting": 1,
    "userPreferencesMask": "0x9E3E078012000000",
    "spi": { "MenuAnimation": 1, "MinAnimate": 1, "...": "one entry per probed SPI action" },
    "registry": { "Desktop.DragFullWindows": 1, "...": "discrete supporting values" }
  },
  "taskbar": {
    "abmState": 1,
    "absAutoHide": true
  }
}
```

`tools/phase0/Get-VisualState.ps1` already produces this shape and is the reference for what a complete capture must include.

Required properties:

- versioned and validated;
- records which categories are owned/affected;
- original baseline cannot be replaced by repeated apply;
- written durably before apply begins;
- retained through partial/error/recovery-required states;
- retired only after verified successful restoration;
- no valid ownership record means no guessed restore.

### StateCoordinator

**[DECIDED]** One coordinator owns all state transitions. Detector callbacks, Vue handlers, tray callbacks, startup, and recovery code submit observations/commands; they do not call Windows managers directly.

Desired state is derived from:

```text
observed CRD state
+ automation enabled/paused
+ enabled automation categories
+ persisted ownership/recovery state
= desired Windows state
```

The coordinator serializes transitions using one worker/event loop. There are no concurrent Visual Effects or taskbar writes. If disconnect arrives while apply is running, the latest desired state eventually wins after serialized reconciliation.

## State model

```text
UNKNOWN
   │ observed/reconstructed state
   ▼
BASELINE
   │ CRD Connected + automation enabled
   ▼
APPLYING
   ├─ complete/verified ──► ACTIVE
   └─ incomplete/error  ──► PARTIAL_ERROR / RECOVERY_REQUIRED

ACTIVE
   │ CRD Disconnected, Pause, Restore Now, or explicit Quit
   ▼
RESTORING
   ├─ complete/verified ──► BASELINE
   └─ incomplete/error  ──► PARTIAL_ERROR / RECOVERY_REQUIRED
```

State dimensions exposed to the UI remain separate:

```text
CRD:        Unknown | Disconnected | Connected
Automation: Enabled | Paused
Tuning:     Baseline | Applying | Active | Restoring | Partial/Error | Recovery Required
```

## Transaction flows

### Apply

```text
1. Serialize through coordinator.
2. If no owned override exists, read complete enabled-category baselines.
3. Persist and validate the recovery snapshot.
4. Mark Applying.
5. Apply Windows Best Performance if Visual Effects automation is enabled.
6. Disable taskbar auto-hide if taskbar automation is enabled.
7. Verify each result where practical.
8. Record ownership/results accurately.
9. Mark Active only if the resulting state justifies it; otherwise Partial/Error.
```

Repeated apply while already active keeps the original snapshot and reconciles safely.

### Restore

```text
1. Serialize through coordinator.
2. Load and validate the owned snapshot.
3. If none exists, report no restorable snapshot and do not guess.
4. Mark Restoring.
5. Restore exact Visual Effects snapshot.
6. Restore original taskbar auto-hide state.
7. Verify each result where practical.
8. Clear ownership and retire snapshot only after successful restoration.
9. Mark Baseline; otherwise retain recovery data and report failure.
```

Repeated restore after successful restoration is a no-op/report, not a new ownership cycle.

## Startup flow

```text
1. Initialize logging.
2. Load config.
3. Load and validate persisted recovery state.
4. Initialize Windows managers.
5. Initialize CRD detector.
6. Query history and reconstruct current CRD state.
7. Start real-time Event Log subscription.
8. Reconcile recovery/ownership, automation config, and observed CRD state.
9. Initialize tray and UI.
10. Emit authoritative status.
```

**[VERIFIED]** The query/subscription race is resolved: step 7 subscribes starting after the bookmark retained from step 6, with read-existing-events enabled, so no transition can be lost between historical reconstruction and live delivery. See [CRD detector](#crd-detector).

Recovery examples:

- persisted ownership + still connected → reconcile remote tuning without replacing baseline;
- persisted ownership + disconnected → restore snapshot;
- tuning-looking Windows state + no ownership → do not assume Remotune caused it.

Recovery does not depend on the main window being visible.

## Shutdown and control flows

### Explicit Quit

```text
stop accepting new automatic transitions
→ serialize pending transition
→ restore owned state
→ persist clean final state
→ stop Event Log subscription
→ release handles/resources
→ remove tray
→ exit
```

### Pause Automation

```text
request pause
→ restore owned state
→ mark automation Paused
→ ignore automatic CRD tuning until resumed
```

### Close window

```text
hide window → keep detector/coordinator/tray running
```

### Restore Now

```text
valid owned snapshot → serialized restore
no valid snapshot    → report no Remotune recovery state; do not guess
```

## Wails and Vue boundaries

Wails owns:

- application lifecycle;
- system tray and menus;
- window show/hide;
- autostart integration;
- Go↔Vue binding/event transport.

Wails does not own tuning decisions or platform mutations.

Vue:

- renders authoritative backend state;
- sends commands such as Pause, Resume, Restore Now, and setting changes;
- waits for backend results/events instead of optimistically claiming Windows changed;
- contains no Win32, CRD detection, ownership, or recovery logic.

## Diagnostics, logging, and privacy

Authoritative diagnostics should include:

- detector health and current CRD state;
- last relevant CRD event without unnecessary identity;
- automation and per-category enablement;
- tuning state and snapshot validity;
- last apply/restore result and last error;
- application version.

Log levels: `INFO`, `WARN`, `ERROR`, `DEBUG`. Log meaningful transitions rather than polling noise, for example baseline captured, apply result, CRD transition, restore result, and errors.

**[VERIFIED]** The redaction requirement is concrete, not hypothetical. Events 1, 2, and 4 carry the account email in the JID, and event 4 additionally carries the client `ip:port`. Neither is needed for state. Parse the JID down to its resource component, which is the session key, and discard the account part before anything is logged or persisted. Useful diagnostics can reference the session by its resource id and the emitting host `ProcessID`.

## Platform quality constraints

- **[VERIFIED]** Normal-user execution is sufficient for every adapter probed so far: Event Log historical query and real-time subscription, all Visual Effects reads, and taskbar read and write. Continue verifying permissions per adapter as new ones appear.
- No aggressive polling; detection is event-driven.
- Keep idle CPU negligible, memory low, and hidden frontend activity minimal.
- Test Explorer restart in Baseline, Active, and Restore states.
- Test Windows 10/11, one/multiple monitors, and secondary taskbars.
- Taskbar recovery after Explorer restart stays within coordinator reconciliation; it does not become a new user-facing feature.
- WebView2 absence and autostart failure are explicit errors, not silent failures.

## Future architecture boundary

Adding a detector such as `RDPDetector`, `RustDeskDetector`, or `AnyDeskDetector` is valid future scope only when explicitly requested. It extends the remote-session trigger. It does not justify new Windows tuning profiles or a replacement control panel.