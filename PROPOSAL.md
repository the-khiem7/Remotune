# CRD Autotune — Project Proposal & Implementation Source of Truth

> **Status:** Approved direction / implementation handoff  
> **Project:** `CRD Autotune`  
> **Primary platform:** Windows  
> **Primary technology:** Wails v3 + Go + Vue  
> **Product form:** Tray-first background utility  
> **Document purpose:** Single source of truth for implementer agents  
> **Last consolidated:** 2026-08-13

---

## 0. How to Use This Document

This file consolidates the complete project discussion and the technical research performed so far.

Implementer agents should treat this document as the default source of truth for:

- product intent;
- terminology;
- user pain points;
- behavioral requirements;
- architecture;
- CRD connection detection;
- Windows tuning strategy;
- state management and crash recovery;
- UI/UX direction;
- Wails framework choice;
- packaging/distribution;
- implementation order;
- edge cases;
- acceptance criteria;
- known uncertainties and research tasks.

If an implementation detail in this document is marked **TBD**, **verify**, **candidate**, **optional**, or **future**, it is not a final invariant and should be validated before hard-coding it.

The most important product principle is:

> **CRD Autotune must temporarily optimize the Controlled machine for a remote session, then safely restore the user's original Windows state.**

Do **not** reinterpret the product as a generic Windows optimizer, a remote desktop client, or a dashboard.

---

# 1. Executive Summary

`CRD Autotune` is a lightweight Windows background utility that automatically changes selected Windows visual and taskbar behaviors when a Chrome Remote Desktop session becomes active.

The main use case is a user operating a **Controlled machine** through **Chrome Remote Desktop (CRD)** from a different **Controlling machine**, under network conditions where visual animation causes noticeable lag or stutter.

When CRD becomes connected, CRD Autotune temporarily switches the Controlled machine into a **Remote Optimized** state:

- reduce/disable selected Windows animation effects;
- disable taskbar auto-hide so the remote taskbar remains fixed and does not compete with the Controlling machine's local taskbar;
- optionally disable other visual behaviors that cause unnecessary screen mutation, such as full-window redraw while dragging.

When the CRD session ends, the application does **not** blindly force Windows into a predefined “Best appearance” preset. Instead, it restores the **exact state that existed before CRD Autotune applied its override**.

The application is designed as a **tray-first utility** rather than a desktop dashboard. The reference UX direction is similar in spirit to **G-Helper**:

- lightweight;
- compact;
- utility-oriented;
- quick status visibility;
- quick toggles;
- minimal navigation;
- runs quietly in the background;
- main window is opened only when needed.

The desired product experience is:

> Configure once, let it run in the tray, and mostly forget that it exists.

---

# 2. Background

## 2.1 Chrome Remote Desktop Roles

This project uses two explicit terms.

### Controlling machine

The computer physically being used by the user to control another computer.

In the current setup:

- **Controlling machine = work machine**

### Controlled machine

The computer being remotely controlled.

Its desktop is captured, encoded, transmitted, and rendered through Chrome Remote Desktop.

In the current setup:

- **Controlled machine = home machine**

`CRD Autotune` runs on the **Controlled machine**, because that is where Windows visual settings must change and where CRD host state can be detected.

---

# 3. Problem Statement

The project exists because the default local desktop experience is not always the best remote desktop experience.

A Windows environment optimized for a user sitting directly in front of the machine may generate visual behavior that becomes wasteful or annoying when the same desktop is streamed through Chrome Remote Desktop.

Two pain points initiated this project.

---

# 4. Pain Point 1 — Animation Causes Poor Remote UX

## 4.1 Situation

The Controlled machine normally uses Windows animation and visual effects.

When the user is physically sitting at that machine, these effects can improve perceived polish.

When the machine is controlled remotely through CRD over a constrained network, those same effects can become undesirable.

## 4.2 Why It Hurts Remote Desktop

An animation causes the screen to change across many consecutive frames.

Conceptually:

```text
Animation on Controlled machine
        ↓
More pixels change over time
        ↓
More screen updates must be captured
        ↓
More visual data must be encoded
        ↓
More data / update work must traverse the remote path
        ↓
Network weakness becomes more visible
        ↓
Remote UX feels laggy / stuttery
```

The exact internal codec and transport behavior is CRD's responsibility, but the product-level observation is simple:

> Avoiding unnecessary visual motion on the Controlled machine can reduce avoidable screen mutation and improve perceived remote responsiveness.

## 4.3 Current Manual Workaround

The user can manually change Windows visual settings to prioritize performance before a remote session and later restore them.

This is repetitive and easy to forget.

## 4.4 Desired Automation

When CRD connects:

```text
Windows visual state
    → Remote Optimized
```

When CRD disconnects:

```text
Windows visual state
    → exact previous user state
```

---

# 5. Pain Point 2 — Auto-Hide Taskbar Conflict

## 5.1 Current Setup

The Controlled machine normally has Windows taskbar auto-hide enabled.

This is desirable when somebody sits directly at that machine.

## 5.2 What Happens During Remote Control

When using the remote desktop view:

1. The Controlling machine has its own real/local taskbar.
2. The Controlled machine's taskbar is visible only as part of the streamed desktop image.
3. Both taskbars can be associated with the same bottom edge of the user's view.
4. Moving the cursor to the bottom edge can trigger:
   - the Controlling machine's real taskbar;
   - the Controlled machine's auto-hidden taskbar.

The result is an awkward interaction where:

- both try to appear;
- the Controlling machine's real taskbar can overlay or cover the remote taskbar;
- accessing the Controlled machine's taskbar becomes annoying.

## 5.3 Desired Behavior

While CRD is connected:

```text
Controlled machine taskbar auto-hide = OFF
```

The taskbar remains permanently visible inside the remote image.

Therefore, moving the cursor to the local bottom edge only needs to trigger the Controlling machine's taskbar, avoiding the “two taskbars fighting for the same edge” problem.

After CRD disconnects:

```text
Controlled machine taskbar auto-hide = restore original value
```

For the user's current setup, that original value is normally `ON`.

---

# 6. Original Solution vs Finalized Solution

The initial solution was described as:

```text
CRD Connected:
- Best performance
- Taskbar auto-hide OFF

CRD Disconnected:
- Best appearance
- Taskbar auto-hide ON
```

This captures the desired behavior at a high level, but the final implementation semantics should be safer.

## 6.1 Final Model

Use:

```text
CRD Connected
    → Snapshot current state
    → Apply Remote Optimized Profile

CRD Disconnected
    → Restore snapshot
```

Do **not** implement:

```text
Connected    → force Windows "Best performance" preset
Disconnected → force Windows "Best appearance" preset
```

## 6.2 Why Snapshot/Restore Is Required

A user may already have customized Windows.

Examples:

- some animations were manually disabled;
- some effects were enabled;
- taskbar auto-hide was already off;
- full-window dragging was already disabled;
- the machine was not using Microsoft's “Best appearance” preset.

If CRD Autotune always restores “Best appearance”, it overwrites user preference.

Correct principle:

> CRD Autotune owns only its temporary override. It does not own the user's baseline Windows configuration.

---

# 7. Product Goals

## 7.1 Primary Goals

1. Reliably detect CRD remote session connection/disconnection on the Controlled machine.
2. Automatically apply remote-friendly Windows settings when CRD is active.
3. Restore the exact pre-session settings when CRD is no longer active.
4. Prevent CRD Autotune itself from leaving Windows in an incorrect state after:
   - app restart;
   - app crash;
   - forced shutdown;
   - manual quit;
   - duplicate/late connection events.
5. Require minimal user interaction.
6. Run primarily as a background tray utility.
7. Be portable and lightweight.
8. Use native Windows mechanisms where practical instead of PowerShell scripting or fragile registry hacks.
9. Make every automatic action observable through status/diagnostics.
10. Keep the architecture clean enough that additional remote-session providers could be added later without rewriting the Windows tuning engine.

