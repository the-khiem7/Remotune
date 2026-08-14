---
baseline_schema: "2.0"
pack: "remotune"
document: "sourcecode"
status: "active"
updated: "2026-08-14"
code_ref: "73f49b65063eacf9953d40d324c9c61e3b4e64eb"
---

# Remotune Planned Source Architecture

## Implementation status

Everything in this document is **[PLANNED]** unless explicitly marked **[DECIDED]** or **[UNVERIFIED]**. There is no application source code in the inspected baseline. Exact folders and type names may change; responsibility boundaries must remain.

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

**[UNVERIFIED]** Phase 0 must determine the exact API/value set and snapshot schema. Supported Win32 APIs are preferred where they map cleanly. Any necessary registry-backed logic must be isolated behind this adapter and thoroughly verified on Windows 10/11.

### TaskbarManager

Product-level operations:

```text
GetAutoHide()
SetAutoHide(bool)
```

**[DECIDED]** It must preserve unrelated taskbar/appbar state. **[UNVERIFIED]** `SHAppBarMessage` with `ABM_GETSTATE` and `ABM_SETSTATE` is the implementation direction pending live evidence.

### Persistence

The active recovery snapshot must be durable before any system mutation. Candidate storage is `%LOCALAPPDATA%\Remotune\`, potentially containing `config.json`, `state.json`, and `logs\`.

Conceptual, not final, snapshot:

```json
{
  "schemaVersion": 1,
  "overrideOwned": true,
  "createdAt": "...",
  "visualEffects": {
    "snapshot": "implementation-defined"
  },
  "taskbar": {
    "autoHide": true
  }
}
```

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

The final implementation must avoid a query/subscription race that loses a transition between historical reconstruction and live subscription. The resolution is **[UNVERIFIED]** and belongs to Phase 0/2 design evidence.

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

Log levels: `INFO`, `WARN`, `ERROR`, `DEBUG`. Log meaningful transitions rather than polling noise, for example baseline captured, apply result, CRD transition, restore result, and errors. Do not persist CRD account/email or unnecessary client identifiers by default.

## Platform quality constraints

- Prefer normal-user execution; verify permissions per adapter.
- No aggressive polling; detection is event-driven.
- Keep idle CPU negligible, memory low, and hidden frontend activity minimal.
- Test Explorer restart in Baseline, Active, and Restore states.
- Test Windows 10/11, one/multiple monitors, and secondary taskbars.
- Taskbar recovery after Explorer restart stays within coordinator reconciliation; it does not become a new user-facing feature.
- WebView2 absence and autostart failure are explicit errors, not silent failures.

## Future architecture boundary

Adding a detector such as `RDPDetector`, `RustDeskDetector`, or `AnyDeskDetector` is valid future scope only when explicitly requested. It extends the remote-session trigger. It does not justify new Windows tuning profiles or a replacement control panel.