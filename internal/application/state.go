package application

type TuningState int

const (
	TuningUnknown TuningState = iota
	TuningBaseline
	TuningApplying
	TuningActive
	TuningRestoring
	TuningPartialError
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
func (s TuningState) CanApply() bool {
	switch s {
	case TuningApplying, TuningRestoring:
		return false // a transition is already in flight
	default:
		return true
	}
}
func (s TuningState) CanRestore() bool {
	switch s {
	case TuningApplying, TuningRestoring:
		return false
	default:
		return true
	}
}
func (s TuningState) IsTransient() bool {
	return s == TuningApplying || s == TuningRestoring
}