---

# 8. Non-Goals

Unless explicitly added later, CRD Autotune is **not**:

- a Chrome Remote Desktop replacement;
- a remote desktop client;
- a VPN;
- a network accelerator;
- a CRD codec optimizer;
- a generic PC cleanup tool;
- a full “Windows debloater”;
- a gaming performance optimizer;
- a general-purpose system tweaking suite;
- a dashboard that users are expected to keep open;
- a service for managing the Controlling machine;
- a tool that guarantees removal of every animation rendered by every Windows application.

The initial product is intentionally narrow.

---

# 9. Product Philosophy

The product should behave more like infrastructure than an app that demands attention.

Desired user lifecycle:

```text
Download
   ↓
Run CRDAutotune.exe
   ↓
Configure desired automation
   ↓
Enable Start with Windows
   ↓
Close window to tray
   ↓
Use CRD normally
   ↓
CRD Autotune silently reacts in background
```

The best outcome is:

> The user notices that remote sessions feel better, but rarely needs to open CRD Autotune.

---

# 10. Platform and Distribution

## 10.1 Primary Platform

Windows.

The product's core behavior depends on:

- Windows Event Log;
- Win32 system parameter APIs;
- Windows Shell taskbar behavior;
- Windows startup integration;
- Windows tray UX.

Cross-platform support is not currently a product requirement.

## 10.2 Distribution Goal

The original framework preference is based partly on Go/Wails producing a simple native executable.

Desired user experience:

```text
Download CRDAutotune.exe
→ run directly
→ no traditional installer required for the basic use case
```

## 10.3 Important Runtime Caveat

Wails uses the platform's native WebView rather than embedding a full browser.

On Windows, Wails v3 uses WebView2 and therefore expects the Microsoft WebView2 Runtime to exist.

Modern Windows 10/11 machines commonly have it, but CRD Autotune should:

- document the dependency;
- detect/report a missing runtime cleanly where possible;
- not market the binary as having literally zero external OS runtime dependencies.

## 10.4 Portable + Autostart Caveat

A portable executable can be moved.

If `Start with Windows` points to the executable's current path and the user later moves/deletes that file, the autostart entry becomes invalid.

The UI or documentation should make this behavior unsurprising.

---

# 11. Framework Decision

## 11.1 Original Candidate

```text
Wails (TBD v2 or v3) + Vue
```

## 11.2 Final Recommendation

```text
Wails v3 + Go + Vue
```

## 11.3 Why Wails

Reasons already aligned with the project:

- Go backend is suitable for Windows-native integration;
- lightweight compared with shipping a full Electron browser runtime;
- frontend can use Vue;
- compiled frontend assets can ship with the application executable;
- matches the desired portable utility model;
- Go is a good fit for background event coordination and native API bindings.

## 11.4 Why Wails v3 Instead of v2

As of this proposal consolidation on 2026-08-13:

- Wails v3 is in Beta;
- its desktop API is described as stable;
- Wails v2 remains the current stable release;
- Wails v3 has first-class system tray APIs;
- Wails v3 exposes an Autostart manager;
- Wails v3's application/window model is better suited to a background utility.

The project accepts the v3 Beta risk because:

- it is a new utility rather than a legacy enterprise application;
- system tray behavior is central to the product;
- the v3 architecture matches the intended lifecycle better.

Implementation agents should pin an explicit Wails v3 version and test upgrades rather than tracking an uncontrolled moving version.

---

# 12. UX Direction — Tray-First Utility

## 12.1 Decision

CRD Autotune should **not** be designed as a large desktop dashboard.

It should be a:

> **G-Helper-inspired compact tray utility with a Windows-native visual language and automation-first UX.**

G-Helper is used only as a product/UI reference for:

- compact dimensions;
- status-first information;
- small utility controls;
- high information density without becoming a dashboard;
- tray-centered usage;
- “open only when needed” behavior.

Do not clone G-Helper pixel-for-pixel.

## 12.2 Main UX Principle

```text
Background behavior = product
Main window         = control plane
Tray                = primary daily surface
```

## 12.3 Window Characteristics

Recommended:

- compact width approximately 380–450 px;
- vertical compact control-panel layout;
- no sidebar unless future scope truly requires it;
- no full-screen dashboard;
- no large analytics cards;
- dark/light theme following Windows if practical;
- WinUI-like spacing/typography/interaction language;
- frameless/custom titlebar is acceptable if it remains reliable and accessible;
- close button should normally hide to tray, not terminate automation.

The utility should visually feel at home on Windows without needing to mimic WinUI 3 exactly.

---

# 13. Main Window Information Architecture

The main screen should answer four questions immediately:

1. Is automation enabled?
2. Is CRD connected?
3. Is the Remote Optimized profile active?
4. What did CRD Autotune change?

Candidate layout:

```text
┌──────────────────────────────────────┐
│ CRD Autotune                   ● ON  │
│ Chrome Remote Desktop                │
│ ● Connected                          │
├──────────────────────────────────────┤
│ Remote Profile                       │
│                                      │
│   ○ Normal       ● Optimized         │
│                                      │
│ Animation                  Off       │
│ Taskbar Auto-hide          Off       │
│ Full-window drag           Off       │
├──────────────────────────────────────┤
│ Automation                           │
│ ☑ Auto tune on CRD connection        │
│ ☑ Restore after disconnect           │
│ ☑ Start with Windows                 │
├──────────────────────────────────────┤
│ Last session                         │
│ Connected · 16:08                    │
│                                      │
│ [ Restore Now ]          [ Settings ]│
└──────────────────────────────────────┘
```

This is a conceptual layout, not a pixel-perfect requirement.

---

# 14. Connection State vs Tuning State

Do not collapse these into one state.

They represent different facts.

Example:

```text
CRD state:          Connected
Optimization state: Active
```

or:

```text
CRD state:          Connected
Optimization state: Paused / Baseline
```

The UI must not imply that optimization is active merely because CRD is connected.

Recommended independent states:

## CRD Connection State

- Unknown
- Disconnected
- Connected

## Automation State

- Enabled
- Paused/Disabled

## Tuning State

- Baseline / Normal
- Applying
- Remote Optimized
- Restoring
- Error / Partial
- Recovery Required

---

# 15. Tray UX

Candidate tray menu:

```text
CRD Autotune
────────────────────────
● CRD Connected
✓ Remote optimizations active
────────────────────────
Open CRD Autotune
Pause Automation
Restore Windows Settings
────────────────────────
Quit
```

Potential additional action:

```text
Force Remote Profile
```

if manual override is implemented.

## 15.1 Tray Semantics

Recommended:

### Open CRD Autotune

Show/focus compact main window.

### Pause Automation

Pause automatic CRD-driven tuning.

Safest default behavior:

- restore the stored baseline first;
- stop applying automatic profiles until resumed.

This prevents a “paused” application from silently owning Windows settings.

### Restore Windows Settings

Restore the stored baseline immediately.

### Quit

Before exiting:

- if CRD Autotune currently owns an active override and a valid snapshot exists, restore it;
- then terminate.

The application should never casually exit while leaving a temporary override orphaned.

---

# 16. Settings Screen

Main window should stay minimal.

Advanced configuration belongs in `Settings`.

Suggested structure:

```text
Remote Profile
├─ Disable client-area animation
├─ Disable window animation
├─ Disable broad UI effects        [advanced]
├─ Disable full-window drag        [optional]
└─ Disable taskbar auto-hide

Behavior
├─ Start with Windows
├─ Start minimized / hidden to tray
├─ Restore settings on exit
├─ Notify on session state change  [optional]
└─ Automatic tuning enabled

Diagnostics
├─ Current CRD detector state
├─ Last CRD event timestamp
├─ Last apply timestamp
├─ Last restore timestamp
├─ Snapshot validity
├─ Last error
└─ Open logs
```

