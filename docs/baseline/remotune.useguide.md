---
baseline_schema: "2.0"
pack: "remotune"
document: "useguide"
status: "active"
updated: "2026-08-24"
code_ref: "42fd37dc2a9a083c070c4bfe3547d4dfd190262b"
---

# Remotune Use and Behavior Contract

**[IMPLEMENTED]** The latest portable artifact is `out/remotune-v0.1.8.exe` (12,096,000 bytes), built locally on 2026-08-21 after the host-native verification gate passed. **[VERIFIED]** Its Start with Windows behavior was observed after a target-machine sign-in. On that target-machine run, Custom is opened explicitly from the fixed `Custom visual effects` section rather than automatically from its radio selection.

**[IMPLEMENTED]** Developers run `wails3 dev` for the native Windows loop, `wails3 task verify` for native verification, and `wails3 task windows:portable` for portable Windows packaging. The Wails dev graph provides frontend Vite hot reload and automatic relaunch after Go changes. The Bun lockfile is required: verification and packaging use `bun install --frozen-lockfile` so `@wailsio/runtime` remains exactly aligned with the pinned Wails release.

> **Availability:** **[IMPLEMENTED]** A versioned portable executable is produced under `out/`; the latest build is `out/remotune-v0.1.8.exe`. **[VERIFIED]** The target machine confirmed startup after sign-in and the owned-snapshot Apply → Pause/restore → Resume/reapply → Explicit Quit/restore sequence. Custom editing remains explicit: select Custom, then use `Edit effects` in the fixed `Custom visual effects` section to show the editor.

## What Remotune will do

Remotune is a lightweight Windows tray utility installed on the computer being controlled through Chrome Remote Desktop.

With automatic tuning enabled:

```text
Before CRD
└─ Windows remains in the user's chosen state

CRD connects
├─ Remotune saves the affected Visual Effects state
├─ Remotune saves taskbar auto-hide state
├─ Windows Visual Effects switch to the selected CRD-on profile
└─ Controlled-machine taskbar auto-hide switches OFF

CRD disconnects
├─ Visual Effects follow the selected CRD-off action
└─ taskbar auto-hide returns to the exact saved state
```

**[IMPLEMENTED]** The current build applies the selected CRD-on profile and restores or deliberately replaces the exact saved state according to the selected CRD-off action. The UI provides CRD-on choices for Let Windows choose, Best Appearance, Best Performance, and Remotune Custom; CRD-off choices for Revert to snapshot, Let Windows choose, Best Appearance, and Best Performance. Remotune does not control CRD encoding, transport, or network behavior. The profile surface has local build/test evidence; target-machine behavior for every selection remains to be observed.

## Install and first run

**[IMPLEMENTED]** The primary package is a versioned portable `remotune-v<version>.exe`. WebView2 is a required Windows runtime dependency because the UI uses Wails. If WebView2 is unavailable, Remotune must explain the missing prerequisite rather than fail silently. **[VERIFIED]** Presence can be detected without elevation by reading the WebView2 EdgeUpdate client key; the absent-runtime failure path itself is **[UNVERIFIED]** because the evidence machine has the runtime installed.

**[VERIFIED]** Normal operation does not require Administrator for the integrations exercised so far: reading CRD events, subscribing to them in real time, reading Visual Effects state, and reading and writing taskbar auto-hide all succeeded as a standard non-elevated user. **[PLANNED]** Any target-system permission incompatibility must still be reported deliberately.

**[VERIFIED]** Enabling `Start with Windows` registers the executable's current path and launched successfully after target-machine sign-in. Moving or deleting the portable executable can still break that registration; this moved-path case remains to be handled or documented clearly.

## Normal operation

1. Place and run `Remotune.exe` on the Controlled Windows machine.
2. Enable Automatic tuning and, if offered, the desired Visual Effects and taskbar categories.
3. Optionally enable Start with Windows.
4. Close the window to hide it to the tray; this does not terminate automation.
5. Use CRD normally. Remotune detects session transitions and reports authoritative status.

The intended interaction model is configure once and leave in the tray.

## Status model

The UI must not collapse different concepts into one ambiguous light. It reports:

| Dimension | Values | Meaning |
|---|---|---|
| CRD | Unknown / Disconnected / Connected | Detector's observed remote-session state |
| Automation | Enabled / Paused | Whether automatic reactions are allowed |
| Tuning | Baseline / Applying / Active / Restoring / Partial/Error / Recovery Required | Ownership and transition state |
| Detector | Starting / Healthy / Degraded | Event Log subscription health; last transition metadata is redacted |

Examples:

- `CRD: Connected`, `Automation: Enabled`, `Tuning: Active` means the enabled remote overrides are owned and active.
- `CRD: Connected`, `Automation: Paused`, `Tuning: Baseline` is valid; a CRD session exists but Remotune is not tuning it.
- `Tuning: Partial/Error` means one or more operations failed and must not be presented as full success.
- `Recovery Required` means durable state exists or state is uncertain and operator attention may be needed.

## Main window

**[PLANNED]** The window is a compact 380–450 px-wide vertical control surface with Windows-native styling and OS theme following where practical. It has no left navigation sidebar unless later evidence demonstrates a real need.

