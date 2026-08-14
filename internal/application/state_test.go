package application

import "testing"

func TestTuningStateString(t *testing.T) {
	cases := map[TuningState]string{
		TuningUnknown:          "Unknown",
		TuningBaseline:         "Baseline",
		TuningApplying:         "Applying",
		TuningActive:           "Active",
		TuningRestoring:        "Restoring",
		TuningPartialError:     "Partial/Error",
		TuningRecoveryRequired: "Recovery Required",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", int(s), got, want)
		}
	}
}

func TestTuningStateTransientBlocksNewTransitions(t *testing.T) {
	for _, s := range []TuningState{TuningApplying, TuningRestoring} {
		if !s.IsTransient() {
			t.Errorf("%s.IsTransient() = false, want true", s)
		}
		if s.CanApply() {
			t.Errorf("%s.CanApply() = true, want false (a transition is already in flight)", s)
		}
		if s.CanRestore() {
			t.Errorf("%s.CanRestore() = true, want false (a transition is already in flight)", s)
		}
	}
}

func TestTuningStateNonTransientStatesAllowTransitions(t *testing.T) {
	for _, s := range []TuningState{
		TuningUnknown, TuningBaseline, TuningActive, TuningPartialError, TuningRecoveryRequired,
	} {
		if s.IsTransient() {
			t.Errorf("%s.IsTransient() = true, want false", s)
		}
		if !s.CanApply() {
			t.Errorf("%s.CanApply() = false, want true", s)
		}
		if !s.CanRestore() {
			t.Errorf("%s.CanRestore() = false, want true", s)
		}
	}
}