---

# 17. CRD Connection Detection — Chosen Direction

Reliable connection detection is the most important technical dependency.

If connection detection is wrong, all automation becomes unreliable.

## 17.1 Avoid Process-Only Detection

Do **not** use this as the primary detector:

```text
Is remoting_host.exe running?
```

Chrome Remote Desktop host/service infrastructure can exist while no client is actively controlling the machine.

Therefore:

```text
CRD host process exists ≠ CRD client currently connected
```

Process presence may be used for diagnostics, but not as source of truth.

## 17.2 Avoid Network-Heuristic Detection as Primary Source

Do not primarily infer a session from:

- network sockets;
- generic outbound CRD traffic;
- service state;
- CPU activity.

These are weaker proxies than actual connection/disconnection events.

---

# 18. CRD Event Log Research

Chromium's source contains Windows Event Log resource messages for the Chrome Remote Desktop / Chromoting host, including messages semantically equivalent to:

- Client connected
- Client disconnected
- Access denied
- Client routing changed

The source identifies these as messages written to the Event Log by the Chromoting Host.

This provides a strong primary signal for CRD session transitions.

## 18.1 Important Implementation Caveat

Do **not** assume the exact event channel, provider name, event ID, or rendered XML shape solely from this proposal.

The Chromium source confirms that these event messages exist, but implementers must inspect real events on target Windows/CRD versions during development.

Expected direction:

- Windows Event Log;
- Chromoting/Chrome Remote Desktop host-related source/provider;
- connect/disconnect messages.

Verify the exact runtime metadata before committing the production query.

---

# 19. Windows Event Log API

Use Windows Event Log APIs rather than polling the Event Viewer UI or launching PowerShell.

## 19.1 Historical Query

Use:

```text
EvtQuery
```

to query relevant recent CRD events.

Purpose:

- bootstrap state when CRD Autotune starts;
- recover after app restart;
- inspect latest meaningful connection transition;
- diagnostics.

## 19.2 Real-Time Subscription

Use:

```text
EvtSubscribe
```

to receive matching events in real time.

Windows Event Log supports push/pull subscription models and XPath/structured queries.

A push callback model is a natural fit for this utility, but the final Go wrapper can choose whichever model is more reliable to integrate with the application lifecycle.

---

# 20. Critical Startup Edge Case

Example:

```text
13:00  CRD connects
13:05  CRD Autotune starts
```

If CRD Autotune only subscribes to future events at 13:05, it never sees the 13:00 connection event.

Therefore startup must not be:

```text
start app
→ EvtSubscribe(future events only)
```

It must conceptually be:

```text
start app
    ↓
query/reconstruct current CRD state
    ↓
subscribe to future CRD events
    ↓
reconcile Windows tuning state
```

This behavior is a **core requirement**, not a nice-to-have.

---

# 21. Detector Bootstrap Strategy

Candidate flow:

```text
Application startup
        │
        ▼
Load persisted CRD Autotune state
        │
        ▼
Query recent relevant CRD events
        │
        ▼
Reconstruct latest session state
        │
        ▼
Start real-time Event Log subscription
        │
        ▼
Reconcile desired vs actual tuning state
```

Potential production improvement:

- use an Event Log bookmark to resume more deterministically across restarts;
- still have a recovery path if logs rotate or bookmark becomes stale.

---

# 22. Event Handling Robustness

The detector must be designed for imperfect event delivery.

Handle:

- duplicate events;
- app startup after connect;
- app startup after disconnect;
- delayed event callbacks;
- Event Log subscription failure;
- Event Log log rotation;
- invalid/stale bookmark;
- service restart;
- CRD host restart;
- quick connect/disconnect transitions;
- app shutdown during a transition.

Avoid letting one unexpected event directly mutate system state without passing through the coordinator/state machine.

---

# 23. Multiple Client Robustness

Chromium's event messages can include a client identity.

Do not build logic that assumes “any disconnect means zero clients” unless actual CRD behavior has been verified.

If multiple concurrent client sessions are possible in the chosen CRD mode/version, maintain active-session/client state rather than a single blind boolean.

If CRD guarantees a single controlling client for this mode, simplify after verification.

This is a **verify-before-simplifying** item.

---

# 24. Windows Tuning Strategy

The preferred implementation is:

> Native Windows APIs from Go.

Avoid making PowerShell or registry editing the primary mechanism when a supported Win32 API exists.

Benefits:

- fewer shell dependencies;
- easier error handling;
- clear get/set semantics;
- better snapshot/restore behavior;
- less fragile than modifying undocumented registry values and restarting Explorer.

---

# 25. Animation APIs

Windows exposes system-wide visual behavior through:

```text
SystemParametersInfo
```

Relevant candidate parameters include:

## 25.1 Client-Area Animation

```text
SPI_GETCLIENTAREAANIMATION
SPI_SETCLIENTAREAANIMATION
```

Use to read/write client-area animation and related transient effects that respect the Windows setting.

## 25.2 Window Animation

```text
SPI_GETANIMATION
SPI_SETANIMATION
```

This setting uses the Windows animation information structure and affects window-related animation behavior.

## 25.3 Broad UI Effects

```text
SPI_GETUIEFFECTS
SPI_SETUIEFFECTS
```

This is broader and more aggressive.

It can affect multiple UI effect categories.

Recommended treatment:

- expose as an advanced profile option;
- test carefully;
- do not assume it is necessary for the default profile until UX benefit is measured.

## 25.4 Full-Window Drag

```text
SPI_GETDRAGFULLWINDOWS
SPI_SETDRAGFULLWINDOWS
```

When disabled, Windows can avoid continuously drawing the full window contents while a user drags a window.

This logically fits the project because it can reduce large-area visual mutation during remote interaction.

Recommended treatment:

- optional optimization;
- include in production profile only after testing;
- snapshot/restore exactly like every other setting.

---

# 26. Default Remote Profile

The intended concept is:

```text
Remote Optimized
```

not Microsoft's global “Best performance” preset.

Recommended baseline definition:

```text
Remote Optimized
├─ Client-area animation: OFF
├─ Window animation: OFF
├─ Taskbar auto-hide: OFF
├─ Full-window drag: configurable / candidate OFF
└─ Broad UI effects: advanced configurable / candidate OFF
```

Final defaults for `Full-window drag` and `Broad UI effects` should be validated with hands-on CRD testing.

The user should eventually be able to choose which optimizations participate in the profile.

---

# 27. Important Animation Limitation

CRD Autotune cannot guarantee that all applications stop animating.

Some applications:

- render their own animations;
- ignore Windows accessibility/visual-effect preferences;
- render continuously with custom GPU frameworks;
- display video or other high-motion content.

Therefore the product promise should be:

> CRD Autotune reduces selected Windows/system-aware visual effects.

It should **not** claim:

> CRD Autotune disables all animation everywhere.

---

# 28. Applying System Parameter Changes

When using `SystemParametersInfo`, implementers must intentionally choose appropriate flags for persistence/broadcast behavior.

The key product rule is:

- CRD Autotune is applying a temporary user-level override;
- other applications may need to observe the updated setting;
- the user's baseline must remain recoverable.

Do not blindly use flags that make temporary values difficult to restore.

The exact `SPIF_*` behavior should be validated during implementation for each selected parameter.

---

# 29. Taskbar Auto-Hide API

Windows Shell exposes taskbar/appbar state through:

```text
SHAppBarMessage
```

Relevant messages:

```text
ABM_GETSTATE
ABM_SETSTATE
```

`ABM_GETSTATE` reports taskbar state flags including auto-hide.

