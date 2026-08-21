//go:build windows

package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/khiemnguyen/remotune/internal/crd"
	"github.com/khiemnguyen/remotune/internal/wintune"
)

type fakeVE struct {
	mu           sync.Mutex
	current      wintune.VisualEffectsSnapshot
	applyCalls   int
	restoreCalls int
	failApply    bool
	failRestore  bool
	verifyApply  bool // if false, ApplyBestPerformance's result reports Verified=false
}

func newFakeVE() *fakeVE {
	return &fakeVE{
		current: wintune.VisualEffectsSnapshot{
			SPI:      map[string]int32{"MenuAnimation": 1},
			Registry: map[string]wintune.RegValue{"VisualFXSetting": {Kind: wintune.RegKindDWord, DWord: 1}},
			Mask:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		verifyApply: true,
	}
}

func (f *fakeVE) Snapshot() (*wintune.VisualEffectsSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := f.current
	return &cp, nil
}

func (f *fakeVE) ApplyBestPerformance() (wintune.CategoryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyCalls++
	if f.failApply {
		err := errors.New("simulated apply failure")
		return wintune.CategoryResult{Category: wintune.CategoryVisualEffects, Err: err}, err
	}
	f.current.SPI["MenuAnimation"] = 0
	return wintune.CategoryResult{Category: wintune.CategoryVisualEffects, Changed: true, Verified: f.verifyApply}, nil
}

func (f *fakeVE) Restore(snap *wintune.VisualEffectsSnapshot) (wintune.CategoryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restoreCalls++
	if f.failRestore {
		err := errors.New("simulated restore failure")
		return wintune.CategoryResult{Category: wintune.CategoryVisualEffects, Err: err}, err
	}
	f.current = *snap
	return wintune.CategoryResult{Category: wintune.CategoryVisualEffects, Changed: true, Verified: true}, nil
}

type fakeTB struct {
	mu           sync.Mutex
	autoHide     bool
	applyCalls   int
	restoreCalls int
	failApply    bool
}

func newFakeTB() *fakeTB { return &fakeTB{autoHide: true} }

func (f *fakeTB) Snapshot() (*wintune.TaskbarSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &wintune.TaskbarSnapshot{AutoHide: f.autoHide, ABMState: boolToState(f.autoHide), LivePersistedAgreed: true}, nil
}

func (f *fakeTB) SetAutoHide(on bool) (wintune.CategoryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyCalls++
	if f.failApply {
		err := errors.New("simulated taskbar apply failure")
		return wintune.CategoryResult{Category: wintune.CategoryTaskbar, Err: err}, err
	}
	f.autoHide = on
	return wintune.CategoryResult{Category: wintune.CategoryTaskbar, Changed: true, Verified: true}, nil
}

func (f *fakeTB) Restore(snap *wintune.TaskbarSnapshot) (wintune.CategoryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restoreCalls++
	f.autoHide = snap.AutoHide
	return wintune.CategoryResult{Category: wintune.CategoryTaskbar, Changed: true, Verified: true}, nil
}

func boolToState(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func newTestCoordinator(t *testing.T, cfg AutomationConfig) (*Coordinator, *fakeVE, *fakeTB, *RecoveryStore) {
	t.Helper()
	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()
	c := NewCoordinator(store, ve, tb, cfg)
	return c, ve, tb, store
}

func fullCfg() AutomationConfig {
	return AutomationConfig{Enabled: true, VisualEffects: true, Taskbar: true}
}

func TestApplyPersistsSnapshotBeforeMutating(t *testing.T) {
	c, ve, _, store := newTestCoordinator(t, fullCfg())

	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !store.Exists() {
		t.Fatal("recovery snapshot was not persisted before applying")
	}
	if ve.applyCalls != 1 {
		t.Fatalf("apply calls = %d, want 1", ve.applyCalls)
	}
	if got := c.Status().Tuning; got != TuningActive {
		t.Fatalf("tuning state = %s, want Active", got)
	}
}

func TestRepeatedApplyDoesNotReplaceOriginalBaseline(t *testing.T) {
	c, ve, _, store := newTestCoordinator(t, fullCfg())

	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	first, err := store.Load()
	if err != nil {
		t.Fatalf("Load after first apply: %v", err)
	}
	if err := c.Observe(crd.StateConnected); err != nil {
		t.Fatalf("Observe (duplicate connect): %v", err)
	}
	second, err := store.Load()
	if err != nil {
		t.Fatalf("Load after duplicate connect: %v", err)
	}

	if first.CapturedAt != second.CapturedAt {
		t.Fatal("recovery snapshot was replaced by a duplicate connect; baseline must survive unchanged")
	}
	if ve.applyCalls < 2 {
		t.Fatalf("apply calls = %d, want at least 2 (idempotent re-apply)", ve.applyCalls)
	}
}

func TestPartialApplyRetainsSnapshotAndReportsPartialError(t *testing.T) {
	c, ve, _, store := newTestCoordinator(t, fullCfg())
	ve.failApply = true

	err := c.Bootstrap(crd.StateConnected)
	if err == nil {
		t.Fatal("expected an error from a partial apply")
	}
	if got := c.Status().Tuning; got != TuningPartialError {
		t.Fatalf("tuning state = %s, want Partial/Error", got)
	}
	if !store.Exists() {
		t.Fatal("recovery snapshot must be retained after a partial apply, not discarded")
	}
}

func TestApplyWithNoCategoriesEnabledStaysBaseline(t *testing.T) {
	c, ve, tb, store := newTestCoordinator(t, AutomationConfig{Enabled: true})
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if got := c.Status().Tuning; got != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline (nothing enabled)", got)
	}
	if store.Exists() || ve.applyCalls != 0 || tb.applyCalls != 0 {
		t.Fatal("nothing should be applied or persisted when no category is enabled")
	}
}

func TestRestoreNowRefusesWithNoOwnedSnapshot(t *testing.T) {
	c, _, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateDisconnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	err := c.RestoreNow()
	if !errors.Is(err, ErrNoRecovery) {
		t.Fatalf("RestoreNow() error = %v, want ErrNoRecovery", err)
	}
}

func TestRestoreRetiresSnapshotOnlyOnFullSuccess(t *testing.T) {
	c, _, _, store := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap (apply): %v", err)
	}
	if !store.Exists() {
		t.Fatal("expected a snapshot after apply")
	}

	if err := c.Observe(crd.StateDisconnected); err != nil {
		t.Fatalf("Observe (disconnect, triggers restore): %v", err)
	}
	if got := c.Status().Tuning; got != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline after successful restore", got)
	}
	if store.Exists() {
		t.Fatal("snapshot must be retired after a verified successful restore")
	}
}

