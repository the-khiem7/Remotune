---
baseline_schema: "2.0"
pack: "remotune"
document: "sourcecode"
status: "active"
updated: "2026-08-21"
code_ref: "uncommitted"
---

# Remotune Source Architecture

**[IMPLEMENTED]** `normalizeAutostartPath` trims optional outer quotes and whitespace from the Windows Run value before comparing it to the current executable. This preserves quoted portable-path registration while removing the false moved-path warning; focused lifecycle tests cover it.

**[IMPLEMENTED]** Host-native development uses the pinned `%USERPROFILE%\go\bin\wails3.exe` and `build/config.yml`. `wails3 dev` runs Wails dev mode; Vite hot-reloads frontend changes while Wails rebuilds and relaunches on Go changes. The root `Taskfile.yml` supplies `wails3 task verify` and `wails3 task windows:portable` without project-specific PowerShell wrappers.

## Implementation status

**[IMPLEMENTED]** Phases 1 through 6. The root Go module `github.com/khiemnguyen/remotune` contains the Wails v3 tray shell, migrated Phase 1–3 adapters, coordinator, lifecycle package, compact Vue/Vite control surface, and Visual Effects profile/editor revision. The standalone `engine/` module is superseded but retained for reference.

## Repository contribution convention

**[DECIDED]** Code files contain code only: do not add explanatory comments, architecture rationale, operational guidance, or implementation narrative to them. The active baseline documentation is the durable location for that material. This applies to future changes; historical comments are not evidence of current product behavior.

**[UNVERIFIED]** Phase 7 hardening remains. Phase 6 profile application and its Custom editor passed host-native verification, but target-machine profile and multi-window observation remain pending. The versioned v0.1.4 binary has build/test evidence and its corrected Vue window, including Close-to-tray, was manually observed on the target machine on 2026-08-20. Restore Now, Start with Windows, Pause, and Resume were exercised in v0.1.3, but the end-to-end safety workflow is open: v0.1.4 reported a disconnected CRD state during an active connection, and the operator reported a prior Explicit Quit that did not restore animation.

