# Remotune — Project Proposal & Implementation Source of Truth

> **Status:** Approved for implementation handoff  
> **Project:** `Remotune`  
> **Platform:** Windows  
> **Primary stack:** Wails v3 + Go + Vue  
> **Product form:** Tray-first background utility  
> **Purpose:** Source of truth for implementer agents  
> **Last consolidated:** 2026-08-13

---

# 0. Document Authority

This file consolidates the complete approved direction for `Remotune`.

Implementer agents should treat this document as the default source of truth for:

- product intent;
- user problem;
- terminology;
- behavioral requirements;
- technical architecture;
- CRD connection detection;
- Windows Visual Effects automation;
- taskbar auto-hide automation;
- persistence and crash recovery;
- UI/UX direction;
- framework choice;
- testing;
- acceptance criteria;
- future scope;
- explicit non-goals.

If an implementation detail is marked **TBD**, **Verify**, **Candidate**, or **Research Required**, it must not be treated as a hard-coded assumption.

The most important rule is:

> **Remotune automates Windows settings that already exist. It does not recreate or replace the Windows settings UI.**

---

# 1. Brand Identity

## 1.1 Official Product Name

```text
Remotune
```

`Remotune` is the official project and product brand.

The previous working name/codename was:

```text
CRD Autotune
```

Do not use `CRD Autotune` as the user-facing product name in new implementation work.

Use `Remotune` consistently for:

- application title;
- tray menu;
- executable name;
- README/project heading;
- package/build metadata where appropriate;
- UI copy;
- release artifacts;
- logs and local application directory names where migration compatibility is not required.

Recommended executable name:

```text
Remotune.exe
```

Recommended local application data directory:

```text
%LOCALAPPDATA%\Remotune\
```

## 1.2 Brand Meaning

The name combines:

```text
Remote + Tune → Remotune
```

It describes the product behavior without locking the brand to one remote desktop provider.

The current implementation is focused on:

```text
Chrome Remote Desktop (CRD)
```

but the brand deliberately remains provider-neutral so future support for other remote-session technologies does not require renaming the product.

## 1.3 Product vs Provider Naming

Use the terms as follows:

```text
Remotune
└─ Product / application brand

Chrome Remote Desktop (CRD)
└─ Current remote-session provider / trigger
```

Therefore:

- say **“Remotune detected a CRD connection”**;
- say **“Remotune applied Best Performance”**;
- do not call the product itself “CRD”;
- do not rename CRD-specific backend components merely to make them provider-neutral when they genuinely implement Chrome Remote Desktop behavior.

Examples of acceptable technical names:

```text
CRDDetector
CrdEventParser
RemoteSessionDetector
```

Examples of user-facing names:

```text
Remotune
Chrome Remote Desktop: Connected
Automatic tuning: Active
```

## 1.4 Brand Positioning

Candidate concise positioning:

> **Remotune automatically tunes Windows for remote desktop sessions and restores the user's original state when the session ends.**

Current CRD-specific product definition:

> **Remotune is a lightweight Windows tray utility that detects Chrome Remote Desktop sessions, switches Windows Visual Effects to Best Performance, disables taskbar auto-hide while remote control is active, and restores the user's previous Windows state after disconnect.**

## 1.5 Brand Boundary

The brand `Remotune` must not be used as justification to broaden the product into a generic Windows tuning suite.

The product remains:

```text
remote-session automation
```

not:

```text
general Windows optimizer
```

The approved product boundary in this proposal remains unchanged.

---

# 2. Executive Summary

`Remotune` is a lightweight Windows tray utility that runs on a computer being controlled through Chrome Remote Desktop.

Its purpose is to automatically switch the Controlled machine into a more remote-friendly Windows configuration while a Chrome Remote Desktop session is active.

The two approved automatic behaviors are:

```text
CRD Connected
├─ Windows Visual Effects → Best Performance
└─ Taskbar Auto-hide      → OFF
```

When CRD disconnects:

```text
CRD Disconnected
├─ Windows Visual Effects → Restore previous state
└─ Taskbar Auto-hide      → Restore previous state
```

The project must **not** recreate the Windows `Performance Options > Visual Effects` screen.

The project must **not** expose all Windows Visual Effects checkboxes as its own feature.

The project must **not** create custom user-facing presets such as:

- Minimal;
- Recommended;
- Aggressive;
- Custom Remote Profile.

Windows already owns that configuration surface.

Remotune only automates the transition.

---

# 3. Background

Chrome Remote Desktop is used with two roles.

## 2.1 Controlling Machine

The machine the user is physically operating.

Current setup:

```text
Controlling machine = work machine
```

## 2.2 Controlled Machine

The machine being remotely controlled.

Its screen is captured, encoded, transferred, and displayed through Chrome Remote Desktop.

Current setup:

```text
Controlled machine = home machine
```

`Remotune` runs on the **Controlled machine**.

That is where:

- the Windows settings exist;
- the Chrome Remote Desktop host runs;
- CRD connection state must be detected;
- the taskbar behavior must be changed.

---

# 4. Problem Statement

A Windows desktop configured for direct physical use is not always ideal for remote use.

Two concrete pain points motivated the project.

---

# 5. Pain Point — Windows Animation Over Remote Desktop

The Controlled machine normally has Windows Visual Effects enabled.

Examples include:

- animations inside windows;
- minimize/maximize animation;
- taskbar animation;
- fade/slide effects;
- window content redraw during interaction;
- other effects managed by Windows Performance Options.

These effects are useful for local appearance, but they increase visual motion.