`ABM_SETSTATE` can change the taskbar's auto-hide/always-on-top state.

## 29.1 Required Behavior

Before changing taskbar state:

```text
read current state
→ store it in snapshot
```

During Remote Optimized mode:

```text
ensure AutoHide bit = disabled
```

On restore:

```text
restore original AutoHide state
```

## 29.2 Preserve Unrelated State

Do not treat `ABM_SETSTATE` as permission to overwrite unrelated appbar bits.

Read-modify-write carefully.

The project only owns the behavior it intentionally changes.

---

# 30. Snapshot Model

Before CRD Autotune applies an automatic Remote Optimized override, capture all settings it may modify.

Conceptual snapshot:

```json
{
  "clientAreaAnimation": true,
  "windowAnimation": true,
  "uiEffects": true,
  "dragFullWindows": true,
  "taskbarAutoHide": true
}
```

Only fields actually managed by the current build need to exist.

The persisted structure should be versioned for forward compatibility.

Example conceptual schema:

```json
{
  "schemaVersion": 1,
  "overrideApplied": true,
  "snapshotCreatedAt": "2026-08-13T16:00:00+07:00",
  "snapshot": {
    "clientAreaAnimation": true,
    "windowAnimation": true,
    "uiEffects": true,
    "dragFullWindows": true,
    "taskbarAutoHide": true
  }
}
```

This is illustrative, not an exact JSON contract.

---

# 31. State Persistence

Candidate path:

```text
%LOCALAPPDATA%\CRDAutotune\
```

Candidate files:

```text
state.json
config.json
logs\
```

Exact naming can be adjusted, but user-specific local application data is appropriate.

Do not require administrator-level shared machine storage unless later requirements demand it.

---

# 32. Transactional Apply

System tuning should be treated like a transaction.

Bad approach:

```text
disable animation
disable taskbar autohide
done
```

Better conceptual approach:

```text
1. Read current settings
2. Validate snapshot
3. Persist snapshot as recovery data
4. Mark transition = Applying
5. Apply each selected optimization
6. Verify values where feasible
7. Mark overrideApplied = true
8. Mark tuning state = RemoteOptimized
```

If an operation fails midway:

- do not pretend the profile is fully active;
- record which changes succeeded;
- attempt rollback or retain enough information for later recovery;
- surface `Partial/Error` state.

---

# 33. Transactional Restore

Conceptual flow:

```text
1. Confirm a valid owned snapshot exists
2. Mark transition = Restoring
3. Restore each managed value
4. Verify where feasible
5. Clear override ownership
6. Retire/delete snapshot only after successful restore
7. Mark tuning state = Baseline
```

Never delete the only recovery snapshot before restoration succeeds.

---

# 34. Crash Recovery

Crash recovery is a core product requirement.

Example:

```text
CRD connected
→ CRD Autotune snapshots state
→ applies Remote Optimized
→ application crashes
→ CRD disconnects while app is dead
→ application starts later
```

At restart:

```text
load persisted state
    ↓
overrideApplied = true
    ↓
detect actual CRD state
    │
    ├─ CRD still connected
    │      → ensure Remote Optimized state
    │
    └─ CRD disconnected
           → restore snapshot
```

The application must not leave Windows permanently optimized for remote use just because the utility crashed.

---

# 35. App Exit Behavior

Recommended default:

```text
Quit CRD Autotune
→ restore owned baseline
→ remove temporary ownership state
→ terminate
```

Closing the main window should **not** equal quitting.

Normal close:

```text
close window
→ hide to tray
→ automation continues
```

Explicit tray `Quit`:

```text
restore if necessary
→ exit
```

---

# 36. Automation State Machine

A coordinator should own all transitions.

Conceptual state machine:

```text
                    CRD CONNECTED
        ┌──────────────────────────────────┐
        │                                  ▼
   BASELINE                           APPLYING
        ▲                                  │
        │                                  ▼
        │                          REMOTE_OPTIMIZED
        │                                  │
        │                                  │ CRD DISCONNECTED
        │                                  ▼
        └──────────────────────────── RESTORING
```

Error paths:

```text
APPLYING   → PARTIAL/ERROR
RESTORING  → RECOVERY_REQUIRED
UNKNOWN    → RECONCILE
```

Manual commands should also go through the coordinator rather than directly calling Windows API wrappers.

---

# 37. Desired-State Reconciliation

The architecture should reason about:

```text
Observed CRD state
+ Automation preference
+ Manual override
+ Persisted ownership state
= Desired tuning state
```

Then reconcile actual Windows settings toward the desired state.

This is more robust than scattering direct calls such as:

```go
onConnect() {
    disableAnimations()
}
```

throughout the application.

---

# 38. Manual Overrides

Useful controls:

## 38.1 Force Remote Profile

Apply the Remote Optimized profile even if CRD detector reports disconnected.

Use cases:

- testing;
- detector troubleshooting;
- user wants temporary optimization manually.

## 38.2 Restore Now

Restore the owned baseline immediately.

## 38.3 Pause Automation

Recommended semantics:

```text
Pause
→ restore baseline if CRD Autotune owns an override
→ ignore automatic CRD transitions until resumed
```

This keeps “paused” intuitive: the app is no longer modifying Windows.

Manual behavior should be explicit in UI so the user always knows whether the active state came from automation or a manual force.

---

# 39. Proposed Backend Architecture

Recommended Go package/domain structure:

```text
CRD Autotune
│
├── app / application
│   └── StateCoordinator
│
├── crd
│   ├── Detector
│   ├── EventLogReader
│   ├── EventSubscriber
│   ├── StateReconstructor
│   └── EventParser
│
├── windows
│   ├── AnimationManager
│   ├── TaskbarManager
│   ├── win32
│   └── SettingSnapshotReader
│
├── profile
│   ├── Profile
│   ├── RemoteOptimized
│   ├── Snapshot
│   └── RestorePlan
│
├── persistence
│   ├── StateStore
│   ├── ConfigStore
│   └── RecoveryStore
│
├── diagnostics
│   ├── Logger
│   └── Health/Status
│
└── ui
    └── Wails services/bindings
```

Exact directory names may change, but separation of concerns should remain.

---

# 40. Core Interfaces

The implementation should be designed around interfaces rather than hard-coded CRD-specific calls in the coordinator.

Conceptually:

```go
type RemoteSessionDetector interface {
    CurrentState(ctx context.Context) (SessionState, error)
    Subscribe(ctx context.Context, events chan<- SessionEvent) error
}
```

```go
type TuningManager interface {
    Snapshot(ctx context.Context) (Snapshot, error)
    Apply(ctx context.Context, profile Profile) error
    Restore(ctx context.Context, snapshot Snapshot) error
}
```

```go
type StateStore interface {
    Load(ctx context.Context) (PersistentState, error)
    Save(ctx context.Context, state PersistentState) error
}
```

These are conceptual contracts, not mandatory exact Go signatures.

---

# 41. Why Keep CRD Detection Abstract

Product scope is CRD-first.

Architecture should leave room for:

```text
RemoteSessionDetector
├─ CRDDetector
├─ RDPDetector        [future]
├─ RustDeskDetector   [future]
└─ AnyDeskDetector    [future]
```

This should **not** cause overengineering now.

Rule:

> Make the detector replaceable, but only implement CRD unless scope is explicitly expanded.

---

# 42. Profile Abstraction

Likewise, tuning can eventually support:

```text
Profile
├─ Baseline/UserState
├─ RemoteOptimized
└─ RemoteAggressive [future]
```

For now:

- `Baseline` is not a fixed preset; it is the captured user state;
- `RemoteOptimized` is the primary temporary override;
- `RemoteAggressive` is optional future scope.

---

# 43. Wails Responsibilities

Wails should primarily own:

