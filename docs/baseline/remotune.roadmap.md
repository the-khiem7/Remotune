---
baseline_schema: "2.0"
pack: "remotune"
document: "roadmap"
status: "active"
updated: "2026-08-14"
code_ref: "73f49b65063eacf9953d40d324c9c61e3b4e64eb"
---

# Remotune Implementation Roadmap

## Current checkpoint

**[DECIDED]** Product direction is approved. **[PLANNED]** No application code exists yet. **[UNVERIFIED]** The platform integration facts required for a safe implementation have not yet been collected from the actual Controlled Windows machine.

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

**Status:** **[PLANNED]**, blocked on access to representative Windows 10/11 environments and the actual CRD installation.

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

## Phase 1 — Windows tuning engine

**Status:** **[PLANNED]**; depends on the Visual Effects and taskbar portions of Phase 0.

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

**Status:** **[PLANNED]**; depends on the CRD portion of Phase 0.

### Deliverables

- historical bootstrap with native `EvtQuery`;
- real-time subscription with native `EvtSubscribe`;
- event parser and current-state reconstructor;
- transition deduplication and detector health/error reporting;
- verified client/session tracking model;
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
| Wrong CRD event assumptions | **[UNVERIFIED]** | Capture real event evidence before coding constants |
| Incomplete Visual Effects snapshot | **[UNVERIFIED]** | Enumerate every affected value and prove Custom round-trip |
| Multiple CRD clients | **[UNVERIFIED]** | Verify semantics before treating any disconnect as zero clients |
| Partial system mutation | **[PLANNED]** | Durable pre-write snapshot, transaction states, verification, retry |
| Race between transitions | **[PLANNED]** | Single serialized coordinator; latest desired state wins |
| Wails v3 Beta change | **[DECIDED]** | Pin exact version and test deliberate upgrades |
| Elevated permissions | **[UNVERIFIED]** | Test normal-user access; do not elevate whole app by default |
| Explorer/multi-monitor differences | **[UNVERIFIED]** | Include representative test matrix |
| WebView2 absent | **[PLANNED]** | Detect and explain prerequisite clearly |
| Portable path moved | **[PLANNED]** | Detect/document broken autostart registration |
| Sensitive CRD event identity | **[PLANNED]** | Do not persist account/email; redact unnecessary identifiers |

## Exact next action

On the actual Controlled Windows machine, perform one CRD connect/disconnect cycle while inspecting the Windows Event Log, and record the exact log channel, provider, connect/disconnect event IDs, event XML, and normal-user read permissions in a Phase 0 evidence note. Do not hard-code detector constants before this evidence exists.