**[VERIFIED]** items rest on the live observations recorded in [Phase 0 recorded evidence](remotune.roadmap.md#phase-0-recorded-evidence), collected on Windows 11 Pro 23H2 with CRD host 152.0.7977.9 as a non-elevated user. `tools/phase0/Get-VisualState.ps1` is the working reference for the snapshot shape described under [Persistence](#persistence).

## Current presentation and packaging flow

`main.go` embeds `frontend/dist`, `assets/app/remotune-256.png`, and `assets/tray/remotune-32.png`. It supplies the app icon to Wails, sets the system-tray icon, creates initially hidden main and Custom Visual Effects windows, and binds `internal/lifecycle.Service` as the only Vue-facing backend surface. The main window injects an editor-opening callback into the service constructor; the service exposes only `OpenCustomEffectsEditor` to Vue and has no dependency on a Windows system-settings launcher.

`frontend/src/App.vue` translates numeric `crd.State` and `application.TuningState` values to display labels at its boundary. It chooses the main or Custom editor presentation from the Wails window query and persists every Custom checklist edit through `SetProfileSettings`. It must not call string operations on raw transport values. `frontend/src/wails.ts` remains the sole import point for generated bindings.

The Go module, `wails3.exe`, and `@wailsio/runtime` are all fixed at `v3.0.0-beta.8`; `bun.lock` makes `bun install --frozen-lockfile` reproducible. A caret range is not acceptable for this Beta framework because it can silently place a different runtime beside the pinned Go transport.

`wails3 dev` invokes host-native Wails dev mode. The Wails execution graph generates bindings, starts Vite on port 9245, builds the native app, and runs it against the development server. `wails3 task verify` generates bindings and resources, runs the reproducible frontend build, then enforces formatting, `go vet`, and short Go tests. `wails3 task windows:portable` reads the semantic version only from `build/config.yml`, preserves the versioned artifact convention, and refuses a locked destination. Docker files are retired infrastructure, not supported development tooling. The `.syso` remains reproducible and ignored; the SVG, PNGs, ICO, and manifest are the committed inputs.

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
github.com/khiemnguyen/remotune  (root module, Wails v3.0.0-beta.8)
├─ main.go, tray.go              (Wails bootstrap, tray menu)
├─ internal/
│  ├─ application/               (Phase 3: Coordinator, RecoveryStore, TuningState, startup Run)
│  ├─ crd/                       (Phase 2: Bootstrap, Reconstruct, Subscribe, event parsing)
│  ├─ wintune/                   (Phase 1: VisualEffectsManager, TaskbarManager, Snapshot, win32 bindings)
│  └─ lifecycle/                 (Phase 4: Service, WebView2 check, autostart, portable path)
├─ cmd/
│  ├─ crdwatch/                  (operator observation tool)
│  ├─ e2e/                       (end-to-end verification)
│  └─ tbset/                     (taskbar state utility)
├─ engine/                       (superseded standalone module, retained for reference)
├─ tools/phase0/                 (Phase 0 evidence scripts and snapshots)
└─ docs/baseline/                (this pack)
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
- reconcile from the same Event Log history at a bounded interval when live delivery may be stale;
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

**[IMPLEMENTED]** `VisualEffectsManager.ApplyProfile(profile, custom)` replaces the single-target boundary while retaining `Snapshot()` and `Restore(snapshot)` as distinct recovery operations. `internal/wintune/profile.go` compiles Best Appearance, Best Performance, and Custom into a three-layer target; Let Windows choose preserves the live effect values and writes label `0`, because Windows does not recompute values when that label changes. The user-facing scope is the 17-item Visual Effects checklist only; it excludes Advanced and Data Execution Prevention settings.

**[VERIFIED]** The accessor surface is `SystemParametersInfo`. Seventeen `SPI_GET*` actions plus `SPI_GETANIMATION` cover the individual effects and all succeed non-elevated; the constants are tabulated in the Phase 0 evidence. The preset selection itself is `VisualFXSetting` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects` (`0` Let Windows choose, `1` Best appearance, `2` Best performance, `3` Custom).

**[VERIFIED]** Two traps are settled. The 19 `VisualEffects` subkeys hold only `DefaultApplied` bookkeeping and are not effect state. And `UserPreferencesMask` contains set bits that could not be attributed to any documented effect, so it is snapshotted and restored as an **opaque 8-byte blob**; unattributed bits are preserved by construction rather than by understanding them. Per-effect reads and writes go through `SystemParametersInfo`.

**[VERIFIED]** A snapshot is only complete when it spans three layers. No single layer suffices:

```text
layer 1  per-effect values via SystemParametersInfo
layer 2  discrete registry values with NO SPI accessor
         Explorer\Advanced: ListviewAlphaSelect, ListviewShadow, TaskbarAnimations, IconsOnly
         DWM: EnableAeroPeek, AlwaysHibernateThumbnails, Composition
         Desktop: FontSmoothing, FontSmoothingType, DragFullWindows, MenuShowDelay
         WindowMetrics: MinAnimate
layer 3  UserPreferencesMask, opaque, verbatim
```

**[VERIFIED]** `ApplyBestPerformance()` reproduces the 22-value set that the real Windows preset changes, enumerated in the Phase 0 evidence. It must not disable effects the preset leaves alone, and it must account for `IconsOnly` rising from 0 to 1 rather than falling.

**[IMPLEMENTED]** `VisualEffectsProfile` is declarative rather than a copied machine snapshot. Built-in profiles compile the supported target combinations; `Remotune Custom` persists the 17 Visual Effects checkbox choices. The compiler derives known mask bits from selected SPI values while carrying unknown mask bits from the live baseline, writes the profile label last, and delegates all writes to the same convergence path as restore. A profile selection is not trusted until the observed three-layer state matches its target.

**[VERIFIED]** `FontSmoothing` is not boolean: the registry held `2` while the SPI accessor reported `1`. Restoring from the boolean would silently downgrade ClearType, so the registry value is authoritative and `FontSmoothingType` travels with it.

**[VERIFIED]** Apply and restore are one bounded convergence loop, not a single write pass. The shell applies and reloads these settings asynchronously, so the observable outcome is re-read and only the diverging values are re-asserted:

```text
repeat up to 4 times:
    1. per-effect SystemParametersInfo writes, SPIF_SENDCHANGE only   -> live session
    2. discrete registry values, except the preset label              -> persistence
    3. UserPreferencesMask, one whole-blob write                      -> persistence
    4. settle, then VisualFXSetting alone                             -> preset label
    5. re-read; if it matches the target, broadcast once and stop
       otherwise narrow the next pass to the values still diverging
on exhaustion: report failure WITH the residual difference, never success
```

Three measured mechanisms force this shape:

- `SPIF_UPDATEINIFILE` persists a write by read-modify-writing the shared mask byte, so consecutive writes to effects sharing a byte lose each other's bits. The flag is not used; persistence is explicit.
- Windows re-labels the configuration as `Custom` asynchronously when effects change, so the label is written last and alone.
- A global `WM_SETTINGCHANGE` mid-sequence makes the shell reload from the registry and undo writes that had already landed, so the broadcast happens only after the state already matches.

The mask is derived from the per-effect values via the attribution table, starting from the snapshot's mask so the two unexplained bits survive. Deriving rather than storing keeps the live and persisted layers consistent by construction.

**[VERIFIED]** Both operations are proven on the supported configuration: an arbitrary `Custom` state survived an apply/restore cycle with no differences, and applying Best Performance from a `Custom` start produced a state identical to the operator-produced preset, so apply is deterministic and start-state independent.

**[UNVERIFIED]** Windows 10 behavior is uncollected, the unattributed mask bit remains unexplained though safely preserved, and the visual immediacy of the Explorer-backed values was not confirmed even though their persisted values round-trip correctly.

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

**[VERIFIED]** `ABM_SETSTATE` alone is **not durable**. The live appbar state and the persisted `StuckRects3` `Settings` byte 8 were observed to diverge, and an override applied only through the API later reverted to the persisted value without any Remotune call. Both layers are therefore written on every change (superseding the earlier prohibition on touching `StuckRects3`):

```text
GetAutoHide()      read live ABM_GETSTATE, read persisted StuckRects3 bit 0
                   disagreement is a health signal, not normal

SetAutoHide(v)     ABM_SETSTATE  -> immediate live effect, flip only ABS_AUTOHIDE
                   StuckRects3   -> flip only bit 0 of Settings byte 8
                   both, always, so no divergence window exists
```

The old objection that `StuckRects3` needs an Explorer restart does not apply: `ABM_SETSTATE` supplies the live effect, and the registry write exists only to remove the divergence. A durable set was verified to survive a `WM_SETTINGCHANGE` broadcast with both layers still agreeing.

**[VERIFIED]** Both baseline directions round-trip. A baseline of auto-hide OFF stayed OFF through apply and restore, and was never forced to ON.

**[UNVERIFIED]** Multi-monitor and secondary-taskbar behavior, Explorer-restart interaction, and Windows 10 parity remain uncollected. Bit preservation was also only trivially exercised, since no bits beyond `ABS_AUTOHIDE` were set on the evidence machine. The precise trigger for the observed spontaneous revert was not identified; a phase-by-phase bisect of a Visual Effects apply did not reproduce it.

### Persistence

**[IMPLEMENTED]** `internal/application.RecoveryStore`, at `%LOCALAPPDATA%\Remotune\recovery.json` (resolved via `os.UserCacheDir`, overridable for tests). One file, not per-category, because ownership is a single unit: a duplicate connect must not be able to replace one category's baseline while leaving another's stale (ledger decision 14).

**[VERIFIED]** `Save` is atomic: it writes to a temp file in the same directory (guaranteeing a same-volume, and therefore atomic, `os.Rename`) and only renames over the target after the write and an `fsync` succeed. `Load` collapses a missing file, a corrupt (non-JSON) file, and a file with a mismatched `SchemaVersion` into the single `ErrNoRecovery` sentinel, so a caller cannot mistake a corrupt-but-present file for valid ownership (ledger decision 75). The remaining candidate contents, `config.json` and `logs\`, are not yet implemented.

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
    "registry": {
      "Desktop.FontSmoothing": 2,
      "Desktop.FontSmoothingType": 2,
      "Advanced.IconsOnly": 0,
      "DWM.EnableAeroPeek": 1,
      "...": "every layer-2 value listed above"
    }
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

**[IMPLEMENTED]** `internal/application.Coordinator`. One coordinator owns all state transitions. Detector callbacks, Vue handlers, tray callbacks, startup, and recovery code submit observations/commands; they do not call Windows managers directly.

Desired state is derived from:

```text
observed CRD state
+ automation enabled/paused
+ enabled automation categories
+ persisted ownership/recovery state
= desired Windows state
```

**[VERIFIED]** The coordinator serializes transitions using one `sync.Mutex` around every exported method, not a separate worker/event loop with a queue: a call arriving while another is in flight blocks until it finishes, then reconciles against whatever is now current. There are no concurrent Visual Effects or taskbar writes. If disconnect arrives while apply is running, the latest desired state eventually wins after serialized reconciliation, verified by a test firing 50 concurrent calls across 8 consecutive clean runs (`go test -race` was unavailable on the evidence machine; see ledger decision 76).

**[VERIFIED]** `VisualEffectsAdapter`, `TaskbarAdapter`, `Bootstrapper`, and `Subscription` are declared as coordinator-local interfaces satisfied by the real `wintune`/`crd` types, rather than the coordinator depending on those concrete types. This is what let 29 coordinator tests run fast and repeatably against fakes without mutating the operator's real desktop on every run, extending the same lesson Phase 1 learned about needing non-mutating test cycles.

**[VERIFIED]** The coordinator has been run end-to-end against the real `wintune`/`crd` adapters on 2026-08-14 (`cmd/e2e`): a full simulated CRD Connected → apply → CRD Disconnected → restore cycle recovered the operator's exact original state with no differences, 3 consecutive clean runs. Each adapter is also independently proven on real hardware (Phase 1/2), and the coordinator's decision logic is independently proven against fakes (Phase 3, 29 tests).

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
5. Apply the configured CRD-on profile if Visual Effects automation is enabled; the current implementation applies Best Performance until Phase 6 is complete.
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

**[IMPLEMENTED]** Wails owns:

- application lifecycle (`application.New`, `app.Run`, `app.Quit`);
- system tray and menus (`app.SystemTray.New`, tray context menu);
- `DisableQuitOnLastWindowClosed` so automation survives window close;
- `ApplicationStarted` refreshes the tray only after Wails owns a valid application context;
- service startup/shutdown invokes `ServiceStartup`/`ServiceShutdown`, starts the detector loop, then cancels it and restores owned state before process exit.

**[IMPLEMENTED]** `internal/lifecycle.Service` bridges Wails to the coordinator:

- `ServiceStartup(ctx, options)` starts the private detector loop after Wails creates its context;
- `ServiceShutdown()` cancels the poll loop, waits for exit, then calls `coord.Quit()`;
- `Status()`, `Pause()`, `Resume()`, `RestoreNow()` proxy to the coordinator under a mutex.

`Run` and `Shutdown` are not generated frontend commands. The native tray uses the package-level `lifecycle.Shutdown(service)` helper so restore-before-quit remains synchronous without widening the renderer binding surface.

Wails does not own tuning decisions or platform mutations.

**[PLANNED]** Vue (Phase 5):

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

Adding a detector such as `RDPDetector`, `RustDeskDetector`, or `AnyDeskDetector` is valid future scope only when explicitly requested. It extends the remote-session trigger. The approved Visual Effects profile surface remains bounded to CRD automation and the Windows Performance Options Visual Effects tab; it does not justify a generic Windows tuning control panel.

## Build environment

**[IMPLEMENTED]** Host-native build policy. Compilation, code generation, verification, and testing execute directly on Windows. Docker/Rancher Desktop is not a supported dependency for the development loop.

```text
%USERPROFILE%\go\bin\wails3.exe → pinned Wails CLI v3.0.0-beta.8
build/config.yml    → Wails dev-mode execution graph
Taskfile.yml        → Wails task graph for verification and portable packaging
```

Key design choices:

- **No mingw-w64**: Wails v3 Windows builds are CGO-free via `go-winloader`.
- **Exact runtime lock**: `frontend/bun.lock` and `bun install --frozen-lockfile` keep the frontend runtime on the same `v3.0.0-beta.8` release as the Go module and CLI.
- **Fast feedback**: Vite HMR updates frontend changes without a native rebuild; Wails detects Go changes and relaunches the native app.
- **Native verification**: `wails3 task verify` generates bindings and frontend assets, checks formatting, runs `go vet ./...`, and executes the Windows-native short test suite.

Historical Docker reference: [docs/runbook/docker-build-environment.md](../runbook/docker-build-environment.md). The active host-native workflow is recorded in this baseline pack.