- application lifecycle;
- system tray;
- window lifecycle;
- autostart integration;
- Go ↔ Vue bindings/events;
- native desktop shell.

Wails should **not** be where Windows tuning logic lives.

Windows-specific functionality should remain in Go domain/infrastructure packages so it can be unit-tested and reasoned about independently of Vue.

---

# 44. Vue Responsibilities

Vue should primarily own presentation:

- status display;
- configuration forms;
- current connection/tuning state;
- diagnostics;
- manual commands;
- visual feedback.

Vue must not become the source of truth for system state.

Correct:

```text
Go backend owns actual state
→ emits state to Vue
→ Vue renders
```

Avoid:

```text
Vue toggled switch
→ UI assumes setting succeeded
```

Every system-changing command should return authoritative result/state.

---

# 45. Wails v3 System Tray

Wails v3 has first-class system tray support intended for background apps and quick-access utilities.

Use it as a core product feature, not an afterthought.

Required behavior:

- tray icon exists while automation is running;
- single-click/double-click behavior should be consistent;
- tray menu status reflects backend state;
- main window can be shown/focused from tray;
- closing main window keeps app alive;
- explicit quit performs restore logic.

---

# 46. Start with Windows

Wails v3 exposes an Autostart manager.

On Windows it uses the normal per-user startup mechanism.

This matches CRD Autotune because the application should be available before a remote session starts.

Desired option:

```text
☑ Start with Windows
```

Recommended:

- disabled or enabled default is a product decision that can be chosen during polishing;
- user can toggle it without admin privileges where the selected mechanism allows;
- UI should report failures rather than silently assuming startup registration succeeded.

---

# 47. Main Status Model

The frontend should receive a single coherent status model from Go.

Example conceptual payload:

```json
{
  "automationEnabled": true,
  "crdState": "connected",
  "tuningState": "remote_optimized",
  "profileSource": "automatic",
  "snapshotValid": true,
  "lastCrdEventAt": "2026-08-13T16:08:00+07:00",
  "settings": {
    "clientAreaAnimation": false,
    "windowAnimation": false,
    "taskbarAutoHide": false,
    "dragFullWindows": false
  },
  "lastError": null
}
```

Again, schema is conceptual.

---

# 48. Diagnostics

Diagnostics are important because connection detection depends on external system events.

The application should provide enough visibility to answer:

- Does CRD Autotune see CRD?
- What was the last event?
- Is the detector subscribed?
- What did it change?
- Did snapshot creation succeed?
- Did restore succeed?
- Why is optimization not active?

Suggested diagnostics fields:

```text
Detector status
CRD state
Last relevant Event Log timestamp
Last transition
Current profile
Snapshot state
Last apply result
Last restore result
Autostart registration state
Application version
Wails version
Windows version
CRD host version [if safely discoverable]
```

---

# 49. Logging

Logging should be lightweight and bounded.

Recommended levels:

- INFO — meaningful state transitions;
- WARN — recoverable inconsistencies;
- ERROR — failed system operations;
- DEBUG — raw diagnostic details when explicitly enabled.

Avoid spam such as logging every periodic reconciliation tick.

## 49.1 Privacy

CRD Event Log messages may contain a client account/email identity.

CRD Autotune does not need that identity for its core behavior.

Therefore:

- do not persist raw client identity by default;
- parse only what is needed;
- redact client emails/usernames from normal logs;
- diagnostic export should make any sensitive fields obvious.

---

# 50. Notifications

Optional.

Potential notifications:

```text
CRD connected — Remote Optimized profile applied
CRD disconnected — Windows settings restored
Recovery completed
Optimization failed
```

Avoid notification spam.

The product should remain quiet by default unless the user opts into notifications or an error requires attention.

---

# 51. Error Handling Principles

Never silently lie about state.

Examples:

If animation was disabled successfully but taskbar update failed:

```text
Remote Optimized: Partial
Taskbar Auto-hide: Failed
```

not:

```text
Remote Optimized: Active
```

If restore fails:

- keep recovery snapshot;
- surface error;
- retry where safe;
- provide `Restore Now`;
- do not clear ownership metadata until recovery is complete.

---

# 52. Privilege Model

The preferred design is per-user and should avoid requiring Administrator privileges for normal operation.

Implementation agents must verify:

- ability to read the relevant Event Log channel/provider as the normal user;
- ability to call chosen system parameter APIs;
- ability to change taskbar state;
- autostart behavior.

If a particular Event Log configuration requires elevation on some systems, this should be treated as a product compatibility issue and not silently solved by making the entire application permanently elevated.

---

# 53. Windows Explorer / Taskbar Edge Cases

Taskbar behavior depends on Windows Shell/Explorer.

Test:

- Explorer restart while CRD is connected;
- Explorer restart while baseline is active;
- Windows 10;
- Windows 11;
- multiple monitors;
- secondary taskbar behavior;
- taskbar alignment/layout differences;
- auto-hide already off before CRD connects.

The correct invariant is always:

> Restore the original user state, not a guessed default.

---

# 54. Multi-Monitor Considerations

The original pain point focuses on the bottom-edge taskbar conflict.

If the Controlled machine has multiple monitors:

- CRD may stream one or multiple displays;
- Windows may expose taskbar behavior differently across displays;
- `ABM_GETSTATE` is fundamentally about taskbar/appbar state, not a bespoke per-monitor CRD concept.

Initial implementation should support the normal system auto-hide setting correctly and document any secondary-taskbar limitations found during testing.

Do not invent unsupported per-monitor state if Windows does not expose it through the chosen API.

---

# 55. Race Conditions

Potential race:

```text
CRD connect event
→ Apply begins
→ CRD disconnect event arrives
→ Apply still running
```

The coordinator must serialize transitions.

Recommended:

- single transition worker/event loop;
- mutex/transaction guard where appropriate;
- deduplicate obsolete desired states;
- always converge toward latest desired state.

Avoid parallel calls that can interleave taskbar/animation writes unpredictably.

---

# 56. Debouncing and Reconciliation

CRD events may be duplicated or occur quickly.

Recommended approach:

- state transitions should be idempotent;
- applying Remote Optimized when already correctly optimized should be safe;
- restoring when already restored should not damage state;
- optional short debounce can suppress duplicate event noise;
- do not delay legitimate disconnect recovery unnecessarily.

---

# 57. Config Model

Candidate user configuration:

```json
{
  "automationEnabled": true,
  "startWithWindows": true,
  "startHidden": true,
  "notificationsEnabled": false,
  "remoteProfile": {
    "clientAreaAnimation": false,
    "windowAnimation": false,
    "uiEffects": null,
    "dragFullWindows": false,
    "taskbarAutoHide": false
  }
}
```

Semantics:

- values in profile represent the desired remote state;
- `null` can conceptually mean “do not manage this setting”;
- snapshot stores baseline separately.

Exact schema may differ.

---

# 58. Ownership Model

CRD Autotune needs to know whether a setting is being changed because of the app or because of the user.

Minimum ownership rule:

```text
No snapshot
→ CRD Autotune must not guess what to restore
```

If the app discovers Windows is in a performance-like state but has no persisted snapshot, it cannot safely conclude that it caused the state.

This is crucial after:

- state file deletion;
- manual system changes;
- an old app version;
- corrupted persistence.

Fail safe rather than “restore” guessed values.

---

# 59. What If User Changes Windows Settings While Remote?

Example:

```text
CRD connects
→ app snapshots Animation=true
→ app disables animation
→ user manually enables animation during remote session
```

This creates an ownership conflict.

Recommended initial policy:

- CRD Autotune's active profile is authoritative while automation is active;
- if a user manually changes a managed value, either:
  - reapply on next reconciliation; or
  - show drift and let the user choose.

For the first production version, simplest behavior is:

> Do not continuously fight the user every few milliseconds. Reconcile on meaningful events/startup and expose drift in diagnostics if needed.

This can be refined after real usage.

---

# 60. Source of Truth for Baseline

Baseline comes from a snapshot taken immediately before the first transition from unmanaged/baseline into an owned Remote Optimized session.

Do not replace the snapshot every time a duplicate `Connected` event arrives.

Bad:

```text
Connected event #1 → snapshot baseline
apply optimized
Connected event #2 → snapshot optimized state
```

This would lose the real baseline.

Correct:

```text
if override not already owned:
    create baseline snapshot
else:
    reuse existing snapshot
```

---

# 61. Session Boundary Rule

A new snapshot is created only when CRD Autotune begins a new owned override cycle.

Conceptually:

```text
Baseline
→ Connected
→ snapshot A
→ Remote Optimized
→ Disconnected
→ restore A
→ clear A

Baseline
→ Connected again
→ snapshot B
...
```

---

# 62. Potential Event Log Bookmark

Windows Event Log supports bookmarks.

Candidate future/production-hardening strategy:

```text
Persist Event Log bookmark
→ app restarts
→ resume subscription after bookmark
```

Benefits:

- more deterministic catch-up;
- less need to query a large historical range.

Still retain recovery logic for:

- stale bookmarks;
- cleared logs;
- log rollover.

A bookmark is an implementation enhancement, not a replacement for the application's own tuning-state persistence.

---

# 63. Testing Strategy

Testing must be split into pure logic and Windows integration.

## 63.1 Unit Tests

Unit-test:

- coordinator state transitions;
- duplicate events;
- startup reconciliation;
- snapshot ownership rules;
- config parsing;
- partial failure handling;
- manual override precedence;
- crash recovery decisions.

Mock:

- detector;
- Windows settings manager;
- state store.

## 63.2 Integration Tests

On real Windows:

- `SystemParametersInfo` get/set pairs;
- taskbar `SHAppBarMessage`;
- Event Log query;
- Event Log subscription;
- autostart;
- Wails tray/window lifecycle.

## 63.3 End-to-End Tests

Real CRD session:

```text
Controlled machine
↕ Chrome Remote Desktop
Controlling machine
```

Measure/observe:

- connect detection latency;
- apply latency;
- disconnect detection latency;
- restore latency;
- taskbar conflict resolved;
- animation behavior reduced;
- no leftover state.

---

# 64. Critical Test Scenarios

At minimum test:

### Scenario A — Normal Connect/Disconnect

```text
Baseline
→ connect
→ optimized
→ disconnect
→ exact baseline
```

### Scenario B — App Starts While Already Connected

```text
CRD connected
→ start CRD Autotune
→ reconstruct connected state
→ optimize
```

### Scenario C — App Starts After Session End but Override Was Left

```text
override persisted
→ CRD disconnected
→ restart app
→ restore snapshot
```

### Scenario D — App Crash While Connected

```text
connected
→ optimized
→ crash
→ restart while still connected
→ remain/reconcile optimized
```

### Scenario E — App Crash + CRD Disconnect Before Restart

```text
connected
→ optimized
→ crash
→ CRD disconnect
→ restart
→ restore baseline
```

### Scenario F — Duplicate Connected Event

Must not overwrite baseline snapshot.

### Scenario G — Duplicate Disconnected Event

Restore must be idempotent.

### Scenario H — Taskbar Was Already Non-Auto-Hide

After disconnect it must remain non-auto-hide.

### Scenario I — Animations Were Already Disabled

After disconnect they must remain disabled.

### Scenario J — Partial Apply Failure

UI must report partial/error and preserve recovery data.

### Scenario K — Restore Failure

Snapshot remains available; app surfaces recovery action.

### Scenario L — Close Main Window

Automation remains active in tray.

### Scenario M — Explicit Quit While Optimized

Restore occurs before process exit.

### Scenario N — Start with Windows

After login the app starts without requiring the main window to stay open.

---

# 65. Performance Requirements

CRD Autotune should itself be lightweight.

Goals:

- negligible idle CPU;
- low memory footprint;
- event-driven detection rather than aggressive polling;
- no high-frequency timer unless required for recovery;
- no continuous UI rendering when main window is hidden;
- no heavyweight browser runtime bundled like Electron.

Do not sacrifice correctness solely to chase tiny memory numbers, but utility-like resource use is part of the product identity.

---

# 66. Reliability Requirements

Higher priority than visual polish:

1. never lose the user's baseline if avoidable;
2. never overwrite the baseline with the optimized state;
3. never report a successful optimization when operations failed;
4. recover from restart/crash;
5. serialize transitions;
6. safely handle unknown detector state;
7. avoid requiring the UI window for background operation.

---

# 67. UX Requirements

The app should feel immediate and understandable.

Required:

- visible CRD state;
- visible tuning state;
- visible automation enabled/paused state;
- one-click restore;
- clear error state;
- no unnecessary navigation;
- tray remains useful without opening the window;
- close-to-tray behavior is obvious;
- advanced profile configuration does not clutter the main screen.

---

# 68. Accessibility / Visual Style

Use a Windows-native visual language.

Recommended:

- system theme awareness;
- readable compact typography;
- clear focus states;
- keyboard navigability for settings;
- do not communicate state using color alone;
- status text/icon must accompany color.

Example:

```text
● Connected
```

not an unexplained green dot alone.

---

# 69. Product Naming

Working/final name currently used:

```text
CRD Autotune
```

Meaning:

- CRD = Chrome Remote Desktop;
- Autotune = automatically tuning Windows behavior for the remote session.

No renaming requirement exists in the current scope.

---

# 70. Possible Future Scope

Do not implement unless core CRD behavior is stable.

Potential expansions:

## 70.1 Other Remote Providers

- Microsoft RDP
- RustDesk
- AnyDesk
- other remote desktop tools

## 70.2 More Remote Profiles

- Balanced
- Remote Optimized
- Remote Aggressive

## 70.3 Additional Visual Optimizations

Only after measurement:

- transparency effects;
- wallpaper handling;
- cursor effects;
- other SystemParametersInfo-managed visual features.

Do not turn the product into a generic tweaker without a clear remote-desktop benefit.

---

# 71. Implementation Phases

This is an implementation order, not a statement that the final product must ship as a half-featured MVP.

## Phase 0 — Technical Spike

Verify on the actual Controlled machine:

- exact CRD Event Log channel;
- provider/source;
- event IDs/XML;
- client connect event;
- client disconnect event;
- whether normal user can query/subscribe;
- selected `SystemParametersInfo` get/set behavior;
- taskbar `SHAppBarMessage` behavior;
- whether settings immediately affect the desktop;
- restore correctness.

Deliverable:

```text
docs/research/windows-crd-events.md
```

or equivalent internal notes.

Do not build the full UI before this spike succeeds.

## Phase 1 — Core Windows Tuning Engine

Implement:

- animation getters/setters;
- taskbar getter/setter;
- snapshot;
- apply;
- restore;
- tests.

No CRD automation required yet.

## Phase 2 — CRD Detector

Implement:

- historical query/bootstrap;
- realtime subscription;
- event parser;
- state reconstruction;
- detector health.

## Phase 3 — State Coordinator

Implement:

- state machine;
- snapshot ownership;
- automatic apply;
- automatic restore;
- restart recovery;
- serialized transitions.

At this point the core product exists even without polished UI.

## Phase 4 — Wails Tray Shell

Implement:

- Wails v3 app lifecycle;
- tray icon/menu;
- background operation;
- close-to-tray;
- explicit quit;
- autostart.

## Phase 5 — Compact Vue UI

Implement:

- main status surface;
- profile controls;
- settings;
- diagnostics;
- manual restore/force actions.

Use G-Helper as a direction reference, not as a clone.

## Phase 6 — Hardening