Conceptual—not pixel-perfect—layout:

```text
┌──────────────────────────────────────┐
│ Remotune                       ● ON  │
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

The surface may display:

- Remotune on/off and CRD connection status;
- overall automatic tuning;
- optional Visual Effects and taskbar automation category toggles;
- current authoritative Visual Effects/taskbar outcome;
- Start with Windows;
- Restore Now and Settings/diagnostics.

**[VERIFIED]** Settings display the Visual Effects radios. On the accepted target-machine run, selecting `CRD ON → Custom` leaves the separate Remotune-owned editor closed; use the fixed `Custom visual effects` row's `Edit effects` action to open it beside the main popup, preferentially on its left, while the main popup remains open. The editor provides a local Custom checklist draft: freely select all desired effects, select `Apply changes` once to save and apply the entire profile, or select `Revert` to discard those unsaved choices. It does not open the native Windows Performance Options dialog or include Advanced/Data Execution Prevention settings, Minimal/Recommended/Aggressive profiles, or a dashboard. The current source contains an automatic-open attempt after the Custom selection, so that source/runtime difference remains an implementation follow-up rather than a user-facing requirement.

## Tray behavior

Candidate tray surface:

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

Exact labels may change, but useful status, Pause/Resume, Restore, Open, and explicit Quit behavior must remain understandable.

## Command contracts

### Close window

**[PLANNED]** Hides the window and leaves background detection/automation running in the tray.

### Pause Automation

**[PLANNED]** Restores any currently owned temporary Windows state, then stops reacting automatically to CRD transitions. It must not leave the system silently tuned after the user pauses.

### Resume Automation

**[PLANNED]** Re-enables automatic reconciliation. If CRD is already connected, the coordinator evaluates current state and applies enabled categories safely without replacing an existing baseline.

### Restore Now / Restore Windows Settings

**[PLANNED]** If a valid Remotune-owned snapshot exists, retries exact restoration. If none exists, it reports that no restorable Remotune snapshot exists and does not guess.

### Quit

**[PLANNED]** Stops accepting automatic transitions, completes serialized recovery work, restores any owned state, persists clean state, closes subscriptions/resources, and exits. Quit is different from closing the window.

### Start with Windows

**[PLANNED]** Enables/disables startup registration and reports failure clearly. Automatic startup is important because Remotune must be running or able to reconstruct state when a CRD session is already active.

## Baseline preservation examples

### User starts with taskbar auto-hide ON

```text
Before:       ON
CRD active:   OFF
After:        ON
```

### User starts with taskbar auto-hide OFF

```text
Before:       OFF
CRD active:   OFF
After:        OFF
```

### User starts with custom Visual Effects

```text
Before:       arbitrary Custom checkbox combination
CRD active:   Windows Best Performance behavior
After:        exact original Custom values
```

Restoring merely to Best Appearance or merely selecting the Custom radio option is incorrect if it loses original checkbox values.

## Recovery behavior

**[PLANNED]** Remotune persists recovery data before changing Windows.

- Crash while CRD remains connected: after restart, reconstruct `Connected`, retain the original snapshot, and reconcile active tuning.
- Crash followed by CRD disconnect: after restart, reconstruct `Disconnected` and restore the persisted snapshot.
- Duplicate connect: do not capture the already-tuned state as a new baseline.
- Duplicate disconnect: do not corrupt state or invent a restoration.
- Partial apply: show Partial/Error and keep the snapshot usable.
- Restore failure: keep recovery data and allow `Restore Now` retry.
- Windows looks tuned but no owned snapshot exists: do not claim ownership or guess a prior state.

## Diagnostics

**[PLANNED]** Diagnostics should remain compact but expose enough to troubleshoot:

- detector health and current CRD state;
- last relevant event (with unnecessary client identity omitted/redacted);
- automation and category states;
- tuning and snapshot validity;
- last apply/restore outcome and last error;
- app version.

Logs use `INFO`, `WARN`, `ERROR`, and `DEBUG`, recording meaningful transitions rather than constant polling noise. Account/email identity from CRD events is not persisted by default.

## Known scope limits

Remotune does not:

- replace CRD or act as a remote desktop client;
- optimize a network, codec, or CRD transport;
- disable every custom animation in every application;
- expose Windows settings outside the approved Visual Effects profile surface;
- provide general Windows tweaking/debloating;
- guarantee an installer-free machine has WebView2;
- currently support RDP, RustDesk, AnyDesk, or other providers.

Potential future provider support remains explicitly out of current scope.

## Operator validation before release

A release is not ready based only on compilation or UI behavior. It must be exercised on representative Windows 10/11 systems for:

- CRD connect/disconnect and app-start-while-connected;
- exact Visual Effects Custom-state round-trip;
- taskbar ON/OFF round-trip;
- crashes and restart recovery;
- duplicate/racing events;
- Explorer restart;
- single/multiple monitors and secondary taskbars;
- normal-user permissions;
- Start with Windows and moved portable executable;
- missing WebView2;
- low idle resource use.

See the [roadmap](remotune.roadmap.md) for formal acceptance gates and the [uncertainty ledger](remotune.hallucination.md) for facts that must be verified before implementation.