Conceptually:

```text
More animation on Controlled machine
        ↓
More visual changes over time
        ↓
More screen changes to capture/encode/transmit
        ↓
Network limitations become more noticeable
        ↓
Remote UX feels more laggy / stuttery
```

The project does not attempt to control CRD's codec or transport.

Instead, it reduces unnecessary visual motion at the source by using the Windows configuration that already exists.

---

# 6. Clarification — What “Best Appearance” and “Best Performance” Mean

The terms in this project refer specifically to:

```text
System Properties
→ Advanced
→ Performance
→ Settings
→ Performance Options
→ Visual Effects
```

The Windows UI contains options such as:

```text
○ Let Windows choose what's best for my computer
○ Adjust for best appearance
○ Adjust for best performance
○ Custom
```

along with the existing Visual Effects checkbox list.

When this proposal says:

```text
Best Performance
```

it means the behavior represented by:

```text
Adjust for best performance
```

in this Windows Performance Options UI.

Remotune must not invent a different meaning for the term.

---

# 7. Pain Point — Auto-Hide Taskbar Conflict

The Controlled machine normally uses taskbar auto-hide.

This is desirable when the machine is used directly.

During CRD usage, however, two taskbars exist in the user's effective view:

```text
Controlling machine
└─ Real/local taskbar

Controlled machine
└─ Remote taskbar rendered inside CRD
```

If both are auto-hidden at the bottom edge, moving the mouse to that edge can cause:

- the local taskbar to appear;
- the remote taskbar to appear;
- the local taskbar to cover the remote taskbar.

This creates an annoying edge interaction.

The approved fix is:

```text
CRD Connected
→ Controlled machine taskbar auto-hide OFF
```

The remote taskbar remains visible inside the streamed image.

Then the bottom-edge interaction primarily affects the Controlling machine's real taskbar.

After CRD disconnects:

```text
Taskbar auto-hide
→ restore previous state
```

---

# 8. Final Product Behavior

The approved lifecycle is:

```text
Before CRD
    │
    │ Windows has user-defined state
    ▼
CRD Connected
    │
    ├─ Snapshot previous Visual Effects state
    ├─ Snapshot previous taskbar auto-hide state
    │
    ├─ Apply Windows Best Performance
    └─ Disable taskbar auto-hide
    │
    ▼
Remote Session Active
    │
    ▼
CRD Disconnected
    │
    ├─ Restore previous Visual Effects state
    └─ Restore previous taskbar auto-hide state
    │
    ▼
Original Windows state
```

---

# 9. Why Restore Previous State Instead of Forcing Best Appearance

The original idea used:

```text
Connected    → Best Performance
Disconnected → Best Appearance
```

This is understandable but unsafe for user customization.

Before CRD connects, the user may already be using:

```text
Custom
```

or may have intentionally disabled some Visual Effects.

If Remotune always forces:

```text
Best Appearance
```

after disconnect, it overwrites those user preferences.

Therefore the final approved semantics are:

```text
Connected
→ Snapshot
→ Best Performance

Disconnected
→ Restore Snapshot
```

This is a backend reliability requirement.

It does **not** mean Remotune should expose the Windows checkbox list in its own UI.

---

# 10. Product Principle

The product owns the **automation**, not the Windows configuration model.

Correct responsibility:

```text
Windows
└─ Owns Performance Options / Visual Effects

Remotune
└─ Detects CRD session
   └─ temporarily switches Windows
      └─ restores Windows afterwards
```

Incorrect direction:

```text
Remotune
└─ Rebuild Performance Options
   ├─ Visual Effect checkboxes
   ├─ Remote presets
   ├─ Custom profile editor
   └─ alternative Windows tuning UI
```

The incorrect direction is explicitly rejected.

---

# 11. Explicitly Rejected Product Direction

The following ideas were discussed and rejected as overengineering:

```text
Remote Profile Engine
Minimal profile
Recommended profile
Aggressive profile
Custom profile
Per-effect user-facing checkbox list
Reimplementation of Performance Options
Mimicking Windows Visual Effects UI
```

Do not implement these unless product direction is explicitly changed later.

The backend may need to understand enough individual Windows state to restore it safely.

That complexity must stay internal.

---

# 12. Scope

The approved functional scope is intentionally small.

## 11.1 Automation Controls

Remotune may allow the user to enable/disable:

```text
Automatic tuning
```

and optionally the two main automation categories:

```text
Visual Effects automation
Taskbar auto-hide automation
```

These toggles control whether Remotune performs the existing Windows actions.

They do not expose the internal Visual Effects checklist.

## 11.2 Manual Recovery

The application should provide:

```text
Restore Now
```

so the user can immediately restore the saved baseline if required.

A manual `Apply Best Performance Now` / diagnostic action may exist for development or troubleshooting, but should not become the primary product workflow.

---

# 13. Non-Goals

Remotune is not:

- a Chrome Remote Desktop replacement;
- a remote desktop client;
- a network optimizer;
- a codec optimizer;
- a Windows debloater;
- a generic system tweaker;
- a replacement for Windows Performance Options;
- a Visual Effects editor;
- a Windows customization suite;
- a dashboard;
- an app expected to remain open on-screen.

---

# 14. Product Form — Tray-First Utility

The application should be designed as a background tray utility.

The reference product direction is **G-Helper**.

What should be learned from G-Helper:

- compact utility window;
- quick status;
- minimal navigation;
- fast controls;
- tray-centered lifecycle;
- open only when needed;
- lightweight feel.

What should **not** be copied:

- its domain-specific controls;
- its exact layout;
- its density if Remotune does not need it;
- any unnecessary settings.

The goal is:

> **G-Helper-inspired utility behavior, not a G-Helper clone.**

---

# 15. UX Philosophy

The desired user relationship with the app is:

```text
Configure once
→ leave it in the tray
→ use CRD normally
→ Remotune reacts automatically
```

The product succeeds if the user rarely needs to open it.

The main window is a control surface.

The background logic is the real product.

---

# 16. Main Window Direction

Recommended dimensions:

```text
Width: approximately 380–450 px
```

Characteristics:

- compact;
- vertical control-panel layout;
- no large dashboard;
- no left navigation sidebar unless truly necessary;
- Windows-native visual language;
- dark/light theme following the OS if practical;
- simple status text;
- concise settings.

Candidate conceptual layout:

```text
┌──────────────────────────────────────┐
│ Remotune                   ● ON  │
│ ● CRD Connected                     │
├──────────────────────────────────────┤
│ Automatic tuning                    │
│                                      │
│ Visual Effects              Enabled  │
│ Taskbar auto-hide           Enabled  │
├──────────────────────────────────────┤
│ Current state                       │
│ Visual Effects       Best Performance│
│ Taskbar              Always visible │
├──────────────────────────────────────┤
│ Start with Windows             ON    │
│                                      │
│ [ Restore Now ]          [ Settings ]│
└──────────────────────────────────────┘
```

This is conceptual, not a pixel-perfect specification.

---

# 17. UI State Must Be Honest

CRD connection state and tuning state are different concepts.

Example:

```text
CRD: Connected
Automation: Paused
```

is valid.

Therefore do not expose only one ambiguous green/red state.

Recommended backend/UI state dimensions:

```text
CRD State
├─ Unknown
├─ Disconnected
└─ Connected

Automation
├─ Enabled
└─ Paused

Tuning
├─ Baseline
├─ Applying
├─ Active
├─ Restoring
├─ Partial/Error
└─ Recovery Required
```

---

# 18. Tray Menu Direction

Candidate tray menu:

```text
Remotune
────────────────────────
● CRD Connected
✓ Automatic tuning active
────────────────────────
Pause Automation
Restore Windows Settings
────────────────────────
Open
Quit
```

The tray should show useful status without requiring the main window.

---

# 19. Close vs Quit

Closing the main window should normally mean:

```text
Hide to tray
```

not:

```text
Terminate application
```

Explicit `Quit` from the tray means the user wants background automation to stop.

Before quitting, Remotune should restore any settings that it currently owns.

---

# 20. Framework

Approved stack:

```text
Wails v3
Go
Vue
```

---

# 21. Why Wails

Wails is appropriate because:

- Go is a good fit for Windows-native integration;
- Go supports simple compiled distribution;
- frontend assets can be bundled with the native application;
- Vue is suitable for a compact control UI;
- resource use is expected to be lighter than Electron-style distribution;
- the backend can remain event-driven.

---

# 22. Why Wails v3

Wails v3 is selected over Wails v2 for this new project because the product strongly benefits from:

- first-class system tray support;
- background utility lifecycle;
- autostart management;
- improved application/window model.

The project accepts the Wails v3 Beta status.

Implementation requirements:

- pin an explicit version;
- test upgrades intentionally;
- do not follow an uncontrolled floating version.

---

# 23. Distribution

Desired primary distribution:

```text
Remotune.exe
```

The goal is a portable utility that can be downloaded and run without a traditional installer for the basic use case.

---

# 24. WebView2 Dependency

Wails on Windows uses WebView2.

Therefore:

```text
Single application executable
```

does not mean:

```text
Zero operating-system runtime dependencies
```

The application should:

- document the WebView2 requirement;
- fail clearly if the runtime is unavailable;
- avoid misleading claims about completely dependency-free execution.

---

# 25. Portable Autostart Caveat

If the user enables:

```text
Start with Windows
```

the startup registration points to the current executable path.

If the user later moves or deletes the portable executable, that registration can become invalid.

The app should handle or document this gracefully.

---

# 26. CRD Connection Detection

Reliable connection detection is the most important technical component.

If detection is unreliable, automation is unreliable.

---

# 27. Do Not Use Process Presence as Source of Truth

Do not infer active connection solely from:

```text
remoting_host.exe is running
```

The CRD host infrastructure can be running while no client is connected.

Therefore:

```text
CRD process running
≠
active CRD remote session
```

Process state can be diagnostic information only.

---

# 28. Do Not Use Network Heuristics as Primary Detection

Do not primarily infer connection from:

- open sockets;
- service state;
- network traffic;
- CPU usage.

These are weaker signals than actual CRD host connection events.

---

# 29. Chosen Detection Direction — Windows Event Log

Chromium/Chrome Remote Desktop emits host-status information to Windows Event Log.

Relevant event semantics include:

```text
Client connected
Client disconnected
```

The implementation should use Windows Event Log as the primary source of truth.

---

# 30. Important Verification Requirement

The project must verify the actual CRD Event Log data on a real target machine before hard-coding:

- channel;
- provider/source;
- event IDs;
- XML fields;
- message format;
- client identifier fields;
- permissions.

Chromium source confirms the existence of connect/disconnect event semantics.

It does not remove the need to inspect the currently installed CRD version.

---

# 31. Event Log APIs

Use native Windows Event Log APIs.

## Historical Query

```text
EvtQuery
```

Purpose:

- inspect recent relevant CRD events;
- reconstruct state during app startup;
- recover after restart;
- diagnostics.

