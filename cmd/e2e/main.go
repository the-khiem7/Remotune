//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/khiemnguyen/remotune/internal/application"
	"github.com/khiemnguyen/remotune/internal/crd"
	"github.com/khiemnguyen/remotune/internal/wintune"
)

func main() {
	if os.Getenv("REMOTUNE_SYSTEM_TESTS") != "1" {
		fmt.Fprintln(os.Stderr, "e2e: skipped (set REMOTUNE_SYSTEM_TESTS=1 to run)")
		os.Exit(0)
	}

	ve := &wintune.VisualEffectsManager{}
	tb := &wintune.TaskbarManager{}
	origVE, err := ve.Snapshot()
	if err != nil {
		fatal("snapshot VE: %v", err)
	}
	origTB, err := tb.Snapshot()
	if err != nil {
		fatal("snapshot TB: %v", err)
	}
	defer func() {
		ve.Restore(origVE)
		tb.Restore(origTB)
		fmt.Println("e2e: safety restore executed")
	}()

	store, err := application.NewRecoveryStore(os.TempDir() + `\remotune-e2e-` + fmt.Sprint(time.Now().UnixNano()))
	if err != nil {
		fatal("recovery store: %v", err)
	}

	cfg := application.AutomationConfig{Enabled: true, VisualEffects: true, Taskbar: true}
	coord := application.NewCoordinator(store, ve, tb, cfg)
	fmt.Println("e2e: bootstrap with CRD Disconnected")
	if err := coord.Bootstrap(crd.StateDisconnected); err != nil {
		fatal("bootstrap: %v", err)
	}
	assertStatus(coord, application.TuningBaseline, "after bootstrap")
	fmt.Println("e2e: observe CRD Connected (triggers apply)")
	if err := coord.Observe(crd.StateConnected); err != nil {
		fatal("observe Connected: %v", err)
	}
	assertStatus(coord, application.TuningActive, "after connect")
	fmt.Println("e2e: apply verified, state = Active")
	afterApplyVE, _ := ve.Snapshot()
	if afterApplyVE.SPI["MenuAnimation"] != 0 {
		fatal("MenuAnimation should be 0 after apply, got %d", afterApplyVE.SPI["MenuAnimation"])
	}
	afterApplyTB, _ := tb.Snapshot()
	if afterApplyTB.AutoHide != false {
		fatal("taskbar auto-hide should be OFF after apply")
	}
	fmt.Println("e2e: observable checks pass (VE tuned, taskbar auto-hide OFF)")
	fmt.Println("e2e: observe CRD Disconnected (triggers restore)")
	if err := coord.Observe(crd.StateDisconnected); err != nil {
		fatal("observe Disconnected: %v", err)
	}
	assertStatus(coord, application.TuningBaseline, "after disconnect")
	fmt.Println("e2e: restore verified, state = Baseline")
	finalVE, _ := ve.Snapshot()
	diff := wintune.DiffVisualEffects(origVE, finalVE)
	if len(diff) > 0 {
		fatal("Visual Effects not exactly restored: %v", diff)
	}
	finalTB, _ := tb.Snapshot()
	if finalTB.AutoHide != origTB.AutoHide {
		fatal("taskbar auto-hide not restored: got %v, want %v", finalTB.AutoHide, origTB.AutoHide)
	}

	fmt.Println("e2e: PASS — full apply/restore cycle through real adapters, exact state recovery confirmed")
	fmt.Printf("e2e: recovery store at: %v (should be empty)\n", store.Exists())
}

func assertStatus(c *application.Coordinator, want application.TuningState, ctx string) {
	got := c.Status().Tuning
	if got != want {
		fatal("status %s: got %s, want %s", ctx, got, want)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "e2e: FAIL — "+format+"\n", args...)
	os.Exit(1)
}