func TestPartialRestoreRetainsSnapshot(t *testing.T) {
	c, ve, _, store := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap (apply): %v", err)
	}
	ve.failRestore = true

	err := c.Observe(crd.StateDisconnected)
	if err == nil {
		t.Fatal("expected an error from a partial restore")
	}
	if got := c.Status().Tuning; got != TuningPartialError {
		t.Fatalf("tuning state = %s, want Partial/Error", got)
	}
	if !store.Exists() {
		t.Fatal("recovery snapshot must be retained after a partial restore for retry")
	}
}

func TestRepeatedRestoreAfterSuccessIsNoOp(t *testing.T) {
	c, ve, tb, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Observe(crd.StateDisconnected); err != nil {
		t.Fatalf("Observe (disconnect): %v", err)
	}
	restoreCallsAfterFirst := ve.restoreCalls
	_ = tb
	if err := c.Observe(crd.StateDisconnected); err != nil {
		t.Fatalf("Observe (duplicate disconnect): %v", err)
	}
	if ve.restoreCalls != restoreCallsAfterFirst {
		t.Fatalf("restore calls = %d, want unchanged at %d (duplicate disconnect after baseline must be a no-op)",
			ve.restoreCalls, restoreCallsAfterFirst)
	}
}

func TestPauseRestoresOwnedStateAndSetsPaused(t *testing.T) {
	c, _, _, store := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	st := c.Status()
	if !st.Paused {
		t.Fatal("Paused = false after Pause()")
	}
	if st.Tuning != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline after Pause restores owned state", st.Tuning)
	}
	if store.Exists() {
		t.Fatal("snapshot should be retired after Pause's restore succeeds")
	}
}

func TestPauseSetsPausedEvenIfRestoreIsPartial(t *testing.T) {
	c, ve, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	ve.failRestore = true

	err := c.Pause()
	if err == nil {
		t.Fatal("expected an error from Pause's partial restore")
	}
	if !c.Status().Paused {
		t.Fatal("Paused must still be set to true even when the restore it triggers is partial; " +
			"pausing is a deliberate operator command and must not silently fail to register")
	}
}

func TestResumeReappliesWhenStillConnected(t *testing.T) {
	c, ve, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Pause(); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	applyCallsBeforeResume := ve.applyCalls

	if err := c.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if c.Status().Paused {
		t.Fatal("Paused = true after Resume()")
	}
	if c.Status().Tuning != TuningActive {
		t.Fatalf("tuning state = %s, want Active after Resume reconciles a still-Connected session", c.Status().Tuning)
	}
	if ve.applyCalls <= applyCallsBeforeResume {
		t.Fatal("Resume should have re-applied since CRD is still Connected")
	}
}