## Real-Time Subscription

```text
EvtSubscribe
```

Purpose:

- receive new CRD connection transitions in real time.

---

# 32. Critical Startup Edge Case

Example:

```text
13:00 CRD connects
13:05 Remotune starts
```

If the app only subscribes to future events, it misses the 13:00 connection.

Therefore startup must be:

```text
Load application state
→ Query/reconstruct current CRD state
→ Start real-time subscription
→ Reconcile desired Windows state
```

This is a core requirement.

---

# 33. CRD Detector Responsibilities

Suggested responsibilities:

```text
CRD Detector
├─ query recent events
├─ parse relevant events
├─ reconstruct current session state
├─ subscribe to future events
├─ deduplicate transitions
├─ expose detector health
└─ report errors
```

The detector should not directly change Windows settings.

It only reports session state.

---

# 34. Event Edge Cases

Handle:

- duplicate `Connected`;
- duplicate `Disconnected`;
- quick connect/disconnect;
- app start while connected;
- app start while disconnected;
- subscription failure;
- Event Log rotation;
- Event Log clear;
- stale bookmark if bookmarks are used;
- CRD host restart;
- service restart;
- delayed callbacks.

---

# 35. Multiple Client Sessions

Do not assume:

```text
Any Disconnect event = no remote clients remain
```

until real CRD behavior is verified.

If the installed CRD mode/version can have multiple simultaneous clients, detector logic must track active clients.

If CRD guarantees one controlling client in the actual use case, implementation may simplify after verification.

---

# 36. Windows Visual Effects Automation

The product requirement is specifically:

```text
CRD Connected
→ Windows Performance Options behaves as Best Performance
```

The user-facing Windows control already exists.

Remotune must automate it.

---

# 37. Important Implementation Boundary

The implementer may need to interact with multiple underlying Windows settings because `Best Performance` represents a collection of Visual Effects.

That internal complexity is acceptable.

The following is not acceptable:

```text
Remotune UI
→ expose all those individual settings to the user
```

Backend complexity for correctness:

```text
Allowed
```

User-facing reimplementation of Windows configuration:

```text
Rejected
```

---

# 38. Visual Effects Snapshot Requirement

Before applying Best Performance, Remotune must capture enough of the current Visual Effects state to restore it exactly.

Possible baseline states include:

```text
Best Appearance
Best Performance
Let Windows choose
Custom
```

A `Custom` state may contain arbitrary checkbox combinations.

Therefore saving only the radio selection may be insufficient.

Implementation research must determine the reliable Windows representation of all state changed by Best Performance.

The recovery requirement is:

> **After CRD disconnects, the relevant Visual Effects state must match what existed before Remotune changed it.**

---

# 39. Do Not Fake Restore

Bad approach:

```text
Before:
Custom

Connected:
Best Performance

Disconnected:
Best Appearance
```

This destroys the original custom state.

Another bad approach:

```text
Before:
Custom

Connected:
Best Performance

Disconnected:
Set radio back to Custom
```

if the individual checkbox values were lost.

Correct implementation must preserve the actual affected state.

---

# 40. Implementation Research for Best Performance

A technical spike must verify how current Windows versions represent and apply the Visual Effects preset.

Possible mechanisms may involve:

- supported Win32 system parameter APIs;
- user-level configuration state;
- Explorer/Performance Options state;
- registry-backed values where Windows itself stores the configuration.

Do not choose a fragile mechanism before verifying:

- Windows 10 behavior;
- Windows 11 behavior;
- immediate application of changes;
- exact restore;
- whether Explorer restart is required;
- whether settings broadcast properly.

Prefer supported Windows APIs where they map cleanly.

If registry-backed state is necessary for exact preset behavior, isolate it behind a dedicated Windows adapter and test it thoroughly.

---

# 41. Taskbar Auto-Hide Automation

Approved behavior:

```text
CRD Connected
→ Taskbar auto-hide OFF
```

After disconnect:

```text
→ restore previous auto-hide state
```

---

# 42. Taskbar API Direction

Windows Shell exposes taskbar/appbar state through:

```text
SHAppBarMessage
```

Relevant messages include:

```text
ABM_GETSTATE
ABM_SETSTATE
```

The implementation should use the Shell API where practical rather than restarting Explorer or relying on shell scripts.

---

# 43. Taskbar State Preservation

If the taskbar is already not auto-hidden before CRD:

```text
Before: Auto-hide OFF
Connected: Auto-hide OFF
Disconnected: Auto-hide OFF
```

If it is auto-hidden before CRD:

```text
Before: Auto-hide ON
Connected: Auto-hide OFF
Disconnected: Auto-hide ON
```

Remotune never assumes the baseline.

It reads it.

---

# 44. Snapshot and Restore

Snapshot/restore is the primary safety mechanism.

Conceptual snapshot:

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

The exact Visual Effects representation is implementation-dependent and must be defined by the technical spike.

---

# 45. Ownership Rule

Remotune must know whether an override belongs to it.

Key rule:

```text
No valid snapshot
→ do not guess what to restore
```

If the app sees Windows in Best Performance but has no ownership record, it cannot assume it caused that state.

---

# 46. Baseline Must Not Be Overwritten

Critical bug to avoid:

```text
Connected event #1
→ snapshot original state
→ apply Best Performance

Connected event #2
→ snapshot Best Performance
```

The second snapshot would destroy the actual baseline.

Correct:

```text
if no owned override exists:
    snapshot baseline
else:
    keep existing baseline
```

