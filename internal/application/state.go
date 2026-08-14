package application

// This file's state machine is pure and platform-independent (no build tag), like
// crd.Reconstruct, so its transition rules are unit-testable without Windows.

// TuningState is the coordinator's ownership/transition state, one of the three
// dimensions exposed to the UI (docs/baseline/remotune.sourcecode.md, "State
// dimensions exposed to the UI remain separate"). CRD connection state and
// automation enablement are tracked separately and are not part of this type.
type TuningState int

const (
	// TuningUnknown is the state before any observation or reconstruction has run.
	TuningUnknown TuningState = iota
	// TuningBaseline means no owned override is active; Windows is in the user's
	// own state as far as Remotune knows.
	TuningBaseline
	// TuningApplying is a transient state while an apply transaction is in flight.
	TuningApplying
	// TuningActive means an owned override was applied and verified.
	TuningActive
	// TuningRestoring is a transient state while a restore transaction is in flight.
	TuningRestoring
	// TuningPartialError means an apply or restore left some categories inconsistent.
	// It must never be presented as success, and recovery data is retained.
	TuningPartialError
	// TuningRecoveryRequired means durable state exists, or state is uncertain, and
	// operator attention may be needed (e.g. a corrupt recovery file could not be
	// validated, but ownership evidence suggests one existed).
	TuningRecoveryRequired
)

func (s TuningState) String() string {
	switch s {
	case TuningBaseline:
		return "Baseline"
	case TuningApplying:
		return "Applying"
	case TuningActive:
		return "Active"
	case TuningRestoring:
		return "Restoring"
	case TuningPartialError:
		return "Partial/Error"
	case TuningRecoveryRequired:
		return "Recovery Required"
	default:
		return "Unknown"
	}
}

// CanApply reports whether an apply transition is meaningful from this state.
//
// Applying from PartialError or RecoveryRequired is allowed deliberately: those
// states retain their recovery data, and a fresh apply attempt (or, more commonly,
// a restore attempt) must be able to proceed from them rather than being wedged.
// Applying while already Active is also allowed and is the idempotent "repeated
// apply keeps the original snapshot" case from the baseline.
func (s TuningState) CanApply() bool {
	switch s {
	case TuningApplying, TuningRestoring:
		return false // a transition is already in flight
	default:
		return true
	}
}

// CanRestore reports whether a restore transition is meaningful from this state.
// Baseline is included: a repeated restore after success is defined as a no-op/report
// rather than a new ownership cycle, and allowing the call keeps that rule in one
// place (the coordinator) instead of requiring every caller to pre-check state.
func (s TuningState) CanRestore() bool {
	switch s {
	case TuningApplying, TuningRestoring:
		return false
	default:
		return true
	}
}

// IsTransient reports whether s is an in-flight transaction state, meaning a second
// transition must not be started concurrently with it.
func (s TuningState) IsTransient() bool {
	return s == TuningApplying || s == TuningRestoring
}