Test:

- crashes;
- Event Log failures;
- Explorer restart;
- repeated sessions;
- long-running background operation;
- multiple monitors;
- Windows 10/11;
- startup/login;
- portable executable movement;
- upgrades/migrations.

---

# 72. Production Acceptance Criteria

The application is considered functionally ready when all of the following are true.

## Detection

- [ ] Detects a normal CRD connect without manual refresh.
- [ ] Detects a normal CRD disconnect without manual refresh.
- [ ] Correctly determines state when app starts during an already-active CRD session.
- [ ] Does not use `remoting_host.exe` process presence as its sole source of truth.
- [ ] Detector failure is surfaced to diagnostics.

## Snapshot / Restore

- [ ] Captures original values before first owned override.
- [ ] Duplicate connect events cannot replace the baseline snapshot.
- [ ] Disconnect restores the original values exactly.
- [ ] Snapshot survives process restart while override is active.
- [ ] Snapshot is not deleted before successful restore.

## Animation

- [ ] Selected Windows animation settings can be read.
- [ ] Selected settings can be disabled for Remote Optimized.
- [ ] Their exact original values are restored.
- [ ] Unsupported/app-specific animation is not falsely presented as controlled.

## Taskbar

- [ ] Auto-hide is disabled while Remote Optimized is active.
- [ ] User's original taskbar auto-hide state is restored.
- [ ] Already-disabled auto-hide remains disabled after session end.
- [ ] Failure to change taskbar state is reported.

## Lifecycle

- [ ] Main window can close while utility remains active.
- [ ] Tray icon remains functional.
- [ ] Explicit Quit restores owned settings before exiting.
- [ ] Start with Windows works or reports a clear failure.
- [ ] Restart after crash can recover/reconcile state.

## UI

- [ ] Main window clearly distinguishes CRD state from tuning state.
- [ ] User can restore baseline manually.
- [ ] User can enable/disable automation.
- [ ] Advanced settings do not overload the main view.
- [ ] UI matches compact Windows utility direction.

---

# 73. Recommended Definition of Done for Core

Core backend is not “done” merely because settings can be toggled.

Core is done only when this invariant is demonstrably true:

> For every CRD Autotune-owned Remote Optimized transition, there is a durable path back to the exact pre-transition user state.

---

# 74. Security Considerations

CRD Autotune is local software with system-setting access.

Principles:

- no network server required;
- no open localhost port required for product functionality;
- do not collect CRD credentials;
- do not store client login identity;
- do not require broad administrator rights unless proven necessary;
- validate any config loaded from disk;
- do not execute shell strings derived from Event Log messages;
- do not expose dangerous generic registry/system-setting commands through the frontend.

Vue should invoke narrow, typed backend operations.

---

# 75. Telemetry

No telemetry requirement has been specified.

Default recommendation for a personal lightweight utility:

```text
No remote telemetry by default.
```

Local diagnostics/logging are sufficient unless a future distribution/support need justifies opt-in telemetry.

---

# 76. Upgrade / Schema Migration

Persisted state is safety-critical.

Therefore:

- version state schema;
- version config schema if needed;
- on incompatible state, preserve recovery data rather than deleting it blindly;
- migration must distinguish:
  - normal preferences;
  - active recovery snapshot.

An upgrade must not strand a machine in Remote Optimized state.

---

# 77. Development Notes for Go/Win32

Preferred style:

- wrap Win32 calls behind narrow Go functions;
- map Windows errors into meaningful Go errors;
- keep raw handles scoped and closed;
- isolate unsafe code;
- avoid leaking Event Log handles/subscriptions;
- cleanly cancel subscription on shutdown;
- make backend services context-aware;
- serialize system mutations.

Implementation may use appropriate Go Windows bindings or direct DLL procedure wrappers, but the public domain layer should not depend on raw syscall details.

---

# 78. Suggested Repository Shape

Example:

```text
crd-autotune/
├── cmd/
│   └── crd-autotune/
├── internal/
│   ├── application/
│   ├── crd/
│   ├── windows/
│   ├── profile/
│   ├── persistence/
│   └── diagnostics/
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   ├── stores/
│   │   └── types/
│   └── ...
├── build/
├── docs/
│   └── research/
├── tests/
├── PROPOSAL.md
└── README.md
```

Follow actual Wails v3 generated structure where it differs; preserve domain separation rather than forcing this tree literally.

---

# 79. Suggested Main Vue Components

Possible component structure:

```text
AppShell
├─ HeaderStatus
├─ CrdStatusCard
├─ RemoteProfileCard
│  ├─ ProfileState
│  └─ ManagedSettingStatus
├─ AutomationCard
├─ LastSessionInfo
└─ QuickActions

SettingsView
├─ RemoteProfileSettings
├─ BehaviorSettings
└─ DiagnosticsPanel
```

Do not create unnecessary component fragmentation for tiny labels.

---

# 80. Suggested Backend Events to Frontend

Conceptual events:

```text
app:status-changed
crd:state-changed
tuning:state-changed
tuning:error
recovery:required
```

Prefer a cohesive `status-changed` payload if many micro-events make synchronization harder.

Frontend should be able to request a full state snapshot at any time.

---

# 81. Startup Sequence

Recommended production startup:

```text
1. Create logger
2. Load config
3. Load persistent recovery state
4. Initialize Windows managers
5. Initialize CRD detector
6. Bootstrap CRD session state
7. Start Event Log subscription
8. Reconcile persisted override vs current CRD state
9. Start Wails/tray surface
10. Emit authoritative status to UI
```

Ordering can be adapted to Wails lifecycle, but recovery must not depend on the main window being visible.

---

# 82. Shutdown Sequence

Recommended:

```text
1. Stop accepting new UI commands
2. Stop/serialize transition processing
3. Determine whether CRD Autotune owns an override
4. Restore baseline on explicit Quit
5. Persist final state
6. Stop Event Log subscription
7. Release Windows handles/resources
8. Remove tray icon
9. Exit
```

OS forced termination cannot guarantee graceful restore, which is why persisted crash recovery exists.

---

# 83. Connection Transition Sequence

## On Connected

```text
CRD event
→ Detector emits Connected
→ Coordinator validates automation enabled
→ If no existing owned override:
     read baseline
     persist snapshot
→ Apply selected Remote Optimized settings
→ Verify
→ Persist active ownership
→ Update tray/UI
```

## On Disconnected

```text
CRD event
→ Detector emits Disconnected
→ Coordinator checks owned override
→ Restore snapshot
→ Verify
→ Clear ownership only on success
→ Update tray/UI
```

---

# 84. Unknown State Policy

If CRD state is unknown because Event Log detection failed:

Do not aggressively change Windows based on a guess.

Recommended behavior:

```text
Detector = Unknown
→ preserve current owned state carefully
→ surface warning
→ attempt recovery/re-query
```

If an active owned override exists but CRD state cannot be determined, bias toward recoverability and user visibility rather than silently discarding the snapshot.

Exact timeout/fallback policy can be tuned during testing.

---

# 85. Fallback Strategy

Primary detector:

```text
Windows Event Log
```

Fallbacks such as:

- process presence;
- service state;
- socket inspection;

may be used only as:

- diagnostics;
- hints;
- compatibility experiments.

Do not silently downgrade to an unreliable heuristic and present it as certain CRD connection state.

If Event Log integration is unavailable on a system:

- automatic mode may be disabled;
- manual `Force Remote Profile` and `Restore Now` can remain useful;
- diagnostics should explain why automation is unavailable.

---

# 86. Measuring Whether the Product Helps

The project began from perceived UX lag/stutter.

After implementation, validate empirically.

Simple observation tests:

- drag windows with/without full-window drawing;
- open/close menus;
- switch windows;
- interact with taskbar;
- compare cursor/taskbar edge interaction;
- compare animation-heavy OS transitions.