---

# 47. Persistence

Candidate storage location:

```text
%LOCALAPPDATA%\Remotune\
```

Candidate files:

```text
config.json
state.json
logs\
```

The exact structure may vary.

The active recovery snapshot must be durable.

---

# 48. Apply Transaction

Applying remote tuning should be transaction-like.

Conceptual flow:

```text
1. Read Visual Effects baseline
2. Read taskbar auto-hide baseline
3. Persist recovery snapshot
4. Mark state = Applying
5. Apply Best Performance if enabled
6. Disable taskbar auto-hide if enabled
7. Verify results where practical
8. Mark override as owned
9. Mark state = Active
```

If part of the operation fails:

- report `Partial/Error`;
- retain recovery data;
- never report full success falsely.

---

# 49. Restore Transaction

Conceptual flow:

```text
1. Load valid owned snapshot
2. Mark state = Restoring
3. Restore Visual Effects state
4. Restore taskbar auto-hide state
5. Verify where practical
6. Clear ownership only after successful restore
7. Retire the snapshot
8. Mark state = Baseline
```

Never delete the only recovery data before successful restore.

---

# 50. Crash Recovery

Crash recovery is mandatory.

Scenario:

```text
CRD connected
→ snapshot saved
→ Best Performance applied
→ taskbar auto-hide disabled
→ Remotune crashes
→ CRD later disconnects
→ Remotune starts again
```

On restart:

```text
Load persisted recovery state
        ↓
Detect current CRD state
        │
        ├─ Still connected
        │   → reconcile remote tuning
        │
        └─ Disconnected
            → restore snapshot
```

The app must not strand Windows in its temporary remote state.

---

# 51. State Coordinator

A single coordinator should own transitions.

Do not scatter direct system mutations across:

- event callbacks;
- Vue handlers;
- tray menu callbacks;
- startup code.

All commands go through one state coordinator.

---

# 52. Core State Machine

Conceptual:

```text
BASELINE
   │
   │ CRD Connected
   ▼
APPLYING
   │
   ▼
ACTIVE
   │
   │ CRD Disconnected
   ▼
RESTORING
   │
   ▼
BASELINE
```

Additional states:

```text
UNKNOWN
PARTIAL_ERROR
RECOVERY_REQUIRED
```

---

# 53. Desired-State Model

The coordinator should derive desired state from:

```text
Observed CRD state
+
Automation enabled/paused
+
Enabled automation categories
+
Persisted ownership/recovery state
=
Desired Windows state
```

Then reconcile actual state.

This is safer than:

```go
onConnect() {
    setBestPerformance()
}
```

with no broader lifecycle model.

---

# 54. Race Conditions

Example:

```text
Connected
→ Apply starts
→ Disconnected arrives before Apply ends
```

The coordinator must serialize system transitions.

Recommended:

- one transition worker/event loop;
- no concurrent Visual Effects writes;
- no concurrent taskbar writes;
- latest desired state eventually wins.

---

# 55. Idempotency

Operations should be safe when repeated.

Examples:

```text
Apply while already active
→ should not destroy baseline

Restore while already restored
→ should not invent a new baseline
```

Duplicate events should not create duplicate ownership cycles.

---

# 56. Explicit Quit

Recommended behavior:

```text
User clicks Quit
→ stop accepting new automatic transitions
→ restore owned state if needed
→ persist clean final state
→ close detector subscription
→ exit
```

---

# 57. Pause Automation

Recommended semantics:

```text
Pause Automation
→ restore owned Windows state
→ stop responding automatically to CRD transitions
```

This keeps `Paused` intuitive.

The app should not remain silently in a temporary Best Performance override after automation is paused.

---

# 58. Restore Now

`Restore Now` is a safety/recovery command.

Behavior:

```text
If valid owned snapshot exists
→ restore it

If no owned snapshot exists
→ do not guess
→ report that no restorable Remotune snapshot exists
```

---

# 59. Suggested Backend Architecture

```text
Remotune
│
├── application
│   └── StateCoordinator
│
├── crd
│   ├── Detector
│   ├── EventLogReader
│   ├── EventSubscriber
│   ├── EventParser
│   └── StateReconstructor
│
├── windows
│   ├── VisualEffectsManager
│   ├── TaskbarManager
│   └── win32
│
├── persistence
│   ├── ConfigStore
│   ├── RecoveryStore
│   └── StateStore
│
├── diagnostics
│   └── Logger
│
└── ui
    ├── Wails lifecycle
    └── Vue frontend
```

Exact folder names may change.

The separation of responsibilities should remain.

---

# 60. VisualEffectsManager Responsibility

`VisualEffectsManager` should expose product-level operations such as:

```text
Snapshot()
ApplyBestPerformance()
Restore(snapshot)
GetCurrentState()
```

It should not expose a user-facing list of every Visual Effects checkbox.

Internal implementation may be more detailed.

---

# 61. TaskbarManager Responsibility

Product-level operations:

```text
GetAutoHide()
SetAutoHide(bool)
```

It should preserve unrelated taskbar state.

---

# 62. CRD Detector Abstraction

Architecture may use a general internal interface:

```go
type RemoteSessionDetector interface {
    CurrentState(ctx context.Context) (SessionState, error)
    Subscribe(ctx context.Context, events chan<- SessionEvent) error
}
```

Only `CRDDetector` is implemented in current scope.

This is allowed because it keeps dependencies clean.

Do not turn this abstraction into a multi-provider feature now.

---

# 63. Future Remote Providers

Potential future expansion:

```text
RemoteSessionDetector
├─ CRDDetector
├─ RDPDetector
├─ RustDeskDetector
└─ AnyDeskDetector
```

This is acceptable future scope because it extends the **remote-session automation trigger**, not the Windows settings UI.

Do not implement unless explicitly requested.

---

# 64. Wails Responsibilities

Wails should own:

- application lifecycle;
- system tray;
- window show/hide;
- autostart;
- Go ↔ Vue communication.

Wails should not own Windows tuning logic.

---

# 65. Vue Responsibilities

Vue should render authoritative backend state.

Correct:

```text
Backend applies setting
→ Backend verifies/returns result
→ Vue renders result
```

Incorrect:

```text
User clicks toggle
→ Vue assumes Windows changed
```

---

# 66. Start with Windows

The app should support:

```text
Start with Windows
```

This is important because automatic detection is only useful if Remotune is already running when the session begins.

Wails v3 autostart functionality is a suitable implementation direction.

---

# 67. Startup Sequence

Recommended conceptual startup:

```text
1. Initialize logging
2. Load config
3. Load persisted recovery state
4. Initialize Windows managers
5. Initialize CRD detector
6. Reconstruct current CRD state
7. Start Event Log subscription
8. Reconcile recovery state vs CRD state
9. Initialize tray/UI
10. Emit authoritative status
```

Recovery logic must not depend on the main window being visible.

---

# 68. Shutdown Sequence

Recommended:

```text
1. Stop new commands
2. Serialize pending transition
3. Restore owned state on explicit Quit
4. Persist final recovery state
5. Stop Event Log subscription
6. Release handles/resources
7. Remove tray
8. Exit
```

---

# 69. Diagnostics

The app should expose enough information to troubleshoot automation without becoming a dashboard.

Useful diagnostics:

```text
CRD detector state
Current CRD connection state
Last relevant CRD event
Automation enabled/paused
Visual Effects automation enabled/disabled
Taskbar automation enabled/disabled
Current tuning state
Snapshot valid/invalid
Last apply result
Last restore result
Last error
Application version
```

---

# 70. Logging

Recommended log levels:

```text
INFO
WARN
ERROR
DEBUG
```

Log meaningful transitions, not constant polling noise.

Examples:

```text
INFO  CRD connected
INFO  Visual Effects baseline captured
INFO  Best Performance applied
INFO  Taskbar auto-hide disabled
INFO  CRD disconnected
INFO  Previous Windows state restored
```

---

# 71. Privacy

CRD Event Log entries may include identifying client information.

Core functionality does not need to persist such identity.

Therefore:

- do not persist client email/account by default;
- redact unnecessary identifiers;
- store only event data needed for connection-state logic.

---

# 72. Privilege Model

Normal operation should preferably not require the entire app to run as Administrator.

Implementation must verify:

- Event Log read/subscription permissions;
- Visual Effects configuration permissions;
- taskbar state change permissions;
- autostart permissions.

If one integration requires elevation on some systems, treat that as a compatibility problem to solve deliberately.

Do not simply run the whole utility elevated by default.

---

# 73. Multi-Monitor Considerations

Test:

- single monitor;
- multiple monitors;
- Windows 10;
- Windows 11;
- secondary taskbars;
- taskbar auto-hide behavior across monitors.

The initial product only needs to manage the Windows taskbar auto-hide behavior reliably.

Do not invent per-monitor features unless the OS and product need justify them.

---

# 74. Explorer Restart

Test Explorer restart during:

```text
Baseline
Active CRD tuning
Restore
```

Taskbar behavior must remain recoverable.

If Explorer restart changes how Shell state must be reapplied, handle that through the existing coordinator rather than adding a new user-facing feature.

---

# 75. Visual Effects Limitation

`Best Performance` affects Windows visual behavior.

It cannot guarantee that every application stops all animation.

Applications can render custom animations independently.

Product claims should remain accurate:

> Remotune switches Windows Visual Effects to Best Performance during CRD.

Do not claim:

> Remotune disables every animation in every app.

---

# 76. Performance Requirements

The utility should be lightweight.

Goals:

- negligible idle CPU;
- low memory use;
- event-driven CRD detection;
- no aggressive polling;
- no unnecessary frontend activity while hidden;
- quick apply/restore transitions.

---

# 77. Reliability Priority

Priority order:

```text
1. Preserve user Windows state
2. Correctly detect CRD connection state
3. Apply Best Performance reliably
4. Restore reliably
5. Manage taskbar auto-hide reliably
6. Recover from crash/restart
7. Run quietly in the tray
8. UI polish
```

---

# 78. Technical Spike — Mandatory Before Full UI Work

Before building the polished Vue UI, verify the Windows behavior on the actual Controlled machine.

## CRD Event Log

Verify:

- exact channel;
- exact provider;
- exact event IDs;
- connect event;
- disconnect event;
- XML/message format;
- permissions;
- startup reconstruction approach.

## Windows Visual Effects

Verify:

- how to apply exact Windows Best Performance programmatically;
- all state that the action changes;
- how to capture previous state completely;
- how to restore Custom state exactly;
- Windows 10 behavior;
- Windows 11 behavior;
- whether settings need broadcast;
- whether Explorer restart is required.

## Taskbar

Verify:

- `ABM_GETSTATE`;
- `ABM_SETSTATE`;
- immediate behavior;
- restore behavior;
- Explorer restart;
- multi-monitor quirks.

Do not build the final visual interface before these fundamentals are proven.

---

# 79. Implementation Roadmap