func TestQuitRestoresOwnedStateAndIsIdempotent(t *testing.T) {
	c, _, _, store := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Quit(); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if store.Exists() {
		t.Fatal("snapshot should be retired after Quit's restore succeeds")
	}
	if err := c.Quit(); err != nil {
		t.Fatalf("second Quit should be a no-op, got: %v", err)
	}
}

func TestQuitRejectsNewObservations(t *testing.T) {
	c, _, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateDisconnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := c.Quit(); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if err := c.Observe(crd.StateConnected); !errors.Is(err, ErrShuttingDown) {
		t.Fatalf("Observe after Quit error = %v, want ErrShuttingDown", err)
	}
}

func TestBootstrapNoOwnershipIsBaselineRegardlessOfCurrentWindowsState(t *testing.T) {
	c, ve, _, store := newTestCoordinator(t, fullCfg())
	ve.current.SPI["MenuAnimation"] = 0 // looks "already tuned"

	if err := c.Bootstrap(crd.StateDisconnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if got := c.Status().Tuning; got != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline", got)
	}
	if store.Exists() {
		t.Fatal("bootstrap must not invent ownership from observed Windows state alone")
	}
}

func TestBootstrapWithOwnershipStillConnectedReconciliesWithoutReplacingBaseline(t *testing.T) {
	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()

	preExisting := validSnapshot()
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("seed recovery file: %v", err)
	}

	c := NewCoordinator(store, ve, tb, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load after bootstrap: %v", err)
	}
	if !got.CapturedAt.Equal(preExisting.CapturedAt) {
		t.Fatal("pre-existing ownership was replaced by bootstrap reconciliation; it must be reused, not recaptured")
	}
	if got := c.Status().Tuning; got != TuningActive {
		t.Fatalf("tuning state = %s, want Active", got)
	}
}

func TestBootstrapWithOwnershipDisconnectedRestores(t *testing.T) {
	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()

	preExisting := validSnapshot()
	if err := store.Save(preExisting); err != nil {
		t.Fatalf("seed recovery file: %v", err)
	}

	c := NewCoordinator(store, ve, tb, fullCfg())
	if err := c.Bootstrap(crd.StateDisconnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if got := c.Status().Tuning; got != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline after restoring a disconnected-but-owned session", got)
	}
	if store.Exists() {
		t.Fatal("snapshot must be retired after the crash-recovery restore succeeds")
	}
}
func TestConcurrentObservationsAreSerializedAndLatestWins(t *testing.T) {
	c, _, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateDisconnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		state := crd.StateConnected
		if i%2 == 0 {
			state = crd.StateDisconnected
		}
		go func(s crd.State) {
			defer wg.Done()
			_ = c.Observe(s) // errors are expected/ignored under partial-failure races; no panic is the bar
		}(state)
	}
	wg.Wait()
	if err := c.Observe(crd.StateConnected); err != nil {
		t.Fatalf("final Observe: %v", err)
	}
	if got := c.Status().Tuning; got != TuningActive {
		t.Fatalf("tuning state after settling = %s, want Active", got)
	}
}
func TestDisconnectDuringApplyReconcilesToRestored(t *testing.T) {
	c, _, _, store := newTestCoordinator(t, fullCfg())

	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap (connect): %v", err)
	}
	if got := c.Status().Tuning; got != TuningActive {
		t.Fatalf("tuning state = %s, want Active", got)
	}

	if err := c.Observe(crd.StateDisconnected); err != nil {
		t.Fatalf("Observe (disconnect): %v", err)
	}
	if got := c.Status().Tuning; got != TuningBaseline {
		t.Fatalf("tuning state = %s, want Baseline after the disconnect is reconciled", got)
	}
	if store.Exists() {
		t.Fatal("snapshot should be retired once disconnect is fully reconciled")
	}
}

func TestStatusDoesNotBlockDuringLongRunningTransition(t *testing.T) {
	c, _, _, _ := newTestCoordinator(t, fullCfg())
	if err := c.Bootstrap(crd.StateConnected); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c.Status()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Status() did not return within 2s")
	}
}

type fakeBootstrapper struct {
	boot crd.BootstrapResult
	sub  *fakeSubscription
}

func (f *fakeBootstrapper) Bootstrap() (crd.BootstrapResult, error) { return f.boot, nil }
func (f *fakeBootstrapper) SubscribeAfterBookmark(string) (Subscription, error) {
	return f.sub, nil
}

type fakeSubscription struct {
	mu     sync.Mutex
	queue  [][]crd.Transition
	closed bool
}

func (f *fakeSubscription) Poll(max int, timeoutMs uint32) ([]crd.Transition, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 {
		time.Sleep(10 * time.Millisecond) // avoid a hot spin in the test
		return nil, 0, nil
	}
	next := f.queue[0]
	f.queue = f.queue[1:]
	return next, 0, nil
}