Potential technical metrics if desired later:

- CRD perceived frame smoothness;
- bandwidth estimates;
- CPU encode/capture load;
- remote interaction latency.

Do not block the first production version on building a telemetry/benchmark suite.

---

# 87. User-Facing Terminology

Prefer:

```text
Remote Optimized
Normal / Restored
CRD Connected
CRD Disconnected
Automation
Restore
```

Avoid exposing technical Win32 flag names in the primary UI.

Advanced diagnostics can show technical names.

Do not call the restore state `Best Appearance` because it may not actually be the Windows Best Appearance preset.

---

# 88. Core Product Copy

Candidate one-sentence product definition:

> **CRD Autotune is a lightweight Windows background utility that detects active Chrome Remote Desktop sessions and temporarily optimizes the host's visual behavior for remote use, then safely restores the user's original desktop configuration when the session ends.**

This definition accurately reflects the approved direction.

---

# 89. Key Decisions Already Made

The following decisions are considered approved unless the project owner changes them later.

1. **Product name:** `CRD Autotune`.
2. **Runs on:** Controlled machine.
3. **Current real setup:** Controlling machine = work machine; Controlled machine = home machine.
4. **Primary pain point:** visual animation makes remote UX worse on constrained network conditions.
5. **Second pain point:** auto-hide taskbar on both local and remote contexts creates bottom-edge conflict.
6. **Connected behavior:** temporarily optimize Windows visual behavior.
7. **Connected taskbar behavior:** disable auto-hide.
8. **Disconnected behavior:** restore previous state.
9. **Do not force “Best appearance” as restore behavior.**
10. **Snapshot/restore is mandatory.**
11. **Desktop form factor:** tray-first background utility.
12. **UI inspiration:** G-Helper.
13. **UI should be compact, not a dashboard.**
14. **Architecture should be CRD-specific at product level but reasonably generic internally.**
15. **Primary CRD detector direction:** Windows Event Log.
16. **Do not use CRD process presence as connection source of truth.**
17. **Startup must reconstruct existing session state before relying on future events.**
18. **Windows tuning should prefer native Win32 APIs.**
19. **Animation family API:** `SystemParametersInfo`.
20. **Taskbar API:** `SHAppBarMessage`.
21. **Framework:** Wails v3 + Vue + Go.
22. **Distribution direction:** lightweight portable executable.
23. **WebView2 runtime dependency must be acknowledged.**
24. **Tray and autostart are first-class product features.**
25. **Crash recovery is mandatory.**
26. **Main UI is a control plane; backend is the actual product.**

---

# 90. TBD / Verify Before Hard-Coding

These remain open implementation research items.

## CRD Event Log

- exact Windows Event Log channel;
- exact provider/source name on current CRD version;
- exact event IDs;
- exact XML fields;
- connect event structure;
- disconnect event structure;
- multi-client behavior;
- permissions for non-admin users;
- behavior when CRD host/service restarts.

## Windows Tuning

- exact best set of default animation flags;
- whether `SPI_SETUIEFFECTS` should be default or advanced;
- whether disabling full-window drag creates enough value to enable by default;
- propagation flags for each SystemParametersInfo write;
- behavior across Windows 10 and Windows 11 builds.

## Taskbar

- Explorer restart behavior;
- multiple monitor behavior;
- secondary taskbar behavior;
- any Windows 11-specific quirks.

## Wails

- pin exact beta version;
- test tray behavior;
- test hide/show lifecycle;
- test WebView2 availability;
- test autostart path behavior for portable executable.

---

# 91. Technical Research References

These references support the technical decisions in this proposal.

## Chromium / Chrome Remote Desktop

### CRD / Chromoting Event Log Messages

Chromium resource definitions include Windows Event Log messages for connect/disconnect events generated by the Chromoting host.

Source:

https://chromium.googlesource.com/chromium/src/%2B/71810ed2d9be11b585b01f93fd1115bbfe52f7aa/remoting/resources/remoting_strings.grd

### Windows CRD Host Service

Chromium source describing the Windows service controlling Me2Me host processes:

https://chromium.googlesource.com/chromium/src.git/%2B/96e2c5522647935d4be7179c28b3f2359cdf3880/remoting/host/host_service_win.cc

---

## Microsoft — Windows Event Log

### Subscribing to Events / EvtSubscribe

https://learn.microsoft.com/en-us/windows/win32/wes/subscribing-to-events

### Querying for Events / EvtQuery

https://learn.microsoft.com/en-us/windows/win32/wes/querying-for-events

### Windows Event Log Functions

https://learn.microsoft.com/en-us/windows/win32/wes/windows-event-log-functions

### Bookmarking Events

https://learn.microsoft.com/en-us/windows/win32/wes/bookmarking-events

### Consuming Events / XPath Queries

https://learn.microsoft.com/en-us/windows/win32/wes/consuming-events

---

## Microsoft — Windows Visual Settings

### SystemParametersInfo

https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfoa

### Client Area Animation

https://learn.microsoft.com/en-us/windows/win32/winauto/client-area-animation

---

## Microsoft — Taskbar / AppBar

### SHAppBarMessage

https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shappbarmessage

### ABM_GETSTATE

https://learn.microsoft.com/en-us/windows/win32/shell/abm-getstate

### ABM_SETSTATE

https://learn.microsoft.com/en-us/windows/win32/shell/abm-setstate

---

## Wails v3

### Wails v3 Homepage / Current Beta Status

https://v3.wails.io/

### Wails v3 Beta Announcement

https://v3.wails.io/blog/tags/beta/

### Wails v3 Roadmap / Beta Compatibility

https://v3.wails.io/status/

### Wails Architecture

https://v3.wails.io/concepts/architecture/

### System Tray

https://v3.wails.io/features/menus/systray/

### Manager API / Autostart

https://v3.wails.io/concepts/manager-api/

### Wails Installation / WebView2

https://v3.wails.io/quick-start/installation/

### What's New in Wails v3

https://v3.wails.io/whats-new/

---

# 92. Final Architectural Summary

The intended architecture can be summarized as:

```text
Chrome Remote Desktop
        │
        │ connect/disconnect events
        ▼
Windows Event Log
        │
        ├─ EvtQuery      → startup/recovery
        └─ EvtSubscribe  → realtime
        │
        ▼
CRD Detector
        │
        ▼
State Coordinator
        │
        ├───────────────┐
        ▼               ▼
Snapshot Store     Remote Profile
        │               │
        │               ▼
        │        Windows Tuning Engine
        │          ├─ SystemParametersInfo
        │          └─ SHAppBarMessage
        │               │
        └───────────────┘
                │
                ▼
          Authoritative State
                │
        ┌───────┴────────┐
        ▼                ▼
   Wails Tray         Vue UI
```

And the core runtime behavior is:

```text
CRD DISCONNECTED
     │
     │ user baseline
     ▼
[ Normal / Restored ]
     │
     │ CRD connects
     ▼
snapshot baseline
     │
     ▼
apply Remote Optimized
     │
     ▼
[ CRD CONNECTED ]
     │
     │ CRD disconnects
     ▼
restore exact snapshot
     │
     ▼
[ Normal / Restored ]
```

---

# 93. Final Product Principle

If implementation decisions conflict, prioritize them in this order:

```text
1. Preserve the user's original Windows state
2. Correctly detect/reconcile CRD session state
3. Reliably apply/restore remote optimizations
4. Run quietly and efficiently in the background
5. Make state observable and recoverable
6. Provide compact, polished tray-first UX
7. Add additional optimizations/features
```

The app should never sacrifice recoverability merely to appear automatic.

> **CRD Autotune is successful when it improves the remote session automatically and disappears from the user's attention without ever taking ownership of their Windows preferences permanently.**