The project does not need an artificial throwaway MVP.

The phases describe implementation order toward the finished product.

## Phase 0 — Windows/CRD Research Spike

Deliver:

- verified CRD Event Log structure;
- verified Best Performance apply/restore method;
- verified taskbar API behavior.

## Phase 1 — Windows Tuning Engine

Implement:

```text
VisualEffectsManager
TaskbarManager
Snapshot
Restore
```

Test manually and programmatically.

## Phase 2 — CRD Detector

Implement:

```text
EvtQuery bootstrap
EvtSubscribe realtime
State reconstruction
Error handling
```

## Phase 3 — Coordinator and Recovery

Implement:

```text
State machine
Persistent ownership
Apply transaction
Restore transaction
Crash recovery
Race handling
```

## Phase 4 — Wails Tray Shell

Implement:

```text
Tray
Close-to-tray
Quit
Autostart
Background lifecycle
```

## Phase 5 — Compact Vue UI

Implement only the approved controls/status.

Do not build Windows Visual Effects mimicry.

## Phase 6 — Hardening

Test:

- long-running behavior;
- repeated sessions;
- crashes;
- restarts;
- Explorer restart;
- Windows login;
- Windows 10/11;
- multi-monitor;
- portable executable movement.

---

# 80. Critical Test Scenarios

## Scenario A — Normal Flow

```text
Baseline
→ CRD connect
→ Best Performance
→ taskbar auto-hide OFF
→ CRD disconnect
→ exact baseline restored
```

## Scenario B — App Starts While CRD Is Already Connected

```text
CRD connected
→ start Remotune
→ reconstruct connected state
→ apply tuning
```

## Scenario C — Crash While Connected

```text
Connected
→ tuning active
→ app crash
→ restart while still connected
→ reconcile and remain correctly tuned
```

## Scenario D — Crash Then CRD Disconnects

```text
Connected
→ tuning active
→ app crash
→ CRD disconnects
→ app restarts
→ detect disconnected
→ restore baseline
```

## Scenario E — Duplicate Connected Event

Must not overwrite baseline.

## Scenario F — Duplicate Disconnected Event

Must not corrupt state.

## Scenario G — Visual Effects Started as Custom

```text
Before = Custom
Connect = Best Performance
Disconnect = exact original Custom values
```

## Scenario H — Taskbar Auto-Hide Already OFF

After disconnect it remains OFF.

## Scenario I — Partial Apply Failure

UI reports partial/error.

Snapshot remains usable.

## Scenario J — Restore Failure

Recovery data remains.

User can retry `Restore Now`.

## Scenario K — Close Main Window

App remains active in tray.

## Scenario L — Explicit Quit

Owned Windows state is restored before exit.

---

# 81. Acceptance Criteria

## CRD Detection

- [ ] Detect normal CRD connection automatically.
- [ ] Detect normal CRD disconnection automatically.
- [ ] Reconstruct state when starting during an active session.
- [ ] Does not rely solely on CRD process presence.
- [ ] Detector failures are visible in diagnostics.

## Visual Effects

- [ ] Applies the Windows `Best Performance` behavior.
- [ ] Captures enough prior Visual Effects state to restore exactly.
- [ ] Restores a prior `Custom` configuration correctly.
- [ ] Duplicate connections cannot overwrite baseline.
- [ ] No Visual Effects editor is exposed in Remotune UI.

## Taskbar

- [ ] Auto-hide is disabled during active CRD tuning.
- [ ] Original auto-hide state is restored.
- [ ] Already-disabled auto-hide remains disabled after restore.

## Recovery

- [ ] Snapshot survives app restart.
- [ ] Crash during a CRD session is recoverable.
- [ ] Snapshot is not destroyed before successful restore.
- [ ] Unknown state does not trigger guessed restoration.

## Lifecycle

- [ ] Main window can hide to tray.
- [ ] Explicit Quit restores owned state.
- [ ] Start with Windows works or reports failure clearly.

## UI

- [ ] Compact tray-utility design.
- [ ] CRD state is visible.
- [ ] Tuning state is visible.
- [ ] Automation can be paused.
- [ ] Restore Now exists.
- [ ] No dashboard.
- [ ] No reimplementation of Windows Performance Options.
- [ ] No user-facing Minimal/Recommended/Aggressive/Custom profiles.

---

# 82. Future Scope

Future scope must respect the product boundary.

Good future expansion:

```text
Support RDP
Support RustDesk
Support AnyDesk
Improve CRD detection compatibility
Improve recovery
Improve diagnostics
```

These extend **remote-session automation**.

Bad future expansion:

```text
Build a new Windows performance control panel
Expose every Visual Effects checkbox
Create custom tuning presets
Become a Windows tweak suite
```

These are outside the intended product identity.

---

# 83. Final Product Positioning

Candidate product definition:

> **Remotune is a lightweight Windows tray utility that automatically switches Windows Visual Effects to Best Performance and disables taskbar auto-hide while a Chrome Remote Desktop session is active, then restores the user's previous Windows state when the session ends.**

---

# 84. Key Decisions — Approved

The following decisions are approved:

1. Official product brand is `Remotune`; previous working codename was `CRD Autotune`.
2. The app runs on the Controlled machine.
3. Current setup:
   - Controlling machine = work machine.
   - Controlled machine = home machine.
4. Primary pain point is remote UX degradation from Windows visual animation over constrained network conditions.
5. Secondary pain point is taskbar auto-hide conflict between local and remote desktops.
6. CRD Connected:
   - Windows Visual Effects → Best Performance.
   - Controlled machine taskbar auto-hide → OFF.