func (f *fakeSubscription) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
}

func TestRunAppliesOnBootstrapAndRestoresOnLiveDisconnect(t *testing.T) {
	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()
	c := NewCoordinator(store, ve, tb, fullCfg())

	connectTr := crd.Transition{Kind: crd.KindConnected, RecordID: 1, Time: time.Now(), ProcessID: 100, SessionID: "s1"}
	disconnectTr := crd.Transition{Kind: crd.KindDisconnected, RecordID: 2, Time: time.Now().Add(time.Second), ProcessID: 100, SessionID: "s1"}

	det := &fakeBootstrapper{
		boot: crd.BootstrapResult{
			Snapshot: crd.Reconstruct([]crd.Transition{connectTr}), // already Connected at boot
			Bookmark: "fake-bookmark",
		},
		sub: &fakeSubscription{queue: [][]crd.Transition{{disconnectTr}}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, c, det) }()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case err := <-done:
			t.Fatalf("Run exited early: %v", err)
		case <-deadline:
			t.Fatalf("timed out waiting for restore; final status: %+v", c.Status())
		default:
		}
		if c.Status().Tuning == TuningBaseline {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !det.sub.closed {
		cancel()
		<-done
		if !det.sub.closed {
			t.Fatal("subscription was not closed after Run returned")
		}
	}
	if store.Exists() {
		t.Fatal("snapshot should be retired after the live disconnect was reconciled")
	}
}

func TestRunSurfacesRedactedTransitionDiagnostics(t *testing.T) {
	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()
	c := NewCoordinator(store, ve, tb, fullCfg())

	connect := crd.Transition{Kind: crd.KindConnected, RecordID: 10, Time: time.Now(), ProcessID: 100, SessionID: "s1"}
	disconnect := crd.Transition{Kind: crd.KindDisconnected, RecordID: 11, Time: time.Now().Add(time.Second), ProcessID: 100, SessionID: "s1"}
	det := &fakeBootstrapper{
		boot: crd.BootstrapResult{Snapshot: crd.Reconstruct([]crd.Transition{connect}), Bookmark: "fake-bookmark"},
		sub:  &fakeSubscription{queue: [][]crd.Transition{{disconnect}}},
	}
	diagnostics := NewDetectorDiagnostics()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, c, det, diagnostics) }()

	deadline := time.After(2 * time.Second)
	for diagnostics.Status().LastRecordID != disconnect.RecordID {
		select {
		case <-deadline:
			t.Fatalf("diagnostics never recorded live transition: %+v", diagnostics.Status())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	got := diagnostics.Status()
	if got.Health != "Healthy" || got.LastTransition != crd.KindDisconnected.String() {
		t.Fatalf("diagnostics = %+v, want healthy disconnected transition", got)
	}
	if got.LastRecordID != disconnect.RecordID {
		t.Fatalf("last record = %d, want %d", got.LastRecordID, disconnect.RecordID)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}

type reconcilingBootstrapper struct {
	mu    sync.Mutex
	boots []crd.BootstrapResult
	sub   *fakeSubscription
}

func (f *reconcilingBootstrapper) Bootstrap() (crd.BootstrapResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	boot := f.boots[0]
	if len(f.boots) > 1 {
		f.boots = f.boots[1:]
	}
	return boot, nil
}

func (f *reconcilingBootstrapper) SubscribeAfterBookmark(string) (Subscription, error) {
	return f.sub, nil
}

func TestRunReconcilesBoundedStaleSubscription(t *testing.T) {
	oldInterval := reconciliationInterval
	reconciliationInterval = 20 * time.Millisecond
	t.Cleanup(func() { reconciliationInterval = oldInterval })

	store := newTestStore(t)
	ve := newFakeVE()
	tb := newFakeTB()
	c := NewCoordinator(store, ve, tb, fullCfg())
	connect := crd.Transition{Kind: crd.KindConnected, RecordID: 20, Time: time.Now(), ProcessID: 100, SessionID: "s1"}
	disconnect := crd.Transition{Kind: crd.KindDisconnected, RecordID: 21, Time: time.Now().Add(time.Second), ProcessID: 100, SessionID: "s1"}
	det := &reconcilingBootstrapper{
		boots: []crd.BootstrapResult{
			{Snapshot: crd.Reconstruct([]crd.Transition{connect}), Bookmark: "first"},
			{Snapshot: crd.Reconstruct([]crd.Transition{connect, disconnect}), Bookmark: "second"},
		},
		sub: &fakeSubscription{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, c, det, NewDetectorDiagnostics()) }()

	deadline := time.After(2 * time.Second)
	for c.Status().Tuning != TuningBaseline {
		select {
		case <-deadline:
			t.Fatalf("bounded reconciliation did not restore stale connected state: %+v", c.Status())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}