7. CRD Disconnected:
   - restore previous Visual Effects state.
   - restore previous taskbar auto-hide state.
8. `Best Performance` refers to Windows `Performance Options > Visual Effects > Adjust for best performance`.
9. Do not automatically force `Best Appearance` on disconnect.
10. Snapshot/restore is mandatory.
11. Do not recreate Performance Options in Remotune.
12. Do not expose Windows Visual Effects checkbox lists.
13. Do not implement Minimal/Recommended/Aggressive/Custom tuning profiles.
14. Product form is tray-first utility.
15. UI direction is inspired by G-Helper.
16. Main window is compact, not dashboard-oriented.
17. Primary CRD detection direction is Windows Event Log.
18. Process presence is not sufficient connection detection.
19. Startup must reconstruct current CRD state.
20. Crash recovery is mandatory.
21. Taskbar automation should use Windows Shell APIs where practical.
22. Framework is Wails v3 + Go + Vue.
23. Wails v3 Beta risk is accepted.
24. Start with Windows is a first-class feature.
25. WebView2 dependency must be acknowledged.
26. Backend owns system state; Vue only renders/control requests.
27. Future expansion should focus on additional remote-session triggers/providers, not replacing Windows settings UI.

---

# 85. Research Items — Verify Before Hard-Coding

## CRD

- Event Log channel.
- Event Log provider/source.
- Event IDs.
- Message/XML format.
- Multiple-client semantics.
- Non-admin permissions.
- Behavior across CRD updates.

## Windows Best Performance

- Reliable programmatic application method.
- Complete list/representation of state changed by the preset.
- Exact snapshot format.
- Exact restoration of arbitrary Custom state.
- Windows 10 differences.
- Windows 11 differences.
- Broadcast/update requirements.
- Explorer restart requirements.

## Taskbar

- Windows 10 behavior.
- Windows 11 behavior.
- multi-monitor behavior.
- Explorer restart behavior.
- preservation of unrelated appbar state.

## Wails

- exact pinned v3 version;
- tray lifecycle;
- autostart;
- WebView2 handling;
- portable path behavior.

---

# 86. Technical References

## Chromium / Chrome Remote Desktop

Chromium resource definitions containing Chromoting host Event Log messages:

https://chromium.googlesource.com/chromium/src/%2B/71810ed2d9be11b585b01f93fd1115bbfe52f7aa/remoting/resources/remoting_strings.grd

Chromium CRD Windows host/service source:

https://chromium.googlesource.com/chromium/src.git/%2B/96e2c5522647935d4be7179c28b3f2359cdf3880/remoting/host/host_service_win.cc

---

## Microsoft — Windows Event Log

EvtSubscribe / event subscription:

https://learn.microsoft.com/en-us/windows/win32/wes/subscribing-to-events

EvtQuery / event queries:

https://learn.microsoft.com/en-us/windows/win32/wes/querying-for-events

Windows Event Log functions:

https://learn.microsoft.com/en-us/windows/win32/wes/windows-event-log-functions

---

## Microsoft — Windows Visual/System Parameters

SystemParametersInfo:

https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfoa

---

## Microsoft — Taskbar/AppBar

SHAppBarMessage:

https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shappbarmessage

ABM_GETSTATE:

https://learn.microsoft.com/en-us/windows/win32/shell/abm-getstate

ABM_SETSTATE:

https://learn.microsoft.com/en-us/windows/win32/shell/abm-setstate

---

## Wails v3

Wails v3:

https://v3.wails.io/

Wails v3 status:

https://v3.wails.io/status/

Wails v3 architecture:

https://v3.wails.io/concepts/architecture/

System tray:

https://v3.wails.io/features/menus/systray/

Autostart / Manager API:

https://v3.wails.io/concepts/manager-api/

Installation / WebView2:

https://v3.wails.io/quick-start/installation/

---

# 87. Final Architecture Summary

```text
Chrome Remote Desktop
        │
        │ connect/disconnect
        ▼
Windows Event Log
        │
        ├─ EvtQuery
        └─ EvtSubscribe
        │
        ▼
CRD Detector
        │
        ▼
State Coordinator
        │
        ├──────────────┬────────────────┐
        ▼              ▼                ▼
Recovery Store   VisualEffectsManager TaskbarManager
                       │                │
                       ▼                ▼
                Best Performance   Auto-hide OFF
                       │                │
                       └──────┬─────────┘
                              ▼
                      Windows Controlled
                              │
                              ▼
                     Authoritative Status
                        ┌─────┴─────┐
                        ▼           ▼
                     Tray UI     Vue Window
```

---

# 88. Final Runtime Summary

```text
CRD DISCONNECTED
      │
      ▼
User's original Windows state
      │
      │ CRD connects
      ▼
Persist baseline snapshot
      │
      ├─ Visual Effects → Best Performance
      └─ Taskbar Auto-hide → OFF
      │
      ▼
CRD ACTIVE
      │
      │ CRD disconnects
      ▼
Restore exact snapshot
      │
      ▼
User's original Windows state
```

---

# 89. Final Engineering Principle

When choosing between more features and more reliability, choose reliability.

When choosing between recreating a Windows setting and automating the Windows setting that already exists, automate the existing setting.

The core invariant is:

> **Every Remotune-owned change must have a durable, reliable path back to the exact user state that existed before Remotune changed it.**

And the product boundary is:

> **Remotune automates remote-session transitions. Windows remains the source of truth for Windows Visual Effects configuration.**
